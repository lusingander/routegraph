package echo

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/lusingander/routegraph/internal/analyzer"
)

func collectPackageFuncs(typeInfo *types.Info, files []*ast.File, fileConsts map[string]string) map[*types.Func]funcInfo {
	funcs := map[*types.Func]funcInfo{}
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
			funcs[obj] = funcInfo{
				decl:       fn,
				typeInfo:   typeInfo,
				fileConsts: fileConsts,
			}
		}
	}
	return funcs
}

func analyzeFunc(ctx *analysisContext, fn funcInfo, initialGroups map[string]analyzer.NodeID, initialFields localFieldGroups) {
	if ctx.visiting[fn.decl] {
		return
	}
	ctx.visiting[fn.decl] = true
	defer delete(ctx.visiting, fn.decl)

	fnCtx := ctx.withCallBindings(initialGroups, initialFields)
	fnCtx.typeInfo = fn.typeInfo
	fnCtx.fileConsts = fn.fileConsts
	fnCtx.consts = cloneConsts(fn.fileConsts)
	collectBlockConsts(fn.decl.Body, fnCtx.consts)

	analyzeBlock(fnCtx, fn.decl.Body)
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
	analyzeCallbackArgs(ctx, call)

	callee := ctx.calleeInfo(call)
	if callee.decl == nil || callee.decl.Type.Params == nil {
		return
	}

	initialGroups, initialFields, ok := callBindings(ctx, callee, call)
	if !ok {
		return
	}

	if recvName, recvFields, ok := receiverFieldBinding(ctx, callee.decl, call); ok {
		initialFields[recvName] = recvFields
	}
	if len(initialGroups) == 0 && len(initialFields) == 0 {
		return
	}

	analyzeFunc(ctx, callee, initialGroups, initialFields)
}

func analyzeCallbackArgs(ctx *analysisContext, call *ast.CallExpr) {
	groupArgs := callGroupArgs(ctx, call)
	if len(groupArgs) == 0 {
		return
	}
	for _, arg := range call.Args {
		lit, ok := arg.(*ast.FuncLit)
		if !ok {
			continue
		}
		initialGroups := funcLiteralGroups(ctx, lit, groupArgs)
		if len(initialGroups) == 0 {
			continue
		}
		analyzeFuncLiteral(ctx, lit, initialGroups)
	}
}

func callGroupArgs(ctx *analysisContext, call *ast.CallExpr) []analyzer.NodeID {
	var groupArgs []analyzer.NodeID
	for _, arg := range call.Args {
		if _, ok := arg.(*ast.FuncLit); ok {
			continue
		}
		nodeID, ok := argumentNodeID(ctx.typeInfo, ctx.fieldGroups, ctx.groups, ctx.fields, arg)
		if !ok {
			nodeID, ok = groupCallNodeID(ctx.fset, ctx.typeInfo, ctx.tree, ctx.fieldGroups, ctx.groups, ctx.fields, ctx.consts, arg)
		}
		if ok {
			groupArgs = append(groupArgs, nodeID)
		}
	}
	return groupArgs
}

func funcLiteralGroups(ctx *analysisContext, lit *ast.FuncLit, groupArgs []analyzer.NodeID) map[string]analyzer.NodeID {
	if lit.Type.Params == nil {
		return nil
	}
	groups := map[string]analyzer.NodeID{}
	groupIndex := 0
	for _, field := range lit.Type.Params.List {
		for _, name := range field.Names {
			if groupIndex >= len(groupArgs) {
				return groups
			}
			if isEchoParam(ctx.typeInfo, name) {
				groups[name.Name] = groupArgs[groupIndex]
				groupIndex++
			}
		}
	}
	return groups
}

func analyzeFuncLiteral(ctx *analysisContext, lit *ast.FuncLit, initialGroups map[string]analyzer.NodeID) {
	litCtx := ctx.withCallBindings(initialGroups, nil)
	collectBlockConsts(lit.Body, litCtx.consts)
	analyzeBlock(litCtx, lit.Body)
}

func bindStructResult(ctx *analysisContext, name string, expr ast.Expr) {
	if structFields, ok := structResultFields(ctx, expr, ctx.groups, ctx.fields, ctx.consts); ok {
		ctx.fields[name] = structFields
	}
}

func callBindings(ctx *analysisContext, callee funcInfo, call *ast.CallExpr) (map[string]analyzer.NodeID, localFieldGroups, bool) {
	return callBindingsWithEnv(ctx, callee, call, ctx.groups, ctx.fields, ctx.consts)
}

func callBindingsWithEnv(ctx *analysisContext, callee funcInfo, call *ast.CallExpr, groups map[string]analyzer.NodeID, fields localFieldGroups, consts map[string]string) (map[string]analyzer.NodeID, localFieldGroups, bool) {
	initialGroups := map[string]analyzer.NodeID{}
	initialFields := localFieldGroups{}
	argIndex := 0
	for _, field := range callee.decl.Type.Params.List {
		for _, name := range field.Names {
			if argIndex >= len(call.Args) {
				return nil, nil, false
			}
			nodeID, ok := argumentNodeID(ctx.typeInfo, ctx.fieldGroups, groups, fields, call.Args[argIndex])
			if !ok {
				nodeID, ok = groupCallNodeID(ctx.fset, ctx.typeInfo, ctx.tree, ctx.fieldGroups, groups, fields, consts, call.Args[argIndex])
			}
			if ok && isEchoParam(callee.typeInfo, name) {
				initialGroups[name.Name] = nodeID
			}
			argIndex++
		}
	}
	return initialGroups, initialFields, true
}

