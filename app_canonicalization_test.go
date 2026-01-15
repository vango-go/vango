package vango

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vango-go/vango/pkg/vdom"
)

func TestAppServeHTTP_CanonicalRedirects(t *testing.T) {
	app := New(Config{})

	tests := []struct {
		path         string
		wantLocation string
	}{
		{path: "/projects/", wantLocation: "/projects"},
		{path: "/a//b", wantLocation: "/a/b"},
		{path: "/a/./b", wantLocation: "/a/b"},
		{path: "/a/../b", wantLocation: "/b"},
		{path: "/users/?filter=active", wantLocation: "/users?filter=active"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, "http://example.com"+tt.path, nil)
		rr := httptest.NewRecorder()
		app.ServeHTTP(rr, req)

		if rr.Code != http.StatusPermanentRedirect {
			t.Fatalf("GET %s status = %d, want %d", tt.path, rr.Code, http.StatusPermanentRedirect)
		}
		if got := rr.Header().Get("Location"); got != tt.wantLocation {
			t.Fatalf("GET %s Location = %q, want %q", tt.path, got, tt.wantLocation)
		}
	}
}

func TestAppServeHTTP_CanonicalPathNoRedirect(t *testing.T) {
	app := New(Config{})
	app.Page("/projects", func(ctx Ctx) *VNode {
		return vdom.Text("ok")
	})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/projects", nil)
	rr := httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("GET /projects status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestAppServeHTTP_CanonicalizesAPIPaths(t *testing.T) {
	app := New(Config{})
	called := false
	app.API(http.MethodGet, "/api/echo", func(ctx Ctx) (any, error) {
		called = true
		return map[string]string{"ok": "true"}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/echo/", nil)
	rr := httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusPermanentRedirect {
		t.Fatalf("GET /api/echo/ status = %d, want %d", rr.Code, http.StatusPermanentRedirect)
	}
	if got := rr.Header().Get("Location"); got != "/api/echo" {
		t.Fatalf("GET /api/echo/ Location = %q, want %q", got, "/api/echo")
	}
	if called {
		t.Fatal("API handler called on canonical redirect")
	}
}

func TestAppServeHTTP_InvalidPathReturnsBadRequest(t *testing.T) {
	app := New(Config{})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/../secret", nil)
	rr := httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("GET /../secret status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestAppServeHTTP_DecodesPathAndParams(t *testing.T) {
	app := New(Config{})
	app.Page("/hello/:name", func(ctx Ctx) *VNode {
		ctx.SetHeader("X-Path", ctx.Path())
		ctx.SetHeader("X-Param", ctx.Param("name"))
		return vdom.Text("ok")
	})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/hello/hello%20world", nil)
	rr := httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("GET /hello/hello%%20world status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("X-Path"); got != "/hello/hello world" {
		t.Fatalf("X-Path = %q, want %q", got, "/hello/hello world")
	}
	if got := rr.Header().Get("X-Param"); got != "hello world" {
		t.Fatalf("X-Param = %q, want %q", got, "hello world")
	}
}
