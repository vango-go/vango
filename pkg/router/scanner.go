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
	case "layout.go":
		route.HasLayout = true
	case "_middleware.go":
		route.HasMiddleware = true
	case "middleware.go":
		route.HasMiddleware = true
	case "_error.go", "_404.go":
		// Error handlers - still scan for functions
	}

	// Check for API route
	route.IsAPI = s.isAPIRoute(relPath)

	setAPIHandler := func(method, funcName string) {
		if route.APIHandlers == nil {
			route.APIHandlers = make(map[string]string)
		}
		if _, exists := route.APIHandlers[method]; exists {
			// Preserve the first-seen handler name; validation will catch ambiguity
			return
		}
		route.APIHandlers[method] = funcName
	}

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
				route.MiddlewareIsFunc = true
				continue
			}

			// Check for Meta function
			if name == "Meta" {
				route.HasMeta = true
				continue
			}

			// Check for page handler functions (IndexPage, AboutPage, ShowPage, etc.)
			// Per spec: function names ending in "Page" are page handlers
			if strings.HasSuffix(name, "Page") {
				route.HasPage = true
				route.HandlerName = name
				continue
			}

			if route.IsAPI {
				// Check for HTTP method handlers (API routes)
				switch name {
				case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
					route.Methods = append(route.Methods, name)
					setAPIHandler(name, name)
					continue
				}

				// Also detect {Resource}GET, {Resource}POST patterns for API routes
				// e.g., UsersGET, HealthGET, UsersPOST
				for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"} {
					if strings.HasSuffix(name, method) {
						route.Methods = append(route.Methods, method)
						setAPIHandler(method, name)
						break
					}
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
						route.MiddlewareIsVar = true
					}
				}
			}
		}
	}

	return route, nil
}

// filePathToURLPath converts a file path to a URL path.
//
// Dynamic route conventions:
//
//  1. Bracket notation: [id].go → :id (Next.js/Remix style)
//
//  2. "Go-friendly" underscore notation (no leading underscore in filenames):
//     - id_.go → :id
//     - slug___.go → *slug (catch-all)
//
// Legacy underscore directory segments are also supported:
//   - users/_id_/posts/index.go → /users/:id/posts
//
// NOTE: Go ignores files whose basename starts with '_' or '.', so reserved
// files must be named layout.go/middleware.go (not _layout.go/_middleware.go).
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

	// Reserved directory-scoped files (layouts/middleware) map to their directory path.
	// Example: routes/projects/layout.go registers at "/projects" (not "/projects/layout").
	baseName := filepath.Base(path)
	if baseName == "layout" || baseName == "middleware" ||
		(strings.HasPrefix(baseName, "_") && !strings.HasSuffix(baseName, "_")) {
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
//
// Supported conventions:
//  1) Bracket: [id] → :id, [id:int] → :id, [...slug] → *slug
//  2) Suffix underscore ("Go-friendly"): id_ → :id, slug___ → *slug
//  3) Legacy underscore segments (directory names only): _id_ → :id, _slug___ → *slug
func (s *Scanner) convertParams(path string) string {
	bracketRe := regexp.MustCompile(`\[([.\w]+)(?::(\w+))?\]`)
	legacyCatchAllRe := regexp.MustCompile(`^_(\w+)___$`)
	legacyParamRe := regexp.MustCompile(`^_(\w+)_$`)
	suffixCatchAllRe := regexp.MustCompile(`^(\w+)___$`)
	suffixParamRe := regexp.MustCompile(`^(\w+)_$`)

	segments := strings.Split(path, "/")
	for i, seg := range segments {
		// Bracket notation within a segment
		seg = bracketRe.ReplaceAllStringFunc(seg, func(match string) string {
			inner := match[1 : len(match)-1] // Remove brackets
			if strings.HasPrefix(inner, "...") {
				return "*" + inner[3:]
			}
			if idx := strings.Index(inner, ":"); idx != -1 {
				return ":" + inner[:idx]
			}
			return ":" + inner
		})

		// Legacy underscore segments: _id_ / _slug___
		if m := legacyCatchAllRe.FindStringSubmatch(seg); len(m) == 2 {
			seg = "*" + m[1]
		} else if m := legacyParamRe.FindStringSubmatch(seg); len(m) == 2 {
			seg = ":" + m[1]
		} else if m := suffixCatchAllRe.FindStringSubmatch(seg); len(m) == 2 {
			// Suffix underscore segments: id_ / slug___
			seg = "*" + m[1]
		} else if m := suffixParamRe.FindStringSubmatch(seg); len(m) == 2 {
			seg = ":" + m[1]
		}

		segments[i] = seg
	}

	return strings.Join(segments, "/")
}

// extractParams extracts parameter definitions from a file path.
//
// Supported conventions:
//   - [id], [id:int], [...path]
//   - id_, slug___ (Go-friendly, filename-safe)
//   - _id_, _slug___ (legacy; best used for directory segments)
func (s *Scanner) extractParams(relPath string) []ParamDef {
	var params []ParamDef

	normalized := strings.ReplaceAll(relPath, "\\", "/")
	normalized = strings.TrimSuffix(normalized, ".go")
	segments := strings.Split(normalized, "/")

	bracketRe := regexp.MustCompile(`\[([.\w]+)(?::(\w+))?\]`)
	legacyCatchAllRe := regexp.MustCompile(`^_(\w+)___$`)
	legacyParamRe := regexp.MustCompile(`^_(\w+)_$`)
	suffixCatchAllRe := regexp.MustCompile(`^(\w+)___$`)
	suffixParamRe := regexp.MustCompile(`^(\w+)_$`)

	for _, seg := range segments {
		if seg == "" {
			continue
		}

		// Bracket notation: [param] / [param:type] / [...param]
		if match := bracketRe.FindStringSubmatch(seg); len(match) > 0 {
			param := ParamDef{Segment: match[0]}
			name := match[1]
			if strings.HasPrefix(name, "...") {
				param.Name = name[3:]
				param.Type = "[]string"
			} else {
				param.Name = name
				if match[2] != "" {
					param.Type = match[2]
				} else {
					param.Type = inferParamTypeFromName(name)
				}
			}
			params = append(params, param)
			continue
		}

		// Legacy underscore segments: _id_ / _slug___
		if match := legacyCatchAllRe.FindStringSubmatch(seg); len(match) == 2 {
			params = append(params, ParamDef{Segment: seg, Name: match[1], Type: "[]string"})
			continue
		}
		if match := legacyParamRe.FindStringSubmatch(seg); len(match) == 2 {
			params = append(params, ParamDef{Segment: seg, Name: match[1], Type: inferParamTypeFromName(match[1])})
			continue
		}

		// Suffix underscore segments: id_ / slug___
		if match := suffixCatchAllRe.FindStringSubmatch(seg); len(match) == 2 {
			params = append(params, ParamDef{Segment: seg, Name: match[1], Type: "[]string"})
			continue
		}
		if match := suffixParamRe.FindStringSubmatch(seg); len(match) == 2 {
			params = append(params, ParamDef{Segment: seg, Name: match[1], Type: inferParamTypeFromName(match[1])})
			continue
		}
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
