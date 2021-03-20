package cli

import (
	"strings"
)

// StructTags struct tags values
type StructTags struct {
	Cli      string
	Default  string
	Help     string
	Complete string
}

type clitag struct {
	long       string
	short      string
	env        string
	required   bool
	positional bool
	noCmd      bool
	cmd        string
	global     bool
}

func parseCliTag(s string) *clitag {
	tag := &clitag{}
	parts := strings.Split(s, ",")
	for _, p := range parts {
		if isFlag(p) {
			if strings.HasPrefix(p, "--") {
				tag.long = strings.TrimPrefix(p, "--")
				continue
			}
			tag.short = strings.TrimPrefix(p, "-")
			continue
		}
		var key, val string
		key = p
		if i := strings.Index(p, ":"); i != -1 {
			key = p[:i]
			val = p[i+1:]
		}
		switch strings.ToLower(key) {
		case "env":
			tag.env = val
		case "required":
			tag.required = true
		case "positional":
			tag.positional = true
		case "global":
			tag.global = true
		case "nocmd":
			tag.noCmd = true
		case "cmd":
			tag.cmd = val
		}
	}
	return tag
}
