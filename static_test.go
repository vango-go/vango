package vango

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeStaticFile(t *testing.T, dir, name, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile %s: %v", name, err)
	}
	return path
}

func TestStaticServing_PrefixHandling(t *testing.T) {
	tmpDir := t.TempDir()
	publicDir := filepath.Join(tmpDir, "public")
	writeStaticFile(t, publicDir, "app.js", "ok")

	app := New(Config{
		Static: StaticConfig{
			Dir:    publicDir,
			Prefix: "/static",
		},
	})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/static/app.js", nil)
	rr := httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("GET /static/app.js status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Body.String(); got != "ok" {
		t.Fatalf("GET /static/app.js body = %q, want %q", got, "ok")
	}

	req = httptest.NewRequest(http.MethodGet, "http://example.com/app.js", nil)
	rr = httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("GET /app.js status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestStaticServing_MethodAndHeadHandling(t *testing.T) {
	tmpDir := t.TempDir()
	publicDir := filepath.Join(tmpDir, "public")
	writeStaticFile(t, publicDir, "app.js", "ok")

	app := New(Config{
		Static: StaticConfig{
			Dir:    publicDir,
			Prefix: "/",
		},
	})

	req := httptest.NewRequest(http.MethodPost, "http://example.com/app.js", nil)
	rr := httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /app.js status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}

	req = httptest.NewRequest(http.MethodHead, "http://example.com/app.js", nil)
	rr = httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("HEAD /app.js status = %d, want %d", rr.Code, http.StatusOK)
	}
	if rr.Body.Len() != 0 {
		t.Fatalf("HEAD /app.js body = %q, want empty", rr.Body.String())
	}
}

func TestStaticServing_CacheControlHeaders(t *testing.T) {
	tmpDir := t.TempDir()
	publicDir := filepath.Join(tmpDir, "public")
	writeStaticFile(t, publicDir, "app.a1b2c3d4.css", "fingerprinted")
	writeStaticFile(t, publicDir, "app.css", "plain")

	app := New(Config{
		Static: StaticConfig{
			Dir:          publicDir,
			Prefix:       "/",
			CacheControl: CacheControlProduction,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/app.a1b2c3d4.css", nil)
	rr := httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("GET /app.a1b2c3d4.css status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("Cache-Control = %q, want %q", got, "public, max-age=31536000, immutable")
	}

	req = httptest.NewRequest(http.MethodGet, "http://example.com/app.css", nil)
	rr = httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("GET /app.css status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Cache-Control"); got != "public, max-age=3600, must-revalidate" {
		t.Fatalf("Cache-Control = %q, want %q", got, "public, max-age=3600, must-revalidate")
	}

	app = New(Config{
		Static: StaticConfig{
			Dir:          publicDir,
			Prefix:       "/",
			CacheControl: CacheControlNone,
		},
	})
	req = httptest.NewRequest(http.MethodGet, "http://example.com/app.css", nil)
	rr = httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	if got := rr.Header().Get("Cache-Control"); got != "no-store, no-cache, must-revalidate" {
		t.Fatalf("Cache-Control = %q, want %q", got, "no-store, no-cache, must-revalidate")
	}
}

func TestIsFingerprinted(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{path: "app.a1b2c3d4.css", want: true},
		{path: "app.A1B2C3D4.css", want: true},
		{path: "app.12345678.css", want: true},
		{path: "app.1234567.css", want: false},
		{path: "app.zzzzzzzz.css", want: false},
		{path: "app.css", want: false},
	}

	for _, tc := range cases {
		if got := isFingerprinted(tc.path); got != tc.want {
			t.Fatalf("isFingerprinted(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestStaticRelPath_RejectsUnsafePaths(t *testing.T) {
	tmpDir := t.TempDir()
	publicDir := filepath.Join(tmpDir, "public")
	writeStaticFile(t, publicDir, "ok.txt", "ok")

	app := New(Config{
		Static: StaticConfig{
			Dir:    publicDir,
			Prefix: "/",
		},
	})

	cases := []string{
		"/\x00",
		"/foo\\bar",
		"/./secret",
		"/../secret",
		"/a/../b",
	}

	for _, path := range cases {
		if rel, ok := app.staticRelPath(path); ok {
			t.Fatalf("staticRelPath(%q) = %q, want reject", path, rel)
		}
	}
}

func TestStaticRelPath_RejectsLeadingSlashAfterPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	publicDir := filepath.Join(tmpDir, "public")
	writeStaticFile(t, publicDir, "ok.txt", "ok")

	app := New(Config{
		Static: StaticConfig{
			Dir:    publicDir,
			Prefix: "/static",
		},
	})

	if rel, ok := app.staticRelPath("/static//etc/passwd"); ok {
		t.Fatalf("staticRelPath returned %q, want reject", rel)
	}
}

func TestStaticServing_CustomHeaders(t *testing.T) {
	tmpDir := t.TempDir()
	publicDir := filepath.Join(tmpDir, "public")
	writeStaticFile(t, publicDir, "app.js", "ok")

	app := New(Config{
		Static: StaticConfig{
			Dir:    publicDir,
			Prefix: "/",
			Headers: map[string]string{
				"X-Static": "true",
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/app.js", nil)
	rr := httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("GET /app.js status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("X-Static"); got != "true" {
		t.Fatalf("X-Static = %q, want %q", got, "true")
	}
	if !strings.Contains(rr.Body.String(), "ok") {
		t.Fatalf("expected static response body, got %q", rr.Body.String())
	}
}
