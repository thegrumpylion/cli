package cli

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/kballard/go-shellquote"
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
	path        *path
	typ         reflect.Type
	def         []string
	long        string
	short       string
	env         string
	help        string
	placeholder string
	global      bool
	positional  bool
	required    bool
	enum        *enum
	isSlice     bool
	isSet       bool
	completers  []Completer
	opts        *cliOptions
}

func (a *argument) IsBool() bool {
	return a.typ.Kind() == reflect.Bool
}

func (a *argument) IsSet() bool {
	return a.isSet
}

func (a *argument) Reset() {
	a.isSet = false
}

func (a *argument) SetValue(val string) error {
	a.isSet = true
	if a.enum != nil {
		return a.path.Set(a.enum.Value(val))
	}
	return a.path.SetScalar(val)
}

func (a *argument) Append(s string) error {
	if a.isSlice {
		a.isSet = true
		return a.path.AppendToSlice(s)
	}
	return fmt.Errorf("not an array or a slice")
}

func (a *argument) SetEnv() error {
	if a.isSet {
		return nil
	}
	val, ok := os.LookupEnv(a.env)
	if !ok {
		return nil
	}
	if a.isSlice {
		words, err := shellquote.Split(val)
		if err != nil {
			return err
		}
		for _, s := range words {
			if err := a.Append(s); err != nil {
				return err
			}
		}
		return nil
	}
	return a.SetValue(val)
}

func (a *argument) SetDefaultValue() error {
	if a.def == nil {
		return nil
	}
	if a.isSlice {
		for _, s := range a.def {
			if err := a.Append(s); err != nil {
				return err
			}
		}
		return nil
	}
	return a.SetValue(a.def[0])
}

func (a *argument) Complete(val string) (out []string) {
	if a.enum != nil {
		return a.enum.Complete(val)
	}
	for _, f := range a.completers {
		out = append(out, f.Complete(val)...)
	}
	sort.Strings(out)
	return
}

func (a *argument) Usage() string {
	if a.positional {
		if a.required {
			return a.placeholder
		}
		return fmt.Sprintf("[%s]", a.placeholder)
	}
	b := strings.Builder{}
	if !a.required {
		b.WriteByte('[')
	}
	if a.short != "" {
		b.WriteString(a.short)
		b.WriteByte('|')
	}
	b.WriteString(a.long)
	if !a.IsBool() {
		b.WriteByte(byte(a.opts.separator))
		b.WriteString(a.placeholder)
	}
	if !a.required {
		b.WriteByte(']')
	}
	return b.String()
}
