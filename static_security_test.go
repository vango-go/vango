package vango

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestStaticServing_BlocksDirectoryTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	publicDir := filepath.Join(tmpDir, "public")
	if err := os.MkdirAll(publicDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	if err := os.WriteFile(filepath.Join(publicDir, "ok.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatalf("WriteFile ok.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "secret.txt"), []byte("secret"), 0o644); err != nil {
		t.Fatalf("WriteFile secret.txt: %v", err)
	}

	app := New(Config{
		Static: StaticConfig{
			Dir:    publicDir,
			Prefix: "/",
		},
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.com/ok.txt", nil)
	app.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /ok.txt status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Body.String(); got != "ok" {
		t.Fatalf("GET /ok.txt body = %q, want %q", got, "ok")
	}

	cases := []string{
		"/../secret.txt",
		"/%2e%2e/secret.txt",
		"/..//secret.txt",
	}
	for _, p := range cases {
		rr = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodGet, "http://example.com"+p, nil)
		app.ServeHTTP(rr, req)

		if rr.Code == http.StatusOK && strings.Contains(rr.Body.String(), "secret") {
			t.Fatalf("GET %s unexpectedly served secret content", p)
		}
		if rr.Code != http.StatusNotFound {
			t.Fatalf("GET %s status = %d, want %d", p, rr.Code, http.StatusNotFound)
		}
	}
}

func TestStaticServing_BlocksAbsolutePathEscape(t *testing.T) {
	tmpDir := t.TempDir()
	publicDir := filepath.Join(tmpDir, "public")
	if err := os.MkdirAll(publicDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	absSecretPath := filepath.Join(tmpDir, "abs-secret.txt")
	if err := os.WriteFile(absSecretPath, []byte("abs-secret"), 0o644); err != nil {
		t.Fatalf("WriteFile abs-secret.txt: %v", err)
	}

	app := New(Config{
		Static: StaticConfig{
			Dir:    publicDir,
			Prefix: "/static",
		},
	})

	// This is primarily exploitable on Unix-like systems where absolute paths
	// start with "/". The core traversal protection is covered in the other test.
	if runtime.GOOS == "windows" {
		t.Skip("absolute-path escape is OS-specific on Windows")
	}

	absURLPath := filepath.ToSlash(absSecretPath) // starts with "/"
	req := httptest.NewRequest(http.MethodGet, "http://example.com/static/"+absURLPath, nil)
	rr := httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	if rr.Code == http.StatusOK && strings.Contains(rr.Body.String(), "abs-secret") {
		t.Fatalf("unexpectedly served absolute-path content from %q", absSecretPath)
	}
	if rr.Code != http.StatusNotFound {
		t.Fatalf("GET /static/<abs> status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

