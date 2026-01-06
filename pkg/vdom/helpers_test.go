package vdom

import "testing"

func TestText(t *testing.T) {
	node := Text("Hello, World!")

	if node.Kind != KindText {
		t.Errorf("Kind = %v, want KindText", node.Kind)
	}
	if node.Text != "Hello, World!" {
		t.Errorf("Text = %v, want 'Hello, World!'", node.Text)
	}
}

func TestTextf(t *testing.T) {
	node := Textf("Count: %d", 42)

	if node.Kind != KindText {
		t.Errorf("Kind = %v, want KindText", node.Kind)
	}
	if node.Text != "Count: 42" {
		t.Errorf("Text = %v, want 'Count: 42'", node.Text)
	}
}

func TestDangerouslySetInnerHTML(t *testing.T) {
	node := DangerouslySetInnerHTML("<strong>Bold</strong>")

	if node.Kind != KindRaw {
		t.Errorf("Kind = %v, want KindRaw", node.Kind)
	}
	if node.Text != "<strong>Bold</strong>" {
		t.Errorf("Text = %v, want '<strong>Bold</strong>'", node.Text)
	}
}

func TestRaw(t *testing.T) {
	// Raw is a legacy alias for DangerouslySetInnerHTML
	node := Raw("<strong>Bold</strong>")

	if node.Kind != KindRaw {
		t.Errorf("Kind = %v, want KindRaw", node.Kind)
	}
	if node.Text != "<strong>Bold</strong>" {
		t.Errorf("Text = %v, want '<strong>Bold</strong>'", node.Text)
	}
}

func TestFragment(t *testing.T) {
	t.Run("with VNodes", func(t *testing.T) {
		node := Fragment(Div(), Span(), P())
		if node.Kind != KindFragment {
			t.Errorf("Kind = %v, want KindFragment", node.Kind)
		}
		if len(node.Children) != 3 {
			t.Errorf("Children len = %v, want 3", len(node.Children))
		}
	})

	t.Run("with nil filtered", func(t *testing.T) {
		node := Fragment(Div(), nil, Span())
		if len(node.Children) != 2 {
			t.Errorf("Children len = %v, want 2", len(node.Children))
		}
	})

	t.Run("with slice", func(t *testing.T) {
		children := []*VNode{Div(), Span()}
		node := Fragment(children)
		if len(node.Children) != 2 {
			t.Errorf("Children len = %v, want 2", len(node.Children))
		}
	})

	t.Run("with string", func(t *testing.T) {
		node := Fragment("Hello")
		if len(node.Children) != 1 {
			t.Fatalf("Children len = %v, want 1", len(node.Children))
		}
		if node.Children[0].Kind != KindText {
			t.Errorf("Child kind = %v, want KindText", node.Children[0].Kind)
		}
	})
}

func TestIf(t *testing.T) {
	node := Div()

	t.Run("condition true", func(t *testing.T) {
		result := If(true, node)
		if result != node {
			t.Error("Expected node when condition is true")
		}
	})

	t.Run("condition false", func(t *testing.T) {
		result := If(false, node)
		if result != nil {
			t.Error("Expected nil when condition is false")
		}
	})
}

func TestIfElse(t *testing.T) {
	nodeA := Div(ID("a"))
	nodeB := Div(ID("b"))

	t.Run("condition true", func(t *testing.T) {
		result := IfElse(true, nodeA, nodeB)
		if result != nodeA {
			t.Error("Expected nodeA when condition is true")
		}
	})

	t.Run("condition false", func(t *testing.T) {
		result := IfElse(false, nodeA, nodeB)
		if result != nodeB {
			t.Error("Expected nodeB when condition is false")
		}
	})
}

func TestWhen(t *testing.T) {
	called := false
	fn := func() *VNode {
		called = true
		return Div()
	}

	t.Run("condition true", func(t *testing.T) {
		called = false
		result := When(true, fn)
		if !called {
			t.Error("Function should be called when condition is true")
		}
		if result == nil {
			t.Error("Expected non-nil result")
		}
	})

	t.Run("condition false", func(t *testing.T) {
		called = false
		result := When(false, fn)
		if called {
			t.Error("Function should not be called when condition is false")
		}
		if result != nil {
			t.Error("Expected nil result")
		}
	})
}

