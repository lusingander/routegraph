package analyzer

import "testing"

func TestRouteTreeInspection(t *testing.T) {
	tree := NewRouteTree()
	apiID := tree.AddGroup(0, FrameworkEcho, KnownPath("/api"), Position{})
	usersID := tree.AddRoute(apiID, FrameworkEcho, "GET", KnownPath("/users"), "listUsers", Position{})

	api, ok := tree.Node(apiID)
	if !ok {
		t.Fatalf("Node(%d) not found", apiID)
	}
	if api.FullPath.Value != "/api" {
		t.Fatalf("api FullPath = %q, want /api", api.FullPath.Value)
	}

	users, ok := tree.Node(usersID)
	if !ok {
		t.Fatalf("Node(%d) not found", usersID)
	}
	if users.ParentID == nil || *users.ParentID != apiID {
		t.Fatalf("users ParentID = %v, want %d", users.ParentID, apiID)
	}

	rootChildren := tree.Children(0)
	if len(rootChildren) != 1 {
		t.Fatalf("len(rootChildren) = %d, want 1", len(rootChildren))
	}
	if rootChildren[0].ID != apiID {
		t.Fatalf("root child ID = %d, want %d", rootChildren[0].ID, apiID)
	}

	apiChildren := tree.Children(apiID)
	if len(apiChildren) != 1 {
		t.Fatalf("len(apiChildren) = %d, want 1", len(apiChildren))
	}
	if apiChildren[0].ID != usersID {
		t.Fatalf("api child ID = %d, want %d", apiChildren[0].ID, usersID)
	}
}
