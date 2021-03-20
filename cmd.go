package cli

import (
	"html/template"
	"io"
	"strings"
)

type command struct {
	Name        string
	path        *path
	parent      *command
	help        string
	description string
	group       string
	hidden      bool
	flags       *flagSet
	positionals []*argument
	subcmdsMap  map[string]*command
	opts        *cliOptions
	subcmds     []*command
}

func (c *command) self() interface{} {
	return c.path.Get()
}

func (c *command) AddArg(a *argument) {
	if a.positional {
		c.positionals = append(c.positionals, a)
		return
	}
	c.flags.Add(a)
}

func (c *command) GetFlag(n string) *argument {
	return c.flags.Get(n)
}

func (c *command) Positionals() []*argument {
	return c.positionals
}

func (c *command) Flags() []*argument {
	return c.flags.All()
}

func (c *command) HasSubcommands() bool {
	return len(c.subcmds) != 0
}

func (c *command) AddSubcommand(name string, p *path, help string) *command {
	sc := &command{
		path:       p,
		parent:     c,
		Name:       name,
		subcmdsMap: map[string]*command{},
		flags:      newFlagSet(),
		opts:       c.opts,
		help:       help,
	}
	c.subcmdsMap[name] = sc
	c.subcmds = append(c.subcmds, sc)
	return sc
}

func (c *command) LookupSubcommand(name string) (sc *command, ok bool) {
	sc, ok = c.subcmdsMap[name]
	return
}

func (c *command) Description() string {
	if desc, ok := c.self().(Descriptioner); ok {
		return desc.Description()
	}
	return c.help
}

func (c *command) SubcmdDescription() (out []string) {
	for _, sc := range c.subcmds {
		b := strings.Builder{}
		b.WriteString(sc.Name)
		l := int(c.opts.cmdColSize)
		if len(sc.Name) >= l {
			b.WriteString("\n  ")
		} else {
			l -= len(sc.Name)
		}
		for i := 0; i < l; i++ {
			b.WriteByte(' ')
		}
		b.WriteString(sc.help)
		out = append(out, b.String())
	}
	return
}

func (c *command) FlagDescription() (out []string) {
	for _, flg := range c.Flags() {
		b := strings.Builder{}
		if flg.short != "" {
			b.WriteString(flg.short)
			b.WriteString(", ")
		} else {
			b.WriteString("    ")
		}
		b.WriteString(flg.long)
		l := int(c.opts.flagColSize)
		if len(flg.long) >= l-4 {
			b.WriteString("\n  ")
		} else {
			l -= len(flg.long) + 4
		}
		for i := 0; i < l; i++ {
			b.WriteByte(' ')
		}
		b.WriteString(flg.help)
		if flg.def != "" {
			b.WriteString(" (default: ")
			b.WriteString(flg.def)
			b.WriteByte(')')
		}
		if flg.required {
			b.WriteString(" (required)")
		}
		out = append(out, b.String())
	}
	return
}

func (c *command) ArgumentDescription() (out []string) {
	for _, arg := range c.Positionals() {
		b := strings.Builder{}
		b.WriteString(arg.placeholder)
		l := int(c.opts.cmdColSize)
		if len(arg.placeholder) >= l {
			b.WriteString("\n  ")
		} else {
			l -= len(arg.placeholder)
		}
		for i := 0; i < l; i++ {
			b.WriteByte(' ')
		}
		b.WriteString(arg.help)
		if arg.def != "" {
			b.WriteString(" (default: ")
			b.WriteString(arg.def)
			b.WriteByte(')')
		}
		if arg.required {
			b.WriteString(" (required)")
		}
		out = append(out, b.String())
	}
	return
}

func (c *command) CompleteFlags(val string) (out []string) {
	for _, v := range c.Flags() {
		if strings.HasPrefix(v.long, val) {
			if v.IsSet() && !v.isSlice {
				continue
			}
			o := v.long + string(c.opts.separator)
			if v.IsBool() {
				o = v.long + " "
			}
			out = append(out, o)
		}
	}
	return
}

func (c *command) CompleteSubcommands(val string) (out []string) {
	for sc := range c.subcmdsMap {
		if strings.HasPrefix(sc, val) {
			out = append(out, sc+" ")
		}
	}
	return
}

func (c *command) Usage(w io.Writer) {
	var t *template.Template
	if c.HasSubcommands() {
		t = template.Must(template.New("").Parse(parentCmdTpl))
	} else {
		t = template.Must(template.New("").Parse(leafCmdTpl))
	}
	if err := t.Execute(w, c); err != nil {
		panic(err)
	}
}

var parentCmdTpl = `Usage:
  {{.Name}}{{range .Flags}} {{.Usage}}{{end}} [command]
{{- if .Description}}

{{.Description}}
{{- end}}

Commands:
{{- range .SubcmdDescription}}
  {{.}}
{{- end}}
{{- if .FlagDescription}}

Flags:
{{- range .FlagDescription}}
  {{.}}
{{- end}}
{{- end}}
`
var leafCmdTpl = `Usage:
  {{.Name}}{{range .Flags}} {{.Usage}}{{end}}{{range .Positionals}} {{.Usage}}{{end}}
{{- if .Description}}

{{.Description}}
{{- end}}
{{- if .FlagDescription}}

Flags:
{{- range .FlagDescription}}
  {{.}}
{{- end}}
{{- end}}
{{- if .ArgumentDescription}}

Arguments:
{{- range .ArgumentDescription}}
  {{.}}
{{- end}}
{{- end}}
`
