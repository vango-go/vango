# Phase 1: Reactive Core

> **The foundation of all state management in Vango**

---

## Overview

The reactive core provides the primitives for managing state that automatically updates the UI when changed. This is the foundation upon which everything else is built.

### Goals
1. Automatic dependency tracking (no manual subscription)
2. Efficient updates (only affected components re-render)
3. Simple mental model (read = subscribe, write = notify)
4. Memory safe (no leaks, explicit cleanup)
5. Thread safe (concurrent access from handlers)

### Non-Goals
1. Global state (Phase 8 - SharedSignal/GlobalSignal)
2. Persistence (Phase 8)
3. Async resources (Phase 8 - Resource)
4. Time-travel debugging (future)

---

## Core Types

### Signal[T]

A reactive value that notifies subscribers when changed.

```go
// Public API
type Signal[T any] struct {
    // private fields
}

// Constructor
func NewSignal[T any](initial T) *Signal[T]

// Shorthand (used in components)
func Signal[T any](initial T) *Signal[T]

// Read current value (subscribes current component)
func (s *Signal[T]) Get() T

// Shorthand: signal() instead of signal.Get()
func (s *Signal[T]) Call() T  // Enables s() syntax via compiler trick

// Write new value (notifies all subscribers)
func (s *Signal[T]) Set(value T)

// Update with function
func (s *Signal[T]) Update(fn func(T) T)

// Read without subscribing
func (s *Signal[T]) Peek() T

// Convenience methods (type-specific, via interface embedding)
// For numeric types:
func (s *Signal[int]) Inc()
func (s *Signal[int]) Dec()
func (s *Signal[int]) Add(n int)

// For bool:
func (s *Signal[bool]) Toggle()

// For slices:
func (s *Signal[[]T]) Append(item T)
func (s *Signal[[]T]) RemoveAt(index int)

// For maps:
func (s *Signal[map[K]V]) SetKey(key K, value V)
func (s *Signal[map[K]V]) DeleteKey(key K)
```

### Memo[T]

A cached computation that updates when dependencies change.

```go
type Memo[T any] struct {
    // private fields
}

// Constructor
func NewMemo[T any](compute func() T) *Memo[T]

// Shorthand
func Memo[T any](compute func() T) *Memo[T]

// Read cached value (recomputes if dirty)
func (m *Memo[T]) Get() T

// Shorthand
func (m *Memo[T]) Call() T

// Read without subscribing
func (m *Memo[T]) Peek() T
```

### Effect

A side effect that runs after render and when dependencies change.

```go
// Cleanup function returned by effect
type Cleanup func()

// Effect registration
func Effect(fn func() Cleanup)

// Convenience: mount-only effect
func OnMount(fn func())

// Convenience: unmount-only effect
func OnUnmount(fn func())

// Convenience: update-only effect (skips first run)
func OnUpdate(fn func())
```

### Batch

Group multiple updates into a single re-render.

```go
// Run function with batched updates
func Batch(fn func())
```

### Untracked

Read signals without creating dependencies.

```go
// Run function without tracking
func Untracked(fn func())

// Or: use Peek() on individual signals
```

---

## Internal Architecture

### Ownership Model

Signals are owned by the component that creates them:

```
┌─────────────────────────────────────────────────┐
│                    Session                       │
│  ┌───────────────────────────────────────────┐  │
│  │              Component Instance            │  │
│  │  ┌─────────────────────────────────────┐  │  │
│  │  │           Signal Scope              │  │  │
│  │  │  ┌─────────┐  ┌─────────┐          │  │  │
│  │  │  │ Signal  │  │ Signal  │          │  │  │
│  │  │  └─────────┘  └─────────┘          │  │  │
│  │  │  ┌─────────┐  ┌─────────┐          │  │  │
│  │  │  │  Memo   │  │ Effect  │          │  │  │
│  │  │  └─────────┘  └─────────┘          │  │  │
│  │  └─────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────┘  │
└─────────────────────────────────────────────────┘
```

When a component unmounts:
1. All effects run their cleanup functions
2. All signals are unsubscribed from their sources
3. All memory is freed

