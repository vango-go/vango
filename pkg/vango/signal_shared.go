package vango

import (
	"sync"
	"sync/atomic"
)

// =============================================================================
// Session Signal Store Interface
// =============================================================================

// SessionSignalStoreKey is the context key for the session signal store.
// The server package should set this during session initialization.
var SessionSignalStoreKey = &struct{ name string }{"SessionSignalStore"}

// SessionSignalStore is the interface for the session-scoped signal store.
// This is implemented by the server package's session store.
type SessionSignalStore interface {
	// GetOrCreateSignal retrieves an existing signal by ID, or creates a new one
	// using the provided factory function if it doesn't exist.
	GetOrCreateSignal(id uint64, createFn func() any) any
}

// =============================================================================
// Shared Signal Definition
// =============================================================================

// sharedSignalID is the counter for generating unique shared signal IDs.
var sharedSignalID uint64

// SharedSignalDef defines a session-scoped signal.
// Each user session gets its own independent Signal[T] instance.
// The signal is created lazily when first accessed in a session.
//
// Note: The spec (§3.9.4) states that NewSharedSignal should return Signal[T].
// However, this is not achievable in Go because:
//   - The definition is created at package init time (before any session exists)
//   - The actual *Signal[T] is created per-session at runtime
//   - Go doesn't support virtual method dispatch on structs
//
// SharedSignalDef provides all Signal[T] methods through proxying, so the API
// is behaviorally equivalent. Use .Signal() to get the underlying *Signal[T]
// for the current session if needed.
//
// Example:
//
//	// Package-level definition
//	var CartItems = vango.NewSharedSignal([]CartItem{})
//
//	// In a component
//	func ShoppingCart() vango.Component {
//	    return vango.Func(func() *vango.VNode {
//	        items := CartItems.Get()  // Gets items for this session only
//	        return Div(
//	            Range(items, func(item CartItem, i int) *vango.VNode {
//	                return Li(Text(item.Name))
//	            }),
//	        )
//	    })
//	}
type SharedSignalDef[T any] struct {
	id      uint64
	initial T
	opts    []SignalOption
}

// NewSharedSignal creates a session-scoped signal definition.
// Each user session gets its own independent Signal[T] with the given initial value.
//
// Returns *SharedSignalDef[T] which provides all Signal[T] methods.
// See SharedSignalDef documentation for why this differs from the spec's Signal[T] return type.
//
// Example:
//
//	var CartItems = vango.NewSharedSignal([]CartItem{})
//	var UserPrefs = vango.NewSharedSignal(Preferences{Theme: "light"})
func NewSharedSignal[T any](initial T, opts ...SignalOption) *SharedSignalDef[T] {
	return &SharedSignalDef[T]{
		id:      atomic.AddUint64(&sharedSignalID, 1),
		initial: initial,
		opts:    opts,
	}
}

// Signal returns the underlying Signal[T] for the current session.
// Creates the signal if it doesn't exist in this session.
// Returns nil if called outside of a session context.
func (s *SharedSignalDef[T]) Signal() *Signal[T] {
	storeVal := GetContext(SessionSignalStoreKey)
	if storeVal == nil {
		return nil
	}

	store, ok := storeVal.(SessionSignalStore)
	if !ok {
		return nil
	}

	createFn := func() any {
		return NewSignal(s.initial, s.opts...)
	}

	sigVal := store.GetOrCreateSignal(s.id, createFn)
	if sigVal == nil {
		return nil
	}

	return sigVal.(*Signal[T])
}

// =============================================================================
// Proxy Methods - Core Signal Operations
// =============================================================================

// Get returns the current value for the current session.
// Returns the initial value if called outside of a session context.
func (s *SharedSignalDef[T]) Get() T {
	sig := s.Signal()
	if sig == nil {
		return s.initial
	}
	return sig.Get()
}

// Peek returns the current value without subscribing.
// Returns the initial value if called outside of a session context.
func (s *SharedSignalDef[T]) Peek() T {
	sig := s.Signal()
	if sig == nil {
		return s.initial
	}
	return sig.Peek()
}

// Set updates the value for the current session.
// Does nothing if called outside of a session context.
func (s *SharedSignalDef[T]) Set(value T) {
	sig := s.Signal()
	if sig != nil {
		sig.Set(value)
	}
}

