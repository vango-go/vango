package server

import (
	"testing"

	"github.com/vango-go/vango/pkg/vdom"
)

// mockComponent is a simple component for testing
type mockComponent struct {
	renderCount int
	tree        *vdom.VNode
}

func (m *mockComponent) Render() *vdom.VNode {
	m.renderCount++
	if m.tree != nil {
		return m.tree
	}
	return &vdom.VNode{
		Tag: "div",
		Children: []*vdom.VNode{
			{Tag: "span", Children: []*vdom.VNode{{Text: "Hello"}}},
		},
	}
}

func TestFuncComponent(t *testing.T) {
	renderCount := 0
	fc := FuncComponent(func() *vdom.VNode {
		renderCount++
		return &vdom.VNode{Tag: "div"}
	})

	tree := fc.Render()
	if tree == nil {
		t.Error("Render should return a VNode")
	}
	if tree.Tag != "div" {
		t.Errorf("Tag = %s, want div", tree.Tag)
	}
	if renderCount != 1 {
		t.Errorf("renderCount = %d, want 1", renderCount)
	}

	// Render again
	fc.Render()
	if renderCount != 2 {
		t.Errorf("renderCount = %d, want 2", renderCount)
	}
}

func TestNewComponentInstance(t *testing.T) {
	comp := &mockComponent{}
	instance := newComponentInstance(comp, nil, nil)

	if instance == nil {
		t.Fatal("newComponentInstance should not return nil")
	}
	if instance.InstanceID == "" {
		t.Error("InstanceID should not be empty")
	}
	if instance.Component != comp {
		t.Error("Component not set correctly")
	}
	if instance.Owner == nil {
		t.Error("Owner should be initialized")
	}
	if instance.IsDirty() {
		t.Error("New instance should not be dirty")
	}
}

func TestComponentInstanceRender(t *testing.T) {
	comp := &mockComponent{
		tree: &vdom.VNode{Tag: "section"},
	}
	instance := newComponentInstance(comp, nil, nil)

	tree := instance.Render()
	if tree == nil {
		t.Fatal("Render should return a VNode")
	}
	if tree.Tag != "section" {
		t.Errorf("Tag = %s, want section", tree.Tag)
	}
	if comp.renderCount != 1 {
		t.Errorf("renderCount = %d, want 1", comp.renderCount)
	}
}

func TestComponentInstanceDirtyFlag(t *testing.T) {
	comp := &mockComponent{}
	instance := newComponentInstance(comp, nil, nil)

	// Initially not dirty
	if instance.IsDirty() {
		t.Error("New instance should not be dirty")
	}

	// Mark dirty
	instance.MarkDirty()
	if !instance.IsDirty() {
		t.Error("Instance should be dirty after MarkDirty")
	}

	// Clear dirty
	instance.ClearDirty()
	if instance.IsDirty() {
		t.Error("Instance should not be dirty after ClearDirty")
	}
}

func TestComponentInstanceLastTree(t *testing.T) {
	comp := &mockComponent{
		tree: &vdom.VNode{Tag: "article"},
	}
	instance := newComponentInstance(comp, nil, nil)

	// Before first render, lastTree is nil
	if instance.LastTree() != nil {
		t.Error("lastTree should be nil before first render")
	}

	// After render, lastTree should be set
	tree := instance.Render()
	if instance.LastTree() != tree {
		t.Error("lastTree should be set after render")
	}
}

func TestComponentInstanceSetLastTree(t *testing.T) {
	comp := &mockComponent{}
	instance := newComponentInstance(comp, nil, nil)

	tree := &vdom.VNode{Tag: "main"}
	instance.SetLastTree(tree)

	if instance.LastTree() != tree {
		t.Error("SetLastTree should update lastTree")
	}
}

func TestComponentInstanceMemoryUsage(t *testing.T) {
	comp := &mockComponent{
		tree: &vdom.VNode{
			Tag: "div",
			Children: []*vdom.VNode{
				{Tag: "span"},
				{Tag: "p"},
			},
		},
	}
	instance := newComponentInstance(comp, nil, nil)

	// Render to populate lastTree
	instance.Render()

	usage := instance.MemoryUsage()
	if usage <= 0 {
		t.Error("MemoryUsage should be positive after render")
	}
}

func TestComponentInstanceMemoryUsageNoTree(t *testing.T) {
	comp := &mockComponent{}
	instance := newComponentInstance(comp, nil, nil)

	// Without rendering, lastTree is nil
	usage := instance.MemoryUsage()
	// Should still return some base usage
	if usage < 0 {
		t.Error("MemoryUsage should not be negative")
	}
}

