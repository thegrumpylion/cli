package cli

import "reflect"

type command struct {
	parser *Parser
	parent *command
	name   string
	hidden bool
	args   map[string]*argument
	argsS  map[string]*argument
	subcmd map[string]*command
}

func (c *command) Parse(args ...interface{}) {
	for _, a := range args {
		t := reflect.TypeOf(a)
		if t.Kind() != reflect.Ptr && t.Elem().Kind() != reflect.Struct {
			panic("not ptr to struct")
		}
		p := c.parser.addRoot(a)
		c.parser.walkStruct(c, t, p, "", false)
	}
}

func (c *command) AddArg(name string, in interface{}, help string) {
	p := c.parser.addRoot(in)
	a := &argument{
		path: p,
		typ:  reflect.TypeOf(in),
		long: name,
		help: help,
	}
	c.args[name] = a
}

func (c *command) AddSubcmd(name string, in interface{}) *command {
	p := c.parser.addRoot(in)
	c.addSubcmd(name, reflect.TypeOf(in), p)
	return c.subcmd[name]
}

func (c *command) addSubcmd(name string, t reflect.Type, p *path) {
	sc := &command{
		parent: c,
		name:   name,
		subcmd: map[string]*command{},
		args:   map[string]*argument{},
	}
	c.parser.walkStruct(sc, t, p, "", false)
	c.subcmd[name] = sc
}
