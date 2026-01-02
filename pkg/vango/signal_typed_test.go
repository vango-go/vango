package vango

import "testing"

func TestIntSignal(t *testing.T) {
	count := NewIntSignal(0)

	if count.Get() != 0 {
		t.Errorf("expected 0, got %d", count.Get())
	}

	count.Inc()
	if count.Get() != 1 {
		t.Errorf("expected 1 after Inc, got %d", count.Get())
	}

	count.Inc()
	count.Inc()
	if count.Get() != 3 {
		t.Errorf("expected 3 after multiple Inc, got %d", count.Get())
	}

	count.Dec()
	if count.Get() != 2 {
		t.Errorf("expected 2 after Dec, got %d", count.Get())
	}

	count.Add(10)
	if count.Get() != 12 {
		t.Errorf("expected 12 after Add(10), got %d", count.Get())
	}

	count.Add(-5)
	if count.Get() != 7 {
		t.Errorf("expected 7 after Add(-5), got %d", count.Get())
	}
}

func TestIntSignalNotifications(t *testing.T) {
	count := NewIntSignal(0)
	listener := newTestListener()

	WithListener(listener, func() {
		_ = count.Get()
	})

	count.Inc()
	if listener.getDirtyCount() != 1 {
		t.Errorf("expected 1 notification after Inc, got %d", listener.getDirtyCount())
	}

	count.Dec()
	if listener.getDirtyCount() != 2 {
		t.Errorf("expected 2 notifications, got %d", listener.getDirtyCount())
	}
}

func TestInt64Signal(t *testing.T) {
	count := NewInt64Signal(0)

	count.Inc()
	if count.Get() != 1 {
		t.Errorf("expected 1, got %d", count.Get())
	}

	count.Add(1000000000000)
	if count.Get() != 1000000000001 {
		t.Errorf("expected 1000000000001, got %d", count.Get())
	}
}

func TestFloat64Signal(t *testing.T) {
	value := NewFloat64Signal(1.5)

	value.Add(2.5)
	if value.Get() != 4.0 {
		t.Errorf("expected 4.0, got %f", value.Get())
	}

	value.Multiply(2.0)
	if value.Get() != 8.0 {
		t.Errorf("expected 8.0, got %f", value.Get())
	}
}

func TestBoolSignal(t *testing.T) {
	flag := NewBoolSignal(false)

	if flag.Get() != false {
		t.Error("expected false initially")
	}

	flag.Toggle()
	if flag.Get() != true {
		t.Error("expected true after Toggle")
	}

	flag.Toggle()
	if flag.Get() != false {
		t.Error("expected false after second Toggle")
	}

	flag.SetTrue()
	if flag.Get() != true {
		t.Error("expected true after SetTrue")
	}

	flag.SetFalse()
	if flag.Get() != false {
		t.Error("expected false after SetFalse")
	}
}

func TestBoolSignalNotifications(t *testing.T) {
	flag := NewBoolSignal(false)
	listener := newTestListener()

	WithListener(listener, func() {
		_ = flag.Get()
	})

	flag.Toggle()
	if listener.getDirtyCount() != 1 {
		t.Errorf("expected 1 notification after Toggle, got %d", listener.getDirtyCount())
	}

	// SetFalse on already false should not notify
	flag.SetFalse()
	if listener.getDirtyCount() != 2 {
		t.Errorf("expected 2 notifications, got %d", listener.getDirtyCount())
	}
}

func TestSliceSignal(t *testing.T) {
	items := NewSliceSignal([]string{})

	if items.Len() != 0 {
		t.Errorf("expected empty slice, got len %d", items.Len())
	}

	items.Append("a")
	if items.Len() != 1 {
		t.Errorf("expected len 1 after Append, got %d", items.Len())
	}

	items.AppendAll("b", "c", "d")
	if items.Len() != 4 {
		t.Errorf("expected len 4 after AppendAll, got %d", items.Len())
	}

	items.RemoveAt(1) // Remove "b"
	if items.Len() != 3 {
		t.Errorf("expected len 3 after RemoveAt, got %d", items.Len())
	}

	// Verify order: should be [a, c, d]
	slice := items.Get()
	if slice[0] != "a" || slice[1] != "c" || slice[2] != "d" {
		t.Errorf("unexpected slice contents: %v", slice)
	}

	items.SetAt(1, "changed")
	slice = items.Get()
	if slice[1] != "changed" {
		t.Errorf("expected 'changed' at index 1, got %s", slice[1])
	}

	items.Clear()
	if items.Len() != 0 {
		t.Errorf("expected empty after Clear, got len %d", items.Len())
	}
}

