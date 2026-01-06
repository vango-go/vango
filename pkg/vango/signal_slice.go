package vango

// SliceSignal wraps Signal[[]T] with convenience methods for slice operations.
//
// Deprecated: Use NewSignal[[]T] instead. Signal[T] now has AppendItem(), PrependItem(),
// InsertAt(), RemoveAt(), SetAt(), UpdateAt(), RemoveWhere(), UpdateWhere(), Filter(),
// Clear(), and Len() methods directly available.
//
// Note: SliceSignal[T] provides type-safe methods (e.g., Append(item T)) while the
// Signal[T] methods use any (e.g., AppendItem(item any)). Use SliceSignal[T] if you
// prefer compile-time type safety over the unified API.
//
//	// Old:
//	items := vango.NewSliceSignal([]Item{})
//	items.Append(newItem)
//
//	// New:
//	items := vango.NewSignal([]Item{})
//	items.AppendItem(newItem)  // Note: uses 'any' parameter
type SliceSignal[T any] struct {
	*Signal[[]T]
}

// NewSliceSignal creates a new SliceSignal with the given initial value.
// If initial is nil, creates an empty slice.
//
// Deprecated: Use NewSignal[[]T] instead. Signal[T] now has slice convenience methods.
// SliceSignal[T] is retained for type-safe slice operations if preferred.
func NewSliceSignal[T any](initial []T) *SliceSignal[T] {
	if initial == nil {
		initial = []T{}
	}
	return &SliceSignal[T]{NewSignal(initial)}
}

// Append adds an item to the end of the slice.
func (s *SliceSignal[T]) Append(item T) {
	s.Update(func(items []T) []T {
		return append(items, item)
	})
}

// AppendAll adds multiple items to the end of the slice.
func (s *SliceSignal[T]) AppendAll(items ...T) {
	s.Update(func(current []T) []T {
		return append(current, items...)
	})
}

// RemoveAt removes the item at the given index.
// Does nothing if index is out of bounds.
func (s *SliceSignal[T]) RemoveAt(index int) {
	s.Update(func(items []T) []T {
		if index < 0 || index >= len(items) {
			return items
		}
		return append(items[:index], items[index+1:]...)
	})
}

// Clear removes all items from the slice.
func (s *SliceSignal[T]) Clear() {
	s.Set([]T{})
}

// Len returns the length of the slice.
// This reads the signal and creates a dependency.
func (s *SliceSignal[T]) Len() int {
	return len(s.Get())
}

// SetAt sets the item at the given index.
// Does nothing if index is out of bounds.
func (s *SliceSignal[T]) SetAt(index int, item T) {
	s.Update(func(items []T) []T {
		if index < 0 || index >= len(items) {
			return items
		}
		// Create a copy to avoid modifying the original
		newItems := make([]T, len(items))
		copy(newItems, items)
		newItems[index] = item
		return newItems
	})
}

// Filter keeps only items that satisfy the predicate.
func (s *SliceSignal[T]) Filter(predicate func(T) bool) {
	s.Update(func(items []T) []T {
		result := make([]T, 0, len(items))
		for _, item := range items {
			if predicate(item) {
				result = append(result, item)
			}
		}
		return result
	})
}

// Prepend adds an item to the beginning of the slice.
func (s *SliceSignal[T]) Prepend(item T) {
	s.Update(func(items []T) []T {
		result := make([]T, 0, len(items)+1)
		result = append(result, item)
		result = append(result, items...)
		return result
	})
}

// InsertAt inserts an item at the given index.
// If index is out of bounds, the item is appended (if index >= len) or prepended (if index < 0).
func (s *SliceSignal[T]) InsertAt(index int, item T) {
	s.Update(func(items []T) []T {
		if index < 0 {
			index = 0
		}
		if index >= len(items) {
			return append(items, item)
		}
		result := make([]T, 0, len(items)+1)
		result = append(result, items[:index]...)
		result = append(result, item)
		result = append(result, items[index:]...)
		return result
	})
}

// RemoveFirst removes and returns the first item from the slice.
// Does nothing if the slice is empty.
func (s *SliceSignal[T]) RemoveFirst() {
	s.Update(func(items []T) []T {
		if len(items) == 0 {
			return items
		}
		return items[1:]
	})
}

// RemoveLast removes the last item from the slice.
// Does nothing if the slice is empty.
func (s *SliceSignal[T]) RemoveLast() {
	s.Update(func(items []T) []T {
		if len(items) == 0 {
			return items
		}
		return items[:len(items)-1]
	})
}

// RemoveWhere removes all items that satisfy the predicate.
func (s *SliceSignal[T]) RemoveWhere(predicate func(T) bool) {
	s.Update(func(items []T) []T {
		result := make([]T, 0, len(items))
		for _, item := range items {
			if !predicate(item) {
				result = append(result, item)
			}
		}
		return result
	})
}

// UpdateAt updates the item at the given index using the provided function.
// Does nothing if index is out of bounds.
func (s *SliceSignal[T]) UpdateAt(index int, fn func(T) T) {
	s.Update(func(items []T) []T {
		if index < 0 || index >= len(items) {
			return items
		}
		// Create a copy to avoid modifying the original
		newItems := make([]T, len(items))
		copy(newItems, items)
		newItems[index] = fn(newItems[index])
		return newItems
	})
}

// UpdateWhere updates all items that satisfy the predicate using the provided function.
func (s *SliceSignal[T]) UpdateWhere(predicate func(T) bool, fn func(T) T) {
	s.Update(func(items []T) []T {
		// Create a copy to avoid modifying the original
		newItems := make([]T, len(items))
		copy(newItems, items)
		for i, item := range newItems {
			if predicate(item) {
				newItems[i] = fn(item)
			}
		}
		return newItems
	})
}
