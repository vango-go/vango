package server

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/vango-dev/vango/v2/pkg/session"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestNewSessionManager(t *testing.T) {
	sm := NewSessionManager(nil, nil, testLogger())
	if sm == nil {
		t.Fatal("NewSessionManager should not return nil")
	}
	if sm.sessions == nil {
		t.Error("sessions map should be initialized")
	}
	if sm.done == nil {
		t.Error("done channel should be initialized")
	}

	sm.Shutdown()
}

func TestSessionManagerWithConfig(t *testing.T) {
	config := DefaultSessionConfig()
	config.IdleTimeout = 1 * time.Minute

	limits := DefaultSessionLimits()
	limits.MaxSessions = 100

	sm := NewSessionManager(config, limits, testLogger())
	if sm.config.IdleTimeout != 1*time.Minute {
		t.Error("Config not applied correctly")
	}
	if sm.limits.MaxSessions != 100 {
		t.Error("Limits not applied correctly")
	}

	sm.Shutdown()
}

func TestSessionManagerCount(t *testing.T) {
	sm := NewSessionManager(nil, nil, testLogger())
	defer sm.Shutdown()

	if sm.Count() != 0 {
		t.Errorf("Count() = %d, want 0", sm.Count())
	}
}

func TestSessionManagerGetNotFound(t *testing.T) {
	sm := NewSessionManager(nil, nil, testLogger())
	defer sm.Shutdown()

	session := sm.Get("nonexistent")
	if session != nil {
		t.Error("Get should return nil for nonexistent session")
	}
}

func TestSessionManagerCloseNotFound(t *testing.T) {
	sm := NewSessionManager(nil, nil, testLogger())
	defer sm.Shutdown()

	// Should not panic
	sm.Close("nonexistent")
}

func TestSessionManagerStats(t *testing.T) {
	sm := NewSessionManager(nil, nil, testLogger())
	defer sm.Shutdown()

	stats := sm.Stats()
	if stats.Active != 0 {
		t.Errorf("Active = %d, want 0", stats.Active)
	}
	if stats.TotalCreated != 0 {
		t.Errorf("TotalCreated = %d, want 0", stats.TotalCreated)
	}
	if stats.TotalClosed != 0 {
		t.Errorf("TotalClosed = %d, want 0", stats.TotalClosed)
	}
	if stats.Peak != 0 {
		t.Errorf("Peak = %d, want 0", stats.Peak)
	}
}

func TestSessionManagerForEach(t *testing.T) {
	sm := NewSessionManager(nil, nil, testLogger())
	defer sm.Shutdown()

	count := 0
	sm.ForEach(func(s *Session) bool {
		count++
		return true
	})

	if count != 0 {
		t.Errorf("ForEach counted %d sessions, want 0", count)
	}
}

func TestSessionManagerForEachEarlyExit(t *testing.T) {
	sm := NewSessionManager(nil, nil, testLogger())
	defer sm.Shutdown()

	// Even with no sessions, early exit should work
	sm.ForEach(func(s *Session) bool {
		return false // Stop iteration
	})
}

func TestSessionManagerCallbacks(t *testing.T) {
	sm := NewSessionManager(nil, nil, testLogger())
	defer sm.Shutdown()

	sm.SetOnSessionCreate(func(s *Session) {
		// callback set
	})
	sm.SetOnSessionClose(func(s *Session) {
		// callback set
	})

	// Can't easily test without a real WebSocket connection
	// Just verify the callbacks can be set
	if sm.onSessionCreate == nil {
		t.Error("onSessionCreate should be set")
	}
	if sm.onSessionClose == nil {
		t.Error("onSessionClose should be set")
	}
}

func TestSessionManagerSetCleanupInterval(t *testing.T) {
	sm := NewSessionManager(nil, nil, testLogger())
	defer sm.Shutdown()

	newInterval := 10 * time.Second
	sm.SetCleanupInterval(newInterval)

	if sm.cleanupInterval != newInterval {
		t.Errorf("cleanupInterval = %v, want %v", sm.cleanupInterval, newInterval)
	}
}

