package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/lusingander/routegraph"
	"github.com/lusingander/routegraph/internal/printer"
)

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout io.Writer) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	routes, err := routegraph.Analyze(ctx, routegraph.AnalyzeOptions{
		Dir: dir,
	})
	if err != nil {
		return err
	}

	return printer.Print(stdout, routes)
}
