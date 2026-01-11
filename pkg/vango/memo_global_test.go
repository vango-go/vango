package vango

import "testing"

func TestNewGlobalMemo_RecomputesOnDependencyChange(t *testing.T) {
	count := NewGlobalSignal(2)
	tripled := NewGlobalMemo(func() int {
		return count.Get() * 3
	})

	if got := tripled.Get(); got != 6 {
		t.Fatalf("initial tripled.Get() = %d, want 6", got)
	}

	count.Set(4)
	if got := tripled.Get(); got != 12 {
		t.Fatalf("after Set(4) tripled.Get() = %d, want 12", got)
	}
}

