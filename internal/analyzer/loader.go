package analyzer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

type GoFile struct {
	Path string
	File *ast.File
}

type GoPackage struct {
	Fset *token.FileSet
	Pkg  *packages.Package
}

func LoadGoFiles(dir string) (*token.FileSet, []GoFile, error) {
	if dir == "" {
		dir = "."
	}
	dir = normalizeDirPattern(dir)

	fset := token.NewFileSet()
	var files []GoFile
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return err
		}
		files = append(files, GoFile{Path: path, File: file})
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return fset, files, nil
}

func normalizeDirPattern(dir string) string {
	if dir == "..." {
		return "."
	}
	if strings.HasSuffix(dir, "/...") {
		base := strings.TrimSuffix(dir, "/...")
		if base == "" {
			return "."
		}
		return base
	}
	return dir
}

func LoadGoPackages(dir string) ([]GoPackage, error) {
	if dir == "" {
		dir = "."
	}
	recursive := strings.HasSuffix(dir, "/...") || dir == "..."
	base := normalizeDirPattern(dir)
	pattern := "."
	if recursive {
		pattern = "./..."
	}

	fset := token.NewFileSet()
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedImports,
		Dir:  base,
		Fset: fset,
	}
	pkgs, err := packages.Load(cfg, pattern)
	if err != nil {
		return nil, err
	}
	result := make([]GoPackage, 0, len(pkgs))
	for _, pkg := range pkgs {
		result = append(result, GoPackage{Fset: fset, Pkg: pkg})
	}
	return result, nil
}
