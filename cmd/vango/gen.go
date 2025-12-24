package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/spf13/cobra"
	"github.com/vango-dev/vango/v2/internal/config"
	"github.com/vango-dev/vango/v2/internal/errors"
	"github.com/vango-dev/vango/v2/pkg/router"
)

func genCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gen <type>",
		Short: "Generate code",
		Long: `Generate code for routes, components, or other Vango constructs.

Types:
  routes      Generate routes_gen.go from app/routes/ directory
  route       Generate a new route file
  api         Generate a new API route file
  component   Generate a new component
  store       Generate a new store file
  middleware  Generate a new middleware file
  openapi     Generate OpenAPI 3.0 specification from API routes

Examples:
  vango gen routes                    # Regenerate routes_gen.go
  vango gen route users/[id]          # Generate app/routes/users/[id].go
  vango gen api products              # Generate app/routes/api/products.go
  vango gen component Card            # Generate app/components/card.go
  vango gen component shared/Button   # Generate app/components/shared/button.go
  vango gen store cart                # Generate app/store/cart.go
  vango gen middleware rate-limit     # Generate app/middleware/rate_limit.go
  vango gen openapi                   # Generate openapi.json`,
	}

	cmd.AddCommand(
		genRoutesCmd(),
		genRouteCmd(),
		genAPICmd(),
		genComponentCmd(),
		genStoreCmd(),
		genMiddlewareCmd(),
		genOpenAPICmd(),
	)

	return cmd
}

// =============================================================================
// vango gen routes
// =============================================================================

func genRoutesCmd() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "routes",
		Short: "Generate routes_gen.go from route files",
		Long: `Scan the routes directory and generate the routes_gen.go file.

This command scans app/routes/ for Go files with Page, Layout, Middleware,
and HTTP method functions (GET, POST, etc.) and generates the route
registration glue code.

The output is deterministic - running it multiple times produces identical
output unless the routes change.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenRoutes(output)
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file (default: app/routes/routes_gen.go)")

	return cmd
}

func runGenRoutes(output string) error {
	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return err
	}

	routesDir := cfg.RoutesPath()
	if output == "" {
		output = filepath.Join(routesDir, "routes_gen.go")
	}

	info("Scanning %s...", routesDir)

	// Use the AST-based scanner
	scanner := router.NewScanner(routesDir)
	routes, err := scanner.Scan()
	if err != nil {
		return err
	}

	info("Found %d routes", len(routes))

	// Get module path from go.mod
	modulePath, err := getModulePath(cfg.Dir())
	if err != nil {
		warn("Could not determine module path: %v", err)
		modulePath = "your-module"
	}

	// Generate code using the router package generator
	gen := router.NewGenerator(routes, modulePath)
	code, err := gen.Generate()
	if err != nil {
		return err
	}

	// Write file
	if err := os.WriteFile(output, code, 0644); err != nil {
		return err
	}

	success("Generated %s", output)
	return nil
}

// =============================================================================
// vango gen route
// =============================================================================

func genRouteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "route <path>",
		Short: "Generate a new route file",
		Long: `Generate a new route file with Page function.

The path uses file-based routing conventions:
  index     → / (root route)
  about     → /about
  users/[id] → /users/:id

Parameters are inferred from the filename:
  [id]       → int parameter (common ID pattern)
  [slug]     → string parameter
  [uuid]     → UUID string parameter
  [...]      → catch-all string slice

Examples:
  vango gen route index              # app/routes/index.go
  vango gen route about              # app/routes/about.go
  vango gen route users/[id]         # app/routes/users/[id].go
  vango gen route projects/[slug]    # app/routes/projects/[slug].go
  vango gen route docs/[...path]     # app/routes/docs/[...path].go`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenRoute(args[0])
		},
	}

	return cmd
}

