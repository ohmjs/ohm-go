package main

//go:generate go run github.com/jpillora/md-tmpl -w README.md
//go:generate go generate ./utils

import (
	"fmt"
	"os"

	"github.com/jpillora/opts"
)

var (
	rootCmd = &struct{}{}
	builder = opts.New(rootCmd).
		Name("ohm").
		EmbedGlobalFlagSet().
		Complete()
)

func main() {
	var (
		cli opts.ParsedOpts
		err error
	)
	cli, err = builder.ParseArgsError(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%[1]v", err)
		os.Exit(1)
	}
	// cli = builder.ParseArgs(os.Args)
	err = cli.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%[2]s\nError: %[1]v\n\n", err, cli.Selected().Help())
		os.Exit(2)
	}
}
