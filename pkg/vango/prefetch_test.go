package vango

import (
	"context"
	"testing"
)

// =============================================================================
// Prefetch Mode Detection Tests
// =============================================================================

// mockPrefetchChecker implements PrefetchModeChecker for testing.
type mockPrefetchChecker struct {
	mode int
}

func (m *mockPrefetchChecker) Mode() int {
	return m.mode
}

func TestIsPrefetchModeNoContext(t *testing.T) {
	// Without any context, should return false
	if IsPrefetchMode() {
		t.Error("Expected false when no context")
	}
}

func TestIsPrefetchModeNormal(t *testing.T) {
	// Set up a mock context in normal mode
	mock := &mockPrefetchChecker{mode: 0}
	setCurrentCtx(mock)
	defer setCurrentCtx(nil)

	if IsPrefetchMode() {
		t.Error("Expected false in normal mode")
	}
}

func TestIsPrefetchModePrefetch(t *testing.T) {
	// Set up a mock context in prefetch mode
	mock := &mockPrefetchChecker{mode: 1}
	setCurrentCtx(mock)
	defer setCurrentCtx(nil)

	if !IsPrefetchMode() {
		t.Error("Expected true in prefetch mode")
	}
}

// =============================================================================
// checkPrefetchWrite Tests
// =============================================================================

func TestCheckPrefetchWriteNormalMode(t *testing.T) {
	// Normal mode - writes should be allowed
	mock := &mockPrefetchChecker{mode: 0}
	setCurrentCtx(mock)
	defer setCurrentCtx(nil)

	if !checkPrefetchWrite("Signal.Set") {
		t.Error("Expected write allowed in normal mode")
	}
}

func TestCheckPrefetchWritePrefetchModeProd(t *testing.T) {
	// Prefetch mode, production - writes should be silently dropped
	oldDevMode := DevMode
	DevMode = false
	defer func() { DevMode = oldDevMode }()

	mock := &mockPrefetchChecker{mode: 1}
	setCurrentCtx(mock)
	defer setCurrentCtx(nil)

	if checkPrefetchWrite("Signal.Set") {
		t.Error("Expected write dropped in prefetch mode (prod)")
	}
}

func TestCheckPrefetchWritePrefetchModeDev(t *testing.T) {
	// Prefetch mode, dev - should panic
	oldDevMode := DevMode
	DevMode = true
	defer func() { DevMode = oldDevMode }()

	mock := &mockPrefetchChecker{mode: 1}
	setCurrentCtx(mock)
	defer setCurrentCtx(nil)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in dev mode during prefetch")
		}
	}()

	checkPrefetchWrite("Signal.Set")
}

// =============================================================================
// checkPrefetchSideEffect Tests
// =============================================================================

func TestCheckPrefetchSideEffectNormalMode(t *testing.T) {
	// Normal mode - side effects should be allowed
	mock := &mockPrefetchChecker{mode: 0}
	setCurrentCtx(mock)
	defer setCurrentCtx(nil)

	if !checkPrefetchSideEffect("Effect") {
		t.Error("Expected side effect allowed in normal mode")
	}
}

func TestCheckPrefetchSideEffectPrefetchModeProd(t *testing.T) {
	// Prefetch mode, production - side effects should be no-op
	oldDevMode := DevMode
	DevMode = false
	defer func() { DevMode = oldDevMode }()

	mock := &mockPrefetchChecker{mode: 1}
	setCurrentCtx(mock)
	defer setCurrentCtx(nil)

	if checkPrefetchSideEffect("Effect") {
		t.Error("Expected side effect blocked in prefetch mode (prod)")
	}
}

func TestCheckPrefetchSideEffectPrefetchModeDev(t *testing.T) {
	// Prefetch mode, dev - should panic
	oldDevMode := DevMode
	DevMode = true
	defer func() { DevMode = oldDevMode }()

	mock := &mockPrefetchChecker{mode: 1}
	setCurrentCtx(mock)
	defer setCurrentCtx(nil)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in dev mode during prefetch")
		}
	}()

	checkPrefetchSideEffect("Effect")
}

// =============================================================================
// Signal Write Enforcement Tests
// =============================================================================

func TestSignalSetBlockedInPrefetch(t *testing.T) {
	// Set up prefetch mode (production)
	oldDevMode := DevMode
	DevMode = false
	defer func() { DevMode = oldDevMode }()

	mock := &mockPrefetchChecker{mode: 1}
	setCurrentCtx(mock)
	defer setCurrentCtx(nil)

	// Create signal
	sig := NewSignal(42)
	original := sig.Peek()

	// Try to set - should be dropped
	sig.Set(100)

	// Value should be unchanged
	if sig.Peek() != original {
		t.Errorf("Expected value unchanged (%d), got %d", original, sig.Peek())
	}
}

func TestSignalUpdateBlockedInPrefetch(t *testing.T) {
	// Set up prefetch mode (production)
	oldDevMode := DevMode
	DevMode = false
	defer func() { DevMode = oldDevMode }()

	mock := &mockPrefetchChecker{mode: 1}
	setCurrentCtx(mock)
	defer setCurrentCtx(nil)

	// Create signal
	sig := NewSignal(42)
	original := sig.Peek()

	// Try to update - should be dropped
	sig.Update(func(v int) int { return v + 1 })

	// Value should be unchanged
	if sig.Peek() != original {
		t.Errorf("Expected value unchanged (%d), got %d", original, sig.Peek())
	}
}