func runGenRoute(path string) error {
	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return err
	}

	// Normalize path
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, ".go")

	// Determine filename and directory
	routesDir := cfg.RoutesPath()
	filename := path + ".go"
	outputPath := filepath.Join(routesDir, filename)

	// Check if file exists
	if _, err := os.Stat(outputPath); err == nil {
		return errors.New("E146").
			WithDetail("File already exists: " + outputPath).
			WithSuggestion("Choose a different path or remove the existing file")
	}

	// Ensure directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Determine package name from directory
	packageName := determinePackageName(dir, routesDir)

	// Extract parameters from path
	params := extractParamsFromPath(path)

	// Determine page function name
	pageFuncName := determinePageFuncName(filepath.Base(path))

	// Get module path
	modulePath, err := getModulePath(cfg.Dir())
	if err != nil {
		modulePath = "your-module"
	}

	// Generate code
	code := generateRouteCode(packageName, pageFuncName, params, modulePath, path)

	// Write file
	if err := os.WriteFile(outputPath, []byte(code), 0644); err != nil {
		return err
	}

	success("Created %s", outputPath)

	// Remind about regenerating routes
	info("")
	info("Run 'vango gen routes' to update routes_gen.go")

	return nil
}

// =============================================================================
// vango gen api
// =============================================================================

func genAPICmd() *cobra.Command {
	var methods []string

	cmd := &cobra.Command{
		Use:   "api <path>",
		Short: "Generate a new API route file",
		Long: `Generate a new API route file with HTTP method handlers.

API routes return JSON and support typed request/response bodies.

Examples:
  vango gen api health                      # GET /api/health
  vango gen api users                       # GET, POST /api/users
  vango gen api users/[id] --methods=GET,PUT,DELETE`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenAPI(args[0], methods)
		},
	}

	cmd.Flags().StringSliceVarP(&methods, "methods", "m", []string{"GET"}, "HTTP methods to generate (GET,POST,PUT,DELETE)")

	return cmd
}

func runGenAPI(path string, methods []string) error {
	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return err
	}

	// Normalize path
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimPrefix(path, "api/")
	path = strings.TrimSuffix(path, ".go")

	// Determine filename and directory
	routesDir := cfg.RoutesPath()
	apiDir := filepath.Join(routesDir, "api")
	filename := path + ".go"
	outputPath := filepath.Join(apiDir, filename)

	// Check if file exists
	if _, err := os.Stat(outputPath); err == nil {
		return errors.New("E146").
			WithDetail("File already exists: " + outputPath).
			WithSuggestion("Choose a different path or remove the existing file")
	}

	// Ensure directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Determine package name
	packageName := determinePackageName(dir, apiDir)
	if packageName == "api" || packageName == "routes" {
		// In api/ directory, use "api" as package name
		packageName = "api"
	}

	// Extract parameters from path
	params := extractParamsFromPath(path)

	// Normalize methods
	normalizedMethods := make([]string, 0, len(methods))
	for _, m := range methods {
		normalizedMethods = append(normalizedMethods, strings.ToUpper(strings.TrimSpace(m)))
	}

	// Get resource name for response types
	resourceName := getResourceName(path)

	// Generate code
	code := generateAPICode(packageName, resourceName, normalizedMethods, params)

	// Write file
	if err := os.WriteFile(outputPath, []byte(code), 0644); err != nil {
		return err
	}

	success("Created %s", outputPath)

	// Remind about regenerating routes
	info("")
	info("Run 'vango gen routes' to update routes_gen.go")

	return nil
}

// =============================================================================
// vango gen component
// =============================================================================

