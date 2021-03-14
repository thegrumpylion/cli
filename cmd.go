package cnc

import (
	"html/template"
	"io"
	"strings"
)

type command struct {
	Name        string
	path        *path
	parent      *command
	group       string
	hidden      bool
	flags       *flagSet
	positionals []*argument
	subcmd      map[string]*command
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

func (c *command) AllFlags() []*argument {
	return c.flags.All()
}

func (c *command) AddSubcommand(name string, p *path) *command {
	sc := &command{
		path:   p,
		parent: c,
		Name:   name,
		subcmd: map[string]*command{},
		flags:  newFlagSet(),
	}
	c.subcmd[name] = sc
	return sc
}

func (c *command) LookupSubcommand(name string) (sc *command, ok bool) {
	sc, ok = c.subcmd[name]
	return
}

func (c *command) CompleteFlags(val string) (out []string) {
	for _, v := range c.AllFlags() {
		if strings.HasPrefix(v.long, val) {
			out = append(out, v.long)
		}
	}
	return
}

func (c *command) CompleteSubcommands(val string) (out []string) {
	for sc := range c.subcmd {
		if strings.HasPrefix(sc, val) {
			out = append(out, sc)
		}
	}
	return
}

func (c *command) Usage(w io.Writer) {
	var t *template.Template
	if c.subcmd != nil {
		t = template.Must(template.New("").Parse(parentCmdTpl))
	} else {
		t = template.Must(template.New("").Parse(leafCmdTpl))
	}
	if err := t.Execute(w, c); err != nil {
		panic(err)
	}
}

var parentCmdTpl = `Usage:
  {{.Name}}{{range .AllFlags}} {{.Usage}}{{end}}

Commands:

`
var leafCmdTpl = `
Usage:
  {{.Name}}{{range .AllFlags}} {{.Usage}}{{end}}

Arguments:
`
