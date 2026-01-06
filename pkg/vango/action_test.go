package vango

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// mockCtx implements a minimal Ctx for testing Action.
type mockCtx struct {
	dispatchFn   func(func())
	stormBudget  StormBudgetChecker
	mu           sync.Mutex
	dispatched   []func()
	stdCtx       context.Context
}

func newMockCtx() *mockCtx {
	return &mockCtx{
		stdCtx: context.Background(),
	}
}

func (m *mockCtx) Dispatch(fn func()) {
	m.mu.Lock()
	m.dispatched = append(m.dispatched, fn)
	m.mu.Unlock()

	// Execute immediately for testing
	fn()
}

func (m *mockCtx) StdContext() context.Context {
	return m.stdCtx
}

func (m *mockCtx) StormBudget() StormBudgetChecker {
	return m.stormBudget
}

// runDispatched runs all dispatched functions.
func (m *mockCtx) runDispatched() {
	m.mu.Lock()
	fns := m.dispatched
	m.dispatched = nil
	m.mu.Unlock()

	for _, fn := range fns {
		fn()
	}
}

func TestActionStateStrings(t *testing.T) {
	tests := []struct {
		state ActionState
		want  string
	}{
		{ActionIdle, "idle"},
		{ActionRunning, "running"},
		{ActionSuccess, "success"},
		{ActionError, "error"},
		{ActionState(99), "unknown"},
	}

	for _, tt := range tests {
		got := tt.state.String()
		if got != tt.want {
			t.Errorf("ActionState(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}

func TestActionBasic(t *testing.T) {
	// Setup mock context
	mockC := newMockCtx()
	setCurrentCtx(mockC)
	defer setCurrentCtx(nil)

	// Create owner for hook slot
	owner := NewOwner(nil)
	setCurrentOwner(owner)
	defer setCurrentOwner(nil)

	// Create action
	var callCount int
	action := NewAction(func(ctx context.Context, arg string) (string, error) {
		callCount++
		return "result: " + arg, nil
	})

	// Initial state should be Idle
	if action.State() != ActionIdle {
		t.Errorf("Initial state = %v, want ActionIdle", action.State())
	}

	// Run action
	accepted := action.Run("test")
	if !accepted {
		t.Error("Run() returned false, want true")
	}

	// Wait for async completion
	time.Sleep(10 * time.Millisecond)

	// Should be in Success state
	if action.State() != ActionSuccess {
		t.Errorf("After Run(), state = %v, want ActionSuccess", action.State())
	}

	// Check result
	result, ok := action.Result()
	if !ok {
		t.Error("Result() returned false, want true")
	}
	if result != "result: test" {
		t.Errorf("Result() = %q, want %q", result, "result: test")
	}

	// Should have no error
	if action.Error() != nil {
		t.Errorf("Error() = %v, want nil", action.Error())
	}
}

func TestActionError(t *testing.T) {
	mockC := newMockCtx()
	setCurrentCtx(mockC)
	defer setCurrentCtx(nil)

	owner := NewOwner(nil)
	setCurrentOwner(owner)
	defer setCurrentOwner(nil)

	expectedErr := errors.New("test error")
	action := NewAction(func(ctx context.Context, arg int) (int, error) {
		return 0, expectedErr
	})

	action.Run(42)
	time.Sleep(10 * time.Millisecond)

	if action.State() != ActionError {
		t.Errorf("State = %v, want ActionError", action.State())
	}

	if action.Error() != expectedErr {
		t.Errorf("Error() = %v, want %v", action.Error(), expectedErr)
	}
}

func TestActionReset(t *testing.T) {
	mockC := newMockCtx()
	setCurrentCtx(mockC)
	defer setCurrentCtx(nil)

	owner := NewOwner(nil)
	setCurrentOwner(owner)
	defer setCurrentOwner(nil)

	action := NewAction(func(ctx context.Context, arg int) (int, error) {
		return arg * 2, nil
	})

	action.Run(5)
	time.Sleep(10 * time.Millisecond)

	if action.State() != ActionSuccess {
		t.Errorf("State = %v, want ActionSuccess", action.State())
	}

	// Reset action
	action.Reset()

	if action.State() != ActionIdle {
		t.Errorf("After Reset(), state = %v, want ActionIdle", action.State())
	}

	if _, ok := action.Result(); ok {
		t.Error("After Reset(), Result() should return false")
	}
}

func TestActionDropWhileRunning(t *testing.T) {
	mockC := newMockCtx()
	setCurrentCtx(mockC)
	defer setCurrentCtx(nil)

	owner := NewOwner(nil)
	setCurrentOwner(owner)
	defer setCurrentOwner(nil)

	started := make(chan struct{})
	finish := make(chan struct{})

	action := NewAction(func(ctx context.Context, arg int) (int, error) {
		close(started)
		<-finish
		return arg, nil
	}, DropWhileRunning())

	// Start first run
	accepted1 := action.Run(1)
	if !accepted1 {
		t.Error("First Run() should be accepted")
	}

	<-started // Wait for work to start

	// Try to run again while running
	accepted2 := action.Run(2)
	if accepted2 {
		t.Error("Second Run() with DropWhileRunning should be rejected")
	}

	// Complete the work
	close(finish)
	time.Sleep(10 * time.Millisecond)
}

func TestActionHelpers(t *testing.T) {
	mockC := newMockCtx()
	setCurrentCtx(mockC)
	defer setCurrentCtx(nil)

	owner := NewOwner(nil)
	setCurrentOwner(owner)
	defer setCurrentOwner(nil)

	action := NewAction(func(ctx context.Context, arg int) (int, error) {
		return arg, nil
	})

	if !action.IsIdle() {
		t.Error("IsIdle() should return true initially")
	}

	action.Run(1)

	// Running is async, but we can check state after a brief wait
	time.Sleep(5 * time.Millisecond)

	if !action.IsSuccess() {
		t.Error("IsSuccess() should return true after completion")
	}

	if action.IsError() {
		t.Error("IsError() should return false on success")
	}
}

// mockBudgetExceeded is a mock budget checker that always returns ErrBudgetExceeded.
type mockBudgetExceeded struct{}

func (m *mockBudgetExceeded) CheckResource() error  { return ErrBudgetExceeded }
func (m *mockBudgetExceeded) CheckAction() error    { return ErrBudgetExceeded }
func (m *mockBudgetExceeded) CheckGoLatest() error  { return ErrBudgetExceeded }
func (m *mockBudgetExceeded) CheckEffectRun() error { return ErrBudgetExceeded }
func (m *mockBudgetExceeded) ResetTick()            {}

// mockBudgetOK is a mock budget checker that always allows operations.
type mockBudgetOK struct{}

func (m *mockBudgetOK) CheckResource() error  { return nil }
func (m *mockBudgetOK) CheckAction() error    { return nil }
func (m *mockBudgetOK) CheckGoLatest() error  { return nil }
func (m *mockBudgetOK) CheckEffectRun() error { return nil }
func (m *mockBudgetOK) ResetTick()            {}

func TestActionBudgetExceeded(t *testing.T) {
	mockC := newMockCtx()
	mockC.stormBudget = &mockBudgetExceeded{}
	setCurrentCtx(mockC)
	defer setCurrentCtx(nil)

	owner := NewOwner(nil)
	setCurrentOwner(owner)
	defer setCurrentOwner(nil)

	var callCount int
	action := NewAction(func(ctx context.Context, arg int) (int, error) {
		callCount++
		return arg * 2, nil
	})

	// Run action - should be rejected due to budget
	accepted := action.Run(5)
	if accepted {
		t.Error("Run() should return false when budget exceeded")
	}

	// Wait briefly to ensure async operations complete
	time.Sleep(10 * time.Millisecond)

	// State should be Error
	if action.State() != ActionError {
		t.Errorf("State = %v, want ActionError", action.State())
	}

	// Error should be ErrBudgetExceeded
	if action.Error() != ErrBudgetExceeded {
		t.Errorf("Error() = %v, want ErrBudgetExceeded", action.Error())
	}

	// Work function should not have been called
	if callCount != 0 {
		t.Errorf("Work function called %d times, want 0", callCount)
	}
}

func TestActionBudgetOK(t *testing.T) {
	mockC := newMockCtx()
	mockC.stormBudget = &mockBudgetOK{}
	setCurrentCtx(mockC)
	defer setCurrentCtx(nil)

	owner := NewOwner(nil)
	setCurrentOwner(owner)
	defer setCurrentOwner(nil)

	var callCount int
	action := NewAction(func(ctx context.Context, arg int) (int, error) {
		callCount++
		return arg * 2, nil
	})

	// Run action - should be accepted
	accepted := action.Run(5)
	if !accepted {
		t.Error("Run() should return true when budget allows")
	}

	// Wait for async completion
	time.Sleep(10 * time.Millisecond)

	// State should be Success
	if action.State() != ActionSuccess {
		t.Errorf("State = %v, want ActionSuccess", action.State())
	}

	// Work function should have been called
	if callCount != 1 {
		t.Errorf("Work function called %d times, want 1", callCount)
	}

	// Check result
	result, ok := action.Result()
	if !ok {
		t.Error("Result() returned false, want true")
	}
	if result != 10 {
		t.Errorf("Result() = %d, want 10", result)
	}
}