func genComponentCmd() *cobra.Command {
	var outputDir string

	cmd := &cobra.Command{
		Use:   "component <name>",
		Short: "Generate a new component",
		Long: `Generate a new component file.

Components are placed in app/components by default.
Use a path prefix to organize into subdirectories.

Examples:
  vango gen component Card              # app/components/card.go
  vango gen component shared/Button     # app/components/shared/button.go
  vango gen component ui/Dialog         # app/components/ui/dialog.go`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenComponent(args[0], outputDir)
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output", "o", "", "Output directory (default: app/components)")

	return cmd
}

func runGenComponent(name, outputDir string) error {
	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return err
	}

	if outputDir == "" {
		outputDir = cfg.ComponentsPath()
	}

	// Parse path to get directory and component name
	var dir, componentName string
	if strings.Contains(name, "/") {
		dir = filepath.Join(outputDir, filepath.Dir(name))
		componentName = filepath.Base(name)
	} else {
		dir = outputDir
		componentName = name
	}

	// Ensure directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Generate filename (lowercase)
	filename := strings.ToLower(componentName) + ".go"
	outputPath := filepath.Join(dir, filename)

	// Check if file exists
	if _, err := os.Stat(outputPath); err == nil {
		return errors.New("E146").
			WithDetail("File already exists: " + outputPath).
			WithSuggestion("Choose a different name or remove the existing file")
	}

	// Determine package name from directory
	packageName := filepath.Base(dir)
	if !isValidIdentifier(packageName) {
		packageName = "components"
	}

	// Check for package collision
	checkPackageCollision(dir, packageName)

	// Generate component code
	code := generateComponentCode(packageName, toPascalCase(componentName))

	// Write file
	if err := os.WriteFile(outputPath, []byte(code), 0644); err != nil {
		return err
	}

	success("Created %s", outputPath)
	return nil
}

// =============================================================================
// vango gen store
// =============================================================================

func genStoreCmd() *cobra.Command {
	var signalType string

	cmd := &cobra.Command{
		Use:   "store <name>",
		Short: "Generate a new store file",
		Long: `Generate a new store file for shared state.

Stores use SharedSignal or GlobalSignal for cross-component state.

Types:
  shared   SharedSignal - scoped to user session (default)
  global   GlobalSignal - shared across all users

Examples:
  vango gen store cart                      # SharedSignal for cart
  vango gen store user --type=shared        # SharedSignal for user
  vango gen store notifications --type=global`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenStore(args[0], signalType)
		},
	}

	cmd.Flags().StringVarP(&signalType, "type", "t", "shared", "Signal type (shared, global)")

	return cmd
}

func runGenStore(name, signalType string) error {
	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return err
	}

	storeDir := cfg.StorePath()

	// Ensure directory exists
	if err := os.MkdirAll(storeDir, 0755); err != nil {
		return err
	}

	// Generate filename
	filename := strings.ToLower(name) + ".go"
	outputPath := filepath.Join(storeDir, filename)

	// Check if file exists
	if _, err := os.Stat(outputPath); err == nil {
		return errors.New("E146").
			WithDetail("File already exists: " + outputPath).
			WithSuggestion("Choose a different name or remove the existing file")
	}

	// Determine signal type
	isGlobal := strings.ToLower(signalType) == "global"

	// Generate store code
	code := generateStoreCode(toPascalCase(name), isGlobal)

	// Write file
	if err := os.WriteFile(outputPath, []byte(code), 0644); err != nil {
		return err
	}

	success("Created %s", outputPath)
	return nil
}

// =============================================================================
// vango gen middleware
// =============================================================================

func genMiddlewareCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "middleware <name>",
		Short: "Generate a new middleware file",
		Long: `Generate a new middleware file.

Middleware intercepts requests and can:
  - Check authentication/authorization
  - Add logging or metrics
  - Modify request context
  - Short-circuit the request chain

Examples:
  vango gen middleware auth
  vango gen middleware rate-limit
  vango gen middleware logging`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenMiddleware(args[0])
		},
	}

	return cmd
}

