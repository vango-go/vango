package vango

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
)

// Note: DebugMode is declared in batch.go and shared across the package.
// It enables dev-time validation like hook order checking.

// HookType identifies the type of hook call for order validation.
type HookType uint8

const (
	HookSignal HookType = iota + 1
	HookMemo
	HookEffect
	HookResource
	HookForm
	HookURLParam
	HookRef
	HookContext
	HookAction // Phase 16: Action API
)

// String returns a human-readable name for the hook type.
func (h HookType) String() string {
	switch h {
	case HookSignal:
		return "Signal"
	case HookMemo:
		return "Memo"
	case HookEffect:
		return "Effect"
	case HookResource:
		return "Resource"
	case HookForm:
		return "Form"
	case HookURLParam:
		return "URLParam"
	case HookRef:
		return "Ref"
	case HookContext:
		return "Context"
	case HookAction:
		return "Action"
	default:
		return "Unknown"
	}
}

// hookRecord records a single hook call for order validation.
type hookRecord struct {
	Type HookType
}

// Owner represents a component scope that owns reactive primitives.
// When an Owner is disposed, all signals, memos, effects, and child owners
// it contains are also disposed. This ensures proper cleanup and prevents
// memory leaks.
//
// Owners form a hierarchy: each component creates an Owner that is a child
// of its parent component's Owner. This mirrors the component tree structure.
type Owner struct {
	id uint64

	// parent is the parent Owner in the hierarchy.
	// nil for the root Owner (typically the session).
	parent *Owner

	// children are child Owners (sub-components).
	children   []*Owner
	childrenMu sync.Mutex

	// effects owned by this scope.
	effects   []*Effect
	effectsMu sync.Mutex

	// cleanups are manual cleanup functions registered via OnCleanup.
	cleanups   []func()
	cleanupsMu sync.Mutex

	// pendingEffects are effects scheduled to run after render.
	pendingEffects   []*Effect
	pendingEffectsMu sync.Mutex

	// values stores context values for this scope.
	values   map[any]any
	valuesMu sync.RWMutex

	// disposed indicates whether this Owner has been disposed.
	disposed atomic.Bool

	// Dev-mode hook order tracking (only used when DebugMode is true)
	hookOrder   []hookRecord // Expected order from first render
	hookIndex   int          // Current index during render
	renderCount int          // 0 = first render, 1+ = subsequent

	// Hook slot storage for stable identity across renders.
	// This is always active (not just in DebugMode) because hooks like
	// URLParam and Resource need stable identity for correctness.
	hookSlots   []any // Stored hook state values (one per hook)
	hookSlotIdx int   // Current slot index during render
}

// NewOwner creates a new Owner with the given parent.
// The new Owner is automatically registered as a child of the parent.
// If parent is nil, creates a root Owner.
func NewOwner(parent *Owner) *Owner {
	o := &Owner{
		id:     nextID(),
		parent: parent,
	}

	if parent != nil {
		parent.addChild(o)
	}

	return o
}

// ID returns the unique identifier for this Owner.
func (o *Owner) ID() uint64 {
	return o.id
}

// Parent returns the parent Owner, or nil if this is a root Owner.
func (o *Owner) Parent() *Owner {
	return o.parent
}

// IsDisposed returns true if this Owner has been disposed.
func (o *Owner) IsDisposed() bool {
	return o.disposed.Load()
}

// addChild registers a child Owner.
func (o *Owner) addChild(child *Owner) {
	o.childrenMu.Lock()
	defer o.childrenMu.Unlock()
	o.children = append(o.children, child)
}

// removeChild removes a child Owner from this Owner's children.
func (o *Owner) removeChild(child *Owner) {
	o.childrenMu.Lock()
	defer o.childrenMu.Unlock()

	for i, c := range o.children {
		if c == child {
			o.children = append(o.children[:i], o.children[i+1:]...)
			return
		}
	}
}

// registerEffect adds an effect to this Owner.
// The effect will be disposed when this Owner is disposed.
func (o *Owner) registerEffect(e *Effect) {
	if o.disposed.Load() {
		return
	}

	o.effectsMu.Lock()
	defer o.effectsMu.Unlock()
	o.effects = append(o.effects, e)
}

// OnCleanup registers a cleanup function to run when this Owner is disposed.
func (o *Owner) OnCleanup(fn func()) {
	if o.disposed.Load() {
		// Already disposed, run cleanup immediately
		fn()
		return
	}

	o.cleanupsMu.Lock()
	defer o.cleanupsMu.Unlock()
	o.cleanups = append(o.cleanups, fn)
}

// scheduleEffect adds an effect to the pending effects queue.
// Effects are run after the render phase via RunPendingEffects.
func (o *Owner) scheduleEffect(e *Effect) {
	if o.disposed.Load() {
		return
	}

	o.pendingEffectsMu.Lock()
	defer o.pendingEffectsMu.Unlock()
	o.pendingEffects = append(o.pendingEffects, e)
}

