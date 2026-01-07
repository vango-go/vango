// Package urlstate is EXPERIMENTAL and not part of the v1 public API.
//
// For URL query state synchronization, use the urlparam package instead:
//
//	import "github.com/vango-go/vango"
//
//	// Use vango.URLParam for URL query state
//	search := vango.URLParam(ctx, "search", "")
//
// This package may be completed for hash-based routing in future versions.
// The implementation is incomplete: URL reading and writing are not integrated
// with the router/navigator system.
//
// DO NOT USE IN PRODUCTION.
package urlstate
