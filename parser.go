package cnc

import (
	"encoding"
	"fmt"
	"io"
	"os"
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
	enums          map[reflect.Type]*enum
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
	helpOut        io.Writer
	errorOut       io.Writer
	completeOut    io.Writer

	execList []interface{}
	globals  *flagSet
	curArg   *argument
	curCmd   *command
	curPos   int
	allPos   bool
	isComp   bool
	isLast   bool
}

// NewParser create new parser
func NewParser(opts ...ParserOption) *Parser {
	p := &Parser{
		cmds:        map[string]*command{},
		enums:       map[reflect.Type]*enum{},
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
	if p.tags.Complete == "" {
		p.tags.Complete = "complete"
	}
	if p.globalsEnabled {
		p.globals = newFlagSet()
	}
	p.completeOut = os.Stdout
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

func isCompletion() bool {
	_, lok := os.LookupEnv("COMP_LINE")
	_, pok := os.LookupEnv("COMP_POINT")
	return lok && pok
}

// Eval marshal string args to struct
func (p *Parser) Eval(args []string) (err error) {

	sm := newStateMachine(p)

	if err := sm.Run(args); err != nil {
		return err
	}

	// check required
	for _, a := range sm.curCmd.AllFlags() {
		if a.required && !a.isSet {
			return fmt.Errorf("required flag not set: %s", a.long)
		}
	}

	p.execList = sm.execList

	return nil
}

// RegisterEnum resgister an enum map to the default parser
func RegisterEnum(enumMap interface{}) {
	defaultParser.RegisterEnum(enumMap)
}

// RegisterEnum resgister an enum map. map must have string key and int/uint
// value. The value must also be a custom type e.g. type MyEnum uint32
func (p *Parser) RegisterEnum(enumMap interface{}) {
	enm := newEnum(enumMap)
	p.enums[enm.typ] = enm
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
		fld := t.Field(i)
		fldName := fld.Name
		fldType := fld.Type

		// get and parse cli tag
		var tag *clitag
		tg := fld.Tag.Get(p.tags.Cli)
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
			if fld.Anonymous {
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
			def:        fld.Tag.Get(p.tags.Default),
			help:       fld.Tag.Get(p.tags.Help),
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
			if enm, ok := p.enums[fldType]; ok {
				a.enum = enm
			}
		}

		// completers
		if val, ok := fld.Tag.Lookup(p.tags.Complete); ok {
			for _, v := range strings.Split(val, ",") {
				cmp := getNamedCompleter((v))
				if cmp == nil {
					panic("no such completer: " + v)
				}
				a.completers = append(a.completers, cmp)
			}
		}
	}
}

func (p *Parser) findRootCommand(name string) (*command, error) {
	c, ok := p.cmds[name]
	if !ok {
		// try base path
		c, ok = p.cmds[filepath.Base(name)]
		if !ok {
			return nil, ErrCommandNotFound{name}
		}
	}
	return c, nil
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

func splitCompositeFlag(s string) (string, string) {
	i := strings.Index(s, "=")
	if i == -1 {
		return s, ""
	}
	flg := s[:i]
	if i == len(s)-1 {
		return flg, ""
	}
	return flg, s[i+1:]
}