func TestSliceSignalFilter(t *testing.T) {
	items := NewSliceSignal([]int{1, 2, 3, 4, 5, 6})

	items.Filter(func(n int) bool { return n%2 == 0 })

	slice := items.Get()
	if len(slice) != 3 {
		t.Errorf("expected 3 even numbers, got %d", len(slice))
	}
	if slice[0] != 2 || slice[1] != 4 || slice[2] != 6 {
		t.Errorf("unexpected filtered result: %v", slice)
	}
}

func TestSliceSignalBoundsCheck(t *testing.T) {
	items := NewSliceSignal([]string{"a", "b", "c"})

	// RemoveAt out of bounds should do nothing
	items.RemoveAt(-1)
	items.RemoveAt(100)
	if items.Len() != 3 {
		t.Errorf("out of bounds RemoveAt should not change length")
	}

	// SetAt out of bounds should do nothing
	items.SetAt(-1, "x")
	items.SetAt(100, "x")
	slice := items.Get()
	if slice[0] != "a" || slice[1] != "b" || slice[2] != "c" {
		t.Errorf("out of bounds SetAt should not change values")
	}
}

func TestSliceSignalNotifications(t *testing.T) {
	items := NewSliceSignal([]int{})
	listener := newTestListener()

	WithListener(listener, func() {
		_ = items.Get()
	})

	items.Append(1)
	if listener.getDirtyCount() != 1 {
		t.Errorf("expected 1 notification after Append, got %d", listener.getDirtyCount())
	}

	items.RemoveAt(0)
	if listener.getDirtyCount() != 2 {
		t.Errorf("expected 2 notifications, got %d", listener.getDirtyCount())
	}
}

func TestMapSignal(t *testing.T) {
	data := NewMapSignal[string, int](nil)

	if data.Len() != 0 {
		t.Errorf("expected empty map, got len %d", data.Len())
	}

	data.SetKey("a", 1)
	if data.Len() != 1 {
		t.Errorf("expected len 1 after SetKey, got %d", data.Len())
	}

	val, ok := data.GetKey("a")
	if !ok || val != 1 {
		t.Errorf("expected (1, true), got (%d, %v)", val, ok)
	}

	data.SetKey("b", 2)
	data.SetKey("c", 3)
	if data.Len() != 3 {
		t.Errorf("expected len 3, got %d", data.Len())
	}

	if !data.HasKey("b") {
		t.Error("expected HasKey('b') to be true")
	}
	if data.HasKey("x") {
		t.Error("expected HasKey('x') to be false")
	}

	data.DeleteKey("b")
	if data.HasKey("b") {
		t.Error("expected 'b' to be deleted")
	}
	if data.Len() != 2 {
		t.Errorf("expected len 2 after delete, got %d", data.Len())
	}

	data.Clear()
	if data.Len() != 0 {
		t.Errorf("expected empty after Clear, got len %d", data.Len())
	}
}

func TestMapSignalKeysValues(t *testing.T) {
	data := NewMapSignal(map[string]int{"a": 1, "b": 2})

	keys := data.Keys()
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}

	values := data.Values()
	if len(values) != 2 {
		t.Errorf("expected 2 values, got %d", len(values))
	}

	// Sum of values should be 3
	sum := 0
	for _, v := range values {
		sum += v
	}
	if sum != 3 {
		t.Errorf("expected sum 3, got %d", sum)
	}
}

func TestMapSignalNotifications(t *testing.T) {
	data := NewMapSignal[string, int](nil)
	listener := newTestListener()

	WithListener(listener, func() {
		_ = data.Get()
	})

	data.SetKey("a", 1)
	if listener.getDirtyCount() != 1 {
		t.Errorf("expected 1 notification after SetKey, got %d", listener.getDirtyCount())
	}

	data.DeleteKey("a")
	if listener.getDirtyCount() != 2 {
		t.Errorf("expected 2 notifications, got %d", listener.getDirtyCount())
	}

	// Delete non-existent key should not notify
	data.DeleteKey("x")
	if listener.getDirtyCount() != 2 {
		t.Errorf("deleting non-existent key should not notify, got %d", listener.getDirtyCount())
	}
}

func TestNilSliceSignal(t *testing.T) {
	// Creating with nil should give empty slice
	items := NewSliceSignal[int](nil)
	if items.Len() != 0 {
		t.Errorf("expected empty slice from nil, got len %d", items.Len())
	}
}

