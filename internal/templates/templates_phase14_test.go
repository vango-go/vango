package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPhase14Templates tests the Phase 14 scaffold templates.
func TestPhase14Templates(t *testing.T) {
	t.Run("all templates are available", func(t *testing.T) {
		expectedTemplates := []string{"minimal", "standard", "full", "api"}
		for _, name := range expectedTemplates {
			tmpl, err := Get(name)
			if err != nil {
				t.Errorf("template %q not found: %v", name, err)
				continue
			}
			if tmpl.Name != name {
				t.Errorf("expected template name %q, got %q", name, tmpl.Name)
			}
		}
	})

	t.Run("invalid template returns error", func(t *testing.T) {
		_, err := Get("nonexistent")
		if err == nil {
			t.Error("expected error for nonexistent template")
		}
	})

	t.Run("List returns all template names", func(t *testing.T) {
		names := List()
		if len(names) < 4 {
			t.Errorf("expected at least 4 templates, got %d", len(names))
		}
	})
}

// TestMinimalTemplate tests the minimal template.
func TestMinimalTemplate(t *testing.T) {
	tmpl, err := Get("minimal")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("contains required files", func(t *testing.T) {
		requiredFiles := []string{
			"main.go",
			"go.mod",
			"vango.json",
			"app/routes/index.go",
			"app/routes/api/health.go",
			"public/styles.css",
			".gitignore",
			"README.md",
		}

		for _, file := range requiredFiles {
			if _, ok := tmpl.Files[file]; !ok {
				t.Errorf("missing required file: %s", file)
			}
		}
	})

	t.Run("main.go uses vango.New", func(t *testing.T) {
		content := tmpl.Files["main.go"]
		if !strings.Contains(content, "vango.New") {
			t.Error("main.go should use vango.New()")
		}
	})

	t.Run("vango.json has paths config", func(t *testing.T) {
		content := tmpl.Files["vango.json"]
		if !strings.Contains(content, `"paths"`) {
			t.Error("vango.json should have paths config")
		}
		if !strings.Contains(content, `"routes"`) {
			t.Error("vango.json should have routes in paths")
		}
	})
}

// TestStandardTemplate tests the standard template.
func TestStandardTemplate(t *testing.T) {
	tmpl, err := Get("standard")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("contains layout file", func(t *testing.T) {
		if _, ok := tmpl.Files["app/routes/_layout.go"]; !ok {
			t.Error("missing _layout.go")
		}
	})

	t.Run("contains about page", func(t *testing.T) {
		if _, ok := tmpl.Files["app/routes/about.go"]; !ok {
			t.Error("missing about.go")
		}
	})

	t.Run("contains navbar and footer", func(t *testing.T) {
		if _, ok := tmpl.Files["app/components/shared/navbar.go"]; !ok {
			t.Error("missing navbar.go")
		}
		if _, ok := tmpl.Files["app/components/shared/footer.go"]; !ok {
			t.Error("missing footer.go")
		}
	})

	t.Run("contains middleware", func(t *testing.T) {
		if _, ok := tmpl.Files["app/middleware/auth.go"]; !ok {
			t.Error("missing auth.go middleware")
		}
	})

	t.Run("contains routes_gen.go", func(t *testing.T) {
		if _, ok := tmpl.Files["app/routes/routes_gen.go"]; !ok {
			t.Error("missing routes_gen.go")
		}
	})
}

// TestFullTemplate tests the full template.
func TestFullTemplate(t *testing.T) {
	tmpl, err := Get("full")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("contains tailwind config", func(t *testing.T) {
		if _, ok := tmpl.Files["tailwind.config.js"]; !ok {
			t.Error("missing tailwind.config.js")
		}
	})

	t.Run("contains package.json", func(t *testing.T) {
		if _, ok := tmpl.Files["package.json"]; !ok {
			t.Error("missing package.json")
		}
	})

	t.Run("contains input.css", func(t *testing.T) {
		if _, ok := tmpl.Files["app/styles/input.css"]; !ok {
			t.Error("missing input.css")
		}
	})
}

