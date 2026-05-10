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

func analyzeFunc(ctx *analysisContext, fn funcInfo, initialGroups map[string]analyzer.NodeID, initialFields localFieldGroups, initialValues map[string]value) {
	key := analysisKey(fn, initialGroups, initialFields, initialValues)
	if ctx.analyzed[key] {
		return
	}
	ctx.analyzed[key] = true

	if ctx.visiting[fn.decl] {
		return
	}
	ctx.visiting[fn.decl] = true
	defer delete(ctx.visiting, fn.decl)

	fnCtx := ctx.withCallBindings(initialGroups, initialFields, initialValues)
	fnCtx.typeInfo = fn.typeInfo
	fnCtx.fileConsts = fn.fileConsts
	fnCtx.consts = cloneConsts(fn.fileConsts)
	collectBlockConsts(fn.decl.Body, fnCtx.consts)
	fnCtx.env = fnCtx.env.withConsts(fnCtx.consts)

	analyzeBlock(fnCtx, fn.decl.Body)
}

func analysisKey(fn funcInfo, groups map[string]analyzer.NodeID, fields localFieldGroups, values map[string]value) string {
	var builder strings.Builder
	builder.WriteString(funcKey(calleeObject(fn)))
	builder.WriteString("|groups:")
	writeGroupBindings(&builder, groups)
	builder.WriteString("|fields:")
	writeFieldBindings(&builder, fields)
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

func writeFieldBindings(builder *strings.Builder, fields localFieldGroups) {
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		builder.WriteString(name)
		builder.WriteByte('{')
		writeGroupBindings(builder, fields[name])
		builder.WriteString("};")
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
	analyzeStructFields(ctx.fset, ctx.typeInfo, ctx.tree, ctx.fieldGroups, ctx.groups, ctx.fields, ctx.consts, stmt)
	switch stmt := stmt.(type) {
	case *ast.DeclStmt:
		analyzeDecl(ctx.fset, ctx.typeInfo, ctx.tree, ctx.fieldGroups, ctx.groups, ctx.fields, ctx.env, stmt)
		analyzeDeclFuncCalls(ctx, stmt)
	case *ast.AssignStmt:
		analyzeAssign(ctx.fset, ctx.typeInfo, ctx.tree, ctx.fieldGroups, ctx.groups, ctx.fields, ctx.env, stmt)
		collectRouteTable(ctx.routeTables, ctx.env, stmt)
		analyzeAssignFuncCalls(ctx, stmt)
	case *ast.ExprStmt:
		analyzeExpr(ctx.fset, ctx.typeInfo, ctx.tree, ctx.fieldGroups, ctx.groups, ctx.fields, ctx.env, stmt.X)
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
		analyzeRouteTableRange(ctx.fset, ctx.typeInfo, ctx.tree, ctx.fieldGroups, ctx.groups, ctx.fields, ctx.routeTables, ctx.env, stmt)
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
		bindRouteValueCallResult(ctx, lhs.Name, rhs)
		bindStructResult(ctx, lhs.Name, rhs)
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
	value := evalRouteValueInContext(ctx, expr, ctx.groups, ctx.fields, ctx.env)
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

	initialGroups, initialFields, ok := callBindings(ctx, callee, call)
	if !ok {
		return
	}

	initialValues := map[string]value{}
	if recvName, recvValue, ok := receiverValueBinding(ctx, callee.decl, call); ok {
		initialValues[recvName] = recvValue
	}
	if recvName, recvFields, ok := receiverFieldBinding(ctx, callee.decl, call); ok {
		initialFields[recvName] = recvFields
	}
	if len(initialGroups) == 0 && len(initialFields) == 0 && len(initialValues) == 0 {
		return
	}

	analyzeFunc(ctx, callee, initialGroups, initialFields, initialValues)
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
			nodeID, ok = groupCallNodeID(ctx.fset, ctx.typeInfo, ctx.tree, ctx.fieldGroups, ctx.groups, ctx.fields, ctx.env, arg)
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
	litCtx := ctx.withCallBindings(initialGroups, nil, nil)
	collectBlockConsts(lit.Body, litCtx.consts)
	litCtx.env = litCtx.env.withConsts(litCtx.consts)
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
				nodeID, ok = groupCallNodeID(ctx.fset, ctx.typeInfo, ctx.tree, ctx.fieldGroups, groups, fields, ctx.env.withConsts(consts), call.Args[argIndex])
			}
			if ok && isEchoParam(callee.typeInfo, name) {
				initialGroups[name.Name] = nodeID
			}
			argIndex++
		}
	}
	return initialGroups, initialFields, true
}

