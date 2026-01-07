package assets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManifestResolve(t *testing.T) {
	m := NewManifest()
	m.Set("vango.js", "vango.abc123.min.js")
	m.Set("styles.css", "styles.def456.css")

	tests := []struct {
		name     string
		source   string
		expected string
	}{
		{"found entry", "vango.js", "vango.abc123.min.js"},
		{"found entry css", "styles.css", "styles.def456.css"},
		{"missing entry returns original", "unknown.js", "unknown.js"},
		{"empty string returns empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.Resolve(tt.source)
			if got != tt.expected {
				t.Errorf("Resolve(%q) = %q, want %q", tt.source, got, tt.expected)
			}
		})
	}
}

func TestManifestHas(t *testing.T) {
	m := NewManifest()
	m.Set("vango.js", "vango.abc123.min.js")

	if !m.Has("vango.js") {
		t.Error("Has(vango.js) = false, want true")
	}
	if m.Has("unknown.js") {
		t.Error("Has(unknown.js) = true, want false")
	}
}

func TestManifestLen(t *testing.T) {
	m := NewManifest()
	if m.Len() != 0 {
		t.Errorf("Len() = %d, want 0", m.Len())
	}

	m.Set("a.js", "a.123.js")
	m.Set("b.js", "b.456.js")

	if m.Len() != 2 {
		t.Errorf("Len() = %d, want 2", m.Len())
	}
}

func TestManifestAll(t *testing.T) {
	m := NewManifest()
	m.Set("a.js", "a.123.js")
	m.Set("b.js", "b.456.js")

	all := m.All()
	if len(all) != 2 {
		t.Errorf("All() has %d entries, want 2", len(all))
	}
	if all["a.js"] != "a.123.js" {
		t.Errorf("All()[a.js] = %q, want a.123.js", all["a.js"])
	}

	// Verify it's a copy (modifying shouldn't affect original)
	all["c.js"] = "c.789.js"
	if m.Has("c.js") {
		t.Error("All() should return a copy, but modification affected original")
	}
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.json")

	content := `{"vango.js": "vango.abc123.min.js", "styles.css": "styles.def456.css"}`
	if err := os.WriteFile(manifestPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	m, err := Load(manifestPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got := m.Resolve("vango.js"); got != "vango.abc123.min.js" {
		t.Errorf("Resolve(vango.js) = %q, want vango.abc123.min.js", got)
	}
	if got := m.Resolve("styles.css"); got != "styles.def456.css" {
		t.Errorf("Resolve(styles.css) = %q, want styles.def456.css", got)
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/manifest.json")
	if err == nil {
		t.Error("Load() should return error for missing file")
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.json")

	if err := os.WriteFile(manifestPath, []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(manifestPath)
	if err == nil {
		t.Error("Load() should return error for invalid JSON")
	}
}

func TestResolverWithPrefix(t *testing.T) {
	m := NewManifest()
	m.Set("vango.js", "vango.abc123.min.js")
	m.Set("styles.css", "styles.def456.css")

	r := NewResolver(m, "/public/")

	tests := []struct {
		name     string
		source   string
		expected string
	}{
		{"found entry", "vango.js", "/public/vango.abc123.min.js"},
		{"found entry css", "styles.css", "/public/styles.def456.css"},
		{"missing entry gets prefix", "unknown.js", "/public/unknown.js"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.Asset(tt.source)
			if got != tt.expected {
				t.Errorf("Asset(%q) = %q, want %q", tt.source, got, tt.expected)
			}
		})
	}
}

func TestResolverWithoutPrefix(t *testing.T) {
	m := NewManifest()
	m.Set("vango.js", "vango.abc123.min.js")

	r := NewResolver(m, "")

	if got := r.Asset("vango.js"); got != "vango.abc123.min.js" {
		t.Errorf("Asset(vango.js) = %q, want vango.abc123.min.js", got)
	}
}

func TestPassthroughResolver(t *testing.T) {
	r := NewPassthroughResolver("/assets/")

	tests := []struct {
		name     string
		source   string
		expected string
	}{
		{"js file", "vango.js", "/assets/vango.js"},
		{"css file", "styles.css", "/assets/styles.css"},
		{"nested path", "images/logo.png", "/assets/images/logo.png"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.Asset(tt.source)
			if got != tt.expected {
				t.Errorf("Asset(%q) = %q, want %q", tt.source, got, tt.expected)
			}
		})
	}
}

func TestPassthroughResolverWithoutPrefix(t *testing.T) {
	r := NewPassthroughResolver("")

	if got := r.Asset("vango.js"); got != "vango.js" {
		t.Errorf("Asset(vango.js) = %q, want vango.js", got)
	}
}
