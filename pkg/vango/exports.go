package vango

import "github.com/vango-go/vango/pkg/vdom"

// =============================================================================
// Re-exports for Spec Compliance
// =============================================================================
//
// These re-exports allow users to write code matching the spec examples:
//
//     count := vango.NewSignal(0)
//     return vango.Func(func() *vango.VNode { ... })
//
// The elements themselves are still imported from vdom:
//
//     import (
//         "github.com/vango-go/vango/pkg/vango"
//         . "github.com/vango-go/vango/pkg/vdom"
//     )

// Func wraps a render function as a Component.
// This is the primary way to create stateful components.
//
// Example:
//
//	func Counter(initial int) vango.Component {
//	    return vango.Func(func() *vango.VNode {
//	        count := vango.NewSignal(initial)
//	        return Div(
//	            H1(Textf("Count: %d", count.Get())),
//	            Button(OnClick(count.Inc), Text("+")),
//	        )
//	    })
//	}
func Func(render func() *vdom.VNode) vdom.Component {
	return vdom.Func(render)
}

// Component is an alias for vdom.Component for convenience.
// Components are anything that can render to a VNode.
type Component = vdom.Component

// VNode is an alias for vdom.VNode for convenience.
// VNode represents a virtual DOM node.
type VNode = vdom.VNode

// VKind is an alias for vdom.VKind.
// VKind is the node type discriminator.
type VKind = vdom.VKind

// Props is an alias for vdom.Props.
// Props holds attributes and event handlers.
type Props = vdom.Props

// Re-export VKind constants
const (
	KindElement   = vdom.KindElement
	KindText      = vdom.KindText
	KindFragment  = vdom.KindFragment
	KindComponent = vdom.KindComponent
	KindRaw       = vdom.KindRaw
)
