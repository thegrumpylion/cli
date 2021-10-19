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

func (c *command) AddArg(a *argument) bool {
	if a.positional {
		c.positionals = append(c.positionals, a)
		return true
	}
	return c.flags.Add(a)
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
		if sc.hidden {
			continue
		}
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
		if flg.def != nil {
			b.WriteString(" (default: ")
			b.WriteString(strings.Join(flg.def, " "))
			b.WriteByte(')')
		}
		if flg.env != "" {
			b.WriteString(" (env: ")
			b.WriteString(flg.env)
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
		if arg.def != nil {
			b.WriteString(" (default: ")
			b.WriteString(strings.Join(arg.def, " "))
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
	if err := t.Execute(w, tplContext{
		Cmd:   c,
		Ident: computeIdent(c.opts.identSize),
	}); err != nil {
		panic(err)
	}
}

type tplContext struct {
	Cmd   *command
	Ident string
}

var parentCmdTpl = `Usage:
{{.Ident}}{{.Cmd.Name}}{{range .Cmd.Flags}} {{.Usage}}{{end}} [command]
{{- if .Cmd.Description}}

{{.Cmd.Description}}
{{- end}}

Commands:
{{- range .Cmd.SubcmdDescription}}
{{$.Ident}}{{.}}
{{- end}}
{{- if .Cmd.FlagDescription}}

Flags:
{{- range .Cmd.FlagDescription}}
{{$.Ident}}{{.}}
{{- end}}
{{- end}}
`
var leafCmdTpl = `Usage:
{{.Ident}}{{.Cmd.Name}}{{range .Cmd.Flags}} {{.Usage}}{{end}}{{range .Cmd.Positionals}} {{.Usage}}{{end}}
{{- if .Cmd.Description}}

{{.Cmd.Description}}
{{- end}}
{{- if .Cmd.FlagDescription}}

Flags:
{{- range .Cmd.FlagDescription}}
{{$.Ident}}{{.}}
{{- end}}
{{- end}}
{{- if .Cmd.ArgumentDescription}}

Arguments:
{{- range .Cmd.ArgumentDescription}}
{{$.Ident}}{{.}}
{{- end}}
{{- end}}
`

func computeIdent(s uint) string {
	sb := strings.Builder{}
	for i := 0; i < int(s); i++ {
		sb.WriteByte(' ')
	}
	return sb.String()
}
