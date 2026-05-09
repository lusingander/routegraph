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
	valueStrings valueKind = "strings"
	valueGroup   valueKind = "group"
	valueRoutes  valueKind = "routes"
)

type value struct {
	Kind valueKind

	String  analyzer.PathExpr
	Strings []analyzer.PathExpr
	Group   analyzer.NodeID
	Routes  []routeTableEntry
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

func cloneEnv(e env) env {
	values := make(map[string]value, len(e.values))
	for name, value := range e.values {
		values[name] = value
	}
	return env{
		values: values,
		consts: cloneConsts(e.consts),
	}
}

func (e env) withConsts(consts map[string]string) env {
	next := cloneEnv(e)
	next.consts = cloneConsts(consts)
	return next
}

func (e env) setGroup(name string, id analyzer.NodeID) {
	e.values[name] = groupValueOf(id)
}

func (e env) group(name string) (analyzer.NodeID, bool) {
	value, ok := e.values[name]
	if !ok || value.Kind != valueGroup {
		return 0, false
	}
	return value.Group, true
}

func (e env) setRoutes(name string, routes []routeTableEntry) {
	e.values[name] = routesValueOf(routes)
}

func (e env) routes(name string) ([]routeTableEntry, bool) {
	value, ok := e.values[name]
	if !ok || value.Kind != valueRoutes {
		return nil, false
	}
	return append([]routeTableEntry(nil), value.Routes...), true
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
	case *ast.CompositeLit:
		values, ok := stringSliceValue(e, expr)
		if !ok {
			return unknownValue()
		}
		return stringsValueOf(values)
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

func stringsValueOf(values []analyzer.PathExpr) value {
	return value{
		Kind:    valueStrings,
		Strings: append([]analyzer.PathExpr(nil), values...),
	}
}

func groupValueOf(id analyzer.NodeID) value {
	return value{
		Kind:  valueGroup,
		Group: id,
	}
}

func routesValueOf(routes []routeTableEntry) value {
	return value{
		Kind:   valueRoutes,
		Routes: append([]routeTableEntry(nil), routes...),
	}
}

func unknownValue() value {
	return value{Kind: valueUnknown}
}

func stringSliceValue(e env, lit *ast.CompositeLit) ([]analyzer.PathExpr, bool) {
	if _, ok := lit.Type.(*ast.ArrayType); !ok {
		return nil, false
	}
	values := make([]analyzer.PathExpr, 0, len(lit.Elts))
	for _, elt := range lit.Elts {
		value := evalValue(e, elt)
		if value.Kind != valueString {
			return nil, false
		}
		values = append(values, value.String)
	}
	return values, true
}
