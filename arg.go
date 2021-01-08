package cli

import (
	"reflect"
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
	enum       bool
	iface      bool
	isSlice    bool
	isArray    bool
	arrayLen   int
	isSet      bool
}

func (a *argument) isBool() bool {
	return a.typ.Kind() == reflect.Bool
}

func (a *argument) setScalarValue(val string) error {
	return a.path.SetScalar(val)
}

func (a *argument) setArrayValue(arr []string) error {
	if a.isArray {
		return a.path.SetArray(arr)
	}
	return a.path.SetSlice(arr)
}

func (a *argument) setValue(val interface{}) error {
	return a.path.Set(val)
}
