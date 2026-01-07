package features_test

import (
	"errors"
	"testing"
	"time"

	"github.com/vango-go/vango/pkg/features/context"
	"github.com/vango-go/vango/pkg/features/form"
	"github.com/vango-go/vango/pkg/features/resource"
	"github.com/vango-go/vango/pkg/features/store"
	"github.com/vango-go/vango/pkg/vango"
	"github.com/vango-go/vango/pkg/vdom"
)

// Integration tests verify that features packages work together correctly.
// These test common workflows that span multiple packages.

// TestFormValidationWorkflow tests the complete form submission workflow
// with validation, field access, and error handling.
func TestFormValidationWorkflow(t *testing.T) {
	type ContactForm struct {
		Name    string `form:"name" validate:"required"`
		Email   string `form:"email" validate:"required,email"`
		Message string `form:"message" validate:"required,min=10"`
	}

	f := form.UseForm(ContactForm{})

	// Initially form should be valid (no validation run yet)
	if !f.IsValid() {
		t.Error("Form should be initially valid")
	}

	// Set values that should trigger validation errors
	f.Set("name", "")
	f.Set("email", "invalid-email")
	f.Set("message", "short")

	// Force validation
	f.Validate()

	// Check validation state
	if f.IsValid() {
		t.Error("Form should be invalid after validation")
	}

	// Check specific errors
	if !f.HasError("name") {
		t.Error("Name should have required error")
	}
	if !f.HasError("email") {
		t.Error("Email should have email format error")
	}
	if !f.HasError("message") {
		t.Error("Message should have min length error")
	}

	// Fix all values
	f.Set("name", "John Doe")
	f.Set("email", "john@example.com")
	f.Set("message", "This is a long enough message for validation")

	// Clear errors and re-validate
	f.ClearErrors()
	f.Validate()

	// Should now be valid
	if !f.IsValid() {
		t.Errorf("Form should be valid after fixing values, errors: %v", f.Errors())
	}

	// Retrieve final data
	data := f.Values()
	if data.Name != "John Doe" {
		t.Errorf("Expected name 'John Doe', got '%s'", data.Name)
	}
	if data.Email != "john@example.com" {
		t.Errorf("Expected email 'john@example.com', got '%s'", data.Email)
	}
}

// TestResourceMatchWorkflow tests the resource loading and rendering workflow
// using the Match pattern for different states.
func TestResourceMatchWorkflow(t *testing.T) {
	done := make(chan struct{})

	// Create a resource that simulates API call
	users := resource.New(func() ([]string, error) {
		time.Sleep(5 * time.Millisecond) // Simulate network
		return []string{"Alice", "Bob", "Charlie"}, nil
	}).OnSuccess(func(data []string) {
		close(done)
	})

	// Helper to create text nodes
	textNode := func(s string) *vdom.VNode {
		return &vdom.VNode{Text: s}
	}

	// Match against loading state
	loadingNode := users.Match(
		resource.OnLoading[[]string](func() *vdom.VNode {
			return textNode("Loading users...")
		}),
		resource.OnReady[[]string](func(data []string) *vdom.VNode {
			return textNode("Users loaded")
		}),
	)

	// Initially should be loading
	if loadingNode == nil || loadingNode.Text != "Loading users..." {
		t.Logf("Loading node: %v", loadingNode)
		// May have completed very fast, that's ok
	}

	// Wait for completion
	select {
	case <-done:
		// Verify final state
		if !users.IsReady() {
			t.Error("Resource should be ready")
		}

		data := users.Data()
		if len(data) != 3 {
			t.Errorf("Expected 3 users, got %d", len(data))
		}

		// Match should now render ready state
		readyNode := users.Match(
			resource.OnLoading[[]string](func() *vdom.VNode {
				return textNode("Loading...")
			}),
			resource.OnReady[[]string](func(data []string) *vdom.VNode {
				return textNode("Loaded " + data[0])
			}),
		)

		if readyNode == nil || readyNode.Text != "Loaded Alice" {
			t.Errorf("Expected 'Loaded Alice', got '%v'", readyNode)
		}

	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for resource")
	}
}

// TestResourceErrorHandling tests the resource error state workflow.
func TestResourceErrorHandling(t *testing.T) {
	done := make(chan struct{})
	expectedErr := errors.New("API error: not found")

	users := resource.New(func() (string, error) {
		return "", expectedErr
	}).OnError(func(err error) {
		close(done)
	})

	select {
	case <-done:
		if !users.IsError() {
			t.Error("Resource should be in error state")
		}

		if users.Error() != expectedErr {
			t.Errorf("Expected error '%v', got '%v'", expectedErr, users.Error())
		}

		// DataOr should return fallback when in error state
		fallback := users.DataOr("default")
		if fallback != "default" {
			t.Errorf("Expected 'default', got '%s'", fallback)
		}

	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for resource error")
	}
}

