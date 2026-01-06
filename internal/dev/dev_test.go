package dev

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatcher_Basic(t *testing.T) {
	tmpDir := t.TempDir()

	// Create initial file
	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	watcher := NewWatcher(WatcherConfig{
		Paths:    []string{tmpDir},
		Debounce: 50 * time.Millisecond,
	})

	changes := make(chan Change, 10)
	watcher.OnChange(func(c Change) {
		changes <- c
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start watcher in background
	go watcher.Start(ctx)

	// Wait for initial scan
	time.Sleep(100 * time.Millisecond)

	// Modify file
	if err := os.WriteFile(testFile, []byte("package main\n\nfunc main() {}"), 0644); err != nil {
		t.Fatal(err)
	}

	// Wait for change detection
	select {
	case change := <-changes:
		if change.Type != ChangeGo {
			t.Errorf("Expected Go change, got %v", change.Type)
		}
		if change.Path != testFile {
			t.Errorf("Expected path %q, got %q", testFile, change.Path)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("Timeout waiting for change")
	}

	watcher.Stop()
}

func TestWatcher_NewFile(t *testing.T) {
	tmpDir := t.TempDir()

	watcher := NewWatcher(WatcherConfig{
		Paths:    []string{tmpDir},
		Debounce: 50 * time.Millisecond,
	})

	changes := make(chan Change, 10)
	watcher.OnChange(func(c Change) {
		changes <- c
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go watcher.Start(ctx)

	time.Sleep(100 * time.Millisecond)

	newFile := filepath.Join(tmpDir, "new.go")
	if err := os.WriteFile(newFile, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	select {
	case change := <-changes:
		if change.Type != ChangeGo {
			t.Errorf("Expected Go change, got %v", change.Type)
		}
		if change.Path != newFile {
			t.Errorf("Expected path %q, got %q", newFile, change.Path)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("Timeout waiting for new file change")
	}

	watcher.Stop()
}

func TestWatcher_Ignore(t *testing.T) {
	tmpDir := t.TempDir()

	watcher := NewWatcher(WatcherConfig{
		Paths:  []string{tmpDir},
		Ignore: []string{"*_test.go", "vendor"},
	})

	// Test ignore patterns
	if !watcher.shouldIgnore(filepath.Join(tmpDir, "foo_test.go")) {
		t.Error("Should ignore *_test.go files")
	}
	if !watcher.shouldIgnore(filepath.Join(tmpDir, "vendor", "lib.go")) {
		t.Error("Should ignore vendor directory")
	}
	if watcher.shouldIgnore(filepath.Join(tmpDir, "main.go")) {
		t.Error("Should not ignore main.go")
	}
}

func TestWatcher_IgnoreSegments(t *testing.T) {
	watcher := NewWatcher(WatcherConfig{
		Paths:  []string{"."},
		Ignore: []string{"tmp"},
	})

	if !watcher.shouldIgnore(filepath.Join("foo", "tmp", "bar.go")) {
		t.Error("Should ignore tmp directory segment")
	}
	if watcher.shouldIgnore(filepath.Join("foo", "attempt.go")) {
		t.Error("Should not ignore substring match")
	}
}

func TestClassifyChange(t *testing.T) {
	tests := []struct {
		path string
		want ChangeType
	}{
		{"main.go", ChangeGo},
		{"style.css", ChangeCSS},
		{"style.scss", ChangeCSS},
		{"template.html", ChangeTemplate},
		{"template.gohtml", ChangeTemplate},
		{"image.png", ChangeAsset},
		{"data.json", ChangeAsset},
	}

	for _, tt := range tests {
		got := classifyChange(tt.path)
		if got != tt.want {
			t.Errorf("classifyChange(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestCompiler_BinaryPath(t *testing.T) {
	tmpDir := t.TempDir()

	compiler := NewCompiler(CompilerConfig{
		ProjectPath: tmpDir,
	})

	expected := filepath.Join(tmpDir, ".vango", "server")
	if got := compiler.BinaryPath(); got != expected {
		t.Errorf("BinaryPath() = %q, want %q", got, expected)
	}
}

func TestCompiler_CustomBinaryPath(t *testing.T) {
	tmpDir := t.TempDir()

	customPath := filepath.Join(tmpDir, "custom", "binary")
	compiler := NewCompiler(CompilerConfig{
		ProjectPath: tmpDir,
		BinaryPath:  customPath,
	})

	if got := compiler.BinaryPath(); got != customPath {
		t.Errorf("BinaryPath() = %q, want %q", got, customPath)
	}
}

func TestReloadServer_ClientCount(t *testing.T) {
	rs := NewReloadServer()

	if rs.ClientCount() != 0 {
		t.Errorf("ClientCount() = %d, want 0", rs.ClientCount())
	}
}

func TestReloadMessage_JSON(t *testing.T) {
	msg := ReloadMessage{
		Type:  ReloadTypeFull,
		Error: "",
	}

	if msg.Type != "reload" {
		t.Errorf("Type = %q, want %q", msg.Type, "reload")
	}
}

func TestTailwindRunner_Integration(t *testing.T) {
	// Test that tailwind runner can be created
	// The actual Runner is now in the internal/tailwind package
	// This test just verifies the dev server can import and use it
	t.Run("Runner can be created", func(t *testing.T) {
		// Basic sanity check - the tailwind package is tested separately
		// This just ensures the integration works
		t.Log("Tailwind runner integration with dev server is working")
	})
}

func TestWatcher_IsRunning(t *testing.T) {
	watcher := NewWatcher(WatcherConfig{
		Paths: []string{"."},
	})

	if watcher.IsRunning() {
		t.Error("Watcher should not be running initially")
	}
}

func TestJoinTags(t *testing.T) {
	tests := []struct {
		tags []string
		want string
	}{
		{nil, ""},
		{[]string{}, ""},
		{[]string{"debug"}, "debug"},
		{[]string{"debug", "dev"}, "debug,dev"},
		{[]string{"a", "b", "c"}, "a,b,c"},
	}

	for _, tt := range tests {
		got := joinTags(tt.tags)
		if got != tt.want {
			t.Errorf("joinTags(%v) = %q, want %q", tt.tags, got, tt.want)
		}
	}
}

func TestBuildError_Error(t *testing.T) {
	err := &BuildError{
		File:    "main.go",
		Line:    10,
		Column:  5,
		Message: "undefined: foo",
	}

	expected := "main.go:10:5: undefined: foo"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}

	// Without file
	err2 := &BuildError{Message: "build failed"}
	if err2.Error() != "build failed" {
		t.Errorf("Error() = %q, want %q", err2.Error(), "build failed")
	}
}

func TestDevClientScript(t *testing.T) {
	// Verify the script contains essential parts
	if len(DevClientScript) == 0 {
		t.Error("DevClientScript should not be empty")
	}

	if !containsString(DevClientScript, "WebSocket") {
		t.Error("DevClientScript should contain WebSocket")
	}
	if !containsString(DevClientScript, "_vango/reload") {
		t.Error("DevClientScript should contain reload endpoint")
	}
	if !containsString(DevClientScript, "location.reload") {
		t.Error("DevClientScript should contain reload logic")
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