func TestUnless(t *testing.T) {
	node := Div()

	t.Run("condition true", func(t *testing.T) {
		result := Unless(true, node)
		if result != nil {
			t.Error("Expected nil when condition is true")
		}
	})

	t.Run("condition false", func(t *testing.T) {
		result := Unless(false, node)
		if result != node {
			t.Error("Expected node when condition is false")
		}
	})
}

func TestSwitch(t *testing.T) {
	nodeA := Div(ID("a"))
	nodeB := Div(ID("b"))
	nodeDefault := Div(ID("default"))

	t.Run("matching case", func(t *testing.T) {
		result := Switch("a",
			Case_("a", nodeA),
			Case_("b", nodeB),
		)
		if result != nodeA {
			t.Error("Expected nodeA for value 'a'")
		}
	})

	t.Run("default case", func(t *testing.T) {
		result := Switch("c",
			Case_("a", nodeA),
			Case_("b", nodeB),
			Default[string](nodeDefault),
		)
		if result != nodeDefault {
			t.Error("Expected default node for unmatched value")
		}
	})

	t.Run("no match no default", func(t *testing.T) {
		result := Switch("c",
			Case_("a", nodeA),
			Case_("b", nodeB),
		)
		if result != nil {
			t.Error("Expected nil for unmatched value without default")
		}
	})

	t.Run("with int values", func(t *testing.T) {
		node1 := Div(ID("1"))
		node2 := Div(ID("2"))
		result := Switch(2,
			Case_(1, node1),
			Case_(2, node2),
		)
		if result != node2 {
			t.Error("Expected node2 for value 2")
		}
	})
}

func TestRange(t *testing.T) {
	items := []string{"a", "b", "c"}

	nodes := Range(items, func(item string, index int) *VNode {
		return Li(Key(index), Text(item))
	})

	if len(nodes) != 3 {
		t.Fatalf("nodes len = %v, want 3", len(nodes))
	}

	for i, node := range nodes {
		if node.Tag != "li" {
			t.Errorf("nodes[%d].Tag = %v, want li", i, node.Tag)
		}
		if len(node.Children) != 1 {
			t.Errorf("nodes[%d].Children len = %v, want 1", i, len(node.Children))
		}
		if node.Children[0].Text != items[i] {
			t.Errorf("nodes[%d] text = %v, want %v", i, node.Children[0].Text, items[i])
		}
	}
}

func TestRangeWithNilFiltered(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}

	nodes := Range(items, func(item int, index int) *VNode {
		if item%2 == 0 {
			return nil // Filter out even numbers
		}
		return Li(Textf("%d", item))
	})

	if len(nodes) != 3 {
		t.Errorf("nodes len = %v, want 3 (odd numbers only)", len(nodes))
	}
}

func TestRangeMap(t *testing.T) {
	items := map[string]int{"a": 1, "b": 2}

	nodes := RangeMap(items, func(key string, value int) *VNode {
		return Li(Textf("%s: %d", key, value))
	})

	if len(nodes) != 2 {
		t.Errorf("nodes len = %v, want 2", len(nodes))
	}
}

func TestRepeat(t *testing.T) {
	nodes := Repeat(5, func(i int) *VNode {
		return Li(Textf("Item %d", i))
	})

	if len(nodes) != 5 {
		t.Fatalf("nodes len = %v, want 5", len(nodes))
	}

	for i, node := range nodes {
		if node.Children[0].Text != Textf("Item %d", i).Text {
			t.Errorf("nodes[%d] text mismatch", i)
		}
	}
}

func TestRepeatZero(t *testing.T) {
	nodes := Repeat(0, func(i int) *VNode {
		return Li()
	})

	if nodes != nil {
		t.Errorf("Repeat(0) should return nil, got len %d", len(nodes))
	}
}

func TestRepeatNegative(t *testing.T) {
	nodes := Repeat(-5, func(i int) *VNode {
		return Li()
	})

	if nodes != nil {
		t.Errorf("Repeat(-5) should return nil, got len %d", len(nodes))
	}
}

