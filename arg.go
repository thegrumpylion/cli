package cnc

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

func newFlagSet() *flagSet {
	return &flagSet{
		long:  map[string]*argument{},
		short: map[string]*argument{},
	}
}

type flagSet struct {
	long  map[string]*argument
	short map[string]*argument
	all   []*argument
}

func (fs *flagSet) Add(a *argument) bool {
	if _, ok := fs.long[a.long]; ok {
		return false
	}
	fs.long[a.long] = a
	if a.short != "" {
		if _, ok := fs.short[a.short]; ok {
			return false
		}
		fs.short[a.short] = a
	}
	fs.all = append(fs.all, a)
	return true
}

func (fs *flagSet) Get(n string) *argument {
	if a, ok := fs.long[n]; ok {
		return a
	}
	return fs.short[n]
}

func (fs *flagSet) All() []*argument {
	return fs.all
}

func (fs *flagSet) Autocomplete(val string) []string {
	flags := []string{}
	for _, f := range fs.all {
		if strings.HasPrefix(f.long, val) {
			flags = append(flags, f.long)
		}
		if strings.HasPrefix(f.short, val) {
			flags = append(flags, f.short)
		}
	}
	sort.Strings(flags)
	return flags
}

type argument struct {
	path       *path
	typ        reflect.Type
	def        string
	long       string
	short      string
	env        string
	help       string
	global     bool
	positional bool
	required   bool
	separate   bool
	enum       *enum
	iface      bool
	isSlice    bool
	isSet      bool
	completers []func(val string) []string
}

func (a *argument) IsBool() bool {
	return a.typ.Kind() == reflect.Bool
}

func (a *argument) SetScalarValue(val string) error {
	return a.path.SetScalar(val)
}

func (a *argument) Append(s string) error {
	if a.isSlice {
		return a.path.AppendToSlice(s)
	}
	return fmt.Errorf("not an array or a slice")
}

func (a *argument) SetValue(val interface{}) error {
	return a.path.Set(val)
}

func (a *argument) Complete(val string) (out []string) {
	for _, f := range a.completers {
		out = append(out, f(val)...)
	}
	if a.enum != nil {
		out = append(out, a.enum.Complete(val)...)
	}
	sort.Strings(out)
	return
}
