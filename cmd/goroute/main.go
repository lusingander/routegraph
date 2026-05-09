package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/lusingander/routegraph"
	"github.com/lusingander/routegraph/internal/printer"
)

type cliOptions struct {
	dir        string
	jsonOutput bool
}

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout io.Writer) error {
	opts, err := parseOptions(args)
	if err != nil {
		return err
	}

	routes, err := routegraph.Analyze(ctx, routegraph.AnalyzeOptions{
		Dir: opts.dir,
	})
	if err != nil {
		return err
	}

	if opts.jsonOutput {
		return printer.PrintJSON(stdout, routes)
	}
	return printer.Print(stdout, routes)
}

func parseOptions(args []string) (cliOptions, error) {
	var opts cliOptions
	fs := flag.NewFlagSet("goroute", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.BoolVar(&opts.jsonOutput, "json", false, "print routes as JSON")
	if err := fs.Parse(args); err != nil {
		return opts, err
	}

	opts.dir = "."
	if fs.NArg() > 0 {
		opts.dir = fs.Arg(0)
	}
	return opts, nil
}
