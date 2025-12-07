package server

import (
	"log/slog"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// SessionManager manages all active sessions.
// It handles session creation, lookup, cleanup, and lifecycle callbacks.
type SessionManager struct {
	// Sessions map protected by RWMutex
	sessions map[string]*Session
	mu       sync.RWMutex

	// Configuration
	config *SessionConfig

	// Cleanup (protected by cleanupMu)
	cleanupInterval time.Duration
	cleanupTicker   *time.Ticker
	cleanupMu       sync.Mutex
	done            chan struct{}
	cleanupDone     chan struct{} // Signals that cleanup goroutine has exited

	// Limits
	limits *SessionLimits

	// Metrics
	totalCreated atomic.Uint64
	totalClosed  atomic.Uint64
	peakSessions int

	// Callbacks
	onSessionCreate func(*Session)
	onSessionClose  func(*Session)

	// Logger
	logger *slog.Logger
}

// NewSessionManager creates a new SessionManager with the given configuration.
func NewSessionManager(config *SessionConfig, limits *SessionLimits, logger *slog.Logger) *SessionManager {
	if config == nil {
		config = DefaultSessionConfig()
	}
	if limits == nil {
		limits = DefaultSessionLimits()
	}
	if logger == nil {
		logger = slog.Default()
	}

	sm := &SessionManager{
		sessions:        make(map[string]*Session),
		config:          config,
		cleanupInterval: 30 * time.Second,
		done:            make(chan struct{}),
		cleanupDone:     make(chan struct{}),
		limits:          limits,
		logger:          logger.With("component", "session_manager"),
	}

	// Start cleanup goroutine
	go sm.cleanupLoop()

	return sm
}

// Create creates a new session for the given WebSocket connection.
func (sm *SessionManager) Create(conn *websocket.Conn, userID string) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check session limit
	if sm.limits.MaxSessions > 0 && len(sm.sessions) >= sm.limits.MaxSessions {
		return nil, ErrMaxSessionsReached
	}

	// Create session
	session := newSession(conn, userID, sm.config, sm.logger)

	// Register session
	sm.sessions[session.ID] = session
	sm.totalCreated.Add(1)

	// Track peak
	if len(sm.sessions) > sm.peakSessions {
		sm.peakSessions = len(sm.sessions)
	}

	// Callback
	if sm.onSessionCreate != nil {
		sm.onSessionCreate(session)
	}

	sm.logger.Info("session created",
		"session_id", session.ID,
		"user_id", userID,
		"active_sessions", len(sm.sessions))

	return session, nil
}

// Get retrieves a session by ID.
func (sm *SessionManager) Get(id string) *Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.sessions[id]
}

// Close closes a session by ID and removes it from the manager.
func (sm *SessionManager) Close(id string) {
	sm.mu.Lock()
	session, exists := sm.sessions[id]
	if exists {
		delete(sm.sessions, id)
	}
	sm.mu.Unlock()

	if exists {
		session.Close()
		sm.totalClosed.Add(1)

		if sm.onSessionClose != nil {
			sm.onSessionClose(session)
		}

		sm.logger.Info("session closed",
			"session_id", id,
			"active_sessions", sm.Count())
	}
}

// Count returns the number of active sessions.
func (sm *SessionManager) Count() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.sessions)
}

// cleanupLoop periodically removes expired sessions.
func (sm *SessionManager) cleanupLoop() {
	defer close(sm.cleanupDone)

	sm.cleanupMu.Lock()
	sm.cleanupTicker = time.NewTicker(sm.cleanupInterval)
	sm.cleanupMu.Unlock()

	defer func() {
		sm.cleanupMu.Lock()
		if sm.cleanupTicker != nil {
			sm.cleanupTicker.Stop()
		}
		sm.cleanupMu.Unlock()
	}()

	for {
		sm.cleanupMu.Lock()
		ticker := sm.cleanupTicker
		sm.cleanupMu.Unlock()

		select {
		case <-ticker.C:
			sm.cleanupExpired()
		case <-sm.done:
			return
		}
	}
}

// cleanupExpired removes sessions that have exceeded their idle timeout.
func (sm *SessionManager) cleanupExpired() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	var expired []string

	for id, session := range sm.sessions {
		if now.Sub(session.LastActive) > sm.config.IdleTimeout {
			expired = append(expired, id)
		}
	}

	for _, id := range expired {
		session := sm.sessions[id]
		delete(sm.sessions, id)

		go func(s *Session) {
			s.Close()
			sm.totalClosed.Add(1)
			if sm.onSessionClose != nil {
				sm.onSessionClose(s)
			}
		}(session)
	}

	if len(expired) > 0 {
		sm.logger.Info("cleaned up expired sessions",
			"count", len(expired),
			"remaining", len(sm.sessions))
	}
}

