package cli

import (
	"reflect"
	"strconv"
	"strings"
)

type argument struct {
	path       *path
	typ        reflect.Type
	def        reflect.Value
	long       string
	short      string
	help       string
	global     bool
	positional bool
	required   bool
	separate   bool
	enum       bool
}

func (a *argument) isBool() bool {
	return a.typ.Kind() == reflect.Bool
}

func (a *argument) isArray() (bool, int) {
	switch a.typ.Kind() {
	case reflect.Array:
		return true, a.typ.Len()
	case reflect.Slice:
		return true, -1
	default:
		return false, 0
	}
}

func (a *argument) setScalarValue(val string) error {
	switch {
	case isBool(a.typ):
		if val == "" || strings.ToLower(val) == "true" {
			a.path.Set(true)
		} else if strings.ToLower(val) == "false" {
			a.path.Set(false)
		} else {
			return ErrInvalidValue(val, a.long)
		}
	case isString(a.typ):
		a.path.Set(val)
	case isFloat(a.typ):
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return err
		}
		a.path.SetFloat(f)
	case isInt(a.typ):
		i, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return err
		}
		a.path.SetInt(i)
	case isUint(a.typ):
		ui, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			return err
		}
		a.path.SetUint(ui)
	}
	return nil
}
