package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vango-dev/vango/v2/pkg/features/store"
	"github.com/vango-dev/vango/v2/pkg/session"
	"github.com/vango-dev/vango/v2/pkg/urlparam"
	"github.com/vango-dev/vango/v2/pkg/vango"
	"github.com/vango-dev/vango/v2/pkg/vdom"
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

	// Phase 12: Session Persistence
	// persistenceManager handles session persistence, LRU eviction, and IP limits
	persistenceManager *session.Manager
	sessionStore       session.SessionStore
	resumeWindow       time.Duration
}

// SessionManagerOptions contains optional Phase 12 configuration.
type SessionManagerOptions struct {
	// SessionStore is the persistence backend for sessions.
	SessionStore session.SessionStore

	// ResumeWindow is how long detached sessions remain resumable.
	ResumeWindow time.Duration

	// MaxDetachedSessions is the maximum detached sessions before LRU eviction.
	MaxDetachedSessions int

	// MaxSessionsPerIP is the maximum sessions per IP address.
	MaxSessionsPerIP int

	// PersistInterval is how often to persist dirty sessions.
	PersistInterval time.Duration
}

// NewSessionManager creates a new SessionManager with the given configuration.
func NewSessionManager(config *SessionConfig, limits *SessionLimits, logger *slog.Logger) *SessionManager {
	return NewSessionManagerWithOptions(config, limits, logger, nil)
}

// NewSessionManagerWithOptions creates a SessionManager with Phase 12 persistence options.
func NewSessionManagerWithOptions(config *SessionConfig, limits *SessionLimits, logger *slog.Logger, opts *SessionManagerOptions) *SessionManager {
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
		resumeWindow:    5 * time.Minute, // Default
	}

	// Phase 12: Configure persistence if options provided
	if opts != nil {
		sm.sessionStore = opts.SessionStore
		if opts.ResumeWindow > 0 {
			sm.resumeWindow = opts.ResumeWindow
		}

		// Create persistence manager if store is provided
		if opts.SessionStore != nil {
			managerConfig := session.DefaultManagerConfig()
			if opts.MaxDetachedSessions > 0 {
				managerConfig.MaxDetachedSessions = opts.MaxDetachedSessions
			}
			if opts.MaxSessionsPerIP > 0 {
				managerConfig.MaxSessionsPerIP = opts.MaxSessionsPerIP
			}
			if opts.ResumeWindow > 0 {
				managerConfig.ResumeWindow = opts.ResumeWindow
			}
			if opts.PersistInterval > 0 {
				managerConfig.PersistInterval = opts.PersistInterval
			}

			sm.persistenceManager = session.NewManager(opts.SessionStore, managerConfig, logger)
		}
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
	sm.ShutdownWithContext(context.Background())
}