// Update atomically reads and updates the value for the current session.
// Does nothing if called outside of a session context.
func (s *SharedSignalDef[T]) Update(fn func(T) T) {
	sig := s.Signal()
	if sig != nil {
		sig.Update(fn)
	}
}

// =============================================================================
// Proxy Methods - Numeric Operations
// =============================================================================

// Inc increments the value by 1.
// Panics if the signal's type is not numeric.
func (s *SharedSignalDef[T]) Inc() {
	sig := s.Signal()
	if sig != nil {
		sig.Inc()
	}
}

// Dec decrements the value by 1.
// Panics if the signal's type is not numeric.
func (s *SharedSignalDef[T]) Dec() {
	sig := s.Signal()
	if sig != nil {
		sig.Dec()
	}
}

// Add adds n to the value.
// Panics if the signal's type is not numeric or if n is the wrong type.
func (s *SharedSignalDef[T]) Add(n any) {
	sig := s.Signal()
	if sig != nil {
		sig.Add(n)
	}
}

// Sub subtracts n from the value.
// Panics if the signal's type is not numeric or if n is the wrong type.
func (s *SharedSignalDef[T]) Sub(n any) {
	sig := s.Signal()
	if sig != nil {
		sig.Sub(n)
	}
}

// Mul multiplies the value by n.
// Panics if the signal's type is not numeric or if n is the wrong type.
func (s *SharedSignalDef[T]) Mul(n any) {
	sig := s.Signal()
	if sig != nil {
		sig.Mul(n)
	}
}

// Div divides the value by n.
// Panics if the signal's type is not numeric or if n is the wrong type.
func (s *SharedSignalDef[T]) Div(n any) {
	sig := s.Signal()
	if sig != nil {
		sig.Div(n)
	}
}

// =============================================================================
// Proxy Methods - Boolean Operations
// =============================================================================

// Toggle inverts a boolean value.
// Panics if the signal's type is not bool.
func (s *SharedSignalDef[T]) Toggle() {
	sig := s.Signal()
	if sig != nil {
		sig.Toggle()
	}
}

// SetTrue sets the value to true.
// Panics if the signal's type is not bool.
func (s *SharedSignalDef[T]) SetTrue() {
	sig := s.Signal()
	if sig != nil {
		sig.SetTrue()
	}
}

// SetFalse sets the value to false.
// Panics if the signal's type is not bool.
func (s *SharedSignalDef[T]) SetFalse() {
	sig := s.Signal()
	if sig != nil {
		sig.SetFalse()
	}
}

// =============================================================================
// Proxy Methods - String Operations
// =============================================================================

// Append appends a suffix to a string value.
// Panics if the signal's type is not string.
func (s *SharedSignalDef[T]) Append(suffix string) {
	sig := s.Signal()
	if sig != nil {
		sig.Append(suffix)
	}
}

// Prepend prepends a prefix to a string value.
// Panics if the signal's type is not string.
func (s *SharedSignalDef[T]) Prepend(prefix string) {
	sig := s.Signal()
	if sig != nil {
		sig.Prepend(prefix)
	}
}

// =============================================================================
// Proxy Methods - Slice Operations
// =============================================================================

// AppendItem appends an item to a slice value.
// Panics if the signal's type is not a slice.
func (s *SharedSignalDef[T]) AppendItem(item any) {
	sig := s.Signal()
	if sig != nil {
		sig.AppendItem(item)
	}
}

// PrependItem prepends an item to a slice value.
// Panics if the signal's type is not a slice.
func (s *SharedSignalDef[T]) PrependItem(item any) {
	sig := s.Signal()
	if sig != nil {
		sig.PrependItem(item)
	}
}

// InsertAt inserts an item at the given index.
// Panics if the signal's type is not a slice.
func (s *SharedSignalDef[T]) InsertAt(index int, item any) {
	sig := s.Signal()
	if sig != nil {
		sig.InsertAt(index, item)
	}
}

// RemoveAt removes the item at the given index.
// Panics if the signal's type is not a slice.
func (s *SharedSignalDef[T]) RemoveAt(index int) {
	sig := s.Signal()
	if sig != nil {
		sig.RemoveAt(index)
	}
}

// RemoveFirst removes the first item from the slice.
// Panics if the signal's type is not a slice.
func (s *SharedSignalDef[T]) RemoveFirst() {
	sig := s.Signal()
	if sig != nil {
		sig.RemoveFirst()
	}
}

