package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestThinClient_MethodNotAllowedAndHead(t *testing.T) {
	s := New(DefaultServerConfig())

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/_vango/client.js", nil)
	s.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
	if rr.Header().Get("Allow") == "" {
		t.Fatal("expected Allow header for method not allowed")
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodHead, "/_vango/client.js", nil)
	s.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d", rr.Code, http.StatusOK)
	}
	if rr.Body.Len() != 0 {
		t.Fatal("expected empty body for HEAD")
	}
}

func TestETagMatches_WeakAndListFormats(t *testing.T) {
	if !etagMatches(`W/`+thinClientETag, thinClientETag) {
		t.Fatal("etagMatches should accept weak validator")
	}
	if !etagMatches(`"nope", `+thinClientETag, thinClientETag) {
		t.Fatal("etagMatches should match within a list")
	}
	if etagMatches(`"nope"`, thinClientETag) {
		t.Fatal("etagMatches matched unexpectedly")
	}
}

