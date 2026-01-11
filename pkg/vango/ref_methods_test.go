package vango

import "testing"

func TestRefMethods_Current_Set_IsSet_Clear(t *testing.T) {
	r := NewRef[int](0)
	if r.IsSet() {
		t.Fatalf("IsSet() should be false initially")
	}
	if got := r.Current(); got != 0 {
		t.Fatalf("Current() = %d, want %d", got, 0)
	}

	r.Set(42)
	if !r.IsSet() {
		t.Fatalf("IsSet() should be true after Set()")
	}
	if got := r.Current(); got != 42 {
		t.Fatalf("Current() after Set = %d, want %d", got, 42)
	}

	r.Clear()
	if r.IsSet() {
		t.Fatalf("IsSet() should be false after Clear()")
	}
	if got := r.Current(); got != 0 {
		t.Fatalf("Current() after Clear = %d, want %d", got, 0)
	}
}

