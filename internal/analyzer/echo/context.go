package echo

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/lusingander/routegraph/internal/analyzer"
)

type analysisContext struct {
	fset        *token.FileSet
	typeInfo    *types.Info
	tree        *analyzer.RouteTree
	funcs       map[*types.Func]funcInfo
	funcNames   map[string]funcInfo
	fieldGroups map[string]analyzer.NodeID
	fileConsts  map[string]string

	groups      map[string]analyzer.NodeID
	fields      localFieldGroups
	routeTables map[string][]routeTableEntry
	consts      map[string]string
	visiting    map[*ast.FuncDecl]bool
}

type funcInfo struct {
	decl       *ast.FuncDecl
	typeInfo   *types.Info
	fileConsts map[string]string
}

func newAnalysisContext(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, funcs map[*types.Func]funcInfo, funcNames map[string]funcInfo, fieldGroups map[string]analyzer.NodeID, fileConsts map[string]string) *analysisContext {
	return &analysisContext{
		fset:        fset,
		typeInfo:    typeInfo,
		tree:        tree,
		funcs:       funcs,
		funcNames:   funcNames,
		fieldGroups: fieldGroups,
		fileConsts:  fileConsts,
		groups:      map[string]analyzer.NodeID{},
		fields:      localFieldGroups{},
		routeTables: map[string][]routeTableEntry{},
		consts:      cloneConsts(fileConsts),
		visiting:    map[*ast.FuncDecl]bool{},
	}
}

func (ctx *analysisContext) withCallBindings(groups map[string]analyzer.NodeID, fields localFieldGroups) *analysisContext {
	next := *ctx
	next.groups = cloneGroups(groups)
	next.fields = cloneLocalFieldGroups(fields)
	next.routeTables = map[string][]routeTableEntry{}
	next.consts = cloneConsts(ctx.fileConsts)
	return &next
}
