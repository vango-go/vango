package session

import (
	"container/list"
	"context"
	"errors"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"
)

// Manager manages session lifecycle, persistence, and memory protection.
// It provides LRU eviction for detached sessions and per-IP session limits.
type Manager struct {
	mu sync.RWMutex

	// All sessions by ID
	sessions map[string]*ManagedSession

	// Detached sessions in LRU order (front = most recently accessed)
	detachedQueue *list.List
	detachedIndex map[string]*list.Element

	// Session count per IP address
	sessionsByIP map[string]int

	// Configuration
	config ManagerConfig

	// Session store for persistence
	store SessionStore

	// Logger
	logger *slog.Logger

	// Random source (for EvictionRandom); overrideable for tests.
	randIntn func(n int) int

	// Lifecycle
	done    chan struct{}
	stopped bool
}

// ManagedSession wraps session data with management metadata.
type ManagedSession struct {
	// ID is the unique session identifier.
	ID string

	// IP is the client IP address for per-IP limiting.
	IP string

	// CreatedAt is when the session was created.
	CreatedAt time.Time

	// LastActive is when the session was last accessed.
	LastActive time.Time

	// DisconnectedAt is when the client disconnected (zero if connected).
	DisconnectedAt time.Time

	// Data is the serialized session state (set when disconnected).
	Data []byte

	// Connected indicates whether the client has an active WebSocket.
	Connected bool

	// UserID is the authenticated user ID, if any.
	UserID string
}

// ManagerConfig configures the session manager.
type ManagerConfig struct {
	// MaxDetachedSessions is the maximum number of detached sessions before LRU eviction.
	// Default: 10000.
	MaxDetachedSessions int

	// MaxSessionsPerIP is the maximum number of active sessions per IP address.
	// Default: 100.
	MaxSessionsPerIP int

	// ResumeWindow is how long a detached session remains resumable.
	// Default: 5 minutes.
	ResumeWindow time.Duration

	// PersistInterval is how often to persist dirty sessions.
	// Default: 30 seconds.
	PersistInterval time.Duration

	// CleanupInterval is how often to clean up expired sessions.
	// Default: 1 minute.
	CleanupInterval time.Duration

	// EvictionPolicy determines how sessions are evicted when limits are exceeded.
	// Default: EvictionLRU.
	EvictionPolicy EvictionPolicy
}

// EvictionPolicy determines which sessions are evicted first.
type EvictionPolicy int

const (
	// EvictionLRU evicts the least recently accessed sessions first.
	EvictionLRU EvictionPolicy = iota

	// EvictionOldest evicts the oldest sessions first (by creation time).
	EvictionOldest

	// EvictionRandom evicts sessions randomly (faster but less fair).
	EvictionRandom
)

// DefaultManagerConfig returns a ManagerConfig with sensible defaults.
func DefaultManagerConfig() ManagerConfig {
	return ManagerConfig{
		MaxDetachedSessions: 10000,
		MaxSessionsPerIP:    100,
		ResumeWindow:        5 * time.Minute,
		PersistInterval:     30 * time.Second,
		CleanupInterval:     1 * time.Minute,
		EvictionPolicy:      EvictionLRU,
	}
}

// Error types for session management.
var (
	// ErrTooManySessionsFromIP is returned when the per-IP session limit is exceeded.
	ErrTooManySessionsFromIP = errors.New("too many sessions from this IP address")

	// ErrMaxSessionsReached is returned when the maximum session limit is reached.
	ErrMaxSessionsReached = errors.New("maximum session limit reached")

	// ErrSessionExpired is returned when trying to resume an expired session.
	ErrSessionExpired = errors.New("session has expired")

	// ErrSessionNotFound is returned when a session doesn't exist.
	ErrSessionNotFound = errors.New("session not found")

	// ErrManagerStopped is returned when operations are attempted on a stopped manager.
	ErrManagerStopped = errors.New("session manager is stopped")
)

// NewManager creates a new session manager.
func NewManager(store SessionStore, config ManagerConfig, logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}

	m := &Manager{
		sessions:      make(map[string]*ManagedSession),
		detachedQueue: list.New(),
		detachedIndex: make(map[string]*list.Element),
		sessionsByIP:  make(map[string]int),
		config:        config,
		store:         store,
		logger:        logger.With("component", "session_manager"),
		randIntn:      rand.IntN,
		done:          make(chan struct{}),
	}

	// Start background goroutines
	go m.cleanupLoop()

	return m
}

// CheckIPLimit verifies that the IP hasn't exceeded its session limit.
// This should be called before creating a new session.
func (m *Manager) CheckIPLimit(ip string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.stopped {
		return ErrManagerStopped
	}

	if m.config.MaxSessionsPerIP > 0 && m.sessionsByIP[ip] >= m.config.MaxSessionsPerIP {
		return ErrTooManySessionsFromIP
	}
	return nil
}

