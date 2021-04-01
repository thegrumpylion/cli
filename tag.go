package cli

import (
	"reflect"
	"strings"
)

// StructTags struct tags values
type StructTags struct {
	Cli      string
	Cmd      string
	Long     string
	Short    string
	Env      string
	Default  string
	Usage    string
	Complete string
}

func (st StructTags) parseTags(t reflect.StructTag) structTags {
	return structTags{
		Cli:      parseCliTag(t.Get(st.Cli)),
		Cmd:      t.Get(st.Cmd),
		Long:     parseLongTag(t.Get(st.Long)),
		Short:    t.Get(st.Short),
		Env:      parseEnvTag(t.Get(st.Env)),
		Default:  t.Get(st.Default),
		Usage:    t.Get(st.Usage),
		Complete: t.Get(st.Complete),
	}
}

type structTags struct {
	Cli      *cliTag
	Cmd      string
	Long     *longTag
	Short    string
	Env      *envTag
	Default  string
	Usage    string
	Complete string
}

func (st structTags) IsIgnored() bool {
	if st.Cli != nil {
		return st.Cli.ignored
	}
	return false
}

func (st structTags) CmdIsIgnored() bool {
	return st.Cmd == "-"
}

func (st structTags) LongIsIgnored() bool {
	if st.Long != nil {
		return st.Long.ignored
	}
	return false
}

func (st structTags) EnvIsIgnored() bool {
	if st.Env != nil {
		return st.Env.ignored
	}
	return false
}

type cliTag struct {
	ignored    bool
	required   bool
	positional bool
	global     bool
}

func parseCliTag(s string) *cliTag {
	tag := &cliTag{}
	parts := strings.Split(s, ",")
	for _, key := range parts {
		switch strings.ToLower(key) {
		case "required":
			tag.required = true
		case "positional":
			tag.positional = true
		case "global":
			tag.global = true
		}
	}
	return tag
}

type longTag struct {
	name     string
	explicit bool
	ignored  bool
}

func parseLongTag(s string) *longTag {
	tag := &longTag{}
	if s == "-" {
		tag.ignored = true
		return tag
	}
	parts := strings.Split(s, ",")
	switch len(parts) {
	case 1:
		tag.name = parts[0]
	case 2:
		tag.name = parts[0]
		if strings.ToLower(parts[1]) == "explicit" {
			tag.explicit = true
		}
	}
	return tag
}

type envTag struct {
	name     string
	explicit bool
	ignored  bool
}

func parseEnvTag(s string) *envTag {
	tag := &envTag{}
	if s == "-" {
		tag.ignored = true
		return tag
	}
	parts := strings.Split(s, ",")
	switch len(parts) {
	case 1:
		tag.name = parts[0]
	case 2:
		tag.name = parts[0]
		if strings.ToLower(parts[1]) == "explicit" {
			tag.explicit = true
		}
	}
	return tag
}
