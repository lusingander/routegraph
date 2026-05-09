package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/kong"
	"github.com/lusingander/routegraph"
	"github.com/lusingander/routegraph/internal/printer"
)

type cliOptions struct {
	JSON bool   `help:"Print routes as JSON."`
	Dir  string `arg:"" optional:"" default:"." help:"Target directory or package pattern."`
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
		Dir: opts.Dir,
	})
	if err != nil {
		return err
	}

	if opts.JSON {
		return printer.PrintJSON(stdout, routes)
	}
	return printer.Print(stdout, routes)
}

func parseOptions(args []string) (cliOptions, error) {
	var opts cliOptions
	parser, err := kong.New(&opts,
		kong.Name("goroute"),
		kong.Description("List Go web routes."),
		kong.Writers(io.Discard, io.Discard),
	)
	if err != nil {
		return opts, err
	}
	if _, err := parser.Parse(args); err != nil {
		return opts, err
	}
	return opts, nil
}
