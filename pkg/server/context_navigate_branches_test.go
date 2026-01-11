package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCtx_Navigate_Branches(t *testing.T) {
	t.Run("ignored during prefetch", func(t *testing.T) {
		s := NewMockSession()
		c := NewTestContext(s).(*ctx)
		c.setMode(ModePrefetch)
		c.Navigate("/x")
		if _, _, has := c.PendingNavigation(); has {
			t.Fatal("pending navigation set during prefetch; want ignored")
		}
	})

	t.Run("invalid path rejected", func(t *testing.T) {
		s := NewMockSession()
		c := NewTestContext(s).(*ctx)
		c.Navigate("https://example.com") // not relative
		if _, _, has := c.PendingNavigation(); has {
			t.Fatal("pending navigation set for invalid path")
		}
	})

	t.Run("SSR fallback redirects", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/start", nil)
		c := &ctx{writer: rr, request: req, status: http.StatusOK}

		c.Navigate("/to", WithReplace())
		if rr.Code != http.StatusSeeOther {
			t.Fatalf("status=%d, want %d", rr.Code, http.StatusSeeOther)
		}
		if loc := rr.Header().Get("Location"); loc != "/to" {
			t.Fatalf("Location=%q, want %q", loc, "/to")
		}
	})
}

