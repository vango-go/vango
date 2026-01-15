package protocol

import (
	"github.com/vango-go/vango/pkg/vdom"
)

// VNodeWire is the wire format for VNodes.
// It contains only serializable data (no event handlers or components).
type VNodeWire struct {
	Kind     vdom.VKind        // Node type
	Tag      string            // Element tag name
	HID      string            // Hydration ID
	Attrs    map[string]string // String attributes only (no handlers)
	Children []*VNodeWire      // Child nodes
	Text     string            // For Text and Raw nodes
}

// VNodeToWire converts a vdom.VNode to wire format.
// Event handlers are stripped; only string attributes are included.
// Component nodes should be rendered before calling this function.
func VNodeToWire(node *vdom.VNode) *VNodeWire {
	if node == nil {
		return nil
	}

	w := &VNodeWire{
		Kind: node.Kind,
		Tag:  node.Tag,
		HID:  node.HID,
		Text: node.Text,
	}

	// Convert props to string attrs (skip event handlers, but add markers)
	if node.Props != nil {
		if attrs := vdom.EffectiveAttrs(node); len(attrs) > 0 {
			w.Attrs = attrs
		}
	}

	// Recursively convert children
	if len(node.Children) > 0 {
		w.Children = make([]*VNodeWire, 0, len(node.Children))
		for _, child := range node.Children {
			if child != nil {
				w.Children = append(w.Children, VNodeToWire(child))
			}
		}
	}

	return w
}

// EncodeVNodeWire encodes a VNodeWire to bytes using the provided encoder.
func EncodeVNodeWire(e *Encoder, node *VNodeWire) {
	if node == nil {
		e.WriteByte(0xFF) // Null marker
		return
	}

	e.WriteByte(byte(node.Kind))

	switch node.Kind {
	case vdom.KindElement:
		e.WriteString(node.Tag)
		e.WriteString(node.HID)

		// Encode attributes
		e.WriteUvarint(uint64(len(node.Attrs)))
		for k, v := range node.Attrs {
			e.WriteString(k)
			e.WriteString(v)
		}

		// Encode children
		e.WriteUvarint(uint64(len(node.Children)))
		for _, child := range node.Children {
			EncodeVNodeWire(e, child)
		}

	case vdom.KindText:
		e.WriteString(node.Text)

	case vdom.KindFragment:
		e.WriteUvarint(uint64(len(node.Children)))
		for _, child := range node.Children {
			EncodeVNodeWire(e, child)
		}

	case vdom.KindRaw:
		e.WriteString(node.Text)

	case vdom.KindComponent:
		// Components should be rendered before encoding
		// Encode as empty fragment
		e.WriteUvarint(0)
	}
}

// DecodeVNodeWire decodes a VNodeWire from the decoder.
// SECURITY: Enforces MaxVNodeDepth to prevent stack overflow attacks.
func DecodeVNodeWire(d *Decoder) (*VNodeWire, error) {
	return decodeVNodeWireWithDepth(d, 0)
}

// decodeVNodeWireWithDepth decodes a VNodeWire with depth tracking.
// This is the internal implementation that tracks recursion depth.
func decodeVNodeWireWithDepth(d *Decoder, depth int) (*VNodeWire, error) {
	// SECURITY: Check depth limit before any work
	if err := checkDepth(depth, MaxVNodeDepth); err != nil {
		return nil, err
	}

	kindByte, err := d.ReadByte()
	if err != nil {
		return nil, err
	}

	// Null marker
	if kindByte == 0xFF {
		return nil, nil
	}

	node := &VNodeWire{
		Kind: vdom.VKind(kindByte),
	}

	switch node.Kind {
	case vdom.KindElement:
		node.Tag, err = d.ReadString()
		if err != nil {
			return nil, err
		}

		node.HID, err = d.ReadString()
		if err != nil {
			return nil, err
		}

		// Decode attributes
		// SECURITY: Use ReadCollectionCount to prevent DoS
		attrCount, err := d.ReadCollectionCount()
		if err != nil {
			return nil, err
		}

		if attrCount > 0 {
			node.Attrs = make(map[string]string, attrCount)
			for i := 0; i < attrCount; i++ {
				key, err := d.ReadString()
				if err != nil {
					return nil, err
				}
				value, err := d.ReadString()
				if err != nil {
					return nil, err
				}
				node.Attrs[key] = value
			}
		}

		// Decode children
		// SECURITY: Use ReadCollectionCount to prevent DoS
		childCount, err := d.ReadCollectionCount()
		if err != nil {
			return nil, err
		}

		if childCount > 0 {
			node.Children = make([]*VNodeWire, childCount)
			for i := 0; i < childCount; i++ {
				// SECURITY: Increment depth for child nodes
				child, err := decodeVNodeWireWithDepth(d, depth+1)
				if err != nil {
					return nil, err
				}
				node.Children[i] = child
			}
		}

	case vdom.KindText:
		node.Text, err = d.ReadString()
		if err != nil {
			return nil, err
		}

	case vdom.KindFragment:
		// SECURITY: Use ReadCollectionCount to prevent DoS
		childCount, err := d.ReadCollectionCount()
		if err != nil {
			return nil, err
		}

		if childCount > 0 {
			node.Children = make([]*VNodeWire, childCount)
			for i := 0; i < childCount; i++ {
				// SECURITY: Increment depth for child nodes
				child, err := decodeVNodeWireWithDepth(d, depth+1)
				if err != nil {
					return nil, err
				}
				node.Children[i] = child
			}
		}

	case vdom.KindRaw:
		node.Text, err = d.ReadString()
		if err != nil {
			return nil, err
		}

	default:
		// Unknown kind - try to continue
	}

	return node, nil
}

// ToVNode converts a VNodeWire back to a vdom.VNode.
// Note: Event handlers cannot be restored from wire format.
func (w *VNodeWire) ToVNode() *vdom.VNode {
	if w == nil {
		return nil
	}

	node := &vdom.VNode{
		Kind: w.Kind,
		Tag:  w.Tag,
		HID:  w.HID,
		Text: w.Text,
	}

	// Convert attrs back to props
	if len(w.Attrs) > 0 {
		node.Props = make(vdom.Props, len(w.Attrs))
		for k, v := range w.Attrs {
			node.Props[k] = v
		}
	}

	// Convert children
	if len(w.Children) > 0 {
		node.Children = make([]*vdom.VNode, len(w.Children))
		for i, child := range w.Children {
			node.Children[i] = child.ToVNode()
		}
	}

	return node
}

// NewTextWire creates a text VNodeWire.
func NewTextWire(text string) *VNodeWire {
	return &VNodeWire{
		Kind: vdom.KindText,
		Text: text,
	}
}

// NewElementWire creates an element VNodeWire.
func NewElementWire(tag string, attrs map[string]string, children ...*VNodeWire) *VNodeWire {
	return &VNodeWire{
		Kind:     vdom.KindElement,
		Tag:      tag,
		Attrs:    attrs,
		Children: children,
	}
}

// NewFragmentWire creates a fragment VNodeWire.
func NewFragmentWire(children ...*VNodeWire) *VNodeWire {
	return &VNodeWire{
		Kind:     vdom.KindFragment,
		Children: children,
	}
}

// NewRawWire creates a raw HTML VNodeWire.
func NewRawWire(html string) *VNodeWire {
	return &VNodeWire{
		Kind: vdom.KindRaw,
		Text: html,
	}
}
