package analyzer

import (
	"context"
)

type AnalyzeOptions struct {
	Dir string
}

type FrameworkAnalyzer interface {
	Analyze(ctx context.Context, pkgs []GoPackage, tree *RouteTree) error
}

func Analyze(ctx context.Context, opts AnalyzeOptions, frameworks ...FrameworkAnalyzer) ([]Route, error) {
	tree, err := AnalyzeTree(ctx, opts, frameworks...)
	if err != nil {
		return nil, err
	}
	return Flatten(tree), nil
}

func AnalyzeTree(ctx context.Context, opts AnalyzeOptions, frameworks ...FrameworkAnalyzer) (*RouteTree, error) {
	tree := NewRouteTree()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(frameworks) == 0 {
		return tree, nil
	}

	pkgs, err := LoadGoPackages(opts.Dir)
	if err != nil {
		return nil, err
	}
	for _, framework := range frameworks {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := framework.Analyze(ctx, pkgs, tree); err != nil {
			return nil, err
		}
	}
	return tree, nil
}
