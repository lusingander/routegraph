package echo

import (
	"go/ast"
	"go/token"

	"github.com/lusingander/routegraph/internal/analyzer"
)

func evalRouteValueInContext(ctx *analysisContext, expr ast.Expr, groups map[string]analyzer.NodeID, env env) value {
	value := evalRouteValue(ctx.fset, ctx.typeInfo, ctx.tree, groups, env, expr)
	if value.Kind != valueUnknown {
		return value
	}
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return value
	}
	return callReturnValue(ctx, call, groups, env)
}

func callReturnValue(ctx *analysisContext, call *ast.CallExpr, groups map[string]analyzer.NodeID, env env) value {
	callee := ctx.calleeInfo(call)
	if callee.decl == nil || callee.decl.Type.Params == nil {
		return unknownValue()
	}
	initialValues, ok := callValueBindingsWithEnv(ctx, callee, call, groups, env)
	if !ok {
		return unknownValue()
	}
	if recvName, recvValue, ok := receiverValueBindingWithEnv(ctx, callee.decl, call, groups, env); ok {
		initialValues[recvName] = recvValue
	}
	value, ok := returnedValue(ctx, callee, initialValues)
	if !ok {
		return unknownValue()
	}
	return value
}

func callValueBindingsWithEnv(ctx *analysisContext, callee funcInfo, call *ast.CallExpr, groups map[string]analyzer.NodeID, env env) (map[string]value, bool) {
	initialValues := map[string]value{}
	argIndex := 0
	for _, field := range callee.decl.Type.Params.List {
		for _, name := range field.Names {
			if argIndex >= len(call.Args) {
				return nil, false
			}
			arg := call.Args[argIndex]
			argValue := evalRouteValueInContext(ctx, arg, groups, env)
			if argValue.Kind == valueUnknown {
				if nodeID, ok := argumentNodeID(ctx.typeInfo, groups, arg); ok {
					argValue = groupValueOf(nodeID)
				}
			}
			if argValue.Kind != valueUnknown {
				initialValues[name.Name] = argValue
			}
			argIndex++
		}
	}
	return initialValues, true
}

func receiverValueBindingWithEnv(ctx *analysisContext, callee *ast.FuncDecl, call *ast.CallExpr, groups map[string]analyzer.NodeID, env env) (string, value, bool) {
	if callee.Recv == nil || len(callee.Recv.List) == 0 || len(callee.Recv.List[0].Names) == 0 {
		return "", value{}, false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", value{}, false
	}
	receiver := evalRouteValueInContext(ctx, selector.X, groups, env)
	if receiver.Kind == valueUnknown {
		return "", value{}, false
	}
	return callee.Recv.List[0].Names[0].Name, receiver, true
}

func returnedValue(ctx *analysisContext, fn funcInfo, initialValues map[string]value) (value, bool) {
	if ctx.visiting[fn.decl] {
		return value{}, false
	}
	ctx.visiting[fn.decl] = true
	defer delete(ctx.visiting, fn.decl)

	fnCtx := *ctx
	fnCtx.typeInfo = fn.typeInfo
	fnCtx.fileConsts = fn.fileConsts
	fnCtx.groups = map[string]analyzer.NodeID{}
	fnCtx.routeTables = cloneRouteTables(ctx.routeTables)
	fnCtx.consts = cloneConsts(fn.fileConsts)
	collectBlockConsts(fn.decl.Body, fnCtx.consts)
	fnCtx.env = newEnv(fnCtx.consts)
	for name, value := range initialValues {
		fnCtx.env.values[name] = cloneValue(value)
		if value.Kind == valueGroup {
			fnCtx.groups[name] = value.Group
		}
	}

	for _, stmt := range fn.decl.Body.List {
		ret, ok := stmt.(*ast.ReturnStmt)
		if ok {
			for _, result := range ret.Results {
				value := evalRouteValueInContext(&fnCtx, result, fnCtx.groups, fnCtx.env)
				if value.Kind != valueUnknown {
					return value, true
				}
			}
			continue
		}
		analyzeReturnValuePreludeStmt(&fnCtx, stmt)
	}
	return value{}, false
}

func analyzeReturnValuePreludeStmt(ctx *analysisContext, stmt ast.Stmt) {
	switch stmt := stmt.(type) {
	case *ast.DeclStmt:
		bindDeclRouteValueResults(ctx, stmt)
	case *ast.AssignStmt:
		for i, rhs := range stmt.Rhs {
			if i >= len(stmt.Lhs) {
				continue
			}
			lhs, ok := stmt.Lhs[i].(*ast.Ident)
			if !ok {
				continue
			}
			bindRouteValueResult(ctx, lhs.Name, rhs)
		}
	}
}

func bindDeclRouteValueResults(ctx *analysisContext, stmt *ast.DeclStmt) {
	genDecl, ok := stmt.Decl.(*ast.GenDecl)
	if !ok || genDecl.Tok != token.VAR {
		return
	}
	for _, spec := range genDecl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for i, value := range valueSpec.Values {
			if i >= len(valueSpec.Names) {
				continue
			}
			bindRouteValueResult(ctx, valueSpec.Names[i].Name, value)
		}
	}
}
