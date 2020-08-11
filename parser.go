package cli

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/any"
)

// ArgCase
type ArgCase uint32

const (
	CaseLower ArgCase = iota
	CaseCamel
	CaseCapital
)

var ErrCommandNotFound = func(cmd string) error { return fmt.Errorf("command not found: %s", cmd) }
var ErrNoSuchFlag = func(flg string) error { return fmt.Errorf("no such flag: %s", flg) }
var ErrInvalidFlag = func(flg string) error { return fmt.Errorf("invalid flag: %s", flg) }
var ErrInvalidValue = func(val, flg string) error { return fmt.Errorf("invalid value: %s for flag: %s", val, flg) }

var defaultParser = NewParser()

type iface struct {
	m map[interface{}]reflect.Type
	f func(in, out interface{})
}

type Parser struct {
	roots  []reflect.Value
	cmds   map[string]*command
	enums  map[reflect.Type]map[string]interface{}
	ifaces map[string]*iface
	strict bool
}

func NewParser() *Parser {
	return &Parser{
		cmds:   map[string]*command{},
		enums:  map[reflect.Type]map[string]interface{}{},
		ifaces: map[string]*iface{},
	}
}

func (p *Parser) addRoot(in interface{}) *path {
	p.roots = append(p.roots, reflect.ValueOf(in))
	return &path{
		root: &p.roots[len(p.roots)-1],
	}
}

func NewRootCommand(name string, args ...interface{}) *command {
	return defaultParser.NewRootCommand(name, args...)
}

func (p *Parser) NewRootCommand(name string, args ...interface{}) *command {
	c := &command{
		parser: p,
		name:   name,
		subcmd: map[string]*command{},
		args:   map[string]*argument{},
	}
	c.Parse(args...)
	p.cmds[name] = c
	return c
}

func Eval(args []string) error {
	return defaultParser.Eval(args)
}

func (p *Parser) Eval(args []string) error {

	currentCmdArgs := map[*argument]struct{}{}
	arrays := map[*argument][]string{}

	c, ok := p.cmds[args[0]]
	// try base path
	if !ok {
		c, ok = p.cmds[filepath.Base(args[0])]
		if !ok {
			return ErrCommandNotFound(args[0])
		}
	}

	for _, v := range c.args {
		currentCmdArgs[v] = struct{}{}
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
			}
		}

		if arg == "-h" || arg == "--help" {
			// handle help
		}

		if arg == "--version" {
			// handle version
		}

		val := ""
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

		delete(currentCmdArgs, a)

		if a.enum {
			if val == "" {
				val = args[i+1]
			}
			em := p.enums[a.typ]
			a.path.Set(em[strings.ToLower(val)])
			i++
			continue
		}

		if isArr, l := a.isArray(); isArr {
			if a.separate {
				if _, ok := arrays[a]; !ok {
					arrays[a] = []string{}
				}
				// if is array and overflows
				if l > 0 && len(arrays[a]) == l {
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
			if l > 0 {
				for j := 0; j < l; j++ {
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

		if !a.isBool() {
			if val == "" {
				val = args[i+1]
				i++
			}
		}

		if err := a.setScalarValue(val); err != nil {
			return err
		}
	}

	return nil
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

func (p *Parser) walkStruct(c *command, t reflect.Type, pth *path, pfx string, isArg bool) {
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
		if tg, ok := f.Tag.Lookup("cli"); ok {
			if tg == "-" {
				continue
			}
			tag = parseCliTag(tg)
		} else {
			tag = parseCliTag("")
		}

		// compute arg name
		name := strings.ToLower(fn)
		if tag.long != "" {
			name = tag.long
		}
		if pfx != "" {
			name = pfx + "." + name
		}

		spth := pth.Subpath(fn)

		if isStruct(ft) {
			// embedded struct parse as args of parent
			if f.Anonymous {
				p.walkStruct(c, ft, spth, pfx, isArg)
				continue
			}
			// we know is an arg so use the name as prefix
			if isArg {
				p.walkStruct(c, ft, spth, name, isArg)
				continue
			}
			// is a ptr to struct but isArg in tag is set or
			// is normal struct so this is an arg
			if tag.isArg || !isPtr(ft) {
				p.walkStruct(c, ft, spth, name, true)
				continue
			}
			// parse struct as a command
			c.addSubcmd(name, ft, spth)
			continue
		}

		a := &argument{
			path:     spth,
			typ:      ft,
			long:     name,
			help:     f.Tag.Get("help"),
			required: tag.required,
		}

		if isInt(ft) || isUint(ft) {
			if _, ok := p.enums[ft]; ok {
				a.enum = true
			}
		}

		c.args[name] = a
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
	arg = strings.ToLower(arg)
	return true, false, arg
}

// isFlag returns true if a token is a flag such as "-v" or "--user" but not "-" or "--"
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
