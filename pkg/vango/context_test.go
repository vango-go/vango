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