// Register adds a new session to the manager.
// The session is marked as connected.
func (m *Manager) Register(sess *ManagedSession) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.stopped {
		return ErrManagerStopped
	}

	// Check IP limit
	if m.config.MaxSessionsPerIP > 0 && m.sessionsByIP[sess.IP] >= m.config.MaxSessionsPerIP {
		return ErrTooManySessionsFromIP
	}

	// Store session
	m.sessions[sess.ID] = sess
	m.sessionsByIP[sess.IP]++
	sess.Connected = true
	sess.LastActive = time.Now()

	m.logger.Debug("session registered",
		"session_id", sess.ID,
		"ip", sess.IP,
		"ip_session_count", m.sessionsByIP[sess.IP])

	return nil
}

// OnDisconnect handles a client disconnect.
// The session becomes detached and can be resumed within ResumeWindow.
func (m *Manager) OnDisconnect(sessionID string, serializedData []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sess, exists := m.sessions[sessionID]
	if !exists || m.stopped {
		return
	}

	now := time.Now()
	sess.Connected = false
	sess.DisconnectedAt = now
	sess.Data = serializedData

	// Ensure the detached queue contains at most one entry per session.
	if elem, ok := m.detachedIndex[sessionID]; ok {
		m.detachedQueue.Remove(elem)
		delete(m.detachedIndex, sessionID)
	}

	// Add to detached queue (front = most recently used)
	elem := m.detachedQueue.PushFront(sessionID)
	m.detachedIndex[sessionID] = elem

	// Evict if over limit
	for m.detachedQueue.Len() > m.config.MaxDetachedSessions {
		m.evictOneLocked()
	}

	// Persist to store if available
	if m.store != nil && len(serializedData) > 0 {
		expiresAt := now.Add(m.config.ResumeWindow)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := m.store.Save(ctx, sessionID, serializedData, expiresAt); err != nil {
				m.logger.Warn("failed to persist detached session",
					"session_id", sessionID,
					"error", err)
			}
		}()
	}

	m.logger.Debug("session disconnected",
		"session_id", sessionID,
		"detached_count", m.detachedQueue.Len())
}

// OnReconnect attempts to restore a session after reconnect.
// Returns the restored session data if found and not expired.
func (m *Manager) OnReconnect(sessionID string) (*ManagedSession, []byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.stopped {
		return nil, nil, ErrManagerStopped
	}

	sess, exists := m.sessions[sessionID]
	if !exists {
		// Try to load from store
		if m.store != nil {
			data, err := m.store.Load(context.Background(), sessionID)
			if err != nil {
				return nil, nil, err
			}
			if data != nil {
				// Session exists in store but not in memory
				// The caller needs to deserialize and restore
				return nil, data, nil
			}
		}
		return nil, nil, ErrSessionNotFound
	}

	// Check if session is expired
	if !sess.DisconnectedAt.IsZero() {
		if time.Since(sess.DisconnectedAt) > m.config.ResumeWindow {
			// Session expired, remove it
			m.removeSessionLocked(sessionID)
			return nil, nil, ErrSessionExpired
		}
	}

	// Remove from detached queue
	if elem, ok := m.detachedIndex[sessionID]; ok {
		m.detachedQueue.Remove(elem)
		delete(m.detachedIndex, sessionID)
	}

	// Mark as connected
	sess.Connected = true
	sess.DisconnectedAt = time.Time{}
	sess.LastActive = time.Now()
	data := sess.Data
	sess.Data = nil

	m.logger.Debug("session reconnected",
		"session_id", sessionID,
		"detached_count", m.detachedQueue.Len())

	return sess, data, nil
}

// Get retrieves a session by ID.
func (m *Manager) Get(sessionID string) *ManagedSession {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[sessionID]
}

// Touch updates the last active time for a session.
func (m *Manager) Touch(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sess, exists := m.sessions[sessionID]; exists {
		sess.LastActive = time.Now()

		// If in detached queue, move to front
		if elem, ok := m.detachedIndex[sessionID]; ok {
			m.detachedQueue.MoveToFront(elem)
		}
	}
}

// Remove removes a session from the manager.
// Called on explicit logout or session termination.
func (m *Manager) Remove(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.removeSessionLocked(sessionID)
}

// removeSessionLocked removes a session (must be called with lock held).
func (m *Manager) removeSessionLocked(sessionID string) {
	sess, exists := m.sessions[sessionID]
	if !exists {
		return
	}

	// Remove from maps
	delete(m.sessions, sessionID)
	m.sessionsByIP[sess.IP]--
	if m.sessionsByIP[sess.IP] <= 0 {
		delete(m.sessionsByIP, sess.IP)
	}

	// Remove from detached queue
	if elem, ok := m.detachedIndex[sessionID]; ok {
		m.detachedQueue.Remove(elem)
		delete(m.detachedIndex, sessionID)
	}

	// Delete from store
	if m.store != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			m.store.Delete(ctx, sessionID)
		}()
	}

	m.logger.Debug("session removed",
		"session_id", sessionID,
		"remaining", len(m.sessions))
}

