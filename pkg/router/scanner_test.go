package router

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScannerFilePathToURLPath(t *testing.T) {
	s := NewScanner("/app/routes")

	tests := []struct {
		relPath string
		want    string
	}{
		{"index.go", "/"},
		{"about.go", "/about"},
		{"projects/index.go", "/projects"},
		{"projects/new.go", "/projects/new"},
		{"projects/[id].go", "/projects/:id"},
		{"projects/[id]/edit.go", "/projects/:id/edit"},
		{"projects/[id]/tasks/[taskId].go", "/projects/:id/tasks/:taskId"},
		{"api/users.go", "/api/users"},
		{"[...slug].go", "/*slug"},
		{"docs/[...path].go", "/docs/*path"},
		{"layout.go", "/"},
		{"projects/layout.go", "/projects"},
		{"[id:int].go", "/:id"},
		{"users/id_.go", "/users/:id"},
		{"docs/slug___.go", "/docs/*slug"},
	}

	for _, tt := range tests {
		got := s.filePathToURLPath(tt.relPath)
		if got != tt.want {
			t.Errorf("filePathToURLPath(%q) = %q, want %q", tt.relPath, got, tt.want)
		}
	}
}

func TestScannerConvertParams(t *testing.T) {
	s := NewScanner("/app/routes")

	tests := []struct {
		path string
		want string
	}{
		{"[id]", ":id"},
		{"[id:int]", ":id"},
		{"projects/[id]", "projects/:id"},
		{"[...slug]", "*slug"},
		{"docs/[...path]", "docs/*path"},
		{"users/[userId]/posts/[postId]", "users/:userId/posts/:postId"},
		{"users/id_", "users/:id"},
		{"docs/slug___", "docs/*slug"},
		{"users/_id_/posts/index", "users/:id/posts/index"},
	}

	for _, tt := range tests {
		got := s.convertParams(tt.path)
		if got != tt.want {
			t.Errorf("convertParams(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestScannerExtractParams(t *testing.T) {
	s := NewScanner("/app/routes")

	// Phase 14 Spec Section 6.2 - Param Type Inference:
	// - [id].go → int (convention for numeric IDs)
	// - [userId].go → int (ends with "id")
	// - [slug].go → string (default for named params)
	// - [...path].go → []string (catch-all route)
	tests := []struct {
		relPath    string
		wantParams []ParamDef
	}{
		{
			"index.go",
			nil,
		},
		{
			"[id].go",
			[]ParamDef{{Name: "id", Type: "int", Segment: "[id]"}}, // Phase 14: "id" infers int
		},
		{
			"[id:int].go",
			[]ParamDef{{Name: "id", Type: "int", Segment: "[id:int]"}},
		},
		{
			"[...slug].go",
			[]ParamDef{{Name: "slug", Type: "[]string", Segment: "[...slug]"}},
		},
		{
			"users/[userId]/posts/[postId:int].go",
			[]ParamDef{
				{Name: "userId", Type: "int", Segment: "[userId]"}, // Phase 14: ends with "Id" infers int
				{Name: "postId", Type: "int", Segment: "[postId:int]"},
			},
		},
		{
			"users/id_.go",
			[]ParamDef{{Name: "id", Type: "int", Segment: "id_"}},
		},
		{
			"docs/slug___.go",
			[]ParamDef{{Name: "slug", Type: "[]string", Segment: "slug___"}},
		},
	}

	for _, tt := range tests {
		got := s.extractParams(tt.relPath)
		if len(got) != len(tt.wantParams) {
			t.Errorf("extractParams(%q) len = %d, want %d", tt.relPath, len(got), len(tt.wantParams))
			continue
		}
		for i, p := range got {
			want := tt.wantParams[i]
			if p.Name != want.Name || p.Type != want.Type || p.Segment != want.Segment {
				t.Errorf("extractParams(%q)[%d] = %+v, want %+v", tt.relPath, i, p, want)
			}
		}
	}
}

func TestScannerIsAPIRoute(t *testing.T) {
	s := NewScanner("/app/routes")

	tests := []struct {
		relPath string
		want    bool
	}{
		{"api/users.go", true},
		{"api/projects/[id].go", true},
		{"projects/index.go", false},
		{"index.go", false},
	}

	for _, tt := range tests {
		got := s.isAPIRoute(tt.relPath)
		if got != tt.want {
			t.Errorf("isAPIRoute(%q) = %v, want %v", tt.relPath, got, tt.want)
		}
	}
}

func TestScannerScan(t *testing.T) {
	// Create temp directory with route files
	dir := t.TempDir()

	// Create test files
	files := map[string]string{
		"index.go":          `package routes; func Page() {}`,
		"about.go":          `package routes; func Page() {}; func Meta() {}`,
		"layout.go":         `package routes; func Layout() {}`,
		"projects/index.go": `package routes; func Page() {}`,
		"projects/[id].go":  `package routes; func Page() {}; func Middleware() {}`,
		"api/users.go":      `package api; func GET() {}; func POST() {}`,
	}

	for path, content := range files {
		fullPath := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(fullPath), err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", fullPath, err)
		}
	}

	scanner := NewScanner(dir)
	routes, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	// Should find all 6 route files
	if len(routes) != 6 {
		t.Errorf("len(routes) = %d, want 6", len(routes))
	}

	// Check specific routes
	routeMap := make(map[string]*ScannedRoute)
	for i := range routes {
		routeMap[routes[i].Path] = &routes[i]
	}

	// Check index
	if r, ok := routeMap["/"]; !ok {
		t.Error("missing / route")
	} else if !r.HasPage && !r.HasLayout {
		t.Error("/ should have Page or Layout")
	}

	// Check about (should have Page and Meta)
	if r, ok := routeMap["/about"]; !ok {
		t.Error("missing /about route")
	} else {
		if !r.HasPage {
			t.Error("/about should have Page")
		}
		if !r.HasMeta {
			t.Error("/about should have Meta")
		}
	}

	// Check projects/[id] (should have param)
	if r, ok := routeMap["/projects/:id"]; !ok {
		t.Error("missing /projects/:id route")
	} else {
		if len(r.Params) != 1 {
			t.Errorf("/projects/:id params = %d, want 1", len(r.Params))
		}
		if !r.HasMiddleware {
			t.Error("/projects/:id should have Middleware")
		}
	}

	// Check API users (should have GET and POST)
	if r, ok := routeMap["/api/users"]; !ok {
		t.Error("missing /api/users route")
	} else {
		if !r.IsAPI {
			t.Error("/api/users should be API")
		}
		if len(r.Methods) != 2 {
			t.Errorf("/api/users methods = %d, want 2", len(r.Methods))
		}
	}
}

func TestScannerSkipsTestFiles(t *testing.T) {
	dir := t.TempDir()

	files := map[string]string{
		"index.go":      `package routes; func Page() {}`,
		"index_test.go": `package routes; func TestPage() {}`,
	}

	for path, content := range files {
		fullPath := filepath.Join(dir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", fullPath, err)
		}
	}

	scanner := NewScanner(dir)
	routes, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	if len(routes) != 1 {
		t.Errorf("len(routes) = %d, want 1 (should skip test files)", len(routes))
	}
}