func TestKey(t *testing.T) {
	t.Run("string key", func(t *testing.T) {
		attr := Key("item-1")
		if attr.Key != "key" {
			t.Errorf("Key = %v, want key", attr.Key)
		}
		if attr.Value != "item-1" {
			t.Errorf("Value = %v, want item-1", attr.Value)
		}
	})

	t.Run("int key", func(t *testing.T) {
		attr := Key(42)
		if attr.Value != "42" {
			t.Errorf("Value = %v, want '42'", attr.Value)
		}
	})

	t.Run("struct key", func(t *testing.T) {
		type ID struct{ Val int }
		attr := Key(ID{Val: 1})
		// Should be formatted via Sprintf
		if attr.Value == "" {
			t.Error("Value should not be empty")
		}
	})
}

func TestNothing(t *testing.T) {
	if Nothing() != nil {
		t.Error("Nothing() should return nil")
	}
}

func TestShow(t *testing.T) {
	node := Div()

	if Show(true, node) != node {
		t.Error("Show(true) should return node")
	}
	if Show(false, node) != nil {
		t.Error("Show(false) should return nil")
	}
}

func TestHide(t *testing.T) {
	node := Div()

	if Hide(true, node) != nil {
		t.Error("Hide(true) should return nil")
	}
	if Hide(false, node) != node {
		t.Error("Hide(false) should return node")
	}
}

func TestEither(t *testing.T) {
	nodeA := Div(ID("a"))
	nodeB := Div(ID("b"))

	if Either(nodeA, nodeB) != nodeA {
		t.Error("Either should return first if not nil")
	}
	if Either(nil, nodeB) != nodeB {
		t.Error("Either should return second if first is nil")
	}
}

func TestGroup(t *testing.T) {
	node := Group(Div(), Span())
	if node.Kind != KindFragment {
		t.Errorf("Kind = %v, want KindFragment", node.Kind)
	}
	if len(node.Children) != 2 {
		t.Errorf("Children len = %v, want 2", len(node.Children))
	}
}

func TestMaybe(t *testing.T) {
	node := Div()
	if Maybe(node) != node {
		t.Error("Maybe should return the node as-is")
	}
	if Maybe(nil) != nil {
		t.Error("Maybe(nil) should return nil")
	}
}

func TestFragmentWithComponent(t *testing.T) {
	comp := Func(func() *VNode { return Span() })
	node := Fragment(comp)
	if len(node.Children) != 1 {
		t.Fatalf("Children len = %v, want 1", len(node.Children))
	}
	if node.Children[0].Kind != KindComponent {
		t.Errorf("Child kind = %v, want KindComponent", node.Children[0].Kind)
	}
}

// =============================================================================
// SPA Link Helper Tests
// =============================================================================

func TestLink(t *testing.T) {
	node := Link("/about", Text("About"))

	if node.Tag != "a" {
		t.Errorf("Tag = %v, want 'a'", node.Tag)
	}
	if node.Props["href"] != "/about" {
		t.Errorf("href = %v, want '/about'", node.Props["href"])
	}
	if node.Props["data-vango-link"] != "" {
		t.Errorf("data-vango-link = %v, want ''", node.Props["data-vango-link"])
	}
	if len(node.Children) != 1 {
		t.Fatalf("Children len = %v, want 1", len(node.Children))
	}
	if node.Children[0].Text != "About" {
		t.Errorf("Child text = %v, want 'About'", node.Children[0].Text)
	}
}

func TestLinkWithClass(t *testing.T) {
	node := Link("/home", Class("nav-link"), Text("Home"))

	if node.Props["href"] != "/home" {
		t.Errorf("href = %v, want '/home'", node.Props["href"])
	}
	if node.Props["class"] != "nav-link" {
		t.Errorf("class = %v, want 'nav-link'", node.Props["class"])
	}
}

