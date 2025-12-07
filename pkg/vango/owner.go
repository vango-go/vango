package vango

import (
	"sync"
	"sync/atomic"
)

// Owner represents a component scope that owns reactive primitives.
// When an Owner is disposed, all signals, memos, effects, and child owners
// it contains are also disposed. This ensures proper cleanup and prevents
// memory leaks.
//
// Owners form a hierarchy: each component creates an Owner that is a child
// of its parent component's Owner. This mirrors the component tree structure.
type Owner struct {
	id uint64

	// parent is the parent Owner in the hierarchy.
	// nil for the root Owner (typically the session).
	parent *Owner

	// children are child Owners (sub-components).
	children   []*Owner
	childrenMu sync.Mutex

	// effects owned by this scope.
	effects   []*Effect
	effectsMu sync.Mutex

	// cleanups are manual cleanup functions registered via OnCleanup.
	cleanups   []func()
	cleanupsMu sync.Mutex

	// pendingEffects are effects scheduled to run after render.
	pendingEffects   []*Effect
	pendingEffectsMu sync.Mutex

	// disposed indicates whether this Owner has been disposed.
	disposed atomic.Bool
}

// NewOwner creates a new Owner with the given parent.
// The new Owner is automatically registered as a child of the parent.
// If parent is nil, creates a root Owner.
func NewOwner(parent *Owner) *Owner {
	o := &Owner{
		id:     nextID(),
		parent: parent,
	}

	if parent != nil {
		parent.addChild(o)
	}

	return o
}

// ID returns the unique identifier for this Owner.
func (o *Owner) ID() uint64 {
	return o.id
}

// Parent returns the parent Owner, or nil if this is a root Owner.
func (o *Owner) Parent() *Owner {
	return o.parent
}

// IsDisposed returns true if this Owner has been disposed.
func (o *Owner) IsDisposed() bool {
	return o.disposed.Load()
}

// addChild registers a child Owner.
func (o *Owner) addChild(child *Owner) {
	o.childrenMu.Lock()
	defer o.childrenMu.Unlock()
	o.children = append(o.children, child)
}

// removeChild removes a child Owner from this Owner's children.
func (o *Owner) removeChild(child *Owner) {
	o.childrenMu.Lock()
	defer o.childrenMu.Unlock()

	for i, c := range o.children {
		if c == child {
			o.children = append(o.children[:i], o.children[i+1:]...)
			return
		}
	}
}

// registerEffect adds an effect to this Owner.
// The effect will be disposed when this Owner is disposed.
func (o *Owner) registerEffect(e *Effect) {
	if o.disposed.Load() {
		return
	}

	o.effectsMu.Lock()
	defer o.effectsMu.Unlock()
	o.effects = append(o.effects, e)
}

// OnCleanup registers a cleanup function to run when this Owner is disposed.
func (o *Owner) OnCleanup(fn func()) {
	if o.disposed.Load() {
		// Already disposed, run cleanup immediately
		fn()
		return
	}

	o.cleanupsMu.Lock()
	defer o.cleanupsMu.Unlock()
	o.cleanups = append(o.cleanups, fn)
}

// scheduleEffect adds an effect to the pending effects queue.
// Effects are run after the render phase via RunPendingEffects.
func (o *Owner) scheduleEffect(e *Effect) {
	if o.disposed.Load() {
		return
	}

	o.pendingEffectsMu.Lock()
	defer o.pendingEffectsMu.Unlock()
	o.pendingEffects = append(o.pendingEffects, e)
}

// RunPendingEffects executes all pending effects.
// This is called after the render phase to run scheduled effects.
// The server runtime calls this after event handlers execute.
func (o *Owner) RunPendingEffects() {
	if o.disposed.Load() {
		return
	}

	o.pendingEffectsMu.Lock()
	effects := o.pendingEffects
	o.pendingEffects = nil
	o.pendingEffectsMu.Unlock()

	for _, e := range effects {
		if e.pending.Load() {
			e.run()
		}
	}
}

// Dispose disposes this Owner and all its children, effects, and cleanups.
// Children are disposed in reverse order (last created first).
// After disposal, the Owner cannot be used.
func (o *Owner) Dispose() {
	if o.disposed.Swap(true) {
		// Already disposed
		return
	}

	// Remove from parent's children list
	if o.parent != nil {
		o.parent.removeChild(o)
	}

	// Dispose children in reverse order
	o.childrenMu.Lock()
	children := make([]*Owner, len(o.children))
	copy(children, o.children)
	o.children = nil
	o.childrenMu.Unlock()

	for i := len(children) - 1; i >= 0; i-- {
		children[i].Dispose()
	}

	// Dispose effects
	o.effectsMu.Lock()
	effects := o.effects
	o.effects = nil
	o.effectsMu.Unlock()

	for _, e := range effects {
		e.dispose()
	}

	// Run cleanups in reverse order
	o.cleanupsMu.Lock()
	cleanups := o.cleanups
	o.cleanups = nil
	o.cleanupsMu.Unlock()

	for i := len(cleanups) - 1; i >= 0; i-- {
		cleanups[i]()
	}

	// Clear pending effects
	o.pendingEffectsMu.Lock()
	o.pendingEffects = nil
	o.pendingEffectsMu.Unlock()
}
