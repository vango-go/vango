package session

import (
	"context"
	"log/slog"
	"testing"
	"time"
)

func TestManager_EvictionPolicy_LRU(t *testing.T) {
	cfg := DefaultManagerConfig()
	cfg.MaxDetachedSessions = 2
	cfg.EvictionPolicy = EvictionLRU
	cfg.CleanupInterval = 24 * time.Hour

	m := NewManager(nil, cfg, slog.Default())
	t.Cleanup(func() { _ = m.Shutdown(context.Background()) })

	now := time.Now()

	// Disconnect a, b; then touch a so that b becomes LRU; then disconnect c to evict b.
	if err := m.Register(&ManagedSession{ID: "a", IP: "1.1.1.1", CreatedAt: now.Add(-3 * time.Minute)}); err != nil {
		t.Fatalf("Register(a) error: %v", err)
	}
	if err := m.Register(&ManagedSession{ID: "b", IP: "1.1.1.1", CreatedAt: now.Add(-2 * time.Minute)}); err != nil {
		t.Fatalf("Register(b) error: %v", err)
	}
	m.OnDisconnect("a", []byte("a"))
	m.OnDisconnect("b", []byte("b"))

	m.Touch("a") // promote a to MRU (front); b should be evicted next.

	if err := m.Register(&ManagedSession{ID: "c", IP: "1.1.1.1", CreatedAt: now.Add(-1 * time.Minute)}); err != nil {
		t.Fatalf("Register(c) error: %v", err)
	}
	m.OnDisconnect("c", []byte("c"))

	if m.Get("b") != nil {
		t.Fatal("expected session b to be evicted under LRU")
	}
	if m.Get("a") == nil || m.Get("c") == nil {
		t.Fatal("expected sessions a and c to remain under LRU")
	}
}

func TestManager_EvictionPolicy_Oldest(t *testing.T) {
	cfg := DefaultManagerConfig()
	cfg.MaxDetachedSessions = 2
	cfg.EvictionPolicy = EvictionOldest
	cfg.CleanupInterval = 24 * time.Hour

	m := NewManager(nil, cfg, slog.Default())
	t.Cleanup(func() { _ = m.Shutdown(context.Background()) })

	now := time.Now()

	// Crafted so LRU victim differs from Oldest victim:
	// Disconnect order: a, b, c => LRU would evict a (least-recently-used).
	// CreatedAt oldest is b => Oldest should evict b.
	if err := m.Register(&ManagedSession{ID: "a", IP: "1.1.1.1", CreatedAt: now.Add(-1 * time.Hour)}); err != nil {
		t.Fatalf("Register(a) error: %v", err)
	}
	if err := m.Register(&ManagedSession{ID: "b", IP: "1.1.1.1", CreatedAt: now.Add(-3 * time.Hour)}); err != nil {
		t.Fatalf("Register(b) error: %v", err)
	}
	if err := m.Register(&ManagedSession{ID: "c", IP: "1.1.1.1", CreatedAt: now.Add(-2 * time.Hour)}); err != nil {
		t.Fatalf("Register(c) error: %v", err)
	}

	m.OnDisconnect("a", []byte("a"))
	m.OnDisconnect("b", []byte("b"))
	m.OnDisconnect("c", []byte("c")) // triggers eviction

	if m.Get("b") != nil {
		t.Fatal("expected session b to be evicted under Oldest policy")
	}
	if m.Get("a") == nil || m.Get("c") == nil {
		t.Fatal("expected sessions a and c to remain under Oldest policy")
	}
}

func TestManager_EvictionPolicy_Random_Deterministic(t *testing.T) {
	cfg := DefaultManagerConfig()
	cfg.MaxDetachedSessions = 2
	cfg.EvictionPolicy = EvictionRandom
	cfg.CleanupInterval = 24 * time.Hour

	m := NewManager(nil, cfg, slog.Default())
	t.Cleanup(func() { _ = m.Shutdown(context.Background()) })

	// Force eviction to pick index 1 from the front at eviction time.
	// When queue is [c, b, a] (front..back), idx=1 => b is evicted.
	m.randIntn = func(n int) int { return 1 }

	now := time.Now()
	for _, id := range []string{"a", "b", "c"} {
		if err := m.Register(&ManagedSession{ID: id, IP: "1.1.1.1", CreatedAt: now}); err != nil {
			t.Fatalf("Register(%s) error: %v", id, err)
		}
		m.OnDisconnect(id, []byte(id))
	}

	if m.Get("b") != nil {
		t.Fatal("expected session b to be evicted under deterministic Random policy")
	}
	if m.Get("a") == nil || m.Get("c") == nil {
		t.Fatal("expected sessions a and c to remain under deterministic Random policy")
	}
}

func TestManager_EvictionPolicy_Random_ClampsOutOfRangeIndex(t *testing.T) {
	cfg := DefaultManagerConfig()
	cfg.MaxDetachedSessions = 2
	cfg.EvictionPolicy = EvictionRandom
	cfg.CleanupInterval = 24 * time.Hour

	m := NewManager(nil, cfg, slog.Default())
	t.Cleanup(func() { _ = m.Shutdown(context.Background()) })

	// Force an out-of-range index; should clamp to the last element (back).
	m.randIntn = func(n int) int { return n + 1000 }

	now := time.Now()
	for _, id := range []string{"a", "b", "c"} {
		if err := m.Register(&ManagedSession{ID: id, IP: "1.1.1.1", CreatedAt: now}); err != nil {
			t.Fatalf("Register(%s) error: %v", id, err)
		}
		m.OnDisconnect(id, []byte(id))
	}

	if m.Get("a") != nil {
		t.Fatal("expected session a (back) to be evicted after Random index clamp")
	}
	if m.Get("b") == nil || m.Get("c") == nil {
		t.Fatal("expected sessions b and c to remain after Random index clamp")
	}
}

func TestManager_OnDisconnect_DeduplicatesDetachedQueue(t *testing.T) {
	cfg := DefaultManagerConfig()
	cfg.MaxDetachedSessions = 100
	cfg.EvictionPolicy = EvictionLRU
	cfg.CleanupInterval = 24 * time.Hour

	m := NewManager(nil, cfg, slog.Default())
	t.Cleanup(func() { _ = m.Shutdown(context.Background()) })

	if err := m.Register(&ManagedSession{ID: "a", IP: "1.1.1.1", CreatedAt: time.Now()}); err != nil {
		t.Fatalf("Register(a) error: %v", err)
	}

	m.OnDisconnect("a", []byte("first"))
	m.OnDisconnect("a", []byte("second"))

	m.mu.RLock()
	defer m.mu.RUnlock()
	if got := m.detachedQueue.Len(); got != 1 {
		t.Fatalf("detachedQueue.Len() got %d want 1", got)
	}
	if _, ok := m.detachedIndex["a"]; !ok {
		t.Fatal("detachedIndex missing entry for session a")
	}
}
