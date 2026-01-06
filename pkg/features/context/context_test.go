package context

import (
	"testing"

	"github.com/vango-go/vango/pkg/vango"
	"github.com/vango-go/vango/pkg/vdom"
)

func TestCreate(t *testing.T) {
	tests := []struct {
		name         string
		defaultValue any
	}{
		{"string context", "default"},
		{"int context", 42},
		{"bool context", true},
		{"struct context", struct{ Name string }{"test"}},
		{"nil context", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := Create(tt.defaultValue)
			if ctx == nil {
				t.Error("Create() returned nil")
			}
		})
	}
}

// Test that Use() returns default when there's no provider
func TestContextDefault(t *testing.T) {
	ctx := Create("default")
	if val := ctx.Use(); val != "default" {
		t.Errorf("got %v, want default", val)
	}
}

func TestProvider(t *testing.T) {
	ctx := Create("default")

	// Test that Provider returns a valid VNode
	node := ctx.Provider("provided-value", vdom.Text("child"))

	if node == nil {
		t.Fatal("Provider() returned nil")
	}

	if node.Kind != vdom.KindComponent {
		t.Errorf("Provider() Kind = %v, want KindComponent", node.Kind)
	}

	if node.Comp == nil {
		t.Error("Provider() Comp is nil")
	}

	// Render the component within an owner context
	root := vango.NewOwner(nil)
	root.StartRender()
	vango.WithOwner(root, func() {
		rendered := node.Comp.Render()
		if rendered == nil {
			t.Error("Provider component Render() returned nil")
		}

		// After rendering, context should be set in the provider's scope
		// Create a child owner to simulate a descendant component
		child := vango.NewOwner(root)
		child.StartRender()
		vango.WithOwner(child, func() {
			val := ctx.Use()
			if val != "provided-value" {
				t.Errorf("After Provider render, Use() = %v, want 'provided-value'", val)
			}
		})
		child.EndRender()
	})
	root.EndRender()
}

func TestProviderWithMultipleChildren(t *testing.T) {
	ctx := Create(0)

	// Test Provider with multiple children
	node := ctx.Provider(42,
		vdom.Div(vdom.Text("child1")),
		vdom.Span(vdom.Text("child2")),
		vdom.P(vdom.Text("child3")),
	)

	if node == nil {
		t.Fatal("Provider() returned nil")
	}

	root := vango.NewOwner(nil)
	root.StartRender()
	vango.WithOwner(root, func() {
		rendered := node.Comp.Render()

		// Should be a fragment with 3 children
		if rendered.Kind != vdom.KindFragment {
			t.Errorf("Provider children should render as Fragment, got %v", rendered.Kind)
		}
	})
	root.EndRender()
}

func TestNestedProvidersByRenderingComponents(t *testing.T) {
	ctx := Create("default")

	// Build a tree: outer provider -> inner provider -> consumer
	outerNode := ctx.Provider("outer",
		ctx.Provider("inner",
			vdom.Text("child"),
		),
	)

	root := vango.NewOwner(nil)
	root.StartRender()
	vango.WithOwner(root, func() {
		// Render outer provider
		outerRendered := outerNode.Comp.Render()
		if outerRendered == nil {
			t.Fatal("Outer provider render returned nil")
		}
	})
	root.EndRender()
}

func TestMultipleContexts(t *testing.T) {
	// Test that multiple different contexts can coexist
	themeCtx := Create("light")
	userCtx := Create("anonymous")
	countCtx := Create(0)

	root := vango.NewOwner(nil)

	// Create providers for each context
	themeProvider := themeCtx.Provider("dark",
		userCtx.Provider("john",
			countCtx.Provider(42,
				vdom.Text("child"),
			),
		),
	)

	root.StartRender()
	vango.WithOwner(root, func() {
		// Render the provider chain - each Provider creates its own owner scope
		rendered := themeProvider.Comp.Render()
		if rendered == nil {
			t.Fatal("Provider render failed")
		}
	})
	root.EndRender()
}

func TestContextWithStructValue(t *testing.T) {
	type Theme struct {
		Primary   string
		Secondary string
		Dark      bool
	}

	defaultTheme := Theme{Primary: "blue", Secondary: "gray", Dark: false}
	ctx := Create(defaultTheme)

	// Without provider, should get default
	if ctx.Use() != defaultTheme {
		t.Error("Expected default theme")
	}
}

func TestContextWithPointerValue(t *testing.T) {
	type User struct {
		ID   int
		Name string
	}

	ctx := Create[*User](nil)

	// Default should be nil
	if ctx.Use() != nil {
		t.Error("Expected nil default")
	}
}

func TestContextWithSliceValue(t *testing.T) {
	ctx := Create([]string{"default"})

	// Default value test
	got := ctx.Use()
	if len(got) != 1 || got[0] != "default" {
		t.Errorf("Use() = %v, want [default]", got)
	}
}