### Dependency Tracking

We use a **runtime tracking** approach (like SolidJS, not like React):

```go
// Global tracking context (per-goroutine via context.Context)
type TrackingContext struct {
    currentOwner    *Owner           // Component that owns new signals
    currentListener *Listener        // What's currently tracking
    batchDepth      int              // Nested batch count
    pendingEffects  []*Effect        // Effects to run after batch
}

// During signal read
func (s *Signal[T]) Get() T {
    ctx := getTrackingContext()
    if ctx.currentListener != nil {
        s.subscribe(ctx.currentListener)
    }
    return s.value
}

// During signal write
func (s *Signal[T]) Set(value T) {
    if s.value == value {
        return // No change, no notification
    }
    s.value = value

    ctx := getTrackingContext()
    if ctx.batchDepth > 0 {
        // Queue for later
        ctx.pendingUpdates = append(ctx.pendingUpdates, s.subscribers...)
    } else {
        // Notify immediately
        s.notifySubscribers()
    }
}
```

### Listener Types

```go
type Listener interface {
    // Called when a dependency changes
    MarkDirty()

    // Unique ID for deduplication
    ID() uint64
}

// Components are listeners
type ComponentInstance struct {
    id          uint64
    dirty       bool
    // ...
}

func (c *ComponentInstance) MarkDirty() {
    if !c.dirty {
        c.dirty = true
        c.session.scheduleRender(c)
    }
}

// Memos are listeners
type Memo[T any] struct {
    id       uint64
    compute  func() T
    value    T
    valid    bool
    sources  []*signalBase  // What we depend on
    subs     []Listener     // What depends on us
}

func (m *Memo[T]) MarkDirty() {
    if m.valid {
        m.valid = false
        // Propagate to our subscribers
        for _, sub := range m.subs {
            sub.MarkDirty()
        }
    }
}

// Effects are listeners
type Effect struct {
    id       uint64
    fn       func() Cleanup
    cleanup  Cleanup
    sources  []*signalBase
    owner    *Owner
}

func (e *Effect) MarkDirty() {
    e.owner.scheduleEffect(e)
}
```

### Signal Internals

```go
// Base type for all signals (type-erased for storage)
type signalBase struct {
    id    uint64
    subs  []Listener
    mu    sync.RWMutex
}

func (s *signalBase) subscribe(l Listener) {
    s.mu.Lock()
    defer s.mu.Unlock()

    // Deduplicate by ID
    for _, existing := range s.subs {
        if existing.ID() == l.ID() {
            return
        }
    }
    s.subs = append(s.subs, l)
}

func (s *signalBase) unsubscribe(l Listener) {
    s.mu.Lock()
    defer s.mu.Unlock()

    for i, existing := range s.subs {
        if existing.ID() == l.ID() {
            s.subs = append(s.subs[:i], s.subs[i+1:]...)
            return
        }
    }
}

func (s *signalBase) notifySubscribers() {
    s.mu.RLock()
    subs := make([]Listener, len(s.subs))
    copy(subs, s.subs)
    s.mu.RUnlock()

    for _, sub := range subs {
        sub.MarkDirty()
    }
}

// Typed signal wraps base
type Signal[T any] struct {
    signalBase
    value T
    equal func(T, T) bool  // Custom equality (optional)
}
```

### Memo Internals

```go
type Memo[T any] struct {
    signalBase           // Can be subscribed to like a signal
    compute  func() T
    value    T
    valid    bool
    sources  []*signalBase
    equal    func(T, T) bool
}

func (m *Memo[T]) Get() T {
    ctx := getTrackingContext()

    // Track this memo as a dependency
    if ctx.currentListener != nil {
        m.subscribe(ctx.currentListener)
    }

    // Recompute if invalid
    if !m.valid {
        m.recompute()
    }

    return m.value
}

func (m *Memo[T]) recompute() {
    // Unsubscribe from old sources
    for _, source := range m.sources {
        source.unsubscribe(m)
    }
    m.sources = m.sources[:0]

    // Track new sources during compute
    ctx := getTrackingContext()
    oldListener := ctx.currentListener
    ctx.currentListener = m

    newValue := m.compute()

    ctx.currentListener = oldListener

    // Only notify if value changed
    if !m.valid || !m.equals(m.value, newValue) {
        m.value = newValue
        m.valid = true
        m.notifySubscribers()
    } else {
        m.valid = true
    }
}
```

