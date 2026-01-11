package vango

import (
	"reflect"
	"testing"
)

func TestSignalBase_GetIDAndNumericOps(t *testing.T) {
	s := NewSignal(10)
	if id := s.base.getID(); id == 0 {
		t.Fatalf("signalBase.getID() should be non-zero")
	}

	s.Inc()
	s.Dec()
	if got := s.Get(); got != 10 {
		t.Fatalf("Inc then Dec should return to 10, got %d", got)
	}

	s.Add(5)
	s.Sub(3)
	s.Mul(2)
	s.Div(4)
	if got := s.Get(); got != 6 { // ((10+5-3)*2)/4 = 6
		t.Fatalf("numeric ops result = %d, want %d", got, 6)
	}
}

func TestSignalSliceOps_UpdateWhere_Filter_SetAt_UpdateAt_RemoveAt(t *testing.T) {
	s := NewSignal([]int{1, 2, 3, 4, 5})

	s.UpdateWhere(
		func(v any) bool { return v.(int)%2 == 0 },
		func(v any) any { return v.(int) * 10 },
	)
	if got := s.Get(); !reflect.DeepEqual(got, []int{1, 20, 3, 40, 5}) {
		t.Fatalf("UpdateWhere result = %v, want %v", got, []int{1, 20, 3, 40, 5})
	}

	s.Filter(func(v any) bool { return v.(int) >= 20 })
	if got := s.Get(); !reflect.DeepEqual(got, []int{20, 40}) {
		t.Fatalf("Filter result = %v, want %v", got, []int{20, 40})
	}

	s.SetAt(1, 41)
	s.UpdateAt(0, func(v any) any { return v.(int) + 1 })
	if got := s.Get(); !reflect.DeepEqual(got, []int{21, 41}) {
		t.Fatalf("SetAt/UpdateAt result = %v, want %v", got, []int{21, 41})
	}

	s.RemoveAt(0)
	if got := s.Get(); !reflect.DeepEqual(got, []int{41}) {
		t.Fatalf("RemoveAt result = %v, want %v", got, []int{41})
	}
}

