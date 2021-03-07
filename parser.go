package cli

import (
	"context"
	"encoding"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/scylladb/go-set/strset"
)

var defaultParser = NewParser()

// Parser is the cli parser
type Parser struct {
	tags           StructTags
	roots          []reflect.Value
	cmds           map[string]*command
	enums          map[reflect.Type]map[string]interface{}
	globalsEnabled bool
	argCase        Case
	envCase        Case
	cmdCase        Case
	argSplicer     Splicer
	envSplicer     Splicer
	helpLong       string
	helpShort      string
	versionLong    string
	versionShort   string
	strategy       OnErrorStrategy
	execList       []interface{}

	globals *flagSet
	curArg  *argument
	curCmd  *command
	curPos  int
	allPos  bool
}

// NewParser create new parser
func NewParser(opts ...ParserOption) *Parser {
	p := &Parser{
		cmds:        map[string]*command{},
		enums:       map[reflect.Type]map[string]interface{}{},
		argCase:     CaseCamelLower,
		envCase:     CaseSnakeUpper,
		cmdCase:     CaseLower,
		argSplicer:  SplicerDot,
		envSplicer:  SplicerUnderscore,
		helpLong:    "--help",
		helpShort:   "-h",
		versionLong: "--version",
	}
	for _, o := range opts {
		o(p)
	}
	if p.tags.Cli == "" {
		p.tags.Cli = "cli"
	}
	if p.tags.Help == "" {
		p.tags.Help = "help"
	}
	if p.tags.Default == "" {
		p.tags.Default = "default"
	}
	return p
}

// NewRootCommand add new root command to defaultParser
func NewRootCommand(name string, arg interface{}) {
	defaultParser.NewRootCommand(name, arg)
}

// NewRootCommand add new root command to this parser
func (p *Parser) NewRootCommand(name string, arg interface{}) {
	t := reflect.TypeOf(arg)
	if t.Kind() != reflect.Ptr && t.Elem().Kind() != reflect.Struct {
		panic("not ptr to struct")
	}
	path := p.addRoot(arg)
	c := &command{
		path:   path,
		name:   name,
		subcmd: map[string]*command{},
		flags:  newFlagSet(),
	}
	p.cmds[name] = c
	p.walkStruct(c, t, path, "", "", false, strset.New())
}

// Eval marshal string args to struct using the defaultParser
func Eval(args []string) error {
	return defaultParser.Eval(args)
}

// Eval marshal string args to struct
func (p *Parser) Eval(args []string) error {

	c, ok := p.cmds[args[0]]
	if !ok {
		// try base path
		c, ok = p.cmds[filepath.Base(args[0])]
		if !ok {
			return ErrCommandNotFound(args[0])
		}
	}
	p.curCmd = c

	// add root command to execution list
	p.execList = append(p.execList, c.path.Get())

	args = args[1:]

	// global flags
	if p.globalsEnabled {
		p.globals = newFlagSet()
		for _, a := range c.AllFlags() {
			if a.global {
				p.globals.Add(a)
			}
		}
	}

	state := p.entryState
	var err error

	for i := 0; i < len(args); i++ {

		arg := args[i]

		t := tokenType(arg)
		if p.allPos {
			t = VAL
		}

		state, err = state(arg, t)
		if err != nil {
			return err
		}
	}

	// check required
	for _, a := range c.AllFlags() {
		if a.required && !a.isSet {
			return fmt.Errorf("required flag not set: %s", a.long)
		}
	}

	return nil
}

// Execute the chain of commands in default parser
func Execute(ctx context.Context) error {
	return defaultParser.Execute(ctx)
}