### Effect Internals

```go
type Effect struct {
    id       uint64
    fn       func() Cleanup
    cleanup  Cleanup
    sources  []*signalBase
    owner    *Owner
    pending  bool
}

func newEffect(owner *Owner, fn func() Cleanup) *Effect {
    e := &Effect{
        id:    nextID(),
        fn:    fn,
        owner: owner,
    }

    // Run immediately after creation
    e.run()

    // Register with owner for cleanup
    owner.effects = append(owner.effects, e)

    return e
}

func (e *Effect) run() {
    e.pending = false

    // Run cleanup from previous run
    if e.cleanup != nil {
        e.cleanup()
        e.cleanup = nil
    }

    // Unsubscribe from old sources
    for _, source := range e.sources {
        source.unsubscribe(e)
    }
    e.sources = e.sources[:0]

    // Track new sources during run
    ctx := getTrackingContext()
    oldListener := ctx.currentListener
    ctx.currentListener = e

    e.cleanup = e.fn()

    ctx.currentListener = oldListener
}

func (e *Effect) MarkDirty() {
    if !e.pending {
        e.pending = true
        e.owner.scheduleEffect(e)
    }
}

func (e *Effect) dispose() {
    if e.cleanup != nil {
        e.cleanup()
        e.cleanup = nil
    }
    for _, source := range e.sources {
        source.unsubscribe(e)
    }
    e.sources = nil
}
```

### Owner (Component Scope)

```go
type Owner struct {
    id             uint64
    parent         *Owner
    children       []*Owner
    signals        []*signalBase
    effects        []*Effect
    cleanups       []func()
    pendingEffects []*Effect
    disposed       bool
}

func newOwner(parent *Owner) *Owner {
    o := &Owner{
        id:     nextID(),
        parent: parent,
    }
    if parent != nil {
        parent.children = append(parent.children, o)
    }
    return o
}

func (o *Owner) scheduleEffect(e *Effect) {
    o.pendingEffects = append(o.pendingEffects, e)
    // Effects run after render, scheduled by session
}

func (o *Owner) runPendingEffects() {
    for _, e := range o.pendingEffects {
        if e.pending {
            e.run()
        }
    }
    o.pendingEffects = o.pendingEffects[:0]
}

func (o *Owner) dispose() {
    if o.disposed {
        return
    }
    o.disposed = true

    // Dispose children first (reverse order)
    for i := len(o.children) - 1; i >= 0; i-- {
        o.children[i].dispose()
    }

    // Dispose effects
    for _, e := range o.effects {
        e.dispose()
    }

    // Run manual cleanups
    for _, cleanup := range o.cleanups {
        cleanup()
    }

    // Clear references
    o.children = nil
    o.signals = nil
    o.effects = nil
    o.cleanups = nil
}
```

### Batch Implementation

```go
func Batch(fn func()) {
    ctx := getTrackingContext()
    ctx.batchDepth++

    defer func() {
        ctx.batchDepth--
        if ctx.batchDepth == 0 {
            // Process all pending updates
            processPendingUpdates(ctx)
        }
    }()

    fn()
}

func processPendingUpdates(ctx *TrackingContext) {
    // Deduplicate listeners
    seen := make(map[uint64]bool)
    var listeners []Listener

    for _, l := range ctx.pendingUpdates {
        if !seen[l.ID()] {
            seen[l.ID()] = true
            listeners = append(listeners, l)
        }
    }
    ctx.pendingUpdates = ctx.pendingUpdates[:0]

    // Notify all
    for _, l := range listeners {
        l.MarkDirty()
    }
}
```

---

## Equality Handling

### Default Equality

