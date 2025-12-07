package vango

import (
	"reflect"
	"sync"
)

// signalBase provides type-erased subscriber management.
// It is embedded in Signal[T] and Memo[T] to share subscription logic.
type signalBase struct {
	id uint64

	// subs are the listeners subscribed to this signal.
	subs []Listener

	// subMu protects the subs slice.
	subMu sync.RWMutex
}

// subscribe adds a listener to this signal's subscribers.
// Deduplicates by listener ID to prevent double-subscription.
func (s *signalBase) subscribe(l Listener) {
	if l == nil {
		return
	}

	s.subMu.Lock()
	defer s.subMu.Unlock()

	// Deduplicate by ID
	lid := l.ID()
	for _, existing := range s.subs {
		if existing.ID() == lid {
			return
		}
	}

	s.subs = append(s.subs, l)
}

// unsubscribe removes a listener from this signal's subscribers.
func (s *signalBase) unsubscribe(l Listener) {
	if l == nil {
		return
	}

	s.subMu.Lock()
	defer s.subMu.Unlock()

	lid := l.ID()
	for i, existing := range s.subs {
		if existing.ID() == lid {
			// Remove by swapping with last element (order doesn't matter)
			s.subs[i] = s.subs[len(s.subs)-1]
			s.subs = s.subs[:len(s.subs)-1]
			return
		}
	}
}

// notifySubscribers notifies all subscribers that this signal changed.
// Uses copy-before-notify pattern to avoid holding locks during notification.
func (s *signalBase) notifySubscribers() {
	// Copy subscribers while holding lock
	s.subMu.RLock()
	subs := make([]Listener, len(s.subs))
	copy(subs, s.subs)
	s.subMu.RUnlock()

	// Check if we're in batch mode
	batchDepth := getBatchDepth()

	if batchDepth > 0 {
		// Queue for later
		for _, sub := range subs {
			queuePendingUpdate(sub)
		}
	} else {
		// Notify immediately
		for _, sub := range subs {
			sub.MarkDirty()
		}
	}
}

// getID returns the unique identifier for this signal.
func (s *signalBase) getID() uint64 {
	return s.id
}

// Signal is a reactive value container.
// Reading a Signal's value during a tracked context (component render, memo
// computation, or effect execution) automatically subscribes the current
// listener to receive notifications when the value changes.
type Signal[T any] struct {
	base signalBase

	// value is the current signal value.
	value T

	// mu protects the value.
	mu sync.RWMutex

	// equal is the equality function used to determine if the value changed.
	// If nil, uses default equality checking.
	equal func(T, T) bool
}

// NewSignal creates a new signal with the given initial value.
func NewSignal[T any](initial T) *Signal[T] {
	return &Signal[T]{
		base: signalBase{
			id: nextID(),
		},
		value: initial,
	}
}

// Get returns the current value and subscribes the current listener.
// If called during a tracked context (component render, memo computation,
// or effect execution), the current listener will be notified when this
// signal's value changes.
func (s *Signal[T]) Get() T {
	// Read value with lock
	s.mu.RLock()
	value := s.value
	s.mu.RUnlock()

	// Track dependency (after releasing value lock to prevent deadlock)
	if listener := getCurrentListener(); listener != nil {
		s.base.subscribe(listener)

		// If listener is an Effect, track this as a source
		if e, ok := listener.(*Effect); ok {
			e.addSource(&s.base)
		}
		// If listener is a Memo, track this as a source
		if m, ok := listener.(memoBase); ok {
			m.addSource(&s.base)
		}
	}

	return value
}

// Peek returns the current value without subscribing.
// This is useful when you need to read a value without creating a dependency.
func (s *Signal[T]) Peek() T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.value
}

// Set updates the signal's value and notifies subscribers if the value changed.
// Uses the signal's equality function to determine if the value changed.
func (s *Signal[T]) Set(value T) {
	s.mu.Lock()
	changed := !s.equals(s.value, value)
	if changed {
		s.value = value
	}
	s.mu.Unlock()

	if changed {
		s.base.notifySubscribers()
	}
}

// Update atomically reads and updates the signal's value.
// The function receives the current value and returns the new value.
func (s *Signal[T]) Update(fn func(T) T) {
	s.mu.Lock()
	oldValue := s.value
	newValue := fn(oldValue)
	changed := !s.equals(oldValue, newValue)
	if changed {
		s.value = newValue
	}
	s.mu.Unlock()

	if changed {
		s.base.notifySubscribers()
	}
}

// WithEquals returns the signal configured with a custom equality function.
// This is useful for custom types where reflect.DeepEqual is too expensive
// or has incorrect semantics.
func (s *Signal[T]) WithEquals(fn func(T, T) bool) *Signal[T] {
	s.equal = fn
	return s
}

// ID returns the unique identifier for this signal.
func (s *Signal[T]) ID() uint64 {
	return s.base.id
}

// equals checks if two values are equal using the configured equality function.
func (s *Signal[T]) equals(a, b T) bool {
	if s.equal != nil {
		return s.equal(a, b)
	}
	return defaultEquals(a, b)
}

// defaultEquals provides type-appropriate equality checking.
// Uses == for comparable types and reflect.DeepEqual for others.
func defaultEquals[T any](a, b T) bool {
	// Try to use == for comparable types
	// This is a type assertion that will succeed for comparable types
	switch av := any(a).(type) {
	case int:
		return av == any(b).(int)
	case int8:
		return av == any(b).(int8)
	case int16:
		return av == any(b).(int16)
	case int32:
		return av == any(b).(int32)
	case int64:
		return av == any(b).(int64)
	case uint:
		return av == any(b).(uint)
	case uint8:
		return av == any(b).(uint8)
	case uint16:
		return av == any(b).(uint16)
	case uint32:
		return av == any(b).(uint32)
	case uint64:
		return av == any(b).(uint64)
	case float32:
		return av == any(b).(float32)
	case float64:
		return av == any(b).(float64)
	case string:
		return av == any(b).(string)
	case bool:
		return av == any(b).(bool)
	default:
		// Fall back to reflect.DeepEqual for slices, maps, structs, etc.
		return reflect.DeepEqual(a, b)
	}
}

// memoBase is an interface to allow Signal to recognize Memo listeners.
// This is needed because Memo is generic and we can't do direct type assertion.
type memoBase interface {
	Listener
	addSource(source *signalBase)
}
