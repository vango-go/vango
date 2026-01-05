package vango

import (
	"sync"
	"testing"
)

func TestOwnerBasic(t *testing.T) {
	owner := NewOwner(nil)

	if owner.ID() == 0 {
		t.Error("owner should have non-zero ID")
	}

	if owner.Parent() != nil {
		t.Error("root owner should have nil parent")
	}

	if owner.IsDisposed() {
		t.Error("new owner should not be disposed")
	}
}

func TestOwnerHierarchy(t *testing.T) {
	root := NewOwner(nil)
	child1 := NewOwner(root)
	child2 := NewOwner(root)
	grandchild := NewOwner(child1)

	if child1.Parent() != root {
		t.Error("child1 parent should be root")
	}

	if child2.Parent() != root {
		t.Error("child2 parent should be root")
	}

	if grandchild.Parent() != child1 {
		t.Error("grandchild parent should be child1")
	}
}

func TestOwnerDispose(t *testing.T) {
	owner := NewOwner(nil)
	owner.Dispose()

	if !owner.IsDisposed() {
		t.Error("owner should be disposed after Dispose()")
	}
}

func TestOwnerDisposeHierarchy(t *testing.T) {
	root := NewOwner(nil)
	child1 := NewOwner(root)
	child2 := NewOwner(root)
	grandchild := NewOwner(child1)

	// Track disposal order
	disposalOrder := []string{}
	var mu sync.Mutex

	addDisposal := func(name string) func() {
		return func() {
			mu.Lock()
			disposalOrder = append(disposalOrder, name)
			mu.Unlock()
		}
	}

	grandchild.OnCleanup(addDisposal("grandchild"))
	child1.OnCleanup(addDisposal("child1"))
	child2.OnCleanup(addDisposal("child2"))
	root.OnCleanup(addDisposal("root"))

	// Dispose root
	root.Dispose()

	// All should be disposed
	if !root.IsDisposed() {
		t.Error("root should be disposed")
	}
	if !child1.IsDisposed() {
		t.Error("child1 should be disposed")
	}
	if !child2.IsDisposed() {
		t.Error("child2 should be disposed")
	}
	if !grandchild.IsDisposed() {
		t.Error("grandchild should be disposed")
	}

	// Children should dispose before parents (reverse order)
	// grandchild before child1, child1 & child2 before root
	if len(disposalOrder) != 4 {
		t.Errorf("expected 4 disposals, got %d", len(disposalOrder))
	}

	// Root should be last
	if disposalOrder[len(disposalOrder)-1] != "root" {
		t.Errorf("root should be last disposed, order: %v", disposalOrder)
	}
}

func TestOwnerOnCleanup(t *testing.T) {
	owner := NewOwner(nil)

	cleanupRan := false
	owner.OnCleanup(func() {
		cleanupRan = true
	})

	if cleanupRan {
		t.Error("cleanup should not run before dispose")
	}

	owner.Dispose()

	if !cleanupRan {
		t.Error("cleanup should run on dispose")
	}
}

func TestOwnerOnCleanupMultiple(t *testing.T) {
	owner := NewOwner(nil)

	order := []int{}
	owner.OnCleanup(func() { order = append(order, 1) })
	owner.OnCleanup(func() { order = append(order, 2) })
	owner.OnCleanup(func() { order = append(order, 3) })

	owner.Dispose()

	// Should run in reverse order
	if len(order) != 3 {
		t.Errorf("expected 3 cleanups, got %d", len(order))
	}
	if order[0] != 3 || order[1] != 2 || order[2] != 1 {
		t.Errorf("expected reverse order [3,2,1], got %v", order)
	}
}

func TestOwnerOnCleanupAfterDispose(t *testing.T) {
	owner := NewOwner(nil)
	owner.Dispose()

	cleanupRan := false
	owner.OnCleanup(func() {
		cleanupRan = true
	})

	// Cleanup should run immediately when registered on disposed owner
	if !cleanupRan {
		t.Error("cleanup should run immediately on disposed owner")
	}
}

