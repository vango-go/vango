package vango

import (
	"sync"
	"testing"
)

func TestSignalBasic(t *testing.T) {
	count := NewSignal(0)

	// Initial value
	if count.Get() != 0 {
		t.Errorf("expected initial value 0, got %d", count.Get())
	}

	// Set value
	count.Set(5)
	if count.Get() != 5 {
		t.Errorf("expected value 5, got %d", count.Get())
	}

	// Update value
	count.Update(func(n int) int { return n * 2 })
	if count.Get() != 10 {
		t.Errorf("expected value 10, got %d", count.Get())
	}
}

func TestSignalPeek(t *testing.T) {
	count := NewSignal(42)

	// Peek should return value without subscribing
	listener := newTestListener()
	WithListener(listener, func() {
		value := count.Peek()
		if value != 42 {
			t.Errorf("expected 42, got %d", value)
		}
	})

	// Listener should not be subscribed
	count.Set(100)
	if listener.getDirtyCount() != 0 {
		t.Errorf("Peek should not subscribe listener, got %d notifications", listener.getDirtyCount())
	}
}

func TestSignalSubscription(t *testing.T) {
	count := NewSignal(0)
	listener := newTestListener()

	// Subscribe by reading within tracked context
	WithListener(listener, func() {
		_ = count.Get()
	})

	// Setting should notify
	count.Set(1)
	if listener.getDirtyCount() != 1 {
		t.Errorf("expected 1 notification, got %d", listener.getDirtyCount())
	}

	// Same value should not notify
	count.Set(1)
	if listener.getDirtyCount() != 1 {
		t.Errorf("same value should not notify, got %d", listener.getDirtyCount())
	}

	// Different value should notify
	count.Set(2)
	if listener.getDirtyCount() != 2 {
		t.Errorf("expected 2 notifications, got %d", listener.getDirtyCount())
	}
}

func TestSignalNoTrackingOutsideContext(t *testing.T) {
	count := NewSignal(0)
	listener := newTestListener()

	// Read outside of tracking context
	_ = count.Get()

	// Set listener after read
	WithListener(listener, func() {
		// Don't read the signal here
	})

	// Should not notify since we didn't read while tracking
	count.Set(1)
	if listener.getDirtyCount() != 0 {
		t.Errorf("expected 0 notifications when not tracking, got %d", listener.getDirtyCount())
	}
}

func TestSignalMultipleSubscribers(t *testing.T) {
	count := NewSignal(0)
	listener1 := newTestListener()
	listener2 := newTestListener()
	listener3 := newTestListener()

	// Subscribe all listeners
	WithListener(listener1, func() {
		_ = count.Get()
	})
	WithListener(listener2, func() {
		_ = count.Get()
	})
	WithListener(listener3, func() {
		_ = count.Get()
	})

	// All should be notified
	count.Set(1)
	if listener1.getDirtyCount() != 1 {
		t.Errorf("listener1 expected 1 notification, got %d", listener1.getDirtyCount())
	}
	if listener2.getDirtyCount() != 1 {
		t.Errorf("listener2 expected 1 notification, got %d", listener2.getDirtyCount())
	}
	if listener3.getDirtyCount() != 1 {
		t.Errorf("listener3 expected 1 notification, got %d", listener3.getDirtyCount())
	}
}

func TestSignalDeduplicateSubscription(t *testing.T) {
	count := NewSignal(0)
	listener := newTestListener()

	// Subscribe multiple times with same listener
	WithListener(listener, func() {
		_ = count.Get()
		_ = count.Get()
		_ = count.Get()
	})

	// Should only notify once
	count.Set(1)
	if listener.getDirtyCount() != 1 {
		t.Errorf("expected 1 notification (deduplicated), got %d", listener.getDirtyCount())
	}
}

func TestSignalCustomEquals(t *testing.T) {
	type user struct {
		ID   int
		Name string
	}

	// Custom equality: only compare ID
	userSignal := NewSignal(user{ID: 1, Name: "Alice"}).WithEquals(func(a, b user) bool {
		return a.ID == b.ID
	})

	listener := newTestListener()
	WithListener(listener, func() {
		_ = userSignal.Get()
	})

	// Same ID, different name - should not notify
	userSignal.Set(user{ID: 1, Name: "Alice Smith"})
	if listener.getDirtyCount() != 0 {
		t.Errorf("expected 0 notifications for same ID, got %d", listener.getDirtyCount())
	}

	// Different ID - should notify
	userSignal.Set(user{ID: 2, Name: "Bob"})
	if listener.getDirtyCount() != 1 {
		t.Errorf("expected 1 notification for different ID, got %d", listener.getDirtyCount())
	}
}