// EvictLRU evicts the least recently used sessions.
// This is called when memory pressure is high.
func (sm *SessionManager) EvictLRU(count int) int {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if count <= 0 || len(sm.sessions) == 0 {
		return 0
	}

	// Build list sorted by LastActive
	type sessionEntry struct {
		id      string
		session *Session
	}
	sessions := make([]sessionEntry, 0, len(sm.sessions))
	for id, s := range sm.sessions {
		sessions = append(sessions, sessionEntry{id: id, session: s})
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].session.LastActive.Before(sessions[j].session.LastActive)
	})

	// Evict oldest
	evicted := 0
	for i := 0; i < count && i < len(sessions); i++ {
		entry := sessions[i]
		delete(sm.sessions, entry.id)

		go func(s *Session) {
			s.Close()
			sm.totalClosed.Add(1)
			if sm.onSessionClose != nil {
				sm.onSessionClose(s)
			}
		}(entry.session)

		evicted++
	}

	if evicted > 0 {
		sm.logger.Info("evicted LRU sessions",
			"count", evicted,
			"remaining", len(sm.sessions))
	}

	return evicted
}

// Shutdown gracefully shuts down all sessions.
func (sm *SessionManager) Shutdown() {
	// Stop cleanup loop and wait for it to exit
	close(sm.done)
	<-sm.cleanupDone

	// Get all sessions
	sm.mu.Lock()
	sessions := make([]*Session, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		sessions = append(sessions, s)
	}
	sm.sessions = make(map[string]*Session)
	sm.mu.Unlock()

	// Close all sessions concurrently
	var wg sync.WaitGroup
	for _, session := range sessions {
		wg.Add(1)
		go func(s *Session) {
			defer wg.Done()
			s.Close()
			if sm.onSessionClose != nil {
				sm.onSessionClose(s)
			}
		}(session)
	}
	wg.Wait()

	sm.logger.Info("session manager shutdown",
		"closed_sessions", len(sessions))
}

// Stats returns aggregated session statistics.
func (sm *SessionManager) Stats() ManagerStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var totalMemory int64
	for _, s := range sm.sessions {
		totalMemory += s.MemoryUsage()
	}

	return ManagerStats{
		Active:       len(sm.sessions),
		TotalCreated: sm.totalCreated.Load(),
		TotalClosed:  sm.totalClosed.Load(),
		Peak:         sm.peakSessions,
		TotalMemory:  totalMemory,
	}
}

// ManagerStats contains aggregated session manager statistics.
type ManagerStats struct {
	Active       int
	TotalCreated uint64
	TotalClosed  uint64
	Peak         int
	TotalMemory  int64
}

// ForEach iterates over all sessions.
// The callback should not perform long-running operations as it holds the read lock.
func (sm *SessionManager) ForEach(fn func(*Session) bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for _, session := range sm.sessions {
		if !fn(session) {
			break
		}
	}
}

// SetOnSessionCreate sets the callback for session creation.
func (sm *SessionManager) SetOnSessionCreate(fn func(*Session)) {
	sm.onSessionCreate = fn
}

// SetOnSessionClose sets the callback for session close.
func (sm *SessionManager) SetOnSessionClose(fn func(*Session)) {
	sm.onSessionClose = fn
}

// SetCleanupInterval sets the cleanup interval.
func (sm *SessionManager) SetCleanupInterval(d time.Duration) {
	sm.cleanupMu.Lock()
	defer sm.cleanupMu.Unlock()
	sm.cleanupInterval = d
	if sm.cleanupTicker != nil {
		sm.cleanupTicker.Reset(d)
	}
}

// TotalMemoryUsage returns the total memory usage of all sessions.
func (sm *SessionManager) TotalMemoryUsage() int64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var total int64
	for _, s := range sm.sessions {
		total += s.MemoryUsage()
	}
	return total
}

// CheckMemoryPressure checks if memory usage exceeds limits and evicts if needed.
func (sm *SessionManager) CheckMemoryPressure() {
	total := sm.TotalMemoryUsage()

	if sm.limits.MaxTotalMemory > 0 && total > sm.limits.MaxTotalMemory {
		// Calculate how many sessions to evict (rough estimate)
		avgSize := total / int64(sm.Count())
		if avgSize == 0 {
			avgSize = 100 * 1024 // Default 100KB
		}
		excess := total - sm.limits.MaxTotalMemory
		evictCount := int(excess/avgSize) + 1

		sm.logger.Warn("memory pressure detected, evicting sessions",
			"total_memory", total,
			"limit", sm.limits.MaxTotalMemory,
			"evict_count", evictCount)

		sm.EvictLRU(evictCount)
	}
}
