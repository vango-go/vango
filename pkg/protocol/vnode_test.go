package protocol

import (
	"testing"

	"github.com/vango-dev/vango/v2/pkg/vdom"
)

func TestVNodeWireEncodeDecode(t *testing.T) {
	tests := []struct {
		name string
		node *VNodeWire
	}{
		{
			name: "text_node",
			node: NewTextWire("Hello, World!"),
		},
		{
			name: "simple_element",
			node: NewElementWire("div", map[string]string{
				"class": "container",
			}),
		},
		{
			name: "element_with_children",
			node: NewElementWire("div", map[string]string{
				"id": "root",
			},
				NewElementWire("h1", nil, NewTextWire("Title")),
				NewElementWire("p", nil, NewTextWire("Content")),
			),
		},
		{
			name: "element_with_hid",
			node: &VNodeWire{
				Kind: vdom.KindElement,
				Tag:  "button",
				HID:  "h42",
				Attrs: map[string]string{
					"class": "btn btn-primary",
					"type":  "submit",
				},
				Children: []*VNodeWire{
					NewTextWire("Click me"),
				},
			},
		},
		{
			name: "fragment",
			node: NewFragmentWire(
				NewElementWire("span", nil, NewTextWire("First")),
				NewElementWire("span", nil, NewTextWire("Second")),
			),
		},
		{
			name: "raw_html",
			node: NewRawWire("<strong>Bold</strong>"),
		},
		{
			name: "deeply_nested",
			node: NewElementWire("div", map[string]string{"class": "l1"},
				NewElementWire("div", map[string]string{"class": "l2"},
					NewElementWire("div", map[string]string{"class": "l3"},
						NewElementWire("div", map[string]string{"class": "l4"},
							NewTextWire("Deep content"),
						),
					),
				),
			),
		},
		{
			name: "empty_element",
			node: NewElementWire("br", nil),
		},
		{
			name: "nil_node",
			node: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Encode
			e := NewEncoder()
			EncodeVNodeWire(e, tc.node)

			// Decode
			d := NewDecoder(e.Bytes())
			decoded, err := DecodeVNodeWire(d)
			if err != nil {
				t.Fatalf("DecodeVNodeWire() error = %v", err)
			}

			// Verify
			verifyVNodeWire(t, decoded, tc.node)
		})
	}
}

func verifyVNodeWire(t *testing.T, got, want *VNodeWire) {
	t.Helper()

	if want == nil {
		if got != nil {
			t.Errorf("got %+v, want nil", got)
		}
		return
	}

	if got == nil {
		t.Fatalf("got nil, want %+v", want)
	}

	if got.Kind != want.Kind {
		t.Errorf("Kind = %v, want %v", got.Kind, want.Kind)
	}
	if got.Tag != want.Tag {
		t.Errorf("Tag = %q, want %q", got.Tag, want.Tag)
	}
	if got.HID != want.HID {
		t.Errorf("HID = %q, want %q", got.HID, want.HID)
	}
	if got.Text != want.Text {
		t.Errorf("Text = %q, want %q", got.Text, want.Text)
	}

	// Compare attrs
	if len(got.Attrs) != len(want.Attrs) {
		t.Errorf("Attrs count = %d, want %d", len(got.Attrs), len(want.Attrs))
	} else {
		for k, wv := range want.Attrs {
			if gv, ok := got.Attrs[k]; !ok || gv != wv {
				t.Errorf("Attrs[%q] = %q, want %q", k, gv, wv)
			}
		}
	}

	// Compare children
	if len(got.Children) != len(want.Children) {
		t.Errorf("Children count = %d, want %d", len(got.Children), len(want.Children))
	} else {
		for i := range want.Children {
			verifyVNodeWire(t, got.Children[i], want.Children[i])
		}
	}
}

func TestVNodeToWire(t *testing.T) {
	// Create a vdom.VNode with event handlers
	vnode := &vdom.VNode{
		Kind: vdom.KindElement,
		Tag:  "button",
		HID:  "h1",
		Props: vdom.Props{
			"class":   "btn",
			"onclick": func() {}, // Should be stripped
			"type":    "submit",
		},
		Children: []*vdom.VNode{
			{Kind: vdom.KindText, Text: "Click me"},
		},
	}

	wire := VNodeToWire(vnode)

	// Verify basic fields
	if wire.Kind != vdom.KindElement {
		t.Errorf("Kind = %v, want Element", wire.Kind)
	}
	if wire.Tag != "button" {
		t.Errorf("Tag = %q, want \"button\"", wire.Tag)
	}
	if wire.HID != "h1" {
		t.Errorf("HID = %q, want \"h1\"", wire.HID)
	}

	// Verify event handler was stripped
	if _, ok := wire.Attrs["onclick"]; ok {
		t.Error("onclick should have been stripped")
	}

	// Verify other attrs remain
	if wire.Attrs["class"] != "btn" {
		t.Errorf("class = %q, want \"btn\"", wire.Attrs["class"])
	}
	if wire.Attrs["type"] != "submit" {
		t.Errorf("type = %q, want \"submit\"", wire.Attrs["type"])
	}

	// Verify children
	if len(wire.Children) != 1 {
		t.Fatalf("Children count = %d, want 1", len(wire.Children))
	}
	if wire.Children[0].Kind != vdom.KindText {
		t.Errorf("Child kind = %v, want Text", wire.Children[0].Kind)
	}
	if wire.Children[0].Text != "Click me" {
		t.Errorf("Child text = %q, want \"Click me\"", wire.Children[0].Text)
	}
}