func TestOwnerDoubleDispose(t *testing.T) {
	owner := NewOwner(nil)

	cleanupCount := 0
	owner.OnCleanup(func() {
		cleanupCount++
	})

	owner.Dispose()
	owner.Dispose() // Should be no-op

	if cleanupCount != 1 {
		t.Errorf("cleanup should only run once, got %d", cleanupCount)
	}
}

func TestOwnerDisposeRemovesFromParent(t *testing.T) {
	root := NewOwner(nil)
	child := NewOwner(root)

	// Dispose child
	child.Dispose()

	// Root should still work
	root.OnCleanup(func() {})
	root.Dispose()

	// No panic should occur
}

func TestOwnerScheduleEffect(t *testing.T) {
	owner := NewOwner(nil)

	// Create a minimal effect for testing
	effect := &Effect{
		id:    nextID(),
		owner: owner,
	}
	effect.pending.Store(true)

	owner.scheduleEffect(effect)

	// Effect should be pending
	if !effect.pending.Load() {
		t.Error("effect should still be pending")
	}
}

func TestOwnerRunPendingEffects(t *testing.T) {
	owner := NewOwner(nil)

	effectRan := false
	effect := &Effect{
		id:    nextID(),
		owner: owner,
		fn: func() Cleanup {
			effectRan = true
			return nil
		},
	}
	effect.pending.Store(true)

	owner.scheduleEffect(effect)
	owner.RunPendingEffects()

	if !effectRan {
		t.Error("effect should have run")
	}

	if effect.pending.Load() {
		t.Error("effect should not be pending after run")
	}
}

func TestOwnerConcurrent(t *testing.T) {
	root := NewOwner(nil)
	var wg sync.WaitGroup
	const numGoroutines = 100

	// Concurrent child creation
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			child := NewOwner(root)
			child.OnCleanup(func() {})
		}()
	}
	wg.Wait()

	// Dispose should work without panicking
	root.Dispose()
}

// ============================================================================
// Hook Order Validation Tests (§3.1.3)
// ============================================================================

func TestHookOrderTrackingDisabledByDefault(t *testing.T) {
	// With DebugMode off, tracking should be no-op
	oldDebugMode := DebugMode
	DebugMode = false
	defer func() { DebugMode = oldDebugMode }()

	owner := NewOwner(nil)

	// Should not panic even with different hook orders
	owner.StartRender()
	owner.TrackHook(HookSignal)
	owner.EndRender()

	owner.StartRender()
	owner.TrackHook(HookMemo) // Different hook type on re-render
	owner.EndRender()
	// No panic expected - tracking is off
}

func TestHookOrderTrackingFirstRender(t *testing.T) {
	// Enable debug mode
	oldDebugMode := DebugMode
	DebugMode = true
	defer func() { DebugMode = oldDebugMode }()

	owner := NewOwner(nil)

	owner.StartRender()
	owner.TrackHook(HookSignal)
	owner.TrackHook(HookMemo)
	owner.TrackHook(HookEffect)
	owner.EndRender()

	// Hook order should be recorded: [Signal, Memo, Effect]
	if len(owner.hookOrder) != 3 {
		t.Errorf("expected 3 hooks recorded, got %d", len(owner.hookOrder))
	}
	if owner.hookOrder[0].Type != HookSignal {
		t.Errorf("expected first hook to be Signal, got %s", owner.hookOrder[0].Type)
	}
	if owner.hookOrder[1].Type != HookMemo {
		t.Errorf("expected second hook to be Memo, got %s", owner.hookOrder[1].Type)
	}
	if owner.hookOrder[2].Type != HookEffect {
		t.Errorf("expected third hook to be Effect, got %s", owner.hookOrder[2].Type)
	}
}

