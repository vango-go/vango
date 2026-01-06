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

	// Persistence options (Phase 12)
	transient  bool   // If true, signal is not persisted
	persistKey string // Explicit key for serialization
}

// NewSignal creates a new signal with the given initial value.
// Options can be passed to configure persistence behavior:
//
//	count := vango.NewSignal(0)                                  // Normal signal
//	cursor := vango.NewSignal(Point{0, 0}, vango.Transient())   // Not persisted
//	userID := vango.NewSignal(0, vango.PersistKey("user_id"))   // Custom key
func NewSignal[T any](initial T, opts ...SignalOption) *Signal[T] {
	owner := getCurrentOwner()
	inRender := owner != nil && isInRender()

	// Track hook call for dev-mode order validation
	if owner != nil {
		owner.TrackHook(HookSignal)
		if inRender {
			if slot := owner.UseHookSlot(); slot != nil {
				sig, ok := slot.(*Signal[T])
				if !ok {
					panic("vango: hook slot type mismatch for Signal")
				}
				return sig
			}
		}
	}

	options := applyOptions(opts)
	sig := &Signal[T]{
		base: signalBase{
			id: nextID(),
		},
		value:      initial,
		transient:  options.transient,
		persistKey: options.persistKey,
	}

	if inRender {
		owner.SetHookSlot(sig)
	}

	return sig
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
	// Check for prefetch mode (Phase 7: Routing, Section 8.3.2)
	// Signal writes are forbidden during prefetch
	if !checkPrefetchWrite("Signal.Set") {
		return // Drop write in production
	}

	// Check for effect-time write (Phase 16: Effect Enforcement)
	checkEffectTimeWrite("Set")

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
	// Check for prefetch mode (Phase 7: Routing, Section 8.3.2)
	if !checkPrefetchWrite("Signal.Update") {
		return
	}

	// Check for effect-time write (Phase 16: Effect Enforcement)
	checkEffectTimeWrite("Update")

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

// setQuietly updates the signal's value without notifying subscribers.
// This is used internally by context Providers during render to propagate
// new values to children without violating the "no signal writes during render"
// rule (§3.1.2). The children will read the updated value when they render.
//
// This method is NOT exported because it bypasses the normal reactive system
// and should only be used in specific internal scenarios like context propagation.
func (s *Signal[T]) setQuietly(value T) {
	s.mu.Lock()
	s.value = value
	s.mu.Unlock()
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

// =============================================================================
// PersistableSignal interface implementation (Phase 12)
// =============================================================================

// IsTransient returns true if the signal should not be persisted.
func (s *Signal[T]) IsTransient() bool {
	return s.transient
}

// PersistKey returns the explicit persistence key, or empty string for auto-key.
func (s *Signal[T]) PersistKey() string {
	return s.persistKey
}

// GetAny returns the current value as an interface{}.
// This is used for serialization without knowing the concrete type.
func (s *Signal[T]) GetAny() any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.value
}

// SetAny sets the value from an interface{}.
// Returns an error if the type doesn't match.
// This is used for deserialization without knowing the concrete type.
func (s *Signal[T]) SetAny(value any) error {
	v, ok := value.(T)
	if !ok {
		return &TypeMismatchError{
			Expected: reflect.TypeOf((*T)(nil)).Elem().String(),
			Actual:   reflect.TypeOf(value).String(),
		}
	}
	s.Set(v)
	return nil
}

// TypeMismatchError is returned when SetAny receives a value of the wrong type.
type TypeMismatchError struct {
	Expected string
	Actual   string
}

func (e *TypeMismatchError) Error() string {
	return "type mismatch: expected " + e.Expected + ", got " + e.Actual
}

// =============================================================================
// Convenience Methods (§3.9.4 Spec Compliance)
// These methods use runtime type assertions. Calling a method on an
// incompatible type will panic with a descriptive error message.
// =============================================================================

// -----------------------------------------------------------------------------
// Numeric Methods (int, int8, int16, int32, int64, uint, uint8, uint16, uint32,
// uint64, float32, float64)
// -----------------------------------------------------------------------------

// Inc increments the value by 1.
// Panics if the signal's type is not numeric.
func (s *Signal[T]) Inc() {
	if !checkPrefetchWrite("Signal.Inc") {
		return
	}
	checkEffectTimeWrite("Inc")

	s.mu.Lock()
	var newValue T
	var changed bool

	switch v := any(s.value).(type) {
	case int:
		newValue = any(v + 1).(T)
		changed = true
	case int8:
		newValue = any(v + 1).(T)
		changed = true
	case int16:
		newValue = any(v + 1).(T)
		changed = true
	case int32:
		newValue = any(v + 1).(T)
		changed = true
	case int64:
		newValue = any(v + 1).(T)
		changed = true
	case uint:
		newValue = any(v + 1).(T)
		changed = true
	case uint8:
		newValue = any(v + 1).(T)
		changed = true
	case uint16:
		newValue = any(v + 1).(T)
		changed = true
	case uint32:
		newValue = any(v + 1).(T)
		changed = true
	case uint64:
		newValue = any(v + 1).(T)
		changed = true
	case float32:
		newValue = any(v + 1).(T)
		changed = true
	case float64:
		newValue = any(v + 1).(T)
		changed = true
	default:
		s.mu.Unlock()
		panic("vango: Inc() called on non-numeric Signal[" + reflect.TypeOf(s.value).String() + "]")
	}

	if changed {
		s.value = newValue
	}
	s.mu.Unlock()

	if changed {
		s.base.notifySubscribers()
	}
}

// Dec decrements the value by 1.
// Panics if the signal's type is not numeric.
func (s *Signal[T]) Dec() {
	if !checkPrefetchWrite("Signal.Dec") {
		return
	}
	checkEffectTimeWrite("Dec")

	s.mu.Lock()
	var newValue T
	var changed bool

	switch v := any(s.value).(type) {
	case int:
		newValue = any(v - 1).(T)
		changed = true
	case int8:
		newValue = any(v - 1).(T)
		changed = true
	case int16:
		newValue = any(v - 1).(T)
		changed = true
	case int32:
		newValue = any(v - 1).(T)
		changed = true
	case int64:
		newValue = any(v - 1).(T)
		changed = true
	case uint:
		newValue = any(v - 1).(T)
		changed = true
	case uint8:
		newValue = any(v - 1).(T)
		changed = true
	case uint16:
		newValue = any(v - 1).(T)
		changed = true
	case uint32:
		newValue = any(v - 1).(T)
		changed = true
	case uint64:
		newValue = any(v - 1).(T)
		changed = true
	case float32:
		newValue = any(v - 1).(T)
		changed = true
	case float64:
		newValue = any(v - 1).(T)
		changed = true
	default:
		s.mu.Unlock()
		panic("vango: Dec() called on non-numeric Signal[" + reflect.TypeOf(s.value).String() + "]")
	}

	if changed {
		s.value = newValue
	}
	s.mu.Unlock()

	if changed {
		s.base.notifySubscribers()
	}
}

// Add adds n to the value.
// The parameter n must be the same numeric type as the signal's value.
// Panics if the signal's type is not numeric or if n is the wrong type.
func (s *Signal[T]) Add(n any) {
	if !checkPrefetchWrite("Signal.Add") {
		return
	}
	checkEffectTimeWrite("Add")

	s.mu.Lock()
	var newValue T
	var changed bool

	switch v := any(s.value).(type) {
	case int:
		nn, ok := n.(int)
		if !ok {
			s.mu.Unlock()
			panic("vango: Add() argument type mismatch: expected int, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v + nn).(T)
		changed = true
	case int8:
		nn, ok := n.(int8)
		if !ok {
			s.mu.Unlock()
			panic("vango: Add() argument type mismatch: expected int8, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v + nn).(T)
		changed = true
	case int16:
		nn, ok := n.(int16)
		if !ok {
			s.mu.Unlock()
			panic("vango: Add() argument type mismatch: expected int16, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v + nn).(T)
		changed = true
	case int32:
		nn, ok := n.(int32)
		if !ok {
			s.mu.Unlock()
			panic("vango: Add() argument type mismatch: expected int32, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v + nn).(T)
		changed = true
	case int64:
		nn, ok := n.(int64)
		if !ok {
			s.mu.Unlock()
			panic("vango: Add() argument type mismatch: expected int64, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v + nn).(T)
		changed = true
	case uint:
		nn, ok := n.(uint)
		if !ok {
			s.mu.Unlock()
			panic("vango: Add() argument type mismatch: expected uint, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v + nn).(T)
		changed = true
	case uint8:
		nn, ok := n.(uint8)
		if !ok {
			s.mu.Unlock()
			panic("vango: Add() argument type mismatch: expected uint8, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v + nn).(T)
		changed = true
	case uint16:
		nn, ok := n.(uint16)
		if !ok {
			s.mu.Unlock()
			panic("vango: Add() argument type mismatch: expected uint16, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v + nn).(T)
		changed = true
	case uint32:
		nn, ok := n.(uint32)
		if !ok {
			s.mu.Unlock()
			panic("vango: Add() argument type mismatch: expected uint32, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v + nn).(T)
		changed = true
	case uint64:
		nn, ok := n.(uint64)
		if !ok {
			s.mu.Unlock()
			panic("vango: Add() argument type mismatch: expected uint64, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v + nn).(T)
		changed = true
	case float32:
		nn, ok := n.(float32)
		if !ok {
			s.mu.Unlock()
			panic("vango: Add() argument type mismatch: expected float32, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v + nn).(T)
		changed = true
	case float64:
		nn, ok := n.(float64)
		if !ok {
			s.mu.Unlock()
			panic("vango: Add() argument type mismatch: expected float64, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v + nn).(T)
		changed = true
	default:
		s.mu.Unlock()
		panic("vango: Add() called on non-numeric Signal[" + reflect.TypeOf(s.value).String() + "]")
	}

	if changed {
		s.value = newValue
	}
	s.mu.Unlock()

	if changed {
		s.base.notifySubscribers()
	}
}

// Sub subtracts n from the value.
// The parameter n must be the same numeric type as the signal's value.
// Panics if the signal's type is not numeric or if n is the wrong type.
func (s *Signal[T]) Sub(n any) {
	if !checkPrefetchWrite("Signal.Sub") {
		return
	}
	checkEffectTimeWrite("Sub")

	s.mu.Lock()
	var newValue T
	var changed bool

	switch v := any(s.value).(type) {
	case int:
		nn, ok := n.(int)
		if !ok {
			s.mu.Unlock()
			panic("vango: Sub() argument type mismatch: expected int, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v - nn).(T)
		changed = true
	case int8:
		nn, ok := n.(int8)
		if !ok {
			s.mu.Unlock()
			panic("vango: Sub() argument type mismatch: expected int8, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v - nn).(T)
		changed = true
	case int16:
		nn, ok := n.(int16)
		if !ok {
			s.mu.Unlock()
			panic("vango: Sub() argument type mismatch: expected int16, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v - nn).(T)
		changed = true
	case int32:
		nn, ok := n.(int32)
		if !ok {
			s.mu.Unlock()
			panic("vango: Sub() argument type mismatch: expected int32, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v - nn).(T)
		changed = true
	case int64:
		nn, ok := n.(int64)
		if !ok {
			s.mu.Unlock()
			panic("vango: Sub() argument type mismatch: expected int64, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v - nn).(T)
		changed = true
	case uint:
		nn, ok := n.(uint)
		if !ok {
			s.mu.Unlock()
			panic("vango: Sub() argument type mismatch: expected uint, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v - nn).(T)
		changed = true
	case uint8:
		nn, ok := n.(uint8)
		if !ok {
			s.mu.Unlock()
			panic("vango: Sub() argument type mismatch: expected uint8, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v - nn).(T)
		changed = true
	case uint16:
		nn, ok := n.(uint16)
		if !ok {
			s.mu.Unlock()
			panic("vango: Sub() argument type mismatch: expected uint16, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v - nn).(T)
		changed = true
	case uint32:
		nn, ok := n.(uint32)
		if !ok {
			s.mu.Unlock()
			panic("vango: Sub() argument type mismatch: expected uint32, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v - nn).(T)
		changed = true
	case uint64:
		nn, ok := n.(uint64)
		if !ok {
			s.mu.Unlock()
			panic("vango: Sub() argument type mismatch: expected uint64, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v - nn).(T)
		changed = true
	case float32:
		nn, ok := n.(float32)
		if !ok {
			s.mu.Unlock()
			panic("vango: Sub() argument type mismatch: expected float32, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v - nn).(T)
		changed = true
	case float64:
		nn, ok := n.(float64)
		if !ok {
			s.mu.Unlock()
			panic("vango: Sub() argument type mismatch: expected float64, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v - nn).(T)
		changed = true
	default:
		s.mu.Unlock()
		panic("vango: Sub() called on non-numeric Signal[" + reflect.TypeOf(s.value).String() + "]")
	}

	if changed {
		s.value = newValue
	}
	s.mu.Unlock()

	if changed {
		s.base.notifySubscribers()
	}
}

// Mul multiplies the value by n.
// The parameter n must be the same numeric type as the signal's value.
// Panics if the signal's type is not numeric or if n is the wrong type.
func (s *Signal[T]) Mul(n any) {
	if !checkPrefetchWrite("Signal.Mul") {
		return
	}
	checkEffectTimeWrite("Mul")

	s.mu.Lock()
	var newValue T
	var changed bool

	switch v := any(s.value).(type) {
	case int:
		nn, ok := n.(int)
		if !ok {
			s.mu.Unlock()
			panic("vango: Mul() argument type mismatch: expected int, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v * nn).(T)
		changed = true
	case int8:
		nn, ok := n.(int8)
		if !ok {
			s.mu.Unlock()
			panic("vango: Mul() argument type mismatch: expected int8, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v * nn).(T)
		changed = true
	case int16:
		nn, ok := n.(int16)
		if !ok {
			s.mu.Unlock()
			panic("vango: Mul() argument type mismatch: expected int16, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v * nn).(T)
		changed = true
	case int32:
		nn, ok := n.(int32)
		if !ok {
			s.mu.Unlock()
			panic("vango: Mul() argument type mismatch: expected int32, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v * nn).(T)
		changed = true
	case int64:
		nn, ok := n.(int64)
		if !ok {
			s.mu.Unlock()
			panic("vango: Mul() argument type mismatch: expected int64, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v * nn).(T)
		changed = true
	case uint:
		nn, ok := n.(uint)
		if !ok {
			s.mu.Unlock()
			panic("vango: Mul() argument type mismatch: expected uint, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v * nn).(T)
		changed = true
	case uint8:
		nn, ok := n.(uint8)
		if !ok {
			s.mu.Unlock()
			panic("vango: Mul() argument type mismatch: expected uint8, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v * nn).(T)
		changed = true
	case uint16:
		nn, ok := n.(uint16)
		if !ok {
			s.mu.Unlock()
			panic("vango: Mul() argument type mismatch: expected uint16, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v * nn).(T)
		changed = true
	case uint32:
		nn, ok := n.(uint32)
		if !ok {
			s.mu.Unlock()
			panic("vango: Mul() argument type mismatch: expected uint32, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v * nn).(T)
		changed = true
	case uint64:
		nn, ok := n.(uint64)
		if !ok {
			s.mu.Unlock()
			panic("vango: Mul() argument type mismatch: expected uint64, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v * nn).(T)
		changed = true
	case float32:
		nn, ok := n.(float32)
		if !ok {
			s.mu.Unlock()
			panic("vango: Mul() argument type mismatch: expected float32, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v * nn).(T)
		changed = true
	case float64:
		nn, ok := n.(float64)
		if !ok {
			s.mu.Unlock()
			panic("vango: Mul() argument type mismatch: expected float64, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v * nn).(T)
		changed = true
	default:
		s.mu.Unlock()
		panic("vango: Mul() called on non-numeric Signal[" + reflect.TypeOf(s.value).String() + "]")
	}

	if changed {
		s.value = newValue
	}
	s.mu.Unlock()

	if changed {
		s.base.notifySubscribers()
	}
}

// Div divides the value by n.
// The parameter n must be the same numeric type as the signal's value.
// Panics if the signal's type is not numeric or if n is the wrong type.
// Note: Integer division truncates toward zero.
func (s *Signal[T]) Div(n any) {
	if !checkPrefetchWrite("Signal.Div") {
		return
	}
	checkEffectTimeWrite("Div")

	s.mu.Lock()
	var newValue T
	var changed bool

	switch v := any(s.value).(type) {
	case int:
		nn, ok := n.(int)
		if !ok {
			s.mu.Unlock()
			panic("vango: Div() argument type mismatch: expected int, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v / nn).(T)
		changed = true
	case int8:
		nn, ok := n.(int8)
		if !ok {
			s.mu.Unlock()
			panic("vango: Div() argument type mismatch: expected int8, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v / nn).(T)
		changed = true
	case int16:
		nn, ok := n.(int16)
		if !ok {
			s.mu.Unlock()
			panic("vango: Div() argument type mismatch: expected int16, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v / nn).(T)
		changed = true
	case int32:
		nn, ok := n.(int32)
		if !ok {
			s.mu.Unlock()
			panic("vango: Div() argument type mismatch: expected int32, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v / nn).(T)
		changed = true
	case int64:
		nn, ok := n.(int64)
		if !ok {
			s.mu.Unlock()
			panic("vango: Div() argument type mismatch: expected int64, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v / nn).(T)
		changed = true
	case uint:
		nn, ok := n.(uint)
		if !ok {
			s.mu.Unlock()
			panic("vango: Div() argument type mismatch: expected uint, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v / nn).(T)
		changed = true
	case uint8:
		nn, ok := n.(uint8)
		if !ok {
			s.mu.Unlock()
			panic("vango: Div() argument type mismatch: expected uint8, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v / nn).(T)
		changed = true
	case uint16:
		nn, ok := n.(uint16)
		if !ok {
			s.mu.Unlock()
			panic("vango: Div() argument type mismatch: expected uint16, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v / nn).(T)
		changed = true
	case uint32:
		nn, ok := n.(uint32)
		if !ok {
			s.mu.Unlock()
			panic("vango: Div() argument type mismatch: expected uint32, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v / nn).(T)
		changed = true
	case uint64:
		nn, ok := n.(uint64)
		if !ok {
			s.mu.Unlock()
			panic("vango: Div() argument type mismatch: expected uint64, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v / nn).(T)
		changed = true
	case float32:
		nn, ok := n.(float32)
		if !ok {
			s.mu.Unlock()
			panic("vango: Div() argument type mismatch: expected float32, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v / nn).(T)
		changed = true
	case float64:
		nn, ok := n.(float64)
		if !ok {
			s.mu.Unlock()
			panic("vango: Div() argument type mismatch: expected float64, got " + reflect.TypeOf(n).String())
		}
		newValue = any(v / nn).(T)
		changed = true
	default:
		s.mu.Unlock()
		panic("vango: Div() called on non-numeric Signal[" + reflect.TypeOf(s.value).String() + "]")
	}

	if changed {
		s.value = newValue
	}
	s.mu.Unlock()

	if changed {
		s.base.notifySubscribers()
	}
}

// -----------------------------------------------------------------------------
// Boolean Methods
// -----------------------------------------------------------------------------

// Toggle inverts a boolean value.
// Panics if the signal's type is not bool.
func (s *Signal[T]) Toggle() {
	if !checkPrefetchWrite("Signal.Toggle") {
		return
	}
	checkEffectTimeWrite("Toggle")

	s.mu.Lock()
	v, ok := any(s.value).(bool)
	if !ok {
		s.mu.Unlock()
		panic("vango: Toggle() called on non-bool Signal[" + reflect.TypeOf(s.value).String() + "]")
	}

	newValue := any(!v).(T)
	s.value = newValue
	s.mu.Unlock()

	s.base.notifySubscribers()
}

// SetTrue sets the value to true.
// Panics if the signal's type is not bool.
func (s *Signal[T]) SetTrue() {
	if !checkPrefetchWrite("Signal.SetTrue") {
		return
	}
	checkEffectTimeWrite("SetTrue")

	s.mu.Lock()
	_, ok := any(s.value).(bool)
	if !ok {
		s.mu.Unlock()
		panic("vango: SetTrue() called on non-bool Signal[" + reflect.TypeOf(s.value).String() + "]")
	}

	newValue := any(true).(T)
	changed := !s.equals(s.value, newValue)
	if changed {
		s.value = newValue
	}
	s.mu.Unlock()

	if changed {
		s.base.notifySubscribers()
	}
}

// SetFalse sets the value to false.
// Panics if the signal's type is not bool.
func (s *Signal[T]) SetFalse() {
	if !checkPrefetchWrite("Signal.SetFalse") {
		return
	}
	checkEffectTimeWrite("SetFalse")

	s.mu.Lock()
	_, ok := any(s.value).(bool)
	if !ok {
		s.mu.Unlock()
		panic("vango: SetFalse() called on non-bool Signal[" + reflect.TypeOf(s.value).String() + "]")
	}

	newValue := any(false).(T)
	changed := !s.equals(s.value, newValue)
	if changed {
		s.value = newValue
	}
	s.mu.Unlock()

	if changed {
		s.base.notifySubscribers()
	}
}

// -----------------------------------------------------------------------------
// String Methods
// -----------------------------------------------------------------------------

// Append appends a suffix to a string value.
// Panics if the signal's type is not string.
func (s *Signal[T]) Append(suffix string) {
	if !checkPrefetchWrite("Signal.Append") {
		return
	}
	checkEffectTimeWrite("Append")

	s.mu.Lock()
	v, ok := any(s.value).(string)
	if !ok {
		s.mu.Unlock()
		panic("vango: Append() called on non-string Signal[" + reflect.TypeOf(s.value).String() + "]")
	}

	newValue := any(v + suffix).(T)
	changed := !s.equals(s.value, newValue)
	if changed {
		s.value = newValue
	}
	s.mu.Unlock()

	if changed {
		s.base.notifySubscribers()
	}
}

// Prepend prepends a prefix to a string value.
// Panics if the signal's type is not string.
func (s *Signal[T]) Prepend(prefix string) {
	if !checkPrefetchWrite("Signal.Prepend") {
		return
	}
	checkEffectTimeWrite("Prepend")

	s.mu.Lock()
	v, ok := any(s.value).(string)
	if !ok {
		s.mu.Unlock()
		panic("vango: Prepend() called on non-string Signal[" + reflect.TypeOf(s.value).String() + "]")
	}

	newValue := any(prefix + v).(T)
	changed := !s.equals(s.value, newValue)
	if changed {
		s.value = newValue
	}
	s.mu.Unlock()

	if changed {
		s.base.notifySubscribers()
	}
}

// -----------------------------------------------------------------------------
// Slice Methods (reflection-based)
// -----------------------------------------------------------------------------

// AppendItem appends an item to a slice value.
// The item must be assignable to the slice's element type.
// Panics if the signal's type is not a slice.
func (s *Signal[T]) AppendItem(item any) {
	if !checkPrefetchWrite("Signal.AppendItem") {
		return
	}
	checkEffectTimeWrite("AppendItem")

	s.mu.Lock()
	v := reflect.ValueOf(s.value)
	if v.Kind() != reflect.Slice {
		s.mu.Unlock()
		panic("vango: AppendItem() called on non-slice Signal[" + reflect.TypeOf(s.value).String() + "]")
	}

	itemVal := reflect.ValueOf(item)
	newSlice := reflect.Append(v, itemVal)
	s.value = newSlice.Interface().(T)
	s.mu.Unlock()

	s.base.notifySubscribers()
}

// PrependItem prepends an item to a slice value.
// The item must be assignable to the slice's element type.
// Panics if the signal's type is not a slice.
func (s *Signal[T]) PrependItem(item any) {
	if !checkPrefetchWrite("Signal.PrependItem") {
		return
	}
	checkEffectTimeWrite("PrependItem")

	s.mu.Lock()
	v := reflect.ValueOf(s.value)
	if v.Kind() != reflect.Slice {
		s.mu.Unlock()
		panic("vango: PrependItem() called on non-slice Signal[" + reflect.TypeOf(s.value).String() + "]")
	}

	elemType := v.Type().Elem()
	itemVal := reflect.ValueOf(item)
	newSlice := reflect.MakeSlice(v.Type(), 0, v.Len()+1)
	newSlice = reflect.Append(newSlice, itemVal.Convert(elemType))
	newSlice = reflect.AppendSlice(newSlice, v)
	s.value = newSlice.Interface().(T)
	s.mu.Unlock()

	s.base.notifySubscribers()
}

// InsertAt inserts an item at the given index.
// If index is out of bounds, the item is appended (if index >= len) or prepended (if index < 0).
// Panics if the signal's type is not a slice.
func (s *Signal[T]) InsertAt(index int, item any) {
	if !checkPrefetchWrite("Signal.InsertAt") {
		return
	}
	checkEffectTimeWrite("InsertAt")

	s.mu.Lock()
	v := reflect.ValueOf(s.value)
	if v.Kind() != reflect.Slice {
		s.mu.Unlock()
		panic("vango: InsertAt() called on non-slice Signal[" + reflect.TypeOf(s.value).String() + "]")
	}

	length := v.Len()
	if index < 0 {
		index = 0
	}
	if index >= length {
		// Append
		itemVal := reflect.ValueOf(item)
		newSlice := reflect.Append(v, itemVal)
		s.value = newSlice.Interface().(T)
		s.mu.Unlock()
		s.base.notifySubscribers()
		return
	}

	// Insert in the middle
	elemType := v.Type().Elem()
	itemVal := reflect.ValueOf(item).Convert(elemType)
	newSlice := reflect.MakeSlice(v.Type(), 0, length+1)
	newSlice = reflect.AppendSlice(newSlice, v.Slice(0, index))
	newSlice = reflect.Append(newSlice, itemVal)
	newSlice = reflect.AppendSlice(newSlice, v.Slice(index, length))
	s.value = newSlice.Interface().(T)
	s.mu.Unlock()

	s.base.notifySubscribers()
}

// RemoveAt removes the item at the given index.
// Does nothing if index is out of bounds.
// Panics if the signal's type is not a slice.
func (s *Signal[T]) RemoveAt(index int) {
	if !checkPrefetchWrite("Signal.RemoveAt") {
		return
	}
	checkEffectTimeWrite("RemoveAt")

	s.mu.Lock()
	v := reflect.ValueOf(s.value)
	if v.Kind() != reflect.Slice {
		s.mu.Unlock()
		panic("vango: RemoveAt() called on non-slice Signal[" + reflect.TypeOf(s.value).String() + "]")
	}

	length := v.Len()
	if index < 0 || index >= length {
		s.mu.Unlock()
		return
	}

	newSlice := reflect.AppendSlice(v.Slice(0, index), v.Slice(index+1, length))
	s.value = newSlice.Interface().(T)
	s.mu.Unlock()

	s.base.notifySubscribers()
}

// RemoveFirst removes the first item from the slice.
// Does nothing if the slice is empty.
// Panics if the signal's type is not a slice.
func (s *Signal[T]) RemoveFirst() {
	if !checkPrefetchWrite("Signal.RemoveFirst") {
		return
	}
	checkEffectTimeWrite("RemoveFirst")

	s.mu.Lock()
	v := reflect.ValueOf(s.value)
	if v.Kind() != reflect.Slice {
		s.mu.Unlock()
		panic("vango: RemoveFirst() called on non-slice Signal[" + reflect.TypeOf(s.value).String() + "]")
	}

	if v.Len() == 0 {
		s.mu.Unlock()
		return
	}

	newSlice := v.Slice(1, v.Len())
	s.value = newSlice.Interface().(T)
	s.mu.Unlock()

	s.base.notifySubscribers()
}

// RemoveLast removes the last item from the slice.
// Does nothing if the slice is empty.
// Panics if the signal's type is not a slice.
func (s *Signal[T]) RemoveLast() {
	if !checkPrefetchWrite("Signal.RemoveLast") {
		return
	}
	checkEffectTimeWrite("RemoveLast")

	s.mu.Lock()
	v := reflect.ValueOf(s.value)
	if v.Kind() != reflect.Slice {
		s.mu.Unlock()
		panic("vango: RemoveLast() called on non-slice Signal[" + reflect.TypeOf(s.value).String() + "]")
	}

	if v.Len() == 0 {
		s.mu.Unlock()
		return
	}

	newSlice := v.Slice(0, v.Len()-1)
	s.value = newSlice.Interface().(T)
	s.mu.Unlock()

	s.base.notifySubscribers()
}

// RemoveWhere removes all items that satisfy the predicate.
// The predicate receives each item as any and returns true to remove it.
// Panics if the signal's type is not a slice.
func (s *Signal[T]) RemoveWhere(predicate func(any) bool) {
	if !checkPrefetchWrite("Signal.RemoveWhere") {
		return
	}
	checkEffectTimeWrite("RemoveWhere")

	s.mu.Lock()
	v := reflect.ValueOf(s.value)
	if v.Kind() != reflect.Slice {
		s.mu.Unlock()
		panic("vango: RemoveWhere() called on non-slice Signal[" + reflect.TypeOf(s.value).String() + "]")
	}

	newSlice := reflect.MakeSlice(v.Type(), 0, v.Len())
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i).Interface()
		if !predicate(item) {
			newSlice = reflect.Append(newSlice, v.Index(i))
		}
	}
	s.value = newSlice.Interface().(T)
	s.mu.Unlock()

	s.base.notifySubscribers()
}

// SetAt sets the item at the given index.
// Does nothing if index is out of bounds.
// Panics if the signal's type is not a slice.
func (s *Signal[T]) SetAt(index int, item any) {
	if !checkPrefetchWrite("Signal.SetAt") {
		return
	}
	checkEffectTimeWrite("SetAt")

	s.mu.Lock()
	v := reflect.ValueOf(s.value)
	if v.Kind() != reflect.Slice {
		s.mu.Unlock()
		panic("vango: SetAt() called on non-slice Signal[" + reflect.TypeOf(s.value).String() + "]")
	}

	if index < 0 || index >= v.Len() {
		s.mu.Unlock()
		return
	}

	// Create a copy to avoid modifying the original
	newSlice := reflect.MakeSlice(v.Type(), v.Len(), v.Cap())
	reflect.Copy(newSlice, v)
	newSlice.Index(index).Set(reflect.ValueOf(item))
	s.value = newSlice.Interface().(T)
	s.mu.Unlock()

	s.base.notifySubscribers()
}

// UpdateAt updates the item at the given index using the provided function.
// Does nothing if index is out of bounds.
// Panics if the signal's type is not a slice.
func (s *Signal[T]) UpdateAt(index int, fn func(any) any) {
	if !checkPrefetchWrite("Signal.UpdateAt") {
		return
	}
	checkEffectTimeWrite("UpdateAt")

	s.mu.Lock()
	v := reflect.ValueOf(s.value)
	if v.Kind() != reflect.Slice {
		s.mu.Unlock()
		panic("vango: UpdateAt() called on non-slice Signal[" + reflect.TypeOf(s.value).String() + "]")
	}

	if index < 0 || index >= v.Len() {
		s.mu.Unlock()
		return
	}

	// Create a copy to avoid modifying the original
	newSlice := reflect.MakeSlice(v.Type(), v.Len(), v.Cap())
	reflect.Copy(newSlice, v)
	oldItem := newSlice.Index(index).Interface()
	newItem := fn(oldItem)
	newSlice.Index(index).Set(reflect.ValueOf(newItem))
	s.value = newSlice.Interface().(T)
	s.mu.Unlock()

	s.base.notifySubscribers()
}

// UpdateWhere updates all items that satisfy the predicate using the provided function.
// Panics if the signal's type is not a slice.
func (s *Signal[T]) UpdateWhere(predicate func(any) bool, fn func(any) any) {
	if !checkPrefetchWrite("Signal.UpdateWhere") {
		return
	}
	checkEffectTimeWrite("UpdateWhere")

	s.mu.Lock()
	v := reflect.ValueOf(s.value)
	if v.Kind() != reflect.Slice {
		s.mu.Unlock()
		panic("vango: UpdateWhere() called on non-slice Signal[" + reflect.TypeOf(s.value).String() + "]")
	}

	// Create a copy to avoid modifying the original
	newSlice := reflect.MakeSlice(v.Type(), v.Len(), v.Cap())
	reflect.Copy(newSlice, v)
	for i := 0; i < newSlice.Len(); i++ {
		item := newSlice.Index(i).Interface()
		if predicate(item) {
			newItem := fn(item)
			newSlice.Index(i).Set(reflect.ValueOf(newItem))
		}
	}
	s.value = newSlice.Interface().(T)
	s.mu.Unlock()

	s.base.notifySubscribers()
}

// Filter keeps only items that satisfy the predicate.
// Panics if the signal's type is not a slice.
func (s *Signal[T]) Filter(predicate func(any) bool) {
	if !checkPrefetchWrite("Signal.Filter") {
		return
	}
	checkEffectTimeWrite("Filter")

	s.mu.Lock()
	v := reflect.ValueOf(s.value)
	if v.Kind() != reflect.Slice {
		s.mu.Unlock()
		panic("vango: Filter() called on non-slice Signal[" + reflect.TypeOf(s.value).String() + "]")
	}

	newSlice := reflect.MakeSlice(v.Type(), 0, v.Len())
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i).Interface()
		if predicate(item) {
			newSlice = reflect.Append(newSlice, v.Index(i))
		}
	}
	s.value = newSlice.Interface().(T)
	s.mu.Unlock()

	s.base.notifySubscribers()
}

// -----------------------------------------------------------------------------
// Map Methods (reflection-based)
// -----------------------------------------------------------------------------

// SetKey sets a key-value pair in a map.
// The key and value must be assignable to the map's key and value types.
// Panics if the signal's type is not a map.
func (s *Signal[T]) SetKey(key, value any) {
	if !checkPrefetchWrite("Signal.SetKey") {
		return
	}
	checkEffectTimeWrite("SetKey")

	s.mu.Lock()
	v := reflect.ValueOf(s.value)
	if v.Kind() != reflect.Map {
		s.mu.Unlock()
		panic("vango: SetKey() called on non-map Signal[" + reflect.TypeOf(s.value).String() + "]")
	}

	// Create a copy of the map to avoid modifying the original
	newMap := reflect.MakeMap(v.Type())
	iter := v.MapRange()
	for iter.Next() {
		newMap.SetMapIndex(iter.Key(), iter.Value())
	}
	newMap.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
	s.value = newMap.Interface().(T)
	s.mu.Unlock()

	s.base.notifySubscribers()
}

// RemoveKey removes a key from a map.
// Does nothing if the key doesn't exist.
// Panics if the signal's type is not a map.
func (s *Signal[T]) RemoveKey(key any) {
	if !checkPrefetchWrite("Signal.RemoveKey") {
		return
	}
	checkEffectTimeWrite("RemoveKey")

	s.mu.Lock()
	v := reflect.ValueOf(s.value)
	if v.Kind() != reflect.Map {
		s.mu.Unlock()
		panic("vango: RemoveKey() called on non-map Signal[" + reflect.TypeOf(s.value).String() + "]")
	}

	// Create a copy of the map to avoid modifying the original
	newMap := reflect.MakeMap(v.Type())
	keyVal := reflect.ValueOf(key)
	iter := v.MapRange()
	for iter.Next() {
		if !iter.Key().Equal(keyVal) {
			newMap.SetMapIndex(iter.Key(), iter.Value())
		}
	}
	s.value = newMap.Interface().(T)
	s.mu.Unlock()

	s.base.notifySubscribers()
}

// UpdateKey updates a key's value using the provided function.
// If the key doesn't exist, the function receives nil/zero value.
// Panics if the signal's type is not a map.
func (s *Signal[T]) UpdateKey(key any, fn func(any) any) {
	if !checkPrefetchWrite("Signal.UpdateKey") {
		return
	}
	checkEffectTimeWrite("UpdateKey")

	s.mu.Lock()
	v := reflect.ValueOf(s.value)
	if v.Kind() != reflect.Map {
		s.mu.Unlock()
		panic("vango: UpdateKey() called on non-map Signal[" + reflect.TypeOf(s.value).String() + "]")
	}

	// Create a copy of the map to avoid modifying the original
	newMap := reflect.MakeMap(v.Type())
	iter := v.MapRange()
	for iter.Next() {
		newMap.SetMapIndex(iter.Key(), iter.Value())
	}

	keyVal := reflect.ValueOf(key)
	oldVal := v.MapIndex(keyVal)
	var oldItem any
	if oldVal.IsValid() {
		oldItem = oldVal.Interface()
	}
	newItem := fn(oldItem)
	newMap.SetMapIndex(keyVal, reflect.ValueOf(newItem))
	s.value = newMap.Interface().(T)
	s.mu.Unlock()

	s.base.notifySubscribers()
}

// HasKey returns true if the key exists in the map.
// Panics if the signal's type is not a map.
func (s *Signal[T]) HasKey(key any) bool {
	s.mu.RLock()
	v := reflect.ValueOf(s.value)
	if v.Kind() != reflect.Map {
		s.mu.RUnlock()
		panic("vango: HasKey() called on non-map Signal[" + reflect.TypeOf(s.value).String() + "]")
	}

	keyVal := reflect.ValueOf(key)
	result := v.MapIndex(keyVal).IsValid()
	s.mu.RUnlock()

	// Track dependency
	if listener := getCurrentListener(); listener != nil {
		s.base.subscribe(listener)
	}

	return result
}

// -----------------------------------------------------------------------------
// Common Methods
// -----------------------------------------------------------------------------

// Clear sets the value to its empty state.
// Works for string (empty string), slice (empty slice), and map (empty map).
// Panics if the signal's type is not one of these.
func (s *Signal[T]) Clear() {
	if !checkPrefetchWrite("Signal.Clear") {
		return
	}
	checkEffectTimeWrite("Clear")

	s.mu.Lock()
	v := reflect.ValueOf(s.value)
	var newValue T
	var changed bool

	switch v.Kind() {
	case reflect.String:
		newValue = any("").(T)
		changed = !s.equals(s.value, newValue)
	case reflect.Slice:
		newValue = reflect.MakeSlice(v.Type(), 0, 0).Interface().(T)
		changed = true // Slices always considered changed for simplicity
	case reflect.Map:
		newValue = reflect.MakeMap(v.Type()).Interface().(T)
		changed = true // Maps always considered changed for simplicity
	default:
		s.mu.Unlock()
		panic("vango: Clear() called on unsupported type Signal[" + reflect.TypeOf(s.value).String() + "]")
	}

	if changed {
		s.value = newValue
	}
	s.mu.Unlock()

	if changed {
		s.base.notifySubscribers()
	}
}

// Len returns the length of the value.
// Works for string, slice, and map types.
// Panics if the signal's type is not one of these.
// This reads the signal and creates a dependency.
func (s *Signal[T]) Len() int {
	s.mu.RLock()
	v := reflect.ValueOf(s.value)
	var length int

	switch v.Kind() {
	case reflect.String:
		length = v.Len()
	case reflect.Slice:
		length = v.Len()
	case reflect.Map:
		length = v.Len()
	default:
		s.mu.RUnlock()
		panic("vango: Len() called on unsupported type Signal[" + reflect.TypeOf(s.value).String() + "]")
	}
	s.mu.RUnlock()

	// Track dependency
	if listener := getCurrentListener(); listener != nil {
		s.base.subscribe(listener)
	}

	return length
}
