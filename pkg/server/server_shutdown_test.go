package server

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestServer_WebSocketHandler_UpgradeErrorDoesNotPanic(t *testing.T) {
	s := New(DefaultServerConfig().WithDevMode())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/_vango/live", nil)

	// Not a websocket upgrade request; should just log/return.
	s.WebSocketHandler().ServeHTTP(rr, req)
}

func TestServer_Shutdown_StopsHTTPServerAndSessions(t *testing.T) {
	s := New(DefaultServerConfig().WithDevMode())

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen failed: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	s.httpServer = &http.Server{Handler: s}

	serveErr := make(chan error, 1)
	go func() { serveErr <- s.httpServer.Serve(ln) }()

	// Ensure Serve has started.
	time.Sleep(10 * time.Millisecond)

	if err := s.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error: %v", err)
	}

	select {
	case err := <-serveErr:
		// http.Server.Serve returns ErrServerClosed on graceful shutdown.
		if err != nil && err != http.ErrServerClosed {
			t.Fatalf("Serve returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Serve() to return")
	}
}
