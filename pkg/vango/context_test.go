package vango

import "testing"

func TestOwnerSetGetValue(t *testing.T) {
	owner := NewOwner(nil)

	// Initially no value
	if owner.GetValue("key") != nil {
		t.Error("expected nil for non-existent key")
	}

	// Set and get
	owner.SetValue("key", "value")
	if owner.GetValue("key") != "value" {
		t.Errorf("expected 'value', got %v", owner.GetValue("key"))
	}

	// Different types
	owner.SetValue("intKey", 42)
	if owner.GetValue("intKey") != 42 {
		t.Errorf("expected 42, got %v", owner.GetValue("intKey"))
	}
}

func TestOwnerValueInheritance(t *testing.T) {
	parent := NewOwner(nil)
	child := NewOwner(parent)
	grandchild := NewOwner(child)

	// Set value on parent
	parent.SetValue("inherited", "from parent")

	// Child and grandchild should see it
	if child.GetValue("inherited") != "from parent" {
		t.Errorf("child should inherit from parent")
	}
	if grandchild.GetValue("inherited") != "from parent" {
		t.Errorf("grandchild should inherit from parent")
	}

	// Child can override
	child.SetValue("inherited", "from child")
	if child.GetValue("inherited") != "from child" {
		t.Errorf("child should see own value")
	}
	if grandchild.GetValue("inherited") != "from child" {
		t.Errorf("grandchild should see child's value")
	}
	if parent.GetValue("inherited") != "from parent" {
		t.Errorf("parent value should be unchanged")
	}
}

func TestSetGetContext(t *testing.T) {
	owner := NewOwner(nil)

	WithOwner(owner, func() {
		SetContext("theme", "dark")
		if GetContext("theme") != "dark" {
			t.Errorf("expected 'dark', got %v", GetContext("theme"))
		}

		// Non-existent key
		if GetContext("nonexistent") != nil {
			t.Error("expected nil for non-existent key")
		}
	})
}

func TestContextWithNoOwner(t *testing.T) {
	// SetContext with no owner should be safe (no-op)
	SetContext("key", "value")

	// GetContext with no owner returns nil
	if GetContext("key") != nil {
		t.Error("expected nil when no owner")
	}
}

func TestContextInheritanceViaSetContext(t *testing.T) {
	parent := NewOwner(nil)
	child := NewOwner(parent)

	WithOwner(parent, func() {
		SetContext("parentKey", "parentValue")
	})

	WithOwner(child, func() {
		// Should inherit from parent
		if GetContext("parentKey") != "parentValue" {
			t.Error("child should inherit context from parent")
		}
	})
}

// =============================================================================
// Context API Tests (CreateContext, Provider, Use)
// =============================================================================

func TestCreateContextWithDefaultValue(t *testing.T) {
	ctx := CreateContext("default")

	if ctx == nil {
		t.Fatal("CreateContext returned nil")
	}
	if ctx.Default() != "default" {
		t.Errorf("Default() = %v, want 'default'", ctx.Default())
	}
}

func TestContextUseWithoutProvider(t *testing.T) {
	ctx := CreateContext("default-value")

	// Without a provider, Use() should return the default value
	owner := NewOwner(nil)
	owner.StartRender()
	WithOwner(owner, func() {
		val := ctx.Use()
		if val != "default-value" {
			t.Errorf("Use() without provider = %v, want 'default-value'", val)
		}
	})
	owner.EndRender()
}

