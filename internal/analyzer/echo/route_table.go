package echo

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/lusingander/routegraph/internal/analyzer"
)

type routeTableEntry struct {
	Method  string
	Path    analyzer.PathExpr
	Handler string
}

func collectRouteTable(routeTables map[string][]routeTableEntry, env env, stmt *ast.AssignStmt) {
	for i, rhs := range stmt.Rhs {
		if i >= len(stmt.Lhs) {
			continue
		}
		lhs, ok := stmt.Lhs[i].(*ast.Ident)
		if !ok {
			continue
		}
		entries, ok := routeTableEntries(rhs, env)
		if !ok {
			continue
		}
		routeTables[lhs.Name] = entries
		env.setRoutes(lhs.Name, entries)
	}
}

func routeTableEntries(expr ast.Expr, env env) ([]routeTableEntry, bool) {
	lit, ok := expr.(*ast.CompositeLit)
	if !ok {
		return nil, false
	}
	arrayType, ok := lit.Type.(*ast.ArrayType)
	if !ok {
		return nil, false
	}
	fieldNames := structFieldNames(arrayType.Elt)
	if len(fieldNames) == 0 {
		return nil, false
	}

	entries := make([]routeTableEntry, 0, len(lit.Elts))
	for _, elt := range lit.Elts {
		entryLit, ok := elt.(*ast.CompositeLit)
		if !ok {
			continue
		}
		fields := map[string]ast.Expr{}
		for i, value := range entryLit.Elts {
			if kv, ok := value.(*ast.KeyValueExpr); ok {
				if key, ok := kv.Key.(*ast.Ident); ok {
					fields[key.Name] = kv.Value
				}
				continue
			}
			if i < len(fieldNames) {
				fields[fieldNames[i]] = value
			}
		}
		method, ok := stringValueFromEnv(fields["method"], env)
		if !ok {
			continue
		}
		path := pathExprFromEnv(fields["path"], env)
		handler := handlerName(fields["handler"])
		entries = append(entries, routeTableEntry{Method: method, Path: path, Handler: handler})
	}
	return entries, len(entries) > 0
}

func structFieldNames(expr ast.Expr) []string {
	structType, ok := expr.(*ast.StructType)
	if !ok {
		return nil
	}
	var names []string
	for _, field := range structType.Fields.List {
		for _, name := range field.Names {
			names = append(names, name.Name)
		}
	}
	return names
}

func analyzeRouteTableRange(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, fieldGroups map[string]analyzer.NodeID, groups map[string]analyzer.NodeID, fields localFieldGroups, routeTables map[string][]routeTableEntry, env env, stmt *ast.RangeStmt) {
	tableIdent, ok := stmt.X.(*ast.Ident)
	if !ok {
		return
	}
	entries := routeTables[tableIdent.Name]
	if len(entries) == 0 {
		entries, _ = env.routes(tableIdent.Name)
	}
	if len(entries) == 0 {
		return
	}
	valueIdent, ok := stmt.Value.(*ast.Ident)
	if !ok {
		return
	}

	for _, bodyStmt := range stmt.Body.List {
		exprStmt, ok := bodyStmt.(*ast.ExprStmt)
		if !ok {
			continue
		}
		call, ok := exprStmt.X.(*ast.CallExpr)
		if !ok || len(call.Args) < 3 {
			continue
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "Add" {
			continue
		}
		parentID, ok := receiverNodeID(typeInfo, fieldGroups, groups, fields, selector.X)
		if !ok {
			continue
		}
		methodField, ok := rangeFieldName(valueIdent.Name, call.Args[0])
		if !ok || methodField != "method" {
			continue
		}
		pathField, ok := rangeFieldName(valueIdent.Name, call.Args[1])
		if !ok || pathField != "path" {
			continue
		}
		handlerField, ok := rangeFieldName(valueIdent.Name, call.Args[2])
		if !ok || handlerField != "handler" {
			continue
		}
		for _, entry := range entries {
			tree.AddRoute(parentID, analyzer.FrameworkEcho, entry.Method, entry.Path, entry.Handler, position(fset, call.Lparen))
		}
	}
}

func rangeFieldName(rangeVar string, expr ast.Expr) (string, bool) {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	ident, ok := selector.X.(*ast.Ident)
	if !ok || ident.Name != rangeVar {
		return "", false
	}
	return selector.Sel.Name, true
}
