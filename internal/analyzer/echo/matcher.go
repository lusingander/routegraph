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
	Method          string
	PathArgIndex    int
	HandlerArgIndex int
}

func routeCallInfo(name string, args []ast.Expr, consts map[string]string) (routeCall, bool) {
	if method, ok := routeMethods[name]; ok {
		if len(args) < 2 {
			return routeCall{}, false
		}
		return routeCall{
			Method:          method,
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
			Method:          "ANY",
			PathArgIndex:    0,
			HandlerArgIndex: 1,
		}, true
	case "Add":
		if len(args) < 3 {
			return routeCall{}, false
		}
		method, ok := stringValue(args[0], consts)
		if !ok {
			method = "UNKNOWN"
		}
		return routeCall{
			Method:          method,
			PathArgIndex:    1,
			HandlerArgIndex: 2,
		}, true
	default:
		return routeCall{}, false
	}
}
