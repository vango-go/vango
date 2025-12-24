package session

import (
	"context"
	"time"
)

// SessionStore defines the interface for session persistence backends.
// Implementations must be safe for concurrent use.
type SessionStore interface {
	// Save persists session state. Called periodically and on graceful shutdown.
	// The expiresAt parameter indicates when the session should expire.
	// If sessionID already exists, it should be overwritten.
	Save(ctx context.Context, sessionID string, data []byte, expiresAt time.Time) error

	// Load retrieves session state by ID.
	// Returns (nil, nil) if the session doesn't exist or has expired.
	// Returns (data, nil) if found and not expired.
	// Returns (nil, err) on backend errors.
	Load(ctx context.Context, sessionID string) ([]byte, error)

	// Delete removes a session. Called on explicit logout or expiration.
	// Should not return an error if the session doesn't exist.
	Delete(ctx context.Context, sessionID string) error

	// Touch updates the expiration time without loading full state.
	// This is more efficient than Load+Save for keep-alive operations.
	// Should not return an error if the session doesn't exist.
	Touch(ctx context.Context, sessionID string, expiresAt time.Time) error

	// SaveAll persists multiple sessions atomically (if possible).
	// Used during graceful shutdown to save all active sessions.
	// Implementations that don't support atomicity should save sequentially.
	SaveAll(ctx context.Context, sessions map[string]SessionData) error

	// Close releases any resources held by the store.
	// Called when the server shuts down.
	Close() error
}

// SessionData contains serialized session state with metadata.
type SessionData struct {
	// Data is the serialized session state.
	Data []byte

	// ExpiresAt is when the session should expire.
	ExpiresAt time.Time
}

// StoreOption is a functional option for configuring stores.
type StoreOption func(interface{})

// SessionNotFoundError is returned when a session doesn't exist.
// Note: Load returns (nil, nil) for missing sessions, not this error.
// This is used by implementations that need an explicit error type.
type SessionNotFoundError struct {
	SessionID string
}

func (e SessionNotFoundError) Error() string {
	return "session not found: " + e.SessionID
}

// ErrStoreClosed is returned when operations are attempted on a closed store.
type ErrStoreClosed struct{}

func (e ErrStoreClosed) Error() string {
	return "session store is closed"
}
