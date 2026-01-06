package vango

// =============================================================================
// Global Signal
// =============================================================================

// GlobalSignal wraps a Signal that is shared across ALL sessions.
// It embeds *Signal[T] so all Signal methods are directly available.
//
// Note: The spec (§3.9.4) states that NewGlobalSignal should return Signal[T].
// GlobalSignal embeds *Signal[T], so all methods are available and behavior is
// identical. The wrapper type provides clear intent that this signal is global.
//
// Use global signals for truly application-wide state like:
// - Server status indicators
// - Application configuration
// - Global counters/metrics
// - Shared notification queues
//
// WARNING: Global signals are shared across all users. Be careful with
// sensitive data and ensure thread-safe access patterns.
//
// Example:
//
//	// Package-level definition
//	var ServerStatus = vango.NewGlobalSignal("online")
//	var OnlineUserCount = vango.NewGlobalSignal(0)
//
//	// In a component
//	func StatusBar() vango.Component {
//	    return vango.Func(func() *vango.VNode {
//	        status := ServerStatus.Get()  // Same value for all users
//	        count := OnlineUserCount.Get()
//	        return Div(
//	            Span(Textf("Status: %s", status)),
//	            Span(Textf("Users: %d", count)),
//	        )
//	    })
//	}
type GlobalSignal[T any] struct {
	*Signal[T]
}

// NewGlobalSignal creates a signal shared across ALL sessions.
// The returned signal is initialized immediately and persists for the
// lifetime of the application.
//
// Example:
//
//	var ServerStatus = vango.NewGlobalSignal("online")
//	var GlobalCounter = vango.NewGlobalSignal(0)
//	var AppConfig = vango.NewGlobalSignal(Config{Debug: false})
func NewGlobalSignal[T any](initial T, opts ...SignalOption) *GlobalSignal[T] {
	return &GlobalSignal[T]{
		Signal: NewSignal(initial, opts...),
	}
}

// Note: All Signal[T] methods (Get, Set, Update, Peek, Inc, Dec, Toggle, etc.)
// are available through embedding. No proxy methods needed.
