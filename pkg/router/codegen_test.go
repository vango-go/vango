package router

import (
	"strings"
	"testing"
)

func TestGeneratorGenerate(t *testing.T) {
	// Current format: generates Register(app *vango.App) registration glue.
	routes := []ScannedRoute{
		{
			Path:      "/",
			FilePath:  "layout.go",
			Package:   "routes",
			HasLayout: true,
		},
		{
			Path:     "/",
			FilePath: "index.go",
			Package:  "routes",
			HasPage:  true,
		},
		{
			Path:     "/about",
			FilePath: "about.go",
			Package:  "routes",
			HasPage:  true,
			HasMeta:  true,
		},
		{
			Path:             "/api",
			FilePath:         "api/middleware.go",
			Package:          "api",
			IsAPI:            true,
			HasMiddleware:    true,
			MiddlewareIsFunc: true,
		},
		{
			Path:     "/users/:id",
			FilePath: "users/[id].go",
			Package:  "routes",
			HasPage:  true,
			Params: []ParamDef{
				{Name: "id", Type: "int", Segment: "[id]"},
			},
		},
		{
			Path:     "/api/users",
			FilePath: "api/users.go",
			Package:  "api",
			IsAPI:    true,
			Methods:  []string{"GET", "POST"},
			APIHandlers: map[string]string{
				"GET":  "GET",
				"POST": "UsersPOST",
			},
		},
		{
			Path:       "/docs/*path",
			FilePath:   "docs/[...path].go",
			Package:    "routes",
			HasPage:    true,
			IsCatchAll: true,
			Params: []ParamDef{
				{Name: "path", Type: "[]string", Segment: "[...path]"},
			},
		},
	}

	gen := NewGenerator(routes, "github.com/example/app")
	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	code := string(output)

	// Check header
	if !strings.Contains(code, "DO NOT EDIT") {
		t.Error("missing DO NOT EDIT header")
	}

	// Check package
	if !strings.Contains(code, "package routes") {
		t.Error("missing package declaration")
	}

	// Check param structs - uses ID abbreviation handling
	if !strings.Contains(code, "type UsersIDParams struct") {
		t.Error("missing UsersIDParams struct")
	}
	if !strings.Contains(code, `ID int`) {
		t.Error("missing ID field in UsersIDParams")
	}

	// Check catch-all param struct
	if !strings.Contains(code, "type DocsPathParams struct") {
		t.Error("missing DocsPathParams struct")
	}
	if !strings.Contains(code, `Path []string`) {
		t.Error("missing Path field in DocsPathParams")
	}

	// Check Register function signature
	if !strings.Contains(code, "func Register(app *vango.App)") {
		t.Error("missing Register function with app *vango.App signature")
	}

	// Check layout registration
	if !strings.Contains(code, `app.Layout("/", Layout)`) {
		t.Error("missing root layout registration")
	}

	// Check middleware registration (directory scoped)
	if !strings.Contains(code, `app.Middleware("/api", api.Middleware()...)`) {
		t.Error("missing /api middleware registration")
	}

	// Check page registration
	if !strings.Contains(code, `app.Page("/", IndexPage)`) {
		t.Error("missing / page registration")
	}
	if !strings.Contains(code, `app.Page("/about", AboutPage)`) {
		t.Error("missing /about page registration")
	}
	if strings.Contains(code, `app.Page("/", IndexPage, Layout`) || strings.Contains(code, `app.Page("/about", AboutPage, Layout`) {
		t.Error("pages should not be registered with explicit layouts (hierarchical layouts come from app.Layout)")
	}

	// Check API registration uses the discovered handler name (GET() vs UsersGET())
	if !strings.Contains(code, `app.API("GET", "/api/users", api.GET)`) {
		t.Error("missing /api/users GET registration with bare GET handler")
	}
	if !strings.Contains(code, `app.API("POST", "/api/users", api.UsersPOST)`) {
		t.Error("missing /api/users POST registration with UsersPOST handler")
	}

	// Check route constants
	if !strings.Contains(code, "const (") {
		t.Error("missing route constants")
	}
	if !strings.Contains(code, `RouteIndex = "/"`) {
		t.Error("missing RouteIndex constant")
	}
	if !strings.Contains(code, `RouteAbout = "/about"`) {
		t.Error("missing RouteAbout constant")
	}
}

func TestGeneratorPathToStructName(t *testing.T) {
	gen := NewGenerator(nil, "")

	tests := []struct {
		path string
		want string
	}{
		{"/", "Index"},
		{"/about", "About"},
		{"/users/:id", "UsersID"},
		{"/users/:userId/posts/:postId", "UsersUserIdPostsPostId"},
		{"/*slug", "Slug"},
		{"/docs/*path", "DocsPath"},
	}

	for _, tt := range tests {
		got := gen.pathToStructName(tt.path)
		if got != tt.want {
			t.Errorf("pathToStructName(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestGeneratorToExportedName(t *testing.T) {
	gen := NewGenerator(nil, "")

	tests := []struct {
		input string
		want  string
	}{
		{"id", "ID"}, // "id" is a common abbreviation
		{"ID", "ID"},
		{"userId", "UserId"},
		{"url", "URL"},
		{"api", "API"},
		{"uuid", "UUID"},
		{"http", "HTTP"},
	}

	for _, tt := range tests {
		got := gen.toExportedName(tt.input)
		if got != tt.want {
			t.Errorf("toExportedName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestGeneratorParamTypeToGoType(t *testing.T) {
	gen := NewGenerator(nil, "")

	tests := []struct {
		paramType string
		want      string
	}{
		{"int", "int"},
		{"int64", "int64"},
		{"int32", "int32"},
		{"uint", "uint"},
		{"uint64", "uint64"},
		{"uuid", "string"},
		{"string", "string"},
		{"", "string"},
		{"[]string", "[]string"},
		{"unknown", "string"},
	}

	for _, tt := range tests {
		got := gen.paramTypeToGoType(tt.paramType)
		if got != tt.want {
			t.Errorf("paramTypeToGoType(%q) = %q, want %q", tt.paramType, got, tt.want)
		}
	}
}

func TestSanitizeIdentifier(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"users", "users"},
		{"user-profile", "user_profile"},
		{"my.route", "my_route"},
		{"[id]", "id"},
		{"123start", "r123start"},
	}

	for _, tt := range tests {
		got := sanitizeIdentifier(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeIdentifier(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
