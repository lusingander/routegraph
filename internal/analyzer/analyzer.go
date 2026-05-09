package analyzer

import (
	"context"
)

type AnalyzeOptions struct {
	Dir string
}

func Analyze(ctx context.Context, opts AnalyzeOptions) ([]Route, error) {
	tree, err := AnalyzeTree(ctx, opts)
	if err != nil {
		return nil, err
	}
	return Flatten(tree), nil
}

func AnalyzeTree(ctx context.Context, opts AnalyzeOptions) (*RouteTree, error) {
	tree := NewRouteTree()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return tree, nil
}
