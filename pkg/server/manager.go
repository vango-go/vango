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
	"github.com/vango-go/vango/pkg/features/store"
	"github.com/vango-go/vango/pkg/session"
	"github.com/vango-go/vango/pkg/urlparam"
	"github.com/vango-go/vango/pkg/vango"
	"github.com/vango-go/vango/pkg/vdom"
)

// SessionManager manages all active sessions.
// It handles session creation, lookup, cleanup, and lifecycle callbacks.
type SessionManager struct {
	// Sessions map protected by RWMutex
	sessions map[string]*Session
	mu       sync.RWMutex

	// Session count per IP address (protected by mu)
	sessionsByIP map[string]int

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

	// Per-IP limits
	maxSessionsPerIP int
	evictOnIPLimit   bool

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

	// EvictOnIPLimit controls whether to evict the oldest detached session
	// for a full IP bucket instead of rejecting new sessions.
	EvictOnIPLimit bool

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
		sessionsByIP:    make(map[string]int),
		config:          config,
		cleanupInterval: 30 * time.Second,
		done:            make(chan struct{}),
		cleanupDone:     make(chan struct{}),
		limits:          limits,
		logger:          logger.With("component", "session_manager"),
		resumeWindow:    5 * time.Minute, // Default
		evictOnIPLimit:  true,
	}

	// Phase 12: Configure persistence if options provided
	if opts != nil {
		sm.sessionStore = opts.SessionStore
		if opts.ResumeWindow > 0 {
			sm.resumeWindow = opts.ResumeWindow
		}
		if opts.MaxSessionsPerIP > 0 {
			sm.maxSessionsPerIP = opts.MaxSessionsPerIP
		}
		if opts.EvictOnIPLimit || opts.MaxSessionsPerIP > 0 {
			sm.evictOnIPLimit = opts.EvictOnIPLimit
		}

		// Create persistence manager if store is provided
		if opts.SessionStore != nil {
			managerConfig := session.DefaultManagerConfig()
			if opts.MaxDetachedSessions > 0 {
				managerConfig.MaxDetachedSessions = opts.MaxDetachedSessions
			}
			managerConfig.MaxSessionsPerIP = 0
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
func (sm *SessionManager) Create(conn *websocket.Conn, userID, ip string) (*Session, error) {
	sm.mu.Lock()

	// Check session limit
	if sm.limits.MaxSessions > 0 && len(sm.sessions) >= sm.limits.MaxSessions {
		sm.mu.Unlock()
		return nil, ErrMaxSessionsReached
	}

	evicted, err := sm.ensureIPCapacityLocked(ip, "")
	if err != nil {
		sm.mu.Unlock()
		sm.closeEvictedSessions(evicted)
		return nil, err
	}

	// Create session
	session := newSession(conn, userID, sm.config, sm.logger)
	session.IP = ip
	session.setOnDetach(sm.OnSessionDisconnect)

	// Register session
	sm.sessions[session.ID] = session
	sm.trackSessionLocked(session, true)

	sm.mu.Unlock()

	sm.closeEvictedSessions(evicted)

	sm.logger.Info("session created",
		"session_id", session.ID,
		"user_id", userID,
		"active_sessions", sm.Count())

	return session, nil
}

func (sm *SessionManager) trackSessionLocked(session *Session, countCreated bool) {
	if session == nil {
		return
	}
	if session.IP != "" {
		sm.sessionsByIP[session.IP]++
	}
	if countCreated {
		sm.totalCreated.Add(1)
	}
	if len(sm.sessions) > sm.peakSessions {
		sm.peakSessions = len(sm.sessions)
	}
	if countCreated && sm.onSessionCreate != nil {
		sm.onSessionCreate(session)
	}
}

func (sm *SessionManager) removeSessionLocked(id string) *Session {
	session, exists := sm.sessions[id]
	if !exists {
		return nil
	}
	delete(sm.sessions, id)
	if session.IP != "" {
		sm.sessionsByIP[session.IP]--
		if sm.sessionsByIP[session.IP] <= 0 {
			delete(sm.sessionsByIP, session.IP)
		}
	}
	return session
}

func (sm *SessionManager) ensureIPCapacityLocked(ip, excludeID string) ([]*Session, error) {
	if sm.maxSessionsPerIP <= 0 || ip == "" {
		return nil, nil
	}

	count := sm.sessionsByIP[ip]
	if excludeID != "" {
		if sess, ok := sm.sessions[excludeID]; ok && sess.IP == ip {
			count--
		}
	}

	if count < sm.maxSessionsPerIP {
		return nil, nil
	}
	if !sm.evictOnIPLimit {
		return nil, ErrTooManySessionsFromIP
	}

	evicted := sm.evictOldestDetachedByIPLocked(ip, excludeID)
	if evicted == nil {
		return nil, ErrTooManySessionsFromIP
	}

	count = sm.sessionsByIP[ip]
	if excludeID != "" {
		if sess, ok := sm.sessions[excludeID]; ok && sess.IP == ip {
			count--
		}
	}

	if count >= sm.maxSessionsPerIP {
		return []*Session{evicted}, ErrTooManySessionsFromIP
	}

	return []*Session{evicted}, nil
}

func (sm *SessionManager) evictOldestDetachedByIPLocked(ip, excludeID string) *Session {
	var oldest *Session
	var oldestAt time.Time

	for id, sess := range sm.sessions {
		if sess == nil || sess.IP != ip || !sess.IsDetached() || id == excludeID {
			continue
		}
		detachedAt := sess.DetachedAt
		if detachedAt.IsZero() {
			detachedAt = sess.LastActive
		}
		if oldest == nil || detachedAt.Before(oldestAt) {
			oldest = sess
			oldestAt = detachedAt
		}
	}

	if oldest == nil {
		return nil
	}

	sm.removeSessionLocked(oldest.ID)
	return oldest
}

func (sm *SessionManager) closeEvictedSessions(sessions []*Session) {
	for _, sess := range sessions {
		if sess == nil {
			continue
		}
		go func(s *Session) {
			s.Close()
			sm.totalClosed.Add(1)
			if sm.onSessionClose != nil {
				sm.onSessionClose(s)
			}
		}(sess)
	}
}

// Get retrieves a session by ID.
func (sm *SessionManager) Get(id string) *Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.sessions[id]
}

// UpdateSessionIP updates the session's IP address and enforces per-IP limits.
// Returns ErrTooManySessionsFromIP if the new IP bucket is full.
func (sm *SessionManager) UpdateSessionIP(session *Session, newIP string) error {
	if session == nil {
		return nil
	}
	if newIP == "" {
		newIP = session.IP
	}

	sm.mu.Lock()
	evicted, err := sm.ensureIPCapacityLocked(newIP, session.ID)
	if err != nil {
		sm.mu.Unlock()
		sm.closeEvictedSessions(evicted)
		return err
	}

	oldIP := session.IP
	if newIP != oldIP {
		if oldIP != "" {
			sm.sessionsByIP[oldIP]--
			if sm.sessionsByIP[oldIP] <= 0 {
				delete(sm.sessionsByIP, oldIP)
			}
		}
		if newIP != "" {
			sm.sessionsByIP[newIP]++
		}
		session.IP = newIP
	}
	sm.mu.Unlock()

	sm.closeEvictedSessions(evicted)
	return nil
}

// Close closes a session by ID and removes it from the manager.
func (sm *SessionManager) Close(id string) {
	sm.mu.Lock()
	session := sm.removeSessionLocked(id)
	sm.mu.Unlock()

	if session != nil {
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
			sm.CheckMemoryPressure()
		case <-sm.done:
			return
		}
	}
}