func returnedStructFields(ctx *analysisContext, fn funcInfo, groups map[string]analyzer.NodeID, fields localFieldGroups) (map[string]analyzer.NodeID, bool) {
	if ctx.visiting[fn.decl] {
		return nil, false
	}
	fnCtx := *ctx
	fnCtx.typeInfo = fn.typeInfo
	fnCtx.fileConsts = fn.fileConsts
	localGroups := cloneGroups(groups)
	localFields := cloneLocalFieldGroups(fields)
	localConsts := cloneConsts(fn.fileConsts)
	collectBlockConsts(fn.decl.Body, localConsts)
	for _, stmt := range fn.decl.Body.List {
		ret, ok := stmt.(*ast.ReturnStmt)
		if ok {
			for _, result := range ret.Results {
				if structFields, ok := structResultFields(&fnCtx, result, localGroups, localFields, localConsts); ok {
					return structFields, true
				}
			}
			continue
		}
		analyzeReturnPreludeStmt(&fnCtx, localGroups, localFields, localConsts, stmt)
	}
	return nil, false
}

func structResultFields(ctx *analysisContext, expr ast.Expr, groups map[string]analyzer.NodeID, fields localFieldGroups, consts map[string]string) (map[string]analyzer.NodeID, bool) {
	if structFields, _, ok := structLiteralFieldGroups(ctx.fset, ctx.typeInfo, ctx.tree, ctx.fieldGroups, groups, fields, consts, expr); ok {
		return structFields, true
	}
	if ident, ok := expr.(*ast.Ident); ok {
		structFields := fields[ident.Name]
		if len(structFields) > 0 {
			return cloneFieldGroup(structFields), true
		}
		return nil, false
	}

	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil, false
	}
	callee := ctx.calleeInfo(call)
	if callee.decl == nil {
		return nil, false
	}
	initialGroups, initialFields, ok := callBindingsWithEnv(ctx, callee, call, groups, fields, consts)
	if !ok {
		return nil, false
	}
	if recvName, recvFields, ok := receiverFieldBindingWithEnv(ctx, callee.decl, call, groups, fields, consts); ok {
		initialFields[recvName] = recvFields
	}
	return returnedStructFields(ctx, callee, initialGroups, initialFields)
}

func analyzeReturnPreludeStmt(ctx *analysisContext, groups map[string]analyzer.NodeID, fields localFieldGroups, consts map[string]string, stmt ast.Stmt) {
	analyzeStructFields(ctx.fset, ctx.typeInfo, ctx.tree, ctx.fieldGroups, groups, fields, consts, stmt)
	switch stmt := stmt.(type) {
	case *ast.DeclStmt:
		analyzeDecl(ctx.fset, ctx.typeInfo, ctx.tree, ctx.fieldGroups, groups, fields, consts, stmt)
		bindDeclStructResults(ctx, groups, fields, consts, stmt)
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
			if structFields, ok := structResultFields(ctx, rhs, groups, fields, consts); ok {
				fields[lhs.Name] = structFields
			}
		}
	}
}

func bindDeclStructResults(ctx *analysisContext, groups map[string]analyzer.NodeID, fields localFieldGroups, consts map[string]string, stmt *ast.DeclStmt) {
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
			if structFields, ok := structResultFields(ctx, value, groups, fields, consts); ok {
				fields[valueSpec.Names[i].Name] = structFields
			}
		}
	}
}

func receiverFieldBinding(ctx *analysisContext, callee *ast.FuncDecl, call *ast.CallExpr) (string, map[string]analyzer.NodeID, bool) {
	return receiverFieldBindingWithEnv(ctx, callee, call, ctx.groups, ctx.fields, ctx.consts)
}

func receiverFieldBindingWithEnv(ctx *analysisContext, callee *ast.FuncDecl, call *ast.CallExpr, groups map[string]analyzer.NodeID, fields localFieldGroups, consts map[string]string) (string, map[string]analyzer.NodeID, bool) {
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
		instanceFields := fields[receiver.Name]
		if len(instanceFields) == 0 {
			return "", nil, false
		}
		return recvName, cloneFieldGroup(instanceFields), true
	case *ast.CallExpr:
		receiverCallee := ctx.calleeInfo(receiver)
		if receiverCallee.decl == nil {
			return "", nil, false
		}
		initialGroups, initialFields, ok := callBindingsWithEnv(ctx, receiverCallee, receiver, groups, fields, consts)
		if !ok {
			return "", nil, false
		}
		if nestedRecvName, nestedRecvFields, ok := receiverFieldBindingWithEnv(ctx, receiverCallee.decl, receiver, groups, fields, consts); ok {
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

func (ctx *analysisContext) calleeInfo(call *ast.CallExpr) funcInfo {
	fn := calleeFunc(ctx.typeInfo, call)
	if info := ctx.funcs[fn]; info.decl != nil {
		return info
	}
	return ctx.funcNames[funcKey(fn)]
}

func funcKey(fn *types.Func) string {
	if fn == nil {
		return ""
	}
	return fn.FullName()
}
