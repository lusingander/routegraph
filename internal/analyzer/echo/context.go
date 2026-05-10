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
	routeTables map[string][]routeTableEntry
	consts      map[string]string
	env         env
	visiting    map[*ast.FuncDecl]bool
	analyzed    map[string]bool
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
		routeTables: map[string][]routeTableEntry{},
		consts:      cloneConsts(fileConsts),
		env:         newEnv(fileConsts),
		visiting:    map[*ast.FuncDecl]bool{},
		analyzed:    map[string]bool{},
	}
}

func (ctx *analysisContext) withCallBindings(groups map[string]analyzer.NodeID, values map[string]value) *analysisContext {
	next := *ctx
	next.groups = cloneGroups(groups)
	next.routeTables = cloneRouteTables(ctx.routeTables)
	next.consts = cloneConsts(ctx.fileConsts)
	next.env = ctx.env.withConsts(ctx.fileConsts)
	for name, id := range groups {
		next.env.setGroup(name, id)
	}
	for name, value := range values {
		next.env.values[name] = cloneValue(value)
	}
	return &next
}
