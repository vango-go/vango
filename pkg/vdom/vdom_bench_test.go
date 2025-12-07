package vdom

import (
	"fmt"
	"testing"
)

func BenchmarkElementCreation(b *testing.B) {
	b.Run("simple div", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = Div(Class("card"))
		}
	})

	b.Run("with children", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = Div(Class("card"),
				H1(Text("Title")),
				P(Text("Content")),
			)
		}
	})

	b.Run("with event handler", func(b *testing.B) {
		handler := func() {}
		for i := 0; i < b.N; i++ {
			_ = Button(OnClick(handler), Text("Click"))
		}
	})

	b.Run("complex card", func(b *testing.B) {
		handler := func() {}
		for i := 0; i < b.N; i++ {
			_ = Div(Class("card"),
				Header(
					H2(Text("Card Title")),
				),
				Main(
					P(Text("Card content goes here")),
					P(Text("More content")),
				),
				Footer(
					Button(OnClick(handler), Text("Save")),
					Button(OnClick(handler), Text("Cancel")),
				),
			)
		}
	})
}

func BenchmarkDeepTreeCreation(b *testing.B) {
	b.Run("depth 5", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = createDeepTree(5)
		}
	})

	b.Run("depth 10", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = createDeepTree(10)
		}
	})
}

func createDeepTree(depth int) *VNode {
	if depth == 0 {
		return Text("Leaf")
	}
	return Div(Class("level"), createDeepTree(depth-1))
}

func BenchmarkWideTreeCreation(b *testing.B) {
	b.Run("10 children", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = createWideTree(10)
		}
	})

	b.Run("100 children", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = createWideTree(100)
		}
	})
}

func createWideTree(width int) *VNode {
	children := make([]*VNode, width)
	for i := 0; i < width; i++ {
		children[i] = Li(Key(i), Textf("Item %d", i))
	}
	return Ul(children)
}

func BenchmarkDiffSameTree(b *testing.B) {
	tree := createLargeTree(100)
	assignBenchHIDs(tree)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Diff(tree, tree)
	}
}

func BenchmarkDiffTextChange(b *testing.B) {
	prev := Div(
		H1(Text("Old Title")),
		P(Text("Content")),
	)
	assignBenchHIDs(prev)

	next := Div(
		H1(Text("New Title")),
		P(Text("Content")),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Diff(prev, next)
	}
}

func BenchmarkDiffAttributeChange(b *testing.B) {
	prev := Div(Class("old"), ID("test"))
	assignBenchHIDs(prev)

	next := Div(Class("new"), ID("test"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Diff(prev, next)
	}
}

func BenchmarkDiffUnkeyedChildren(b *testing.B) {
	b.Run("10 children", func(b *testing.B) {
		prev := createUnkeyedList(10)
		assignBenchHIDs(prev)
		next := createUnkeyedListModified(10)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Diff(prev, next)
		}
	})

	b.Run("100 children", func(b *testing.B) {
		prev := createUnkeyedList(100)
		assignBenchHIDs(prev)
		next := createUnkeyedListModified(100)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Diff(prev, next)
		}
	})
}

func BenchmarkDiffKeyedReorder(b *testing.B) {
	b.Run("10 children", func(b *testing.B) {
		prev := createKeyedList(10)
		assignBenchHIDs(prev)
		next := createReorderedKeyedList(10)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Diff(prev, next)
		}
	})

	b.Run("100 children", func(b *testing.B) {
		prev := createKeyedList(100)
		assignBenchHIDs(prev)
		next := createReorderedKeyedList(100)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Diff(prev, next)
		}
	})
}

func BenchmarkDiffKeyedAddition(b *testing.B) {
	prev := createKeyedList(100)
	assignBenchHIDs(prev)
	next := createKeyedListWithAddition(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Diff(prev, next)
	}
}

func BenchmarkDiffKeyedRemoval(b *testing.B) {
	prev := createKeyedList(100)
	assignBenchHIDs(prev)
	next := createKeyedListWithRemoval(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Diff(prev, next)
	}
}

func BenchmarkDiffLargeTree(b *testing.B) {
	b.Run("100 nodes", func(b *testing.B) {
		prev := createLargeTree(100)
		assignBenchHIDs(prev)
		next := createLargeTreeWithChange(100)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Diff(prev, next)
		}
	})

	b.Run("1000 nodes", func(b *testing.B) {
		prev := createLargeTree(1000)
		assignBenchHIDs(prev)
		next := createLargeTreeWithChange(1000)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Diff(prev, next)
		}
	})
}

