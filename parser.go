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

type command struct {
	parent *command
	name   string
	hidden bool
	args   map[string]*arg
	argsS  map[string]*arg
	subcmd map[string]*command
}

func (c *command) Parse(args ...interface{}) {

}

func (c *command) AddArg(long, short, help string) {

}

func (c *command) AddSubcmd(name string, in interface{}) *command {

	return nil
}

type arg struct {
	path  path
	typ   reflect.Type
	long  string
	short string
	tag   *clitag
}

type path struct {
	root interface{}
	path []string
}

type Parser struct {
	roots map[int]interface{}
	enums map[string]map[string]interface{}
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

func walkStruct(t reflect.Type, pfx string, isArg bool, fanc func(typ reflect.Type, name string, tag *clitag)) {
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
		if isStruct(ft) {
			if f.Anonymous {
				walkStruct(ft, pfx, isArg, fanc)
				continue
			}
			if isArg {
				walkStruct(ft, name, isArg, fanc)
				continue
			}
			if tag.isArg || !isPtr(ft) {
				walkStruct(ft, name, true, fanc)
				continue
			}
		}
		fanc(ft, name, tag)
	}
}

func parseCmd(t reflect.Type, name string, par *command, acase ArgCase) *command {
	c := &command{
		parent: par,
		name:   name,
		subcmd: map[string]*command{},
		args:   map[string]*arg{},
	}
	walkStruct(t, "", false, func(typ reflect.Type, name string, tag *clitag) {
		if isStruct(typ) {
			c.subcmd[name] = parseCmd(typ, name, c, acase)
			return
		}
		arg := &arg{
			typ:  typ,
			long: name,
			tag:  tag,
		}
		if tag.short != "" {
			arg.short = tag.short
		}
		c.args[name] = arg
	})
	return c
}
