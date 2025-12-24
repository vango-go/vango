// Package pref provides user preference management with sync capabilities.
//
// Preferences are persisted values that:
//   - Work for anonymous users (stored in LocalStorage via client)
//   - Sync when user logs in (merge with database)
//   - Stay consistent across tabs and devices
//
// Example:
//
//	// Simple theme preference
//	theme := pref.New("theme", "light")
//
//	// With merge strategy
//	settings := pref.New("settings", Settings{}, pref.MergeWith(pref.DBWins))
//
//	// Read/write
//	current := theme.Get()
//	theme.Set("dark")
package pref

import (
	"encoding/json"
	"sync"
	"time"
)

// MergeStrategy determines how conflicts are resolved when local and remote values differ.
type MergeStrategy int

const (
	// DBWins uses the server/database value, discards local.
	// Best for settings that should be authoritative from server.
	DBWins MergeStrategy = iota

	// LocalWins uses the local value, updates server.
	// Best for preferences that the user just modified.
	LocalWins

	// Prompt notifies the user to choose.
	// Best for important settings where the user should decide.
	Prompt

	// LWW uses last-write-wins with timestamps.
	// Best for frequently changing preferences.
	LWW
)

// PrefOption is a functional option for configuring preferences.
type PrefOption func(*prefConfig)

type prefConfig struct {
	mergeStrategy   MergeStrategy
	conflictHandler func(local, remote any) any
	persistLocal    bool
	syncToServer    bool
}

// MergeWith sets the merge strategy for conflict resolution.
func MergeWith(strategy MergeStrategy) PrefOption {
	return func(c *prefConfig) {
		c.mergeStrategy = strategy
	}
}

// OnConflict sets a custom conflict handler.
// The handler receives local and remote values and returns the resolved value.
func OnConflict(handler func(local, remote any) any) PrefOption {
	return func(c *prefConfig) {
		c.conflictHandler = handler
	}
}

// LocalOnly prevents syncing this preference to the server.
// Useful for device-specific settings like volume.
func LocalOnly() PrefOption {
	return func(c *prefConfig) {
		c.syncToServer = false
	}
}

// Pref represents a user preference with sync capabilities.
type Pref[T any] struct {
	key       string
	value     T
	defaults  T
	updatedAt time.Time
	config    prefConfig

	mu sync.RWMutex

	// Context for persistence (set during initialization)
	persistLocal func(key string, value any, updatedAt time.Time)
	persistDB    func(key string, value any, updatedAt time.Time)
	broadcast    func(key string, value any, updatedAt time.Time)
}

// New creates a new preference with the given key and default value.
func New[T any](key string, defaultValue T, opts ...PrefOption) *Pref[T] {
	config := prefConfig{
		mergeStrategy: LWW,
		persistLocal:  true,
		syncToServer:  true,
	}
	for _, opt := range opts {
		opt(&config)
	}

	return &Pref[T]{
		key:       key,
		value:     defaultValue,
		defaults:  defaultValue,
		updatedAt: time.Now(),
		config:    config,
	}
}

// Get returns the current preference value.
func (p *Pref[T]) Get() T {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.value
}

// Set updates the preference value and triggers sync.
func (p *Pref[T]) Set(value T) {
	p.mu.Lock()
	p.value = value
	p.updatedAt = time.Now()
	updatedAt := p.updatedAt
	p.mu.Unlock()

	// Broadcast to other tabs
	if p.broadcast != nil {
		p.broadcast(p.key, value, updatedAt)
	}

	// Persist locally
	if p.config.persistLocal && p.persistLocal != nil {
		p.persistLocal(p.key, value, updatedAt)
	}

	// Persist to server if authenticated
	if p.config.syncToServer && p.persistDB != nil {
		p.persistDB(p.key, value, updatedAt)
	}
}

// Reset resets the preference to its default value.
func (p *Pref[T]) Reset() {
	p.Set(p.defaults)
}

// Key returns the preference key.
func (p *Pref[T]) Key() string {
	return p.key
}

// UpdatedAt returns when the preference was last updated.
func (p *Pref[T]) UpdatedAt() time.Time {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.updatedAt
}

// SetFromRemote updates the value from a remote source (another tab or server).
// Uses the configured merge strategy to resolve conflicts.
func (p *Pref[T]) SetFromRemote(value T, remoteUpdatedAt time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()

	resolved := p.resolveConflict(p.value, value, p.updatedAt, remoteUpdatedAt)
	if resolvedT, ok := resolved.(T); ok {
		p.value = resolvedT
		// Use the newer timestamp
		if remoteUpdatedAt.After(p.updatedAt) {
			p.updatedAt = remoteUpdatedAt
		}
	}
}

// resolveConflict applies the merge strategy to resolve conflicts.
func (p *Pref[T]) resolveConflict(local, remote any, localTime, remoteTime time.Time) any {
	// Custom handler takes precedence
	if p.config.conflictHandler != nil {
		return p.config.conflictHandler(local, remote)
	}

	switch p.config.mergeStrategy {
	case DBWins:
		return remote
	case LocalWins:
		return local
	case LWW:
		if remoteTime.After(localTime) {
			return remote
		}
		return local
	case Prompt:
		// For prompt, we need to notify the UI
		// For now, default to LWW
		if remoteTime.After(localTime) {
			return remote
		}
		return local
	default:
		return local
	}
}

// SetPersistHandlers sets the persistence handlers.
// Called during component initialization.
func (p *Pref[T]) SetPersistHandlers(
	local func(key string, value any, updatedAt time.Time),
	db func(key string, value any, updatedAt time.Time),
	broadcast func(key string, value any, updatedAt time.Time),
) {
	p.persistLocal = local
	p.persistDB = db
	p.broadcast = broadcast
}

// MarshalJSON implements json.Marshaler.
func (p *Pref[T]) MarshalJSON() ([]byte, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return json.Marshal(struct {
		Key       string    `json:"key"`
		Value     T         `json:"value"`
		UpdatedAt time.Time `json:"updated_at"`
	}{
		Key:       p.key,
		Value:     p.value,
		UpdatedAt: p.updatedAt,
	})
}

// UnmarshalJSON implements json.Unmarshaler.
func (p *Pref[T]) UnmarshalJSON(data []byte) error {
	var temp struct {
		Key       string    `json:"key"`
		Value     T         `json:"value"`
		UpdatedAt time.Time `json:"updated_at"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.key = temp.Key
	p.value = temp.Value
	p.updatedAt = temp.UpdatedAt
	return nil
}
