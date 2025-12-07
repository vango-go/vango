// Package vdom provides the Virtual DOM implementation for Vango.
//
// The Virtual DOM (VDOM) provides an in-memory representation of the UI
// that can be efficiently diffed to produce minimal DOM updates. In Vango's
// server-driven architecture, the VDOM lives on the server and diffs produce
// binary patches sent to the client.
//
// # Core Types
//
// VNode is the fundamental building block representing elements, text,
// fragments, components, and raw HTML. Props holds attributes and event
// handlers. Attr and EventHandler are used to build Props.
//
// # Element API
//
// Elements are created using variadic factory functions:
//
//	Div(Class("card"), ID("main"),
//	    H1(Text("Title")),
//	    P(Text("Content")),
//	    OnClick(handler),
//	)
//
// # Diffing
//
// The Diff function compares two VNode trees and returns a slice of Patch
// operations. Keyed reconciliation is used when children have Key attributes.
//
// # Hydration
//
// AssignHIDs walks the tree and assigns hydration IDs to interactive elements
// (those with event handlers). These IDs link server VNodes to client DOM.
package vdom
