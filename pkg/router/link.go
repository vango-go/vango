package router

import (
	"github.com/vango-dev/vango/v2/pkg/vdom"
)

// Link creates an anchor element with client-side navigation.
// When clicked, the thin client intercepts and sends a navigate event
// to the server instead of performing a full page reload.
func Link(href string, children ...any) *vdom.VNode {
	return vdom.A(
		vdom.Href(href),
		vdom.Attr{Key: "data-link", Value: "true"},
		children,
	)
}

// LinkWithPrefetch creates a link that prefetches the target page on hover.
// This provides faster navigation by loading the target page before the user clicks.
func LinkWithPrefetch(href string, children ...any) *vdom.VNode {
	return vdom.A(
		vdom.Href(href),
		vdom.Attr{Key: "data-link", Value: "true"},
		vdom.Attr{Key: "data-prefetch", Value: "true"},
		children,
	)
}

// ActiveLink creates a link that adds an active class when the current path matches.
// The activeClass is applied when href matches the current path.
// The exactMatch parameter controls whether the match must be exact or can be a prefix.
func ActiveLink(href string, activeClass string, exactMatch bool, children ...any) *vdom.VNode {
	attrs := []any{
		vdom.Href(href),
		vdom.Attr{Key: "data-link", Value: "true"},
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
	return vdom.Attr{Key: "data-prefetch", Value: "true"}
}

// DataLink creates an anchor attribute that enables client-side navigation.
func DataLink() vdom.Attr {
	return vdom.Attr{Key: "data-link", Value: "true"}
}
