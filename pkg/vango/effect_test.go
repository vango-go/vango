package vango

import (
	"sync/atomic"
	"testing"
)

func TestEffectRunsOnCreate(t *testing.T) {
	owner := NewOwner(nil)
	defer owner.Dispose()

	ran := false
	WithOwner(owner, func() {
		CreateEffect(func() Cleanup {
			ran = true
			return nil
		})
	})

	if !ran {
		t.Error("effect should run immediately on creation")
	}
}

func TestEffectTracksDependencies(t *testing.T) {
	owner := NewOwner(nil)
	defer owner.Dispose()

	count := NewSignal(0)
	runCount := 0

	WithOwner(owner, func() {
		CreateEffect(func() Cleanup {
			_ = count.Get()
			runCount++
			return nil
		})
	})

	if runCount != 1 {
		t.Errorf("expected 1 run, got %d", runCount)
	}

	// Changing the signal should schedule the effect
	count.Set(1)

	// Run pending effects
	owner.RunPendingEffects(nil)

	if runCount != 2 {
		t.Errorf("expected 2 runs after signal change, got %d", runCount)
	}
}

func TestEffectCleanup(t *testing.T) {
	owner := NewOwner(nil)

	cleanupRan := false
	WithOwner(owner, func() {
		CreateEffect(func() Cleanup {
			return func() {
				cleanupRan = true
			}
		})
	})

	if cleanupRan {
		t.Error("cleanup should not run immediately")
	}

	owner.Dispose()

	if !cleanupRan {
		t.Error("cleanup should run on dispose")
	}
}

func TestEffectCleanupBeforeRerun(t *testing.T) {
	owner := NewOwner(nil)
	defer owner.Dispose()

	count := NewSignal(0)
	cleanupCount := 0
	runCount := 0

	WithOwner(owner, func() {
		CreateEffect(func() Cleanup {
			_ = count.Get()
			runCount++
			return func() {
				cleanupCount++
			}
		})
	})

	if runCount != 1 {
		t.Errorf("expected 1 run, got %d", runCount)
	}
	if cleanupCount != 0 {
		t.Errorf("expected 0 cleanups, got %d", cleanupCount)
	}

	// Trigger re-run
	count.Set(1)
	owner.RunPendingEffects(nil)

	if runCount != 2 {
		t.Errorf("expected 2 runs, got %d", runCount)
	}
	if cleanupCount != 1 {
		t.Errorf("expected 1 cleanup before re-run, got %d", cleanupCount)
	}
}

func TestEffectDynamicDependencies(t *testing.T) {
	owner := NewOwner(nil)
	defer owner.Dispose()

	flag := NewSignal(true)
	a := NewSignal(1)
	b := NewSignal(2)

	runCount := 0
	var lastValue int

	WithOwner(owner, func() {
		CreateEffect(func() Cleanup {
			runCount++
			if flag.Get() {
				lastValue = a.Get()
			} else {
				lastValue = b.Get()
			}
			return nil
		})
	})

	if runCount != 1 || lastValue != 1 {
		t.Errorf("expected 1 run with value 1, got %d runs with value %d", runCount, lastValue)
	}

	// Changing b should NOT trigger (not currently tracked)
	b.Set(20)
	owner.RunPendingEffects(nil)
	if runCount != 1 {
		t.Errorf("changing b should not trigger, got %d runs", runCount)
	}

	// Changing a should trigger
	a.Set(10)
	owner.RunPendingEffects(nil)
	if runCount != 2 || lastValue != 10 {
		t.Errorf("expected 2 runs with value 10, got %d runs with value %d", runCount, lastValue)
	}

	// Switch to b
	flag.Set(false)
	owner.RunPendingEffects(nil)
	if lastValue != 20 {
		t.Errorf("expected value 20 after switching, got %d", lastValue)
	}

	// Now a should not trigger, b should
	runCount = 0
	a.Set(100)
	owner.RunPendingEffects(nil)
	if runCount != 0 {
		t.Errorf("changing a should not trigger when using b, got %d runs", runCount)
	}

	b.Set(200)
	owner.RunPendingEffects(nil)
	if lastValue != 200 {
		t.Errorf("expected value 200, got %d", lastValue)
	}
}

