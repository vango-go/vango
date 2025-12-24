package session

import (
	"context"
	"sync"
	"time"
)

// MemoryStore is an in-memory session store implementation.
// It's the default store and suitable for single-server deployments.
// For multi-server deployments, use RedisStore or SQLStore.
type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*storedSession
	closed   bool
	done     chan struct{}
}

type storedSession struct {
	data      []byte
	expiresAt time.Time
}

// MemoryStoreOption configures MemoryStore behavior.
type MemoryStoreOption func(*memoryStoreConfig)

type memoryStoreConfig struct {
	cleanupInterval time.Duration
}

// WithCleanupInterval sets how often expired sessions are cleaned up.
// Default: 1 minute.
func WithCleanupInterval(d time.Duration) MemoryStoreOption {
	return func(c *memoryStoreConfig) {
		c.cleanupInterval = d
	}
}

// NewMemoryStore creates a new in-memory session store.
func NewMemoryStore(opts ...MemoryStoreOption) *MemoryStore {
	cfg := &memoryStoreConfig{
		cleanupInterval: 1 * time.Minute,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	store := &MemoryStore{
		sessions: make(map[string]*storedSession),
		done:     make(chan struct{}),
	}

	go store.cleanupLoop(cfg.cleanupInterval)
	return store
}

// Save stores session data with an expiration time.
func (m *MemoryStore) Save(ctx context.Context, sessionID string, data []byte, expiresAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrStoreClosed{}
	}

	// Make a copy of data to prevent mutations
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	m.sessions[sessionID] = &storedSession{
		data:      dataCopy,
		expiresAt: expiresAt,
	}
	return nil
}

// Load retrieves session data if it exists and hasn't expired.
func (m *MemoryStore) Load(ctx context.Context, sessionID string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, ErrStoreClosed{}
	}

	s, ok := m.sessions[sessionID]
	if !ok {
		return nil, nil
	}

	// Check expiration
	if time.Now().After(s.expiresAt) {
		return nil, nil
	}

	// Return a copy to prevent mutations
	dataCopy := make([]byte, len(s.data))
	copy(dataCopy, s.data)
	return dataCopy, nil
}

// Delete removes a session from the store.
func (m *MemoryStore) Delete(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrStoreClosed{}
	}

	delete(m.sessions, sessionID)
	return nil
}

// Touch updates the expiration time for a session.
func (m *MemoryStore) Touch(ctx context.Context, sessionID string, expiresAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrStoreClosed{}
	}

	if s, ok := m.sessions[sessionID]; ok {
		s.expiresAt = expiresAt
	}
	return nil
}

// SaveAll saves multiple sessions atomically.
func (m *MemoryStore) SaveAll(ctx context.Context, sessions map[string]SessionData) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrStoreClosed{}
	}

	for id, sd := range sessions {
		// Make a copy of data to prevent mutations
		dataCopy := make([]byte, len(sd.Data))
		copy(dataCopy, sd.Data)

		m.sessions[id] = &storedSession{
			data:      dataCopy,
			expiresAt: sd.ExpiresAt,
		}
	}
	return nil
}

// Close shuts down the store and releases resources.
func (m *MemoryStore) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	m.closed = true
	close(m.done)
	m.sessions = nil
	return nil
}

// Count returns the number of sessions in the store.
// This is for monitoring/testing purposes.
func (m *MemoryStore) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// cleanupLoop periodically removes expired sessions.
func (m *MemoryStore) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.cleanup()
		case <-m.done:
			return
		}
	}
}

// cleanup removes expired sessions.
func (m *MemoryStore) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return
	}

	now := time.Now()
	var expired []string

	for id, s := range m.sessions {
		if now.After(s.expiresAt) {
			expired = append(expired, id)
		}
	}

	for _, id := range expired {
		delete(m.sessions, id)
	}
}
