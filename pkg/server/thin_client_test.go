package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestThinClientServed(t *testing.T) {
	s := New(DefaultServerConfig().WithDevMode())

	req := httptest.NewRequest(http.MethodGet, "/_vango/client.js", nil)
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/javascript") {
		t.Fatalf("Content-Type = %q, want application/javascript", ct)
	}

	if got := rr.Header().Get("ETag"); got == "" {
		t.Fatal("ETag should be set")
	}

	if rr.Body.Len() == 0 {
		t.Fatal("body should not be empty")
	}
}

func TestThinClientETagNotModified(t *testing.T) {
	s := New(DefaultServerConfig().WithDevMode())

	req1 := httptest.NewRequest(http.MethodGet, "/_vango/client.js", nil)
	rr1 := httptest.NewRecorder()
	s.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr1.Code, http.StatusOK)
	}
	etag := rr1.Header().Get("ETag")
	if etag == "" {
		t.Fatal("ETag should be set")
	}

	req2 := httptest.NewRequest(http.MethodGet, "/_vango/client.js", nil)
	req2.Header.Set("If-None-Match", etag)
	rr2 := httptest.NewRecorder()
	s.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusNotModified {
		t.Fatalf("status = %d, want %d", rr2.Code, http.StatusNotModified)
	}
	if rr2.Body.Len() != 0 {
		t.Fatal("body should be empty for 304 response")
	}
}

