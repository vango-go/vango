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
