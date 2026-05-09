package echo

import (
	"go/ast"
	"go/token"
	"strconv"

	"github.com/lusingander/routegraph/internal/analyzer"
)

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
