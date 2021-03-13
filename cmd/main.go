package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/thegrumpylion/cnc"
)

type SubCmdA struct {
	Name string
	Enum MuhEnm
}

type SubCmdB struct {
	Num int
}

type MuhEnm int

const (
	Ena MuhEnm = iota + 1
	Dio
	Tria
)

var enumMap = map[string]MuhEnm{
	"ena":  Ena,
	"dio":  Dio,
	"tria": Tria,
}

type RootCmd struct {
	SubA   *SubCmdA
	SubB   *SubCmdB
	Name   string
	Number int
	Flag   bool
}

func main() {
	cnc.RegisterEnum(enumMap)
	c := &RootCmd{}
	cnc.NewRootCommand(filepath.Base(os.Args[0]), c)
	err := cnc.Eval(os.Args)
	if err != nil {
		panic(err)
	}
	fmt.Println("c.Name", c.Name)
	fmt.Println("c.Number", c.Number)
	fmt.Println("c.Flag", c.Flag)
	fmt.Println("c.SubA.Name", c.SubA.Name)
	fmt.Println("c.SubA.Enum", c.SubA.Enum)
	fmt.Println("c.SubB.Num", c.SubB.Num)
}
