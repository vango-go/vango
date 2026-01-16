# Full Developer Guide

## Preface

### Version & Status

* **Document:** Vango Developer Guide
* **Version:** v0.1.0
* **Status:** Complete / Live
* **Last Updated:** 2026-01-07
* **Normative Source of Truth:** `VANGO_ARCHITECTURE_AND_GUIDE.md` (render rules, routing/runtime contracts, wire protocol, thin client behavior)

This guide is **application-focused**: it teaches an LLM or developer how to build Vango apps correctly and idiomatically, without requiring them to understand Vango internals (VDOM encoder/diff, binary protocol layout, session serialization, etc.). When anything here conflicts with the normative spec, **the spec is authoritative**.

### Audience

This guide is written for:

1. **Go backend developers building full web apps**
   You want a single-language stack, direct DB access, and deployment as a single Go binary.

2. **SPA refugees (React/Vue/Next) who want the same UX with less complexity**
   You want modern interactivity without a large client bundle, hydration pitfalls, or a client/server state sync problem.

3. **Server-driven UI adopters (LiveView/Livewire-style) who want Go performance and ergonomics**
   You accept the render-purity constraints and want clear “pit of success” patterns.

### Imports Policy (Normalized)

All examples in this guide assume **only two Vango imports**:

```go
import "github.com/vango-go/vango"
import . "github.com/vango-go/vango/el"
```

(You will still see standard-library imports like `time`, `context`, `fmt` where required.)

### The “Pit of Success” Philosophy

Vango is designed so that the default, straightforward approach yields:

* **Server-side state and logic** (signals/resources/actions run on the server)
* **Client-side feel** (fast incremental updates via patches; SPA navigation when connected)
* **Predictable correctness** through strict contracts:

  * render functions are pure
  * stable hook order
  * single-writer session loop

In practice, Vango aims for the common 80/20 split:

* ~80%: server-driven pages with **no custom client JS**
* ~15%: client hooks for **60fps interactions** (drag/drop, fine-grained UI behavior)
* ~5%: JS/WASM islands for **third-party widgets** or **compute-heavy client work**

---

# Getting Started

This section covers everything you need to set up a Vango development environment.

---

## System Requirements

### Go Version

Vango requires **Go 1.22 or later**. We recommend always using the latest stable Go release.

```bash
# Check your Go version
go version
# go version go1.22.0 darwin/arm64

# Install or update Go: https://go.dev/dl/
```

### Node.js (Not Required)

Vango does not require Node.js for its core features or for the default Tailwind CSS pipeline. Tailwind is managed via a **standalone binary** that Vango downloads and manages automatically.

Node.js is only needed if you explicitly choose to use other JavaScript-based tooling (like custom PostCSS plugins not supported by the standalone binary).

### Operating System

Vango works on macOS, Linux, and Windows. For production deployments, we recommend Linux containers.

---

## Installing the Vango CLI

The Vango CLI provides project scaffolding, development server, code generation, and build tools.

```bash
# Install the CLI
go install github.com/vango-go/vango/cmd/vango@latest

# Verify installation
vango version
# vango v1.x.x
```

**CLI Commands Overview:**

| Command | Purpose |
|---------|---------|
| `vango create` | Scaffold a new project |
| `vango dev` | Start development server with hot reload |
| `vango build` | Build production binary and assets |
| `vango gen routes` | Regenerate route glue code |
| `vango gen component` | Generate component boilerplate |
| `vango gen route` | Generate route file with registration |

---

## Creating Your First App

```bash
# Create with default template (Tailwind + example pages)
vango create myapp
cd myapp

# Start development server
vango dev
# → Opens browser at http://localhost:8080
```

Your app is a standard Go module. The `go.mod` imports the Vango framework:

```go
module myapp

go 1.22

require (
    github.com/vango-go/vango v1.x.x
)
```

**Module Layout Expectations:**

- Your module path determines import paths throughout your app
- Keep your module path simple (e.g., `myapp` or `github.com/org/myapp`)
- Vango's code generator uses your module path to generate correct imports

---

## Editor Setup

### VS Code (Recommended)

Install the Go extension and configure Tailwind IntelliSense for Go files:

```json
// .vscode/settings.json
{
    "go.useLanguageServer": true,
    "go.lintTool": "golangci-lint",
    "go.formatTool": "goimports",
    "editor.formatOnSave": true,

    // Tailwind IntelliSense for Go files
    "tailwindCSS.includeLanguages": {
        "go": "html"
    },
    "tailwindCSS.experimental.classRegex": [
        ["Class\\(([^)]*)\\)", "\"([^\"]*)\""]
    ]
}
```

The `classRegex` pattern enables Tailwind autocomplete inside `Class(...)` calls in your Go code.

### GoLand / IntelliJ

- Import the project as a Go module
- Enable "Format on Save" with goimports
- Install the Tailwind CSS plugin for class name completion

### Vim / Neovim

Use gopls for Go support and the tailwindcss-language-server for Tailwind completion. Configure the language server to recognize `Class(...)` patterns.

---

# Part I: The Core

Fundamentals required to write a correct “Hello World” application: how Vango runs, how you write UI, and how you style it.

---

## 1. Mental Model & Architecture

### 1.1 Server-Driven UI

In a traditional SPA:

* state + rendering live in the browser
* the server is an API provider
* UI correctness depends on client/server sync and hydration consistency

In Vango:

* **state + rendering live on the server**
* the browser runs a **thin client (~12KB)** whose job is:

  1. capture user events
  2. send them to the server
  3. apply server-sent **binary patches** to the DOM

**Key implication:** your UI is a projection of server state. There is no client/server state synchronization problem because the server is the source of truth.

### 1.2 The Session Loop (Single Writer)

Each browser tab has a server-side **session** backed by a WebSocket connection. Each session runs a **single-threaded event loop**:

* events are processed one-at-a-time
* state updates are applied deterministically
* the UI re-renders and patches are emitted after the event is handled

**The single most important correctness rule:**

> **All signal writes MUST happen on the session loop.**

This is why Vango is race-resistant by default: it enforces a single-writer model.

If you do background work in goroutines, you must marshal results back to the session loop using `ctx.Dispatch(...)` (detailed later in the guide; mentioned here because it explains many “why did this panic?” issues).

### 1.3 The Request Lifecycle

#### Initial Load (SSR)

1. Browser requests `GET /some/path?query=...`
2. Vango matches the route and executes the page handler (render phase)
3. Vango renders a VDOM tree and produces **HTML** (SSR)
4. Browser displays content immediately
5. Thin client JS loads and connects via WebSocket to enable interactivity

#### WebSocket Upgrade and Route Mount

The thin client connects to:

* `/_vango/live?path=<current-path-and-query>`

That `?path=` parameter is required so the server mounts **the same route** the browser is currently showing. If the server mounts a different route/tree than SSR, event handler IDs won’t align and you’ll see “handler not found” or self-heal reloads.

#### Event/Patch Cycle (Interactive Loop)

1. user triggers an event (click/input/submit)
2. thin client sends a binary event referencing the element’s HID
3. server resolves HID → handler function
4. handler runs on the session loop (may write signals)
5. Vango re-renders affected components
6. Vango diffs old vs new VDOM and sends minimal patches
7. thin client applies patches to the DOM

### 1.4 The Thin Client

The thin client provides:

* WebSocket connect/reconnect
* event capture + binary event encoding
* patch application
* progressive enhancement (links/forms work without WS; enhanced when connected)
* self-heal (hard reload on patch mismatch or invalid session resume)

**Script inclusion:** You can include `VangoScripts()` explicitly in your root layout, or rely on auto-injection during SSR. Both work; explicit inclusion is recommended for clarity.

```go
func Layout(ctx vango.Ctx, children vango.Slot) *vango.VNode {
	return Html(
		Head(
			Meta(Charset("utf-8")),
			Meta(Name("viewport"), Content("width=device-width, initial-scale=1")),
			TitleEl(Text("My App")),
		),
		Body(
			children,
			VangoScripts(), // recommended explicit inclusion
		),
	)
}
```

### 1.5 Critical Contracts (What Must Be True)

Vango correctness depends on these application responsibilities:

1. **Render Purity (MUST):**
   Render functions (including page handlers) must not do blocking I/O, spawn goroutines, or perform non-deterministic side effects.

2. **Hook Order Stability (MUST):**
   Calls like `vango.NewSignal`, `vango.NewMemo`, `vango.Effect`, `vango.NewResource`, etc. must occur in a stable order/count across renders. Do not create hooks conditionally or in variable-length loops.

3. **Session Loop Writes (MUST):**
   Signal writes happen on the session loop (event handlers are on-loop; goroutines must use `ctx.Dispatch`).

4. **Stable Keys in Lists (SHOULD):**
   Dynamic lists should use stable domain keys (database IDs, UUIDs) to preserve identity and produce correct/efficient patches.

### 1.6 Page Handlers Are Reactive (Important)

Page handlers registered via `app.Page(...)` are **reactive**: they execute during the render cycle and are wrapped in `vango.Func` internally. That means page handlers follow the same rules as any render function:

* **MUST** be render-pure (no blocking I/O, no goroutines)
* **MUST** maintain stable hook order
* **SHOULD** use `Resource` for data loading (not blocking DB calls directly)

A page handler typically delegates to a component that uses structured primitives for I/O.

---

## 2. Component Syntax & Composition

### 2.1 The Go DSL (HTML in Go)

Vango UI is built using the `el` package DSL:

* elements: `Div(...)`, `Button(...)`, `Input(...)`, etc.
* text: `Text("...")`, `Textf("...")`
* attributes/modifiers: `Class(...)`, `ID(...)`, `Href(...)`, `Type(...)`, etc.

Example:

```go
import "github.com/vango-go/vango"
import . "github.com/vango-go/vango/el"

func IndexPage(ctx vango.Ctx) *vango.VNode {
	return Div(
		Class("container mx-auto p-8"),
		H1(Class("text-3xl font-bold"), Text("Welcome to Vango")),
		P(Class("text-gray-600"), Text("Server-driven UI with a client-side feel.")),
	)
}
```

**Mental model:** element functions build a VNode tree; Vango diffs successive trees and sends patches.

### 2.2 Creating Components

There are two common “shapes”:

#### A) Stateless VNode-returning functions

Use when the component has no local reactive state and is a pure function of inputs.

```go
func Badge(label string) *vango.VNode {
	return Span(
		Class("inline-flex items-center rounded px-2 py-1 text-xs"),
		Text(label),
	)
}
```

#### B) Stateful reactive components via `vango.Func`

Use when the component needs signals, resources, actions, memos, or effects.

```go
func Counter(initial int) vango.Component {
	return vango.Func(func() *vango.VNode {
		count := vango.NewSignal(initial)

		return Div(
			Class("flex items-center gap-3"),
			Button(Class("px-3 py-2 rounded bg-gray-200"), OnClick(count.Dec), Text("-")),
			Span(Class("font-mono"), Textf("%d", count.Get())),
			Button(Class("px-3 py-2 rounded bg-gray-200"), OnClick(count.Inc), Text("+")),
		)
	})
}
```

**Key point:** hooks like `vango.NewSignal(...)` are correct inside `vango.Func` (and inside reactive page handlers). The rule is not “no signals in render”; the rule is “stable hook order.”

### 2.3 Props & Children (Slots)

Vango supports composition by passing:

* typed props (Go parameters)
* children via `vango.Slot` (for layouts and “slot-like” APIs)

#### Layouts use `vango.Slot`

```go
func Layout(ctx vango.Ctx, children vango.Slot) *vango.VNode {
	return Html(
		Head(/* ... */),
		Body(
			Header(Class("p-4 border-b"), Text("My App")),
			Main(Class("p-4"), children),
			VangoScripts(),
		),
	)
}
```

#### Component “children” patterns

A common pattern is to accept `children ...any` and pass them through to an element constructor:

```go
func Card(children ...any) *vango.VNode {
	return Div(
		Class("rounded border p-4 bg-white"),
		Group(children...),
	)
}
```

You can also expose more structured slots by taking explicit `*vango.VNode` or `vango.Slot` parameters if you want stronger shape guarantees.

### 2.4 Event Handlers (OnClick, OnInput, OnSubmit) and Modifiers

Events are attached using DSL helpers like `OnClick`, `OnInput`, `OnChange`, `OnSubmit`, etc.

#### Common patterns

```go
func Example() vango.Component {
	return vango.Func(func() *vango.VNode {
		name := vango.NewSignal("")

		return Div(
			Input(
				Type("text"),
				Value(name.Get()),
				OnInput(func(v string) { name.Set(v) }),
			),
			Button(
				OnClick(func() { /* do something */ }),
				Text("Save"),
			),
		)
	})
}
```

#### PreventDefault and debouncing

Browser behavior and timing are expressed with modifiers. Two canonical cases from the current guide:

* **Form submit without navigation:** `vango.PreventDefault(...)`
* **Debounced input handling:** `vango.Debounce(d, handler)`

```go
func SearchBox() vango.Component {
	return vango.Func(func() *vango.VNode {
		q := vango.NewSignal("")

		return Form(
			OnSubmit(vango.PreventDefault(func() {
				// submit logic without full reload
			})),
			Input(
				Type("search"),
				Value(q.Get()),
				OnInput(vango.Debounce(300, func(v string) { // duration units depend on your imports; typically time.Millisecond
					q.Set(v)
				})),
			),
		)
	})
}
```

The important architectural point: event handlers run on the **session loop**, so signal writes inside them are safe and deterministic.

---

## 3. Styling & Design Systems

Vango does not require a particular styling strategy. The current developer guide defines two primary approaches:

1. Tailwind pipeline (default template)
2. Plain static CSS (Tailwind disabled)

### 3.1 Tailwind Pipeline (Default)

The default template uses Tailwind with a zero-config pipeline (no Node.js required). Vango runs a Tailwind scan over your Go files to discover classes used in `Class("...")` calls and produces a CSS output file in `public/`.

**Critical constraint (from current guide):**

> Tailwind’s scanner is string-based. **No dynamic string construction** for class names.

Good (detected):

```go
Div(Class("bg-blue-500 text-white"))
Div(Class("text-red-500", "font-bold"))
Div(Class(isActive && "bg-blue-500"))
```

Bad (not detected):

```go
Div(Class("btn-" + variant))
Div(Classf("text-%s-500", color))
```

If you need variants, use explicit branching that returns full class strings.

### 3.2 Conditional Classes with `Class(...)`

Prefer variadic `Class(...)` with conditional entries. This keeps class usage discoverable by Tailwind and readable by humans and coding agents:

```go
func NavItem(active bool, label string) *vango.VNode {
	return Div(
		Class(
			"px-3 py-2 rounded",
			active && "bg-gray-900 text-white",
			!active && "text-gray-700 hover:bg-gray-100",
		),
		Text(label),
	)
}
```

### 3.3 Standard CSS (No Tailwind)

You can disable Tailwind in `vango.json` (as described in the current guide) and use static CSS files served from `public/`.

Typical approach:

* place CSS in `public/styles.css` (or your chosen static path)
* reference it via `ctx.Asset("styles.css")` from your layout (manifest-aware in production)

```go
func Layout(ctx vango.Ctx, children vango.Slot) *vango.VNode {
	return Html(
		Head(
			LinkEl(Rel("stylesheet"), Href(ctx.Asset("styles.css"))),
		),
		Body(children, VangoScripts()),
	)
}
```

### 3.4 Dark Mode (Theme Toggle)

The current guide’s recommended dark-mode approach is class-based (e.g., Tailwind `dark:` or CSS variables toggled by `html.dark`).

Two important constraints in a server-driven, patch-based UI:

1. **Theme initialization should happen before paint** to avoid a flash (FOUC).
2. **DOM event listener wiring in inline scripts is fragile** under patch-based navigation unless it’s implemented as a client hook. For theme toggles, prefer a hook (the current guide references a `ThemeToggle` hook pattern).

A practical, robust approach:

* inline script in `<head>` to set the initial `dark` class based on `localStorage` and OS preference
* a client hook on the toggle button to flip `document.documentElement.classList` and persist to `localStorage`

(Exact hook packaging/registration is covered later in the guide’s client boundary section; the styling contract here is simply: toggle a stable `dark` class on `<html>` and let CSS handle the rest.)

# Part II: Reactivity

This part teaches the reactive programming model you use to build real Vango applications: **Signals** for state, **Memos** for derived state, and **Effects/Transactions** for side effects and atomic multi-writes. All examples assume:

```go
import "github.com/vango-go/vango"
import . "github.com/vango-go/vango/el"
```

(When an example needs `context` or `time`, it also imports those from the standard library.)

Two cross-cutting rules apply to everything in this part:

* **Render purity:** render functions (including reactive page handlers and `vango.Func`) must not do blocking I/O or spawn goroutines.
* **Stable hook order:** calls like `vango.NewSignal`, `vango.NewMemo`, `vango.Effect`, `vango.NewResource`, `vango.NewAction`, etc. must occur in a stable order/count across renders (never conditionally or in variable-length loops).

---

## 4. Signals

Signals are the core state primitive in Vango: a signal is a mutable value container that participates in dependency tracking.

### 4.1 The Signal Pattern

A signal supports three essential operations:

* **`Get()`**: read the value **and** register a dependency (the current component/memo will re-run when the signal changes)
* **`Set(v)`**: write a new value (must occur on the session loop; see §6.3)
* **`Peek()`**: read the value **without** registering a dependency (useful to avoid accidental reactivity)

```go
func Example() vango.Component {
	return vango.Func(func() *vango.VNode {
		count := vango.NewSignal(0)

		return Div(
			Textf("Count: %d", count.Get()), // Get() tracks dependency
			Button(OnClick(func() {
				count.Set(count.Peek() + 1) // Peek() avoids creating a dependency here
			}), Text("+")),
		)
	})
}
```

**What dependency tracking means in practice**

If a render function calls `count.Get()`, that render function becomes a dependent of `count`. When `count.Set(...)` is called, Vango marks the dependent computations dirty and re-runs them on the session loop, then diffs and patches the DOM.

#### Reactive Granularity: The Recomputation Unit

Vango uses **component-level dependency tracking**, not expression-level. The recomputation unit is:

| Construct | Recomputation Behavior |
|-----------|------------------------|
| `vango.Func` (component) | The entire render function re-runs if any signal read via `.Get()` during that render changes |
| `vango.NewMemo` | The memo function re-runs if any signal/memo read via `.Get()` during memo computation changes |
| Effects | Effects do not re-run from signal reads; they run once on mount and can optionally run cleanup/re-setup via returned cleanup function |

**Implications for structuring code:**

1. **Large components re-render fully.** If a component reads 10 signals, a change to any one re-runs the entire render function. The virtual DOM diff then minimizes actual DOM changes.

2. **Use memos to isolate expensive computation.** If you have an expensive filter/sort, wrap it in `vango.NewMemo`. The memo only recomputes when its specific dependencies change, and its result is cached.

3. **Component boundaries provide isolation.** Child components (separate `vango.Func` instances) only re-render if *their* dependencies change. A parent re-rendering doesn't force children to re-render unless the parent passes new props that the child reads.

```go
// Example: memo isolates expensive work
func ProductList() vango.Component {
    return vango.Func(func() *vango.VNode {
        products := vango.NewSignal(loadProducts())
        filter := vango.NewSignal("")
        sortBy := vango.NewSignal("name")

        // Memo: only recomputes when filter or products change
        // Does NOT recompute when sortBy changes (not read inside)
        filtered := vango.NewMemo(func() []Product {
            f := filter.Get()
            return filterProducts(products.Get(), f)
        })

        // Separate memo for sorting (recomputes when sortBy or filtered changes)
        sorted := vango.NewMemo(func() []Product {
            return sortProducts(filtered.Get(), sortBy.Get())
        })

        return Div(
            Input(OnInput(filter.Set)),
            Select(OnChange(sortBy.Set), /* options */),
            Ul(Range(sorted.Get(), renderProduct)),
        )
    })
}
```

This is similar to SolidJS's model (fine-grained at the memo level) rather than React's model (subtree re-render with reconciliation). The key insight: **signals + memos give you explicit control over what recomputes when**, while virtual DOM diffing ensures only necessary DOM mutations occur.

### 4.2 Scope 1: Local State (`NewSignal`)

`vango.NewSignal(...)` creates **component-instance state**. Each mounted instance of the component gets its own value. When the component unmounts, the signal is discarded.

Use for:

* form field values local to a component
* toggles, accordion open/closed, modal state
* ephemeral UI state that should reset when leaving the page/component

```go
func Counter(initial int) vango.Component {
	return vango.Func(func() *vango.VNode {
		count := vango.NewSignal(initial)

		return Div(
			Button(OnClick(count.Dec), Text("-")),
			Span(Textf("%d", count.Get())),
			Button(OnClick(count.Inc), Text("+")),
		)
	})
}
```

### 4.3 Scope 2: Session State (`NewSharedSignal`)

`vango.NewSharedSignal(...)` creates **session-scoped state**: it is shared across pages/components **within the same browser tab/session** and persists across navigation. It is appropriate for “app shell” state.

Use for:

* current user (session projection)
* notifications/toasts
* shopping cart for the tab
* cross-page UI state (sidebar collapsed, last visited section)

```go
// app/store/cart.go (example pattern)
package store

import "github.com/vango-go/vango"

// Session-scoped: one per browser tab/session.
var CartItems = vango.NewSharedSignal([]CartItem{})

func AddToCart(item CartItem) {
	CartItems.Set(append(CartItems.Get(), item))
}
```

