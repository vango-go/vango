package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vango-go/vango/internal/config"
)

func TestNew(t *testing.T) {
	cfg := config.New()
	cfg.Build.Minify = true
	cfg.Build.Target = "linux/amd64"
	cfg.Build.Tags = []string{"production"}

	builder := New(cfg, Options{})

	if !builder.options.Minify {
		t.Error("Minify should be true from config")
	}
	if builder.options.Target != "linux/amd64" {
		t.Errorf("Target = %q, want %q", builder.options.Target, "linux/amd64")
	}
	if len(builder.options.Tags) != 1 || builder.options.Tags[0] != "production" {
		t.Errorf("Tags = %v, want [production]", builder.options.Tags)
	}
}

func TestNew_OptionsOverride(t *testing.T) {
	cfg := config.New()
	cfg.Build.Minify = false

	builder := New(cfg, Options{
		Minify: true,
		Target: "darwin/arm64",
	})

	if !builder.options.Minify {
		t.Error("Minify should be true from options")
	}
	if builder.options.Target != "darwin/arm64" {
		t.Errorf("Target = %q, want %q", builder.options.Target, "darwin/arm64")
	}
}

func TestHashFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create test file
	content := []byte("hello world")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	hash, err := hashFile(testFile)
	if err != nil {
		t.Fatalf("hashFile error: %v", err)
	}

	if len(hash) != 64 { // SHA256 produces 64 hex characters
		t.Errorf("Hash length = %d, want 64", len(hash))
	}

	// Hash should be consistent
	hash2, _ := hashFile(testFile)
	if hash != hash2 {
		t.Error("Hash should be consistent")
	}

	// Different content should produce different hash
	os.WriteFile(testFile, []byte("different content"), 0644)
	hash3, _ := hashFile(testFile)
	if hash == hash3 {
		t.Error("Different content should produce different hash")
	}
}

func TestHashFile_NotFound(t *testing.T) {
	_, err := hashFile("/nonexistent/file.txt")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "src.txt")
	dstFile := filepath.Join(tmpDir, "dst.txt")

	// Create source file
	content := []byte("test content")
	if err := os.WriteFile(srcFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	// Copy
	if err := copyFile(srcFile, dstFile); err != nil {
		t.Fatalf("copyFile error: %v", err)
	}

	// Verify
	copied, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}

	if string(copied) != string(content) {
		t.Errorf("Content = %q, want %q", string(copied), string(content))
	}
}

func TestCopyFile_SrcNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	err := copyFile("/nonexistent/file.txt", filepath.Join(tmpDir, "dst.txt"))
	if err == nil {
		t.Error("Expected error for nonexistent source")
	}
}

func TestBuilder_Clean(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "dist")

	// Create output directory with files
	os.MkdirAll(filepath.Join(outputDir, "public"), 0755)
	os.WriteFile(filepath.Join(outputDir, "server"), []byte("binary"), 0755)
	os.WriteFile(filepath.Join(outputDir, "public", "vango.js"), []byte("js"), 0644)

	// Create config
	cfg := config.New()
	cfg.Build.Output = "dist"
	cfg.SaveTo(filepath.Join(tmpDir, "vango.json"))

	builder := New(cfg, Options{})

	// Clean
	if err := builder.Clean(); err != nil {
		t.Fatalf("Clean error: %v", err)
	}

	// Verify
	if _, err := os.Stat(outputDir); !os.IsNotExist(err) {
		t.Error("Output directory should be removed")
	}
}

func TestBuilder_Progress(t *testing.T) {
	cfg := config.New()

	var steps []string
	builder := New(cfg, Options{
		OnProgress: func(step string) {
			steps = append(steps, step)
		},
	})

	builder.progress("Step 1")
	builder.progress("Step 2")

	if len(steps) != 2 {
		t.Errorf("Steps = %v, want 2 steps", steps)
	}
	if steps[0] != "Step 1" {
		t.Errorf("First step = %q, want %q", steps[0], "Step 1")
	}
}

func TestResult_Fields(t *testing.T) {
	result := &Result{
		Binary:     "/path/to/server",
		Public:     "/path/to/public",
		ClientSize: 12345,
		CSSSize:    5678,
		Manifest: map[string]string{
			"vango.js": "vango.abc123.js",
		},
	}

	if result.Binary != "/path/to/server" {
		t.Errorf("Binary = %q", result.Binary)
	}
	if result.ClientSize != 12345 {
		t.Errorf("ClientSize = %d", result.ClientSize)
	}
	if result.Manifest["vango.js"] != "vango.abc123.js" {
		t.Errorf("Manifest mismatch")
	}
}

func TestOptions_Defaults(t *testing.T) {
	opts := Options{}

	if opts.Minify {
		t.Error("Minify should be false by default")
	}
	if opts.SourceMaps {
		t.Error("SourceMaps should be false by default")
	}
	if opts.Target != "" {
		t.Error("Target should be empty by default")
	}
}
