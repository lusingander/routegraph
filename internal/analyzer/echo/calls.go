package echo

import (
	"go/ast"
	"go/token"
	"go/types"
	"sort"
	"strconv"
	"strings"

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

func analyzeFunc(ctx *analysisContext, fn funcInfo, initialGroups map[string]analyzer.NodeID, initialValues map[string]value) {
	key := analysisKey(fn, initialGroups, initialValues)
	if ctx.analyzed[key] {
		return
	}
	ctx.analyzed[key] = true

	if ctx.visiting[fn.decl] {
		return
	}
	ctx.visiting[fn.decl] = true
	defer delete(ctx.visiting, fn.decl)

	fnCtx := ctx.withCallBindings(initialGroups, initialValues)
	fnCtx.typeInfo = fn.typeInfo
	fnCtx.fileConsts = fn.fileConsts
	fnCtx.consts = cloneConsts(fn.fileConsts)
	collectBlockConsts(fn.decl.Body, fnCtx.consts)
	fnCtx.env = fnCtx.env.withConsts(fnCtx.consts)

	analyzeBlock(fnCtx, fn.decl.Body)
}

func analysisKey(fn funcInfo, groups map[string]analyzer.NodeID, values map[string]value) string {
	var builder strings.Builder
	builder.WriteString(funcKey(calleeObject(fn)))
	builder.WriteString("|groups:")
	writeGroupBindings(&builder, groups)
	builder.WriteString("|values:")
	writeValueBindings(&builder, values)
	return builder.String()
}

func calleeObject(fn funcInfo) *types.Func {
	if fn.decl == nil || fn.typeInfo == nil {
		return nil
	}
	if fn.decl.Recv != nil {
		if obj, ok := fn.typeInfo.Defs[fn.decl.Name].(*types.Func); ok {
			return obj
		}
	}
	if obj, ok := fn.typeInfo.Defs[fn.decl.Name].(*types.Func); ok {
		return obj
	}
	return nil
}

func writeGroupBindings(builder *strings.Builder, groups map[string]analyzer.NodeID) {
	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		builder.WriteString(name)
		builder.WriteByte('=')
		builder.WriteString(strconv.Itoa(int(groups[name])))
		builder.WriteByte(';')
	}
}

func writeValueBindings(builder *strings.Builder, values map[string]value) {
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		builder.WriteString(name)
		builder.WriteByte('=')
		writeValueKey(builder, values[name])
		builder.WriteByte(';')
	}
}

func writeValueKey(builder *strings.Builder, value value) {
	builder.WriteString(string(value.Kind))
	switch value.Kind {
	case valueString:
		builder.WriteByte(':')
		builder.WriteString(value.String.Value)
	case valueStrings:
		for _, item := range value.Strings {
			builder.WriteByte(':')
			builder.WriteString(item.Value)
		}
	case valueGroup:
		builder.WriteByte(':')
		builder.WriteString(strconv.Itoa(int(value.Group)))
	case valueRoutes:
		builder.WriteByte(':')
		builder.WriteString(strconv.Itoa(len(value.Routes)))
	case valueStruct:
		fieldNames := make([]string, 0, len(value.Fields))
		for name := range value.Fields {
			fieldNames = append(fieldNames, name)
		}
		sort.Strings(fieldNames)
		for _, name := range fieldNames {
			builder.WriteByte(':')
			builder.WriteString(name)
			builder.WriteByte('=')
			writeValueKey(builder, value.Fields[name])
		}
	}
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
	switch stmt := stmt.(type) {
	case *ast.DeclStmt:
		analyzeDecl(ctx.fset, ctx.typeInfo, ctx.tree, ctx.groups, ctx.env, stmt)
		analyzeDeclFuncCalls(ctx, stmt)
	case *ast.AssignStmt:
		analyzeAssign(ctx.fset, ctx.typeInfo, ctx.tree, ctx.groups, ctx.env, stmt)
		collectRouteTable(ctx.routeTables, ctx.env, stmt)
		analyzeAssignFuncCalls(ctx, stmt)
	case *ast.ExprStmt:
		analyzeExpr(ctx.fset, ctx.typeInfo, ctx.tree, ctx.groups, ctx.env, stmt.X)
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
		analyzeRouteTableRange(ctx.fset, ctx.typeInfo, ctx.tree, ctx.groups, ctx.routeTables, ctx.env, stmt)
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
				bindRouteValueCallResult(ctx, valueSpec.Names[i].Name, value)
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
		bindRouteValueCallResult(ctx, lhs.Name, rhs)
	}
}

