package vdom

import "testing"

// Helper to assign HIDs for testing
func assignTestHIDs(node *VNode) {
	gen := NewHIDGenerator()
	assignAllHIDsRecursive(node, gen)
}

func assignAllHIDsRecursive(node *VNode, gen *HIDGenerator) {
	if node == nil {
		return
	}
	if node.Kind == KindElement || node.Kind == KindText {
		node.HID = gen.Next()
	}
	for _, child := range node.Children {
		assignAllHIDsRecursive(child, gen)
	}
}

func TestDiffBothNil(t *testing.T) {
	patches := Diff(nil, nil)
	if len(patches) != 0 {
		t.Errorf("Expected 0 patches, got %d", len(patches))
	}
}

func TestDiffNodeRemoved(t *testing.T) {
	prev := Div()
	prev.HID = "h1"

	patches := Diff(prev, nil)

	if len(patches) != 1 {
		t.Fatalf("Expected 1 patch, got %d", len(patches))
	}
	if patches[0].Op != PatchRemoveNode {
		t.Errorf("Op = %v, want PatchRemoveNode", patches[0].Op)
	}
	if patches[0].HID != "h1" {
		t.Errorf("HID = %v, want h1", patches[0].HID)
	}
}

func TestDiffTextChange(t *testing.T) {
	prev := Text("Hello")
	prev.HID = "h1"
	next := Text("World")

	patches := Diff(prev, next)

	if len(patches) != 1 {
		t.Fatalf("Expected 1 patch, got %d", len(patches))
	}
	if patches[0].Op != PatchSetText {
		t.Errorf("Op = %v, want PatchSetText", patches[0].Op)
	}
	if patches[0].HID != "h1" {
		t.Errorf("HID = %v, want h1", patches[0].HID)
	}
	if patches[0].Value != "World" {
		t.Errorf("Value = %v, want World", patches[0].Value)
	}
}

func TestDiffTextUnchanged(t *testing.T) {
	prev := Text("Hello")
	prev.HID = "h1"
	next := Text("Hello")

	patches := Diff(prev, next)

	if len(patches) != 0 {
		t.Errorf("Expected 0 patches for unchanged text, got %d", len(patches))
	}
}

func TestDiffKindChange(t *testing.T) {
	prev := Text("Hello")
	prev.HID = "h1"
	next := Div(Text("Hello"))

	patches := Diff(prev, next)

	if len(patches) != 1 {
		t.Fatalf("Expected 1 patch, got %d", len(patches))
	}
	if patches[0].Op != PatchReplaceNode {
		t.Errorf("Op = %v, want PatchReplaceNode", patches[0].Op)
	}
}

func TestDiffTagChange(t *testing.T) {
	prev := Div()
	prev.HID = "h1"
	next := Span()

	patches := Diff(prev, next)

	if len(patches) != 1 {
		t.Fatalf("Expected 1 patch, got %d", len(patches))
	}
	if patches[0].Op != PatchReplaceNode {
		t.Errorf("Op = %v, want PatchReplaceNode", patches[0].Op)
	}
}

func TestDiffAttributeAdded(t *testing.T) {
	prev := Div()
	prev.HID = "h1"
	next := Div(Class("new"))

	patches := Diff(prev, next)

	if len(patches) != 1 {
		t.Fatalf("Expected 1 patch, got %d", len(patches))
	}
	if patches[0].Op != PatchSetAttr {
		t.Errorf("Op = %v, want PatchSetAttr", patches[0].Op)
	}
	if patches[0].Key != "class" {
		t.Errorf("Key = %v, want class", patches[0].Key)
	}
	if patches[0].Value != "new" {
		t.Errorf("Value = %v, want new", patches[0].Value)
	}
}

func TestDiffAttributeRemoved(t *testing.T) {
	prev := Div(Class("old"))
	prev.HID = "h1"
	next := Div()

	patches := Diff(prev, next)

	if len(patches) != 1 {
		t.Fatalf("Expected 1 patch, got %d", len(patches))
	}
	if patches[0].Op != PatchRemoveAttr {
		t.Errorf("Op = %v, want PatchRemoveAttr", patches[0].Op)
	}
	if patches[0].Key != "class" {
		t.Errorf("Key = %v, want class", patches[0].Key)
	}
}

