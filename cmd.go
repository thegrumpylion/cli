package cnc

import (
	"strings"
)

type command struct {
	path        *path
	parent      *command
	name        string
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
		name:   name,
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

var parentCmdTpl = `Usage:
  {{.Name}}{{}}
`
