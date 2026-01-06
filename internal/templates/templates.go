package templates

import (
	"bytes"
	"os"
	"path/filepath"
	"text/template"

	"github.com/vango-go/vango/internal/errors"
)

// Config contains template configuration.
type Config struct {
	// ProjectName is the name of the project.
	ProjectName string

	// ModulePath is the Go module path.
	ModulePath string

	// Description is a short project description.
	Description string

	// HasTailwind enables Tailwind CSS.
	HasTailwind bool

	// HasDatabase enables database setup.
	HasDatabase bool

	// DatabaseType is the database type (sqlite, postgres).
	DatabaseType string

	// HasAuth enables authentication scaffolding.
	HasAuth bool
}

// Template represents a project template.
type Template struct {
	// Name is the template name.
	Name string

	// Description describes the template.
	Description string

	// Files is a map of relative paths to file contents.
	Files map[string]string
}

// Available templates.
var templates = map[string]*Template{
	"minimal":  minimalTemplate(),
	"standard": standardTemplate(),
	"full":     fullTemplate(),
	"api":      apiTemplate(),
}

// Get returns a template by name.
func Get(name string) (*Template, error) {
	tmpl, ok := templates[name]
	if !ok {
		return nil, errors.New("E145").
			WithDetail("Template '" + name + "' not found").
			WithSuggestion("Available templates: minimal, standard, full, api")
	}
	return tmpl, nil
}

// List returns all available template names.
func List() []string {
	names := make([]string, 0, len(templates))
	for name := range templates {
		names = append(names, name)
	}
	return names
}

