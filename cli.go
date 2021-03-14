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

var defaultCLI = NewCLI()

// CLI is the cli parser
type CLI struct {
	options     *cliOptions
	roots       []reflect.Value
	cmds        map[string]*command
	enums       map[reflect.Type]*enum
	helpOut     io.Writer
	errorOut    io.Writer
	completeOut io.Writer

	execList []interface{}
}

// NewCLI create new parser
func NewCLI(options ...Option) *CLI {
	cli := &CLI{
		cmds:  map[string]*command{},
		enums: map[reflect.Type]*enum{},
	}
	opts := &cliOptions{
		argCase:     CaseCamelLower,
		envCase:     CaseSnakeUpper,
		cmdCase:     CaseLower,
		argSplicer:  SplicerDot,
		envSplicer:  SplicerUnderscore,
		helpLong:    "--help",
		helpShort:   "-h",
		versionLong: "--version",
	}
	for _, o := range options {
		o(opts)
	}
	if opts.tags.Cli == "" {
		opts.tags.Cli = "cli"
	}
	if opts.tags.Help == "" {
		opts.tags.Help = "help"
	}
	if opts.tags.Default == "" {
		opts.tags.Default = "default"
	}
	if opts.tags.Complete == "" {
		opts.tags.Complete = "complete"
	}
	cli.options = opts
	cli.completeOut = os.Stdout
	return cli
}

// NewRootCommand add new root command to defaultCLI
func NewRootCommand(name string, arg interface{}) {
	defaultCLI.NewRootCommand(name, arg)
}

// NewRootCommand add new root command to this parser
func (cli *CLI) NewRootCommand(name string, arg interface{}) {
	t := reflect.TypeOf(arg)
	if t.Kind() != reflect.Ptr && t.Elem().Kind() != reflect.Struct {
		panic("not ptr to struct")
	}
	path := cli.addRoot(arg)
	c := &command{
		path:   path,
		name:   name,
		subcmd: map[string]*command{},
		flags:  newFlagSet(),
	}
	cli.cmds[name] = c
	cli.walkStruct(c, t, path, "", "", false, strset.New())
}

// Parse marshal string args to struct using the defaultCLI
func Parse(args []string) error {
	return defaultCLI.Parse(args)
}

func isCompletion() bool {
	_, lok := os.LookupEnv("COMP_LINE")
	_, pok := os.LookupEnv("COMP_POINT")
	return lok && pok
}

// Parse marshal string args to struct
func (cli *CLI) Parse(args []string) (err error) {

	p := newParser(cli)

	if err := p.Run(args); err != nil {
		return err
	}

	// check required
	for _, a := range p.curCmd.AllFlags() {
		if a.required && !a.isSet {
			return fmt.Errorf("required flag not set: %s", a.long)
		}
	}

	cli.execList = p.ExecList()

	return nil
}

// RegisterEnum resgister an enum map to the default parser
func RegisterEnum(enumMap interface{}) {
	defaultCLI.RegisterEnum(enumMap)
}

// RegisterEnum resgister an enum map. map must have string key and int/uint
// value. The value must also be a custom type e.g. type MyEnum uint32
func (cli *CLI) RegisterEnum(enumMap interface{}) {
	enm := newEnum(enumMap)
	cli.enums[enm.typ] = enm
}

func (cli *CLI) addRoot(in interface{}) *path {
	cli.roots = append(cli.roots, reflect.ValueOf(in))
	return &path{
		root: &cli.roots[len(cli.roots)-1],
	}
}

func (cli *CLI) isHelp(arg string) bool {
	return arg == cli.options.helpLong || arg == cli.options.helpShort
}

func (cli *CLI) isVersion(arg string) bool {
	return arg == cli.options.versionLong || arg == cli.options.versionShort
}

var textUnmarshaler = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()

func (cli *CLI) walkStruct(c *command, t reflect.Type, pth *path, pfx, envpfx string, isArg bool, globals *strset.Set) {
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
		tg := fld.Tag.Get(cli.options.tags.Cli)
		if tg == "-" {
			continue
		}
		tag = parseCliTag(tg)

		// compute arg name
		name := cli.options.argCase.Parse(fldName)
		if tag.long != "" {
			name = tag.long
		}
		if pfx != "" {
			name = cli.options.argSplicer.Splice(pfx, name)
		}

		// compute env var name
		env := cli.options.envCase.Parse(fldName)
		if tag.long != "" {
			env = tag.env
		}
		if envpfx != "" {
			env = cli.options.envSplicer.Splice(envpfx, env)
		}

		// create subpath for the current field
		spth := pth.Subpath(fldName)

		if isStruct(fldType) && !fldType.Implements(textUnmarshaler) {
			// embedded struct parse as args of parent
			if fld.Anonymous {
				cli.walkStruct(c, fldType, spth, pfx, envpfx, isArg, globals)
				continue
			}
			// we know is an arg so use the name as prefix
			if isArg {
				cli.walkStruct(c, fldType, spth, name, env, isArg, globals)
				continue
			}
			// is a ptr to struct but isArg in tag is set or
			// is normal struct so this is an arg
			if tag.isArg || !isPtr(fldType) {
				cli.walkStruct(c, fldType, spth, name, env, true, globals)
				continue
			}
			// parse struct as a command
			cname := cli.options.cmdCase.Parse(fldName)
			if tag.cmd != "" {
				cname = tag.cmd
			}
			sc := c.AddSubcommand(cname, spth)
			cli.walkStruct(sc, fldType, spth, "", "", false, globals.Copy())
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
		if cli.options.globalsEnabled {
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
			def:        fld.Tag.Get(cli.options.tags.Default),
			help:       fld.Tag.Get(cli.options.tags.Help),
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
			if enm, ok := cli.enums[fldType]; ok {
				a.enum = enm
			}
		}

		// completers
		if val, ok := fld.Tag.Lookup(cli.options.tags.Complete); ok {
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

func (cli *CLI) findRootCommand(name string) (*command, error) {
	c, ok := cli.cmds[name]
	if !ok {
		// try base path
		c, ok = cli.cmds[filepath.Base(name)]
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
