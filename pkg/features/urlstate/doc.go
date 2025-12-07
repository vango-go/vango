// Package urlstate provides synchronization between component state and URL parameters.
//
// It enables deep linking, browser navigation integration, and shareable URLs by
// binding reactive signals to query parameters or the URL hash.
//
// Features:
//   - Type-safe parameter handling (generics)
//   - Automatic type coercion
//   - Debounced updates
//   - Serialization/Deserialization customization
//   - History management (push vs replace)
//
// Usage:
//
//	// Bind "search" query param to a string signal
//	search := urlstate.Use("search", "")
//
//	// Update signal -> updates URL
//	search.Set("new query")
//
//	// Read signal <- reflects URL changes
//	val := search.Get()
package urlstate
