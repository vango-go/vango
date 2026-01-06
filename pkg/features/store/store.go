// Package store provides session-scoped and global signal storage.
//
// Deprecated: Use vango.NewSharedSignal and vango.NewGlobalSignal instead.
// This package is retained for backward compatibility.
//
// Migration:
//
//	// Old:
//	import "github.com/vango-go/vango/pkg/features/store"
//	var Cart = store.NewSharedSignal([]Item{})
//
//	// New:
//	import "github.com/vango-go/vango/pkg/vango"
//	var Cart = vango.NewSharedSignal([]Item{})
package store

import (
	"sync"
	"sync/atomic"

	"github.com/vango-go/vango/pkg/vango"
)

// SessionKey is the context key for the session store.
//
// Deprecated: Use vango.SessionSignalStoreKey instead.
var SessionKey = &struct{ name string }{"SessionStore"}

// SessionStore holds session-scoped signals.
// It implements vango.SessionSignalStore for compatibility with the vango package.
type SessionStore struct {
	signals sync.Map // map[uint64]any
}

// NewSessionStore creates a new session store.
func NewSessionStore() *SessionStore {
	return &SessionStore{}
}

// GetOrCreateSignal implements vango.SessionSignalStore.
// This allows SessionStore to be used with vango.NewSharedSignal.
func (s *SessionStore) GetOrCreateSignal(id uint64, createFn func() any) any {
	// Try to load existing
	if val, ok := s.signals.Load(id); ok {
		return val
	}

	// Create new and try to store
	newVal := createFn()
	actual, _ := s.signals.LoadOrStore(id, newVal)
	return actual
}

// NewGlobalSignal creates a signal shared across all sessions.
//
// Deprecated: Use vango.NewGlobalSignal instead.
//
//	// Old:
//	var Status = store.NewGlobalSignal("online")
//
//	// New:
//	var Status = vango.NewGlobalSignal("online")
func NewGlobalSignal[T any](initial T) *Global[T] {
	return &Global[T]{
		Signal: vango.NewSignal(initial),
	}
}

// Global wraps a vango.Signal for global state.
//
// Deprecated: Use vango.GlobalSignal instead.
type Global[T any] struct {
	*vango.Signal[T]
}

// NewSharedSignal creates a definition for a session-scoped signal.
// Accessing it will look up or create the signal in the current session context.
//
// Deprecated: Use vango.NewSharedSignal instead.
//
//	// Old:
//	var Cart = store.NewSharedSignal([]Item{})
//
//	// New:
//	var Cart = vango.NewSharedSignal([]Item{})
func NewSharedSignal[T any](initial T) *Shared[T] {
	return &Shared[T]{
		id:      nextID(),
		initial: initial,
	}
}

// Shared represents a session-scoped signal definition.
//
// Deprecated: Use vango.SharedSignalDef instead.
type Shared[T any] struct {
	id      uint64
	initial T
}

var idCounter uint64

func nextID() uint64 {
	return atomic.AddUint64(&idCounter, 1)
}

// Get retrieves the current value of the signal for the current session.
// It subscribes the current listener if active.
func (s *Shared[T]) Get() T {
	sig := s.getSignal()
	if sig == nil {
		// Fallback to initial if no session context (e.g. testing without setup)
		return s.initial
	}
	return sig.Get()
}

// Set updates the value of the signal for the current session.
func (s *Shared[T]) Set(val T) {
	sig := s.getSignal()
	if sig != nil {
		sig.Set(val)
	}
}

// Update updates the value using a transformer function.
func (s *Shared[T]) Update(fn func(T) T) {
	sig := s.getSignal()
	if sig != nil {
		val := sig.Peek()
		sig.Set(fn(val))
	}
}

// getSignal retrieves or creates the underlying vango.Signal for the current session.
func (s *Shared[T]) getSignal() *vango.Signal[T] {
	// Try store.SessionKey first (legacy), then vango.SessionSignalStoreKey
	ctxVal := vango.GetContext(SessionKey)
	if ctxVal == nil {
		ctxVal = vango.GetContext(vango.SessionSignalStoreKey)
	}
	if ctxVal == nil {
		return nil
	}

	// Support both SessionStore and vango.SessionSignalStore interfaces
	var store vango.SessionSignalStore
	switch v := ctxVal.(type) {
	case *SessionStore:
		store = v
	case vango.SessionSignalStore:
		store = v
	default:
		return nil
	}

	createFn := func() any {
		return vango.NewSignal(s.initial)
	}

	sigVal := store.GetOrCreateSignal(s.id, createFn)
	if sigVal == nil {
		return nil
	}
	return sigVal.(*vango.Signal[T])
}
