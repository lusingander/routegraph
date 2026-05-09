package echo

import (
	"go/ast"
	"go/token"
	"strconv"

	"github.com/lusingander/routegraph/internal/analyzer"
)

type valueKind string

const (
	valueUnknown valueKind = "unknown"
	valueString  valueKind = "string"
)

type value struct {
	Kind valueKind

	String analyzer.PathExpr
}

type env struct {
	values map[string]value
	consts map[string]string
}

func newEnv(consts map[string]string) env {
	return env{
		values: map[string]value{},
		consts: consts,
	}
}

func evalValue(e env, expr ast.Expr) value {
	switch expr := expr.(type) {
	case *ast.BasicLit:
		if expr.Kind != token.STRING {
			return unknownValue()
		}
		value, err := strconv.Unquote(expr.Value)
		if err != nil {
			return unknownValue()
		}
		return stringValueOf(analyzer.KnownPath(value))
	case *ast.Ident:
		if value, ok := e.values[expr.Name]; ok {
			return value
		}
		if value, ok := e.consts[expr.Name]; ok {
			return stringValueOf(analyzer.KnownPath(value))
		}
		return unknownValue()
	case *ast.BinaryExpr:
		if expr.Op != token.ADD {
			return unknownValue()
		}
		left := evalValue(e, expr.X)
		right := evalValue(e, expr.Y)
		if left.Kind != valueString || right.Kind != valueString {
			return unknownValue()
		}
		if !left.String.Known || !right.String.Known {
			return unknownValue()
		}
		return stringValueOf(analyzer.KnownPath(left.String.Value + right.String.Value))
	case *ast.ParenExpr:
		return evalValue(e, expr.X)
	default:
		return unknownValue()
	}
}

func stringValueOf(path analyzer.PathExpr) value {
	return value{
		Kind:   valueString,
		String: path,
	}
}

func unknownValue() value {
	return value{Kind: valueUnknown}
}