func TestDiffAttributeChanged(t *testing.T) {
	prev := Div(Class("old"))
	prev.HID = "h1"
	next := Div(Class("new"))

	patches := Diff(prev, next)

	if len(patches) != 1 {
		t.Fatalf("Expected 1 patch, got %d", len(patches))
	}
	if patches[0].Op != PatchSetAttr {
		t.Errorf("Op = %v, want PatchSetAttr", patches[0].Op)
	}
	if patches[0].Value != "new" {
		t.Errorf("Value = %v, want new", patches[0].Value)
	}
}

func TestDiffMultipleAttributeChanges(t *testing.T) {
	prev := Div(Class("old"), ID("test"))
	prev.HID = "h1"
	next := Div(Class("new"), TitleAttr("hello"))

	patches := Diff(prev, next)

	// Expect: SetAttr(class), RemoveAttr(id), SetAttr(title)
	if len(patches) != 3 {
		t.Fatalf("Expected 3 patches, got %d", len(patches))
	}

	ops := make(map[PatchOp]int)
	for _, p := range patches {
		ops[p.Op]++
	}

	if ops[PatchSetAttr] != 2 {
		t.Errorf("Expected 2 SetAttr patches, got %d", ops[PatchSetAttr])
	}
	if ops[PatchRemoveAttr] != 1 {
		t.Errorf("Expected 1 RemoveAttr patch, got %d", ops[PatchRemoveAttr])
	}
}

func TestDiffEventHandlersIgnored(t *testing.T) {
	handler := func() {}
	prev := Button(OnClick(handler))
	prev.HID = "h1"
	next := Button(OnClick(func() {}))

	patches := Diff(prev, next)

	// Event handlers should be ignored in diffing
	if len(patches) != 0 {
		t.Errorf("Expected 0 patches (events ignored), got %d", len(patches))
	}
}

func TestDiffKeyIgnored(t *testing.T) {
	prev := Li(Key("a"))
	prev.HID = "h1"
	next := Li(Key("b"))

	patches := Diff(prev, next)

	// Key is not a real attribute, should be ignored
	if len(patches) != 0 {
		t.Errorf("Expected 0 patches (key ignored as attribute), got %d", len(patches))
	}
}

func TestDiffChildAdded(t *testing.T) {
	prev := Ul()
	prev.HID = "h1"
	next := Ul(Li(Text("Item")))

	patches := Diff(prev, next)

	if len(patches) != 1 {
		t.Fatalf("Expected 1 patch, got %d", len(patches))
	}
	if patches[0].Op != PatchInsertNode {
		t.Errorf("Op = %v, want PatchInsertNode", patches[0].Op)
	}
	if patches[0].ParentID != "h1" {
		t.Errorf("ParentID = %v, want h1", patches[0].ParentID)
	}
	if patches[0].Index != 0 {
		t.Errorf("Index = %v, want 0", patches[0].Index)
	}
}

func TestDiffChildRemoved(t *testing.T) {
	child := Li(Text("Item"))
	child.HID = "h2"
	prev := Ul(child)
	prev.HID = "h1"
	next := Ul()

	patches := Diff(prev, next)

	if len(patches) != 1 {
		t.Fatalf("Expected 1 patch, got %d", len(patches))
	}
	if patches[0].Op != PatchRemoveNode {
		t.Errorf("Op = %v, want PatchRemoveNode", patches[0].Op)
	}
	if patches[0].HID != "h2" {
		t.Errorf("HID = %v, want h2", patches[0].HID)
	}
}

func TestDiffUnkeyedReorder(t *testing.T) {
	// Unkeyed children: reordering results in text patches
	prev := Ul(
		Li(Text("A")),
		Li(Text("B")),
	)
	assignTestHIDs(prev)

	next := Ul(
		Li(Text("B")),
		Li(Text("A")),
	)

	patches := Diff(prev, next)

	// Should have 2 SetText patches (A->B, B->A)
	setTextCount := 0
	for _, p := range patches {
		if p.Op == PatchSetText {
			setTextCount++
		}
	}
	if setTextCount != 2 {
		t.Errorf("Expected 2 SetText patches, got %d", setTextCount)
	}
}

