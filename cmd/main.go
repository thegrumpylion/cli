package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/thegrumpylion/cli"
)

type SubCmdA struct {
	Name string
}

type SubCmdB struct {
	Num int
}

type RootCmd struct {
	SubA   *SubCmdA
	SubB   *SubCmdB
	Name   string
	Number int
	Flag   bool
}

func main() {
	c := &RootCmd{}
	cli.NewRootCommand(filepath.Base(os.Args[0]), c)
	err := cli.Eval(os.Args)
	if err != nil {
		panic(err)
	}
	fmt.Println(c.Name)
	fmt.Println(c.Number)
	fmt.Println(c.Flag)
}
