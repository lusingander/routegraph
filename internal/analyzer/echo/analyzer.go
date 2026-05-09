package echo

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"

	"github.com/lusingander/routegraph/internal/analyzer"
)

func Analyze(ctx context.Context, dir string, tree *analyzer.RouteTree) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	pkgs, err := analyzer.LoadGoPackages(dir)
	if err != nil {
		return err
	}

	for _, pkg := range pkgs {
		if len(pkg.Pkg.Errors) > 0 {
			return fmt.Errorf("%s", pkg.Pkg.Errors[0])
		}
		pkgConsts := collectPackageConsts(pkg.Pkg.Syntax)
		funcs := collectPackageFuncs(pkg.Pkg.Syntax)
		fieldGroups := map[string]analyzer.NodeID{}
		for _, file := range pkg.Pkg.Syntax {
			if err := ctx.Err(); err != nil {
				return err
			}
			analyzeFile(pkg.Fset, pkg.Pkg.TypesInfo, tree, funcs, fieldGroups, file, pkgConsts)
		}
	}
	return nil
}

func analyzeFile(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, funcs map[string]*ast.FuncDecl, fieldGroups map[string]analyzer.NodeID, file *ast.File, pkgConsts map[string]string) {
	fileConsts := cloneConsts(pkgConsts)
	collectFileConsts(file, fileConsts)
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		analyzeFunc(fset, typeInfo, tree, funcs, fieldGroups, fn, fileConsts, nil, map[string]bool{})
	}
}

func collectPackageFuncs(files []*ast.File) map[string]*ast.FuncDecl {
	funcs := map[string]*ast.FuncDecl{}
	for _, file := range files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || fn.Recv != nil {
				continue
			}
			funcs[fn.Name.Name] = fn
		}
	}
	return funcs
}

func analyzeFunc(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, funcs map[string]*ast.FuncDecl, fieldGroups map[string]analyzer.NodeID, fn *ast.FuncDecl, fileConsts map[string]string, initialGroups map[string]analyzer.NodeID, visiting map[string]bool) {
	if visiting[fn.Name.Name] {
		return
	}
	visiting[fn.Name.Name] = true
	defer delete(visiting, fn.Name.Name)

	groups := cloneGroups(initialGroups)
	routeTables := map[string][]routeTableEntry{}
	consts := cloneConsts(fileConsts)
	collectBlockConsts(fn.Body, consts)

	for _, stmt := range fn.Body.List {
		analyzeStructFields(fset, typeInfo, tree, fieldGroups, groups, consts, stmt)
		switch stmt := stmt.(type) {
		case *ast.AssignStmt:
			analyzeAssign(fset, typeInfo, tree, fieldGroups, groups, consts, stmt)
			collectRouteTable(routeTables, consts, stmt)
		case *ast.ExprStmt:
			analyzeExpr(fset, typeInfo, tree, fieldGroups, groups, consts, stmt.X)
			analyzeFuncCall(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, stmt.X, visiting)
		case *ast.RangeStmt:
			analyzeRouteTableRange(fset, typeInfo, tree, fieldGroups, groups, routeTables, stmt)
		}
	}
}

func analyzeFuncCall(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, funcs map[string]*ast.FuncDecl, fieldGroups map[string]analyzer.NodeID, fileConsts map[string]string, groups map[string]analyzer.NodeID, expr ast.Expr, visiting map[string]bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return
	}
	calleeIdent, ok := call.Fun.(*ast.Ident)
	if !ok {
		return
	}
	callee := funcs[calleeIdent.Name]
	if callee == nil || callee.Type.Params == nil {
		return
	}

	initialGroups := map[string]analyzer.NodeID{}
	argIndex := 0
	for _, field := range callee.Type.Params.List {
		for _, name := range field.Names {
			if argIndex >= len(call.Args) {
				return
			}
			nodeID, ok := argumentNodeID(typeInfo, fieldGroups, groups, call.Args[argIndex])
			if ok && isEchoParam(typeInfo, name) {
				initialGroups[name.Name] = nodeID
			}
			argIndex++
		}
	}
	if len(initialGroups) == 0 {
		return
	}

	analyzeFunc(fset, typeInfo, tree, funcs, fieldGroups, callee, fileConsts, initialGroups, visiting)
}