func bindRouteValueCallResult(ctx *analysisContext, name string, expr ast.Expr) {
	call, ok := expr.(*ast.CallExpr)
	if !ok || isGroupCall(call) {
		return
	}
	bindRouteValueResult(ctx, name, expr)
}

func bindRouteValueResult(ctx *analysisContext, name string, expr ast.Expr) {
	value := evalRouteValueInContext(ctx, expr, ctx.groups, ctx.env)
	switch value.Kind {
	case valueString, valueStrings, valueGroup, valueRoutes, valueStruct:
		ctx.env.values[name] = value
		if value.Kind == valueGroup {
			ctx.groups[name] = value.Group
		}
	}
}

func isGroupCall(call *ast.CallExpr) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	return ok && selector.Sel.Name == "Group"
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

	initialGroups, ok := callBindings(ctx, callee, call)
	if !ok {
		return
	}

	initialValues := map[string]value{}
	if recvName, recvValue, ok := receiverValueBinding(ctx, callee.decl, call); ok {
		initialValues[recvName] = recvValue
	}
	if len(initialGroups) == 0 && len(initialValues) == 0 {
		return
	}

	analyzeFunc(ctx, callee, initialGroups, initialValues)
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
		nodeID, ok := argumentNodeID(ctx.typeInfo, ctx.groups, arg)
		if !ok {
			nodeID, ok = groupCallNodeID(ctx.fset, ctx.typeInfo, ctx.tree, ctx.groups, ctx.env, arg)
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
	litCtx.env = litCtx.env.withConsts(litCtx.consts)
	analyzeBlock(litCtx, lit.Body)
}

func callBindings(ctx *analysisContext, callee funcInfo, call *ast.CallExpr) (map[string]analyzer.NodeID, bool) {
	return callBindingsWithEnv(ctx, callee, call, ctx.groups, ctx.consts)
}

func callBindingsWithEnv(ctx *analysisContext, callee funcInfo, call *ast.CallExpr, groups map[string]analyzer.NodeID, consts map[string]string) (map[string]analyzer.NodeID, bool) {
	initialGroups := map[string]analyzer.NodeID{}
	argIndex := 0
	for _, field := range callee.decl.Type.Params.List {
		for _, name := range field.Names {
			if argIndex >= len(call.Args) {
				return nil, false
			}
			nodeID, ok := argumentNodeID(ctx.typeInfo, groups, call.Args[argIndex])
			if !ok {
				nodeID, ok = groupCallNodeID(ctx.fset, ctx.typeInfo, ctx.tree, groups, ctx.env.withConsts(consts), call.Args[argIndex])
			}
			if ok && isEchoParam(callee.typeInfo, name) {
				initialGroups[name.Name] = nodeID
			}
			argIndex++
		}
	}
	return initialGroups, true
}

func receiverValueBinding(ctx *analysisContext, callee *ast.FuncDecl, call *ast.CallExpr) (string, value, bool) {
	if callee.Recv == nil || len(callee.Recv.List) == 0 || len(callee.Recv.List[0].Names) == 0 {
		return "", value{}, false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", value{}, false
	}
	receiver := evalRouteValueInContext(ctx, selector.X, ctx.groups, ctx.env)
	if receiver.Kind == valueUnknown {
		return "", value{}, false
	}
	return callee.Recv.List[0].Names[0].Name, receiver, true
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