func TestHookOrderTrackingSecondRenderSameOrder(t *testing.T) {
	// Enable debug mode
	oldDebugMode := DebugMode
	DebugMode = true
	defer func() { DebugMode = oldDebugMode }()

	owner := NewOwner(nil)

	// First render
	owner.StartRender()
	owner.TrackHook(HookSignal)
	owner.TrackHook(HookMemo)
	owner.EndRender()

	// Second render with same order - should not panic
	owner.StartRender()
	owner.TrackHook(HookSignal)
	owner.TrackHook(HookMemo)
	owner.EndRender()
	// No panic expected
}

func TestHookOrderTrackingWrongTypePanics(t *testing.T) {
	// Enable debug mode
	oldDebugMode := DebugMode
	DebugMode = true
	defer func() { DebugMode = oldDebugMode }()

	owner := NewOwner(nil)

	// First render
	owner.StartRender()
	owner.TrackHook(HookSignal)
	owner.TrackHook(HookMemo)
	owner.EndRender()

	// Second render with different type at index 1
	defer func() {
		r := recover()
		if r == nil {
			t.Error("expected panic for hook order change")
		}
		msg, ok := r.(string)
		if !ok {
			t.Errorf("expected string panic, got %T: %v", r, r)
		}
		if !contains(msg, "VANGO E002") {
			t.Errorf("expected panic message to contain 'VANGO E002', got: %s", msg)
		}
	}()

	owner.StartRender()
	owner.TrackHook(HookSignal)
	owner.TrackHook(HookEffect) // Wrong! Was Memo before
	owner.EndRender()
}

func TestHookOrderTrackingExtraHookPanics(t *testing.T) {
	// Enable debug mode
	oldDebugMode := DebugMode
	DebugMode = true
	defer func() { DebugMode = oldDebugMode }()

	owner := NewOwner(nil)

	// First render
	owner.StartRender()
	owner.TrackHook(HookSignal)
	owner.EndRender()

	// Second render with extra hook
	defer func() {
		r := recover()
		if r == nil {
			t.Error("expected panic for extra hook")
		}
		msg, ok := r.(string)
		if !ok {
			t.Errorf("expected string panic, got %T: %v", r, r)
		}
		if !contains(msg, "VANGO E002") {
			t.Errorf("expected panic message to contain 'VANGO E002', got: %s", msg)
		}
	}()

	owner.StartRender()
	owner.TrackHook(HookSignal)
	owner.TrackHook(HookMemo) // Extra hook!
	owner.EndRender()
}

func TestHookOrderTrackingMissingHookPanics(t *testing.T) {
	// Enable debug mode
	oldDebugMode := DebugMode
	DebugMode = true
	defer func() { DebugMode = oldDebugMode }()

	owner := NewOwner(nil)

	// First render with 2 hooks
	owner.StartRender()
	owner.TrackHook(HookSignal)
	owner.TrackHook(HookMemo)
	owner.EndRender()

	// Second render with only 1 hook - should panic on EndRender
	defer func() {
		r := recover()
		if r == nil {
			t.Error("expected panic for missing hook")
		}
		msg, ok := r.(string)
		if !ok {
			t.Errorf("expected string panic, got %T: %v", r, r)
		}
		if !contains(msg, "VANGO E002") {
			t.Errorf("expected panic message to contain 'VANGO E002', got: %s", msg)
		}
	}()

	owner.StartRender()
	owner.TrackHook(HookSignal)
	// Missing HookMemo!
	owner.EndRender()
}

func TestHookTypeString(t *testing.T) {
	tests := []struct {
		hook     HookType
		expected string
	}{
		{HookSignal, "Signal"},
		{HookMemo, "Memo"},
		{HookEffect, "Effect"},
		{HookResource, "Resource"},
		{HookForm, "Form"},
		{HookURLParam, "URLParam"},
		{HookRef, "Ref"},
		{HookContext, "Context"},
		{HookType(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.hook.String(); got != tt.expected {
				t.Errorf("HookType(%d).String() = %q, want %q", tt.hook, got, tt.expected)
			}
		})
	}
}

