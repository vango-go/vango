package router

import (
	"strings"
	"testing"
)

// TestGeneratorDeterminism tests that the generator produces deterministic output.
func TestGeneratorDeterminism(t *testing.T) {
	routes := []ScannedRoute{
		{Path: "/users", FilePath: "users/index.go", Package: "users", HasPage: true},
		{Path: "/about", FilePath: "about.go", Package: "routes", HasPage: true},
		{Path: "/", FilePath: "index.go", Package: "routes", HasPage: true},
	}

	gen := NewGenerator(routes, "example.com/test")

	// Generate multiple times and compare
	output1, err := gen.Generate()
	if err != nil {
		t.Fatal(err)
	}

	output2, err := gen.Generate()
	if err != nil {
		t.Fatal(err)
	}

	if string(output1) != string(output2) {
		t.Error("Generator output is not deterministic")
	}
}

// TestGeneratorSortedRoutes tests that routes are sorted in the output.
func TestGeneratorSortedRoutes(t *testing.T) {
	routes := []ScannedRoute{
		{Path: "/z-route", FilePath: "z.go", Package: "routes", HasPage: true},
		{Path: "/a-route", FilePath: "a.go", Package: "routes", HasPage: true},
		{Path: "/m-route", FilePath: "m.go", Package: "routes", HasPage: true},
	}

	gen := NewGenerator(routes, "example.com/test")
	output, err := gen.Generate()
	if err != nil {
		t.Fatal(err)
	}

	content := string(output)

	// Check routes appear in sorted order
	aIdx := strings.Index(content, "/a-route")
	mIdx := strings.Index(content, "/m-route")
	zIdx := strings.Index(content, "/z-route")

	if aIdx > mIdx || mIdx > zIdx {
		t.Error("Routes should be sorted alphabetically")
	}
}

// TestGeneratorHeader tests the generated header.
func TestGeneratorHeader(t *testing.T) {
	routes := []ScannedRoute{}
	gen := NewGenerator(routes, "example.com/test")

	output, err := gen.Generate()
	if err != nil {
		t.Fatal(err)
	}

	content := string(output)

	t.Run("contains DO NOT EDIT comment", func(t *testing.T) {
		if !strings.Contains(content, "DO NOT EDIT") {
			t.Error("should contain 'DO NOT EDIT' comment")
		}
	})

	t.Run("package is routes", func(t *testing.T) {
		if !strings.Contains(content, "package routes") {
			t.Error("should have 'package routes'")
		}
	})
}

// TestGeneratorRegisterFunction tests the Register function generation.
func TestGeneratorRegisterFunction(t *testing.T) {
	routes := []ScannedRoute{
		{Path: "/", FilePath: "index.go", Package: "routes", HasPage: true},
		{Path: "/about", FilePath: "about.go", Package: "routes", HasPage: true},
		{Path: "/api/health", FilePath: "api/health.go", Package: "api", Methods: []string{"GET"}, IsAPI: true},
	}

	gen := NewGenerator(routes, "example.com/test")
	output, err := gen.Generate()
	if err != nil {
		t.Fatal(err)
	}

	content := string(output)

	t.Run("has Register function", func(t *testing.T) {
		if !strings.Contains(content, "func Register(r *Router)") {
			t.Error("should have Register function")
		}
	})

	t.Run("has Page routes comment", func(t *testing.T) {
		if !strings.Contains(content, "// Page routes") {
			t.Error("should have '// Page routes' comment")
		}
	})

	t.Run("has API routes comment", func(t *testing.T) {
		if !strings.Contains(content, "// API routes") {
			t.Error("should have '// API routes' comment")
		}
	})

	t.Run("registers page routes with r.Page", func(t *testing.T) {
		if !strings.Contains(content, `r.Page("/", IndexPage`) {
			t.Error("should register root page")
		}
		if !strings.Contains(content, `r.Page("/about", AboutPage`) {
			t.Error("should register about page")
		}
	})

	t.Run("registers API routes with r.API", func(t *testing.T) {
		if !strings.Contains(content, `r.API("GET", "/api/health"`) {
			t.Error("should register health API")
		}
	})
}

