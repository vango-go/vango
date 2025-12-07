package vango

// BoolSignal wraps Signal[bool] with convenience methods for boolean operations.
type BoolSignal struct {
	*Signal[bool]
}

// NewBoolSignal creates a new BoolSignal with the given initial value.
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