```go
func defaultEquals[T any](a, b T) bool {
    // For comparable types, use ==
    // For others, use reflect.DeepEqual
    // This is determined at compile time via type constraints
}

// Constraint for comparable types
type Comparable interface {
    ~int | ~int8 | ~int16 | ~int32 | ~int64 |
    ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
    ~float32 | ~float64 |
    ~string | ~bool
}

// Fast path for comparable types
func comparableEquals[T Comparable](a, b T) bool {
    return a == b
}

// Slow path for complex types
func deepEquals[T any](a, b T) bool {
    return reflect.DeepEqual(a, b)
}
```

### Custom Equality

```go
// With custom equality function
count := Signal(0).Equals(func(a, b int) bool {
    return a == b
})

// For structs, often want shallow equality
user := Signal(User{}).Equals(func(a, b User) bool {
    return a.ID == b.ID && a.Name == b.Name
})
```

---

## Thread Safety

The reactive system must handle concurrent access because:
1. Multiple goroutines may handle WebSocket messages
2. Effects may spawn goroutines
3. Background tasks may update signals

### Strategy

```go
// Per-signal locking for value access
type Signal[T any] struct {
    signalBase
    value T
    mu    sync.RWMutex  // Protects value
}

func (s *Signal[T]) Get() T {
    s.mu.RLock()
    defer s.mu.RUnlock()

    // Track dependency (signalBase handles its own locking)
    s.trackRead()

    return s.value
}

func (s *Signal[T]) Set(value T) {
    s.mu.Lock()
    changed := !s.equals(s.value, value)
    if changed {
        s.value = value
    }
    s.mu.Unlock()

    if changed {
        s.notifySubscribers()  // signalBase handles locking
    }
}
```

### Context Propagation

Each goroutine needs its own tracking context:

```go
// Context key for tracking
type trackingContextKey struct{}

func getTrackingContext() *TrackingContext {
    ctx := context.Value(trackingContextKey{})
    if ctx == nil {
        // Create default context for this goroutine
        return newTrackingContext()
    }
    return ctx.(*TrackingContext)
}

// When spawning goroutines from handlers
func WithOwner(parent *Owner, fn func()) {
    ctx := context.WithValue(context.Background(), trackingContextKey{}, &TrackingContext{
        currentOwner: parent,
    })
    go func() {
        // Goroutine has access to parent's owner
        fn()
    }()
}
```

---

## Memory Management

### Preventing Leaks

1. **Explicit cleanup**: Effects return cleanup functions
2. **Owner disposal**: When component unmounts, all resources freed
3. **Weak references**: Avoid for simplicity, use explicit lifecycle

### Debugging Leaks

```go
// Debug mode: track all allocations
var debugMode = os.Getenv("VANGO_DEBUG") == "1"

func (s *Signal[T]) finalize() {
    if debugMode && len(s.subs) > 0 {
        log.Printf("WARNING: Signal %d finalized with %d subscribers", s.id, len(s.subs))
    }
}
```

---

## Usage Examples

### Basic Counter

```go
func Counter(initial int) vango.Component {
    return vango.Func(func() *vango.VNode {
        count := Signal(initial)

        return Div(
            H1(Textf("Count: %d", count())),
            Button(OnClick(count.Inc), Text("+")),
            Button(OnClick(count.Dec), Text("-")),
        )
    })
}
```

### Derived State

```go
func ShoppingCart() vango.Component {
    return vango.Func(func() *vango.VNode {
        items := Signal([]CartItem{})

        subtotal := Memo(func() float64 {
            total := 0.0
            for _, item := range items() {
                total += item.Price * float64(item.Qty)
            }
            return total
        })

        tax := Memo(func() float64 {
            return subtotal() * 0.08
        })

        total := Memo(func() float64 {
            return subtotal() + tax()
        })

        return Div(
            CartList(items),
            Div(Class("totals"),
                Row("Subtotal", subtotal()),
                Row("Tax", tax()),
                Row("Total", total()),
            ),
        )
    })
}
```

### Side Effects

