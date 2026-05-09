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
	return analyzer.Analyze(ctx, analyzer.AnalyzeOptions{
		Dir: opts.Dir,
	}, echoanalyzer.Analyzer{})
}
