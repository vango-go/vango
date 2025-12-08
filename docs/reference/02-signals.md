# Signals Reference

Complete API for Vango's reactive primitives.

## Signal

```go
// Create
count := vango.Signal(0)
user := vango.Signal[*User](nil)
items := vango.Signal([]Item{})

// Read (subscribes component)
value := count()

// Write
count.Set(5)
count.Update(func(n int) int { return n + 1 })

// Convenience (integers)
count.Inc()
count.Dec()

// Convenience (booleans)
enabled.Toggle()
```

## Memo

```go
// Cached derived value
doubled := vango.Memo(func() int {
    return count() * 2
})

// Chain memos
total := vango.Memo(func() float64 {
    return subtotal() + tax()
})
```

## Effect

```go
vango.Effect(func() vango.Cleanup {
    // Side effect code
    data := fetchData(id)
    dataSignal.Set(data)

    // Optional cleanup
    return func() {
        // Runs before re-run or unmount
    }
})
```

## Resource

Async data loading with loading/error states:

```go
user := vango.Resource(func() (*User, error) {
    return db.Users.FindByID(userID)
})

// Check state
if user.Loading() {
    return Spinner()
}
if err := user.Error(); err != nil {
    return ErrorMessage(err)
}
return Profile(user.Data())
```

## Ref

Reference to DOM elements or values:

```go
inputRef := vango.Ref[js.Value](nil)

Input(Ref(inputRef))

// Later
inputRef.Current().Call("focus")
```

## Context

Pass values down the component tree:

```go
// Create
var ThemeCtx = vango.CreateContext("light")

// Provide
ThemeCtx.Provider("dark",
    ChildComponents(),
)

// Consume
theme := ThemeCtx.Use()  // "dark"
```
