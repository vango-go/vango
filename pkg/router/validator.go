package router

import (
	"fmt"
	"sort"
	"strings"
)

// =============================================================================
// Route Validation (Section 1.3, 1.6)
// =============================================================================

// Validator validates scanned routes for conflicts and errors.
// Per Section 1.3 and 1.6 of the Routing Spec.
type Validator struct {
	routes []ScannedRoute
	errors []ValidationError
}

// ValidationError represents a route validation error.
type ValidationError struct {
	// Type is the error category
	Type ValidationErrorType

	// Message is the human-readable error message
	Message string

	// Files are the source files involved
	Files []string

	// Path is the conflicting URL pattern
	Path string

	// Details contains additional error-specific information
	Details string
}

func (e ValidationError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Type, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// ValidationErrorType categorizes validation errors.
type ValidationErrorType string

const (
	// ErrorDuplicateRoute indicates multiple files resolve to the same URL pattern.
	// Example: [id].go and _id_.go both resolve to /:id
	ErrorDuplicateRoute ValidationErrorType = "DUPLICATE_ROUTE"

	// ErrorParamConstraintConflict indicates the same param has different type constraints.
	// Example: [id:int].go and [id].go at the same path
	ErrorParamConstraintConflict ValidationErrorType = "PARAM_CONSTRAINT_CONFLICT"

	// ErrorAPIAmbiguity indicates both bare (GET) and prefixed (HealthGET) handlers exist.
	// Example: Both GET() and HealthGET() exported in /api/health.go
	ErrorAPIAmbiguity ValidationErrorType = "API_AMBIGUITY"

	// ErrorParamTypeMismatch indicates annotation type differs from Params struct field type.
	// Example: [id:int].go but Params struct has id as string
	ErrorParamTypeMismatch ValidationErrorType = "PARAM_TYPE_MISMATCH"
)

// MultiValidationError wraps multiple validation errors.
type MultiValidationError struct {
	Errors []ValidationError
}

func (e *MultiValidationError) Error() string {
	if len(e.Errors) == 0 {
		return "no validation errors"
	}
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d route validation errors:\n", len(e.Errors)))
	for i, err := range e.Errors {
		sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err.Error()))
	}
	return sb.String()
}

// NewValidator creates a new route validator.
func NewValidator(routes []ScannedRoute) *Validator {
	return &Validator{
		routes: routes,
	}
}

// Validate checks all routes for conflicts and errors.
// Returns nil if all routes are valid, or a MultiValidationError with all errors.
func (v *Validator) Validate() error {
	v.errors = nil

	v.validateDuplicateRoutes()
	v.validateParamConstraints()
	v.validateAPIAmbiguity()

	if len(v.errors) > 0 {
		return &MultiValidationError{Errors: v.errors}
	}
	return nil
}

// validateDuplicateRoutes checks for routes that resolve to the same URL pattern.
// Per Section 1.3: Duplicate route files should be detected and reported.
// Example: /projects/[id].go and /projects/_id_.go both → /projects/:id
func (v *Validator) validateDuplicateRoutes() {
	// Group routes by their resolved URL pattern
	// Only consider actual page/API routes, not layouts, middleware, or error pages
	byPath := make(map[string][]ScannedRoute)
	for _, route := range v.routes {
		// Skip non-route files (layouts, middleware, error handlers)
		if route.HasLayout && !route.HasPage {
			continue
		}
		if route.HasMiddleware && !route.HasPage {
			continue
		}
		// Skip files that don't define any handlers
		if !route.HasPage && len(route.Methods) == 0 {
			continue
		}

		byPath[route.Path] = append(byPath[route.Path], route)
	}

	// Check for duplicates
	for path, routes := range byPath {
		if len(routes) <= 1 {
			continue
		}

		// Multiple routes resolve to the same path - that's a conflict
		files := make([]string, len(routes))
		for i, r := range routes {
			files[i] = r.FilePath
		}

		v.errors = append(v.errors, ValidationError{
			Type:    ErrorDuplicateRoute,
			Message: fmt.Sprintf("Duplicate route detected at %s", path),
			Path:    path,
			Files:   files,
			Details: fmt.Sprintf("Files: %s", strings.Join(files, ", ")),
		})
	}
}

// validateParamConstraints checks for conflicting type constraints on the same parameter.
// Per Section 1.6: Parameter constraints must be consistent.
// Example: /users/[id:int].go and /users/[id].go at the same path would conflict.
func (v *Validator) validateParamConstraints() {
	// Group routes by parent path (directory)
	type paramKey struct {
		parentPath string
		paramName  string
	}
	paramTypes := make(map[paramKey][]struct {
		typ      string
		filePath string
	})

	for _, route := range v.routes {
		for _, param := range route.Params {
			// Get the parent path (directory containing this file)
			parentPath := getParentPath(route.Path)

			key := paramKey{parentPath: parentPath, paramName: param.Name}
			paramTypes[key] = append(paramTypes[key], struct {
				typ      string
				filePath string
			}{
				typ:      param.Type,
				filePath: route.FilePath,
			})
		}
	}

	// Check for conflicts
	for key, entries := range paramTypes {
		if len(entries) <= 1 {
			continue
		}

		// Check if all entries have the same type
		firstType := entries[0].typ
		hasConflict := false
		for _, e := range entries[1:] {
			if e.typ != firstType {
				hasConflict = true
				break
			}
		}

		if !hasConflict {
			continue
		}

		// Conflict found
		files := make([]string, len(entries))
		types := make([]string, len(entries))
		for i, e := range entries {
			files[i] = e.filePath
			types[i] = e.typ
		}

		v.errors = append(v.errors, ValidationError{
			Type:    ErrorParamConstraintConflict,
			Message: fmt.Sprintf("Conflicting parameter constraints for '%s' at %s", key.paramName, key.parentPath),
			Path:    key.parentPath,
			Files:   files,
			Details: fmt.Sprintf("Types: %s", strings.Join(types, " vs ")),
		})
	}
}

