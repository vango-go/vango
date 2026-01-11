package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCtx_PathAndDone(t *testing.T) {
	t.Run("from request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/p", nil)
		c := &ctx{request: req, stdCtx: context.Background()}
		if c.Path() != "/p" {
			t.Fatalf("Path()=%q, want %q", c.Path(), "/p")
		}
		if c.Done() != req.Context().Done() {
			t.Fatal("Done() did not delegate to request context")
		}
	})

	t.Run("from session route", func(t *testing.T) {
		s := NewMockSession()
		s.CurrentRoute = "/r"
		c := &ctx{session: s, stdCtx: context.Background()}
		if c.Path() != "/r" {
			t.Fatalf("Path()=%q, want %q", c.Path(), "/r")
		}
	})

	t.Run("from stdCtx", func(t *testing.T) {
		std, cancel := context.WithCancel(context.Background())
		c := &ctx{stdCtx: std}
		cancel()

		select {
		case <-c.Done():
			// ok
		case <-time.After(500 * time.Millisecond):
			t.Fatal("Done() did not close after cancel")
		}
	})
}
