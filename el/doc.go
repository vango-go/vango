// Package el provides the UI DSL for Vango.
//
// It re-exports HTML element constructors, attribute helpers, event helpers,
// and common VDOM utilities from github.com/vango-go/vango/pkg/vdom.
//
// Typical usage:
//
//	import (
//	    "github.com/vango-go/vango/pkg/vango"
//	    . "github.com/vango-go/vango/el"
//	)
//
// This keeps the DSL in a dedicated package while the reactive APIs live in vango.
package el
