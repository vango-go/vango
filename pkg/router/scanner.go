package router

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
)

// Note: We use token.VAR to detect variable declarations (like var Middleware = ...)

// Scanner scans a directory for route files.
type Scanner struct {
	rootDir string
}

// NewScanner creates a new route scanner.
func NewScanner(rootDir string) *Scanner {
	return &Scanner{rootDir: rootDir}
}

// Scan reads all route files and returns route definitions.
// Routes are validated and sorted by specificity.
func (s *Scanner) Scan() ([]ScannedRoute, error) {
	return s.ScanWithOptions(ScanOptions{Validate: true, Sort: true})
}

// ScanOptions configures scanning behavior.
type ScanOptions struct {
	// Validate enables route validation (duplicate detection, constraint conflicts, etc.)
	Validate bool

	// Sort enables specificity sorting (static > typed > plain > catch-all)
	Sort bool
}

// ScanWithOptions reads all route files with configurable validation and sorting.
func (s *Scanner) ScanWithOptions(opts ScanOptions) ([]ScannedRoute, error) {
	var routes []ScannedRoute

	err := filepath.WalkDir(s.rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Skip non-Go files
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip test files
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		route, err := s.scanFile(path)
		if err != nil {
			return fmt.Errorf("scanning %s: %w", path, err)
		}

		if route != nil {
			routes = append(routes, *route)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Validate routes if enabled
	if opts.Validate {
		validator := NewValidator(routes)
		if err := validator.Validate(); err != nil {
			return nil, err
		}
	}

	// Sort by specificity if enabled
	if opts.Sort {
		SortBySpecificity(routes)
	}

	return routes, nil
}

// scanFile parses a Go file and extracts route information.
func (s *Scanner) scanFile(path string) (*ScannedRoute, error) {
	// Parse Go file
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	route := &ScannedRoute{
		FilePath: path,
		Package:  f.Name.Name,
	}

	// Determine URL path from file path
	relPath, err := filepath.Rel(s.rootDir, path)
	if err != nil {
		return nil, err
	}

	route.Path = s.filePathToURLPath(relPath)
	route.Params = s.extractParams(relPath)
	// Check for catch-all in both notation styles
	route.IsCatchAll = strings.Contains(relPath, "[...") || strings.Contains(relPath, "___")

	// Check for special files
	baseName := filepath.Base(path)
	switch baseName {
	case "_layout.go":
		route.HasLayout = true
	case "_middleware.go":
		route.HasMiddleware = true
	case "_error.go", "_404.go":
		// Error handlers - still scan for functions
	}

	// Check for API route
	route.IsAPI = s.isAPIRoute(relPath)

	// Scan for exported functions and variables
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Name == nil || !d.Name.IsExported() {
				continue
			}

			name := d.Name.Name

			// Check for Layout function
			if name == "Layout" {
				route.HasLayout = true
				continue
			}

			// Check for Middleware function
			if name == "Middleware" {
				route.HasMiddleware = true
				continue
			}

			// Check for Meta function
			if name == "Meta" {
				route.HasMeta = true
				continue
			}

			// Check for HTTP method handlers (API routes)
			switch name {
			case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
				route.Methods = append(route.Methods, name)
				continue
			}

			// Check for page handler functions (IndexPage, AboutPage, ShowPage, etc.)
			// Per spec: function names ending in "Page" are page handlers
			if strings.HasSuffix(name, "Page") {
				route.HasPage = true
				route.HandlerName = name
				continue
			}

			// Also detect {Resource}GET, {Resource}POST patterns for API routes
			// e.g., UsersGET, HealthGET, UsersPOST
			for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"} {
				if strings.HasSuffix(name, method) {
					route.Methods = append(route.Methods, method)
					break
				}
			}

		case *ast.GenDecl:
			// Check for Middleware variable (var Middleware = []router.Middleware{...})
			if d.Tok != token.VAR {
				continue
			}
			for _, spec := range d.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, ident := range vs.Names {
					if ident.Name == "Middleware" && ident.IsExported() {
						route.HasMiddleware = true
					}
				}
			}
		}
	}

	return route, nil
}

// filePathToURLPath converts a file path to a URL path.
// Supports two dynamic route conventions:
//
//  1. Bracket notation: [id].go → :id (Next.js/Remix style)
//     - May not work on Windows or with some Go toolchains
//
//  2. Underscore notation: _id_.go → :id (Go-friendly)
//     - Preferred for compatibility with all systems
//     - Leading underscore indicates dynamic segment
//     - Trailing underscore distinguishes from reserved files like _layout.go
//
// Examples:
//   - projects/_id_.go  → /projects/:id
//   - users/_id_/posts/_postId_.go → /users/:id/posts/:postId
//   - [id].go → :id (if your system supports it)
func (s *Scanner) filePathToURLPath(relPath string) string {
	// Remove .go extension
	path := strings.TrimSuffix(relPath, ".go")

	// Convert Windows path separators
	path = strings.ReplaceAll(path, "\\", "/")

	// Handle index files
	if strings.HasSuffix(path, "/index") {
		path = strings.TrimSuffix(path, "/index")
	}
	if path == "index" {
		path = ""
	}

	// Handle layout and error files (they define handlers but not routes)
	baseName := filepath.Base(path)
	if strings.HasPrefix(baseName, "_") && !strings.HasSuffix(baseName, "_") {
		// Special files start with _ but don't end with _ (e.g., _layout, _middleware)
		// Get the directory path for layouts
		dir := filepath.Dir(path)
		if dir == "." {
			path = ""
		} else {
			path = strings.ReplaceAll(dir, "\\", "/")
		}
	}

	// Convert params to router notation
	path = s.convertParams(path)

	// Add leading slash
	if path == "" {
		return "/"
	}
	return "/" + path
}

