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

func TestAnalyzeMatch(t *testing.T) {
	tree := analyzer.NewRouteTree()
	if err := Analyze(context.Background(), "../../../testdata/echo_match", tree); err != nil {
		t.Fatal(err)
	}

	routes := analyzer.Flatten(tree)
	if len(routes) != 4 {
		t.Fatalf("len(routes) = %d, want 4: %#v", len(routes), routes)
	}

	assertRoute(t, routes[0], "GET", "/api/users", "users", true)
	assertRoute(t, routes[1], "POST", "/api/users", "users", true)
	assertRoute(t, routes[2], "PUT", "/api/users/:id", "updateUser", true)
	assertRoute(t, routes[3], "PATCH", "/api/users/:id", "updateUser", true)
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

func TestAnalyzeFunctionSplitAcrossPackages(t *testing.T) {
	tree := analyzer.NewRouteTree()
	if err := Analyze(context.Background(), "../../../testdata/cross_package/...", tree); err != nil {
		t.Fatal(err)
	}

	routes := analyzer.Flatten(tree)
	if len(routes) != 1 {
		t.Fatalf("len(routes) = %d, want 1: %#v", len(routes), routes)
	}

	assertRoute(t, routes[0], "GET", "/api/users", "listUsers", true)
}

func TestAnalyzeConvergingFunctionSplitOnce(t *testing.T) {
	tree := analyzer.NewRouteTree()
	if err := Analyze(context.Background(), "../../../testdata/function_converge", tree); err != nil {
		t.Fatal(err)
	}

	routes := analyzer.Flatten(tree)
	if len(routes) != 3 {
		t.Fatalf("len(routes) = %d, want 3: %#v", len(routes), routes)
	}

	assertRoute(t, routes[0], "GET", "/api/health", "health", true)
	assertRoute(t, routes[1], "GET", "/api/users", "listUsers", true)
	assertRoute(t, routes[2], "GET", "/api/admins", "listAdmins", true)
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

func TestAnalyzeConstructorMethodCall(t *testing.T) {
	tree := analyzer.NewRouteTree()
	if err := Analyze(context.Background(), "../../../testdata/constructor_method", tree); err != nil {
		t.Fatal(err)
	}

	routes := analyzer.Flatten(tree)
	if len(routes) != 1 {
		t.Fatalf("len(routes) = %d, want 1: %#v", len(routes), routes)
	}

	assertRoute(t, routes[0], "GET", "/api/users", "listUsers", true)
}

func TestAnalyzeInstanceFields(t *testing.T) {
	tree := analyzer.NewRouteTree()
	if err := Analyze(context.Background(), "../../../testdata/instance_fields", tree); err != nil {
		t.Fatal(err)
	}

	routes := analyzer.Flatten(tree)
	if len(routes) != 2 {
		t.Fatalf("len(routes) = %d, want 2: %#v", len(routes), routes)
	}

	assertRoute(t, routes[0], "GET", "/users", "listUsers", true)
	assertRoute(t, routes[1], "GET", "/admin/stats", "stats", true)
}

func TestAnalyzeChainedConstructorMethod(t *testing.T) {
	tree := analyzer.NewRouteTree()
	if err := Analyze(context.Background(), "../../../testdata/chained_constructor", tree); err != nil {
		t.Fatal(err)
	}

	routes := analyzer.Flatten(tree)
	if len(routes) != 1 {
		t.Fatalf("len(routes) = %d, want 1: %#v", len(routes), routes)
	}

	assertRoute(t, routes[0], "GET", "/api/users", "listUsers", true)
}

func TestAnalyzeReturnValue(t *testing.T) {
	tree := analyzer.NewRouteTree()
	if err := Analyze(context.Background(), "../../../testdata/return_value", tree); err != nil {
		t.Fatal(err)
	}

	routes := analyzer.Flatten(tree)
	if len(routes) != 2 {
		t.Fatalf("len(routes) = %d, want 2: %#v", len(routes), routes)
	}

	assertRoute(t, routes[0], "GET", "/api/users", "listUsers", true)
	assertRoute(t, routes[1], "GET", "/admin/stats", "stats", true)
}

func TestAnalyzeCallback(t *testing.T) {
	tree := analyzer.NewRouteTree()
	if err := Analyze(context.Background(), "../../../testdata/callback", tree); err != nil {
		t.Fatal(err)
	}

	routes := analyzer.Flatten(tree)
	if len(routes) != 1 {
		t.Fatalf("len(routes) = %d, want 1: %#v", len(routes), routes)
	}

	assertRoute(t, routes[0], "GET", "/api/users", "listUsers", true)
}

func TestAnalyzeControlFlow(t *testing.T) {
	tree := analyzer.NewRouteTree()
	if err := Analyze(context.Background(), "../../../testdata/control_flow", tree); err != nil {
		t.Fatal(err)
	}

	routes := analyzer.Flatten(tree)
	if len(routes) != 3 {
		t.Fatalf("len(routes) = %d, want 3: %#v", len(routes), routes)
	}

	assertRoute(t, routes[0], "GET", "/api/users", "listUsers", true)
	assertRoute(t, routes[1], "GET", "/api/fallback", "fallback", true)
	assertRoute(t, routes[2], "GET", "/api/health", "health", true)
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

func TestAnalyzeEnvStructField(t *testing.T) {
	tree := analyzer.NewRouteTree()
	if err := Analyze(context.Background(), "../../../testdata/env_struct_field", tree); err != nil {
		t.Fatal(err)
	}

	routes := analyzer.Flatten(tree)
	if len(routes) != 1 {
		t.Fatalf("len(routes) = %d, want 1: %#v", len(routes), routes)
	}

	assertRoute(t, routes[0], "GET", "/api/users", "users", true)
}

func TestAnalyzeEnvStructMethod(t *testing.T) {
	tree := analyzer.NewRouteTree()
	if err := Analyze(context.Background(), "../../../testdata/env_struct_method", tree); err != nil {
		t.Fatal(err)
	}

	routes := analyzer.Flatten(tree)
	if len(routes) != 1 {
		t.Fatalf("len(routes) = %d, want 1: %#v", len(routes), routes)
	}

	assertRoute(t, routes[0], "GET", "/api/users", "users", true)
}

func TestAnalyzeEnvConstructorMethod(t *testing.T) {
	tree := analyzer.NewRouteTree()
	if err := Analyze(context.Background(), "../../../testdata/env_constructor_method", tree); err != nil {
		t.Fatal(err)
	}

	routes := analyzer.Flatten(tree)
	if len(routes) != 1 {
		t.Fatalf("len(routes) = %d, want 1: %#v", len(routes), routes)
	}

	assertRoute(t, routes[0], "GET", "/api/users", "users", true)
}

func TestAnalyzeEnvChainedConstructor(t *testing.T) {
	tree := analyzer.NewRouteTree()
	if err := Analyze(context.Background(), "../../../testdata/env_chained_constructor", tree); err != nil {
		t.Fatal(err)
	}

	routes := analyzer.Flatten(tree)
	if len(routes) != 1 {
		t.Fatalf("len(routes) = %d, want 1: %#v", len(routes), routes)
	}

	assertRoute(t, routes[0], "GET", "/admin/stats", "stats", true)
}

func TestAnalyzeEnvReturnValue(t *testing.T) {
	tree := analyzer.NewRouteTree()
	if err := Analyze(context.Background(), "../../../testdata/env_return_value", tree); err != nil {
		t.Fatal(err)
	}

	routes := analyzer.Flatten(tree)
	if len(routes) != 2 {
		t.Fatalf("len(routes) = %d, want 2: %#v", len(routes), routes)
	}

	assertRoute(t, routes[0], "GET", "/api/users", "users", true)
	assertRoute(t, routes[1], "GET", "/admin/stats", "stats", true)
}

func TestAnalyzeEnvInstanceIsolation(t *testing.T) {
	tree := analyzer.NewRouteTree()
	if err := Analyze(context.Background(), "../../../testdata/env_instance_isolation", tree); err != nil {
		t.Fatal(err)
	}

	routes := analyzer.Flatten(tree)
	if len(routes) != 2 {
		t.Fatalf("len(routes) = %d, want 2: %#v", len(routes), routes)
	}

	assertRoute(t, routes[0], "GET", "/users", "users", true)
	assertRoute(t, routes[1], "GET", "/admin/stats", "stats", true)
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

func TestAnalyzePackageRouteTable(t *testing.T) {
	tree := analyzer.NewRouteTree()
	if err := Analyze(context.Background(), "../../../testdata/package_route_table", tree); err != nil {
		t.Fatal(err)
	}

	routes := analyzer.Flatten(tree)
	if len(routes) != 3 {
		t.Fatalf("len(routes) = %d, want 3: %#v", len(routes), routes)
	}

	assertRoute(t, routes[0], "GET", "/api/users", "users", true)
	assertRoute(t, routes[1], "POST", "/api/users", "users", true)
	assertRoute(t, routes[2], "DELETE", "/api/users/:id", "deleteUser", true)
}

func TestAnalyzeStaticFileAndRouteNotFound(t *testing.T) {
	tree := analyzer.NewRouteTree()
	if err := Analyze(context.Background(), "../../../testdata/static_file", tree); err != nil {
		t.Fatal(err)
	}

	routes := analyzer.Flatten(tree)
	if len(routes) != 6 {
		t.Fatalf("len(routes) = %d, want 6: %#v", len(routes), routes)
	}

	assertRoute(t, routes[0], "GET", "/assets/*", "<static>", true)
	assertRoute(t, routes[1], "GET", "/", "<file>", true)
	assertRoute(t, routes[2], "ROUTE_NOT_FOUND", "/*", "notFound", true)
	assertRoute(t, routes[3], "GET", "/api/docs/*", "<static>", true)
	assertRoute(t, routes[4], "GET", "/api/openapi.json", "<file>", true)
	assertRoute(t, routes[5], "ROUTE_NOT_FOUND", "/api/*", "apiNotFound", true)
}

func TestAnalyzeStructRouteTable(t *testing.T) {
	tree := analyzer.NewRouteTree()
	if err := Analyze(context.Background(), "../../../testdata/struct_route_table", tree); err != nil {
		t.Fatal(err)
	}

	routes := analyzer.Flatten(tree)
	if len(routes) != 3 {
		t.Fatalf("len(routes) = %d, want 3: %#v", len(routes), routes)
	}

	assertRoute(t, routes[0], "GET", "/api/users", "users", true)
	assertRoute(t, routes[1], "POST", "/api/users", "users", true)
	assertRoute(t, routes[2], "DELETE", "/api/users/:id", "deleteUser", true)
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
	assertWarnings(t, routes[0], "dynamic path expression")
	assertWarnings(t, routes[1], "dynamic path expression")
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

func assertWarnings(t *testing.T, route analyzer.Route, warnings ...string) {
	t.Helper()
	if len(route.Warnings) != len(warnings) {
		t.Fatalf("len(Warnings) = %d, want %d: %#v", len(route.Warnings), len(warnings), route.Warnings)
	}
	for i, want := range warnings {
		if route.Warnings[i] != want {
			t.Fatalf("Warnings[%d] = %q, want %q", i, route.Warnings[i], want)
		}
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
