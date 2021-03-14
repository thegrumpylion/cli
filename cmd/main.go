package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/thegrumpylion/cnc"
)

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

type SubCmdA struct {
	Name string
	Enum MuhEnm
}

func (c *SubCmdA) Run(ctx context.Context) error {
	fmt.Println("running subcmda")
	fmt.Println("name", c.Name)
	fmt.Println("enum", c.Enum)
	return nil
}

type SubCmdB struct {
	Num int
}

func (c *SubCmdB) Run(ctx context.Context) error {
	fmt.Println("running subcmdb")
	fmt.Println("num", c.Num)
	return nil
}

type RootCmd struct {
	SubA   *SubCmdA
	SubB   *SubCmdB
	Name   string
	Number int
	Flag   bool
	File   string `complete:"files"`
	Host   string `complete:"hosts"`
}

func (c *RootCmd) Run(ctx context.Context) error {
	fmt.Println("running rootcmd")
	fmt.Println("name", c.Name)
	fmt.Println("number", c.Number)
	fmt.Println("flag", c.Flag)
	fmt.Println("file", c.File)
	fmt.Println("host", c.Host)
	return nil
}

func main() {
	cnc.RegisterEnum(enumMap)

	c := &RootCmd{}

	cnc.NewRootCommand(filepath.Base(os.Args[0]), c)

	if err := cnc.Parse(os.Args); err != nil {
		panic(err)
	}

	if err := cnc.Execute(context.Background()); err != nil {
		panic(err)
	}
}
