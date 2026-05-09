package echo

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/lusingander/routegraph/internal/analyzer"
)

func collectPackageFuncs(files []*ast.File) map[string]*ast.FuncDecl {
	funcs := map[string]*ast.FuncDecl{}
	for _, file := range files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || fn.Recv != nil {
				continue
			}
			funcs[fn.Name.Name] = fn
		}
	}
	return funcs
}

func analyzeFunc(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, funcs map[string]*ast.FuncDecl, fieldGroups map[string]analyzer.NodeID, fn *ast.FuncDecl, fileConsts map[string]string, initialGroups map[string]analyzer.NodeID, visiting map[string]bool) {
	if visiting[fn.Name.Name] {
		return
	}
	visiting[fn.Name.Name] = true
	defer delete(visiting, fn.Name.Name)

	groups := cloneGroups(initialGroups)
	routeTables := map[string][]routeTableEntry{}
	consts := cloneConsts(fileConsts)
	collectBlockConsts(fn.Body, consts)

	for _, stmt := range fn.Body.List {
		analyzeStructFields(fset, typeInfo, tree, fieldGroups, groups, consts, stmt)
		switch stmt := stmt.(type) {
		case *ast.DeclStmt:
			analyzeDecl(fset, typeInfo, tree, fieldGroups, groups, consts, stmt)
		case *ast.AssignStmt:
			analyzeAssign(fset, typeInfo, tree, fieldGroups, groups, consts, stmt)
			collectRouteTable(routeTables, consts, stmt)
		case *ast.ExprStmt:
			analyzeExpr(fset, typeInfo, tree, fieldGroups, groups, consts, stmt.X)
			analyzeFuncCall(fset, typeInfo, tree, funcs, fieldGroups, fileConsts, groups, stmt.X, visiting)
		case *ast.RangeStmt:
			analyzeRouteTableRange(fset, typeInfo, tree, fieldGroups, groups, routeTables, stmt)
		}
	}
}

func analyzeFuncCall(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, funcs map[string]*ast.FuncDecl, fieldGroups map[string]analyzer.NodeID, fileConsts map[string]string, groups map[string]analyzer.NodeID, expr ast.Expr, visiting map[string]bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return
	}
	calleeIdent, ok := call.Fun.(*ast.Ident)
	if !ok {
		return
	}
	callee := funcs[calleeIdent.Name]
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
