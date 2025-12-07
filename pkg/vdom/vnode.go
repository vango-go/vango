package vdom

import "strings"

// VKind is the node type discriminator.
type VKind uint8

const (
	KindElement   VKind = iota // <div>, <button>, etc.
	KindText                   // Plain text node
	KindFragment               // Grouping without wrapper
	KindComponent              // Nested component
	KindRaw                    // Raw HTML (dangerous)
)

// String returns the string representation of the VKind.
func (k VKind) String() string {
	switch k {
	case KindElement:
		return "Element"
	case KindText:
		return "Text"
	case KindFragment:
		return "Fragment"
	case KindComponent:
		return "Component"
	case KindRaw:
		return "Raw"
	default:
		return "Unknown"
	}
}

// VNode is the virtual DOM node.
type VNode struct {
	Kind     VKind      // Node type
	Tag      string     // Element tag name (e.g., "div")
	Props    Props      // Attributes and event handlers
	Children []*VNode   // Child nodes
	Key      string     // Reconciliation key
	Text     string     // For KindText and KindRaw
	Comp     Component  // For KindComponent
	HID      string     // Hydration ID (assigned during render)
}

// Props holds attributes and event handlers.
type Props map[string]any

// IsInteractive returns true if this node has event handlers and needs a HID.
func (v *VNode) IsInteractive() bool {
	if v == nil || v.Kind != KindElement {
		return false
	}
	for key := range v.Props {
		if strings.HasPrefix(key, "on") {
			return true
		}
	}
	return false
}

// Attr represents a single attribute.
type Attr struct {
	Key   string
	Value any
}

// IsEmpty returns true if this is an empty/nil attribute.
func (a Attr) IsEmpty() bool {
	return a.Key == ""
}

// EventHandler represents an event handler.
type EventHandler struct {
	Event   string // "onclick", "oninput", etc.
	Handler any    // Function to call
}

// Component is anything that can render to a VNode.
type Component interface {
	Render() *VNode
}

// FuncComponent wraps a render function.
type FuncComponent struct {
	render func() *VNode
}

// Render implements Component.
func (f *FuncComponent) Render() *VNode {
	return f.render()
}

// Func creates a component from a render function.
func Func(render func() *VNode) Component {
	return &FuncComponent{render: render}
}
