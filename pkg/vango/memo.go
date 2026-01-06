package vango

import (
	"reflect"
	"sync"
	"sync/atomic"
)

// Memo is a cached computation that automatically tracks its dependencies.
// When any dependency changes, the memo is invalidated and will recompute
// on the next read.
//
// Memos are lazy: they only compute their value when Get() is called.
// If multiple signals change before a read, the memo only recomputes once.
//
// Memos can also be subscribed to, behaving like signals themselves.
// This allows building chains of derived values.
type Memo[T any] struct {
	base signalBase

	// compute is the function that computes the memo's value.
	compute func() T

	// value is the cached computed value.
	value T

	// valueMu protects value access.
	valueMu sync.RWMutex

	// valid indicates whether the cached value is current.
	// When false, the next Get() will recompute.
	valid atomic.Bool

	// sources are the signals/memos this memo depends on.
	sources   []*signalBase
	sourcesMu sync.Mutex

	// equal is the equality function for determining value changes.
	equal func(T, T) bool

	// computing prevents infinite recursion in circular dependencies.
	computing atomic.Bool
}

// NewMemo creates a new memo with the given computation function.
// The computation is not run immediately; it runs lazily on first Get().
func NewMemo[T any](compute func() T) *Memo[T] {
	owner := getCurrentOwner()
	inRender := owner != nil && isInRender()

	// Track hook call for dev-mode order validation
	if owner != nil {
		owner.TrackHook(HookMemo)
		if inRender {
			if slot := owner.UseHookSlot(); slot != nil {
				memo, ok := slot.(*Memo[T])
				if !ok {
					panic("vango: hook slot type mismatch for Memo")
				}
				// Update compute function in case closures changed
				memo.compute = compute
				memo.valid.Store(false)
				return memo
			}
		}
	}

	memo := &Memo[T]{
		base: signalBase{
			id: nextID(),
		},
		compute: compute,
	}

	if inRender {
		owner.SetHookSlot(memo)
	}

	return memo
}

// Get returns the memo's value, recomputing if necessary.
// Creates a dependency on this memo for the current listener.
func (m *Memo[T]) Get() T {
	// Track dependency on this memo
	if listener := getCurrentListener(); listener != nil {
		m.base.subscribe(listener)

		// Track source for cleanup
		if e, ok := listener.(*Effect); ok {
			e.addSource(&m.base)
		}
		if mb, ok := listener.(memoBase); ok {
			mb.addSource(&m.base)
		}
	}

	// Recompute if invalid
	if !m.valid.Load() {
		m.recompute()
	}

	m.valueMu.RLock()
	value := m.value
	m.valueMu.RUnlock()
	return value
}

// Peek returns the memo's value without subscribing.
// Still triggers recomputation if the value is invalid.
func (m *Memo[T]) Peek() T {
	if !m.valid.Load() {
		m.recompute()
	}
	m.valueMu.RLock()
	value := m.value
	m.valueMu.RUnlock()
	return value
}

// MarkDirty invalidates the memo and propagates to subscribers.
// Implements the Listener interface.
func (m *Memo[T]) MarkDirty() {
	// Use CAS for idempotent marking
	if m.valid.CompareAndSwap(true, false) {
		// Propagate to subscribers
		m.base.notifySubscribers()
	}
}

// ID returns the unique identifier for this memo.
// Implements the Listener interface.
func (m *Memo[T]) ID() uint64 {
	return m.base.id
}

// addSource adds a source dependency.
// Implements the memoBase interface.
func (m *Memo[T]) addSource(source *signalBase) {
	m.sourcesMu.Lock()
	defer m.sourcesMu.Unlock()

	// Check for duplicates
	for _, s := range m.sources {
		if s == source {
			return
		}
	}
	m.sources = append(m.sources, source)
}

// WithEquals configures the memo with a custom equality function.
func (m *Memo[T]) WithEquals(fn func(T, T) bool) *Memo[T] {
	m.equal = fn
	return m
}

// recompute runs the computation and updates the cached value.
func (m *Memo[T]) recompute() {
	// Prevent infinite recursion in circular dependencies
	if m.computing.Swap(true) {
		// Already computing - circular dependency detected
		return
	}
	defer m.computing.Store(false)

	// Unsubscribe from old sources
	m.sourcesMu.Lock()
	for _, source := range m.sources {
		source.unsubscribe(m)
	}
	m.sources = m.sources[:0]
	m.sourcesMu.Unlock()

	// Track new sources during computation
	old := setCurrentListener(m)

	// Compute new value
	newValue := m.compute()

	// Restore previous listener
	setCurrentListener(old)

	// Update value with mutex protection
	m.valueMu.Lock()
	// Check if value changed (for downstream notification)
	changed := !m.equals(m.value, newValue)
	m.value = newValue
	m.valueMu.Unlock()

	m.valid.Store(true)

	// If value changed and we have subscribers, notify them
	// Note: This is tricky - we need to notify only if we're being read
	// and the value actually changed from what was last observed
	_ = changed // Value change is implicit in the MarkDirty call that triggered recompute
}

// equals checks if two values are equal.
func (m *Memo[T]) equals(a, b T) bool {
	if m.equal != nil {
		return m.equal(a, b)
	}
	return memoDefaultEquals(a, b)
}

// memoDefaultEquals provides default equality for memo values.
func memoDefaultEquals[T any](a, b T) bool {
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
		return reflect.DeepEqual(a, b)
	}
}

// Ensure Memo implements memoBase
var _ memoBase = (*Memo[int])(nil)