// TestAPITemplate tests the API-only template.
func TestAPITemplate(t *testing.T) {
	tmpl, err := Get("api")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("contains health API", func(t *testing.T) {
		if _, ok := tmpl.Files["app/routes/api/health.go"]; !ok {
			t.Error("missing health.go API")
		}
	})

	t.Run("does not contain UI files", func(t *testing.T) {
		if _, ok := tmpl.Files["app/routes/_layout.go"]; ok {
			t.Error("API template should not have _layout.go")
		}
		if _, ok := tmpl.Files["public/styles.css"]; ok {
			t.Error("API template should not have styles.css")
		}
	})
}

// TestTemplateCreate tests creating a project from a template.
func TestTemplateCreate(t *testing.T) {
	tmpl, err := Get("minimal")
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	cfg := Config{
		ProjectName: "test-app",
		ModulePath:  "example.com/test-app",
		Description: "A test application",
	}

	if err := tmpl.Create(dir, cfg); err != nil {
		t.Fatal(err)
	}

	t.Run("creates main.go with correct module path", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(dir, "main.go"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(content), "example.com/test-app") {
			t.Error("main.go should contain module path")
		}
	})

	t.Run("creates go.mod with correct module", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(content), "module example.com/test-app") {
			t.Error("go.mod should contain correct module")
		}
	})

	t.Run("creates vango.json with project name", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join(dir, "vango.json"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(content), "test-app") {
			t.Error("vango.json should contain project name")
		}
	})

	t.Run("creates directory structure", func(t *testing.T) {
		dirs := []string{
			"app/routes",
			"app/routes/api",
			"public",
		}
		for _, d := range dirs {
			if _, err := os.Stat(filepath.Join(dir, d)); os.IsNotExist(err) {
				t.Errorf("missing directory: %s", d)
			}
		}
	})
}

// TestGetWithOptions tests template customization.
func TestGetWithOptions(t *testing.T) {
	t.Run("adds database files when HasDatabase is true", func(t *testing.T) {
		cfg := Config{
			ProjectName: "test-app",
			ModulePath:  "example.com/test-app",
			HasDatabase: true,
		}

		tmpl, err := GetWithOptions("standard", cfg)
		if err != nil {
			t.Fatal(err)
		}

		// Should have db/db.go
		if _, ok := tmpl.Files["db/db.go"]; !ok {
			t.Error("should have db/db.go when HasDatabase is true")
		}

		// Should have projects route
		if _, ok := tmpl.Files["app/routes/projects/[id].go"]; !ok {
			t.Error("should have projects route when HasDatabase is true")
		}
	})

	t.Run("adds auth files when HasAuth is true", func(t *testing.T) {
		cfg := Config{
			ProjectName: "test-app",
			ModulePath:  "example.com/test-app",
			HasAuth:     true,
		}

		tmpl, err := GetWithOptions("standard", cfg)
		if err != nil {
			t.Fatal(err)
		}

		// Should have admin routes
		if _, ok := tmpl.Files["app/routes/admin/_middleware.go"]; !ok {
			t.Error("should have admin middleware when HasAuth is true")
		}
		if _, ok := tmpl.Files["app/routes/admin/_layout.go"]; !ok {
			t.Error("should have admin layout when HasAuth is true")
		}
		if _, ok := tmpl.Files["app/routes/admin/index.go"]; !ok {
			t.Error("should have admin index when HasAuth is true")
		}
	})
}

// TestStylesCSS tests the CSS content.
func TestStylesCSS(t *testing.T) {
	tmpl, err := Get("standard")
	if err != nil {
		t.Fatal(err)
	}

	css := tmpl.Files["public/styles.css"]

	t.Run("contains CSS variables", func(t *testing.T) {
		if !strings.Contains(css, ":root") {
			t.Error("CSS should have :root variables")
		}
		if !strings.Contains(css, "--background") {
			t.Error("CSS should have --background variable")
		}
	})

	t.Run("contains navbar styles", func(t *testing.T) {
		if !strings.Contains(css, ".navbar") {
			t.Error("CSS should have .navbar styles")
		}
	})

	t.Run("contains connection state indicators", func(t *testing.T) {
		if !strings.Contains(css, ".vango-reconnecting") {
			t.Error("CSS should have .vango-reconnecting styles")
		}
		if !strings.Contains(css, ".vango-offline") {
			t.Error("CSS should have .vango-offline styles")
		}
	})
}
