package routegraph

import (
	"context"

	"github.com/lusingander/routegraph/internal/analyzer"
	echoanalyzer "github.com/lusingander/routegraph/internal/analyzer/echo"
)

type Route = analyzer.Route

type AnalyzeOptions struct {
	Dir string
}

func Analyze(ctx context.Context, opts AnalyzeOptions) ([]Route, error) {
	tree := analyzer.NewRouteTree()
	if err := echoanalyzer.Analyze(ctx, opts.Dir, tree); err != nil {
		return nil, err
	}
	return analyzer.Flatten(tree), nil
}
