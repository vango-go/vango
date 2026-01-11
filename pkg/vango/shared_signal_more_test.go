package vango

import "testing"

func TestSharedSignalDef_OutsideSessionContextUsesInitialAndNoops(t *testing.T) {
	count := NewSharedSignal(5)
	if got := count.Get(); got != 5 {
		t.Fatalf("Get() outside session = %d, want %d", got, 5)
	}
	if got := count.Peek(); got != 5 {
		t.Fatalf("Peek() outside session = %d, want %d", got, 5)
	}
	if got := count.Len(); got != 0 {
		t.Fatalf("Len() outside session = %d, want %d", got, 0)
	}

	count.Inc()
	if got := count.Get(); got != 5 {
		t.Fatalf("Inc() outside session should be no-op; Get() = %d, want %d", got, 5)
	}
}

func TestSharedSignalDef_SignalCreationAndProxyMethods(t *testing.T) {
	owner := NewOwner(nil)
	store := NewSimpleSessionSignalStore()

	n := NewSharedSignal(0)
	b := NewSharedSignal(false)
	s := NewSharedSignal("a")
	sl := NewSharedSignal([]int{1})
	m := NewSharedSignal(map[string]int{"x": 1})

	WithOwner(owner, func() {
		SetContext(SessionSignalStoreKey, store)

		n.Inc()
		n.Dec()
		n.Add(10)
		n.Sub(1)
		n.Mul(2)
		n.Div(2)
		n.Update(func(v int) int { return v + 2 })
		if got := n.Get(); got != 11 {
			t.Fatalf("numeric shared signal = %d, want %d", got, 11)
		}

		b.Toggle()
		b.SetFalse()
		b.SetTrue()
		if got := b.Get(); got != true {
			t.Fatalf("bool shared signal = %v, want %v", got, true)
		}

		s.Append("b")
		s.Prepend("z")
		if got := s.Get(); got != "zab" {
			t.Fatalf("string shared signal = %q, want %q", got, "zab")
		}

		sl.AppendItem(2)
		sl.PrependItem(0)
		sl.InsertAt(1, 99)
		sl.SetAt(1, 98)
		sl.UpdateAt(1, func(v any) any { return v.(int) + 1 })
		sl.UpdateWhere(func(v any) bool { return v.(int) == 99 }, func(v any) any { return 100 })
		sl.Filter(func(v any) bool { return v.(int) != 100 })
		sl.RemoveWhere(func(v any) bool { return v.(int) == 99 })
		if got := sl.Len(); got != 3 {
			t.Fatalf("slice Len() = %d, want %d", got, 3)
		}
		sl.RemoveFirst()
		sl.RemoveLast()
		sl.InsertAt(1, 9)
		sl.RemoveAt(1)
		if got := sl.Get(); len(got) != 1 || got[0] != 1 {
			t.Fatalf("slice after removes = %v, want %v", got, []int{1})
		}

		m.SetKey("y", 2)
		if !m.HasKey("y") {
			t.Fatalf("map HasKey(y) = false, want true")
		}
		m.UpdateKey("y", func(v any) any { return v.(int) + 1 })
		m.RemoveKey("x")
		if got := m.Get(); got["y"] != 3 {
			t.Fatalf("map[y] = %d, want %d", got["y"], 3)
		}

		m.Clear()
		if got := m.Len(); got != 0 {
			t.Fatalf("Clear() should empty map; Len() = %d, want %d", got, 0)
		}
	})
}

func TestSharedSignalDef_SignalNilWhenNoStoreOrWrongType(t *testing.T) {
	def := NewSharedSignal(1)

	owner1 := NewOwner(nil)
	WithOwner(owner1, func() {
		if sig := def.Signal(); sig != nil {
			t.Fatalf("Signal() should be nil when SessionSignalStoreKey not set")
		}
		if got := def.Get(); got != 1 {
			t.Fatalf("Get() without store = %d, want %d", got, 1)
		}
	})

	owner2 := NewOwner(nil)
	WithOwner(owner2, func() {
		SetContext(SessionSignalStoreKey, "not-a-store")
		if sig := def.Signal(); sig != nil {
			t.Fatalf("Signal() should be nil when SessionSignalStoreKey has wrong type")
		}
	})
}
