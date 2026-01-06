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

// =============================================================================
// Convenience Methods Tests (§3.9.4)
// =============================================================================

func TestSignalIncDec(t *testing.T) {
	count := NewSignal(0)

	count.Inc()
	if count.Get() != 1 {
		t.Errorf("expected 1, got %d", count.Get())
	}

	count.Inc()
	count.Inc()
	if count.Get() != 3 {
		t.Errorf("expected 3, got %d", count.Get())
	}

	count.Dec()
	if count.Get() != 2 {
		t.Errorf("expected 2, got %d", count.Get())
	}

	// Test with int64
	count64 := NewSignal(int64(100))
	count64.Inc()
	if count64.Get() != 101 {
		t.Errorf("expected 101, got %d", count64.Get())
	}
	count64.Dec()
	if count64.Get() != 100 {
		t.Errorf("expected 100, got %d", count64.Get())
	}

	// Test with float64
	countF := NewSignal(1.5)
	countF.Inc()
	if countF.Get() != 2.5 {
		t.Errorf("expected 2.5, got %f", countF.Get())
	}
	countF.Dec()
	if countF.Get() != 1.5 {
		t.Errorf("expected 1.5, got %f", countF.Get())
	}
}

func TestSignalAddSubMulDiv(t *testing.T) {
	count := NewSignal(10)

	count.Add(5)
	if count.Get() != 15 {
		t.Errorf("expected 15, got %d", count.Get())
	}

	count.Sub(3)
	if count.Get() != 12 {
		t.Errorf("expected 12, got %d", count.Get())
	}

	count.Mul(2)
	if count.Get() != 24 {
		t.Errorf("expected 24, got %d", count.Get())
	}

	count.Div(4)
	if count.Get() != 6 {
		t.Errorf("expected 6, got %d", count.Get())
	}

	// Test with float64
	value := NewSignal(10.0)
	value.Add(2.5)
	if value.Get() != 12.5 {
		t.Errorf("expected 12.5, got %f", value.Get())
	}
	value.Div(2.5)
	if value.Get() != 5.0 {
		t.Errorf("expected 5.0, got %f", value.Get())
	}
}

func TestSignalToggle(t *testing.T) {
	visible := NewSignal(false)

	visible.Toggle()
	if !visible.Get() {
		t.Error("expected true after toggle")
	}

	visible.Toggle()
	if visible.Get() {
		t.Error("expected false after second toggle")
	}
}

func TestSignalSetTrueSetFalse(t *testing.T) {
	enabled := NewSignal(false)
	listener := newTestListener()

	WithListener(listener, func() {
		_ = enabled.Get()
	})

	enabled.SetTrue()
	if !enabled.Get() {
		t.Error("expected true")
	}
	if listener.getDirtyCount() != 1 {
		t.Errorf("expected 1 notification, got %d", listener.getDirtyCount())
	}

	// Setting true again should not notify
	enabled.SetTrue()
	if listener.getDirtyCount() != 1 {
		t.Errorf("expected 1 notification (no change), got %d", listener.getDirtyCount())
	}

	enabled.SetFalse()
	if enabled.Get() {
		t.Error("expected false")
	}
	if listener.getDirtyCount() != 2 {
		t.Errorf("expected 2 notifications, got %d", listener.getDirtyCount())
	}
}

func TestSignalStringAppendPrepend(t *testing.T) {
	text := NewSignal("")

	text.Append("hello")
	if text.Get() != "hello" {
		t.Errorf("expected 'hello', got '%s'", text.Get())
	}

	text.Append(" world")
	if text.Get() != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", text.Get())
	}

	text.Prepend("say: ")
	if text.Get() != "say: hello world" {
		t.Errorf("expected 'say: hello world', got '%s'", text.Get())
	}
}

func TestSignalSliceAppendItem(t *testing.T) {
	items := NewSignal([]int{})

	items.AppendItem(1)
	if len(items.Get()) != 1 || items.Get()[0] != 1 {
		t.Errorf("expected [1], got %v", items.Get())
	}

	items.AppendItem(2)
	items.AppendItem(3)
	expected := []int{1, 2, 3}
	got := items.Get()
	if len(got) != len(expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}
	for i, v := range expected {
		if got[i] != v {
			t.Errorf("expected %v, got %v", expected, got)
			break
		}
	}
}

func TestSignalSlicePrependItem(t *testing.T) {
	items := NewSignal([]int{2, 3})

	items.PrependItem(1)
	expected := []int{1, 2, 3}
	got := items.Get()
	if len(got) != len(expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}
	for i, v := range expected {
		if got[i] != v {
			t.Errorf("expected %v, got %v", expected, got)
			break
		}
	}
}

func TestSignalSliceRemoveAt(t *testing.T) {
	items := NewSignal([]int{1, 2, 3, 4, 5})

	items.RemoveAt(2) // Remove 3
	expected := []int{1, 2, 4, 5}
	got := items.Get()
	if len(got) != len(expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}

	// Out of bounds should do nothing
	items.RemoveAt(100)
	items.RemoveAt(-1)
	if len(items.Get()) != 4 {
		t.Errorf("out of bounds RemoveAt should do nothing")
	}
}

func TestSignalSliceRemoveFirstLast(t *testing.T) {
	items := NewSignal([]int{1, 2, 3, 4, 5})

	items.RemoveFirst()
	if items.Get()[0] != 2 {
		t.Errorf("expected first element to be 2, got %d", items.Get()[0])
	}

	items.RemoveLast()
	last := items.Get()[len(items.Get())-1]
	if last != 4 {
		t.Errorf("expected last element to be 4, got %d", last)
	}
}

