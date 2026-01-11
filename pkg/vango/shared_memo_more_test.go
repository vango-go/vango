package vango

import (
	"sync/atomic"
	"testing"
)

func TestSharedMemoDef_PeekAndWithEquals(t *testing.T) {
	var computeCalls atomic.Int32
	dep := NewSignal(1)
	m := NewSharedMemo(func() int {
		computeCalls.Add(1)
		return dep.Get()
	})

	// Outside session context: computes directly.
	if got := m.Peek(); got != 1 {
		t.Fatalf("Peek() outside session = %d, want %d", got, 1)
	}
	if computeCalls.Load() != 1 {
		t.Fatalf("compute calls = %d, want %d", computeCalls.Load(), 1)
	}

	owner := NewOwner(nil)
	store := NewSimpleSessionSignalStore()
	WithOwner(owner, func() {
		SetContext(SessionSignalStoreKey, store)

		if got := m.Get(); got != 1 {
			t.Fatalf("Get() in session = %d, want %d", got, 1)
		}
		if got := m.Peek(); got != 1 {
			t.Fatalf("Peek() in session = %d, want %d", got, 1)
		}

		m.WithEquals(func(a, b int) bool { return a == b })
		memo := m.Memo()
		if memo == nil || memo.equal == nil {
			t.Fatalf("WithEquals should configure existing memo instance")
		}
	})
}

func TestGlobalMemo_WithEqualsChaining(t *testing.T) {
	m := NewGlobalMemo(func() int { return 1 })
	if got := m.Get(); got != 1 {
		t.Fatalf("Get() = %d, want %d", got, 1)
	}
	if m2 := m.WithEquals(func(a, b int) bool { return a == b }); m2 != m {
		t.Fatalf("WithEquals should return same GlobalMemo for chaining")
	}
}

