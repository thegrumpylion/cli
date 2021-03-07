package cli

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type path struct {
	root *reflect.Value
	path []string
	val  *reflect.Value
}

func (p *path) Subpath(name string) *path {
	return &path{
		root: p.root,
		path: append(p.path, name),
	}
}

func (p *path) Get() interface{} {
	return p.value().Interface()
}

func (p *path) Set(i interface{}) error {
	return func() error {
		if r := recover(); r != nil {
			return fmt.Errorf("error: %v", r)
		}
		p.valueDeref().Set(reflect.ValueOf(i))
		return nil
	}()
}

func (p *path) SetScalar(s string) error {
	return setScalarValue(p.valueDeref(), s)
}

func (p *path) AppendToSlice(s string) error {
	v := p.valueDeref()
	e := reflect.New(v.Type().Elem()).Elem()
	if err := setScalarValue(e, s); err != nil {
		return err
	}
	v.Set(reflect.Append(v, e))
	return nil
}

func (p *path) valueDeref() reflect.Value {
	v := p.value()
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v
}

func (p *path) value() reflect.Value {
	if p.val != nil {
		return *p.val
	}
	v := *p.root
	for _, s := range p.path {
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				v.Set(reflect.New(v.Type().Elem()))
			}
			v = v.Elem()
		}
		v = v.FieldByName(s)
	}
	if v.Kind() == reflect.Ptr && v.IsNil() {
		v.Set(reflect.New(v.Type().Elem()))
	}
	// cache
	p.val = &reflect.Value{}
	*p.val = v
	return v
}

func setScalarValue(v reflect.Value, s string) error {
	if isPtr(v.Type()) {
		v = v.Elem()
	}
	switch {
	case isBool(v.Type()):
		if s == "" || strings.ToLower(s) == "true" {
			v.SetBool(true)
		} else if strings.ToLower(s) == "false" {
			v.SetBool(false)
		} else {
			return ErrInvalidValue(s, "")
		}
	case isString(v.Type()):
		v.SetString(s)
	case isFloat(v.Type()):
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return err
		}
		v.SetFloat(f)
	case isInt(v.Type()):
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return err
		}
		v.SetInt(i)
	case isUint(v.Type()):
		ui, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return err
		}
		v.SetUint(ui)
	}
	return nil
}
