package vango

import (
	"sync"
	"sync/atomic"
)

// =============================================================================
// Shared Memo Definition
// =============================================================================

// sharedMemoID is the counter for generating unique shared memo IDs.
var sharedMemoID uint64

// sharedMemoKeyPrefix isolates shared memo keys from shared signal keys inside
// a SessionSignalStore. This prevents type collisions when both primitives are
// used in the same session.
const sharedMemoKeyPrefix uint64 = 1 << 63

func sharedMemoKey(id uint64) uint64 {
	return sharedMemoKeyPrefix | id
}

// SharedMemoDef defines a session-scoped memo.
// Each user session gets its own independent Memo[T] instance.
//
// Like SharedSignalDef, this exists because package-level shared primitives
// are created before any session exists; the per-session memo is created
// lazily the first time it is accessed within a session.
type SharedMemoDef[T any] struct {
	id      uint64
	compute func() T

	mu    sync.RWMutex
	equal func(T, T) bool
}

// NewSharedMemo creates a session-scoped memo definition.
// Each user session gets its own independent Memo[T] instance.
//
// Example:
//
//	var CartTotal = vango.NewSharedMemo(func() float64 {
//	    total := 0.0
//	    for _, item := range CartItems.Get() {
//	        total += item.Price * float64(item.Qty)
//	    }
//	    return total
//	})
func NewSharedMemo[T any](compute func() T) *SharedMemoDef[T] {
	return &SharedMemoDef[T]{
		id:      sharedMemoKey(atomic.AddUint64(&sharedMemoID, 1)),
		compute: compute,
	}
}

// Memo returns the underlying Memo[T] for the current session.
// Creates the memo if it doesn't exist in this session.
// Returns nil if called outside of a session context.
func (m *SharedMemoDef[T]) Memo() *Memo[T] {
	storeVal := GetContext(SessionSignalStoreKey)
	if storeVal == nil {
		return nil
	}

	store, ok := storeVal.(SessionSignalStore)
	if !ok {
		return nil
	}

	createFn := func() any {
		memo := NewMemo(m.compute)
		m.mu.RLock()
		eq := m.equal
		m.mu.RUnlock()
		if eq != nil {
			memo.WithEquals(eq)
		}
		return memo
	}

	val := store.GetOrCreateSignal(m.id, createFn)
	if val == nil {
		return nil
	}
	return val.(*Memo[T])
}

// Get returns the memo's value for the current session.
// If called outside a session context, it computes the value directly.
func (m *SharedMemoDef[T]) Get() T {
	memo := m.Memo()
	if memo == nil {
		return m.compute()
	}
	return memo.Get()
}

// Peek returns the memo's value without subscribing for the current session.
// If called outside a session context, it computes the value directly.
func (m *SharedMemoDef[T]) Peek() T {
	memo := m.Memo()
	if memo == nil {
		return m.compute()
	}
	return memo.Peek()
}

// WithEquals configures the memo with a custom equality function.
// The function applies to future memo instances created in new sessions.
// If called within a session where the memo already exists, it updates that
// memo instance as well.
func (m *SharedMemoDef[T]) WithEquals(fn func(T, T) bool) *SharedMemoDef[T] {
	m.mu.Lock()
	m.equal = fn
	m.mu.Unlock()

	if memo := m.Memo(); memo != nil {
		memo.WithEquals(fn)
	}
	return m
}

