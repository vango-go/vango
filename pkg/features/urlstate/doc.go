// Package urlstate is EXPERIMENTAL and not part of the v1 public API.
//
// For URL query state synchronization, use the urlparam package instead:
//
//	import "github.com/vango-go/vango"
//
//	// Use vango.URLParam for URL query state
//	search := vango.URLParam(ctx, "search", "")
//
// URLState integrates with the session's URL patch navigator (URL_PUSH / URL_REPLACE)
// when called within a Vango render/handler context.
//
// HashState is experimental and uses a PatchDispatch event ("vango:hash") that the
// thin client interprets as a history hash update.
//
// DO NOT USE IN PRODUCTION.
package urlstate
