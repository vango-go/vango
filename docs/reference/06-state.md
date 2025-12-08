# State Reference

Advanced state management patterns.

## Signal Scopes

| Scope | Usage | API |
|-------|-------|-----|
| **Signal** | One component instance | `vango.Signal(0)` |
| **SharedSignal** | One user session | `vango.SharedSignal(0)` |
| **GlobalSignal** | All users | `vango.GlobalSignal(0)` |

## SharedSignal

State shared across components in the same session:

```go
// store/cart.go
var CartItems = vango.SharedSignal([]CartItem{})

// Any component in the session
items := CartItems()
CartItems.Update(func(items []CartItem) []CartItem {
    return append(items, newItem)
})
```

## GlobalSignal

State shared across ALL users (real-time features):

```go
var OnlineUsers = vango.GlobalSignal([]User{})

// Updates broadcast to all connected clients
OnlineUsers.Update(func(users []User) []User {
    return append(users, currentUser)
})
```

## Persistence

```go
// Browser session storage (per tab)
state := vango.Signal(State{}).Persist(vango.SessionStorage, "key")

// Browser local storage
prefs := vango.Signal(Prefs{}).Persist(vango.LocalStorage, "prefs")

// Server database
settings := vango.Signal(Settings{}).Persist(vango.Database, "user:123:settings")
```

## Immutable Update Helpers

```go
// Slices
items.Update(vango.Append(newItem))
items.Update(vango.Prepend(newItem))
items.Update(vango.RemoveAt(index))
items.Update(vango.UpdateAt(index, newValue))

// Maps
data.Update(vango.SetKey("key", value))
data.Update(vango.RemoveKey("key"))
```

## Batching

```go
vango.Batch(func() {
    firstName.Set("John")
    lastName.Set("Doe")
    age.Set(30)
    // Single re-render
})
```