func TestEstimateVNodeSize(t *testing.T) {
	// Nil node
	if estimateVNodeSize(nil) != 0 {
		t.Error("estimateVNodeSize(nil) should be 0")
	}

	// Simple node
	simple := &vdom.VNode{Tag: "div"}
	size := estimateVNodeSize(simple)
	if size <= 0 {
		t.Error("estimateVNodeSize should be positive for non-nil node")
	}

	// Node with children
	withChildren := &vdom.VNode{
		Tag: "div",
		Children: []*vdom.VNode{
			{Tag: "span"},
			{Tag: "p"},
		},
	}
	childSize := estimateVNodeSize(withChildren)
	if childSize <= size {
		t.Error("Node with children should have larger size")
	}

	// Text node
	textNode := &vdom.VNode{Text: "Hello, World!"}
	textSize := estimateVNodeSize(textNode)
	if textSize <= 0 {
		t.Error("Text node should have positive size")
	}
}

func TestEstimateVNodeSizeWithProps(t *testing.T) {
	node := &vdom.VNode{
		Tag: "input",
		Props: map[string]any{
			"type":        "text",
			"placeholder": "Enter name",
			"value":       "test",
		},
	}

	size := estimateVNodeSize(node)
	if size <= 0 {
		t.Error("Node with props should have positive size")
	}
}

func TestEstimateVNodeSizeDeepTree(t *testing.T) {
	// Create a deep tree
	root := &vdom.VNode{Tag: "div"}
	current := root
	for i := 0; i < 10; i++ {
		child := &vdom.VNode{Tag: "div"}
		current.Children = []*vdom.VNode{child}
		current = child
	}

	size := estimateVNodeSize(root)
	if size <= 0 {
		t.Error("Deep tree should have positive size")
	}

	// Size should grow with depth
	shallowRoot := &vdom.VNode{Tag: "div"}
	shallowSize := estimateVNodeSize(shallowRoot)
	if size <= shallowSize {
		t.Error("Deep tree should be larger than shallow tree")
	}
}

func TestComponentRenderMultipleTimes(t *testing.T) {
	count := 0
	comp := FuncComponent(func() *vdom.VNode {
		count++
		return &vdom.VNode{Tag: "div", Props: map[string]any{"data-count": count}}
	})

	instance := newComponentInstance(comp, nil, nil)

	// Render multiple times
	for i := 0; i < 5; i++ {
		tree := instance.Render()
		if tree == nil {
			t.Fatalf("Render %d returned nil", i)
		}
	}

	if count != 5 {
		t.Errorf("count = %d, want 5", count)
	}
}

func TestComponentInstanceDispose(t *testing.T) {
	comp := &mockComponent{}
	instance := newComponentInstance(comp, nil, nil)

	// Render to set up tree
	instance.Render()

	// Dispose
	instance.Dispose()

	// Verify cleanup
	if instance.Component != nil {
		t.Error("Component should be nil after dispose")
	}
	if instance.lastTree != nil {
		t.Error("lastTree should be nil after dispose")
	}
}

func TestComponentInstanceParentChild(t *testing.T) {
	parent := newComponentInstance(&mockComponent{}, nil, nil)
	child := newComponentInstance(&mockComponent{}, parent, nil)

	parent.AddChild(child)

	if len(parent.Children) != 1 {
		t.Errorf("Parent should have 1 child, got %d", len(parent.Children))
	}
	if child.Parent != parent {
		t.Error("Child parent should be set")
	}

	parent.RemoveChild(child)
	if len(parent.Children) != 0 {
		t.Error("Parent should have no children after remove")
	}
}

func TestComponentInstanceProps(t *testing.T) {
	comp := &mockComponent{}
	instance := newComponentInstance(comp, nil, nil)

	instance.SetProp("name", "test")
	instance.SetProp("count", 42)

	if instance.GetProp("name") != "test" {
		t.Error("GetProp should return set value")
	}
	if instance.GetProp("count") != 42 {
		t.Error("GetProp should return set value")
	}
	if instance.GetProp("missing") != nil {
		t.Error("GetProp should return nil for missing key")
	}
}

func TestGenerateComponentID(t *testing.T) {
	id1 := generateComponentID()
	id2 := generateComponentID()

	if id1 == "" {
		t.Error("ID should not be empty")
	}
	if id1 == id2 {
		t.Error("IDs should be unique")
	}
	if id1[0] != 'c' {
		t.Error("ID should start with 'c'")
	}
}