// TestGeneratorRouteConstants tests route constant generation.
func TestGeneratorRouteConstants(t *testing.T) {
	routes := []ScannedRoute{
		{Path: "/", FilePath: "index.go", Package: "routes", HasPage: true},
		{Path: "/about", FilePath: "about.go", Package: "routes", HasPage: true},
		{Path: "/users/:id", FilePath: "users/[id].go", Package: "users", HasPage: true, Params: []ParamDef{{Name: "id", Type: "int"}}},
	}

	gen := NewGenerator(routes, "example.com/test")
	output, err := gen.Generate()
	if err != nil {
		t.Fatal(err)
	}

	content := string(output)

	t.Run("generates route constants", func(t *testing.T) {
		if !strings.Contains(content, "RouteIndex") {
			t.Error("should have RouteIndex constant")
		}
		if !strings.Contains(content, "RouteAbout") {
			t.Error("should have RouteAbout constant")
		}
	})
}

// TestGeneratorParamStructs tests param struct generation.
func TestGeneratorParamStructs(t *testing.T) {
	routes := []ScannedRoute{
		{
			Path:     "/users/:id",
			FilePath: "users/[id].go",
			Package:  "users",
			HasPage:  true,
			Params:   []ParamDef{{Name: "id", Type: "int", Segment: "[id]"}},
		},
		{
			Path:     "/posts/:slug",
			FilePath: "posts/[slug].go",
			Package:  "posts",
			HasPage:  true,
			Params:   []ParamDef{{Name: "slug", Type: "string", Segment: "[slug]"}},
		},
	}

	gen := NewGenerator(routes, "example.com/test")
	output, err := gen.Generate()
	if err != nil {
		t.Fatal(err)
	}

	content := string(output)

	t.Run("generates param struct for users route", func(t *testing.T) {
		// Path is /users/:id → struct name is UsersIDParams (id → ID)
		if !strings.Contains(content, "UsersIDParams") {
			t.Error("should have UsersIDParams struct")
		}
		if !strings.Contains(content, "ID int") {
			t.Error("should have ID int field")
		}
	})
}

// TestPathToStructName tests path to struct name conversion.
func TestPathToStructName(t *testing.T) {
	gen := &Generator{}

	// Note: Only exact matches like "id" get uppercased to "ID"
	// Compound names like "userId" get title-cased to "UserId"
	tests := []struct {
		path     string
		expected string
	}{
		{"/", "Index"},
		{"/about", "About"},
		{"/users/:id", "UsersID"},                          // "id" → "ID" (exact match)
		{"/users/:userId/posts/:postId", "UsersUserIdPostsPostId"}, // "userId" → "UserId" (title case)
		{"/docs/*path", "DocsPath"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := gen.pathToStructName(tt.path)
			if got != tt.expected {
				t.Errorf("pathToStructName(%q) = %q, want %q", tt.path, got, tt.expected)
			}
		})
	}
}

// TestToExportedName tests identifier capitalization.
func TestToExportedName(t *testing.T) {
	gen := &Generator{}

	tests := []struct {
		input    string
		expected string
	}{
		{"id", "ID"},
		{"url", "URL"},
		{"api", "API"},
		{"http", "HTTP"},
		{"uuid", "UUID"},
		{"user", "User"},
		{"userName", "UserName"},
		{"post_id", "Post_id"}, // Underscore preserved
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := gen.toExportedName(tt.input)
			if got != tt.expected {
				t.Errorf("toExportedName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestParamTypeToGoType tests param type conversion.
func TestParamTypeToGoType(t *testing.T) {
	gen := &Generator{}

	tests := []struct {
		input    string
		expected string
	}{
		{"int", "int"},
		{"int64", "int64"},
		{"int32", "int32"},
		{"uint", "uint"},
		{"uint64", "uint64"},
		{"uuid", "string"},
		{"[]string", "[]string"},
		{"string", "string"},
		{"", "string"},
		{"unknown", "string"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := gen.paramTypeToGoType(tt.input)
			if got != tt.expected {
				t.Errorf("paramTypeToGoType(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
