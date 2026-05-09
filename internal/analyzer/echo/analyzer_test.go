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
	assertBasicTree(t, tree)
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

func TestAnalyzeSkipsNonEchoReceivers(t *testing.T) {
	tree := analyzer.NewRouteTree()
	if err := Analyze(context.Background(), "../../../testdata/type_aware", tree); err != nil {
		t.Fatal(err)
	}

	routes := analyzer.Flatten(tree)
	if len(routes) != 1 {
		t.Fatalf("len(routes) = %d, want 1: %#v", len(routes), routes)
	}

	assertRoute(t, routes[0], "GET", "/ok", "realHandler", true)
}

func TestAnalyzeSkipsFakeAdd(t *testing.T) {
	tree := analyzer.NewRouteTree()
	if err := Analyze(context.Background(), "../../../testdata/false_positive", tree); err != nil {
		t.Fatal(err)
	}

	routes := analyzer.Flatten(tree)
	if len(routes) != 1 {
		t.Fatalf("len(routes) = %d, want 1: %#v", len(routes), routes)
	}

	assertRoute(t, routes[0], "GET", "/real", "realHandler", true)
}

func TestAnalyzeFunctionSplit(t *testing.T) {
	tree := analyzer.NewRouteTree()
	if err := Analyze(context.Background(), "../../../testdata/function_split", tree); err != nil {
		t.Fatal(err)
	}

	routes := analyzer.Flatten(tree)
	if len(routes) != 2 {
		t.Fatalf("len(routes) = %d, want 2: %#v", len(routes), routes)
	}

	assertRoute(t, routes[0], "GET", "/api/users", "listUsers", true)
	assertRoute(t, routes[1], "POST", "/api/users", "createUser", true)
}

func TestAnalyzeFunctionSplitAcrossFiles(t *testing.T) {
	tree := analyzer.NewRouteTree()
	if err := Analyze(context.Background(), "../../../testdata/function_split_multifile", tree); err != nil {
		t.Fatal(err)
	}

	routes := analyzer.Flatten(tree)
	if len(routes) != 1 {
		t.Fatalf("len(routes) = %d, want 1: %#v", len(routes), routes)
	}

	assertRoute(t, routes[0], "GET", "/api/users", "listUsers", true)
}

func TestAnalyzeMethodCall(t *testing.T) {
	tree := analyzer.NewRouteTree()
	if err := Analyze(context.Background(), "../../../testdata/method_call", tree); err != nil {
		t.Fatal(err)
	}

	routes := analyzer.Flatten(tree)
	if len(routes) != 2 {
		t.Fatalf("len(routes) = %d, want 2: %#v", len(routes), routes)
	}

	assertRoute(t, routes[0], "GET", "/api/users", "listUsers", true)
	assertRoute(t, routes[1], "GET", "/api/admin/stats", "stats", true)
}

func TestAnalyzeSkipsUncalledHelper(t *testing.T) {
	tree := analyzer.NewRouteTree()
	if err := Analyze(context.Background(), "../../../testdata/helper_only", tree); err != nil {
		t.Fatal(err)
	}

	routes := analyzer.Flatten(tree)
	if len(routes) != 0 {
		t.Fatalf("len(routes) = %d, want 0: %#v", len(routes), routes)
	}
}

func TestAnalyzeCyclicFunctionSplit(t *testing.T) {
	tree := analyzer.NewRouteTree()
	if err := Analyze(context.Background(), "../../../testdata/function_cycle", tree); err != nil {
		t.Fatal(err)
	}

	routes := analyzer.Flatten(tree)
	if len(routes) != 2 {
		t.Fatalf("len(routes) = %d, want 2: %#v", len(routes), routes)
	}

	assertRoute(t, routes[0], "GET", "/api/a", "handlerA", true)
	assertRoute(t, routes[1], "GET", "/api/b", "handlerB", true)
}

func TestAnalyzeStructField(t *testing.T) {
	tree := analyzer.NewRouteTree()
	if err := Analyze(context.Background(), "../../../testdata/struct_field", tree); err != nil {
		t.Fatal(err)
	}

	routes := analyzer.Flatten(tree)
	if len(routes) != 2 {
		t.Fatalf("len(routes) = %d, want 2: %#v", len(routes), routes)
	}

	assertRoute(t, routes[0], "GET", "/api/users", "listUsers", true)
	assertRoute(t, routes[1], "GET", "/api/admin/stats", "stats", true)
}

func TestAnalyzeRouteTable(t *testing.T) {
	tree := analyzer.NewRouteTree()
	if err := Analyze(context.Background(), "../../../testdata/route_table", tree); err != nil {
		t.Fatal(err)
	}

	routes := analyzer.Flatten(tree)
	if len(routes) != 3 {
		t.Fatalf("len(routes) = %d, want 3: %#v", len(routes), routes)
	}

	assertRoute(t, routes[0], "GET", "/api/users", "listUsers", true)
	assertRoute(t, routes[1], "POST", "/api/users", "createUser", true)
	assertRoute(t, routes[2], "GET", "/api/admin/stats", "stats", true)
}

