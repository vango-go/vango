---
title: "Vango Architecture & Guide"
slug: vango-architecture-guide
version: 1.0.0
status: Pre-Launch (Normative)
---

# Vango Architecture & Guide

> **The Go Framework for Modern Web Applications**

---

## Table of Contents

1. [Vision & Philosophy](#1-vision-philosophy)
2. [Architecture Overview](#2-architecture-overview)
3. [Component Model](#3-component-model)
   - [3.9 Frontend API Reference](#39-frontend-api-reference) — Complete reference for all APIs:
     - Elements, Attributes, Event Handlers
     - Signal, Memo, Resource, Action, Effect, Ref
     - Helpers (If, Range, Fragment, Key, Text)
     - Context, Form, URLParam
   - [3.10 Structured Side Effects](#310-structured-side-effects) — Action, Interval, Subscribe, GoLatest, budgets
4. [The Server-Driven Runtime](#4-the-server-driven-runtime)
5. [The Thin Client](#5-the-thin-client)
6. [The WASM Runtime](#6-the-wasm-runtime)
7. [State Management](#7-state-management)
8. [Interaction Primitives](#8-interaction-primitives)
   - [8.2 Client Hooks](#82-client-hooks) — 60fps interactions (drag-drop, sortable)
   - [8.4 Standard Hooks](#84-standard-hooks) — Sortable, Draggable, Tooltip, etc.
   - [8.5 Custom Hooks](#85-custom-hooks) — Define your own client behaviors
9. [Routing & Navigation](#9-routing-navigation)
10. [Data & APIs](#10-data-apis)
11. [Forms & Validation](#11-forms-validation)
   - [11.4 Toast Notifications](#114-toast-notifications)
   - [11.5 File Uploads](#115-file-uploads)
12. [JavaScript Islands](#12-javascript-islands)
13. [Styling](#13-styling)
14. [Performance & Scaling](#14-performance-scaling)
15. [Security](#15-security)
   - [15.9 Authentication & Middleware](#159-authentication-middleware)
16. [Testing](#16-testing)
17. [Developer Experience](#17-developer-experience)
18. [Migration Guide](#18-migration-guide)
19. [Examples](#19-examples)
20. [FAQ](#20-faq)
21. [Appendix A: Side-Effect Patterns (Informative)](#appendix-a-side-effect-patterns-informative)
22. [Appendix: Protocol Specification](#22-appendix-protocol-specification)

---


## Implementation Notes (Non-Normative)

This document specifies Vango’s v1 design. Implementation status, phase checklists, and verification notes live in `IMPLEMENTATION_AUDIT.md`.

---

## Document Conventions

This guide mixes **normative specification** with **informative explanation**.

- **Normative** statements use RFC 2119 keywords: **MUST**, **MUST NOT**, **SHOULD**, **SHOULD NOT**, **MAY**.
- **Informative** sections include rationale, examples, and pseudocode. Code blocks labeled “simplified” are illustrative and may omit edge cases.

## 1. Vision & Philosophy

### 1.1 The Problem with Modern Web Development

Modern web development is fragmented:

- **Two languages**: JavaScript on frontend, another language on backend
- **Two data models**: DTOs, JSON serialization, API contracts
- **Two state systems**: Client state, server state, synchronization hell
- **Heavy bundles**: 200KB+ of JavaScript before interactivity
- **Complex toolchains**: Webpack, Babel, TypeScript, bundlers, transpilers

What if we could write web applications in a single language, with direct access to the database, instant interactivity, and no bundle size concerns?

### 1.2 The Vango Approach

Vango is a **server-driven web framework** where:

1. **Components run on the server** by default
2. **UI updates flow as binary patches** over WebSocket
3. **The client is a thin renderer** (~12KB)
4. **You write Go everywhere** — no JavaScript required
5. **WASM is available** for offline or latency-sensitive features

This is similar to Phoenix LiveView (Elixir) or Laravel Livewire (PHP), but with Go's performance, type safety, and concurrency model.

### 1.3 Design Principles

| Principle | Meaning |
|-----------|---------|
| **Server-First** | Most code runs on the server. Client is minimal. |
| **One Language** | Go from database to DOM. No context switching. |
| **Type-Safe** | Compiler catches errors. No runtime surprises. |
| **Instant Interactive** | SSR means no waiting for bundles. |
| **Progressive Enhancement** | Works without JS, enhanced with WebSocket. |
| **Escape Hatches** | WASM and JS islands when you need them. |

### 1.4 When to Use Vango

**Ideal for:**
- CRUD applications (admin panels, dashboards)
- Collaborative apps (project management, documents)
- Data-heavy interfaces (analytics, reporting)
- Real-time features (chat, notifications, live updates)
- Internal tools (where Go backend teams own the frontend)

**Consider alternatives for:**
- Offline-first applications that must function without any server connection (use WASM mode)
- Extremely latency-sensitive apps where nearly all UI requires a client-side tight loop (games, pro drawing/music tools)
- Static content sites (use a static site generator)

---

## 2. Architecture Overview

### 2.1 The Three Modes

Vango supports three rendering modes. Choose based on your needs:

```
┌─────────────────────────────────────────────────────────────────┐
│                        VANGO MODES                              │
├─────────────────┬─────────────────┬─────────────────────────────┤
│  SERVER-DRIVEN  │     HYBRID      │           WASM              │
│   (Default)     │                 │                             │
├─────────────────┼─────────────────┼─────────────────────────────┤
│ Components run  │ Most on server, │ Components run in           │
│ on server       │ some on client  │ browser WASM                │
├─────────────────┼─────────────────┼─────────────────────────────┤
│ 12KB client     │ 12KB + partial  │ ~300KB WASM                 │
│                 │ WASM            │                             │
├─────────────────┼─────────────────┼─────────────────────────────┤
│ Requires        │ Requires        │ Works offline               │
│ connection      │ connection      │                             │
├─────────────────┼─────────────────┼─────────────────────────────┤
│ Best for: most  │ Best for: apps  │ Best for: offline,          │
│ web apps        │ with some       │ latency-critical            │
│                 │ latency needs   │                             │
└─────────────────┴─────────────────┴─────────────────────────────┘
```

### 2.2 Server-Driven Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                           BROWSER                                │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │                    Thin Client (12KB)                      │  │
│  │  ┌─────────────┐  ┌──────────────┐  ┌─────────────────┐    │  │
│  │  │   Event     │  │   Patch      │  │   Optimistic    │    │  │
│  │  │   Capture   │──│   Applier    │──│   Updates       │    │  │
│  │  └─────────────┘  └──────────────┘  └─────────────────┘    │  │
│  └──────────────────────────┬─────────────────────────────────┘  │
│                             │ WebSocket (Binary)                 │
└─────────────────────────────┼────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────┴────────────────────────────────────┐
│                           SERVER                                 │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │                    Vango Runtime                           │  │
│  │  ┌─────────────┐  ┌──────────────┐  ┌─────────────────┐    │  │
│  │  │   Session   │  │   Component  │  │   Diff          │    │  │
│  │  │   Manager   │──│   Tree       │──│   Engine        │    │  │
│  │  └─────────────┘  └──────────────┘  └─────────────────┘    │  │
│  │         │                │                   │             │  │
│  │         ▼                ▼                   ▼             │  │
│  │  ┌─────────────┐  ┌──────────────┐  ┌─────────────────┐    │  │
│  │  │   Signal    │  │   Event      │  │   Patch         │    │  │
│  │  │   Store     │  │   Router     │  │   Encoder       │    │  │
│  │  └─────────────┘  └──────────────┘  └─────────────────┘    │  │
│  └────────────────────────────────────────────────────────────┘  │
│                             │                                    │
│                             ▼                                    │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │              Direct Access (No HTTP/JSON)                  │  │
│  │  ┌─────────────┐  ┌──────────────┐  ┌─────────────────┐    │  │
│  │  │  Database   │  │    Cache     │  │   Services      │    │  │
│  │  └─────────────┘  └──────────────┘  └─────────────────┘    │  │
│  └────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
```

### 2.3 Request Lifecycle

**Initial Page Load:**
```
1. Browser requests GET /projects/123
2. Server matches route → ProjectPage(id=123)
3. Component renders → VNode tree
4. VNode tree → HTML string (SSR)
5. HTML sent to browser (user sees content immediately)
6. Thin client JS loads (~12KB)
7. WebSocket connection established
8. Page is now interactive
```

**User Interaction:**
```
1. User clicks "Complete Task" button
2. Thin client captures click event
3. Binary event sent: {type: CLICK, hid: "h42"}
4. Server finds session, finds handler for hid="h42"
5. Handler runs: task.Complete()
6. Affected signals update, component re-renders
7. Diff: old VNode vs new VNode → patches
8. Binary patches sent: [{SET_TEXT, "h17", "✓ Done"}]
9. Thin client applies patches to DOM
10. User sees "✓ Done" (~50-80ms total)
```

### 2.4 Same Components, Different Modes

The same component code works in all modes:

```go
func Counter(initial int) vango.Component {
    return vango.Func(func() *vango.VNode {
        count := vango.NewSignal(initial)

        return Div(Class("counter"),
            H1(Textf("Count: %d", count.Get())),
            Button(OnClick(count.Inc), Text("+")),
            Button(OnClick(count.Dec), Text("-")),
        )
    })
}
```

| Mode | Where `Signal` lives | Where `OnClick` runs | How DOM updates |
|------|---------------------|---------------------|-----------------|
| Server-Driven | Server memory | Server | Binary patches over WS |
| WASM | Browser WASM memory | Browser WASM | Direct DOM manipulation |
| Hybrid | Depends on component | Depends on component | Mixed |

---

## 3. Component Model

### 3.1 Core Concepts


Vango has five core concepts:

| Concept | Current Go API | Description |
|---------|----------------|-------------|
| **Signal** | `vango.NewSignal(...)` | Reactive state; read via `.Get()` |
| **Memo** | `vango.NewMemo(...)` | Derived state/calculations |
| **Effect** | `vango.Effect(...)` | Wiring/integration (prefer structured helpers; see §3.10) |
| **Element** | `el.Div(...)`, etc. | UI structure via DSL |
| **Component**| `vango.Func(...)` | Composition and state encapsulation |

| Concept | Purpose | Example |
|---------|---------|---------|
| **Element** | UI structure | `Div(Class("card"), ...)` |
| **Signal** | Reactive state | `count := vango.NewSignal(0)` |
| **Memo** | Derived state | `doubled := vango.NewMemo(...)` |
| **Effect** | Wiring/integration | `vango.Effect(func() {...})` |
| **Component** | Composition | `func Card() vango.Component` |

Additional first-class primitives: **Resource** (async queries), **Action** (async mutations), Effect helpers (**Interval**, **Subscribe**, **GoLatest**), and **Ref** (client-only handles). See §3.10.

### 3.1.1 Runtime Context (`Ctx`)


Many APIs in this guide (navigation, dispatching work back onto the session loop, URL query state, toasts, etc.) require an active **runtime context**.

Vango provides runtime context in two ways:

1. **Ctx parameter**: routing entrypoints receive a `ctx vango.Ctx` parameter (e.g. `func Page(ctx vango.Ctx, ...) *vango.VNode`).
2. **Ambient access inside render/effect/handlers**: within `vango.Func` render closures, effects, and event handlers, you can retrieve the current context:

```go
// UseCtx returns the current runtime context for the active session tick.
// It MUST only be called when a session tick is active: during component render,
// event handler execution, Effect execution, or any callback invoked on the session
// loop via ctx.Dispatch(...).
func vango.UseCtx() vango.Ctx
```

If you need a `ctx` inside a render/effect closure, prefer `ctx := vango.UseCtx()` rather than relying on an unscoped global.

**Normative**: Calling context-dependent helpers/primitives when `UseCtx()` is invalid MUST panic with a diagnostic error. This includes `NewAction` (§3.10.2), `Interval` (§3.10.3.2), `Subscribe` (§3.10.3.3), and `GoLatest` (§3.10.3.4).

### 3.1.2 Rules of Render (Pit of Success)

Render functions (`vango.Func(func() *vango.VNode { ... })`) are the hot path and must be predictable.

Normative rules:
- Render MUST be pure: no blocking I/O, no goroutine creation (goroutines are fine in effects/handlers), and no non-deterministic reads (time, randomness, global mutable state) unless explicitly modeled as state.
- Render MUST NOT write signals (no `Set/Update/Inc/...`) because it can create re-entrant update loops.
- Event handlers (e.g. `OnClick`) are where signal writes SHOULD happen for user interactions.
- Effects (`vango.Effect`) are where side effects are wired into the lifecycle; prefer `Resource`, `Action`, and Effect helpers (`Interval`/`Subscribe`/`GoLatest`) over raw goroutines (see §3.9.6 and §3.10).
- Prefer `sig.Peek()` / `vango.Untracked` for analytics/logging reads to avoid accidental reactive dependencies.

### 3.1.3 Hook-Order Semantics for Render-Time Stateful Primitives


Any API that allocates component-scoped state during render MUST obey hook-order semantics. This includes: `NewSignal`, `NewMemo`, `Effect`, `NewResource`, `NewResourceKeyed`, `NewAction`, `NewRef`, `URLParam`, `UseForm`, `CreateContext.Use()`, and lifecycle hooks (`OnMount`, `OnUnmount`, `OnUpdate`).

**Normative Rule**: In a given component’s render function, the sequence of hook calls MUST be identical on every render tick. Hooks MUST NOT be called conditionally or inside loops whose iteration count depends on reactive state.

**Clarifications**:
- `vango.Effect` is a render-time hook (it allocates managed effect state). It MUST only be called during render and MUST obey ordering.
- `CreateContext.Use()` is a reactive hook: it subscribes the component to the nearest provider. It MUST obey hook-order semantics.
- Lifecycle hooks (`OnMount`, etc.) are hook-order primitives.
- `vango.NewAction(...)` allocates managed state and MUST obey hook-order semantics (see §3.10.2).

This requires a stable “slot” model:
- Within a given component instance, hook calls MUST occur in the same order on every render.
- Hook calls MUST NOT be conditional (inside `if`, `switch` cases with early returns, or data-dependent branches) and MUST NOT occur in loops with variable iteration counts.
- Violations SHOULD produce a dev-mode error such as: “Hook order changed (Signal/Memo/Effect/Resource/Action/Ref/Form/Param/Lifecycle)”.

Conditional rendering / early returns are allowed as long as all hook-creating calls occur unconditionally before the branch.

This rule is why patterns like “don’t read signals conditionally” and “use `Key(...)` for lists” matter: they preserve identity and stability for both DOM reconciliation and per-component reactive state.

### 3.2 Elements

Elements are functions that accept mixed attributes and children:

```go
import (
    . "vango/el"  // Dot import for concise syntax
    "vango"
)

// Basic element
Div(Class("card"), Text("Hello"))

// Nested elements
Div(Class("container"),
    H1(Text("Title")),
    P(Class("subtitle"), Text("Description")),
    Button(OnClick(handleClick), Text("Click me")),
)

// Attributes and children can be mixed freely
Form(
    Method("POST"),
    Class("login-form"),
    Input(Type("email"), Name("email"), Placeholder("Email")),
    Input(Type("password"), Name("password")),
    Button(Type("submit"), Text("Login")),
)
```

**Why this syntax?**
- Pure Go — no custom parser, standard tooling works
- Type-safe — compiler catches errors
- Flexible — attributes and children intermix naturally
- Readable — structure mirrors HTML output

### 3.3 Signals

Signals are reactive values that trigger re-renders when changed:

```go
func Counter(initial int) vango.Component {
    return vango.Func(func() *vango.VNode {
        // Create a signal
        count := vango.NewSignal(initial)

        // Read the value (subscribes this component)
        currentValue := count.Get()

        // Update the value (triggers re-render)
        increment := func() {
            count.Set(count.Get() + 1)
        }

        return Div(
            Text(fmt.Sprintf("Count: %d", count.Get())),
            Button(OnClick(increment), Text("+")),
        )
    })
}
```

> **Go API Design Note**: To follow Go’s naming constraints (where a type and a function cannot share a name in the same package), Vango uses the `New*` prefix for all signal and state constructors. Additionally, signal values are accessed via an explicit `.Get()` method to distinguish them from function calls and to enable precise reactivity tracking.

**Signal API:**
```go
// Create
count := vango.NewSignal(0)           // Signal[int]
user := vango.NewSignal[*User](nil)            // Signal[*User]
items := vango.NewSignal([]Item{})    // Signal[[]Item]

// Read (subscribes component to changes)
value := count.Get()

// Write
count.Set(5)
count.Update(func(n int) int { return n + 1 })

// Convenience methods
count.Inc()       // +1 (integers only)
count.Dec()       // -1 (integers only)
enabled.Toggle()  // !current (booleans only)
```

### 3.4 Memos

Memos are cached computations that update when dependencies change:

```go
func ShoppingCart() vango.Component {
    return vango.Func(func() *vango.VNode {
        items := vango.NewSignal([]CartItem{})
        taxRate := vango.NewSignal(0.08)

        // Recomputes only when items.Get() changes
        subtotal := vango.NewMemo(func() float64 {
            total := 0.0
            for _, item := range items.Get() {
                total += item.Price * float64(item.Qty)
            }
            return total
        })

        // Recomputes when subtotal.Get() or taxRate.Get() changes
        tax := vango.NewMemo(func() float64 {
            return subtotal.Get() * taxRate.Get()
        })

        // Memos can depend on other memos
        total := vango.NewMemo(func() float64 {
            return subtotal.Get() + tax.Get()
        })

        return Div(
            CartItems(items),
            Div(Class("totals"),
                Row("Subtotal", subtotal.Get()),
                Row("Tax", tax.Get()),
                Row("Total", total.Get()),
            ),
        )
    })
}
```

### 3.5 Effects

Effects run after render and wire side effects into the lifecycle. Prefer structured primitives (§3.10) for common async work; for example, use `Resource` for async data loading:

```go
func UserProfile(userID int) vango.Component {
    return vango.Func(func() *vango.VNode {
        ctx := vango.UseCtx()
        user := vango.NewResource(func() (*User, error) {
            return db.Users.FindByID(ctx.StdContext(), userID)
        })

        return user.Match(
            vango.OnLoading(func() *vango.VNode {
                return LoadingSpinner()
            }),
            vango.OnError(func(err error) *vango.VNode {
                return ErrorMessage(err)
            }),
            vango.OnReady(func(u *User) *vango.VNode {
                return Div(
                    H1(Text(u.Name)),
                    P(Text(u.Email)),
                )
            }),
        )
    })
}
```

**Effect Timing:**
| When | What Happens |
|------|--------------|
| After first render | Effect runs |
| Signal dependency changes | Effect re-runs (after cleanup) |
| Component unmounts | Cleanup runs |

### 3.6 Component Types

**Stateless Components** — Pure functions returning VNodes:
```go
func Greeting(name string) *vango.VNode {
    return H1(Class("greeting"), Textf("Hello, %s!", name))
}

// Usage
Div(
    Greeting("Alice"),
    Greeting("Bob"),
)
```

**Stateful Components** — Functions returning `vango.Component`:
```go
func Counter(initial int) vango.Component {
    return vango.Func(func() *vango.VNode {
        count := vango.NewSignal(initial)
        return Div(
            Text(fmt.Sprintf("%d", count.Get())),
            Button(OnClick(count.Inc), Text("+")),
        )
    })
}

// Usage
Div(
    Counter(0),
    Counter(100),
)
```

**Components with Children:**
```go
func Card(title string, children ...any) *vango.VNode {
    return Div(Class("card"),
        H2(Class("card-title"), Text(title)),
        Div(Class("card-body"), children...),
    )
}

// Usage
Card("Settings",
    Form(
        Input(Type("text"), Name("name")),
        Button(Type("submit"), Text("Save")),
    ),
)
```

### 3.7 Conditional Rendering

```go
// Simple conditional
If(isLoggedIn,
    UserMenu(user),
)

// If-else
IfElse(isLoggedIn,
    UserMenu(user),
    LoginButton(),
)

// Inline conditional (when you need the else to be nil)
func() *vango.VNode {
    if isLoggedIn {
        return UserMenu(user)
    }
    return nil
}()

// Switch-like patterns
func StatusBadge(status string) *vango.VNode {
    switch status {
    case "active":
        return Badge("success", "Active")
    case "pending":
        return Badge("warning", "Pending")
    default:
        return Badge("gray", "Unknown")
    }
}
```

### 3.8 List Rendering

```go
// With keys (required for efficient updates)
func TaskList(tasks []Task) *vango.VNode {
    return Ul(Class("task-list"),
        Range(tasks, func(task Task, i int) *vango.VNode {
            return Li(
                Key(task.ID),  // Stable key for reconciliation
                Class("task"),
                Text(task.Title),
            )
        }),
    )
}

// Without Range helper (manual approach)
func TaskList(tasks []Task) *vango.VNode {
    items := make([]any, len(tasks))
    for i, task := range tasks {
        items[i] = Li(Key(task.ID), Text(task.Title))
    }
    return Ul(items...)
}
```

---

## 3.9 Frontend API Reference

This section provides a complete reference for all Vango frontend APIs. For quick lookup, use the following categories:

- [3.9.1 HTML Elements](#391-html-elements)
- [3.9.2 Attributes](#392-attributes)
- [3.9.3 Event Handlers](#393-event-handlers)
- [3.9.9 Helper Functions](#399-helper-functions)
- [3.9.4 Signal API](#394-signal-api)
- [3.9.5 Memo API](#395-memo-api)
- [3.9.7 Resource API](#397-resource-api)
- [3.10 Structured Side Effects](#310-structured-side-effects)
- [3.9.6 Effect API](#396-effect-api) (wiring-only by default; see §3.10.4)
- [3.9.8 Ref API](#398-ref-api)
- [3.9.10 Context API](#3910-context-api)
- [3.9.11 Form API](#3911-form-api)
- [3.9.12 URLParam](#3912-urlparam-query-state)

---

### 3.9.1 HTML Elements


All standard HTML elements are available as functions. Import with dot notation for concise syntax:

```go
import . "vango/el"
```

#### Element Signatures

Every element function accepts variadic `any` arguments that can be:
- **Attributes**: `Class("card")`, `ID("main")`, `StyleAttr("color: red")`
- **Event handlers**: `OnClick(fn)`, `OnInput(fn)`
- **Children**: Other elements, `Text("...")`, `*vango.VNode`, `vango.Component`
- **nil**: Safely ignored

```go
func Div(args ...any) *vango.VNode
func Span(args ...any) *vango.VNode
func Button(args ...any) *vango.VNode
// ... all HTML elements follow this pattern
```

#### Document Structure

| Element | Description | Common Attributes |
|---------|-------------|-------------------|
| `Html()` | Root element | `Lang("en")` |
| `Head()` | Document head | - |
| `Body()` | Document body | `Class()` |
| `Title()` | Page title | - |
| `Meta()` | Metadata | `Name()`, `Content()`, `Charset()` |
| `Link()` | External resource | `Rel()`, `Href()`, `Type()` |
| `Script()` | Script element | `Src()`, `Type()`, `Defer_()`, `Async()` |
| `Style()` | Inline styles | `Type()` |

#### Content Sectioning

| Element | Description | Semantic Use |
|---------|-------------|--------------|
| `Header()` | Header section | Page/section header |
| `Footer()` | Footer section | Page/section footer |
| `Main()` | Main content | Primary page content |
| `Nav()` | Navigation | Navigation links |
| `Section()` | Generic section | Thematic content grouping |
| `Article()` | Article | Self-contained content |
| `Aside()` | Sidebar | Tangentially related content |
| `H1()` - `H6()` | Headings | Section headings |
| `Hgroup()` | Heading group | Multi-level heading |
| `Address()` | Contact info | Contact information |

#### Text Content

| Element | Description | Common Use |
|---------|-------------|------------|
| `Div()` | Generic container | Layout, grouping |
| `P()` | Paragraph | Text paragraphs |
| `Span()` | Inline container | Inline styling |
| `Pre()` | Preformatted | Code blocks |
| `Blockquote()` | Quote block | Quotations |
| `Ul()` | Unordered list | Bullet lists |
| `Ol()` | Ordered list | Numbered lists |
| `Li()` | List item | List items |
| `Dl()` | Description list | Term-description pairs |
| `Dt()` | Description term | Term in dl |
| `Dd()` | Description details | Description in dl |
| `Figure()` | Figure | Illustrations, diagrams |
| `Figcaption()` | Figure caption | Caption for figure |
| `Hr()` | Horizontal rule | Thematic break |

#### Inline Text

| Element | Description | Renders As |
|---------|-------------|------------|
| `A()` | Anchor/link | Hyperlink |
| `Strong()` | Strong importance | Bold |
| `Em()` | Emphasis | Italic |
| `B()` | Bring attention | Bold (no semantic) |
| `I()` | Alternate voice | Italic (no semantic) |
| `U()` | Unarticulated | Underline |
| `S()` | Strikethrough | Strikethrough |
| `Small()` | Side comment | Smaller text |
| `Mark()` | Highlight | Highlighted |
| `Sub()` | Subscript | Subscript |
| `Sup()` | Superscript | Superscript |
| `Code()` | Code | Monospace |
| `Kbd()` | Keyboard input | Key styling |
| `Samp()` | Sample output | Output styling |
| `Var()` | Variable | Variable styling |
| `Abbr()` | Abbreviation | With title tooltip |
| `Time_()` | Time | Datetime value |
| `Br()` | Line break | Newline |
| `Wbr()` | Word break | Optional break |

#### Forms

| Element | Description | Key Attributes |
|---------|-------------|----------------|
| `Form()` | Form container | `Action()`, `Method()`, `OnSubmit()` |
| `Input()` | Input field | `Type()`, `Name()`, `Value()`, `Placeholder()` |
| `Textarea()` | Multi-line input | `Name()`, `Rows()`, `Cols()` |
| `Select()` | Dropdown | `Name()`, `Multiple()` |
| `Option()` | Select option | `Value()`, `Selected()` |
| `Optgroup()` | Option group | `Label()` |
| `Button()` | Button | `Type()`, `Disabled()` |
| `Label()` | Form label | `For()` |
| `Fieldset()` | Field group | - |
| `Legend()` | Fieldset legend | - |
| `Datalist()` | Input suggestions | `ID()` |
| `Output()` | Calculation result | `Name()`, `For()` |
| `Progress()` | Progress bar | `Value()`, `Max()` |
| `Meter()` | Scalar measurement | `Value()`, `Min()`, `Max()` |

#### Tables

| Element | Description | Key Attributes |
|---------|-------------|----------------|
| `Table()` | Table container | - |
| `Thead()` | Table header | - |
| `Tbody()` | Table body | - |
| `Tfoot()` | Table footer | - |
| `Tr()` | Table row | - |
| `Th()` | Header cell | `Scope()`, `Colspan()`, `Rowspan()` |
| `Td()` | Data cell | `Colspan()`, `Rowspan()` |
| `Caption()` | Table caption | - |
| `Colgroup()` | Column group | - |
| `Col()` | Column | `Span()` |

#### Media

| Element | Description | Key Attributes |
|---------|-------------|----------------|
| `Img()` | Image | `Src()`, `Alt()`, `Width()`, `Height()` |
| `Picture()` | Responsive image | - |
| `Source()` | Media source | `Srcset()`, `Media()`, `Type()` |
| `Video()` | Video | `Src()`, `Controls()`, `Autoplay()` |
| `Audio()` | Audio | `Src()`, `Controls()` |
| `Track()` | Text track | `Src()`, `Kind()`, `Srclang()` |
| `Iframe()` | Inline frame | `Src()`, `Width()`, `Height()` |
| `Embed()` | External content | `Src()`, `Type()` |
| `Object()` | External object | `Data()`, `Type()` |
| `Canvas()` | Drawing canvas | `Width()`, `Height()` |
| `Svg()` | SVG container | `Viewbox()`, `Width()`, `Height()` |

#### Interactive

| Element | Description | Key Attributes |
|---------|-------------|----------------|
| `Details()` | Disclosure widget | `Open()` |
| `Summary()` | Details summary | - |
| `Dialog()` | Dialog box | `Open()` |
| `Menu()` | Menu container | - |

---

### 3.9.2 Attributes


Attributes are functions that return attribute values. They can be mixed with children in element calls.

#### Global Attributes

These work on any HTML element:

```go
// Identity
ID("main-content")          // id="main-content"
Class("card", "active")     // class="card active"
StyleAttr("color: red")     // style="color: red"

// Data attributes
Data("id", "123")           // data-id="123"
Data("user-role", "admin")  // data-user-role="admin"

// Accessibility
Role("button")              // role="button"
AriaLabel("Close")          // aria-label="Close"
AriaHidden(true)            // aria-hidden="true"
AriaExpanded(false)         // aria-expanded="false"
AriaDescribedBy("desc")     // aria-describedby="desc"
AriaLabelledBy("title")     // aria-labelledby="title"
AriaLive("polite")          // aria-live="polite"
AriaAtomic(true)            // aria-atomic="true"
AriaBusy(false)             // aria-busy="false"
AriaControls("menu")        // aria-controls="menu"
AriaCurrent("page")         // aria-current="page"
AriaDisabled(true)          // aria-disabled="true"
AriaHasPopup("menu")        // aria-haspopup="menu"
AriaPressed("true")         // aria-pressed="true"
AriaSelected(true)          // aria-selected="true"

// Keyboard
TabIndex(0)                 // tabindex="0"
TabIndex(-1)                // tabindex="-1"
AccessKey("s")              // accesskey="s"

// Visibility
Hidden()                    // hidden
TitleAttr("Tooltip text")   // title="Tooltip text"

// Behavior
ContentEditable(true)       // contenteditable="true"
Draggable()                 // draggable="true"
Spellcheck(false)           // spellcheck="false"
Translate(false)            // translate="no"

// Language/Direction
Lang("en")                  // lang="en"
Dir("ltr")                  // dir="ltr"
```

#### Link Attributes

```go
// Anchor
Href("/users")              // href="/users"
Href(router.User(123))      // href="/users/123" (type-safe)
Target("_blank")            // target="_blank"
Rel("noopener")             // rel="noopener"
Download()                  // download
Download("file.pdf")        // download="file.pdf"
Hreflang("en")              // hreflang="en"
Ping("/track")              // ping="/track"
ReferrerPolicy("origin")    // referrerpolicy="origin"
```

#### Form Input Attributes

```go
// Common
Name("email")               // name="email"
Value("hello")              // value="hello"
Type("email")               // type="email"
Placeholder("Enter email")  // placeholder="Enter email"

// Input Types
Type("text")                // type="text"
Type("password")            // type="password"
Type("email")               // type="email"
Type("number")              // type="number"
Type("tel")                 // type="tel"
Type("url")                 // type="url"
Type("search")              // type="search"
Type("date")                // type="date"
Type("time")                // type="time"
Type("datetime-local")      // type="datetime-local"
Type("month")               // type="month"
Type("week")                // type="week"
Type("color")               // type="color"
Type("file")                // type="file"
Type("hidden")              // type="hidden"
Type("checkbox")            // type="checkbox"
Type("radio")               // type="radio"
Type("range")               // type="range"
Type("submit")              // type="submit"
Type("reset")               // type="reset"
Type("button")              // type="button"
Type("image")               // type="image"

// States
Disabled()                  // disabled
Readonly()                  // readonly
Required()                  // required
Checked()                   // checked
Selected()                  // selected
Multiple()                  // multiple
Autofocus()                 // autofocus
Autocomplete("email")       // autocomplete="email"

// Validation
Pattern(`[0-9]+`)           // pattern="[0-9]+"
MinLength(3)                // minlength="3"
MaxLength(100)              // maxlength="100"
Min("0")                    // min="0"
Max("100")                  // max="100"
Step("0.01")                // step="0.01"

// Files
Accept("image/*")           // accept="image/*"
Capture("user")             // capture="user"

// Text areas
Rows(5)                     // rows="5"
Cols(40)                    // cols="40"
Wrap("soft")                // wrap="soft"
```

#### Form Attributes

```go
// Form element
Action("/submit")           // action="/submit"
Method("POST")              // method="POST"
Enctype("multipart/form-data")  // enctype="..."
Novalidate()                // novalidate
Autocomplete("off")         // autocomplete="off"

// Label
For("input-id")             // for="input-id"

// Button
FormAction("/other")        // formaction="/other"
FormMethod("GET")           // formmethod="GET"
FormNovalidate()            // formnovalidate
```

#### Media Attributes

```go
// Image
Src("/img/photo.jpg")       // src="/img/photo.jpg"
Alt("Description")          // alt="Description"
Width(300)                  // width="300"
Height(200)                 // height="200"
Loading("lazy")             // loading="lazy"
Decoding("async")           // decoding="async"
Srcset("...")               // srcset="..."
Sizes("...")                // sizes="..."

// Video/Audio
Controls()                  // controls
Autoplay()                  // autoplay
Loop()                      // loop
Muted()                     // muted
Preload("metadata")         // preload="metadata"
Poster("/poster.jpg")       // poster="/poster.jpg"
Playsinline()               // playsinline

// Iframe
Sandbox("allow-scripts")    // sandbox="allow-scripts"
Allow("fullscreen")         // allow="fullscreen"
Allowfullscreen()           // allowfullscreen
```

#### Table Attributes

```go
Colspan(2)                  // colspan="2"
Rowspan(3)                  // rowspan="3"
Scope("col")                // scope="col"
Headers("h1 h2")            // headers="h1 h2"
```

#### Miscellaneous Attributes

```go
// Lists
Start(5)                    // start="5" (ol)
Reversed()                  // reversed (ol)

// Details
Open()                      // open

// Meta
Charset("utf-8")            // charset="utf-8"
Content("...")              // content="..."
HttpEquiv("refresh")        // http-equiv="refresh"

// Link
Rel("stylesheet")           // rel="stylesheet"
As("style")                 // as="style"
Crossorigin("anonymous")    // crossorigin="anonymous"
Integrity("sha384-...")     // integrity="..."

// Custom/raw attribute
Attr("x-custom", "value")   // x-custom="value"
```

---

### 3.9.3 Event Handlers


Event handlers trigger server-side callbacks or client-side behavior.

#### Handler Function Signatures (Normative)

Go does not support function overloading. Vango’s event helper functions (e.g. `OnClick`, `OnInput`) therefore accept `any` and resolve/validate supported handler function signatures at runtime (via type assertions / reflection).

Normative rules:
- Each event helper MUST support only a documented set of handler signatures.
- If a handler has an unsupported signature, Vango SHOULD fail fast in dev mode with a descriptive error (and SHOULD include the event name and the unexpected type). In production it MAY ignore the handler and log an error.
- Where “shorthand” signatures exist (e.g. `func(string)` for input), they MUST be defined as a deterministic projection of the full event payload.

Supported signatures (common):

| Helper | Supported handler types | Notes |
|--------|--------------------------|------|
| `OnClick`, `OnDblClick`, `OnMouseDown/Up/Move/Enter/Leave/Over/Out`, `OnContextMenu` | `func()` or `func(vango.MouseEvent)` | `func()` is allowed when payload is not needed. |
| `OnWheel` | `func(vango.WheelEvent)` | |
| `OnKeyDown`, `OnKeyUp` | `func()` or `func(vango.KeyboardEvent)` | Prefer the event form for key/modifier inspection. |
| `OnInput`, `OnChange` | `func(string)` or `func(vango.InputEvent)` | `func(string)` receives `e.Value`. |
| `OnSubmit` | `func()` or `func(vango.FormData)` | `FormData` contains decoded form fields. |
| `OnScroll` | `func(vango.ScrollEvent)` | |
| `OnDragStart/Drag/DragEnd/DragEnter/DragLeave/Drop` | `func(vango.DragEvent)` | |
| `OnDragOver` | `func(vango.DragEvent) bool` | Return `true` to allow drop; returning `false` preserves native behavior. |
| `OnEvent(name, ...)` (hooks) | `func(vango.HookEvent)` | Fixed signature. |

Payload completeness:
- Event structs are decoded from browser events on a best-effort basis.
- If the browser cannot supply a field, it MUST decode as the Go zero value for that field.
- Pointer-heavy or high-frequency event payloads (e.g. per-mousemove) SHOULD be kept small; use hooks for 60fps interactions.

#### Mouse Events

```go
// Click
OnClick(func() {
    // Handle click
})

OnClick(func(e vango.MouseEvent) {
    // Access event details
    fmt.Println(e.ClientX, e.ClientY)
})

// Double click
OnDblClick(func() { })

// Mouse buttons
OnMouseDown(func(e vango.MouseEvent) { })
OnMouseUp(func(e vango.MouseEvent) { })

// Mouse movement
OnMouseMove(func(e vango.MouseEvent) { })
OnMouseEnter(func() { })
OnMouseLeave(func() { })
OnMouseOver(func() { })
OnMouseOut(func() { })

// Context menu
OnContextMenu(func(e vango.MouseEvent) { })

// Wheel
OnWheel(func(e vango.WheelEvent) { })
```

**MouseEvent:**
```go
type MouseEvent struct {
    ClientX   int     // X relative to viewport
    ClientY   int     // Y relative to viewport
    PageX     int     // X relative to document
    PageY     int     // Y relative to document
    OffsetX   int     // X relative to target
    OffsetY   int     // Y relative to target
    Button    int     // 0=left, 1=middle, 2=right
    Buttons   int     // Bitmask of pressed buttons
    CtrlKey   bool    // Ctrl held
    ShiftKey  bool    // Shift held
    AltKey    bool    // Alt held
    MetaKey   bool    // Meta/Cmd held
}
```

#### Keyboard Events

```go
OnKeyDown(func(e vango.KeyboardEvent) {
    if e.Key == "Enter" && !e.ShiftKey {
        submit()
    }
})

OnKeyUp(func(e vango.KeyboardEvent) { })

OnKeyPress(func(e vango.KeyboardEvent) { })  // Deprecated, use KeyDown
```

**KeyboardEvent:**
```go
type KeyboardEvent struct {
    Key       string  // "Enter", "a", "Escape", etc.
    Code      string  // "Enter", "KeyA", "Escape", etc.
    CtrlKey   bool
    ShiftKey  bool
    AltKey    bool
    MetaKey   bool
    Repeat    bool    // True if key is held down
    Location  int     // 0=standard, 1=left, 2=right, 3=numpad
}
```

**Common Key Values:**
| Key Constant | Value |
|--------------|-------|
| `vango.KeyEnter` | "Enter" |
| `vango.KeyEscape` | "Escape" |
| `vango.KeySpace` | " " |
| `vango.KeyTab` | "Tab" |
| `vango.KeyBackspace` | "Backspace" |
| `vango.KeyDelete` | "Delete" |
| `vango.KeyArrowUp` | "ArrowUp" |
| `vango.KeyArrowDown` | "ArrowDown" |
| `vango.KeyArrowLeft` | "ArrowLeft" |
| `vango.KeyArrowRight` | "ArrowRight" |
| `vango.KeyHome` | "Home" |
| `vango.KeyEnd` | "End" |
| `vango.KeyPageUp` | "PageUp" |
| `vango.KeyPageDown` | "PageDown" |

#### Form Events

```go
// Input changes (fires on each keystroke)
OnInput(func(value string) {
    searchQuery.Set(value)
})

OnInput(func(e vango.InputEvent) {
    fmt.Println(e.Value)
})

// Change (fires on blur/commit)
OnChange(func(value string) {
    filter.Set(value)
})

// Form submission
OnSubmit(func(data vango.FormData) {
    email := data.Get("email")
    password := data.Get("password")
    handleLogin(email, password)
})

// Prevent default (the callback returning is implicit prevention)
// To explicitly prevent:
OnSubmit(func(data vango.FormData) {
    // Submitting via Vango already prevents browser default
})

// Focus
OnFocus(func() { })
OnBlur(func() { })
OnFocusIn(func() { })
OnFocusOut(func() { })

// Selection
OnSelect(func() { })

// Invalid (form validation)
OnInvalid(func() { })

// Reset
OnReset(func() { })
```

**FormData:**
```go
type FormData struct {
    values map[string][]string
}

func (f FormData) Get(key string) string          // First value
func (f FormData) GetAll(key string) []string     // All values
func (f FormData) Has(key string) bool            // Key exists
func (f FormData) Keys() []string                 // All keys
```

#### Drag Events

```go
// Draggable element
OnDragStart(func(e vango.DragEvent) {
    e.SetData("text/plain", item.ID)
})
OnDrag(func(e vango.DragEvent) { })
OnDragEnd(func(e vango.DragEvent) { })

// Drop target
OnDragEnter(func(e vango.DragEvent) { })
OnDragOver(func(e vango.DragEvent) bool {
    return true  // Allow drop
})
OnDragLeave(func(e vango.DragEvent) { })
OnDrop(func(e vango.DropEvent) {
    data := e.GetData("text/plain")
})
```

#### Touch Events

```go
OnTouchStart(func(e vango.TouchEvent) { })
OnTouchMove(func(e vango.TouchEvent) { })
OnTouchEnd(func(e vango.TouchEvent) { })
OnTouchCancel(func(e vango.TouchEvent) { })
```

**TouchEvent:**
```go
type TouchEvent struct {
    Touches        []Touch  // All current touches
    TargetTouches  []Touch  // Touches on this element
    ChangedTouches []Touch  // Touches that changed
}

type Touch struct {
    Identifier int
    ClientX    int
    ClientY    int
    PageX      int
    PageY      int
}
```

#### Animation/Transition Events

```go
OnAnimationStart(func(e vango.AnimationEvent) { })
OnAnimationEnd(func(e vango.AnimationEvent) { })
OnAnimationIteration(func(e vango.AnimationEvent) { })
OnAnimationCancel(func(e vango.AnimationEvent) { })

OnTransitionStart(func(e vango.TransitionEvent) { })
OnTransitionEnd(func(e vango.TransitionEvent) { })
OnTransitionRun(func(e vango.TransitionEvent) { })
OnTransitionCancel(func(e vango.TransitionEvent) { })
```

#### Media Events

```go
OnPlay(func() { })
OnPause(func() { })
OnEnded(func() { })
OnTimeUpdate(func(currentTime float64) { })
OnDurationChange(func(duration float64) { })
OnVolumeChange(func(volume float64, muted bool) { })
OnSeeking(func() { })
OnSeeked(func() { })
OnLoadStart(func() { })
OnLoadedData(func() { })
OnLoadedMetadata(func() { })
OnCanPlay(func() { })
OnCanPlayThrough(func() { })
OnWaiting(func() { })
OnPlaying(func() { })
OnProgress(func() { })
OnStalled(func() { })
OnSuspend(func() { })
OnError(func(err error) { })
```

#### Scroll Events

```go
OnScroll(func(e vango.ScrollEvent) {
    fmt.Println(e.ScrollTop, e.ScrollLeft)
})

// Throttled version (recommended for performance)
OnScroll(vango.Throttle(100*time.Millisecond, func(e vango.ScrollEvent) {
    if e.ScrollTop > 100 {
        showBackToTop.Set(true)
    }
}))
```

#### Window/Document Events

These are used at the layout or page level:

```go
OnLoad(func() { })
OnUnload(func() { })
OnBeforeUnload(func() string {
    return "You have unsaved changes"  // Browser shows confirmation
})
OnResize(func(width, height int) { })
OnPopState(func(state any) { })
OnHashChange(func(oldURL, newURL string) { })
OnOnline(func() { })
OnOffline(func() { })
OnVisibilityChange(func(hidden bool) { })
```

#### Event Modifiers

Modify event behavior:

```go
// Prevent default browser behavior
OnClick(vango.PreventDefault(func() {
    // Click handled, default prevented
}))

// Stop event propagation
OnClick(vango.StopPropagation(func() {
    // Click won't bubble up
}))

// Both
OnClick(vango.PreventDefault(vango.StopPropagation(func() {
    // ...
})))

// Self-only (only fire if target is this element)
OnClick(vango.Self(func() {
    // Only fires if clicked element is this exact element
}))

// Once (remove after first trigger)
OnClick(vango.Once(func() {
    // Only fires once
}))

// Passive (for scroll performance)
OnScroll(vango.Passive(func(e vango.ScrollEvent) {
    // Cannot call preventDefault
}))

// Capture phase
OnClick(vango.Capture(func() {
    // Fires during capture phase
}))

// Debounce
OnInput(vango.Debounce(300*time.Millisecond, func(value string) {
    search(value)
}))

// Throttle
OnMouseMove(vango.Throttle(100*time.Millisecond, func(e vango.MouseEvent) {
    updatePosition(e.ClientX, e.ClientY)
}))

// Key modifiers
OnKeyDown(vango.Hotkey("Enter", func() {
    submit()
}))

OnKeyDown(vango.Keys([]string{"Enter", "NumpadEnter"}, func() {
    submit()
}))

OnKeyDown(vango.KeyWithModifiers("s", vango.Ctrl, func() {
    save()  // Ctrl+S
}))

OnKeyDown(vango.KeyWithModifiers("s", vango.Ctrl|vango.Shift, func() {
    saveAs()  // Ctrl+Shift+S
}))
```

---

### 3.9.4 Signal API


Signals are reactive values that trigger re-renders when changed.

#### Canonical Signatures

```go
func NewSignal[T any](initial T, opts ...SignalOption) Signal[T]
func NewSharedSignal[T any](initial T, opts ...SignalOption) Signal[T]
func NewGlobalSignal[T any](initial T, opts ...SignalOption) Signal[T]
```

**Normative**: All three constructors return the same `Signal[T]` type. The scope (per-instance, per-session, or global) is determined by the constructor call.

Signals expose methods like `Get`, `Set`, `Update`, and `Peek`.

#### Creating Signals

```go
// Basic signal with initial value
count := vango.NewSignal(0)                    // Signal[int]
name := vango.NewSignal("Alice")               // Signal[string]
user := vango.NewSignal[*User](nil)            // Signal[*User] with nil
items := vango.NewSignal([]Item{})             // Signal[[]Item]
prefs := vango.NewSignal(Preferences{})        // Signal[Preferences]

// Session-scoped signal (shared within a user session)
var CartItems = vango.NewSharedSignal([]CartItem{})

// Global signal (shared across ALL sessions)
var OnlineUsers = vango.NewGlobalSignal([]User{})
```

#### Reading Values

```go
// Call the signal to get current value
// This also subscribes the current component to changes
currentCount := count.Get()
userName := name.Get()

// Read without subscribing (rarely needed)
value := count.Peek()
```

#### Writing Values

```go
// Set new value
count.Set(5)
name.Set("Bob")

// Update with function
count.Update(func(n int) int {
    return n + 1
})

// For structs, use Update to avoid mutation
user.Update(func(u *User) *User {
    return &User{
        ID:   u.ID,
        Name: newName,
        Age:  u.Age,
    }
})
```

#### Convenience Methods

For numeric signals:
```go
count.Inc()              // Increment by 1
count.Dec()              // Decrement by 1
count.Add(5)             // Add value
count.Sub(3)             // Subtract value
count.Mul(2)             // Multiply
count.Div(2)             // Divide
```

For boolean signals:
```go
visible.Toggle()         // Flip true/false
visible.SetTrue()        // Set to true
visible.SetFalse()       // Set to false
```

For string signals:
```go
text.Append(" world")    // Append string
text.Prepend("Hello ")   // Prepend string
text.Clear()             // Set to ""
```

For slice signals:
```go
items.Append(newItem)                           // Add to end
items.Prepend(newItem)                          // Add to start
items.InsertAt(2, newItem)                      // Insert at index
items.RemoveAt(0)                               // Remove by index
items.RemoveLast()                              // Remove last
items.RemoveFirst()                             // Remove first
items.RemoveWhere(func(i Item) bool {           // Remove matching
    return i.Done
})
items.UpdateAt(0, func(i Item) Item {           // Update at index
    return Item{...i, Done: true}
})
items.UpdateWhere(                              // Update matching
    func(i Item) bool { return i.ID == id },
    func(i Item) Item { return Item{...i, Done: true} },
)
items.Clear()                                   // Remove all
items.SetAt(0, newItem)                         // Replace at index
```

For map signals:
```go
users.SetKey("123", user)                       // Set key
users.RemoveKey("123")                          // Remove key
users.UpdateKey("123", func(u User) User {      // Update key
    return User{...u, LastSeen: time.Now()}
})
users.HasKey("123")                             // Check key exists
users.Clear()                                   // Remove all
```

#### Signal Metadata

```go
// Check if signal has been modified
if count.IsDirty() {
    // Signal changed since last render
}

// Get subscriber count (debugging)
fmt.Println(count.SubscriberCount())

// Named signals (for debugging)
count := vango.NewSignal(0).Named("counter")
```

#### Durability & Persistence

Signals live on the server. Vango does **not** support synchronously initializing a signal from browser storage (e.g. `localStorage`) because the server has no access to it at component initialization time.

Instead, Vango provides **session durability**:
- **ResumeWindow**: refresh/reconnect restores in-memory session state within a grace period.
- **SessionStore (optional)**: sessions (including persisted signal values) survive server restarts and non-sticky deployments.

Signals are **persisted by default** when session serialization is enabled. Mark truly ephemeral UI state as transient, and use explicit keys when you need stable migrations/debugging.

```go
// Persisted by default (in-session, and to SessionStore if configured)
form := vango.NewSignal(FormData{})

// Not persisted (cursor positions, hover state, etc.)
cursor := vango.NewSignal(Point{0, 0}, vango.Transient())

// Stable key for serialized session data (recommended for important values)
userID := vango.NewSignal(0, vango.PersistKey("user_id"))
```

For **user preferences** (theme, sidebar, language) use `Pref` (see State Management) rather than signals.

#### Batching Updates

```go
// Multiple updates trigger single re-render
vango.Batch(func() {
    count.Set(5)
    name.Set("Bob")
    items.Append(newItem)
})
```

`Batch` is a compatibility alias for an anonymous Transaction (`Tx`). See State Management → Transactions & Snapshots for atomicity, snapshots, and cross-goroutine dispatch.

---

### 3.9.5 Memo API


Memos are cached computations that update when dependencies change.

#### Creating Memos

```go
// Basic memo
doubled := vango.NewMemo(func() int {
    return count.Get() * 2  // Re-runs when count changes
})

// Memo depending on multiple signals
fullName := vango.NewMemo(func() string {
    return firstName.Get() + " " + lastName.Get()
})

// Session-scoped memo
var CartTotal = vango.NewSharedMemo(func() float64 {
    total := 0.0
    for _, item := range CartItems.Get() {
        total += item.Price * float64(item.Qty)
    }
    return total
})

// Global memo
var ActiveUserCount = vango.NewGlobalMemo(func() int {
    return len(OnlineUsers.Get())
})
```

#### Reading Memos

```go
// Call to get cached value
value := doubled.Get()

// Read without subscribing
value := doubled.Peek()
```

#### Memo Chains

Memos can depend on other memos:

```go
var FilteredItems = vango.NewSharedMemo(func() []Item {
    return filterItems(Items.Get(), Filter.Get())
})

var SortedItems = vango.NewSharedMemo(func() []Item {
    return sortItems(FilteredItems.Get(), SortOrder.Get())
})

var PagedItems = vango.NewSharedMemo(func() []Item {
    items := SortedItems.Get()
    start := (Page.Get() - 1) * PageSize.Get()
    end := min(start + PageSize.Get(), len(items))
    return items[start:end]
})
```

#### Memo Options

```go
// Equality function (for complex types)
items := vango.NewMemo(func() []Item {
    return fetchItems()
}).Equals(func(a, b []Item) bool {
    return reflect.DeepEqual(a, b)
})

// Named memo (for debugging)
total := vango.NewMemo(func() float64 {
    return calculate()
}).Named("cart-total")
```

---

### 3.9.6 Effect API


Effects run after render and handle side effects.

> ### Prefer Structured Helpers Over Raw Effect
>
> Before writing a raw Effect with goroutines, use the appropriate structured helper:
>
> | Need | Use | Reference |
> |------|-----|-----------|
> | Async data loading | `Resource` | §3.9.7 |
> | User-triggered mutation | `Action` | §3.10.2 |
> | Periodic work (timers, polling) | `Interval` | §3.10.3.2 |
> | Event streams (WebSocket, SSE) | `Subscribe` | §3.10.3.3 |
> | Async work with reactive deps | `Effect` + `GoLatest` | §3.10.3.4 |
>
> Raw Effect is for pure wiring when none of the above fit. The helpers handle cancellation, stale suppression, dispatch, and transaction naming automatically.

Effects execute on the session event loop. Long-running or blocking work (database queries, HTTP calls, filesystem access) MUST be performed off the session loop (e.g. in a goroutine), and results MUST be applied via `ctx.Dispatch(...)` (see §7.0).

Effects are scheduled after commit; the runtime executes effect bodies on the session loop, so effect bodies must be short and must only schedule asynchronous work.

Effect enforcement (normative):
- Effects are “wiring-only” by default: synchronous signal writes during the Effect callback are treated as suspicious and are warned/panicked depending on configuration (see §3.10.4).
- Signal writes that occur inside `ctx.Dispatch(...)` callbacks (including those invoked by `Interval`, `Subscribe`, and `GoLatest`) are NOT effect-time writes and do not require opt-in.
- Use `AllowWrites()` to opt in for rare, intentional effect-body writes; use `EffectTxName(name)` to label effect diagnostics/telemetry (see §3.10.4).


Concurrency note (normative):
- `ctx.Dispatch(...)` MUST be safe to call from any goroutine.
- `ctx.StdContext()` MUST be safe to call from any goroutine (or safe to capture at effect start and use from goroutines).
- Other `ctx` methods (e.g. `Navigate`, `SetUser`, direct session access) MUST NOT be assumed goroutine-safe unless explicitly documented.

`ctx.StdContext()` semantics (normative):
- `ctx.StdContext()` MUST be derived from the current session “tick” context (the handler/effect execution context for the current transaction/commit cycle).
- `ctx.StdContext()` SHOULD carry tracing and any applicable deadlines/timeouts for the current tick.
- When an effect cleanup runs (dependency change or unmount), the runtime MUST run the returned cleanup. If the cleanup is a `cancel` function produced by `context.WithCancel(ctx.StdContext())`, cancellation MUST propagate to in-flight work via the derived context.

Cancelled vs stale (normative):
- **Cancelled** means “this work is no longer relevant because the effect was cleaned up (dependency changed or component unmounted)”; cancelled work MUST be ignored.
- **Stale** means “the dependency changed between starting work and applying results”; stale results MUST be ignored (typically via a `Peek()` guard).

Pedagogy note:
- If your dependency is a stable function parameter (e.g. `userID int` passed into a component instance), “stale” checks are usually unnecessary because a parameter change typically implies unmount/remount.
- If your dependency is a signal (e.g. `userID := vango.NewSignal(...)`), capture with `id := userID.Get()` and stale-check with `if userID.Peek() != id { return }`.

#### Creating Effects

```go
// Basic effect
vango.Effect(func() vango.Cleanup {
    fmt.Println("Component mounted")

    return func() {
        fmt.Println("Component unmounting")
    }
})

// Effect with dependencies (runs when dependencies change)
vango.Effect(func() vango.Cleanup {
    ctx := vango.UseCtx()
    fmt.Println("User changed to", userID.Get())

    // Tracked: re-run effect when userID changes.
    id := userID.Get()

    // Cancel any in-flight work when dependencies change/unmount.
    cctx, cancel := context.WithCancel(ctx.StdContext())

    go func(id int) {
        // Best-effort cancellation: if your DB supports context, pass cctx.
        u, err := db.Users.FindByID(cctx, id)
        if err != nil && cctx.Err() != nil {
            return // cancelled; ignore
        }

        ctx.Dispatch(func() {
            if cctx.Err() != nil {
                return // cancelled; ignore
            }
            if userID.Peek() != id {
                return // stale result; ignore
            }

            if err != nil {
                currentUser.Set(nil)
                return
            }
            currentUser.Set(u)
        })
    }(id)

    return cancel
})

// Prefer GoLatest (§3.10.3.4) for this exact pattern:
// it standardizes cancellation, stale suppression, dispatching, and transaction naming.

// If your I/O library does not accept a context, you can still ignore stale results safely:
vango.Effect(func() vango.Cleanup {
    ctx := vango.UseCtx()
    id := userID.Get() // tracked

    go func(id int) {
        u, err := db.Users.FindByID(id) // no ctx support
        ctx.Dispatch(func() {
            if userID.Peek() != id {
                return // stale; ignore
            }
            if err != nil {
                currentUser.Set(nil)
                return
            }
            currentUser.Set(u)
        })
    }(id)

    // No cancellation possible; stale suppression is still required.
    return nil
})

Informative: to reduce boilerplate, Vango (or your application) MAY provide a small helper to standardize stale-result guards (e.g., `GuardCurrent(sig, captured)`), but the core correctness requirement is simply: “compare `sig.Peek()` against the captured value before applying results.”

// Effect that runs only once (on mount)
vango.OnMount(func() {
    analytics.TrackPageView()
})

// Effect that runs on unmount only
vango.OnUnmount(func() {
    cleanup()
})
```

#### Effect Timing

| Lifecycle | Function | When It Runs |
|-----------|----------|--------------|
| Mount | `vango.OnMount(fn)` | After first render |
| Update | `vango.OnUpdate(fn)` | After each re-render |
| Unmount | `vango.OnUnmount(fn)` | Before component removes |
| Effect | `vango.Effect(fn)` | After render, re-runs on dep change |

```go
func Timer() vango.Component {
    return vango.Func(func() *vango.VNode {
        elapsed := vango.NewSignal(0)

        vango.OnMount(func() {
            fmt.Println("Timer started")
        })

        vango.Effect(func() vango.Cleanup {
            return vango.Interval(time.Second, elapsed.Inc, vango.IntervalTxName("timer:tick"))
        })

        vango.OnUnmount(func() {
            fmt.Println("Timer stopped")
        })

        return Div(Textf("Elapsed: %d seconds", elapsed.Get()))
    })
}
```

Signal writes are single-writer (session loop). Use `ctx.Dispatch(...)` for cross-goroutine updates; for periodic updates (timers/polling) prefer `vango.Interval(...)` (§3.10.3.2).

#### Effect Dependencies

Effects automatically track signal dependencies:

```go
vango.Effect(func() vango.Cleanup {
    // This effect re-runs when userID.Get() changes
    ctx := vango.UseCtx()
    id := userID.Get()

    cctx, cancel := context.WithCancel(ctx.StdContext())
    go func(id int) {
        // Best-effort cancellation: if your fetcher supports context, pass cctx.
        user, err := fetchUser(cctx, id)
        if err != nil && cctx.Err() != nil {
            return // cancelled; ignore
        }

        ctx.Dispatch(func() {
            if cctx.Err() != nil {
                return // cancelled; ignore
            }
            if userID.Peek() != id {
                return // stale; ignore
            }
            if err != nil {
                errorState.Set(err)
                return
            }
            userData.Set(user)
        })
    }(id)

    return cancel
})
```

#### Untracked Reads

To read a signal without creating a dependency:

```go
vango.Effect(func() vango.Cleanup {
    userId := userID.Get()  // Tracked - effect re-runs on change

    vango.Untracked(func() {
        config := globalConfig()  // Not tracked
        // Effect won't re-run when globalConfig changes
    })

    return nil
})
```

---

### 3.9.7 Resource API

Resources handle async data loading with loading/error/success states. **Normative**: `Resource[T]` is a pointer type (`*Resource[T]`); constructors return a stable pointer that persists across re-renders.

Resource is for async **queries** that should load automatically (on mount and/or when keys change). For async **mutations** that require explicit user intent (save, delete, submit), use `Action` (§3.10.2).

#### Canonical Signatures

```go
// Resource fetches once and is controlled by the component lifecycle.
func NewResource[T any](fetch func() (T, error), opts ...ResourceOption) *Resource[T]

// ResourceKeyed re-fetches when the key changes.
func NewResourceKeyed[K comparable, T any](key Signal[K], fetch func(K) (T, error), opts ...ResourceOption) *Resource[T]
```

Legacy note: earlier drafts used an overload-style shorthand `vango.Resource(key, fetch)`. Treat that as equivalent to `vango.NewResourceKeyed(key, fetch)`; only the explicit form is normative in this spec.

#### Creating Resources

```go
// Basic resource
user := vango.NewResource(func() (*User, error) {
    return db.Users.FindByID(userID)
})

// Resource with key (re-fetches when key changes)
user := vango.NewResourceKeyed(userID, func(id int) (*User, error) {
    return db.Users.FindByID(id)
})

// Multiple resources
type PageData struct {
    User     *User
    Projects []Project
}

data := vango.NewResource(func() (PageData, error) {
    user, err := db.Users.FindByID(userID)
    if err != nil {
        return PageData{}, err
    }
    projects, err := db.Projects.FindByUser(userID)
    if err != nil {
        return PageData{}, err
    }
    return PageData{User: user, Projects: projects}, nil
})
```

#### Resource States

```go
type ResourceState int

const (
    vango.Pending ResourceState = iota  // Not started
    vango.Loading                       // In progress
    vango.Ready                         // Success
    vango.Error                         // Failed
)
```

#### Resource Semantics (Normative)

- A `Resource` instance is component-scoped state. It MUST persist across re-renders of the same component instance (hook-order semantics; see §3.1.3).
- Fetch execution MUST NOT block the session loop. Resources MUST run fetch work off the session loop and apply results via `ctx.Dispatch(...)`.
- Fetch signature note: `Resource` fetchers are specified as `func() (T, error)`. If you need cancelable I/O, the closure SHOULD capture `ctx := vango.UseCtx()` and use `ctx.StdContext()` inside the fetch (e.g. `return fetch(ctx.StdContext())`). Cancellation is best-effort; stale-result suppression is required (below).
  - Design decision: the spec prefers closure capture to keep the public API surface minimal. Vango MAY add context-aware resource fetcher signatures in a future version.
- Start timing:
  - A resource MAY begin fetching immediately after the first render commit (mount).
  - A resource MUST NOT perform blocking work during render.
- Key changes / races:
  - If a keyed resource’s key changes while a fetch is in flight, the runtime MUST ensure stale results do not overwrite newer state.
  - Cancellation is best-effort: the runtime MAY cancel work if the fetcher is cancelable, but at minimum it MUST ignore stale results.
  - Cancelled vs stale: cancelled work (effect cleanup/unmount) MUST be ignored; stale results (key changed mid-flight) MUST be ignored.
- Retry:
  - `Retry(n, baseDelay)` MUST attempt up to `n` retries on failure.
  - Backoff SHOULD be exponential (`baseDelay * 2^attempt`) and SHOULD include jitter to avoid thundering herds.
  - Retries for a superseded key/fetch MUST be abandoned.
- Staleness:
  - `StaleTime(d)` defines the duration a Ready value is considered fresh.
  - `Fetch()` SHOULD be a no-op while data is fresh; `Refetch()` MUST bypass staleness.
- Dedupe/cache:
  - By default, resources are NOT deduped across component instances. If you need sharing, use an explicit shared store (`NewSharedSignal`/`NewSharedMemo`) or an application cache layer.
- `Mutate`:
  - `Mutate` MUST update the local cached value immediately (optimistic/local edits) and SHOULD reset freshness so the UI does not immediately refetch unless invalidated.
- Storm budgets (normative; see §3.10.5):
  - Resource starts MUST be subject to the session `StormBudget` configuration.
  - When throttled, a Resource MUST surface `ErrBudgetExceeded` via its Error state and MUST support `errors.Is(err, ErrBudgetExceeded)`.
  - Resource instances MUST support a per-resource override via `OnResourceBudgetExceeded(mode)` to change the exceeded behavior (throttle vs trip breaker).

#### Reading Resources

```go
// Pattern 1: Switch on state
switch user.State() {
case vango.Pending, vango.Loading:
    return LoadingSpinner()
case vango.Error:
    return ErrorMessage(user.Error())
case vango.Ready:
    return UserCard(user.Data())
}

// Pattern 2: Match helper
return user.Match(
    vango.OnLoading(func() *vango.VNode {
        return LoadingSpinner()
    }),
    vango.OnError(func(err error) *vango.VNode {
        return ErrorMessage(err)
    }),
    vango.OnReady(func(u *User) *vango.VNode {
        return UserCard(u)
    }),
)

// Pattern 3: With defaults
u := user.DataOr(&User{Name: "Loading..."})
return UserCard(u)
```

#### Resource Methods

```go
// Get current data (nil if not ready)
data := user.Data()

// Get data with default
data := user.DataOr(defaultValue)

// Get error (nil if no error)
err := user.Error()

// Get state
state := user.State()

// Manual refetch
user.Refetch()

// Mutate local data (optimistic update)
user.Mutate(func(u *User) *User {
    return &User{...u, Name: newName}
})

// Invalidate (marks stale, refetches)
user.Invalidate()
```

#### Resource Options

```go
// Initial data (show while loading)
user := vango.NewResource(fetchUser).InitialData(cachedUser)

// Stale time (how long data is considered fresh)
user := vango.NewResource(fetchUser).StaleTime(5 * time.Minute)

// Retry on error
user := vango.NewResource(fetchUser).Retry(3, 1*time.Second)

// On success/error callbacks
user := vango.NewResource(fetchUser).
    OnSuccess(func(u *User) {
        analytics.TrackUserLoad()
    }).
    OnError(func(err error) {
        logger.Error("Failed to load user", "error", err)
    })

// Storm budget override (see §3.10.5)
authCheck := vango.NewResource(fetchAuth,
    vango.OnResourceBudgetExceeded(vango.BudgetTripBreaker),
)
```

---

### 3.9.8 Ref API

Refs provide direct access to DOM elements or component instances.

**Client-runtime only**

`Ref[js.Value]` (and any “real DOM handle” ref) is only meaningful in **client runtimes** (WASM components and JavaScript islands). In server-driven mode, the server cannot hold DOM handles; use HIDs + patches for operations like focus/blur/scroll.

Implementation note: `js.Value` comes from `syscall/js`. Server builds SHOULD keep client-only code behind build tags so server-driven applications typecheck without the WASM/JS runtime.

#### Ref API Signature

**Normative**: `Ref[T]` is a pointer type (`*Ref[T]`); constructors return a stable pointer.

```go
func NewRef[T any](initial T) *Ref[T]
```

#### Creating Refs

```go
// DOM element ref
inputRef := vango.NewRef[js.Value](nil)

// Use in component
Input(
    Ref(inputRef),
    Type("text"),
)

// Access after mount
vango.OnMount(func() {
    inputRef.Current().Call("focus")
})
```

#### Ref Methods

```go
// Get current value
el := inputRef.Current()

// Check if set
if inputRef.IsSet() {
    // ...
}
```

#### Forward Refs

```go
// Child component that exposes a ref
func FancyInput(ref *vango.Ref[js.Value]) *vango.VNode {
    return Input(
        Ref(ref),
        Class("fancy"),
    )
}

// Parent usage
inputRef := vango.NewRef[js.Value](nil)
FancyInput(inputRef)

vango.OnMount(func() {
    inputRef.Current().Call("focus")
})
```

---

### 3.9.9 Helper Functions


#### Conditional Rendering

```go
// If: Render if condition true
If(isLoggedIn, UserMenu())

// IfElse: Render one or the other
IfElse(isLoggedIn, UserMenu(), LoginButton())

// When: Like If but lazy (closure)
When(isLoggedIn, func() *vango.VNode {
    return UserMenu()  // Only evaluated if true
})

// Unless: Render if condition false
Unless(isLoading, Content())

// Switch: Multiple conditions
Switch(status,
    Case("active", ActiveBadge()),
    Case("pending", PendingBadge()),
    Default(UnknownBadge()),
)
```

#### List Rendering

```go
// Range: Map slice to elements
Range(items, func(item Item, index int) *vango.VNode {
    return Li(Key(item.ID), Text(item.Name))
})

// RangeMap: Map map to elements
RangeMap(users, func(id string, user User) *vango.VNode {
    return Li(Key(id), Text(user.Name))
})

// Repeat: Render n times
Repeat(5, func(i int) *vango.VNode {
    return Star()
})
```

#### Fragment and Key

```go
// Fragment: Group without wrapper element
Fragment(
    Header(),
    Main(),
    Footer(),
)

// Key: Stable identity for reconciliation
Li(Key(item.ID), Text(item.Name))

// Key with multiple parts
Li(Key(item.Type, item.ID), Text(item.Name))
```

#### Text Helpers

```go
// Text: Static text node
Text("Hello, world!")

// Textf: Formatted text
Textf("Count: %d", count.Get())

// DangerouslySetInnerHTML: Unescaped HTML (use carefully!)
DangerouslySetInnerHTML("<strong>Bold</strong>")

// Raw(...) is a legacy alias for DangerouslySetInnerHTML(...).
```

#### Slots and Children

```go
// Slot: Named slot placeholder
func Layout(slots map[string]*vango.VNode) *vango.VNode {
    return Div(
        Header(slots["header"]),
        Main(slots["content"]),
        Footer(slots["footer"]),
    )
}

// Usage
Layout(map[string]*vango.VNode{
    "header":  H1(Text("Title")),
    "content": ArticleContent(),
    "footer":  Copyright(),
})

// Children: Variadic children
func Card(title string, children ...any) *vango.VNode {
    return Div(Class("card"),
        H2(Text(title)),
        Div(Class("card-body"), children...),
    )
}
```

#### Null and Empty

```go
// Null: Explicit nothing
func MaybeShow() *vango.VNode {
    if !shouldShow {
        return vango.Null()
    }
    return Content()
}

// Empty: Empty fragment
vango.Empty()  // Renders nothing
```

---

### 3.9.10 Context API

Context provides dependency injection through the component tree.

#### Creating Context

```go
// Define context with default value
var ThemeContext = vango.CreateContext("light")

// Context with type
var UserContext = vango.CreateContext[*User](nil)
```

#### Providing Values

```go
func App() vango.Component {
    return vango.Func(func() *vango.VNode {
        theme := vango.NewSignal("dark")

        return ThemeContext.Provider(theme.Get(),
            Header(),
            Main(),
            Footer(),
        )
    })
}
```

#### Consuming Values

```go
func Button() *vango.VNode {
    theme := ThemeContext.Use()  // "light" or "dark"

    return ButtonEl(
        Class("btn", theme+"-theme"),
        Text("Click"),
    )
}
```

#### Multiple Contexts

```go
func App() vango.Component {
    return vango.Func(func() *vango.VNode {
        user := getCurrentUser()
        theme := "dark"
        locale := "en"

        return UserContext.Provider(user,
            ThemeContext.Provider(theme,
                LocaleContext.Provider(locale,
                    Router(),
                ),
            ),
        )
    })
}
```

---

### 3.9.11 Form API

Vango provides a structured form system for complex forms with validation.

#### UseForm Hook

```go
// Define form structure
type ContactForm struct {
    Name    string `form:"name" validate:"required,min=2"`
    Email   string `form:"email" validate:"required,email"`
    Message string `form:"message" validate:"required,max=1000"`
}

func ContactPage() vango.Component {
    return vango.Func(func() *vango.VNode {
        form := vango.UseForm(ContactForm{})

        submit := func() {
            if !form.Validate() {
                return  // Errors shown automatically
            }
            sendEmail(form.Values())
            form.Reset()
        }

        return Form(OnSubmit(submit),
            form.Field("Name", Input(Type("text"))),
            form.Field("Email", Input(Type("email"))),
            form.Field("Message", Textarea()),
            Button(Type("submit"), Text("Send")),
        )
    })
}
```

#### Form Methods

```go
form := vango.UseForm(MyForm{})

// Read values
values := form.Values()           // MyForm struct
value := form.Get("fieldName")    // Single field value

// Write values
form.Set("fieldName", value)      // Set single field
form.SetValues(MyForm{...})       // Set all values
form.Reset()                      // Reset to initial

// Validation
isValid := form.Validate()        // Run all validators
errors := form.Errors()           // map[string][]string
fieldErrs := form.FieldErrors("name")  // []string
hasError := form.HasError("name") // bool
form.ClearErrors()                // Clear all errors

// State
form.IsDirty()                    // Has any field changed
form.FieldDirty("name")           // Has specific field changed
form.IsSubmitting()               // Currently submitting
form.SetSubmitting(true)          // Set submitting state
```

#### Field Method

```go
// form.Field returns a configured input with:
// - Value binding
// - Error display
// - Validation on blur

form.Field("Email",
    Input(Type("email"), Placeholder("you@example.com")),
    vango.Required("Email is required"),
    vango.Email("Invalid email format"),
    vango.MaxLength(100, "Email too long"),
)

// Renders as:
// <div class="field">
//   <input type="email" name="email" value="..." />
//   <span class="error">Invalid email format</span>
// </div>
```

#### Built-in Validators

```go
// String validators
vango.Required("Field is required")
vango.MinLength(n, "Too short")
vango.MaxLength(n, "Too long")
vango.Pattern(regex, "Invalid format")
vango.Email("Invalid email")
vango.URL("Invalid URL")
vango.UUID("Invalid UUID")

// Numeric validators
vango.Min(n, "Too small")
vango.Max(n, "Too large")
vango.Between(min, max, "Out of range")
vango.Positive("Must be positive")
vango.NonNegative("Must be non-negative")

// Comparison validators
vango.EqualTo("password", "Passwords must match")
vango.NotEqualTo("oldPassword", "Must be different")

// Date validators
vango.DateAfter(time, "Must be after...")
vango.DateBefore(time, "Must be before...")
vango.Future("Must be in the future")
vango.Past("Must be in the past")

// Custom validator
vango.Custom(func(value any) error {
    if !isUnique(value.(string)) {
        return errors.New("Already taken")
    }
    return nil
})

// Async validator
vango.Async(func(value any) (error, bool) {
    // Returns (error, isComplete)
    // isComplete=false while loading
    available := checkUsername(value.(string))
    if !available {
        return errors.New("Username taken"), true
    }
    return nil, true
})
```

#### Form Arrays

```go
type OrderForm struct {
    CustomerName string
    Items        []OrderItem
}

type OrderItem struct {
    ProductID int
    Quantity  int
}

func OrderFormPage() vango.Component {
    return vango.Func(func() *vango.VNode {
        form := vango.UseForm(OrderForm{})

        addItem := func() {
            form.AppendTo("Items", OrderItem{})
        }

        return Form(
            form.Field("CustomerName", Input(Type("text"))),

            // Render array items
            form.Array("Items", func(item vango.FormArrayItem, i int) *vango.VNode {
                return Div(Class("item-row"),
                    item.Field("ProductID", ProductSelect()),
                    item.Field("Quantity", Input(Type("number"))),
                    Button(OnClick(item.Remove), Text("Remove")),
                )
            }),

            Button(OnClick(addItem), Text("Add Item")),
            Button(Type("submit"), Text("Submit Order")),
        )
    })
}
```

---

### 3.9.12 URLParam (Query State)

Synchronize reactive state with **URL query parameters** (not path params). This enables shareable URLs, back-button friendly filters/search, and SSR-friendly state hydration.

#### Canonical Signatures

```go
func URLParam[T any](key string, def T, opts ...URLParamOption) Signal[T]
```

`URLParam` is scoped to the current route and session. It MUST be called during render/effect/handler execution (i.e. when `vango.UseCtx()` is available).

#### Basic Usage

```go
func ProductList(ctx vango.Ctx) vango.Component {
    return vango.Func(func() *vango.VNode {
        // Search: replace history + debounce to avoid back-button spam
        search := vango.URLParam("q", "", vango.Replace, vango.Debounce(300*time.Millisecond))

        // Page: default is Push (history entries are often desirable for pagination)
        page := vango.URLParam("page", 1)

        return Div(
            Input(
                Type("search"),
                Value(search.Get()),
                OnInput(search.Set),
            ),
            Pagination(page.Get(), page.Set),
        )
    })
}
```

#### History Mode: `vango.Push` vs `vango.Replace`

URLParam mode options are **values** (not functions) to avoid collisions with navigation methods:

```go
search := vango.URLParam("q", "", vango.Replace)
page := vango.URLParam("page", 1, vango.Push) // explicit (default is Push)
```

#### Complex Types via Encoding

```go
type Filters struct {
    Category string `url:"cat"`
    SortBy   string `url:"sort"`
    Page     int    `url:"page"`
}

// Flat encoding: ?cat=electronics&sort=price&page=1
filters := vango.URLParam("", Filters{}, vango.Encoding(vango.URLEncodingFlat))

// JSON encoding: ?filter=eyJjYXQiOiJlbGVjdHJvbmljcyIsInNvcnQiOiJwcmljZSJ9 (compressed/base64)
filters := vango.URLParam("filter", Filters{}, vango.Encoding(vango.URLEncodingJSON))

// Comma encoding: ?tags=go,web,api
tags := vango.URLParam("tags", []string{}, vango.Encoding(vango.URLEncodingComma))
```

#### Path Params vs Query Params

- **Path params** (`/projects/{id}`) come from the router: `ctx.Param("id")`
- **Query params** (`?tab=settings`) are managed by `vango.URLParam(...)`

#### Protocol & Navigation Distinction

- `vango.URLParam(...)` updates **query params on the current route** via small URL-only patches.
- `ctx.Navigate("/path")` changes the **route** (path + optional query) and includes DOM patches in the navigation envelope.

---

## 3.10 Structured Side Effects

This section defines Vango’s structured side-effect model: one obvious, hard-to-misuse answer for each common task. It is **normative**.

### 3.10.1 Selection Guide

Use the narrowest primitive that matches your intent:

```
Need to load data on mount or when a key changes?
  └─→ Use Resource (§3.9.7)

Need to perform a user-triggered mutation (save, delete, submit)?
  └─→ Use Action (§3.10.2)

Need periodic updates (polling, timers)?
  └─→ Use Effect + Interval (§3.10.3.2)

Need to react to an event stream (WebSocket, SSE, pub/sub)?
  └─→ Use Effect + Subscribe (§3.10.3.3)

Need async work triggered by reactive state that doesn't fit above?
  └─→ Use Effect + GoLatest (§3.10.3.4)

Need pure wiring (no async, just connecting things)?
  └─→ Use Effect (§3.9.6), with enforcement rules (§3.10.4)
```

### 3.10.2 Action API

Action is the structured primitive for async **mutations**, complementing Resource (async **queries**).

#### 3.10.2.1 Type Definitions

```go
package vango

type ActionState int

const (
    ActionIdle ActionState = iota
    ActionRunning
    ActionSuccess
    ActionError
)

type Action[A any, R any] struct {
    // unexported fields
}

func (a *Action[A, R]) Run(arg A) (accepted bool)
func (a *Action[A, R]) State() ActionState
func (a *Action[A, R]) Result() (R, bool)
func (a *Action[A, R]) Error() error
func (a *Action[A, R]) Reset()
```

#### 3.10.2.2 Constructor

```go
func NewAction[A any, R any](
    do func(ctx context.Context, a A) (R, error),
    opts ...ActionOption,
) *Action[A, R]
```

**Normative**:
- `NewAction` MUST be called during render, effect, or handler execution (i.e. when `UseCtx()` is valid).
- `NewAction` returns a stable pointer that persists across re-renders for a given component instance. Like other render-time stateful primitives, it MUST obey hook-order semantics (§3.1.3).
- Options are applied at creation time. If subsequent renders invoke `NewAction` at the same hook slot with different options, the runtime MUST ignore subsequent option changes and SHOULD emit a dev-mode warning.

#### 3.10.2.3 Options

```go
type ActionOption interface{ isActionOption() }

// Concurrency policies
func CancelLatest() ActionOption        // Cancel prior in-flight on new Run
func DropWhileRunning() ActionOption    // Ignore Run while Running
func Queue(max int) ActionOption        // Buffer up to max, execute sequentially

// Naming and observability
func ActionTxName(name string) ActionOption
func OnActionStart(fn func()) ActionOption
func OnActionSuccess[R any](fn func(R)) ActionOption
func OnActionError(fn func(error)) ActionOption

// Storm budget override (§3.10.5)
func OnActionBudgetExceeded(mode BudgetExceededMode) ActionOption
```

#### 3.10.2.4 Normative Semantics

1. **Off-loop execution**: The `do` function MUST execute off the session event loop.
2. **On-loop state transitions**: All Action state transitions (Idle→Running, Running→Success, Running→Error) MUST be applied on the session loop via `ctx.Dispatch(...)`, executing inside an implicit transaction.
3. **Concurrency policies**:
   - `CancelLatest()`: Calling `Run` while Running MUST cancel the prior in-flight call (via context cancellation) before starting the new one. Returns `true`.
   - `DropWhileRunning()`: Calling `Run` while Running MUST be a no-op. Returns `false`.
   - `Queue(max)`: Calls to `Run` while Running MUST be queued up to `max`. Returns `true` if queued successfully, `false` if queue is full. Exceeding `max` MUST also set the Action to `ActionError` with `ErrQueueFull`.
4. **Default policy**: If no concurrency policy is specified, the default MUST be `CancelLatest()`.
5. **Multiple policies**: At most one concurrency policy MAY be specified. If multiple are provided, the runtime MUST panic in dev mode and MUST deterministically select the first policy in production while emitting telemetry.
6. **Run return value**: `Run` MUST return `true` if the call was accepted (started or queued), `false` if rejected (dropped due to `DropWhileRunning` or queue full).
7. **State transitions**: Calling `Run` from any state transitions the Action to `ActionRunning` (subject to concurrency policy). Success MUST set `ActionSuccess` and overwrite stored result; failure MUST set `ActionError` and overwrite stored error. `Reset()` MUST set `ActionIdle` and clear stored result/error.
8. **Transaction naming**: State transitions MUST appear as named transactions in DevTools/logging:
   - With `ActionTxName("profile:save")`: `Action:profile:save:running`, `Action:profile:save:success`, etc.
   - Without a name, the runtime SHOULD use a stable call-site based identifier (source location in dev; component identity in prod).

#### 3.10.2.5 Example

```go
func ProfileEditor() vango.Component {
    return vango.Func(func() *vango.VNode {
        profile := vango.NewSignal(Profile{})

        save := vango.NewAction(
            func(ctx context.Context, p Profile) (Profile, error) {
                return api.SaveProfile(ctx, p)
            },
            vango.CancelLatest(),
            vango.ActionTxName("profile:save"),
            vango.OnActionSuccess(func(p Profile) {
                profile.Set(p)
                toast.Success(vango.UseCtx(), "Profile saved")
            }),
            vango.OnActionError(func(err error) {
                toast.Error(vango.UseCtx(), "Failed to save: "+err.Error())
            }),
        )

        return Form(
            OnSubmit(func() {
                save.Run(profile.Get())
            }),
            Button(
                Type("submit"),
                Disabled(save.State() == vango.ActionRunning),
                IfElse(save.State() == vango.ActionRunning,
                    Text("Saving..."),
                    Text("Save"),
                ),
            ),
        )
    })
}
```

### 3.10.3 Effect Helpers

Effect helpers standardize common Effect patterns, handling cleanup, dispatch, and transaction naming automatically.

#### 3.10.3.1 Call-Site and Lifetime Semantics

**Normative**: `Interval`, `Subscribe`, and `GoLatest` SHOULD be called inside an Effect (or any lifecycle hook that accepts cleanup), and their returned Cleanup SHOULD be returned from that Effect.

Canonical pattern:
```go
vango.Effect(func() vango.Cleanup {
    return vango.Interval(time.Second, tick)
})
```

Anti-pattern (leak):
```go
vango.Effect(func() vango.Cleanup {
    vango.Interval(time.Second, tick) // cleanup dropped → leak
    return nil
})
```

#### 3.10.3.2 Interval

```go
func Interval(
    d time.Duration,
    fn func(),
    opts ...IntervalOption,
) Cleanup

type IntervalOption interface{ isIntervalOption() }
func IntervalTxName(name string) IntervalOption
func IntervalImmediate() IntervalOption // first tick occurs immediately
```

Normative semantics:
1. `Interval` MUST schedule ticks off the session loop.
2. Each tick MUST invoke `fn` on the session loop via `ctx.Dispatch(fn)`.
3. The returned Cleanup MUST stop future ticks and MUST be called when the owning component unmounts.
4. By default, the first tick MUST occur after `d`. With `IntervalImmediate()`, the first tick MUST occur as soon as possible (still dispatched through the session loop).
5. With `IntervalTxName("heartbeat")`, tick work MUST appear as `Interval:heartbeat` in transaction naming.

#### 3.10.3.3 Subscribe

```go
type Stream[T any] interface {
    Subscribe(handler func(T)) (unsubscribe func())
}

func Subscribe[T any](
    stream Stream[T],
    fn func(T),
    opts ...SubscribeOption,
) Cleanup

type SubscribeOption interface{ isSubscribeOption() }
func SubscribeTxName(name string) SubscribeOption
```

Normative semantics:
1. Upon receiving a message, `Subscribe` MUST invoke `fn(msg)` on the session loop via `ctx.Dispatch`.
2. The returned Cleanup MUST unsubscribe from the stream.
3. With `SubscribeTxName("chat:message")`, message handling MUST appear as `Subscribe:chat:message` in transaction naming.

#### 3.10.3.4 GoLatest

`GoLatest` is the standard helper for async integration work inside Effect when Resource or Action do not fit.

```go
func GoLatest[K comparable, R any](
    key K,
    work func(ctx context.Context, key K) (R, error),
    apply func(result R, err error),
    opts ...GoLatestOption,
) Cleanup

type GoLatestOption interface{ isGoLatestOption() }
func GoLatestTxName(name string) GoLatestOption
func GoLatestForceRestart() GoLatestOption // restart even when key unchanged
func OnGoLatestBudgetExceeded(mode BudgetExceededMode) GoLatestOption
```

Normative semantics:
1. **Key coalescing (default)**: If invoked with a key equal to the previous invocation at the same call site, the runtime MUST NOT cancel, restart, or start new work. If `GoLatestForceRestart()` is provided, the runtime MUST cancel any in-flight work and MUST start new work even when keys are equal.
2. **Cancel-latest (different keys)**: A new key MUST cancel prior in-flight work for that call site via context cancellation.
3. **Off-loop execution**: `work` MUST execute off the session loop.
4. **On-loop apply**: `apply` MUST be invoked on the session loop via `ctx.Dispatch(...)`.
5. **Stale suppression**: If the key changes while work is in flight, `apply` MUST NOT be invoked for the stale result.
6. **Cleanup**: The returned Cleanup MUST cancel in-flight work and prevent future applies.

Telemetry note (normative): Any telemetry that includes GoLatest keys MUST redact those keys by default (e.g., hash). Raw keys MUST NOT be logged unless explicitly enabled via `DebugConfig.LogRawKeys`, which MUST default to `false` in production.

### 3.10.4 Effect Enforcement

Effect is a necessary escape hatch for wiring and integration, but is “wiring-only” by default.

#### 3.10.4.1 Options

```go
type EffectOption interface{ isEffectOption() }
func AllowWrites() EffectOption
func EffectTxName(name string) EffectOption
```

`EffectTxName(name)` sets the identifier used in Effect warnings/telemetry and DevTools entries related to that Effect. It does not propagate to helper transactions (Interval/Subscribe/GoLatest), which use their own `*TxName` options or fall back to default naming.

#### 3.10.4.2 Strictness Configuration

```go
type StrictEffectMode int

const (
    StrictEffectOff StrictEffectMode = iota
    StrictEffectWarn
    StrictEffectPanic
)

type EffectConfig struct {
    Mode StrictEffectMode
}
```

#### 3.10.4.3 Normative Semantics

1. An “effect-time write” is a signal mutation performed synchronously during execution of the Effect callback on the session loop, before the callback returns. Mutations occurring inside goroutines, or inside `ctx.Dispatch(...)` callbacks, are not effect-time writes.
2. Effect-time writes include all signal mutation APIs (`Set`, `Update`, `Inc`, `Toggle`, slice/map writers, etc.).
3. In `StrictEffectWarn` mode, effect-time writes without `AllowWrites()` MUST emit a warning. In `StrictEffectPanic` mode, they MUST panic. With `AllowWrites()`, they MUST be allowed.
4. Even when `AllowWrites()` is set, the runtime SHOULD capture telemetry that the effect performed writes.

Default strictness (normative):
- Development: `StrictEffectWarn`
- Production: `StrictEffectOff` (unless explicitly configured)

### 3.10.5 Storm Budgets

Budgets protect the server from amplification bugs (e.g., effects re-running excessively and starting repeated I/O).

#### 3.10.5.1 Configuration

```go
type BudgetExceededMode int

const (
    BudgetThrottle BudgetExceededMode = iota // Default
    BudgetTripBreaker
)

type StormBudgetConfig struct {
    // Per-window start limits (0 = unlimited)
    MaxResourceStartsPerSecond int
    MaxActionStartsPerSecond   int
    MaxGoLatestStartsPerSecond int

    // Per-tick limits
    MaxEffectRunsPerTick int

    // Throttle window duration (default: 1s if zero)
    WindowDuration time.Duration

    // Default behavior when exceeded
    OnExceeded BudgetExceededMode
}
```

#### 3.10.5.2 Normative Semantics

1. When budgets are exceeded, the runtime MUST deny or delay new starts and MUST surface an error through the relevant primitive:
   - `Resource`: transitions to Error state with `ErrBudgetExceeded`.
   - `Action`: transitions to Error state with `ErrBudgetExceeded`; `Run` returns `false`.
   - `GoLatest`: MUST NOT start `work`; MUST invoke `apply(zero, ErrBudgetExceeded)` at most once per throttle window per call site.
2. In `BudgetTripBreaker` mode, the runtime MUST terminate the session or transition it to a “session invalidated” state.
3. `Resource`, `Action`, and `GoLatest` MUST support per-primitive overrides (`OnResourceBudgetExceeded`, `OnActionBudgetExceeded`, `OnGoLatestBudgetExceeded`).
4. Budget telemetry SHOULD include: budget type, limit, current count, call-site name, action taken, and whether an error was surfaced.

#### 3.10.5.3 Sentinel Errors

```go
var ErrBudgetExceeded = errors.New("vango: storm budget exceeded")
var ErrQueueFull = errors.New("vango: action queue full")
```

### 3.10.6 Naming and Privacy

Transaction naming and telemetry are core DX and observability concerns.

#### 3.10.6.1 Naming Rules

All helpers and Action state transitions MUST emit named transactions according to these rules:

| Primitive | With TxName option | Default (dev) | Default (prod) |
|-----------|-------------------|---------------|----------------|
| `Action` | `Action:<name>:<state>` | `Action:<file>:<line>:<state>` | `Action:<component>:<instance>:<state>` |
| `Interval` | `Interval:<name>` | `Interval:<file>:<line>` | `Interval:<component>:<instance>` |
| `Subscribe` | `Subscribe:<name>` | `Subscribe:<file>:<line>` | `Subscribe:<component>:<instance>` |
| `GoLatest` | `GoLatest:<name>` | `GoLatest:<file>:<line>` | `GoLatest:<component>:<instance>` |

#### 3.10.6.2 Debug and Privacy Controls

```go
type DebugConfig struct {
    // Include file:line in transaction names (default: true in dev, false in prod)
    IncludeSourceLocations bool

    // Log unhashed keys in GoLatest telemetry (default: false; MUST be false in prod)
    LogRawKeys bool
}
```

## 4. The Server-Driven Runtime

### 4.1 Session Management

Each browser tab creates a WebSocket connection with its own session:

```go
// Internal session structure (simplified)
type Session struct {
    ID          string
    Conn        *websocket.Conn
    Signals     map[uint32]*SignalBase    // All signals for this session
    Components  map[uint32]*ComponentInst // Mounted component instances
    LastTree    *vdom.VNode               // For diffing
    Handlers    map[string]func()         // hid → handler
    CreatedAt   time.Time
    LastActive  time.Time
}
```

**Session Lifecycle:**
```
1. WebSocket handshake (validates origin/CSRF, resumes or creates session)
2. Initial render (components mount, effects run)
3. Interaction loop (events → updates → patches)
4. Disconnect (session becomes *detached* for `ResumeWindow`)
5. Reconnect within window (session resumes; server re-syncs UI)
6. Resume window expires (session evicted; subsequent reconnect starts fresh)
```


#### Session States

Vango explicitly models session state to support refreshes and flaky networks:

- **Connected**: WebSocket is active.
- **Detached**: WebSocket dropped, but state is retained for `ResumeWindow`.
- **Expired**: Session is evicted (or cannot be restored from store).


#### Session Durability

Vango provides two layers of durability:

1. **ResumeWindow** (in-memory): refresh/reconnect restores state without a full reload.
2. **SessionStore** (optional): serialize session state so it can survive server restarts and (eventually) non-sticky deployments.

```go
app := vango.New(vango.Config{
    Session: vango.SessionConfig{
        ResumeWindow: 30 * time.Second,

        // Optional: Persist detached sessions / restart recovery
        Store: vango.RedisStore(redisClient),

        // Memory protection (DoS hardening)
        MaxDetachedSessions: 10000,
        MaxSessionsPerIP:    100,
        EvictionPolicy:      vango.EvictionLRU,

        // Storm budgets (see §3.10.5)
        StormBudget: vango.StormBudgetConfig{
            MaxResourceStartsPerSecond: 100,
            MaxActionStartsPerSecond:   50,
            MaxGoLatestStartsPerSecond: 100,
            MaxEffectRunsPerTick:       50,
            WindowDuration:             time.Second,
            OnExceeded:                 vango.BudgetThrottle,
        },
    },
})
```


#### What Gets Persisted

Session serialization persists **signal values** (and other session values) that are JSON-serializable. Use `vango.Transient()` for ephemeral state and `vango.PersistKey(...)` when you need stable keys across deployments (see Signals).

#### SessionStore Interface

Session persistence is implemented via a pluggable store interface:

```go
type SessionStore interface {
    Save(ctx context.Context, sessionID string, data []byte, expiresAt time.Time) error
    Load(ctx context.Context, sessionID string) ([]byte, error)      // (nil, nil) = not found/expired
    Delete(ctx context.Context, sessionID string) error
    Touch(ctx context.Context, sessionID string, expiresAt time.Time) error
    SaveAll(ctx context.Context, sessions map[string]SessionData) error
    Close() error
}
```


### 4.2 The Event Loop


```go
// Simplified server event loop (per session)
func (s *Session) eventLoop() {
    for {
        select {
        case event := <-s.events:
            // Find and run handler
            handler := s.Handlers[event.HID]
            if handler != nil {
                handler()
            }

            // Re-render affected components
            s.renderDirtyComponents()

            // Diff and send patches
            patches := vdom.Diff(s.LastTree, s.CurrentTree)
            s.sendPatches(patches)
            s.LastTree = s.CurrentTree

        case <-s.done:
            return
        }
    }
}
```

The event loop runs each handler (and other scheduled work) inside an implicit Transaction (`Tx`) and commits once per logical action. See State Management → Transactions & Snapshots for the Tx-aware pipeline and bounded stabilization behavior.

### 4.3 Binary Protocol Overview

> **Note:** This section provides a **simplified conceptual overview**. For the complete and canonical protocol specification including all event types, patch types, and their numeric assignments, see **§22 Appendix: Protocol Specification**.

The protocol is optimized for minimal bandwidth:

**Handshake framing**

The initial WebSocket handshake is sent as a UTF-8 JSON **text frame**. After `HANDSHAKE_ACK`, the connection switches to **binary frames** for all events and patches.

**Client → Server (Events):**
```
┌─────────┬──────────────┬─────────────────┐
│ Type    │ HID          │ Payload         │
│ 1 byte  │ varint       │ varies          │
└─────────┴──────────────┴─────────────────┘
```

See §22.2 for the complete event type registry. Common events include CLICK, INPUT, SUBMIT, KEYDOWN, SCROLL, NAVIGATE, and CUSTOM (for hooks and islands).

**Server → Client (Patches):**
```
┌─────────────┬───────────────────────────────┐
│ Patch Count │ Patches...                    │
│ varint      │                               │
└─────────────┴───────────────────────────────┘

Each Patch:
┌─────────┬──────────────┬─────────────────┐
│ Type    │ Target HID   │ Payload         │
│ 1 byte  │ varint       │ varies          │
└─────────┴──────────────┴─────────────────┘
```

See §22.3 for the complete patch type registry. Common patches include SET_TEXT, SET_ATTR, REMOVE_ATTR, ADD_CLASS, REMOVE_CLASS, INSERT_BEFORE, APPEND_CHILD, REMOVE_NODE, REPLACE_NODE, and URL_PUSH/URL_REPLACE for query state.

### 4.4 Hydration IDs

Every interactive element gets a hydration ID during SSR:

```html
<!-- Server-rendered HTML -->
<div class="counter">
    <h1 data-hid="h1">Count: 5</h1>
    <button data-hid="h2">+</button>
    <button data-hid="h3">-</button>
</div>
```

The mapping is stored server-side:
```go
session.Handlers["h2"] = count.Inc  // + button
session.Handlers["h3"] = count.Dec  // - button
```

When the button is clicked, the client sends `{type: CLICK, hid: 2}` (wire format: `[0x01][0x02]` — event type + varint HID), and the server runs the mapped handler.

#### 4.4.1 HID assignment and encoding

- HIDs are assigned during SSR render and emitted as `data-hid` attributes.
- Within a session, a HID MUST uniquely identify one live DOM node at a time.
- The server MUST use the same HID when sending patches targeting that node.

**HID format (normative):**
- **DOM attribute:** `data-hid="h<decimal>"` — e.g., `data-hid="h42"`. The `h` prefix is for human readability.
- **Wire protocol:** HIDs are encoded as **unsigned varints** (per §22.5) representing the numeric portion only. For example, `data-hid="h42"` becomes varint `42` on the wire.
- **Bijection:** The client MUST parse `data-hid` by stripping the `h` prefix and interpreting the remainder as a decimal integer. The server MUST emit the numeric HID in the wire protocol.

#### 4.4.2 Stability, keys, and list moves

VDOM diffing and patching assumes stable identity. In dynamic lists, **keys** are the identity mechanism:

- If a node is rendered with `Key(...)`, its HID SHOULD remain stable for that keyed identity, even if it moves positions in the list.
- If a list is rendered without keys, identity becomes positional and HIDs MAY be reassigned across renders, increasing the chance of incorrect patches during inserts/moves.

Rule of thumb: when rendering collections, use `Key(...)` to preserve identity and enable correct `MOVE_NODE`/reorder patches.

#### 4.4.3 Missing HID and self-healing

If the client cannot find a patch target HID in the DOM (or detects a structural mismatch), it MUST treat the connection as out of sync and recover by performing a full reload to the current URL (or an equivalent full resync mechanism).

### 4.5 Component Mounting

```go
// When a route matches, the page component mounts
func (s *Session) mountPage(route Route, params Params) {
    // Create component instance
    component := route.Component(params)

    // Set up signal scope
    s.currentComponent = component.ID

    // Run the component function (creates signals, effects)
    vtree := component.Render()

    // Collect handlers
    s.collectHandlers(vtree)

    // Initial render to HTML (for SSR)
    html := s.renderToHTML(vtree)

    // Store for future diffs
    s.LastTree = vtree
}
```

---

## 5. The Thin Client

### 5.1 Responsibilities

The thin client (~12KB gzipped) handles:

1. **WebSocket Connection** — Connect, reconnect, heartbeat
2. **Event Capture** — Click, input, submit, keyboard, etc.
3. **Patch Application** — Apply DOM updates from server
4. **Optimistic Updates** — Optional client-side predictions

It does NOT handle:
- Component logic
- State management
- Routing decisions
- Data fetching

### 5.2 Core Implementation

The following snippet is **illustrative** (not normative) and intentionally omits many edge cases.

Normative client behavior:
- The client SHOULD preserve native browser behavior unless a Vango handler will handle the action.
- The client MUST NOT blanket-`preventDefault()` on all clicks; for example, normal links, text selection, focus behavior, and modifier-clicks must continue to work.
- For progressive enhancement, when the WebSocket is unavailable, forms and links MUST fall back to normal HTTP navigation/submission.

#### Event Interception Decision Table

The thin client SHOULD use the following decision table when deciding whether to intercept a browser event and send a Vango event.

**Click / Navigate interception (left click)**

| Condition | Intercept? | Notes |
|----------|------------|-------|
| `e.defaultPrevented` | No | Respect other handlers. |
| `e.button != 0` (right/middle click) | No | Preserve context menus and middle-click. |
| Any modifier key held (`Ctrl/Meta/Shift/Alt`) | No | Preserve “open in new tab/window”, multi-select, etc. |
| Element is `<a>` with `target != "" && target != "_self"` | No | Browser controls windowing. |
| Element is `<a>` with `download` | No | Preserve download behavior. |
| Element is `<a>` whose resolved URL is cross-origin | No | Do not hijack external navigation. |
| WebSocket not connected/healthy | No | Progressive enhancement fallback. |
| No Vango handler exists for this element + event | No | Do not change native behavior. |
| Otherwise | Yes | Call `preventDefault()` and send the event. |

**How the client knows a handler exists**

Elements that have server-handled events SHOULD include a `data-ve` attribute (comma-separated) listing the event types they handle (e.g. `data-ve="click,submit,input"`). The client SHOULD only intercept events that are present in `data-ve`.

**Thin client attribute schema (Normative)**

- `data-hid="<id>"`: hydration/handler id used by the binary protocol to target handlers and DOM patches.
- If an element has `data-ve`, it MUST also have `data-hid`.
- Purely static nodes (no events, no dynamic content that requires patching) MAY omit `data-hid`.
- Any element that is a target of a DOM patch MUST have a stable `data-hid`.
- `data-ve="..."`: comma-separated list of server-handled event names for this element. `data-hid` alone MUST NOT imply interception. `data-ve` is used for client-side interception decisions; it does not restrict the binary protocol from sending other event types (e.g. `CUSTOM` or `HOOK_EVENT`).
  - Canonical event names for `data-ve`: `click`, `input`, `change`, `submit`, `keydown`, `keyup`, `focus`, `blur`, `scroll`, `hook`.
  - The `hook` capability name corresponds to `CUSTOM`/`HOOK_EVENT` types in the binary protocol.
- `data-optimistic='{"...": "..."}'`: optional JSON payload describing an optimistic UI change to apply immediately on intercept, before sending the event.

```javascript
// Simplified thin client (~200 lines total)
class VangoClient {
    constructor() {
        this.ws = null;
        this.reconnectAttempts = 0;
        this.connect();
        this.attachEventListeners();
    }

    connect() {
        this.ws = new WebSocket(`wss://${location.host}/_vango/live`);
        this.ws.binaryType = 'arraybuffer';

        this.ws.onopen = () => {
            this.reconnectAttempts = 0;
            this.sendHandshake();
        };

        this.ws.onmessage = (e) => {
            this.handleMessage(new Uint8Array(e.data));
        };

        this.ws.onclose = () => {
            this.scheduleReconnect();
        };
    }

    attachEventListeners() {
        // Click events
        document.addEventListener('click', (e) => {
            const el = e.target.closest('[data-hid]');
            if (!el) return;
            if (e.defaultPrevented) return;
            if (e.button !== 0) return;
            if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
            if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;

            const ve = (el.dataset.ve || '').split(',').map(s => s.trim()).filter(Boolean);
            if (!ve.includes('click')) return;

            // Do not hijack special anchor behavior.
            if (el.tagName === 'A') {
                const target = el.getAttribute('target');
                if (target && target !== '_self') return;
                if (el.hasAttribute('download')) return;

                const href = el.getAttribute('href');
                if (href) {
                    const url = new URL(href, location.href);
                    if (url.origin !== location.origin) return;
                }
            }

            // Optimistic updates (if configured) happen before the event is sent.
            if (el.dataset.optimistic) {
                const opt = JSON.parse(el.dataset.optimistic);
                applyOptimisticUpdate(opt);
            }

            e.preventDefault();
            this.sendEvent(0x01, el.dataset.hid);  // CLICK per §22.2
        });

        // Input events (debounced)
        document.addEventListener('input', debounce((e) => {
            const el = e.target.closest('[data-hid]');
            if (el) {
                if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;
                const ve = (el.dataset.ve || '').split(',').map(s => s.trim()).filter(Boolean);
                if (!ve.includes('input')) return;
                this.sendEvent(0x03, el.dataset.hid, el.value);  // INPUT per §22.2
            }
        }, 100));

        // Form submit
        document.addEventListener('submit', (e) => {
            const form = e.target.closest('[data-hid]');
            if (form) {
                if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;
                const ve = (form.dataset.ve || '').split(',').map(s => s.trim()).filter(Boolean);
                if (!ve.includes('submit')) return;
                e.preventDefault();
                this.sendEvent(0x05, form.dataset.hid, new FormData(form));  // SUBMIT per §22.2
            }
        });
    }

    sendEvent(type, hid, payload) {
        const buffer = encodeEvent(type, hid, payload);
        this.ws.send(buffer);
    }

    handleMessage(buffer) {
        const patches = decodePatches(buffer);
        patches.forEach(patch => this.applyPatch(patch));
    }

    applyPatch(patch) {
        const el = document.querySelector(`[data-hid="${patch.hid}"]`);
        if (!el) return;

        // Patch types per §22.3
        switch (patch.type) {
            case 0x01: // SET_TEXT
                el.textContent = patch.text;
                break;
            case 0x02: // SET_ATTR
                el.setAttribute(patch.key, patch.value);
                break;
            case 0x03: // REMOVE_ATTR
                el.removeAttribute(patch.key);
                break;
            case 0x04: // ADD_CLASS
                el.classList.add(patch.className);
                break;
            case 0x05: // REMOVE_CLASS
                el.classList.remove(patch.className);
                break;
            case 0x07: // INSERT_BEFORE
                const node = createNode(patch.vnode);
                el.parentNode.insertBefore(node, el);
                break;
            case 0x0A: // REMOVE_NODE
                el.remove();
                break;
            case 0x0B: // REPLACE_NODE
                const replacement = createNode(patch.vnode);
                el.replaceWith(replacement);
                break;
            // ... see §22.3 for complete list
        }
    }
}

// Initialize on DOM ready
new VangoClient();
```

### 5.3 Optimistic Updates

For instant feedback without waiting for server round-trip. Vango uses a **simple declarative schema** that applies changes to the element itself (or its parent), avoiding CSS selector injection risks.

**Optimistic update schema (normative):**
```json
{
    "class": "className",     // Toggle this class on the element
    "text": "new text",       // Set element's textContent
    "attr": {"name": "val"},  // Set attribute
    "value": "new value"      // Set input value
}
```

Only include the fields you need — empty/omitted fields are ignored.

```go
// Server-side component - toggle class optimistically
Button(
    OnClick(toggleComplete),
    OptimisticClass("completed", true),  // Adds class immediately
    Text("Complete"),
)

// Server-side component - update text optimistically
Span(
    ID("count"),
    Text(fmt.Sprintf("%d", count.Get())),
)
Button(
    OnClick(count.Inc),
    OptimisticText(fmt.Sprintf("%d", count.Get() + 1)),  // Updates button text
    Text("+"),
)
```

Rendered HTML:
```html
<button data-hid="h5" data-ve="click" data-optimistic='{"class":"completed"}'>
    Complete
</button>
```

Client behavior:
```javascript
// In the thin client's click interception handler:
if (el.dataset.optimistic) {
    // Apply optimistic update immediately to this element.
    const opt = JSON.parse(el.dataset.optimistic);
    applyOptimisticUpdate(el, opt);
}
// Then send the event to the server for confirmation (and reconcile on error).
```

For optimistic updates targeting **other elements**, use the `OptimisticParent*` helpers (see §8.3) which operate on the parent element.

This gives 0ms perceived latency for common operations while maintaining server authority.

### 5.4 Reconnection & UX

The thin client handles transient network failures with exponential backoff, and exposes connection state to the UI via CSS classes and events.

**Connection state classes** are applied to `<html>` (`document.documentElement`):

- `html.vango-connecting` — initial connection attempt in progress
- `html.vango-connected` — WebSocket open and healthy
- `html.vango-reconnecting` — reconnect attempts in progress
- `html.vango-disconnected` — gave up / session expired (requires hard reload / user action)

```javascript
scheduleReconnect() {
    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);
    this.reconnectAttempts++;

    setTimeout(() => {
        this.connect();
    }, delay);
}

setState(state) {
    const root = document.documentElement;
    root.classList.remove("vango-connecting", "vango-connected", "vango-reconnecting", "vango-disconnected");
    root.classList.add(`vango-${state}`);

    document.dispatchEvent(new CustomEvent("vango:connection", {
        detail: { state },
    }));
}

// On reconnect within ResumeWindow, the server resumes the session and re-syncs UI.
// If resume fails (expired/evicted), the client performs a hard reload to recover.
```


---

## 6. The WASM Runtime

### 6.1 When to Use WASM Mode

Use WASM for:
- **Offline-first apps** — PWAs that must work without network
- **Latency-critical interactions** — Drawing, music production, games
- **Heavy client computation** — Image processing, data visualization
- **Specific components** — Within otherwise server-driven app

### 6.2 Enabling WASM Mode

**Full WASM mode** (entire app runs in browser):
```go
// vango.json
{
    "mode": "wasm"
}
```

**Hybrid mode** (specific components run client-side):
```go
// Mark a component as client-executed (full component runtime required)
func DrawingCanvas() vango.Component {
    return vango.ClientComponent(func() *vango.VNode {
        // This code runs in WASM, not on server
        canvas := vango.NewRef[js.Value](nil)

        vango.Effect(func() vango.Cleanup {
            ctx := canvas.Current().Call("getContext", "2d")
            // Set up drawing...
            return nil
        })

        return Canvas(
            Ref(canvas),
            Width(800),
            Height(600),
            OnMouseMove(handleDraw),
        )
    })
}
```

### 6.3 Hybrid Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        BROWSER                               │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                  Server-Driven UI                      │  │
│  │   ┌─────────┐  ┌─────────┐  ┌─────────────────────┐   │  │
│  │   │ Header  │  │ Sidebar │  │     Main Content    │   │  │
│  │   │ (12KB)  │  │ (12KB)  │  │       (12KB)        │   │  │
│  │   └─────────┘  └─────────┘  └──────────┬──────────┘   │  │
│  │                                        │               │  │
│  │   ┌────────────────────────────────────▼───────────┐  │  │
│  │   │             WASM Island (Hybrid)              │  │  │
│  │   │   ┌─────────────────────────────────────────┐  │  │  │
│  │   │   │          DrawingCanvas (~50KB)          │  │  │  │
│  │   │   │        Runs entirely in WASM            │  │  │  │
│  │   │   └─────────────────────────────────────────┘  │  │  │
│  │   └────────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

The WASM bundle only includes client-executed widgets/components, not the whole app.

### 6.4 Client vs Server Signals

```go
// Regular signal: lives on server (server-driven) or WASM (WASM mode)
count := vango.NewSignal(0)
// Local to the browser tab (client-only)
cursorPos := vango.NewLocalSignal(Position{0, 0})

// Synced with persistent store (e.g. LocalStorage)
savedValue := vango.NewSyncedSignal(0)
```

**LocalSignal** in server-driven mode:
- Requires minimal WASM runtime (~20KB) or JS implementation
- State never leaves the browser
- Great for UI state (hover, focus, scroll position)

**SyncedSignal**:
- Client updates immediately (optimistic)
- Server confirms or rejects
- Automatic reconciliation

### 6.5 WASM Islands (Hybrid Mode)

WASM islands are Vango’s “escape hatch” for UI that genuinely needs a tight client-side loop while preserving Vango’s server-first default.

#### 6.5.1 What a WASM island is

A **WASM island** is a **client-executed component subtree** embedded inside an otherwise server-driven Vango page. In Hybrid mode, the server continues to render the page and apply patches, but the island’s internal rendering and interaction loop runs in browser WASM.

#### 6.5.2 When to use WASM islands (thesis)

Server events + hooks cover most UI, but some interactions are **continuous** and degrade when expressed as “discrete events → server rerender”:

- Physics simulation (force graphs): continuous calculation, not event-driven
- Canvas drawing: <16ms feedback
- Complex gestures: multi-touch, pressure, custom recognition
- Heavy client data processing (filter/sort/transform large datasets)
- Offline computation (in Hybrid mode: local compute; in WASM mode: full offline-first)

Non-goals:
- WASM islands do **not** replace server-driven defaults.
- WASM islands are **not** “client components everywhere.”
- WASM islands do **not** introduce REST as the primary integration model (Vango remains server-driven by default).

#### 6.5.3 Relationship to Vango rendering modes

Vango supports three modes:

- **Server-driven (default)**: components run on the server; thin client applies binary patches (~12KB).
- **Hybrid**: server-driven plus WASM islands for specific components; thin client stays ~12KB, islands are loaded on demand.
- **Full WASM**: the entire app runs in browser WASM (`"mode":"wasm"`), enabling offline-first.

This section specifies **WASM islands in Hybrid mode**, and defines an optional “full runtime” tier that also underpins Full WASM mode.

#### 6.5.4 Two-tier island runtime model

The core design decision is to separate “widget islands” from “full Vango-in-WASM.” This keeps Hybrid islands small and prevents accidental bundling of a full client framework.

**Tier 1: Widget Island Runtime (default for Hybrid)**

Intent: high-performance client widgets (canvas/webgl/simulation) with minimal runtime.

Characteristics:
- Imperative rendering loop (RAF/timers) and direct browser API usage.
- No requirement to ship a general-purpose VDOM diff engine.
- Island is a **leaf boundary** from the server’s perspective: the server never diffs/patches inside it.
- Communication with the server is **semantic events**, not per-frame streaming.

**Tier 2: Full Component Runtime (advanced)**

Intent: run normal Vango component semantics in the browser: signals/memos/effects, component tree execution, and (optionally) VDOM diffing.

Characteristics:
- Larger footprint (aligns with Full WASM sizing expectations).
- Enables offline-first mode and richer client ownership of UI state.
- Used for Full WASM mode and rare Hybrid islands that truly require full component semantics (explicit opt-in).

#### 6.5.5 Developer-facing API

Vango exposes **two** Hybrid WASM primitives to match the two runtime tiers.

**Tier 1: Widget islands (`WASMWidget`)**

`WASMWidget(...)` embeds an *opaque* container whose contents are owned by a WASM module (imperative rendering loop; no Vango component semantics inside the module).

```go
func ForceGraph(nodes []Node, edges []Edge) *vango.VNode {
    return vango.WASMWidget(
        "force-graph",
        vango.WASMModule("/wasm/force-graph.wasm"),
        map[string]any{
            "nodes": nodes,
            "edges": edges,
        },
    )
}
```

Widget module contract (conceptual):
- `mount(container, props)` → instance
- `update(instance, newProps)` (optional; idempotent)
- `destroy(instance)` (optional)

**Tier 2: Client components (`ClientComponent`)**

`ClientComponent(...)` marks a normal Vango component subtree to execute in browser WASM with the full component runtime (signals/memos/effects/component execution).

```go
func DrawingCanvas() vango.Component {
    return vango.ClientComponent(func() *vango.VNode {
        return Canvas(/* ... */)
    })
}
```

Compatibility:
- `ClientRequired(...)` is a compatibility alias for `ClientComponent(...)` (it always implies the **full** component runtime).

**Props model**

Islands receive **typed props** from the server. Props must be serializable (JSON-compatible or Vango’s binary encoding). For Tier 1, props should be treated as configuration + initial state, not a frequently updated stream.

Guideline: large datasets should be passed by reference (resource id / URL) when appropriate, or chunked, rather than embedded repeatedly.

**Event model (semantic commits)**

Islands communicate back to the server via semantic events (e.g., “node pinned,” “edge created,” “layout saved”). This matches the hook philosophy: the client can run at 60fps, and the server receives only final results.

**Island message bridge (shared abstraction with JavaScript islands)**

Vango standardizes a single bridge for islands (JS and WASM):
- `vango.SendToIsland(islandID, msg)` — server → island
- `vango.OnIslandMessage(islandID, handler)` — island → server

Transport: implementations reuse the existing WebSocket binary protocol:
- island → server messages use a `CUSTOM` event subtype
- server → island messages use an `ISLAND_MESSAGE` control patch
(see Protocol Specification).

**State management inside islands**

Tier 1 (Widget Runtime):
- Ephemeral tight-loop state (positions, camera, hover) should be module-local (plain Go structs), not Vango signals.
- Persisted/shared state remains server authoritative; islands send semantic events to mutate it.
- Avoid high-frequency server sync; prefer commit events at interaction boundaries (mouse up, confirm).

Tier 2 (Full Runtime):
- Islands may run the full reactive model client-side (including `LocalSignal` / `SyncedSignal` where appropriate).
- In Full WASM mode, server interaction becomes resource/API-driven.

#### 6.5.6 Runtime architecture (SSR + progressive handoff)

On the server render pass, an island renders as a container element with:
- island id/type,
- bundle identifier,
- serialized props payload,
- optional SSR fallback/skeleton.

The page becomes interactive immediately via the thin client (links, server events), while WASM loads in the background.

**Thin client: island loader**

Loading flow:
1. Thin client boots (~12KB) and app is usable.
2. Client scans DOM for island markers.
3. Client preloads needed WASM bundles (background).
4. Client instantiates the WASM runtime once per bundle.
5. Client mounts islands with props.
6. If props change later, client calls island `update`.

The server may emit preload hints for island bundles when they appear in SSR output (parallel to hook modulepreload hints).

**Lifecycle contract**

All WASM islands implement:
- `mount(container, props) -> instance`
- `update(instance, newProps)` (optional; idempotent)
- `destroy(instance)` (optional)

Race safety: if an island loads slowly and the user navigates away, the loader must detect that the container is gone and destroy immediately after mount (mirrors hook loader race handling).

#### 6.5.7 Island boundary and patching rules

Canonical invariant (normative): **the server patch engine MUST NOT patch inside any island-managed subtree** (WASM widgets, client components, and JavaScript islands). Islands are opaque boundaries.

To prevent server patches from corrupting island-managed state:

- The Vango patch engine MUST NOT patch inside the island container’s subtree.
- The patch engine MAY:
  - update container attributes (size, class),
  - update the island props payload,
  - or replace the entire island container (triggering destroy/remount).

This same “opaque subtree” rule applies to JavaScript islands.

#### 6.5.8 Build and packaging

Vango supports:
- Hybrid build: server binary + thin client + island bundles.
- Full WASM build: WASM app bundle (Tier 2 runtime) configured by `"mode":"wasm"`.

During `vango build`, Vango statically identifies `WASMWidget(...)` and `ClientComponent(...)` usage (with `ClientRequired` treated as an alias of `ClientComponent`) and constructs an island dependency graph.

Bundling strategy:
- Island bundles are route-chunked or named bundles (implementation choice), but must be:
  - content-hashed (`graph.<hash>.wasm`),
  - long-cacheable,
  - loadable on demand.

**Compiler strategy (configurable)**

Because performance/compatibility differs by compiler/runtime, compiler choice is explicit and reproducible:
- Tier 1 defaults to a small-widget-optimized compiler (e.g., TinyGo).
- Tier 2 defaults to full compatibility (e.g., standard Go WASM target).

Add to `vango.json`:
- default compiler per tier,
- per-island overrides.

Vango may provide build-time warnings when a Tier 1 widget pulls in Tier 2 runtime primitives, or when bundle sizes suggest unintended tier escalation.

**Size budgets and build-time reporting**

To preserve the “thin client + partial WASM” story, the build includes an analyzer that reports:
- per-island bundle sizes (raw, gzip/brotli),
- per-route aggregate island sizes,
- dependency attribution,
- configurable budgets (warn by default; hard fail optional).

#### 6.5.9 Performance guidance

Primary pitfall: interop granularity. For canvas/simulation islands:
- avoid per-node/per-primitive JS boundary calls,
- prefer batching (paths, buffers, single draws per frame),
- keep per-frame allocations minimal to avoid GC jitter.

Event throttling: islands should not emit high-frequency events to the server; prefer commit events at interaction boundaries (mouse up, confirm).

Fallbacks: if WASM fails to load or instantiate, the SSR fallback remains visible, and the system logs a clear error (optionally exposing a user-visible “Interactive mode unavailable” state).

#### 6.5.10 Security model

WASM islands run in the user’s browser and are therefore untrusted. All island → server messages must be validated as if they were form input:
- message size limits and structural validation (depth/allocation caps),
- server-side authorization on any mutation,
- rate limiting / debouncing for event spam.

#### 6.5.11 Summary of normative requirements

1. Hybrid islands are marked explicitly (`WASMWidget` for Tier 1; `ClientComponent` for Tier 2) and compiled into separate WASM bundles; bundles include only those islands.
2. Two runtime tiers exist: Tier 1 widget runtime (default for Hybrid) and Tier 2 full component runtime (explicit opt-in; primarily Full WASM mode).
3. Thin client loads islands on demand, analogous to hook loading strategy.
4. Island boundary is a leaf: server patches do not mutate island internals.
5. Island-server communication is semantic, validated, rate-limited, and server-authoritative.
6. Build analyzer and size budgets are first-class to prevent unintentional tier escalation and bundle regressions.

---

## 7. State Management

Vango's state management is designed around the server-driven architecture. Unlike client-side frameworks that need complex solutions (Redux, MobX, Zustand) to synchronize state, Vango keeps state on the server where the data already lives.

### 7.0 Rules of Correctness (Normative)

1. **Single-writer session loop:** Signal writes (**Set/Update** and all convenience writers like `Inc`, `Toggle`, etc.) MUST occur on the session event loop.
2. **Cross-goroutine updates:** Goroutines MUST NOT write signals directly; they MUST marshal updates back via `ctx.Dispatch(...)`.
3. **No in-place mutation of composite values:** If a signal stores a slice/map/struct that contains slices/maps/pointers, updates SHOULD treat the value as immutable and return a fresh copy (see §7.6). This avoids aliasing bugs and makes debugging/DevTools reliable.
4. **Hook-order stability:** Render-time creation of `Signal/Memo/Effect/Resource/Ref` MUST follow stable hook-order semantics (see §3.1.3).

### 7.1 Why Vango Doesn't Need Redux

Traditional SPAs face a fundamental problem: state lives in the browser but data lives on the server. This creates:

- **Synchronization hell** — Keep client state in sync with server
- **Cache invalidation** — When is cached data stale?
- **Optimistic updates** — Show changes before server confirms
- **Conflict resolution** — What if two tabs modify the same data?

Vango sidesteps most of this by keeping state on the server:

```
Traditional SPA:                     Vango Server-Driven:

┌─────────────────────┐              ┌─────────────────────┐
│  Browser            │              │  Browser            │
│  ┌───────────────┐  │              │                     │
│  │ Redux Store   │  │              │  (minimal state)    │
│  │ - users       │  │              │  - UI-only state    │
│  │ - products    │  │              │  - optimistic vals  │
│  │ - cart        │  │              │                     │
│  │ - filters     │  │              └──────────┬──────────┘
│  └───────────────┘  │                         │
│         ↕ sync      │                         │ WebSocket
└─────────┬───────────┘              ┌──────────▼──────────┐
          │                          │  Server             │
┌─────────▼───────────┐              │  ┌───────────────┐  │
│  Server             │              │  │ All State     │  │
│  (source of truth)  │              │  │ Lives Here    │  │
└─────────────────────┘              │  └───────────────┘  │
                                     └─────────────────────┘
```

### 7.2 Signal Scopes

Vango provides three signal scopes for different use cases:

```go
// ┌─────────────────────────────────────────────────────────────────┐
// │                     SIGNAL SCOPES                                │
// ├─────────────────┬─────────────────┬─────────────────────────────┤
// │    NewSignal     │  NewSharedSignal │    NewGlobalSignal         │
// ├─────────────────┼─────────────────┼─────────────────────────────┤
// │ One component   │ One session     │ All sessions                │
// │ instance        │ (one user/tab)  │ (all users)                 │
// ├─────────────────┼─────────────────┼─────────────────────────────┤
// │ Form input,     │ Shopping cart,  │ Live cursors,               │
// │ local UI state  │ cart, filters   │ collaborative editing,      │
// │                 │ filters         │ presence indicators         │
// └─────────────────┴─────────────────┴─────────────────────────────┘

// Component-local: created per instance, GC'd on unmount
func Counter(initial int) vango.Component {
    return vango.Func(func() *vango.VNode {
        count := vango.NewSignal(initial)  // Local to this Counter
        return Div(Text(fmt.Sprintf("%d", count.Get())))
    })
}

// Session-shared: defined at package level, scoped to user session
var CartItems = vango.NewSharedSignal([]CartItem{})

// Global: shared across ALL connected users
var OnlineUsers = vango.NewGlobalSignal([]User{})
```

### 7.3 Shared State Patterns

#### Basic Shared State (Store Pattern)

```go
// store/cart.go — Define your store
package store

import "vango"

// State
var CartItems = vango.NewSharedSignal([]CartItem{})

// Derived state (automatically updates when CartItems changes)
var CartTotal = vango.NewSharedMemo(func() float64 {
    total := 0.0
    for _, item := range CartItems.Get() {
        total += item.Price * float64(item.Qty)
    }
    return total
})

var CartCount = vango.NewSharedMemo(func() int {
    count := 0
    for _, item := range CartItems.Get() {
        count += item.Qty
    }
    return count
})

// Actions (functions that modify state)
func AddItem(product Product, qty int) {
    CartItems.Update(func(items []CartItem) []CartItem {
        // Copy to avoid mutating shared backing arrays.
        next := make([]CartItem, len(items))
        copy(next, items)

        // Check if already in cart
        for i, item := range next {
            if item.ProductID == product.ID {
                next[i].Qty += qty
                return next
            }
        }
        // Add new item
        return append(next, CartItem{
            ProductID: product.ID,
            Product:   product,
            Qty:       qty,
        })
    })
}

func RemoveItem(productID int) {
    CartItems.Update(func(items []CartItem) []CartItem {
        result := make([]CartItem, 0, len(items))
        for _, item := range items {
            if item.ProductID != productID {
                result = append(result, item)
            }
        }
        return result
    })
}

func ClearCart() {
    CartItems.Set([]CartItem{})
}
```

```go
// components/header.go — Read from store
func Header() *vango.VNode {
    return Nav(Class("header"),
        Logo(),
        NavLinks(),
        // Reads from shared state — auto-updates when cart changes
        CartBadge(store.CartCount()),
    )
}
```

```go
// components/product_card.go — Write to store
func ProductCard(p Product) *vango.VNode {
    return Div(Class("product-card"),
        Img(Src(p.ImageURL)),
        H3(Text(p.Name)),
        Price(p.Price),
        Button(
            OnClick(func() { store.AddItem(p, 1) }),
            Text("Add to Cart"),
        ),
    )
}
```

#### Complex State with Nested Objects

```go
// store/project.go
package store

type ProjectState struct {
    Project  *Project
    Tasks    []Task
    Filter   TaskFilter
    Selected map[int]bool
}

var State = vango.NewSharedSignal(ProjectState{})

// Selectors (derived state for specific parts)
var FilteredTasks = vango.NewSharedMemo(func() []Task {
    s := State.Get()
    return filterTasks(s.Tasks, s.Filter)
})

var SelectedCount = vango.NewSharedMemo(func() int {
    count := 0
    for _, selected := range State.Get().Selected {
        if selected {
            count++
        }
    }
    return count
})

// Actions with targeted updates
func ToggleTask(taskID int) {
    State.Update(func(s ProjectState) ProjectState {
        // Copy slice to avoid in-place mutation of shared backing arrays.
        nextTasks := make([]Task, len(s.Tasks))
        copy(nextTasks, s.Tasks)

        for i := range nextTasks {
            if nextTasks[i].ID == taskID {
                nextTasks[i].Done = !nextTasks[i].Done
                break
            }
        }
        s.Tasks = nextTasks
        return s
    })
}

func SetFilter(filter TaskFilter) {
    State.Update(func(s ProjectState) ProjectState {
        s.Filter = filter
        return s
    })
}

func SelectTask(taskID int, selected bool) {
    State.Update(func(s ProjectState) ProjectState {
        // Copy map to avoid mutating shared references.
        nextSelected := make(map[int]bool, len(s.Selected)+1)
        for k, v := range s.Selected {
            nextSelected[k] = v
        }
        nextSelected[taskID] = selected
        s.Selected = nextSelected
        return s
    })
}
```

### 7.4 Global State (Real-time Collaboration)

GlobalSignals synchronize across all connected sessions:

```go
// store/presence.go
package store

// Shared across ALL users
var OnlineUsers = vango.NewGlobalSignal([]User{})
var CursorPositions = vango.NewGlobalSignal(map[string]Position{})

// When a user connects
func UserJoined(user User) {
    OnlineUsers.Update(func(users []User) []User {
        next := make([]User, 0, len(users)+1)
        next = append(next, users...)
        next = append(next, user)
        return next
    })
}

// When a user disconnects (called automatically by Vango)
func UserLeft(userID string) {
    OnlineUsers.Update(func(users []User) []User {
        result := make([]User, 0, len(users))
        for _, u := range users {
            if u.ID != userID {
                result = append(result, u)
            }
        }
        return result
    })

    CursorPositions.Update(func(pos map[string]Position) map[string]Position {
        next := make(map[string]Position, len(pos))
        for k, v := range pos {
            next[k] = v
        }
        delete(next, userID)
        return next
    })
}

// Broadcast cursor movement
func MoveCursor(userID string, pos Position) {
    CursorPositions.Update(func(positions map[string]Position) map[string]Position {
        next := make(map[string]Position, len(positions)+1)
        for k, v := range positions {
            next[k] = v
        }
        next[userID] = pos
        return next
    })
}
```

```go
// components/collaborative_canvas.go
func CollaborativeCanvas() vango.Component {
    return vango.Func(func() *vango.VNode {
        ctx := vango.UseCtx()
        cursors := store.CursorPositions()
        currentUser := ctx.User()

        return Div(Class("canvas"),
            // Render other users' cursors
            Range(cursors, func(userID string, pos Position) *vango.VNode {
                if userID == currentUser.ID {
                    return nil  // Don't show own cursor
                }
                return CursorIndicator(userID, pos)
            }),

            // Track mouse movement
            Div(
                Class("canvas-area"),
                OnMouseMove(func(e vango.MouseEvent) {
                    store.MoveCursor(currentUser.ID, Position{x, y})
                }),
            ),
        )
    })
}
```

### 7.5 Resource Pattern for Async Data

For loading data with loading/error/success states:

```go
// Resource wraps async data with loading state
func UserProfile(userID int) vango.Component {
    return vango.Func(func() *vango.VNode {
        // Resource handles loading/error/ready states
        user := vango.NewResource(func() (*User, error) {
            return db.Users.FindByID(userID)
        })

        // Pattern 1: Switch on state
        switch user.State() {
        case vango.Loading:
            return Skeleton("user-profile")
        case vango.Error:
            return ErrorCard(user.Error())
        case vango.Ready:
            return UserCard(user.Data())
        }
        return nil
    })
}
```

```go
// Pattern 2: Match helper for cleaner code
func UserProfile(userID int) vango.Component {
    return vango.Func(func() *vango.VNode {
        user := vango.NewResource(func() (*User, error) {
            return db.Users.FindByID(userID)
        })

        return user.Match(
            vango.OnLoading(func() *vango.VNode {
                return Skeleton("user-profile")
            }),
            vango.OnError(func(err error) *vango.VNode {
                return ErrorCard(err)
            }),
            vango.OnReady(func(u *User) *vango.VNode {
                return UserCard(u)
            }),
        )
    })
}
```

```go
// Pattern 3: With refetch capability
func ProjectPage(projectID int) vango.Component {
    return vango.Func(func() *vango.VNode {
        project := vango.NewResource(func() (*Project, error) {
            return db.Projects.FindByID(projectID)
        })

        return Div(
            Header(
                H1(Text(project.Data().Name)),
                Button(
                    OnClick(project.Refetch),  // Manually trigger reload
                    Text("Refresh"),
                ),
            ),
            // ...
        )
    })
}
```

### 7.6 Immutable Update Helpers

To avoid accidental mutations, use these helper patterns:

Normative rule: update functions that operate on slices/maps/structs MUST NOT mutate shared backing arrays or referenced maps in place. Return a fresh copy when modifying composite values.

```go
// DANGEROUS: Mutating in place
items.Update(func(i []Item) []Item {
    i[0].Done = true  // Mutation! Other references see this change
    return i
})

// SAFE: Create new slice/struct
items.Update(func(items []Item) []Item {
    result := make([]Item, len(items))
    copy(result, items)
    result[0] = Item{...result[0], Done: true}
    return result
})
```

**Built-in helpers for common operations:**

```go
// Array operations
items.Append(newItem)                           // Add to end
items.Prepend(newItem)                          // Add to start
items.InsertAt(index, newItem)                  // Insert at position
items.RemoveAt(index)                           // Remove by index
items.RemoveWhere(func(i Item) bool { ... })    // Remove by predicate
items.UpdateAt(index, func(i Item) Item { ... }) // Update single item
items.UpdateWhere(predicate, updater)           // Update matching items

// Map operations
users.SetKey(id, user)                          // Set key
users.RemoveKey(id)                             // Remove key
users.UpdateKey(id, func(u User) User { ... })  // Update key

// Examples:
tasks.UpdateAt(0, func(t Task) Task {
    return Task{...t, Done: true}
})

tasks.RemoveWhere(func(t Task) bool {
    return t.Done
})

userMap.UpdateKey(userID, func(u User) User {
    return User{...u, LastSeen: time.Now()}
})
```

### 7.7 Computed Chains (Derived State)

Memos can depend on other memos, creating a computation graph:

```go
// store/analytics.go
package store

var RawData = vango.NewSharedSignal([]DataPoint{})
var DateRange = vango.NewSharedSignal(DateRange{Start: weekAgo, End: now})
var Grouping = vango.NewSharedSignal("day")  // day, week, month

// Level 1: Filter by date
var FilteredData = vango.NewSharedMemo(func() []DataPoint {
    data := RawData.Get()
    range_ := DateRange.Get()
    return filterByDate(data, range_.Start, range_.End)
})

// Level 2: Group filtered data
var GroupedData = vango.NewSharedMemo(func() map[string][]DataPoint {
    data := FilteredData.Get()  // Depends on FilteredData
    grouping := Grouping.Get()
    return groupByPeriod(data, grouping)
})

// Level 3: Aggregate grouped data
var ChartData = vango.NewSharedMemo(func() []ChartPoint {
    grouped := GroupedData.Get()  // Depends on GroupedData
    return aggregateForChart(grouped)
})

// Level 3 (parallel): Summary stats
var SummaryStats = vango.NewSharedMemo(func() Stats {
    data := FilteredData.Get()  // Also depends on FilteredData
    return Stats{
        Total:   sum(data),
        Average: avg(data),
        Max:     max(data),
        Min:     min(data),
    }
})
```

```
Dependency Graph:

RawData ──┬──► FilteredData ──┬──► GroupedData ──► ChartData
          │                   │
DateRange─┘                   └──► SummaryStats

Grouping ─────────────────────────► GroupedData
```

When `DateRange` changes:
1. `FilteredData` recomputes
2. `GroupedData` recomputes (depends on FilteredData)
3. `ChartData` recomputes (depends on GroupedData)
4. `SummaryStats` recomputes (depends on FilteredData)

Memos that don't depend on changed values are NOT recomputed.

### 7.8 Batching Updates

Multiple signal updates in sequence trigger multiple re-renders. Use `Batch` to combine them:

```go
// Without batch: 3 re-renders
func resetFilters() {
    store.SearchQuery.Set("")     // Re-render 1
    store.Category.Set("all")     // Re-render 2
    store.SortOrder.Set("newest") // Re-render 3
}

// With batch: 1 re-render
func resetFilters() {
    vango.Batch(func() {
        store.SearchQuery.Set("")
        store.Category.Set("all")
        store.SortOrder.Set("newest")
    })
    // Single re-render after batch completes
}
```

`Batch` is a compatibility alias for an anonymous Transaction (`Tx`). See the next section for atomicity, consistent reads, and better debugging/observability.

### 7.8.1 Transactions & Snapshots

#### Problem

Vango already supports fine-grained reactivity (Signals/Memos/Effects) and provides `vango.Batch` to combine multiple signal updates into a single re-render.

However, batching alone does not fully solve three recurring production problems:

1. Intermediate (partially-updated) UI states during multi-signal updates, especially when updates occur outside a single event handler (effects, async completions, timers).
2. Non-obvious causality in debugging: “What action caused these signal writes and this patch set?”
3. Inconsistent reads when complex update logic needs a stable “before” view (e.g., compute delta from previous state, revert on error, optimistic reconciliation).

To address this, Vango introduces **Transactions** and **Snapshots**: a stricter semantic layer built on top of Signals/Memos/Effects that makes state transitions atomic, consistent, and debuggable.

This is not a database transaction system. It provides atomicity and consistency for in-session reactive state only.

#### Design Goals

Correctness:
- Atomic visibility of multi-signal updates (no partial UI).
- Deterministic effect scheduling relative to commits.
- Clear semantics for nested updates and update loops.

DX:
- First-class “action boundaries” that group signal changes, renders, patches, and effects into one traceable unit (a transaction).
- Logs/DevTools can answer: “What changed, why, and what did it cause?”

Performance:
- Reduce redundant renders/diffs/patches by committing once per logical action.
- Allow the runtime to coalesce cascading updates from effects into bounded flush cycles.

Safety:
- Preserve the single-writer per-session model (session runtime is not thread-safe; concurrency must marshal back to the session loop).

#### Definitions

- Transaction (Tx): a state transition boundary that collects signal writes and commits them atomically.
- Commit: the moment all buffered writes become visible, a render/diff/patch pass runs, and dependent effects are scheduled.
- Snapshot: a stable “before” view of committed signal values used for consistent reads and delta computations.
- Write set: all signal writes performed during a Tx.
- Tx report: debug/DevTools representation of a transaction (writes, renders, patches, effects, durations).

#### Core Semantics

**S1. Transaction atomicity**

All signal writes within a Tx become visible atomically at commit. No render/diff/patch may observe a partially-applied write set.

This generalizes `vango.Batch`, which exists to prevent multiple re-renders from sequential updates.

**S2. Consistent reads (read-your-writes + “before” snapshots)**

Within a Tx:
- Reading a signal returns the most recent value written in the current Tx if present (“read your writes”).
- A Snapshot taken at Tx start returns the committed “before” values, even after writes occur.

**S3. Deterministic update pipeline**

A commit runs the following pipeline in order:

1. Apply the write set to the session’s signal store
2. Mark dependents dirty (components + memos + effects)
3. Render dirty components
4. VDOM diff + patch encode + send
5. Run effect cleanups and schedule effect runs (effects run after render)
6. If effects produced additional signal writes, run another commit cycle (bounded; see S7)

Normative ordering rules:
- Effects MUST run after patches are computed (and typically after patch send) to avoid re-entrant dirtying during diff/encode.
- Signal writes performed by effects MUST be collected into the next commit cycle, and the runtime MUST enforce `MaxCommitCycles` to prevent infinite loops.

**S4. Implicit Tx per session “tick” (default)**

By default, the runtime executes these operations within an implicit Tx:
- every incoming UI event handler (HID handler)
- component mount/unmount callbacks
- effect executions (after render)
- application of async results that marshal back to the session loop (see `ctx.Dispatch`)

Because of the implicit Tx, most application code does not need explicit transactions for correctness. Use `TxNamed(...)` primarily for naming/action boundaries in logs, DevTools, and tracing, and for grouping multi-signal updates that are not already inside a single handler.

**S5. Nested Tx behavior**

Transactions may nest. Nested transactions do not commit independently:
- inner Tx merges its write set into the nearest outer Tx
- commit occurs only when the outermost Tx exits

**S6. Rollback (signal state only)**

If a panic occurs inside a Tx body:
- the Tx write set is discarded (no signal state changes are committed)
- the panic is recovered by the runtime (in production) and routed to the existing error boundary mechanism (and surfaced in DevTools in dev mode)

Rollback applies only to signal state, not external side effects (DB writes, network calls).

**S7. Bounded stabilization for effect-triggered updates**

Effects may set signals and legitimately require additional render passes. To prevent infinite reactive loops, the runtime enforces:
- a maximum number of commit cycles per tick (default `MaxCommitCycles = 25` in dev mode; lower in prod)
- if exceeded: abort further cycles, log an error with the dependency chain, and surface in DevTools

#### API Specification

**Transaction API**

```go
// Tx runs fn inside a transaction.
// If called inside an existing Tx, it nests (no intermediate commit).
func Tx(fn func())

// TxNamed runs fn inside a named transaction.
// Name is used for logs, DevTools, and tracing.
func TxNamed(name string, fn func())
```

Behavior:
- All `Signal.Set`, `Signal.Update`, and convenience writes (`Inc`, `Toggle`, etc.) inside the Tx are buffered into the write set.
- At Tx exit (outermost), the write set commits and triggers a single render/diff/patch cycle.

Relationship to Batch:
- `vango.Batch` is a compatibility alias for `vango.Tx`.

**Snapshot API**

Snapshots are designed for stable “before” reads and for consistent delta computations.

```go
// Snapshot captures a stable view of committed signal values.
// Values may be captured lazily on first access.
type Snapshot struct{}

// Snapshot returns a snapshot of the current committed state.
// If called inside a Tx, this snapshot reflects state at Tx start.
func Snapshot() Snapshot

// SnapshotGet returns the committed “before” value for sig.
// If called outside a Tx, it reads the current committed value.
func SnapshotGet[T any](snap Snapshot, sig Signal[T]) T
```

Note: Go does not currently support generic methods, so `SnapshotGet` is a function rather than `snap.Get(sig)`.

Snapshot capture semantics (normative):
- If called outside a Tx, a snapshot reflects the current committed state.
- If called inside a Tx, a snapshot reflects the committed state at the start of the **outermost** Tx, and MUST NOT include buffered writes performed later in the Tx.
- Snapshots are **shallow** views: composite values (slices, maps, pointers) may still alias. Use copying for rollback/state restoration (see below).

Rule of thumb:
- Use `sig.Get()` in render/memo/effect when you want reactivity.
- Use `sig.Peek()` (or `vango.Untracked`) for reads that should not create reactive dependencies.
- Use `SnapshotGet(snap, sig)` when you specifically need the “before” value stable across Tx writes.

**Snapshot vs Copy (rollback correctness)**

Snapshots provide consistent “before” reads for computation, but they do not automatically deep-copy composite values. If you need to *restore* prior state (rollback), you MUST copy slices/maps/struct graphs as appropriate for your domain types (see §7.6).

**Cross-goroutine updates (required for correctness)**

The session runtime is single-writer and not thread-safe. Async goroutines must not mutate session state directly and must send results back to the session loop.

```go
// Dispatch schedules fn to run on the session event loop inside an implicit Tx.
// Safe to call from any goroutine.
func (ctx Ctx) Dispatch(fn func())
```

Concurrency note (normative):
- `ctx.Dispatch(...)` MUST be safe to call from any goroutine.
- `ctx.StdContext()` MUST be safe to call from any goroutine (or safe to capture and use from goroutines).
- Other `ctx` methods MUST NOT be assumed goroutine-safe unless explicitly documented.

`ctx.StdContext()` semantics (normative):
- `ctx.StdContext()` MUST be derived from the current session “tick” context.
- `ctx.StdContext()` SHOULD carry tracing and any applicable deadlines/timeouts for the current tick.
- If user code derives a cancelable context (e.g. `cctx, cancel := context.WithCancel(ctx.StdContext())`) and returns `cancel` as an effect cleanup, the runtime MUST call that cleanup on dependency changes and unmount, ensuring cancellation propagates.

Dev-mode enforcement:
- If `Signal.Set/Update` is called from outside the session loop, dev mode panics with a clear message:
  - “Signal write from non-session goroutine; use ctx.Dispatch”

Production behavior is configurable (auto-dispatch best-effort vs drop with error logging).

#### Examples

**Example 1: Replace Batch with named Tx (better logs + DevTools)**

```go
func resetFilters() {
    vango.TxNamed("filters:reset", func() {
        store.SearchQuery.Set("")
        store.Category.Set("all")
        store.SortOrder.Set("newest")
    })
}
```

**Example 2: Using Snapshot for “before/after” deltas**

Use case: toggle a boolean and log analytics with the previous value.

```go
func toggleSidebar() {
    vango.TxNamed("ui:toggle_sidebar", func() {
        snap := vango.Snapshot()

        wasOpen := vango.SnapshotGet(snap, store.SidebarOpen)
        store.SidebarOpen.Set(!wasOpen)

        analytics.Track("sidebar_toggled", map[string]any{
            "from": wasOpen,
            "to":   !wasOpen,
        })
    })
}
```

**Example 3: Effect-triggered writes are coalesced and bounded**

```go
func UserProfile(userID Signal[int]) vango.Component {
    return vango.Func(func() *vango.VNode {
        user := vango.NewSignal[*User](nil)
        loadErr := vango.NewSignal[error](nil)

        vango.Effect(func() vango.Cleanup {
            id := userID.Get()
            return vango.GoLatest(id,
                func(ctx context.Context, id int) (*User, error) {
                    return db.Users.FindByID(ctx, id)
                },
                func(u *User, err error) {
                    vango.TxNamed("user:load_result", func() {
                        if err != nil {
                            loadErr.Set(err)
                            user.Set(nil)
                            return
                        }
                        loadErr.Set(nil)
                        user.Set(u)
                    })
                },
                vango.GoLatestTxName("user:load"),
            )
        })

        if user.Get() == nil && loadErr.Get() == nil {
            return Spinner()
        }
        if loadErr.Get() != nil {
            return ErrorMessage(loadErr.Get())
        }
        return ProfileView(user.Get())
    })
}
```

#### Runtime Architecture (Implementation Guidance)

This section describes runtime behavior and internal structure; it is not a public API contract.

**Data structures (per session)**

```go
type sessionState struct {
    version uint64

    // Active transaction stack (nesting)
    txStack []txn

    // Pending work queues
    dirtyComponents set[componentID]
    pendingEffects  []effectID
}

type txn struct {
    id      uint64
    name    string
    baseVer uint64

    // Buffered writes: signalID -> newValue
    writes map[signalID]any

    // Snapshot cache: signalID -> baseValue (lazy)
    snapCache map[signalID]any
}
```

**Signal read/write behavior**

Read (`sig.Get()`):
- If in a Tx and `sig` has a buffered write, return the buffered value.
- Else return the committed value.
- If in a reactive context (render/memo/effect), record dependency as usual.

Peek (`sig.Peek()`):
- Same value resolution as read, but does not record a dependency.

Write (`sig.Set` / `sig.Update`):
- If in a Tx: update the write buffer.
- If not in a Tx: begin an implicit Tx for the current tick (session loop), buffer write, commit at end of tick.

**Event loop pseudocode (Tx-aware)**

```go
func (s *Session) eventLoop() {
    for {
        select {
        case event := <-s.events:
            s.beginTx("event:" + event.Type)
            s.runHandler(event)
            s.commitTx()

        case job := <-s.dispatchQueue:
            s.beginTx("dispatch")
            job()
            s.commitTx()

        case <-s.done:
            return
        }
    }
}
```

**Commit pipeline pseudocode**

```go
func (s *Session) commitTx() {
    // Commit only outermost
    if len(s.txStack) > 1 {
        s.mergeTopIntoParent()
        return
    }

    tx := s.popTx()
    if len(tx.writes) == 0 {
        return
    }

    // 1) Apply writes and bump version
    s.applyWrites(tx.writes)
    s.version++

    // 2) Render/diff/patch
    s.renderDirtyComponents()
    patches := vdom.Diff(s.LastTree, s.CurrentTree)
    s.sendPatches(patches)
    s.LastTree = s.CurrentTree

    // 3) Effects: cleanup then run (after render)
    s.runEffects()

    // 4) If effects wrote signals, repeat (bounded)
    if s.hasPendingWrites() {
        // ...
    }
}
```

#### Developer Experience

Named transactions are the primary “action boundary” for logs, DevTools, and tracing:
- Prefer `TxNamed("domain:action", ...)` for user-visible interactions.
- Signal-level action names remain useful for sub-actions inside a Tx, but Tx name is the default grouping key.

DevTools enhancements (dev mode):
- Tx timeline: `(id, name, start/end, duration)`
- Write set: signals changed (old → new)
- Render set: components re-rendered + per-component patch counts
- Patch stats: patch count and encoded bytes
- Effects: which effects re-ran and why (dependencies)
- “MaxCommitCycles exceeded” diagnostics with dependency chain

Observability (OpenTelemetry):
- represent Tx as child spans under the event span, or as structured span events (lower overhead)
- recommended attributes:
  - `vango.tx.id`, `vango.tx.name`
  - `vango.tx.writes.count`
  - `vango.tx.renders.count`
  - `vango.tx.patches.count`, `vango.tx.patches.bytes`
  - `vango.tx.effects.count`
  - `vango.tx.cycles`

Testing:
- test helpers that flush updates may expose a Tx report in dev/test mode so tests can assert that a state transition commits in one Tx with bounded patches.

#### Anti-Patterns and Guidance

1. Do not write signals from arbitrary goroutines. Use `ctx.Dispatch`.
2. Avoid irreversible external side effects mid-transaction unless errors are handled (Tx rollback only covers signal state).
3. Prefer Snapshots for “before” reads when updates are complex or nested.
4. Use `sig.Peek()` / `vango.Untracked` for non-reactive reads (analytics/logging) to avoid accidental dependencies.

### 7.9 Durability & Persistence

Because Vango is server-driven, the server cannot synchronously read browser storage during component initialization. Instead of a “magic” `Signal.Persist(...)`, Vango provides two explicit primitives:

1. **Session durability** (signals restored on refresh/reconnect, and optionally across restarts via `SessionStore`)
2. **User preferences** (client-backed `Pref` with merge + sync)

#### Session Durability (Signals)

When session serialization is enabled, signals are persisted by default (as JSON) and restored on resume. Mark ephemeral state as transient.

```go
// Persisted by default (as part of the session)
draft := vango.NewSignal(FormDraft{})

// Excluded from persistence
cursor := vango.NewSignal(Point{0, 0}, vango.Transient())

// Stable key for serialization (recommended for important values)
wizardStep := vango.NewSignal(1, vango.PersistKey("checkout_step"))
```

#### User Preferences (Pref)

Preferences (theme, sidebar, language) are **not** normal signals. They are backed by browser storage and can optionally sync to the server on login.

```go
import "github.com/vango-go/vango/pkg/pref"

// Anonymous users: stored locally. Authenticated users: can sync to DB.
var Theme = pref.New("theme", "system", pref.MergeWith(pref.DBWins))

// Read/write
current := Theme.Get()
Theme.Set("dark")
```

Merge strategies:
- `pref.DBWins`: server/DB authoritative (default for logged-in users)
- `pref.LocalWins`: local authoritative (device-specific preferences)
- `pref.LWW`: last-write-wins (timestamp-based)
- `pref.Prompt`: application-controlled conflict resolution

Sync behavior:
- **Cross-tab**: uses `BroadcastChannel` to keep tabs consistent
- **Cross-device**: when authenticated, writes can sync via server and be broadcast to other active sessions

### 7.10 Debugging Signals

In development mode, Vango logs all signal changes:

```go
// Optional: Add action names for clearer logs
count.Set(count.Get() + 1, "increment button clicked")
```

```
[12:34:56.789] Signal store.CartItems: [] → [{id:1, qty:1}]
               Action: "add to cart"
               Source: components/product_card.go:42
               Subscribers: [Header, Sidebar, CartPage]

[12:34:56.801] Memo store.CartTotal: recomputed 0 → 29.99
               Dependencies: [store.CartItems]

[12:34:56.802] Re-render: Header (1 patch), Sidebar (2 patches)
```

**DevTools integration:**

```go
// vango.json
{
    "devTools": {
        "signalLogging": true,
        "dependencyGraph": true,
        "performanceMetrics": true
    }
}
```

Opens a browser panel showing:
- All signals and current values
- Dependency graph visualization
- Re-render triggers and timing
- Time-travel through signal history

### 7.11 When to Use Each Pattern

| Scenario | Pattern | Example |
|----------|---------|---------|
| Form input | Local Signal | `input := vango.NewSignal("")` |
| UI state (modals, tabs) | Local Signal | `isOpen := vango.NewSignal(false)` |
| Shopping cart | NewSharedSignal | `var Cart = vango.NewSharedSignal(...)` |
| User preferences | Pref | `var Theme = pref.New("theme", "system")` |
| Filter/search state | NewSharedSignal | `var Filter = vango.NewSharedSignal(...)` |
| Async data loading | NewResource | `user := vango.NewResource(...)` |
| Derived calculations | NewSharedMemo | `var Total = vango.NewSharedMemo(...)` |
| Real-time presence | NewGlobalSignal | `var OnlineUsers = vango.NewGlobalSignal(...)` |
| Collaborative editing | NewGlobalSignal | `var DocContent = vango.NewGlobalSignal(...)` |

### 7.12 Anti-Patterns to Avoid

```go
// ❌ DON'T: Create signals outside component context
var badSignal = vango.NewSignal(0)  // Package-level local signal

func MyComponent() vango.Component {
    return vango.Func(func() *vango.VNode {
        // badSignal is shared across ALL instances!
        return Text(fmt.Sprintf("%d", badSignal.Get()))
    })
}

// ✅ DO: Create local signals inside the component
func MyComponent() vango.Component {
    return vango.Func(func() *vango.VNode {
        goodSignal := vango.NewSignal(0)  // Per-instance
        return Text(fmt.Sprintf("%d", goodSignal.Get()))
    })
}

// ✅ OR: Use NewSharedSignal explicitly for intentional sharing
var intentionallyShared = vango.NewSharedSignal(0)
```

```go
// ❌ DON'T: Read signals conditionally
func BadComponent() vango.Component {
    return vango.Func(func() *vango.VNode {
        if someCondition {
            value := mySignal.Get()  // Subscription depends on condition!
        }
        return Div()
    })
}

// ✅ DO: Read signals unconditionally, use value conditionally
func GoodComponent() vango.Component {
    return vango.Func(func() *vango.VNode {
        value := mySignal.Get()  // Always subscribe
        if someCondition {
            return Div(Text(fmt.Sprintf("%d", value)))
        }
        return Div()
    })
}
```

```go
// ❌ DON'T: Heavy computation in signal update
items.Update(func(i []Item) []Item {
    // This runs synchronously, blocking the event loop
    result := veryExpensiveOperation(i)  // Bad!
    return result
})

// ✅ DO: Use Effect for heavy async work
vango.Effect(func() vango.Cleanup {
    ctx := vango.UseCtx()
    input := items.Get() // Tracked: re-run effect when items changes
    go func(input []Item) {
        result := veryExpensiveOperation(input)
        ctx.Dispatch(func() {
            processedItems.Set(result)
        })
    }(input)
    return nil
})
```

---

## 8. Interaction Primitives

Vango provides a spectrum of interaction patterns, from simple server events to rich client-side behaviors. The key insight is that **the server doesn't need to know about every drag pixel—it only needs to know the final result**.

### 8.1 Design Philosophy

Vango uses a four-tier interaction model:

```
┌───────────────────────────────────────────────────────────────────────────────────────────┐
│                               INTERACTION SPECTRUM                                          │
├───────────────────┬─────────────────────┬──────────────────────────┬───────────────────────┤
│   Server Events   │    Client Hooks     │       WASM Islands        │       JS Islands      │
├───────────────────┼─────────────────────┼──────────────────────────┼───────────────────────┤
│   OnClick         │   Hook("Sortable")  │  WASMWidget("canvas")     │  JSIsland("editor")   │
│   OnSubmit        │   Hook("Draggable") │  WASMWidget("webgl")      │  JSIsland("chart")    │
│   OnInput         │   Hook("Tooltip")   │  WASMWidget("sim")        │  JSIsland("map")      │
├───────────────────┼─────────────────────┼──────────────────────────┼───────────────────────┤
│   Server runs     │   Client runs the   │  Client runs tight loop;  │  Third-party/client   │
│   the handler     │   behavior, server  │  server receives commits  │  logic + bridge       │
│                   │   owns state        │  (semantic events)        │                       │
├───────────────────┼─────────────────────┼──────────────────────────┼───────────────────────┤
│   ~0 client KB    │   ~15KB (bundled)   │  12KB + partial WASM      │  Variable (lazy)      │
├───────────────────┼─────────────────────┼──────────────────────────┼───────────────────────┤
│   50-100ms        │   60fps interaction │  60fps interaction        │  60fps interaction    │
│   latency OK      │   + single event    │  + semantic commits       │  + bridge events      │
└───────────────────┴─────────────────────┴──────────────────────────┴───────────────────────┘
```

**When to use each tier:**

| Tier | Use When | Examples |
|------|----------|----------|
| **Server Events** | Latency is acceptable (buttons, forms, navigation) | Click handlers, form submission, toggles |
| **Client Hooks** | Need 60fps feedback but server owns state | Drag-and-drop, sortable lists, tooltips, dropdowns |
| **WASM Islands** | Need a tight client-side loop with Go logic | Canvas drawing, physics simulation, heavy local compute |
| **JS Islands** | Full third-party library or complex client logic | Rich text editors, charts, maps, video players |

**The Hook pattern** is the key innovation here. It delegates client-side interaction physics to specialized JavaScript while keeping state management on the server. This gives you:

- **60fps animations** during drag operations
- **Zero network traffic** during interactions
- **Simple Go code** (just handle the final result)
- **Graceful failure** (interaction works, sync fails gracefully)

### 8.2 Client Hooks

Client Hooks are the recommended way to handle interactions that need 60fps visual feedback (drag-and-drop, sortable lists, tooltips, etc.). The hook handles all client-side animation and behavior, then sends a single event to the server when the interaction completes.

**Hook Interception**: Applying a `Hook(...)` to an element automatically marks it for event interception. The runtime ensures the generated element includes `hook` in its `data-ve` attribute, and the thin client uses the specialized `Hook` attribute to manage the lifecycle of the client-side behavior.


#### The Hook Attribute

```go
// Attach a hook to any element
Div(
    Hook("Sortable", map[string]any{
        "group":     "tasks",
        "animation": 150,
        "handle":    ".drag-handle",
    }),

    // Handle events from the hook
    OnEvent("reorder", func(e vango.HookEvent) {
        // This fires ONCE when drag ends, not during drag
        fromIndex := e.Int("fromIndex")
        toIndex := e.Int("toIndex")
        db.Tasks.Reorder(fromIndex, toIndex)
    }),

    // Children...
)
```

#### Why Hooks Instead of Server Events

Consider drag-and-drop. With server events:

```
User drags card → Stream of dragover events → Server processes each →
Client predicts DOM changes → Server sends patches → Reconciliation

Problems:
- Network during drag (latency spikes visible)
- Complex prediction logic on client
- Server CPU processing drag events
- Not truly 60fps
```

With hooks:

```
User drags card → Hook handles animation at 60fps →
User drops card → ONE event to server → Server updates DB

Benefits:
- Zero network during drag
- Native 60fps from specialized library
- Simple server code (just handle result)
- Works even with high latency
```

#### HookEvent API

```go
type HookEvent struct {
    Name string         // Event name (e.g., "reorder", "drop")
    Data map[string]any // Event data from the hook
}

// Type-safe accessors
func (e HookEvent) String(key string) string
func (e HookEvent) Int(key string) int
func (e HookEvent) Float(key string) float64
func (e HookEvent) Bool(key string) bool
func (e HookEvent) Strings(key string) []string
func (e HookEvent) Raw(key string) any
```

#### Complete Example: Sortable List

```go
func SortableList(items []Item, onReorder func(fromIdx, toIdx int)) *vango.VNode {
    return Ul(
        Class("sortable-list"),

        // Hook handles all drag animation at 60fps
        Hook("Sortable", map[string]any{
            "animation":  150,
            "ghostClass": "sortable-ghost",
        }),

        // Only fires when drag completes
        OnEvent("reorder", func(e vango.HookEvent) {
            onReorder(e.Int("fromIndex"), e.Int("toIndex"))
        }),

        Range(items, func(item Item, i int) *vango.VNode {
            return Li(
                Key(item.ID),
                Data("id", item.ID),
                Text(item.Name),
            )
        }),
    )
}
```

#### Complete Example: Kanban Board

```go
func KanbanBoard(columns []Column) vango.Component {
    return vango.Func(func() *vango.VNode {
        return Div(Class("kanban-board"),
            Range(columns, func(col Column) *vango.VNode {
                return Div(
                    Key(col.ID),
                    Class("kanban-column"),
                    Data("column-id", col.ID),

                    // Hook handles all drag visuals at 60fps
                    Hook("Sortable", map[string]any{
                        "group":      "cards",
                        "animation":  150,
                        "ghostClass": "card-ghost",
                    }),

                    // Only fires when drag ends
                    OnEvent("reorder", func(e vango.HookEvent) {
                        cardID := e.String("id")
                        toColumn := e.String("toColumn")
                        toIndex := e.Int("newIndex")

                        // Update database
                        err := db.Cards.Move(cardID, toColumn, toIndex)
                        if err != nil {
                            // Hook can revert the visual change
                            e.Revert()
                            toast.Error(ctx, "Failed to move card")
                        }
                    }),

                    H3(Class("column-title"), Text(col.Name)),

                    Div(Class("card-list"),
                        Range(col.Cards, func(card Card, i int) *vango.VNode {
                            return Div(
                                Key(card.ID),
                                Data("id", card.ID),
                                Class("kanban-card"),
                                CardContent(card),
                            )
                        }),
                    ),
                )
            }),
        )
    })
}
```

**What happens during a drag:**

1. User starts dragging a card
2. SortableJS (bundled in thin client) handles animation at 60fps
3. Cards shuffle smoothly as cursor moves
4. User drops the card
5. Client sends ONE event: `{id: "card-123", toColumn: "done", newIndex: 2}`
6. Server updates database
7. Server re-renders and sends confirmation patch
8. If server fails, client can revert (`e.Revert()`)

**Network traffic during drag:** Zero.
**Frames per second:** 60.
**Go code complexity:** Minimal.

### 8.3 Optimistic Updates

For simple interactions like button clicks and toggles, optimistic updates provide instant visual feedback while the server processes the action.

> **Note:** For complex interactions like drag-and-drop, use [Client Hooks](#82-client-hooks) instead. Hooks handle the animation natively and don't need prediction logic.

#### When to Use Optimistic Updates

| Use Case | Recommended Approach |
|----------|---------------------|
| Toggle checkbox | Optimistic update |
| Like button | Optimistic update |
| Increment counter | Optimistic update |
| Delete item (simple) | Optimistic update |
| Drag-and-drop | Client Hook (§8.2) |
| Sortable list | Client Hook (§8.2) |
| Complex animations | Client Hook (§8.2) |

#### Simple Optimistic Attributes

```go
// Toggle a class optimistically
Button(
    OnClick(toggleComplete),
    OptimisticClass("completed", !task.Done),  // Toggle class instantly
    Text("Complete"),
)

// Update text optimistically
Button(
    OnClick(incrementLikes),
    OptimisticText(fmt.Sprintf("%d", likes + 1)),  // Show new count instantly
    Textf("%d likes", likes),
)

// Toggle attribute optimistically
Button(
    OnClick(toggleDisabled),
    OptimisticAttr("disabled", "true"),
    Text("Submit"),
)
```

#### How It Works

```
1. User clicks button
2. Client applies optimistic change immediately (class, text, etc.)
3. Client sends event to server
4. Server processes and sends confirmation
5a. If success: Server patch matches optimistic state (no visual change)
5b. If failure: Server patch reverts to original state
```

#### Complete Example: Task Toggle

```go
func TaskItem(task Task) *vango.VNode {
    return Li(
        Key(task.ID),
        Class("task-item"),
        ClassIf(task.Done, "completed"),

        Input(
            Type("checkbox"),
            Checked(task.Done),
            OnChange(func() {
                db.Tasks.Toggle(task.ID)
            }),
            // Toggle the parent's class optimistically
            OptimisticParentClass("completed", !task.Done),
        ),

        Span(Text(task.Title)),
    )
}
```

#### Complete Example: Like Button

```go
func LikeButton(postID string, likes int, userLiked bool) *vango.VNode {
    return Button(
        Class("like-button"),
        ClassIf(userLiked, "liked"),

        OnClick(func() {
            if userLiked {
                db.Posts.Unlike(postID)
            } else {
                db.Posts.Like(postID)
            }
        }),

        // Optimistic visual feedback
        OptimisticClass("liked", !userLiked),
        OptimisticText(func() string {
            if userLiked {
                return fmt.Sprintf("%d", likes-1)
            }
            return fmt.Sprintf("%d", likes+1)
        }()),

        Icon("heart"),
        Span(Textf("%d", likes)),
    )
}
```

#### Signal-Based Optimistic Updates

For more control, update signals optimistically with manual rollback:

```go
func TaskList() vango.Component {
    return vango.Func(func() *vango.VNode {
        ctx := vango.UseCtx()
        tasks := vango.NewSignal(initialTasks)

        toggleTask := func(taskID string) {
            // Capture original state for rollback
            originalTasks := append([]Task(nil), tasks.Get()...) // copies slice + elements (no shared backing array)
            // If Task contains pointer/map/slice fields, deep-copy those too to avoid aliasing.

            // Optimistically update signal (triggers re-render immediately)
            tasks.Update(func(t []Task) []Task {
                result := make([]Task, len(t))
                copy(result, t)

                for i := range result {
                    if result[i].ID == taskID {
                        result[i].Done = !result[i].Done
                        break
                    }
                }
                return result
            })

            // Server action
            go func() {
                err := db.Tasks.Toggle(taskID)
                if err != nil {
                    // Marshal back to the session loop and revert atomically.
                    ctx.Dispatch(func() {
                        vango.TxNamed("tasks:toggle_rollback", func() {
                            tasks.Set(originalTasks)
                            toast.Error(ctx, "Failed to update task")
                        })
                    })
                }
            }()
        }

        return Ul(
            Range(tasks.Get(), func(task Task, i int) *vango.VNode {
                return Li(
                    Key(task.ID),
                    ClassIf(task.Done, "completed"),
                    OnClick(func() { toggleTask(task.ID) }),
                    Text(task.Title),
                )
            }),
        )
    })
}
```

### 8.4 Standard Hooks

Vango bundles a set of standard hooks for common interaction patterns. These are included in the thin client (~15KB total with hooks).

#### Available Standard Hooks

| Hook | Purpose | Events |
|------|---------|--------|
| `Sortable` | Drag-to-reorder lists and grids | `reorder` |
| `Draggable` | Free-form element dragging | `dragend` |
| `Droppable` | Drop zones for draggable elements | `drop` |
| `Resizable` | Resize handles on elements | `resize` |
| `Tooltip` | Hover tooltips | (none - visual only) |
| `Dropdown` | Click-outside-to-close behavior | `close` |
| `Collapsible` | Expand/collapse with animation | `toggle` |

#### Sortable Hook

For drag-to-reorder lists:

```go
Ul(
    Class("task-list"),

    Hook("Sortable", map[string]any{
        "animation":  150,           // Animation duration (ms)
        "handle":     ".drag-handle", // Optional: restrict to handle
        "ghostClass": "ghost",       // Class for placeholder
        "group":      "tasks",       // Allow drag between lists with same group
    }),

    OnEvent("reorder", func(e vango.HookEvent) {
        fromIndex := e.Int("fromIndex")
        toIndex := e.Int("toIndex")
        db.Tasks.Reorder(fromIndex, toIndex)
    }),

    Range(tasks, TaskItem),
)
```

**Sortable Event Data:**
```go
e.Int("fromIndex")      // Original index
e.Int("toIndex")        // New index
e.String("id")          // data-id of moved element
e.String("fromGroup")   // Group moved from (if cross-list)
e.String("toGroup")     // Group moved to (if cross-list)
```

#### Draggable Hook

For free-form dragging:

```go
Div(
    Class("floating-panel"),

    Hook("Draggable", map[string]any{
        "handle":  ".panel-header",  // Drag handle
        "bounds":  "parent",         // Constrain to parent
        "axis":    "both",           // "x", "y", or "both"
        "grid":    []int{10, 10},    // Snap to grid
    }),

    OnEvent("dragend", func(e vango.HookEvent) {
        x := e.Int("x")
        y := e.Int("y")
        db.Panels.UpdatePosition(panelID, x, y)
    }),

    PanelContent(),
)
```

#### Droppable Hook

For drop zones:

```go
Div(
    Class("upload-zone"),

    Hook("Droppable", map[string]any{
        "accept":     ".draggable-file",  // CSS selector
        "hoverClass": "drag-over",        // Class when dragging over
    }),

    OnEvent("drop", func(e vango.HookEvent) {
        itemID := e.String("id")
        handleDrop(itemID)
    }),

    Text("Drop files here"),
)
```

#### Resizable Hook

For resizable elements:

```go
Div(
    Class("resizable-panel"),
    Style(fmt.Sprintf("width: %dpx; height: %dpx", width, height)),

    Hook("Resizable", map[string]any{
        "handles":   "e,s,se",       // Which edges: n,e,s,w,ne,se,sw,nw
        "minWidth":  200,
        "maxWidth":  800,
        "minHeight": 100,
    }),

    OnEvent("resize", func(e vango.HookEvent) {
        width := e.Int("width")
        height := e.Int("height")
        db.Panels.UpdateSize(panelID, width, height)
    }),

    PanelContent(),
)
```

#### Tooltip Hook

For hover tooltips (visual only, no events):

```go
Button(
    Hook("Tooltip", map[string]any{
        "content":   "Click to save",
        "placement": "top",          // top, bottom, left, right
        "delay":     200,            // Delay before showing (ms)
    }),

    Text("Save"),
)

// Dynamic tooltip content
Button(
    Hook("Tooltip", map[string]any{
        "content":   fmt.Sprintf("Last saved: %s", lastSaved.Format(time.Kitchen)),
        "placement": "bottom",
    }),

    Text("Status"),
)
```

#### Dropdown Hook

For click-outside-to-close behavior:

```go
func DropdownMenu() vango.Component {
    return vango.Func(func() *vango.VNode {
        open := vango.NewSignal(false)

        return Div(
            Class("dropdown"),

            Button(OnClick(open.Toggle), Text("Menu")),

            If(open.Get(),
                Div(
                    Class("dropdown-content"),

                    Hook("Dropdown", map[string]any{
                        "closeOnEscape": true,
                        "closeOnClick":  true,  // Close when clicking inside
                    }),

                    OnEvent("close", func(e vango.HookEvent) {
                        open.Set(false)
                    }),

                    MenuItem("Edit"),
                    MenuItem("Delete"),
                ),
            ),
        )
    })
}
```

#### Collapsible Hook

For animated expand/collapse:

```go
Div(
    Class("accordion-item"),

    Button(
        Class("accordion-header"),
        OnClick(func() { /* toggle handled by hook */ }),
        Text("Section Title"),
    ),

    Div(
        Class("accordion-content"),

        Hook("Collapsible", map[string]any{
            "open":     isOpen,
            "duration": 200,
        }),

        OnEvent("toggle", func(e vango.HookEvent) {
            isNowOpen := e.Bool("open")
            // Update state if needed
        }),

        SectionContent(),
    ),
)
```

### 8.5 Custom Hooks

For behaviors not covered by standard hooks, you can define custom hooks in JavaScript.

#### Creating a Custom Hook

```javascript
// public/js/hooks.js
export default {
    // Custom color picker hook
    ColorPicker: {
        mounted(el, config, pushEvent) {
            // Initialize when element mounts
            this.picker = new Pickr({
                el: el,
                default: config.color || '#000000',
                components: {
                    preview: true,
                    hue: true,
                },
            });

            // Send events to server
            this.picker.on('change', (color) => {
                pushEvent('color-changed', {
                    color: color.toHEXA().toString()
                });
            });
        },

        updated(el, config, pushEvent) {
            // Called when config changes
            if (config.color) {
                this.picker.setColor(config.color);
            }
        },

        destroyed(el) {
            // Cleanup when element unmounts
            this.picker.destroy();
        }
    },

    // Custom chart hook
    Chart: {
        mounted(el, config, pushEvent) {
            this.chart = new Chart(el.getContext('2d'), {
                type: config.type,
                data: config.data,
            });
        },

        updated(el, config, pushEvent) {
            this.chart.data = config.data;
            this.chart.update();
        },

        destroyed(el) {
            this.chart.destroy();
        }
    }
};
```

#### Registering Custom Hooks

```json
// vango.json
{
    "hooks": "./public/js/hooks.js"
}
```

Or programmatically:

```go
// main.go
func main() {
    app := vango.New()
    app.RegisterHooks("./public/js/hooks.js")
    app.Run()
}
```

#### Using Custom Hooks

```go
// Use just like standard hooks
Div(
    Hook("ColorPicker", map[string]any{
        "color": currentColor,
    }),

    OnEvent("color-changed", func(e vango.HookEvent) {
        newColor := e.String("color")
        db.Settings.SetColor(newColor)
    }),
)
```

#### Hook Lifecycle

| Method | When Called |
|--------|-------------|
| `mounted(el, config, pushEvent)` | Element added to DOM |
| `updated(el, config, pushEvent)` | Hook config changed |
| `destroyed(el)` | Element removed from DOM |

**Parameters:**
- `el` — The DOM element the hook is attached to
- `config` — The configuration map passed from Go
- `pushEvent(name, data)` — Function to send events to server

#### pushEvent API

```javascript
// Send event to server
pushEvent('event-name', {
    key: 'value',
    number: 42,
    array: [1, 2, 3]
});

// In Go, handle with OnEvent
OnEvent("event-name", func(e vango.HookEvent) {
    value := e.String("key")
    number := e.Int("number")
    array := e.Strings("array")  // Returns []string
})
```

#### Revert Capability

Hooks can support revert for failed server operations. Revert works via a **control message** from server to client — functions cannot be serialized, so the hook instance maintains its own revert state locally.

**How it works:**
1. The hook tracks the previous state internally (e.g., `this.lastColor`).
2. When pushing an event, the hook registers a revert callback keyed by the event's HID.
3. If the server calls `e.Revert()`, a control message is sent to the client targeting that HID.
4. The client invokes the registered revert callback for that hook instance.

```javascript
// In hook
ColorPicker: {
    mounted(el, config, pushEvent) {
        this.picker = new Pickr({...});
        this.lastColor = config.color;

        this.picker.on('change', (color) => {
            const newColor = color.toHEXA().toString();
            const previousColor = this.lastColor;

            // Register revert callback with the thin client
            this.registerRevert(() => {
                this.picker.setColor(previousColor);
            });

            pushEvent('color-changed', { color: newColor });
            this.lastColor = newColor;
        });
    },

    // Called by thin client when server sends revert control message
    revert() {
        if (this._revertCallback) {
            this._revertCallback();
            this._revertCallback = null;
        }
    },

    registerRevert(fn) {
        this._revertCallback = fn;
    }
}
```

```go
// In Go - revert on error
OnEvent("color-changed", func(e vango.HookEvent) {
    ctx := vango.UseCtx()
    err := db.Settings.SetColor(e.String("color"))
    if err != nil {
        e.Revert()  // Sends control message to client hook instance
        toast.Error(ctx, "Failed to save color")
    }
})
```

**Wire protocol:** `e.Revert()` sends a `HOOK_REVERT` control patch (see §22.3) targeting the hook's HID. The thin client dispatches this to the hook instance's `revert()` method.

### 8.6 Keyboard Navigation

> **Note:** Keyboard navigation uses server events since latency is acceptable for key presses.
>
> **Naming note:** Keyboard shortcuts use `Hotkey()` (not `Key()`) to avoid collision with the VDOM identity helper `Key(id)` used for list reconciliation.

#### Scoped Keyboard Handlers

```go
func BoardPage() vango.Component {
    return vango.Func(func() *vango.VNode {
        selected := vango.NewSignal[*Card](nil)

        return KeyboardScope(
            // Arrow navigation
            Hotkey("ArrowDown", func() { selectNext(selected) }),
            Hotkey("ArrowUp", func() { selectPrev(selected) }),
            Hotkey("ArrowRight", func() { moveToNextColumn(selected) }),
            Hotkey("ArrowLeft", func() { moveToPrevColumn(selected) }),

            // Actions
            Hotkey("Enter", func() { openCard(selected.Get()) }),
            Hotkey("e", func() { editCard(selected.Get()) }),
            Hotkey("d", func() { deleteCard(selected.Get()) }),
            Hotkey("Escape", func() { selected.Set(nil) }),

            // With modifiers
            Hotkey("Cmd+k", openCommandPalette),
            Hotkey("Cmd+Enter", saveAndClose),
            Hotkey("Shift+?", showKeyboardShortcuts),

            // The actual content
            BoardContent(selected),
        )
    })
}
```

#### Hotkey Modifiers

```go
// Available modifiers
Hotkey("Cmd+k", handler)       // Cmd on Mac, Ctrl on Windows/Linux
Hotkey("Ctrl+k", handler)      // Always Ctrl
Hotkey("Alt+k", handler)       // Alt/Option
Hotkey("Shift+k", handler)     // Shift
Hotkey("Cmd+Shift+k", handler) // Combinations

// Special keys
Hotkey("Enter", handler)
Hotkey("Escape", handler)
Hotkey("Tab", handler)
Hotkey("Backspace", handler)
Hotkey("Delete", handler)
Hotkey("Space", handler)
Hotkey("ArrowUp", handler)
Hotkey("ArrowDown", handler)
Hotkey("ArrowLeft", handler)
Hotkey("ArrowRight", handler)
```

#### Global Keyboard Shortcuts

```go
// Register global shortcuts (work anywhere on the page)
func App() vango.Component {
    return vango.Func(func() *vango.VNode {
        // Global shortcuts
        vango.GlobalHotkey("Cmd+k", openCommandPalette)
        vango.GlobalHotkey("/", focusSearch)
        vango.GlobalHotkey("?", showHelp)
        vango.GlobalHotkey("Escape", closeModals)

        return Div(
            Header(),
            Main(children...),
            Footer(),
        )
    })
}
```

#### Nested Scopes

```go
// Outer scope
KeyboardScope(
    Hotkey("Escape", closePanel),

    // Inner scope (takes priority when focused)
    KeyboardScope(
        Hotkey("Escape", clearSelection),  // This fires first
        Hotkey("Enter", confirmSelection),

        SelectionList(...),
    ),
)
```

### 8.7 Focus Management

```go
// Auto-focus on mount
Input(
    Autofocus(),
    Type("text"),
)

// Programmatic focus (server-driven)
//
// In server-driven mode, the server cannot hold DOM handles. Prefer `Autofocus()`
// on elements that appear/mount when UI state changes (e.g. when a modal opens).
func SearchModal() vango.Component {
    return vango.Func(func() *vango.VNode {
        open := vango.NewSignal(true) // Example state; typically driven by your UI.
        if !open.Get() {
            return vango.Empty()
        }
        return Div(Class("modal"),
            Input(
                Type("search"),
                Placeholder("Search..."),
                Autofocus(),
            ),
        )
    })
}

// Focus trap (keep focus within modal)
FocusTrap(
    Div(Class("modal"),
        Input(Type("text")),
        Button(Text("Cancel")),
        Button(Text("Submit")),
    ),
)
```

### 8.8 Scroll Management

```go
// Scroll into view
func scrollToCard(cardID string) {
    vango.ScrollIntoView(
        fmt.Sprintf("[data-card-id='%s']", cardID),
        ScrollConfig{
            Behavior: "smooth",      // "smooth" | "instant"
            Block:    "center",      // "start" | "center" | "end"
            Inline:   "nearest",
        },
    )
}

// Scroll position tracking
func InfiniteList() vango.Component {
    return vango.Func(func() *vango.VNode {
        items := vango.NewSignal(initialItems)
        loading := vango.NewSignal(false)

        return Div(
            Class("list-container"),
            OnScroll(func(e vango.ScrollEvent) {
                // Load more when near bottom
                if e.ScrollTop + e.ClientHeight >= e.ScrollHeight - 100 {
                    if !loading.Get() {
                        loading.Set(true)
                        loadMore(items, loading)
                    }
                }
            }),

            Range(items.Get(), ItemComponent),
            If(loading.Get(), Spinner()),
        )
    })
}

// Preserve scroll position
func PreserveScroll(children ...any) *vango.VNode {
    return Div(
        ScrollRestore("list-scroll"),  // Key for storage
        children...,
    )
}
```

### 8.9 Touch Gestures (Mobile)

```go
// Swipe actions
Div(
    Swipeable(SwipeConfig{
        OnSwipeLeft: func() {
            showActions()
        },
        OnSwipeRight: func() {
            archive()
        },
        Threshold: 50,  // Minimum pixels to trigger
    }),

    ListItem(item),
)

// Long press
Div(
    OnLongPress(func() {
        showContextMenu()
    }, 500),  // 500ms threshold

    CardContent(card),
)

// Pinch zoom (for WASM components)
Canvas(
    OnPinch(func(e vango.PinchEvent) {
        setZoom(e.Scale)
    }),
)
```

### 8.10 Thin Client Implementation

The thin client handles events, patches, and hooks:

```javascript
// Approximate size breakdown
// Base thin client: ~12KB
// + Standard hooks (Sortable, Draggable, etc.): ~3KB
// + Simple optimistic updates: ~0.5KB
// + Keyboard/scroll utilities: ~0.5KB
// Total: ~16KB gzipped

class VangoClient {
    hooks = {};        // Registered hook implementations
    hookInstances = {} // Active hook instances per element

    constructor() {
        // Register standard hooks
        this.registerHook('Sortable', SortableHook);
        this.registerHook('Draggable', DraggableHook);
        this.registerHook('Droppable', DroppableHook);
        this.registerHook('Resizable', ResizableHook);
        this.registerHook('Tooltip', TooltipHook);
        this.registerHook('Dropdown', DropdownHook);
        this.registerHook('Collapsible', CollapsibleHook);
    }

    // Hook lifecycle management
    mountHook(el, hookName, config) {
        const Hook = this.hooks[hookName];
        if (!Hook) {
            console.warn(`Unknown hook: ${hookName}`);
            return;
        }

        const hid = el.dataset.hid;
        const pushEvent = (event, data) => {
            this.sendEvent(HOOK_EVENT, hid, { event, ...data });
        };

        const instance = new Hook();
        instance.mounted(el, config, pushEvent);
        this.hookInstances[hid] = { instance, hookName };
    }

    updateHook(el, config) {
        const hid = el.dataset.hid;
        const entry = this.hookInstances[hid];
        if (entry) {
            entry.instance.updated(el, config, (event, data) => {
                this.sendEvent(HOOK_EVENT, hid, { event, ...data });
            });
        }
    }

    destroyHook(hid) {
        const entry = this.hookInstances[hid];
        if (entry) {
            const el = document.querySelector(`[data-hid="${hid}"]`);
            entry.instance.destroyed(el);
            delete this.hookInstances[hid];
        }
    }

    // Simple optimistic updates (class, text, attr)
    applyOptimistic(el, type, value) {
        switch (type) {
            case 'class':
                el.classList.toggle(value.class, value.add);
                break;
            case 'text':
                el.textContent = value;
                break;
            case 'attr':
                el.setAttribute(value.name, value.value);
                break;
        }
    }

    // Keyboard scopes (server events)
    keyboardScopes = [];

    attachKeyboardListeners() {
        document.addEventListener('keydown', (e) => {
            const key = this.normalizeKey(e);

            // Check scopes from innermost to outermost
            for (const scope of [...this.keyboardScopes].reverse()) {
                if (scope.handlers[key]) {
                    e.preventDefault();
                    this.sendEvent(KEY, scope.hid, { key });
                    return;
                }
            }
        });
    }
}

// Standard hook implementation (simplified)
class SortableHook {
    mounted(el, config, pushEvent) {
        this.sortable = new Sortable(el, {
            animation: config.animation || 150,
            handle: config.handle,
            ghostClass: config.ghostClass || 'sortable-ghost',
            group: config.group,
            onEnd: (evt) => {
                pushEvent('reorder', {
                    id: evt.item.dataset.id,
                    fromIndex: evt.oldIndex,
                    toIndex: evt.newIndex,
                    fromGroup: evt.from.dataset.group,
                    toGroup: evt.to.dataset.group,
                });
            }
        });
    }

    updated(el, config, pushEvent) {
        // Update config if needed
    }

    destroyed(el) {
        this.sortable?.destroy();
    }
}
```

**Key architectural difference from the old design:**

| Old (Complex Predictions) | New (Hook Pattern) |
|--------------------------|-------------------|
| Client predicts DOM changes | Hook library handles DOM |
| Server streams drag events | Server only gets final result |
| Complex reconciliation logic | No reconciliation needed |
| ~2KB prediction engine | ~3KB proven libraries |

### 8.11 When to Use WASM Instead

Server events and client hooks handle most cases. Use WASM components when you need:

| Scenario | Why WASM |
|----------|----------|
| Physics simulation | Continuous calculation, not event-driven |
| Canvas drawing | <16ms feedback required |
| Complex gestures | Multi-touch, pressure, custom recognition |
| Heavy data processing | Filter/sort large datasets client-side |
| Offline computation | Work without server connection |

In Hybrid mode:
- Use `WASMWidget(...)` for Tier 1 widget islands (opaque subtree; semantic commit events back to the server).
- Use `ClientComponent(...)` (or `ClientRequired(...)`) only when you need full Vango component semantics in the browser.

```go
// Example: Physics-based graph (WASM)
func ForceGraph(nodes []Node, edges []Edge) *vango.VNode {
    return vango.WASMWidget(
        "force-graph",
        vango.WASMModule("/wasm/force-graph.wasm"),
        map[string]any{
            "nodes": nodes,
            "edges": edges,
        },
    )
}
```

---

## 9. Routing & Navigation

### 9.1 File-Based Routing

```
app/routes/
├── routes_gen.go          # GENERATED — explicit route registration
├── _layout.go             # package routes — wraps / and subroutes
├── _middleware.go         # optional — directory middleware stack
├── index.go               # package routes — / (IndexPage)
├── about.go               # package routes — /about (AboutPage)
├── projects/
│   ├── index.go           # package projects — /projects (IndexPage)
│   └── [id].go            # package projects — /projects/:id (ShowPage)
└── api/
    └── health.go          # package api — GET /api/health (HealthGET)
```

Vango generates a deterministic `routes_gen.go` and requires explicit registration (no `init()` side effects):

```go
// main.go
routes.Register(app.Router())
```


### 9.2 Page Components

**Route handlers vs reactive renders (normative):**

- **Route handlers** (page functions like `ShowPage`) are called once per navigation. Blocking I/O (DB queries, API calls) is permitted here because the function executes once and returns a VNode tree.
- **Reactive render functions** (`vango.Func`) may re-execute on any signal change. These MUST be pure (no blocking I/O, no side effects) per §3.1.2. For data loading in reactive components, use `Resource`, `GoLatest`, or Effects.

The distinction: if your function takes `ctx vango.Ctx` and params and returns `*vango.VNode` directly, it's a route entry point. If it returns `vango.Component` (wrapping `vango.Func`), the inner render function must be pure.

```go
// app/routes/projects/[id].go
package projects

import (
    . "vango/el"
    "vango"
)

// Params are automatically typed from filename
type Params struct {
    ID int `param:"id"`
}

// ShowPage is a ROUTE HANDLER — blocking I/O is allowed here
func ShowPage(ctx vango.Ctx, p Params) *vango.VNode {
    project := db.Projects.FindByID(p.ID)  // OK: runs once per navigation

    return Div(Class("project-page"),
        H1(Text(project.Name)),
        P(Text(project.Description)),
        TaskBoard(project.Tasks),
    )
}
```


### 9.3 Layouts

```go
// app/routes/_layout.go
package routes

func Layout(ctx vango.Ctx, children vango.Slot) *vango.VNode {
    return Html(
        Head(
            Title(Text("My App")),
            Link(Rel("stylesheet"), Href("/styles.css")),
        ),
        Body(
            Navbar(ctx.User()),
            Main(Class("container"), children),
            Footer(),
            VangoScripts(),  // Injects thin client
        ),
    )
}
```


### 9.4 Navigation

```go
// Programmatic navigation
func handleSave(ctx vango.Ctx) {
    project := saveProject()
    ctx.Navigate("/projects/" + project.ID)
}

// Link component
A(Href("/projects/123"), Text("View Project"))

// With prefetch (loads data before navigation)
A(
    Href("/projects/123"),
    Prefetch(),  // Preloads on hover
    Text("View Project"),
)
```


### 9.5 How Navigation Works

**Server-Driven Mode:**
```
1. User clicks link to /projects/123
2. Thin client intercepts, sends NAVIGATE event
3. Server:
   - Matches route
   - Mounts new page component
   - Renders to VNode
   - Diffs against current page
   - Sends navigation envelope: [URL_UPDATE, patches...]
4. Client updates URL (pushState/replaceState)
5. Client applies patches
```

#### 9.5.1 URL-Only Updates vs Navigation (Normative)

Vango has two distinct URL update mechanisms:

- **Query-only URL patches** (`URL_PUSH` / `URL_REPLACE`): MUST update only the current route’s **query parameters** and MUST NOT remount the route. These patches MAY be sent alongside normal DOM patches from the same transaction when UI depends on the updated query state (e.g. filters).
- **Navigation** (`NAVIGATE` event): MUST change the active route (path + optional query), MUST remount the route component tree, and MUST send a navigation envelope that includes the URL update plus the DOM patches needed to transition the current page to the new page.

Progressive enhancement requirements:
- If the WebSocket is not connected/healthy, or if an anchor does not have a Vango click handler (`data-ve` does not include `click`), the browser MUST perform native navigation.
- If patch application fails (DOM mismatch, hook error), the client MUST self-heal by hard reloading to the target URL (or an equivalent full resync mechanism).

No full page reload, no WASM download, minimal data transfer.


---

## 10. Data & APIs

### 10.1 Direct Database Access

In server-driven mode, components have direct access to backend:

```go
func UserList() vango.Component {
    return vango.Func(func() *vango.VNode {
        rctx := vango.UseCtx()
        users := vango.NewSignal([]User{})
        search := vango.NewSignal("")

        vango.Effect(func() vango.Cleanup {
            q := search.Get() // Tracked
            cctx, cancel := context.WithCancel(rctx.StdContext())

            go func(q string) {
                // Best-effort cancellation: if your DB supports context, pass cctx.
                results, err := db.Users.Search(cctx, q)
                if err != nil && cctx.Err() != nil {
                    return // cancelled; ignore
                }
                rctx.Dispatch(func() {
                    if cctx.Err() != nil {
                        return // cancelled; ignore
                    }
                    if search.Peek() != q {
                        return // stale; ignore
                    }
                    if err != nil {
                        users.Set([]User{})
                        return
                    }
                    users.Set(results)
                })
            }(q)

            return cancel
        })

        return Div(
            Input(
                Type("search"),
                Value(search.Get()),
                OnInput(search.Set),
                Placeholder("Search users..."),
            ),
            Ul(
                Range(users.Get(), func(u User, i int) *vango.VNode {
                    return Li(Key(u.ID), Text(u.Name))
                }),
            ),
        )
    })
}
```

### 10.2 API Routes

For external clients (mobile apps, third parties):

```go
// app/routes/api/projects.go
package api

func GET(ctx vango.Ctx) ([]Project, error) {
    return db.Projects.All()
}

func POST(ctx vango.Ctx, input CreateProjectInput) (*Project, error) {
    if err := validate(input); err != nil {
        return nil, vango.BadRequest(err)
    }
    return db.Projects.Create(input)
}
```

Generated endpoints:
- `GET /api/projects` → Returns JSON array
- `POST /api/projects` → Creates project, returns JSON

### 10.3 External API Calls

```go
func WeatherWidget(city string) vango.Component {
    return vango.Func(func() *vango.VNode {
        rctx := vango.UseCtx()
        weather := vango.NewSignal[*Weather](nil)

        vango.Effect(func() vango.Cleanup {
            cctx, cancel := context.WithCancel(rctx.StdContext())
            city := city

            go func(city string) {
                // HTTP call from server (not browser!)
                req, _ := http.NewRequestWithContext(cctx, "GET", "https://api.weather.com/v1/"+city, nil)
                resp, err := http.DefaultClient.Do(req)
                if err != nil && cctx.Err() != nil {
                    return // cancelled; ignore
                }
                if err != nil {
                    rctx.Dispatch(func() {
                        if cctx.Err() != nil {
                            return
                        }
                        weather.Set(nil)
                    })
                    return
                }
                defer resp.Body.Close()

                var w Weather
                if err := json.NewDecoder(resp.Body).Decode(&w); err != nil {
                    rctx.Dispatch(func() {
                        if cctx.Err() != nil {
                            return
                        }
                        weather.Set(nil)
                    })
                    return
                }

                rctx.Dispatch(func() {
                    if cctx.Err() != nil {
                        return
                    }
                    weather.Set(&w)
                })
            }(city)

            return cancel
        })

        if weather.Get() == nil {
            return Loading()
        }

        return Div(Class("weather"),
            Text(weather.Get().Description),
            Text(fmt.Sprintf("%.1f°C", weather.Get().Temp)),
        )
    })
}
```

Benefits:
- API keys stay on server (secure)
- No CORS issues
- Server can cache responses

---

## 11. Forms & Validation

### 11.1 Basic Forms

```go
func LoginForm(ctx vango.Ctx) vango.Component {
    return vango.Func(func() *vango.VNode {
        email := vango.NewSignal("")
        password := vango.NewSignal("")
        error := vango.NewSignal("")

        submit := func() {
            user, err := auth.Login(email.Get(), password.Get())
            if err != nil {
                error.Set(err.Error())
                return
            }
            ctx.SetUser(user)
            ctx.Navigate("/dashboard")
        }

        return Form(OnSubmit(submit),
            If(error.Get() != "",
                Div(Class("error"), Text(error.Get())),
            ),

            Label(Text("Email")),
            Input(
                Type("email"),
                Value(email.Get()),
                OnInput(email.Set),
                Required(),
            ),

            Label(Text("Password")),
            Input(
                Type("password"),
                Value(password.Get()),
                OnInput(password.Set),
                Required(),
            ),

            Button(Type("submit"), Text("Login")),
        )
    })
}
```

### 11.2 Form Library

```go
func ContactForm() vango.Component {
    return vango.Func(func() *vango.VNode {
        form := vango.UseForm(ContactInput{})

        submit := func() {
            if !form.Validate() {
                return
            }
            sendEmail(form.Values())
            form.Reset()
        }

        return Form(OnSubmit(submit),
            form.Field("Name",
                Input(Type("text")),
                vango.Required("Name is required"),
                vango.MinLength(2, "Name too short"),
            ),

            form.Field("Email",
                Input(Type("email")),
                vango.Required("Email is required"),
                vango.Email("Invalid email"),
            ),

            form.Field("Message",
                Textarea(),
                vango.Required("Message is required"),
                vango.MaxLength(1000, "Message too long"),
            ),

            Button(
                Type("submit"),
                Disabled(form.IsSubmitting()),
                Text("Send"),
            ),
        )
    })
}
```

### 11.3 Progressive Enhancement

Forms work without JavaScript:

```go
Form(
    Method("POST"),
    Action("/api/contact"),  // Fallback for no-JS
    OnSubmit(handleSubmit),  // Enhanced with WS when available
    // ...
)
```

### 11.4 Toast Notifications

Since Vango uses persistent WebSocket connections, traditional HTTP flash cookies don't work. Instead, use the toast package:

```go
import "github.com/vango-go/vango/pkg/toast"

func DeleteProject(ctx vango.Ctx, id int) error {
    if err := db.Projects.Delete(id); err != nil {
        toast.Error(ctx, "Failed to delete project")
        return err
    }
    
    toast.Success(ctx, "Project deleted")
    ctx.Navigate("/projects")
    return nil
}
```

**Client-Side Handler** (user provides):
```javascript
// Listen for toast events and render with your preferred library
window.addEventListener("vango:toast", (e) => {
    Toastify({ text: e.detail.message, className: e.detail.level }).showToast();
});
```

### 11.5 File Uploads

Large file uploads over WebSocket block the event loop. Use the hybrid HTTP+WS approach:

```go
import "github.com/vango-go/vango/pkg/upload"

// 1. Mount upload handler (main.go)
r.Post("/upload", upload.Handler(uploadStore))

// 2. Handle in component (after client POSTs and receives temp_id via WS form)
func CreatePost(ctx vango.Ctx, formData vango.FormData) error {
    tempID := formData.Get("attachment_temp_id")
    
    if tempID != "" {
        file, err := upload.Claim(uploadStore, tempID)
        if err != nil {
            return err
        }
        // Use file.Path or file.URL
    }
    
    toast.Success(ctx, "Post created!")
    return nil
}
```

---

## 12. JavaScript Islands

### 12.1 When to Use

Use JS islands for:
- Third-party libraries (charts, rich text editors, maps)
- Browser APIs not exposed to WASM
- Existing JS widgets during migration

### 12.2 Basic Usage

```go
func AnalyticsDashboard(data []DataPoint) *vango.VNode {
    return Div(Class("dashboard"),
        H1(Text("Analytics")),

        // JavaScript island for chart library
        JSIsland("revenue-chart",
            JSModule("/js/charts.js"),
            JSProps{
                "data":   data,
                "type":   "line",
                "height": 400,
            },
        ),
    )
}
```

### 12.3 JavaScript Side

```javascript
// public/js/charts.js
import { Chart } from 'chart.js';

export function mount(container, props) {
    const chart = new Chart(container, {
        type: props.type,
        data: formatData(props.data),
        options: { maintainAspectRatio: false }
    });

    // Return cleanup function
    return () => chart.destroy();
}

// Called when props change (optional)
export function update(container, props, chart) {
    chart.data = formatData(props.data);
    chart.update();
}
```

### 12.4 Communication Bridge

```go
// Send data to an island (JS or WASM)
vango.SendToIsland("revenue-chart", map[string]any{
    "action": "highlight",
    "series": "revenue",
})

// Receive events from an island (JS or WASM)
vango.OnIslandMessage("revenue-chart", func(msg map[string]any) {
    if msg["event"] == "point-click" {
        showDetails(msg["dataIndex"].(int))
    }
})
```

```javascript
// In charts.js
import { sendToVango, onVangoMessage } from '@vango/bridge';

chart.on('click', (e) => {
    sendToVango('revenue-chart', {
        event: 'point-click',
        dataIndex: e.dataIndex
    });
});

onVangoMessage('revenue-chart', (msg) => {
    if (msg.action === 'highlight') {
        chart.highlightSeries(msg.series);
    }
});
```

### 12.5 SSR Behavior

Islands render as placeholders during SSR:

```html
<div id="revenue-chart"
     data-island="true"
     data-island-module="/js/charts.js"
     data-island-props='{"type":"line","height":400}'>
    <!-- Optional loading skeleton -->
    <div class="chart-skeleton"></div>
</div>
```

The thin client hydrates islands after connecting:

```javascript
// In thin client
document.querySelectorAll('[data-island]').forEach(async (el) => {
    const mod = await import(el.dataset.islandModule);
    const props = JSON.parse(el.dataset.islandProps);
    el._cleanup = mod.mount(el, props);
});
```

**Patching rule (island boundary)**

See §6.5.7 for the canonical island boundary invariant. JavaScript islands follow the same opaque-subtree rule: the server may update the container or replace it, but MUST NOT patch inside the island-managed subtree.

---

## 13. Styling

### 13.1 Global CSS

```go
// In layout
Head(
    Link(Rel("stylesheet"), Href("/styles.css")),
)
```

### 13.2 Tailwind CSS

Vango integrates with Tailwind automatically:

```go
// Just use Tailwind classes
Div(Class("flex items-center justify-between p-4 bg-white shadow"),
    H1(Class("text-2xl font-bold text-gray-900"), Text("Title")),
    Button(Class("px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"),
        Text("Action"),
    ),
)
```

```bash
$ vango dev
→ Detected tailwind.config.js
→ Running Tailwind CSS in watch mode...
```

### 13.3 CSS Variables for Theming

```go
func ThemeProvider(theme Theme, children ...any) *vango.VNode {
    return Div(
        Style(fmt.Sprintf(`
            --color-primary: %s;
            --color-secondary: %s;
            --color-background: %s;
        `, theme.Primary, theme.Secondary, theme.Background)),
        children...,
    )
}
```

### 13.4 Dynamic Styles

```go
func ProgressBar(percent int) *vango.VNode {
    return Div(Class("progress-bar"),
        Div(
            Class("progress-fill"),
            Style(fmt.Sprintf("width: %d%%", percent)),
        ),
    )
}
```

### 13.5 VangoUI (CLI-First Component Library)

VangoUI is a server-first component library designed for Vango’s architecture. It is distributed via the CLI (copy source into your project so you own it):

```bash
vango add init
vango add button card dialog
```

VangoUI components are Tailwind-based, use CSS variables for theming, and rely on client hooks only for "physics" (drag/drop, focus traps, positioning).


---

## 14. Performance & Scaling

### 14.1 Server Resource Usage

**Memory per session:**
| App Complexity | Typical Memory |
|----------------|----------------|
| Simple (blog, marketing) | 10-50 KB |
| Medium (dashboard, CRUD) | 50-200 KB |
| Complex (project management) | 200 KB - 1 MB |

**Scaling calculation:**
```
1,000 concurrent users × 200 KB = 200 MB
10,000 concurrent users × 200 KB = 2 GB
100,000 concurrent users × 200 KB = 20 GB
```

This is a useful order-of-magnitude estimate. Real capacity depends on component complexity, session churn, patch rates, and persistence settings.

### 14.2 Reducing Memory Usage

**Stateless pages:**
```go
// For read-only pages, don't maintain session state
func BlogPost(slug string) *vango.VNode {
    post := db.Posts.FindBySlug(slug)
    return Article(
        H1(Text(post.Title)),
        Div(DangerouslySetInnerHTML(post.Content)),
    )
}
```

**Automatic cleanup:**
```go
// Sessions are evicted after inactivity
vango.Config{
    SessionTimeout: 5 * time.Minute,
    SessionMaxAge:  1 * time.Hour,
}
```

**State externalization:**
```go
// Store large state in Redis, not memory
tasks := vango.NewSignal([]Task{}).Store(redis.Store)
```

### 14.3 WebSocket Scaling

Go handles WebSocket connections efficiently:

```go
// Using efficient connection pooling
vango.Config{
    MaxConnsPerSession: 1,  // One WS per tab
    ReadBufferSize:     1024,
    WriteBufferSize:    1024,
    EnableCompression:  true,
}
```

For horizontal scaling:
- Use sticky sessions (route by session ID)
- Or use Redis pub/sub for cross-server messaging

### 14.4 Latency Optimization

**Server location matters:**
| User Location | Server Location | Round-trip |
|---------------|-----------------|------------|
| NYC | NYC | 5-10ms |
| NYC | SF | 40-60ms |
| NYC | London | 70-90ms |
| NYC | Tokyo | 150-200ms |

**Recommendations:**
- Deploy in regions close to users
- Use edge locations for static assets
- Consider optimistic updates for high-latency scenarios

### 14.5 Bundle Size

| Mode | Client Size (gzip) |
|------|-------------------|
| Server-Driven | ~12 KB |
| Server-Driven + Optimistic | ~15 KB |
| Hybrid (partial WASM) | 12 KB + WASM components |
| Full WASM | ~250-400 KB |

Hybrid builds include an island bundle analyzer (see WASM islands) that reports per-island and per-route sizes (raw + gzip/brotli), dependency attribution, and configurable size budgets (warn by default; hard fail optional).

---

### 14.6 Observability

Vango adopts a **middleware-first** observability model:

- **No `ctx.Trace()` API**: tracing is infrastructure, not application logic.
- **OpenTelemetry**: middleware starts spans for each event (click/input/nav), records patch counts and errors.
- **Context propagation**: `ctx.StdContext()` carries trace context into DB drivers and HTTP clients.
- **Prometheus metrics (optional)**: session counts, detached sessions, event rates, patch rates, reconnects.

Metrics endpoints should be protected (auth/IP allowlist) in production.

---

## 15. Security

Vango provides **security by design** with secure defaults that protect against common vulnerabilities.

### 15.1 Secure Defaults

| Setting | Default | Notes |
|---------|---------|------|
| `CheckOrigin` | Same-origin only | Cross-origin WebSocket rejected. |
| CSRF | Warning if disabled | Planned to become required in a future major version. |
| `on*` attributes | Stripped unless handler | Prevents XSS injection. |
| Protocol limits | Allocation and nesting limits | Prevents common DoS patterns. |

### 15.2 XSS Prevention

#### Text Escaping

All text content is escaped by default:

```go
// Safe - content is escaped
Div(Text(userInput))  // <script> becomes &lt;script&gt;

// Explicit opt-in for raw HTML
Div(DangerouslySetInnerHTML(trustedHTML))
```

> **Security Warning**: `DangerouslySetInnerHTML` MUST only accept trusted or sanitized HTML; it bypasses all framework-level XSS escaping.

`DangerouslySetInnerHTML(...)` is the canonical unsafe HTML escape hatch. `Raw(...)` is a legacy alias.

#### Attribute Sanitization

Event handler attributes (`onclick`, `onmouseover`, etc.) are automatically filtered:

```go
// This is BLOCKED - attribute stripped during render
Attr("onclick", "alert(1)")

// This is SAFE - uses internal event handler
OnClick(myHandler)
```

> **Note**: The filter is case-insensitive. `ONCLICK`, `onClick`, and `onclick` are all blocked.

### 15.3 CSRF Protection

Enable CSRF protection in production:

```go
vango.Config{
    CSRFSecret: []byte("your-32-byte-secret-key-here!!"),
}
```

CSRF uses the **Double Submit Cookie** pattern:
1. Server sets `__vango_csrf` cookie via `server.SetCSRFCookie()`
2. Server embeds token in HTML as `window.__VANGO_CSRF__`
3. Client sends token in WebSocket handshake
4. Server validates handshake token matches cookie

```go
// In your page handler
func ServePage(w http.ResponseWriter, r *http.Request) {
    token := server.GenerateCSRFToken()
    server.SetCSRFCookie(w, token)
    // Embed token in page for client
}
```

> **Warning**: If `CSRFSecret` is nil, a warning is logged on startup. In production deployments, CSRF MUST be enabled.

### 15.4 WebSocket Origin Validation

By default, Vango rejects cross-origin WebSocket connections (prevents CSWSH):

```go
// Default behavior - same-origin only
config := server.DefaultServerConfig()
// config.CheckOrigin = SameOriginCheck (secure default)

// Explicit cross-origin (dev only!)
config.CheckOrigin = func(r *http.Request) bool { return true }
```

### 15.5 Session Security

```go
vango.Config{
    SessionCookie: http.Cookie{
        Name:     "vango_session",
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteLaxMode, // Lax is recommended; Strict can break common OAuth flows.
    },
}
```

### 15.6 Protocol Defense

The binary protocol includes allocation + nesting limits to prevent DoS and stack overflow attacks:

| Limit | Value | Purpose |
|-------|-------|---------|
| Max string/bytes | 4MB | Prevent OOM. |
| Max collection | 100K items | Prevent CPU exhaustion. |
| Max VNode depth | 256 | Prevent stack overflow. |
| Max patch depth | 128 | Prevent stack overflow. |
| Hard cap | 16MB | Absolute ceiling. |

Production hardening adds fuzz testing for protocol decoders to ensure invalid inputs return errors (never panics).

### 15.7 Event Handler Safety

Handlers are server-side function references, not code strings:

```go
// This creates a server-side handler mapping
Button(OnClick(func() {
    doSensitiveAction()  // Runs on server
}))
```

The client only sends `{hid: "h42", type: 0x01}`. It cannot:
- Execute arbitrary functions
- Access handlers from other sessions
- Inject JavaScript

### 15.8 Input Validation

```go
func CreateProject(ctx vango.Ctx, input CreateInput) (*Project, error) {
    // Validate on server (always!)
    if err := validate.Struct(input); err != nil {
        return nil, vango.BadRequest(err)
    }

    // Sanitize
    input.Name = sanitize.String(input.Name)

    return db.Projects.Create(input)
}
```

### 15.9 Authentication & Middleware

Vango uses a **dual-layer architecture** that separates HTTP middleware from Vango's event-loop middleware:

**Layer 1: HTTP Stack** (runs once per session):
- Standard `func(http.Handler) http.Handler` middleware
- Authentication, CORS, logging, panic recovery
- Compatible with Chi, Gorilla, rs/cors, etc.

**Layer 2: Vango Event Stack** (runs on every interaction):
- Lightweight `func(ctx vango.Ctx, next func() error) error` middleware  
- Authorization guards (RBAC), event validation
- No HTTP overhead on the hot path

#### The Context Bridge

The WebSocket upgrade presents a challenge: HTTP request context dies after the upgrade. The Context Bridge solves this:

```go
app := vango.New(vango.Config{
    // This runs ONCE during WebSocket upgrade
    OnSessionStart: func(httpCtx context.Context, s *vango.Session) {
        // Copy user from HTTP context to Vango session
        if user := myauth.UserFromContext(httpCtx); user != nil {
            auth.Set(s, user)
        }
    },
})
```

#### Type-Safe Auth Package

```go
import "github.com/vango-go/vango/pkg/auth"

// Require auth (returns error if not logged in)
func Dashboard(ctx vango.Ctx) (vango.Component, error) {
    user, err := auth.Require[*User](ctx)
    if err != nil {
        return nil, err  // ErrorBoundary handles redirect
    }
    return renderDashboard(user), nil
}

// Optional auth (guest allowed)
func HomePage(ctx vango.Ctx) vango.Component {
    user, ok := auth.Get[*User](ctx)
    if ok {
        return LoggedInHome(user)
    }
    return GuestHome()
}
```

#### Integration with Chi Router

Vango exposes itself as a standard `http.Handler` for ecosystem compatibility:

```go
func main() {
    app := vango.New(vango.Config{
        OnSessionStart: hydrateSession,
    })

    r := chi.NewRouter()
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    r.Use(AuthMiddleware)  // Your auth middleware
    
    r.Get("/api/health", healthHandler)
    r.Handle("/*", app.Handler())  // Vango handles the rest
    
    http.ListenAndServe(":3000", r)
}
```

#### Route-Level Auth Guards

```go
// app/routes/admin/_layout.go
func Middleware() []router.Middleware {
    return []router.Middleware{
        auth.RequireRole(func(u *User) bool {
            return u.IsAdmin
        }),
    }
}
```

---

## 16. Testing

### 16.1 Unit Testing Components

```go
func TestCounter(t *testing.T) {
    // Create test context
    ctx := vango.TestContext()

    // Mount component
    c := Counter(5)
    tree := ctx.Mount(c)

    // Assert initial render
    assert.Contains(t, tree.Text(), "Count: 5")

    // Simulate click
    ctx.Click("[data-testid=increment]")

    // Assert update
    assert.Contains(t, tree.Text(), "Count: 6")
}
```

### 16.2 Testing with Signals

```go
func TestSignalUpdates(t *testing.T) {
    ctx := vango.TestContext()

    count := ctx.NewSignal(0)
    tree := ctx.Render(func() *vango.VNode {
        return Div(Textf("Count: %d", count.Get()))
    })

    assert.Equal(t, "Count: 0", tree.Text())

    count.Set(10)
    ctx.Flush()  // Process signal updates

    assert.Equal(t, "Count: 10", tree.Text())
}
```

### 16.3 Integration Testing

```go
func TestLoginFlow(t *testing.T) {
    app := vango.TestApp()

    // Navigate to login
    page := app.Navigate("/login")

    // Fill form
    page.Fill("[name=email]", "test@example.com")
    page.Fill("[name=password]", "password123")
    page.Click("[type=submit]")

    // Assert redirect
    assert.Equal(t, "/dashboard", page.URL())
    assert.Contains(t, page.Text(), "Welcome back")
}
```

### 16.4 E2E Testing (Playwright)

```typescript
// tests/login.spec.ts
test('user can log in', async ({ page }) => {
    await page.goto('/login');

    await page.fill('[name=email]', 'test@example.com');
    await page.fill('[name=password]', 'password123');
    await page.click('[type=submit]');

    await expect(page).toHaveURL('/dashboard');
    await expect(page.locator('h1')).toContainText('Welcome');
});
```

### 16.5 Session Lifecycle Testing

Vango provides test utilities for disconnect/reconnect and restart scenarios:

```go
func TestCartSurvivesRestart(t *testing.T) {
    app := vango.TestApp()
    sess := vango.NewTestSession(app)

    sess.Mount(CartPage)
    sess.Click("#add-item")

    require.NoError(t, sess.SimulateRefresh())
    require.NoError(t, sess.SimulateServerRestart())
}
```

---

## 17. Developer Experience

### 17.1 Project Structure

```
my-app/
├── app/
│   ├── routes/                        # File-based routing
│   │   ├── routes_gen.go              # GENERATED — route registration glue
│   │   ├── _layout.go                 # Root layout wrapper
│   │   ├── _middleware.go             # Optional directory middleware
│   │   ├── index.go                   # /
│   │   ├── about.go                   # /about
│   │   └── api/
│   │       └── health.go              # GET /api/health
│   ├── components/                    # Shared UI components
│   │   ├── shared/                    # Cross-domain components
│   │   └── ui/                        # VangoUI (populated by `vango add`)
│   ├── store/                         # Shared state
│   └── middleware/                    # App middleware (auth, ratelimit, etc.)
├── db/                                # Data access layer
├── go.mod
├── go.sum
├── main.go
├── public/                            # Static assets (served at /)
│   ├── favicon.ico
│   └── styles.css
└── vango.json            # Configuration
```

**Static serving contract:** `public/` is served at site root (`/`), so `public/styles.css` is available at `/styles.css`.


### 17.2 CLI Commands

```bash
# Create new project
vango create my-app

# Development server (hot reload)
vango dev

# Generate/repair routes glue (auto-run by `vango dev` on route file changes)
vango gen routes

# Generate pages and APIs
vango gen route projects/[id]
vango gen api users

# VangoUI (CLI-first, copy-into-your-project distribution)
vango add init
vango add button card dialog

# Production build
vango build

# Run tests
vango test
```

`vango.json` drives generation paths and static serving:

```json
{
  "port": 3000,
  "paths": { "routes": "app/routes", "ui": "app/components/ui" },
  "static": { "dir": "public", "prefix": "/" }
}
```


### 17.3 Hot Reload

```bash
$ vango dev
→ Server starting on http://localhost:3000
→ Watching for changes...

[12:34:56] Changed: app/components/button.go
[12:34:56] Rebuilding... (42ms)
[12:34:56] Reloaded 2 connected clients
```

Changes are instant:
1. File change detected
2. Go recompilation (~50ms for incremental)
3. Connected browsers receive refresh signal
4. Only affected components re-render


### 17.4 Error Messages

**Compile-time errors:**
```
app/routes/projects/[id].go:23:15: cannot use string as int in argument
    project := db.Projects.FindByID(params.ID)
                                    ^^^^^^^^^
    Hint: params.ID is string, but FindByID expects int
          Use: strconv.Atoi(params.ID)
```

**Runtime errors:**
```
ERROR in /projects/123
  app/routes/projects/[id].go:45

  Signal read outside component context

    count := vango.NewSignal(0)
    value := count.Get()  // ← Error: no active component

  Hint: Signal reads must happen inside a component's render function
        or an Effect. Move this code inside vango.Func(func() {...})
```

**Hydration mismatches (dev mode):**
```
HYDRATION MISMATCH at /dashboard

  Server rendered:
    <div class="status">Offline</div>

  Client expected:
    <div class="status">Online</div>

  Difference: text content

  Location: app/components/status.go:12

  Hint: This component reads browser-only state (navigator.onLine)
        during render. Use an Effect instead:

        vango.Effect(func() vango.Cleanup {
            status.Set(getOnlineStatus())
            return nil
        })
```

### 17.5 VS Code Extension

- Syntax highlighting for Vango components
- Go to definition for components
- Autocomplete for element attributes
- Error highlighting
- Hot reload integration

---

## 18. Migration Guide

### 18.1 From React

**React:**
```jsx
function Counter({ initial }) {
    const [count, setCount] = useState(initial);

    return (
        <div className="counter">
            <h1>Count: {count}</h1>
            <button onClick={() => setCount(c => c + 1)}>+</button>
        </div>
    );
}
```

**Vango:**
```go
func Counter(initial int) vango.Component {
    return vango.Func(func() *vango.VNode {
        count := vango.NewSignal(initial)

        return Div(Class("counter"),
            H1(Textf("Count: %d", count.Get())),
            Button(OnClick(count.Inc), Text("+")),
        )
    })
}
```

**Key differences:**
| React | Vango |
|-------|-------|
| `useState` | `vango.NewSignal` |
| `useEffect` | `vango.Effect` |
| `useMemo` | `vango.NewMemo` |
| JSX | Function calls |
| Runs in browser | Runs on server |

### 18.2 From Vue

**Vue:**
```vue
<template>
    <div class="counter">
        <h1>Count: {{ count }}</h1>
        <button @click="count++">+</button>
    </div>
</template>

<script setup>
import { ref } from 'vue'
const count = ref(0)
</script>
```

**Vango:**
```go
func Counter(initial int) vango.Component {
    return vango.Func(func() *vango.VNode {
        count := vango.NewSignal(initial)

        return Div(Class("counter"),
            H1(Textf("Count: %d", count.Get())),
            Button(OnClick(count.Inc), Text("+")),
        )
    })
}
```

### 18.3 Gradual Migration

You can migrate incrementally:

1. **Add Vango to existing Go backend**
2. **Create new pages in Vango**
3. **Use JS islands for existing React components**
4. **Gradually rewrite components in Go**

```go
// During migration: wrap existing React component
func LegacyDashboard() *vango.VNode {
    return JSIsland("dashboard",
        JSModule("/js/legacy/dashboard.js"),  // Existing React code
        JSProps{"user": currentUser},
    )
}
```

---

## 19. Examples

### 19.1 Todo App

```go
// app/routes/todos.go
package routes

func Page(ctx vango.Ctx) vango.Component {
    return vango.Func(func() *vango.VNode {
        rctx := vango.UseCtx()
        todos := vango.NewSignal([]Todo{})
        newTodo := vango.NewSignal("")

        // Load todos from database
        vango.Effect(func() vango.Cleanup {
            userID := ctx.UserID()
            cctx, cancel := context.WithCancel(rctx.StdContext())

            go func(userID string) {
                items, err := db.Todos.ForUser(cctx, userID)
                if err != nil && cctx.Err() != nil {
                    return // cancelled; ignore
                }
                rctx.Dispatch(func() {
                    if cctx.Err() != nil {
                        return // cancelled; ignore
                    }
                    if err != nil {
                        todos.Set([]Todo{})
                        return
                    }
                    todos.Set(items)
                })
            }(userID)

            return cancel
        })

        addTodo := func() {
            if newTodo.Get() == "" {
                return
            }
            todo := db.Todos.Create(ctx.UserID(), newTodo.Get())
            todos.Update(func(t []Todo) []Todo {
                next := make([]Todo, 0, len(t)+1)
                next = append(next, t...)
                next = append(next, todo)
                return next
            })
            newTodo.Set("")
        }

        toggleTodo := func(id int) func() {
            return func() {
                db.Todos.Toggle(id)
                todos.Update(func(t []Todo) []Todo {
                    next := make([]Todo, len(t))
                    copy(next, t)
                    for i := range next {
                        if next[i].ID == id {
                            next[i].Done = !next[i].Done
                        }
                    }
                    return next
                })
            }
        }

        return Div(Class("todo-app"),
            H1(Text("My Todos")),

            Form(OnSubmit(addTodo), Class("add-form"),
                Input(
                    Type("text"),
                    Value(newTodo.Get()),
                    OnInput(newTodo.Set),
                    Placeholder("What needs to be done?"),
                ),
                Button(Type("submit"), Text("Add")),
            ),

            Ul(Class("todo-list"),
                Range(todos.Get(), func(todo Todo, i int) *vango.VNode {
                    return Li(
                        Key(todo.ID),
                        Class("todo-item"),
                        ClassIf(todo.Done, "completed"),

                        Input(
                            Type("checkbox"),
                            Checked(todo.Done),
                            OnChange(toggleTodo(todo.ID)),
                        ),
                        Span(Text(todo.Text)),
                    )
                }),
            ),
        )
    })
}
```

### 19.2 Real-time Chat

```go
func ChatRoom(roomID string) vango.Component {
    return vango.Func(func() *vango.VNode {
        messages := vango.NewGlobalSignal([]Message{})  // Shared across all users
        input := vango.NewSignal("")

        sendMessage := func() {
            if input.Get() == "" {
                return
            }
            msg := Message{
                User:    currentUser(),
                Text:    input.Get(),
                Time:    time.Now(),
            }
            messages.Update(func(m []Message) []Message {
                next := make([]Message, 0, len(m)+1)
                next = append(next, m...)
                next = append(next, msg)
                return next
            })
            input.Set("")
        }

        return Div(Class("chat-room"),
            Div(Class("messages"),
                Range(messages.Get(), func(msg Message, i int) *vango.VNode {
                    return Div(Class("message"),
                        Strong(Text(msg.User.Name)),
                        Span(Text(msg.Text)),
                        Time_(Text(msg.Time.Format("3:04 PM"))),
                    )
                }),
            ),

            Form(OnSubmit(sendMessage), Class("input-area"),
                Input(
                    Type("text"),
                    Value(input.Get()),
                    OnInput(input.Set),
                    Placeholder("Type a message..."),
                ),
                Button(Type("submit"), Text("Send")),
            ),
        )
    })
}
```

### 19.3 Dashboard with Charts

```go
func Dashboard() vango.Component {
    return vango.Func(func() *vango.VNode {
        rctx := vango.UseCtx()
        stats := vango.NewSignal[*Stats](nil)
        period := vango.NewSignal("week")

        vango.Effect(func() vango.Cleanup {
            p := period.Get() // Tracked
            cctx, cancel := context.WithCancel(rctx.StdContext())

            go func(p string) {
                s, err := analytics.GetStats(cctx, p)
                if err != nil && cctx.Err() != nil {
                    return // cancelled; ignore
                }
                rctx.Dispatch(func() {
                    if cctx.Err() != nil {
                        return // cancelled; ignore
                    }
                    if period.Peek() != p {
                        return // stale; ignore
                    }
                    if err != nil {
                        stats.Set(nil)
                        return
                    }
                    stats.Set(s)
                })
            }(p)

            return cancel
        })

        if stats.Get() == nil {
            return Loading()
        }

        return Div(Class("dashboard"),
            Header(
                H1(Text("Dashboard")),
                Select(
                    Value(period.Get()),
                    OnChange(period.Set),
                    Option(Value("day"), Text("Today")),
                    Option(Value("week"), Text("This Week")),
                    Option(Value("month"), Text("This Month")),
                ),
            ),

            Div(Class("stats-grid"),
                StatCard("Revenue", stats.Get().Revenue, "+12%"),
                StatCard("Users", stats.Get().Users, "+5%"),
                StatCard("Orders", stats.Get().Orders, "+8%"),
            ),

            // JS island for complex chart
            JSIsland("revenue-chart",
                JSModule("/js/charts.js"),
                JSProps{
                    "data": stats.Get().RevenueHistory,
                    "type": "area",
                },
            ),
        )
    })
}
```

---

## 20. FAQ

### General

**Q: Is this like Phoenix LiveView?**
A: Yes! Vango is inspired by LiveView but for Go. Server-driven UI with binary patches over WebSocket.

**Q: Do I need to know JavaScript?**
A: For most apps, no. You only need JS for islands (third-party libraries) or very latency-sensitive features.

**Q: What about SEO?**
A: SSR is built-in. Search engines see fully-rendered HTML. No JavaScript required for content.

**Q: Can I use this for mobile apps?**
A: Vango is for web apps. For mobile, consider using the API routes with a native app, or a WebView wrapper.

### Performance

**Q: What's the latency for interactions?**
A: Typically 50-100ms (network round-trip + processing). Use optimistic updates for instant feel.

**Q: How many concurrent users can one server handle?**
A: Depends on complexity, but typically 10,000-100,000+ with proper session management.

**Q: Is the 12KB client cached?**
A: Yes. After first load, it's served from browser cache. Only the WebSocket connection is new.

### Architecture

**Q: What happens if WebSocket disconnects?**
A: The client auto-reconnects. Server sends full state on reconnect. No manual sync needed.

**Q: Can I use this with an existing Go backend?**
A: Yes! Vango integrates with `net/http`. Mount it alongside your existing API routes.

**Q: How does authentication work?**
A: Use your existing auth. Vango reads session cookies. User is available via `ctx.User()`.

### Development

**Q: Is hot reload fast?**
A: Yes. Go incremental compilation is ~50ms. Changes appear instantly in browser.

**Q: Can I debug server-side code?**
A: Yes. Use Delve or your IDE's debugger. Set breakpoints in event handlers.

**Q: How do I deploy?**
A: Single binary. Deploy like any Go server. No Node.js, no build step in production.

---

## Appendix A: Side-Effect Patterns (Informative)

This appendix is **informative**. Normative API definitions and requirements for Action, Effect helpers, Effect enforcement, and storm budgets are in §3.10.

### A.1 Desugared Patterns (Informative)

These examples show what the helpers *conceptually* expand to. They are for understanding and advanced customization; real implementations may differ.

**Note**: Signal writes in these desugared examples occur inside `ctx.Dispatch(...)` transactions, not in the Effect body, and therefore are not “effect-time writes” (§3.10.4).

#### A.1.1 Interval (Conceptual)

```go
vango.Effect(func() vango.Cleanup {
    ctx := vango.UseCtx()
    ticker := time.NewTicker(d)
    done := make(chan struct{})

    go func() {
        for {
            select {
            case <-ticker.C:
                ctx.Dispatch(func() {
                    vango.TxNamed("Interval:...", fn)
                })
            case <-done:
                return
            }
        }
    }()

    return func() {
        close(done)
        ticker.Stop()
    }
})
```

#### A.1.2 GoLatest (Conceptual)

```go
// Per-call-site state (managed by the runtime)
var lastKey K
var lastSeq uint64

vango.Effect(func() vango.Cleanup {
    ctx := vango.UseCtx()

    // Key coalescing: if same key, do nothing.
    if key == lastKey {
        return nil
    }

    lastKey = key
    lastSeq++
    mySeq := lastSeq

    cctx, cancel := context.WithCancel(ctx.StdContext())

    go func(mySeq uint64, k K) {
        result, err := work(cctx, k)
        if cctx.Err() != nil {
            return
        }

        ctx.Dispatch(func() {
            if cctx.Err() != nil {
                return
            }
            if lastSeq != mySeq {
                return // stale
            }
            vango.TxNamed("GoLatest:...", func() {
                apply(result, err)
            })
        })
    }(mySeq, key)

    return cancel
})
```

### A.2 Migration Recipes (Informative)

#### A.2.1 From Effect + Ticker to Interval

Before:
```go
vango.Effect(func() vango.Cleanup {
    ctx := vango.UseCtx()
    ticker := time.NewTicker(time.Second)
    done := make(chan struct{})
    go func() {
        for {
            select {
            case <-ticker.C:
                ctx.Dispatch(elapsed.Inc)
            case <-done:
                return
            }
        }
    }()
    return func() {
        close(done)
        ticker.Stop()
    }
})
```

After:
```go
vango.Effect(func() vango.Cleanup {
    return vango.Interval(time.Second, elapsed.Inc)
})
```

#### A.2.2 From Manual Async + Stale Checks to GoLatest

Before:
```go
vango.Effect(func() vango.Cleanup {
    ctx := vango.UseCtx()
    cctx, cancel := context.WithCancel(ctx.StdContext())
    id := userID.Get()

    go func(id int) {
        user, err := api.FetchUser(cctx, id)
        if cctx.Err() != nil {
            return
        }
        ctx.Dispatch(func() {
            if cctx.Err() != nil {
                return
            }
            if userID.Peek() != id {
                return
            }
            if err != nil {
                fetchErr.Set(err)
            } else {
                currentUser.Set(user)
            }
        })
    }(id)

    return cancel
})
```

After:
```go
vango.Effect(func() vango.Cleanup {
    return vango.GoLatest(userID.Get(),
        func(ctx context.Context, id int) (*User, error) {
            return api.FetchUser(ctx, id)
        },
        func(user *User, err error) {
            if err != nil {
                fetchErr.Set(err)
            } else {
                currentUser.Set(user)
            }
        },
    )
})
```

### A.3 Agent Guidance (Informative)

If you are inside an Effect and think you need a goroutine, prefer:
- Time-based repetition → `Interval`
- Event stream subscribe/unsubscribe → `Subscribe`
- Keyed async work → `GoLatest`

Before submitting Effect code, verify:
- Cleanup is returned from the Effect (not dropped).
- No manual `time.Ticker` (use `Interval`).
- No manual stale-result checks (use `GoLatest`).


## 22. Appendix: Protocol Specification

### 22.1 WebSocket Handshake

The handshake is sent as a JSON **text frame**. After the server replies with `HANDSHAKE_ACK`, all subsequent messages are **binary frames** following the formats in §22.2 and §22.3.

```
Client → Server:
{
    "type": "HANDSHAKE",
    "version": "1.0",
    "csrf": "<token>",
    "session": "<session-id>",  // From cookie, if reconnecting
    "viewport": {"width": 1920, "height": 1080}
}

Server → Client:
{
    "type": "HANDSHAKE_ACK",
    "session": "<new-or-existing-session-id>",
    "serverTime": 1699999999999
}
```

### 22.2 Binary Event Format

```
┌─────────────────────────────────────────────────────────────┐
│ Byte 0    │ Bytes 1-N      │ Remaining bytes               │
│ EventType │ HID (varint)   │ Payload (type-specific)       │
└─────────────────────────────────────────────────────────────┘

EventType values:
  0x01: CLICK
  0x02: DBLCLICK
  0x03: INPUT          Payload: [varint: length][utf8: value]
  0x04: CHANGE         Payload: [varint: length][utf8: value]
  0x05: SUBMIT         Payload: [form encoding]
  0x06: FOCUS
  0x07: BLUR
  0x08: KEYDOWN        Payload: [key: uint16][modifiers: uint8]
  0x09: KEYUP          Payload: [key: uint16][modifiers: uint8]
  0x0A: MOUSEENTER
  0x0B: MOUSELEAVE
  0x0C: SCROLL         Payload: [scrollX: int32][scrollY: int32]
  0x0D: NAVIGATE       Payload: [varint: length][utf8: path]
  0x0E: CUSTOM         Payload: [varint: type][varint: length][data]
```

#### 22.2.1 CUSTOM event subtypes

`CUSTOM` is reserved for extensibility. The `type` field selects the subtype; the `data` field contains subtype-specific bytes.

Reserved subtypes:

- `0x01: HOOK_EVENT` — client → server messages emitted by hooks
- `0x02: ISLAND_MESSAGE` — client → server messages emitted by JS/WASM islands

Recommended `ISLAND_MESSAGE` payload:
```
data = [varint: islandID-len][utf8: islandID][varint: msg-len][msg-bytes]
```

### 22.3 Binary Patch Format

```
┌─────────────────────────────────────────────────────────────┐
│ Bytes 0-N        │ Patches...                               │
│ Count (varint)   │                                          │
└─────────────────────────────────────────────────────────────┘

Each Patch:
┌─────────────────────────────────────────────────────────────┐
│ Byte 0     │ Bytes 1-N      │ Remaining bytes              │
│ PatchType  │ HID (varint)   │ Payload (type-specific)      │
└─────────────────────────────────────────────────────────────┘

PatchType values:
  0x01: SET_TEXT       Payload: [varint: length][utf8: text]
  0x02: SET_ATTR       Payload: [varint: key-len][key][varint: val-len][val]
  0x03: REMOVE_ATTR    Payload: [varint: key-len][key]
  0x04: ADD_CLASS      Payload: [varint: length][utf8: class]
  0x05: REMOVE_CLASS   Payload: [varint: length][utf8: class]
  0x06: SET_STYLE      Payload: [varint: prop-len][prop][varint: val-len][val]
  0x07: INSERT_BEFORE  Payload: [varint: ref-hid][encoded-vnode]
  0x08: INSERT_AFTER   Payload: [varint: ref-hid][encoded-vnode]
  0x09: APPEND_CHILD   Payload: [encoded-vnode]
  0x0A: REMOVE_NODE    Payload: (none)
  0x0B: REPLACE_NODE   Payload: [encoded-vnode]
  0x0C: SET_VALUE      Payload: [varint: length][utf8: value]
  0x0D: SET_CHECKED    Payload: [bool: checked]
  0x0E: SET_SELECTED   Payload: [bool: selected]
  0x0F: FOCUS          Payload: (none)
  0x10: BLUR           Payload: (none)
  0x11: SCROLL_TO      Payload: [int32: x][int32: y]
  0x12: ISLAND_MESSAGE Payload: [varint: id-len][utf8: islandID][varint: msg-len][msg-bytes] (HID = 0)
  0x13: HOOK_REVERT    Payload: (none) — triggers revert on the hook instance at HID
  0x30: URL_PUSH       Payload: [varint: kv-count][key][val]... (HID = 0, query params only)
  0x31: URL_REPLACE    Payload: [varint: kv-count][key][val]... (HID = 0, query params only)
```

### 22.4 VNode Encoding

```
┌─────────────────────────────────────────────────────────────┐
│ Byte 0    │ Remaining bytes (type-specific)                 │
│ NodeType  │                                                 │
└─────────────────────────────────────────────────────────────┘

NodeType 0x01: Element
  [varint: tag-length][tag]
  [varint: hid]  // 0 if no hid
  [varint: attr-count]
  for each attr:
    [varint: key-length][key][varint: val-length][val]
  [varint: child-count]
  for each child:
    [encoded-vnode]  // Recursive

NodeType 0x02: Text
  [varint: length][utf8: text]

NodeType 0x03: Fragment
  [varint: child-count]
  for each child:
    [encoded-vnode]
```

### 22.5 Varint Encoding

Unsigned variable-length integer (same as Protocol Buffers):

```
Value 0-127:        1 byte   [0xxxxxxx]
Value 128-16383:    2 bytes  [1xxxxxxx] [0xxxxxxx]
Value 16384+:       3+ bytes [1xxxxxxx] [1xxxxxxx] [0xxxxxxx] ...
```

This keeps small numbers (most HIDs, lengths) as single bytes.

*This document is the authoritative reference for Vango's architecture. For implementation details, see the source code and inline documentation.*
 
