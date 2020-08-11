package cli

import (
	"reflect"
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
	if s != "" {
		val := reflect.ValueOf(tag).Elem()
		parts := strings.Split(s, ",")
		for _, p := range parts {
			if i := strings.Index(p, "="); i != -1 {
				val.FieldByName(p[:i]).SetString(p[i+1:])
				continue
			}
			val.FieldByName(p).SetBool(true)
		}
	}
	return tag
}