// TestContextProviderWorkflow tests the context creation and consumption workflow.
func TestContextProviderWorkflow(t *testing.T) {
	// Define a theme context with default value
	type Theme struct {
		Primary   string
		Secondary string
		Dark      bool
	}

	defaultTheme := Theme{
		Primary:   "#ffffff",
		Secondary: "#000000",
		Dark:      false,
	}
	ThemeCtx := context.Create(defaultTheme)

	// Create component hierarchy with owner
	root := vango.NewOwner(nil)

	vango.WithOwner(root, func() {
		// Without a Provider, Use() should return the default value
		theme := ThemeCtx.Use()

		if theme.Primary != "#ffffff" {
			t.Errorf("Expected default primary '#ffffff', got '%s'", theme.Primary)
		}
		if theme.Dark {
			t.Error("Expected dark mode to be false (default)")
		}

		// Test that Provider().Render() sets up the context correctly
		// by simulating a render cycle with the provider
		providerTheme := Theme{
			Primary:   "#007bff",
			Secondary: "#6c757d",
			Dark:      true,
		}
		providerNode := ThemeCtx.Provider(providerTheme, vdom.Text("child"))

		// Render the provider in a child owner (simulates component rendering)
		child := vango.NewOwner(root)
		vango.WithOwner(child, func() {
			// Render the provider component
			if providerNode.Comp != nil {
				providerNode.Comp.Render()
			}

			// Now Use() should return the provided value
			childTheme := ThemeCtx.Use()
			if childTheme.Primary != "#007bff" {
				t.Errorf("Child expected primary '#007bff', got '%s'", childTheme.Primary)
			}
			if !childTheme.Dark {
				t.Error("Child expected dark mode to be true")
			}
		})
	})
}

// TestStoreSessionIsolation tests that session stores are properly isolated.
func TestStoreSessionIsolation(t *testing.T) {
	// Create shared signal definition (package-level in real usage)
	counter := store.NewSharedSignal(0)

	// Session A
	rootA := vango.NewOwner(nil)
	storeA := store.NewSessionStore()

	// Session B
	rootB := vango.NewOwner(nil)
	storeB := store.NewSessionStore()

	// Session A sets counter to 100
	vango.WithOwner(rootA, func() {
		vango.SetContext(store.SessionKey, storeA)
		counter.Set(100)

		if counter.Get() != 100 {
			t.Errorf("Session A: Expected 100, got %d", counter.Get())
		}
	})

	// Session B should have its own value (0 initially)
	vango.WithOwner(rootB, func() {
		vango.SetContext(store.SessionKey, storeB)

		// Should NOT see Session A's value
		if counter.Get() != 0 {
			t.Errorf("Session B: Expected 0, got %d", counter.Get())
		}

		// Set its own value
		counter.Set(50)
		if counter.Get() != 50 {
			t.Errorf("Session B: Expected 50, got %d", counter.Get())
		}
	})

	// Verify Session A still has its value
	vango.WithOwner(rootA, func() {
		vango.SetContext(store.SessionKey, storeA)
		if counter.Get() != 100 {
			t.Errorf("Session A (revisit): Expected 100, got %d", counter.Get())
		}
	})
}

// TestGlobalStoreSharedAcrossSessions tests that global signals are shared.
func TestGlobalStoreSharedAcrossSessions(t *testing.T) {
	// Global signal (shared across all sessions)
	globalCounter := store.NewGlobalSignal(0)

	// Set from "Session A"
	globalCounter.Set(42)

	// Should be visible from "Session B"
	if globalCounter.Get() != 42 {
		t.Errorf("Expected 42, got %d", globalCounter.Get())
	}

	// Update and verify
	globalCounter.Update(func(n int) int { return n + 8 })
	if globalCounter.Get() != 50 {
		t.Errorf("Expected 50, got %d", globalCounter.Get())
	}
}

// TestFormArrayWorkflow tests dynamic form arrays.
func TestFormArrayWorkflow(t *testing.T) {
	type OrderForm struct {
		CustomerName string   `form:"customer_name"`
		Items        []string `form:"items"`
	}

	f := form.UseForm(OrderForm{
		CustomerName: "Test Customer",
		Items:        []string{"Item 1"},
	})

	// Initially has one item
	if f.ArrayLen("items") != 1 {
		t.Errorf("Expected 1 item, got %d", f.ArrayLen("items"))
	}

	// Append items
	f.AppendTo("items", "Item 2")
	f.AppendTo("items", "Item 3")

	if f.ArrayLen("items") != 3 {
		t.Errorf("Expected 3 items, got %d", f.ArrayLen("items"))
	}

	// Remove middle item
	f.RemoveAt("items", 1)

	if f.ArrayLen("items") != 2 {
		t.Errorf("Expected 2 items after removal, got %d", f.ArrayLen("items"))
	}

	// Verify final data
	data := f.Values()
	if len(data.Items) != 2 {
		t.Errorf("Expected 2 items in data, got %d", len(data.Items))
	}
	if data.Items[0] != "Item 1" || data.Items[1] != "Item 3" {
		t.Errorf("Unexpected items: %v", data.Items)
	}
}