// MemoryUsage estimates the memory usage of this Owner and its children.
func (o *Owner) MemoryUsage() int64 {
	if o == nil {
		return 0
	}

	var size int64 = 256 // Base struct + mutex overhead

	o.effectsMu.Lock()
	effectsCount := len(o.effects)
	o.effectsMu.Unlock()
	size += estimateSliceMemory(effectsCount, 8)

	o.cleanupsMu.Lock()
	cleanupsCount := len(o.cleanups)
	o.cleanupsMu.Unlock()
	size += estimateSliceMemory(cleanupsCount, 8)

	o.pendingEffectsMu.Lock()
	pendingCount := len(o.pendingEffects)
	o.pendingEffectsMu.Unlock()
	size += estimateSliceMemory(pendingCount, 8)

	o.valuesMu.RLock()
	valuesCopy := make(map[any]any, len(o.values))
	for k, v := range o.values {
		valuesCopy[k] = v
	}
	o.valuesMu.RUnlock()

	size += estimateMapMemory(len(valuesCopy), 16, 16)
	for k, v := range valuesCopy {
		size += estimateAnyMemory(k, 0)
		size += estimateAnyMemory(v, 0)
	}

	o.childrenMu.Lock()
	children := append([]*Owner(nil), o.children...)
	o.childrenMu.Unlock()
	size += estimateSliceMemory(len(children), 8)
	for _, child := range children {
		size += child.MemoryUsage()
	}

	return size
}

const estimateAnyMaxDepth = 4

func estimateSliceMemory(length, elementSize int) int64 {
	return 24 + int64(length*elementSize)
}

func estimateMapMemory(length, keySize, valueSize int) int64 {
	buckets := (length / 8) + 1
	bucketOverhead := int64(buckets * 8)
	entrySize := int64(keySize + valueSize + 8)
	return 48 + bucketOverhead + int64(length)*entrySize
}

func estimateAnyMemory(value any, depth int) int64 {
	v := reflect.ValueOf(value)
	if !v.IsValid() {
		return 0
	}
	if depth >= estimateAnyMaxDepth {
		return 16
	}

	switch v.Kind() {
	case reflect.String:
		return 16 + int64(len(v.String()))
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return 8
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return 8
	case reflect.Float32, reflect.Float64:
		return 8
	case reflect.Interface, reflect.Pointer, reflect.Ptr:
		if v.IsNil() {
			return 0
		}
		return 8 + estimateAnyMemory(v.Elem().Interface(), depth+1)
	case reflect.Slice:
		if v.IsNil() {
			return 0
		}
		size := estimateSliceMemory(v.Len(), 16)
		for i := 0; i < v.Len(); i++ {
			size += estimateAnyMemory(v.Index(i).Interface(), depth+1)
		}
		return size
	case reflect.Array:
		size := int64(16)
		for i := 0; i < v.Len(); i++ {
			size += estimateAnyMemory(v.Index(i).Interface(), depth+1)
		}
		return size
	case reflect.Map:
		if v.IsNil() {
			return 0
		}
		size := estimateMapMemory(v.Len(), 16, 16)
		iter := v.MapRange()
		for iter.Next() {
			size += estimateAnyMemory(iter.Key().Interface(), depth+1)
			size += estimateAnyMemory(iter.Value().Interface(), depth+1)
		}
		return size
	case reflect.Struct:
		size := int64(16)
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			if field.CanInterface() {
				size += estimateAnyMemory(field.Interface(), depth+1)
			} else {
				size += 16
			}
		}
		return size
	default:
		return 16
	}
}

// RunPendingEffects executes all pending effects.
// This is called after the render phase to run scheduled effects.
// The server runtime calls this after event handlers execute.
//
// The budget parameter is optional (can be nil). When provided, effects are
// checked against the per-tick effect budget before running. Effects that
// exceed the budget are re-scheduled for the next tick.
func (o *Owner) RunPendingEffects(budget StormBudgetChecker) {
	if o.disposed.Load() {
		return
	}

	o.pendingEffectsMu.Lock()
	effects := o.pendingEffects
	o.pendingEffects = nil
	o.pendingEffectsMu.Unlock()

	for _, e := range effects {
		if e.pending.Load() {
			// Check storm budget before each effect run
			if budget != nil {
				if err := budget.CheckEffectRun(); err != nil {
					// Budget exceeded - re-schedule for next tick
					if Debug.LogStormBudget {
						println("Storm budget exceeded: re-scheduling effect")
					}
					e.pending.Store(true)
					o.scheduleEffect(e)
					continue
				}
			}
			e.run()
		}
	}

	// Recursively run pending effects on child owners
	o.childrenMu.Lock()
	children := make([]*Owner, len(o.children))
	copy(children, o.children)
	o.childrenMu.Unlock()

	for _, child := range children {
		child.RunPendingEffects(budget)
	}
}

