package router

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewOpenAPIGenerator(t *testing.T) {
	gen := NewOpenAPIGenerator("/app/routes", "github.com/example/app", OpenAPIInfo{
		Title:   "My API",
		Version: "2.0.0",
	})

	if gen == nil {
		t.Fatal("NewOpenAPIGenerator returned nil")
	}

	if gen.info.Title != "My API" {
		t.Errorf("Title = %q, want %q", gen.info.Title, "My API")
	}

	if gen.info.Version != "2.0.0" {
		t.Errorf("Version = %q, want %q", gen.info.Version, "2.0.0")
	}
}

func TestNewOpenAPIGenerator_Defaults(t *testing.T) {
	gen := NewOpenAPIGenerator("/app/routes", "github.com/example/app", OpenAPIInfo{})

	if gen.info.Title != "API" {
		t.Errorf("Default Title = %q, want %q", gen.info.Title, "API")
	}

	if gen.info.Version != "1.0.0" {
		t.Errorf("Default Version = %q, want %q", gen.info.Version, "1.0.0")
	}
}

func TestOpenAPIGenerator_Generate(t *testing.T) {
	// Create temporary API structure
	tmpDir := t.TempDir()
	routesDir := filepath.Join(tmpDir, "routes")
	apiDir := filepath.Join(routesDir, "api")
	os.MkdirAll(apiDir, 0755)

	// Create a sample API file
	healthGo := `package api

import "github.com/vango-go/vango"

// HealthResponse is the JSON response for health checks.
type HealthResponse struct {
	Status  string ` + "`json:\"status\"`" + `
	Version string ` + "`json:\"version\"`" + `
}

// HealthGET handles GET /api/health.
func HealthGET(ctx vango.Ctx) (*HealthResponse, error) {
	return &HealthResponse{
		Status:  "ok",
		Version: "1.0.0",
	}, nil
}
`
	os.WriteFile(filepath.Join(apiDir, "health.go"), []byte(healthGo), 0644)

	// Create generator
	gen := NewOpenAPIGenerator(routesDir, "github.com/example/app", OpenAPIInfo{
		Title:       "Test API",
		Description: "Test API Description",
		Version:     "1.0.0",
	})

	// Generate spec
	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	// Parse the output
	var spec OpenAPISpec
	if err := json.Unmarshal(output, &spec); err != nil {
		t.Fatalf("Failed to parse generated spec: %v", err)
	}

	// Verify basic structure
	if spec.OpenAPI != "3.0.3" {
		t.Errorf("OpenAPI version = %q, want %q", spec.OpenAPI, "3.0.3")
	}

	if spec.Info.Title != "Test API" {
		t.Errorf("Info.Title = %q, want %q", spec.Info.Title, "Test API")
	}

	if spec.Info.Description != "Test API Description" {
		t.Errorf("Info.Description = %q, want %q", spec.Info.Description, "Test API Description")
	}

	// Verify paths
	if len(spec.Paths) == 0 {
		t.Error("Expected at least one path")
	}

	healthPath, ok := spec.Paths["/api/health"]
	if !ok {
		t.Fatal("Missing /api/health path")
	}

	getOp, ok := healthPath["get"]
	if !ok {
		t.Fatal("Missing GET operation for /api/health")
	}

	if getOp.OperationID != "HealthGET" {
		t.Errorf("OperationID = %q, want %q", getOp.OperationID, "HealthGET")
	}

	// Verify schemas
	if spec.Components == nil || spec.Components.Schemas == nil {
		t.Fatal("Missing components/schemas")
	}

	healthSchema, ok := spec.Components.Schemas["HealthResponse"]
	if !ok {
		t.Fatal("Missing HealthResponse schema")
	}

	if healthSchema.Type != "object" {
		t.Errorf("HealthResponse type = %q, want %q", healthSchema.Type, "object")
	}

	if _, ok := healthSchema.Properties["status"]; !ok {
		t.Error("Missing 'status' property in HealthResponse")
	}

	if _, ok := healthSchema.Properties["version"]; !ok {
		t.Error("Missing 'version' property in HealthResponse")
	}
}