// ShutdownWithContext gracefully shuts down all sessions with context for timeout.
func (sm *SessionManager) ShutdownWithContext(ctx context.Context) error {
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

	// Phase 12: Persist all sessions before closing
	if sm.persistenceManager != nil {
		// Serialize and notify persistence manager for each session
		for _, sess := range sessions {
			data := sm.serializeSessionForPersistence(sess)
			if len(data) > 0 {
				sm.persistenceManager.OnDisconnect(sess.ID, data)
			}
		}

		// Shutdown persistence manager (persists all to store)
		if err := sm.persistenceManager.Shutdown(ctx); err != nil {
			sm.logger.Warn("persistence manager shutdown error", "error", err)
		}
	}

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

	return nil
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

// =============================================================================
// Phase 12: Session Persistence Integration
// =============================================================================

// CheckIPLimit checks if the IP has exceeded the session limit.
// Returns ErrTooManySessionsFromIP if the limit is exceeded.
// This should be called before creating a new session.
func (sm *SessionManager) CheckIPLimit(ip string) error {
	if sm.persistenceManager == nil {
		return nil // No limit checking without persistence manager
	}
	return sm.persistenceManager.CheckIPLimit(ip)
}

// OnSessionDisconnect is called when a WebSocket connection closes.
// It persists the session state for potential reconnection.
func (sm *SessionManager) OnSessionDisconnect(sess *Session) {
	if sm.persistenceManager == nil {
		return
	}

	// Serialize session for persistence
	data := sm.serializeSessionForPersistence(sess)

	// Register with persistence manager for LRU tracking
	managedSess := &session.ManagedSession{
		ID:         sess.ID,
		IP:         sess.IP,
		CreatedAt:  sess.CreatedAt,
		LastActive: sess.LastActive,
		UserID:     sess.UserID,
	}
	sm.persistenceManager.Register(managedSess)
	sm.persistenceManager.OnDisconnect(sess.ID, data)

	sm.logger.Debug("session disconnected and persisted",
		"session_id", sess.ID,
		"data_size", len(data))
}

// OnSessionReconnect attempts to restore a session after reconnection.
// Returns the restored session and true if successful, or nil and false if not found.
func (sm *SessionManager) OnSessionReconnect(sessionID string) (*Session, bool) {
	if sm.persistenceManager == nil {
		return nil, false
	}

	// Try to restore from persistence manager
	_, data, err := sm.persistenceManager.OnReconnect(sessionID)
	if err != nil || data == nil {
		return nil, false
	}

	// Deserialize and restore session
	sess := sm.restoreSessionFromPersistence(sessionID, data)
	if sess == nil {
		return nil, false
	}

	// Re-register the session
	sm.mu.Lock()
	sm.sessions[sess.ID] = sess
	sm.mu.Unlock()

	sm.logger.Debug("session reconnected from persistence",
		"session_id", sessionID)

	return sess, true
}

// serializeSessionForPersistence converts a Session to bytes for storage.
func (sm *SessionManager) serializeSessionForPersistence(sess *Session) []byte {
	ss := &session.SerializableSession{
		ID:         sess.ID,
		UserID:     sess.UserID,
		CreatedAt:  sess.CreatedAt,
		LastActive: sess.LastActive,
		Route:      sess.CurrentRoute,
	}

	data, err := session.Serialize(ss)
	if err != nil {
		sm.logger.Warn("failed to serialize session",
			"session_id", sess.ID,
			"error", err)
		return nil
	}
	return data
}

// restoreSessionFromPersistence creates a Session from persisted bytes.
// This creates a fully initialized session that can be resumed.
func (sm *SessionManager) restoreSessionFromPersistence(sessionID string, data []byte) *Session {
	ss, err := session.Deserialize(data)
	if err != nil {
		sm.logger.Warn("failed to deserialize session",
			"session_id", sessionID,
			"error", err)
		return nil
	}

	// Create fully initialized session
	sess := &Session{
		ID:           ss.ID,
		UserID:       ss.UserID,
		CreatedAt:    ss.CreatedAt,
		LastActive:   time.Now(),
		CurrentRoute: ss.Route,

		// Initialize component tracking (populated on RebuildHandlers)
		allComponents: make(map[*ComponentInstance]struct{}),
		handlers:      make(map[string]Handler),
		components:    make(map[string]*ComponentInstance),

		// Create fresh owner and HID generator
		owner:  vango.NewOwner(nil),
		hidGen: vdom.NewHIDGenerator(),

		// Initialize channels
		events:     make(chan *Event, sm.config.MaxEventQueue),
		renderCh:   make(chan struct{}, 1),
		dispatchCh: make(chan func(), sm.config.MaxEventQueue),
		done:       make(chan struct{}),

		config: sm.config,
		logger: sm.logger.With("session_id", ss.ID),
		data:   make(map[string]any),
	}

	// Initialize session-scoped store for SharedSignal support
	sess.owner.SetValue(store.SessionKey, store.NewSessionStore())

	// Initialize URL navigator for URLParam support
	navigator := urlparam.NewNavigator(sess.queueURLPatch)
	sess.owner.SetValue(urlparam.NavigatorKey, navigator)

	// Restore session data values
	if ss.Values != nil {
		values := make(map[string]any, len(ss.Values))
		for k, v := range ss.Values {
			var val any
			if err := json.Unmarshal(v, &val); err == nil {
				values[k] = val
			}
		}
		sess.RestoreData(values)
	}

	sm.logger.Debug("session restored from persistence",
		"session_id", sess.ID,
		"user_id", sess.UserID)

	return sess
}

// ResumeWindow returns the configured resume window duration.
// This is how long detached sessions remain resumable.
func (sm *SessionManager) ResumeWindow() time.Duration {
	if sm.resumeWindow == 0 {
		return 5 * time.Minute // Default
	}
	return sm.resumeWindow
}

// PersistenceManager returns the underlying persistence manager for advanced use.
// Returns nil if persistence is not configured.
func (sm *SessionManager) PersistenceManager() *session.Manager {
	return sm.persistenceManager
}

// HasPersistence returns true if session persistence is configured.
func (sm *SessionManager) HasPersistence() bool {
	return sm.persistenceManager != nil
}