Then in UI:

```go
func CartBadge() vango.Component {
	return vango.Func(func() *vango.VNode {
		n := len(store.CartItems.Get())
		return Span(Class("text-sm"), Textf("Cart (%d)", n))
	})
}
```

### 4.4 Scope 3: Global State (`NewGlobalSignal`)

`vango.NewGlobalSignal(...)` creates **global app-wide state**, shared across sessions/users. Any user can observe updates in real time (as patches).

Use for:

* presence/online users
* global announcements
* shared collaborative state (careful: multi-tenant scoping is your responsibility)

```go
// store/presence.go
package store

import "github.com/vango-go/vango"

// Global: shared across all sessions/users.
var OnlineCount = vango.NewGlobalSignal(0)

func UserConnected()  { OnlineCount.Set(OnlineCount.Get() + 1) }
func UserDisconnected(){ OnlineCount.Set(OnlineCount.Get() - 1) }
```

**Important:** global state is not automatically tenant-scoped. If your app is multi-tenant, represent global state as a map keyed by tenant, or otherwise isolate it explicitly.

### 4.5 Immutable Update Patterns (Slices and Maps)

Signals should be updated using **copy-on-write** patterns. Mutating a slice/map in place and re-setting the same underlying reference is error-prone and can lead to confusing behavior. Prefer creating new values or using provided helpers.

#### Slice patterns

```go
// Add
items.Set(append(items.Get(), newItem))

// Remove by index
items.RemoveAt(i)

// Remove by predicate
items.RemoveWhere(func(x Item) bool { return x.ID == id })

// Update element by index
items.UpdateAt(i, func(x Item) Item {
	x.Done = !x.Done
	return x
})

// Filter
items.Filter(func(x Item) bool { return x.Active })
```

#### Map patterns

```go
users := vango.NewSignal(map[int]User{})

// Set entry (copy-on-write)
users.SetEntry(u.ID, u)

// Delete entry
users.DeleteEntry(id)

// Update entry (copy-on-write)
users.UpdateEntry(id, func(u User) User {
	u.LastSeenUnix = time.Now().Unix()
	return u
})
```

### 4.6 Avoiding Hook-Order Violations with Signals

Signals are created inside render functions (including reactive page handlers), but you must create them in a stable order.

Bad (conditional creation):

```go
func Bad(show bool) vango.Component {
	return vango.Func(func() *vango.VNode {
		a := vango.NewSignal(0)
		if show {
			b := vango.NewSignal(0) // violates stable hook order
			_ = b
		}
		return Div(Textf("%d", a.Get()))
	})
}
```

Good (always create the same hooks; conditionally use them):

```go
func Good(show bool) vango.Component {
	return vango.Func(func() *vango.VNode {
		a := vango.NewSignal(0)
		b := vango.NewSignal(0) // always created
		if show {
			return Div(Textf("%d %d", a.Get(), b.Get()))
		}
		return Div(Textf("%d", a.Get()))
	})
}
```

---

## 5. Memos (Derived State)

Memos represent computed values derived from signals and other memos. They are cached and recomputed when their dependencies change.

### 5.1 Creating Memos (`NewMemo`)

```go
func Example() vango.Component {
	return vango.Func(func() *vango.VNode {
		count := vango.NewSignal(2)

		double := vango.NewMemo(func() int {
			return count.Get() * 2
		})

		return Div(
			Textf("count=%d", count.Get()),
			Textf("double=%d", double.Get()),
			Button(OnClick(count.Inc), Text("+")),
		)
	})
}
```

* `double.Get()` reads the memo value (and memo itself participates in dependency tracking like signals).
* The memo recomputes when `count` changes.

### 5.2 Dependency Graph and “Accidental Dependencies”

Memo dependencies are discovered by which signals/memos are read via `Get()` inside the memo function. Two practical rules:

1. **Make dependencies explicit:** avoid calling helper functions that read signals indirectly (it makes the dependency graph opaque).
2. **Avoid conditional dependencies:** if you only read a signal on some branches, updates to that signal may not be tracked when the branch is not taken.

Opaque dependency (avoid):

```go
func currentFilter() string {
	return filterSignal.Get() // hidden dependency
}

filtered := vango.NewMemo(func() []Item {
	f := currentFilter() // dependency is non-obvious
	return applyFilter(allItems.Get(), f)
})
```

Explicit dependency (prefer):

```go
filtered := vango.NewMemo(func() []Item {
	f := filterSignal.Get()
	items := allItems.Get()
	return applyFilter(items, f)
})
```

Conditional dependency pitfall (avoid):

```go
display := vango.NewMemo(func() string {
	if showDetails.Get() {
		return details.Get() // only tracked when showDetails is true
	}
	return summary.Get()
})
```

Safer pattern: read dependencies unconditionally, then branch:

```go
display := vango.NewMemo(func() string {
	show := showDetails.Get()
	d := details.Get()
	s := summary.Get()
	if show {
		return d
	}
	return s
})
```

### 5.3 When to Use Memos

Use a memo when it provides real value:

* caching an expensive computation
* isolating derived selectors that are used in multiple places
* keeping render functions clean and predictable

Example: expensive filtering

```go
func Good() vango.Component {
	return vango.Func(func() *vango.VNode {
		query := vango.NewSignal("")
		items := vango.NewSignal(loadBigList()) // example

		filtered := vango.NewMemo(func() []Item {
			return expensiveFilter(items.Get(), query.Get())
		})

		return Div(
			Input(Type("search"), Value(query.Get()), OnInput(query.Set)),
			Ul(Range(filtered.Get(), func(it Item, _ int) *vango.VNode {
				return Li(Key(it.ID), Text(it.Name))
			})),
		)
	})
}
```

Avoid memos for trivial expressions where inline code is clearer and the overhead isn’t justified.

### 5.4 Session-Scoped Derived State (Shared Memos)

In real apps, you often want session-scoped derived values that follow session-scoped signals (e.g., current user → isAuthenticated). The existing guide uses `vango.NewSharedMemo(...)` for this pattern:

```go
// store/auth.go
package store

import "github.com/vango-go/vango"

var CurrentUser = vango.NewSharedSignal[*User](nil)

var IsAuthenticated = vango.NewSharedMemo(func() bool {
	return CurrentUser.Get() != nil
})
```

(If your codebase chooses to keep “derived session state” inside components instead, that’s fine; the important point is: derived computations should remain reactive and dependency-driven.)

---

## 6. Effects & Transactions

Effects are for side effects that run in response to reactive changes. Transactions batch multiple writes into one commit.

### 6.1 Effects (`vango.Effect`) and Cleanup

An effect runs after a commit when its dependencies have changed. In Vango, an effect function **returns a cleanup** (or `nil`). Cleanup runs when:

* the effect is re-run (due to dependency change)
* the owning component unmounts

```go
import (
	"time"
	"github.com/vango-go/vango"
	. "github.com/vango-go/vango/el"
)

func Clock() vango.Component {
	return vango.Func(func() *vango.VNode {
		now := vango.NewSignal(time.Now())

		vango.Effect(func() vango.Cleanup {
			// Interval returns a Cleanup; returning it ties lifecycle to the component.
			return vango.Interval(1*time.Second, func() {
				now.Set(time.Now())
			})
		})

		return Div(Text(now.Get().Format(time.RFC3339)))
	})
}
```

**Key discipline:** do not write effects that directly cause render-loop amplification (e.g., an effect that writes to a signal it depends on without a guard). Use explicit guards or structured helpers, and rely on storm budgets/limits where configured.

### 6.2 Structured Effect Helpers

The existing guide describes several helpers that are designed to be called inside effects and return cleanup:

* `vango.Interval(duration, fn, opts...)`
* `vango.Subscribe(stream, fn, opts...)`
* `vango.GoLatest(key, work, apply, opts...)`

These helpers centralize correct lifecycle and session-loop dispatch behavior so you don’t hand-roll goroutine plumbing.

#### `GoLatest` pattern (cancel stale work, apply on session loop)

```go
import (
	"context"
	"time"
	"github.com/vango-go/vango"
	. "github.com/vango-go/vango/el"
)

func SearchBox() vango.Component {
	return vango.Func(func() *vango.VNode {
		q := vango.NewSignal("")
		results := vango.NewSignal([]Result{})
		errSig := vango.NewSignal[error](nil)
		loading := vango.NewSignal(false)

		vango.Effect(func() vango.Cleanup {
			key := q.Get() // dependency
			if key == "" {
				results.Set(nil)
				errSig.Set(nil)
				loading.Set(false)
				return nil
			}

			loading.Set(true)

			return vango.GoLatest(
				key,
				func(ctx context.Context, k string) ([]Result, error) {
					// example debounce inside work
					select {
					case <-time.After(300 * time.Millisecond):
					case <-ctx.Done():
						return nil, ctx.Err()
					}
					return search(ctx, k) // your I/O
				},
				func(r []Result, e error) {
					// apply runs on the session loop; safe to write signals
					if e != nil {
						errSig.Set(e)
					} else {
						results.Set(r)
						errSig.Set(nil)
					}
					loading.Set(false)
				},
			)
		})

		return Div(
			Input(Type("search"), Value(q.Get()), OnInput(q.Set)),
			loading.Get() && Div(Text("Loading...")),
			errSig.Get() != nil && Div(Text(errSig.Get().Error())),
			Ul(Range(results.Get(), func(r Result, _ int) *vango.VNode {
				return Li(Key(r.ID), Text(r.Label))
			})),
		)
	})
}
```

### 6.3 The Dispatch Rule (Goroutines → Session Loop)

Vango sessions are single-writer. Event handlers run on the session loop, but goroutines do not. Therefore:

> If a goroutine needs to read/write reactive state, it must use `ctx.Dispatch(func(){...})` to run that work on the session loop.

**Critical rule: Never call `vango.UseCtx()` from inside a goroutine.** `UseCtx()` returns the context for the *current render*—calling it from a goroutine races with render completion and may return stale or invalid context. Instead, capture `ctx` in the render function *before* spawning the goroutine:

```go
// CORRECT: capture ctx before goroutine
ctx := vango.UseCtx()
go func() {
    result := fetchData()
    ctx.Dispatch(func() { data.Set(result) })
}()

// WRONG: calling UseCtx() inside goroutine
go func() {
    ctx := vango.UseCtx() // ❌ races with render; undefined behavior
    ctx.Dispatch(func() { data.Set("x") })
}()
```

Correct pattern:

```go
import (
	"context"
	"github.com/vango-go/vango"
	. "github.com/vango-go/vango/el"
)

func Example() vango.Component {
	return vango.Func(func() *vango.VNode {
		ctx := vango.UseCtx()
		data := vango.NewSignal("")

		Button(OnClick(func() {
			go func() {
				// Do I/O off-loop
				val := fetchSomething(context.Background())

				// Marshal back to session loop for signal write
				ctx.Dispatch(func() {
					data.Set(val)
				})
			}()
		}), Text("Load"))

		return Div(Text(data.Get()))
	})
}
```

Incorrect pattern (do not do this):

```go
go func() {
	data.Set("x") // signal write outside session loop → panic/undefined behavior
}()
```

**Practical guidance:** prefer `Resource`, `Action`, and structured effect helpers (`GoLatest`, `Subscribe`, `Interval`) over ad-hoc goroutines. They exist specifically to keep your code safe and predictable in the single-writer model.

### 6.4 Transactions (`vango.Tx`, `vango.TxNamed`)

Transactions batch multiple signal writes into a single commit so you don’t trigger multiple renders/patches.

```go
func UpdateProfile(first, last, email *vango.Signal[string], f, l, e string) {
	vango.Tx(func() {
		first.Set(f)
		last.Set(l)
		email.Set(e)
	})
}
```

Named transactions add observability value (DevTools, logs):

```go
vango.TxNamed("UpdateUserProfile", func() {
	first.Set("John")
	last.Set("Doe")
	email.Set("j@d.com")
})
```

Use transactions when:

* one user action updates multiple signals
* you want atomic UI updates (all-or-nothing from the user’s perspective)
* you want clearer debugging (named intent)

### 6.5 Effect-Time Writes and Feedback Loops

Effects often write signals (e.g., `Interval` updating a clock). That is allowed, but you must avoid unbounded feedback:

Bad (self-triggering without guard):

```go
vango.Effect(func() vango.Cleanup {
	// If the effect depends on count.Get(), this Set will retrigger it indefinitely.
	count.Set(count.Get() + 1)
	return nil
})
```

Good (use event handlers, transactions, or structured helpers with explicit cadence/guards):

* write in event handlers (user-driven)
* use `Interval` for cadence
* use `GoLatest` for key-driven cancellation
* ensure writes do not create an infinite dependency cycle

### 6.6 Summary: Choosing the Right Primitive

Use this mapping as your default:

* **Signal**: mutable state
* **Memo**: derived/computed state
* **Effect**: side effects tied to reactive dependencies + lifecycle cleanup
* **Tx/TxNamed**: batch multi-writes into one commit
* **Dispatch**: the bridge from goroutines back to the session loop (when you truly need goroutines)

# Part III: Data Flow

This part covers Vango’s structured primitives for asynchronous work and server-side I/O: **Resources** for reading/loading and **Actions** for mutations. The goal is to let you do real database/network work **without violating render purity** and **without breaking the single-writer session loop**.

All examples assume:

```go
import "github.com/vango-go/vango"
import . "github.com/vango-go/vango/el"
```

(Examples that perform async work also import standard library `context` and/or `time` as needed.)

---

## 7. Resources (Reading Data)

A **Resource** is Vango’s primary primitive for loading data asynchronously while rendering. It is designed for server-driven UI: render functions remain pure while data fetches occur off-loop and results are applied safely on the session loop.

### 7.1 The Resource Primitive (`vango.NewResource`)

Use `vango.NewResource(func() (T, error) { ... })` inside a render function (`vango.Func` or a reactive page handler). The loader function runs asynchronously; the component renders through state transitions.

```go
import (
	"github.com/vango-go/vango"
	. "github.com/vango-go/vango/el"
)

type User struct {
	ID   int
	Name string
}

func UserCard(userID int) vango.Component {
	return vango.Func(func() *vango.VNode {
		ctx := vango.UseCtx()
		std := ctx.StdContext()

		user := vango.NewResource(func() (*User, error) {
			// Blocking I/O is allowed here; it is *not* running in render.
			return loadUser(std, userID)
		})

		return user.Match(
			vango.OnPending(func() *vango.VNode {
				// Pending: created but not started yet (often transient)
				return Div(Text("…"))
			}),
			vango.OnLoading(func() *vango.VNode {
				return Div(Text("Loading user…"))
			}),
			vango.OnError(func(err error) *vango.VNode {
				return Div(Class("text-red-600"), Text(err.Error()))
			}),
			vango.OnReady(func(u *User) *vango.VNode {
				return Div(
					H2(Class("font-bold"), Text(u.Name)),
					P(Textf("User #%d", u.ID)),
				)
			}),
		)
	})
}
```

**Rules for the loader function:**

* It runs off the session loop, so it may do blocking DB/HTTP work.
* Use `ctx.StdContext()` for cancellation/timeouts.
* Do not call `vango.Ctx` methods (Navigate/Dispatch/etc.) from inside the loader.

### 7.2 Resource States: Pending, Loading, Ready, Error

Resources expose four states (from the current guide):

* `vango.Pending`
* `vango.Loading`
* `vango.Ready`
* `vango.Error`

You can branch via `State()` or prefer `Match(...)` for clarity.

```go
r := vango.NewResource(loadSomething)

switch r.State() {
case vango.Pending:
	return Div(Text("…"))
case vango.Loading:
	return Spinner()
case vango.Error:
	return ErrorCard(r.Error())
case vango.Ready:
	return View(r.Data())
default:
	return Div(Text("unexpected state"))
}
```

**Empty results are application logic**: `Ready` can still mean “empty list,” so you handle that explicitly in your ready branch.

### 7.3 Keyed Resources (`vango.NewResourceKeyed`)

When the identity of the resource depends on a reactive value (URL param, signal, or memo), use `NewResourceKeyed`. It deduplicates by key and refetches when the key changes.

Key can be:

* a signal-like value (`*Signal[K]`, URLParam, memo-like)
* a `func() K` (reactively read inside it)

**Example: keyed by a signal**

```go
import "github.com/vango-go/vango"
import . "github.com/vango-go/vango/el"

func ProjectViewer() vango.Component {
	return vango.Func(func() *vango.VNode {
		ctx := vango.UseCtx()
		std := ctx.StdContext()

		selectedID := vango.NewSignal(0)

		project := vango.NewResourceKeyed(selectedID, func(id int) (*Project, error) {
			if id == 0 {
				return nil, nil
			}
			return loadProject(std, id)
		})

		return Div(
			Button(OnClick(func() { selectedID.Set(1) }), Text("Load #1")),
			Button(OnClick(func() { selectedID.Set(2) }), Text("Load #2")),
			project.Match(
				vango.OnLoading(func() *vango.VNode { return Div(Text("Loading…")) }),
				vango.OnReady(func(p *Project) *vango.VNode {
					if p == nil {
						return Div(Text("No selection"))
					}
					return Div(Text(p.Name))
				}),
				vango.OnError(func(err error) *vango.VNode { return Div(Text(err.Error())) }),
			),
		)
	})
}
```

**Key behaviors (from the current guide’s semantics):**

* same key → reuse cached result
* different key → cancel stale work, start new load, transition to Loading
* unmount → cancel in-flight work

**Important pitfall to avoid**

Do not create a new signal from a prop inside render and expect it to react to prop changes. If the key should change, pass a real reactive key (signal/URLParam/memo) from the parent.

### 7.4 Caching and Deduplication

Resource caching is primarily **per-resource instance** and key-based:

* A keyed resource avoids restarting work if the key hasn’t changed.
* It cancels stale requests when the key changes.
* It prevents simple “refetch on every render” mistakes by giving identity to the load.

If you need cross-component/process caching, do it at the service/repository layer (in-memory cache, Redis, singleflight, etc.). Resource is the UI-facing async state machine, not your global cache.

---

## 8. Actions (Mutating Data)

An **Action** is the standard primitive for async mutations triggered by the user (save, delete, submit, etc.). Actions have explicit states and concurrency policies, and they handle safe application of results back to the session loop.

### 8.1 The Action Primitive (`vango.NewAction`)

The current guide defines the canonical shape:

```go
action := vango.NewAction(
	func(ctx context.Context, arg A) (R, error) { ... },
	vango.CancelLatest(), // default policy
)

accepted := action.Run(arg) // bool
action.State()              // ActionIdle/Running/Success/Error
result, ok := action.Result()
action.Error()
action.Reset()
```

**Example: submit a form**

```go
import (
	"context"
	"github.com/vango-go/vango"
	. "github.com/vango-go/vango/el"
)

type SaveInput struct {
	Name string
}

type SaveResult struct {
	ID int
}

func SaveWidget() vango.Component {
	return vango.Func(func() *vango.VNode {
		ctx := vango.UseCtx()
		name := vango.NewSignal("")

		save := vango.NewAction(
			func(std context.Context, in SaveInput) (*SaveResult, error) {
				// do I/O off-loop
				return saveToDB(std, in.Name)
			},
			vango.DropWhileRunning(), // common for “Save” buttons
		)

		return Div(
			Input(Type("text"), Value(name.Get()), OnInput(name.Set)),
			Button(
				Disabled(save.State() == vango.ActionRunning),
				OnClick(func() {
					save.Run(SaveInput{Name: name.Get()})
				}),
				Text("Save"),
			),
			save.State() == vango.ActionRunning && Div(Text("Saving…")),
			save.State() == vango.ActionError && Div(Class("text-red-600"), Text(save.Error().Error())),
			save.State() == vango.ActionSuccess && func() *vango.VNode {
				if res, ok := save.Result(); ok && res != nil {
					return Div(Class("text-green-700"), Textf("Saved as #%d", res.ID))
				}
				return Div(Class("text-green-700"), Text("Saved"))
			}(),
		)
	})
}
```

### 8.2 Concurrency Policies (CancelLatest, DropWhileRunning, Queue)

Actions can be triggered multiple times; the concurrency policy defines what happens when `Run(...)` is called while a prior run is still in flight.

From the current guide:

* `vango.CancelLatest()` (default): cancel prior in-flight work; newest wins
  Good for search/filter, typeahead, “recompute preview”.
* `vango.DropWhileRunning()`: ignore new calls while running
  Good for save/delete buttons to avoid double-submit.
* `vango.Queue(n)`: queue up to `n` calls and run sequentially
  Good for “process items” or ordered operations.

```go
// Search: newest wins
search := vango.NewAction(doSearch, vango.CancelLatest())

// Save: drop repeats while pending
save := vango.NewAction(doSave, vango.DropWhileRunning())

// Process: ordered queue
process := vango.NewAction(doWork, vango.Queue(10))
```

### 8.3 Action States: Loading UX and Error Handling

Actions expose `State()` with the following states (from the current guide):

* `vango.ActionIdle`
* `vango.ActionRunning`
* `vango.ActionSuccess`
* `vango.ActionError`

Use these states to drive UI:

* disable buttons during `Running`
* show spinners/progress
* surface error messages
* show success confirmation
* allow retry + `Reset()`

