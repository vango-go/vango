package vango

// StringSignal wraps Signal[string] with convenience methods for string operations.
//
// Deprecated: Use NewSignal[string] instead. Signal[T] now has Append(), Prepend(),
// Clear(), and Len() methods directly available.
//
//	// Old:
//	text := vango.NewStringSignal("")
//	text.Append("hello")
//
//	// New:
//	text := vango.NewSignal("")
//	text.Append("hello")
type StringSignal struct {
	*Signal[string]
}

// NewStringSignal creates a new StringSignal with the given initial value.
//
// Deprecated: Use NewSignal[string] instead. Signal[T] now has Append(), Prepend(),
// Clear(), and Len() methods directly available.
func NewStringSignal(initial string) *StringSignal {
	return &StringSignal{NewSignal(initial)}
}

// Append adds the given string to the end.
func (s *StringSignal) Append(suffix string) {
	s.Update(func(v string) string { return v + suffix })
}

// Prepend adds the given string to the beginning.
func (s *StringSignal) Prepend(prefix string) {
	s.Update(func(v string) string { return prefix + v })
}

// Clear sets the value to an empty string.
func (s *StringSignal) Clear() {
	s.Set("")
}

// Len returns the length of the string.
// This reads the signal and creates a dependency.
func (s *StringSignal) Len() int {
	return len(s.Get())
}

// IsEmpty returns true if the string is empty.
// This reads the signal and creates a dependency.
func (s *StringSignal) IsEmpty() bool {
	return s.Get() == ""
}