func TestDiffKeyedReorder(t *testing.T) {
	prev := Ul(
		Li(Key("a"), Text("A")),
		Li(Key("b"), Text("B")),
		Li(Key("c"), Text("C")),
	)
	assignTestHIDs(prev)

	next := Ul(
		Li(Key("c"), Text("C")),
		Li(Key("a"), Text("A")),
		Li(Key("b"), Text("B")),
	)

	patches := Diff(prev, next)

	// Should have MoveNode patches, not Insert/Remove
	moveCount := 0
	for _, p := range patches {
		if p.Op == PatchMoveNode {
			moveCount++
		}
	}
	if moveCount == 0 {
		t.Error("Expected at least one MoveNode patch for keyed reorder")
	}
}

func TestDiffKeyedAddition(t *testing.T) {
	prev := Ul(
		Li(Key("a"), Text("A")),
		Li(Key("c"), Text("C")),
	)
	assignTestHIDs(prev)

	next := Ul(
		Li(Key("a"), Text("A")),
		Li(Key("b"), Text("B")),
		Li(Key("c"), Text("C")),
	)

	patches := Diff(prev, next)

	// Should have InsertNode for "b"
	insertCount := 0
	for _, p := range patches {
		if p.Op == PatchInsertNode {
			insertCount++
		}
	}
	if insertCount != 1 {
		t.Errorf("Expected 1 InsertNode patch, got %d", insertCount)
	}
}

func TestDiffKeyedRemoval(t *testing.T) {
	prev := Ul(
		Li(Key("a"), Text("A")),
		Li(Key("b"), Text("B")),
		Li(Key("c"), Text("C")),
	)
	assignTestHIDs(prev)

	next := Ul(
		Li(Key("a"), Text("A")),
		Li(Key("c"), Text("C")),
	)

	patches := Diff(prev, next)

	// Should have RemoveNode for "b"
	removeCount := 0
	for _, p := range patches {
		if p.Op == PatchRemoveNode {
			removeCount++
		}
	}
	if removeCount != 1 {
		t.Errorf("Expected 1 RemoveNode patch, got %d", removeCount)
	}
}

func TestDiffFragmentChildren(t *testing.T) {
	prev := Fragment(Div(), Span())
	assignTestHIDs(prev)

	next := Fragment(Div(), P())

	patches := Diff(prev, next)

	// Second child changed from Span to P - should be ReplaceNode
	replaceCount := 0
	for _, p := range patches {
		if p.Op == PatchReplaceNode {
			replaceCount++
		}
	}
	if replaceCount != 1 {
		t.Errorf("Expected 1 ReplaceNode patch, got %d", replaceCount)
	}
}

func TestDiffRawHtmlChange(t *testing.T) {
	prev := Raw("<b>Bold</b>")
	prev.HID = "h1"
	next := Raw("<i>Italic</i>")

	patches := Diff(prev, next)

	if len(patches) != 1 {
		t.Fatalf("Expected 1 patch, got %d", len(patches))
	}
	if patches[0].Op != PatchReplaceNode {
		t.Errorf("Op = %v, want PatchReplaceNode", patches[0].Op)
	}
}

func TestDiffRawHtmlUnchanged(t *testing.T) {
	prev := Raw("<b>Bold</b>")
	prev.HID = "h1"
	next := Raw("<b>Bold</b>")

	patches := Diff(prev, next)

	if len(patches) != 0 {
		t.Errorf("Expected 0 patches for unchanged raw HTML, got %d", len(patches))
	}
}

func TestDiffDeepTree(t *testing.T) {
	prev := Div(
		Header(H1(Text("Title"))),
		Main(
			Article(
				P(Text("Paragraph 1")),
				P(Text("Paragraph 2")),
			),
		),
		Footer(Text("Footer")),
	)
	assignTestHIDs(prev)

	// Change one deep node
	next := Div(
		Header(H1(Text("New Title"))),
		Main(
			Article(
				P(Text("Paragraph 1")),
				P(Text("Paragraph 2")),
			),
		),
		Footer(Text("Footer")),
	)

	patches := Diff(prev, next)

	// Should only have 1 SetText patch for the title
	if len(patches) != 1 {
		t.Fatalf("Expected 1 patch, got %d", len(patches))
	}
	if patches[0].Op != PatchSetText {
		t.Errorf("Op = %v, want PatchSetText", patches[0].Op)
	}
	if patches[0].Value != "New Title" {
		t.Errorf("Value = %v, want 'New Title'", patches[0].Value)
	}
}

