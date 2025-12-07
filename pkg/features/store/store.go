package store

import (
	"sync"
	"sync/atomic"

	"github.com/vango-dev/vango/v2/pkg/vango"
)

// SessionKey is the context key for the session store.
var SessionKey = &struct{ name string }{"SessionStore"}

// SessionStore holds session-scoped signals.
type SessionStore struct {
	signals sync.Map // map[uint64]any
}

// NewSessionStore creates a new session store.
func NewSessionStore() *SessionStore {
	return &SessionStore{}
}

// GlobalSignal creates a signal shared across all sessions.
// This is just a standard vango.Signal but declared globally.
// It returns a wrapper to match the interface if needed, or just *Signal.
// For consistency with SharedSignal, we return *Global[T].
func GlobalSignal[T any](initial T) *Global[T] {
	return &Global[T]{
		Signal: vango.NewSignal(initial),
	}
}

// Global wraps a vango.Signal for global state.
type Global[T any] struct {
	*vango.Signal[T]
}

// SharedSignal creates a definition for a session-scoped signal.
// Accessing it will look up or create the signal in the current session context.
func SharedSignal[T any](initial T) *Shared[T] {
	return &Shared[T]{
		id:      nextID(),
		initial: initial,
	}
}

// Shared represents a session-scoped signal definition.
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
	ctxVal := vango.GetContext(SessionKey)
	if ctxVal == nil {
		return nil
	}

	store, ok := ctxVal.(*SessionStore)
	if !ok {
		return nil
	}

	// Double-checked locking optimization is hard with sync.Map,
	// but LoadOrStore is atomic.

	// Check if exists
	if val, ok := store.signals.Load(s.id); ok {
		return val.(*vango.Signal[T])
	}

	// Create new
	newSig := vango.NewSignal(s.initial)
	actual, loaded := store.signals.LoadOrStore(s.id, newSig)
	if loaded {
		return actual.(*vango.Signal[T])
	}
	return newSig
}
