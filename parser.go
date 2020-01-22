package cli

import (
	"errors"
	"os"
	"reflect"
	"strings"
)

// ArgCase
type ArgCase uint32

const (
	CaseLower ArgCase = iota
	CaseCamel
	CaseCapital
)

type path struct {
	root *reflect.Value
	path []string
}

func (p *path) Subpath(name string) *path {
	return &path{
		root: p.root,
		path: append(p.path, name),
	}
}

func (p *path) Set(in reflect.Value) {
	v := *p.root
	for _, s := range p.path {
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				v.Set(reflect.New(v.Type().Elem()))
			}
			v = v.Elem()
		}
		v = v.FieldByName(s)
	}
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}
	v.Set(in)
}

type arg struct {
	path       *path
	typ        reflect.Type
	long       string
	short      string
	global     bool
	positional bool
	required   bool
	separate   bool
}

type command struct {
	parent *command
	roots  []reflect.Value
	name   string
	hidden bool
	args   map[string]*arg
	argsS  map[string]*arg
	subcmd map[string]*command
}

func NewCommand(name string, args ...interface{}) *command {
	c := &command{
		name:   name,
		subcmd: map[string]*command{},
		args:   map[string]*arg{},
	}
	c.Parse(args...)
	return c
}

func (c *command) Parse(args ...interface{}) {
	for _, a := range args {
		t := reflect.TypeOf(a)
		if t.Kind() != reflect.Ptr && t.Elem().Kind() != reflect.Struct {
			panic("not ptr to struct")
		}
		p := c.addRoot(a)
		walkStruct(c, t, p, "", false)
	}
}

func (c *command) AddArg(name string, in interface{}) {
	p := c.addRoot(in)
	c.addArg(name, reflect.TypeOf(in), p)
}

func (c *command) AddSubcmd(name string, in interface{}) *command {
	p := c.addRoot(in)
	c.addSubcmd(name, reflect.TypeOf(in), p)
	return c.subcmd[name]
}

func (c *command) addRoot(in interface{}) *path {
	c.roots = append(c.roots, reflect.ValueOf(in))
	return &path{
		root: &c.roots[len(c.roots)-1],
	}
}

func (c *command) addArg(name string, t reflect.Type, p *path) {
	a := &arg{
		path: p,
		typ:  t,
		long: name,
	}
	c.args[name] = a
}

func (c *command) addSubcmd(name string, t reflect.Type, p *path) {
	sc := &command{
		parent: c,
		name:   name,
		subcmd: map[string]*command{},
		args:   map[string]*arg{},
	}
	walkStruct(sc, t, p, "", false)
	c.subcmd[name] = sc
}

type Parser struct {
	roots  []reflect.Value
	enums  map[string]map[string]interface{}
	ifaces map[string]map[interface{}]interface{}
}

func (p *Parser) RegisterEnum(name string, enmap map[string]interface{}) {
	p.enums[name] = enmap
}

func (p *Parser) RegisterInterface(name string, enum string, infmap map[interface{}]interface{}) {

}

func Parse(in interface{}) (*Parser, error) {
	return ParseArgs(in, os.Args)
}

func ParseArgs(in interface{}, args []string) (*Parser, error) {
	v := reflect.ValueOf(in)
	if v.Type().Kind() != reflect.Ptr {
		return nil, errors.New("not a ptr")
	}
	v = v.Elem()
	if v.Type().Kind() != reflect.Struct {
		return nil, errors.New("not a struct")
	}
	return nil, nil
}

func walkStruct(c *command, t reflect.Type, pth *path, pfx string, isArg bool) {
	if isPtr(t) {
		t = t.Elem()
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fn := f.Name
		ft := f.Type
		var tag *clitag
		if tg, ok := f.Tag.Lookup("cli"); ok {
			if tg == "-" {
				continue
			}
			tag = parseCliTag(tg)
		} else {
			tag = parseCliTag("")
		}
		name := strings.ToLower(fn)
		if tag.long != "" {
			name = tag.long
		}
		if pfx != "" {
			name = pfx + "." + name
		}
		spth := pth.Subpath(fn)
		if isStruct(ft) {
			if f.Anonymous {
				walkStruct(c, ft, spth, pfx, isArg)
				continue
			}
			if isArg {
				walkStruct(c, ft, spth, name, isArg)
				continue
			}
			if tag.isArg || !isPtr(ft) {
				walkStruct(c, ft, spth, name, true)
				continue
			}
			c.addSubcmd(name, ft, spth)
			continue
		}
		c.addArg(name, ft, spth)
	}
}

func parseTag(s string) map[string]string {
	m := map[string]string{}
	if s != "" {
		parts := strings.Split(s, ",")
		for _, p := range parts {
			if i := strings.Index(p, "="); i != -1 {
				m[p[:i]] = p[i+1:]
				continue
			}
			m[p] = ""
		}
	}
	return m
}