// cleanupExpired removes sessions that have exceeded their idle timeout.
func (sm *SessionManager) cleanupExpired() {
	sm.mu.Lock()

	now := time.Now()
	var expired []string

	for id, session := range sm.sessions {
		timeout := sm.config.IdleTimeout
		if session != nil && session.IsDetached() {
			// Detached sessions are only kept for ResumeWindow.
			timeout = sm.ResumeWindow()
		}
		if now.Sub(session.LastActive) > timeout {
			expired = append(expired, id)
		}
	}

	var toClose []*Session
	for _, id := range expired {
		if session := sm.removeSessionLocked(id); session != nil {
			toClose = append(toClose, session)
		}
	}
	remaining := len(sm.sessions)
	sm.mu.Unlock()

	sm.closeEvictedSessions(toClose)

	if len(expired) > 0 {
		sm.logger.Info("cleaned up expired sessions",
			"count", len(expired),
			"remaining", remaining)
	}
}

// EvictLRU evicts the least recently used sessions.
// This is called when memory pressure is high.
func (sm *SessionManager) EvictLRU(count int) int {
	sm.mu.Lock()

	if count <= 0 || len(sm.sessions) == 0 {
		sm.mu.Unlock()
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
	var toClose []*Session
	for i := 0; i < count && i < len(sessions); i++ {
		entry := sessions[i]
		if session := sm.removeSessionLocked(entry.id); session != nil {
			toClose = append(toClose, session)
		}

		evicted++
	}
	remaining := len(sm.sessions)
	sm.mu.Unlock()

	sm.closeEvictedSessions(toClose)

	if evicted > 0 {
		sm.logger.Info("evicted LRU sessions",
			"count", evicted,
			"remaining", remaining)
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
	sm.sessionsByIP = make(map[string]int)
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
	sessions := make([]*Session, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		sessions = append(sessions, s)
	}
	active := len(sm.sessions)
	peak := sm.peakSessions
	sm.mu.RUnlock()

	var totalMemory int64
	for _, s := range sessions {
		totalMemory += s.MemoryUsage()
	}

	return ManagerStats{
		Active:       active,
		TotalCreated: sm.totalCreated.Load(),
		TotalClosed:  sm.totalClosed.Load(),
		Peak:         peak,
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
	sessions := make([]*Session, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		sessions = append(sessions, s)
	}
	sm.mu.RUnlock()

	var total int64
	for _, s := range sessions {
		total += s.MemoryUsage()
	}
	return total
}

// CheckMemoryPressure checks if memory usage exceeds limits and evicts if needed.
func (sm *SessionManager) CheckMemoryPressure() {
	if sm.limits.MaxMemoryPerSession > 0 {
		sm.evictOversizedSessions()
	}

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

func (sm *SessionManager) evictOversizedSessions() {
	type sessionEntry struct {
		id      string
		session *Session
		usage   int64
	}

	sm.mu.RLock()
	sessions := make([]sessionEntry, 0, len(sm.sessions))
	for id, sess := range sm.sessions {
		sessions = append(sessions, sessionEntry{id: id, session: sess})
	}
	sm.mu.RUnlock()

	oversized := make([]sessionEntry, 0)
	for _, entry := range sessions {
		usage := entry.session.MemoryUsage()
		if usage > sm.limits.MaxMemoryPerSession {
			entry.usage = usage
			oversized = append(oversized, entry)
		}
	}

	if len(oversized) == 0 {
		return
	}

	sort.Slice(oversized, func(i, j int) bool {
		return oversized[i].session.LastActive.Before(oversized[j].session.LastActive)
	})

	sm.mu.Lock()
	var toClose []*Session
	for _, entry := range oversized {
		if session := sm.removeSessionLocked(entry.id); session != nil {
			toClose = append(toClose, session)
		}
	}
	remaining := len(sm.sessions)
	sm.mu.Unlock()

	sm.closeEvictedSessions(toClose)

	if len(toClose) > 0 {
		sm.logger.Warn("session memory limit exceeded, evicting sessions",
			"count", len(toClose),
			"limit", sm.limits.MaxMemoryPerSession,
			"remaining", remaining)
	}
}

// =============================================================================
// Phase 12: Session Persistence Integration
// =============================================================================

// CheckIPLimit checks if the IP has exceeded the session limit.
// Returns ErrTooManySessionsFromIP if the limit is exceeded.
// This should be called before creating a new session.
func (sm *SessionManager) CheckIPLimit(ip string) error {
	if sm.maxSessionsPerIP <= 0 || ip == "" {
		return nil
	}

	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.sessionsByIP[ip] >= sm.maxSessionsPerIP {
		return ErrTooManySessionsFromIP
	}
	return nil
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
	sess.setOnDetach(sm.OnSessionDisconnect)

	// Re-register the session
	sm.mu.Lock()
	sm.sessions[sess.ID] = sess
	sm.trackSessionLocked(sess, false)
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
	sessionStore := store.NewSessionStore()
	sess.owner.SetValue(store.SessionKey, sessionStore)
	// Also set vango.SessionSignalStoreKey for vango.NewSharedSignal support.
	sess.owner.SetValue(vango.SessionSignalStoreKey, sessionStore)

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
