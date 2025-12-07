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
