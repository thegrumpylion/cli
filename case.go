package cli

import (
	"strings"

	"github.com/iancoleman/strcase"
)

// Case string case
type Case uint32

const (
	CaseNone Case = iota
	CaseLower
	CaseUpper
	CaseCamel
	CaseCamelLower
	CaseSnake
	CaseSnakeUpper
	CaseKebab
	CaseKebabUpper
)

var caseFuncs = map[Case]func(string) string{
	CaseNone: func(s string) string {
		return s
	},
	CaseLower: func(s string) string {
		return strings.ToLower(s)
	},
	CaseUpper: func(s string) string {
		return strings.ToUpper(s)
	},
	CaseCamel: func(s string) string {
		return strcase.ToCamel(s)
	},
	CaseCamelLower: func(s string) string {
		return strcase.ToLowerCamel(s)
	},
	CaseSnake: func(s string) string {
		return strcase.ToSnake(s)
	},
	CaseSnakeUpper: func(s string) string {
		return strcase.ToScreamingSnake(s)
	},
	CaseKebab: func(s string) string {
		return strcase.ToKebab(s)
	},
	CaseKebabUpper: func(s string) string {
		return strcase.ToScreamingKebab(s)
	},
}