func TestLinkPrefetch(t *testing.T) {
	node := LinkPrefetch("/about", Text("About"))

	if node.Tag != "a" {
		t.Errorf("Tag = %v, want 'a'", node.Tag)
	}
	if node.Props["href"] != "/about" {
		t.Errorf("href = %v, want '/about'", node.Props["href"])
	}
	if node.Props["data-vango-link"] != "" {
		t.Errorf("data-vango-link = %v, want ''", node.Props["data-vango-link"])
	}
	if node.Props["data-prefetch"] != "" {
		t.Errorf("data-prefetch = %v, want ''", node.Props["data-prefetch"])
	}
}

// mockPathProvider implements PathProvider for testing.
type mockPathProvider struct {
	path string
}

func (m *mockPathProvider) Path() string {
	return m.path
}

func TestNavLink(t *testing.T) {
	t.Run("active when path matches", func(t *testing.T) {
		ctx := &mockPathProvider{path: "/about"}
		node := NavLink(ctx, "/about", Text("About"))

		if node.Props["href"] != "/about" {
			t.Errorf("href = %v, want '/about'", node.Props["href"])
		}
		if node.Props["class"] != "active" {
			t.Errorf("class = %v, want 'active'", node.Props["class"])
		}
	})

	t.Run("not active when path differs", func(t *testing.T) {
		ctx := &mockPathProvider{path: "/home"}
		node := NavLink(ctx, "/about", Text("About"))

		if node.Props["class"] != nil {
			t.Errorf("class = %v, want nil", node.Props["class"])
		}
	})

	t.Run("not active when prefix only", func(t *testing.T) {
		ctx := &mockPathProvider{path: "/about/team"}
		node := NavLink(ctx, "/about", Text("About"))

		// NavLink requires exact match, so /about/team should NOT match /about
		if node.Props["class"] != nil {
			t.Errorf("class = %v, want nil (NavLink requires exact match)", node.Props["class"])
		}
	})

	t.Run("nil context", func(t *testing.T) {
		node := NavLink(nil, "/about", Text("About"))

		// Should not add active class when ctx is nil
		if node.Props["class"] != nil {
			t.Errorf("class = %v, want nil", node.Props["class"])
		}
		// Should still create a valid link
		if node.Props["href"] != "/about" {
			t.Errorf("href = %v, want '/about'", node.Props["href"])
		}
	})
}

func TestNavLinkPrefix(t *testing.T) {
	t.Run("active when path matches exactly", func(t *testing.T) {
		ctx := &mockPathProvider{path: "/admin"}
		node := NavLinkPrefix(ctx, "/admin", Text("Admin"))

		if node.Props["class"] != "active" {
			t.Errorf("class = %v, want 'active'", node.Props["class"])
		}
	})

	t.Run("active when path is sub-route", func(t *testing.T) {
		ctx := &mockPathProvider{path: "/admin/users"}
		node := NavLinkPrefix(ctx, "/admin", Text("Admin"))

		if node.Props["class"] != "active" {
			t.Errorf("class = %v, want 'active'", node.Props["class"])
		}
	})

	t.Run("active when path is deep sub-route", func(t *testing.T) {
		ctx := &mockPathProvider{path: "/admin/users/123/edit"}
		node := NavLinkPrefix(ctx, "/admin", Text("Admin"))

		if node.Props["class"] != "active" {
			t.Errorf("class = %v, want 'active'", node.Props["class"])
		}
	})

	t.Run("not active when path just starts with same chars", func(t *testing.T) {
		ctx := &mockPathProvider{path: "/administrator"}
		node := NavLinkPrefix(ctx, "/admin", Text("Admin"))

		// /administrator should NOT match /admin (must be /admin or /admin/...)
		if node.Props["class"] != nil {
			t.Errorf("class = %v, want nil (/administrator is not a sub-route of /admin)", node.Props["class"])
		}
	})

	t.Run("not active when different path", func(t *testing.T) {
		ctx := &mockPathProvider{path: "/home"}
		node := NavLinkPrefix(ctx, "/admin", Text("Admin"))

		if node.Props["class"] != nil {
			t.Errorf("class = %v, want nil", node.Props["class"])
		}
	})

	t.Run("nil context", func(t *testing.T) {
		node := NavLinkPrefix(nil, "/admin", Text("Admin"))

		if node.Props["class"] != nil {
			t.Errorf("class = %v, want nil", node.Props["class"])
		}
	})
}
