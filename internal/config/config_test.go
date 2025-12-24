package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	cfg := New()

	if cfg.Dev.Port != DefaultPort {
		t.Errorf("Dev.Port = %d, want %d", cfg.Dev.Port, DefaultPort)
	}
	if cfg.Dev.Host != DefaultHost {
		t.Errorf("Dev.Host = %q, want %q", cfg.Dev.Host, DefaultHost)
	}
	if cfg.Build.Output != DefaultOutput {
		t.Errorf("Build.Output = %q, want %q", cfg.Build.Output, DefaultOutput)
	}
	if cfg.UI.Registry != DefaultRegistry {
		t.Errorf("UI.Registry = %q, want %q", cfg.UI.Registry, DefaultRegistry)
	}
}

func TestLoad(t *testing.T) {
	tmpDir := t.TempDir()

	// Test loading non-existent config
	_, err := Load(tmpDir)
	if err == nil {
		t.Error("Expected error for missing config")
	}

	// Create a config file
	configPath := filepath.Join(tmpDir, ConfigFileName)
	configJSON := `{
  "dev": {
    "port": 8080,
    "host": "0.0.0.0",
    "openBrowser": false
  },
  "build": {
    "output": "build",
    "minify": true
  },
  "tailwind": {
    "enabled": true,
    "config": "./tailwind.config.js"
  },
  "ui": {
    "version": "1.0.0",
    "installed": ["button", "card"]
  }
}
`
	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatal(err)
	}

	// Load the config
	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.Dev.Port != 8080 {
		t.Errorf("Dev.Port = %d, want %d", cfg.Dev.Port, 8080)
	}
	if cfg.Dev.Host != "0.0.0.0" {
		t.Errorf("Dev.Host = %q, want %q", cfg.Dev.Host, "0.0.0.0")
	}
	if cfg.Dev.OpenBrowser {
		t.Error("Dev.OpenBrowser should be false")
	}
	if cfg.Build.Output != "build" {
		t.Errorf("Build.Output = %q, want %q", cfg.Build.Output, "build")
	}
	if !cfg.Tailwind.Enabled {
		t.Error("Tailwind.Enabled should be true")
	}
	if cfg.UI.Version != "1.0.0" {
		t.Errorf("UI.Version = %q, want %q", cfg.UI.Version, "1.0.0")
	}
	if len(cfg.UI.Installed) != 2 {
		t.Errorf("UI.Installed len = %d, want %d", len(cfg.UI.Installed), 2)
	}
}

func TestLoadFile_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ConfigFileName)

	// Write invalid JSON
	if err := os.WriteFile(configPath, []byte("not valid json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFile(configPath)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "E120") {
		t.Errorf("Expected E120 error, got: %v", err)
	}
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ConfigFileName)

	cfg := New()
	cfg.Dev.Port = 9000
	cfg.UI.Version = "2.0.0"

	// Save should fail without configPath set
	err := cfg.Save()
	if err == nil {
		t.Error("Expected error when saving without path")
	}

	// SaveTo should work
	err = cfg.SaveTo(configPath)
	if err != nil {
		t.Fatalf("SaveTo error: %v", err)
	}

	// Reload and verify
	loaded, err := LoadFile(configPath)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if loaded.Dev.Port != 9000 {
		t.Errorf("Dev.Port = %d, want %d", loaded.Dev.Port, 9000)
	}
	if loaded.UI.Version != "2.0.0" {
		t.Errorf("UI.Version = %q, want %q", loaded.UI.Version, "2.0.0")
	}

	// Now Save should work
	loaded.Dev.Port = 9001
	err = loaded.Save()
	if err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// Reload again
	reloaded, err := LoadFile(configPath)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if reloaded.Dev.Port != 9001 {
		t.Errorf("Dev.Port = %d, want %d", reloaded.Dev.Port, 9001)
	}
}

func TestValidate(t *testing.T) {
	cfg := New()

	// Valid config
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate should pass for valid config: %v", err)
	}

	// Invalid port
	cfg.Dev.Port = -1
	if err := cfg.Validate(); err == nil {
		t.Error("Validate should fail for negative port")
	}

	cfg.Dev.Port = 70000
	if err := cfg.Validate(); err == nil {
		t.Error("Validate should fail for port > 65535")
	}
}

func TestDevAddress(t *testing.T) {
	cfg := New()
	cfg.Dev.Port = 8080
	cfg.Dev.Host = "0.0.0.0"

	addr := cfg.DevAddress()
	if addr != "0.0.0.0:8080" {
		t.Errorf("DevAddress = %q, want %q", addr, "0.0.0.0:8080")
	}
}

func TestDevURL(t *testing.T) {
	cfg := New()

	url := cfg.DevURL()
	if url != "http://localhost:3000" {
		t.Errorf("DevURL = %q, want %q", url, "http://localhost:3000")
	}

	cfg.Dev.HTTPS = true
	url = cfg.DevURL()
	if url != "https://localhost:3000" {
		t.Errorf("DevURL with HTTPS = %q, want %q", url, "https://localhost:3000")
	}
}

