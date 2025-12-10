# Common Pitfalls

Mistakes developers commonly make when building Vango applications—and how to avoid them.

## Component Re-creation in Render()

**Problem:** Creating child components inside `Render()` causes new instances on every re-render, breaking signal subscriptions and spawning duplicate goroutines.

```go
// ❌ BAD: New instance every render loses signal connections
func (r *Root) Render() *VNode {
    dash := NewDashboard(...)  // Created fresh every time!
    return dash.Render()
}

// ✅ GOOD: Use methods on the parent component
func (r *Root) Render() *VNode {
    return r.renderDashboard()  // Method reuses parent's state
}

// ✅ GOOD: Create child components once in constructor
func NewRoot() *Root {
    r := &Root{
        dashboard: NewDashboard(...),  // Created once
    }
    return r
}
```

---

## Conditional Signal Reads

**Problem:** Reading signals conditionally causes subscription state to change between renders, leading to missed updates.

```go
// ❌ BAD: Subscription changes per render
if showDetails {
    details := detailsSignal()  // Only subscribes sometimes!
}

// ✅ GOOD: Always read, conditionally use
details := detailsSignal()  // Always subscribe
if showDetails {
    // use details
}
```

---

## Data() vs DataAttr()

**Problem:** `Data()` and `DataAttr()` do different things—one creates an HTML element, the other creates attributes.

```go
// Data() creates a data-* attribute
Data("id", "123")           // → data-id="123"

// DataElement() creates the <data> HTML element (rare)
DataElement(Value("123"))   // → <data value="123">...</data>

// DataAttr() is an alias for Data() - both work
DataAttr("user-id", "42")   // → data-user-id="42"
```

> [!TIP]
> Most developers want `Data()` for data attributes. The `<data>` HTML element is rarely used.

---

## Nullable Pointer Access in If()

**Problem:** `If()` is a Go function—all arguments are evaluated *before* the function runs. This causes nil pointer panics when accessing nullable fields.

```go
// ❌ PANIC: card.DueDate.Format() is called even when DueDate is nil!
If(card.DueDate != nil,
    Span(Text(card.DueDate.Format("Jan 2"))),  // Evaluated before If() runs
)

// ✅ CORRECT: Use When() for lazy evaluation
When(card.DueDate != nil, func() *VNode {
    return Span(Text(card.DueDate.Format("Jan 2")))  // Only called when not nil
})

// ✅ OR: Precompute values outside the VNode
hasDue := card.DueDate != nil
dueStr := ""
if hasDue {
    dueStr = card.DueDate.Format("Jan 2")
}
return If(hasDue, Span(Text(dueStr)))
```

> [!WARNING]
> This applies to all VDOM helpers: `If()`, `ClassIf()`, `Style()`, `fmt.Sprintf()` inside arguments, etc. Use `When()`, `IfLazy()`, or `ShowWhen()` when the content depends on the condition being true.

---

## Third-Party Library DOM Conflicts

**Problem:** Libraries like SortableJS manipulate the DOM directly, conflicting with Vango's virtual DOM patches.

**Symptoms:**
- Duplicate elements after drag operations
- Elements "jumping back" to old positions
- Ghost elements appearing

**Solutions:**
1. **Isolate hook zones**: Keep Vango-managed elements (buttons, inputs) outside sortable containers
2. **Use `data-*` attributes**: Libraries should identify elements by `data-id`, not internal HIDs

```go
// Keep "Add Card" outside the sortable zone
Div(Class("column"),
    // Sortable zone - managed by SortableJS
    Div(
        Class("cards-container"),
        hooks.Hook("Sortable", config),
        Range(cards, renderCard),
    ),
    // Static zone - managed by Vango only
    Button(Text("+ Add Card"), OnClick(addCard)),
)
```

---

## Hook Handler Type Signatures

**Problem:** Using the wrong `HookEvent` type causes silent handler failures—no errors, no logs.

```go
// ✅ CORRECT: Use hooks.HookEvent from pkg/features/hooks
import "github.com/vango-dev/vango/v2/pkg/features/hooks"

hooks.OnEvent("onreorder", func(e hooks.HookEvent) {
    id := e.String("id")
    toIndex := e.Int("toIndex")
})

// ❌ WRONG: Different HookEvent type causes silent no-op
import "github.com/vango-dev/vango/v2/pkg/server"
hooks.OnEvent("onreorder", func(e server.HookEvent) { ... })  // Never called!
```

> [!WARNING]
> If your hook handler never fires, check that you're importing `hooks.HookEvent`, not `server.HookEvent`.

---

## Navigation Without File-Based Router

**Problem:** `<a href>` tags don't automatically work for SPA navigation without the full router configured.

**Options:**

**1. Use `Link()` helper** (recommended):
```go
Link("/settings", Text("Settings"))  // Client-side navigation with URL update
```

**2. Manual signal-based routing**:
```go
type App struct {
    path *vango.Signal[string]
}

func (a *App) Render() *VNode {
    switch a.path.Get() {
    case "/":
        return a.renderHome()
    case "/settings":
        return a.renderSettings()
    default:
        return a.render404()
    }
}
```

> [!NOTE]
> Manual routing updates state but not the browser URL. Use `Link()` or the full file-based router for proper URL synchronization.

---

## Background Signal Races

**Problem:** Updating signals in goroutines started during component construction can race with SSR render.

```go
// ❌ BAD: Race between SSR render and goroutine
func NewDashboard() *Dashboard {
    d := &Dashboard{
        data: vango.NewSignal(nil),
    }
    go d.loadData()  // May complete AFTER SSR!
    return d
}

// ✅ GOOD: Initialize with loading state, update after mount
func NewDashboard() *Dashboard {
    d := &Dashboard{
        data:    vango.NewSignal(nil),
        loading: vango.NewSignal(true),
    }
    return d
}

func (d *Dashboard) OnMount() {
    go d.loadData()  // Safe: after hydration
}
```

---

## See Also

- [Components](./03-components.md) — Component model deep dive
- [Signals](../reference/02-signals.md) — Reactivity reference
- [Hooks](../reference/05-hooks.md) — Client-side interaction patterns
