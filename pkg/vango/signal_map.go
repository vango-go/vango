package vango

// MapSignal wraps Signal[map[K]V] with convenience methods for map operations.
//
// Deprecated: Use NewSignal[map[K]V] instead. Signal[T] now has SetKey(), RemoveKey(),
// UpdateKey(), HasKey(), Clear(), and Len() methods directly available.
//
// Note: MapSignal[K,V] provides type-safe methods (e.g., SetKey(key K, value V)) while
// the Signal[T] methods use any (e.g., SetKey(key, value any)). Use MapSignal[K,V] if
// you prefer compile-time type safety over the unified API.
//
//	// Old:
//	users := vango.NewMapSignal(map[string]User{})
//	users.SetKey("alice", alice)
//
//	// New:
//	users := vango.NewSignal(map[string]User{})
//	users.SetKey("alice", alice)  // Note: uses 'any' parameters
type MapSignal[K comparable, V any] struct {
	*Signal[map[K]V]
}

// NewMapSignal creates a new MapSignal with the given initial value.
// If initial is nil, creates an empty map.
//
// Deprecated: Use NewSignal[map[K]V] instead. Signal[T] now has map convenience methods.
// MapSignal[K,V] is retained for type-safe map operations if preferred.
func NewMapSignal[K comparable, V any](initial map[K]V) *MapSignal[K, V] {
	if initial == nil {
		initial = make(map[K]V)
	}
	return &MapSignal[K, V]{NewSignal(initial)}
}

// SetKey sets a key-value pair in the map.
func (s *MapSignal[K, V]) SetKey(key K, value V) {
	s.Update(func(m map[K]V) map[K]V {
		// Create a copy to avoid modifying the original
		newMap := make(map[K]V, len(m)+1)
		for k, v := range m {
			newMap[k] = v
		}
		newMap[key] = value
		return newMap
	})
}

// RemoveKey removes a key from the map.
func (s *MapSignal[K, V]) RemoveKey(key K) {
	s.Update(func(m map[K]V) map[K]V {
		if _, ok := m[key]; !ok {
			// Key doesn't exist, no change
			return m
		}
		// Create a copy to avoid modifying the original
		newMap := make(map[K]V, len(m))
		for k, v := range m {
			if k != key {
				newMap[k] = v
			}
		}
		return newMap
	})
}

// DeleteKey removes a key from the map.
// Deprecated: use RemoveKey instead.
func (s *MapSignal[K, V]) DeleteKey(key K) {
	s.RemoveKey(key)
}

// UpdateKey updates the value for a key using the provided function.
// Does nothing if the key doesn't exist.
func (s *MapSignal[K, V]) UpdateKey(key K, fn func(V) V) {
	s.Update(func(m map[K]V) map[K]V {
		v, ok := m[key]
		if !ok {
			// Key doesn't exist, no change
			return m
		}
		// Create a copy to avoid modifying the original
		newMap := make(map[K]V, len(m))
		for k, val := range m {
			newMap[k] = val
		}
		newMap[key] = fn(v)
		return newMap
	})
}

// GetKey returns the value for a key.
// This reads the signal and creates a dependency.
func (s *MapSignal[K, V]) GetKey(key K) (V, bool) {
	m := s.Get()
	v, ok := m[key]
	return v, ok
}

// HasKey returns true if the key exists in the map.
// This reads the signal and creates a dependency.
func (s *MapSignal[K, V]) HasKey(key K) bool {
	_, ok := s.GetKey(key)
	return ok
}

// Len returns the number of keys in the map.
// This reads the signal and creates a dependency.
func (s *MapSignal[K, V]) Len() int {
	return len(s.Get())
}

// Clear removes all keys from the map.
func (s *MapSignal[K, V]) Clear() {
	s.Set(make(map[K]V))
}

// Keys returns all keys in the map.
// This reads the signal and creates a dependency.
func (s *MapSignal[K, V]) Keys() []K {
	m := s.Get()
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Values returns all values in the map.
// This reads the signal and creates a dependency.
func (s *MapSignal[K, V]) Values() []V {
	m := s.Get()
	values := make([]V, 0, len(m))
	for _, v := range m {
		values = append(values, v)
	}
	return values
}
