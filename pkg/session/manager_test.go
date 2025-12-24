package session

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

// TestManagerRegister tests session registration.
func TestManagerRegister(t *testing.T) {
	store := NewMemoryStore()
	config := DefaultManagerConfig()
	config.CleanupInterval = 1 * time.Hour // Disable cleanup for tests
	manager := NewManager(store, config, slog.Default())
	defer manager.Shutdown(context.Background())

	sess := &ManagedSession{
		ID:         "session-1",
		IP:         "192.168.1.1",
		CreatedAt:  time.Now(),
		LastActive: time.Now(),
	}

	err := manager.Register(sess)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Verify session is registered
	got := manager.Get(sess.ID)
	if got == nil {
		t.Error("Session not found after Register")
	}
	if !got.Connected {
		t.Error("Session not marked as connected")
	}
}

// TestManagerIPLimit tests per-IP session limits.
func TestManagerIPLimit(t *testing.T) {
	store := NewMemoryStore()
	config := DefaultManagerConfig()
	config.MaxSessionsPerIP = 2
	config.CleanupInterval = 1 * time.Hour
	manager := NewManager(store, config, slog.Default())
	defer manager.Shutdown(context.Background())

	// Register up to the limit
	for i := 0; i < 2; i++ {
		sess := &ManagedSession{
			ID:         string(rune('a' + i)),
			IP:         "192.168.1.1",
			CreatedAt:  time.Now(),
			LastActive: time.Now(),
		}
		err := manager.Register(sess)
		if err != nil {
			t.Fatalf("Register %d failed: %v", i, err)
		}
	}

	// Try to exceed the limit
	sess := &ManagedSession{
		ID:         "c",
		IP:         "192.168.1.1",
		CreatedAt:  time.Now(),
		LastActive: time.Now(),
	}
	err := manager.Register(sess)
	if err != ErrTooManySessionsFromIP {
		t.Errorf("Expected ErrTooManySessionsFromIP, got %v", err)
	}

	// Different IP should work
	sess.IP = "192.168.1.2"
	err = manager.Register(sess)
	if err != nil {
		t.Errorf("Register with different IP failed: %v", err)
	}
}

// TestManagerDisconnectReconnect tests the disconnect/reconnect flow.
func TestManagerDisconnectReconnect(t *testing.T) {
	store := NewMemoryStore()
	config := DefaultManagerConfig()
	config.ResumeWindow = 5 * time.Minute
	config.CleanupInterval = 1 * time.Hour
	manager := NewManager(store, config, slog.Default())
	defer manager.Shutdown(context.Background())

	sess := &ManagedSession{
		ID:         "session-1",
		IP:         "192.168.1.1",
		CreatedAt:  time.Now(),
		LastActive: time.Now(),
	}

	err := manager.Register(sess)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Simulate disconnect
	serializedData := []byte(`{"id":"session-1","user_id":"user-1"}`)
	manager.OnDisconnect(sess.ID, serializedData)

	// Verify session is detached
	got := manager.Get(sess.ID)
	if got == nil {
		t.Fatal("Session not found after disconnect")
	}
	if got.Connected {
		t.Error("Session still marked as connected after disconnect")
	}
	if got.DisconnectedAt.IsZero() {
		t.Error("DisconnectedAt not set")
	}

	// Simulate reconnect
	restored, data, err := manager.OnReconnect(sess.ID)
	if err != nil {
		t.Fatalf("OnReconnect failed: %v", err)
	}
	if restored == nil {
		t.Fatal("OnReconnect returned nil session")
	}
	if !restored.Connected {
		t.Error("Session not marked as connected after reconnect")
	}
	if string(data) != string(serializedData) {
		t.Error("OnReconnect returned wrong data")
	}
}

// TestManagerLRUEviction tests LRU eviction of detached sessions.
func TestManagerLRUEviction(t *testing.T) {
	store := NewMemoryStore()
	config := DefaultManagerConfig()
	config.MaxDetachedSessions = 2
	config.CleanupInterval = 1 * time.Hour
	manager := NewManager(store, config, slog.Default())
	defer manager.Shutdown(context.Background())

	// Register and disconnect 3 sessions
	for i := 0; i < 3; i++ {
		sess := &ManagedSession{
			ID:         string(rune('a' + i)),
			IP:         "192.168.1.1",
			CreatedAt:  time.Now(),
			LastActive: time.Now(),
		}
		if err := manager.Register(sess); err != nil {
			t.Fatalf("Register %d failed: %v", i, err)
		}
		manager.OnDisconnect(sess.ID, []byte(`{}`))
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	stats := manager.Stats()
	if stats.Detached > config.MaxDetachedSessions {
		t.Errorf("Detached count %d exceeds limit %d", stats.Detached, config.MaxDetachedSessions)
	}

	// The oldest session ("a") should have been evicted
	got := manager.Get("a")
	if got != nil {
		t.Error("Oldest session should have been evicted")
	}

	// Newer sessions should still exist
	if manager.Get("b") == nil {
		t.Error("Session 'b' should still exist")
	}
	if manager.Get("c") == nil {
		t.Error("Session 'c' should still exist")
	}
}

// TestManagerStats tests statistics collection.
func TestManagerStats(t *testing.T) {
	store := NewMemoryStore()
	config := DefaultManagerConfig()
	config.CleanupInterval = 1 * time.Hour
	manager := NewManager(store, config, slog.Default())
	defer manager.Shutdown(context.Background())

	// Register some sessions
	for i := 0; i < 5; i++ {
		sess := &ManagedSession{
			ID:         string(rune('a' + i)),
			IP:         "192.168.1." + string(rune('1'+i)),
			CreatedAt:  time.Now(),
			LastActive: time.Now(),
		}
		manager.Register(sess)
	}

	// Disconnect some
	manager.OnDisconnect("a", []byte(`{}`))
	manager.OnDisconnect("b", []byte(`{}`))

	stats := manager.Stats()
	if stats.Total != 5 {
		t.Errorf("Total: got %d, want 5", stats.Total)
	}
	if stats.Connected != 3 {
		t.Errorf("Connected: got %d, want 3", stats.Connected)
	}
	if stats.Detached != 2 {
		t.Errorf("Detached: got %d, want 2", stats.Detached)
	}
	if stats.UniqueIPs != 5 {
		t.Errorf("UniqueIPs: got %d, want 5", stats.UniqueIPs)
	}
}

// TestManagerShutdown tests graceful shutdown with session persistence.
func TestManagerShutdown(t *testing.T) {
	store := NewMemoryStore()
	config := DefaultManagerConfig()
	config.CleanupInterval = 1 * time.Hour
	manager := NewManager(store, config, slog.Default())

	// Register and disconnect a session (with data)
	sess := &ManagedSession{
		ID:         "shutdown-test",
		IP:         "192.168.1.1",
		CreatedAt:  time.Now(),
		LastActive: time.Now(),
	}
	manager.Register(sess)
	manager.OnDisconnect(sess.ID, []byte(`{"important":"data"}`))

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := manager.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// Verify session was persisted to store
	data, err := store.Load(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("Load from store failed: %v", err)
	}
	if data == nil {
		t.Error("Session not persisted on shutdown")
	}
}