```go
switch save.State() {
case vango.ActionRunning:
	return Button(Disabled(true), Text("Saving…"))
case vango.ActionError:
	return Div(
		Button(OnClick(func() { save.Run(SaveInput{Name: name.Get()}) }), Text("Retry")),
		Div(Class("text-red-600"), Text(save.Error().Error())),
	)
default:
	return Button(OnClick(func() { save.Run(SaveInput{Name: name.Get()}) }), Text("Save"))
}
```

`Run(...)` returns a **bool** indicating whether the invocation was accepted under the current concurrency policy. If you care, check it; if not, you can ignore it.

### 8.4 Optimistic UI

Optimistic UI means you update the UI immediately, then reconcile when the server confirms success (or rollback on error). In Vango, the usual pattern is:

1. write optimistic state in the event handler (session loop)
2. trigger the action for the real I/O
3. on error, restore prior state

```go
import (
	"context"
	"github.com/vango-go/vango"
	. "github.com/vango-go/vango/el"
)

type Task struct {
	ID   int
	Name string
	Done bool
}

func TaskRow(t Task) vango.Component {
	return vango.Func(func() *vango.VNode {
		done := vango.NewSignal(t.Done)

		toggle := vango.NewAction(
			func(ctx context.Context, newDone bool) (bool, error) {
				return newDone, setTaskDone(ctx, t.ID, newDone)
			},
			vango.CancelLatest(),
		)

		return Div(
			Input(
				Type("checkbox"),
				Checked(done.Get()),
				OnChange(func() {
					prev := done.Get()
					next := !prev

					// optimistic update (session loop)
					done.Set(next)

					// async mutation
					toggle.Run(next)

					// if it fails, rollback
					// (use an effect to observe action completion)
				}),
			),
			Span(Text(t.Name)),
			toggle.State() == vango.ActionRunning && Span(Class("ml-2 text-sm"), Text("Saving…")),
			vango.Func(func() *vango.VNode {
				// Observe completion via reactive reads
				if toggle.State() == vango.ActionError {
					// rollback
					// Note: this is a signal write driven by reactive state; keep it guarded.
					// A safer alternative is a dedicated Effect that triggers only on error transitions.
					done.Set(t.Done)
					return Span(Class("ml-2 text-sm text-red-600"), Text("Failed"))
				}
				return nil
			})(),
		)
	})
}
```

In practice, prefer handling rollback in an explicit `vango.Effect` that watches the action state transition to error/success, to keep rollback logic from repeatedly firing if the component re-renders while still in an error state.

---

## 9. Operational Correctness for Data Flow

This is the condensed correctness model you should keep in mind while designing data flow.

### 9.1 Where blocking I/O is allowed

* **Allowed:** inside Resource loaders and Action work functions (they run off the session loop)
* **Not allowed:** inside render functions (page handlers, `vango.Func`) directly

Your default patterns should therefore be:

* “show UI with loading states” → **Resource**
* “user presses a button to mutate” → **Action**
* “reactive async with cancellation” → **Effect + GoLatest** (or a keyed resource depending on shape)

### 9.2 Context and cancellation

Use `ctx.StdContext()` inside loaders/work functions. It carries cancellation signals appropriate to the request/session lifecycle.

Do not use `vango.Ctx` methods from background goroutines or async loader functions. If you must do something session-related as a result of async work, it should occur in the apply callback (GoLatest) or after the resource/action has resolved (on the session loop).

### 9.3 Common pitfalls (and the correct fix)

* **Pitfall:** Querying the database in a page handler directly
  **Fix:** Move I/O into `NewResource` / `NewResourceKeyed`.

* **Pitfall:** Action invoked repeatedly (double submit)
  **Fix:** `DropWhileRunning()` + disable UI while running.

* **Pitfall:** Search runs per keystroke and results race
  **Fix:** `CancelLatest()` or `Effect + GoLatest` with debounce/cancellation.

* **Pitfall:** Writing signals from goroutines started in loaders
  **Fix:** don’t; loaders shouldn’t mutate signals directly. Let Resource/Action resolution drive UI. If you truly have a goroutine, marshal back with `ctx.Dispatch`.

---

## 10. Practical Patterns (Resource + Action Together)

### 10.1 Load → Edit → Save

A canonical CRUD page:

* Resource loads entity
* UI edits local signals
* Action saves
* on success, update local state or navigate

```go
import (
	"context"
	"github.com/vango-go/vango"
	. "github.com/vango-go/vango/el"
)

type Params struct {
	ID int `param:"id"`
}

func ProjectEditPage(ctx vango.Ctx, p Params) *vango.VNode {
	return ProjectEdit(p.ID)
}

func ProjectEdit(id int) vango.Component {
	return vango.Func(func() *vango.VNode {
		ctx := vango.UseCtx()
		std := ctx.StdContext()

		project := vango.NewResource(func() (*Project, error) {
			return loadProject(std, id)
		})

		return project.Match(
			vango.OnLoading(func() *vango.VNode { return Div(Text("Loading…")) }),
			vango.OnError(func(err error) *vango.VNode { return Div(Text(err.Error())) }),
			vango.OnReady(func(p *Project) *vango.VNode {
				name := vango.NewSignal(p.Name)

				save := vango.NewAction(
					func(ctx context.Context, newName string) (*Project, error) {
						return updateProjectName(ctx, p.ID, newName)
					},
					vango.DropWhileRunning(),
				)

				return Div(
					H1(Textf("Edit Project #%d", p.ID)),
					Input(Type("text"), Value(name.Get()), OnInput(name.Set)),
					Button(
						Disabled(save.State() == vango.ActionRunning),
						OnClick(func() { save.Run(name.Get()) }),
						Text("Save"),
					),
					save.State() == vango.ActionRunning && Div(Text("Saving…")),
					save.State() == vango.ActionError && Div(Class("text-red-600"), Text(save.Error().Error())),
					save.State() == vango.ActionSuccess && Div(Class("text-green-700"), Text("Saved")),
				)
			}),
		)
	})
}
```

This pattern keeps render pure, makes loading/saving states explicit, and preserves the single-writer model.

# Part IV: Application Architecture

This part explains how to structure a complete Vango application: project layout, routing and pages, forms, client-boundary escape hatches, and the CLI/tooling workflow. It is written so an LLM or developer can scaffold, extend, and maintain a Vango app without needing to consult framework internals.

All examples assume:

```go
import "github.com/vango-go/vango"
import . "github.com/vango-go/vango/el"
```

(Plus standard library imports where needed.)

---

## 11. Project Layout and Code Organization

This section provides the recommended structure for Vango apps. Following these conventions ensures the CLI tools work correctly and keeps codebases clean as they grow.

### 11.1 Recommended Project Structure

```
myapp/
├── cmd/
│   └── server/
│       └── main.go                 # Entry point: config, server startup
│
├── app/
│   ├── routes/                     # File-based routing (required)
│   │   ├── routes_gen.go           # Generated route registration (do not edit)
│   │   ├── layout.go               # Root layout (wraps all pages)
│   │   ├── index.go                # Home page (/)
│   │   ├── about.go                # /about
│   │   ├── projects/
│   │   │   ├── layout.go           # Nested layout for /projects/*
│   │   │   ├── index.go            # /projects (list)
│   │   │   └── id_/
│   │   │       ├── index.go        # /projects/{id}
│   │   │       ├── edit.go         # /projects/{id}/edit
│   │   │       └── tasks.go        # /projects/{id}/tasks
│   │   └── api/
│   │       └── health.go           # /api/health (typed API endpoint)
│   │
│   ├── components/                 # Shared UI components
│   │   ├── button.go
│   │   ├── card.go
│   │   ├── modal.go
│   │   ├── form/
│   │   │   ├── input.go
│   │   │   ├── select.go
│   │   │   └── validation.go
│   │   └── layout/
│   │       ├── header.go
│   │       ├── sidebar.go
│   │       └── footer.go
│   │
│   ├── store/                      # Shared reactive state
│   │   ├── auth.go                 # Current user (session-scoped)
│   │   ├── notifications.go        # Toast/notification state
│   │   └── preferences.go          # User preferences
│   │
│   └── middleware/                 # HTTP and Vango middleware
│       ├── auth.go
│       ├── logging.go
│       └── rate_limit.go
│
├── internal/                       # Private application code
│   ├── services/                   # Business logic
│   │   ├── projects.go
│   │   ├── tasks.go
│   │   └── users.go
│   │
│   ├── db/                         # Database layer
│   │   ├── db.go                   # Connection management
│   │   ├── projects.go             # Project queries
│   │   ├── tasks.go
│   │   └── migrations/
│   │       └── ...
│   │
│   └── config/                     # Configuration loading
│       └── config.go
│
├── public/                         # Static assets (served directly)
│   ├── styles.css                  # Compiled CSS (Tailwind output)
│   ├── js/
│   │   └── hooks/                  # Client hook bundles
│   │       └── sortable.js
│   ├── images/
│   │   ├── logo.svg
│   │   └── hero.png
│   ├── fonts/
│   │   └── inter.woff2
│   └── favicon.ico
│
├── app/styles/                     # CSS source (Tailwind input)
│   └── input.css
│
├── vango.json                      # Vango CLI configuration
├── go.mod
├── go.sum
└── tailwind.config.js              # Optional Tailwind configuration
```

### 11.2 Entry Point (`cmd/server/main.go`)

The entry point follows standard Go patterns:

```go
// cmd/server/main.go
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/vango-go/vango"
	"myapp/app/routes"
	"myapp/internal/config"
	"myapp/internal/db"
)

func main() {
	// 1. Load configuration
	cfg := config.Load()

	// 2. Initialize dependencies
	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	// 3. Create Vango app
	app := vango.New(cfg.VangoConfig())

	// 4. Register routes (generated code)
	routes.Register(app, database)

	// 5. Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 6. Start server
	slog.Info("starting server", "port", cfg.Port)
	if err := app.Run(ctx, ":"+cfg.Port); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
```

### 11.3 Where to Put Your Code

| Code Type | Location | Why |
|-----------|----------|-----|
| Page handlers | `app/routes/` | File-based routing requires this location |
| Shared UI components | `app/components/` | Reusable across pages; no route dependencies |
| Session/global state | `app/store/` | Shared signals/memos used across pages |
| HTTP/Vango middleware | `app/middleware/` | Auth guards, logging, rate limiting |
| Business logic | `internal/services/` | Private; testable; no UI dependencies |
| Database queries | `internal/db/` | Private; swappable; uses `context.Context` |
| Configuration | `internal/config/` | Environment-based config loading |
| Static files | `public/` | Served directly; fingerprinted in production |

### 11.4 Shared Components (`app/components/`)

Components used across multiple pages live in `app/components/`:

```go
// app/components/button.go
package components

import (
	"github.com/vango-go/vango"
	. "github.com/vango-go/vango/el"
)

type ButtonVariant string

const (
	ButtonPrimary   ButtonVariant = "primary"
	ButtonSecondary ButtonVariant = "secondary"
	ButtonDanger    ButtonVariant = "danger"
)

func Button(variant ButtonVariant, children ...any) func(...any) *vango.VNode {
	baseClass := "px-4 py-2 rounded font-medium transition-colors"

	variantClass := map[ButtonVariant]string{
		ButtonPrimary:   "bg-blue-500 text-white hover:bg-blue-600",
		ButtonSecondary: "bg-gray-200 text-gray-800 hover:bg-gray-300",
		ButtonDanger:    "bg-red-500 text-white hover:bg-red-600",
	}[variant]

	return func(attrs ...any) *vango.VNode {
		allAttrs := append([]any{Class(baseClass, variantClass)}, attrs...)
		allAttrs = append(allAttrs, children...)
		return ButtonEl(allAttrs...)
	}
}

// Usage:
// Button(ButtonPrimary, Text("Save"))(OnClick(handleSave))
// Button(ButtonDanger, Text("Delete"))(OnClick(handleDelete), Disabled(true))
```

**Component organization pattern:**

```
app/components/
├── button.go           # Single component per file
├── card.go
├── modal.go
├── badge.go
│
├── form/               # Group related components
│   ├── input.go
│   ├── textarea.go
│   ├── select.go
│   └── field.go        # Field wrapper with label/error
│
├── layout/             # Layout primitives
│   ├── header.go
│   ├── sidebar.go
│   └── footer.go
│
└── data/               # Data display components
    ├── table.go
    ├── pagination.go
    └── empty_state.go
```

### 11.5 Shared State (`app/store/`)

Session-scoped and global state lives in `app/store/`:

```go
// app/store/auth.go
package store

import "github.com/vango-go/vango"

type User struct {
	ID    int
	Email string
	Role  string
}

// CurrentUser holds the authenticated user for this session.
// Session-scoped: each browser tab has its own value.
var CurrentUser = vango.NewSharedSignal[*User](nil)

// IsAuthenticated is derived state.
var IsAuthenticated = vango.NewSharedMemo(func() bool {
	return CurrentUser.Get() != nil
})

// IsAdmin is derived state.
var IsAdmin = vango.NewSharedMemo(func() bool {
	u := CurrentUser.Get()
	return u != nil && u.Role == "admin"
})
```

```go
// app/store/notifications.go
package store

import "github.com/vango-go/vango"

type Toast struct {
	ID      string
	Type    string // "success", "error", "info"
	Message string
}

var Toasts = vango.NewSharedSignal([]Toast{})

func ShowToast(toastType, message string) {
	id := generateID()
	Toasts.Set(append(Toasts.Get(), Toast{ID: id, Type: toastType, Message: message}))
}

func DismissToast(id string) {
	Toasts.RemoveWhere(func(t Toast) bool {
		return t.ID == id
	})
}
```

**State scope guidelines:**

| Scope | When to Use | Example |
|-------|-------------|---------|
| `NewSignal` (component) | UI state local to one component | Form inputs, accordion open/closed |
| `NewSharedSignal` (session) | State shared across pages for one user | Shopping cart, current user, notifications |
| `NewGlobalSignal` (all users) | Real-time shared state | Online user count, live cursors |

### 11.6 Internal Packages (`internal/`)

Go's `internal/` directory contains private packages that can't be imported by external code. This is where your business logic and data access live:

```go
// internal/services/projects.go
package services

import (
	"context"
	"myapp/internal/db"
)

type ProjectService struct {
	db *db.Queries
}

func NewProjectService(db *db.Queries) *ProjectService {
	return &ProjectService{db: db}
}

func (s *ProjectService) List(ctx context.Context, userID int) ([]db.Project, error) {
	return s.db.ListProjects(ctx, userID)
}

func (s *ProjectService) Create(ctx context.Context, userID int, name string) (*db.Project, error) {
	return s.db.CreateProject(ctx, db.CreateProjectParams{
		UserID: userID,
		Name:   name,
	})
}
```

**Key principle:** Services accept `context.Context` and return data/errors. They do not import Vango or know about signals. This keeps business logic testable and decoupled from the UI framework.

### 11.7 Dependency Flow (Preventing Circular Imports)

The dependency graph should flow one way:

```
routes → components → (no app dependencies)
routes → store → (no app dependencies)
routes → internal/services → internal/db

Never: components → routes (circular)
Never: store → routes (circular)
Never: internal → app (breaks encapsulation)
```

If a component needs route-specific behavior, pass it as a prop or callback:

```go
// ❌ Bad: Component imports route-specific code
package components
import "myapp/app/routes/projects"  // Circular!

// ✅ Good: Component accepts callback
func TaskRow(task Task, onComplete func()) *vango.VNode {
	return Tr(
		Td(Text(task.Title)),
		Td(Button(ButtonPrimary, Text("Complete"))(OnClick(onComplete))),
	)
}
```

### 11.8 Static Assets (`public/`)

The `public/` directory is served directly by Vango:

| File Path | URL |
|-----------|-----|
| `public/styles.css` | `/styles.css` |
| `public/images/logo.svg` | `/images/logo.svg` |
| `public/favicon.ico` | `/favicon.ico` |

In production, Vango applies appropriate caching:
- Fingerprinted assets (`app.abc123.css`): Immutable, long-lived cache
- Non-fingerprinted assets: Short cache with revalidation

Use `ctx.Asset("...")` in layouts to get manifest-aware paths:

```go
LinkEl(Rel("stylesheet"), Href(ctx.Asset("styles.css")))
// Production: /styles.abc123.css
// Development: /styles.css
```

---

## 12. Routing & Pages

Vango uses file-based routing rooted at `app/routes`. Route handlers are reactive render functions (they run during the render cycle and must remain pure). Layouts and middleware compose hierarchically along the directory tree.

### 12.1 File-Based Routing

**Directory structure maps to URL paths:**

* `app/routes/index.go` → `/`
* `app/routes/about.go` → `/about`
* `app/routes/projects/index.go` → `/projects`
* `app/routes/projects/[id].go` (or `id_.go`) → `/projects/:id`
* `app/routes/docs/[...path].go` → `/docs/*path`
* `app/routes/layout.go` → root layout
* `app/routes/**/layout.go` → nested layouts
* `app/routes/**/middleware.go` → directory middleware
* `app/routes/api/*.go` → typed API endpoints (registered via `app.API`)

**Canonical conventions from the current guide:**

* Prefer simple files for simple pages (`about.go` rather than nested directories).
* Avoid route files beginning with `_` or `.` (Go ignores them).
* Avoid directory names that are invalid in Go import paths; bracketed directory names are deprecated. For nested param directories, use Go-friendly directory names (e.g., `id_/index.go`).

### 12.2 Route Parameters

Vango supports typed params via filename annotations.

#### Parameter filename syntax

* `[id]` → string
* `[id:int]` → int (invalid values 404)
* `[id:uuid]` → string with runtime UUID validation
* `[...path]` → catch-all

#### Handler signature

Handlers with params must use a `Params` struct with `param` tags.

```go
type Params struct {
	ID int `param:"id"`
}

func ShowPage(ctx vango.Ctx, p Params) *vango.VNode {
	// p.ID is already validated and parsed (because filename was [id:int])
	return Div(Textf("Project %d", p.ID))
}
```

UUID params map to `string` (do not use `uuid.UUID` in Params):

```go
type Params struct {
	ID string `param:"id"` // from [id:uuid]
}
```

Catch-all params can be `[]string`:

```go
type Params struct {
	Path []string `param:"path"` // from [...path]
}
```

### 12.3 Layouts (Nested and Persistent UI)

Layouts wrap pages and can be nested by directory. Layout signature:

```go
func Layout(ctx vango.Ctx, children vango.Slot) *vango.VNode
```

**Root layout example (HTML shell + assets + scripts):**

```go
func Layout(ctx vango.Ctx, children vango.Slot) *vango.VNode {
	return Html(Lang("en"),
		Head(
			Meta(Charset("utf-8")),
			Meta(Name("viewport"), Content("width=device-width, initial-scale=1")),
			TitleEl(Text("My Vango App")),
			LinkEl(Rel("stylesheet"), Href(ctx.Asset("styles.css"))),
		),
		Body(
			Header(Class("border-b p-4"),
				Nav(Class("flex gap-4"),
					Link("/", Text("Home")),
					Link("/about", Text("About")),
				),
			),
			Main(Class("p-4"), children),
			VangoScripts(),
		),
	)
}
```

**Nested layout example (section shell):**

```go
// app/routes/projects/layout.go
func Layout(ctx vango.Ctx, children vango.Slot) *vango.VNode {
	return Div(Class("flex min-h-screen"),
		Aside(Class("w-64 border-r p-4"),
			Nav(Class("space-y-2"),
				NavLink(ctx, "/projects", Text("All Projects")),
				NavLink(ctx, "/projects/new", Text("New Project")),
			),
		),
		Main(Class("flex-1 p-6"), children),
	)
}
```

**Design principle:** layouts are your persistent UI boundary (headers, sidebars, global toasts, connection banners, etc.). Keep them render-pure; load data via Resources if needed.

### 12.4 Page Handlers and Components

A page handler registered via routing is reactive and must remain render-pure. The simplest correct pattern is:

* page handler extracts params
* returns a component that uses Resources/Actions as needed

```go
type Params struct {
	ID int `param:"id"`
}

func ShowPage(ctx vango.Ctx, p Params) *vango.VNode {
	return ProjectPage(p.ID)
}

func ProjectPage(id int) vango.Component {
	return vango.Func(func() *vango.VNode {
		ctx := vango.UseCtx()
		std := ctx.StdContext()

		project := vango.NewResource(func() (*Project, error) {
			return loadProject(std, id)
		})

		return project.Match(
			vango.OnLoading(func() *vango.VNode { return Div(Text("Loading…")) }),
			vango.OnError(func(err error) *vango.VNode { return Div(Text(err.Error())) }),
			vango.OnReady(func(p *Project) *vango.VNode {
				return Div(
					H1(Class("text-2xl font-bold"), Text(p.Name)),
					P(Text(p.Description)),
				)
			}),
		)
	})
}
```

This avoids blocking I/O in the handler while keeping route logic and UI composition clean.

### 12.5 Navigation

Vango supports:

* declarative navigation via Link helpers (SPA when connected; normal navigation when not)
* programmatic navigation via `ctx.Navigate`

#### Link helpers

From the current guide’s quick reference:

* `Link("/path", children...)`
* `LinkPrefetch("/path", children...)`
* `NavLink(ctx, "/path", children...)` (active class when current)
* `NavLinkPrefix(ctx, "/admin", ...)` (active for subroutes)

