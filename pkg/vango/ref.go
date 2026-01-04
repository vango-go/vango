package vango

import "sync"

// HookRef is the hook type for NewRef (not exported in owner.go yet, we'll add it).
// Note: We add HookRef to the HookType enum in owner.go.

// Ref holds a mutable reference to a value.
// In client runtimes (WASM), this is typically used to hold DOM element references.
// In server-driven mode, refs have limited use (DOM handles don't exist on server).
//
// Ref[T] is safe for concurrent access.
type Ref[T any] struct {
	value T
	isSet bool
	mu    sync.RWMutex
}

// NewRef creates a new Ref with the given initial value.
//
// This is a hook-like API and MUST be called unconditionally during render.
// See §3.1.3 Hook-Order Semantics.
//
// Example:
//
//	inputRef := vango.NewRef[js.Value](nil)
//	return Input(
//	    Ref(inputRef),
//	    Type("text"),
//	)
func NewRef[T any](initial T) *Ref[T] {
	// Track hook call for dev-mode order validation
	TrackHook(HookRef)

	return &Ref[T]{
		value: initial,
		isSet: false, // Not set until attached to DOM
	}
}

// Current returns the current value of the ref.
// For DOM refs, this returns the DOM element after mount.
func (r *Ref[T]) Current() T {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.value
}

// Set sets the ref's value.
// This is typically called by the runtime when attaching to a DOM element.
func (r *Ref[T]) Set(value T) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.value = value
	r.isSet = true
}

// IsSet returns true if the ref has been set (attached to DOM).
func (r *Ref[T]) IsSet() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.isSet
}

// Clear resets the ref to its zero value.
func (r *Ref[T]) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	var zero T
	r.value = zero
	r.isSet = false
}