func TestNilMapSignal(t *testing.T) {
	// Creating with nil should give empty map
	data := NewMapSignal[string, int](nil)
	if data.Len() != 0 {
		t.Errorf("expected empty map from nil, got len %d", data.Len())
	}
}

// ============================================================================
// Additional tests for full coverage of Phase 1 typed signals
// ============================================================================

func TestIntSignalArithmetic(t *testing.T) {
	count := NewIntSignal(10)

	count.Sub(3)
	if count.Get() != 7 {
		t.Errorf("expected 7 after Sub(3), got %d", count.Get())
	}

	count.Mul(4)
	if count.Get() != 28 {
		t.Errorf("expected 28 after Mul(4), got %d", count.Get())
	}

	count.Div(7)
	if count.Get() != 4 {
		t.Errorf("expected 4 after Div(7), got %d", count.Get())
	}
}

func TestInt64SignalFull(t *testing.T) {
	count := NewInt64Signal(100)

	count.Dec()
	if count.Get() != 99 {
		t.Errorf("expected 99 after Dec, got %d", count.Get())
	}

	count.Sub(9)
	if count.Get() != 90 {
		t.Errorf("expected 90 after Sub(9), got %d", count.Get())
	}

	count.Mul(2)
	if count.Get() != 180 {
		t.Errorf("expected 180 after Mul(2), got %d", count.Get())
	}

	count.Div(3)
	if count.Get() != 60 {
		t.Errorf("expected 60 after Div(3), got %d", count.Get())
	}
}

func TestFloat64SignalFull(t *testing.T) {
	value := NewFloat64Signal(10.0)

	value.Sub(2.5)
	if value.Get() != 7.5 {
		t.Errorf("expected 7.5 after Sub(2.5), got %f", value.Get())
	}

	value.Mul(2.0)
	if value.Get() != 15.0 {
		t.Errorf("expected 15.0 after Mul(2.0), got %f", value.Get())
	}

	value.Div(3.0)
	if value.Get() != 5.0 {
		t.Errorf("expected 5.0 after Div(3.0), got %f", value.Get())
	}
}

func TestSliceSignalPrepend(t *testing.T) {
	items := NewSliceSignal([]string{"b", "c"})

	items.Prepend("a")
	slice := items.Get()
	if len(slice) != 3 || slice[0] != "a" || slice[1] != "b" || slice[2] != "c" {
		t.Errorf("unexpected slice after Prepend: %v", slice)
	}
}

func TestSliceSignalInsertAt(t *testing.T) {
	items := NewSliceSignal([]string{"a", "c"})

	items.InsertAt(1, "b")
	slice := items.Get()
	if len(slice) != 3 || slice[0] != "a" || slice[1] != "b" || slice[2] != "c" {
		t.Errorf("unexpected slice after InsertAt: %v", slice)
	}

	// InsertAt with negative index prepends
	items.InsertAt(-5, "first")
	slice = items.Get()
	if slice[0] != "first" {
		t.Errorf("expected 'first' at index 0, got %v", slice)
	}

	// InsertAt beyond length appends
	items.InsertAt(100, "last")
	slice = items.Get()
	if slice[len(slice)-1] != "last" {
		t.Errorf("expected 'last' at end, got %v", slice)
	}
}

func TestSliceSignalRemoveFirst(t *testing.T) {
	items := NewSliceSignal([]string{"a", "b", "c"})

	items.RemoveFirst()
	slice := items.Get()
	if len(slice) != 2 || slice[0] != "b" {
		t.Errorf("unexpected slice after RemoveFirst: %v", slice)
	}

	// RemoveFirst on empty slice does nothing
	empty := NewSliceSignal([]int{})
	empty.RemoveFirst()
	if empty.Len() != 0 {
		t.Error("RemoveFirst on empty should do nothing")
	}
}

func TestSliceSignalRemoveLast(t *testing.T) {
	items := NewSliceSignal([]string{"a", "b", "c"})

	items.RemoveLast()
	slice := items.Get()
	if len(slice) != 2 || slice[1] != "b" {
		t.Errorf("unexpected slice after RemoveLast: %v", slice)
	}

	// RemoveLast on empty slice does nothing
	empty := NewSliceSignal([]int{})
	empty.RemoveLast()
	if empty.Len() != 0 {
		t.Error("RemoveLast on empty should do nothing")
	}
}

