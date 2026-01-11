package vango

import "testing"

func TestNumericSignals_AllSupportedTypes(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		s := NewSignal(int(10))
		s.Inc()
		s.Dec()
		s.Add(int(5))
		s.Sub(int(3))
		s.Mul(int(2))
		s.Div(int(4))
		if got := s.Get(); got != 6 {
			t.Fatalf("got %d, want %d", got, 6)
		}
	})

	t.Run("int8", func(t *testing.T) {
		s := NewSignal(int8(10))
		s.Inc()
		s.Dec()
		s.Add(int8(5))
		s.Sub(int8(3))
		s.Mul(int8(2))
		s.Div(int8(4))
		if got := s.Get(); got != int8(6) {
			t.Fatalf("got %d, want %d", got, 6)
		}
	})

	t.Run("int16", func(t *testing.T) {
		s := NewSignal(int16(10))
		s.Inc()
		s.Dec()
		s.Add(int16(5))
		s.Sub(int16(3))
		s.Mul(int16(2))
		s.Div(int16(4))
		if got := s.Get(); got != int16(6) {
			t.Fatalf("got %d, want %d", got, 6)
		}
	})

	t.Run("int32", func(t *testing.T) {
		s := NewSignal(int32(10))
		s.Inc()
		s.Dec()
		s.Add(int32(5))
		s.Sub(int32(3))
		s.Mul(int32(2))
		s.Div(int32(4))
		if got := s.Get(); got != int32(6) {
			t.Fatalf("got %d, want %d", got, 6)
		}
	})

	t.Run("int64", func(t *testing.T) {
		s := NewSignal(int64(10))
		s.Inc()
		s.Dec()
		s.Add(int64(5))
		s.Sub(int64(3))
		s.Mul(int64(2))
		s.Div(int64(4))
		if got := s.Get(); got != int64(6) {
			t.Fatalf("got %d, want %d", got, 6)
		}
	})

	t.Run("uint", func(t *testing.T) {
		s := NewSignal(uint(10))
		s.Inc()
		s.Dec()
		s.Add(uint(5))
		s.Sub(uint(3))
		s.Mul(uint(2))
		s.Div(uint(4))
		if got := s.Get(); got != uint(6) {
			t.Fatalf("got %d, want %d", got, 6)
		}
	})

	t.Run("uint8", func(t *testing.T) {
		s := NewSignal(uint8(10))
		s.Inc()
		s.Dec()
		s.Add(uint8(5))
		s.Sub(uint8(3))
		s.Mul(uint8(2))
		s.Div(uint8(4))
		if got := s.Get(); got != uint8(6) {
			t.Fatalf("got %d, want %d", got, 6)
		}
	})

	t.Run("uint16", func(t *testing.T) {
		s := NewSignal(uint16(10))
		s.Inc()
		s.Dec()
		s.Add(uint16(5))
		s.Sub(uint16(3))
		s.Mul(uint16(2))
		s.Div(uint16(4))
		if got := s.Get(); got != uint16(6) {
			t.Fatalf("got %d, want %d", got, 6)
		}
	})

	t.Run("uint32", func(t *testing.T) {
		s := NewSignal(uint32(10))
		s.Inc()
		s.Dec()
		s.Add(uint32(5))
		s.Sub(uint32(3))
		s.Mul(uint32(2))
		s.Div(uint32(4))
		if got := s.Get(); got != uint32(6) {
			t.Fatalf("got %d, want %d", got, 6)
		}
	})

	t.Run("uint64", func(t *testing.T) {
		s := NewSignal(uint64(10))
		s.Inc()
		s.Dec()
		s.Add(uint64(5))
		s.Sub(uint64(3))
		s.Mul(uint64(2))
		s.Div(uint64(4))
		if got := s.Get(); got != uint64(6) {
			t.Fatalf("got %d, want %d", got, 6)
		}
	})

	t.Run("float32", func(t *testing.T) {
		s := NewSignal(float32(10))
		s.Inc()
		s.Dec()
		s.Add(float32(5))
		s.Sub(float32(3))
		s.Mul(float32(2))
		s.Div(float32(4))
		if got := s.Get(); got != float32(6) {
			t.Fatalf("got %v, want %v", got, float32(6))
		}
	})

	t.Run("float64", func(t *testing.T) {
		s := NewSignal(float64(10))
		s.Inc()
		s.Dec()
		s.Add(float64(5))
		s.Sub(float64(3))
		s.Mul(float64(2))
		s.Div(float64(4))
		if got := s.Get(); got != float64(6) {
			t.Fatalf("got %v, want %v", got, float64(6))
		}
	})
}

