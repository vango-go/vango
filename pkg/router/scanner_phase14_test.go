package router

import (
	"testing"
)

// TestInferParamTypeFromName tests the parameter type inference.
func TestInferParamTypeFromName(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		// ID patterns → int
		{"id", "int"},
		{"userId", "int"},
		{"user_id", "int"},
		{"postId", "int"},
		{"post_id", "int"},
		{"projectId", "int"},

		// UUID patterns → string
		{"uuid", "string"},
		{"userUuid", "string"},

		// String patterns
		{"slug", "string"},
		{"name", "string"},
		{"title", "string"},
		{"path", "string"},
		{"key", "string"},
		{"token", "string"},
		{"code", "string"},

		// Numeric patterns → int
		{"page", "int"},
		{"limit", "int"},
		{"offset", "int"},
		{"count", "int"},
		{"index", "int"},
		{"num", "int"},
		{"number", "int"},
		{"year", "int"},
		{"month", "int"},
		{"day", "int"},

		// Unknown defaults to string
		{"foo", "string"},
		{"bar", "string"},
		{"something", "string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferParamTypeFromName(tt.name)
			if got != tt.expected {
				t.Errorf("inferParamTypeFromName(%q) = %q, want %q", tt.name, got, tt.expected)
			}
		})
	}
}

// TestExtractParams tests parameter extraction from file paths.
func TestExtractParams(t *testing.T) {
	scanner := NewScanner("/test")

	tests := []struct {
		path     string
		expected []ParamDef
	}{
		// Simple parameter
		{
			path: "users/[id].go",
			expected: []ParamDef{
				{Name: "id", Type: "int", Segment: "[id]"},
			},
		},
		// Explicit type
		{
			path: "users/[id:int64].go",
			expected: []ParamDef{
				{Name: "id", Type: "int64", Segment: "[id:int64]"},
			},
		},
		// String parameter
		{
			path: "posts/[slug].go",
			expected: []ParamDef{
				{Name: "slug", Type: "string", Segment: "[slug]"},
			},
		},
		// Multiple parameters
		{
			path: "users/[userId]/posts/[postId].go",
			expected: []ParamDef{
				{Name: "userId", Type: "int", Segment: "[userId]"},
				{Name: "postId", Type: "int", Segment: "[postId]"},
			},
		},
		// Catch-all
		{
			path: "docs/[...path].go",
			expected: []ParamDef{
				{Name: "path", Type: "[]string", Segment: "[...path]"},
			},
		},
		// No parameters
		{
			path:     "about.go",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := scanner.extractParams(tt.path)

			if len(got) != len(tt.expected) {
				t.Fatalf("extractParams(%q) returned %d params, want %d", tt.path, len(got), len(tt.expected))
			}

			for i, param := range got {
				exp := tt.expected[i]
				if param.Name != exp.Name {
					t.Errorf("param[%d].Name = %q, want %q", i, param.Name, exp.Name)
				}
				if param.Type != exp.Type {
					t.Errorf("param[%d].Type = %q, want %q", i, param.Type, exp.Type)
				}
				if param.Segment != exp.Segment {
					t.Errorf("param[%d].Segment = %q, want %q", i, param.Segment, exp.Segment)
				}
			}
		})
	}
}

// TestFilePathToURLPath tests file path to URL path conversion.
func TestFilePathToURLPath(t *testing.T) {
	scanner := NewScanner("/test")

	tests := []struct {
		filePath string
		expected string
	}{
		{"index.go", "/"},
		{"about.go", "/about"},
		{"users/index.go", "/users"},
		{"users/[id].go", "/users/:id"},
		{"users/[userId]/posts/[postId].go", "/users/:userId/posts/:postId"},
		{"docs/[...path].go", "/docs/*path"},
		{"api/health.go", "/api/health"},
		{"layout.go", "/"},
		{"users/layout.go", "/users"},
	}

	for _, tt := range tests {
		t.Run(tt.filePath, func(t *testing.T) {
			got := scanner.filePathToURLPath(tt.filePath)
			if got != tt.expected {
				t.Errorf("filePathToURLPath(%q) = %q, want %q", tt.filePath, got, tt.expected)
			}
		})
	}
}

// TestIsAPIRoute tests API route detection.
func TestIsAPIRoute(t *testing.T) {
	scanner := NewScanner("/test")

	tests := []struct {
		path     string
		expected bool
	}{
		{"api/health.go", true},
		{"api/users/[id].go", true},
		{"api", true},
		{"about.go", false},
		{"users/index.go", false},
		{"apikeys.go", false}, // Not under api/
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := scanner.isAPIRoute(tt.path)
			if got != tt.expected {
				t.Errorf("isAPIRoute(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}
