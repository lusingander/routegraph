package echo

import (
	"context"
	"testing"

	"github.com/lusingander/routegraph/internal/analyzer"
)

func TestAnalyzeBasicRoutes(t *testing.T) {
	tree := analyzer.NewRouteTree()
	if err := Analyze(context.Background(), "../../../testdata/basic", tree); err != nil {
		t.Fatal(err)
	}

	routes := analyzer.Flatten(tree)
	if len(routes) != 3 {
		t.Fatalf("len(routes) = %d, want 3: %#v", len(routes), routes)
	}

	assertRoute(t, routes[0], "GET", "/api/v1/users", "listUsers", true)
	assertRoute(t, routes[1], "POST", "/api/v1/users", "createUser", true)
	assertRoute(t, routes[2], "GET", "/api/v1/admin/stats", "h.Stats", true)
}

func TestAnalyzeUnknownPath(t *testing.T) {
	tree := analyzer.NewRouteTree()
	if err := Analyze(context.Background(), "../../../testdata/unknown_path", tree); err != nil {
		t.Fatal(err)
	}

	routes := analyzer.Flatten(tree)
	if len(routes) != 1 {
		t.Fatalf("len(routes) = %d, want 1: %#v", len(routes), routes)
	}

	assertRoute(t, routes[0], "GET", "<unknown>/users", "listUsers", false)
}

func TestAnalyzeConstPath(t *testing.T) {
	tree := analyzer.NewRouteTree()
	if err := Analyze(context.Background(), "../../../testdata/const_path", tree); err != nil {
		t.Fatal(err)
	}

	routes := analyzer.Flatten(tree)
	if len(routes) != 3 {
		t.Fatalf("len(routes) = %d, want 3: %#v", len(routes), routes)
	}

	assertRoute(t, routes[0], "GET", "/api/v1/users", "listUsers", true)
	assertRoute(t, routes[1], "GET", "/api/v1/users/:id", "getUser", true)
	assertRoute(t, routes[2], "POST", "/api/v1/admin/stats", "createStat", true)
}

func TestAnalyzeAnyAdd(t *testing.T) {
	tree := analyzer.NewRouteTree()
	if err := Analyze(context.Background(), "../../../testdata/any_add", tree); err != nil {
		t.Fatal(err)
	}

	routes := analyzer.Flatten(tree)
	if len(routes) != 3 {
		t.Fatalf("len(routes) = %d, want 3: %#v", len(routes), routes)
	}

	assertRoute(t, routes[0], "ANY", "/health", "health", true)
	assertRoute(t, routes[1], "GET", "/api/users", "listUsers", true)
	assertRoute(t, routes[2], "UNKNOWN", "/api/dynamic", "dynamicHandler", true)
}

func assertRoute(t *testing.T, route analyzer.Route, method, path, handler string, known bool) {
	t.Helper()
	if route.Framework != string(analyzer.FrameworkEcho) {
		t.Fatalf("Framework = %q, want %q", route.Framework, analyzer.FrameworkEcho)
	}
	if route.Method != method {
		t.Fatalf("Method = %q, want %q", route.Method, method)
	}
	if route.Path != path {
		t.Fatalf("Path = %q, want %q", route.Path, path)
	}
	if route.Handler != handler {
		t.Fatalf("Handler = %q, want %q", route.Handler, handler)
	}
	if route.Known != known {
		t.Fatalf("Known = %v, want %v", route.Known, known)
	}
	if route.File == "" || route.Line == 0 {
		t.Fatalf("location not set: file=%q line=%d", route.File, route.Line)
	}
}