// Execute the chain of commands
func (p *Parser) Execute(ctx context.Context) error {

	var err error
	lastCmd := len(p.execList) - 1
	pPostRunners := []PersistentPostRunner{}

	for i, inf := range p.execList {
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

// RegisterEnum resgister an enum map to the default parser
func RegisterEnum(enumMap interface{}) {
	defaultParser.RegisterEnum(enumMap)
}

// RegisterEnum resgister an enum map. map must have string key and int/uint
// value. The value must also be a custom type e.g. type MyEnum uint32
func (p *Parser) RegisterEnum(enumMap interface{}) {
	v := reflect.ValueOf(enumMap)
	t := reflect.TypeOf(enumMap)
	if t.Kind() != reflect.Map {
		panic("enumMap must be a map")
	}

	// key is the string of enum
	if t.Key().Kind() != reflect.String {
		panic("enumMap key must be string")
	}

	// element is enum int/uint custom type
	te := t.Elem()
	if te.PkgPath() == "" {
		panic("enumMap element must be custom type")
	}
	if !(isInt(te) || isUint(te)) {
		panic("enumMap element must be int/uint")
	}

	enm := map[string]interface{}{}
	for _, k := range v.MapKeys() {
		enm[strings.ToLower(k.String())] = v.MapIndex(k).Interface()
	}

	p.enums[te] = enm
}

func (p *Parser) addRoot(in interface{}) *path {
	p.roots = append(p.roots, reflect.ValueOf(in))
	return &path{
		root: &p.roots[len(p.roots)-1],
	}
}

func (p *Parser) isHelp(arg string) bool {
	return arg == p.helpLong || arg == p.helpShort
}

func (p *Parser) isVersion(arg string) bool {
	return arg == p.versionLong || arg == p.versionShort
}

var textUnmarshaler = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()

func (p *Parser) walkStruct(c *command, t reflect.Type, pth *path, pfx, envpfx string, isArg bool, globals *strset.Set) {
	if isPtr(t) {
		t = t.Elem()
	}
	for i := 0; i < t.NumField(); i++ {

		// get field
		f := t.Field(i)
		fldName := f.Name
		fldType := f.Type

		// get and parse cli tag
		var tag *clitag
		tg := f.Tag.Get(p.tags.Cli)
		if tg == "-" {
			continue
		}
		tag = parseCliTag(tg)

		// compute arg name
		name := p.argCase.Parse(fldName)
		if tag.long != "" {
			name = tag.long
		}
		if pfx != "" {
			name = p.argSplicer.Splice(pfx, name)
		}

		// compute env var name
		env := p.envCase.Parse(fldName)
		if tag.long != "" {
			env = tag.env
		}
		if envpfx != "" {
			env = p.envSplicer.Splice(envpfx, env)
		}

		// create subpath for the current field
		spth := pth.Subpath(fldName)

		if isStruct(fldType) && !fldType.Implements(textUnmarshaler) {
			// embedded struct parse as args of parent
			if f.Anonymous {
				p.walkStruct(c, fldType, spth, pfx, envpfx, isArg, globals)
				continue
			}
			// we know is an arg so use the name as prefix
			if isArg {
				p.walkStruct(c, fldType, spth, name, env, isArg, globals)
				continue
			}
			// is a ptr to struct but isArg in tag is set or
			// is normal struct so this is an arg
			if tag.isArg || !isPtr(fldType) {
				p.walkStruct(c, fldType, spth, name, env, true, globals)
				continue
			}
			// parse struct as a command
			cname := p.cmdCase.Parse(fldName)
			if tag.cmd != "" {
				cname = tag.cmd
			}
			sc := c.AddSubcommand(cname, spth)
			p.walkStruct(sc, fldType, spth, "", "", false, globals.Copy())
			continue
		}

		// generate long and short flags
		long := "--" + name
		short := ""
		if tag.short != "" {
			if len(tag.short) != 1 {
				panic("wrong short tag: " + tag.short)
			}
			short = "-" + tag.short
		}

		// check for global args propagation collision
		if p.globalsEnabled {
			if globals.Has(long) {
				panic("global args propagation collision: " + long)
			}
			if tag.global {
				globals.Add(long)
			}
			if short != "" {
				if globals.Has(short) {
					panic("global args propagation collision: " + short)
				}
				if tag.global {
					globals.Add(short)
				}
			}
		}

		// create arg and add to command
		a := &argument{
			path:       spth,
			typ:        fldType,
			long:       long,
			short:      short,
			env:        env,
			required:   tag.required,
			positional: tag.positional,
			global:     tag.global,
			def:        f.Tag.Get(p.tags.Default),
			help:       f.Tag.Get(p.tags.Help),
		}
		c.AddArg(a)

		// get the underlaying type if pointer
		if isPtr(fldType) {
			fldType = fldType.Elem()
		}

		if isArray(fldType) {
			switch fldType.Kind() {
			case reflect.Array:
				panic("array type not supported")
			case reflect.Slice:
				a.isSlice = true
			}
		}

		// check for enums
		if isInt(fldType) || isUint(fldType) {
			if _, ok := p.enums[fldType]; ok {
				a.enum = true
			}
		}
	}
}

func isFlag(s string) bool {
	if len(s) == 2 && s[0] == '-' && !strings.ContainsAny(s, "1234567890-") {
		return true
	}
	if len(s) > 2 && s[0] == '-' && s[1] == '-' {
		return true
	}
	return false
}

type Token int

const (
	VAL Token = iota
	FLAG
	COMPFLAG
	ALLPOS
)

type StateFunc func(s string, t Token) (StateFunc, error)

func (p *Parser) entryState(s string, t Token) (StateFunc, error) {
	switch t {
	case VAL:
		return p.valueOrCmdState(s, t)
	case FLAG:
		return p.flagState(s, t)
	case COMPFLAG:
		return p.compositFlagState(s, t)
	case ALLPOS:
		p.allPos = true
		return p.entryState, nil
	default:
		return nil, fmt.Errorf("unknown token: %d", t)
	}
}

func (p *Parser) valueOrCmdState(s string, t Token) (StateFunc, error) {
	if t != VAL {
		return nil, fmt.Errorf("unexpected token: %d at valueOrCmdState", t)
	}
	if p.curCmd.subcmd != nil {
		cc, ok := p.curCmd.subcmd[s]
		if !ok {
			return nil, ErrCommandNotFound(s)
		}
		p.curCmd = cc
		// handle globals
		if p.globalsEnabled {
			for _, a := range p.curCmd.AllFlags() {
				if a.global {
					p.globals.Add(a)
				}
			}
		}
		// add subcommand to execution list
		p.execList = append(p.execList, p.curCmd.path.Get())
		return p.entryState, nil
	}
	if p.curPos == len(p.curCmd.positionals) {
		return nil, fmt.Errorf("too many positional arguments")
	}
	a := p.curCmd.positionals[p.curPos]
	p.curPos++
	if a.isSlice {
		return p.sliceValueState(s, t)
	}
	return p.valueState(s, t)
}

func (p *Parser) valueState(s string, t Token) (StateFunc, error) {
	if t != VAL {
		return nil, fmt.Errorf("unexpected token: %d at valueState", t)
	}
	a := p.curArg
	if a.enum {
		em := p.enums[a.typ]
		if err := a.setValue(em[strings.ToLower(s)]); err != nil {
			return nil, err
		}
		return p.entryState, nil
	}
	if tum, ok := a.path.Get().(encoding.TextUnmarshaler); ok {
		if err := tum.UnmarshalText([]byte(s)); err != nil {
			return nil, err
		}
		return p.entryState, nil
	}
	if err := p.curArg.setScalarValue(s); err != nil {
		return nil, err
	}
	return p.entryState, nil
}
func (p *Parser) sliceValueState(s string, t Token) (StateFunc, error) {
	if t != VAL {
		return nil, fmt.Errorf("unexpected token: %d at sliceValueState", t)
	}
	a := p.curArg
	if err := a.append(s); err != nil {
		return nil, err
	}
	if a.separate {
		return p.entryState, nil
	}
	return p.sliceValueState, nil
}

func (p *Parser) flagState(s string, t Token) (StateFunc, error) {
	if t != FLAG {
		return nil, fmt.Errorf("unexpected token: %d at flagState", t)
	}
	if p.isHelp(s) {
		// handle help
	}
	if p.isVersion(s) {
		// handle version
	}
	a := p.curCmd.GetFlag(s)
	if a == nil {
		if p.globalsEnabled {
			if a = p.globals.Get(s); a == nil {
				return nil, ErrNoSuchFlag(s)
			}
		} else {
			return nil, ErrNoSuchFlag(s)
		}
	}
	p.curArg = a
	if a.isBool() {
		return p.valueState("true", VAL)
	}
	if a.isSlice {
		return p.sliceValueState, nil
	}
	return p.valueState, nil
}

func (p *Parser) compositFlagState(s string, t Token) (StateFunc, error) {
	if t != COMPFLAG {
		return nil, fmt.Errorf("unexpected token: %d at compositFlagState", t)
	}
	i := strings.Index(s, "=")
	flg := s[:i]
	val := s[i+1:]
	a := p.curCmd.GetFlag(flg)
	if a == nil {
		if p.globalsEnabled {
			fmt.Println("glob", p.globals.All())
			if a = p.globals.Get(s); a == nil {
				return nil, ErrNoSuchFlag(s)
			}
		} else {
			return nil, ErrNoSuchFlag(s)
		}
	}
	p.curArg = a
	if a.isSlice {
		if !a.separate {
			return nil, fmt.Errorf("slice flag must be separated to use composite flag")
		}
		return p.sliceValueState(s, VAL)
	}
	return p.valueState(val, VAL)
}

func tokenType(s string) Token {
	if isFlag(s) {
		if i := strings.Index(s, "="); i != -1 {
			return COMPFLAG
		}
		return FLAG
	}
	if s == "--" {
		return ALLPOS
	}
	return VAL
}
