package cli

import (
	"strings"
	"unicode"

	"github.com/iancoleman/strcase"
)

// Case string case
type Case uint32

const (
	// CaseNone string as is
	CaseNone Case = iota
	// CaseLower caselower
	CaseLower
	// CaseUpper CASEUPPER
	CaseUpper
	// CaseCamel CaseCamel
	CaseCamel
	// CaseCamelLower caseCamelLower
	CaseCamelLower
	// CaseSnake case_snake
	CaseSnake
	// CaseSnakeUpper CASE_SNAKE_UPPER
	CaseSnakeUpper
	// CaseKebab case-kebab
	CaseKebab
	// CaseKebabUpper CASE-KEBAB-UPPER
	CaseKebabUpper
)

// Parse resturns s in case c
func (c Case) Parse(s string) string {
	switch c {
	case CaseLower:
		return strings.ToLower(s)
	case CaseUpper:
		return strings.ToUpper(s)
	case CaseCamel:
		return strcase.ToCamel(s)
	case CaseCamelLower:
		if isUpper(s) {
			return strings.ToLower(s)
		}
		return strcase.ToLowerCamel(s)
	case CaseSnake:
		return strcase.ToSnake(s)
	case CaseSnakeUpper:
		return strcase.ToScreamingSnake(s)
	case CaseKebab:
		return strcase.ToKebab(s)
	case CaseKebabUpper:
		return strcase.ToScreamingKebab(s)
	default:
		return s
	}
}

// Splicer neated arg/env string splicer
type Splicer uint32

const (
	// SplicerNone no splicer
	SplicerNone = iota
	// SplicerDot . splicer
	SplicerDot
	// SplicerDash - splicer
	SplicerDash
	// SplicerUnderscore _ splicer
	SplicerUnderscore
)

// Splice concatenates a and b separated by char s
func (s Splicer) Splice(a, b string) string {
	switch s {
	case SplicerDot:
		return a + "." + b
	case SplicerDash:
		return a + "-" + b
	case SplicerUnderscore:
		return a + "_" + b
	default:
		return a + b
	}
}

func isUpper(s string) bool {
	for _, r := range s {
		if !unicode.IsUpper(r) && unicode.IsLetter(r) {
			return false
		}
	}
	return true
}
