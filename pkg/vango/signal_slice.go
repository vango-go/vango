package vango

// SliceSignal wraps Signal[[]T] with convenience methods for slice operations.
type SliceSignal[T any] struct {
	*Signal[[]T]
}

// NewSliceSignal creates a new SliceSignal with the given initial value.
// If initial is nil, creates an empty slice.
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
