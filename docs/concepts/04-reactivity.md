# Reactivity

Vango uses a fine-grained reactivity system based on **Signals**, **Memos**, and **Effects**.

## Signals

Signals are reactive values. When a signal changes, any component reading it re-renders.

```go
count := vango.Signal(0)   // Create
value := count()           // Read (subscribes)
count.Set(5)               // Write (triggers re-render)
count.Update(func(n int) int { return n + 1 })
```

**Convenience methods:**
```go
count.Inc()       // +1 for integers
count.Dec()       // Convenience (booleans)
enabled.Toggle()  // !current for booleans

// Thread Safety
// Signals are thread-safe and can be updated from any goroutine.
// This triggers a thread-safe update to the component tree.
enabled.Set(true)
```

## Memos

Memos are cached derived values. They recompute only when dependencies change.

```go
items := vango.Signal([]Item{})

total := vango.Memo(func() float64 {
    sum := 0.0
    for _, item := range items() {
        sum += item.Price
    }
    return sum
})
```

Accessing `total()` returns the cached value. It only recalculates when `items()` changes.

## Effects

Effects run side effects after render.

```go
vango.Effect(func() vango.Cleanup {
    // Runs after mount and when dependencies change
    user, _ := db.Users.Find(userID)
    userSignal.Set(user)

    // Optional cleanup (runs before next effect or on unmount)
    return func() { /* cleanup */ }
})
```

| When | What Happens |
|------|--------------|
| After first render | Effect runs |
| Signal dependency changes | Cleanup runs, then effect re-runs |
| Component unmounts | Cleanup runs |

## Background Updates

Vango supports updates triggered by background activities (e.g., streaming responses, timers, external events).

**Behavior:**
- When a Signal is updated from a goroutine, it marks dependent components as "dirty".
- The Runtime's event loop (listening to `renderCh`) picks up the signal and triggers a re-render.
- Patches are pushed to the client immediately.

**Best Practice:**
- Use `vango.Signal` for all shared state.
- Update signals freely from any goroutine (`go func() { ... }`).
- Ensure your background loop checks for context cancellation or channel closure.

## Batching

Group multiple updates into one re-render:

```go
vango.Batch(func() {
    firstName.Set("John")
    lastName.Set("Doe")
    // Only one re-render
})
```

## Anti-Patterns

**Don't:** Read signals conditionally
```go
if condition {
    value := signal()  // Subscription changes per render!
}
```

**Do:** Read unconditionally, use value conditionally
```go
value := signal()  // Always subscribe
if condition {
    // use value
}
```