func TestDefaultEqualsAndMemoDefaultEquals_CoverAllCases(t *testing.T) {
	if !defaultEquals(int(1), int(1)) || defaultEquals(int(1), int(2)) {
		t.Fatalf("defaultEquals(int) unexpected")
	}
	if !defaultEquals(int8(1), int8(1)) || defaultEquals(int8(1), int8(2)) {
		t.Fatalf("defaultEquals(int8) unexpected")
	}
	if !defaultEquals(int16(1), int16(1)) || defaultEquals(int16(1), int16(2)) {
		t.Fatalf("defaultEquals(int16) unexpected")
	}
	if !defaultEquals(int32(1), int32(1)) || defaultEquals(int32(1), int32(2)) {
		t.Fatalf("defaultEquals(int32) unexpected")
	}
	if !defaultEquals(int64(1), int64(1)) || defaultEquals(int64(1), int64(2)) {
		t.Fatalf("defaultEquals(int64) unexpected")
	}
	if !defaultEquals(uint(1), uint(1)) || defaultEquals(uint(1), uint(2)) {
		t.Fatalf("defaultEquals(uint) unexpected")
	}
	if !defaultEquals(uint8(1), uint8(1)) || defaultEquals(uint8(1), uint8(2)) {
		t.Fatalf("defaultEquals(uint8) unexpected")
	}
	if !defaultEquals(uint16(1), uint16(1)) || defaultEquals(uint16(1), uint16(2)) {
		t.Fatalf("defaultEquals(uint16) unexpected")
	}
	if !defaultEquals(uint32(1), uint32(1)) || defaultEquals(uint32(1), uint32(2)) {
		t.Fatalf("defaultEquals(uint32) unexpected")
	}
	if !defaultEquals(uint64(1), uint64(1)) || defaultEquals(uint64(1), uint64(2)) {
		t.Fatalf("defaultEquals(uint64) unexpected")
	}
	if !defaultEquals(float32(1.5), float32(1.5)) || defaultEquals(float32(1.5), float32(1.6)) {
		t.Fatalf("defaultEquals(float32) unexpected")
	}
	if !defaultEquals(float64(1.5), float64(1.5)) || defaultEquals(float64(1.5), float64(1.6)) {
		t.Fatalf("defaultEquals(float64) unexpected")
	}
	if !defaultEquals("a", "a") || defaultEquals("a", "b") {
		t.Fatalf("defaultEquals(string) unexpected")
	}
	if !defaultEquals(true, true) || defaultEquals(true, false) {
		t.Fatalf("defaultEquals(bool) unexpected")
	}
	if !defaultEquals([]int{1, 2}, []int{1, 2}) || defaultEquals([]int{1, 2}, []int{1}) {
		t.Fatalf("defaultEquals(slice) unexpected")
	}

	if !memoDefaultEquals(int(1), int(1)) || memoDefaultEquals(int(1), int(2)) {
		t.Fatalf("memoDefaultEquals(int) unexpected")
	}
	if !memoDefaultEquals(int8(1), int8(1)) || memoDefaultEquals(int8(1), int8(2)) {
		t.Fatalf("memoDefaultEquals(int8) unexpected")
	}
	if !memoDefaultEquals(int16(1), int16(1)) || memoDefaultEquals(int16(1), int16(2)) {
		t.Fatalf("memoDefaultEquals(int16) unexpected")
	}
	if !memoDefaultEquals(int32(1), int32(1)) || memoDefaultEquals(int32(1), int32(2)) {
		t.Fatalf("memoDefaultEquals(int32) unexpected")
	}
	if !memoDefaultEquals(int64(1), int64(1)) || memoDefaultEquals(int64(1), int64(2)) {
		t.Fatalf("memoDefaultEquals(int64) unexpected")
	}
	if !memoDefaultEquals(uint(1), uint(1)) || memoDefaultEquals(uint(1), uint(2)) {
		t.Fatalf("memoDefaultEquals(uint) unexpected")
	}
	if !memoDefaultEquals(uint8(1), uint8(1)) || memoDefaultEquals(uint8(1), uint8(2)) {
		t.Fatalf("memoDefaultEquals(uint8) unexpected")
	}
	if !memoDefaultEquals(uint16(1), uint16(1)) || memoDefaultEquals(uint16(1), uint16(2)) {
		t.Fatalf("memoDefaultEquals(uint16) unexpected")
	}
	if !memoDefaultEquals(uint32(1), uint32(1)) || memoDefaultEquals(uint32(1), uint32(2)) {
		t.Fatalf("memoDefaultEquals(uint32) unexpected")
	}
	if !memoDefaultEquals(uint64(1), uint64(1)) || memoDefaultEquals(uint64(1), uint64(2)) {
		t.Fatalf("memoDefaultEquals(uint64) unexpected")
	}
	if !memoDefaultEquals(float32(1.5), float32(1.5)) || memoDefaultEquals(float32(1.5), float32(1.6)) {
		t.Fatalf("memoDefaultEquals(float32) unexpected")
	}
	if !memoDefaultEquals(float64(1.5), float64(1.5)) || memoDefaultEquals(float64(1.5), float64(1.6)) {
		t.Fatalf("memoDefaultEquals(float64) unexpected")
	}
	if !memoDefaultEquals("a", "a") || memoDefaultEquals("a", "b") {
		t.Fatalf("memoDefaultEquals(string) unexpected")
	}
	if !memoDefaultEquals(true, true) || memoDefaultEquals(true, false) {
		t.Fatalf("memoDefaultEquals(bool) unexpected")
	}
	if !memoDefaultEquals([]int{1, 2}, []int{1, 2}) || memoDefaultEquals([]int{1, 2}, []int{1}) {
		t.Fatalf("memoDefaultEquals(slice) unexpected")
	}
}