func TestOpenAPIGenerator_WithPathParams(t *testing.T) {
	// Create temporary API structure
	tmpDir := t.TempDir()
	routesDir := filepath.Join(tmpDir, "routes")
	apiDir := filepath.Join(routesDir, "api", "users")
	os.MkdirAll(apiDir, 0755)

	// Create API file with path parameter
	usersGo := `package users

import "github.com/vango-go/vango"

type UserResponse struct {
	ID    string ` + "`json:\"id\"`" + `
	Name  string ` + "`json:\"name\"`" + `
	Email string ` + "`json:\"email\"`" + `
}

// UserGET handles GET /api/users/:id.
func UserGET(ctx vango.Ctx) (*UserResponse, error) {
	return &UserResponse{ID: "1"}, nil
}
`
	os.WriteFile(filepath.Join(apiDir, "[id].go"), []byte(usersGo), 0644)

	// Generate
	gen := NewOpenAPIGenerator(routesDir, "github.com/example/app", OpenAPIInfo{
		Title:   "User API",
		Version: "1.0.0",
	})

	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}

	var spec OpenAPISpec
	if err := json.Unmarshal(output, &spec); err != nil {
		t.Fatalf("Failed to parse generated spec: %v", err)
	}

	// Check that path uses OpenAPI format {id}
	found := false
	for path := range spec.Paths {
		if strings.Contains(path, "{id}") {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected path with {id} parameter, got paths: %v", spec.Paths)
	}
}

func TestConvertParamsToOpenAPI(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"users/[id]", "users/{id}"},
		{"users/[id:int]", "users/{id}"},
		{"posts/[slug]", "posts/{slug}"},
		{"api/v1/users/[userId]/posts/[postId]", "api/v1/users/{userId}/posts/{postId}"},
	}

	for _, tt := range tests {
		got := convertParamsToOpenAPI(tt.input)
		if got != tt.expected {
			t.Errorf("convertParamsToOpenAPI(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestExtractHTTPMethod(t *testing.T) {
	tests := []struct {
		funcName string
		expected string
	}{
		{"HealthGET", "GET"},
		{"UsersPOST", "POST"},
		{"UserPUT", "PUT"},
		{"UserDELETE", "DELETE"},
		{"UserPATCH", "PATCH"},
		{"HomePage", ""},
		{"SomeFunction", ""},
	}

	for _, tt := range tests {
		got := extractHTTPMethod(tt.funcName)
		if got != tt.expected {
			t.Errorf("extractHTTPMethod(%q) = %q, want %q", tt.funcName, got, tt.expected)
		}
	}
}

func TestGoTypeToOpenAPIType(t *testing.T) {
	gen := NewOpenAPIGenerator("/routes", "module", OpenAPIInfo{})

	tests := []struct {
		goType   string
		expected string
	}{
		{"string", "string"},
		{"int", "integer"},
		{"int64", "integer"},
		{"uint", "integer"},
		{"float64", "number"},
		{"bool", "boolean"},
		{"*string", "string"},
		{"[]string", "array"},
		{"SomeStruct", "object"},
	}

	for _, tt := range tests {
		got := gen.goTypeToOpenAPIType(tt.goType)
		if got != tt.expected {
			t.Errorf("goTypeToOpenAPIType(%q) = %q, want %q", tt.goType, got, tt.expected)
		}
	}
}

func TestOpenAPISpec_JSON(t *testing.T) {
	spec := OpenAPISpec{
		OpenAPI: "3.0.3",
		Info: OpenAPISpecInfo{
			Title:       "Test",
			Description: "Desc",
			Version:     "1.0.0",
		},
		Paths: map[string]OpenAPIPath{
			"/test": {
				"get": &OpenAPIOperation{
					OperationID: "testGet",
					Responses: map[string]OpenAPIResponse{
						"200": {Description: "OK"},
					},
				},
			},
		},
	}

	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	if !strings.Contains(string(data), `"openapi": "3.0.3"`) {
		t.Error("Missing openapi version in output")
	}

	if !strings.Contains(string(data), `"title": "Test"`) {
		t.Error("Missing title in output")
	}

	if !strings.Contains(string(data), `"/test"`) {
		t.Error("Missing path in output")
	}
}

func TestExtractParamsFromURLPath(t *testing.T) {
	tests := []struct {
		path       string
		paramCount int
	}{
		{"/api/health", 0},
		{"/api/users/{id}", 1},
		{"/api/users/{userId}/posts/{postId}", 2},
	}

	for _, tt := range tests {
		params := extractParamsFromURLPath(tt.path)
		if len(params) != tt.paramCount {
			t.Errorf("extractParamsFromURLPath(%q): got %d params, want %d", tt.path, len(params), tt.paramCount)
		}
	}
}
