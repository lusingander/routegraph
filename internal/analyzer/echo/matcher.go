package echo

import "go/ast"

var routeMethods = map[string]string{
	"GET":     "GET",
	"POST":    "POST",
	"PUT":     "PUT",
	"PATCH":   "PATCH",
	"DELETE":  "DELETE",
	"OPTIONS": "OPTIONS",
	"HEAD":    "HEAD",
}

type routeCall struct {
	Methods         []string
	PathArgIndex    int
	HandlerArgIndex int
}

func routeCallInfo(name string, args []ast.Expr, env env) (routeCall, bool) {
	if method, ok := routeMethods[name]; ok {
		if len(args) < 2 {
			return routeCall{}, false
		}
		return routeCall{
			Methods:         []string{method},
			PathArgIndex:    0,
			HandlerArgIndex: 1,
		}, true
	}
	switch name {
	case "Any":
		if len(args) < 2 {
			return routeCall{}, false
		}
		return routeCall{
			Methods:         []string{"ANY"},
			PathArgIndex:    0,
			HandlerArgIndex: 1,
		}, true
	case "Add":
		if len(args) < 3 {
			return routeCall{}, false
		}
		method, ok := stringValueFromEnv(args[0], env)
		if !ok {
			method = "UNKNOWN"
		}
		return routeCall{
			Methods:         []string{method},
			PathArgIndex:    1,
			HandlerArgIndex: 2,
		}, true
	case "Match":
		if len(args) < 3 {
			return routeCall{}, false
		}
		methods, ok := stringValuesFromEnv(args[0], env)
		if !ok {
			methods = []string{"UNKNOWN"}
		}
		return routeCall{
			Methods:         methods,
			PathArgIndex:    1,
			HandlerArgIndex: 2,
		}, true
	default:
		return routeCall{}, false
	}
}
