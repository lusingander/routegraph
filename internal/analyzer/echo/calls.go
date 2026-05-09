package echo

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/lusingander/routegraph/internal/analyzer"
)

func collectPackageFuncs(typeInfo *types.Info, files []*ast.File) map[*types.Func]*ast.FuncDecl {
	funcs := map[*types.Func]*ast.FuncDecl{}
	for _, file := range files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			obj, ok := typeInfo.Defs[fn.Name].(*types.Func)
			if !ok {
				continue
			}
			funcs[obj] = fn
		}
	}
	return funcs
}

func analyzeFunc(ctx *analysisContext, fn *ast.FuncDecl, initialGroups map[string]analyzer.NodeID, initialFields localFieldGroups) {
	if ctx.visiting[fn] {
		return
	}
	ctx.visiting[fn] = true
	defer delete(ctx.visiting, fn)

	fnCtx := ctx.withCallBindings(initialGroups, initialFields)
	collectBlockConsts(fn.Body, fnCtx.consts)

	analyzeBlock(fnCtx, fn.Body)
}

func analyzeBlock(ctx *analysisContext, block *ast.BlockStmt) {
	if block == nil {
		return
	}
	for _, stmt := range block.List {
		analyzeStmt(ctx, stmt)
	}
}

func analyzeStmt(ctx *analysisContext, stmt ast.Stmt) {
	analyzeStructFields(ctx.fset, ctx.typeInfo, ctx.tree, ctx.fieldGroups, ctx.groups, ctx.fields, ctx.consts, stmt)
	switch stmt := stmt.(type) {
	case *ast.DeclStmt:
		analyzeDecl(ctx.fset, ctx.typeInfo, ctx.tree, ctx.fieldGroups, ctx.groups, ctx.fields, ctx.consts, stmt)
		analyzeDeclFuncCalls(ctx, stmt)
	case *ast.AssignStmt:
		analyzeAssign(ctx.fset, ctx.typeInfo, ctx.tree, ctx.fieldGroups, ctx.groups, ctx.fields, ctx.consts, stmt)
		collectRouteTable(ctx.routeTables, ctx.consts, stmt)
		analyzeAssignFuncCalls(ctx, stmt)
	case *ast.ExprStmt:
		analyzeExpr(ctx.fset, ctx.typeInfo, ctx.tree, ctx.fieldGroups, ctx.groups, ctx.fields, ctx.consts, stmt.X)
		analyzeFuncCall(ctx, stmt.X)
	case *ast.IfStmt:
		if stmt.Init != nil {
			analyzeStmt(ctx, stmt.Init)
		}
		analyzeBlock(ctx, stmt.Body)
		analyzeElse(ctx, stmt.Else)
	case *ast.ForStmt:
		if stmt.Init != nil {
			analyzeStmt(ctx, stmt.Init)
		}
		analyzeBlock(ctx, stmt.Body)
		if stmt.Post != nil {
			analyzeStmt(ctx, stmt.Post)
		}
	case *ast.RangeStmt:
		nodeCount := len(ctx.tree.Nodes)
		analyzeRouteTableRange(ctx.fset, ctx.typeInfo, ctx.tree, ctx.fieldGroups, ctx.groups, ctx.fields, ctx.routeTables, stmt)
		if len(ctx.tree.Nodes) == nodeCount {
			analyzeBlock(ctx, stmt.Body)
		}
	}
}

func analyzeElse(ctx *analysisContext, stmt ast.Stmt) {
	switch stmt := stmt.(type) {
	case nil:
		return
	case *ast.BlockStmt:
		analyzeBlock(ctx, stmt)
	case *ast.IfStmt:
		analyzeStmt(ctx, stmt)
	}
}

func analyzeDeclFuncCalls(ctx *analysisContext, stmt *ast.DeclStmt) {
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
			analyzeFuncCall(ctx, value)
			if i < len(valueSpec.Names) {
				bindStructResult(ctx, valueSpec.Names[i].Name, value)
			}
		}
	}
}

func analyzeAssignFuncCalls(ctx *analysisContext, stmt *ast.AssignStmt) {
	for i, rhs := range stmt.Rhs {
		analyzeFuncCall(ctx, rhs)
		if i >= len(stmt.Lhs) {
			continue
		}
		lhs, ok := stmt.Lhs[i].(*ast.Ident)
		if !ok {
			continue
		}
		bindStructResult(ctx, lhs.Name, rhs)
	}
}

func analyzeFuncCall(ctx *analysisContext, expr ast.Expr) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return
	}

	callee := ctx.funcs[calleeFunc(ctx.typeInfo, call)]
	if callee == nil || callee.Type.Params == nil {
		return
	}

	initialGroups, initialFields, ok := callBindings(ctx, callee, call)
	if !ok {
		return
	}

	if recvName, recvFields, ok := receiverFieldBinding(ctx, callee, call); ok {
		initialFields[recvName] = recvFields
	}
	if len(initialGroups) == 0 && len(initialFields) == 0 {
		return
	}

	analyzeFunc(ctx, callee, initialGroups, initialFields)
}

func bindStructResult(ctx *analysisContext, name string, expr ast.Expr) {
	if structFields, _, ok := structLiteralFieldGroups(ctx.fset, ctx.typeInfo, ctx.tree, ctx.fieldGroups, ctx.groups, ctx.fields, ctx.consts, expr); ok {
		ctx.fields[name] = structFields
		return
	}

	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return
	}
	callee := ctx.funcs[calleeFunc(ctx.typeInfo, call)]
	if callee == nil {
		return
	}
	initialGroups, initialFields, ok := callBindings(ctx, callee, call)
	if !ok {
		return
	}
	if recvName, recvFields, ok := receiverFieldBinding(ctx, callee, call); ok {
		initialFields[recvName] = recvFields
	}
	if returnedFields, ok := returnedStructFields(ctx, callee, initialGroups, initialFields); ok {
		ctx.fields[name] = returnedFields
	}
}