func TestSignalIncBlockedInPrefetch(t *testing.T) {
	// Set up prefetch mode (production)
	oldDevMode := DevMode
	DevMode = false
	defer func() { DevMode = oldDevMode }()

	mock := &mockPrefetchChecker{mode: 1}
	setCurrentCtx(mock)
	defer setCurrentCtx(nil)

	// Create signal
	sig := NewSignal(42)
	original := sig.Peek()

	// Try to inc - should be dropped
	sig.Inc()

	// Value should be unchanged
	if sig.Peek() != original {
		t.Errorf("Expected value unchanged (%d), got %d", original, sig.Peek())
	}
}

func TestSignalToggleBlockedInPrefetch(t *testing.T) {
	// Set up prefetch mode (production)
	oldDevMode := DevMode
	DevMode = false
	defer func() { DevMode = oldDevMode }()

	mock := &mockPrefetchChecker{mode: 1}
	setCurrentCtx(mock)
	defer setCurrentCtx(nil)

	// Create signal
	sig := NewSignal(true)
	original := sig.Peek()

	// Try to toggle - should be dropped
	sig.Toggle()

	// Value should be unchanged
	if sig.Peek() != original {
		t.Errorf("Expected value unchanged (%v), got %v", original, sig.Peek())
	}
}

func TestSignalAppendBlockedInPrefetch(t *testing.T) {
	// Set up prefetch mode (production)
	oldDevMode := DevMode
	DevMode = false
	defer func() { DevMode = oldDevMode }()

	mock := &mockPrefetchChecker{mode: 1}
	setCurrentCtx(mock)
	defer setCurrentCtx(nil)

	// Create signal
	sig := NewSignal("hello")
	original := sig.Peek()

	// Try to append - should be dropped
	sig.Append(" world")

	// Value should be unchanged
	if sig.Peek() != original {
		t.Errorf("Expected value unchanged (%q), got %q", original, sig.Peek())
	}
}

// =============================================================================
// Signal Read Allowed Tests (reads should work in prefetch)
// =============================================================================

func TestSignalGetAllowedInPrefetch(t *testing.T) {
	// Set up prefetch mode (production)
	oldDevMode := DevMode
	DevMode = false
	defer func() { DevMode = oldDevMode }()

	mock := &mockPrefetchChecker{mode: 1}
	setCurrentCtx(mock)
	defer setCurrentCtx(nil)

	// Create signal and read - should work
	sig := NewSignal(42)
	value := sig.Get()

	if value != 42 {
		t.Errorf("Expected Get to return 42, got %d", value)
	}
}

func TestSignalPeekAllowedInPrefetch(t *testing.T) {
	// Set up prefetch mode (production)
	oldDevMode := DevMode
	DevMode = false
	defer func() { DevMode = oldDevMode }()

	mock := &mockPrefetchChecker{mode: 1}
	setCurrentCtx(mock)
	defer setCurrentCtx(nil)

	// Create signal and peek - should work
	sig := NewSignal(42)
	value := sig.Peek()

	if value != 42 {
		t.Errorf("Expected Peek to return 42, got %d", value)
	}
}

// =============================================================================
// Subscribe/GoLatest Enforcement Tests (Section 8.3.2)
// =============================================================================

func TestSubscribeBlockedInPrefetch(t *testing.T) {
	// Set up prefetch mode (production)
	oldDevMode := DevMode
	DevMode = false
	defer func() { DevMode = oldDevMode }()

	mock := &mockPrefetchChecker{mode: 1}
	setCurrentCtx(mock)
	defer setCurrentCtx(nil)

	// Subscribe should return a no-op cleanup without panicking
	cleanup := Subscribe[int](nil, func(int) {})
	if cleanup == nil {
		t.Error("Expected non-nil cleanup function")
	}
	// Calling cleanup should not panic
	cleanup()
}

func TestSubscribePanicsInDevPrefetch(t *testing.T) {
	// Set up prefetch mode (dev)
	oldDevMode := DevMode
	DevMode = true
	defer func() { DevMode = oldDevMode }()

	mock := &mockPrefetchChecker{mode: 1}
	setCurrentCtx(mock)
	defer setCurrentCtx(nil)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in dev mode during prefetch")
		}
	}()

	Subscribe[int](nil, func(int) {})
}

func TestGoLatestBlockedInPrefetch(t *testing.T) {
	// Set up prefetch mode (production)
	oldDevMode := DevMode
	DevMode = false
	defer func() { DevMode = oldDevMode }()

	mock := &mockPrefetchChecker{mode: 1}
	setCurrentCtx(mock)
	defer setCurrentCtx(nil)

	// GoLatest should return a no-op cleanup without panicking
	cleanup := GoLatest[string, int]("key", func(_ context.Context, _ string) (int, error) {
		return 0, nil
	}, func(_ int, _ error) {})
	if cleanup == nil {
		t.Error("Expected non-nil cleanup function")
	}
	// Calling cleanup should not panic
	cleanup()
}

func TestGoLatestPanicsInDevPrefetch(t *testing.T) {
	// Set up prefetch mode (dev)
	oldDevMode := DevMode
	DevMode = true
	defer func() { DevMode = oldDevMode }()

	mock := &mockPrefetchChecker{mode: 1}
	setCurrentCtx(mock)
	defer setCurrentCtx(nil)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic in dev mode during prefetch")
		}
	}()

	GoLatest[string, int]("key", func(_ context.Context, _ string) (int, error) {
		return 0, nil
	}, func(_ int, _ error) {})
}
