package context

import (
	"testing"

	"github.com/vango-dev/vango/v2/pkg/vango"
	"github.com/vango-dev/vango/v2/pkg/vdom"
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
			if ctx.defaultValue != tt.defaultValue {
				t.Errorf("Create() defaultValue = %v, want %v", ctx.defaultValue, tt.defaultValue)
			}
		})
	}
}

func TestContext(t *testing.T) {
	// 1. Define context
	defaultValue := "default"
	ctx := Create(defaultValue)

	if ctx.Use() != defaultValue {
		t.Errorf("Expected default value '%s', got '%s'", defaultValue, ctx.Use())
	}

	// 2. Mock Owner hierarchy to simulate component tree
	// Root owner
	root := vango.NewOwner(nil)

	vango.WithOwner(root, func() {
		// Set context in parent
		// Simulate Provider logic manually since we can't execute component
		vango.SetContext(ctx, "provided")

		// Create child owner simulating nested component
		child := vango.NewOwner(root)

		vango.WithOwner(child, func() {
			val := ctx.Use()
			if val != "provided" {
				t.Errorf("Expected 'provided', got '%v'", val)
			}

			// Test nesting/overriding
			vango.SetContext(ctx, "nested")
			val = ctx.Use()
			if val != "nested" {
				t.Errorf("Expected 'nested', got '%v'", val)
			}
		})

		// Parent should still be "provided"
		val := ctx.Use()
		if val != "provided" {
			t.Errorf("Parent context should remain 'provided', got '%v'", val)
		}
	})
}

// Additional test to verify default value without owner
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
	vango.WithOwner(root, func() {
		rendered := node.Comp.Render()
		if rendered == nil {
			t.Error("Provider component Render() returned nil")
		}

		// After rendering, context should be set
		val := ctx.Use()
		if val != "provided-value" {
			t.Errorf("After Provider render, Use() = %v, want 'provided-value'", val)
		}
	})
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
	vango.WithOwner(root, func() {
		rendered := node.Comp.Render()

		// Should be a fragment with 3 children
		if rendered.Kind != vdom.KindFragment {
			t.Errorf("Provider children should render as Fragment, got %v", rendered.Kind)
		}
	})
}

func TestUseWithWrongType(t *testing.T) {
	// Create a string context
	ctx := Create("default")

	root := vango.NewOwner(nil)
	vango.WithOwner(root, func() {
		// Set context with wrong type (int instead of string)
		// This simulates a type mismatch scenario
		vango.SetContext(ctx, 12345) // wrong type

		// Use should return default because type assertion fails
		val := ctx.Use()
		if val != "default" {
			t.Errorf("Use() with wrong type should return default, got %v", val)
		}
	})
}

func TestNestedProviders(t *testing.T) {
	ctx := Create("default")

	root := vango.NewOwner(nil)

	vango.WithOwner(root, func() {
		// Outer provider
		vango.SetContext(ctx, "outer")

		if ctx.Use() != "outer" {
			t.Errorf("Expected 'outer', got %v", ctx.Use())
		}

		// Create nested owner (simulating nested component)
		inner := vango.NewOwner(root)
		vango.WithOwner(inner, func() {
			// Inner provider overrides
			vango.SetContext(ctx, "inner")

			if ctx.Use() != "inner" {
				t.Errorf("Expected 'inner', got %v", ctx.Use())
			}

			// Even deeper nesting
			deepest := vango.NewOwner(inner)
			vango.WithOwner(deepest, func() {
				// Should inherit from inner
				if ctx.Use() != "inner" {
					t.Errorf("Expected 'inner' (inherited), got %v", ctx.Use())
				}

				// Override at deepest level
				vango.SetContext(ctx, "deepest")
				if ctx.Use() != "deepest" {
					t.Errorf("Expected 'deepest', got %v", ctx.Use())
				}
			})

			// Back to inner scope
			if ctx.Use() != "inner" {
				t.Errorf("Expected 'inner' after exiting deepest, got %v", ctx.Use())
			}
		})

		// Back to outer scope
		if ctx.Use() != "outer" {
			t.Errorf("Expected 'outer' after exiting inner, got %v", ctx.Use())
		}
	})
}

func TestMultipleContexts(t *testing.T) {
	// Test that multiple different contexts can coexist
	themeCtx := Create("light")
	userCtx := Create("anonymous")
	countCtx := Create(0)

	root := vango.NewOwner(nil)
	vango.WithOwner(root, func() {
		vango.SetContext(themeCtx, "dark")
		vango.SetContext(userCtx, "john")
		vango.SetContext(countCtx, 42)

		if themeCtx.Use() != "dark" {
			t.Errorf("themeCtx.Use() = %v, want 'dark'", themeCtx.Use())
		}
		if userCtx.Use() != "john" {
			t.Errorf("userCtx.Use() = %v, want 'john'", userCtx.Use())
		}
		if countCtx.Use() != 42 {
			t.Errorf("countCtx.Use() = %v, want 42", countCtx.Use())
		}
	})
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

	root := vango.NewOwner(nil)
	vango.WithOwner(root, func() {
		customTheme := Theme{Primary: "red", Secondary: "black", Dark: true}
		vango.SetContext(ctx, customTheme)

		got := ctx.Use()
		if got != customTheme {
			t.Errorf("Use() = %+v, want %+v", got, customTheme)
		}
	})
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

	root := vango.NewOwner(nil)
	vango.WithOwner(root, func() {
		user := &User{ID: 1, Name: "Alice"}
		vango.SetContext(ctx, user)

		got := ctx.Use()
		if got == nil {
			t.Fatal("Use() returned nil")
		}
		if got.ID != 1 || got.Name != "Alice" {
			t.Errorf("Use() = %+v, want {ID:1 Name:Alice}", got)
		}
	})
}

func TestContextWithSliceValue(t *testing.T) {
	ctx := Create([]string{"default"})

	root := vango.NewOwner(nil)
	vango.WithOwner(root, func() {
		items := []string{"a", "b", "c"}
		vango.SetContext(ctx, items)

		got := ctx.Use()
		if len(got) != 3 {
			t.Errorf("Use() len = %d, want 3", len(got))
		}
	})
}
