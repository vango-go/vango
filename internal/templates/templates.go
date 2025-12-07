package templates

import (
	"bytes"
	"os"
	"path/filepath"
	"text/template"

	"github.com/vango-dev/vango/v2/internal/errors"
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
	"minimal": minimalTemplate(),
	"full":    fullTemplate(),
	"api":     apiTemplate(),
}

// Get returns a template by name.
func Get(name string) (*Template, error) {
	tmpl, ok := templates[name]
	if !ok {
		return nil, errors.New("E145").
			WithDetail("Template '" + name + "' not found").
			WithSuggestion("Available templates: minimal, full, api")
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
func minimalTemplate() *Template {
	return &Template{
		Name:        "minimal",
		Description: "Just the essentials for a Vango app",
		Files: map[string]string{
			"main.go": `package main

import (
	"log"

	"{{.ModulePath}}/app/routes"
	"github.com/vango-dev/vango/v2/pkg/server"
)

func main() {
	srv := server.New(server.Config{
		Addr: ":3000",
	})

	routes.Register(srv)

	log.Println("Server running at http://localhost:3000")
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
`,
			"go.mod": `module {{.ModulePath}}

go 1.22

require github.com/vango-dev/vango/v2 v2.0.0
`,
			"vango.json": `{
  "dev": {
    "port": 3000,
    "openBrowser": true
  },
  "build": {
    "output": "dist",
    "minify": true
  }
}
`,
			"app/routes/index.go": `package routes

import (
	. "github.com/vango-dev/vango/v2/pkg/vdom"
	"github.com/vango-dev/vango/v2/pkg/vango"
)

// Page is the home page component.
func Page(ctx *vango.Ctx) vango.Component {
	return vango.Func(func() *VNode {
		return Div(Class("container"),
			H1(Text("Welcome to {{.ProjectName}}")),
			P(Text("{{.Description}}")),
		)
	})
}
`,
			"app/routes/routes.go": `package routes

import "github.com/vango-dev/vango/v2/pkg/server"

// Register registers all routes.
func Register(srv *server.Server) {
	srv.Get("/", Page)
}
`,
		},
	}
}

// fullTemplate returns the full template with examples.
func fullTemplate() *Template {
	return &Template{
		Name:        "full",
		Description: "Complete starter with example pages and components",
		Files: map[string]string{
			"main.go": `package main

import (
	"log"
	"os"

	"{{.ModulePath}}/app/routes"
	"github.com/vango-dev/vango/v2/pkg/server"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	srv := server.New(server.Config{
		Addr: ":" + port,
	})

	routes.Register(srv)

	log.Printf("Server running at http://localhost:%s", port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
`,
			"go.mod": `module {{.ModulePath}}

go 1.22

require github.com/vango-dev/vango/v2 v2.0.0
`,
			"vango.json": `{
  "dev": {
    "port": 3000,
    "host": "localhost",
    "openBrowser": true
  },
  "build": {
    "output": "dist",
    "minify": true
  }{{if .HasTailwind}},
  "tailwind": {
    "enabled": true,
    "config": "./tailwind.config.js"
  }{{end}}
}
`,
			"app/routes/index.go": `package routes

import (
	. "github.com/vango-dev/vango/v2/pkg/vdom"
	"github.com/vango-dev/vango/v2/pkg/vango"
	"{{.ModulePath}}/app/components"
)

// Page is the home page component.
func Page(ctx *vango.Ctx) vango.Component {
	return vango.Func(func() *VNode {
		return Div(Class("min-h-screen bg-gray-100"),
			components.Navbar(),
			Main(Class("container mx-auto px-4 py-8"),
				H1(Class("text-4xl font-bold text-gray-900 mb-4"),
					Text("Welcome to {{.ProjectName}}"),
				),
				P(Class("text-lg text-gray-600 mb-8"),
					Text("{{.Description}}"),
				),
				components.Counter(0),
			),
		)
	})
}
`,
			"app/routes/about.go": `package routes

import (
	. "github.com/vango-dev/vango/v2/pkg/vdom"
	"github.com/vango-dev/vango/v2/pkg/vango"
	"{{.ModulePath}}/app/components"
)

// AboutPage is the about page component.
func AboutPage(ctx *vango.Ctx) vango.Component {
	return vango.Func(func() *VNode {
		return Div(Class("min-h-screen bg-gray-100"),
			components.Navbar(),
			Main(Class("container mx-auto px-4 py-8"),
				H1(Class("text-4xl font-bold text-gray-900 mb-4"),
					Text("About"),
				),
				P(Class("text-lg text-gray-600"),
					Text("This is a Vango application."),
				),
			),
		)
	})
}
`,
			"app/routes/routes.go": `package routes

import "github.com/vango-dev/vango/v2/pkg/server"

// Register registers all routes.
func Register(srv *server.Server) {
	srv.Get("/", Page)
	srv.Get("/about", AboutPage)
}
`,
			"app/components/counter.go": `package components

import (
	"fmt"

	. "github.com/vango-dev/vango/v2/pkg/vdom"
	"github.com/vango-dev/vango/v2/pkg/vango"
)

// Counter is a simple counter component.
func Counter(initial int) vango.Component {
	return vango.Func(func() *VNode {
		count := vango.NewIntSignal(initial)

		return Div(Class("bg-white rounded-lg shadow p-6 max-w-sm"),
			H2(Class("text-2xl font-semibold text-gray-800 mb-4"),
				Text("Counter"),
			),
			P(Class("text-4xl font-bold text-blue-600 mb-4"),
				Text(fmt.Sprintf("%d", count.Get())),
			),
			Div(Class("flex gap-2"),
				Button(
					Class("px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"),
					OnClick(func() { count.Dec() }),
					Text("-"),
				),
				Button(
					Class("px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"),
					OnClick(func() { count.Inc() }),
					Text("+"),
				),
			),
		)
	})
}
`,
			"app/components/navbar.go": `package components

import (
	. "github.com/vango-dev/vango/v2/pkg/vdom"
)

// Navbar is the navigation bar component.
func Navbar() *VNode {
	return Nav(Class("bg-white shadow"),
		Div(Class("container mx-auto px-4"),
			Div(Class("flex justify-between items-center h-16"),
				A(Href("/"), Class("text-xl font-bold text-gray-900"),
					Text("{{.ProjectName}}"),
				),
				Div(Class("flex gap-4"),
					A(Href("/"), Class("text-gray-600 hover:text-gray-900"),
						Text("Home"),
					),
					A(Href("/about"), Class("text-gray-600 hover:text-gray-900"),
						Text("About"),
					),
				),
			),
		),
	)
}
`,
			"public/favicon.ico":    "", // Empty placeholder
			"app/styles/input.css": `@tailwind base;
@tailwind components;
@tailwind utilities;
`,
			"tailwind.config.js": `/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    './app/**/*.go',
  ],
  theme: {
    extend: {},
  },
  plugins: [],
}
`,
			"README.md": `# {{.ProjectName}}

{{.Description}}

## Getting Started

` + "```" + `bash
# Start development server
vango dev

# Build for production
vango build

# Run production build
./dist/server
` + "```" + `

## Project Structure

` + "```" + `
{{.ProjectName}}/
├── app/
│   ├── routes/          # Page components
│   │   ├── index.go     # Home page (/)
│   │   ├── about.go     # About page (/about)
│   │   └── routes.go    # Route registration
│   └── components/      # Reusable components
│       ├── counter.go
│       └── navbar.go
├── public/              # Static assets
├── main.go              # Entry point
├── vango.json           # Vango configuration
└── README.md
` + "```" + `

## Learn More

- [Vango Documentation](https://vango.dev/docs)
- [API Reference](https://vango.dev/docs/api)
`,
		},
	}
}

// apiTemplate returns the API-only template.
func apiTemplate() *Template {
	return &Template{
		Name:        "api",
		Description: "API-only project without UI",
		Files: map[string]string{
			"main.go": `package main

import (
	"log"
	"os"

	"{{.ModulePath}}/app/handlers"
	"github.com/vango-dev/vango/v2/pkg/server"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	srv := server.New(server.Config{
		Addr: ":" + port,
	})

	handlers.Register(srv)

	log.Printf("API server running at http://localhost:%s", port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
`,
			"go.mod": `module {{.ModulePath}}

go 1.22

require github.com/vango-dev/vango/v2 v2.0.0
`,
			"vango.json": `{
  "dev": {
    "port": 3000
  },
  "build": {
    "output": "dist",
    "minify": true
  }
}
`,
			"app/handlers/health.go": `package handlers

import (
	"encoding/json"
	"net/http"
)

// HealthCheck handles GET /api/health
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}
`,
			"app/handlers/handlers.go": `package handlers

import "github.com/vango-dev/vango/v2/pkg/server"

// Register registers all API handlers.
func Register(srv *server.Server) {
	srv.HandleFunc("/api/health", HealthCheck)
}
`,
			"README.md": `# {{.ProjectName}}

{{.Description}}

## Getting Started

` + "```" + `bash
# Start development server
vango dev

# Build for production
vango build
` + "```" + `

## API Endpoints

- ` + "`" + `GET /api/health` + "`" + ` - Health check endpoint
`,
		},
	}
}
