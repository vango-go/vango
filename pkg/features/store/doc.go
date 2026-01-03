// Package store provides global and session-scoped state management.
//
// It allows defining signals that are shared across components, either globally
// (across all users) or scoped to a specific user session.
//
// Usage:
//
//	// Define a shared signal (session-scoped)
//	var Cart = store.NewSharedSignal([]Item{})
//
//	// Define a global signal (app-wide)
//	var ServerStatus = store.NewGlobalSignal("online")
//
//	func MyComponent() *vdom.VNode {
//	    // Access signal
//	    items := Cart.Get()
//	    status := ServerStatus.Get()
//	    ...
//	}
//
// Integration:
// The server runtime must initialize the session store context on the root owner:
//
//	vango.WithOwner(session.Owner(), func() {
//	    vango.SetContext(store.SessionKey, store.NewSessionStore())
//	})
package store