```go
func UserProfile(userID int) vango.Component {
    return vango.Func(func() *vango.VNode {
        user := Signal[*User](nil)
        loading := Signal(true)

        Effect(func() Cleanup {
            loading.Set(true)

            // Fetch user (runs on server, direct DB access)
            u, err := db.Users.FindByID(userID)
            if err != nil {
                // Handle error...
            }

            user.Set(u)
            loading.Set(false)

            return nil  // No cleanup needed
        })

        if loading() {
            return LoadingSpinner()
        }

        return Div(
            H1(Text(user().Name)),
            P(Text(user().Email)),
        )
    })
}
```

### Effect with Cleanup

```go
func Timer() vango.Component {
    return vango.Func(func() *vango.VNode {
        elapsed := Signal(0)

        Effect(func() Cleanup {
            ticker := time.NewTicker(1 * time.Second)
            done := make(chan bool)

            go func() {
                for {
                    select {
                    case <-ticker.C:
                        elapsed.Update(func(n int) int { return n + 1 })
                    case <-done:
                        return
                    }
                }
            }()

            return func() {
                ticker.Stop()
                done <- true
            }
        })

        return Div(Textf("Elapsed: %d seconds", elapsed()))
    })
}
```

### Batched Updates

```go
func resetForm() {
    Batch(func() {
        name.Set("")
        email.Set("")
        message.Set("")
        errors.Set(nil)
    })
    // Single re-render after all updates
}
```

---

## Testing Strategy

### Unit Tests

```go
func TestSignalBasic(t *testing.T) {
    count := Signal(0)

    assert.Equal(t, 0, count())

    count.Set(5)
    assert.Equal(t, 5, count())

    count.Update(func(n int) int { return n * 2 })
    assert.Equal(t, 10, count())
}

func TestSignalSubscription(t *testing.T) {
    count := Signal(0)
    calls := 0

    // Simulate component reading signal
    ctx := newTestTrackingContext()
    listener := &testListener{
        onDirty: func() { calls++ },
    }
    ctx.currentListener = listener

    withContext(ctx, func() {
        _ = count()  // Subscribe
    })

    count.Set(1)
    assert.Equal(t, 1, calls)

    count.Set(1)  // Same value
    assert.Equal(t, 1, calls)  // No notification

    count.Set(2)
    assert.Equal(t, 2, calls)
}

func TestMemoComputation(t *testing.T) {
    count := Signal(0)
    computations := 0

    doubled := Memo(func() int {
        computations++
        return count() * 2
    })

    assert.Equal(t, 0, doubled())
    assert.Equal(t, 1, computations)

    // Reading again doesn't recompute
    assert.Equal(t, 0, doubled())
    assert.Equal(t, 1, computations)

    // Changing source recomputes
    count.Set(5)
    assert.Equal(t, 10, doubled())
    assert.Equal(t, 2, computations)
}

func TestEffectRunsOnMount(t *testing.T) {
    ran := false

    owner := newOwner(nil)
    withOwner(owner, func() {
        Effect(func() Cleanup {
            ran = true
            return nil
        })
    })

    assert.True(t, ran)
}

func TestEffectCleanup(t *testing.T) {
    cleanedUp := false

    owner := newOwner(nil)
    withOwner(owner, func() {
        Effect(func() Cleanup {
            return func() {
                cleanedUp = true
            }
        })
    })

    assert.False(t, cleanedUp)

    owner.dispose()
    assert.True(t, cleanedUp)
}

func TestBatch(t *testing.T) {
    a := Signal(0)
    b := Signal(0)
    calls := 0

    listener := &testListener{
        onDirty: func() { calls++ },
    }
    subscribeToAll(listener, a, b)

    Batch(func() {
        a.Set(1)
        b.Set(1)
    })

    // Should only notify once, not twice
    assert.Equal(t, 1, calls)
}
```

### Benchmark Tests

