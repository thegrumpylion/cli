package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/thegrumpylion/cnc"
)

type packCmd struct {
	Source  string `cli:"positional,required" help:"source directory"`
	Target  string `cli:"positional,required" help:"target file name"`
	Verbose bool   `help:"be extra chatty"`
}

func (c *packCmd) Run(ctx context.Context) error {
	if c.Verbose {
		fmt.Printf("packing %s to %s, aint that cool?\n", c.Source, c.Target)
		return nil
	}
	fmt.Printf("packing %s to %s\n", c.Source, c.Target)
	return nil
}

type unpackCmd struct {
	Source  string `cli:"positional,required" help:"source file"`
	Target  string `cli:"positional,required" help:"target directory"`
	Verbose bool   `help:"be extra chatty"`
}

func (c *unpackCmd) Run(ctx context.Context) error {
	if c.Verbose {
		fmt.Printf("unpacking %s to %s, aint that cool?\n", c.Source, c.Target)
		return nil
	}
	fmt.Printf("unpacking %s to %s\n", c.Source, c.Target)
	return nil
}

type listCmd struct {
	Archive string `cli:"positional,required"`
	Verbose bool   `help:"be extra chatty"`
}

func (c *listCmd) Run(ctx context.Context) error {

	return nil
}

type rootCmd struct {
	Pack   *packCmd   `help:"pack a directory to an archive"`
	Unpack *unpackCmd `help:"unpack an archive to a directory"`
	List   *listCmd   `help:"list the contents of an archive"`
}

func (c *rootCmd) Run(ctx context.Context) error {

	return nil
}

func (c *rootCmd) Description() string {
	return `minimal archiving`
}

func main() {
	cnc := cnc.NewCLI(cnc.WithSeparator(cnc.SeparatorEquals))

	c := &rootCmd{}

	cnc.NewRootCommand(filepath.Base(os.Args[0]), c)

	if err := cnc.Parse(os.Args); err != nil {
		panic(err)
	}

	if err := cnc.Execute(context.Background()); err != nil {
		panic(err)
	}
}