```go
Nav(
	NavLink(ctx, "/", Text("Home")),
	NavLink(ctx, "/projects", Text("Projects")),
)
```

#### Programmatic navigation

```go
Button(OnClick(func() {
	ctx := vango.UseCtx()
	ctx.Navigate("/projects/123")
}), Text("Go"))
```

For "replace" navigation (no history entry), the current guide shows using a replace option (exact option symbol may differ in final API; keep your codebase consistent with the implemented option). The intent is:

* push: default
* replace: no new entry

### 12.6 Navigation Lifecycle & Unsaved Changes

Vango provides navigation lifecycle hooks to intercept navigation events. These are essential for "unsaved changes" warnings and analytics.

#### OnBeforeNavigate (Guard Navigation)

`OnBeforeNavigate` runs **before** navigation occurs and can cancel it. This runs on the **client side** (in the thin client) because it must intercept the navigation before the server is contacted.

```go
func EditForm() vango.Component {
	return vango.Func(func() *vango.VNode {
		ctx := vango.UseCtx()
		isDirty := vango.NewSignal(false)

		// Register navigation guard (client-side execution)
		vango.OnBeforeNavigate(func(to string) bool {
			if isDirty.Peek() {
				// Return false to cancel navigation
				// The thin client shows a browser confirm dialog
				return false // Will prompt: "You have unsaved changes. Leave anyway?"
			}
			return true // Allow navigation
		})

		return Form(
			OnInput(func() { isDirty.Set(true) }),
			// ... form fields ...
			Button(Type("submit"), OnClick(func() {
				// Save and clear dirty flag
				isDirty.Set(false)
			}), Text("Save")),
		)
	})
}
```

**Key constraints:**