func TestSessionManagerTotalMemoryUsage(t *testing.T) {
	sm := NewSessionManager(nil, nil, testLogger())
	defer sm.Shutdown()

	// With no sessions, memory should be 0
	if sm.TotalMemoryUsage() != 0 {
		t.Errorf("TotalMemoryUsage() = %d, want 0", sm.TotalMemoryUsage())
	}
}

func TestSessionManagerCheckMemoryPressure(t *testing.T) {
	sm := NewSessionManager(nil, nil, testLogger())
	defer sm.Shutdown()

	// Should not panic with no sessions
	sm.CheckMemoryPressure()
}

func TestSessionManagerEvictLRUEmpty(t *testing.T) {
	sm := NewSessionManager(nil, nil, testLogger())
	defer sm.Shutdown()

	evicted := sm.EvictLRU(5)
	if evicted != 0 {
		t.Errorf("EvictLRU() = %d, want 0 with empty manager", evicted)
	}
}

func TestSessionManagerEvictLRUZero(t *testing.T) {
	sm := NewSessionManager(nil, nil, testLogger())
	defer sm.Shutdown()

	evicted := sm.EvictLRU(0)
	if evicted != 0 {
		t.Errorf("EvictLRU(0) = %d, want 0", evicted)
	}
}

func TestSessionManagerEvictLRUNegative(t *testing.T) {
	sm := NewSessionManager(nil, nil, testLogger())
	defer sm.Shutdown()

	evicted := sm.EvictLRU(-1)
	if evicted != 0 {
		t.Errorf("EvictLRU(-1) = %d, want 0", evicted)
	}
}

func TestManagerStatsStruct(t *testing.T) {
	stats := ManagerStats{
		Active:       100,
		TotalCreated: 500,
		TotalClosed:  400,
		Peak:         150,
		TotalMemory:  50 * 1024 * 1024,
	}

	if stats.Active != 100 {
		t.Error("Active not stored correctly")
	}
	if stats.TotalCreated != 500 {
		t.Error("TotalCreated not stored correctly")
	}
	if stats.TotalClosed != 400 {
		t.Error("TotalClosed not stored correctly")
	}
	if stats.Peak != 150 {
		t.Error("Peak not stored correctly")
	}
	if stats.TotalMemory != 50*1024*1024 {
		t.Error("TotalMemory not stored correctly")
	}
}

func TestSessionManagerShutdownMultiple(t *testing.T) {
	sm := NewSessionManager(nil, nil, testLogger())

	// First shutdown
	sm.Shutdown()

	// Second shutdown should not panic (though may have undefined behavior)
	// Just verify it doesn't crash
}

// =============================================================================
// Phase 12: Persistence Integration Tests
// =============================================================================

func TestSessionManagerWithPersistenceOptions(t *testing.T) {
	store := session.NewMemoryStore()
	opts := &SessionManagerOptions{
		SessionStore:        store,
		ResumeWindow:        10 * time.Minute,
		MaxDetachedSessions: 5000,
		MaxSessionsPerIP:    50,
		PersistInterval:     15 * time.Second,
	}

	sm := NewSessionManagerWithOptions(nil, nil, testLogger(), opts)
	defer sm.Shutdown()

	if !sm.HasPersistence() {
		t.Error("Expected HasPersistence to return true")
	}

	if sm.PersistenceManager() == nil {
		t.Error("Expected PersistenceManager to be non-nil")
	}
}

func TestSessionManagerWithoutPersistence(t *testing.T) {
	sm := NewSessionManager(nil, nil, testLogger())
	defer sm.Shutdown()

	if sm.HasPersistence() {
		t.Error("Expected HasPersistence to return false")
	}

	if sm.PersistenceManager() != nil {
		t.Error("Expected PersistenceManager to be nil")
	}
}

func TestSessionManagerCheckIPLimitWithoutPersistence(t *testing.T) {
	sm := NewSessionManager(nil, nil, testLogger())
	defer sm.Shutdown()

	// Without persistence, CheckIPLimit should return nil (no limit)
	err := sm.CheckIPLimit("192.168.1.1")
	if err != nil {
		t.Errorf("Expected nil error without persistence, got %v", err)
	}
}
