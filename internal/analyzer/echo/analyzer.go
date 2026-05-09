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

	for _, file := range files {
		for _, decl := range file.File.Decls {
			if err := ctx.Err(); err != nil {
				return err
			}
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			analyzeFunc(fset, tree, fn)
		}
	}
	return nil
}

func analyzeFunc(fset *token.FileSet, tree *analyzer.RouteTree, fn *ast.FuncDecl) {
	groups := map[string]analyzer.NodeID{}

	for _, stmt := range fn.Body.List {
		switch stmt := stmt.(type) {
		case *ast.AssignStmt:
			analyzeAssign(fset, tree, groups, stmt)
		case *ast.ExprStmt:
			analyzeExpr(fset, tree, groups, stmt.X)
		}
	}
}

func analyzeAssign(fset *token.FileSet, tree *analyzer.RouteTree, groups map[string]analyzer.NodeID, stmt *ast.AssignStmt) {
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
		path := pathExpr(call.Args[0])
		groups[lhs.Name] = tree.AddGroup(parentID, analyzer.FrameworkEcho, path, position(fset, call.Lparen))
	}
}

func analyzeExpr(fset *token.FileSet, tree *analyzer.RouteTree, groups map[string]analyzer.NodeID, expr ast.Expr) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || len(call.Args) < 2 {
		return
	}
	method, ok := routeMethods[selector.Sel.Name]
	if !ok {
		return
	}

	parentID := receiverNodeID(groups, selector.X)
	path := pathExpr(call.Args[0])
	handler := handlerName(call.Args[1])
	tree.AddRoute(parentID, analyzer.FrameworkEcho, method, path, handler, position(fset, call.Lparen))
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

func pathExpr(expr ast.Expr) analyzer.PathExpr {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return analyzer.UnknownPath("dynamic path expression")
	}
	value, err := strconv.Unquote(lit.Value)
	if err != nil {
		return analyzer.UnknownPath("invalid string literal")
	}
	return analyzer.KnownPath(value)
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