// Create generates a project from the template.
func (t *Template) Create(dir string, cfg Config) error {
	for relPath, content := range t.Files {
		// Execute template
		tmpl, err := template.New(relPath).Parse(content)
		if err != nil {
			return errors.Newf(errors.CategoryCLI, "invalid template %s: %v", relPath, err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, cfg); err != nil {
			return errors.Newf(errors.CategoryCLI, "template execute error %s: %v", relPath, err)
		}

		// Write file
		fullPath := filepath.Join(dir, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return err
		}

		if err := os.WriteFile(fullPath, buf.Bytes(), 0644); err != nil {
			return err
		}
	}

	return nil
}

// minimalTemplate returns the minimal template.
// Just the essentials: home page and health API.
func minimalTemplate() *Template {
	return &Template{
		Name:        "minimal",
		Description: "Just the essentials for a Vango app",
		Files: map[string]string{
			"main.go":                      mainGoMinimal,
			"go.mod":                       goModTemplate,
			"vango.json":                   vangoJSONMinimal,
			"app/routes/index.go":          indexGoTemplate,
			"app/routes/api/health.go":     healthGoTemplate,
			"public/favicon.ico":           "", // Empty placeholder
			"public/styles.css":            stylesCSS,
			".gitignore":                   gitignoreTemplate,
			"README.md":                    readmeTemplate,
			"app/store/.gitkeep":           "",
			"app/components/ui/.gitkeep":   "",
			"app/components/shared/.gitkeep": "",
			"db/.gitkeep":                  "",
		},
	}
}

// standardTemplate returns the standard template.
// This is the default: home, about, health API, navbar, footer.
func standardTemplate() *Template {
	return &Template{
		Name:        "standard",
		Description: "Standard starter with example pages and components",
		Files: map[string]string{
			"main.go":                          mainGoStandard,
			"go.mod":                           goModTemplate,
			"vango.json":                       vangoJSONStandard,
			"app/routes/routes_gen.go":         routesGenGoTemplate,
			"app/routes/_layout.go":            layoutGoTemplate,
			"app/routes/index.go":              indexGoTemplate,
			"app/routes/about.go":              aboutGoTemplate,
			"app/routes/api/health.go":         healthGoTemplate,
			"app/components/shared/navbar.go":  navbarGoTemplate,
			"app/components/shared/footer.go":  footerGoTemplate,
			"app/middleware/auth.go":           authMiddlewareTemplate,
			"app/store/.gitkeep":               "",
			"app/components/ui/.gitkeep":       "",
			"db/.gitkeep":                      "",
			"public/favicon.ico":               "", // Empty placeholder
			"public/styles.css":                stylesCSS,
			".gitignore":                       gitignoreTemplate,
			"README.md":                        readmeTemplate,
		},
	}
}

// fullTemplate returns the full template with all features.
func fullTemplate() *Template {
	files := standardTemplate().Files
	// Add Tailwind config
	files["tailwind.config.js"] = tailwindConfigTemplate
	files["app/styles/input.css"] = tailwindInputCSS
	files["package.json"] = packageJSONTemplate
	return &Template{
		Name:        "full",
		Description: "Complete starter with Tailwind CSS and all features",
		Files:       files,
	}
}

// apiTemplate returns the API-only template.
func apiTemplate() *Template {
	return &Template{
		Name:        "api",
		Description: "API-only project without UI",
		Files: map[string]string{
			"main.go":                  mainGoAPI,
			"go.mod":                   goModTemplate,
			"vango.json":               vangoJSONAPI,
			"app/routes/api/health.go": healthGoTemplate,
			".gitignore":               gitignoreTemplate,
			"README.md":                readmeAPITemplate,
		},
	}
}

// =============================================================================
// Template Content Strings
// =============================================================================

const goModTemplate = `module {{.ModulePath}}

go 1.22

require github.com/vango-go/vango v0.1.0
`

const mainGoMinimal = `package main

import (
	"log"
	"net/http"

	"github.com/vango-go/vango/pkg/vango"
	"{{.ModulePath}}/app/routes"
)

func main() {
	app := vango.New(vango.Config{
		Dev: true, // Disable in production
	})

	// Standard middleware
	app.Use(vango.Logger())
	app.Use(vango.Recover())

	// Register routes from generated glue
	routes.Register(app.Router())

	// Serve static files from public/ at site root /
	app.Static("/", "public")

	log.Println("🚀 Vango server starting on http://localhost:3000")
	if err := http.ListenAndServe(":3000", app.Handler()); err != nil {
		log.Fatal(err)
	}
}
`

const mainGoStandard = `package main

import (
	"log"
	"net/http"
	"time"

	"github.com/vango-go/vango/pkg/vango"
	"{{.ModulePath}}/app/routes"
)

func main() {
	app := vango.New(vango.Config{
		Dev: true, // Disable in production

		Session: vango.SessionConfig{
			ResumeWindow: 30 * time.Second,
			// --- Production Configuration (uncomment when ready) ---
			// Store: vango.RedisStore(redisClient),
			// MaxDetachedSessions: 10000,
			// MaxSessionsPerIP:    100,
		},

		// --- Protocol Limits (production hardening) ---
		// MaxAllocation:      4 * 1024 * 1024,  // 4MB max message
		// MaxCollectionCount: 100_000,
		// MaxVNodeDepth:      256,
	})

	// --- Observability (uncomment for production) ---
	// app.Use(middleware.OpenTelemetry())
	// app.Use(middleware.Prometheus())

	// Standard middleware
	app.Use(vango.Logger())
	app.Use(vango.Recover())

	// Register routes from generated glue
	routes.Register(app.Router())

	// Serve static files from public/ at site root /
	app.Static("/", "public")

	log.Println("🚀 Vango server starting on http://localhost:3000")
	if err := http.ListenAndServe(":3000", app.Handler()); err != nil {
		log.Fatal(err)
	}
}
`

const mainGoAPI = `package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/health", handleHealth)

	log.Printf("API server running at http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

// handleHealth handles GET /api/health
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"version": "1.0.0",
	})
}
`

const vangoJSONMinimal = `{
  "name": "{{.ProjectName}}",
  "version": "0.1.0",
  "port": 3000,

  "paths": {
    "routes": "app/routes",
    "components": "app/components",
    "ui": "app/components/ui",
    "store": "app/store",
    "middleware": "app/middleware"
  },

  "static": {
    "dir": "public",
    "prefix": "/"
  },

  "dev": {
    "watch": ["app", "db", "public"],
    "hotReload": true
  }
}
`

const vangoJSONStandard = `{
  "name": "{{.ProjectName}}",
  "version": "0.1.0",
  "port": 3000,

  "paths": {
    "routes": "app/routes",
    "components": "app/components",
    "ui": "app/components/ui",
    "store": "app/store",
    "middleware": "app/middleware"
  },

  "static": {
    "dir": "public",
    "prefix": "/"
  },

  "dev": {
    "watch": ["app", "db", "public"],
    "hotReload": true
  },

  "tailwind": {
    "enabled": {{if .HasTailwind}}true{{else}}false{{end}},
    "config": "tailwind.config.js",
    "input": "public/styles.css"
  }
}
`

const vangoJSONAPI = `{
  "name": "{{.ProjectName}}",
  "version": "0.1.0",
  "port": 3000,

  "dev": {
    "watch": ["app"]
  },

  "build": {
    "output": "./dist",
    "minifyAssets": true,
    "stripSymbols": true
  }
}
`

const routesGenGoTemplate = `// Code generated by vango. DO NOT EDIT.

package routes

import (
	"github.com/vango-go/vango/pkg/router"

	"{{.ModulePath}}/app/routes/api"
)

// Register adds all routes to the router.
// Generated by ` + "`vango dev`" + ` or ` + "`vango gen routes`" + `.
func Register(r *router.Router) {
	// Page routes (package routes)
	r.Page("/", IndexPage, Layout)
	r.Page("/about", AboutPage, Layout)

	// API routes (package api)
	r.API("GET", "/api/health", api.HealthGET)
}
`

const layoutGoTemplate = `package routes

import (
	"github.com/vango-go/vango/pkg/router"
	"github.com/vango-go/vango/pkg/server"
	"github.com/vango-go/vango/pkg/vdom"
	. "github.com/vango-go/vango/el"

	"{{.ModulePath}}/app/components/shared"
)

// Layout wraps all routes in this directory and subdirectories.
func Layout(ctx server.Ctx, children router.Slot) *vdom.VNode {
	return Html(Lang("en"),
		Head(
			Meta(Charset("utf-8")),
			Meta(Name("viewport"), Content("width=device-width, initial-scale=1")),
			Title(Text("{{.ProjectName}}")),
			LinkEl(Rel("stylesheet"), Href("/styles.css")),
			LinkEl(Rel("icon"), Href("/favicon.ico")),
		),
		Body(
			shared.Navbar(),
			Main(Class("container"), children),
			shared.Footer(),
			VangoScripts(), // Thin client (~12KB)
		),
	)
}
`

const indexGoTemplate = `package routes

import (
	"github.com/vango-go/vango/pkg/server"
	"github.com/vango-go/vango/pkg/vango"
	"github.com/vango-go/vango/pkg/vdom"
	. "github.com/vango-go/vango/el"
)

// IndexPage is the home page component.
func IndexPage(ctx server.Ctx) vdom.Component {
	return vango.Func(func() *vdom.VNode {
		count := vango.NewSignal(0)

		return Div(Class("hero"),
			H1(Text("Welcome to {{.ProjectName}}")),
			P(Text("Server-driven UI for Go")),

			Div(Class("counter"),
				Button(OnClick(count.Dec), Text("-")),
				Span(Textf(" %d ", count.Get())),
				Button(OnClick(count.Inc), Text("+")),
			),
		)
	})
}
`

const aboutGoTemplate = `package routes

import (
	"github.com/vango-go/vango/pkg/server"
	"github.com/vango-go/vango/pkg/vdom"
	. "github.com/vango-go/vango/el"
)

// AboutPage is the about page (stateless).
func AboutPage(ctx server.Ctx) *vdom.VNode {
	return Div(Class("about"),
		H1(Text("About")),
		P(Text("Built with Vango — the Go framework for modern web apps.")),
		Ul(
			Li(Text("Server-driven architecture")),
			Li(Text("~12KB client runtime")),
			Li(Text("Direct database access")),
			Li(Text("Type-safe components")),
		),
	)
}
`

const healthGoTemplate = `package api

import "github.com/vango-go/vango/pkg/server"

// HealthResponse is the JSON response for health checks.
type HealthResponse struct {
	Status  string ` + "`json:\"status\"`" + `
	Version string ` + "`json:\"version\"`" + `
}

// HealthGET handles GET /api/health.
func HealthGET(ctx server.Ctx) (*HealthResponse, error) {
	return &HealthResponse{
		Status:  "ok",
		Version: "1.0.0",
	}, nil
}
`

const navbarGoTemplate = `package shared

import (
	"github.com/vango-go/vango/pkg/vdom"
	. "github.com/vango-go/vango/el"
)

// Navbar renders the site navigation.
// Uses Link() for SPA navigation with data-vango-link attribute.
func Navbar() *vdom.VNode {
	return Nav(Class("navbar"),
		Link("/", Class("logo"), Text("{{.ProjectName}}")),
		Div(Class("nav-links"),
			Link("/", Text("Home")),
			Link("/about", Text("About")),
		),
	)
}
`

const footerGoTemplate = `package shared

import (
	"github.com/vango-go/vango/pkg/vdom"
	. "github.com/vango-go/vango/el"
)

// Footer renders the site footer.
func Footer() *vdom.VNode {
	return Footer_(Class("footer"),
		P(Text("Built with Vango")),
	)
}
`

const authMiddlewareTemplate = `package middleware

import (
	"github.com/vango-go/vango/pkg/router"
	"github.com/vango-go/vango/pkg/server"
	"github.com/vango-go/vango/pkg/vango"
)

// User interface for type assertion (implement in your app).
type User interface {
	GetRole() string
}

// RequireAuth redirects unauthenticated users to login.
var RequireAuth = router.MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
	if ctx.User() == nil {
		ctx.Redirect("/login", 302)
		return nil
	}
	return next()
})

// RequireRole checks if user has a specific role.
func RequireRole(role string) router.Middleware {
	return router.MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
		user := ctx.User()
		if user == nil {
			ctx.Redirect("/login", 302)
			return nil
		}
		if u, ok := user.(User); ok && u.GetRole() != role {
			return vango.Forbidden("insufficient permissions")
		}
		return next()
	})
}
`

const stylesCSS = `/* Base styles - replace with Tailwind via ` + "`vango create --with-tailwind`" + ` */
:root {
  --background: #ffffff;
  --foreground: #171717;
  --primary: #2563eb;
  --muted: #f5f5f5;
  --border: #e5e5e5;
}

* {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

body {
  font-family: system-ui, -apple-system, sans-serif;
  background: var(--background);
  color: var(--foreground);
  line-height: 1.6;
}

.container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 2rem;
}

/* Navbar */
.navbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 2rem;
  border-bottom: 1px solid var(--border);
}

.navbar .logo {
  font-weight: 700;
  font-size: 1.25rem;
  text-decoration: none;
  color: inherit;
}

.nav-links {
  display: flex;
  gap: 1.5rem;
}

.nav-links a {
  text-decoration: none;
  color: inherit;
}

.nav-links a:hover {
  color: var(--primary);
}

/* Hero */
.hero {
  text-align: center;
  padding: 4rem 0;
}

.hero h1 {
  font-size: 3rem;
  margin-bottom: 0.5rem;
}

.hero p {
  color: #666;
  margin-bottom: 2rem;
}

/* Counter */
.counter {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
}

.counter button {
  width: 2.5rem;
  height: 2.5rem;
  font-size: 1.25rem;
  border: 1px solid var(--border);
  border-radius: 0.375rem;
  background: var(--background);
  cursor: pointer;
}

.counter button:hover {
  background: var(--muted);
}

.counter span {
  min-width: 3rem;
  text-align: center;
  font-size: 1.5rem;
  font-weight: 600;
}

/* Footer */
.footer {
  text-align: center;
  padding: 2rem;
  color: #666;
  font-size: 0.875rem;
}

/* About */
.about h1 {
  margin-bottom: 1rem;
}

.about ul {
  margin-top: 1rem;
  margin-left: 1.5rem;
}

.about li {
  margin-bottom: 0.5rem;
}

/* Admin Layout (for --with-auth scaffold) */
.admin-layout {
  display: flex;
  min-height: calc(100vh - 120px);
}

.admin-sidebar {
  width: 240px;
  background: var(--muted);
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.admin-sidebar a {
  padding: 0.5rem 1rem;
  text-decoration: none;
  color: inherit;
  border-radius: 0.375rem;
}

.admin-sidebar a:hover {
  background: var(--border);
}

.admin-content {
  flex: 1;
  padding: 1rem 2rem;
}

/* Connection state indicators (Phase 12 reconnection UX) */
.vango-reconnecting::after {
  content: "Reconnecting...";
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  padding: 0.5rem;
  background: #fbbf24;
  text-align: center;
  font-size: 0.875rem;
  z-index: 9999;
}

.vango-offline::after {
  content: "Connection lost";
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  padding: 0.5rem;
  background: #ef4444;
  color: white;
  text-align: center;
  font-size: 0.875rem;
  z-index: 9999;
}
`

const gitignoreTemplate = `# Binaries
/{{.ProjectName}}
/dist/
*.exe
*.dll
*.so
*.dylib

# Dependencies
/vendor/

# IDE
.idea/
.vscode/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db

# Test
coverage.out
*.cover

# Vango
.vango/

# Node
node_modules/
`

const readmeTemplate = `# {{.ProjectName}}

{{.Description}}

A web application built with [Vango](https://vango.dev).

## Quick Start

` + "```bash" + `
# Development server with hot reload
vango dev

# Open http://localhost:3000
` + "```" + `

## Commands

` + "```bash" + `
vango dev                    # Start dev server
vango build                  # Production build
vango test                   # Run tests

vango gen route users/[id]   # Generate route
vango gen component Card     # Generate component
vango gen api products       # Generate API route

vango add button             # Add VangoUI component
` + "```" + `

## Project Structure

` + "```" + `
app/
├── routes/         # File-based routing (pages, layouts, middleware)
├── components/     # Shared UI components
├── store/          # Shared state (SharedSignal, GlobalSignal)
└── middleware/     # Middleware definitions
db/                 # Database layer
public/             # Static assets (served at /)
` + "```" + `

## Production Checklist

Before deploying to production:

1. Set ` + "`Dev: false`" + ` in ` + "`main.go`" + `
2. Configure ` + "`SessionStore`" + ` (Redis recommended)
3. Set session limits (` + "`MaxDetachedSessions`" + `, ` + "`MaxSessionsPerIP`" + `)
4. Enable observability (` + "`middleware.OpenTelemetry()`" + `)
5. Configure CSRF secret

See the [Production Guide](https://docs.vango.dev/production) for details.

## Learn More

- [Documentation](https://docs.vango.dev)
- [API Reference](https://pkg.go.dev/github.com/vango-go/vango)
- [Examples](https://github.com/vango-go/examples)
`

const readmeAPITemplate = `# {{.ProjectName}}

{{.Description}}

## Getting Started

` + "```bash" + `
# Start development server
vango dev

# Build for production
vango build
` + "```" + `

## API Endpoints

- ` + "`GET /api/health`" + ` - Health check endpoint
`

const tailwindConfigTemplate = `/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    './*.go',
    './app/**/*.go',
  ],
  theme: {
    extend: {},
  },
  plugins: [],
}
`

const tailwindInputCSS = `@tailwind base;
@tailwind components;
@tailwind utilities;
`

const packageJSONTemplate = `{
  "name": "{{.ProjectName}}",
  "private": true,
  "scripts": {
    "build:css": "npx tailwindcss -i ./app/styles/input.css -o ./public/styles.css --minify",
    "watch:css": "npx tailwindcss -i ./app/styles/input.css -o ./public/styles.css --watch"
  },
  "devDependencies": {
    "tailwindcss": "^3.4.0"
  }
}
`

// =============================================================================
// Extended Templates (--with-db, --with-auth)
// =============================================================================

// GetWithOptions returns a template with optional features enabled.
func GetWithOptions(name string, cfg Config) (*Template, error) {
	tmpl, err := Get(name)
	if err != nil {
		return nil, err
	}

	// Clone the files map
	files := make(map[string]string)
	for k, v := range tmpl.Files {
		files[k] = v
	}

	// Add database files if requested
	if cfg.HasDatabase {
		addDatabaseFiles(files, cfg)
	}

	// Add auth files if requested
	if cfg.HasAuth {
		addAuthFiles(files, cfg)
	}

	// Update routes_gen.go if we have additional routes
	if cfg.HasDatabase || cfg.HasAuth {
		files["app/routes/routes_gen.go"] = generateRoutesGen(cfg)
	}

	return &Template{
		Name:        tmpl.Name,
		Description: tmpl.Description,
		Files:       files,
	}, nil
}

func addDatabaseFiles(files map[string]string, cfg Config) {
	// Add projects route using file-based routing convention [id].go
	files["app/routes/projects/[id].go"] = projectShowGoTemplate

	// Add users API
	files["app/routes/api/users.go"] = usersAPIGoTemplate

	// Add db package placeholder
	files["db/db.go"] = dbGoTemplate
}

func addAuthFiles(files map[string]string, cfg Config) {
	// Add admin routes
	files["app/routes/admin/_middleware.go"] = adminMiddlewareTemplate
	files["app/routes/admin/_layout.go"] = adminLayoutTemplate
	files["app/routes/admin/index.go"] = adminIndexTemplate
}

func generateRoutesGen(cfg Config) string {
	imports := []string{
		`"github.com/vango-go/vango/pkg/router"`,
		`"` + cfg.ModulePath + `/app/routes/api"`,
	}
	routes := []string{
		`r.Page("/", IndexPage, Layout)`,
		`r.Page("/about", AboutPage, Layout)`,
		`r.API("GET", "/api/health", api.HealthGET)`,
	}

	if cfg.HasDatabase {
		imports = append(imports, `"`+cfg.ModulePath+`/app/routes/projects"`)
		routes = append(routes,
			`r.Page("/projects", projects.IndexPage, Layout)`,
			`r.Page("/projects/:id", projects.ShowPage, Layout)`,
			`r.API("GET", "/api/users", api.UsersGET)`,
			`r.API("POST", "/api/users", api.UsersPOST)`,
		)
	}

	if cfg.HasAuth {
		imports = append(imports, `"`+cfg.ModulePath+`/app/routes/admin"`)
		routes = append(routes,
			`r.Page("/admin", admin.IndexPage, admin.Layout, admin.Middleware...)`,
		)
	}

	return `// Code generated by vango. DO NOT EDIT.

package routes

import (
	` + joinImports(imports) + `
)

// Register adds all routes to the router.
// Generated by ` + "`vango dev`" + ` or ` + "`vango gen routes`" + `.
func Register(r *router.Router) {
	` + joinRoutes(routes) + `
}
`
}

func joinImports(imports []string) string {
	result := ""
	for i, imp := range imports {
		if i > 0 {
			result += "\n\t"
		}
		result += imp
	}
	return result
}

func joinRoutes(routes []string) string {
	result := ""
	for i, route := range routes {
		if i > 0 {
			result += "\n\t"
		}
		result += route
	}
	return result
}

const projectShowGoTemplate = `package projects

import (
	"github.com/vango-go/vango/pkg/server"
	"github.com/vango-go/vango/pkg/vango"
	"github.com/vango-go/vango/pkg/vdom"
	. "github.com/vango-go/vango/el"

	"{{.ModulePath}}/db"
)

// Params defines the route parameters for /projects/:id.
type Params struct {
	ID int ` + "`param:\"id\"`" + `
}

// ShowPage handles /projects/:id.
func ShowPage(ctx server.Ctx, p Params) vdom.Component {
	return vango.Func(func() *vdom.VNode {
		project := vango.NewSignal[*db.Project](nil)

		vango.Effect(func() vango.Cleanup {
			proj, _ := db.Projects.Find(p.ID)
			project.Set(proj)
			return nil
		})

		if project.Get() == nil {
			return Div(Text("Loading..."))
		}

		return Div(
			H1(Text(project.Get().Name)),
			P(Text(project.Get().Description)),
		)
	})
}
`

const usersAPIGoTemplate = `package api

import (
	"github.com/vango-go/vango/pkg/server"
	"github.com/vango-go/vango/pkg/vango"

	"{{.ModulePath}}/db"
)

// --- Request/Response Types ---

type User struct {
	ID    string ` + "`json:\"id\"`" + `
	Email string ` + "`json:\"email\"`" + `
	Name  string ` + "`json:\"name\"`" + `
}

type CreateUserInput struct {
	Email string ` + "`json:\"email\" validate:\"required,email\"`" + `
	Name  string ` + "`json:\"name\" validate:\"required\"`" + `
}

// --- Handlers ---

// UsersGET handles GET /api/users.
func UsersGET(ctx server.Ctx) ([]*User, error) {
	return db.Users.All()
}

// UsersPOST handles POST /api/users.
// Returns 201 Created with Location header.
func UsersPOST(ctx server.Ctx, input CreateUserInput) (*vango.Response[User], error) {
	user, err := db.Users.Create(input)
	if err != nil {
		return nil, err
	}

	// Set Location header manually
	ctx.SetHeader("Location", "/api/users/"+user.ID)

	return vango.Created(user), nil
}
`

const dbGoTemplate = `package db

// Project represents a project entity.
type Project struct {
	ID          int
	Name        string
	Description string
}

// User represents a user entity.
type User struct {
	ID    string
	Email string
	Name  string
	Role  string
}

// Projects is a placeholder projects repository.
var Projects = &projectsRepo{}

type projectsRepo struct{}

func (r *projectsRepo) Find(id int) (*Project, error) {
	// TODO: Implement database query
	return &Project{
		ID:          id,
		Name:        "Sample Project",
		Description: "This is a sample project",
	}, nil
}

// Users is a placeholder users repository.
var Users = &usersRepo{}

type usersRepo struct{}

func (r *usersRepo) All() ([]*User, error) {
	// TODO: Implement database query
	return []*User{}, nil
}

func (r *usersRepo) Create(input any) (User, error) {
	// TODO: Implement database insert
	return User{ID: "new-id"}, nil
}
`

const adminMiddlewareTemplate = `package admin

import (
	"github.com/vango-go/vango/pkg/router"

	"{{.ModulePath}}/app/middleware"
)

// Middleware applies to all routes in /admin/*.
var Middleware = []router.Middleware{
	middleware.RequireAuth,
}
`

const adminLayoutTemplate = `package admin

import (
	"github.com/vango-go/vango/pkg/router"
	"github.com/vango-go/vango/pkg/server"
	"github.com/vango-go/vango/pkg/vdom"
	. "github.com/vango-go/vango/el"
)

// Layout wraps all admin routes.
// Uses Link() for SPA navigation with data-vango-link attribute.
func Layout(ctx server.Ctx, children router.Slot) *vdom.VNode {
	return Div(Class("admin-layout"),
		Nav(Class("admin-sidebar"),
			Link("/admin", Text("Dashboard")),
			Link("/admin/users", Text("Users")),
			Link("/admin/settings", Text("Settings")),
		),
		Div(Class("admin-content"),
			children,
		),
	)
}
`

const adminIndexTemplate = `package admin

import (
	"github.com/vango-go/vango/pkg/server"
	"github.com/vango-go/vango/pkg/vdom"
	. "github.com/vango-go/vango/el"
)

// User interface for type assertion.
type User interface {
	GetName() string
}

// IndexPage is the admin dashboard.
func IndexPage(ctx server.Ctx) *vdom.VNode {
	name := "Admin"
	if u, ok := ctx.User().(User); ok {
		name = u.GetName()
	}

	return Div(
		H1(Text("Admin Dashboard")),
		P(Textf("Welcome, %s", name)),
	)
}
`
