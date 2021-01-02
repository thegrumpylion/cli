package cli

import (
	"strings"
)

type clitag struct {
	long       string
	short      string
	required   bool
	positional bool
	isArg      bool
	global     bool
	csv        bool
	embed      bool
	enum       string
	iface      string
	flag       string
}

func parseCliTag(s string) *clitag {
	tag := &clitag{}
	parts := strings.Split(s, ",")
	for _, p := range parts {
		if isFlag(p) {
			if strings.HasPrefix(p, "--") {
				tag.long = p
				continue
			}
			tag.short = p
			continue
		}
		var key, val string
		key = p
		if i := strings.Index(p, ":"); i != -1 {
			key = p[:i]
			val = p[i+1:]
		}
		switch strings.ToLower(key) {
		case "required":
			tag.required = true
		case "positional":
			tag.positional = true
		case "global":
			tag.global = true
		case "embed":
			tag.embed = true
		case "iface":
			tag.iface = val
		}
	}
	return tag
}
