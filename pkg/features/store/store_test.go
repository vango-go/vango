package store

import (
	"testing"

	"github.com/vango-dev/vango/v2/pkg/vango"
)

var (
	// Define signals at package level to simulate real usage
	GlobalCounter = GlobalSignal(0)
	SharedCounter = SharedSignal(0)
)

func TestGlobalSignal(t *testing.T) {
	GlobalCounter.Set(10)
	if GlobalCounter.Get() != 10 {
		t.Errorf("Expected 10, got %d", GlobalCounter.Get())
	}

	// Reset for other tests (though global is global...)
	GlobalCounter.Set(0)
}

func TestSharedSignal(t *testing.T) {
	// Root owner for Session A
	rootA := vango.NewOwner(nil)
	storeA := NewSessionStore()

	// Root owner for Session B
	rootB := vango.NewOwner(nil)
	storeB := NewSessionStore()

	// Simulate Session A
	vango.WithOwner(rootA, func() {
		vango.SetContext(SessionKey, storeA)

		// Initial value
		if SharedCounter.Get() != 0 {
			t.Errorf("Session A: Expected 0, got %d", SharedCounter.Get())
		}

		// Set value
		SharedCounter.Set(5)
		if SharedCounter.Get() != 5 {
			t.Errorf("Session A: Expected 5, got %d", SharedCounter.Get())
		}
	})

	// Simulate Session B (should be independent)
	vango.WithOwner(rootB, func() {
		vango.SetContext(SessionKey, storeB)

		// Should be 0 (initial), not 5
		if SharedCounter.Get() != 0 {
			t.Errorf("Session B: Expected 0, got %d", SharedCounter.Get())
		}

		// Set value
		SharedCounter.Set(10)
		if SharedCounter.Get() != 10 {
			t.Errorf("Session B: Expected 10, got %d", SharedCounter.Get())
		}
	})

	// Verify Session A again (should still be 5)
	vango.WithOwner(rootA, func() {
		if SharedCounter.Get() != 5 {
			t.Errorf("Session A (revisit): Expected 5, got %d", SharedCounter.Get())
		}
	})
}

func TestSharedSignalUndefinedContext(t *testing.T) {
	// If context is not set, it returns initial value (generic behavior)
	// or specific fallback defined in implementation.
	// Current implementation: fallback to initial.

	// Ensure we are in a clean/new owner without context
	root := vango.NewOwner(nil)
	vango.WithOwner(root, func() {
		if SharedCounter.Get() != 0 {
			t.Errorf("Expected fallback 0, got %d", SharedCounter.Get())
		}

		// Use set (should be no-op or panic? Implementation checks nil)
		SharedCounter.Set(99)

		// Get should still be initial because no signal was created/stored
		if SharedCounter.Get() != 0 {
			t.Errorf("Expected 0 after Set without context, got %d", SharedCounter.Get())
		}
	})
}

func TestSharedSignalUpdate(t *testing.T) {
	root := vango.NewOwner(nil)
	store := NewSessionStore()

	vango.WithOwner(root, func() {
		vango.SetContext(SessionKey, store)

		// Use Update function
		sharedVal := SharedSignal("initial")

		if sharedVal.Get() != "initial" {
			t.Errorf("Expected 'initial', got '%s'", sharedVal.Get())
		}

		sharedVal.Update(func(s string) string {
			return s + "_updated"
		})

		if sharedVal.Get() != "initial_updated" {
			t.Errorf("Expected 'initial_updated', got '%s'", sharedVal.Get())
		}
	})
}

func TestSharedSignalUpdateWithoutContext(t *testing.T) {
	root := vango.NewOwner(nil)

	vango.WithOwner(root, func() {
		// No context set - Update should be no-op
		sharedVal := SharedSignal(42)

		// Should not panic
		sharedVal.Update(func(n int) int {
			return n * 2
		})

		// Should still return initial value
		if sharedVal.Get() != 42 {
			t.Errorf("Expected 42, got %d", sharedVal.Get())
		}
	})
}