// evictOneLocked evicts one detached session according to the configured EvictionPolicy
// (must be called with lock held).
func (m *Manager) evictOneLocked() {
	if m.detachedQueue.Len() == 0 {
		return
	}

	var sessionID string

	switch m.config.EvictionPolicy {
	case EvictionLRU:
		// Least recently used detached session is at the back.
		if back := m.detachedQueue.Back(); back != nil {
			sessionID = back.Value.(string)
		}
	case EvictionOldest:
		// Oldest by creation time among detached sessions.
		var oldestID string
		var oldestTime time.Time
		found := false

		for e := m.detachedQueue.Front(); e != nil; e = e.Next() {
			id := e.Value.(string)
			sess := m.sessions[id]
			if sess == nil {
				continue
			}
			if !found || sess.CreatedAt.Before(oldestTime) {
				found = true
				oldestID = id
				oldestTime = sess.CreatedAt
			}
		}

		if found {
			sessionID = oldestID
		} else if back := m.detachedQueue.Back(); back != nil {
			sessionID = back.Value.(string)
		}
	case EvictionRandom:
		// Random detached session for speed; deterministic in tests via randIntn override.
		n := m.detachedQueue.Len()
		if n == 0 {
			return
		}

		intn := m.randIntn
		if intn == nil {
			intn = rand.IntN
		}

		idx := intn(n)
		if idx < 0 {
			idx = 0
		} else if idx >= n {
			idx = n - 1
		}

		e := m.detachedQueue.Front()
		for i := 0; i < idx && e != nil; i++ {
			e = e.Next()
		}
		if e == nil {
			e = m.detachedQueue.Back()
		}
		if e != nil {
			sessionID = e.Value.(string)
		}
	default:
		// Fail-safe: treat unknown values as LRU.
		if back := m.detachedQueue.Back(); back != nil {
			sessionID = back.Value.(string)
		}
	}

	if sessionID == "" {
		return
	}

	sess := m.sessions[sessionID]

	// Persist before eviction if store is configured.
	if m.store != nil && sess != nil && len(sess.Data) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		expiresAt := time.Now().Add(m.config.ResumeWindow)
		_ = m.store.Save(ctx, sessionID, sess.Data, expiresAt)
		cancel()
	}

	m.removeSessionLocked(sessionID)

	m.logger.Debug("evicted session",
		"session_id", sessionID,
		"policy", m.config.EvictionPolicy,
		"reason", "detached_limit_exceeded")
}

// cleanupLoop periodically cleans up expired sessions.
func (m *Manager) cleanupLoop() {
	ticker := time.NewTicker(m.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.cleanupExpired()
		case <-m.done:
			return
		}
	}
}

// cleanupExpired removes sessions that have exceeded ResumeWindow.
func (m *Manager) cleanupExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.stopped {
		return
	}

	now := time.Now()
	var expired []string

	for id, sess := range m.sessions {
		// Only check detached sessions
		if sess.DisconnectedAt.IsZero() {
			continue
		}

		if now.Sub(sess.DisconnectedAt) > m.config.ResumeWindow {
			expired = append(expired, id)
		}
	}

	for _, id := range expired {
		m.removeSessionLocked(id)
	}

	if len(expired) > 0 {
		m.logger.Debug("cleaned up expired sessions",
			"count", len(expired),
			"remaining", len(m.sessions))
	}
}

// Shutdown gracefully shuts down the manager, persisting all sessions.
func (m *Manager) Shutdown(ctx context.Context) error {
	m.mu.Lock()

	if m.stopped {
		m.mu.Unlock()
		return nil
	}

	m.stopped = true
	close(m.done)

	// Collect all sessions to persist
	sessionsToSave := make(map[string]SessionData)
	for id, sess := range m.sessions {
		if len(sess.Data) > 0 {
			sessionsToSave[id] = SessionData{
				Data:      sess.Data,
				ExpiresAt: time.Now().Add(m.config.ResumeWindow),
			}
		}
	}

	m.mu.Unlock()

	// Persist all sessions
	if m.store != nil && len(sessionsToSave) > 0 {
		if err := m.store.SaveAll(ctx, sessionsToSave); err != nil {
			m.logger.Warn("failed to persist sessions on shutdown",
				"error", err,
				"count", len(sessionsToSave))
			return err
		}
		m.logger.Info("persisted sessions on shutdown",
			"count", len(sessionsToSave))
	}

	return nil
}

// Stats returns manager statistics.
func (m *Manager) Stats() ManagerStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	connected := 0
	for _, sess := range m.sessions {
		if sess.Connected {
			connected++
		}
	}

	return ManagerStats{
		Total:     len(m.sessions),
		Connected: connected,
		Detached:  m.detachedQueue.Len(),
		UniqueIPs: len(m.sessionsByIP),
	}
}

// ManagerStats contains session manager statistics.
type ManagerStats struct {
	// Total is the total number of sessions (connected + detached).
	Total int

	// Connected is the number of sessions with active WebSocket connections.
	Connected int

	// Detached is the number of sessions waiting for reconnection.
	Detached int

	// UniqueIPs is the number of unique client IP addresses.
	UniqueIPs int
}
