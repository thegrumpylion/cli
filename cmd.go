package cli

import (
	"reflect"

	"github.com/scylladb/go-set/strset"
)

type command struct {
	path   *path
	parser *Parser
	parent *command
	name   string
	hidden bool
	args   map[string]*argument
	argsS  map[string]*argument
	subcmd map[string]*command
}

func (c *command) parse(arg interface{}) {
	t := reflect.TypeOf(arg)
	if t.Kind() != reflect.Ptr && t.Elem().Kind() != reflect.Struct {
		panic("not ptr to struct")
	}
	p := c.parser.addRoot(arg)
	c.path = p
	c.parser.walkStruct(c, t, p, "", false, strset.New())
}

func (c *command) addSubcmd(name string, t reflect.Type, p *path) {
	sc := &command{
		path:   p,
		parent: c,
		name:   name,
		subcmd: map[string]*command{},
		args:   map[string]*argument{},
	}
	c.parser.walkStruct(sc, t, p, "", false, strset.New())
	c.subcmd[name] = sc
}
