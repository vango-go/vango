package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPhase14ConfigSchema tests the Phase 14 configuration schema.
func TestPhase14ConfigSchema(t *testing.T) {
	t.Run("New creates config with Phase 14 defaults", func(t *testing.T) {
		cfg := New()

		// Check top-level fields
		if cfg.Version != "0.1.0" {
			t.Errorf("expected Version '0.1.0', got %q", cfg.Version)
		}
		if cfg.Port != DefaultPort {
			t.Errorf("expected Port %d, got %d", DefaultPort, cfg.Port)
		}

		// Check Paths config
		if cfg.Paths.Routes != "app/routes" {
			t.Errorf("expected Paths.Routes 'app/routes', got %q", cfg.Paths.Routes)
		}
		if cfg.Paths.Components != "app/components" {
			t.Errorf("expected Paths.Components 'app/components', got %q", cfg.Paths.Components)
		}
		if cfg.Paths.UI != "app/components/ui" {
			t.Errorf("expected Paths.UI 'app/components/ui', got %q", cfg.Paths.UI)
		}
		if cfg.Paths.Store != "app/store" {
			t.Errorf("expected Paths.Store 'app/store', got %q", cfg.Paths.Store)
		}
		if cfg.Paths.Middleware != "app/middleware" {
			t.Errorf("expected Paths.Middleware 'app/middleware', got %q", cfg.Paths.Middleware)
		}

		// Check Static config
		if cfg.Static.Dir != "public" {
			t.Errorf("expected Static.Dir 'public', got %q", cfg.Static.Dir)
		}
		if cfg.Static.Prefix != "/" {
			t.Errorf("expected Static.Prefix '/', got %q", cfg.Static.Prefix)
		}

		// Check Dev config
		if cfg.Dev.HotReload != true {
			t.Error("expected Dev.HotReload true")
		}
		if len(cfg.Dev.Watch) == 0 {
			t.Error("expected Dev.Watch to have default values")
		}

		// Check Build config
		if cfg.Build.MinifyAssets != true {
			t.Error("expected Build.MinifyAssets true")
		}
		if cfg.Build.StripSymbols != true {
			t.Error("expected Build.StripSymbols true")
		}

		// Check Session config
		if cfg.Session.ResumeWindow != "30s" {
			t.Errorf("expected Session.ResumeWindow '30s', got %q", cfg.Session.ResumeWindow)
		}
	})

	t.Run("LoadFile applies defaults", func(t *testing.T) {
		// Create minimal config file
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "vango.json")
		content := `{
  "name": "test-app"
}`
		if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadFile(cfgPath)
		if err != nil {
			t.Fatal(err)
		}

		// Check that defaults were applied
		if cfg.Port != DefaultPort {
			t.Errorf("expected default port %d, got %d", DefaultPort, cfg.Port)
		}
		if cfg.Paths.Routes != "app/routes" {
			t.Errorf("expected default Paths.Routes, got %q", cfg.Paths.Routes)
		}
		if cfg.Static.Prefix != "/" {
			t.Errorf("expected default Static.Prefix, got %q", cfg.Static.Prefix)
		}
	})

	t.Run("paths prefer new Paths over legacy fields", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "vango.json")
		content := `{
  "name": "test-app",
  "paths": {
    "routes": "custom/routes"
  },
  "routes": "legacy/routes"
}`
		if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := LoadFile(cfgPath)
		if err != nil {
			t.Fatal(err)
		}

		// New Paths config should take precedence
		if !strings.Contains(cfg.RoutesPath(), "custom/routes") {
			t.Errorf("expected RoutesPath to use new Paths config, got %q", cfg.RoutesPath())
		}
	})
}

// TestPathHelpers tests the path helper methods.
func TestPathHelpers(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "vango.json")
	content := `{
  "name": "test-app",
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
  }
}`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("RoutesPath returns absolute path", func(t *testing.T) {
		path := cfg.RoutesPath()
		if !filepath.IsAbs(path) {
			t.Errorf("expected absolute path, got %q", path)
		}
		if !strings.HasSuffix(path, "app/routes") {
			t.Errorf("expected path to end with 'app/routes', got %q", path)
		}
	})

	t.Run("ComponentsPath returns absolute path", func(t *testing.T) {
		path := cfg.ComponentsPath()
		if !filepath.IsAbs(path) {
			t.Errorf("expected absolute path, got %q", path)
		}
		if !strings.HasSuffix(path, "app/components") {
			t.Errorf("expected path to end with 'app/components', got %q", path)
		}
	})

	t.Run("StorePath returns absolute path", func(t *testing.T) {
		path := cfg.StorePath()
		if !filepath.IsAbs(path) {
			t.Errorf("expected absolute path, got %q", path)
		}
		if !strings.HasSuffix(path, "app/store") {
			t.Errorf("expected path to end with 'app/store', got %q", path)
		}
	})

	t.Run("MiddlewarePath returns absolute path", func(t *testing.T) {
		path := cfg.MiddlewarePath()
		if !filepath.IsAbs(path) {
			t.Errorf("expected absolute path, got %q", path)
		}
		if !strings.HasSuffix(path, "app/middleware") {
			t.Errorf("expected path to end with 'app/middleware', got %q", path)
		}
	})

	t.Run("StaticPrefix returns correct prefix", func(t *testing.T) {
		prefix := cfg.StaticPrefix()
		if prefix != "/" {
			t.Errorf("expected prefix '/', got %q", prefix)
		}
	})
}

// TestConfigSaveLoad tests round-trip save and load.
func TestConfigSaveLoad(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "vango.json")

	// Create and save config
	cfg := New()
	cfg.Name = "test-project"
	cfg.Version = "1.0.0"
	cfg.Paths.Routes = "custom/routes"
	cfg.Static.Prefix = "/static"

	if err := cfg.SaveTo(cfgPath); err != nil {
		t.Fatal(err)
	}

	// Load config and verify
	loaded, err := LoadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Name != "test-project" {
		t.Errorf("expected Name 'test-project', got %q", loaded.Name)
	}
	if loaded.Version != "1.0.0" {
		t.Errorf("expected Version '1.0.0', got %q", loaded.Version)
	}
	if loaded.Paths.Routes != "custom/routes" {
		t.Errorf("expected Paths.Routes 'custom/routes', got %q", loaded.Paths.Routes)
	}
	if loaded.Static.Prefix != "/static" {
		t.Errorf("expected Static.Prefix '/static', got %q", loaded.Static.Prefix)
	}
}