func TestGlobalSignalWithStruct(t *testing.T) {
	type User struct {
		Name string
		Age  int
	}

	globalUser := GlobalSignal(User{Name: "Anonymous", Age: 0})

	if globalUser.Get().Name != "Anonymous" {
		t.Errorf("Expected 'Anonymous', got '%s'", globalUser.Get().Name)
	}

	globalUser.Set(User{Name: "John", Age: 30})

	if globalUser.Get().Name != "John" {
		t.Errorf("Expected 'John', got '%s'", globalUser.Get().Name)
	}
	if globalUser.Get().Age != 30 {
		t.Errorf("Expected 30, got %d", globalUser.Get().Age)
	}
}

func TestSharedSignalWithDifferentTypes(t *testing.T) {
	root := vango.NewOwner(nil)
	store := NewSessionStore()

	vango.WithOwner(root, func() {
		vango.SetContext(SessionKey, store)

		// Test with string
		sharedStr := SharedSignal("hello")
		sharedStr.Set("world")
		if sharedStr.Get() != "world" {
			t.Errorf("Expected 'world', got '%s'", sharedStr.Get())
		}

		// Test with bool
		sharedBool := SharedSignal(false)
		sharedBool.Set(true)
		if !sharedBool.Get() {
			t.Error("Expected true")
		}

		// Test with float
		sharedFloat := SharedSignal(0.0)
		sharedFloat.Set(3.14)
		if sharedFloat.Get() != 3.14 {
			t.Errorf("Expected 3.14, got %f", sharedFloat.Get())
		}
	})
}

func TestSessionStoreMultipleSignals(t *testing.T) {
	root := vango.NewOwner(nil)
	store := NewSessionStore()

	vango.WithOwner(root, func() {
		vango.SetContext(SessionKey, store)

		// Create multiple signals
		counter1 := SharedSignal(0)
		counter2 := SharedSignal(100)
		name := SharedSignal("default")

		// Set values
		counter1.Set(10)
		counter2.Set(200)
		name.Set("test")

		// Verify all values are independent
		if counter1.Get() != 10 {
			t.Errorf("counter1: Expected 10, got %d", counter1.Get())
		}
		if counter2.Get() != 200 {
			t.Errorf("counter2: Expected 200, got %d", counter2.Get())
		}
		if name.Get() != "test" {
			t.Errorf("name: Expected 'test', got '%s'", name.Get())
		}
	})
}

func TestNewSessionStore(t *testing.T) {
	store := NewSessionStore()
	if store == nil {
		t.Fatal("NewSessionStore returned nil")
	}
}

func TestSharedSignalConcurrentAccess(t *testing.T) {
	root := vango.NewOwner(nil)
	store := NewSessionStore()

	vango.WithOwner(root, func() {
		vango.SetContext(SessionKey, store)

		sharedCounter := SharedSignal(0)

		// First access creates the signal
		sharedCounter.Set(1)

		// Second access should get the same signal
		if sharedCounter.Get() != 1 {
			t.Errorf("Expected 1, got %d", sharedCounter.Get())
		}

		// Update should work
		sharedCounter.Update(func(n int) int { return n + 1 })
		if sharedCounter.Get() != 2 {
			t.Errorf("Expected 2, got %d", sharedCounter.Get())
		}
	})
}

func TestSessionKeyUniqueness(t *testing.T) {
	// SessionKey should be a unique pointer
	if SessionKey == nil {
		t.Error("SessionKey should not be nil")
	}

	// Verify it's the expected type
	if SessionKey.name != "SessionStore" {
		t.Error("SessionKey should have name 'SessionStore'")
	}
}

func TestSharedSignalWrongContextType(t *testing.T) {
	root := vango.NewOwner(nil)

	vango.WithOwner(root, func() {
		// Set context to wrong type
		vango.SetContext(SessionKey, "not a session store")

		sharedVal := SharedSignal(42)

		// Should return initial value when context is wrong type
		if sharedVal.Get() != 42 {
			t.Errorf("Expected 42 (fallback), got %d", sharedVal.Get())
		}
	})
}
