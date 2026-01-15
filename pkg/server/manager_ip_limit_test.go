package server

import (
	"log/slog"
	"testing"
	"time"
)

func TestSessionManagerCreate_EvictsOldestDetachedByIP(t *testing.T) {
	opts := &SessionManagerOptions{
		MaxSessionsPerIP: 1,
		EvictOnIPLimit:   true,
	}
	sm := NewSessionManagerWithOptions(DefaultSessionConfig(), DefaultSessionLimits(), slog.Default(), opts)
	t.Cleanup(func() { sm.Shutdown() })

	sess1, err := sm.Create(nil, "u1", "203.0.113.10")
	if err != nil {
		t.Fatalf("Create() sess1 error: %v", err)
	}
	sess1.detach("test", nil)
	sess1.DetachedAt = time.Now().Add(-time.Minute)
	sess1.LastActive = sess1.DetachedAt

	sess2, err := sm.Create(nil, "u2", "203.0.113.10")
	if err != nil {
		t.Fatalf("Create() sess2 error: %v", err)
	}

	if sm.Get(sess1.ID) != nil {
		t.Fatal("expected sess1 to be evicted")
	}
	if sm.Get(sess2.ID) == nil {
		t.Fatal("expected sess2 to remain")
	}
	if sm.Count() != 1 {
		t.Fatalf("Count()=%d, want 1 after eviction", sm.Count())
	}
}

func TestSessionManagerCreate_RejectsWhenNoDetached(t *testing.T) {
	opts := &SessionManagerOptions{
		MaxSessionsPerIP: 1,
		EvictOnIPLimit:   true,
	}
	sm := NewSessionManagerWithOptions(DefaultSessionConfig(), DefaultSessionLimits(), slog.Default(), opts)
	t.Cleanup(func() { sm.Shutdown() })

	if _, err := sm.Create(nil, "u1", "203.0.113.20"); err != nil {
		t.Fatalf("Create() sess1 error: %v", err)
	}

	if _, err := sm.Create(nil, "u2", "203.0.113.20"); err != ErrTooManySessionsFromIP {
		t.Fatalf("Create() error=%v, want %v", err, ErrTooManySessionsFromIP)
	}
}

func TestSessionManagerUpdateSessionIP_EvictsDetached(t *testing.T) {
	opts := &SessionManagerOptions{
		MaxSessionsPerIP: 1,
		EvictOnIPLimit:   true,
	}
	sm := NewSessionManagerWithOptions(DefaultSessionConfig(), DefaultSessionLimits(), slog.Default(), opts)
	t.Cleanup(func() { sm.Shutdown() })

	sess1, err := sm.Create(nil, "u1", "203.0.113.30")
	if err != nil {
		t.Fatalf("Create() sess1 error: %v", err)
	}
	sess2, err := sm.Create(nil, "u2", "203.0.113.31")
	if err != nil {
		t.Fatalf("Create() sess2 error: %v", err)
	}
	sess2.detach("test", nil)
	sess2.DetachedAt = time.Now().Add(-time.Minute)
	sess2.LastActive = sess2.DetachedAt

	if err := sm.UpdateSessionIP(sess1, "203.0.113.31"); err != nil {
		t.Fatalf("UpdateSessionIP() error: %v", err)
	}
	if sess1.IP != "203.0.113.31" {
		t.Fatalf("session IP=%q, want %q", sess1.IP, "203.0.113.31")
	}
	if sm.Get(sess2.ID) != nil {
		t.Fatal("expected sess2 to be evicted")
	}
}

func TestSessionManagerUpdateSessionIP_RejectsWhenFull(t *testing.T) {
	opts := &SessionManagerOptions{
		MaxSessionsPerIP: 1,
		EvictOnIPLimit:   false,
	}
	sm := NewSessionManagerWithOptions(DefaultSessionConfig(), DefaultSessionLimits(), slog.Default(), opts)
	t.Cleanup(func() { sm.Shutdown() })

	sess1, err := sm.Create(nil, "u1", "203.0.113.40")
	if err != nil {
		t.Fatalf("Create() sess1 error: %v", err)
	}
	if _, err := sm.Create(nil, "u2", "203.0.113.41"); err != nil {
		t.Fatalf("Create() sess2 error: %v", err)
	}

	if err := sm.UpdateSessionIP(sess1, "203.0.113.41"); err != ErrTooManySessionsFromIP {
		t.Fatalf("UpdateSessionIP() error=%v, want %v", err, ErrTooManySessionsFromIP)
	}
}
