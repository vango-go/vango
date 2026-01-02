package vango

import "fmt"

// DebugMode enables debug logging throughout the vango package.
// When true, operations like TxNamed will log transaction boundaries.
// This should be set at startup and not changed during runtime.
var DebugMode bool

// Batch groups multiple signal updates into a single notification phase.
// All signal updates within the batch function are collected, deduplicated,
// and then all affected listeners are notified once when the batch completes.
//
// This is useful for updating multiple related signals without triggering
// intermediate re-renders.
//
// Batches can be nested. Notifications only fire when the outermost batch completes.
//
// Example:
//
//	Batch(func() {
//	    firstName.Set("John")
//	    lastName.Set("Doe")
//	    age.Set(30)
//	})
//	// Component re-renders once with all three changes
func Batch(fn func()) {
	incrementBatchDepth()

	defer func() {
		if decrementBatchDepth() {
			// Batch complete, process pending updates
			processPendingUpdates()
		}
	}()

	fn()
}

// processPendingUpdates deduplicates and notifies all pending listeners.
func processPendingUpdates() {
	updates := drainPendingUpdates()
	if len(updates) == 0 {
		return
	}

	// Deduplicate by listener ID
	seen := make(map[uint64]bool, len(updates))
	unique := make([]Listener, 0, len(updates))

	for _, listener := range updates {
		id := listener.ID()
		if !seen[id] {
			seen[id] = true
			unique = append(unique, listener)
		}
	}

	// Notify all unique listeners
	for _, listener := range unique {
		listener.MarkDirty()
	}
}

// Untracked runs a function without tracking signal reads as dependencies.
// This is useful when you need to read a signal's value without creating
// a subscription.
//
// Example:
//
//	Untracked(func() {
//	    // Reading count here won't subscribe the current component
//	    value := count.Get()
//	    fmt.Println("Current value:", value)
//	})
//
// Note: For single signal reads, use signal.Peek() instead which is more
// efficient and clearer in intent.
func Untracked(fn func()) {
	old := setCurrentListener(nil)
	defer setCurrentListener(old)
	fn()
}

// UntrackedGet reads a signal's value without creating a dependency.
// This is a convenience function equivalent to signal.Peek().
func UntrackedGet[T any](s *Signal[T]) T {
	return s.Peek()
}

// Tx runs fn as a transaction, grouping all signal updates.
// This is an alias for Batch() that aligns with the transaction
// terminology used in the spec (§3.9.4 and §7).
//
// All signal updates within the transaction are collected, deduplicated,
// and notifications only fire when the transaction completes.
//
// Example:
//
//	Tx(func() {
//	    user.Set(newUser)
//	    profile.Set(newProfile)
//	    settings.Set(newSettings)
//	})
//	// Single re-render with all changes
func Tx(fn func()) {
	Batch(fn)
}

// TxNamed runs fn as a named transaction for debugging and tracing.
// The transaction name is logged in debug mode for observability.
//
// This is useful for understanding which transactions trigger re-renders
// during development and for production tracing.
//
// Example:
//
//	TxNamed("user-profile-update", func() {
//	    user.Set(newUser)
//	    profile.Set(newProfile)
//	})
//	// Debug output: [TX] user-profile-update start/end
func TxNamed(name string, fn func()) {
	if DebugMode {
		fmt.Printf("[TX] %s start\n", name)
		defer fmt.Printf("[TX] %s end\n", name)
	}
	Batch(fn)
}
