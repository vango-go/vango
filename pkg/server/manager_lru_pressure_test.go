package server

import (
	"log/slog"
	"testing"
	"time"
)

func TestSessionManager_EvictLRUAndCheckMemoryPressure(t *testing.T) {
	cfg := DefaultSessionConfig()
	limits := DefaultSessionLimits()

	sm := NewSessionManager(cfg, limits, slog.Default())
	t.Cleanup(func() { sm.Shutdown() })

	// Seed sessions with ordered LastActive values.
	old := newSession(nil, "", cfg, slog.Default())
	old.LastActive = time.Now().Add(-2 * time.Hour)
	newer := newSession(nil, "", cfg, slog.Default())
	newer.LastActive = time.Now().Add(-1 * time.Hour)
	newest := newSession(nil, "", cfg, slog.Default())
	newest.LastActive = time.Now()

	sm.mu.Lock()
	sm.sessions[old.ID] = old
	sm.sessions[newer.ID] = newer
	sm.sessions[newest.ID] = newest
	sm.mu.Unlock()

	evicted := sm.EvictLRU(2)
	if evicted != 2 {
		t.Fatalf("EvictLRU=%d, want 2", evicted)
	}

	// Wait for eviction goroutines to close sessions and for map to reflect removals.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if sm.Count() == 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if sm.Count() != 1 {
		t.Fatalf("Count()=%d, want 1 after EvictLRU", sm.Count())
	}

	// Now ensure CheckMemoryPressure triggers an eviction when over the limit.
	// Add another session to ensure Count() > 1.
	extra := newSession(nil, "", cfg, slog.Default())
	sm.mu.Lock()
	sm.sessions[extra.ID] = extra
	sm.mu.Unlock()

	// Pick a limit that forces exactly one eviction based on the current total.
	total := sm.TotalMemoryUsage()
	if total <= 1 {
		t.Fatalf("unexpected TotalMemoryUsage=%d", total)
	}
	sm.limits.MaxTotalMemory = total - 1

	sm.CheckMemoryPressure()

	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if sm.Count() == 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if sm.Count() != 1 {
		t.Fatalf("Count()=%d, want 1 after CheckMemoryPressure eviction", sm.Count())
	}
}
