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
	for _, pkg := range pkgs {
		if len(pkg.Pkg.Errors) > 0 {
			return fmt.Errorf("%s", pkg.Pkg.Errors[0])
		}
		pkgConsts := collectPackageConsts(pkg.Pkg.Syntax)
		funcs := collectPackageFuncs(pkg.Pkg.TypesInfo, pkg.Pkg.Syntax)
		fieldGroups := map[string]analyzer.NodeID{}
		for _, file := range pkg.Pkg.Syntax {
			if err := ctx.Err(); err != nil {
				return err
			}
			analyzeFile(pkg.Fset, pkg.Pkg.TypesInfo, tree, funcs, fieldGroups, file, pkgConsts)
		}
	}
	return nil
}

func analyzeFile(fset *token.FileSet, typeInfo *types.Info, tree *analyzer.RouteTree, funcs map[*types.Func]*ast.FuncDecl, fieldGroups map[string]analyzer.NodeID, file *ast.File, pkgConsts map[string]string) {
	fileConsts := cloneConsts(pkgConsts)
	collectFileConsts(file, fileConsts)
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil || fn.Recv != nil {
			continue
		}
		analyzeFunc(fset, typeInfo, tree, funcs, fieldGroups, fn, fileConsts, nil, nil, map[*ast.FuncDecl]bool{})
	}
}
