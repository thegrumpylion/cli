package cnc

import (
	"io/ioutil"
	"path/filepath"
	"strings"
)

// Completer interface
type Completer interface {
	// Complete returns suggestions filtered by val
	Complete(val string) []string
}

// FuncCompleter creates a Completer from a func of the same signature
type FuncCompleter struct {
	f func(val string) []string
}

func (fc *FuncCompleter) Complete(val string) []string {
	return fc.f(val)
}

// NewFuncCmpleter instantiates a new FuncCompleter from f
func NewFuncCmpleter(f func(val string) []string) *FuncCompleter {
	return &FuncCompleter{f}
}

var namedCompleteres = map[string]Completer{
	"files": NewFuncCmpleter(filesCompleter),
	"hosts": NewFuncCmpleter(hostsCompleter),
}

// RegisterNamedCompleter adds named completers to be accessed by struct tag `complete:""`
func RegisterNamedCompleter(name string, comp Completer) {
	namedCompleteres[name] = comp
}

func getNamedCompleter(name string) Completer {
	return namedCompleteres[name]
}

var filesCompleter = func(val string) []string {
	d, f := filepath.Split(val)
	if d == "" {
		d = "."
	}
	files, err := ioutil.ReadDir(d)
	if err != nil {
		return nil
	}
	out := []string{}
	for _, fl := range files {
		name := fl.Name()
		if strings.HasPrefix(name, f) {
			name = filepath.Join(d, name)
			if fl.IsDir() {
				name += "/"
			} else {
				name += " "
			}
			out = append(out, name)
		}
	}
	return out
}

var hostsCompleter = func(val string) []string {
	data, err := ioutil.ReadFile("/etc/hosts")
	if err != nil {
		return nil
	}
	out := []string{}
	for _, line := range strings.Split(strings.Trim(string(data), " \t\r\n"), "\n") {
		line = strings.Replace(strings.Trim(line, " \t"), "\t", " ", -1)
		if len(line) == 0 || line[0] == ';' || line[0] == '#' {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) > 1 && len(parts[1]) > 0 && strings.HasPrefix(parts[1], val) {
			out = append(out, parts[1]+" ")
		}
	}
	return out
}
