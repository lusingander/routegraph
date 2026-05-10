package echo

import (
	"go/token"

	"github.com/lusingander/routegraph/internal/analyzer"
)

func cloneConsts(consts map[string]string) map[string]string {
	cloned := make(map[string]string, len(consts))
	for name, value := range consts {
		cloned[name] = value
	}
	return cloned
}

func cloneGroups(groups map[string]analyzer.NodeID) map[string]analyzer.NodeID {
	cloned := make(map[string]analyzer.NodeID, len(groups))
	for name, id := range groups {
		cloned[name] = id
	}
	return cloned
}

func cloneRouteTables(routeTables map[string][]routeTableEntry) map[string][]routeTableEntry {
	cloned := make(map[string][]routeTableEntry, len(routeTables))
	for name, entries := range routeTables {
		cloned[name] = append([]routeTableEntry(nil), entries...)
	}
	return cloned
}

func position(fset *token.FileSet, pos token.Pos) analyzer.Position {
	p := fset.Position(pos)
	return analyzer.Position{File: p.Filename, Line: p.Line}
}