func TestSliceSignalRemoveWhere(t *testing.T) {
	items := NewSliceSignal([]int{1, 2, 3, 4, 5})

	items.RemoveWhere(func(n int) bool { return n%2 == 0 })
	slice := items.Get()
	if len(slice) != 3 {
		t.Errorf("expected 3 odd numbers, got %d", len(slice))
	}
	if slice[0] != 1 || slice[1] != 3 || slice[2] != 5 {
		t.Errorf("unexpected slice: %v", slice)
	}
}

func TestSliceSignalUpdateAt(t *testing.T) {
	items := NewSliceSignal([]int{1, 2, 3})

	items.UpdateAt(1, func(n int) int { return n * 10 })
	slice := items.Get()
	if slice[1] != 20 {
		t.Errorf("expected 20 at index 1, got %d", slice[1])
	}

	// UpdateAt out of bounds does nothing
	items.UpdateAt(-1, func(n int) int { return 999 })
	items.UpdateAt(100, func(n int) int { return 999 })
	slice = items.Get()
	if slice[0] != 1 || slice[1] != 20 || slice[2] != 3 {
		t.Errorf("out of bounds UpdateAt should not change values: %v", slice)
	}
}

func TestSliceSignalUpdateWhere(t *testing.T) {
	items := NewSliceSignal([]int{1, 2, 3, 4, 5})

	items.UpdateWhere(
		func(n int) bool { return n%2 == 0 },
		func(n int) int { return n * 10 },
	)
	slice := items.Get()
	if slice[1] != 20 || slice[3] != 40 {
		t.Errorf("expected even numbers multiplied by 10: %v", slice)
	}
	if slice[0] != 1 || slice[2] != 3 || slice[4] != 5 {
		t.Errorf("odd numbers should be unchanged: %v", slice)
	}
}

func TestMapSignalRemoveKey(t *testing.T) {
	data := NewMapSignal(map[string]int{"a": 1, "b": 2})

	data.RemoveKey("a")
	if data.HasKey("a") {
		t.Error("expected 'a' to be removed")
	}
	if data.Len() != 1 {
		t.Errorf("expected len 1, got %d", data.Len())
	}

	// RemoveKey on non-existent key does nothing
	listener := newTestListener()
	WithListener(listener, func() {
		_ = data.Get()
	})
	data.RemoveKey("nonexistent")
	if listener.getDirtyCount() != 0 {
		t.Error("RemoveKey on non-existent key should not notify")
	}
}

func TestMapSignalUpdateKey(t *testing.T) {
	data := NewMapSignal(map[string]int{"a": 1, "b": 2})

	data.UpdateKey("a", func(v int) int { return v * 10 })
	val, _ := data.GetKey("a")
	if val != 10 {
		t.Errorf("expected 10 after UpdateKey, got %d", val)
	}

	// UpdateKey on non-existent key does nothing
	listener := newTestListener()
	WithListener(listener, func() {
		_ = data.Get()
	})
	data.UpdateKey("nonexistent", func(v int) int { return v * 10 })
	if listener.getDirtyCount() != 0 {
		t.Error("UpdateKey on non-existent key should not notify")
	}
}

func TestStringSignal(t *testing.T) {
	str := NewStringSignal("hello")

	if str.Get() != "hello" {
		t.Errorf("expected 'hello', got %s", str.Get())
	}

	str.Append(" world")
	if str.Get() != "hello world" {
		t.Errorf("expected 'hello world' after Append, got %s", str.Get())
	}

	str.Prepend("say: ")
	if str.Get() != "say: hello world" {
		t.Errorf("expected 'say: hello world' after Prepend, got %s", str.Get())
	}

	if str.Len() != 16 {
		t.Errorf("expected len 16, got %d", str.Len())
	}

	if str.IsEmpty() {
		t.Error("expected IsEmpty to be false")
	}

	str.Clear()
	if str.Get() != "" {
		t.Errorf("expected empty string after Clear, got %s", str.Get())
	}

	if !str.IsEmpty() {
		t.Error("expected IsEmpty to be true after Clear")
	}
}

func TestStringSignalNotifications(t *testing.T) {
	str := NewStringSignal("")
	listener := newTestListener()

	WithListener(listener, func() {
		_ = str.Get()
	})

	str.Append("a")
	if listener.getDirtyCount() != 1 {
		t.Errorf("expected 1 notification after Append, got %d", listener.getDirtyCount())
	}

	str.Prepend("b")
	if listener.getDirtyCount() != 2 {
		t.Errorf("expected 2 notifications, got %d", listener.getDirtyCount())
	}

	// Setting to same value should not notify
	str.Set(str.Peek())
	if listener.getDirtyCount() != 2 {
		t.Errorf("setting same value should not notify, got %d", listener.getDirtyCount())
	}
}
