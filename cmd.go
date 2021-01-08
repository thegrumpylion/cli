package cli

type command struct {
	path        *path
	parent      *command
	name        string
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