func TestOnMount(t *testing.T) {
	owner := NewOwner(nil)
	defer owner.Dispose()

	ran := false
	WithOwner(owner, func() {
		OnMount(func() {
			ran = true
		})
	})

	if !ran {
		t.Error("OnMount should run immediately")
	}
}

func TestOnUnmount(t *testing.T) {
	owner := NewOwner(nil)

	ran := false
	WithOwner(owner, func() {
		OnUnmount(func() {
			ran = true
		})
	})

	if ran {
		t.Error("OnUnmount should not run before dispose")
	}

	owner.Dispose()

	if !ran {
		t.Error("OnUnmount should run on dispose")
	}
}

func TestOnUpdate(t *testing.T) {
	owner := NewOwner(nil)
	defer owner.Dispose()

	count := NewSignal(0)
	updateCount := 0

	WithOwner(owner, func() {
		OnUpdate(
			func() { _ = count.Get() }, // deps: track count
			func() { updateCount++ },   // callback: only on updates
		)
	})

	// Should not run callback on first render
	if updateCount != 0 {
		t.Errorf("OnUpdate should not run callback on first render, got %d", updateCount)
	}

	// Should run on update
	count.Set(1)
	owner.RunPendingEffects(nil)

	if updateCount != 1 {
		t.Errorf("OnUpdate should run callback on update, got %d", updateCount)
	}

	// And again
	count.Set(2)
	owner.RunPendingEffects(nil)

	if updateCount != 2 {
		t.Errorf("OnUpdate should run callback on each update, got %d", updateCount)
	}
}

func TestEffectID(t *testing.T) {
	owner := NewOwner(nil)
	defer owner.Dispose()

	var e1, e2 *Effect
	WithOwner(owner, func() {
		e1 = CreateEffect(func() Cleanup { return nil })
		e2 = CreateEffect(func() Cleanup { return nil })
	})

	if e1.ID() == e2.ID() {
		t.Error("effects should have unique IDs")
	}
}

func TestEffectNoOwner(t *testing.T) {
	// Effect without owner should still work
	ran := false
	e := CreateEffect(func() Cleanup {
		ran = true
		return nil
	})

	if !ran {
		t.Error("effect should run without owner")
	}

	// But cleanup won't be automatic
	e.dispose()
}

func TestEffectDisposed(t *testing.T) {
	owner := NewOwner(nil)

	count := NewSignal(0)
	var runCount atomic.Int32

	var effect *Effect
	WithOwner(owner, func() {
		effect = CreateEffect(func() Cleanup {
			_ = count.Get()
			runCount.Add(1)
			return nil
		})
	})

	if runCount.Load() != 1 {
		t.Errorf("expected 1 run, got %d", runCount.Load())
	}

	// Dispose the owner
	owner.Dispose()

	// Effect should be disposed
	if !effect.disposed.Load() {
		t.Error("effect should be disposed")
	}

	// Changing signal should not trigger effect
	count.Set(100)

	if runCount.Load() != 1 {
		t.Errorf("disposed effect should not run, got %d runs", runCount.Load())
	}
}

func TestEffectMarkDirtyIdempotent(t *testing.T) {
	owner := NewOwner(nil)
	defer owner.Dispose()

	runCount := 0
	count := NewSignal(0)

	WithOwner(owner, func() {
		CreateEffect(func() Cleanup {
			_ = count.Get()
			runCount++
			return nil
		})
	})

	if runCount != 1 {
		t.Errorf("expected 1 initial run, got %d", runCount)
	}

	// Multiple sets should only schedule once
	count.Set(1)
	count.Set(2)
	count.Set(3)

	owner.RunPendingEffects(nil)

	if runCount != 2 {
		t.Errorf("expected 2 total runs (deduped), got %d", runCount)
	}
}
