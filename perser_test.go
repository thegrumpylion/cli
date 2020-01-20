package cli

import (
	"fmt"
	"reflect"
	"testing"
)

func TestRefl(t *testing.T) {
	args := &struct {
		S struct {
			A string
			B int
		}
	}{}

	var a, b reflect.Value
	var av, bv interface{}
	av = "value"
	bv = 42

	v := reflect.ValueOf(args)
	if v.Type().Kind() != reflect.Ptr {
		t.Fatalf("cannot set")
	}

	v = v.Elem()
	if v.Type().Kind() != reflect.Struct {
		t.Fatalf("not a struct")
	}

	a = v.FieldByName("S").FieldByName("A")
	b = v.FieldByName("S").FieldByName("B")

	a.Set(reflect.ValueOf(av))
	b.Set(reflect.ValueOf(bv))

	fmt.Println(args)
}

func TestParse(t *testing.T) {
	type subCmd struct {
		SubStr   string
		SubInt   int
		SubBool  bool
		SubFloat float64
	}
	type embbb struct {
		EmbStr   string
		EmbInt   int
		EmbBool  bool
		EmbFloat float64
		EmbCmd   *subCmd
	}
	args := struct {
		embbb
		extern   embbb
		Str      string
		Int      int
		Bool     bool
		Float    float64
		StrPtr   *string
		IntPtr   *int
		BoolPtr  *bool
		FloatPtr *float64
		Cmd      *subCmd
	}{}

	cmd := parseCmd(reflect.TypeOf(args), "test", nil, CaseLower)

	fmt.Println(cmd)
}

// MuhEnm
type MuhEnm int

const (
	Ena MuhEnm = iota
	Dio
	Tria
)

func TestTypeName(t *testing.T) {
	m := map[string]MuhEnm{}

	ty := reflect.TypeOf(m)

	te := ty.Elem()
	tk := ty.Key()

	fmt.Println(te.PkgPath() + "." + te.Name())
	fmt.Println(tk.PkgPath() + "." + tk.Name())
}

func TestArray(t *testing.T) {
	a := [4]string{}
	s := []string{}

	ta := reflect.TypeOf(a)
	ts := reflect.TypeOf(s)

	fmt.Println(ta.Kind(), ts.Kind())
	fmt.Println(ta.Len())
}