func TestAnalyzeEchoCoverageRefinements(t *testing.T) {
	tree := analyzer.NewRouteTree()
	if err := Analyze(context.Background(), "../../../testdata/echo_refinement", tree); err != nil {
		t.Fatal(err)
	}

	routes := analyzer.Flatten(tree)
	if len(routes) != 4 {
		t.Fatalf("len(routes) = %d, want 4: %#v", len(routes), routes)
	}

	assertRoute(t, routes[0], "GET", "/api/users", "listUsers", true)
	assertRoute(t, routes[1], "POST", "/v2/users", "createUser", true)
	assertRoute(t, routes[2], "GET", "/chained/health", "health", true)
	assertRoute(t, routes[3], "GET", "/local", "localHandler", true)
}

func TestAnalyzeDynamicRoutePath(t *testing.T) {
	tree := analyzer.NewRouteTree()
	if err := Analyze(context.Background(), "../../../testdata/dynamic_route_path", tree); err != nil {
		t.Fatal(err)
	}

	routes := analyzer.Flatten(tree)
	if len(routes) != 2 {
		t.Fatalf("len(routes) = %d, want 2: %#v", len(routes), routes)
	}

	assertRoute(t, routes[0], "GET", "/api/<unknown>", "dynamicHandler", false)
	assertRoute(t, routes[1], "UNKNOWN", "/api/<unknown>", "dynamicAddHandler", false)
	assertDynamicRoutePathTree(t, tree)
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

func assertBasicTree(t *testing.T, tree *analyzer.RouteTree) {
	t.Helper()
	if len(tree.Nodes) != 7 {
		t.Fatalf("len(tree.Nodes) = %d, want 7: %#v", len(tree.Nodes), tree.Nodes)
	}
	assertNode(t, tree.Nodes[0], analyzer.NodeKindRoot, "", "/", nil)
	assertNode(t, tree.Nodes[1], analyzer.NodeKindGroup, "", "/api", nodeIDPtr(0))
	assertNode(t, tree.Nodes[2], analyzer.NodeKindGroup, "", "/api/v1", nodeIDPtr(1))
	assertNode(t, tree.Nodes[3], analyzer.NodeKindRoute, "GET", "/api/v1/users", nodeIDPtr(2))
	assertNode(t, tree.Nodes[4], analyzer.NodeKindRoute, "POST", "/api/v1/users", nodeIDPtr(2))
	assertNode(t, tree.Nodes[5], analyzer.NodeKindGroup, "", "/api/v1/admin", nodeIDPtr(2))
	assertNode(t, tree.Nodes[6], analyzer.NodeKindRoute, "GET", "/api/v1/admin/stats", nodeIDPtr(5))
}

func assertDynamicRoutePathTree(t *testing.T, tree *analyzer.RouteTree) {
	t.Helper()
	if len(tree.Nodes) != 4 {
		t.Fatalf("len(tree.Nodes) = %d, want 4: %#v", len(tree.Nodes), tree.Nodes)
	}
	assertNode(t, tree.Nodes[0], analyzer.NodeKindRoot, "", "/", nil)
	assertNode(t, tree.Nodes[1], analyzer.NodeKindGroup, "", "/api", nodeIDPtr(0))
	assertNode(t, tree.Nodes[2], analyzer.NodeKindRoute, "GET", "/api/<unknown>", nodeIDPtr(1))
	assertNode(t, tree.Nodes[3], analyzer.NodeKindRoute, "UNKNOWN", "/api/<unknown>", nodeIDPtr(1))
	if tree.Nodes[2].FullPath.Known {
		t.Fatalf("node 2 FullPath.Known = true, want false")
	}
	if tree.Nodes[3].FullPath.Known {
		t.Fatalf("node 3 FullPath.Known = true, want false")
	}
}

func assertNode(t *testing.T, node analyzer.RouteNode, kind analyzer.NodeKind, method, path string, parentID *analyzer.NodeID) {
	t.Helper()
	if node.Kind != kind {
		t.Fatalf("node %d Kind = %q, want %q", node.ID, node.Kind, kind)
	}
	if node.Method != method {
		t.Fatalf("node %d Method = %q, want %q", node.ID, node.Method, method)
	}
	if node.FullPath.Value != path {
		t.Fatalf("node %d FullPath = %q, want %q", node.ID, node.FullPath.Value, path)
	}
	if parentID == nil {
		if node.ParentID != nil {
			t.Fatalf("node %d ParentID = %d, want nil", node.ID, *node.ParentID)
		}
		return
	}
	if node.ParentID == nil {
		t.Fatalf("node %d ParentID = nil, want %d", node.ID, *parentID)
	}
	if *node.ParentID != *parentID {
		t.Fatalf("node %d ParentID = %d, want %d", node.ID, *node.ParentID, *parentID)
	}
}

func nodeIDPtr(id analyzer.NodeID) *analyzer.NodeID {
	return &id
}
