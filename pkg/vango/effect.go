package vango

import (
	"sync"
	"sync/atomic"
)

// Effect represents a reactive side effect that runs when its dependencies change.
// Effects are created using the Effect function and are automatically tracked
// for dependencies during their execution.
//
// Effects run immediately when created, and re-run whenever any signal or memo
// they read during execution changes. They can return a Cleanup function that
// will be called before the effect re-runs or when the effect is disposed.
type Effect struct {
	id uint64

	// fn is the effect function to run.
	fn func() Cleanup

	// cleanup is the cleanup function from the last run.
	cleanup Cleanup

	// sources are the signals/memos this effect depends on.
	sources   []*signalBase
	sourcesMu sync.Mutex

	// owner is the Owner that owns this effect.
	owner *Owner

	// pending indicates the effect is scheduled for re-run.
	pending atomic.Bool

	// disposed indicates the effect has been disposed.
	disposed atomic.Bool
}

// MarkDirty marks the effect as needing to re-run.
// Implements the Listener interface.
func (e *Effect) MarkDirty() {
	if e.disposed.Load() {
		return
	}

	// Use CAS to ensure we only schedule once
	if e.pending.CompareAndSwap(false, true) {
		if e.owner != nil {
			e.owner.scheduleEffect(e)
		}
	}
}

// ID returns the unique identifier for this effect.
// Implements the Listener interface.
func (e *Effect) ID() uint64 {
	return e.id
}

// run executes the effect function.
// This is called during initial creation and when dependencies change.
func (e *Effect) run() {
	if e.disposed.Load() {
		return
	}

	// Clear pending flag
	e.pending.Store(false)

	// Run cleanup from previous run
	if e.cleanup != nil {
		e.cleanup()
		e.cleanup = nil
	}

	// Unsubscribe from old sources
	e.sourcesMu.Lock()
	for _, source := range e.sources {
		source.unsubscribe(e)
	}
	e.sources = e.sources[:0]
	e.sourcesMu.Unlock()

	// Track new sources during execution
	old := setCurrentListener(e)

	// Run the effect function
	e.cleanup = e.fn()

	// Restore previous listener
	setCurrentListener(old)
}

// addSource adds a source dependency.
// Called by signals when they are read during effect execution.
func (e *Effect) addSource(source *signalBase) {
	e.sourcesMu.Lock()
	defer e.sourcesMu.Unlock()

	// Check for duplicates
	for _, s := range e.sources {
		if s == source {
			return
		}
	}
	e.sources = append(e.sources, source)
}

// dispose cleans up the effect and unsubscribes from all sources.
func (e *Effect) dispose() {
	if e.disposed.Swap(true) {
		return
	}

	// Run cleanup
	if e.cleanup != nil {
		e.cleanup()
		e.cleanup = nil
	}

	// Unsubscribe from all sources
	e.sourcesMu.Lock()
	for _, source := range e.sources {
		source.unsubscribe(e)
	}
	e.sources = nil
	e.sourcesMu.Unlock()
}

// CreateEffect creates and runs a new effect within the current owner context.
// The effect function runs immediately and re-runs when any signal or memo
// it reads changes. If the function returns a Cleanup, it will be called
// before the effect re-runs or when the effect is disposed.
//
// Example:
//
//	CreateEffect(func() Cleanup {
//	    fmt.Println("Count is:", count.Get())
//	    return func() { fmt.Println("Cleanup") }
//	})
func CreateEffect(fn func() Cleanup) *Effect {
	owner := getCurrentOwner()

	e := &Effect{
		id:    nextID(),
		fn:    fn,
		owner: owner,
	}

	if owner != nil {
		owner.registerEffect(e)
	}

	// Run immediately
	e.run()

	return e
}

// OnMount creates an effect that runs only once on mount.
// This is equivalent to CreateEffect with no reactive dependencies.
//
// Example:
//
//	OnMount(func() {
//	    fmt.Println("Component mounted")
//	})
func OnMount(fn func()) {
	CreateEffect(func() Cleanup {
		fn()
		return nil
	})
}

// OnUnmount registers a function to run when the owner is disposed.
// This is typically used for cleanup when a component unmounts.
//
// Example:
//
//	OnUnmount(func() {
//	    fmt.Println("Component unmounted")
//	})
func OnUnmount(fn func()) {
	owner := getCurrentOwner()
	if owner != nil {
		owner.OnCleanup(fn)
	}
}

// OnUpdate creates an effect that skips the callback on the first run.
// This is useful when you only want to react to changes, not the initial value.
//
// The deps function is called to establish dependencies. The callback is only
// called on subsequent runs when those dependencies change.
//
// Example:
//
//	OnUpdate(
//	    func() { _ = count.Get() },           // deps: read signals to track
//	    func() { fmt.Println("Updated!") },   // callback: only on changes
//	)
func OnUpdate(deps func(), callback func()) {
	first := true
	CreateEffect(func() Cleanup {
		deps() // Always call to track dependencies
		if first {
			first = false
			return nil
		}
		callback()
		return nil
	})
}
