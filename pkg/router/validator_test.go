package router

import (
	"strings"
	"testing"
)

// =============================================================================
// Duplicate Route Detection Tests
// =============================================================================

func TestValidateDuplicateRoutes(t *testing.T) {
	// Two files resolve to same path
	routes := []ScannedRoute{
		{
			FilePath: "/routes/users/[id].go",
			Path:     "/users/:id",
			HasPage:  true,
		},
		{
			FilePath: "/routes/users/_id_.go",
			Path:     "/users/:id",
			HasPage:  true,
		},
	}

	validator := NewValidator(routes)
	err := validator.Validate()

	if err == nil {
		t.Fatal("Expected validation error for duplicate routes")
	}

	multiErr, ok := err.(*MultiValidationError)
	if !ok {
		t.Fatalf("Expected MultiValidationError, got %T", err)
	}

	if len(multiErr.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(multiErr.Errors))
	}

	if multiErr.Errors[0].Type != ErrorDuplicateRoute {
		t.Errorf("Expected ErrorDuplicateRoute, got %s", multiErr.Errors[0].Type)
	}
}

func TestValidateNoDuplicateRouteForDifferentPaths(t *testing.T) {
	// Different paths - no conflict
	routes := []ScannedRoute{
		{
			FilePath: "/routes/users/[id].go",
			Path:     "/users/:id",
			HasPage:  true,
		},
		{
			FilePath: "/routes/posts/[id].go",
			Path:     "/posts/:id",
			HasPage:  true,
		},
	}

	validator := NewValidator(routes)
	err := validator.Validate()

	if err != nil {
		t.Errorf("Expected no error for different paths, got: %v", err)
	}
}

func TestValidateSkipsLayoutFiles(t *testing.T) {
	// Layout and page at same path - not a conflict
	routes := []ScannedRoute{
		{
			FilePath:  "/routes/_layout.go",
			Path:      "/",
			HasLayout: true,
		},
		{
			FilePath: "/routes/index.go",
			Path:     "/",
			HasPage:  true,
		},
	}

	validator := NewValidator(routes)
	err := validator.Validate()

	if err != nil {
		t.Errorf("Expected no error when layout coexists with page, got: %v", err)
	}
}

func TestValidateSkipsMiddlewareFiles(t *testing.T) {
	// Middleware and page at same path - not a conflict
	routes := []ScannedRoute{
		{
			FilePath:      "/routes/_middleware.go",
			Path:          "/",
			HasMiddleware: true,
		},
		{
			FilePath: "/routes/index.go",
			Path:     "/",
			HasPage:  true,
		},
	}

	validator := NewValidator(routes)
	err := validator.Validate()

	if err != nil {
		t.Errorf("Expected no error when middleware coexists with page, got: %v", err)
	}
}

// =============================================================================
// API Ambiguity Tests
// =============================================================================

