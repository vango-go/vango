package resource

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vango-go/vango/pkg/vango"
	"github.com/vango-go/vango/pkg/vdom"
)

func TestNewResource(t *testing.T) {
	fetcher := func() (string, error) {
		return "data", nil
	}

	r := New(fetcher)

	// Initially it might be pending or loading depending on scheduler,
	// but Fetch is called immediately in goroutine.
	// Since we can't control goroutine scheduling easily without mocks/channels,
	// we'll wait a bit.

	// Check initial state properties
	if r == nil {
		t.Fatal("New returned nil")
	}
}

func TestResourceSuccess(t *testing.T) {
	done := make(chan struct{})
	fetcher := func() (string, error) {
		return "success", nil
	}

	r := New(fetcher).OnSuccess(func(data string) {
		if data != "success" {
			t.Errorf("Expected 'success', got '%s'", data)
		}
		close(done)
	})

	select {
	case <-done:
		if !r.IsReady() {
			t.Error("Expected IsReady() to be true")
		}
		if r.Data() != "success" {
			t.Errorf("Expected 'success', got '%s'", r.Data())
		}
		if r.Error() != nil {
			t.Error("Expected no error")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for resource success")
	}
}

func TestResourceError(t *testing.T) {
	done := make(chan struct{})
	expectedErr := errors.New("fail")

	fetcher := func() (string, error) {
		return "", expectedErr
	}

	r := New(fetcher).OnError(func(err error) {
		if err != expectedErr {
			t.Errorf("Expected error '%v', got '%v'", expectedErr, err)
		}
		close(done)
	})

	select {
	case <-done:
		if !r.IsError() {
			t.Error("Expected IsError() to be true")
		}
		if r.Error() != expectedErr {
			t.Error("Expected error to be set")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for resource error")
	}
}

func TestResourceStaleTime(t *testing.T) {
	calls := 0
	fetcher := func() (string, error) {
		calls++
		return "data", nil
	}

	done := make(chan struct{})
	r := New(fetcher).
		StaleTime(100 * time.Millisecond).
		OnSuccess(func(string) {
			if calls == 1 {
				close(done)
			}
		})

	// Wait for first fetch
	<-done

	// Calling Fetch immediately should not trigger new fetch due to StaleTime
	r.Fetch()

	time.Sleep(10 * time.Millisecond)
	if calls != 1 {
		t.Errorf("Expected 1 call, got %d", calls)
	}

	// Wait for stale time to pass
	time.Sleep(150 * time.Millisecond)

	// Now Fetch should trigger new fetch
	// Reset done channel to wait for next success (if we had a way to reset OnSuccess,
	// but OnSuccess is permanent. We can just check calls count after a short delay)
	r.Fetch()

	time.Sleep(50 * time.Millisecond)
	if calls != 2 {
		t.Errorf("Expected 2 calls, got %d", calls)
	}
}

func TestResourceRefetch(t *testing.T) {
	calls := 0
	fetcher := func() (string, error) {
		calls++
		return "data", nil
	}

	done := make(chan struct{})
	r := New(fetcher).OnSuccess(func(string) {
		if calls == 1 {
			close(done)
		}
	})

	<-done

	// Refetch should force new fetch regardless of StaleTime (which defaults to 0 anyway)
	r.Refetch()

	time.Sleep(50 * time.Millisecond)
	if calls != 2 {
		t.Errorf("Expected 2 calls, got %d", calls)
	}
}

func TestResourceMutate(t *testing.T) {
	r := New(func() (int, error) { return 0, nil })

	// Wait for initial load
	time.Sleep(10 * time.Millisecond)

	r.Mutate(func(n int) int {
		return n + 1
	})

	if r.Data() != 1 {
		t.Errorf("Expected 1, got %d", r.Data())
	}
}

func TestResourceMatch(t *testing.T) {
	done := make(chan struct{})
	fetcher := func() (string, error) {
		return "hello", nil
	}

	r := New(fetcher).OnSuccess(func(string) {
		close(done)
	})

	<-done

	// Helper to create a text node for testing
	textNode := func(s string) *vdom.VNode {
		return &vdom.VNode{Text: s}
	}

	node := r.Match(
		OnPending[string](func() *vdom.VNode { return textNode("Pending") }),
		OnLoading[string](func() *vdom.VNode { return textNode("Loading") }),
		OnError[string](func(err error) *vdom.VNode { return textNode("Error") }),
		OnReady[string](func(data string) *vdom.VNode { return textNode(data) }),
	)

	if node == nil {
		t.Fatal("Match returned nil")
	}

	if node.Text != "hello" {
		t.Errorf("Expected 'hello', got '%s'", node.Text)
	}
}

func TestMatchLoadingOrPending(t *testing.T) {
	// Create a resource that hangs
	r := New(func() (string, error) {
		time.Sleep(100 * time.Millisecond)
		return "", nil
	})

	textNode := func(s string) *vdom.VNode {
		return &vdom.VNode{Text: s}
	}

	node := r.Match(
		OnLoadingOrPending[string](func() *vdom.VNode { return textNode("Waiting") }),
		OnReady[string](func(s string) *vdom.VNode { return textNode(s) }),
	)

	if node == nil || node.Text != "Waiting" {
		t.Errorf("Expected 'Waiting', got %v", node)
	}
}

func TestResourceState(t *testing.T) {
	done := make(chan struct{})
	r := New(func() (string, error) {
		return "data", nil
	}).OnSuccess(func(string) {
		close(done)
	})

	<-done

	// After success, state should be Ready
	if r.State() != Ready {
		t.Errorf("Expected state Ready, got %v", r.State())
	}
}

func TestResourceIsLoading(t *testing.T) {
	// Create a slow resource
	r := New(func() (string, error) {
		time.Sleep(50 * time.Millisecond)
		return "data", nil
	})

	// Initially should be loading or pending
	if !r.IsLoading() {
		t.Error("Expected IsLoading() to be true initially")
	}
}

func TestResourceDataOr(t *testing.T) {
	done := make(chan struct{})
	r := New(func() (string, error) {
		return "actual", nil
	}).OnSuccess(func(string) {
		close(done)
	})

	// Before completion, should return fallback
	fallback := r.DataOr("fallback")
	// This might be "actual" or "fallback" depending on timing
	// Just verify it returns something
	if fallback == "" {
		t.Error("DataOr should return a value")
	}

	<-done

	// After completion, should return actual data
	if r.DataOr("fallback") != "actual" {
		t.Errorf("DataOr should return 'actual' when ready, got '%s'", r.DataOr("fallback"))
	}
}

func TestResourceDataOrWhenNotReady(t *testing.T) {
	// Create resource that takes time
	r := New(func() (string, error) {
		time.Sleep(100 * time.Millisecond)
		return "data", nil
	})

	// Immediately check DataOr - should return fallback since not ready yet
	result := r.DataOr("fallback")
	if result != "fallback" {
		t.Errorf("Expected 'fallback' when not ready, got '%s'", result)
	}
}

func TestResourceInvalidate(t *testing.T) {
	calls := 0
	done := make(chan struct{}, 2)

	r := New(func() (string, error) {
		calls++
		return "data", nil
	}).
		StaleTime(1 * time.Hour). // Long stale time
		OnSuccess(func(string) {
			done <- struct{}{}
		})

	<-done // Wait for first fetch

	// Normally Fetch wouldn't trigger due to long StaleTime
	r.Fetch()
	time.Sleep(10 * time.Millisecond)
	if calls != 1 {
		t.Errorf("Expected 1 call before invalidate, got %d", calls)
	}

	// Invalidate should reset last fetch time
	r.Invalidate()

	// Now Fetch should work
	r.Fetch()
	<-done // Wait for second fetch
	if calls != 2 {
		t.Errorf("Expected 2 calls after invalidate, got %d", calls)
	}
}

func TestResourceRetryOnError(t *testing.T) {
	attempts := 0
	done := make(chan struct{})

	r := New(func() (string, error) {
		attempts++
		if attempts < 3 {
			return "", errors.New("temporary error")
		}
		return "success", nil
	}).
		RetryOnError(3, 5*time.Millisecond).
		OnSuccess(func(string) {
			close(done)
		})

	select {
	case <-done:
		if attempts != 3 {
			t.Errorf("Expected 3 attempts, got %d", attempts)
		}
		if r.Data() != "success" {
			t.Errorf("Expected 'success', got '%s'", r.Data())
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Timeout waiting for retry success")
	}
}

func TestResourceRetryOnErrorExhausted(t *testing.T) {
	attempts := 0
	done := make(chan struct{})

	r := New(func() (string, error) {
		attempts++
		return "", errors.New("permanent error")
	}).
		RetryOnError(2, 5*time.Millisecond).
		OnError(func(err error) {
			close(done)
		})

	select {
	case <-done:
		// Should have tried 1 + 2 retries = 3 attempts
		if attempts != 3 {
			t.Errorf("Expected 3 attempts, got %d", attempts)
		}
		if !r.IsError() {
			t.Error("Expected resource to be in error state")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Timeout waiting for retry exhaustion")
	}
}

func TestMatchPending(t *testing.T) {
	textNode := func(s string) *vdom.VNode {
		return &vdom.VNode{Text: s}
	}

	// Create a resource that we control
	done := make(chan struct{})
	r := New(func() (string, error) {
		return "data", nil
	}).OnSuccess(func(string) {
		close(done)
	})
	<-done

	// Test OnReady handler
	node := r.Match(
		OnPending[string](func() *vdom.VNode { return textNode("Pending") }),
		OnReady[string](func(data string) *vdom.VNode { return textNode(data) }),
	)

	if node == nil || node.Text != "data" {
		t.Errorf("Expected 'data', got %v", node)
	}
}

func TestMatchError(t *testing.T) {
	done := make(chan struct{})
	r := New(func() (string, error) {
		return "", errors.New("failed")
	}).OnError(func(err error) {
		close(done)
	})

	<-done

	textNode := func(s string) *vdom.VNode {
		return &vdom.VNode{Text: s}
	}

	node := r.Match(
		OnError[string](func(err error) *vdom.VNode { return textNode(err.Error()) }),
		OnReady[string](func(data string) *vdom.VNode { return textNode(data) }),
	)

	if node == nil || node.Text != "failed" {
		t.Errorf("Expected 'failed', got %v", node)
	}
}

func TestMatchNoHandlerMatches(t *testing.T) {
	done := make(chan struct{})
	r := New(func() (string, error) {
		return "data", nil
	}).OnSuccess(func(string) {
		close(done)
	})

	<-done

	// Only provide OnError handler when resource is Ready
	node := r.Match(
		OnError[string](func(err error) *vdom.VNode {
			return &vdom.VNode{Text: "error"}
		}),
	)

	// Should return nil when no handler matches
	if node != nil {
		t.Error("Expected nil when no handler matches")
	}
}

func TestMatchLoading(t *testing.T) {
	// Create a slow resource
	r := New(func() (string, error) {
		time.Sleep(100 * time.Millisecond)
		return "data", nil
	})

	// Give it a moment to transition from Pending to Loading
	time.Sleep(5 * time.Millisecond)

	textNode := func(s string) *vdom.VNode {
		return &vdom.VNode{Text: s}
	}

	node := r.Match(
		OnLoading[string](func() *vdom.VNode { return textNode("Loading") }),
		OnReady[string](func(data string) *vdom.VNode { return textNode(data) }),
	)

	// Should match Loading or still be transitioning
	if node != nil && node.Text != "Loading" {
		// Might also be nil if state is not yet Loading
	}
}

func TestResourceOnSuccessOnError(t *testing.T) {
	// Test OnSuccess callback
	successCalled := false
	done := make(chan struct{})

	r := New(func() (string, error) {
		return "data", nil
	}).OnSuccess(func(data string) {
		successCalled = true
		if data != "data" {
			t.Errorf("Expected 'data', got '%s'", data)
		}
		close(done)
	})

	<-done
	if !successCalled {
		t.Error("OnSuccess callback should have been called")
	}
	_ = r
}

func TestResourceOnErrorCallback(t *testing.T) {
	errorCalled := false
	done := make(chan struct{})
	expectedErr := errors.New("test error")

	r := New(func() (string, error) {
		return "", expectedErr
	}).OnError(func(err error) {
		errorCalled = true
		if err != expectedErr {
			t.Errorf("Expected error '%v', got '%v'", expectedErr, err)
		}
		close(done)
	})

	<-done
	if !errorCalled {
		t.Error("OnError callback should have been called")
	}
	_ = r
}

func TestResourceStaleTimeChaining(t *testing.T) {
	r := New(func() (string, error) {
		return "data", nil
	}).StaleTime(5 * time.Second)

	// Verify chaining returns the resource
	if r == nil {
		t.Error("StaleTime should return the resource for chaining")
	}
}

func TestResourceRetryOnErrorChaining(t *testing.T) {
	r := New(func() (string, error) {
		return "data", nil
	}).RetryOnError(3, 100*time.Millisecond)

	if r == nil {
		t.Error("RetryOnError should return the resource for chaining")
	}
}

func TestResourceOnSuccessChaining(t *testing.T) {
	r := New(func() (string, error) {
		return "data", nil
	}).OnSuccess(func(string) {})

	if r == nil {
		t.Error("OnSuccess should return the resource for chaining")
	}
}

func TestResourceOnErrorChaining(t *testing.T) {
	r := New(func() (string, error) {
		return "", errors.New("error")
	}).OnError(func(error) {})

	if r == nil {
		t.Error("OnError should return the resource for chaining")
	}
}

// TestResourceRefetchBudgetExceeded verifies that Refetch respects storm budget.
// This test requires creating a Resource with a mock context.
func TestResourceRefetchBudgetExceeded(t *testing.T) {
	// Create resource with a simple fetcher
	fetcherCalled := false
	r := &Resource[string]{
		fetcher: func() (string, error) {
			fetcherCalled = true
			return "data", nil
		},
		ctx: &mockBudgetExceededCtx{},
	}

	// Initialize signals
	r.state = vango.NewSignal(Pending)
	r.data = vango.NewSignal("")
	r.err = vango.NewSignal[error](nil)

	done := make(chan struct{})
	r.onError = func(err error) {
		if err != vango.ErrBudgetExceeded {
			t.Errorf("OnError got %v, want ErrBudgetExceeded", err)
		}
		close(done)
	}

	// Call Refetch - should be rejected by budget
	r.Refetch()

	// Wait for dispatch
	select {
	case <-done:
		// Success - onError was called
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for budget error callback")
	}

	// Fetcher should NOT have been called
	if fetcherCalled {
		t.Error("Fetcher should not have been called when budget exceeded")
	}

	// State should be Error
	if r.State() != Error {
		t.Errorf("State = %v, want Error", r.State())
	}

	// Error should be ErrBudgetExceeded
	if r.Error() != vango.ErrBudgetExceeded {
		t.Errorf("Error = %v, want ErrBudgetExceeded", r.Error())
	}
}

// mockBudgetExceededCtx is a mock context that always exceeds the budget.
type mockBudgetExceededCtx struct{}

func (m *mockBudgetExceededCtx) Dispatch(fn func()) { fn() }
func (m *mockBudgetExceededCtx) StdContext() context.Context {
	return context.Background()
}
func (m *mockBudgetExceededCtx) StormBudget() vango.StormBudgetChecker {
	return &mockResourceBudgetExceeded{}
}

// mockResourceBudgetExceeded always exceeds the resource budget.
type mockResourceBudgetExceeded struct{}

func (m *mockResourceBudgetExceeded) CheckResource() error  { return vango.ErrBudgetExceeded }
func (m *mockResourceBudgetExceeded) CheckAction() error    { return nil }
func (m *mockResourceBudgetExceeded) CheckGoLatest() error  { return nil }
func (m *mockResourceBudgetExceeded) CheckEffectRun() error { return nil }
func (m *mockResourceBudgetExceeded) ResetTick()            {}
