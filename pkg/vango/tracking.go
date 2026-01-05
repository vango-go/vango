package vango

import (
	"runtime"
	"sync"
)

// TrackingContext holds the reactive state for a goroutine.
// Each goroutine has its own tracking context to support concurrent
// component rendering and signal access.
type TrackingContext struct {
	// currentOwner is the Owner that will own newly created signals/effects.
	// Set during component rendering to establish ownership hierarchy.
	currentOwner *Owner

	// currentListener is what's currently tracking dependencies.
	// When a signal is read, it subscribes this listener.
	// nil means no tracking (reads don't create subscriptions).
	currentListener Listener

	// batchDepth tracks nested Batch() calls.
	// When > 0, signal updates queue notifications instead of firing immediately.
	batchDepth int

	// pendingUpdates accumulates listeners to notify when batch completes.
	// Deduplicated by ID before notification.
	pendingUpdates []Listener

	// currentCtx holds the current runtime context (server.Ctx).
	// Set during event handling and render to provide access via UseCtx().
	// Stored as any to avoid circular imports with server package.
	currentCtx any

	// ==========================================================================
	// Phase 16: Effect-local call-site tracking for helpers like GoLatest
	// ==========================================================================

	// currentEffect points to the Effect currently executing its body.
	// Set during effect.run() to allow helpers to store call-site state.
	// nil when not inside an effect body.
	currentEffect *Effect

	// effectCallSiteIdx tracks the current call-site index within an effect run.
	// Incremented each time a helper (like GoLatest) is invoked within an effect.
	// Reset to 0 at the start of each effect run.
	effectCallSiteIdx int

	// inEffectBody is true while executing the synchronous body of an effect.
	// Used for effect-time write detection (SPEC_ADDENDUM.md §A.3).
	// False during Dispatch callbacks and goroutines spawned by effects.
	inEffectBody bool

	// effectAllowWrites is true if the current effect has AllowWrites() option.
	// Used to suppress effect-time write warnings.
	effectAllowWrites bool
}

// trackingContexts stores per-goroutine tracking contexts.
// Using sync.Map for concurrent access from multiple goroutines.
var trackingContexts sync.Map

// getGoroutineID returns a unique identifier for the current goroutine.
// This uses the runtime stack to extract the goroutine ID.
// Note: This is an implementation detail and should not be relied upon externally.
func getGoroutineID() uint64 {
	// Use a buffer to read the stack
	var buf [64]byte
	n := runtime.Stack(buf[:], false)

	// The stack starts with "goroutine <id> "
	// Parse the ID from the stack trace
	var id uint64
	for i := 10; i < n; i++ { // Skip "goroutine "
		if buf[i] == ' ' {
			break
		}
		id = id*10 + uint64(buf[i]-'0')
	}
	return id
}

// getTrackingContext returns the tracking context for the current goroutine.
// If no context exists, creates a new one.
func getTrackingContext() *TrackingContext {
	gid := getGoroutineID()

	if ctx, ok := trackingContexts.Load(gid); ok {
		return ctx.(*TrackingContext)
	}

	// Create new context for this goroutine
	ctx := &TrackingContext{}
	trackingContexts.Store(gid, ctx)
	return ctx
}

// setTrackingContext sets the tracking context for the current goroutine.
// Used internally for context propagation.
func setTrackingContext(ctx *TrackingContext) {
	gid := getGoroutineID()
	if ctx == nil {
		trackingContexts.Delete(gid)
	} else {
		trackingContexts.Store(gid, ctx)
	}
}

// getCurrentListener returns the current listener being tracked.
// Returns nil if no tracking is active.
func getCurrentListener() Listener {
	ctx := getTrackingContext()
	return ctx.currentListener
}

// setCurrentListener sets the current listener for dependency tracking.
// Returns the previous listener so it can be restored.
func setCurrentListener(l Listener) Listener {
	ctx := getTrackingContext()
	old := ctx.currentListener
	ctx.currentListener = l
	return old
}

// getCurrentOwner returns the current owner for the goroutine.
// Returns nil if no owner context is set.
func getCurrentOwner() *Owner {
	ctx := getTrackingContext()
	return ctx.currentOwner
}

// setCurrentOwner sets the current owner for signal/effect creation.
// Returns the previous owner so it can be restored.
func setCurrentOwner(o *Owner) *Owner {
	ctx := getTrackingContext()
	old := ctx.currentOwner
	ctx.currentOwner = o
	return old
}

// getBatchDepth returns the current batch nesting depth.
func getBatchDepth() int {
	ctx := getTrackingContext()
	return ctx.batchDepth
}