func evalRouteValueInContext(ctx *analysisContext, expr ast.Expr, groups map[string]analyzer.NodeID, fields localFieldGroups, env env) value {
	value := evalRouteValue(ctx.fset, ctx.typeInfo, ctx.tree, ctx.fieldGroups, groups, fields, env, expr)
	if value.Kind != valueUnknown {
		return value
	}
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return value
	}
	return callReturnValue(ctx, call, groups, fields, env)
}

func callReturnValue(ctx *analysisContext, call *ast.CallExpr, groups map[string]analyzer.NodeID, fields localFieldGroups, env env) value {
	callee := ctx.calleeInfo(call)
	if callee.decl == nil || callee.decl.Type.Params == nil {
		return unknownValue()
	}
	initialValues, ok := callValueBindingsWithEnv(ctx, callee, call, groups, fields, env)
	if !ok {
		return unknownValue()
	}
	if recvName, recvValue, ok := receiverValueBindingWithEnv(ctx, callee.decl, call, groups, fields, env); ok {
		initialValues[recvName] = recvValue
	}
	value, ok := returnedValue(ctx, callee, initialValues)
	if !ok {
		return unknownValue()
	}
	return value
}

func callValueBindingsWithEnv(ctx *analysisContext, callee funcInfo, call *ast.CallExpr, groups map[string]analyzer.NodeID, fields localFieldGroups, env env) (map[string]value, bool) {
	initialValues := map[string]value{}
	argIndex := 0
	for _, field := range callee.decl.Type.Params.List {
		for _, name := range field.Names {
			if argIndex >= len(call.Args) {
				return nil, false
			}
			arg := call.Args[argIndex]
			argValue := evalRouteValueInContext(ctx, arg, groups, fields, env)
			if argValue.Kind == valueUnknown {
				if nodeID, ok := argumentNodeID(ctx.typeInfo, ctx.fieldGroups, groups, fields, arg); ok {
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

func receiverValueBindingWithEnv(ctx *analysisContext, callee *ast.FuncDecl, call *ast.CallExpr, groups map[string]analyzer.NodeID, fields localFieldGroups, env env) (string, value, bool) {
	if callee.Recv == nil || len(callee.Recv.List) == 0 || len(callee.Recv.List[0].Names) == 0 {
		return "", value{}, false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", value{}, false
	}
	receiver := evalRouteValueInContext(ctx, selector.X, groups, fields, env)
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
	fnCtx.fields = localFieldGroups{}
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
				value := evalRouteValueInContext(&fnCtx, result, fnCtx.groups, fnCtx.fields, fnCtx.env)
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
		analyzeDecl(ctx.fset, ctx.typeInfo, ctx.tree, ctx.fieldGroups, groups, fields, ctx.env.withConsts(consts), stmt)
		bindDeclStructResults(ctx, groups, fields, consts, stmt)
	case *ast.AssignStmt:
		analyzeAssign(ctx.fset, ctx.typeInfo, ctx.tree, ctx.fieldGroups, groups, fields, ctx.env.withConsts(consts), stmt)
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

func receiverValueBinding(ctx *analysisContext, callee *ast.FuncDecl, call *ast.CallExpr) (string, value, bool) {
	if callee.Recv == nil || len(callee.Recv.List) == 0 || len(callee.Recv.List[0].Names) == 0 {
		return "", value{}, false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", value{}, false
	}
	receiver := evalRouteValueInContext(ctx, selector.X, ctx.groups, ctx.fields, ctx.env)
	if receiver.Kind == valueUnknown {
		return "", value{}, false
	}
	return callee.Recv.List[0].Names[0].Name, receiver, true
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
