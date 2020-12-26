package cli

import (
	"reflect"
)

type argument struct {
	path       *path
	typ        reflect.Type
	def        string
	long       string
	short      string
	help       string
	global     bool
	positional bool
	required   bool
	separate   bool
	enum       bool
	iface      bool
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
	return a.path.SetScalar(val)
}

func (a *argument) setArrayValue(arr []string) error {
	_, l := a.isArray()
	if l > 0 {
		return a.path.SetArray(arr)
	}
	return a.path.SetSlice(arr)
}

func (a *argument) setValue(val interface{}) error {
	return a.path.Set(val)
}
