package echo

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/lusingander/routegraph/internal/analyzer"
)

type routeTableEntry struct {
	Methods []string
	Path    analyzer.PathExpr
	Handler string
}

func collectPackageRouteTables(files []*ast.File, env env) map[string][]routeTableEntry {
	routeTables := map[string][]routeTableEntry{}
	for _, file := range files {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.VAR {
				continue
			}
			collectRouteTableSpecs(routeTables, env, genDecl.Specs)
		}
	}
	return routeTables
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

func collectRouteTableSpecs(routeTables map[string][]routeTableEntry, env env, specs []ast.Spec) {
	for _, spec := range specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for i, value := range valueSpec.Values {
			if i >= len(valueSpec.Names) {
				continue
			}
			entries, ok := routeTableEntries(value, env)
			if !ok {
				continue
			}
			name := valueSpec.Names[i].Name
			routeTables[name] = entries
			env.setRoutes(name, entries)
		}
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
		methods, ok := routeTableMethods(fields, env)
		if !ok {
			continue
		}
		path := pathExprFromEnv(routeTableField(fields, "path"), env)
		handler := handlerName(routeTableField(fields, "handler"))
		entries = append(entries, routeTableEntry{Methods: methods, Path: path, Handler: handler})
	}
	return entries, len(entries) > 0
}

func routeTableMethods(fields map[string]ast.Expr, env env) ([]string, bool) {
	if expr := routeTableField(fields, "methods"); expr != nil {
		return stringValuesFromEnv(expr, env)
	}
	method, ok := stringValueFromEnv(routeTableField(fields, "method"), env)
	if !ok {
		return nil, false
	}
	return []string{method}, true
}

func routeTableField(fields map[string]ast.Expr, name string) ast.Expr {
	if expr := fields[name]; expr != nil {
		return expr
	}
	if len(name) == 0 {
		return nil
	}
	upperName := string(name[0]-'a'+'A') + name[1:]
	return fields[upperName]
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
		if !ok {
			continue
		}
		parentID, ok := receiverNodeID(typeInfo, fieldGroups, groups, fields, selector.X)
		if !ok {
			continue
		}
		addRouteTableEntries(fset, tree, parentID, entries, valueIdent.Name, selector.Sel.Name, call)
	}
}

func addRouteTableEntries(fset *token.FileSet, tree *analyzer.RouteTree, parentID analyzer.NodeID, entries []routeTableEntry, rangeVar, methodName string, call *ast.CallExpr) {
	switch methodName {
	case "Add":
		if len(call.Args) < 3 {
			return
		}
		methodField, ok := rangeFieldName(rangeVar, call.Args[0])
		if !ok || !sameField(methodField, "method") {
			return
		}
		pathField, ok := rangeFieldName(rangeVar, call.Args[1])
		if !ok || !sameField(pathField, "path") {
			return
		}
		handlerField, ok := rangeFieldName(rangeVar, call.Args[2])
		if !ok || !sameField(handlerField, "handler") {
			return
		}
	case "Match":
		if len(call.Args) < 3 {
			return
		}
		methodsField, ok := rangeFieldName(rangeVar, call.Args[0])
		if !ok || !sameField(methodsField, "methods") {
			return
		}
		pathField, ok := rangeFieldName(rangeVar, call.Args[1])
		if !ok || !sameField(pathField, "path") {
			return
		}
		handlerField, ok := rangeFieldName(rangeVar, call.Args[2])
		if !ok || !sameField(handlerField, "handler") {
			return
		}
	default:
		return
	}
	for _, entry := range entries {
		for _, method := range entry.Methods {
			tree.AddRoute(parentID, analyzer.FrameworkEcho, method, entry.Path, entry.Handler, position(fset, call.Lparen))
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

func sameField(got, want string) bool {
	if got == want {
		return true
	}
	if len(want) == 0 {
		return false
	}
	return got == string(want[0]-'a'+'A')+want[1:]
}
