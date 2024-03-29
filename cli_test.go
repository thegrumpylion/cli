package cli

import (
	"context"
	"errors"
	"os"
	"strings"
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

	strargs := []string{
		"--str",
		"--int",
		"--bool",
		"--float",
		"--strPtr",
		"--intPtr",
		"--boolPtr",
		"--floatPtr",
		"--embStr",
		"--embInt",
		"--embBool",
		"--embFloat",
	}

	strcmds := []string{
		"cmd",
		"embcmd",
	}

	NewCommand("root", args)

	root := defaultCLI.cmds["root"]

	for _, sa := range strargs {
		if _, ok := root.flags.long[sa]; !ok {
			t.Fatalf("arg %s not found\n", sa)
		}
	}

	for _, sc := range strcmds {
		if _, ok := root.subcmdsMap[sc]; !ok {
			t.Fatalf("cmd %s not found\n", sc)
		}
	}
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
	NewCommand("root", args)
	err := Parse([]string{"root", "--enum", "dio"})
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

	NewCommand("root", args)

	err := Parse([]string{"root", "--string", "stringVal"})
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

	NewCommand("root", args)

	err := Parse([]string{"root", "--int", "-23", "--int8", "-3", "--int16", "-24000", "--int32", "-70123", "--int64", "-10200300"})
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

	NewCommand("root", args)

	err := Parse([]string{"root", "--uint", "23", "--uint8", "3", "--uint16", "24000", "--uint32", "70123", "--uint64", "10200300"})
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

func TestSliceArg(t *testing.T) {
	args := &struct {
		Names []string
	}{}

	NewCommand("root", args)

	err := Parse([]string{"root", "--names", "maria", "--names", "andreas", "--names", "giannis"})
	if err != nil {
		t.Fatal(err)
	}

	vals := []string{"maria", "andreas", "giannis"}
	for i, n := range vals {
		if args.Names[i] != n {
			t.Fatalf("%s not %s\n", args.Names[i], n)
		}
	}
}

type SubCmdA struct {
	Name string
}

func (c *SubCmdA) Run(ctx context.Context) error {
	v := ctx.Value(testStateKey{})
	if v == nil {
		return errors.New("SubCmdA.Run: testState not found in context")
	}
	s, ok := v.(*testState)
	if !ok {
		return errors.New("SubCmdA.Run: v is not *testState")
	}
	s.suba = true
	if c.Name != "tester" {
		s.t.Fatal("SubCmdA.Run: Name is not tester")
	}
	return nil
}

type SubCmdB struct {
	Num int
}

func (c *SubCmdB) Run(ctx context.Context) error {
	v := ctx.Value(testStateKey{})
	if v == nil {
		return errors.New("SubCmdB.Run: testState not found in context")
	}
	s, ok := v.(*testState)
	if !ok {
		return errors.New("SubCmdB.Run: v is not *testState")
	}
	s.subb = true
	if c.Num != 42 {
		s.t.Fatal("SubCmdB.Run: Num is not 42")
	}
	return nil
}

type RootCmd struct {
	SomeFlag string
	SubA     *SubCmdA
	SubB     *SubCmdB
}

func (c *RootCmd) PersistentPreRun(ctx context.Context) error {
	v := ctx.Value(testStateKey{})
	if v == nil {
		return errors.New("RootCmd.PersistentPreRun: testState not found in context")
	}
	s, ok := v.(*testState)
	if !ok {
		return errors.New("RootCmd.PersistentPreRun: v is not *testState")
	}
	s.ppr = true
	return nil
}

type testState struct {
	t    *testing.T
	ppr  bool
	suba bool
	subb bool
}

type testStateKey struct{}

func TestExecuteRoot(t *testing.T) {
	a := &RootCmd{}

	NewCommand("root", a)

	err := Parse([]string{"root"})
	if err != nil {
		t.Fatal(err)
	}

	state := &testState{t: t}

	ctx := context.WithValue(context.Background(), testStateKey{}, state)
	err = Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if !state.ppr {
		t.Fatal("state.ppr not set")
	}
}

func TestRunSubA(t *testing.T) {
	a := &RootCmd{}

	NewCommand("root", a)

	err := Parse([]string{"root", "suba", "--name", "tester"})
	if err != nil {
		t.Fatal(err)
	}

	state := &testState{t: t}

	ctx := context.WithValue(context.Background(), testStateKey{}, state)
	err = Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if !state.ppr {
		t.Fatal("state.ppr not set")
	}

	if !state.suba {
		t.Fatal("state.suba not set")
	}
}

func TestRunSubB(t *testing.T) {
	a := &RootCmd{}

	NewCommand("root", a)

	err := Parse([]string{"root", "subb", "--num", "42"})
	if err != nil {
		t.Fatal(err)
	}

	state := &testState{t: t}

	ctx := context.WithValue(context.Background(), testStateKey{}, state)
	err = Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if !state.ppr {
		t.Fatal("state.ppr not set")
	}

	if !state.subb {
		t.Fatal("state.subb not set")
	}
}

func TestRunWithRootFlag(t *testing.T) {
	a := &RootCmd{}

	NewCommand("root", a)

	err := Parse([]string{"root", "--someFlag", "someValue", "suba", "--name", "tester"})
	if err != nil {
		t.Fatal(err)
	}

	if a.SomeFlag != "someValue" {
		t.Fatal("SomeFlag != someValue")
	}

}

type tUm struct {
	Key   string
	Value string
}

func (t *tUm) UnmarshalText(b []byte) error {
	s := string(b)
	i := strings.Index(s, ":")
	if i == -1 {
		return errors.New("invalid")
	}
	t.Key = s[:i]
	t.Value = s[i+1:]
	return nil
}

func TestTextUnmarshaler(t *testing.T) {
	args := &struct {
		Pair *tUm
	}{}

	NewCommand("root", args)

	err := Parse([]string{"root", "--pair", "theKey:theValue"})
	if err != nil {
		t.Fatal(err)
	}

	if args.Pair.Key != "theKey" {
		t.Fatal("key != theKey ==", args.Pair.Key)
	}

	if args.Pair.Value != "theValue" {
		t.Fatal("value != theValue ==", args.Pair.Value)
	}
}

func TestGlobals(t *testing.T) {
	type subcmd struct {
		A string
		B int
	}
	args := &struct {
		Subcmd *subcmd
		G      string `cli:"global"`
	}{}
	p := NewCLI(WithGlobalArgsEnabled())
	p.NewCommand("root", args)
	err := p.Parse([]string{"root", "subcmd", "--a", "a", "--b", "1", "--g", "global"})
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if args.G != "global" {
		t.Fatal("--g != global")
	}
	if args.Subcmd.A != "a" {
		t.Fatal("--a != a")
	}
	if args.Subcmd.B != 1 {
		t.Fatal("--b != 1")
	}
}

func TestGlobalsConflict(t *testing.T) {
	type subcmd struct {
		G string
	}
	args := &struct {
		G      string `cli:"global"`
		Subcmd *subcmd
	}{}
	p := NewCLI(WithGlobalArgsEnabled())
	defer func() {
		if i := recover(); i == nil {
			t.Fatal("should have paniced globals conflict")
		}
	}()
	p.NewCommand("root", args)
}

func TestIsFlag(t *testing.T) {
	if !isFlag("--help") {
		t.Fatal("--help is flag")
	}
	if !isFlag("-h") {
		t.Fatal("-h is flag")
	}
	if isFlag("-42") {
		t.Fatal("-42 is a number not a flag")
	}
}

func TestDefaultValues(t *testing.T) {

	args := &struct {
		Num    int      `default:"5"`
		String string   `default:"some_value"`
		List   []string `default:"one two three"`
	}{}

	NewCommand("root", args)

	if err := Parse([]string{"root"}); err != nil {
		t.Fatal("failed to parse args: " + err.Error())
	}

	if args.Num != 5 {
		t.Fatal("Num should have been 5")
	}
	if args.String != "some_value" {
		t.Fatal("String should have been some_value")
	}
	if len(args.List) != 3 {
		t.Fatal("List should have len 3")
	}
	for i, v := range []string{"one", "two", "three"} {
		if args.List[i] != v {
			t.Fatalf("List[%d] should have been %q", i, v)
		}
	}
}

func TestEnv(t *testing.T) {

	args := &struct {
		Num    int
		String string
		List   []string
	}{}

	os.Setenv("NUM", "5")
	os.Setenv("STRING", "some_value")
	os.Setenv("LIST", "one two three")

	NewCommand("root", args)

	if err := Parse([]string{"root"}); err != nil {
		t.Fatal("failed to parse args: " + err.Error())
	}

	if args.Num != 5 {
		t.Fatal("Num should have been 5")
	}
	if args.String != "some_value" {
		t.Fatal("String should have been some_value")
	}
	if len(args.List) != 3 {
		t.Fatal("List should have len 3")
	}
	for i, v := range []string{"one", "two", "three"} {
		if args.List[i] != v {
			t.Fatalf("List[%d] should have been %q", i, v)
		}
	}
}
