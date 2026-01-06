package vango

// BoolSignal wraps Signal[bool] with convenience methods for boolean operations.
//
// Deprecated: Use NewSignal[bool] instead. Signal[T] now has Toggle(), SetTrue(),
// and SetFalse() methods directly available.
//
//	// Old:
//	visible := vango.NewBoolSignal(false)
//	visible.Toggle()
//
//	// New:
//	visible := vango.NewSignal(false)
//	visible.Toggle()
type BoolSignal struct {
	*Signal[bool]
}

// NewBoolSignal creates a new BoolSignal with the given initial value.
//
// Deprecated: Use NewSignal[bool] instead. Signal[T] now has Toggle(), SetTrue(),
// and SetFalse() methods directly available.
func NewBoolSignal(initial bool) *BoolSignal {
	return &BoolSignal{NewSignal(initial)}
}

// Toggle inverts the boolean value.
func (s *BoolSignal) Toggle() {
	s.Update(func(b bool) bool { return !b })
}

// SetTrue sets the value to true.
func (s *BoolSignal) SetTrue() {
	s.Set(true)
}

// SetFalse sets the value to false.
func (s *BoolSignal) SetFalse() {
	s.Set(false)
}
