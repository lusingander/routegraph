package analyzer

import (
	"context"
	"testing"
)

type fakeFrameworkAnalyzer struct{}

func (fakeFrameworkAnalyzer) Analyze(ctx context.Context, pkgs []GoPackage, tree *RouteTree) error {
	if len(pkgs) == 0 {
		return nil
	}
	tree.AddRoute(0, FrameworkEcho, "GET", KnownPath("/fake"), "handler", Position{})
	return nil
}

func TestAnalyzeTreeWithFrameworkAnalyzer(t *testing.T) {
	tree, err := AnalyzeTree(context.Background(), AnalyzeOptions{Dir: "../../testdata/basic"}, fakeFrameworkAnalyzer{})
	if err != nil {
		t.Fatal(err)
	}

	routes := Flatten(tree)
	if len(routes) != 1 {
		t.Fatalf("len(routes) = %d, want 1: %#v", len(routes), routes)
	}
	if routes[0].Path != "/fake" {
		t.Fatalf("Path = %q, want /fake", routes[0].Path)
	}
}

func TestAnalyzeTreeWithoutFrameworkAnalyzer(t *testing.T) {
	tree, err := AnalyzeTree(context.Background(), AnalyzeOptions{Dir: "../../testdata/basic"})
	if err != nil {
		t.Fatal(err)
	}
	if len(tree.Nodes) != 1 {
		t.Fatalf("len(tree.Nodes) = %d, want 1", len(tree.Nodes))
	}
}
