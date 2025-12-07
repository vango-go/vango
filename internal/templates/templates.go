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
	"bytes"
	"log"
	"net/http"

	"github.com/vango-dev/vango/v2/pkg/render"
	. "github.com/vango-dev/vango/v2/pkg/vdom"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		page := HomePage()

		renderer := render.NewRenderer(render.RendererConfig{})
		var buf bytes.Buffer
		if err := renderer.RenderPage(&buf, render.PageData{
			Title: "{{.ProjectName}}",
			Body:  page.Render(),
			Styles: []string{` + "`" + `
				body { font-family: system-ui, sans-serif; max-width: 800px; margin: 0 auto; padding: 2rem; }
				h1 { color: #2563eb; }
			` + "`" + `},
		}); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(buf.Bytes())
	})

	log.Println("Server running at http://localhost:3000")
	if err := http.ListenAndServe(":3000", mux); err != nil {
		log.Fatal(err)
	}
}

// HomePage is the home page component.
func HomePage() Component {
	return Func(func() *VNode {
		return Div(
			H1("Welcome to {{.ProjectName}}"),
			P("{{.Description}}"),
		)
	})
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
	"bytes"
	"log"
	"net/http"
	"os"

	"github.com/vango-dev/vango/v2/pkg/render"
	. "github.com/vango-dev/vango/v2/pkg/vdom"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	mux := http.NewServeMux()

	// Serve static files
	mux.Handle("/public/", http.StripPrefix("/public/", http.FileServer(http.Dir("public"))))

	// Page routes
	mux.HandleFunc("/", renderPage("{{.ProjectName}}", HomePage))
	mux.HandleFunc("/about", renderPage("About - {{.ProjectName}}", AboutPage))

	log.Printf("Server running at http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

// renderPage returns an HTTP handler that renders a Vango component.
func renderPage(title string, component func() Component) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page := component()

		renderer := render.NewRenderer(render.RendererConfig{})
		var buf bytes.Buffer
		if err := renderer.RenderPage(&buf, render.PageData{
			Title: title,
			Body:  page.Render(),
			StyleSheets: []string{"/public/styles.css"},
		}); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(buf.Bytes())
	}
}

// HomePage is the home page component.
func HomePage() Component {
	return Func(func() *VNode {
		return Div(Class("min-h-screen bg-gray-100"),
			Navbar(),
			Main(Class("container mx-auto px-4 py-8"),
				H1(Class("text-4xl font-bold text-gray-900 mb-4"),
					"Welcome to {{.ProjectName}}",
				),
				P(Class("text-lg text-gray-600 mb-8"),
					"{{.Description}}",
				),
				Counter(),
			),
		)
	})
}

// AboutPage is the about page component.
func AboutPage() Component {
	return Func(func() *VNode {
		return Div(Class("min-h-screen bg-gray-100"),
			Navbar(),
			Main(Class("container mx-auto px-4 py-8"),
				H1(Class("text-4xl font-bold text-gray-900 mb-4"),
					"About",
				),
				P(Class("text-lg text-gray-600"),
					"This is a Vango application - a server-driven web framework for Go.",
				),
			),
		)
	})
}

// Navbar is the navigation bar component.
func Navbar() *VNode {
	return Nav(Class("bg-white shadow"),
		Div(Class("container mx-auto px-4"),
			Div(Class("flex justify-between items-center h-16"),
				A(Href("/"), Class("text-xl font-bold text-gray-900"),
					"{{.ProjectName}}",
				),
				Div(Class("flex gap-4"),
					A(Href("/"), Class("text-gray-600 hover:text-gray-900"), "Home"),
					A(Href("/about"), Class("text-gray-600 hover:text-gray-900"), "About"),
				),
			),
		),
	)
}

// Counter is a simple counter component demonstrating interactivity.
// Note: Full interactivity requires WebSocket connection (vango dev).
func Counter() *VNode {
	return Div(Class("bg-white rounded-lg shadow p-6 max-w-sm"),
		H2(Class("text-2xl font-semibold text-gray-800 mb-4"), "Counter"),
		P(Class("text-gray-600 mb-4"),
			"This demonstrates Vango components. Full interactivity with signals ",
			"and event handlers works over WebSocket in development mode.",
		),
		Div(Class("flex gap-2"),
			Button(
				Class("px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"),
				"-",
			),
			Span(Class("px-4 py-2 text-2xl font-bold"), "0"),
			Button(
				Class("px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"),
				"+",
			),
		),
	)
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
			"public/favicon.ico":  "", // Empty placeholder
			"public/styles.css":   "", // Generated by Tailwind or empty
			"app/styles/input.css": `@tailwind base;
@tailwind components;
@tailwind utilities;
`,
			"tailwind.config.js": `/** @type {import('tailwindcss').Config} */
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
├── main.go              # Entry point with routes and components
├── app/
│   └── styles/          # Source CSS (Tailwind input)
├── public/              # Static assets
│   └── styles.css       # Compiled CSS
├── vango.json           # Vango configuration
├── tailwind.config.js   # Tailwind configuration
└── README.md
` + "```" + `

## Development

The development server provides:
- Hot reload on file changes
- WebSocket connection for live updates
- Tailwind CSS compilation (if enabled)

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
		"status": "ok",
	})
}
`,
			"go.mod": `module {{.ModulePath}}

go 1.22
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
