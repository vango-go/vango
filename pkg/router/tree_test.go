package router

import (
	"testing"

	"github.com/vango-dev/vango/v2/pkg/server"
	"github.com/vango-dev/vango/v2/pkg/vdom"
)

// dummyPageHandler is a test page handler.
func dummyPageHandler(ctx server.Ctx, params any) vdom.Component { return nil }

// dummyLayoutHandler is a test layout handler.
func dummyLayoutHandler(ctx server.Ctx, children Slot) *vdom.VNode { return nil }

func TestRouteNodeFindChild(t *testing.T) {
	root := newRouteNode("")
	root.addChild("users")
	root.addChild("projects")

	tests := []struct {
		segment string
		want    bool
	}{
		{"users", true},
		{"projects", true},
		{"tasks", false},
		{"", false},
	}

	for _, tt := range tests {
		child := root.findChild(tt.segment)
		got := child != nil
		if got != tt.want {
			t.Errorf("findChild(%q) = %v, want %v", tt.segment, got, tt.want)
		}
	}
}

func TestRouteNodeAddChild(t *testing.T) {
	root := newRouteNode("")

	// Add new child
	child1 := root.addChild("users")
	if child1 == nil {
		t.Fatal("addChild returned nil")
	}
	if child1.segment != "users" {
		t.Errorf("segment = %q, want %q", child1.segment, "users")
	}

	// Adding same child returns existing
	child2 := root.addChild("users")
	if child1 != child2 {
		t.Error("addChild should return existing child")
	}

	// Verify children count
	if len(root.children) != 1 {
		t.Errorf("len(children) = %d, want 1", len(root.children))
	}
}

func TestRouteNodeInsertRoute(t *testing.T) {
	root := newRouteNode("")

	// Insert static route
	node := root.insertRoute("/users/list")
	if node == nil {
		t.Fatal("insertRoute returned nil")
	}

	// Insert param route
	node = root.insertRoute("/users/:id")
	if node == nil || !node.isParam {
		t.Error("expected param node")
	}
	if node.paramName != "id" {
		t.Errorf("paramName = %q, want %q", node.paramName, "id")
	}

	// Insert catch-all route
	node = root.insertRoute("/files/*path")
	if node == nil || !node.isCatchAll {
		t.Error("expected catch-all node")
	}
	if node.paramName != "path" {
		t.Errorf("paramName = %q, want %q", node.paramName, "path")
	}
}

func TestRouteNodeMatchStatic(t *testing.T) {
	root := newRouteNode("")
	node := root.insertRoute("/users/list")
	node.pageHandler = dummyPageHandler

	tests := []struct {
		path      string
		wantMatch bool
	}{
		{"/users/list", true},
		{"/users", false},
		{"/users/list/extra", false},
		{"/projects", false},
		{"", false},
	}

	for _, tt := range tests {
		params := make(map[string]string)
		_, _, ok := root.match(splitPath(tt.path), params, nil)
		if ok != tt.wantMatch {
			t.Errorf("match(%q) = %v, want %v", tt.path, ok, tt.wantMatch)
		}
	}
}

func TestRouteNodeMatchParams(t *testing.T) {
	root := newRouteNode("")
	node := root.insertRoute("/users/:id")
	node.pageHandler = dummyPageHandler

	params := make(map[string]string)
	matchedNode, _, ok := root.match(splitPath("/users/123"), params, nil)

	if !ok {
		t.Fatal("expected match")
	}
	if matchedNode == nil {
		t.Fatal("expected non-nil node")
	}
	if params["id"] != "123" {
		t.Errorf("params[id] = %q, want %q", params["id"], "123")
	}
}

func TestRouteNodeMatchCatchAll(t *testing.T) {
	root := newRouteNode("")
	node := root.insertRoute("/files/*path")
	node.pageHandler = dummyPageHandler

	params := make(map[string]string)
	matchedNode, _, ok := root.match(splitPath("/files/a/b/c"), params, nil)

	if !ok {
		t.Fatal("expected match")
	}
	if matchedNode == nil {
		t.Fatal("expected non-nil node")
	}
	if params["path"] != "a/b/c" {
		t.Errorf("params[path] = %q, want %q", params["path"], "a/b/c")
	}
}

func TestRouteNodeMatchLayoutCollection(t *testing.T) {
	root := newRouteNode("")

	// Add root layout
	root.layoutHandler = dummyLayoutHandler

	// Add /users with layout
	usersNode := root.insertRoute("/users")
	usersNode.layoutHandler = dummyLayoutHandler

	// Add /users/list
	listNode := root.insertRoute("/users/list")
	listNode.pageHandler = dummyPageHandler

	params := make(map[string]string)
	var layouts []LayoutHandler

	// Include root layout
	if root.layoutHandler != nil {
		layouts = append(layouts, root.layoutHandler)
	}

	_, matchedLayouts, ok := root.match(splitPath("/users/list"), params, layouts)

	if !ok {
		t.Fatal("expected match")
	}
	// Layouts: root (passed in) + users (collected during match)
	if len(matchedLayouts) < 2 {
		t.Errorf("len(layouts) = %d, want at least 2", len(matchedLayouts))
	}
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		path string
		want []string
	}{
		{"", nil},
		{"/", nil},
		{"/users", []string{"users"}},
		{"/users/list", []string{"users", "list"}},
		{"users/list", []string{"users", "list"}},
		{"/a/b/c/", []string{"a", "b", "c"}},
	}

	for _, tt := range tests {
		got := splitPath(tt.path)
		if len(got) != len(tt.want) {
			t.Errorf("splitPath(%q) = %v, want %v", tt.path, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("splitPath(%q)[%d] = %q, want %q", tt.path, i, got[i], tt.want[i])
			}
		}
	}
}

func TestParseParamSegment(t *testing.T) {
	tests := []struct {
		seg      string
		wantName string
		wantType string
	}{
		{":id", "id", "string"},
		{":id:int", "id", "int"},
		{":userId:uuid", "userId", "uuid"},
		{":name", "name", "string"},
	}

	for _, tt := range tests {
		name, paramType := parseParamSegment(tt.seg)
		if name != tt.wantName {
			t.Errorf("parseParamSegment(%q) name = %q, want %q", tt.seg, name, tt.wantName)
		}
		if paramType != tt.wantType {
			t.Errorf("parseParamSegment(%q) type = %q, want %q", tt.seg, paramType, tt.wantType)
		}
	}
}