// RemoveLast removes the last item from the slice.
// Panics if the signal's type is not a slice.
func (s *SharedSignalDef[T]) RemoveLast() {
	sig := s.Signal()
	if sig != nil {
		sig.RemoveLast()
	}
}

// RemoveWhere removes all items that satisfy the predicate.
// Panics if the signal's type is not a slice.
func (s *SharedSignalDef[T]) RemoveWhere(predicate func(any) bool) {
	sig := s.Signal()
	if sig != nil {
		sig.RemoveWhere(predicate)
	}
}

// SetAt sets the item at the given index.
// Panics if the signal's type is not a slice.
func (s *SharedSignalDef[T]) SetAt(index int, item any) {
	sig := s.Signal()
	if sig != nil {
		sig.SetAt(index, item)
	}
}

// UpdateAt updates the item at the given index.
// Panics if the signal's type is not a slice.
func (s *SharedSignalDef[T]) UpdateAt(index int, fn func(any) any) {
	sig := s.Signal()
	if sig != nil {
		sig.UpdateAt(index, fn)
	}
}

// UpdateWhere updates all items that satisfy the predicate.
// Panics if the signal's type is not a slice.
func (s *SharedSignalDef[T]) UpdateWhere(predicate func(any) bool, fn func(any) any) {
	sig := s.Signal()
	if sig != nil {
		sig.UpdateWhere(predicate, fn)
	}
}

// Filter keeps only items that satisfy the predicate.
// Panics if the signal's type is not a slice.
func (s *SharedSignalDef[T]) Filter(predicate func(any) bool) {
	sig := s.Signal()
	if sig != nil {
		sig.Filter(predicate)
	}
}

// =============================================================================
// Proxy Methods - Map Operations
// =============================================================================

// SetKey sets a key-value pair in a map.
// Panics if the signal's type is not a map.
func (s *SharedSignalDef[T]) SetKey(key, value any) {
	sig := s.Signal()
	if sig != nil {
		sig.SetKey(key, value)
	}
}

// RemoveKey removes a key from a map.
// Panics if the signal's type is not a map.
func (s *SharedSignalDef[T]) RemoveKey(key any) {
	sig := s.Signal()
	if sig != nil {
		sig.RemoveKey(key)
	}
}

// UpdateKey updates a key's value using the provided function.
// Panics if the signal's type is not a map.
func (s *SharedSignalDef[T]) UpdateKey(key any, fn func(any) any) {
	sig := s.Signal()
	if sig != nil {
		sig.UpdateKey(key, fn)
	}
}

// HasKey returns true if the key exists in the map.
// Panics if the signal's type is not a map.
// Returns false if called outside of a session context.
func (s *SharedSignalDef[T]) HasKey(key any) bool {
	sig := s.Signal()
	if sig == nil {
		return false
	}
	return sig.HasKey(key)
}

// =============================================================================
// Proxy Methods - Common Operations
// =============================================================================

// Clear sets the value to its empty state.
// Panics if the signal's type is not string, slice, or map.
func (s *SharedSignalDef[T]) Clear() {
	sig := s.Signal()
	if sig != nil {
		sig.Clear()
	}
}

// Len returns the length of the value.
// Panics if the signal's type is not string, slice, or map.
// Returns 0 if called outside of a session context.
func (s *SharedSignalDef[T]) Len() int {
	sig := s.Signal()
	if sig == nil {
		return 0
	}
	return sig.Len()
}

// =============================================================================
// Simple Session Signal Store Implementation
// =============================================================================

// SimpleSessionSignalStore is a basic implementation of SessionSignalStore
// using sync.Map for concurrent access.
type SimpleSessionSignalStore struct {
	signals sync.Map // map[uint64]any
}

// NewSimpleSessionSignalStore creates a new simple session signal store.
func NewSimpleSessionSignalStore() *SimpleSessionSignalStore {
	return &SimpleSessionSignalStore{}
}

// GetOrCreateSignal retrieves an existing signal or creates a new one.
func (s *SimpleSessionSignalStore) GetOrCreateSignal(id uint64, createFn func() any) any {
	// Try to load existing
	if val, ok := s.signals.Load(id); ok {
		return val
	}

	// Create new and try to store
	newVal := createFn()
	actual, _ := s.signals.LoadOrStore(id, newVal)
	return actual
}
