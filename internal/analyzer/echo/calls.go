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

func analyzeFunc(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, funcs map[*types.Func]*ast.FuncDecl, fieldGroups map[string]analyzer.NodeID, fn *ast.FuncDecl, fileConsts map[string]string, initialGroups map[string]analyzer.NodeID, initialFields localFieldGroups, visiting map[*ast.FuncDecl]bool) {
	if visiting[fn] {
		return
	}
	visiting[fn] = true
	defer delete(visiting, fn)

	groups := cloneGroups(initialGroups)
	fields := cloneLocalFieldGroups(initialFields)
	routeTables := map[string][]routeTableEntry{}
	consts := cloneConsts(fileConsts)
	collectBlockConsts(fn.Body, consts)

	analyzeBlock(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, fields, routeTables, consts, fn.Body, visiting)
}

func analyzeBlock(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, funcs map[*types.Func]*ast.FuncDecl, fieldGroups map[string]analyzer.NodeID, fileConsts map[string]string, groups map[string]analyzer.NodeID, fields localFieldGroups, routeTables map[string][]routeTableEntry, consts map[string]string, block *ast.BlockStmt, visiting map[*ast.FuncDecl]bool) {
	if block == nil {
		return
	}
	for _, stmt := range block.List {
		analyzeStmt(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, fields, routeTables, consts, stmt, visiting)
	}
}

func analyzeStmt(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, funcs map[*types.Func]*ast.FuncDecl, fieldGroups map[string]analyzer.NodeID, fileConsts map[string]string, groups map[string]analyzer.NodeID, fields localFieldGroups, routeTables map[string][]routeTableEntry, consts map[string]string, stmt ast.Stmt, visiting map[*ast.FuncDecl]bool) {
	analyzeStructFields(fset, typeInfo, tree, fieldGroups, groups, fields, consts, stmt)
	switch stmt := stmt.(type) {
	case *ast.DeclStmt:
		analyzeDecl(fset, typeInfo, tree, fieldGroups, groups, fields, consts, stmt)
		analyzeDeclFuncCalls(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, fields, stmt, visiting)
	case *ast.AssignStmt:
		analyzeAssign(fset, typeInfo, tree, fieldGroups, groups, fields, consts, stmt)
		collectRouteTable(routeTables, consts, stmt)
		analyzeAssignFuncCalls(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, fields, consts, stmt, visiting)
	case *ast.ExprStmt:
		analyzeExpr(fset, typeInfo, tree, fieldGroups, groups, fields, consts, stmt.X)
		analyzeFuncCall(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, fields, stmt.X, visiting)
	case *ast.IfStmt:
		if stmt.Init != nil {
			analyzeStmt(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, fields, routeTables, consts, stmt.Init, visiting)
		}
		analyzeBlock(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, fields, routeTables, consts, stmt.Body, visiting)
		analyzeElse(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, fields, routeTables, consts, stmt.Else, visiting)
	case *ast.ForStmt:
		if stmt.Init != nil {
			analyzeStmt(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, fields, routeTables, consts, stmt.Init, visiting)
		}
		analyzeBlock(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, fields, routeTables, consts, stmt.Body, visiting)
		if stmt.Post != nil {
			analyzeStmt(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, fields, routeTables, consts, stmt.Post, visiting)
		}
	case *ast.RangeStmt:
		nodeCount := len(tree.Nodes)
		analyzeRouteTableRange(fset, typeInfo, tree, fieldGroups, groups, fields, routeTables, stmt)
		if len(tree.Nodes) == nodeCount {
			analyzeBlock(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, fields, routeTables, consts, stmt.Body, visiting)
		}
	}
}

func analyzeElse(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, funcs map[*types.Func]*ast.FuncDecl, fieldGroups map[string]analyzer.NodeID, fileConsts map[string]string, groups map[string]analyzer.NodeID, fields localFieldGroups, routeTables map[string][]routeTableEntry, consts map[string]string, stmt ast.Stmt, visiting map[*ast.FuncDecl]bool) {
	switch stmt := stmt.(type) {
	case nil:
		return
	case *ast.BlockStmt:
		analyzeBlock(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, fields, routeTables, consts, stmt, visiting)
	case *ast.IfStmt:
		analyzeStmt(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, fields, routeTables, consts, stmt, visiting)
	}
}

func analyzeDeclFuncCalls(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, funcs map[*types.Func]*ast.FuncDecl, fieldGroups map[string]analyzer.NodeID, fileConsts map[string]string, groups map[string]analyzer.NodeID, fields localFieldGroups, stmt *ast.DeclStmt, visiting map[*ast.FuncDecl]bool) {
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
			analyzeFuncCall(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, fields, value, visiting)
			if i < len(valueSpec.Names) {
				bindStructResult(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, fields, valueSpec.Names[i].Name, value, visiting)
			}
		}
	}
}

func analyzeAssignFuncCalls(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, funcs map[*types.Func]*ast.FuncDecl, fieldGroups map[string]analyzer.NodeID, fileConsts map[string]string, groups map[string]analyzer.NodeID, fields localFieldGroups, consts map[string]string, stmt *ast.AssignStmt, visiting map[*ast.FuncDecl]bool) {
	for i, rhs := range stmt.Rhs {
		analyzeFuncCall(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, fields, rhs, visiting)
		if i >= len(stmt.Lhs) {
			continue
		}
		lhs, ok := stmt.Lhs[i].(*ast.Ident)
		if !ok {
			continue
		}
		bindStructResult(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, fields, lhs.Name, rhs, visiting)
	}
}

