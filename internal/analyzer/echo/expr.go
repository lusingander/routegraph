package echo

import (
	"go/ast"
	"go/token"

	"github.com/lusingander/routegraph/internal/analyzer"
)

func pathExpr(expr ast.Expr, consts map[string]string) analyzer.PathExpr {
	return pathExprFromEnv(expr, newEnv(consts))
}

func pathExprFromEnv(expr ast.Expr, env env) analyzer.PathExpr {
	value := evalValue(env, expr)
	if value.Kind == valueString {
		return value.String
	}
	return analyzer.UnknownPath("dynamic path expression")
}

func stringValue(expr ast.Expr, consts map[string]string) (string, bool) {
	return stringValueFromEnv(expr, newEnv(consts))
}

func stringValueFromEnv(expr ast.Expr, env env) (string, bool) {
	value := evalValue(env, expr)
	if value.Kind != valueString || !value.String.Known {
		return "", false
	}
	return value.String.Value, true
}

func stringValuesFromEnv(expr ast.Expr, env env) ([]string, bool) {
	value := evalValue(env, expr)
	if value.Kind != valueStrings {
		return nil, false
	}
	values := make([]string, 0, len(value.Strings))
	for _, item := range value.Strings {
		if !item.Known {
			return nil, false
		}
		values = append(values, item.Value)
	}
	return values, true
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
