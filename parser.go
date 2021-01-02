package cli

import (
	"context"
	"encoding"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/scylladb/go-set/strset"
)

// ParserOption option type for Parser
type ParserOption func(p *Parser)

// WithArgCase set the arg case. default is CaseCamelLower
func WithArgCase(c Case) ParserOption {
	return func(p *Parser) {
		p.argCaseFunc = caseFuncs[c]
	}
}

// WithCmdCase set the cmd case. default is CaseLower
func WithCmdCase(c Case) ParserOption {
	return func(p *Parser) {
		p.cmdCaseFunc = caseFuncs[c]
	}
}

// WithOnErrorStrategy sets the execution strategy for handling errors
func WithOnErrorStrategy(str OnErrorStrategy) ParserOption {
	return func(p *Parser) {
		p.strategy = str
	}
}

func WithGlobalArgsEnabled() ParserOption {
	return func(p *Parser) {
		p.globalsEnabled = true
	}
}

var defaultParser = NewParser()

type iface struct {
	m map[interface{}]reflect.Type
	f func(in, out interface{})
}

type Parser struct {
	strict         bool
	roots          []reflect.Value
	cmds           map[string]*command
	enums          map[reflect.Type]map[string]interface{}
	ifaces         map[string]*iface
	execTree       []interface{}
	globalsEnabled bool
	strategy       OnErrorStrategy
	argCaseFunc    func(string) string
	cmdCaseFunc    func(string) string
}

// NewParser create new parser
func NewParser(opts ...ParserOption) *Parser {
	p := &Parser{
		cmds:   map[string]*command{},
		enums:  map[reflect.Type]map[string]interface{}{},
		ifaces: map[string]*iface{},
	}
	for _, o := range opts {
		o(p)
	}
	// default arg case is CamelLower
	if p.argCaseFunc == nil {
		p.argCaseFunc = caseFuncs[CaseCamelLower]
	}
	// default cmd case is Lower
	if p.cmdCaseFunc == nil {
		p.cmdCaseFunc = caseFuncs[CaseLower]
	}
	return p
}

func (p *Parser) addRoot(in interface{}) *path {
	p.roots = append(p.roots, reflect.ValueOf(in))
	return &path{
		root: &p.roots[len(p.roots)-1],
	}
}

// NewRootCommand add new root command to defaultParser
func NewRootCommand(name string, arg interface{}) {
	defaultParser.NewRootCommand(name, arg)
}

// NewRootCommand add new root command to this parser
func (p *Parser) NewRootCommand(name string, arg interface{}) {
	c := &command{
		parser: p,
		name:   name,
		subcmd: map[string]*command{},
		args:   map[string]*argument{},
	}
	c.parse(arg)
	p.cmds[name] = c
}

// Eval marshal string args to struct using the defaultParser
func Eval(args []string) error {
	return defaultParser.Eval(args)
}

type argSet map[*argument]struct{}

func (s argSet) Insert(a *argument) bool {
	if _, ok := s[a]; ok {
		return false
	}
	s[a] = struct{}{}
	return true
}

func (s argSet) Delete(a *argument) bool {
	if _, ok := s[a]; !ok {
		return false
	}
	delete(s, a)
	return true
}

func (s argSet) List() (args []*argument) {
	for a := range s {
		args = append(args, a)
	}
	return
}

