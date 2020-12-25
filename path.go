package cli

import (
	"reflect"
)

type path struct {
	root *reflect.Value
	path []string
}

func (p *path) Subpath(name string) *path {
	return &path{
		root: p.root,
		path: append(p.path, name),
	}
}

func (p *path) Init() interface{} {
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
	return v.Interface()
}

func (p *path) Set(in interface{}) {
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
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}
	v.Set(reflect.ValueOf(in))
}

func (p *path) SetFloat(in float64) {
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
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}
	v.SetFloat(in)
}

func (p *path) SetInt(in int64) {
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
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}
	v.SetInt(in)
}

func (p *path) SetUint(in uint64) {
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
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}
	v.SetUint(in)
}

func (p *path) Get() interface{} {
	v := *p.root
	for _, s := range p.path {
		if !v.IsValid() {
			return nil
		}
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		v = v.FieldByName(s)
	}
	if !v.IsValid() {
		return nil
	}
	// if v.Kind() == reflect.Ptr {
	// 	v = v.Elem()
	// }
	// if !v.IsValid() {
	// 	return nil
	// }
	return v.Interface()
}
