package cli

import (
	"fmt"
	"reflect"
	"strings"
)

type enum struct {
	typ    reflect.Type
	names  map[interface{}]string
	values map[string]interface{}
}

func newEnum(enumMap interface{}) *enum {
	v := reflect.ValueOf(enumMap)
	t := reflect.TypeOf(enumMap)
	if t.Kind() != reflect.Map {
		panic("enumMap must be a map")
	}

	// key is the string of enum
	if t.Key().Kind() != reflect.String {
		panic("enumMap key must be string")
	}

	// element is enum int/uint custom type
	te := t.Elem()
	if te.PkgPath() == "" {
		panic("enumMap element must be custom type")
	}
	if !(isInt(te) || isUint(te)) {
		panic("enumMap element must be int/uint")
	}

	enm := &enum{
		typ:    te,
		names:  map[interface{}]string{},
		values: map[string]interface{}{},
	}
	for _, k := range v.MapKeys() {
		name := strings.ToUpper(k.String())
		value := v.MapIndex(k).Interface()
		enm.values[name] = value
		enm.names[value] = name
	}
	return enm
}

func (e *enum) Name(v interface{}) string {
	if reflect.TypeOf(v) != e.typ {
		panic(fmt.Sprintf("invalid enum value type: %s not %s", reflect.TypeOf(v).Name(), e.typ.Name()))
	}
	return e.names[v]
}

func (e *enum) Value(s string) interface{} {
	return e.values[strings.ToUpper(s)]
}

func (e *enum) Complete(val string) (out []string) {
	for v := range e.values {
		if strings.HasPrefix(v, strings.ToUpper(val)) {
			out = append(out, v+" ")
		}
	}
	return
}