// incrementBatchDepth increases the batch depth by 1.
func incrementBatchDepth() {
	ctx := getTrackingContext()
	ctx.batchDepth++
}

// decrementBatchDepth decreases the batch depth by 1.
// Returns true if batch depth reached 0 (batch complete).
func decrementBatchDepth() bool {
	ctx := getTrackingContext()
	ctx.batchDepth--
	return ctx.batchDepth == 0
}

// queuePendingUpdate adds a listener to the pending updates queue.
// Called during batch mode when a signal is updated.
func queuePendingUpdate(l Listener) {
	ctx := getTrackingContext()
	ctx.pendingUpdates = append(ctx.pendingUpdates, l)
}

// drainPendingUpdates returns and clears the pending updates queue.
// Called when a batch completes to process all queued notifications.
func drainPendingUpdates() []Listener {
	ctx := getTrackingContext()
	updates := ctx.pendingUpdates
	ctx.pendingUpdates = nil
	return updates
}

// WithOwner runs a function with the specified owner as the current owner.
// This is used when spawning goroutines that need to create signals/effects
// that belong to a specific component.
//
// Example:
//
//	go func() {
//	    WithOwner(parentOwner, func() {
//	        // Signals created here belong to parentOwner
//	        signal := NewSignal(0)
//	    })
//	}()
func WithOwner(owner *Owner, fn func()) {
	old := setCurrentOwner(owner)
	defer setCurrentOwner(old)
	fn()
}

// WithListener runs a function with the specified listener for tracking.
// This is used internally to set up dependency tracking during rendering.
func WithListener(l Listener, fn func()) {
	old := setCurrentListener(l)
	defer setCurrentListener(old)
	fn()
}

// cleanupGoroutineContext removes the tracking context for the current goroutine.
// Should be called when a goroutine is about to exit to prevent memory leaks.
// This is optional - contexts are lightweight and will be overwritten if reused.
func cleanupGoroutineContext() {
	gid := getGoroutineID()
	trackingContexts.Delete(gid)
}

// getCurrentCtx returns the current runtime context for the goroutine.
// Returns nil if no context is set.
func getCurrentCtx() any {
	ctx := getTrackingContext()
	return ctx.currentCtx
}

// setCurrentCtx sets the current runtime context.
// Returns the previous context so it can be restored.
func setCurrentCtx(c any) any {
	ctx := getTrackingContext()
	old := ctx.currentCtx
	ctx.currentCtx = c
	return old
}

// WithCtx runs a function with the specified runtime context.
// This is used by the server to establish context during event handling
// and component rendering.
//
// Example (internal use by server):
//
//	WithCtx(ctx, func() {
//	    // UseCtx() will return ctx here
//	    component.Render()
//	})
func WithCtx(c any, fn func()) {
	old := setCurrentCtx(c)
	defer setCurrentCtx(old)
	fn()
}

// TrackHook records a hook call for the current owner.
// This is used by feature packages (resource, form, urlparam) to participate
// in hook-order validation during dev mode.
//
// External packages should call this at the beginning of their hook constructors:
//
//	func UseForm[T any](initial T) *Form[T] {
//	    vango.TrackHook(vango.HookForm)
//	    // ... rest of implementation
//	}
//
// If no owner is set (outside of render context), this is a no-op.
func TrackHook(ht HookType) {
	if owner := getCurrentOwner(); owner != nil {
		owner.TrackHook(ht)
	}
}

// UseHookSlot returns stable hook state from the current owner.
// On first render, returns nil (caller should create value and call SetHookSlot).
// On subsequent renders, returns the previously stored value.
//
// This provides stable identity for hooks (like URLParam, Resource) across renders.
//
// Usage pattern:
//
//	func SomeHook[T any]() *T {
//	    slot := vango.UseHookSlot()
//	    if slot != nil {
//	        return slot.(*T)  // Subsequent render
//	    }
//	    instance := &T{...}  // First render
//	    vango.SetHookSlot(instance)
//	    return instance
//	}
//
// If no owner is set (outside render context), returns nil.
func UseHookSlot() any {
	if owner := getCurrentOwner(); owner != nil {
		return owner.UseHookSlot()
	}
	return nil
}

// SetHookSlot stores a value in the current hook slot.
// Must be called after UseHookSlot returns nil (first render only).
//
// If no owner is set (outside render context), this is a no-op.
func SetHookSlot(value any) {
	if owner := getCurrentOwner(); owner != nil {
		owner.SetHookSlot(value)
	}
}

// =============================================================================
// Phase 16: Effect-local call-site tracking accessors
// =============================================================================

