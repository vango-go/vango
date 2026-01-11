package server

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/vango-go/vango/pkg/session"
	"github.com/vango-go/vango/pkg/urlparam"
)

func TestSessionManager_CreateAndCleanupExpired(t *testing.T) {
	clientConn, serverConn := newWebSocketPair(t)
	_ = clientConn // closed by t.Cleanup

	cfg := DefaultSessionConfig()
	cfg.IdleTimeout = 10 * time.Millisecond

	sm := NewSessionManager(cfg, DefaultSessionLimits(), slog.Default())
	t.Cleanup(func() { sm.Shutdown() })

	created := make(chan *Session, 1)
	closed := make(chan *Session, 1)
	sm.SetOnSessionCreate(func(s *Session) { created <- s })
	sm.SetOnSessionClose(func(s *Session) { closed <- s })

	sess, err := sm.Create(serverConn, "u1")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	select {
	case <-created:
	default:
		t.Fatal("expected onSessionCreate callback")
	}

	// Force expiry and run cleanup.
	sess.LastActive = time.Now().Add(-time.Hour)
	sm.cleanupExpired()

	deadline := time.Now().Add(2 * time.Second)
	for sm.Count() != 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if sm.Count() != 0 {
		t.Fatalf("Count()=%d, want 0 after cleanupExpired", sm.Count())
	}

	select {
	case <-closed:
		// ok
	case <-time.After(2 * time.Second):
		t.Fatal("expected onSessionClose callback")
	}
}

func TestSessionManager_cleanupExpired_UsesResumeWindowForDetachedSessions(t *testing.T) {
	cfg := DefaultSessionConfig()
	cfg.IdleTimeout = time.Hour

	opts := &SessionManagerOptions{
		ResumeWindow: 10 * time.Millisecond,
	}

	sm := NewSessionManagerWithOptions(cfg, DefaultSessionLimits(), slog.Default(), opts)
	t.Cleanup(func() { sm.Shutdown() })

	sess := newSession(nil, "", cfg, slog.Default())
	sess.detached.Store(true)
	sess.LastActive = time.Now().Add(-time.Hour)

	sm.mu.Lock()
	sm.sessions[sess.ID] = sess
	sm.mu.Unlock()

	sm.cleanupExpired()

	deadline := time.Now().Add(2 * time.Second)
	for sm.Count() != 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if sm.Count() != 0 {
		t.Fatalf("Count()=%d, want 0 after detached cleanupExpired", sm.Count())
	}
}

func TestSessionManager_PersistenceReconnectFromStore_RestoresSessionSkeleton(t *testing.T) {
	store := session.NewMemoryStore()
	opts := &SessionManagerOptions{
		SessionStore:    store,
		ResumeWindow:    1 * time.Minute,
		PersistInterval: 1 * time.Millisecond,
	}

	sm1 := NewSessionManagerWithOptions(DefaultSessionConfig(), DefaultSessionLimits(), slog.Default(), opts)
	t.Cleanup(func() { sm1.Shutdown() })
	if !sm1.HasPersistence() {
		t.Fatal("expected HasPersistence()=true")
	}

	orig := newSession(nil, "u1", DefaultSessionConfig(), slog.Default())
	orig.ID = "persisted-session"
	orig.IP = "127.0.0.1"
	orig.CurrentRoute = "/projects/123"

	// Persist a detached session.
	sm1.OnSessionDisconnect(orig)

	// Wait for async store.Save spawned by session.Manager.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		data, _ := store.Load(context.Background(), orig.ID)
		if data != nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	data, _ := store.Load(context.Background(), orig.ID)
	if data == nil {
		t.Fatal("expected session data to be saved in store")
	}

	// Simulate a server restart: a new server.SessionManager has no in-memory sessions.
	sm2 := NewSessionManagerWithOptions(DefaultSessionConfig(), DefaultSessionLimits(), slog.Default(), opts)
	t.Cleanup(func() { sm2.Shutdown() })

	restored, ok := sm2.OnSessionReconnect(orig.ID)
	if !ok || restored == nil {
		t.Fatalf("OnSessionReconnect ok=%v sess=%v, want ok=true and non-nil", ok, restored)
	}
	if restored.ID != orig.ID || restored.UserID != "u1" || restored.CurrentRoute != "/projects/123" {
		t.Fatalf("restored mismatch: ID=%q UserID=%q Route=%q", restored.ID, restored.UserID, restored.CurrentRoute)
	}

	if got := restored.owner.GetValue(urlparam.NavigatorKey); got == nil {
		t.Fatal("expected urlparam.NavigatorKey to be initialized on restored session")
	}
}

