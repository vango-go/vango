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

// Scanner scans a directory for route files.
type Scanner struct {
	rootDir string
}

// NewScanner creates a new route scanner.
func NewScanner(rootDir string) *Scanner {
	return &Scanner{rootDir: rootDir}
}

// Scan reads all route files and returns route definitions.
func (s *Scanner) Scan() ([]ScannedRoute, error) {
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
	route.IsCatchAll = strings.Contains(relPath, "[...")

	// Check for special files
	baseName := filepath.Base(path)
	switch baseName {
	case "_layout.go":
		route.HasLayout = true
	case "_error.go", "_404.go":
		// Error handlers - still scan for functions
	}

	// Check for API route
	route.IsAPI = s.isAPIRoute(relPath)

	// Scan for exported functions
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil || !fn.Name.IsExported() {
			continue
		}

		switch fn.Name.Name {
		case "Page":
			route.HasPage = true
		case "Layout":
			route.HasLayout = true
		case "Meta":
			route.HasMeta = true
		case "Middleware":
			route.HasMiddleware = true
		case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
			route.Methods = append(route.Methods, fn.Name.Name)
		}
	}

	return route, nil
}

// filePathToURLPath converts a file path to a URL path.
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
	if strings.HasPrefix(baseName, "_") {
		// Get the directory path for layouts
		dir := filepath.Dir(path)
		if dir == "." {
			path = ""
		} else {
			path = strings.ReplaceAll(dir, "\\", "/")
		}
	}

	// Convert [param] to :param
	path = s.convertParams(path)

	// Add leading slash
	if path == "" {
		return "/"
	}
	return "/" + path
}

// convertParams converts bracket notation to router notation.
func (s *Scanner) convertParams(path string) string {
	// [id] → :id
	// [id:int] → :id (type stored separately)
	// [...slug] → *slug

	// Match [param] or [param:type] or [...param]
	re := regexp.MustCompile(`\[([.\w]+)(?::(\w+))?\]`)
	result := re.ReplaceAllStringFunc(path, func(match string) string {
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

	return result
}

// extractParams extracts parameter definitions from a file path.
func (s *Scanner) extractParams(relPath string) []ParamDef {
	var params []ParamDef

	// Match [param] or [param:type] or [...param]
	re := regexp.MustCompile(`\[([.\w]+)(?::(\w+))?\]`)
	matches := re.FindAllStringSubmatch(relPath, -1)

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
				param.Type = match[2]
			} else {
				param.Type = "string" // Default to string
			}
		}

		params = append(params, param)
	}

	return params
}

// isAPIRoute checks if a relative path is an API route.
func (s *Scanner) isAPIRoute(relPath string) bool {
	relPath = strings.ReplaceAll(relPath, "\\", "/")
	return strings.HasPrefix(relPath, "api/") || relPath == "api"
}
