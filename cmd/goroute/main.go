package main

import (
	"context"
	"fmt"
	"os"

	"github.com/lusingander/routegraph"
	"github.com/lusingander/routegraph/internal/printer"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	dir := "."
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}

	routes, err := routegraph.Analyze(context.Background(), routegraph.AnalyzeOptions{
		Dir: dir,
	})
	if err != nil {
		return err
	}

	return printer.Print(os.Stdout, routes)
}
