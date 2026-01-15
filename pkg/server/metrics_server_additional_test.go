package server

import (
	"log/slog"
	"testing"
	"time"
)

func TestServer_Metrics_ReflectsSessionManagerStats(t *testing.T) {
	s := New(DefaultServerConfig().WithDevMode())
	t.Cleanup(func() { s.Sessions().Shutdown() })

	metrics := s.Metrics()
	if metrics.ActiveSessions != 0 || metrics.TotalSessions != 0 {
		t.Fatalf("initial metrics sessions=%d total=%d, want 0/0", metrics.ActiveSessions, metrics.TotalSessions)
	}
	if metrics.CollectedAt.IsZero() {
		t.Fatal("CollectedAt is zero")
	}

	_, serverConn := newWebSocketPair(t)
	sess, err := s.Sessions().Create(serverConn, "u1", "127.0.0.1")
	if err != nil {
		t.Fatalf("Create session error: %v", err)
	}
	sess.logger = slog.Default()

	metrics = s.Metrics()
	if metrics.ActiveSessions != 1 {
		t.Fatalf("ActiveSessions=%d, want 1", metrics.ActiveSessions)
	}
	if metrics.TotalSessions != 1 {
		t.Fatalf("TotalSessions=%d, want 1", metrics.TotalSessions)
	}
	if metrics.CollectedAt.Before(time.Now().Add(-5 * time.Second)) {
		t.Fatal("CollectedAt seems too old")
	}
}
