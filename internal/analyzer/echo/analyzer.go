package echo

import (
	"context"
	"go/ast"
	"go/token"
	"strconv"

	"github.com/lusingander/routegraph/internal/analyzer"
)

func Analyze(ctx context.Context, dir string, tree *analyzer.RouteTree) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	fset, files, err := analyzer.LoadGoFiles(dir)
	if err != nil {
		return err
	}

	pkgConsts := collectPackageConsts(files)
	for _, file := range files {
		fileConsts := cloneConsts(pkgConsts)
		collectFileConsts(file.File, fileConsts)
		for _, decl := range file.File.Decls {
			if err := ctx.Err(); err != nil {
				return err
			}
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			analyzeFunc(fset, tree, fn, fileConsts)
		}
	}
	return nil
}

func analyzeFunc(fset *token.FileSet, tree *analyzer.RouteTree, fn *ast.FuncDecl, fileConsts map[string]string) {
	groups := map[string]analyzer.NodeID{}
	consts := cloneConsts(fileConsts)
	collectBlockConsts(fn.Body, consts)

	for _, stmt := range fn.Body.List {
		switch stmt := stmt.(type) {
		case *ast.AssignStmt:
			analyzeAssign(fset, tree, groups, consts, stmt)
		case *ast.ExprStmt:
			analyzeExpr(fset, tree, groups, consts, stmt.X)
		}
	}
}

func analyzeAssign(fset *token.FileSet, tree *analyzer.RouteTree, groups map[string]analyzer.NodeID, consts map[string]string, stmt *ast.AssignStmt) {
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

		parentID := receiverNodeID(groups, selector.X)
		path := pathExpr(call.Args[0], consts)
		groups[lhs.Name] = tree.AddGroup(parentID, analyzer.FrameworkEcho, path, position(fset, call.Lparen))
	}
}

func analyzeExpr(fset *token.FileSet, tree *analyzer.RouteTree, groups map[string]analyzer.NodeID, consts map[string]string, expr ast.Expr) {
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

	parentID := receiverNodeID(groups, selector.X)
	path := pathExpr(call.Args[pathArgIndex], consts)
	handler := handlerName(call.Args[pathArgIndex+1])
	tree.AddRoute(parentID, analyzer.FrameworkEcho, method, path, handler, position(fset, call.Lparen))
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

func receiverNodeID(groups map[string]analyzer.NodeID, expr ast.Expr) analyzer.NodeID {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return 0
	}
	if id, ok := groups[ident.Name]; ok {
		return id
	}
	return 0
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

func collectPackageConsts(files []analyzer.GoFile) map[string]string {
	consts := map[string]string{}
	for _, file := range files {
		collectFileConsts(file.File, consts)
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
