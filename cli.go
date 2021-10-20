package cli

import (
	"context"
	"encoding"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/kballard/go-shellquote"
	"github.com/scylladb/go-set/strset"
)

type Descriptioner interface {
	Description() string
}

type Helper interface {
	Help() string
}

type Versioner interface {
	Version() string
}

var defaultCLI = NewCLI()

// CLI holds the cli state and configration
type CLI struct {
	options     *cliOptions
	roots       []reflect.Value
	cmds        map[string]*command
	enums       map[reflect.Type]*enum
	helpOut     io.Writer
	errorOut    io.Writer
	completeOut io.Writer
	runList     []interface{}
	osExit      func(int)
}

// NewCLI create new CLI
func NewCLI(options ...Option) *CLI {
	cli := &CLI{
		cmds:   map[string]*command{},
		enums:  map[reflect.Type]*enum{},
		osExit: os.Exit,
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
	if opts.tags.Long == "" {
		opts.tags.Long = "long"
	}
	if opts.tags.Short == "" {
		opts.tags.Short = "short"
	}
	if opts.tags.Env == "" {
		opts.tags.Env = "env"
	}
	if opts.tags.Usage == "" {
		opts.tags.Usage = "usage"
	}
	if opts.tags.Default == "" {
		opts.tags.Default = "default"
	}
	if opts.tags.Complete == "" {
		opts.tags.Complete = "complete"
	}
	if !(opts.separator == SeparatorEquals || opts.separator == SeparatorSpace) {
		opts.separator = SeparatorSpace
	}
	if opts.cmdColSize == 0 {
		opts.cmdColSize = 13
	}
	if opts.flagColSize == 0 {
		opts.flagColSize = 23
	}
	if opts.identSize == 0 {
		opts.identSize = 4
	}
	cli.options = opts
	cli.completeOut = os.Stdout
	cli.helpOut = os.Stdout
	return cli
}

// ParseCommandAndRun combines ParseCommand & Run
func ParseCommandAndRun(ctx context.Context, cmd interface{}) error {
	if err := ParseCommand(cmd); err != nil {
		return err
	}
	return Run(ctx)
}

// ParseCommandAndRun combines ParseCommand & Run
func (cli *CLI) ParseCommandAndRun(ctx context.Context, cmd interface{}) error {
	if err := cli.ParseCommand(cmd); err != nil {
		return err
	}
	return cli.Run(ctx)
}

// ParseCommand creates a new root command from 1st OS arg
// and cmd and parses os.Args as input on default CLI
func ParseCommand(cmd interface{}) error {
	NewCommand(filepath.Base(os.Args[0]), cmd)
	return Parse(os.Args)
}

// ParseCommand creates a new root command from 1st OS arg
// and cmd and parses os.Args as input
func (cli *CLI) ParseCommand(cmd interface{}) error {
	cli.NewCommand(filepath.Base(os.Args[0]), cmd)
	return cli.Parse(os.Args)
}

// NewCommand add new root command to defaultCLI
func NewCommand(name string, cmd interface{}) {
	defaultCLI.NewCommand(name, cmd)
}

// NewCommand add new root command to this CLI
func (cli *CLI) NewCommand(name string, cmd interface{}) {
	t := reflect.TypeOf(cmd)
	if t.Kind() != reflect.Ptr && t.Elem().Kind() != reflect.Struct {
		panic("not ptr to struct")
	}
	path := cli.addRoot(cmd)
	c := &command{
		path:       path,
		Name:       name,
		subcmdsMap: map[string]*command{},
		flags:      newFlagSet(),
		opts:       cli.options,
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

	// check for required and set default value
	for _, a := range p.currentCmd().Flags() {
		if err := a.SetEnv(); err != nil {
			return err
		}
		if !a.IsSet() {
			if a.required {
				return fmt.Errorf("required flag not set: %s", a.long)
			}
			if err := a.SetDefaultValue(); err != nil {
				panic("failed to set default value for flag: " + a.long)
			}
		}
	}
	for _, a := range p.currentCmd().Positionals() {
		if a.required && !a.IsSet() {
			return fmt.Errorf("required argument not set: %s", a.placeholder)
		}
	}

	cli.runList = p.RunList()

	return nil
}

// RegisterEnum resgister an enum map to the default CLI
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

func (cli *CLI) walkStruct(
	c *command,
	t reflect.Type,
	pth *path,
	pfx, envpfx string,
	isArg bool,
	globals *strset.Set,
) {
	if isPtr(t) {
		t = t.Elem()
	}
	for i := 0; i < t.NumField(); i++ {

		// get field
		fld := t.Field(i)

		// check if unexported
		if !fld.IsExported() {
			continue
		}

		fldName := fld.Name
		fldType := fld.Type

		// parse tags
		tags := cli.options.tags.parseTags(fld.Tag)

		if tags.IsIgnored() {
			continue
		}

		// compute arg name, TODO: optimize
		name := cli.options.argCase.Parse(fldName)
		if tags.LongIsIgnored() {
			name = ""
		}
		if tags.Long.name != "" {
			name = tags.Long.name
		}
		if pfx != "" && !tags.LongIsIgnored() {
			name = cli.options.argSplicer.Splice(pfx, name)
		}

		// compute env var name, TODO: optimize
		env := cli.options.envCase.Parse(fldName)
		if tags.EnvIsIgnored() {
			env = ""
		}
		if tags.Env.name != "" {
			env = tags.Env.name
		}
		if envpfx != "" && !tags.EnvIsIgnored() {
			env = cli.options.envSplicer.Splice(envpfx, env)
		}

		// create subpath for the current field
		spth := pth.Subpath(fldName)

		// is struct and does not have custom unmarshaler
		if isStruct(fldType) && !fldType.Implements(textUnmarshaler) {
			// is an embedded struct, parse as args of parent
			if fld.Anonymous {
				cli.walkStruct(c, fldType, spth, pfx, envpfx, isArg, globals)
				continue
			}
			// we know is an arg so use the name as prefix
			if isArg {
				cli.walkStruct(c, fldType, spth, name, env, isArg, globals)
				continue
			}
			// is a ptr to struct but nocmd in tag is set or is a normal struct then this is an arg
			if tags.CmdIsIgnored() || !isPtr(fldType) {
				cli.walkStruct(c, fldType, spth, name, env, true, globals)
				continue
			}
			// parse struct as a command
			cname := cli.options.cmdCase.Parse(fldName)
			if tags.Cmd != "" {
				cname = tags.Cmd
			}
			// add subcommand to the current command
			sc := c.AddSubcommand(cname, spth, fld.Tag.Get(cli.options.tags.Usage))
			// down the rabbit hole we go
			cli.walkStruct(sc, fldType, spth, "", "", false, globals.Copy())
			continue
		}

		// check for explicit long flag name
		if tags.Long.explicit {
			name = tags.Long.name
		}

		// check for explicit env var name
		if tags.Env.explicit {
			env = tags.Env.name
		}

		// generate long and short flags
		long := "--" + name
		short := ""
		if tags.Short != "" {
			if len(tags.Short) != 1 {
				panic("wrong short tag: " + tags.Short)
			}
			short = "-" + tags.Short
		}

		// check for global args propagation collision
		if cli.options.globalsEnabled {
			if globals.Has(long) {
				panic("global args propagation collision: " + long)
			}
			if tags.Cli.global {
				globals.Add(long)
			}
			if short != "" {
				if globals.Has(short) {
					panic("global args propagation collision: " + short)
				}
				if tags.Cli.global {
					globals.Add(short)
				}
			}
		}

		// create arg and add to command
		a := &argument{
			opts:        cli.options,
			path:        spth,
			typ:         fldType,
			long:        long,
			short:       short,
			env:         env,
			required:    tags.Cli.required,
			positional:  tags.Cli.positional,
			global:      tags.Cli.global,
			help:        fld.Tag.Get(cli.options.tags.Usage),
			placeholder: strings.ToUpper(name),
		}

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

		// default value
		if def := fld.Tag.Get(cli.options.tags.Default); def != "" {
			defval := []string{def}
			if a.isSlice {
				words, err := shellquote.Split(def)
				if err != nil {
					panic("default value for array/slice cannot br parsed: " + err.Error())
				}
				defval = words
			}
			a.def = defval
		}

		// add the argument to the current command
		if added := c.AddArg(a); !added {
			panic(fmt.Sprintf("flag name already added for command: %s long: %s short: %s", c.Name, a.long, a.short))
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
	if len(s) == 2 && s[0] == '-' && !strings.ContainsAny(string(s[1]), "1234567890-") {
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
