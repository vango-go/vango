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

	// ==========================================================================
	// Phase 16: Effect-local call-site state for helpers like GoLatest
	// ==========================================================================

	// callSiteData stores per-call-site state within this effect.
	// Keyed by call-site index (incrementing counter during each effect run).
	// Used by GoLatest to maintain state across effect reruns.
	callSiteData map[int]any

	// allowWrites indicates if this effect has the AllowWrites option.
	// When true, signal writes during the effect body don't trigger warnings.
	allowWrites bool

	// txName is the transaction name for this effect (for observability).
	txName string
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
	oldListener := setCurrentListener(e)

	// Set effect-local tracking context (Phase 16)
	// This enables GoLatest and other helpers to access per-call-site state
	oldEffect := setCurrentEffect(e)
	resetEffectCallSiteIdx()
	oldInBody := setInEffectBody(true)
	oldAllowWrites := setEffectAllowWrites(e.allowWrites)

	// Run the effect function
	e.cleanup = e.fn()

	// Restore tracking context (effect body is done)
	setEffectAllowWrites(oldAllowWrites)
	setInEffectBody(oldInBody)
	setCurrentEffect(oldEffect)

	// Restore previous listener
	setCurrentListener(oldListener)
}

// GetCallSiteData retrieves stored state for a specific call-site index.
// Returns nil if no state has been stored for this call-site.
// Used by effect helpers like GoLatest to maintain state across effect reruns.
func (e *Effect) GetCallSiteData(idx int) any {
	if e.callSiteData == nil {
		return nil
	}
	return e.callSiteData[idx]
}

// SetCallSiteData stores state for a specific call-site index.
// Used by effect helpers like GoLatest to maintain state across effect reruns.
func (e *Effect) SetCallSiteData(idx int, data any) {
	if e.callSiteData == nil {
		e.callSiteData = make(map[int]any)
	}
	e.callSiteData[idx] = data
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

// EffectOption is an option for configuring an Effect.
type EffectOption interface {
	isEffectOption()
	applyEffect(e *Effect)
}

type effectOptionFunc func(*Effect)

func (f effectOptionFunc) isEffectOption()         {}
func (f effectOptionFunc) applyEffect(e *Effect) { f(e) }

// AllowWrites marks an effect as intentionally performing signal writes.
// Without this option, signal writes during the effect body will trigger
// a warning (StrictEffectWarn) or panic (StrictEffectPanic).
//
// Most patterns don't need this - writes inside ctx.Dispatch callbacks
// (as with Interval, Subscribe, GoLatest) are not effect-time writes.
//
// Use this only for rare cases like synchronous initialization:
//
//	vango.CreateEffect(func() vango.Cleanup {
//	    syncedValue.Set(legacySystem.ReadCurrentSync())
//	    return nil
//	}, vango.AllowWrites())
//
// See SPEC_ADDENDUM.md §A.3.
func AllowWrites() EffectOption {
	return effectOptionFunc(func(e *Effect) {
		e.allowWrites = true
	})
}

// EffectTxName sets the transaction name for the effect.
// This name appears in warnings and DevTools entries for this effect.
// It does NOT propagate to helper transactions (Interval/Subscribe/GoLatest).
func EffectTxName(name string) EffectOption {
	return effectOptionFunc(func(e *Effect) {
		e.txName = name
	})
}

// CreateEffect creates and runs a new effect within the current owner context.
// The effect function runs immediately and re-runs when any signal or memo
// it reads changes. If the function returns a Cleanup, it will be called
// before the effect re-runs or when the effect is disposed.
//
// Options:
//   - AllowWrites() - Allow signal writes during effect body without warning
//   - EffectTxName(name) - Set transaction name for observability
//
// Example:
//
//	CreateEffect(func() Cleanup {
//	    fmt.Println("Count is:", count.Get())
//	    return func() { fmt.Println("Cleanup") }
//	})
func CreateEffect(fn func() Cleanup, opts ...EffectOption) *Effect {
	owner := getCurrentOwner()

	// Track hook call for dev-mode order validation
	if owner != nil {
		owner.TrackHook(HookEffect)
	}

	e := &Effect{
		id:    nextID(),
		fn:    fn,
		owner: owner,
	}

	// Apply options
	for _, opt := range opts {
		opt.applyEffect(e)
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