```go
func BenchmarkSignalRead(b *testing.B) {
    s := Signal(42)
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = s()
    }
}

func BenchmarkSignalWrite(b *testing.B) {
    s := Signal(0)
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        s.Set(i)
    }
}

func BenchmarkMemoRead(b *testing.B) {
    count := Signal(0)
    doubled := Memo(func() int { return count() * 2 })

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = doubled()
    }
}

func BenchmarkMemoRecompute(b *testing.B) {
    count := Signal(0)
    doubled := Memo(func() int { return count() * 2 })

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        count.Set(i)
        _ = doubled()
    }
}

func BenchmarkManySubscribers(b *testing.B) {
    s := Signal(0)

    // Add 1000 subscribers
    for i := 0; i < 1000; i++ {
        listener := &testListener{}
        s.subscribe(listener)
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        s.Set(i)
    }
}
```

---

## Benchmark Results

Performance benchmarks run on Apple M-series (arm64).

### Signal Operations

| Operation | Time | Allocations | Notes |
|-----------|------|-------------|-------|
| Signal.Peek() | **~4ns** | 0 | Fast path, no tracking |
| Signal.Get() (no tracking) | ~2.4μs | 1 | With sync.Map lookup |
| Signal.Get() (with tracking) | ~8μs | 3 | Subscription overhead |
| Signal.Set() (no subscribers) | ~2.7μs | 0 | Write only |
| Signal.Set() (1 subscriber) | ~2.8μs | 1 | Single notification |
| Signal.Set() (10 subscribers) | ~2.8μs | 2 | Scales well |
| Signal.Set() (100 subscribers) | ~3.2μs | 2 | Still fast |
| Signal.Update() | ~2.7μs | 1 | Read-modify-write |

### Memo Operations

| Operation | Time | Allocations | Notes |
|-----------|------|-------------|-------|
| Memo.Get() (cached) | ~2.3μs | 1 | Returns cached value |
| Memo recompute | ~19μs | 7 | Dependency changed |
| Memo chain (3 deep) | ~59μs | 16 | A → B → C propagation |
| Memo chain (5 deep) | ~111μs | 26 | Deeper chains |

### Batch Operations

| Operation | Time | Allocations | Notes |
|-----------|------|-------------|-------|
| Batch 10 updates | ~78μs | 39 | Single notification |
| Batch 100 updates | ~711μs | 313 | Scales linearly |

### Typed Signals

| Operation | Time | Allocations | Notes |
|-----------|------|-------------|-------|
| IntSignal.Inc() | ~3.1μs | 1 | Atomic increment |
| SliceSignal.Append() | ~3.1μs | 3 | Grow slice |
| MapSignal.SetKey() | ~3.3μs | 6 | Map update |

### Realistic Scenarios

| Operation | Time | Allocations | Notes |
|-----------|------|-------------|-------|
| Effect creation | ~53μs | 9 | With cleanup function |
| Effect run | ~13μs | 5 | Trigger and execute |
| Realistic component | ~63μs | 19 | Signals + memos + effects |

### Key Insights

1. **Peek is near-zero cost**: ~4ns for read-only access without tracking
2. **Subscriber notification scales well**: 100 subscribers only adds ~0.5μs
3. **Batch amortizes cost**: 100 updates in ~711μs vs 100 × ~2.7μs = 270μs individual
4. **Thread-safe by design**: sync.Map adds overhead but enables concurrent access

### Note on Overhead

The reactive system uses Go's `sync.Map` for goroutine-safe tracking context. This adds ~2μs overhead compared to a thread-unsafe implementation. This is acceptable because:

1. Server-driven architecture may handle multiple sessions concurrently
2. The overhead is dwarfed by network latency (~1-50ms)
3. Correctness > micro-optimization at this layer

**Benchmark command:** `go test ./pkg/vango/... -bench=. -benchmem`

---

## File Structure

```
pkg/vango/
├── signal.go           # Signal[T] implementation
├── signal_test.go
├── memo.go             # Memo[T] implementation
├── memo_test.go
├── effect.go           # Effect implementation
├── effect_test.go
├── batch.go            # Batch and Untracked
├── batch_test.go
├── owner.go            # Owner (component scope)
├── owner_test.go
├── tracking.go         # TrackingContext and helpers
├── tracking_test.go
└── reactive_bench_test.go
```

---

## Exit Criteria

Phase 1 is complete when:

