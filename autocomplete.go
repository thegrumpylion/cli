package cli

import (
	"io/ioutil"
	"path/filepath"
	"strings"
)

type Autocompleter interface {
	Autocomplete(val string) []string
}

var filesAutocompleter = func(val string) []string {
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
				name = name + "/"
			}
			out = append(out, name)
		}
	}
	return out
}

var hostsAutocompleter = func(val string) []string {
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
			out = append(out, parts[1])
		}
	}
	return out
}
