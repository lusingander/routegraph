package echo

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/lusingander/routegraph/internal/analyzer"
)

func analyzeDecl(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, fieldGroups map[string]analyzer.NodeID, groups map[string]analyzer.NodeID, fields localFieldGroups, env env, stmt *ast.DeclStmt) {
	genDecl, ok := stmt.Decl.(*ast.GenDecl)
	if !ok || genDecl.Tok != token.VAR {
		return
	}
	for _, spec := range genDecl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for i, value := range valueSpec.Values {
			if i >= len(valueSpec.Names) {
				continue
			}
			nodeID, ok := groupCallNodeID(fset, typeInfo, tree, fieldGroups, groups, fields, env, value)
			if ok {
				groups[valueSpec.Names[i].Name] = nodeID
				env.setGroup(valueSpec.Names[i].Name, nodeID)
				continue
			}
			bindValue(env, valueSpec.Names[i].Name, value)
		}
	}
}

func analyzeAssign(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, fieldGroups map[string]analyzer.NodeID, groups map[string]analyzer.NodeID, fields localFieldGroups, env env, stmt *ast.AssignStmt) {
	for i, rhs := range stmt.Rhs {
		if i >= len(stmt.Lhs) {
			continue
		}
		lhs, ok := stmt.Lhs[i].(*ast.Ident)
		if !ok {
			continue
		}

		nodeID, ok := groupCallNodeID(fset, typeInfo, tree, fieldGroups, groups, fields, env, rhs)
		if ok {
			groups[lhs.Name] = nodeID
			env.setGroup(lhs.Name, nodeID)
			continue
		}
		bindValue(env, lhs.Name, rhs)
	}
}

func bindValue(env env, name string, expr ast.Expr) {
	value := evalValue(env, expr)
	switch value.Kind {
	case valueString, valueStrings:
		env.values[name] = value
	}
}

func analyzeExpr(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, fieldGroups map[string]analyzer.NodeID, groups map[string]analyzer.NodeID, fields localFieldGroups, env env, expr ast.Expr) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || len(call.Args) < 2 {
		return
	}
	route, ok := routeCallInfo(selector.Sel.Name, call.Args, env)
	if !ok {
		return
	}

	parentID, ok := routeReceiverNodeID(fset, typeInfo, tree, fieldGroups, groups, fields, env, selector.X)
	if !ok {
		return
	}
	path := pathExprFromEnv(call.Args[route.PathArgIndex], env)
	if route.StaticWildcard {
		path = analyzer.JoinPath(path, analyzer.KnownPath("*"))
	}
	handler := route.HandlerName
	if handler == "" && route.HandlerArgIndex >= 0 {
		handler = handlerName(call.Args[route.HandlerArgIndex])
	}
	for _, method := range route.Methods {
		tree.AddRoute(parentID, analyzer.FrameworkEcho, method, path, handler, position(fset, call.Lparen))
	}
}

func routeReceiverNodeID(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, fieldGroups map[string]analyzer.NodeID, groups map[string]analyzer.NodeID, fields localFieldGroups, env env, expr ast.Expr) (analyzer.NodeID, bool) {
	if nodeID, ok := groupCallNodeID(fset, typeInfo, tree, fieldGroups, groups, fields, env, expr); ok {
		return nodeID, true
	}
	if ident, ok := expr.(*ast.Ident); ok {
		if id, ok := env.group(ident.Name); ok {
			return id, true
		}
	}
	return receiverNodeID(typeInfo, fieldGroups, groups, fields, expr)
}

func groupCallNodeID(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, fieldGroups map[string]analyzer.NodeID, groups map[string]analyzer.NodeID, fields localFieldGroups, env env, expr ast.Expr) (analyzer.NodeID, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok || len(call.Args) == 0 {
		return 0, false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Group" {
		return 0, false
	}
	parentID, ok := receiverNodeID(typeInfo, fieldGroups, groups, fields, selector.X)
	if !ok {
		return 0, false
	}
	path := pathExprFromEnv(call.Args[0], env)
	return tree.AddGroup(parentID, analyzer.FrameworkEcho, path, position(fset, call.Lparen)), true
}