func TestSignalSliceEquality(t *testing.T) {
	items := NewSignal([]int{1, 2, 3})
	listener := newTestListener()

	WithListener(listener, func() {
		_ = items.Get()
	})

	// Same values - should not notify (DeepEqual)
	items.Set([]int{1, 2, 3})
	if listener.getDirtyCount() != 0 {
		t.Errorf("expected 0 notifications for equal slice, got %d", listener.getDirtyCount())
	}

	// Different values - should notify
	items.Set([]int{1, 2, 3, 4})
	if listener.getDirtyCount() != 1 {
		t.Errorf("expected 1 notification for different slice, got %d", listener.getDirtyCount())
	}
}

func TestSignalMapEquality(t *testing.T) {
	data := NewSignal(map[string]int{"a": 1})
	listener := newTestListener()

	WithListener(listener, func() {
		_ = data.Get()
	})

	// Same values - should not notify (DeepEqual)
	data.Set(map[string]int{"a": 1})
	if listener.getDirtyCount() != 0 {
		t.Errorf("expected 0 notifications for equal map, got %d", listener.getDirtyCount())
	}

	// Different values - should notify
	data.Set(map[string]int{"a": 2})
	if listener.getDirtyCount() != 1 {
		t.Errorf("expected 1 notification for different map, got %d", listener.getDirtyCount())
	}
}

func TestSignalConcurrentAccess(t *testing.T) {
	count := NewSignal(0)
	var wg sync.WaitGroup
	const numGoroutines = 100
	const numIterations = 100

	// Concurrent reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				_ = count.Get()
			}
		}()
	}
	wg.Wait()

	// Concurrent writes
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				count.Set(id*numIterations + j)
			}
		}(i)
	}
	wg.Wait()

	// Concurrent read/write
	wg.Add(numGoroutines * 2)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				_ = count.Get()
			}
		}()
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				count.Set(id)
			}
		}(i)
	}
	wg.Wait()
}

func TestSignalConcurrentSubscription(t *testing.T) {
	count := NewSignal(0)
	var wg sync.WaitGroup
	const numGoroutines = 100

	listeners := make([]*testListener, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		listeners[i] = newTestListener()
	}

	// Concurrent subscription
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			WithListener(listeners[idx], func() {
				_ = count.Get()
			})
		}(i)
	}
	wg.Wait()

	// Setting should notify all subscribers
	count.Set(1)

	// Each listener should have been notified exactly once
	for i, listener := range listeners {
		if listener.getDirtyCount() != 1 {
			t.Errorf("listener %d expected 1 notification, got %d", i, listener.getDirtyCount())
		}
	}
}

func TestSignalID(t *testing.T) {
	s1 := NewSignal(0)
	s2 := NewSignal(0)
	s3 := NewSignal(0)

	// IDs should be unique
	if s1.ID() == s2.ID() {
		t.Error("signals should have unique IDs")
	}
	if s2.ID() == s3.ID() {
		t.Error("signals should have unique IDs")
	}
	if s1.ID() == s3.ID() {
		t.Error("signals should have unique IDs")
	}
}

func TestSignalNilValue(t *testing.T) {
	var ptr *int
	s := NewSignal(ptr)

	if s.Get() != nil {
		t.Error("expected nil initial value")
	}

	// Setting to nil again should not notify
	listener := newTestListener()
	WithListener(listener, func() {
		_ = s.Get()
	})

	s.Set(nil)
	if listener.getDirtyCount() != 0 {
		t.Errorf("setting nil to nil should not notify, got %d", listener.getDirtyCount())
	}

	// Setting to non-nil should notify
	val := 42
	s.Set(&val)
	if listener.getDirtyCount() != 1 {
		t.Errorf("expected 1 notification, got %d", listener.getDirtyCount())
	}
}

func TestSignalUpdateNoChange(t *testing.T) {
	count := NewSignal(5)
	listener := newTestListener()

	WithListener(listener, func() {
		_ = count.Get()
	})

	// Update that returns same value should not notify
	count.Update(func(n int) int { return n })
	if listener.getDirtyCount() != 0 {
		t.Errorf("update returning same value should not notify, got %d", listener.getDirtyCount())
	}

	// Update that returns different value should notify
	count.Update(func(n int) int { return n + 1 })
	if listener.getDirtyCount() != 1 {
		t.Errorf("expected 1 notification, got %d", listener.getDirtyCount())
	}
}