// convertParams converts parameter notation to router notation.
// Supports two conventions:
//
//  1. Bracket notation (Next.js/Remix style):
//     - [id] → :id
//     - [id:int] → :id (type stored separately)
//     - [...slug] → *slug (catch-all)
//
//  2. Underscore notation (Go-friendly, preferred):
//     - _id_ → :id
//     - _slug___ → *slug (triple underscore for catch-all)
func (s *Scanner) convertParams(path string) string {
	// First handle bracket notation: [param] or [param:type] or [...param]
	bracketRe := regexp.MustCompile(`\[([.\w]+)(?::(\w+))?\]`)
	result := bracketRe.ReplaceAllStringFunc(path, func(match string) string {
		inner := match[1 : len(match)-1] // Remove brackets

		// Handle catch-all
		if strings.HasPrefix(inner, "...") {
			return "*" + inner[3:]
		}

		// Handle typed params - just use the name
		if idx := strings.Index(inner, ":"); idx != -1 {
			return ":" + inner[:idx]
		}

		return ":" + inner
	})

	// Then handle underscore notation: _param_ or _param___
	// Triple underscore suffix indicates catch-all: _slug___ → *slug
	catchAllRe := regexp.MustCompile(`_(\w+)___`)
	result = catchAllRe.ReplaceAllString(result, "*$1")

	// Single underscore suffix for regular params: _id_ → :id
	paramRe := regexp.MustCompile(`_(\w+)_`)
	result = paramRe.ReplaceAllString(result, ":$1")

	return result
}

// extractParams extracts parameter definitions from a file path.
// Uses intelligent type inference based on naming conventions:
//   - [id], [userID], [postId], _id_ → int (common ID pattern)
//   - [uuid], _uuid_ → string (UUID stored as string)
//   - [slug], [name], [title], _slug_ → string
//   - [...path], [...rest], _path___ → []string (catch-all)
//   - [param:int], [id:int64] → explicit type annotation
func (s *Scanner) extractParams(relPath string) []ParamDef {
	var params []ParamDef

	// Match bracket notation: [param] or [param:type] or [...param]
	bracketRe := regexp.MustCompile(`\[([.\w]+)(?::(\w+))?\]`)
	matches := bracketRe.FindAllStringSubmatch(relPath, -1)

	for _, match := range matches {
		param := ParamDef{
			Segment: match[0],
		}

		name := match[1]
		if strings.HasPrefix(name, "...") {
			param.Name = name[3:]
			param.Type = "[]string" // Catch-all is always string slice
		} else {
			param.Name = name
			if match[2] != "" {
				// Explicit type annotation [param:type]
				param.Type = match[2]
			} else {
				// Infer type from naming conventions
				param.Type = inferParamTypeFromName(name)
			}
		}

		params = append(params, param)
	}

	// Match underscore notation for catch-all: _param___
	catchAllRe := regexp.MustCompile(`_(\w+)___`)
	catchAllMatches := catchAllRe.FindAllStringSubmatch(relPath, -1)
	for _, match := range catchAllMatches {
		params = append(params, ParamDef{
			Segment: match[0],
			Name:    match[1],
			Type:    "[]string", // Catch-all is always string slice
		})
	}

	// Match underscore notation for regular params: _param_
	// Skip if already matched as catch-all (followed by another underscore)
	paramRe := regexp.MustCompile(`_(\w+)_`)
	paramIndexes := paramRe.FindAllStringSubmatchIndex(relPath, -1)
	for _, indexes := range paramIndexes {
		matchEnd := indexes[1]
		// Skip if followed by another underscore (part of catch-all _param___)
		if matchEnd < len(relPath) && relPath[matchEnd] == '_' {
			continue
		}
		segment := relPath[indexes[0]:indexes[1]]
		name := relPath[indexes[2]:indexes[3]]
		params = append(params, ParamDef{
			Segment: segment,
			Name:    name,
			Type:    inferParamTypeFromName(name),
		})
	}

	return params
}

// inferParamTypeFromName infers the Go type from a parameter name.
// This follows common naming conventions to reduce boilerplate.
func inferParamTypeFromName(name string) string {
	lower := strings.ToLower(name)

	// UUID patterns → string (check BEFORE ID patterns since "uuid" ends with "id")
	if lower == "uuid" || strings.HasSuffix(lower, "uuid") || strings.HasSuffix(lower, "_uuid") {
		return "string"
	}

	// Common ID patterns → int
	// Matches: id, userId, user_id, postId, post_id, etc.
	if lower == "id" {
		return "int"
	}
	if strings.HasSuffix(lower, "id") || strings.HasSuffix(lower, "_id") {
		return "int"
	}

	// Common string patterns
	switch lower {
	case "slug", "name", "title", "path", "key", "token", "code":
		return "string"
	}

	// Numeric patterns
	switch lower {
	case "page", "limit", "offset", "count", "index", "num", "number":
		return "int"
	case "year", "month", "day":
		return "int"
	}

	// Default to string for unknown parameters
	return "string"
}

// isAPIRoute checks if a relative path is an API route.
func (s *Scanner) isAPIRoute(relPath string) bool {
	relPath = strings.ReplaceAll(relPath, "\\", "/")
	return strings.HasPrefix(relPath, "api/") || relPath == "api"
}