// getCurrentEffect returns the currently executing Effect.
// Returns nil if not inside an effect body.
func getCurrentEffect() *Effect {
	ctx := getTrackingContext()
	return ctx.currentEffect
}

// setCurrentEffect sets the currently executing Effect.
// Returns the previous effect so it can be restored.
func setCurrentEffect(e *Effect) *Effect {
	ctx := getTrackingContext()
	old := ctx.currentEffect
	ctx.currentEffect = e
	return old
}

// getEffectCallSiteIdx returns the current call-site index within an effect.
func getEffectCallSiteIdx() int {
	ctx := getTrackingContext()
	return ctx.effectCallSiteIdx
}

// incrementEffectCallSiteIdx increments and returns the call-site index.
// Called by effect helpers (GoLatest, etc.) to get their unique call-site ID.
func incrementEffectCallSiteIdx() int {
	ctx := getTrackingContext()
	idx := ctx.effectCallSiteIdx
	ctx.effectCallSiteIdx++
	return idx
}

// resetEffectCallSiteIdx resets the call-site index to 0.
// Called at the start of each effect run.
func resetEffectCallSiteIdx() {
	ctx := getTrackingContext()
	ctx.effectCallSiteIdx = 0
}

// isInEffectBody returns true if currently executing the synchronous body of an effect.
// False during Dispatch callbacks and goroutines spawned by effects.
func isInEffectBody() bool {
	ctx := getTrackingContext()
	return ctx.inEffectBody
}

// setInEffectBody sets whether we're inside an effect body.
// Returns the previous value so it can be restored.
func setInEffectBody(v bool) bool {
	ctx := getTrackingContext()
	old := ctx.inEffectBody
	ctx.inEffectBody = v
	return old
}

// effectHasAllowWrites returns true if the current effect has AllowWrites() option.
func effectHasAllowWrites() bool {
	ctx := getTrackingContext()
	return ctx.effectAllowWrites
}

// setEffectAllowWrites sets whether the current effect allows writes.
// Returns the previous value so it can be restored.
func setEffectAllowWrites(v bool) bool {
	ctx := getTrackingContext()
	old := ctx.effectAllowWrites
	ctx.effectAllowWrites = v
	return old
}

// checkEffectTimeWrite checks if a signal write is happening during an effect body
// and emits warnings or panics based on EffectStrictMode.
// This should be called at the beginning of all signal mutation methods.
func checkEffectTimeWrite(method string) {
	// Only check if we're in an effect body
	if !isInEffectBody() {
		return
	}

	// Check if effect has AllowWrites
	if effectHasAllowWrites() {
		return
	}

	// Effect-time write without AllowWrites
	switch EffectStrictMode {
	case StrictEffectOff:
		// No enforcement
		return

	case StrictEffectWarn:
		// Get caller location for warning
		// Note: In production, this would use runtime.Caller for file:line
		warningMessage := "Warning: Effect wrote signal via " + method + "()\n" +
			"  → For periodic updates, use vango.Interval()\n" +
			"  → For event streams, use vango.Subscribe()\n" +
			"  → For async work, use Effect + vango.GoLatest()\n" +
			"  → For intentional writes, add vango.AllowWrites()"
		// Log warning (would use proper logging in production)
		println(warningMessage)

	case StrictEffectPanic:
		panic("Effect wrote signal via " + method + "() without AllowWrites()")
	}
}

// GetEffectCallSiteState retrieves or creates typed state for the current call-site
// within the currently executing Effect. This is the primary API for effect helpers
// like GoLatest that need to maintain state across effect reruns.
//
// The factory function is called only on first invocation for this call-site.
// Subsequent calls (in effect reruns) return the previously created state.
//
// Returns nil if called outside an effect body.
//
// Usage pattern in effect helpers:
//
//	func GoLatest[K, R any](...) Cleanup {
//	    state := GetEffectCallSiteState(func() *goLatestState[K] {
//	        return &goLatestState[K]{}
//	    })
//	    if state == nil {
//	        panic("GoLatest must be called inside an Effect")
//	    }
//	    // Use state...
//	}
func GetEffectCallSiteState[T any](factory func() *T) *T {
	effect := getCurrentEffect()
	if effect == nil {
		return nil
	}

	// Get unique index for this call-site within the effect
	idx := incrementEffectCallSiteIdx()

	// Check if state already exists for this call-site
	existing := effect.GetCallSiteData(idx)
	if existing != nil {
		return existing.(*T)
	}

	// First time: create and store new state
	state := factory()
	effect.SetCallSiteData(idx, state)
	return state
}