// Eval marshal string args to struct
func (p *Parser) Eval(args []string) error {

	currentCmdArgs := argSet{}
	globalArgs := argSet{}
	arrays := map[*argument][]string{}

	c, ok := p.cmds[args[0]]
	// try base path
	if !ok {
		c, ok = p.cmds[filepath.Base(args[0])]
		if !ok {
			return ErrCommandNotFound(args[0])
		}
	}

	p.execTree = append(p.execTree, c.path.Get())

	for _, a := range c.args {
		currentCmdArgs.Insert(a)
		if p.globalsEnabled && a.global {
			globalArgs.Insert(a)
		}
	}

	args = args[1:]
	positional := false
	positionals := []string{}
	for i := 0; i < len(args); i++ {

		arg := args[i]

		if arg == "--" {
			positional = true
			continue
		}

		if positional {
			positionals = append(positionals, arg)
			continue
		}

		if !isFlag(arg) {
			if c.subcmd != nil {
				cc, ok := c.subcmd[arg]
				if !ok {
					return ErrCommandNotFound(arg)
				}
				c = cc
				p.execTree = append(p.execTree, c.path.Get())
				continue
			}
			positionals = append(positionals, arg)
		}

		if arg == "-h" || arg == "--help" {
			// handle help
		}

		if arg == "--version" {
			// handle version
		}

		val := ""
		// get flag and value in case --flag=value
		if i := strings.Index(arg, "="); i != -1 {
			arg = arg[:i]
			val = arg[i+1:]
		}

		valid, short, arg := p.validateFlag(arg)
		if !valid {
			return ErrInvalidFlag(arg)
		}

		a := &argument{}
		var ok bool
		if short {
			a, ok = c.argsS[arg]
		} else {
			a, ok = c.args[arg]
		}
		if !ok {
			return ErrNoSuchFlag(arg)
		}

		currentCmdArgs.Delete(a)

		// handle arrays and slices
		if a.isArray || a.isSlice {
			if a.separate {
				if _, ok := arrays[a]; !ok {
					arrays[a] = []string{}
				}
				// if is array and overflows
				if a.isArray && len(arrays[a]) == a.arrayLen {
					return errors.New("array over capacity")
				}
				if val == "" {
					val = args[i+1]
					i++
				}
				arrays[a] = append(arrays[a], val)
				continue
			}
			// clear array
			arrays[a] = []string{}
			if a.isArray {
				for j := 0; j < a.arrayLen; j++ {
					val = args[i+1]
					if isFlag(val) {
						continue
					}
					arrays[a] = append(arrays[a], val)
					i++
				}
				continue
			}
			for _, val = range args[i+1:] {
				if isFlag(val) {
					continue
				}
				arrays[a] = append(arrays[a], val)
				i++
			}
		}

		// get the value in case --flag value
		if !a.isBool() && val == "" {
			val = args[i+1]
			i++
		}

		// handle encoding.TextUnmarshaler
		if tum, ok := a.path.Get().(encoding.TextUnmarshaler); ok {
			if err := tum.UnmarshalText([]byte(val)); err != nil {
				return err
			}
			continue
		}

		// handle enum
		if a.enum {
			em := p.enums[a.typ]
			a.setValue(em[strings.ToLower(val)])
			i++
			continue
		}

		// handle scalar
		if err := a.setScalarValue(val); err != nil {
			return err
		}
	}

	if len(arrays) != 0 {
		for a, v := range arrays {
			if err := a.setArrayValue(v); err != nil {
				return err
			}
		}
	}

	return nil
}

type OnErrorStrategy uint

const (
	OnErrorBreak OnErrorStrategy = iota
	OnErrorPostRunners
	OnErrorPostRunnersContinue
	OnErrorContinue
)

func Execute(ctx context.Context) error {
	return defaultParser.Execute(ctx)
}

type lastErrorKey struct{}

func LastErrorFromContext(ctx context.Context) error {
	return ctx.Value(lastErrorKey{}).(error)
}

func (p *Parser) Execute(ctx context.Context) error {

	var err error
	lastCmd := len(p.execTree) - 1
	pPostRunners := []PersistentPostRunner{}

	for i, inf := range p.execTree {
		// PersistentPostRun pushed on a stack to run in a reverse order
		if rnr, ok := inf.(PersistentPostRunner); ok {
			pPostRunners = append([]PersistentPostRunner{rnr}, pPostRunners...)
		}
		// PersistentPreRun
		if rnr, ok := inf.(PersistentPreRunner); ok {
			err = rnr.PersistentPreRun(ctx)
			if err != nil {
				if !(p.strategy == OnErrorContinue) {
					break
				}
				ctx = context.WithValue(ctx, lastErrorKey{}, err)
			}
		}
		if i == lastCmd {
			// PreRun
			if rnr, ok := inf.(PreRunner); ok {
				err = rnr.PreRun(ctx)
				if err != nil {
					if !(p.strategy == OnErrorContinue) {
						break
					}
					ctx = context.WithValue(ctx, lastErrorKey{}, err)
				}
			}
			// Run
			if rnr, ok := inf.(Runner); ok {
				err = rnr.Run(ctx)
				if err != nil {
					if !(p.strategy == OnErrorContinue) {
						break
					}
					ctx = context.WithValue(ctx, lastErrorKey{}, err)
				}
			}
			// PostRun
			if rnr, ok := inf.(PostRunner); ok {
				err = rnr.PostRun(ctx)
				if err != nil {
					if !(p.strategy == OnErrorContinue) {
						break
					}
					ctx = context.WithValue(ctx, lastErrorKey{}, err)
				}
			}
		}
	}
	// check for error and strategy
	if err != nil && p.strategy == OnErrorBreak {
		return err
	}
	// PersistentPostRun
	for _, rnr := range pPostRunners {
		err = rnr.PersistentPostRun(ctx)
		if err != nil {
			if p.strategy == OnErrorPostRunners {
				return err
			}
			ctx = context.WithValue(ctx, lastErrorKey{}, err)
		}
	}
	return err
}

func RegisterEnum(enmap interface{}) {
	defaultParser.RegisterEnum(enmap)
}

func (p *Parser) RegisterEnum(enmap interface{}) {
	v := reflect.ValueOf(enmap)
	t := reflect.TypeOf(enmap)
	if t.Kind() != reflect.Map {
		panic("enmap must be a map")
	}

	// key is the string of enum
	if t.Key().Kind() != reflect.String {
		panic("enmap key must be string")
	}

	// element is enum int/uint custom type
	te := t.Elem()
	if te.PkgPath() == "" {
		panic("enmap element must be custom type")
	}
	if !(isInt(te) || isUint(te)) {
		panic("enmap element must be int/uint")
	}

	enm := map[string]interface{}{}
	for _, k := range v.MapKeys() {
		enm[strings.ToLower(k.String())] = v.MapIndex(k).Interface()
	}

	p.enums[te] = enm
}
func RegisterInterface(id string, infmap interface{}, f func(in, out interface{})) {
	defaultParser.RegisterInterface(id, infmap, f)
}