func runGenMiddleware(name string) error {
	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return err
	}

	middlewareDir := cfg.MiddlewarePath()

	// Ensure directory exists
	if err := os.MkdirAll(middlewareDir, 0755); err != nil {
		return err
	}

	// Normalize name (replace hyphens with underscores for Go)
	goName := strings.ReplaceAll(name, "-", "_")

	// Generate filename
	filename := strings.ToLower(goName) + ".go"
	outputPath := filepath.Join(middlewareDir, filename)

	// Check if file exists
	if _, err := os.Stat(outputPath); err == nil {
		return errors.New("E146").
			WithDetail("File already exists: " + outputPath).
			WithSuggestion("Choose a different name or remove the existing file")
	}

	// Generate middleware code
	code := generateMiddlewareCode(toPascalCase(goName))

	// Write file
	if err := os.WriteFile(outputPath, []byte(code), 0644); err != nil {
		return err
	}

	success("Created %s", outputPath)
	return nil
}

// =============================================================================
// vango gen openapi
// =============================================================================

func genOpenAPICmd() *cobra.Command {
	var (
		output      string
		title       string
		description string
		version     string
	)

	cmd := &cobra.Command{
		Use:   "openapi",
		Short: "Generate OpenAPI 3.0 specification",
		Long: `Generate an OpenAPI 3.0 specification from your API routes.

This scans app/routes/api/ for Go files with HTTP method handlers
(GET, POST, PUT, DELETE, etc.) and generates a complete OpenAPI spec
with paths, parameters, request bodies, and response schemas.

Type information is extracted from:
  - Function signatures (request/response types)
  - Struct definitions (JSON field names, validation tags)
  - Path parameters ([id], [slug], etc.)

Examples:
  vango gen openapi                          # Generate openapi.json
  vango gen openapi -o docs/api.json         # Custom output path
  vango gen openapi --title "My API"         # Custom API title
  vango gen openapi --version 2.0.0          # Custom version`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenOpenAPI(output, title, description, version)
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "openapi.json", "Output file path")
	cmd.Flags().StringVar(&title, "title", "", "API title (default: project name)")
	cmd.Flags().StringVar(&description, "description", "", "API description")
	cmd.Flags().StringVar(&version, "version", "1.0.0", "API version")

	return cmd
}

func runGenOpenAPI(output, title, description, version string) error {
	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return err
	}

	routesDir := cfg.RoutesPath()

	// Get project name for default title
	if title == "" {
		title = cfg.Name
		if title == "" {
			title = "API"
		}
	}

	// Get module path from go.mod
	modulePath, err := getModulePath(cfg.Dir())
	if err != nil {
		warn("Could not determine module path: %v", err)
		modulePath = "your-module"
	}

	info("Scanning %s/api/...", routesDir)

	// Generate OpenAPI spec
	gen := router.NewOpenAPIGenerator(routesDir, modulePath, router.OpenAPIInfo{
		Title:       title,
		Description: description,
		Version:     version,
	})

	spec, err := gen.Generate()
	if err != nil {
		return err
	}

	// Determine output path
	if !filepath.IsAbs(output) {
		output = filepath.Join(cfg.Dir(), output)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}

	// Write file
	if err := os.WriteFile(output, spec, 0644); err != nil {
		return err
	}

	success("Generated %s", output)
	return nil
}

// =============================================================================
// Helper Functions
// =============================================================================

func getModulePath(projectDir string) (string, error) {
	goModPath := filepath.Join(projectDir, "go.mod")
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimPrefix(line, "module "), nil
		}
	}

	return "", fmt.Errorf("module declaration not found in go.mod")
}

func determinePackageName(dir, baseDir string) string {
	// Use the directory name as package name
	name := filepath.Base(dir)
	if name == "." || name == "" || dir == baseDir {
		return "routes"
	}
	// Sanitize for Go identifier
	name = strings.ReplaceAll(name, "-", "_")
	if !isValidIdentifier(name) {
		return "routes"
	}
	return name
}