- `OnBeforeNavigate` registers a client-side callback; the function body is serialized and runs in the thin client
- Use `signal.Peek()` (not `.Get()`) to read state without creating reactive dependencies
- Return `false` to cancel navigation (thin client shows browser's "Leave site?" dialog)
- Return `true` to allow navigation to proceed
- The guard is scoped to the component's lifetime; it's automatically unregistered on unmount

#### OnNavigate (After Navigation)

`OnNavigate` runs **after** navigation completes. Use it for analytics, scroll restoration, or other post-navigation effects.

```go
func Layout(ctx vango.Ctx, children vango.Slot) *vango.VNode {
	// Track page views
	vango.OnNavigate(func(path string) {
		analytics.TrackPageView(path)
	})

	return Html(
		Body(children, VangoScripts()),
	)
}
```

**Key points:**

- Runs on both SPA navigation (WebSocket) and initial page load
- `path` includes query string (e.g., `/projects?filter=active`)
- Registered in layouts to track all navigation within that layout's scope

### 12.7 Connection Status UX

The thin client maintains a WebSocket connection to the server. When this connection is interrupted, users need clear feedback. Vango provides primitives for building connection status UI.

#### Connection Lifecycle

```
┌─────────────────────────────────────────────────────────────────┐
│                    CONNECTION LIFECYCLE                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. Connected      →  Normal operation                          │
│                                                                  │
│  2. Disconnected   →  Within ResumeWindow: attempt reconnect    │
│                       Thin client shows reconnecting indicator   │
│                                                                  │
│  3. Reconnected    →  Session state preserved                   │
│                       Continue where left off                    │
│                                                                  │
│  4. Resume Failed  →  Full page reload (self-heal)              │
│     (after window)    Fresh session, content from SSR           │
│                                                                  │
│  5. Patch Mismatch →  Full page reload (self-heal)              │
│                       DOM re-syncs with server state            │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

#### ConnectionStatus Primitive

`vango.ConnectionStatus()` returns the current connection state as a reactive value:

```go
func ConnectionBanner() vango.Component {
	return vango.Func(func() *vango.VNode {
		// Returns: "connected" | "reconnecting" | "disconnected"
		status := vango.ConnectionStatus()

		// Don't render anything when connected
		if status == "connected" {
			return nil
		}

		return Div(
			Class("fixed top-0 inset-x-0 z-50 p-2 text-center text-white"),
			vango.Switch(status,
				vango.Case("reconnecting",
					Div(Class("bg-yellow-500"),
						Text("Reconnecting..."),
					),
				),
				vango.Case("disconnected",
					Div(Class("bg-red-500"),
						Text("Connection lost. "),
						Button(
							Class("underline"),
							OnClick(func() { vango.ForceReconnect() }),
							Text("Retry"),
						),
					),
				),
			),
		)
	})
}
```

#### Canonical Layout Pattern

Include the connection banner in your root layout:

```go
func Layout(ctx vango.Ctx, children vango.Slot) *vango.VNode {
	return Html(
		Head(
			Meta(Charset("utf-8")),
			Meta(Name("viewport"), Content("width=device-width, initial-scale=1")),
			TitleEl(Text("My App")),
			LinkEl(Rel("stylesheet"), Href(ctx.Asset("styles.css"))),
		),
		Body(
			ConnectionBanner(), // Always visible when disconnected
			Header(/* nav */),
			Main(children),
			Footer(/* footer */),
			VangoScripts(),
		),
	)
}
```

#### Connection Status CSS

Style the banner to be unobtrusive but visible:

```css
/* Reconnecting: yellow warning */
.banner-reconnecting {
	background: #f59e0b;
	animation: pulse 2s infinite;
}

/* Disconnected: red error */
.banner-disconnected {
	background: #ef4444;
}

@keyframes pulse {
	0%, 100% { opacity: 1; }
	50% { opacity: 0.7; }
}
```

#### Programmatic Reconnect

```go
// Force a reconnect attempt (useful for "Retry" buttons)
vango.ForceReconnect()

// Check if currently connected (non-reactive; use ConnectionStatus() for reactive)
if vango.IsConnected() {
	// ...
}
```

**Key constraints:**

- `ConnectionStatus()` is reactive and triggers re-renders when status changes
- The thin client automatically attempts reconnection within `ResumeWindow`
- After `ResumeWindow` expires, the thin client triggers a full page reload (self-heal)
- `ForceReconnect()` is a hint; the thin client may debounce rapid retry attempts

### 12.8 API Routes (Typed JSON Endpoints)

API routes are typed functions that return data, not raw HTTP handlers. They are placed in `app/routes/api/` and registered via `app.API(...)`. This provides a clean separation between server-rendered pages and JSON endpoints for mobile apps, external integrations, or AJAX calls.

#### Basic API Handler

```go
// app/routes/api/health.go
package api

import "github.com/vango-go/vango"

// HealthResponse is the JSON response for health checks.
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// HealthGET handles GET /api/health.
// Registered as: app.API("GET", "/api/health", api.HealthGET)
func HealthGET(ctx vango.Ctx) (*HealthResponse, error) {
	return &HealthResponse{
		Status:  "ok",
		Version: "1.0.0",
	}, nil
}
```

**Naming convention:** `{Resource}{METHOD}` — e.g., `UsersGET`, `UserGET`, `UsersPOST`, `UserDELETE`.

#### CRUD Example

```go
// app/routes/api/users.go
package api

import "github.com/vango-go/vango"

// UsersGET handles GET /api/users - list all users
func UsersGET(ctx vango.Ctx) ([]User, error) {
	return db.Users.List(ctx.StdContext())
}

// UserGET handles GET /api/users/{id} - get single user
type UserParams struct {
	ID int `param:"id"`
}

func UserGET(ctx vango.Ctx, p UserParams) (*User, error) {
	user, err := db.Users.FindByID(ctx.StdContext(), p.ID)
	if err != nil {
		return nil, vango.NotFound("user not found")
	}
	return user, nil
}

// UsersPOST handles POST /api/users - create user
type CreateUserInput struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func UsersPOST(ctx vango.Ctx, input CreateUserInput) (*User, error) {
	return db.Users.Create(ctx.StdContext(), input)
}

// UserDELETE handles DELETE /api/users/{id} - delete user
func UserDELETE(ctx vango.Ctx, p UserParams) (any, error) {
	if err := db.Users.Delete(ctx.StdContext(), p.ID); err != nil {
		return nil, err
	}
	return map[string]bool{"ok": true}, nil
}
```

#### Handler Signatures

API handlers support several signatures:

```go
// No input, returns data
func HealthGET(ctx vango.Ctx) (*Response, error)

// URL params only
func UserGET(ctx vango.Ctx, p Params) (*User, error)

// JSON body only (for POST/PUT/PATCH)
func UsersPOST(ctx vango.Ctx, input CreateInput) (*User, error)

// Both URL params and JSON body
func UserPATCH(ctx vango.Ctx, p Params, input UpdateInput) (*User, error)
```

#### Request Body Handling

When an API handler declares a body parameter (e.g., `input CreateUserInput`), Vango automatically:

1. Reads the HTTP request body
2. JSON-decodes it into the specified type
3. Passes it to your handler

**Content-Type rules:**
* Missing `Content-Type` header: accepted (assumes JSON)
* `Content-Type: application/json`: accepted
* Other `Content-Type` values: rejected with `415 Unsupported Media Type`

**Body size limit:** Default is **1 MiB**. Configure via:

```go
vango.Config{
	API: vango.APIConfig{
		MaxBodyBytes: 5 * 1024 * 1024, // 5 MiB
	},
}
```

#### Error Handling and HTTP Status Codes

Return typed errors to control the HTTP status code:

```go
// 404 Not Found
return nil, vango.NotFound("user not found")

// 400 Bad Request
return nil, vango.BadRequest("invalid email format")

// 401 Unauthorized
return nil, vango.Unauthorized("authentication required")

// 403 Forbidden
return nil, vango.Forbidden("insufficient permissions")

// 500 Internal Server Error (default for untyped errors)
return nil, err

// Custom status code
return nil, vango.WithStatus(422, "validation failed")
```

**Error response format:**

```json
{
  "error": "user not found"
}
```

#### Route Registration (Generated)

The CLI generates route registration in `routes_gen.go`:

```go
// Code generated by vango. DO NOT EDIT.
func Register(app *vango.App) {
	// Page routes
	app.Page("/", routes.IndexPage)
	app.Page("/projects", projects.IndexPage)

	// API routes
	app.API("GET", "/api/health", api.HealthGET)
	app.API("GET", "/api/users", api.UsersGET)
	app.API("GET", "/api/users/:id", api.UserGET)
	app.API("POST", "/api/users", api.UsersPOST)
	app.API("DELETE", "/api/users/:id", api.UserDELETE)
}
```

#### API vs Page Routes

| Aspect | Page Routes (`app.Page`) | API Routes (`app.API`) |
|--------|--------------------------|------------------------|
| Returns | `*vango.VNode` | Data + error (JSON-encoded) |
| Use case | Server-rendered UI | JSON for mobile/external clients |
| Response | HTML (SSR) or patches (WS) | JSON |
| Auth | Session-based | Session or token-based |

**When to use API routes:**
* Mobile app backends
* External integrations
* AJAX calls from islands/hooks
* Webhooks from third-party services

---

## 13. Forms & Validation

Forms in Vango follow a server-driven model with progressive enhancement: they should work as plain HTML forms, and become interactive when connected.

### 13.1 Basic Forms with Signals + Action

A straightforward pattern:

* signals for field state
* action for submit
* `PreventDefault` for SPA-style submit (when connected)

```go
import (
	"context"
	"github.com/vango-go/vango"
	. "github.com/vango-go/vango/el"
)

func ContactForm() vango.Component {
	return vango.Func(func() *vango.VNode {
		ctx := vango.UseCtx()
		name := vango.NewSignal("")
		email := vango.NewSignal("")
		message := vango.NewSignal("")

		submit := vango.NewAction(
			func(std context.Context, _ struct{}) (struct{}, error) {
				return struct{}{}, submitContact(std, name.Get(), email.Get(), message.Get())
			},
			vango.DropWhileRunning(),
		)

		return Form(
			OnSubmit(vango.PreventDefault(func() {
				submit.Run(struct{}{})
			})),

			Label(Text("Name")),
			Input(Type("text"), Value(name.Get()), OnInput(name.Set)),

			Label(Text("Email")),
			Input(Type("email"), Value(email.Get()), OnInput(email.Set)),

			Label(Text("Message")),
			Textarea(Value(message.Get()), OnInput(message.Set)),

			Button(Type("submit"), Disabled(submit.State() == vango.ActionRunning), Text("Send")),
			submit.State() == vango.ActionRunning && Div(Text("Sending…")),
			submit.State() == vango.ActionError && Div(Class("text-red-600"), Text(submit.Error().Error())),
			submit.State() == vango.ActionSuccess && Div(Class("text-green-700"), Text("Sent")),
		)
	})
}
```

### 13.2 Validation (Server-Side Source of Truth)

All validation must be server-side (even if you add client hints). Typical pattern:

* validate in service/action
* return structured field errors
* render them near fields

One workable approach is to keep a signal of validation errors keyed by field name:

```go
type FieldError struct {
	Field   string
	Message string
}

func errorFor(errs []FieldError, field string) string {
	for _, e := range errs {
		if e.Field == field {
			return e.Message
		}
	}
	return ""
}
```

Then in the component, set `errsSignal.Set(errs)` on validation failure and render `errorFor(...)` under each input. Keep the UI side minimal and let the server define correctness.

### 13.3 Progressive Enhancement

Forms and links should work without WebSocket:

* Without WS: normal POST navigation
* With WS: Vango intercepts/enhances for SPA-style UX

In practice, that means:

* always set proper `Method(...)`, `Action(...)`, and `Name(...)` attributes when you intend a form to be usable without WS
* use `PreventDefault` + Action when you want in-place submits

Keep "enhanced" behavior additive rather than required.

### 13.4 File Uploads

> **Important:** File uploads use **plain HTTP handlers**, not typed API endpoints. Multipart form data requires direct access to `http.Request` which the typed API abstraction doesn't expose. Mount upload handlers separately on your router.

Vango provides the `upload` package for secure file upload handling with a two-phase pattern:

1. **Phase 1 (HTTP):** Client uploads file → server stores temporarily → returns `temp_id`
2. **Phase 2 (WebSocket):** Client submits form with `temp_id` → server claims file and processes

#### Setting Up the Upload Handler

```go
// main.go
import "github.com/vango-go/vango/pkg/upload"

func main() {
    // Create a disk-based upload store (dir, maxSize)
    store, err := upload.NewDiskStore("/tmp/uploads", 10*1024*1024)
    if err != nil {
        log.Fatal(err)
    }

    // Mount the upload handler (plain HTTP, not typed API)
    mux.Handle("POST /api/upload", upload.Handler(store))

    // Or with custom config
    mux.Handle("POST /api/upload", upload.HandlerWithConfig(store, &upload.Config{
        MaxFileSize:       10 * 1024 * 1024, // 10MB
        AllowedTypes:      []string{"image/png", "image/jpeg", "application/pdf"},
        AllowedExtensions: []string{".png", ".jpg", ".jpeg", ".pdf"},
        TempExpiry:        time.Hour,
    }))
}
```

The handler returns JSON: `{"temp_id": "abc123"}`

#### Client-Side Form

```go
func FileUploadForm() vango.Component {
    return vango.Func(func() *vango.VNode {
        tempID := vango.NewSignal("")
        uploading := vango.NewSignal(false)

        return Div(
            // Phase 1: Upload form (standard HTTP POST)
            Form(
                Method("POST"),
                Action("/api/upload"),
                EncType("multipart/form-data"),
                Target("upload-frame"), // Use hidden iframe for no-reload upload

                Input(Type("file"), Name("file"), Accept("image/*")),
                Button(Type("submit"), Text("Upload")),
            ),

            // Hidden iframe to capture upload response
            IFrame(ID("upload-frame"), Name("upload-frame"), Style("display:none")),

            // Phase 2: Process form (once we have temp_id)
            tempID.Get() != "" && Form(
                OnSubmit(vango.PreventDefault(func() {
                    // Server action claims the file using temp_id
                })),
                Input(Type("hidden"), Name("temp_id"), Value(tempID.Get())),
                Button(Type("submit"), Text("Save")),
            ),
        )
    })
}
```

#### Claiming Files in Your Handler

```go
func ProcessUpload(ctx vango.Ctx, input struct{ TempID string }) error {
    // Claim the file from temporary storage
    file, err := upload.Claim(store, input.TempID)
    if err != nil {
        if errors.Is(err, upload.ErrNotFound) {
            return vango.BadRequest("upload expired or invalid")
        }
        return err
    }
    defer file.Close()

    // file.Filename - original filename
    // file.ContentType - detected MIME type (sniffed, not trusted from client)
    // file.Size - file size in bytes
    // file.Path - local path (for DiskStore)
    // file.Reader - io.ReadCloser for the content

    // Move to permanent storage, save to database, etc.
    return nil
}
```

#### Storage Backends

The `upload.Store` interface supports custom backends:

```go
type Store interface {
    Save(filename, contentType string, size int64, r io.Reader) (tempID string, err error)
    Claim(tempID string) (*File, error)
    Cleanup(maxAge time.Duration) error
}
```

Vango provides `DiskStore` out of the box. For S3/GCS, implement the interface or use community packages.

#### Security Considerations

- **MIME type sniffing**: The handler detects content type from file bytes, never trusting client headers
- **Size limits**: Applied at HTTP level via `MaxBytesReader` before parsing
- **Extension validation**: Optional `AllowedExtensions` and `RequireExtensionMatch` for defense-in-depth
- **Temp file cleanup**: Call `store.Cleanup(maxAge)` periodically (e.g., cron job every 5 minutes)

**Key point:** Because uploads are plain HTTP handlers, they don't participate in Vango's typed API system. This is intentional—multipart parsing requires raw request access. Document your upload endpoints separately from your typed API routes.

---

## 14. The Client Boundary (Escape Hatches)

Vango is optimized for server-driven UI, but it supports client-side extensions for cases where network roundtrips or DOM ownership constraints make server-driven insufficient.

### 14.1 When to Leave Go (Guidelines)

Use client-side code only when needed:

* **60fps interactions**: dragging, resizing, drawing, complex focus/selection behavior
* **third-party widgets**: editors, maps, visualization libraries that require full DOM ownership
* **compute-heavy client work**: image processing, physics, offline computation (WASM)

Otherwise, keep logic in Go for simplicity and security.

### 14.2 Hooks (Client-side behavior attached to DOM elements)

Hooks run on the client for specific elements and can send events back to the server. Vango includes common hooks (e.g., Sortable, Tooltip, Dialog, ThemeToggle) and supports custom hooks.

**Server-side wiring pattern:**

```go
Div(
	Hook("Sortable", map[string]any{
		"handle": ".drag-handle",
	}),
	OnEvent("reorder", func(e vango.HookEvent) {
		from := e.Int("fromIndex")
		to := e.Int("toIndex")
		// update state + persist
	}),
)
```

#### Standard Hooks Library

The thin client includes these built-in hooks—no registration or additional JS required:

| Hook | Purpose | Common Config | Events |
|------|---------|---------------|--------|
| `Sortable` | Drag-to-reorder lists | `handle`, `group`, `animation`, `ghostClass` | `reorder` (fromIndex, toIndex) |
| `Draggable` | Make elements draggable | `axis`, `bounds`, `handle` | `dragstart`, `dragend`, `drag` |
| `Droppable` | Drop targets for draggables | `accept`, `hoverClass` | `drop`, `dragenter`, `dragleave` |
| `Resizable` | Resize elements | `handles`, `minWidth`, `minHeight` | `resize`, `resizeend` |
| `Tooltip` | Hover tooltips | `content`, `placement`, `delay` | — |
| `Dropdown` | Dropdown menus | `trigger`, `placement`, `closeOnClick` | `open`, `close` |
| `Collapsible` | Expand/collapse content | `open`, `duration` | `toggle` |
| `FocusTrap` | Trap focus within element (modals) | `initialFocus`, `returnFocus` | — |
| `Portal` | Render content elsewhere in DOM | `target` | — |
| `Dialog` | Modal dialogs | `open`, `closeOnEscape`, `closeOnOverlay` | `close` |
| `Popover` | Popovers (anchored to trigger) | `trigger`, `placement`, `offset` | `open`, `close` |
| `ThemeToggle` | Light/dark theme switching | `storageKey`, `defaultTheme` | `change` |

**Go-side helpers** (optional typed wrappers):

```go
import "github.com/vango-go/vango/pkg/features/hooks/standard"

// Using typed helper (provides autocomplete for config)
Div(standard.Sortable(standard.SortableConfig{Handle: ".drag-handle", Animation: 150}))

// Or use generic Hook() with map
Div(Hook("Sortable", map[string]any{"handle": ".drag-handle"}))
```

#### Hook Registration Contract

**How the thin client discovers hooks:**

1. **Built-in hooks** (listed above) ship with the Vango thin client—just use `Hook("Name", config)`
2. **Custom hooks** must be registered before first use via:
   ```javascript
   // public/js/hooks/my_hook.js
   import { MySortableHook } from './my_sortable.js';

   // Register when Vango client is ready
   window.__vango__?.registerHook('MySortable', MySortableHook);
   ```
3. **Bundle location**: custom hooks live in `public/js/hooks/` and are loaded via `<script>` tags or bundled with your app's JS

**Hook interface (JavaScript):**

```javascript
// public/js/hooks/my_sortable.js
export class MySortableHook {
    mounted(el, config, pushEvent) {
        // Called when element enters DOM
        // el: the DOM element with the hook
        // config: the map[string]any from Go (parsed from data-hook-config)
        // pushEvent: function to send events to server → pushEvent('eventName', {data}, revertFn?)
        this.el = el;
        this.instance = new Sortable(el, {
            handle: config.handle,
            onEnd: (evt) => {
                // Optional third arg: revert function called if server rejects
                pushEvent('reorder', {
                    fromIndex: evt.oldIndex,
                    toIndex: evt.newIndex,
                }, () => { /* revert DOM changes */ });
            },
        });
    }

    updated(el, config, pushEvent) {
        // Called when server patches the element (config may have changed)
        // pushEvent is provided here too for hooks that send events on update
        this.config = config;
    }

    destroyed(el) {
        // Called when element leaves DOM; cleanup resources
        this.instance.destroy();
    }
}
```

**Rendered HTML attributes:** The `Hook()` function renders `data-hook="Name"` and `data-hook-config='{"key":"value"}'` attributes. The thin client queries `[data-hook]` to discover and initialize hooks.

**Naming convention:** Hook names are PascalCase (e.g., `"Sortable"`, `"ThemeToggle"`). The thin client looks up `window.__vango__.hooks["HookName"]` at mount time.

**Key constraint:** hooks should treat their DOM subtree as Vango-managed; they should avoid directly mutating structure unless the hook is designed to do so safely and communicate results back to the server.

### 14.3 JS Islands (Opaque DOM subtrees)

Use islands when a third-party library needs to own a subtree completely. Vango will not patch inside the island boundary; updates occur via island message passing.

**Server-side usage pattern:**

```go
// Rich text editor island
func RichEditor(content string) *vango.VNode {
    return Div(
        ID("editor"),
        JSIsland("rich-editor", map[string]any{
            "content":     content,
            "placeholder": "Write something...",
        }),
    )
}
```

**Island definition (JavaScript):**

```javascript
// public/js/islands/rich-editor.js
export default {
    mount(container, config, send) {
        // Third-party editor owns this DOM
        this.editor = new RichTextEditor(container, {
            content: config.content,
            placeholder: config.placeholder,
            onChange: (newContent) => {
                send('change', { content: newContent });
            },
        });
    },

    update(config) {
        // Receive updates from server
        if (config.content !== this.editor.getContent()) {
            this.editor.setContent(config.content);
        }
    },

    destroy() {
        this.editor.destroy();
    },
};
```

**Handle island messages server-side:**

```go
vango.OnIslandMessage("rich-editor", func(event string, data map[string]any) {
    if event == "change" {
        content := data["content"].(string)
        contentSignal.Set(content)
    }
})
```

**Key constraint:** treat island events as untrusted input. Validate all data sent from islands before using it. This boundary prevents patch mismatch issues by separating DOM ownership.

### 14.4 WASM (Client compute / offline-first)

WASM compiles your Go components to WebAssembly, running them entirely in the browser. This is a fundamentally different execution model from server-driven rendering.

#### When to Use WASM: Decision Matrix

| Use Case | Server-Driven | WASM |
|----------|:-------------:|:----:|
| CRUD apps, dashboards, admin panels | ✓ | |
| Forms, wizards, multi-step workflows | ✓ | |
| Real-time collaboration (cursors, presence) | ✓ | |
| Image/video processing in browser | | ✓ |
| Complex canvas graphics (drawing apps) | | ✓ |
| Offline-first with local persistence | | ✓ |
| Physics simulations, games | | ✓ |
| Latency-critical interactions (<16ms) | | ✓ |
| Low-bandwidth environments | ✓ | |
| SEO-critical content | ✓ | |
| Simple drag-drop, tooltips, animations | Hooks | |

**Default to server-driven.** For 95% of web applications, server-driven mode provides the best balance of simplicity, security, and performance. WASM adds significant complexity (larger bundles ~300KB+, client-side state management, offline sync logic) and should only be chosen when you have a specific requirement that server-driven cannot meet.

#### Tradeoffs Summary

| Aspect | Server-Driven | WASM |
|--------|---------------|------|
| Bundle size | ~12KB | ~300KB+ |
| State location | Server (secure) | Browser (requires validation) |
| Offline support | Requires connection | Works fully offline |
| Interaction latency | ~50-100ms (network) | <1ms (local) |
| Security model | Server-authoritative | Must validate everything |
| Complexity | Lower | Higher |

#### Pattern

```go
Div(
	ID("physics"),
	WASMWidget("physics", map[string]any{"gravity": 9.8}),
)
```

As with islands, validate any messages that come back to the server and keep authoritative state/permissions server-side when applicable. WASM components should treat server state as authoritative for anything security-sensitive.

---

## 15. CLI & Tooling

The CLI provides scaffolding, dev server, route generation, and production builds. This workflow is what enables file-based routing and the "pit of success" defaults.

### 15.1 Development Server: `vango dev`

The development server provides:

- **Hot reload**: Go file changes trigger recompile and browser refresh
- **Tailwind watch**: CSS rebuilds on class name changes (if using Tailwind)
- **Route regeneration**: New pages automatically register routes
- **DevTools integration**: Browser extension support for debugging

```bash
# Start dev server
vango dev

# With options
vango dev --port 3000        # Custom port (default: 8080)
vango dev --host 0.0.0.0     # Listen on all interfaces
vango dev --no-browser       # Don't auto-open browser
```

**What `vango dev` does internally:**

```
1. Generates route glue (app/routes/routes_gen.go)
2. Starts Tailwind watcher (if configured)
3. Compiles and runs your app
4. Watches for file changes
5. On change: regenerate routes → recompile → restart → notify browser
```

**Development Mode Behavior:**

During development (`DevMode: true`), Vango enables:

- Detailed error pages with stack traces and source locations
- Effect strictness warnings (warns on effect-time writes)
- DevTools transaction names include source file:line references
- Longer timeouts for step-through debugging

### 15.2 Development vs Production Mode

| Behavior | Development | Production |
|----------|-------------|------------|
| Error pages | Detailed with stack traces | Generic error messages |
| Effect strictness | Warn on effect-time writes | Off (unless configured) |
| DevTools | Full transaction names with source locations | Component-level names only |
| Session store | In-memory (lost on restart) | Redis or database |
| Static caching | No caching (reload on every request) | Long-lived with fingerprinting |

### 15.3 Scaffolding: `vango create`

`vango create` scaffolds a new project with templates:

```bash
# Create with default template (Tailwind + example pages)
vango create myapp

# Create in current directory
vango create .

# Create with minimal template (bare bones)
vango create myapp --minimal

# Create with all features (Tailwind, database, auth)
vango create myapp --full
```

**Available Templates:**

| Template | Description |
|----------|-------------|
| **(default)** | Standard starter with home, about, health API, navbar, footer, and Tailwind CSS |
| `--minimal` | Bare bones: single page, no Tailwind |
| `--with-tailwind` | Explicitly include Tailwind CSS (managed via standalone binary) |
| `--with-db=sqlite` | Include database setup (sqlite, postgres) |
| `--with-auth` | Include session-based authentication scaffolding |
| `--full` | All features: Tailwind, database (sqlite), auth |

### 15.4 Code Generators

The CLI provides generators for common patterns:

```bash
# Regenerate route glue (required after adding/removing route files)
vango gen routes

# Generate a new component
vango gen component UserCard

# Generate a new route with registration
vango gen route projects/[id]
```

Route generation is **deterministic** and treated as build output. The generated `routes_gen.go` file should be committed to version control.

### 15.5 Build: `vango build`

For production, you need a single binary with fingerprinted assets:

```bash
vango build
```

**Build Output:**

```
dist/
├── server              # Go binary
├── public/             # Fingerprinted static assets
│   ├── styles.a1b2c3.css
│   └── ...
└── manifest.json       # Asset name → fingerprinted name mapping
```

Your app loads the manifest in production so `ctx.Asset("styles.css")` resolves to `styles.a1b2c3.css`.

### 15.6 Testing Workflow

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific package
go test ./app/routes/...

# Run with coverage
go test -cover ./...

# Run with race detector (recommended in CI)
go test -race ./...
```

### 15.7 CI/CD Setup

**GitHub Actions Example:**

```yaml
# .github/workflows/ci.yml
name: CI

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Install Vango CLI
        run: go install github.com/vango-go/vango/cmd/vango@latest

      - name: Generate routes
        run: vango gen routes

      - name: Build project (includes Tailwind)
        run: vango build

      - name: Run tests
        run: go test -race -v ./...

  deploy:
    needs: test
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v4
      # ... build and deploy steps
```

**CI/CD Checklist:**

- [ ] `vango gen routes` runs before build (or verify `routes_gen.go` is committed)
- [ ] `vango build` produces the binary and fingerprinted assets
- [ ] `go test -race ./...` runs with race detector
- [ ] Asset manifest is included in deployment artifacts

**Reproducible Builds:**

```bash
# Pin Vango CLI version for reproducible CI
go install github.com/vango-go/vango/cmd/vango@v1.2.3

# Or use Go's tool directive in go.mod (Go 1.22+)
```

# Part V: Production & Operations

This part covers “Day 2” concerns: authentication, security, observability, persistence, scaling, and deployment. It integrates the operational constraints implied by Vango’s server-driven, session-based architecture: per-tab sessions, WebSocket upgrades, session resume windows, and strict render purity.

All examples assume:

```go
import "github.com/vango-go/vango"
import . "github.com/vango-go/vango/el"
```

(Plus standard library imports where needed.)

---

## 16. Auth & Middleware

Authentication in Vango is an **HTTP → Session** problem. Credentials and cookies exist in the HTTP request pipeline, while interactive rendering occurs over a long-lived WebSocket session. You must bridge identity from HTTP into the session so it remains available during WebSocket renders.

### 16.1 The Context Bridge (HTTP Identity → Session Identity)

**Key fact:** during WebSocket renders, `ctx.Request()` is not a full real HTTP request (the original headers/cookies are not present). Therefore:

* do not depend on headers/cookies during reactive renders
* capture identity during SSR / session start and store it in session-scoped state

There are two complementary bridge points described in the current guide:

1. **HTTP middleware** attaches identity to the `context.Context` of the request.
2. **Vango session start/resume hooks** copy that identity into the session.

#### HTTP middleware attaches user to request context

This is standard Go middleware:

```go
import (
	"net/http"
	"context"
	"github.com/vango-go/vango"
)

type User struct {
	ID   int
	Role string
}

func withUser(ctx context.Context, u *User) context.Context {
	return vango.WithUser(ctx, u)
}

func AuthHTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, err := validateFromCookie(r) // your auth
		if err == nil && u != nil {
			r = r.WithContext(withUser(r.Context(), u))
		}
		next.ServeHTTP(w, r)
	})
}
```

#### Session start/resume copies the user into session storage

Use `vango.Config{ OnSessionStart, OnSessionResume }` to ensure the session has identity even after reconnect/refresh within the resume window:

```go
import (
	"context"
	"github.com/vango-go/vango"
)

func NewApp() *vango.App {
	app := vango.New(vango.Config{
		OnSessionStart: func(httpCtx context.Context, s *vango.Session) {
			if u, ok := vango.UserFromContext(httpCtx).(*User); ok && u != nil {
				// Store user in the session for WS renders
				s.Set("user", u)
			}
		},

		OnSessionResume: func(httpCtx context.Context, s *vango.Session) error {
			// Revalidate from the HTTP request context during resume.
			// If it fails, reject resume and force a new session.
			u, err := validateFromRequestContext(httpCtx)
			if err != nil {
				return err
			}
			s.Set("user", u)
			return nil
		},
	})
	return app
}
```

Then in any render (SSR or WS), read identity from the session:

```go
func CurrentUser() *User {
	ctx := vango.UseCtx()
	if sess := ctx.Session(); sess != nil {
		if u, ok := sess.Get("user").(*User); ok {
			return u
		}
	}
	// SSR fallback: ctx.User() may exist during SSR if middleware attached it
	if u, ok := ctx.User().(*User); ok {
		return u
	}
	return nil
}
```

This pattern makes identity stable across SSR, WS renders, and session resume.

### 16.2 Route Guards (Middleware for Protection)

Vango supports directory-level middleware that runs inside Vango’s routing pipeline (distinct from HTTP middleware). Use it for route protection (redirects, authorization decisions that depend on session identity).

Example: `app/routes/admin/middleware.go`

```go
import "github.com/vango-go/vango"

func Middleware() []any {
	// Exact middleware type depends on your router integration; the current guide
	// expresses it as Vango route middleware that can short-circuit with Navigate.
	return []any{
		func(ctx vango.Ctx, next func() error) error {
			u := CurrentUser()
			if u == nil {
				ctx.Navigate("/login?redirect="+ctx.Path(), vango.WithReplace())
				return nil
			}
			if u.Role != "admin" {
				ctx.Navigate("/forbidden", vango.WithReplace())
				return nil
			}
			return next()
		},
	}
}
```

**Operational principle:** enforce authorization both at the route boundary (guards) and at mutation boundaries (actions). Never rely solely on UI to hide privileged controls.

### 16.3 Session Durability and Secure Resume

Vango sessions are per-tab. Durability tiers:

* **No resume:** `ResumeWindow: 0`
* **In-memory resume:** `ResumeWindow > 0`, sessions can reattach within the window after refresh/reconnect
* **Persistent sessions:** configure a session store (e.g., Redis) to survive process restarts and support non-sticky deployments

Important security integration:

* authenticated sessions should not silently resume with stale identity
* if a session was authenticated before disconnect, rehydration on resume must occur; otherwise resume should be rejected and the client will reload into the HTTP pipeline

Practically:

* set a resume window appropriate for your UX (e.g., 30s)
* for production, use a store when you need restart resilience or horizontal scaling
* always implement `OnSessionResume` to revalidate credentials for authenticated apps

### 16.4 Auth Freshness in Long-Lived Sessions

Traditional web frameworks validate auth on every HTTP request. Vango breaks this model:

```
Traditional HTTP:
Request → Middleware validates → Handler → Response (can set cookies)
Request → Middleware validates → Handler → Response
...repeats for every interaction...

Vango WebSocket:
HTTP Request → Middleware validates → SSR → Response
    ↓
WebSocket connects (cookies sent ONCE during upgrade)
    ↓
Event → Handler → Patch (NO cookies, NO middleware)
Event → Handler → Patch
...continues for hours without HTTP validation...
```

**Real-world impact:**

| Scenario | Risk Without Freshness Checks |
|----------|-------------------------------|
| User terminated from company | Continues accessing internal tools for hours |
| Account compromised, password changed | Attacker's session stays active |
| Subscription expires | User continues using premium features |
| Token naturally expires | Silent failures or undefined behavior |

Vango addresses this with a **Passive + Active** auth freshness model.

### 16.5 The Passive + Active Model

```
┌─────────────────────────────────────────────────────────────────┐
│                    AUTH FRESHNESS MODEL                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  PASSIVE CHECK (Every Event)                                    │
│  ────────────────────────────                                   │
│  Cost: ~1μs (local timestamp comparison)                        │
│  Logic: if time.Now() > session expiry timestamp                │
│  Purpose: Catch naturally expired tokens/sessions               │
│  Implementation: Automatic if SessionKeyExpiryUnixMs is set     │
│                                                                  │
│  ACTIVE CHECK (Periodic / On-Demand)                            │
│  ───────────────────────────────────                            │
│  Cost: Network round-trip (Redis, provider API, or database)    │
│  Logic: Call configured Check function                          │
│  Purpose: Catch revoked sessions, auth version bumps            │
│  Implementation: Session-loop-native timer (async, bounded)     │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Passive checks** are automatic and nearly free. **Active checks** require configuration but catch revocations that passive checks cannot detect.

### 16.6 Configuring Active Revalidation (`AuthCheckConfig`)

```go
import (
	"context"
	"time"

	"github.com/vango-go/vango"
	"github.com/vango-go/vango/pkg/auth"
)

app := vango.New(vango.Config{
	Session: vango.SessionConfig{
		// ... other session config ...

		AuthCheck: &vango.AuthCheckConfig{
			// How often to run active checks (e.g., every 5 minutes)
			Interval: 5 * time.Minute,

			// Timeout for each check (default: 5s)
			Timeout: 5 * time.Second,

			// What to do on transient failures (timeouts, network errors)
			// FailOpenWithGrace: keep session alive but expire after MaxStale
			// FailClosed: expire immediately on any failure
			FailureMode: vango.FailOpenWithGrace,

			// Max time since last successful check before forced expiry
			// Only applies when FailureMode is FailOpenWithGrace
			MaxStale: 15 * time.Minute,

			// The check function (runs in goroutine, MUST NOT access session directly)
			Check: func(ctx context.Context, p auth.Principal) error {
				// Verify session is still valid with your auth backend
				return sessionStore.Validate(ctx, p.SessionID)
			},

			// What to do when auth expires
			OnExpired: vango.AuthExpiredConfig{
				Action: vango.ForceReload, // or NavigateTo, Custom
				Path:   "/login",          // for NavigateTo
			},
		},
	},
})
```

**AuthExpiredAction options:**

| Action | Behavior | Use Case |
|--------|----------|----------|
| `ForceReload` | `location.reload()` (re-enters HTTP pipeline) | Default; works with all providers |
| `NavigateTo` | `location.assign(path)` (hard navigation) | Custom login page |
| `Custom` | Call your handler function | Complex logout flows |

**Failure modes:**

| Mode | Behavior | When to Use |
|------|----------|-------------|
| `FailOpenWithGrace` | Keep session alive on transient failures; expire after `MaxStale` | Most apps (availability-first) |
| `FailClosed` | Expire immediately on any check failure | High-security apps |

**Key constraints:**

- The `Check` function runs in a goroutine (safe for network I/O)
- It receives a `Principal` snapshot and MUST NOT access session state directly
- Results are dispatched back to the session loop (single-writer safe)
- Maximum one in-flight check per session (no goroutine pileup)
- Hard failures (`auth.ErrSessionRevoked`, `auth.ErrSessionExpired`) always expire immediately

### 16.7 On-Demand Revalidation for High-Value Actions

For sensitive operations (transfers, deletions, permission changes), revalidate immediately before execution:

```go
func TransferAction(ctx vango.Ctx, amount int, toAccount string) error {
	// Revalidate auth before high-value operation
	if err := ctx.RevalidateAuth(); err != nil {
		// Session will transition to expired state; abort the operation
		return err
	}

	// Auth is fresh; proceed with transfer
	return billing.Transfer(ctx.StdContext(), amount, toAccount)
}
```

**Key points:**

- `ctx.RevalidateAuth()` runs the configured `Check` function synchronously
- On failure, the session transitions to expired behavior (reload/navigate)
- For non-blocking flows, run revalidation in a goroutine and use `ctx.Dispatch()` to update state

### 16.8 Runtime-Only Session Keys (Important)

Vango supports session persistence (for server restarts, non-sticky deploys). However, auth state should **not** be persisted—it must be rehydrated from the source of truth on every session start/resume.

**Contract:**

- `vango:auth:principal` and `vango:auth:expiry_unix_ms` are **runtime-only** session keys
- They **MUST NOT** be serialized into the session persistence blob
- They **MUST** be rehydrated in `OnSessionStart` and `OnSessionResume` from HTTP context + your auth backend

```go
import "github.com/vango-go/vango/pkg/auth"

OnSessionStart: func(httpCtx context.Context, s *vango.Session) {
	principal, ok := provider.Principal(httpCtx)
	if !ok {
		return // Guest user
	}

	// Store runtime-only auth state (never persisted)
	s.Set(auth.SessionKeyPrincipal, principal)
	s.Set(auth.SessionKeyExpiryUnixMs, principal.ExpiresAtUnixMs)
	s.Set(auth.SessionKeyHadAuth, true) // This boolean MAY be persisted
},

OnSessionResume: func(httpCtx context.Context, s *vango.Session) error {
	wasAuth, _ := s.Get(auth.SessionKeyHadAuth).(bool)

	principal, ok := provider.Principal(httpCtx)
	if !ok || principal.ID == "" {
		if wasAuth {
			// Was authenticated but can't rehydrate → reject resume
			return auth.ErrSessionRevoked
		}
		return nil // Guest session
	}

	// Rehydrate runtime-only keys
	s.Set(auth.SessionKeyPrincipal, principal)
	s.Set(auth.SessionKeyExpiryUnixMs, principal.ExpiresAtUnixMs)

	// Optional: run active check on resume
	return provider.Verify(httpCtx, principal)
},
```

**Why runtime-only?**

- Prevents "stale principal after restart" bugs
- Avoids reliance on JSON type fidelity (time.Time, structs become maps)
- Aligns with session-first auth: source of truth is your session store, not a Vango snapshot

### 16.9 Auth Expiry UX (Multi-Tab Coordination)

When auth expires, all tabs for the same user should be notified. Vango uses `BroadcastChannel` (with `localStorage` fallback) for cross-tab coordination.

**How it works:**

1. Server sends auth expiry command to the affected session
2. Thin client broadcasts `{type: 'expired'}` to `BroadcastChannel('vango:auth')`
3. Other tabs receive the broadcast and reload into the HTTP pipeline
4. All tabs re-enter the auth flow together

**Client-side (built into thin client):**

```javascript
// Simplified from Vango thin client
class AuthCoordinator {
	constructor() {
		this.channel = 'vango:auth';

		if (typeof BroadcastChannel !== 'undefined') {
			this.bc = new BroadcastChannel(this.channel);
			this.bc.onmessage = (e) => this.handleMessage(e.data);
			return;
		}

		// Fallback: localStorage events (works cross-tab, same origin)
		window.addEventListener('storage', (e) => {
			if (e.key === `__vango_auth_${this.channel}` && e.newValue) {
				this.handleMessage(JSON.parse(e.newValue).payload);
			}
		});
	}

	handleMessage(payload) {
		if (payload?.type === 'expired' || payload?.type === 'logout') {
			window.location.reload();
		}
	}

	broadcast(type, reason) {
		if (this.bc) {
			this.bc.postMessage({ type, reason });
			return;
		}
		// localStorage fallback
		const key = `__vango_auth_${this.channel}`;
		localStorage.setItem(key, JSON.stringify({ payload: { type, reason }, ts: Date.now() }));
		localStorage.removeItem(key);
	}
}
```

**Server-side expiry triggers broadcast automatically:**

When `triggerAuthExpired()` is called (passive expiry, active check failure, or on-demand failure), Vango sends a broadcast command before terminating the session.

### 16.10 Session-First Auth (Recommended Pattern)

For most Vango apps, **session-first auth** is recommended over JWT-first:

| Aspect | JWT-First | Session-First |
|--------|-----------|---------------|
| Token expiry | Problem (mid-session failure) | Non-issue (you control TTL) |
| Revocation | Hard (need denylist/introspection) | Easy (delete from store) |
| Token refresh | Complex over WebSocket | N/A (server extends session) |
| State location | Split (token + server) | Unified (server) |

**The session-first flow:**

1. User logs in with OAuth/OIDC provider (Clerk, Auth0, etc.)
2. Server exchanges code for tokens, gets user info
3. Server creates a **first-party session** in Redis/Postgres
4. Server sets an HttpOnly session cookie
5. On each HTTP request, middleware validates the session
6. `OnSessionStart`/`OnSessionResume` bridges identity into Vango session
7. Active checks verify the session is still valid (auth version, expiry)

**Revocation strategies:**

| Strategy | Implementation | Use Case |
|----------|---------------|----------|
| Delete session | `DELETE FROM sessions WHERE id = ?` | Single session logout |
| Delete all user sessions | `DELETE FROM sessions WHERE user_id = ?` | "Log out everywhere" |
| Bump auth version | `UPDATE users SET auth_version = auth_version + 1` | Password change, security event |

The auth version strategy stores a version in both the session and user record. Active checks compare them—if the session version is less than the current user version, the session is revoked.

### 16.11 Provider Adapters

Vango provides adapter interfaces for common auth providers. The core `auth.Provider` interface:

```go
package auth

type Provider interface {
	// Middleware validates HTTP requests and populates context
	Middleware() func(http.Handler) http.Handler

	// Principal extracts identity from validated request context
	Principal(ctx context.Context) (Principal, bool)

	// Verify checks if session is still valid (for active revalidation)
	Verify(ctx context.Context, p Principal) error
}
```

**Available adapters:**

| Adapter | Module | Go Version | Notes |
|---------|--------|------------|-------|
| `sessionauth` | `vango/pkg/auth/sessionauth` | 1.22+ | Recommended; works with any session store |
| `clerk` | `github.com/vango-go/vango-clerk` | 1.24+ | Separate module (Clerk SDK requirement) |
| `auth0` | `github.com/vango-go/vango-auth0` | 1.22+ | JWT-first with auth version support |

**Example: Session-first with Redis**

```go
import (
	"github.com/vango-go/vango/pkg/auth/sessionauth"
	"myapp/internal/sessions"
)

// Create session store
sessionStore := sessions.NewRedisStore(redisClient)

// Create provider
provider := sessionauth.New(sessionStore)

// Use in config
app := vango.New(vango.Config{
	Session: vango.SessionConfig{
		OnSessionStart: func(httpCtx context.Context, s *vango.Session) {
			principal, ok := provider.Principal(httpCtx)
			if !ok {
				return
			}
			s.Set(auth.SessionKeyPrincipal, principal)
			s.Set(auth.SessionKeyExpiryUnixMs, principal.ExpiresAtUnixMs)
		},
		OnSessionResume: func(httpCtx context.Context, s *vango.Session) error {
			principal, ok := provider.Principal(httpCtx)
			if !ok {
				return auth.ErrSessionRevoked
			}
			s.Set(auth.SessionKeyPrincipal, principal)
			s.Set(auth.SessionKeyExpiryUnixMs, principal.ExpiresAtUnixMs)
			return provider.Verify(httpCtx, principal)
		},
		AuthCheck: &vango.AuthCheckConfig{
			Interval: 5 * time.Minute,
			Check: func(ctx context.Context, p auth.Principal) error {
				return provider.Verify(ctx, p)
			},
		},
	},
})

// Wrap with HTTP middleware
handler := provider.Middleware()(app)
```

---

## 17. Observability

You need visibility into a server-driven UI runtime: sessions, events, patches, resource/action activity, and failures (patch mismatch reloads, auth expiry, etc.).

### 17.1 Structured Logging (`slog` integration)

Vango supports injecting a logger via config (current guide references `Logger: slog.Default()`).

```go
import "log/slog"

app := vango.New(vango.Config{
	Logger: slog.Default(),
	DevMode: false,
})
```

At the application layer, log intent around mutations and security decisions. Avoid logging secrets/PII.

```go
func DeleteProject(projectID int) vango.Component {
	return vango.Func(func() *vango.VNode {
		ctx := vango.UseCtx()
		u := CurrentUser()

		del := vango.NewAction(func(std context.Context, _ struct{}) (struct{}, error) {
			slog.Info("delete_project_attempt", "user_id", u.ID, "project_id", projectID)
			err := deleteProject(std, u, projectID)
			if err != nil {
				slog.Error("delete_project_failed", "user_id", u.ID, "project_id", projectID, "error", err)
				return struct{}{}, err
			}
			slog.Info("delete_project_succeeded", "user_id", u.ID, "project_id", projectID)
			return struct{}{}, nil
		}, vango.DropWhileRunning())

		return Button(OnClick(func() { del.Run(struct{}{}) }), Text("Delete"))
	})
}
```

### 17.2 Metrics (Sessions, Patch Sizes, Latency)

Even if you don’t adopt a full metrics stack on day one, there are specific signals you should track because they correspond to operational failure modes:

* active sessions
* detached sessions (waiting in resume window)
* event rate
* patch sizes (p95/p99)
* render/handler durations
* resource/action error rates
* storm-budget exceeded counts (see below)

These metrics let you answer:

* “Are we CPU-bound due to too many renders?”
* “Are patches huge because keys are unstable or we’re re-rendering too much?”
* “Are we leaking detached sessions?”

### 17.3 Tracing (OpenTelemetry)

The current guide describes using `ctx.StdContext()` for I/O, which is exactly where trace propagation belongs: service/database calls should accept `context.Context` and start spans inside those functions.

Operationally:

* trace “event → handler → service calls → patches sent”
* trace “resource load → DB/HTTP”
* trace “action run → side effects → navigation”

Even without deep internal instrumentation, ensuring your app-level services use the passed context is the key move.

---

## 18. Persistence & Scaling

Vango apps are stateful by default (server-side per-tab sessions). Scaling and persistence therefore revolve around session storage, routing strategy, and controlling runaway work.

### 18.1 Session Stores (Redis / Persistence Across Restarts)

In development, an in-memory session store is fine. In production, if you want:

* resilience across process restarts
* non-sticky load balancing
* safe rolling deploys without forcing everyone to lose state

…configure a store.

The current guide expresses this via `vango.SessionConfig{ Store: vango.RedisStore(...) }` (and notes a “PostgresStore” option as well).

```go
import "time"

app := vango.New(vango.Config{
	Session: vango.SessionConfig{
		ResumeWindow: 30 * time.Second,
		Store:        vango.RedisStore(redisClient), // production durability
		MaxDetachedSessions: 10000,
		MaxSessionsPerIP:    100,
	},
})
```

**Operational note:** per-IP session limits depend on correct proxy configuration. If you are behind a load balancer, configure trusted proxies so that client IP is derived correctly; otherwise all users may appear as one IP and hit the per-IP limit.

### 18.2 Persistence Keys and Migration

Once you persist sessions, **signal keys become a backwards-compatibility contract**. If keys change between deployments, existing sessions will lose their state or fail to deserialize.

#### The Problem with Anonymous Keys

By default, signals created without explicit keys receive auto-generated keys based on creation order:

```go
// ❌ Bad: Anonymous signal, key based on creation order
var Cart = vango.NewSharedSignal([]CartItem{})  // Key: "signal_5"

// If you reorder code or add a new signal before this one,
// "signal_5" now refers to a DIFFERENT signal → data corruption
```

#### Stable Keys with `WithKey()`

Always use explicit keys for signals that will be persisted:

```go
// ✅ Good: Stable key won't break existing sessions
var Cart = vango.NewSharedSignal([]CartItem{},
	vango.WithKey("user_cart"))

var Preferences = vango.NewSharedSignal(UserPrefs{},
	vango.WithKey("user_prefs"))

var DraftPost = vango.NewSharedSignal[*Post](nil,
	vango.WithKey("draft_post"))
```

**Key naming conventions:**

- Use descriptive, domain-specific names: `"user_cart"`, `"draft_form"`, `"selected_filters"`
- Prefix with context if needed: `"checkout_shipping_address"`, `"editor_unsaved_changes"`
- Never use generic names like `"signal_1"` or `"temp"`

#### Migration Strategies

When you need to change the structure of persisted data:

**1. Additive changes (safe):**
New fields with zero values are safe—existing sessions simply have the new fields unset.

```go
// v1
type Prefs struct {
	Theme string
}

// v2 (safe: DarkMode defaults to false for existing sessions)
type Prefs struct {
	Theme    string
	DarkMode bool  // New field, zero value is safe
}
```

**2. Structural changes (requires migration key):**
If you change the type incompatibly, use a new key and handle migration:

```go
// Old: single address
var ShippingAddress = vango.NewSharedSignal(Address{},
	vango.WithKey("shipping_address"))

// New: multiple addresses (incompatible change)
var ShippingAddresses = vango.NewSharedSignal([]Address{},
	vango.WithKey("shipping_addresses_v2"))  // New key!

// In OnSessionStart/OnSessionResume, migrate old → new if needed
OnSessionStart: func(httpCtx context.Context, s *vango.Session) {
	// Check if old key exists and new doesn't
	if old, ok := s.Get("shipping_address").(Address); ok {
		if _, hasNew := s.Get("shipping_addresses_v2").([]Address); !hasNew {
			s.Set("shipping_addresses_v2", []Address{old})
			s.Delete("shipping_address")
		}
	}
},
```

**3. Key rename (requires careful rollout):**
If you must rename a key, do a two-phase rollout:

1. Deploy code that reads from both old and new keys, writes to new key
2. After all sessions have been touched, deploy code that only uses new key

#### Runtime-Only Keys (Auth State)

Remember that auth keys (`vango:auth:principal`, `vango:auth:expiry_unix_ms`) are **runtime-only** and excluded from persistence (see §16.8). Only application state signals need stable keys.

#### Checklist for Persistent Signals

- [ ] All session-scoped signals use `WithKey()` with stable, descriptive names
- [ ] Key names are documented (consider a `keys.go` constants file)
- [ ] Structural changes use new keys + migration logic
- [ ] Auth state is rehydrated, not persisted

### 18.3 Horizontal Scaling (Sticky Sessions vs Distributed Stores)

You have three broad deployment patterns:

1. **Single instance**
   simplest; in-memory sessions

2. **Multiple instances with sticky sessions**
   keep sessions in memory, but your load balancer must route the same tab consistently to the same instance (affinity). Useful early but complicates failover.

3. **Multiple instances with a distributed session store**
   sessions can resume across instances; better for rolling deploys and failover.

In all multi-instance patterns, be cautious with **global signals**: they represent shared, real-time state across users. In a multi-process deployment, global updates must be broadcast across instances (via a pub/sub layer) if you want consistent cross-instance behavior.

#### Global Signal Broadcasting (Multi-Instance)

When running multiple Vango instances, global signals only update sessions on the local instance by default. To broadcast global signal changes across all instances, configure a pub/sub backend:

```go
import "github.com/redis/go-redis/v9"

redisClient := redis.NewClient(&redis.Options{Addr: "localhost:6379"})

app := vango.New(vango.Config{
    // Session store for persistence across instances
    Session: vango.SessionConfig{
        Store: vango.RedisStore(redisClient),
    },

    // Global signal broadcast for cross-instance consistency
    GlobalSignalBroadcast: vango.RedisBroadcast(redisClient),
})
```

**How it works:**

1. When a global signal is updated on instance A, the change is published to Redis pub/sub
2. All other instances (B, C, ...) receive the publication and update their local copy
3. Each instance re-renders affected sessions and pushes patches to their connected clients

**Available broadcast backends:**

| Backend | Use Case |
|---------|----------|
| `vango.RedisBroadcast(client)` | Production; uses Redis pub/sub |
| `vango.NATSBroadcast(conn)` | High-throughput; uses NATS |
| `nil` (default) | Single instance only; no cross-instance sync |

**Manual pattern (if not using built-in broadcast):**

If you need custom broadcast logic or the built-in broadcast doesn't fit your infrastructure:

```go
// On the instance that updates the global signal:
store.OnlineCount.Set(newValue)
redisClient.Publish(ctx, "vango:global:OnlineCount", strconv.Itoa(newValue))

// On all instances, subscribe and update:
go func() {
    pubsub := redisClient.Subscribe(ctx, "vango:global:OnlineCount")
    for msg := range pubsub.Channel() {
        val, _ := strconv.Atoi(msg.Payload)
        store.OnlineCount.Set(val) // Updates local sessions
    }
}()
```

**When you DON'T need broadcast:**

- Single instance deployment
- Global signals that are read-only after initialization (e.g., feature flags loaded at startup)
- Global signals that are updated by only one instance (e.g., a single background worker)

### 18.4 Deployment (Docker / Reverse Proxy / WebSocket)

Vango is a normal `http.Handler`, but its runtime depends on:

* SSR HTTP requests
* WebSocket upgrades to `/_vango/*` endpoints
* long-lived WS connections

#### Reverse proxy requirements

WebSocket upgrades through reverse proxies are notoriously finicky. The following requirements must be met:

* must pass `Upgrade` and `Connection: upgrade` headers
* must set appropriate timeouts for long-lived WS (default timeouts will disconnect users)
* must forward `X-Forwarded-Proto` and real client IP
* must ensure `/_vango/*` routes reach the same upstream handler as the rest of the app

If you mount Vango at a subpath (e.g., `/app`), you must still route `/_vango/*` to the app handler (or configure the client to use a different WS base while preserving `?path=`).

#### Nginx Configuration (Copy-Paste Ready)

```nginx
upstream vango_app {
    server 127.0.0.1:8080;
    # For multiple instances with sticky sessions:
    # ip_hash;
    # server 127.0.0.1:8081;
    # server 127.0.0.1:8082;
}

server {
    listen 80;
    listen 443 ssl http2;
    server_name myapp.com;

    # SSL configuration (required for CookieSecure: true)
    ssl_certificate     /etc/ssl/certs/myapp.crt;
    ssl_certificate_key /etc/ssl/private/myapp.key;

    # Redirect HTTP to HTTPS
    if ($scheme = http) {
        return 301 https://$host$request_uri;
    }

    location / {
        proxy_pass http://vango_app;
        proxy_http_version 1.1;

        # WebSocket upgrade headers (CRITICAL)
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # Standard proxy headers
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # Long timeout for WebSocket connections (24 hours)
        # Without this, Nginx closes idle connections after 60s
        proxy_read_timeout 86400s;
        proxy_send_timeout 86400s;

        # Disable buffering for real-time updates
        proxy_buffering off;
    }

    # Static assets with aggressive caching (optional: serve directly from Nginx)
    location /static/ {
        alias /var/www/myapp/public/;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
```

**Key points:**

* `proxy_read_timeout 86400s` — Without this, Nginx closes WebSocket connections after 60 seconds of inactivity. Vango sessions are long-lived; set this high (24 hours shown).
* `proxy_set_header Upgrade` / `Connection "upgrade"` — These headers are **required** for WebSocket upgrade. Missing them is the #1 cause of "WebSocket won't connect" issues.
* `X-Forwarded-For` / `X-Forwarded-Proto` — Required for `TrustedProxies` to work correctly. Without these, your app can't determine the real client IP or protocol.

#### Caddy Configuration (Alternative)

Caddy handles WebSocket upgrades automatically:

```caddyfile
myapp.com {
    reverse_proxy localhost:8080
}
```

Caddy's defaults are production-ready for Vango with no additional configuration needed.

#### Asset caching and fingerprinting

In production, serve fingerprinted assets with immutable caching. Use `ctx.Asset("...")` everywhere you reference static files so the manifest-aware resolver can return hashed paths.

### 18.5 Storm Budgets (DoS and Runaway Work Protection)

Storm budgets are per-session limits on structured async work that prevent runaway effects, infinite loops, and DoS attacks from overwhelming the server.

#### Full Configuration

```go
import "time"

app := vango.New(vango.Config{
	Session: vango.SessionConfig{
		StormBudget: vango.StormBudgetConfig{
			// Per-second limits for async work starts
			MaxResourceStartsPerSecond: 100,  // NewResource / NewResourceKeyed
			MaxActionStartsPerSecond:   50,   // action.Run() calls
			MaxGoLatestStartsPerSecond: 100,  // GoLatest invocations

			// Per-tick limit for synchronous effect executions
			MaxEffectRunsPerTick: 50,

			// Time window for rate limiting calculations
			WindowDuration: time.Second,

			// What to do when a budget is exceeded
			OnExceeded: vango.BudgetThrottle,
		},
	},
})
```

#### Configuration Fields

| Field | Default | Description |
|-------|---------|-------------|
| `MaxResourceStartsPerSecond` | 100 | Max `NewResource`/`NewResourceKeyed` starts per second |
| `MaxActionStartsPerSecond` | 50 | Max `action.Run()` invocations per second |
| `MaxGoLatestStartsPerSecond` | 100 | Max `GoLatest` work starts per second |
| `MaxEffectRunsPerTick` | 50 | Max effect executions per reactive tick |
| `WindowDuration` | 1s | Sliding window for rate calculations |
| `OnExceeded` | `BudgetThrottle` | Behavior when limit is hit |

#### Exceeded Behaviors

| Mode | Behavior | When to Use |
|------|----------|-------------|
| `vango.BudgetThrottle` | Deny/delay new starts; surface error to caller; session continues | Most cases (default). Allows recovery. |
| `vango.BudgetTripBreaker` | Terminate/invalidate the session entirely | Critical resources; suspected abuse; unrecoverable loops |

#### Operational Guidance

**Why budgets exist:**
* A bug that creates an effect loop could spawn thousands of resources per second
* A malicious client could trigger rapid action calls to exhaust server resources
* Budgets reduce the blast radius of both accidental and intentional overload

**When budgets trigger:**
* `BudgetThrottle`: The specific operation fails with an error. The session remains valid. Monitor "budget exceeded" metrics to identify the problematic code path.
* `BudgetTripBreaker`: The session is terminated. The client receives a disconnect and must reconnect (starting a fresh session). Use this for resources you cannot afford to have abused.

**Design principle:**
Budgets are a **safety net**, not a primary design mechanism. Your app should still be designed to avoid self-triggering storms. If you're hitting budgets during normal operation, fix the underlying issue rather than raising limits.

**Recommended monitoring:**
Track budget exceeded counts by type (resource/action/effect/golatest) to catch bugs before they become incidents.

---

## 19. Application Configuration

This section documents the complete `vango.Config` surface and all sub-configuration structs. Understanding these options is essential for production deployments.

### 19.1 Core Configuration (`vango.Config`)

```go
import (
	"log/slog"
	"time"
	"github.com/vango-go/vango"
)

app := vango.New(vango.Config{
	// Development mode enables detailed errors, strict effect warnings,
	// and source-location transaction names. Set to false in production.
	DevMode: true,

	// Session configuration (detailed in §16.2)
	Session: vango.SessionConfig{...},

	// Static file serving (detailed in §16.3)
	Static: vango.StaticConfig{...},

	// Security settings (detailed in §16.4)
	Security: vango.SecurityConfig{...},

	// Logging and observability
	Logger: slog.Default(),

	// Asset manifest resolver for fingerprinted assets (production)
	AssetResolver: resolver,

	// Session lifecycle hooks (detailed in §13.1)
	OnSessionStart:  func(httpCtx context.Context, s *vango.Session) { ... },
	OnSessionResume: func(httpCtx context.Context, s *vango.Session) error { ... },
})
```

### 19.2 Session Configuration (`vango.SessionConfig`)

```go
vango.SessionConfig{
	// How long a disconnected session is kept in memory.
	// Allows refresh/reconnect to restore state within this window.
	ResumeWindow: 30 * time.Second,

	// Maximum detached sessions kept in memory (DoS protection).
	// When exceeded, oldest sessions are evicted per EvictionPolicy.
	MaxDetachedSessions: 10000,

	// Maximum concurrent sessions per client IP (DoS protection).
	// See the CRITICAL warning below about proxy configuration.
	MaxSessionsPerIP: 100,

	// When true, evict the oldest detached session when MaxSessionsPerIP
	// is reached instead of rejecting the new connection.
	EvictOnIPLimit: true,

	// Eviction policy when limits are reached.
	EvictionPolicy: vango.EvictionLRU,

	// Optional: Persist sessions for server restarts and non-sticky deploys.
	Store: vango.RedisStore(redisClient),
	// Or: Store: vango.PostgresStore(db),

	// Storm budgets per session (detailed in §16.5)
	StormBudget: vango.StormBudgetConfig{...},
}
```

> **CRITICAL: Proxy Configuration and MaxSessionsPerIP**
>
> If `TrustedProxies` (in `SecurityConfig`) is not configured correctly when running behind a load balancer (e.g., Nginx, AWS ALB, Cloudflare, or any L7 proxy), `RemoteAddr` will be the balancer's IP address rather than the real client IP.
>
> **This collapses all users into a single IP bucket and will trigger `MaxSessionsPerIP` immediately, effectively DoS-ing your own application.**
>
> Always configure `TrustedProxies` when behind a reverse proxy:
>
> ```go
> Security: vango.SecurityConfig{
>     TrustedProxies: []string{"10.0.0.0/8", "172.16.0.0/12"},
> },
> ```

**Session Durability Tiers:**

| Tier | What Survives | Configuration |
|------|---------------|---------------|
| **None** | Nothing (fresh session on every connect) | `ResumeWindow: 0` |
| **In-Memory** | Refresh/reconnect within ResumeWindow | `ResumeWindow: 30*time.Second` |
| **Persistent** | Server restarts, non-sticky deploys | `Store: vango.RedisStore(...)` |

### 19.3 Static File Configuration (`vango.StaticConfig`)

```go
vango.StaticConfig{
	// Directory containing static files (relative to working directory)
	Dir: "public",

	// URL prefix for static files (usually "/" for root)
	Prefix: "/",

	// Cache control strategy for production.
	// Fingerprinted files get immutable caching; others get short cache with revalidate.
	CacheControl: vango.CacheControlProduction,

	// Enable Brotli/gzip compression for text assets
	Compression: true,

	// Custom headers added to all static file responses
	Headers: map[string]string{
		"X-Content-Type-Options": "nosniff",
	},
}
```

**Cache Strategies:**

| Strategy | Behavior | When to Use |
|----------|----------|-------------|
| `CacheControlNone` | No caching headers | Development |
| `CacheControlProduction` | Fingerprinted: immutable (1 year); Others: short + revalidate | Production |
| `CacheControlCustom` | You control headers | Special requirements |

### 19.4 Security Configuration (`vango.SecurityConfig`)

```go
vango.SecurityConfig{
	// CSRF protection secret (CSRF is enabled when this is set).
	// Must be 32 bytes for HMAC-SHA256.
	CSRFSecret: []byte("your-32-byte-secret-key-here!!"),

	// Origin checking for WebSocket connections.
	// Explicit allowlist of permitted origins.
	AllowedOrigins: []string{
		"https://myapp.com",
		"https://www.myapp.com",
	},
	// Or allow same-origin automatically (simpler for single-domain apps):
	AllowSameOrigin: true,

	// Trusted reverse proxies for X-Forwarded-For / X-Forwarded-Proto.
	// CIDR notation or specific IPs. REQUIRED when behind load balancers.
	TrustedProxies: []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},

	// Cookie security defaults (applied to ctx.SetCookie / ctx.SetCookieStrict)
	CookieSecure:   true,               // Requires HTTPS; cookies dropped on HTTP
	CookieHttpOnly: true,               // No JavaScript access
	CookieSameSite: http.SameSiteLaxMode,
	CookieDomain:   "",                 // Empty = current domain only

	// External redirect allowlist (for OAuth/OIDC flows)
	AllowedRedirectHosts: []string{"accounts.google.com", "login.microsoftonline.com"},
}
```

**Cookie Security Notes:**

* When `CookieSecure` is `true` and a request is not over HTTPS, cookies are dropped.
* `ctx.SetCookieStrict` returns `vango.ErrSecureCookiesRequired` in this case.
* Use `vango.WithCookieHTTPOnly(false)` only for cookies that JS must read (rare).

#### XSS Prevention: HTML Escaping Contract

Vango escapes content **by default**. You don't need to manually escape user input in most cases:

```go
// ✅ SAFE: Text content is automatically escaped
Text(userInput)  // <script>alert('xss')</script> → &lt;script&gt;alert('xss')&lt;/script&gt;

// ✅ SAFE: Attributes are escaped
Href(userProvidedURL)  // Also validates URL scheme (rejects javascript:, data:, etc.)
Class(userClass)       // Escaped; can't break out of attribute

// ✅ SAFE: Format strings escape interpolated values
Textf("Hello, %s!", userName)
```

**Opt-in to raw HTML (dangerous):**

When you need to render raw HTML (e.g., from a CMS, markdown renderer, or sanitizer), use `DangerouslySetInnerHTML`:

```go
import . "github.com/vango-go/vango/el"

// ⚠️ DANGEROUS: Bypasses escaping - only use with sanitized input!
DangerouslySetInnerHTML(sanitizedHTML)

// Legacy alias (same function):
Raw(sanitizedHTML)
```

**Always sanitize user-provided HTML:**

```go
import "github.com/microcosm-cc/bluemonday"

// Create a policy once (reuse across requests)
var policy = bluemonday.UGCPolicy()

func SafeUserHTML(userHTML string) *vango.VNode {
    sanitized := policy.Sanitize(userHTML)
    return DangerouslySetInnerHTML(sanitized)
}
```

**Common sanitization policies:**

| Policy | Use Case |
|--------|----------|
| `bluemonday.StrictPolicy()` | Strip all HTML, text only |
| `bluemonday.UGCPolicy()` | User-generated content (comments, posts) |
| `bluemonday.NewPolicy()` | Build custom allowlist |

**Security Summary:**

| Content Type | Escaping | Notes |
|--------------|----------|-------|
| `Text(...)` | ✅ Auto-escaped | Safe for user input |
| `Textf(...)` | ✅ Auto-escaped | Safe for user input |
| Attributes | ✅ Auto-escaped | `Href` also validates URL scheme |
| `DangerouslySetInnerHTML(...)` | ❌ Not escaped | Must sanitize first |
| `Raw(...)` | ❌ Not escaped | Legacy alias; same warning |

**URL Scheme Validation:**

`Href()` validates URL schemes and rejects dangerous protocols:

```go
Href("javascript:alert(1)")  // Rejected - renders empty or safe fallback
Href("data:text/html,...")   // Rejected
Href("/safe/path")           // ✅ Allowed (relative)
Href("https://example.com")  // ✅ Allowed
```

#### Content Security Policy (CSP)

CSP is configured via HTTP middleware. Vango apps have specific requirements:

```go
func CSPMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Build CSP header
        csp := strings.Join([]string{
            "default-src 'self'",
            "script-src 'self'",                    // Own scripts only
            "style-src 'self' 'unsafe-inline'",     // Tailwind needs unsafe-inline
            "img-src 'self' data: https:",          // Images from self, data URIs, HTTPS
            "connect-src 'self' wss:",              // REQUIRED: WebSocket connections
            "font-src 'self'",
            "object-src 'none'",
            "base-uri 'self'",
            "frame-ancestors 'none'",
        }, "; ")

        w.Header().Set("Content-Security-Policy", csp)
        next.ServeHTTP(w, r)
    })
}

// Usage
mux := http.NewServeMux()
handler := CSPMiddleware(app.Handler())
```

**Critical for Vango:** The `connect-src` directive **must** include `wss:` (or the specific WebSocket origin) or the thin client cannot establish WebSocket connections.

**CSP Directive Reference for Vango:**

| Directive | Recommended Value | Notes |
|-----------|-------------------|-------|
| `default-src` | `'self'` | Fallback for unspecified directives |
| `script-src` | `'self'` | Only load own scripts; avoid `'unsafe-inline'` |
| `style-src` | `'self' 'unsafe-inline'` | Tailwind/inline styles need `'unsafe-inline'` |
| `connect-src` | `'self' wss:` | **Required** for WebSocket |
| `img-src` | `'self' data: https:` | Allow data URIs for inline images |
| `font-src` | `'self'` | Fonts from same origin |
| `object-src` | `'none'` | Block plugins (Flash, Java) |
| `frame-ancestors` | `'none'` | Prevent clickjacking (like X-Frame-Options) |

**If using nonces for scripts:**

```go
func CSPWithNonce(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Generate nonce per request
        nonce := generateSecureNonce() // crypto/rand, base64 encoded

        csp := fmt.Sprintf(
            "default-src 'self'; script-src 'self' 'nonce-%s'; style-src 'self' 'unsafe-inline'; connect-src 'self' wss:",
            nonce,
        )
        w.Header().Set("Content-Security-Policy", csp)

        // Pass nonce to Vango via context for use in templates
        ctx := context.WithValue(r.Context(), "csp-nonce", nonce)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

**CSP Reporting (optional):**

```go
// Add report-uri or report-to for CSP violation reports
csp += "; report-uri /csp-report"
```

### 19.5 Storm Budget Configuration (`vango.StormBudgetConfig`)

Storm budgets protect against runaway effects, infinite loops, and DoS attacks by limiting the rate of async work per session.

```go
vango.StormBudgetConfig{
	// Per-second limits for async work starts
	MaxResourceStartsPerSecond: 100,
	MaxActionStartsPerSecond:   50,
	MaxGoLatestStartsPerSecond: 100,

	// Per-tick limit for effect executions
	MaxEffectRunsPerTick: 50,

	// Time window for rate limiting calculations
	WindowDuration: time.Second,

	// What to do when a budget is exceeded
	OnExceeded: vango.BudgetThrottle,
}
```

**Exceeded Behaviors:**

| Mode | Behavior | When to Use |
|------|----------|-------------|
| `BudgetThrottle` | Deny/delay new starts; surface error to caller | Most cases (default) |
| `BudgetTripBreaker` | Terminate/invalidate the session entirely | Critical resources; severe abuse |

**Operational Interpretation:**

* Storm budgets reduce blast radius if you accidentally create an effect loop or a resource that restarts repeatedly.
* They are a **safety net**, not a primary design mechanism. Your app should still be designed to avoid self-triggering storms.
* Monitor "budget exceeded" metrics to catch bugs before they become incidents.

### 19.6 Limits Configuration (`vango.LimitsConfig`)

Request-level limits protect against oversized payloads:

```go
vango.LimitsConfig{
	MaxEventPayloadBytes: 64 * 1024,       // 64KB per event
	MaxFormBytes:         10 * 1024 * 1024, // 10MB for form uploads
	MaxQueryParams:       50,
	MaxHeaderBytes:       8 * 1024,         // 8KB headers
}
```

### 19.7 API Configuration (`vango.APIConfig`)

For typed API endpoints:

```go
vango.APIConfig{
	MaxBodyBytes: 1 * 1024 * 1024, // 1MB default for JSON bodies
}
```

### 19.8 Environment-Based Configuration Pattern

Structure configuration around deployment environments:

```go
// internal/config/config.go
package config

import (
	"net/http"
	"os"
	"time"

	"github.com/vango-go/vango"
)

type Config struct {
	Environment   string
	Port          string
	DatabaseURL   string
	RedisURL      string
	SessionSecret string
	IsDevelopment bool
	IsProduction  bool
}

func Load() Config {
	env := getEnv("ENVIRONMENT", "development")
	return Config{
		Environment:   env,
		Port:          getEnv("PORT", "8080"),
		DatabaseURL:   mustGetEnv("DATABASE_URL"),
		RedisURL:      getEnv("REDIS_URL", ""),
		SessionSecret: mustGetEnv("SESSION_SECRET"),
		IsDevelopment: env == "development",
		IsProduction:  env == "production",
	}
}

func (c Config) VangoConfig(redisClient *redis.Client) vango.Config {
	cfg := vango.Config{
		DevMode: c.IsDevelopment,
		Session: vango.SessionConfig{
			ResumeWindow:        30 * time.Second,
			MaxDetachedSessions: 10000,
			MaxSessionsPerIP:    100,
		},
		Static: vango.StaticConfig{
			Dir:    "public",
			Prefix: "/",
		},
	}

	if c.IsProduction {
		cfg.Session.Store = vango.RedisStore(redisClient)
		cfg.Security = vango.SecurityConfig{
			CSRFSecret:     []byte(c.SessionSecret),
			AllowedOrigins: []string{"https://myapp.com"},
			TrustedProxies: []string{"10.0.0.0/8"},
			CookieSecure:   true,
			CookieHttpOnly: true,
			CookieSameSite: http.SameSiteLaxMode,
		}
		cfg.Static.CacheControl = vango.CacheControlProduction
		cfg.Static.Compression = true
	}

	return cfg
}
```

### 19.9 Configuration Categories (Quick Reference)

This table summarizes which configuration settings affect correctness, performance, and security:

| Category | Settings | Correctness | Performance | Security |
|----------|----------|:-----------:|:-----------:|:--------:|
| **Session** | `ResumeWindow`, `Store` | ✓ | ✓ | |
| **Limits** | `MaxDetachedSessions`, `MaxSessionsPerIP` | | ✓ | ✓ |
| **Storm Budgets** | `MaxResource/Action/Effect*` limits | | ✓ | ✓ |
| **CSRF** | `CSRFSecret`, Cookie flags | | | ✓ |
| **Origins** | `AllowedOrigins`, `AllowSameOrigin` | | | ✓ |
| **Cookies** | `Secure`, `HttpOnly`, `SameSite` | | | ✓ |
| **Static** | `CacheControl`, `Compression` | | ✓ | |
| **DevMode** | Error detail, effect warnings | ✓ | | |

**Reading the table:**
- **Correctness:** Misconfiguration can cause functional bugs or unexpected behavior
- **Performance:** Affects latency, memory usage, or bandwidth
- **Security:** Affects protection against attacks or data exposure

### 19.10 Integrating with Existing Routers

Vango is a standard `http.Handler`, so it integrates with any Go router:

**With Chi:**

```go
import "github.com/go-chi/chi/v5"
import "github.com/go-chi/chi/v5/middleware"

r := chi.NewRouter()
r.Use(middleware.Logger)
r.Use(middleware.Recoverer)

// Mount Vango at root
r.Mount("/", app)

// Add non-Vango routes
r.Get("/api/external", externalAPIHandler)
```

**With Gorilla Mux:**

```go
import "github.com/gorilla/mux"

r := mux.NewRouter()
r.PathPrefix("/").Handler(app)
```

**With standard library:**

```go
mux := http.NewServeMux()
mux.Handle("/", app)
```

> **CRITICAL: Sub-path Mounting and WebSocket Endpoints**
>
> Vango's WebSocket endpoint lives at `/_vango/*`. If you mount the app at a sub-path (e.g., `/app`), you **MUST** also route `/_vango/*` to the same handler:
>
> ```go
> // Mounting Vango at a sub-path requires dual mount
> r.Mount("/app", app)
> r.Mount("/_vango", app)  // Required for WebSocket connectivity
> ```
>
> Alternatively, configure the thin client to use a different WebSocket base URL (advanced; not recommended for most cases).

---

## Production Readiness Checklist (Integrated)

Before shipping a Vango app to production, verify each item with the concrete configuration shown.

### Security / Auth

```go
Security: vango.SecurityConfig{
	// ✅ CSRF protection (32-byte secret)
	CSRFSecret: []byte("your-32-byte-secret-key-here!!"),

	// ✅ Explicit WebSocket origin allowlist
	AllowedOrigins: []string{"https://myapp.com", "https://www.myapp.com"},

	// ✅ Secure cookie defaults
	CookieSecure:   true,
	CookieHttpOnly: true,
	CookieSameSite: http.SameSiteLaxMode,

	// ✅ Trusted proxies (REQUIRED if behind load balancer)
	TrustedProxies: []string{"10.0.0.0/8"},
},

// ✅ Revalidate identity on session resume
OnSessionResume: func(httpCtx context.Context, s *vango.Session) error {
	user, err := auth.ValidateFromRequest(httpCtx)
	if err != nil {
		return err // Rejects resume, forces new session
	}
	s.Set("user", user)
	return nil
},
```

- [ ] CSRF protection enabled (`CSRFSecret` set, 32 bytes)
- [ ] WebSocket origins explicitly allowlisted (`AllowedOrigins` or `AllowSameOrigin`)
- [ ] Cookie settings: `Secure=true`, `HttpOnly=true`, `SameSite=Lax`
- [ ] `TrustedProxies` configured if behind any reverse proxy/load balancer
- [ ] `OnSessionResume` revalidates credentials for authenticated apps
- [ ] Route guards protect sensitive routes; actions enforce authorization server-side

### Operational Limits

```go
Session: vango.SessionConfig{
	// ✅ Resume window for refresh/reconnect
	ResumeWindow: 30 * time.Second,

	// ✅ Memory protection
	MaxDetachedSessions: 10000,
	MaxSessionsPerIP:    100,
	EvictOnIPLimit:      true,

	// ✅ Session persistence for restarts/scaling
	Store: vango.RedisStore(redisClient),

	// ✅ Storm budgets
	StormBudget: vango.StormBudgetConfig{
		MaxResourceStartsPerSecond: 100,
		MaxActionStartsPerSecond:   50,
		MaxGoLatestStartsPerSecond: 100,
		MaxEffectRunsPerTick:       50,
		WindowDuration:             time.Second,
		OnExceeded:                 vango.BudgetThrottle,
	},
},
```

- [ ] `ResumeWindow` set (typically 30s; 0 to disable resume)
- [ ] `MaxDetachedSessions` set (10000 is a reasonable default)
- [ ] `MaxSessionsPerIP` set (100 is reasonable; adjust for multi-tenant)
- [ ] **`TrustedProxies` configured** (otherwise `MaxSessionsPerIP` will DoS your own users)
- [ ] Storm budgets configured with appropriate limits
- [ ] Session store configured for production (Redis/Postgres) if you need restart resilience

### Static Assets & Caching

```go
Static: vango.StaticConfig{
	Dir:          "public",
	Prefix:       "/",
	CacheControl: vango.CacheControlProduction,
	Compression:  true,
},

// Use ctx.Asset() in layouts for fingerprinted paths
LinkEl(Rel("stylesheet"), Href(ctx.Asset("styles.css")))
```

- [ ] `CacheControl: vango.CacheControlProduction` for immutable caching
- [ ] `Compression: true` for Brotli/gzip
- [ ] All asset references use `ctx.Asset(...)` for manifest-aware paths
- [ ] Asset manifest generated during build (`vango build`)

### Deployment & Infrastructure

- [ ] Reverse proxy passes `Upgrade` and `Connection: upgrade` headers for WebSocket
- [ ] `/_vango/*` routes reach the Vango handler (critical for WS)
- [ ] WebSocket idle timeouts set high enough (sessions are long-lived)
- [ ] `X-Forwarded-For` and `X-Forwarded-Proto` headers forwarded
- [ ] HTTP server timeouts configured (`ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`)

Example Nginx configuration:

```nginx
location / {
    proxy_pass http://app:8080;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_read_timeout 86400s;  # Long timeout for WebSocket
}
```

### Observability

```go
vango.Config{
	Logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})),
}
```

- [ ] Structured logging enabled (`slog` with JSON handler for production)
- [ ] Metrics exported for: active sessions, detached sessions, event rate, patch sizes, budget exceeded counts
- [ ] Trace propagation via `ctx.StdContext()` through all service/database calls

### Complete Production Configuration Example

```go
import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/vango-go/vango"
)

func ProductionConfig(redisClient *redis.Client, csrfSecret []byte) vango.Config {
	return vango.Config{
		DevMode: false,

		Logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})),

		Session: vango.SessionConfig{
			ResumeWindow:        30 * time.Second,
			MaxDetachedSessions: 10000,
			MaxSessionsPerIP:    100,
			EvictOnIPLimit:      true,
			Store:               vango.RedisStore(redisClient),
			StormBudget: vango.StormBudgetConfig{
				MaxResourceStartsPerSecond: 100,
				MaxActionStartsPerSecond:   50,
				MaxGoLatestStartsPerSecond: 100,
				MaxEffectRunsPerTick:       50,
				WindowDuration:             time.Second,
				OnExceeded:                 vango.BudgetThrottle,
			},
		},

		Security: vango.SecurityConfig{
			CSRFSecret:     csrfSecret,
			AllowedOrigins: []string{"https://myapp.com"},
			TrustedProxies: []string{"10.0.0.0/8", "172.16.0.0/12"},
			CookieSecure:   true,
			CookieHttpOnly: true,
			CookieSameSite: http.SameSiteLaxMode,
		},

		Static: vango.StaticConfig{
			Dir:          "public",
			Prefix:       "/",
			CacheControl: vango.CacheControlProduction,
			Compression:  true,
		},

		OnSessionStart: func(httpCtx context.Context, s *vango.Session) {
			if user, ok := vango.UserFromContext(httpCtx).(*User); ok && user != nil {
				s.Set("user", user)
			}
		},

		OnSessionResume: func(httpCtx context.Context, s *vango.Session) error {
			user, err := auth.ValidateFromRequest(httpCtx)
			if err != nil {
				return err
			}
			s.Set("user", user)
			return nil
		},
	}
}
```

---
# Appendices

These appendices provide (A) a practical API cheatsheet aligned to the current guide, (B) common pitfalls with concrete fixes, and (C) a migration guide for teams coming from SPAs.

All examples assume:

```go
import "github.com/vango-go/vango"
import . "github.com/vango-go/vango/el"
```

(Plus standard library imports as needed.)

---

## A. API Cheatsheet

This is a “working set” of the APIs and patterns a developer/LLM uses most when building apps.

### A.1 Core Rendering and Context

**Reactive component wrapper**

```go
func MyComponent() vango.Component {
	return vango.Func(func() *vango.VNode {
		ctx := vango.UseCtx()
		_ = ctx
		return Div(Text("hi"))
	})
}
```

**Context**

* `vango.UseCtx()` returns the current `vango.Ctx` within a render function.
* Use `ctx.StdContext()` for I/O cancellation in Resource/Action work.
* Do not depend on `ctx.Request()` during WebSocket renders for headers/cookies; use the auth context bridge into session state.

### A.2 Signals

```go
s := vango.NewSignal(initial) // component-scoped
s.Get()   // tracked read
s.Set(v)  // write (must be on session loop)
s.Peek()  // untracked read
```

**Session-scoped and global-scoped**

```go
var Shared = vango.NewSharedSignal(initial) // per browser tab/session
var Global = vango.NewGlobalSignal(initial) // shared across users
```

**Common helpers shown in the current guide**

Slice helpers (copy-on-write):

```go
items.RemoveAt(i)
items.RemoveWhere(func(x T) bool { ... })
items.UpdateAt(i, func(x T) T { ... })
items.Filter(func(x T) bool { ... })
```

Map helpers (copy-on-write):

```go
m.SetEntry(k, v)
m.DeleteEntry(k)
m.UpdateEntry(k, func(v V) V { ... })
```

### A.3 Memos

```go
m := vango.NewMemo(func() T {
	return computeFromSignals()
})
m.Get()
```

Session-scoped derived values (pattern used in current guide):

```go
var IsAuthed = vango.NewSharedMemo(func() bool { ... })
```

### A.4 Effects and Cleanup

Effects must return `vango.Cleanup` (or `nil`):

```go
vango.Effect(func() vango.Cleanup {
	// setup...
	return nil
})
```

Structured helpers used inside effects (return Cleanup):

```go
vango.Interval(d, fn, opts...)     // periodic work tied to component lifetime
vango.Subscribe(stream, fn, opts...) // subscription tied to lifetime
vango.GoLatest(key, work, apply, opts...) // cancel stale work by key
```

### A.5 Transactions

Batch multiple writes into one commit:

```go
vango.Tx(func() {
	a.Set(1)
	b.Set(2)
})

vango.TxNamed("domain:action", func() {
	// same, but labeled for observability
})
```

### A.6 Resources (Async reads)

Create:

```go
r := vango.NewResource(func() (T, error) { ... })
```

Keyed:

```go
r := vango.NewResourceKeyed(key, func(k K) (T, error) { ... })
// key can be a signal-like (signal, URLParam, memo) or a function returning K
```

States:

```go
r.State() == vango.Pending
r.State() == vango.Loading
r.State() == vango.Ready
r.State() == vango.Error

r.Data()  // valid when Ready
r.Error() // valid when Error
```

Matchers (recommended):

```go
return r.Match(
	vango.OnPending(func() *vango.VNode { ... }),
	vango.OnLoading(func() *vango.VNode { ... }),
	vango.OnReady(func(data T) *vango.VNode { ... }),
	vango.OnError(func(err error) *vango.VNode { ... }),
)
```

### A.7 Actions (Async mutations)

Create:

```go
a := vango.NewAction(
	func(ctx context.Context, arg A) (R, error) { ... },
	vango.CancelLatest(), // default
)
accepted := a.Run(arg) // bool
```

States:

```go
a.State() == vango.ActionIdle
a.State() == vango.ActionRunning
a.State() == vango.ActionSuccess
a.State() == vango.ActionError

res, ok := a.Result()
err := a.Error()
a.Reset()
```

Concurrency policies (as described in the current guide):

```go
vango.CancelLatest()
vango.DropWhileRunning()
vango.Queue(n)
```

### A.8 Navigation and URL State

Link helpers (intercepted when WS healthy; normal navigation otherwise):

```go
Link("/path", Text("Label"))
LinkPrefetch("/path", Text("Label"))
NavLink(ctx, "/path", Text("Label"))
NavLinkPrefix(ctx, "/admin", Text("Admin"))
```

Programmatic navigation:

```go
ctx := vango.UseCtx()
ctx.Navigate("/path")
ctx.Navigate("/path", vango.WithReplace()) // replace vs push
```

**Canonical API:** Navigation options use the `vango.With*` prefix (e.g., `vango.WithReplace()`, `vango.WithState(data)`). All navigation helpers are in the `vango` package, not a separate `server` package.

URL params (query-string state):

```go
p := vango.URLParam("status", "all")
q := vango.URLParam("q", "", vango.Replace) // update mode per current guide patterns
debounced := vango.URLParam("q", "", vango.Replace, vango.URLDebounce(300*time.Millisecond))
```

### A.9 Assets and Thin Client

Manifest-aware asset resolution:

```go
Href(ctx.Asset("styles.css"))
Img(Src(ctx.Asset("images/logo.svg")))
```

Thin client injection:

```go
Body(children, VangoScripts())
```

WebSocket URL contract (operationally critical):

* thin client connects to `/_vango/live?path=<current-path-and-query>`
* custom WS URL overrides must preserve `?path=` or you risk SSR/WS tree mismatch (“handler not found”)

---

## B. Common Pitfalls

This section lists the failure modes you’ll actually see in development and production, with the most likely causes and the correct fix.

### B.1 “Hook order changed” panic

**Symptom:** runtime panic indicating hook order changed.

**Cause:** conditional or variable-count hook creation (`NewSignal`, `NewMemo`, `Effect`, `NewResource`, `NewAction`, etc.) inside a render function.

**Fix:** always call hooks in the same order/count; conditionally *use* results, not create them.

Bad:

```go
if show {
	_ = vango.NewSignal(0)
}
```

Good:

```go
s := vango.NewSignal(0)
if show {
	_ = s
}
```

Bad (hooks in a loop where length changes):

```go
for range items.Get() {
	_ = vango.NewSignal(0)
}
```

Good: store collection in a single signal and render with `Range(...)` + `Key(...)`.

### B.2 “Signal write outside session loop”

**Symptom:** panic or undefined behavior when writing a signal.

**Cause:** writing signals from a goroutine (or any context not running on the session loop).

**Fix:** marshal back via `ctx.Dispatch(func(){ ... })`, or use `Resource`/`Action`/structured helpers which apply results on the loop.

Bad:

```go
go func() { s.Set(1) }()
```

Good:

```go
ctx := vango.UseCtx()
go func() {
	ctx.Dispatch(func() { s.Set(1) })
}()
```

### B.3 Blocking I/O inside render (stalls session / bad latency)

**Symptom:** slow UI, “everything freezes,” sessions back up.

**Cause:** database/HTTP calls executed directly inside page handlers or `vango.Func` render bodies.

**Fix:** move I/O into `Resource` or `Action`. Page handlers are reactive and must remain pure.

### B.4 “Events received, patches = 0” (UI doesn’t update)

**Most common causes in this guide’s model:**

1. signals are being created or read outside the tracked render context
2. VNodes that close over signals are cached and reused (so reactivity doesn’t “see” the reads)

**Fixes:**

* create signals inside `vango.Func` (or reactive page handlers) and read them via `.Get()` during render
* do not cache reactive VNodes at module scope

Bad:

```go
var node = Div(Text(count.Get())) // cached VNode closes over reactive state
```

Good:

```go
return Div(Textf("%d", count.Get()))
```

### B.5 “Handler not found” / SSR vs WS mismatch

**Symptom:** server logs or client console indicate handler missing; may self-heal reload.

**Primary cause:** SSR-rendered tree and WS-mounted tree differ, causing hydration ID (HID) drift.

**Key operational requirement:** the thin client must connect with `?path=<current-path>` so the server mounts the same route on WS init.

**Fix direction:**

* ensure `/_vango/live?path=...` is preserved
* ensure route rendering is deterministic for the same inputs (avoid nondeterminism in render)
* ensure nested components are expanded consistently before HID assignment (handled by framework, but avoid patterns that change tree shape between SSR and WS)

### B.6 Patch mismatch reloads

**Symptom:** page reloads unexpectedly; “patch mismatch” observed.

**Likely causes:**

* unstable list keys (index keys; keys change on reorder)
* third-party scripts mutating Vango-managed DOM
* mis-specified island/hook boundaries

**Fix:**

* use stable `Key(domainID)` for dynamic lists
* confine DOM-owning libraries to islands
* do not mutate structure of Vango-managed DOM from custom JS except via hook protocols intended for that purpose

### B.7 Route param surprises

**Symptom:** unexpected handler invoked; “abc” reaches an int handler.

**Fix:** use typed params in filenames (e.g., `[id:int]`) so invalid URLs 404 at routing.

### B.8 Mounting under a subpath breaks WS

**Symptom:** app works for SSR but interactivity fails.

**Cause:** Vango WebSocket endpoints live under `/_vango/*`. If you mount the app under `/app`, you must still route `/_vango/*` to the same handler (or override WS URL while preserving `?path=` semantics).

### B.9 Incident Runbooks (When It's On Fire)

Quick action checklists for production incidents:

#### Sessions Spiking

**Symptoms:** `sessions_active` metric climbing fast, memory pressure, degraded response times.

**Triage:**
1. Check if it's real traffic (bot attack vs viral feature)
2. Verify `MaxSessionsPerIP` is configured and `TrustedProxies` is set correctly
3. Check for session leaks (sessions not cleaning up after tab close)
4. Check detached session count—if high, `EvictOnIPLimit` may not be enabled

**Mitigation:**
- Scale horizontally if legitimate load
- Lower `ResumeWindow` temporarily to reduce detached session accumulation
- Enable `EvictOnIPLimit: true` if not already set
- If attack: block IPs at load balancer, tighten `MaxSessionsPerIP`

#### Patch Mismatch Storms

**Symptoms:** clients repeatedly reload, "patch mismatch" in logs, high SSR load.

**Triage:**
1. Check recent deploys—did render logic change?
2. Look for unstable keys in list rendering (index keys, keys that change on reorder)
3. Check for third-party scripts mutating Vango-managed DOM
4. Verify island boundaries are correct (libraries owning DOM outside islands)

**Mitigation:**
- Rollback recent deploy if it introduced the issue
- Add stable `Key(domainID)` to list items
- Move DOM-mutating libraries into islands
- Check browser extensions aren't interfering (hard to fix, but good to know)

#### CPU Spikes on Effects / Effect Storms

**Symptoms:** CPU pegged, storm budget exceeded logs, degraded latency.

**Triage:**
1. Check storm budget metrics—which type is spiking? (resource/action/effect/golatest)
2. Look for infinite effect loops (effect writes signal it depends on without guard)
3. Profile with `pprof` to identify hot paths
4. Review recent effect changes

**Mitigation:**
- Storm budgets should already be limiting blast radius (verify they're configured)
- If a specific component is the culprit, disable it or add guards
- Consider `vango.BudgetTripBreaker` for sessions that hit budgets repeatedly (terminates session)

#### "Handler not found" Epidemic

**Symptoms:** many clients showing "handler not found" errors, interactivity broken.

**Triage:**
1. Check if SSR and WS are running same code version (rolling deploy mismatch)
2. Verify `?path=` is preserved on WebSocket URL
3. Check for nondeterministic render (random IDs, time-based conditionals)

**Mitigation:**
- Ensure atomic deploys (SSR and WS updated together)
- Force page refresh for affected users (they'll get fresh SSR + matching WS)
- Fix nondeterministic render if identified

#### Memory Leak (Gradual)

**Symptoms:** memory grows over hours/days, eventually OOM.

**Triage:**
1. Check session count growth vs request rate
2. Look for Resource/Action instances not being cleaned up
3. Profile heap with `pprof`
4. Check for goroutine leaks (effects not returning cleanup, subscriptions not cancelled)

**Mitigation:**
- Ensure all `Effect` functions return cleanup when needed
- Use structured helpers (`GoLatest`, `Subscribe`, `Interval`) which handle cleanup
- Configure session eviction limits
- Restart pods as temporary mitigation while debugging

#### Observability Checklist for Incidents

When investigating any incident, gather:

```
□ sessions_active / sessions_detached metrics
□ events_total by type (especially error rate)
□ patch_size_bytes p99 (large patches = slow updates)
□ storm_budget_exceeded_total by type
□ resource_duration / action_duration p99
□ Recent deploy history
□ Sample of error logs (grep for "mismatch", "handler not found", "budget exceeded")
```

---

## C. Migration Guide (React/Next.js → Vango)

This section is written for teams migrating from a client-side SPA mental model to Vango’s server-driven model.

### C.1 Mental Model Translation

**React SPA:**

* client owns state
* server is data API
* mutations require API + client cache invalidation

**Vango:**

* server owns state
* browser is event/patch terminal
* “cache invalidation” becomes updating signals; UI follows automatically

Mapping:

| SPA Concept                | Vango Equivalent                          |
| -------------------------- | ----------------------------------------- |
| React component with hooks | `vango.Func` component                    |
| `useState`                 | `vango.NewSignal`                         |
| derived selectors          | `vango.NewMemo`                           |
| `useEffect`                | `vango.Effect` (returns cleanup)          |
| React Query / SWR          | `vango.NewResource` / `NewResourceKeyed`  |
| mutation hooks             | `vango.NewAction` with concurrency policy |
| client router navigation   | `Link` helpers / `ctx.Navigate`           |
| URL query state            | `vango.URLParam`                          |

### C.2 Data Fetching Migration

In React, you fetch via API calls in the browser. In Vango, you usually call services/DB directly server-side and wrap with `Resource`.

React:

```js
const { data, isLoading } = useQuery(['project', id], () => fetch(`/api/projects/${id}`))
```

Vango:

```go
project := vango.NewResource(func() (*Project, error) {
	return services.Projects.GetByID(ctx.StdContext(), id)
})
```

### C.3 Mutation Migration

React typically does:

* optimistic UI
* POST to API
* reconcile cache

Vango does:

* optimistic signal update (optional)
* `Action` for I/O
* update signals (or navigate) on success

Choose concurrency policy deliberately:

* save button: `DropWhileRunning()`
* search: `CancelLatest()`
* queued operations: `Queue(n)`

### C.4 Routing Migration

In Next.js, you have file-based routing with client navigation. In Vango you also have file-based routing, but execution is server-side and reactive:

* page handlers must remain render-pure
* layouts replace the typical SPA “app shell”
* navigation is progressively enhanced: Link works without JS, becomes SPA when WS is healthy

### C.5 Incremental Adoption Patterns

If you have an existing Go backend or SPA:

* mount Vango as a subtree (e.g., `/new/*`) while keeping old routes
* embed legacy React widgets as islands temporarily
* share auth by attaching identity in HTTP middleware and bridging it into session start/resume

---

## D. `vango.json` Reference

The `vango.json` file is the contract between your project and the Vango CLI. It controls route generation, static file handling, build output, Tailwind integration, and development tooling.

### D.1 Complete Schema

```json
{
    "module": "myapp",
    "routes": {
        "dir": "app/routes",
        "output": "app/routes/routes_gen.go",
        "package": "routes"
    },
    "static": {
        "dir": "public",
        "prefix": "/"
    },
    "build": {
        "output": "dist",
        "fingerprint": true,
        "manifest": "manifest.json"
    },
    "tailwind": {
        "enabled": true,
        "config": "tailwind.config.js",
        "input": "app/styles/input.css",
        "output": "public/styles.css"
    },
    "devtools": {
        "enabled": true
    }
}
```

### D.2 Field Reference

#### `module` (required)

```json
"module": "myapp"
```

Your Go module path (must match `go.mod`). Used by the route generator to produce correct import paths in `routes_gen.go`.

#### `routes` (required)

```json
"routes": {
    "dir": "app/routes",
    "output": "app/routes/routes_gen.go",
    "package": "routes"
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `dir` | string | `"app/routes"` | Directory containing route files (page handlers, layouts, middleware) |
| `output` | string | `"app/routes/routes_gen.go"` | Path for generated route registration code |
| `package` | string | `"routes"` | Go package name for the generated file |

**Behavior:**
- `vango gen routes` scans `dir` for `*.go` files matching route conventions
- Generates `output` with `RegisterRoutes(app *vango.App)` function
- The generated file should be committed to version control

#### `static` (required)

```json
"static": {
    "dir": "public",
    "prefix": "/"
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `dir` | string | `"public"` | Directory containing static assets |
| `prefix` | string | `"/"` | URL prefix for static file serving |

**Behavior:**
- Files in `dir` are served at `prefix + filename`
- Example: `public/images/logo.png` → `/images/logo.png`
- In production with fingerprinting, use `ctx.Asset(...)` to resolve hashed paths

#### `build` (required for production)

```json
"build": {
    "output": "dist",
    "fingerprint": true,
    "manifest": "manifest.json"
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `output` | string | `"dist"` | Directory for build artifacts |
| `fingerprint` | boolean | `true` | Whether to hash static assets for cache busting |
| `manifest` | string | `"manifest.json"` | Filename for asset manifest (maps original → fingerprinted names) |

**Build output structure:**

```
dist/
├── server                  # Compiled Go binary
├── public/                 # Static assets (fingerprinted if enabled)
│   ├── styles.a1b2c3d4.css
│   ├── images/
│   │   └── logo.e5f6g7h8.png
│   └── ...
└── manifest.json           # {"styles.css": "styles.a1b2c3d4.css", ...}
```

#### `tailwind` (optional)

```json
"tailwind": {
    "enabled": true,
    "config": "tailwind.config.js",
    "input": "app/styles/input.css",
    "output": "public/styles.css"
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | boolean | `true` | Whether to run Tailwind CSS pipeline |
| `config` | string | `"tailwind.config.js"` | Path to Tailwind config (optional in Tailwind v4) |
| `input` | string | `"app/styles/input.css"` | Tailwind source file (contains `@import "tailwindcss"`) |
| `output` | string | `"public/styles.css"` | Compiled CSS output path |

**Behavior:**
- When `enabled: true`, `vango dev` starts Tailwind watcher alongside the Go compiler
- Tailwind scans `*.go` files for `Class(...)` calls to extract class names
- Uses standalone Tailwind binary (no Node.js required)
- Set `enabled: false` if using pure CSS or a different CSS pipeline

**Disabling Tailwind:**

```json
"tailwind": {
    "enabled": false
}
```

#### `devtools` (optional)

```json
"devtools": {
    "enabled": true
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | boolean | `true` in dev, `false` in prod | Enable browser devtools integration |

**Behavior:**
- When enabled, the thin client exposes debugging hooks for browser extensions
- Transaction names include source file:line references
- Automatically disabled when `DevMode: false` in `vango.Config`

### D.3 CLI ↔ Config Resolution Order

The CLI resolves configuration in this order (later sources override earlier):

1. **Built-in defaults** (hardcoded in CLI)
2. **`vango.json`** in project root
3. **Environment variables** (prefixed with `VANGO_`)
4. **Command-line flags** (highest priority)

**Environment variable mapping:**

| Config Field | Environment Variable |
|--------------|---------------------|
| `routes.dir` | `VANGO_ROUTES_DIR` |
| `static.dir` | `VANGO_STATIC_DIR` |
| `build.output` | `VANGO_BUILD_OUTPUT` |
| `tailwind.enabled` | `VANGO_TAILWIND_ENABLED` |

**Example: Override for CI:**

```bash
# Use different build output in CI
VANGO_BUILD_OUTPUT=./artifacts vango build
```

### D.4 Minimal vs Full Examples

**Minimal `vango.json` (using all defaults):**

```json
{
    "module": "myapp"
}
```

This works because the CLI applies sensible defaults for all other fields.

**Full `vango.json` (explicit everything):**

```json
{
    "module": "github.com/myorg/myapp",
    "routes": {
        "dir": "app/routes",
        "output": "app/routes/routes_gen.go",
        "package": "routes"
    },
    "static": {
        "dir": "public",
        "prefix": "/"
    },
    "build": {
        "output": "dist",
        "fingerprint": true,
        "manifest": "manifest.json"
    },
    "tailwind": {
        "enabled": true,
        "config": "tailwind.config.js",
        "input": "app/styles/input.css",
        "output": "public/styles.css"
    },
    "devtools": {
        "enabled": true
    }
}
```

**No Tailwind (pure CSS):**

```json
{
    "module": "myapp",
    "tailwind": {
        "enabled": false
    }
}
```

**Custom directory structure:**

```json
{
    "module": "myapp",
    "routes": {
        "dir": "src/pages",
        "output": "src/pages/generated.go",
        "package": "pages"
    },
    "static": {
        "dir": "assets",
        "prefix": "/static/"
    }
}
```

### D.5 Validation and Errors

The CLI validates `vango.json` on startup. Common errors:

| Error | Cause | Fix |
|-------|-------|-----|
| `module is required` | Missing `module` field | Add `"module": "yourmodule"` matching `go.mod` |
| `routes.dir does not exist` | Invalid routes directory | Create the directory or fix the path |
| `tailwind.input not found` | Missing Tailwind source file | Create the file or set `tailwind.enabled: false` |
| `module mismatch with go.mod` | `module` doesn't match `go.mod` | Ensure they match exactly |

**Tip:** Run `vango validate` to check your configuration without starting the server.