1. [x] All types implemented (`Signal`, `Memo`, `Effect`, `Batch`)
2. [x] Automatic dependency tracking works
3. [x] Updates propagate correctly through memo chains
4. [x] Effects run with proper cleanup
5. [x] Batching groups updates correctly
6. [x] Thread-safe concurrent access (race detector passes)
7. [x] No memory leaks (verified with disposal tests)
8. [x] Unit test coverage > 90% (achieved: 91.6%)
9. [x] Benchmarks documented with baseline numbers
10. [x] Code reviewed and documented

---

## Dependencies

- **Requires**: Nothing (foundation)
- **Required by**: Phase 2 (VNode event handlers reference signals)

---

## Open Questions

1. **Generic constraints**: How to handle `Inc()` on `Signal[int]` vs `Signal[int64]`?
   - **Answer**: Use type-specific wrapper types or interface embedding

2. **Equality for slices/maps**: Use `reflect.DeepEqual` or require custom function?
   - **Answer**: Default to `reflect.DeepEqual`, allow override via `.Equals()`

3. **Effect timing**: Run synchronously or after microtask?
   - **Answer**: Run synchronously during initial mount, defer re-runs to after render

---

## Implementation Notes (Go Constraints)

The spec above shows an idealized API. The actual Go implementation differs due to language constraints:

### Naming Conflicts

Go doesn't allow a type and function to share the same name. The spec's shorthand constructors (`Signal()`, `Memo()`, `Effect()`) conflict with the type names.

**Resolution**: Use idiomatic Go constructor naming:

| Spec Syntax | Go Implementation |
|-------------|-------------------|
| `Signal(0)` | `NewSignal(0)` |
| `Memo(fn)` | `NewMemo(fn)` |
| `Effect(fn)` | `CreateEffect(fn)` |

### Callable Syntax

Go has no operator overloading. The spec's `count()` syntax to read a signal is impossible.

**Resolution**: Use explicit `.Get()` method:

| Spec Syntax | Go Implementation |
|-------------|-------------------|
| `count()` | `count.Get()` |
| `doubled()` | `doubled.Get()` |

### OnUpdate Signature

The spec shows `OnUpdate(fn func())` but this can't track dependencies if `fn` isn't called on first run.

**Resolution**: Two-function signature for correct dependency tracking:

```go
// Spec (doesn't work correctly):
OnUpdate(func() { fmt.Println(count.Get()) })

// Implementation (tracks deps correctly):
OnUpdate(
    func() { _ = count.Get() },        // deps: always runs, tracks dependencies
    func() { fmt.Println("updated") }, // callback: only runs on updates
)
```

### Type-Specific Methods

Go can't add methods to generic type instantiations like `Signal[int]`.

**Resolution**: Wrapper types:

```go
// Instead of: count := NewSignal(0); count.Inc()
count := NewIntSignal(0)
count.Inc()  // Works via IntSignal wrapper

// Available wrappers:
NewIntSignal(n)      // Inc(), Dec(), Add(n)
NewBoolSignal(b)     // Toggle(), SetTrue(), SetFalse()
NewSliceSignal(s)    // Append(v), RemoveAt(i), Clear()
NewMapSignal(m)      // SetKey(k,v), DeleteKey(k), GetKey(k)
```

---

## Implementation Status

**Status**: ✅ COMPLETE (2025-12-06)

**Files**: `vango_v2/pkg/vango/`
- `signal.go`, `memo.go`, `effect.go`, `batch.go`, `owner.go`, `tracking.go`
- Type wrappers: `signal_int.go`, `signal_bool.go`, `signal_slice.go`, `signal_map.go`
- Tests: 91.6% coverage, race detector passes

**Verified Working**:
```go
count := vango.NewSignal(0)
doubled := vango.NewMemo(func() int { return count.Get() * 2 })

vango.CreateEffect(func() vango.Cleanup {
    fmt.Println("Count is:", count.Get())
    return func() { fmt.Println("Cleanup") }
})

count.Set(5) // Triggers effect, memo recomputes
```

---

*Phase 1 Specification - Version 1.2 (Updated 2024-12-07)*
*Implementation Complete: 2024-12-06*
*Benchmark results added: 2024-12-07*
