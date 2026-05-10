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
	valueStruct  valueKind = "struct"
)

type value struct {
	Kind valueKind

	String  analyzer.PathExpr
	Strings []analyzer.PathExpr
	Group   analyzer.NodeID
	Routes  []routeTableEntry
	Fields  map[string]value
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
		values[name] = cloneValue(value)
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

func (e env) groupValue(expr ast.Expr) (analyzer.NodeID, bool) {
	value := evalValue(e, expr)
	if value.Kind != valueGroup {
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
		if ok {
			return stringsValueOf(values)
		}
		fields, ok := structFieldsValue(e, expr)
		if ok {
			return structValueOf(fields)
		}
		return unknownValue()
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
	case *ast.SelectorExpr:
		receiver := evalValue(e, expr.X)
		if receiver.Kind != valueStruct {
			return unknownValue()
		}
		field, ok := receiver.Fields[expr.Sel.Name]
		if !ok {
			return unknownValue()
		}
		return field
	default:
		return unknownValue()
	}
}

func cloneValue(v value) value {
	cloned := v
	cloned.Strings = append([]analyzer.PathExpr(nil), v.Strings...)
	cloned.Routes = append([]routeTableEntry(nil), v.Routes...)
	if v.Fields != nil {
		cloned.Fields = make(map[string]value, len(v.Fields))
		for name, field := range v.Fields {
			cloned.Fields[name] = cloneValue(field)
		}
	}
	return cloned
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

func structValueOf(fields map[string]value) value {
	cloned := make(map[string]value, len(fields))
	for name, field := range fields {
		cloned[name] = cloneValue(field)
	}
	return value{
		Kind:   valueStruct,
		Fields: cloned,
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

func structFieldsValue(e env, lit *ast.CompositeLit) (map[string]value, bool) {
	fields := map[string]value{}
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		name, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}
		value := evalValue(e, kv.Value)
		if value.Kind == valueUnknown {
			continue
		}
		fields[name.Name] = value
	}
	return fields, len(fields) > 0
}