func determinePageFuncName(filename string) string {
	// Remove .go extension
	name := strings.TrimSuffix(filename, ".go")

	// Handle special cases
	if name == "index" || name == "" {
		return "IndexPage"
	}

	// Remove parameter brackets for function naming
	name = strings.ReplaceAll(name, "[", "")
	name = strings.ReplaceAll(name, "]", "")
	name = strings.ReplaceAll(name, "...", "")

	// Handle dynamic routes
	if strings.HasPrefix(name, ":") {
		return "ShowPage"
	}

	return toPascalCase(name) + "Page"
}

func extractParamsFromPath(path string) []paramInfo {
	var params []paramInfo

	parts := strings.Split(path, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]") {
			inner := part[1 : len(part)-1]

			// Handle catch-all
			if strings.HasPrefix(inner, "...") {
				params = append(params, paramInfo{
					Name:    inner[3:],
					Type:    "[]string",
					Segment: part,
				})
				continue
			}

			// Handle typed params [id:int]
			if idx := strings.Index(inner, ":"); idx != -1 {
				params = append(params, paramInfo{
					Name:    inner[:idx],
					Type:    inner[idx+1:],
					Segment: part,
				})
				continue
			}

			// Infer type from name
			params = append(params, paramInfo{
				Name:    inner,
				Type:    inferParamType(inner),
				Segment: part,
			})
		}
	}

	return params
}

type paramInfo struct {
	Name    string
	Type    string
	Segment string
}

func inferParamType(name string) string {
	lower := strings.ToLower(name)
	switch lower {
	case "id":
		return "int"
	case "uuid":
		return "string" // UUIDs stored as strings
	case "slug", "name", "path":
		return "string"
	default:
		// Default to string for unknown params
		return "string"
	}
}

func getResourceName(path string) string {
	// Get the last non-param segment
	parts := strings.Split(path, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if !strings.HasPrefix(parts[i], "[") {
			return toPascalCase(parts[i])
		}
	}
	return "Resource"
}

func toPascalCase(s string) string {
	if s == "" {
		return s
	}

	// Handle common abbreviations
	upper := strings.ToUpper(s)
	switch upper {
	case "ID", "URL", "API", "HTTP", "UUID":
		return upper
	}

	// Split on separators and capitalize each part
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})

	var result strings.Builder
	for _, part := range parts {
		if len(part) > 0 {
			runes := []rune(part)
			runes[0] = unicode.ToUpper(runes[0])
			result.WriteString(string(runes))
		}
	}

	return result.String()
}

func isValidIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 && !unicode.IsLetter(r) && r != '_' {
			return false
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}
	return true
}

func checkPackageCollision(dir, packageName string) {
	// Check if there are existing .go files with a different package
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
			// Read first line to check package
			data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				continue
			}
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "package ") {
					existingPkg := strings.TrimPrefix(line, "package ")
					existingPkg = strings.TrimSpace(existingPkg)
					if existingPkg != packageName {
						warn("Package collision: directory contains package '%s', new file uses '%s'", existingPkg, packageName)
					}
					return
				}
			}
		}
	}
}

// =============================================================================
// Code Generation Templates
// =============================================================================

