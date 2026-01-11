package vango

import "testing"

func TestSignalPersistenceOptionsAndSetAny(t *testing.T) {
	s := NewSignal(123, Transient(), PersistKey("user_id"))

	if !s.IsTransient() {
		t.Fatalf("IsTransient() = false, want true")
	}
	if got := s.PersistKey(); got != "user_id" {
		t.Fatalf("PersistKey() = %q, want %q", got, "user_id")
	}
	if got := s.GetAny(); got != 123 {
		t.Fatalf("GetAny() = %v, want %v", got, 123)
	}

	if err := s.SetAny(456); err != nil {
		t.Fatalf("SetAny(correct type) error: %v", err)
	}
	if got := s.Get(); got != 456 {
		t.Fatalf("Get() after SetAny = %d, want %d", got, 456)
	}

	err := s.SetAny("nope")
	if err == nil {
		t.Fatalf("SetAny(wrong type) expected error")
	}
	if _, ok := err.(*TypeMismatchError); !ok {
		t.Fatalf("SetAny(wrong type) error type = %T, want *TypeMismatchError", err)
	}
	if got := err.Error(); got == "" {
		t.Fatalf("TypeMismatchError.Error() should be non-empty")
	}
}