func TestPaths(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ConfigFileName)

	cfg := New()
	cfg.SaveTo(configPath)

	// Test relative paths
	if got := cfg.OutputPath(); got != filepath.Join(tmpDir, "dist") {
		t.Errorf("OutputPath = %q, want %q", got, filepath.Join(tmpDir, "dist"))
	}
	if got := cfg.RoutesPath(); got != filepath.Join(tmpDir, "app/routes") {
		t.Errorf("RoutesPath = %q, want %q", got, filepath.Join(tmpDir, "app/routes"))
	}
	if got := cfg.ComponentsPath(); got != filepath.Join(tmpDir, "app/components") {
		t.Errorf("ComponentsPath = %q, want %q", got, filepath.Join(tmpDir, "app/components"))
	}
	if got := cfg.PublicPath(); got != filepath.Join(tmpDir, "public") {
		t.Errorf("PublicPath = %q, want %q", got, filepath.Join(tmpDir, "public"))
	}

	// Test absolute paths
	cfg.Build.Output = "/absolute/path"
	if got := cfg.OutputPath(); got != "/absolute/path" {
		t.Errorf("OutputPath absolute = %q, want %q", got, "/absolute/path")
	}
}

func TestHasTailwind(t *testing.T) {
	cfg := New()

	if cfg.HasTailwind() {
		t.Error("HasTailwind should be false by default")
	}

	cfg.Tailwind.Enabled = true
	if !cfg.HasTailwind() {
		t.Error("HasTailwind should be true when enabled")
	}
}

func TestTailwindConfigPath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ConfigFileName)

	cfg := New()
	cfg.SaveTo(configPath)

	// Default path
	if got := cfg.TailwindConfigPath(); got != filepath.Join(tmpDir, "tailwind.config.js") {
		t.Errorf("TailwindConfigPath default = %q, want %q", got, filepath.Join(tmpDir, "tailwind.config.js"))
	}

	// Custom relative path
	cfg.Tailwind.Config = "./config/tailwind.js"
	if got := cfg.TailwindConfigPath(); got != filepath.Join(tmpDir, "config/tailwind.js") {
		t.Errorf("TailwindConfigPath custom = %q, want %q", got, filepath.Join(tmpDir, "config/tailwind.js"))
	}

	// Absolute path
	cfg.Tailwind.Config = "/absolute/tailwind.js"
	if got := cfg.TailwindConfigPath(); got != "/absolute/tailwind.js" {
		t.Errorf("TailwindConfigPath absolute = %q, want %q", got, "/absolute/tailwind.js")
	}
}

func TestExists(t *testing.T) {
	tmpDir := t.TempDir()

	if Exists(tmpDir) {
		t.Error("Exists should be false for empty directory")
	}

	// Create config file
	configPath := filepath.Join(tmpDir, ConfigFileName)
	if err := os.WriteFile(configPath, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	if !Exists(tmpDir) {
		t.Error("Exists should be true after creating config")
	}
}

func TestFindProjectRoot(t *testing.T) {
	// Create nested directory structure
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "a", "b", "c")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Should fail when no config exists
	_, err := FindProjectRoot(nestedDir)
	if err == nil {
		t.Error("FindProjectRoot should fail when no config exists")
	}

	// Create config in root
	configPath := filepath.Join(tmpDir, ConfigFileName)
	if err := os.WriteFile(configPath, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Should find root from nested directory
	root, err := FindProjectRoot(nestedDir)
	if err != nil {
		t.Fatalf("FindProjectRoot error: %v", err)
	}
	if root != tmpDir {
		t.Errorf("FindProjectRoot = %q, want %q", root, tmpDir)
	}

	// Should find root from middle directory
	root, err = FindProjectRoot(filepath.Join(tmpDir, "a"))
	if err != nil {
		t.Fatalf("FindProjectRoot error: %v", err)
	}
	if root != tmpDir {
		t.Errorf("FindProjectRoot = %q, want %q", root, tmpDir)
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{123, "123"},
		{3000, "3000"},
		{65535, "65535"},
		{-1, "-1"},
		{-100, "-100"},
	}

	for _, tt := range tests {
		got := itoa(tt.n)
		if got != tt.want {
			t.Errorf("itoa(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestApplyDefaults(t *testing.T) {
	cfg := &Config{}
	cfg.applyDefaults()

	if cfg.Dev.Port != DefaultPort {
		t.Errorf("Dev.Port = %d, want %d", cfg.Dev.Port, DefaultPort)
	}
	if cfg.Dev.Host != DefaultHost {
		t.Errorf("Dev.Host = %q, want %q", cfg.Dev.Host, DefaultHost)
	}
	if cfg.Build.Output != DefaultOutput {
		t.Errorf("Build.Output = %q, want %q", cfg.Build.Output, DefaultOutput)
	}
	if cfg.Routes != "app/routes" {
		t.Errorf("Routes = %q, want %q", cfg.Routes, "app/routes")
	}
}

func TestUIComponentsPath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ConfigFileName)

	cfg := New()
	cfg.SaveTo(configPath)

	// Default path (Phase 14: Paths.UI)
	expected := filepath.Join(tmpDir, "app/components/ui")
	if got := cfg.UIComponentsPath(); got != expected {
		t.Errorf("UIComponentsPath = %q, want %q", got, expected)
	}

	// Custom path via Paths.UI (Phase 14 primary config)
	cfg.Paths.UI = "components/ui"
	expected = filepath.Join(tmpDir, "components/ui")
	if got := cfg.UIComponentsPath(); got != expected {
		t.Errorf("UIComponentsPath custom = %q, want %q", got, expected)
	}
}