func analyzeAssign(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, fieldGroups map[string]analyzer.NodeID, groups map[string]analyzer.NodeID, consts map[string]string, stmt *ast.AssignStmt) {
	for i, rhs := range stmt.Rhs {
		call, ok := rhs.(*ast.CallExpr)
		if !ok {
			continue
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "Group" || len(call.Args) == 0 || i >= len(stmt.Lhs) {
			continue
		}
		lhs, ok := stmt.Lhs[i].(*ast.Ident)
		if !ok {
			continue
		}

		parentID, ok := receiverNodeID(typeInfo, fieldGroups, groups, selector.X)
		if !ok {
			continue
		}
		path := pathExpr(call.Args[0], consts)
		groups[lhs.Name] = tree.AddGroup(parentID, analyzer.FrameworkEcho, path, position(fset, call.Lparen))
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

	parentID, ok := receiverNodeID(typeInfo, fieldGroups, groups, selector.X)
	if !ok {
		return
	}
	path := pathExpr(call.Args[pathArgIndex], consts)
	handler := handlerName(call.Args[pathArgIndex+1])
	tree.AddRoute(parentID, analyzer.FrameworkEcho, method, path, handler, position(fset, call.Lparen))
}

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

type routeTableEntry struct {
	Method  string
	Path    analyzer.PathExpr
	Handler string
}

func collectRouteTable(routeTables map[string][]routeTableEntry, consts map[string]string, stmt *ast.AssignStmt) {
	for i, rhs := range stmt.Rhs {
		if i >= len(stmt.Lhs) {
			continue
		}
		lhs, ok := stmt.Lhs[i].(*ast.Ident)
		if !ok {
			continue
		}
		entries, ok := routeTableEntries(rhs, consts)
		if !ok {
			continue
		}
		routeTables[lhs.Name] = entries
	}
}

func routeTableEntries(expr ast.Expr, consts map[string]string) ([]routeTableEntry, bool) {
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
		method, ok := stringValue(fields["method"], consts)
		if !ok {
			continue
		}
		path := pathExpr(fields["path"], consts)
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

func analyzeRouteTableRange(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, fieldGroups map[string]analyzer.NodeID, groups map[string]analyzer.NodeID, routeTables map[string][]routeTableEntry, stmt *ast.RangeStmt) {
	tableIdent, ok := stmt.X.(*ast.Ident)
	if !ok {
		return
	}
	entries := routeTables[tableIdent.Name]
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
		parentID, ok := receiverNodeID(typeInfo, fieldGroups, groups, selector.X)
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

func routeMethod(name string, args []ast.Expr, consts map[string]string) (method string, pathArgIndex int, ok bool) {
	if method, ok := routeMethods[name]; ok {
		return method, 0, true
	}
	switch name {
	case "Any":
		return "ANY", 0, len(args) >= 2
	case "Add":
		if len(args) < 3 {
			return "", 0, false
		}
		method, ok := stringValue(args[0], consts)
		if !ok {
			method = "UNKNOWN"
		}
		return method, 1, true
	default:
		return "", 0, false
	}
}

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

func pathExpr(expr ast.Expr, consts map[string]string) analyzer.PathExpr {
	if value, ok := stringValue(expr, consts); ok {
		return analyzer.KnownPath(value)
	}
	return analyzer.UnknownPath("dynamic path expression")
}

func stringValue(expr ast.Expr, consts map[string]string) (string, bool) {
	switch expr := expr.(type) {
	case *ast.BasicLit:
		if expr.Kind != token.STRING {
			return "", false
		}
		value, err := strconv.Unquote(expr.Value)
		if err != nil {
			return "", false
		}
		return value, true
	case *ast.Ident:
		value, ok := consts[expr.Name]
		return value, ok
	case *ast.BinaryExpr:
		if expr.Op != token.ADD {
			return "", false
		}
		left, ok := stringValue(expr.X, consts)
		if !ok {
			return "", false
		}
		right, ok := stringValue(expr.Y, consts)
		if !ok {
			return "", false
		}
		return left + right, true
	case *ast.ParenExpr:
		return stringValue(expr.X, consts)
	default:
		return "", false
	}
}

func collectPackageConsts(files []*ast.File) map[string]string {
	consts := map[string]string{}
	for _, file := range files {
		collectFileConsts(file, consts)
	}
	return consts
}

func collectFileConsts(file *ast.File, consts map[string]string) {
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			continue
		}
		collectConstSpecs(genDecl.Specs, consts)
	}
}

func collectBlockConsts(block *ast.BlockStmt, consts map[string]string) {
	for _, stmt := range block.List {
		declStmt, ok := stmt.(*ast.DeclStmt)
		if !ok {
			continue
		}
		genDecl, ok := declStmt.Decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			continue
		}
		collectConstSpecs(genDecl.Specs, consts)
	}
}

func collectConstSpecs(specs []ast.Spec, consts map[string]string) {
	var previous []ast.Expr
	for _, spec := range specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		values := valueSpec.Values
		if len(values) == 0 {
			values = previous
		} else {
			previous = values
		}
		for i, name := range valueSpec.Names {
			if i >= len(values) {
				continue
			}
			value, ok := stringValue(values[i], consts)
			if !ok {
				delete(consts, name.Name)
				continue
			}
			consts[name.Name] = value
		}
	}
}

func cloneConsts(consts map[string]string) map[string]string {
	cloned := make(map[string]string, len(consts))
	for name, value := range consts {
		cloned[name] = value
	}
	return cloned
}

func cloneGroups(groups map[string]analyzer.NodeID) map[string]analyzer.NodeID {
	cloned := make(map[string]analyzer.NodeID, len(groups))
	for name, id := range groups {
		cloned[name] = id
	}
	return cloned
}

func handlerName(expr ast.Expr) string {
	switch expr := expr.(type) {
	case *ast.Ident:
		return expr.Name
	case *ast.SelectorExpr:
		return handlerName(expr.X) + "." + expr.Sel.Name
	case *ast.FuncLit:
		return "<func literal>"
	default:
		return "<unknown>"
	}
}

func position(fset *token.FileSet, pos token.Pos) analyzer.Position {
	p := fset.Position(pos)
	return analyzer.Position{File: p.Filename, Line: p.Line}
}