func TestTrackHookPublicAPI(t *testing.T) {
	// Enable debug mode
	oldDebugMode := DebugMode
	DebugMode = true
	defer func() { DebugMode = oldDebugMode }()

	owner := NewOwner(nil)

	// Set owner context
	WithOwner(owner, func() {
		owner.StartRender()

		// Public TrackHook should work
		TrackHook(HookSignal)
		TrackHook(HookForm)
		TrackHook(HookResource)

		owner.EndRender()
	})

	if len(owner.hookOrder) != 3 {
		t.Errorf("expected 3 hooks recorded via public API, got %d", len(owner.hookOrder))
	}
}

func TestTrackHookNoOwnerNoOp(t *testing.T) {
	// Public TrackHook should be no-op when no owner is set
	TrackHook(HookSignal) // Should not panic
}

// ============================================================================
// Hook Slot Tests
// ============================================================================

func TestHookSlotFirstRenderReturnsNil(t *testing.T) {
	owner := NewOwner(nil)

	WithOwner(owner, func() {
		owner.StartRender()

		// First render: UseHookSlot returns nil
		slot := UseHookSlot()
		if slot != nil {
			t.Errorf("expected nil on first render, got %v", slot)
		}

		// Store a value
		SetHookSlot("test-value")

		owner.EndRender()
	})
}

func TestHookSlotSubsequentRenderReturnsStored(t *testing.T) {
	owner := NewOwner(nil)

	// First render
	WithOwner(owner, func() {
		owner.StartRender()
		slot := UseHookSlot()
		if slot != nil {
			t.Fatal("expected nil on first render")
		}
		SetHookSlot("my-hook-value")
		owner.EndRender()
	})

	// Second render: should return stored value
	WithOwner(owner, func() {
		owner.StartRender()
		slot := UseHookSlot()
		if slot != "my-hook-value" {
			t.Errorf("expected stored value, got %v", slot)
		}
		owner.EndRender()
	})
}

func TestHookSlotMultipleSlots(t *testing.T) {
	owner := NewOwner(nil)

	// First render: create multiple slots
	WithOwner(owner, func() {
		owner.StartRender()

		slot1 := UseHookSlot()
		if slot1 != nil {
			t.Fatal("slot1 should be nil on first render")
		}
		SetHookSlot("first")

		slot2 := UseHookSlot()
		if slot2 != nil {
			t.Fatal("slot2 should be nil on first render")
		}
		SetHookSlot(42)

		slot3 := UseHookSlot()
		if slot3 != nil {
			t.Fatal("slot3 should be nil on first render")
		}
		SetHookSlot(struct{ Name string }{"test"})

		owner.EndRender()
	})

	// Second render: all slots should return their values
	WithOwner(owner, func() {
		owner.StartRender()

		slot1 := UseHookSlot()
		if slot1 != "first" {
			t.Errorf("slot1 = %v, want 'first'", slot1)
		}

		slot2 := UseHookSlot()
		if slot2 != 42 {
			t.Errorf("slot2 = %v, want 42", slot2)
		}

		slot3 := UseHookSlot()
		if s, ok := slot3.(struct{ Name string }); !ok || s.Name != "test" {
			t.Errorf("slot3 = %v, want struct{Name:test}", slot3)
		}

		owner.EndRender()
	})
}

func TestHookSlotNoOwnerReturnsNil(t *testing.T) {
	// Outside render context, UseHookSlot should return nil
	slot := UseHookSlot()
	if slot != nil {
		t.Errorf("expected nil when no owner, got %v", slot)
	}

	// SetHookSlot should be a no-op
	SetHookSlot("value") // Should not panic
}

// contains checks if s contains substr
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