func generateRouteCode(packageName, pageFuncName string, params []paramInfo, modulePath, path string) string {
	var code strings.Builder

	code.WriteString(fmt.Sprintf("package %s\n\n", packageName))
	code.WriteString("import (\n")
	code.WriteString("\t\"github.com/vango-dev/vango\"\n")
	code.WriteString("\t. \"github.com/vango-dev/vango/el\"\n")
	code.WriteString(")\n\n")

	// Generate Params struct if needed
	if len(params) > 0 {
		code.WriteString("// Params for this route.\n")
		code.WriteString("type Params struct {\n")
		for _, p := range params {
			fieldName := toPascalCase(p.Name)
			goType := paramTypeToGoType(p.Type)
			code.WriteString(fmt.Sprintf("\t%s %s `param:\"%s\"`\n", fieldName, goType, p.Name))
		}
		code.WriteString("}\n\n")
	}

	// Generate Page function
	code.WriteString(fmt.Sprintf("// %s handles /%s.\n", pageFuncName, path))
	if len(params) > 0 {
		code.WriteString(fmt.Sprintf("func %s(ctx vango.Ctx, p Params) vango.Component {\n", pageFuncName))
	} else {
		code.WriteString(fmt.Sprintf("func %s(ctx vango.Ctx) vango.Component {\n", pageFuncName))
	}
	code.WriteString("\treturn vango.Func(func() *vango.VNode {\n")
	code.WriteString("\t\treturn Div(\n")
	code.WriteString(fmt.Sprintf("\t\t\tH1(Text(\"%s\")),\n", pageFuncName))
	if len(params) > 0 {
		for _, p := range params {
			fieldName := toPascalCase(p.Name)
			code.WriteString(fmt.Sprintf("\t\t\tP(Textf(\"%s: %%v\", p.%s)),\n", p.Name, fieldName))
		}
	}
	code.WriteString("\t\t)\n")
	code.WriteString("\t})\n")
	code.WriteString("}\n")

	return code.String()
}

func generateAPICode(packageName, resourceName string, methods []string, params []paramInfo) string {
	var code strings.Builder

	code.WriteString(fmt.Sprintf("package %s\n\n", packageName))
	code.WriteString("import \"github.com/vango-dev/vango\"\n\n")

	// Generate response types
	code.WriteString(fmt.Sprintf("// %sResponse is the JSON response type.\n", resourceName))
	code.WriteString(fmt.Sprintf("type %sResponse struct {\n", resourceName))
	code.WriteString("\tID     string `json:\"id\"`\n")
	code.WriteString("\tStatus string `json:\"status\"`\n")
	code.WriteString("}\n\n")

	// Generate input types for POST/PUT
	for _, m := range methods {
		if m == "POST" || m == "PUT" || m == "PATCH" {
			code.WriteString(fmt.Sprintf("// %sInput is the request body for %s.\n", resourceName, m))
			code.WriteString(fmt.Sprintf("type %sInput struct {\n", resourceName))
			code.WriteString("\tName string `json:\"name\" validate:\"required\"`\n")
			code.WriteString("}\n\n")
			break
		}
	}

	// Generate Params struct if needed
	if len(params) > 0 {
		code.WriteString("// Params for this route.\n")
		code.WriteString("type Params struct {\n")
		for _, p := range params {
			fieldName := toPascalCase(p.Name)
			goType := paramTypeToGoType(p.Type)
			code.WriteString(fmt.Sprintf("\t%s %s `param:\"%s\"`\n", fieldName, goType, p.Name))
		}
		code.WriteString("}\n\n")
	}

	// Generate handler functions
	for _, m := range methods {
		funcName := resourceName + m
		code.WriteString(fmt.Sprintf("// %s handles %s /api/%s.\n", funcName, m, strings.ToLower(resourceName)))

		switch m {
		case "GET":
			if len(params) > 0 {
				code.WriteString(fmt.Sprintf("func %s(ctx vango.Ctx, p Params) (*%sResponse, error) {\n", funcName, resourceName))
				code.WriteString(fmt.Sprintf("\treturn &%sResponse{\n", resourceName))
				code.WriteString("\t\tID:     \"1\",\n")
				code.WriteString("\t\tStatus: \"ok\",\n")
				code.WriteString("\t}, nil\n")
			} else {
				code.WriteString(fmt.Sprintf("func %s(ctx vango.Ctx) ([]*%sResponse, error) {\n", funcName, resourceName))
				code.WriteString(fmt.Sprintf("\treturn []*%sResponse{}, nil\n", resourceName))
			}
		case "POST":
			code.WriteString(fmt.Sprintf("func %s(ctx vango.Ctx, input %sInput) (*%sResponse, error) {\n", funcName, resourceName, resourceName))
			code.WriteString(fmt.Sprintf("\treturn &%sResponse{\n", resourceName))
			code.WriteString("\t\tID:     \"new-id\",\n")
			code.WriteString("\t\tStatus: \"created\",\n")
			code.WriteString("\t}, nil\n")
		case "PUT", "PATCH":
			if len(params) > 0 {
				code.WriteString(fmt.Sprintf("func %s(ctx vango.Ctx, p Params, input %sInput) (*%sResponse, error) {\n", funcName, resourceName, resourceName))
			} else {
				code.WriteString(fmt.Sprintf("func %s(ctx vango.Ctx, input %sInput) (*%sResponse, error) {\n", funcName, resourceName, resourceName))
			}
			code.WriteString(fmt.Sprintf("\treturn &%sResponse{\n", resourceName))
			code.WriteString("\t\tStatus: \"updated\",\n")
			code.WriteString("\t}, nil\n")
		case "DELETE":
			if len(params) > 0 {
				code.WriteString(fmt.Sprintf("func %s(ctx vango.Ctx, p Params) error {\n", funcName))
			} else {
				code.WriteString(fmt.Sprintf("func %s(ctx vango.Ctx) error {\n", funcName))
			}
			code.WriteString("\treturn nil\n")
		default:
			code.WriteString(fmt.Sprintf("func %s(ctx vango.Ctx) error {\n", funcName))
			code.WriteString("\treturn nil\n")
		}
		code.WriteString("}\n\n")
	}

	return code.String()
}

