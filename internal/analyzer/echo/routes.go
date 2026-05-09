package echo

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/lusingander/routegraph/internal/analyzer"
)

func analyzeDecl(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, fieldGroups map[string]analyzer.NodeID, groups map[string]analyzer.NodeID, consts map[string]string, stmt *ast.DeclStmt) {
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
			nodeID, ok := groupCallNodeID(fset, typeInfo, tree, fieldGroups, groups, consts, value)
			if !ok {
				continue
			}
			groups[valueSpec.Names[i].Name] = nodeID
		}
	}
}

func analyzeAssign(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, fieldGroups map[string]analyzer.NodeID, groups map[string]analyzer.NodeID, consts map[string]string, stmt *ast.AssignStmt) {
	for i, rhs := range stmt.Rhs {
		if i >= len(stmt.Lhs) {
			continue
		}
		lhs, ok := stmt.Lhs[i].(*ast.Ident)
		if !ok {
			continue
		}

		nodeID, ok := groupCallNodeID(fset, typeInfo, tree, fieldGroups, groups, consts, rhs)
		if !ok {
			continue
		}
		groups[lhs.Name] = nodeID
	}
}

func analyzeExpr(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, fieldGroups map[string]analyzer.NodeID, groups map[string]analyzer.NodeID, consts map[string]string, expr ast.Expr) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || len(call.Args) < 2 {
		return
	}
	method, pathArgIndex, ok := routeMethod(selector.Sel.Name, call.Args, consts)
	if !ok {
		return
	}

	parentID, ok := routeReceiverNodeID(fset, typeInfo, tree, fieldGroups, groups, consts, selector.X)
	if !ok {
		return
	}
	path := pathExpr(call.Args[pathArgIndex], consts)
	handler := handlerName(call.Args[pathArgIndex+1])
	tree.AddRoute(parentID, analyzer.FrameworkEcho, method, path, handler, position(fset, call.Lparen))
}

func routeReceiverNodeID(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, fieldGroups map[string]analyzer.NodeID, groups map[string]analyzer.NodeID, consts map[string]string, expr ast.Expr) (analyzer.NodeID, bool) {
	if nodeID, ok := groupCallNodeID(fset, typeInfo, tree, fieldGroups, groups, consts, expr); ok {
		return nodeID, true
	}
	return receiverNodeID(typeInfo, fieldGroups, groups, expr)
}

func groupCallNodeID(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, fieldGroups map[string]analyzer.NodeID, groups map[string]analyzer.NodeID, consts map[string]string, expr ast.Expr) (analyzer.NodeID, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok || len(call.Args) == 0 {
		return 0, false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Group" {
		return 0, false
	}
	parentID, ok := receiverNodeID(typeInfo, fieldGroups, groups, selector.X)
	if !ok {
		return 0, false
	}
	path := pathExpr(call.Args[0], consts)
	return tree.AddGroup(parentID, analyzer.FrameworkEcho, path, position(fset, call.Lparen)), true
}
