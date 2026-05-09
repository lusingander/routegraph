package echo

import (
	"go/ast"
	"go/types"

	"github.com/lusingander/routegraph/internal/analyzer"
)

func argumentNodeID(typeInfo *types.Info, fieldGroups map[string]analyzer.NodeID, groups map[string]analyzer.NodeID, expr ast.Expr) (analyzer.NodeID, bool) {
	kind := echoTypeKind(typeInfo, expr)
	if kind == "" {
		return 0, false
	}
	if fieldSelector, ok := expr.(*ast.SelectorExpr); ok {
		if id, ok := fieldNodeID(typeInfo, fieldGroups, fieldSelector); ok {
			return id, true
		}
	}
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return 0, kind == "Echo"
	}
	if id, ok := groups[ident.Name]; ok {
		return id, true
	}
	return 0, kind == "Echo"
}

func receiverNodeID(typeInfo *types.Info, fieldGroups map[string]analyzer.NodeID, groups map[string]analyzer.NodeID, expr ast.Expr) (analyzer.NodeID, bool) {
	return argumentNodeID(typeInfo, fieldGroups, groups, expr)
}

func isEchoParam(typeInfo *types.Info, ident *ast.Ident) bool {
	return echoObjectKind(typeInfo.ObjectOf(ident)) != ""
}

func echoTypeKind(typeInfo *types.Info, expr ast.Expr) string {
	if typeInfo == nil {
		return ""
	}
	t := typeInfo.TypeOf(expr)
	if t == nil {
		return ""
	}
	return echoTypeName(t)
}

func echoObjectKind(obj types.Object) string {
	if obj == nil {
		return ""
	}
	return echoTypeName(obj.Type())
}

func echoTypeName(t types.Type) string {
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok {
		return ""
	}
	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return ""
	}
	if obj.Pkg().Path() != "github.com/labstack/echo/v4" {
		return ""
	}
	if obj.Name() != "Echo" && obj.Name() != "Group" {
		return ""
	}
	return obj.Name()
}

func fieldNodeID(typeInfo *types.Info, fieldGroups map[string]analyzer.NodeID, selector *ast.SelectorExpr) (analyzer.NodeID, bool) {
	structName := structTypeName(typeInfo.TypeOf(selector.X))
	if structName == "" {
		return 0, false
	}
	id, ok := fieldGroups[structName+"."+selector.Sel.Name]
	return id, ok
}

func structTypeName(t types.Type) string {
	if t == nil {
		return ""
	}
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok {
		return ""
	}
	obj := named.Obj()
	if obj == nil {
		return ""
	}
	return obj.Name()
}
