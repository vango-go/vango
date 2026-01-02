package vango

// IntSignal wraps Signal[int] with convenience methods for integer operations.
type IntSignal struct {
	*Signal[int]
}

// NewIntSignal creates a new IntSignal with the given initial value.
func NewIntSignal(initial int) *IntSignal {
	return &IntSignal{NewSignal(initial)}
}

// Inc increments the value by 1.
func (s *IntSignal) Inc() {
	s.Update(func(n int) int { return n + 1 })
}

// Dec decrements the value by 1.
func (s *IntSignal) Dec() {
	s.Update(func(n int) int { return n - 1 })
}

// Add adds the given value.
func (s *IntSignal) Add(n int) {
	s.Update(func(v int) int { return v + n })
}

// Sub subtracts the given value.
func (s *IntSignal) Sub(n int) {
	s.Update(func(v int) int { return v - n })
}

// Mul multiplies by the given value.
func (s *IntSignal) Mul(n int) {
	s.Update(func(v int) int { return v * n })
}

// Div divides by the given value.
// Note: Integer division truncates toward zero.
func (s *IntSignal) Div(n int) {
	s.Update(func(v int) int { return v / n })
}

// Int64Signal wraps Signal[int64] with convenience methods for integer operations.
type Int64Signal struct {
	*Signal[int64]
}

// NewInt64Signal creates a new Int64Signal with the given initial value.
func NewInt64Signal(initial int64) *Int64Signal {
	return &Int64Signal{NewSignal(initial)}
}

// Inc increments the value by 1.
func (s *Int64Signal) Inc() {
	s.Update(func(n int64) int64 { return n + 1 })
}

// Dec decrements the value by 1.
func (s *Int64Signal) Dec() {
	s.Update(func(n int64) int64 { return n - 1 })
}

// Add adds the given value.
func (s *Int64Signal) Add(n int64) {
	s.Update(func(v int64) int64 { return v + n })
}

// Sub subtracts the given value.
func (s *Int64Signal) Sub(n int64) {
	s.Update(func(v int64) int64 { return v - n })
}

// Mul multiplies by the given value.
func (s *Int64Signal) Mul(n int64) {
	s.Update(func(v int64) int64 { return v * n })
}

// Div divides by the given value.
// Note: Integer division truncates toward zero.
func (s *Int64Signal) Div(n int64) {
	s.Update(func(v int64) int64 { return v / n })
}

// Float64Signal wraps Signal[float64] with convenience methods for float operations.
type Float64Signal struct {
	*Signal[float64]
}

// NewFloat64Signal creates a new Float64Signal with the given initial value.
func NewFloat64Signal(initial float64) *Float64Signal {
	return &Float64Signal{NewSignal(initial)}
}

// Add adds the given value.
func (s *Float64Signal) Add(n float64) {
	s.Update(func(v float64) float64 { return v + n })
}

// Sub subtracts the given value.
func (s *Float64Signal) Sub(n float64) {
	s.Update(func(v float64) float64 { return v - n })
}

// Mul multiplies by the given value.
func (s *Float64Signal) Mul(n float64) {
	s.Update(func(v float64) float64 { return v * n })
}

// Multiply is an alias for Mul. Deprecated: use Mul instead.
func (s *Float64Signal) Multiply(n float64) {
	s.Mul(n)
}

// Div divides by the given value.
func (s *Float64Signal) Div(n float64) {
	s.Update(func(v float64) float64 { return v / n })
}
