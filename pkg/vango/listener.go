package vango

// Listener is anything that can be notified when a dependency changes.
// This interface is implemented by components, memos, and effects.
type Listener interface {
	// MarkDirty notifies the listener that one of its dependencies has changed.
	// For components, this schedules a re-render.
	// For memos, this invalidates the cached value.
	// For effects, this schedules the effect to re-run.
	MarkDirty()

	// ID returns a unique identifier for this listener.
	// Used for deduplication during batch processing.
	ID() uint64
}

// Cleanup is a function returned by effects to clean up resources.
// It is called before the effect re-runs and when the effect is disposed.
type Cleanup func()