// validateAPIAmbiguity checks for ambiguous API handler exports.
// Per Section 1.6: Both bare (GET) and prefixed (HealthGET) handlers should not exist.
// Example: /api/health.go exports both GET() and HealthGET() for GET method.
func (v *Validator) validateAPIAmbiguity() {
	for _, route := range v.routes {
		if !route.IsAPI {
			continue
		}

		// Count methods - if same method appears multiple times, that's ambiguous
		methodCounts := make(map[string]int)
		for _, method := range route.Methods {
			methodCounts[method]++
		}

		for method, count := range methodCounts {
			if count > 1 {
				v.errors = append(v.errors, ValidationError{
					Type:    ErrorAPIAmbiguity,
					Message: fmt.Sprintf("Ambiguous API handler exports in %s", route.FilePath),
					Path:    route.Path,
					Files:   []string{route.FilePath},
					Details: fmt.Sprintf("Both %s() and prefixed %s handler are exported for %s method", method, method, method),
				})
			}
		}
	}
}

// getParentPath returns the parent directory path of a URL path.
func getParentPath(urlPath string) string {
	// Remove trailing param segment
	lastSlash := strings.LastIndex(urlPath, "/")
	if lastSlash <= 0 {
		return "/"
	}
	return urlPath[:lastSlash]
}

// =============================================================================
// Route Specificity Sorting (Section 1.3.1)
// =============================================================================

// SortBySpecificity sorts routes by specificity for proper matching order.
// Per Section 1.3.1:
//   - Static segments > typed parameters > plain parameters > catch-all
//   - More specific routes should be matched first
//
// Order (most specific first):
//  1. Static routes (/users/profile)
//  2. Routes with typed params (/users/:id where id is int)
//  3. Routes with plain params (/users/:id where id is string)
//  4. Catch-all routes (/users/*path)
func SortBySpecificity(routes []ScannedRoute) {
	sort.SliceStable(routes, func(i, j int) bool {
		specI := calculateSpecificity(routes[i])
		specJ := calculateSpecificity(routes[j])
		return specI > specJ // Higher specificity first
	})
}

// calculateSpecificity returns a numeric score for route specificity.
// Higher scores = more specific = matched first.
func calculateSpecificity(route ScannedRoute) int {
	// Catch-all routes are least specific
	if route.IsCatchAll {
		return 0
	}

	// Start with segment count (more segments = more specific)
	segments := strings.Split(strings.Trim(route.Path, "/"), "/")
	if route.Path == "/" {
		segments = []string{}
	}
	score := len(segments) * 100

	// Add points for static vs dynamic segments
	for _, seg := range segments {
		if strings.HasPrefix(seg, ":") {
			// Dynamic segment
			// Check if typed (more specific than plain string)
			paramName := seg[1:]
			if isTypedParam(paramName, route.Params) {
				score += 20 // Typed param
			} else {
				score += 10 // Plain string param
			}
		} else if strings.HasPrefix(seg, "*") {
			// Catch-all segment (handled above, but just in case)
			score += 1
		} else {
			// Static segment (most specific)
			score += 50
		}
	}

	return score
}

// isTypedParam checks if a parameter has an explicit type constraint.
func isTypedParam(name string, params []ParamDef) bool {
	for _, p := range params {
		if p.Name == name {
			// Check if it's not the default string type
			return p.Type != "" && p.Type != "string"
		}
	}
	return false
}

// =============================================================================
// Helper Functions
// =============================================================================

// ValidateAndSort validates routes and sorts them by specificity.
// This is the main entry point for the scanner to call.
func ValidateAndSort(routes []ScannedRoute) ([]ScannedRoute, error) {
	// Validate first
	validator := NewValidator(routes)
	if err := validator.Validate(); err != nil {
		return nil, err
	}

	// Sort by specificity
	SortBySpecificity(routes)

	return routes, nil
}

// FormatValidationError formats a validation error for display.
// Per the spec, errors should be formatted clearly:
//
//	ERROR: Duplicate route detected
//	  /projects/[id].go → /projects/:id
//	  /projects/_id_.go → /projects/:id
func FormatValidationError(err ValidationError) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("ERROR: %s\n", err.Message))

	if len(err.Files) > 0 {
		for _, file := range err.Files {
			sb.WriteString(fmt.Sprintf("  %s → %s\n", file, err.Path))
		}
	}

	if err.Details != "" {
		sb.WriteString(fmt.Sprintf("  Details: %s\n", err.Details))
	}

	return sb.String()
}