func analyzeFuncCall(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, funcs map[*types.Func]*ast.FuncDecl, fieldGroups map[string]analyzer.NodeID, fileConsts map[string]string, groups map[string]analyzer.NodeID, fields localFieldGroups, expr ast.Expr, visiting map[*ast.FuncDecl]bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return
	}

	callee := funcs[calleeFunc(typeInfo, call)]
	if callee == nil || callee.Type.Params == nil {
		return
	}

	initialGroups, initialFields, ok := callBindings(fset, typeInfo, tree, fieldGroups, groups, fields, fileConsts, callee, call)
	if !ok {
		return
	}

	if recvName, recvFields, ok := receiverFieldBinding(fields, callee, call); ok {
		initialFields[recvName] = recvFields
	}
	if len(initialGroups) == 0 && len(initialFields) == 0 {
		return
	}

	analyzeFunc(fset, typeInfo, tree, funcs, fieldGroups, callee, fileConsts, initialGroups, initialFields, visiting)
}

func bindStructResult(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, funcs map[*types.Func]*ast.FuncDecl, fieldGroups map[string]analyzer.NodeID, consts map[string]string, groups map[string]analyzer.NodeID, fields localFieldGroups, name string, expr ast.Expr, visiting map[*ast.FuncDecl]bool) {
	if structFields, _, ok := structLiteralFieldGroups(fset, typeInfo, tree, fieldGroups, groups, fields, consts, expr); ok {
		fields[name] = structFields
		return
	}

	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return
	}
	callee := funcs[calleeFunc(typeInfo, call)]
	if callee == nil {
		return
	}
	initialGroups, initialFields, ok := callBindings(fset, typeInfo, tree, fieldGroups, groups, fields, consts, callee, call)
	if !ok {
		return
	}
	if recvName, recvFields, ok := receiverFieldBinding(fields, callee, call); ok {
		initialFields[recvName] = recvFields
	}
	if returnedFields, ok := returnedStructFields(fset, typeInfo, tree, fieldGroups, consts, callee, initialGroups, initialFields, visiting); ok {
		fields[name] = returnedFields
	}
}

func callBindings(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, fieldGroups map[string]analyzer.NodeID, groups map[string]analyzer.NodeID, fields localFieldGroups, consts map[string]string, callee *ast.FuncDecl, call *ast.CallExpr) (map[string]analyzer.NodeID, localFieldGroups, bool) {
	initialGroups := map[string]analyzer.NodeID{}
	initialFields := localFieldGroups{}
	argIndex := 0
	for _, field := range callee.Type.Params.List {
		for _, name := range field.Names {
			if argIndex >= len(call.Args) {
				return nil, nil, false
			}
			nodeID, ok := argumentNodeID(typeInfo, fieldGroups, groups, fields, call.Args[argIndex])
			if !ok {
				nodeID, ok = groupCallNodeID(fset, typeInfo, tree, fieldGroups, groups, fields, consts, call.Args[argIndex])
			}
			if ok && isEchoParam(typeInfo, name) {
				initialGroups[name.Name] = nodeID
			}
			argIndex++
		}
	}
	return initialGroups, initialFields, true
}

func returnedStructFields(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, fieldGroups map[string]analyzer.NodeID, consts map[string]string, fn *ast.FuncDecl, groups map[string]analyzer.NodeID, fields localFieldGroups, visiting map[*ast.FuncDecl]bool) (map[string]analyzer.NodeID, bool) {
	if visiting[fn] {
		return nil, false
	}
	localGroups := cloneGroups(groups)
	localFields := cloneLocalFieldGroups(fields)
	localConsts := cloneConsts(consts)
	collectBlockConsts(fn.Body, localConsts)
	for _, stmt := range fn.Body.List {
		ret, ok := stmt.(*ast.ReturnStmt)
		if ok {
			for _, result := range ret.Results {
				structFields, _, ok := structLiteralFieldGroups(fset, typeInfo, tree, fieldGroups, localGroups, localFields, localConsts, result)
				if ok {
					return structFields, true
				}
			}
			continue
		}
		analyzeReturnPreludeStmt(fset, typeInfo, tree, fieldGroups, localGroups, localFields, localConsts, stmt)
	}
	return nil, false
}

func analyzeReturnPreludeStmt(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, fieldGroups map[string]analyzer.NodeID, groups map[string]analyzer.NodeID, fields localFieldGroups, consts map[string]string, stmt ast.Stmt) {
	analyzeStructFields(fset, typeInfo, tree, fieldGroups, groups, fields, consts, stmt)
	switch stmt := stmt.(type) {
	case *ast.DeclStmt:
		analyzeDecl(fset, typeInfo, tree, fieldGroups, groups, fields, consts, stmt)
	case *ast.AssignStmt:
		analyzeAssign(fset, typeInfo, tree, fieldGroups, groups, fields, consts, stmt)
		for i, rhs := range stmt.Rhs {
			if i >= len(stmt.Lhs) {
				continue
			}
			lhs, ok := stmt.Lhs[i].(*ast.Ident)
			if !ok {
				continue
			}
			if structFields, _, ok := structLiteralFieldGroups(fset, typeInfo, tree, fieldGroups, groups, fields, consts, rhs); ok {
				fields[lhs.Name] = structFields
			}
		}
	}
}

func receiverFieldBinding(fields localFieldGroups, callee *ast.FuncDecl, call *ast.CallExpr) (string, map[string]analyzer.NodeID, bool) {
	if callee.Recv == nil || len(callee.Recv.List) == 0 || len(callee.Recv.List[0].Names) == 0 {
		return "", nil, false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", nil, false
	}
	ident, ok := selector.X.(*ast.Ident)
	if !ok {
		return "", nil, false
	}
	instanceFields := fields[ident.Name]
	if len(instanceFields) == 0 {
		return "", nil, false
	}
	recvName := callee.Recv.List[0].Names[0].Name
	return recvName, cloneFieldGroup(instanceFields), true
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