func TestValidateAPIAmbiguity(t *testing.T) {
	// Both GET and prefixed GET exported
	routes := []ScannedRoute{
		{
			FilePath: "/routes/api/health.go",
			Path:     "/api/health",
			IsAPI:    true,
			Methods:  []string{"GET", "GET"}, // Both GET() and HealthGET()
		},
	}

	validator := NewValidator(routes)
	err := validator.Validate()

	if err == nil {
		t.Fatal("Expected validation error for API ambiguity")
	}

	multiErr, ok := err.(*MultiValidationError)
	if !ok {
		t.Fatalf("Expected MultiValidationError, got %T", err)
	}

	found := false
	for _, e := range multiErr.Errors {
		if e.Type == ErrorAPIAmbiguity {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected ErrorAPIAmbiguity error")
	}
}

func TestValidateNoAPIAmbiguityForSingleMethod(t *testing.T) {
	// Only one GET method - no conflict
	routes := []ScannedRoute{
		{
			FilePath: "/routes/api/health.go",
			Path:     "/api/health",
			IsAPI:    true,
			Methods:  []string{"GET"},
		},
	}

	validator := NewValidator(routes)
	err := validator.Validate()

	if err != nil {
		t.Errorf("Expected no error for single method, got: %v", err)
	}
}

func TestValidateNoAPIAmbiguityForDifferentMethods(t *testing.T) {
	// Different methods - no conflict
	routes := []ScannedRoute{
		{
			FilePath: "/routes/api/users.go",
			Path:     "/api/users",
			IsAPI:    true,
			Methods:  []string{"GET", "POST", "PUT"},
		},
	}

	validator := NewValidator(routes)
	err := validator.Validate()

	if err != nil {
		t.Errorf("Expected no error for different methods, got: %v", err)
	}
}

// =============================================================================
// Specificity Sorting Tests
// =============================================================================

func TestSortBySpecificityStaticFirst(t *testing.T) {
	routes := []ScannedRoute{
		{Path: "/users/:id", Params: []ParamDef{{Name: "id", Type: "string"}}},
		{Path: "/users/profile"},
	}

	SortBySpecificity(routes)

	if routes[0].Path != "/users/profile" {
		t.Errorf("Expected static /users/profile first, got %s", routes[0].Path)
	}
}

func TestSortBySpecificityTypedParamBeforePlain(t *testing.T) {
	routes := []ScannedRoute{
		{Path: "/users/:id", Params: []ParamDef{{Name: "id", Type: "string"}}},
		{Path: "/users/:id", Params: []ParamDef{{Name: "id", Type: "int"}}},
	}

	SortBySpecificity(routes)

	// Typed (int) should come before plain (string)
	if routes[0].Params[0].Type != "int" {
		t.Errorf("Expected typed param first, got type=%s", routes[0].Params[0].Type)
	}
}

func TestSortBySpecificityCatchAllLast(t *testing.T) {
	routes := []ScannedRoute{
		{Path: "/docs/*path", IsCatchAll: true},
		{Path: "/docs/:section"},
		{Path: "/docs/intro"},
	}

	SortBySpecificity(routes)

	// Static first, then param, then catch-all
	if routes[0].Path != "/docs/intro" {
		t.Errorf("Expected static first, got %s", routes[0].Path)
	}
	if routes[2].Path != "/docs/*path" {
		t.Errorf("Expected catch-all last, got %s", routes[2].Path)
	}
}

func TestSortBySpecificityMoreSegmentsFirst(t *testing.T) {
	routes := []ScannedRoute{
		{Path: "/users"},
		{Path: "/users/:id/posts"},
	}

	SortBySpecificity(routes)

	// More segments = more specific
	if routes[0].Path != "/users/:id/posts" {
		t.Errorf("Expected longer path first, got %s", routes[0].Path)
	}
}

// =============================================================================
// ValidateAndSort Integration Test
// =============================================================================

func TestValidateAndSort(t *testing.T) {
	routes := []ScannedRoute{
		{Path: "/users/:id", HasPage: true, Params: []ParamDef{{Name: "id", Type: "string"}}},
		{Path: "/users/profile", HasPage: true},
		{Path: "/users", HasPage: true},
	}

	sorted, err := ValidateAndSort(routes)
	if err != nil {
		t.Fatalf("ValidateAndSort failed: %v", err)
	}

	if len(sorted) != 3 {
		t.Errorf("Expected 3 routes, got %d", len(sorted))
	}

	// Should be sorted by specificity
	expected := []string{"/users/profile", "/users/:id", "/users"}
	for i, route := range sorted {
		if route.Path != expected[i] {
			t.Errorf("Route %d: expected %s, got %s", i, expected[i], route.Path)
		}
	}
}

func TestValidateAndSortReturnsErrorOnDuplicate(t *testing.T) {
	routes := []ScannedRoute{
		{Path: "/users/:id", HasPage: true, FilePath: "a.go"},
		{Path: "/users/:id", HasPage: true, FilePath: "b.go"},
	}

	_, err := ValidateAndSort(routes)
	if err == nil {
		t.Error("Expected error for duplicate routes")
	}
}

// =============================================================================
// Error Formatting Tests
// =============================================================================

func TestFormatValidationError(t *testing.T) {
	err := ValidationError{
		Type:    ErrorDuplicateRoute,
		Message: "Duplicate route detected at /users/:id",
		Path:    "/users/:id",
		Files:   []string{"/routes/users/[id].go", "/routes/users/_id_.go"},
		Details: "Multiple files resolve to same path",
	}

	formatted := FormatValidationError(err)

	if !strings.Contains(formatted, "ERROR:") {
		t.Error("Expected ERROR: prefix")
	}
	if !strings.Contains(formatted, "/users/:id") {
		t.Error("Expected path in output")
	}
	if !strings.Contains(formatted, "[id].go") {
		t.Error("Expected file path in output")
	}
}

func TestMultiValidationErrorString(t *testing.T) {
	multiErr := &MultiValidationError{
		Errors: []ValidationError{
			{Type: ErrorDuplicateRoute, Message: "Error 1"},
			{Type: ErrorAPIAmbiguity, Message: "Error 2"},
		},
	}

	str := multiErr.Error()

	if !strings.Contains(str, "2 route validation errors") {
		t.Error("Expected error count in message")
	}
	if !strings.Contains(str, "Error 1") {
		t.Error("Expected first error")
	}
	if !strings.Contains(str, "Error 2") {
		t.Error("Expected second error")
	}
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestValidateEmptyRoutes(t *testing.T) {
	validator := NewValidator([]ScannedRoute{})
	err := validator.Validate()

	if err != nil {
		t.Errorf("Expected no error for empty routes, got: %v", err)
	}
}

func TestValidateSingleRoute(t *testing.T) {
	routes := []ScannedRoute{
		{Path: "/", HasPage: true},
	}

	validator := NewValidator(routes)
	err := validator.Validate()

	if err != nil {
		t.Errorf("Expected no error for single route, got: %v", err)
	}
}

func TestSortBySpecificityEmpty(t *testing.T) {
	routes := []ScannedRoute{}
	SortBySpecificity(routes) // Should not panic
}

func TestSortBySpecificitySingle(t *testing.T) {
	routes := []ScannedRoute{{Path: "/"}}
	SortBySpecificity(routes) // Should not panic

	if routes[0].Path != "/" {
		t.Error("Single route should remain unchanged")
	}
}