func TestSignalSliceRemoveWhere(t *testing.T) {
	items := NewSignal([]int{1, 2, 3, 4, 5, 6})

	// Remove even numbers
	items.RemoveWhere(func(item any) bool {
		return item.(int)%2 == 0
	})

	expected := []int{1, 3, 5}
	got := items.Get()
	if len(got) != len(expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}
}

func TestSignalSliceInsertAt(t *testing.T) {
	items := NewSignal([]int{1, 3})

	items.InsertAt(1, 2)
	expected := []int{1, 2, 3}
	got := items.Get()
	if len(got) != len(expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}

	// Insert at beginning (negative index)
	items.InsertAt(-1, 0)
	if items.Get()[0] != 0 {
		t.Errorf("expected 0 at start, got %d", items.Get()[0])
	}

	// Insert at end (out of bounds)
	items.InsertAt(100, 100)
	lastIdx := len(items.Get()) - 1
	if items.Get()[lastIdx] != 100 {
		t.Errorf("expected 100 at end, got %d", items.Get()[lastIdx])
	}
}

func TestSignalSliceSetAt(t *testing.T) {
	items := NewSignal([]int{1, 2, 3})

	items.SetAt(1, 20)
	if items.Get()[1] != 20 {
		t.Errorf("expected 20 at index 1, got %d", items.Get()[1])
	}

	// Out of bounds should do nothing
	items.SetAt(100, 999)
	if len(items.Get()) != 3 {
		t.Error("out of bounds SetAt should not change slice")
	}
}

func TestSignalSliceUpdateAt(t *testing.T) {
	items := NewSignal([]int{1, 2, 3})

	items.UpdateAt(1, func(item any) any {
		return item.(int) * 10
	})
	if items.Get()[1] != 20 {
		t.Errorf("expected 20 at index 1, got %d", items.Get()[1])
	}
}

func TestSignalSliceFilter(t *testing.T) {
	items := NewSignal([]int{1, 2, 3, 4, 5, 6})

	// Keep odd numbers
	items.Filter(func(item any) bool {
		return item.(int)%2 != 0
	})

	expected := []int{1, 3, 5}
	got := items.Get()
	if len(got) != len(expected) {
		t.Errorf("expected %v, got %v", expected, got)
	}
}

func TestSignalMapSetKeyRemoveKey(t *testing.T) {
	users := NewSignal(map[string]int{})

	users.SetKey("alice", 30)
	if v, ok := users.Get()["alice"]; !ok || v != 30 {
		t.Errorf("expected alice=30, got %v", users.Get())
	}

	users.SetKey("bob", 25)
	users.RemoveKey("alice")

	if _, ok := users.Get()["alice"]; ok {
		t.Error("alice should be removed")
	}
	if v, ok := users.Get()["bob"]; !ok || v != 25 {
		t.Errorf("expected bob=25, got %v", users.Get())
	}
}

func TestSignalMapUpdateKey(t *testing.T) {
	scores := NewSignal(map[string]int{"alice": 100})

	scores.UpdateKey("alice", func(v any) any {
		return v.(int) + 50
	})

	if scores.Get()["alice"] != 150 {
		t.Errorf("expected alice=150, got %d", scores.Get()["alice"])
	}
}

func TestSignalMapHasKey(t *testing.T) {
	users := NewSignal(map[string]int{"alice": 30})

	if !users.HasKey("alice") {
		t.Error("expected HasKey(alice) to be true")
	}
	if users.HasKey("bob") {
		t.Error("expected HasKey(bob) to be false")
	}
}

func TestSignalClearAndLen(t *testing.T) {
	// Test string
	text := NewSignal("hello")
	if text.Len() != 5 {
		t.Errorf("expected len 5, got %d", text.Len())
	}
	text.Clear()
	if text.Get() != "" {
		t.Errorf("expected empty string, got '%s'", text.Get())
	}

	// Test slice
	items := NewSignal([]int{1, 2, 3})
	if items.Len() != 3 {
		t.Errorf("expected len 3, got %d", items.Len())
	}
	items.Clear()
	if items.Len() != 0 {
		t.Errorf("expected len 0, got %d", items.Len())
	}

	// Test map
	data := NewSignal(map[string]int{"a": 1, "b": 2})
	if data.Len() != 2 {
		t.Errorf("expected len 2, got %d", data.Len())
	}
	data.Clear()
	if data.Len() != 0 {
		t.Errorf("expected len 0, got %d", data.Len())
	}
}

func TestSignalTypeMismatchPanics(t *testing.T) {
	// Test Inc on string
	str := NewSignal("hello")
	assertPanics(t, "Inc() on string should panic", func() {
		str.Inc()
	})

	// Test Toggle on int
	num := NewSignal(42)
	assertPanics(t, "Toggle() on int should panic", func() {
		num.Toggle()
	})

	// Test Append on int
	assertPanics(t, "Append() on int should panic", func() {
		num.Append("test")
	})

	// Test AppendItem on string
	assertPanics(t, "AppendItem() on string should panic", func() {
		str.AppendItem("item")
	})

	// Test SetKey on slice
	items := NewSignal([]int{})
	assertPanics(t, "SetKey() on slice should panic", func() {
		items.SetKey("key", "value")
	})

	// Test Clear on int
	assertPanics(t, "Clear() on int should panic", func() {
		num.Clear()
	})

	// Test Len on int
	assertPanics(t, "Len() on int should panic", func() {
		num.Len()
	})
}

func TestSignalAddTypeMismatch(t *testing.T) {
	count := NewSignal(0)
	assertPanics(t, "Add(string) on int signal should panic", func() {
		count.Add("not an int")
	})

	// Wrong numeric type
	assertPanics(t, "Add(float64) on int signal should panic", func() {
		count.Add(1.5)
	})
}

func assertPanics(t *testing.T, msg string, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("%s - expected panic but got none", msg)
		}
	}()
	fn()
}
