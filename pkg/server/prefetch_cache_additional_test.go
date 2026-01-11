package server

import (
	"testing"
	"time"

	"github.com/vango-go/vango/pkg/vdom"
)

func TestPrefetchCache_DeleteClearLen(t *testing.T) {
	cfg := DefaultPrefetchConfig()
	cfg.TTL = 500 * time.Millisecond
	cfg.MaxEntries = 2

	cache := NewPrefetchCache(cfg)
	if cache.Len() != 0 {
		t.Fatalf("Len=%d, want 0", cache.Len())
	}

	cache.Set("/a", &vdom.VNode{Kind: vdom.KindText, Text: "a"})
	cache.Set("/b", &vdom.VNode{Kind: vdom.KindText, Text: "b"})
	if cache.Len() != 2 {
		t.Fatalf("Len=%d, want 2", cache.Len())
	}

	cache.Delete("/a")
	if cache.Get("/a") != nil {
		t.Fatal("Get(/a) != nil after Delete")
	}
	if cache.Len() != 1 {
		t.Fatalf("Len=%d, want 1", cache.Len())
	}

	cache.Clear()
	if cache.Len() != 0 {
		t.Fatalf("Len=%d, want 0 after Clear", cache.Len())
	}
}

func TestGlobalPrefetchSemaphore_LimitIsConfigurable(t *testing.T) {
	// Restore default to avoid cross-test interference.
	old := GlobalPrefetchSemaphore()
	t.Cleanup(func() { globalPrefetchManager.semaphore = old })

	SetGlobalPrefetchLimit(1)
	sem := GlobalPrefetchSemaphore()
	if !sem.Acquire() {
		t.Fatal("Acquire()=false, want true for first slot")
	}
	if sem.Acquire() {
		t.Fatal("Acquire()=true, want false at limit=1")
	}
	sem.Release()
	if !sem.Acquire() {
		t.Fatal("Acquire()=false after Release, want true")
	}
}