func TestContextProviderScoping(t *testing.T) {
	ctx := CreateContext("default")

	// Create a hierarchy: root -> providerOwner (has context) / siblingOwner (no context)
	root := NewOwner(nil)

	// Simulate rendering the provider component
	providerOwner := NewOwner(root)
	providerOwner.StartRender()
	WithOwner(providerOwner, func() {
		// Provider stores contextValue in its own scope
		sig := &Signal[string]{
			base: signalBase{id: nextID()},
			value: "provided-value",
		}
		cv := &contextValue[string]{signal: sig}
		providerOwner.SetValue(ctx.key, cv)
	})
	providerOwner.EndRender()

	// Create a sibling owner (not a child of providerOwner)
	siblingOwner := NewOwner(root)
	siblingOwner.StartRender()
	WithOwner(siblingOwner, func() {
		// Sibling should NOT see the provider's value
		val := ctx.Use()
		if val != "default" {
			t.Errorf("Sibling should not see provider value, got %v", val)
		}
	})
	siblingOwner.EndRender()

	// Create a descendant of the provider
	descendantOwner := NewOwner(providerOwner)
	descendantOwner.StartRender()
	WithOwner(descendantOwner, func() {
		// Descendant SHOULD see the provider's value
		val := ctx.Use()
		if val != "provided-value" {
			t.Errorf("Descendant should see provider value, got %v", val)
		}
	})
	descendantOwner.EndRender()
}

func TestContextUseIsReactive(t *testing.T) {
	ctx := CreateContext("initial")

	root := NewOwner(nil)
	providerOwner := NewOwner(root)

	// Create the context signal in provider scope
	sig := &Signal[string]{
		base:  signalBase{id: nextID()},
		value: "initial",
	}
	cv := &contextValue[string]{signal: sig}
	providerOwner.SetValue(ctx.key, cv)

	// Create a mock listener to track subscriptions
	listener := &mockListener{id: nextID()}

	// Render a component that uses the context
	childOwner := NewOwner(providerOwner)
	childOwner.StartRender()
	WithOwner(childOwner, func() {
		// Set the listener to track subscriptions
		WithListener(listener, func() {
			val := ctx.Use()
			if val != "initial" {
				t.Errorf("Initial value = %v, want 'initial'", val)
			}
		})
	})
	childOwner.EndRender()

	// Check that the listener was subscribed
	if !listener.dirty {
		// The listener should be subscribed to the signal
		// Let's update the signal and see if the listener is notified
		sig.Set("updated")
		if !listener.dirty {
			t.Error("Use() should subscribe to context changes, but listener was not notified")
		}
	}
}

func TestContextNestedProviders(t *testing.T) {
	ctx := CreateContext("default")

	root := NewOwner(nil)

	// Outer provider
	outerOwner := NewOwner(root)
	outerSig := &Signal[string]{
		base:  signalBase{id: nextID()},
		value: "outer",
	}
	outerOwner.SetValue(ctx.key, &contextValue[string]{signal: outerSig})

	// Inner provider (overrides outer)
	innerOwner := NewOwner(outerOwner)
	innerSig := &Signal[string]{
		base:  signalBase{id: nextID()},
		value: "inner",
	}
	innerOwner.SetValue(ctx.key, &contextValue[string]{signal: innerSig})

	// Deepest child (should see inner value)
	deepestOwner := NewOwner(innerOwner)
	deepestOwner.StartRender()
	WithOwner(deepestOwner, func() {
		val := ctx.Use()
		if val != "inner" {
			t.Errorf("Deepest should see inner value, got %v", val)
		}
	})
	deepestOwner.EndRender()

	// Middle child (between outer and inner, should see outer)
	// Actually this is a child of outer but not of inner...
	// Let me create the right hierarchy
	middleOwner := NewOwner(outerOwner) // Sibling of innerOwner
	middleOwner.StartRender()
	WithOwner(middleOwner, func() {
		val := ctx.Use()
		if val != "outer" {
			t.Errorf("Middle (sibling of inner) should see outer value, got %v", val)
		}
	})
	middleOwner.EndRender()
}

func TestContextHookOrderTracking(t *testing.T) {
	// Enable debug mode for this test
	oldDebugMode := DebugMode
	DebugMode = true
	defer func() { DebugMode = oldDebugMode }()

	ctx := CreateContext("default")

	owner := NewOwner(nil)

	// First render - establish hook order
	owner.StartRender()
	WithOwner(owner, func() {
		ctx.Use()
	})
	owner.EndRender()

	// Second render - same order should work
	owner.StartRender()
	WithOwner(owner, func() {
		ctx.Use()
	})
	owner.EndRender() // Should not panic

	// Third render - missing hook should panic
	owner.StartRender()
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when hook order changes")
		}
	}()
	// Don't call ctx.Use() - this should cause a panic
	WithOwner(owner, func() {
		// Missing ctx.Use()
	})
	owner.EndRender()
}

