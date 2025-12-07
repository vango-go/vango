// Package hooks provides client-side interaction hooks for Vango components.
//
// Hooks enable high-performance client-side physics and interactions (like drag-and-drop)
// while keeping state management on the server.
//
// Usage:
//
//	Div(
//	    Hook("Sortable", map[string]any{"group": "tasks"}),
//	    OnEvent("reorder", func(e HookEvent) { ... }),
//	)
package hooks
