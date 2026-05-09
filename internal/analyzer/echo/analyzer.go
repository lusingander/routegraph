package echo

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/lusingander/routegraph/internal/analyzer"
)

type Analyzer struct{}

func Analyze(ctx context.Context, dir string, tree *analyzer.RouteTree) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	pkgs, err := analyzer.LoadGoPackages(dir)
	if err != nil {
		return err
	}

	return Analyzer{}.Analyze(ctx, pkgs, tree)
}

func (Analyzer) Analyze(ctx context.Context, pkgs []analyzer.GoPackage, tree *analyzer.RouteTree) error {
	funcs := map[*types.Func]funcInfo{}
	funcNames := map[string]funcInfo{}
	pkgConsts := make([]map[string]string, len(pkgs))
	for _, pkg := range pkgs {
		if len(pkg.Pkg.Errors) > 0 {
			return fmt.Errorf("%s", pkg.Pkg.Errors[0])
		}
	}
	for i, pkg := range pkgs {
		pkgConsts[i] = collectPackageConsts(pkg.Pkg.Syntax)
		for fn, info := range collectPackageFuncs(pkg.Pkg.TypesInfo, pkg.Pkg.Syntax, pkgConsts[i]) {
			funcs[fn] = info
			funcNames[funcKey(fn)] = info
		}
	}
	fieldGroups := map[string]analyzer.NodeID{}
	analyzed := map[string]bool{}
	for i, pkg := range pkgs {
		for _, file := range pkg.Pkg.Syntax {
			if err := ctx.Err(); err != nil {
				return err
			}
			analyzeFile(pkg.Fset, pkg.Pkg.TypesInfo, tree, funcs, funcNames, fieldGroups, analyzed, file, pkgConsts[i])
		}
	}
	return nil
}

func analyzeFile(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, funcs map[*types.Func]funcInfo, funcNames map[string]funcInfo, fieldGroups map[string]analyzer.NodeID, analyzed map[string]bool, file *ast.File, pkgConsts map[string]string) {
	fileConsts := cloneConsts(pkgConsts)
	collectFileConsts(file, fileConsts)
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil || fn.Recv != nil {
			continue
		}
		fnObj, ok := typeInfo.Defs[fn.Name].(*types.Func)
		if !ok {
			continue
		}
		info := funcs[fnObj]
		if info.decl == nil {
			continue
		}
		ctx := newAnalysisContext(fset, typeInfo, tree, funcs, funcNames, fieldGroups, fileConsts)
		ctx.analyzed = analyzed
		analyzeFunc(ctx, info, nil, nil)
	}
}