func TestDiffSameTree(t *testing.T) {
	tree := Div(
		Class("container"),
		H1(Text("Title")),
		P(Text("Content")),
		Button(OnClick(func() {}), Text("Click")),
	)
	assignTestHIDs(tree)

	// Create identical tree
	tree2 := Div(
		Class("container"),
		H1(Text("Title")),
		P(Text("Content")),
		Button(OnClick(func() {}), Text("Click")),
	)

	patches := Diff(tree, tree2)

	if len(patches) != 0 {
		t.Errorf("Expected 0 patches for identical trees, got %d", len(patches))
	}
}

func TestPropsEqual(t *testing.T) {
	tests := []struct {
		name string
		a, b any
		want bool
	}{
		{"equal strings", "a", "a", true},
		{"different strings", "a", "b", false},
		{"equal ints", 1, 1, true},
		{"different ints", 1, 2, false},
		{"equal bools", true, true, true},
		{"different bools", true, false, false},
		{"equal floats", 1.5, 1.5, true},
		{"different floats", 1.5, 2.5, false},
		{"nil values", nil, nil, true},
		{"one nil", nil, "a", false},
		{"different types", 1, "1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := propsEqual(tt.a, tt.b); got != tt.want {
				t.Errorf("propsEqual(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestPropToString(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{"string", "hello", "hello"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"int", 42, "42"},
		{"int64", int64(123), "123"},
		{"float64", 3.14, "3.14"},
		{"struct", struct{ X int }{1}, "{1}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := propToString(tt.value); got != tt.want {
				t.Errorf("propToString(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestIsEventHandler(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{"onclick", true},
		{"oninput", true},
		{"onmouseover", true},
		{"class", false},
		{"id", false},
		{"on", false}, // "on" alone is NOT a valid event handler
		{"", false},
		{"ONCLICK", true}, // case-insensitive
		{"OnClick", true}, // mixed case
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if got := isEventHandler(tt.key); got != tt.want {
				t.Errorf("isEventHandler(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestGetKey(t *testing.T) {
	t.Run("from Key field", func(t *testing.T) {
		node := &VNode{Kind: KindElement, Key: "test"}
		if got := getKey(node); got != "test" {
			t.Errorf("getKey() = %v, want test", got)
		}
	})

	t.Run("from Props", func(t *testing.T) {
		node := &VNode{Kind: KindElement, Props: Props{"key": "test"}}
		if got := getKey(node); got != "test" {
			t.Errorf("getKey() = %v, want test", got)
		}
	})

	t.Run("nil node", func(t *testing.T) {
		if got := getKey(nil); got != "" {
			t.Errorf("getKey(nil) = %v, want empty", got)
		}
	})

	t.Run("no key", func(t *testing.T) {
		node := &VNode{Kind: KindElement}
		if got := getKey(node); got != "" {
			t.Errorf("getKey() = %v, want empty", got)
		}
	})
}

func TestHasKeys(t *testing.T) {
	t.Run("with keys", func(t *testing.T) {
		children := []*VNode{
			{Key: "a"},
			{Key: "b"},
		}
		if !hasKeys(children) {
			t.Error("hasKeys should return true")
		}
	})

	t.Run("without keys", func(t *testing.T) {
		children := []*VNode{
			{Kind: KindElement},
			{Kind: KindElement},
		}
		if hasKeys(children) {
			t.Error("hasKeys should return false")
		}
	})

	t.Run("mixed", func(t *testing.T) {
		children := []*VNode{
			{Kind: KindElement},
			{Key: "a"},
		}
		if !hasKeys(children) {
			t.Error("hasKeys should return true for mixed")
		}
	})
}

func TestHIDCopied(t *testing.T) {
	prev := Div(Class("test"))
	prev.HID = "h1"
	next := Div(Class("test"))

	Diff(prev, next)

	if next.HID != "h1" {
		t.Errorf("HID not copied to next node: got %v, want h1", next.HID)
	}
}
