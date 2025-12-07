package vdom

import "testing"

func TestVKindString(t *testing.T) {
	tests := []struct {
		kind VKind
		want string
	}{
		{KindElement, "Element"},
		{KindText, "Text"},
		{KindFragment, "Fragment"},
		{KindComponent, "Component"},
		{KindRaw, "Raw"},
		{VKind(255), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.kind.String(); got != tt.want {
				t.Errorf("VKind.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVNodeIsInteractive(t *testing.T) {
	tests := []struct {
		name string
		node *VNode
		want bool
	}{
		{
			name: "nil node",
			node: nil,
			want: false,
		},
		{
			name: "text node",
			node: &VNode{Kind: KindText, Text: "hello"},
			want: false,
		},
		{
			name: "element without handlers",
			node: &VNode{Kind: KindElement, Tag: "div", Props: Props{"class": "test"}},
			want: false,
		},
		{
			name: "element with onclick",
			node: &VNode{Kind: KindElement, Tag: "button", Props: Props{"onclick": func() {}}},
			want: true,
		},
		{
			name: "element with oninput",
			node: &VNode{Kind: KindElement, Tag: "input", Props: Props{"oninput": func() {}}},
			want: true,
		},
		{
			name: "element with multiple handlers",
			node: &VNode{Kind: KindElement, Tag: "div", Props: Props{
				"onclick":    func() {},
				"onmouseover": func() {},
			}},
			want: true,
		},
		{
			name: "element with nil props",
			node: &VNode{Kind: KindElement, Tag: "div"},
			want: false,
		},
		{
			name: "fragment node",
			node: &VNode{Kind: KindFragment},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.node.IsInteractive(); got != tt.want {
				t.Errorf("VNode.IsInteractive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAttrIsEmpty(t *testing.T) {
	tests := []struct {
		name string
		attr Attr
		want bool
	}{
		{"empty attr", Attr{}, true},
		{"attr with key", Attr{Key: "class", Value: "test"}, false},
		{"attr with empty value", Attr{Key: "disabled", Value: ""}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.attr.IsEmpty(); got != tt.want {
				t.Errorf("Attr.IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFuncComponent(t *testing.T) {
	called := false
	comp := Func(func() *VNode {
		called = true
		return Div(Class("test"))
	})

	node := comp.Render()

	if !called {
		t.Error("Func component was not called")
	}

	if node == nil {
		t.Fatal("Render returned nil")
	}

	if node.Kind != KindElement {
		t.Errorf("Kind = %v, want KindElement", node.Kind)
	}

	if node.Tag != "div" {
		t.Errorf("Tag = %v, want div", node.Tag)
	}
}

func TestPatchOpString(t *testing.T) {
	tests := []struct {
		op   PatchOp
		want string
	}{
		{PatchSetText, "SetText"},
		{PatchSetAttr, "SetAttr"},
		{PatchRemoveAttr, "RemoveAttr"},
		{PatchInsertNode, "InsertNode"},
		{PatchRemoveNode, "RemoveNode"},
		{PatchMoveNode, "MoveNode"},
		{PatchReplaceNode, "ReplaceNode"},
		{PatchSetValue, "SetValue"},
		{PatchSetChecked, "SetChecked"},
		{PatchFocus, "Focus"},
		{PatchOp(255), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.op.String(); got != tt.want {
				t.Errorf("PatchOp.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
