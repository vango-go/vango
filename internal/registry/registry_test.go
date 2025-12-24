package registry

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vango-dev/vango/v2/internal/config"
)

func TestNew(t *testing.T) {
	cfg := config.New()
	reg := New(cfg)

	if reg == nil {
		t.Fatal("New returned nil")
	}
	if reg.config != cfg {
		t.Error("Config not set")
	}
	if reg.client == nil {
		t.Error("HTTP client not set")
	}
}

func TestParseHeader(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		wantChecksum string
		wantVersion  string
	}{
		{
			name: "valid header",
			content: `// Source: vango.dev/ui/button
// Version: 1.0.0
// Checksum: sha256:a1b2c3d4

package ui
`,
			wantChecksum: "a1b2c3d4",
			wantVersion:  "1.0.0",
		},
		{
			name:         "no header",
			content:      "package ui\n\nfunc Button() {}",
			wantChecksum: "",
			wantVersion:  "",
		},
		{
			name: "partial header",
			content: `// Source: vango.dev/ui/button
// Version: 2.0.0

package ui
`,
			wantChecksum: "",
			wantVersion:  "2.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checksum, version := parseHeader(tt.content)
			if checksum != tt.wantChecksum {
				t.Errorf("checksum = %q, want %q", checksum, tt.wantChecksum)
			}
			if version != tt.wantVersion {
				t.Errorf("version = %q, want %q", version, tt.wantVersion)
			}
		})
	}
}

func TestRegistry_resolveDependencies(t *testing.T) {
	manifest := &Manifest{
		Components: map[string]Component{
			"button": {
				Files:     []string{"button.go"},
				DependsOn: []string{"utils"},
			},
			"dialog": {
				Files:     []string{"dialog.go"},
				DependsOn: []string{"utils", "focustrap"},
			},
			"utils": {
				Files:     []string{"utils.go"},
				DependsOn: nil,
			},
			"focustrap": {
				Files:     []string{"focustrap.go"},
				DependsOn: []string{"utils"},
			},
		},
	}

	cfg := config.New()
	reg := New(cfg)

	// Test simple dependency
	order, err := reg.resolveDependencies(manifest, []string{"button"})
	if err != nil {
		t.Fatalf("resolveDependencies error: %v", err)
	}
	if len(order) != 2 {
		t.Errorf("Expected 2 components, got %d: %v", len(order), order)
	}
	// utils should come before button
	utilsIdx := indexOf(order, "utils")
	buttonIdx := indexOf(order, "button")
	if utilsIdx > buttonIdx {
		t.Error("utils should be installed before button")
	}

	// Test deeper dependency chain
	order, err = reg.resolveDependencies(manifest, []string{"dialog"})
	if err != nil {
		t.Fatalf("resolveDependencies error: %v", err)
	}
	if len(order) != 3 {
		t.Errorf("Expected 3 components, got %d: %v", len(order), order)
	}
}

func TestRegistry_resolveDependencies_NotFound(t *testing.T) {
	manifest := &Manifest{
		Components: map[string]Component{},
	}

	cfg := config.New()
	reg := New(cfg)

	_, err := reg.resolveDependencies(manifest, []string{"nonexistent"})
	if err == nil {
		t.Error("Expected error for nonexistent component")
	}
}

func TestRegistry_Init(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.New()
	cfg.Paths.UI = filepath.Join(tmpDir, "ui") // Phase 14: Use Paths.UI
	cfg.SaveTo(filepath.Join(tmpDir, "vango.json"))

	reg := New(cfg)

	if err := reg.Init(nil); err != nil {
		t.Fatalf("Init error: %v", err)
	}

	// Check utils.go
	utilsPath := filepath.Join(tmpDir, "ui", "utils.go")
	if _, err := os.Stat(utilsPath); os.IsNotExist(err) {
		t.Error("utils.go not created")
	}

	// Check base.go
	basePath := filepath.Join(tmpDir, "ui", "base.go")
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		t.Error("base.go not created")
	}

	// Check content
	utils, _ := os.ReadFile(utilsPath)
	if !containsStr(string(utils), "func CN") {
		t.Error("utils.go should contain CN function")
	}

	base, _ := os.ReadFile(basePath)
	if !containsStr(string(base), "BaseConfig") {
		t.Error("base.go should contain BaseConfig")
	}
}

func TestRegistry_ListInstalled_Empty(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.New()
	cfg.Paths.UI = filepath.Join(tmpDir, "nonexistent") // Phase 14: Use Paths.UI
	cfg.SaveTo(filepath.Join(tmpDir, "vango.json"))

	reg := New(cfg)

	installed, err := reg.ListInstalled()
	if err != nil {
		t.Fatalf("ListInstalled error: %v", err)
	}
	if len(installed) != 0 {
		t.Errorf("Expected 0 installed, got %d", len(installed))
	}
}

func TestRegistry_ListInstalled_WithComponents(t *testing.T) {
	tmpDir := t.TempDir()
	uiDir := filepath.Join(tmpDir, "ui")
	os.MkdirAll(uiDir, 0755)

	// Create a component file
	content := `// Source: vango.dev/ui/button
// Version: 1.0.0
// Checksum: sha256:a1b2c3d4e5f6

package ui

func Button() {}
`
	os.WriteFile(filepath.Join(uiDir, "button.go"), []byte(content), 0644)

	cfg := config.New()
	cfg.Paths.UI = uiDir // Phase 14: Use Paths.UI instead of UI.Path
	cfg.SaveTo(filepath.Join(tmpDir, "vango.json"))

	reg := New(cfg)

	installed, err := reg.ListInstalled()
	if err != nil {
		t.Fatalf("ListInstalled error: %v", err)
	}
	if len(installed) != 1 {
		t.Fatalf("Expected 1 installed, got %d", len(installed))
	}

	comp := installed[0]
	if comp.Name != "button" {
		t.Errorf("Name = %q, want %q", comp.Name, "button")
	}
	if comp.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", comp.Version, "1.0.0")
	}
}

func TestInstalledComponent_Fields(t *testing.T) {
	ic := InstalledComponent{
		Name:     "button",
		Version:  "1.0.0",
		Checksum: "abc123",
		Modified: true,
	}

	if ic.Name != "button" {
		t.Error("Name mismatch")
	}
	if !ic.Modified {
		t.Error("Modified should be true")
	}
}

func TestManifest_Fields(t *testing.T) {
	m := Manifest{
		ManifestVersion: 1,
		Version:         "1.0.0",
		Registry:        "https://vango.dev",
		Components: map[string]Component{
			"button": {
				Files:     []string{"button.go"},
				DependsOn: []string{"utils"},
			},
		},
	}

	if m.ManifestVersion != 1 {
		t.Error("ManifestVersion mismatch")
	}
	if len(m.Components) != 1 {
		t.Error("Components length mismatch")
	}
}

func TestComponentInfo_Fields(t *testing.T) {
	info := ComponentInfo{
		Name:          "button",
		Files:         []string{"button.go"},
		Dependencies:  []string{"utils"},
		Installed:     true,
		LocalVersion:  "1.0.0",
		LatestVersion: "1.1.0",
		Modified:      false,
	}

	if info.Name != "button" {
		t.Error("Name mismatch")
	}
	if !info.Installed {
		t.Error("Installed should be true")
	}
	if info.Modified {
		t.Error("Modified should be false")
	}
}

func indexOf(slice []string, item string) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