func generateComponentCode(packageName, componentName string) string {
	return fmt.Sprintf(`package %s

import (
	"github.com/vango-dev/vango"
	. "github.com/vango-dev/vango/el"
)

// %s is a component.
func %s() vango.Component {
	return vango.Func(func() *vango.VNode {
		return Div(Class("%s"),
			// Add your component content here
		)
	})
}
`, packageName, componentName, componentName, strings.ToLower(componentName))
}

func generateStoreCode(name string, isGlobal bool) string {
	signalType := "SharedSignal"
	signalFunc := "vango.SharedSignal"
	comment := "across the user's session"

	if isGlobal {
		signalType = "GlobalSignal"
		signalFunc = "vango.GlobalSignal"
		comment = "across all users"
	}

	return fmt.Sprintf(`package store

import "github.com/vango-dev/vango"

// %sState holds the state for %s.
// This state is shared %s.
type %sState struct {
	// Add your state fields here
	Items []string
	Count int
}

// %s is a %s for %s.
var %s = %s(%sState{
	Items: []string{},
	Count: 0,
})

// Example usage in a component:
//
//   state := store.%s.Get()
//   items := state.Items
//
//   store.%s.Update(func(s store.%sState) store.%sState {
//       s.Count++
//       return s
//   })
`, name, name, comment, name, name, signalType, name, name, signalFunc, name, name, name, name, name)
}

func generateMiddlewareCode(name string) string {
	return fmt.Sprintf(`package middleware

import (
	"github.com/vango-dev/vango"
	"github.com/vango-dev/vango/router"
)

// %s is a middleware that...
// Add your middleware description here.
func %s(next router.Handler) router.Handler {
	return func(ctx vango.Ctx) error {
		// Before handler execution

		// Call the next handler
		err := next(ctx)

		// After handler execution

		return err
	}
}

// Example: %s with options
func %sWithOptions(/* options */) router.Middleware {
	return func(next router.Handler) router.Handler {
		return func(ctx vango.Ctx) error {
			// Use options here
			return next(ctx)
		}
	}
}
`, name, name, name, name)
}

func paramTypeToGoType(paramType string) string {
	switch paramType {
	case "int":
		return "int"
	case "int64":
		return "int64"
	case "int32":
		return "int32"
	case "uint":
		return "uint"
	case "uint64":
		return "uint64"
	case "uuid":
		return "string"
	case "[]string":
		return "[]string"
	case "string", "":
		return "string"
	default:
		return "string"
	}
}
