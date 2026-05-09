package analyzer

type Framework string

const (
	FrameworkEcho Framework = "echo"
)

type NodeKind string

const (
	NodeKindRoot  NodeKind = "root"
	NodeKindGroup NodeKind = "group"
	NodeKindRoute NodeKind = "route"
)

type NodeID int

type Position struct {
	File string
	Line int
}

type PathExpr struct {
	Value  string
	Known  bool
	Reason string
}

type RouteNode struct {
	ID       NodeID
	ParentID *NodeID

	Framework Framework
	Kind      NodeKind

	Method string

	PathPart PathExpr
	FullPath PathExpr

	Handler string
	Pos     Position

	Warnings []string
}

type RouteTree struct {
	Nodes []RouteNode
}

type Route struct {
	Framework string `json:"framework"`

	Method  string `json:"method"`
	Path    string `json:"path"`
	Handler string `json:"handler"`

	File string `json:"file"`
	Line int    `json:"line"`

	Known    bool     `json:"known"`
	Warnings []string `json:"warnings"`
}

func NewRouteTree() *RouteTree {
	tree := &RouteTree{}
	tree.AddRoot(FrameworkEcho)
	return tree
}

func (t *RouteTree) AddRoot(framework Framework) NodeID {
	id := NodeID(len(t.Nodes))
	t.Nodes = append(t.Nodes, RouteNode{
		ID:        id,
		Framework: framework,
		Kind:      NodeKindRoot,
		PathPart:  KnownPath("/"),
		FullPath:  KnownPath("/"),
	})
	return id
}

func (t *RouteTree) AddGroup(parentID NodeID, framework Framework, path PathExpr, pos Position) NodeID {
	id := NodeID(len(t.Nodes))
	parent := parentID
	parentPath := t.Nodes[parentID].FullPath
	fullPath := JoinPath(parentPath, path)
	t.Nodes = append(t.Nodes, RouteNode{
		ID:        id,
		ParentID:  &parent,
		Framework: framework,
		Kind:      NodeKindGroup,
		PathPart:  path,
		FullPath:  fullPath,
		Pos:       pos,
		Warnings:  pathWarnings(fullPath),
	})
	return id
}

func (t *RouteTree) AddRoute(parentID NodeID, framework Framework, method string, path PathExpr, handler string, pos Position) NodeID {
	id := NodeID(len(t.Nodes))
	parent := parentID
	parentPath := t.Nodes[parentID].FullPath
	fullPath := JoinPath(parentPath, path)
	t.Nodes = append(t.Nodes, RouteNode{
		ID:        id,
		ParentID:  &parent,
		Framework: framework,
		Kind:      NodeKindRoute,
		Method:    method,
		PathPart:  path,
		FullPath:  fullPath,
		Handler:   handler,
		Pos:       pos,
		Warnings:  pathWarnings(fullPath),
	})
	return id
}

func pathWarnings(path PathExpr) []string {
	if path.Known || path.Reason == "" {
		return nil
	}
	return []string{path.Reason}
}

func (t *RouteTree) Node(id NodeID) (RouteNode, bool) {
	if id < 0 || int(id) >= len(t.Nodes) {
		return RouteNode{}, false
	}
	return t.Nodes[id], true
}

func (t *RouteTree) Children(parentID NodeID) []RouteNode {
	children := make([]RouteNode, 0)
	for _, node := range t.Nodes {
		if node.ParentID == nil || *node.ParentID != parentID {
			continue
		}
		children = append(children, node)
	}
	return children
}

func Flatten(tree *RouteTree) []Route {
	routes := make([]Route, 0)
	for _, node := range tree.Nodes {
		if node.Kind != NodeKindRoute {
			continue
		}
		routes = append(routes, Route{
			Framework: string(node.Framework),
			Method:    node.Method,
			Path:      node.FullPath.Value,
			Handler:   node.Handler,
			File:      node.Pos.File,
			Line:      node.Pos.Line,
			Known:     node.FullPath.Known,
			Warnings:  append([]string(nil), node.Warnings...),
		})
	}
	return routes
}