func BenchmarkHIDGeneration(b *testing.B) {
	gen := NewHIDGenerator()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = gen.Next()
	}
}

func BenchmarkAssignHIDs(b *testing.B) {
	b.Run("small tree", func(b *testing.B) {
		tree := createInteractiveTree(10)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			gen := NewHIDGenerator()
			ClearHIDs(tree)
			AssignHIDs(tree, gen)
		}
	})

	b.Run("large tree", func(b *testing.B) {
		tree := createInteractiveTree(100)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			gen := NewHIDGenerator()
			ClearHIDs(tree)
			AssignHIDs(tree, gen)
		}
	})
}

func BenchmarkRange(b *testing.B) {
	items := make([]int, 100)
	for i := range items {
		items[i] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Range(items, func(item int, index int) *VNode {
			return Li(Key(index), Textf("Item %d", item))
		})
	}
}

func BenchmarkFragment(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Fragment(
			Div(),
			Span(),
			P(),
			Button(),
			Input(),
		)
	}
}

// Helper functions for benchmarks

func assignBenchHIDs(node *VNode) {
	gen := NewHIDGenerator()
	assignAllHIDsBench(node, gen)
}

func assignAllHIDsBench(node *VNode, gen *HIDGenerator) {
	if node == nil {
		return
	}
	if node.Kind == KindElement || node.Kind == KindText {
		node.HID = gen.Next()
	}
	for _, child := range node.Children {
		assignAllHIDsBench(child, gen)
	}
}

func createLargeTree(n int) *VNode {
	children := make([]*VNode, n/10)
	for i := range children {
		items := make([]*VNode, 10)
		for j := range items {
			items[j] = Li(Textf("Item %d", i*10+j))
		}
		children[i] = Ul(items)
	}
	return Div(Class("container"), children)
}

func createLargeTreeWithChange(n int) *VNode {
	children := make([]*VNode, n/10)
	for i := range children {
		items := make([]*VNode, 10)
		for j := range items {
			text := fmt.Sprintf("Item %d", i*10+j)
			if i == 0 && j == 0 {
				text = "Changed Item"
			}
			items[j] = Li(Text(text))
		}
		children[i] = Ul(items)
	}
	return Div(Class("container"), children)
}

func createUnkeyedList(n int) *VNode {
	children := make([]*VNode, n)
	for i := range children {
		children[i] = Li(Textf("Item %d", i))
	}
	return Ul(children)
}

func createUnkeyedListModified(n int) *VNode {
	children := make([]*VNode, n)
	for i := range children {
		text := fmt.Sprintf("Item %d", i)
		if i == n/2 {
			text = "Modified"
		}
		children[i] = Li(Text(text))
	}
	return Ul(children)
}

func createKeyedList(n int) *VNode {
	children := make([]*VNode, n)
	for i := range children {
		children[i] = Li(Key(fmt.Sprintf("key-%d", i)), Textf("Item %d", i))
	}
	return Ul(children)
}

func createReorderedKeyedList(n int) *VNode {
	children := make([]*VNode, n)
	// Reverse order
	for i := range children {
		j := n - 1 - i
		children[i] = Li(Key(fmt.Sprintf("key-%d", j)), Textf("Item %d", j))
	}
	return Ul(children)
}

func createKeyedListWithAddition(n int) *VNode {
	children := make([]*VNode, n+1)
	for i := 0; i < n/2; i++ {
		children[i] = Li(Key(fmt.Sprintf("key-%d", i)), Textf("Item %d", i))
	}
	children[n/2] = Li(Key("new-key"), Text("New Item"))
	for i := n/2 + 1; i <= n; i++ {
		children[i] = Li(Key(fmt.Sprintf("key-%d", i-1)), Textf("Item %d", i-1))
	}
	return Ul(children)
}

func createKeyedListWithRemoval(n int) *VNode {
	children := make([]*VNode, n-1)
	j := 0
	for i := 0; i < n; i++ {
		if i == n/2 {
			continue // Skip middle item
		}
		children[j] = Li(Key(fmt.Sprintf("key-%d", i)), Textf("Item %d", i))
		j++
	}
	return Ul(children)
}

func createInteractiveTree(n int) *VNode {
	handler := func() {}
	children := make([]*VNode, n)
	for i := range children {
		children[i] = Button(OnClick(handler), Textf("Button %d", i))
	}
	return Div(children)
}