// HasPendingEffects returns true if this owner or any child has pending effects.
func (o *Owner) HasPendingEffects() bool {
	if o.disposed.Load() {
		return false
	}

	o.pendingEffectsMu.Lock()
	hasPending := len(o.pendingEffects) > 0
	o.pendingEffectsMu.Unlock()

	if hasPending {
		return true
	}

	o.childrenMu.Lock()
	children := make([]*Owner, len(o.children))
	copy(children, o.children)
	o.childrenMu.Unlock()

	for _, child := range children {
		if child.HasPendingEffects() {
			return true
		}
	}

	return false
}

// Dispose disposes this Owner and all its children, effects, and cleanups.
// Children are disposed in reverse order (last created first).
// After disposal, the Owner cannot be used.
func (o *Owner) Dispose() {
	if o.disposed.Swap(true) {
		// Already disposed
		return
	}

	// Remove from parent's children list
	if o.parent != nil {
		o.parent.removeChild(o)
	}

	// Dispose children in reverse order
	o.childrenMu.Lock()
	children := make([]*Owner, len(o.children))
	copy(children, o.children)
	o.children = nil
	o.childrenMu.Unlock()

	for i := len(children) - 1; i >= 0; i-- {
		children[i].Dispose()
	}

	// Dispose effects
	o.effectsMu.Lock()
	effects := o.effects
	o.effects = nil
	o.effectsMu.Unlock()

	for _, e := range effects {
		e.dispose()
	}

	// Run cleanups in reverse order
	o.cleanupsMu.Lock()
	cleanups := o.cleanups
	o.cleanups = nil
	o.cleanupsMu.Unlock()

	for i := len(cleanups) - 1; i >= 0; i-- {
		cleanups[i]()
	}

	// Clear pending effects
	o.pendingEffectsMu.Lock()
	o.pendingEffects = nil
	o.pendingEffectsMu.Unlock()
}

// =============================================================================
// Dev-mode Hook Order Validation
// =============================================================================

// StartRender is called at the beginning of a component render.
// It resets the hook slot index for stable identity, and in debug mode,
// also resets the hook order validation index.
func (o *Owner) StartRender() {
	// Track render phase for hook-slot semantics
	beginRender()

	// Always reset slot index for stable hook identity
	o.hookSlotIdx = 0

	// Debug mode: also reset order validation index
	if DebugMode {
		o.hookIndex = 0
	}
}

// EndRender is called at the end of a component render.
// In debug mode, it validates that all expected hooks were called.
func (o *Owner) EndRender() {
	// End render phase tracking
	endRender()

	if !DebugMode {
		return
	}
	if o.renderCount == 0 {
		// First render complete, lock in hook order
		o.renderCount = 1
	} else if o.hookIndex < len(o.hookOrder) {
		panic(fmt.Sprintf("[VANGO E002] Hook order changed: expected %d hooks, got %d",
			len(o.hookOrder), o.hookIndex))
	}
}

// TrackHook records a hook call during render for order validation.
// In debug mode, it validates that hooks are called in the same order
// on every render. Violations cause a panic with a descriptive error.
func (o *Owner) TrackHook(ht HookType) {
	if !DebugMode {
		return
	}

	if o.renderCount == 0 {
		// First render: record hook order
		o.hookOrder = append(o.hookOrder, hookRecord{Type: ht})
	} else {
		// Subsequent renders: validate order
		if o.hookIndex >= len(o.hookOrder) {
			panic(fmt.Sprintf("[VANGO E002] Hook order changed: extra %s hook at index %d",
				ht, o.hookIndex))
		}
		expected := o.hookOrder[o.hookIndex]
		if expected.Type != ht {
			panic(fmt.Sprintf("[VANGO E002] Hook order changed at index %d: expected %s, got %s",
				o.hookIndex, expected.Type, ht))
		}
	}
	o.hookIndex++
}

// =============================================================================
// Hook Slot Storage for Stable Identity
// =============================================================================

// UseHookSlot returns the stored value for the current hook slot,
// or stores and returns the initial value on first render.
// This provides stable identity for hooks (like URLParam, Resource) across renders.
//
// Usage pattern:
//
//	func SomeHook[T any]() *T {
//	    slot := owner.UseHookSlot(nil)
//	    if slot != nil {
//	        return slot.(*T)  // Subsequent render: return stored instance
//	    }
//	    instance := &T{...}  // First render: create new instance
//	    owner.SetHookSlot(instance)
//	    return instance
//	}
func (o *Owner) UseHookSlot() any {
	idx := o.hookSlotIdx
	o.hookSlotIdx++

	if idx < len(o.hookSlots) {
		// Subsequent render: return stored value
		return o.hookSlots[idx]
	}

	// First render: no slot yet, return nil
	// Caller should create the value and call SetHookSlot
	return nil
}

// SetHookSlot stores a value in the current hook slot.
// Must be called after UseHookSlot returns nil (first render).
func (o *Owner) SetHookSlot(value any) {
	o.hookSlots = append(o.hookSlots, value)
}
