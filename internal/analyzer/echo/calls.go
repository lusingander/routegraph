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

func analyzeFunc(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, funcs map[*types.Func]*ast.FuncDecl, fieldGroups map[string]analyzer.NodeID, fn *ast.FuncDecl, fileConsts map[string]string, initialGroups map[string]analyzer.NodeID, visiting map[*ast.FuncDecl]bool) {
	if visiting[fn] {
		return
	}
	visiting[fn] = true
	defer delete(visiting, fn)

	groups := cloneGroups(initialGroups)
	routeTables := map[string][]routeTableEntry{}
	consts := cloneConsts(fileConsts)
	collectBlockConsts(fn.Body, consts)

	analyzeBlock(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, routeTables, consts, fn.Body, visiting)
}

func analyzeBlock(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, funcs map[*types.Func]*ast.FuncDecl, fieldGroups map[string]analyzer.NodeID, fileConsts map[string]string, groups map[string]analyzer.NodeID, routeTables map[string][]routeTableEntry, consts map[string]string, block *ast.BlockStmt, visiting map[*ast.FuncDecl]bool) {
	if block == nil {
		return
	}
	for _, stmt := range block.List {
		analyzeStmt(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, routeTables, consts, stmt, visiting)
	}
}

func analyzeStmt(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, funcs map[*types.Func]*ast.FuncDecl, fieldGroups map[string]analyzer.NodeID, fileConsts map[string]string, groups map[string]analyzer.NodeID, routeTables map[string][]routeTableEntry, consts map[string]string, stmt ast.Stmt, visiting map[*ast.FuncDecl]bool) {
	analyzeStructFields(fset, typeInfo, tree, fieldGroups, groups, consts, stmt)
	switch stmt := stmt.(type) {
	case *ast.DeclStmt:
		analyzeDecl(fset, typeInfo, tree, fieldGroups, groups, consts, stmt)
		analyzeDeclFuncCalls(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, stmt, visiting)
	case *ast.AssignStmt:
		analyzeAssign(fset, typeInfo, tree, fieldGroups, groups, consts, stmt)
		collectRouteTable(routeTables, consts, stmt)
		analyzeAssignFuncCalls(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, stmt, visiting)
	case *ast.ExprStmt:
		analyzeExpr(fset, typeInfo, tree, fieldGroups, groups, consts, stmt.X)
		analyzeFuncCall(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, stmt.X, visiting)
	case *ast.IfStmt:
		if stmt.Init != nil {
			analyzeStmt(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, routeTables, consts, stmt.Init, visiting)
		}
		analyzeBlock(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, routeTables, consts, stmt.Body, visiting)
		analyzeElse(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, routeTables, consts, stmt.Else, visiting)
	case *ast.ForStmt:
		if stmt.Init != nil {
			analyzeStmt(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, routeTables, consts, stmt.Init, visiting)
		}
		analyzeBlock(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, routeTables, consts, stmt.Body, visiting)
		if stmt.Post != nil {
			analyzeStmt(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, routeTables, consts, stmt.Post, visiting)
		}
	case *ast.RangeStmt:
		nodeCount := len(tree.Nodes)
		analyzeRouteTableRange(fset, typeInfo, tree, fieldGroups, groups, routeTables, stmt)
		if len(tree.Nodes) == nodeCount {
			analyzeBlock(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, routeTables, consts, stmt.Body, visiting)
		}
	}
}

func analyzeElse(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, funcs map[*types.Func]*ast.FuncDecl, fieldGroups map[string]analyzer.NodeID, fileConsts map[string]string, groups map[string]analyzer.NodeID, routeTables map[string][]routeTableEntry, consts map[string]string, stmt ast.Stmt, visiting map[*ast.FuncDecl]bool) {
	switch stmt := stmt.(type) {
	case nil:
		return
	case *ast.BlockStmt:
		analyzeBlock(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, routeTables, consts, stmt, visiting)
	case *ast.IfStmt:
		analyzeStmt(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, routeTables, consts, stmt, visiting)
	}
}

func analyzeDeclFuncCalls(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, funcs map[*types.Func]*ast.FuncDecl, fieldGroups map[string]analyzer.NodeID, fileConsts map[string]string, groups map[string]analyzer.NodeID, stmt *ast.DeclStmt, visiting map[*ast.FuncDecl]bool) {
	genDecl, ok := stmt.Decl.(*ast.GenDecl)
	if !ok || genDecl.Tok != token.VAR {
		return
	}
	for _, spec := range genDecl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for _, value := range valueSpec.Values {
			analyzeFuncCall(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, value, visiting)
		}
	}
}

func analyzeAssignFuncCalls(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, funcs map[*types.Func]*ast.FuncDecl, fieldGroups map[string]analyzer.NodeID, fileConsts map[string]string, groups map[string]analyzer.NodeID, stmt *ast.AssignStmt, visiting map[*ast.FuncDecl]bool) {
	for _, rhs := range stmt.Rhs {
		analyzeFuncCall(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, rhs, visiting)
	}
}

func analyzeFuncCall(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, funcs map[*types.Func]*ast.FuncDecl, fieldGroups map[string]analyzer.NodeID, fileConsts map[string]string, groups map[string]analyzer.NodeID, expr ast.Expr, visiting map[*ast.FuncDecl]bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return
	}

	callee := funcs[calleeFunc(typeInfo, call)]
	if callee == nil || callee.Type.Params == nil {
		return
	}

	initialGroups := map[string]analyzer.NodeID{}
	argIndex := 0
	for _, field := range callee.Type.Params.List {
		for _, name := range field.Names {
			if argIndex >= len(call.Args) {
				return
			}
			nodeID, ok := argumentNodeID(typeInfo, fieldGroups, groups, call.Args[argIndex])
			if !ok {
				nodeID, ok = groupCallNodeID(fset, typeInfo, tree, fieldGroups, groups, fileConsts, call.Args[argIndex])
			}
			if ok && isEchoParam(typeInfo, name) {
				initialGroups[name.Name] = nodeID
			}
			argIndex++
		}
	}
	if len(initialGroups) == 0 {
		return
	}

	analyzeFunc(fset, typeInfo, tree, funcs, fieldGroups, callee, fileConsts, initialGroups, visiting)
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
