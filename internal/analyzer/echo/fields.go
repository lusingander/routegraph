package echo

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/lusingander/routegraph/internal/analyzer"
)

func analyzeStructFields(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, fieldGroups map[string]analyzer.NodeID, groups map[string]analyzer.NodeID, consts map[string]string, stmt ast.Stmt) {
	switch stmt := stmt.(type) {
	case *ast.ReturnStmt:
		for _, result := range stmt.Results {
			analyzeStructLiteral(fset, typeInfo, tree, fieldGroups, groups, consts, result)
		}
	case *ast.AssignStmt:
		for _, rhs := range stmt.Rhs {
			analyzeStructLiteral(fset, typeInfo, tree, fieldGroups, groups, consts, rhs)
		}
	}
}

func analyzeStructLiteral(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, fieldGroups map[string]analyzer.NodeID, groups map[string]analyzer.NodeID, consts map[string]string, expr ast.Expr) {
	if unary, ok := expr.(*ast.UnaryExpr); ok && unary.Op == token.AND {
		expr = unary.X
	}
	lit, ok := expr.(*ast.CompositeLit)
	if !ok {
		return
	}
	structName := structTypeName(typeInfo.TypeOf(lit))
	if structName == "" {
		return
	}

	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		field, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}
		if id, ok := argumentNodeID(typeInfo, fieldGroups, groups, kv.Value); ok {
			fieldGroups[structName+"."+field.Name] = id
			continue
		}
		call, ok := kv.Value.(*ast.CallExpr)
		if !ok {
			continue
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "Group" || len(call.Args) == 0 {
			continue
		}
		parentID, ok := receiverNodeID(typeInfo, fieldGroups, groups, selector.X)
		if !ok {
			continue
		}
		path := pathExpr(call.Args[0], consts)
		fieldGroups[structName+"."+field.Name] = tree.AddGroup(parentID, analyzer.FrameworkEcho, path, position(fset, call.Lparen))
	}
}
