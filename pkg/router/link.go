package router

import (
	"github.com/vango-go/vango/pkg/vdom"
)

// Link creates an anchor element with SPA navigation enabled.
// When clicked, the thin client intercepts and sends a navigate event
// to the server instead of performing a full page reload.
//
// Deprecated: Use el.Link() instead. With the dot import of el, this is
// simply Link("/path", Text("label")). This function remains for backwards
// compatibility but will be removed in a future version.
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
// Deprecated: Use el.LinkPrefetch() instead. With the dot import of el, this is
// simply LinkPrefetch("/path", Text("label")). This function remains for backwards
// compatibility but will be removed in a future version.
func LinkWithPrefetch(href string, children ...any) *vdom.VNode {
	return vdom.A(
		vdom.Href(href),
		vdom.Attr{Key: "data-vango-link", Value: ""},
		vdom.Attr{Key: "data-prefetch", Value: ""},
		children,
	)
}

// ActiveLink creates a link with custom active class handling via data attributes.
// This is for power users who need custom active class names or client-side
// active state detection.
//
// For most use cases, prefer el.NavLink(ctx, path, children...) which uses
// server-side path matching to add the "active" class directly.
//
// The activeClass is set via data-active-class attribute.
// The exactMatch parameter controls whether the match must be exact (data-active-exact).
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

// NavLink creates a link with default active class handling.
//
// Deprecated: Use el.NavLink(ctx, path, children...) instead. The new NavLink
// uses server-side path matching which works with SSR and doesn't require
// client-side JavaScript. This function remains for backwards compatibility
// but will be removed in a future version.
func NavLink(href string, children ...any) *vdom.VNode {
	return ActiveLink(href, "active", true, children...)
}

// Prefetch creates a prefetch attribute for links.
// Add this to any anchor element to enable hover prefetching.
//
// Example:
//
//	A(Href("/about"), Prefetch(), Text("About"))
func Prefetch() vdom.Attr {
	return vdom.Attr{Key: "data-prefetch", Value: ""}
}

// VangoLink creates an anchor attribute that enables client-side navigation.
// This is the canonical marker for SPA navigation.
//
// Example:
//
//	A(Href("/about"), VangoLink(), Text("About"))
func VangoLink() vdom.Attr {
	return vdom.Attr{Key: "data-vango-link", Value: ""}
}

// DataLink creates an anchor attribute that enables client-side navigation.
//
// Deprecated: Use VangoLink() instead. This function is kept for backwards
// compatibility but will be removed in a future version.
func DataLink() vdom.Attr {
	return vdom.Attr{Key: "data-vango-link", Value: ""}
}
