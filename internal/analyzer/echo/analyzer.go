package echo

import (
	"context"

	"github.com/lusingander/routegraph/internal/analyzer"
)

func Analyze(ctx context.Context, dir string, tree *analyzer.RouteTree) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_, _, err := analyzer.LoadGoFiles(dir)
	if err != nil {
		return err
	}
	return nil
}