func (p *Parser) RegisterInterface(id string, infmap interface{}, f func(in, out interface{})) {
	v := reflect.ValueOf(infmap)
	t := reflect.TypeOf(infmap)
	if t.Kind() != reflect.Map {
		panic("infmap must be a map")
	}

	// map key is the enum type
	ke := t.Key()
	if ke.PkgPath() == "" {
		panic("infmap key must be custom type")
	}
	if _, ok := p.enums[ke]; !ok {
		enmName := ke.PkgPath() + "." + ke.Name()
		panic(fmt.Sprintf("enum %s not registered", enmName))
	}

	m := map[interface{}]reflect.Type{}
	for _, k := range v.MapKeys() {
		m[k.Interface()] = v.MapIndex(k).Elem().Elem().Type()
	}

	p.ifaces[id] = &iface{
		m: m,
		f: f,
	}
}

var textUnmarshaler = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()

func (p *Parser) walkStruct(c *command, t reflect.Type, pth *path, pfx string, isArg bool, globals *strset.Set) {
	if isPtr(t) {
		t = t.Elem()
	}
	for i := 0; i < t.NumField(); i++ {

		// get field
		f := t.Field(i)
		fn := f.Name
		ft := f.Type

		// get and parse cli tag
		var tag *clitag
		tg := f.Tag.Get("cli")
		if tg == "-" {
			continue
		}
		tag = parseCliTag(tg)

		// compute arg name
		name := p.argCaseFunc(fn)
		if tag.long != "" {
			name = tag.long
		}
		if pfx != "" {
			name = pfx + "." + name
		}

		spth := pth.Subpath(fn)

		if isStruct(ft) && !ft.Implements(textUnmarshaler) {
			// embedded struct parse as args of parent
			if f.Anonymous {
				p.walkStruct(c, ft, spth, pfx, isArg, globals)
				continue
			}
			// we know is an arg so use the name as prefix
			if isArg {
				p.walkStruct(c, ft, spth, name, isArg, globals)
				continue
			}
			// is a ptr to struct but isArg in tag is set or
			// is normal struct so this is an arg
			if tag.isArg || !isPtr(ft) {
				p.walkStruct(c, ft, spth, name, true, globals)
				continue
			}
			// parse struct as a command
			cname := p.cmdCaseFunc(fn)
			if tag.cmd != "" {
				cname = tag.cmd
			}
			c.addSubcmd(cname, ft, spth)
			continue
		}

		// check for global args propagation collision
		if p.globalsEnabled && tag.global {
			if globals.Has(name) {
				panic("global args propagation collision")
			}
			globals.Add(name)
		}

		a := &argument{
			path:     spth,
			typ:      ft,
			long:     name,
			help:     f.Tag.Get("help"),
			required: tag.required,
		}
		c.args[name] = a

		if isInterface(ft) {
			if tag.iface == "" {
				panic("no iface name configured for " + ft.Name())
			}
			ifc, ok := p.ifaces[tag.iface]
			if !ok {
				panic("iface not registered: " + tag.iface)
			}
			fmt.Printf("%#v\n", ifc.m)
			continue
		}

		// get the underlaying type if pointer
		if isPtr(ft) {
			ft = ft.Elem()
		}

		if isArray(ft) {
			switch ft.Kind() {
			case reflect.Array:
				a.isArray = true
				a.arrayLen = ft.Len()
			case reflect.Slice:
				a.isSlice = true
				a.arrayLen = -1
			}
		}

		if isInt(ft) || isUint(ft) {
			if _, ok := p.enums[ft]; ok {
				a.enum = true
			}
		}
	}
}

func (p *Parser) validateFlag(flg string) (valid, short bool, arg string) {
	if flg == "-" {
		return false, false, ""
	}
	if len(flg) == 2 && flg[0] == '-' {
		return true, true, string(flg[1])
	}
	if flg[0] == '-' && flg[1] != '-' && p.strict {
		return false, false, ""
	}
	arg = strings.TrimLeft(flg, "-")
	return true, false, arg
}

func isFlag(s string) bool {
	return strings.HasPrefix(s, "-") && strings.TrimLeft(s, "-") != ""
}

func MarshalAny(in, out interface{}) {
	i, ok := in.(proto.Message)
	if !ok {
		panic("in not proto.Message")
	}
	o, ok := out.(*any.Any)
	if !ok {
		panic("out not *any.Any")
	}
	b, err := proto.Marshal(i)
	if err != nil {
		panic(err)
	}
	o.TypeUrl = proto.MessageName(i)
	o.Value = b
}