func TestContextProviderRerender(t *testing.T) {
	ctx := CreateContext("default")

	root := NewOwner(nil)
	providerOwner := NewOwner(root)

	// First render
	providerOwner.StartRender()
	WithOwner(providerOwner, func() {
		// First render creates the contextValue
		sig := &Signal[string]{
			base:  signalBase{id: nextID()},
			value: "v1",
		}
		providerOwner.SetValue(ctx.key, &contextValue[string]{signal: sig})
	})
	providerOwner.EndRender()

	// Get the signal for later checks
	cv := providerOwner.GetValueLocal(ctx.key).(*contextValue[string])
	originalSignal := cv.signal

	// Second render - update the value using setQuietly (as Provider does)
	providerOwner.StartRender()
	WithOwner(providerOwner, func() {
		// Check if we have existing contextValue
		existing := providerOwner.GetValueLocal(ctx.key)
		if existingCV, ok := existing.(*contextValue[string]); ok {
			// Update existing signal quietly (no notifications during render)
			existingCV.signal.setQuietly("v2")
		}
	})
	providerOwner.EndRender()

	// Same signal should be reused
	if cv.signal != originalSignal {
		t.Error("Provider should reuse the same signal across renders")
	}

	// Value should be updated
	if cv.signal.Get() != "v2" {
		t.Errorf("Signal value = %v, want 'v2'", cv.signal.Get())
	}
}

func TestContextProviderDoesNotNotifyDuringRender(t *testing.T) {
	ctx := CreateContext("initial")

	root := NewOwner(nil)
	providerOwner := NewOwner(root)

	// Create the context signal in provider scope
	sig := &Signal[string]{
		base:  signalBase{id: nextID()},
		value: "initial",
	}
	cv := &contextValue[string]{signal: sig}
	providerOwner.SetValue(ctx.key, cv)

	// Create a mock listener to track notifications
	listener := &mockListener{id: nextID()}
	sig.base.subscribe(listener)

	// Simulate Provider re-render updating the value quietly
	providerOwner.StartRender()
	WithOwner(providerOwner, func() {
		// This is what Provider.Render() does - setQuietly, not Set
		cv.signal.setQuietly("updated")
	})
	providerOwner.EndRender()

	// Listener should NOT have been notified (no dirty flag)
	if listener.dirty {
		t.Error("setQuietly should not notify subscribers during render")
	}

	// But the value should be updated
	if cv.signal.Peek() != "updated" {
		t.Errorf("Signal value = %v, want 'updated'", cv.signal.Peek())
	}
}

func TestGetValueLocal(t *testing.T) {
	parent := NewOwner(nil)
	child := NewOwner(parent)

	// Set value on parent
	parent.SetValue("key", "parent-value")

	// GetValueLocal on child should return nil (not inherited)
	if child.GetValueLocal("key") != nil {
		t.Error("GetValueLocal should not inherit from parent")
	}

	// GetValue on child should return parent's value
	if child.GetValue("key") != "parent-value" {
		t.Error("GetValue should inherit from parent")
	}

	// Set value on child
	child.SetValue("key", "child-value")

	// Now GetValueLocal should return child's value
	if child.GetValueLocal("key") != "child-value" {
		t.Errorf("GetValueLocal = %v, want 'child-value'", child.GetValueLocal("key"))
	}
}

// mockListener implements Listener for testing
type mockListener struct {
	id    uint64
	dirty bool
}

func (m *mockListener) ID() uint64 {
	return m.id
}

func (m *mockListener) MarkDirty() {
	m.dirty = true
}