func callBindings(ctx *analysisContext, callee *ast.FuncDecl, call *ast.CallExpr) (map[string]analyzer.NodeID, localFieldGroups, bool) {
	initialGroups := map[string]analyzer.NodeID{}
	initialFields := localFieldGroups{}
	argIndex := 0
	for _, field := range callee.Type.Params.List {
		for _, name := range field.Names {
			if argIndex >= len(call.Args) {
				return nil, nil, false
			}
			nodeID, ok := argumentNodeID(ctx.typeInfo, ctx.fieldGroups, ctx.groups, ctx.fields, call.Args[argIndex])
			if !ok {
				nodeID, ok = groupCallNodeID(ctx.fset, ctx.typeInfo, ctx.tree, ctx.fieldGroups, ctx.groups, ctx.fields, ctx.consts, call.Args[argIndex])
			}
			if ok && isEchoParam(ctx.typeInfo, name) {
				initialGroups[name.Name] = nodeID
			}
			argIndex++
		}
	}
	return initialGroups, initialFields, true
}

func returnedStructFields(ctx *analysisContext, fn *ast.FuncDecl, groups map[string]analyzer.NodeID, fields localFieldGroups) (map[string]analyzer.NodeID, bool) {
	if ctx.visiting[fn] {
		return nil, false
	}
	localGroups := cloneGroups(groups)
	localFields := cloneLocalFieldGroups(fields)
	localConsts := cloneConsts(ctx.consts)
	collectBlockConsts(fn.Body, localConsts)
	for _, stmt := range fn.Body.List {
		ret, ok := stmt.(*ast.ReturnStmt)
		if ok {
			for _, result := range ret.Results {
				structFields, _, ok := structLiteralFieldGroups(ctx.fset, ctx.typeInfo, ctx.tree, ctx.fieldGroups, localGroups, localFields, localConsts, result)
				if ok {
					return structFields, true
				}
			}
			continue
		}
		analyzeReturnPreludeStmt(ctx, localGroups, localFields, localConsts, stmt)
	}
	return nil, false
}

func analyzeReturnPreludeStmt(ctx *analysisContext, groups map[string]analyzer.NodeID, fields localFieldGroups, consts map[string]string, stmt ast.Stmt) {
	analyzeStructFields(ctx.fset, ctx.typeInfo, ctx.tree, ctx.fieldGroups, groups, fields, consts, stmt)
	switch stmt := stmt.(type) {
	case *ast.DeclStmt:
		analyzeDecl(ctx.fset, ctx.typeInfo, ctx.tree, ctx.fieldGroups, groups, fields, consts, stmt)
	case *ast.AssignStmt:
		analyzeAssign(ctx.fset, ctx.typeInfo, ctx.tree, ctx.fieldGroups, groups, fields, consts, stmt)
		for i, rhs := range stmt.Rhs {
			if i >= len(stmt.Lhs) {
				continue
			}
			lhs, ok := stmt.Lhs[i].(*ast.Ident)
			if !ok {
				continue
			}
			if structFields, _, ok := structLiteralFieldGroups(ctx.fset, ctx.typeInfo, ctx.tree, ctx.fieldGroups, groups, fields, consts, rhs); ok {
				fields[lhs.Name] = structFields
			}
		}
	}
}

func receiverFieldBinding(ctx *analysisContext, callee *ast.FuncDecl, call *ast.CallExpr) (string, map[string]analyzer.NodeID, bool) {
	if callee.Recv == nil || len(callee.Recv.List) == 0 || len(callee.Recv.List[0].Names) == 0 {
		return "", nil, false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", nil, false
	}
	recvName := callee.Recv.List[0].Names[0].Name
	switch receiver := selector.X.(type) {
	case *ast.Ident:
		instanceFields := ctx.fields[receiver.Name]
		if len(instanceFields) == 0 {
			return "", nil, false
		}
		return recvName, cloneFieldGroup(instanceFields), true
	case *ast.CallExpr:
		receiverCallee := ctx.funcs[calleeFunc(ctx.typeInfo, receiver)]
		if receiverCallee == nil {
			return "", nil, false
		}
		initialGroups, initialFields, ok := callBindings(ctx, receiverCallee, receiver)
		if !ok {
			return "", nil, false
		}
		if nestedRecvName, nestedRecvFields, ok := receiverFieldBinding(ctx, receiverCallee, receiver); ok {
			initialFields[nestedRecvName] = nestedRecvFields
		}
		returnedFields, ok := returnedStructFields(ctx, receiverCallee, initialGroups, initialFields)
		if !ok {
			return "", nil, false
		}
		return recvName, returnedFields, true
	default:
		return "", nil, false
	}
}

func calleeFunc(typeInfo *types.Info, call *ast.CallExpr) *types.Func {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		fn, _ := typeInfo.Uses[fun].(*types.Func)
		return fn
	case *ast.SelectorExpr:
		if selection := typeInfo.Selections[fun]; selection != nil {
			fn, _ := selection.Obj().(*types.Func)
			return fn
		}
		fn, _ := typeInfo.Uses[fun.Sel].(*types.Func)
		return fn
	default:
		return nil
	}
}