func TestVNodeWireToVNode(t *testing.T) {
	wire := NewElementWire("div", map[string]string{
		"class": "container",
		"id":    "main",
	},
		NewTextWire("Hello"),
	)
	wire.HID = "h42"

	vnode := wire.ToVNode()

	if vnode.Kind != vdom.KindElement {
		t.Errorf("Kind = %v, want Element", vnode.Kind)
	}
	if vnode.Tag != "div" {
		t.Errorf("Tag = %q, want \"div\"", vnode.Tag)
	}
	if vnode.HID != "h42" {
		t.Errorf("HID = %q, want \"h42\"", vnode.HID)
	}
	if vnode.Props["class"] != "container" {
		t.Errorf("class = %v, want \"container\"", vnode.Props["class"])
	}
	if vnode.Props["id"] != "main" {
		t.Errorf("id = %v, want \"main\"", vnode.Props["id"])
	}
	if len(vnode.Children) != 1 {
		t.Fatalf("Children count = %d, want 1", len(vnode.Children))
	}
	if vnode.Children[0].Text != "Hello" {
		t.Errorf("Child text = %q, want \"Hello\"", vnode.Children[0].Text)
	}
}

func TestVNodeWireRoundTrip(t *testing.T) {
	// Create a complex vdom.VNode
	original := &vdom.VNode{
		Kind: vdom.KindElement,
		Tag:  "form",
		HID:  "h1",
		Props: vdom.Props{
			"class":    "login-form",
			"method":   "POST",
			"onsubmit": func() {}, // Will be stripped
		},
		Children: []*vdom.VNode{
			{
				Kind: vdom.KindElement,
				Tag:  "input",
				HID:  "h2",
				Props: vdom.Props{
					"type":        "text",
					"name":        "username",
					"placeholder": "Username",
					"oninput":     func() {}, // Will be stripped
				},
			},
			{
				Kind: vdom.KindElement,
				Tag:  "button",
				HID:  "h3",
				Props: vdom.Props{
					"type":    "submit",
					"onclick": func() {}, // Will be stripped
				},
				Children: []*vdom.VNode{
					{Kind: vdom.KindText, Text: "Login"},
				},
			},
		},
	}

	// Convert to wire format
	wire := VNodeToWire(original)

	// Encode
	e := NewEncoder()
	EncodeVNodeWire(e, wire)

	// Decode
	d := NewDecoder(e.Bytes())
	decoded, err := DecodeVNodeWire(d)
	if err != nil {
		t.Fatalf("DecodeVNodeWire() error = %v", err)
	}

	// Convert back to vdom.VNode
	result := decoded.ToVNode()

	// Verify structure is preserved (without event handlers)
	if result.Tag != "form" {
		t.Errorf("Tag = %q, want \"form\"", result.Tag)
	}
	if result.HID != "h1" {
		t.Errorf("HID = %q, want \"h1\"", result.HID)
	}
	if len(result.Children) != 2 {
		t.Fatalf("Children count = %d, want 2", len(result.Children))
	}

	// Verify event handlers are not present
	if _, ok := result.Props["onsubmit"]; ok {
		t.Error("onsubmit should not be present after round-trip")
	}
}

func TestVNodeWireNilHandling(t *testing.T) {
	// Test nil children in VNode
	vnode := &vdom.VNode{
		Kind: vdom.KindElement,
		Tag:  "div",
		Children: []*vdom.VNode{
			{Kind: vdom.KindText, Text: "First"},
			nil, // Nil child
			{Kind: vdom.KindText, Text: "Third"},
		},
	}

	wire := VNodeToWire(vnode)

	// Nil children should be filtered out
	if len(wire.Children) != 2 {
		t.Errorf("Children count = %d, want 2 (nil filtered)", len(wire.Children))
	}
}

func BenchmarkVNodeToWire(b *testing.B) {
	vnode := &vdom.VNode{
		Kind: vdom.KindElement,
		Tag:  "div",
		HID:  "h1",
		Props: vdom.Props{
			"class": "container",
			"id":    "main",
		},
		Children: []*vdom.VNode{
			{Kind: vdom.KindText, Text: "Hello"},
			{Kind: vdom.KindElement, Tag: "span", Children: []*vdom.VNode{
				{Kind: vdom.KindText, Text: "World"},
			}},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = VNodeToWire(vnode)
	}
}

func BenchmarkEncodeVNodeWire(b *testing.B) {
	wire := NewElementWire("div", map[string]string{
		"class": "container",
		"id":    "main",
	},
		NewTextWire("Hello"),
		NewElementWire("span", nil, NewTextWire("World")),
	)

	e := NewEncoder()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Reset()
		EncodeVNodeWire(e, wire)
	}
}

func BenchmarkDecodeVNodeWire(b *testing.B) {
	wire := NewElementWire("div", map[string]string{
		"class": "container",
		"id":    "main",
	},
		NewTextWire("Hello"),
		NewElementWire("span", nil, NewTextWire("World")),
	)

	e := NewEncoder()
	EncodeVNodeWire(e, wire)
	data := e.Bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d := NewDecoder(data)
		_, _ = DecodeVNodeWire(d)
	}
}
