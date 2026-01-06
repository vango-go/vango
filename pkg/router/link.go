package router

import (
	"github.com/vango-go/vango/pkg/vdom"
)

// Link creates an anchor element with SPA navigation enabled.
// When clicked, the thin client intercepts and sends a navigate event
// to the server instead of performing a full page reload.
//
// Per the routing contract (Section 5.1), this sets the data-vango-link
// attribute which is the canonical marker for SPA navigation.
func Link(href string, children ...any) *vdom.VNode {
	return vdom.A(
		vdom.Href(href),
		vdom.Attr{Key: "data-vango-link", Value: ""},
		children,
	)
}

// LinkWithPrefetch creates a link that prefetches the target page on hover.
// This provides faster navigation by loading the target page before the user clicks.
//
// Sets: href, data-vango-link, and data-prefetch.
func LinkWithPrefetch(href string, children ...any) *vdom.VNode {
	return vdom.A(
		vdom.Href(href),
		vdom.Attr{Key: "data-vango-link", Value: ""},
		vdom.Attr{Key: "data-prefetch", Value: ""},
		children,
	)
}

// ActiveLink creates a link that adds an active class when the current path matches.
// The activeClass is applied when href matches the current path.
// The exactMatch parameter controls whether the match must be exact or can be a prefix.
func ActiveLink(href string, activeClass string, exactMatch bool, children ...any) *vdom.VNode {
	attrs := []any{
		vdom.Href(href),
		vdom.Attr{Key: "data-vango-link", Value: ""},
		vdom.Attr{Key: "data-active-class", Value: activeClass},
	}

	if exactMatch {
		attrs = append(attrs, vdom.Attr{Key: "data-active-exact", Value: "true"})
	}

	attrs = append(attrs, children...)
	return vdom.A(attrs...)
}

// NavLink is an alias for ActiveLink with common defaults.
// It adds "active" class when the path matches exactly.
func NavLink(href string, children ...any) *vdom.VNode {
	return ActiveLink(href, "active", true, children...)
}

// Prefetch creates a prefetch attribute for links.
// Add this to any anchor element to enable hover prefetching.
func Prefetch() vdom.Attr {
	return vdom.Attr{Key: "data-prefetch", Value: ""}
}

// VangoLink creates an anchor attribute that enables client-side navigation.
// This is the canonical marker for SPA navigation.
func VangoLink() vdom.Attr {
	return vdom.Attr{Key: "data-vango-link", Value: ""}
}

// DataLink creates an anchor attribute that enables client-side navigation.
// Deprecated: Use VangoLink() instead. This function is kept for backwards
// compatibility but will be removed in a future version.
func DataLink() vdom.Attr {
	return vdom.Attr{Key: "data-vango-link", Value: ""}
}