// TestResourceRefetchWithMutation tests the resource mutation workflow.
func TestResourceRefetchWithMutation(t *testing.T) {
	calls := 0
	done := make(chan struct{}, 2)

	counter := resource.New(func() (int, error) {
		calls++
		result := calls * 10
		// Signal completion after we know the fetch will return
		defer func() { done <- struct{}{} }()
		return result, nil
	})

	// Wait for initial fetch to signal AND for data to be set
	<-done
	// Small delay to allow the goroutine to call data.Set() after fetcher returns
	time.Sleep(10 * time.Millisecond)

	// Mutate optimistically
	counter.Mutate(func(n int) int {
		return n + 5
	})

	if counter.Data() != 15 { // 10 + 5
		t.Errorf("Expected 15, got %d", counter.Data())
	}

	// Refetch to get fresh data
	counter.Refetch()
	<-done
	// Small delay to allow data to be set
	time.Sleep(10 * time.Millisecond)

	// Should have refetched (calls=2, so 20)
	if counter.Data() != 20 {
		t.Errorf("Expected 20 after refetch, got %d", counter.Data())
	}
}

// TestContextWithFallback tests context fallback behavior.
func TestContextWithFallback(t *testing.T) {
	type Config struct {
		APIKey string
	}

	// Create context with empty default
	ConfigCtx := context.Create(Config{})

	root := vango.NewOwner(nil)

	vango.WithOwner(root, func() {
		// Without a Provider, Use() should return the default value (empty)
		config := ConfigCtx.Use()

		// Should return zero value (default)
		if config.APIKey != "" {
			t.Errorf("Expected empty APIKey, got '%s'", config.APIKey)
		}

		// Create a Provider node and render it to set up context
		providerNode := ConfigCtx.Provider(Config{APIKey: "secret123"}, vdom.Text("child"))

		// Render provider in child owner
		child := vango.NewOwner(root)
		vango.WithOwner(child, func() {
			if providerNode.Comp != nil {
				providerNode.Comp.Render()
			}

			// Now Use() should return the provided value
			config = ConfigCtx.Use()
			if config.APIKey != "secret123" {
				t.Errorf("Expected 'secret123', got '%s'", config.APIKey)
			}
		})

		// Outside the provider's scope, should still get default
		config = ConfigCtx.Use()
		if config.APIKey != "" {
			t.Errorf("Expected empty APIKey outside provider scope, got '%s'", config.APIKey)
		}
	})
}

// TestFormDirtyState tests form dirty tracking.
func TestFormDirtyState(t *testing.T) {
	type Profile struct {
		Name  string `form:"name"`
		Email string `form:"email"`
	}

	f := form.UseForm(Profile{Name: "Initial", Email: "initial@test.com"})

	// Initially not dirty
	if f.IsDirty() {
		t.Error("Form should not be dirty initially")
	}

	// Make a change
	f.Set("name", "Changed")

	// Should now be dirty
	if !f.IsDirty() {
		t.Error("Form should be dirty after change")
	}

	// Individual field dirty check
	if !f.FieldDirty("name") {
		t.Error("Name field should be dirty")
	}
	if f.FieldDirty("email") {
		t.Error("Email field should not be dirty")
	}

	// Reset clears dirty state
	f.Reset()
	if f.IsDirty() {
		t.Error("Form should not be dirty after reset")
	}
}

// TestFormReset tests form reset functionality.
func TestFormReset(t *testing.T) {
	type Login struct {
		Username string `form:"username"`
		Password string `form:"password"`
	}

	initial := Login{Username: "admin", Password: ""}
	f := form.UseForm(initial)

	// Modify
	f.Set("username", "changed")
	f.Set("password", "secret")

	if f.GetString("username") != "changed" {
		t.Error("Username should be changed")
	}

	// Reset
	f.Reset()

	// Should be back to initial
	if f.GetString("username") != "admin" {
		t.Errorf("Expected 'admin', got '%s'", f.GetString("username"))
	}
	if f.GetString("password") != "" {
		t.Errorf("Expected empty password, got '%s'", f.GetString("password"))
	}
}

// TestResourceRetryWorkflow tests automatic retry on failure.
func TestResourceRetryWorkflow(t *testing.T) {
	attempts := 0
	done := make(chan struct{})

	data := resource.New(func() (string, error) {
		attempts++
		if attempts < 3 {
			return "", errors.New("temporary failure")
		}
		return "success", nil
	}).
		RetryOnError(3, 5*time.Millisecond).
		OnSuccess(func(s string) {
			close(done)
		})

	select {
	case <-done:
		if attempts != 3 {
			t.Errorf("Expected 3 attempts, got %d", attempts)
		}
		if data.Data() != "success" {
			t.Errorf("Expected 'success', got '%s'", data.Data())
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Timeout waiting for retry success")
	}
}
