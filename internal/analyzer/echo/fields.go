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
	structFields, structName, ok := structLiteralFieldGroups(fset, typeInfo, tree, fieldGroups, groups, consts, expr)
	if !ok {
		return
	}
	for field, id := range structFields {
		fieldGroups[structName+"."+field] = id
	}
}

func structLiteralFieldGroups(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, fieldGroups map[string]analyzer.NodeID, groups map[string]analyzer.NodeID, consts map[string]string, expr ast.Expr) (map[string]analyzer.NodeID, string, bool) {
	if unary, ok := expr.(*ast.UnaryExpr); ok && unary.Op == token.AND {
		expr = unary.X
	}
	lit, ok := expr.(*ast.CompositeLit)
	if !ok {
		return nil, "", false
	}
	structName := structTypeName(typeInfo.TypeOf(lit))
	if structName == "" {
		return nil, "", false
	}

	result := map[string]analyzer.NodeID{}
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
			result[field.Name] = id
			continue
		}
		if id, ok := groupCallNodeID(fset, typeInfo, tree, fieldGroups, groups, newEnv(consts), kv.Value); ok {
			result[field.Name] = id
		}
	}
	return result, structName, len(result) > 0
}
