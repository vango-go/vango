package vtest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vango-dev/vango/v2/pkg/server"
	"github.com/vango-dev/vango/v2/pkg/session"
)

// Common errors for test session operations.
var (
	ErrSessionExpired    = errors.New("session expired or evicted")
	ErrSessionNotFound   = errors.New("session not found in store")
	ErrSerializeFailed   = errors.New("failed to serialize session")
	ErrDeserializeFailed = errors.New("failed to deserialize session")
)

// TestSession wraps a server.Session with lifecycle simulation methods.
// Use this for testing session persistence, reconnection, and server restarts.
type TestSession struct {
	*server.Session
	manager *server.SessionManager
	store   session.SessionStore
	config  TestSessionConfig
}

// TestSessionConfig configures test session behavior.
type TestSessionConfig struct {
	// ResumeWindow is how long sessions survive after disconnect.
	ResumeWindow time.Duration

	// Store is the session store for persistence tests.
	// If nil, an in-memory store is created.
	Store session.SessionStore
}

// TestSessionOption configures a TestSession.
type TestSessionOption func(*TestSessionConfig)

// WithResumeWindow sets the resume window for the test session.
func WithResumeWindow(d time.Duration) TestSessionOption {
	return func(c *TestSessionConfig) {
		c.ResumeWindow = d
	}
}

// WithStore sets a custom session store for persistence tests.
func WithStore(store session.SessionStore) TestSessionOption {
	return func(c *TestSessionConfig) {
		c.Store = store
	}
}

// NewTestSession creates a session for testing with optional configuration.
//
// Example:
//
//	manager := server.NewSessionManager(nil, nil, nil)
//	sess := vtest.NewTestSession(manager)
//	sess.Set("cart", []Item{{ID: "1"}})
//
//	// Simulate disconnect and reconnect
//	sess.SimulateDisconnect()
//	err := sess.SimulateReconnect()
//	require.NoError(t, err)
//
//	// Cart should still be there
//	cart := sess.Get("cart").([]Item)
//	assert.Len(t, cart, 1)
func NewTestSession(manager *server.SessionManager, opts ...TestSessionOption) *TestSession {
	config := TestSessionConfig{
		ResumeWindow: 5 * time.Minute,
	}
	for _, opt := range opts {
		opt(&config)
	}

	// Create store if not provided
	if config.Store == nil {
		config.Store = session.NewMemoryStore()
	}

	// Create a new session using a mock connection
	sess := server.NewMockSession()

	return &TestSession{
		Session: sess,
		manager: manager,
		store:   config.Store,
		config:  config,
	}
}

// SimulateDisconnect simulates a WebSocket disconnect.
// The session state is preserved for potential reconnection.
func (t *TestSession) SimulateDisconnect() error {
	ctx := context.Background()

	// Serialize and save to store
	data, err := t.serializeSession()
	if err != nil {
		return err
	}

	expiresAt := time.Now().Add(t.config.ResumeWindow)
	return t.store.Save(ctx, t.Session.ID, data, expiresAt)
}

// SimulateReconnect simulates a WebSocket reconnect within ResumeWindow.
// Returns ErrSessionExpired if the session was evicted or expired.
//
// Example:
//
//	sess.SimulateDisconnect()
//	time.Sleep(100 * time.Millisecond)
//	err := sess.SimulateReconnect()
//	if err != nil {
//	    // Session was evicted or expired
//	}
func (t *TestSession) SimulateReconnect() error {
	ctx := context.Background()

	// Try to load from store
	data, err := t.store.Load(ctx, t.Session.ID)
	if err != nil {
		return err
	}
	if data == nil {
		return ErrSessionExpired
	}

	// Deserialize state back into session
	return t.deserializeSession(data)
}

// SimulateRefresh simulates a full page refresh.
// This serializes the session to the store, then restores it as a new connection.
//
// Example:
//
//	// User modifies state
//	sess.Set("preferences", prefs)
//
//	// User hits F5
//	err := sess.SimulateRefresh()
//	require.NoError(t, err)
//
//	// State should be restored
//	restored := sess.Get("preferences")
func (t *TestSession) SimulateRefresh() error {
	ctx := context.Background()
	sessionID := t.Session.ID

	// Serialize current state
	data, err := t.serializeSession()
	if err != nil {
		return errors.Join(ErrSerializeFailed, err)
	}

	// Store to backend
	expiresAt := time.Now().Add(t.config.ResumeWindow)
	if err := t.store.Save(ctx, sessionID, data, expiresAt); err != nil {
		return err
	}

	// Create a fresh mock session (simulating new connection)
	newSess := server.NewMockSession()

	// Restore the old session ID (via cookie in real scenario)
	newSess.ID = sessionID

	// Load and restore state from store
	storedData, err := t.store.Load(ctx, sessionID)
	if err != nil {
		return err
	}
	if storedData == nil {
		return ErrSessionNotFound
	}

	t.Session = newSess

	// Deserialize into the new session
	if err := t.deserializeSession(storedData); err != nil {
		return errors.Join(ErrDeserializeFailed, err)
	}

	return nil
}

