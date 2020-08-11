package cli

import (
	"fmt"
	"testing"
)

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
	args := &struct {
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

	cmd := NewRootCommand("test", args)

	fmt.Println(cmd)
}

func TestEnumRegistration(t *testing.T) {

	type MuhEnm int

	const (
		Ena MuhEnm = iota + 1
		Dio
		Tria
	)

	enumMap := map[string]MuhEnm{
		"ena":  Ena,
		"dio":  Dio,
		"tria": Tria,
	}

	args := &struct {
		Enum MuhEnm
	}{}

	RegisterEnum(enumMap)
	NewRootCommand("root", args)
	err := Eval([]string{"root", "--enum", "dio"})
	if err != nil {
		t.Fatal(err)
	}

	if args.Enum != Dio {
		t.Fatal("args.Enum != Dio")
	}

}

func TestString(t *testing.T) {
	args := &struct {
		String string
	}{}

	NewRootCommand("root", args)

	err := Eval([]string{"root", "--string", "stringVal"})
	if err != nil {
		t.Fatal(err)
	}

	if args.String != "stringVal" {
		t.Fatal("args.String != stringVal")
	}
}

func TestInt(t *testing.T) {
	args := &struct {
		Int   int
		Int8  int8
		Int16 int16
		Int32 int32
		Int64 int64
	}{}

	NewRootCommand("root", args)

	err := Eval([]string{"root", "--int", "-23", "--int8", "-3", "--int16", "-24000", "--int32", "-70123", "--int64", "-10200300"})
	if err != nil {
		t.Fatal(err)
	}

	if args.Int != -23 {
		t.Fatal("args.Int != -23")
	}

	if args.Int8 != -3 {
		t.Fatal("args.Int8 != -3")
	}

	if args.Int16 != -24000 {
		t.Fatal("args.Int16 != -24000")
	}

	if args.Int32 != -70123 {
		t.Fatal("args.Int32 != -70123")
	}

	if args.Int64 != -10200300 {
		t.Fatal("args.Int64 != -10200300")
	}
}

func TestUint(t *testing.T) {
	args := &struct {
		Uint   uint
		Uint8  uint8
		Uint16 uint16
		Uint32 uint32
		Uint64 uint64
	}{}

	NewRootCommand("root", args)

	err := Eval([]string{"root", "--uint", "23", "--uint8", "3", "--uint16", "24000", "--uint32", "70123", "--uint64", "10200300"})
	if err != nil {
		t.Fatal(err)
	}

	if args.Uint != 23 {
		t.Fatal("args.Uint != 23")
	}

	if args.Uint8 != 3 {
		t.Fatal("args.Uint8 != 3")
	}

	if args.Uint16 != 24000 {
		t.Fatal("args.Uint16 != 24000")
	}

	if args.Uint32 != 70123 {
		t.Fatal("args.Uint32 != 70123")
	}

	if args.Uint64 != 10200300 {
		t.Fatal("args.Uint64 != 10200300")
	}
}
