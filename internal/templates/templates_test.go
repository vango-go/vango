package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGet(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"minimal", false},
		{"full", false},
		{"api", false},
		{"nonexistent", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := Get(tt.name)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if tmpl == nil {
					t.Error("Template should not be nil")
				}
				if tmpl.Name != tt.name {
					t.Errorf("Name = %q, want %q", tmpl.Name, tt.name)
				}
			}
		})
	}
}

func TestList(t *testing.T) {
	names := List()
	if len(names) < 3 {
		t.Errorf("Expected at least 3 templates, got %d", len(names))
	}

	// Check that all expected templates are present
	expected := map[string]bool{
		"minimal": false,
		"full":    false,
		"api":     false,
	}

	for _, name := range names {
		if _, ok := expected[name]; ok {
			expected[name] = true
		}
	}

	for name, found := range expected {
		if !found {
			t.Errorf("Template %q not found in list", name)
		}
	}
}

func TestTemplate_Create_Minimal(t *testing.T) {
	tmpDir := t.TempDir()

	tmpl, _ := Get("minimal")
	cfg := Config{
		ProjectName: "test-app",
		ModulePath:  "github.com/test/test-app",
		Description: "A test application",
	}

	if err := tmpl.Create(tmpDir, cfg); err != nil {
		t.Fatalf("Create error: %v", err)
	}

	// Check that files were created
	expectedFiles := []string{
		"main.go",
		"go.mod",
		"vango.json",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(tmpDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("File %q not created", file)
		}
	}

	// Check content substitution in main.go
	mainGo, _ := os.ReadFile(filepath.Join(tmpDir, "main.go"))
	if !strings.Contains(string(mainGo), "test-app") {
		t.Error("Project name not substituted in main.go")
	}
	if !strings.Contains(string(mainGo), "A test application") {
		t.Error("Description not substituted in main.go")
	}

	// Check go.mod has module path
	goMod, _ := os.ReadFile(filepath.Join(tmpDir, "go.mod"))
	if !strings.Contains(string(goMod), "github.com/test/test-app") {
		t.Error("Module path not substituted in go.mod")
	}
}

func TestTemplate_Create_Full(t *testing.T) {
	tmpDir := t.TempDir()

	tmpl, _ := Get("full")
	cfg := Config{
		ProjectName: "my-app",
		ModulePath:  "myapp",
		Description: "My awesome app",
		HasTailwind: true,
	}

	if err := tmpl.Create(tmpDir, cfg); err != nil {
		t.Fatalf("Create error: %v", err)
	}

	// Check core files
	expectedFiles := []string{
		"main.go",
		"go.mod",
		"vango.json",
		"README.md",
		"tailwind.config.js",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(tmpDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("File %q not created", file)
		}
	}

	// Check main.go contains components
	mainGo, _ := os.ReadFile(filepath.Join(tmpDir, "main.go"))
	mainGoStr := string(mainGo)
	if !strings.Contains(mainGoStr, "Counter") {
		t.Error("Counter component not in main.go")
	}
	if !strings.Contains(mainGoStr, "Navbar") {
		t.Error("Navbar component not in main.go")
	}
	if !strings.Contains(mainGoStr, "HomePage") {
		t.Error("HomePage component not in main.go")
	}

	// Check README
	readme, _ := os.ReadFile(filepath.Join(tmpDir, "README.md"))
	if !strings.Contains(string(readme), "my-app") {
		t.Error("Project name not in README")
	}

	// Check tailwind config when enabled
	vangoJSON, _ := os.ReadFile(filepath.Join(tmpDir, "vango.json"))
	if !strings.Contains(string(vangoJSON), "tailwind") {
		t.Error("Tailwind should be in vango.json when enabled")
	}
}

func TestTemplate_Create_API(t *testing.T) {
	tmpDir := t.TempDir()

	tmpl, _ := Get("api")
	cfg := Config{
		ProjectName: "my-api",
		ModulePath:  "myapi",
		Description: "My API",
	}

	if err := tmpl.Create(tmpDir, cfg); err != nil {
		t.Fatalf("Create error: %v", err)
	}

	// Check core files
	expectedFiles := []string{
		"main.go",
		"go.mod",
		"vango.json",
		"README.md",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(tmpDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("File %q not created", file)
		}
	}

	// Check main.go has health handler
	mainGo, _ := os.ReadFile(filepath.Join(tmpDir, "main.go"))
	if !strings.Contains(string(mainGo), "handleHealth") {
		t.Error("Health handler not in main.go")
	}
	if !strings.Contains(string(mainGo), "/api/health") {
		t.Error("Health endpoint not registered")
	}
}

func TestConfig_Fields(t *testing.T) {
	cfg := Config{
		ProjectName: "test",
		ModulePath:  "github.com/test/test",
		Description: "Test desc",
		HasTailwind: true,
		HasDatabase: false,
	}

	if cfg.ProjectName != "test" {
		t.Error("ProjectName mismatch")
	}
	if !cfg.HasTailwind {
		t.Error("HasTailwind should be true")
	}
	if cfg.HasDatabase {
		t.Error("HasDatabase should be false")
	}
}

func TestTemplate_Description(t *testing.T) {
	minimal, _ := Get("minimal")
	if minimal.Description == "" {
		t.Error("Minimal template should have description")
	}

	full, _ := Get("full")
	if full.Description == "" {
		t.Error("Full template should have description")
	}

	api, _ := Get("api")
	if api.Description == "" {
		t.Error("API template should have description")
	}
}
