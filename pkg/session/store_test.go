package session

import (
	"context"
	"testing"
	"time"
)

// TestMemoryStore tests the in-memory session store implementation.
func TestMemoryStore(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	ctx := context.Background()
	sessionID := "test-session-123"
	data := []byte(`{"id":"test-session-123","user_id":"user-1"}`)
	expiresAt := time.Now().Add(5 * time.Minute)

	// Test Save
	t.Run("Save", func(t *testing.T) {
		err := store.Save(ctx, sessionID, data, expiresAt)
		if err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	})

	// Test Load
	t.Run("Load", func(t *testing.T) {
		loaded, err := store.Load(ctx, sessionID)
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		if loaded == nil {
			t.Fatal("Load returned nil data")
		}
		if string(loaded) != string(data) {
			t.Errorf("Load returned wrong data: got %s, want %s", loaded, data)
		}
	})

	// Test Load non-existent
	t.Run("LoadNonExistent", func(t *testing.T) {
		loaded, err := store.Load(ctx, "non-existent")
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		if loaded != nil {
			t.Error("Load returned data for non-existent session")
		}
	})

	// Test Touch
	t.Run("Touch", func(t *testing.T) {
		newExpiry := time.Now().Add(10 * time.Minute)
		err := store.Touch(ctx, sessionID, newExpiry)
		if err != nil {
			t.Fatalf("Touch failed: %v", err)
		}

		// Verify session still exists
		loaded, err := store.Load(ctx, sessionID)
		if err != nil || loaded == nil {
			t.Error("Session not found after Touch")
		}
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		err := store.Delete(ctx, sessionID)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		// Verify session is gone
		loaded, err := store.Load(ctx, sessionID)
		if err != nil {
			t.Fatalf("Load after Delete failed: %v", err)
		}
		if loaded != nil {
			t.Error("Session still exists after Delete")
		}
	})

	// Test SaveAll
	t.Run("SaveAll", func(t *testing.T) {
		sessions := map[string]SessionData{
			"session-1": {Data: []byte(`{"id":"session-1"}`), ExpiresAt: expiresAt},
			"session-2": {Data: []byte(`{"id":"session-2"}`), ExpiresAt: expiresAt},
			"session-3": {Data: []byte(`{"id":"session-3"}`), ExpiresAt: expiresAt},
		}

		err := store.SaveAll(ctx, sessions)
		if err != nil {
			t.Fatalf("SaveAll failed: %v", err)
		}

		// Verify all sessions exist
		for id := range sessions {
			loaded, err := store.Load(ctx, id)
			if err != nil || loaded == nil {
				t.Errorf("Session %s not found after SaveAll", id)
			}
		}
	})
}

// TestMemoryStoreExpiry tests that expired sessions are not returned.
func TestMemoryStoreExpiry(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	ctx := context.Background()
	sessionID := "expiring-session"
	data := []byte(`{"test":"data"}`)

	// Save with very short expiry
	expiresAt := time.Now().Add(10 * time.Millisecond)
	err := store.Save(ctx, sessionID, data, expiresAt)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Wait for expiry
	time.Sleep(20 * time.Millisecond)

	// Load should return nil for expired session
	loaded, err := store.Load(ctx, sessionID)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded != nil {
		t.Error("Load returned data for expired session")
	}
}

// TestMemoryStoreConcurrency tests concurrent access to the store.
func TestMemoryStoreConcurrency(t *testing.T) {
	store := NewMemoryStore()
	defer store.Close()

	ctx := context.Background()
	expiresAt := time.Now().Add(5 * time.Minute)

	// Run concurrent operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			sessionID := string(rune('a' + id))
			data := []byte(`{"id":"` + sessionID + `"}`)

			for j := 0; j < 100; j++ {
				_ = store.Save(ctx, sessionID, data, expiresAt)
				_, _ = store.Load(ctx, sessionID)
				_ = store.Touch(ctx, sessionID, expiresAt)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