// SimulateServerRestart simulates a full server restart.
// All in-memory sessions are lost; only persisted sessions survive.
//
// Example:
//
//	// Add persistent data
//	sess.Set("user_id", "123")
//
//	// Server goes down
//	err := sess.SimulateServerRestart()
//	require.NoError(t, err)
//
//	// Should recover from store
//	userID := sess.Get("user_id")
//	assert.Equal(t, "123", userID)
func (t *TestSession) SimulateServerRestart() error {
	ctx := context.Background()
	sessionID := t.Session.ID

	// Save session to store (simulating graceful shutdown)
	data, err := t.serializeSession()
	if err != nil {
		return errors.Join(ErrSerializeFailed, err)
	}

	expiresAt := time.Now().Add(t.config.ResumeWindow)
	if err := t.store.Save(ctx, sessionID, data, expiresAt); err != nil {
		return err
	}

	// Create new session (simulating user reconnecting after restart)
	newSess := server.NewMockSession()
	newSess.ID = sessionID

	// Try to restore from store
	storedData, err := t.store.Load(ctx, sessionID)
	if err != nil {
		return err
	}
	if storedData == nil {
		// Session wasn't persisted, start fresh
		t.Session = newSess
		return nil
	}

	t.Session = newSess

	// Restore state
	if err := t.deserializeSession(storedData); err != nil {
		return errors.Join(ErrDeserializeFailed, err)
	}

	return nil
}

// SimulateEviction simulates the session being evicted due to memory pressure.
// After eviction, reconnect attempts will fail unless the session was persisted.
func (t *TestSession) SimulateEviction() error {
	ctx := context.Background()
	return t.store.Delete(ctx, t.Session.ID)
}

// SimulateTimeout simulates the session expiring due to inactivity.
// The session is removed from memory and the store.
func (t *TestSession) SimulateTimeout() error {
	ctx := context.Background()
	return t.store.Delete(ctx, t.Session.ID)
}

// serializeSession converts session state to bytes.
// Delegates to Session.Serialize() which handles all session data.
func (t *TestSession) serializeSession() ([]byte, error) {
	return t.Session.Serialize()
}

// deserializeSession restores session state from bytes.
// Delegates to Session.Deserialize() which handles all session data.
func (t *TestSession) deserializeSession(data []byte) error {
	return t.Session.Deserialize(data)
}

// Store returns the underlying session store for advanced testing.
func (t *TestSession) Store() session.SessionStore {
	return t.store
}

// Manager returns the underlying session manager for advanced testing.
func (t *TestSession) Manager() *server.SessionManager {
	return t.manager
}

// AssertPersisted verifies that the session is properly saved in the store.
func (t *TestSession) AssertPersisted(tb testing.TB) {
	tb.Helper()
	ctx := context.Background()

	data, err := t.store.Load(ctx, t.Session.ID)
	if err != nil {
		tb.Fatalf("failed to load session from store: %v", err)
	}
	if data == nil {
		tb.Fatal("session not found in store")
	}
}

// AssertNotPersisted verifies that the session is not in the store.
func (t *TestSession) AssertNotPersisted(tb testing.TB) {
	tb.Helper()
	ctx := context.Background()

	data, err := t.store.Load(ctx, t.Session.ID)
	if err != nil {
		tb.Fatalf("failed to check session in store: %v", err)
	}
	if data != nil {
		tb.Fatal("session unexpectedly found in store")
	}
}

// PersistNow immediately persists the session to the store.
// Useful for testing persistence without waiting for automatic saves.
func (t *TestSession) PersistNow() error {
	ctx := context.Background()

	data, err := t.serializeSession()
	if err != nil {
		return err
	}

	expiresAt := time.Now().Add(t.config.ResumeWindow)
	return t.store.Save(ctx, t.Session.ID, data, expiresAt)
}

// LoadFromStore loads and restores session state from the store.
// Returns ErrSessionNotFound if the session doesn't exist.
func (t *TestSession) LoadFromStore() error {
	ctx := context.Background()

	data, err := t.store.Load(ctx, t.Session.ID)
	if err != nil {
		return err
	}
	if data == nil {
		return ErrSessionNotFound
	}

	return t.deserializeSession(data)
}
