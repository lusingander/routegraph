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
