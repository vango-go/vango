---
title: "Vango Developer Guide"
slug: vango-developer-guide
version: 0.1.0
status: Draft (Expandable)
last_updated: 2026-01-07
---

# Vango Developer Guide

This is the **practical, application-focused** guide for building production Vango apps.

- For the **normative framework contract** (render rules, routing contracts, protocol, thin client behavior), treat `VANGO_ARCHITECTURE_AND_GUIDE.md` as the source of truth.
- This guide focuses on **how to assemble an app**: project structure, routing/layouts, data access, forms, auth, assets, testing, and deployment.

## How this guide is organized

Each section is written to be expanded over multiple passes:
- **Scope**: what the section covers and what it intentionally does not cover.
- **Decisions**: the default recommendations and the alternatives you can pick.
- **Recipes**: copy/paste-friendly patterns with explanations and pitfalls.
- **Checklists**: production readiness, security, performance, and ops checklists.

---

## Table of Contents

1. [Audience, Scope, and Guarantees](#1-audience-scope-and-guarantees)
2. [Mental Model: How Vango Apps Work](#2-mental-model-how-vango-apps-work)
3. [Install, Requirements, and Tooling](#3-install-requirements-and-tooling)
4. [Creating a New App](#4-creating-a-new-app)
5. [Project Layout and Code Organization](#5-project-layout-and-code-organization)
6. [App Entry Point and Configuration](#6-app-entry-point-and-configuration)
7. [Routing, Layouts, and Middleware](#7-routing-layouts-and-middleware)
8. [Components and UI Composition](#8-components-and-ui-composition)
9. [State Management in Real Apps](#9-state-management-in-real-apps)
10. [Data Loading and Caching](#10-data-loading-and-caching)
11. [Mutations, Background Work, and Side Effects](#11-mutations-background-work-and-side-effects)
12. [Forms, Validation, and UX](#12-forms-validation-and-ux)
13. [Navigation, URL State, and Progressive Enhancement](#13-navigation-url-state-and-progressive-enhancement)
14. [Styling and Design Systems](#14-styling-and-design-systems)
15. [Static Assets, Bundles, and the Thin Client](#15-static-assets-bundles-and-the-thin-client)
16. [Client Hooks, JavaScript Islands, and WASM](#16-client-hooks-javascript-islands-and-wasm)
17. [Authentication, Authorization, and Sessions](#17-authentication-authorization-and-sessions)
18. [Security Checklist](#18-security-checklist)
19. [Performance and Scaling](#19-performance-and-scaling)
20. [Observability: Logging, Metrics, Tracing](#20-observability-logging-metrics-tracing)
21. [Testing Strategy](#21-testing-strategy)
22. [Deployment and Operations](#22-deployment-and-operations)
23. [Migration and Interop](#23-migration-and-interop)
24. [Recipes and Reference Apps](#24-recipes-and-reference-apps)
25. [Troubleshooting and FAQ](#25-troubleshooting-and-faq)
26. [Appendices](#26-appendices)

---

## API Quick Reference

This section provides canonical API signatures per the normative spec. Refer to `VANGO_ARCHITECTURE_AND_GUIDE.md` for full details.

> [!IMPORTANT]
> **CRITICAL ERRATA & NOTATIONS**
> 1. **Page handlers are REACTIVE**: Page handlers registered via `app.Page(...)` execute during the render cycle, not once per navigation. They are wrapped in `vango.Func` internally and MUST follow render-purity rules: no blocking I/O, no goroutines, stable hook order. Use `Resource` for data loading.
> 2. **Pervasive `Append`, `Inc`, `RemoveAt` usage**: In many examples, we use shorthand like `count.Inc()` or `items.Append(item)`. While these are supported for convenience, the underlying primitive is `signal.Set(newVal)`.
> 3. **`vango.Effect` returning `Cleanup`**: Many older examples in this guide may show `vango.Effect(func() { ... })`. Per the normative spec, `Effect` MUST return `vango.Cleanup` (or `nil`).
> 4. **`vango.UseCtx()` in goroutines**: Never call `vango.UseCtx()` or use a captured `ctx` to read/write state from a goroutine without using `ctx.Dispatch`.
> 5. **Route filename conventions**: The framework uses `index.go`, `about.go`, `[id].go`, `layout.go`, and `middleware.go`. Avoid `page.go` and nested directories for simple pages.
> 6. **Route Handler Signatures**: All handlers with parameters MUST use a `Params` struct with `param` tags, e.g., `func ShowPage(ctx vango.Ctx, p Params)`. Positional arguments are not supported.
> 7. **UUID Parameters**: Parameters annotated with `:uuid` in filenames are mapped to Go `string` types, with runtime format validation provided by the router. Do not use `uuid.UUID` in `Params` structs.
> 8. **WebSocket `?path=` parameter**: The thin client connects to `/_vango/live?path=<current-path>`. This is required for immediate interactivity after SSR (server must know initial route). If `?path=` is absent or invalid, Vango defaults to `/` or triggers a hard-reload per self-heal rules. Custom WS URL overrides must preserve this parameter.

### Core Reactive Primitives

```go
// Signals
signal := vango.NewSignal(initialValue)
signal.Get()                    // Read (creates dependency)
signal.Set(newValue)            // Write (must be on session loop)
signal.Peek()                   // Read without dependency

// Memos
memo := vango.NewMemo(func() T { return derivedValue })
memo.Get()                      // Read derived value

// Effects (return Cleanup, call helpers inside)
vango.Effect(func() vango.Cleanup {
    return vango.Interval(time.Second, fn)  // Helper returns Cleanup
})

// Transactions
vango.Tx(func() { /* multiple writes, one commit */ })
vango.TxNamed("domain:action", func() { /* named for observability */ })
```

### Data Loading & Mutations

```go
// Resource (async data loading)
resource := vango.NewResource(func() (T, error) { ... })
resource := vango.NewResourceKeyed(key, func(k K) (T, error) { ... })
// key can be *Signal[K] or func() K

// States: vango.Pending, vango.Loading, vango.Ready, vango.Error
resource.Match(
    vango.OnPending(func() *vdom.VNode { ... }),
    vango.OnLoading(func() *vdom.VNode { ... }),
    vango.OnReady(func(data T) *vdom.VNode { ... }),
    vango.OnError(func(err error) *vdom.VNode { ... }),
)
// Action (async mutations)
action := vango.NewAction(
    func(ctx context.Context, arg A) (R, error) { ... },
    vango.CancelLatest(),  // Default concurrency policy
)
accepted := action.Run(arg)  // Returns bool
action.State()               // ActionIdle, ActionRunning, ActionSuccess, ActionError
result, ok := action.Result()
action.Error()
action.Reset()
```

### Effect Helpers (call inside Effect, return Cleanup)

```go
vango.Interval(duration, fn, opts...)             // Returns Cleanup
vango.Subscribe(stream, fn, opts...)              // Returns Cleanup  
vango.GoLatest(key, work, apply, opts...)         // Returns Cleanup
```

### Navigation & Links

```go
// Link helpers (SPA navigation with data-vango-link)
Link("/path", Text("Label"))
LinkPrefetch("/path", Text("Label"))  // With hover prefetch
NavLink(ctx, "/path", Text("Label"))  // With active class

// Programmatic navigation
ctx.Navigate("/path")
ctx.Navigate("/path", server.WithReplace())

// URL params (query string state)
param := vango.URLParam("key", defaultValue, vango.Replace)
search := vango.URLParam("q", "", vango.Replace, vango.URLDebounce(300*time.Millisecond))

// Assets (fingerprinted/manifest-aware)
src := ctx.Asset("images/logo.png") // Returns hashed path in prod
```

### Shared & Global State

```go
// Session-scoped (per browser tab)
var UserState = vango.NewSharedSignal(initial)

// Global-scoped (all users, real-time)
var OnlinePlayers = vango.NewGlobalSignal(initial)
```

### Routing File Conventions

```
routes/
├── index.go              → /            (IndexPage)
├── about.go              → /about       (AboutPage)  
├── projects/
│   ├── index.go          → /projects    (IndexPage)
│   └── [id].go           → /projects/:id (ShowPage)
├── layout.go             → Layout wrapper
├── middleware.go         → Middleware stack
└── api/
    └── health.go         → API handler (HealthGET)
```

### Client Hooks & Islands

```go
// Hooks: 60fps client-side behavior
Hook("Sortable", config)
OnEvent("reorder", func(e vango.HookEvent) { ... })

// Islands: Third-party JS libraries
JSIsland("editor", config)

// WASM: Compute-heavy client work
WASMWidget("physics", config)
```

---

## 1. Audience, Scope, and Guarantees

### 1.1 Who This Guide Is For

This guide is written for three primary audiences:

**Go Backend Developers Building Full Web Apps**

You're comfortable with Go and want to build complete, interactive web applications without becoming a JavaScript expert. You value type safety, direct database access, and the simplicity of deploying a single binary. Vango lets you write your entire application—from database queries to DOM updates—in Go.

**Teams Migrating from SPAs**

You've experienced the pain of two-language codebases, complex state synchronization, and 200KB+ JavaScript bundles. You're looking for a simpler architecture that keeps most logic server-side while still delivering modern, interactive UIs. This guide will help you translate your SPA mental models into Vango patterns.

**Teams Adopting Server-Driven UI**

You've heard about Phoenix LiveView or Laravel Livewire and want similar capabilities with Go's performance and concurrency model. You understand the value proposition of server-driven UI and are ready to learn the specific patterns and constraints of building apps this way.

### 1.2 What "Building a Vango App" Means

This guide covers the full lifecycle of building production Vango applications:

```
┌─────────────────────────────────────────────────────────────────┐
│                    VANGO APP DEVELOPMENT                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. Project Setup     →  Scaffold, configure, structure         │
│  2. Routing           →  Define pages, layouts, middleware      │
│  3. Components        →  Build UI with Go DSL                   │
│  4. State             →  Signals, memos, shared state           │
│  5. Data              →  Resources, actions, loading states     │
│  6. Forms             →  Validation, submission, UX             │
│  7. Auth              →  Sessions, middleware, authorization    │
│  8. Styling           →  CSS, theming, design systems           │
│  9. Deploy            →  Build, configure, operate              │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**What this guide does NOT cover:**
- Framework internals (VDOM diffing, binary protocol encoding, session serialization)
- Contributing to Vango itself
- Building custom runtimes or modifying the thin client

For those topics, refer to `VANGO_ARCHITECTURE_AND_GUIDE.md` and the contributor documentation.

### 1.3 Framework Contracts vs. Application Best Practices

This guide distinguishes between two types of guidance:

**Framework Contracts (Normative)**

These are rules that Vango enforces. Breaking them causes bugs, panics, or undefined behavior. Examples:
- Render functions must be pure (no I/O, no goroutine creation)
- Signal writes must happen on the session loop (use `ctx.Dispatch` from goroutines)
- Hook-order semantics must be stable (no conditional hook calls)

When we describe a contract, we use RFC 2119 keywords: **MUST**, **MUST NOT**, **SHOULD**, **SHOULD NOT**, **MAY**.

**Application Best Practices (Informative)**

These are recommendations based on production experience. Deviating from them is sometimes appropriate. Examples:
- Prefer `Resource` over manual Effect + goroutine for data loading
- Use `URLParam` for shareable, back-button-friendly state
- Keep route handlers thin; push logic into services

When we describe a best practice, we use softer language: "we recommend," "prefer," "consider."

### 1.4 The Pit of Success: Default Recommendations

Vango is designed to make the right thing easy. These defaults will serve most apps well:

| Decision | Default Recommendation | Alternative When |
|----------|------------------------|------------------|
| **Rendering Mode** | Server-driven | WASM for offline-first; Hybrid for latency-critical widgets |
| **State Location** | Server (signals) | Client (`LocalSignal`) for ephemeral UI state only |
| **Data Loading** | `Resource` in components | Route handler for nav-blocking data |
| **Mutations** | `Action` with concurrency policy | Direct handler for trivial updates |
| **Forms** | `UseForm` with validation | Manual signals for very simple cases |
| **Styling** | Tailwind CSS (Standalone) | Pure CSS if you prefer no build step |
| **Client Code** | None (server handles everything) | Hooks for 60fps; Islands for third-party libs |

**The 80/20 Rule**

For most CRUD apps, dashboards, and admin panels:
- 80% of your code is server-driven pages with no client-side JavaScript
- 15% uses client hooks for smooth drag-drop, tooltips, or animations
- 5% might need a JavaScript or WASM island for specialized widgets

Start with server-driven, add client extensions when you have a specific need.

### 1.5 The Contract Boundary

Understanding the boundary between Vango's responsibilities and your application's responsibilities is critical for debugging and correctness.

**What Vango Guarantees:**

| Guarantee | Description |
|-----------|-------------|
| **Reactive Updates** | When a signal changes, all dependent components re-render and patches are sent |
| **Session Durability** | Within `ResumeWindow`, refreshing the page restores session state (the thin client resumes by sending the last `sessionId` from `sessionStorage` in the WS handshake) |
| **Patch Correctness** | If your render functions are pure and keys are stable, patches correctly update the DOM |
| **Event Delivery** | User events are delivered to the correct handler identified by HID |
| **XSS Prevention** | Text content is escaped by default; raw HTML requires explicit opt-in |
| **CSRF Protection** | Cookie + handshake token validation protects against cross-site attacks |

**What Your Application Must Guarantee:**

| Responsibility | Description |
|----------------|-------------|
| **Render Purity** | No I/O, goroutines, or non-deterministic reads during render |
| **Hook Order Stability** | Same sequence of `NewSignal`, `NewMemo`, `Effect`, etc. on every render |
| **Session Loop Writes** | Signal mutations only on the session loop (use `ctx.Dispatch` from goroutines) |
| **Key Stability** | Dynamic lists use stable keys for correct identity tracking |
| **Input Validation** | All user input is validated server-side before use |
| **Authorization** | Every action checks the user has permission |

**When Things Go Wrong**

| Symptom | Likely Cause | Where to Look |
|---------|--------------|---------------|
| "Hook order changed" panic | Conditional hook calls or variable loop counts | §3.1.3 in the spec |
| Stale UI after mutation | Signal write from goroutine without `ctx.Dispatch` | §7.0, §3.9.6 |
| Patch mismatch reload | Unstable keys, direct DOM mutation, or render impurity | §4.4.3, §3.1.2 |
| Missing event handler | SSR/WS tree shape mismatch, HID drift, missing initial route mount | See below |
| Events received, patches = 0 | Signal created/read outside tracked render (static VNode cached) | Ensure handler executes during render |
| Blocking I/O stalls session | Blocking database/HTTP call in page handler | §2.5 (page handlers are reactive) |

**SSR/WS Alignment Issues ("Handler not found"):**

The most common cause of "handler not found" errors is a mismatch between the SSR-rendered HTML and the WebSocket session's component tree. This happens when:

1. **WS session mounts a different route than SSR** — The server must mount the same route during WS init as was rendered during SSR. The thin client sends `?path=<current-path>` to ensure this.

2. **Different tree shape between SSR and WS** — If SSR renders one component tree but the WS session renders a different one (e.g., due to non-deterministic rendering or conditional logic), HIDs won't match.

3. **Non-reactive page wrapper** — If page handlers were accidentally evaluated once and cached (a bug we fixed), HIDs would drift over time.

4. **Nested components not expanded consistently** — SSR renders nested `vango.Component` nodes inline (so their elements consume HIDs). If the WS session mounts a tree where those nested components are not expanded before HID assignment/handler registration, all subsequent HIDs drift and events may target missing handlers.

**Fix direction:** Ensure the WS session mounts the current route using the `?path=` parameter and renders the same tree shape as SSR. Both SSR and WS renders must be deterministic for the same inputs.

**"Events received, patches = 0" Issues:**

If events are reaching the server but no patches are sent back, the likely cause is signals being created or read outside the tracked render context:

1. **Static VNode cached at module level** — If you cache a VNode that closes over signals, updates to those signals won't trigger re-renders because the VNode isn't part of the reactive tree.

2. **Signal created outside `vango.Func`** — Signals must be created inside a tracked render function to participate in the reactive system.

**Fix direction:** Ensure all signal creation and reads happen during render execution (inside `vango.Func` or page handlers). Don't cache VNodes that depend on reactive state.

### 1.6 Reading This Guide Alongside the Spec

This guide uses shorthand and focuses on application patterns. When you need the precise, normative contract:

- **Render Rules** → `VANGO_ARCHITECTURE_AND_GUIDE.md` §3.1.2
- **Signal/Memo/Effect API** → §3.9.4, §3.9.5, §3.9.6
- **Resource and Action** → §3.9.7, §3.10.2
- **Routing Contracts** → §9.1–9.9
- **Protocol Specification** → §22

This guide will link to specific spec sections when introducing concepts. If a pattern in this guide seems to conflict with the spec, the spec is authoritative.

---

## 2. Mental Model: How Vango Apps Work

Understanding how Vango works at a conceptual level will help you write better applications and debug issues faster. This section provides the mental models you need.

### 2.1 The Big Picture: Server-Driven UI

In a traditional SPA, your JavaScript code runs in the browser, maintains state locally, and makes API calls to a backend:

```
Traditional SPA:
┌─────────────────────┐          ┌─────────────────────┐
│      Browser        │          │       Server        │
│  ┌───────────────┐  │   API    │  ┌───────────────┐  │
│  │ React/Vue/etc │◄─┼──────────┼─►│ REST/GraphQL  │  │
│  │ State, Logic  │  │  JSON    │  │ Data Access   │  │
│  │ Rendering     │  │          │  └───────────────┘  │
│  └───────────────┘  │          │                     │
└─────────────────────┘          └─────────────────────┘
```

In Vango, your Go code runs on the server. The browser runs a minimal client (~12KB) that:
1. Captures user events and sends them to the server
2. Receives binary patches and applies them to the DOM

```
Vango Server-Driven:
┌─────────────────────┐          ┌─────────────────────┐
│      Browser        │          │       Server        │
│  ┌───────────────┐  │   WS     │  ┌───────────────┐  │
│  │ Thin Client   │◄─┼──────────┼─►│ Vango Runtime │  │
│  │ Event Capture │  │  Binary  │  │ Components    │  │
│  │ Patch Apply   │  │  Patches │  │ State, Logic  │  │
│  └───────────────┘  │          │  │ Data Access   │  │
│                     │          │  └───────────────┘  │
└─────────────────────┘          └─────────────────────┘
```

**Key insight**: Your components, signals, and business logic all run on the server. The browser is just a display and input device.

### 2.2 Request Lifecycle: Initial Load

When a user visits your app:

```
1. Browser requests   GET /projects/123
                              │
                              ▼
2. Server matches route   → ProjectPage(ctx, id=123)
                              │
                              ▼
3. Component renders      → Creates signals, builds VNode tree
                              │
                              ▼
4. VNode → HTML          → SSR produces full HTML document
                              │
                              ▼
5. HTML sent to browser   ← User sees content immediately
                              │
                              ▼
6. Thin client loads      → ~12KB JavaScript initializes
                              │
                              ▼
7. WebSocket connects     → Binary handshake, session established
                              │
                              ▼
8. Page is interactive    → Events flow, patches apply
```

**User experience**: Content appears immediately (SSR). Interactivity follows within ~100-200ms as the thin client connects.

**Developer experience**: You write one component that handles both SSR and interactive updates. No hydration bugs; the server owns state.

### 2.3 Request Lifecycle: User Interaction

When a user interacts with your app:

```
1. User clicks "Complete Task" button
                              │
                              ▼
2. Thin client captures   → Click event on data-hid="h42"
                              │
                              ▼
3. Binary event sent      → {type: CLICK, hid: "h42"}
                              │
                              ▼
4. Server finds handler   → session.Handlers["h42"] = completeTask
                              │
                              ▼
5. Handler runs           → task.Status.Set("done")
                              │
                              ▼
6. Signal triggers render → Affected components re-render
                              │
                              ▼
7. Diff: old vs new       → Only changed nodes produce patches
                              │
                              ▼
8. Binary patches sent    → [{SET_TEXT, "h17", "✓ Done"}]
                              │
                              ▼
9. Client applies patches → DOM updates, user sees "✓ Done"
```

**Latency**: Typical round-trip is 50-100ms. For most applications, this feels instant. For 60fps interactions (drag-drop, drawing), use client hooks.

### 2.4 Session and Event Loop

Each browser tab maintains a WebSocket connection with its own server-side session:

```go
// Conceptual model of a session (simplified)
type Session struct {
    ID         string
    Conn       *websocket.Conn        // WebSocket connection
    Signals    map[uint32]*SignalBase // All signals for this session
    Components map[uint32]*Component  // Mounted component tree
    Handlers   map[string]func()      // HID → event handler mapping
    LastTree   *VNode                 // Previous render for diffing
}
```

The session runs an **event loop** that processes one event at a time:

```go
// Simplified event loop
for event := range session.events {
    // 1. Find and run the handler
    handler := session.Handlers[event.HID]
    handler()  // This may Set/Update signals
    
    // 2. Re-render components whose signals changed
    session.renderDirtyComponents()
    
    // 3. Diff and send patches
    patches := diff(session.LastTree, session.CurrentTree)
    session.sendPatches(patches)
    session.LastTree = session.CurrentTree
}
```

**Key constraint**: All signal writes happen on this event loop. If you do async work in a goroutine, you **must** marshal results back via `ctx.Dispatch(func() { ... })`.

### 2.5 Route Handlers and Render Functions

> **IMPORTANT UPDATE**: Page handlers registered via `app.Page(...)` are now **reactive**. They are wrapped in `vango.Func` internally and execute during the component render cycle, not once per navigation. This means page handlers **must follow render-purity and hook-order rules** just like any other render function.

Vango page handlers produce UI and participate in the reactive system:

**Page Handlers** (Reactive—execute during render)

```go
// Page handler signature
func IndexPage(ctx vango.Ctx) *vango.VNode

// With parameters
func ShowPage(ctx vango.Ctx, p Params) *vango.VNode

// Runs when: component renders (on mount AND on reactive updates)
// Must be: PURE (no blocking I/O, no goroutines, stable hook order)
// Produces: VNode tree for this route
```

Page handlers are the entry point for a page. They receive the routing context and URL parameters. Because they execute as part of the reactive render cycle:

- **DO NOT** perform blocking I/O (database queries, HTTP calls) directly
- **DO NOT** spawn goroutines
- **DO** create signals, memos, effects, and resources inside them
- **DO** follow hook-order rules (same sequence of hooks on every render)

**Correct Pattern—Use Resource for Data Loading:**

```go
func ShowPage(ctx vango.Ctx, p Params) *vango.VNode {
    return ProjectPageComponent(p.ID)
}

func ProjectPageComponent(id int) vango.Component {
    return vango.Func(func() *vango.VNode {
        ctx := vango.UseCtx()

        // Resource handles async loading properly
        project := vango.NewResource(func() (*Project, error) {
            return services.Projects.GetByID(ctx.StdContext(), id)
        })

        return project.Match(
            vango.OnLoading(ProjectSkeleton),
            vango.OnError(ErrorCard),
            vango.OnReady(ProjectView),
        )
    })
}
```

**Render Functions** (Run on state changes)

```go
func TaskList(tasks Signal[[]Task]) vango.Component {
    return vango.Func(func() *vango.VNode {
        // Runs when: tasks signal changes
        // Must be: pure (no I/O, no goroutines)
        // Produces: VNode tree for this component
        return Ul(
            Range(tasks.Get(), func(t Task, _ int) *vango.VNode {
                return Li(Key(t.ID), Text(t.Name))
            }),
        )
    })
}
```

Both page handlers and render functions inside `vango.Func` are reactive. They re-run automatically when any signal they read (via `.Get()`) changes. They **must** be pure—no I/O, no side effects.

**When to use what:**

| Scenario | Use |
|----------|-----|
| Load data for a page | `Resource` inside page handler or component |
| Async mutations | `Action` with concurrency policy |
| Transform or filter existing data | `Memo` |
| React to user input | Event handler → signal `.Set()` |
| Prefetch data before navigation | Router-level prefetch (future feature) |

### 2.6 Where State Lives: The Three Modes

Vango supports three rendering modes. Understanding them helps you make architecture decisions:

```
┌─────────────────────────────────────────────────────────────────┐
│                        VANGO MODES                              │
├─────────────────┬─────────────────┬─────────────────────────────┤
│  SERVER-DRIVEN  │     HYBRID      │           WASM              │
│   (Default)     │                 │                             │
├─────────────────┼─────────────────┼─────────────────────────────┤
│ State lives on  │ Most state on   │ State lives in browser      │
│ the server      │ server; some    │ WASM memory                 │
│                 │ client-side     │                             │
├─────────────────┼─────────────────┼─────────────────────────────┤
│ Signals update  │ Server signals  │ Signals update locally      │
│ via WS patches  │ + LocalSignals  │ No server round-trip        │
├─────────────────┼─────────────────┼─────────────────────────────┤
│ ~12KB client    │ ~12KB + hooks   │ ~300KB+ WASM                │
├─────────────────┼─────────────────┼─────────────────────────────┤
│ Requires        │ Requires        │ Works fully offline         │
│ connection      │ connection      │                             │
├─────────────────┼─────────────────┼─────────────────────────────┤
│ Best for: CRUD, │ Best for: apps  │ Best for: offline-first,    │
│ dashboards,     │ with some 60fps │ latency-critical apps       │
│ admin panels    │ interactions    │ (games, drawing tools)      │
└─────────────────┴─────────────────┴─────────────────────────────┘
```

**The same component code works in all modes**:

```go
func Counter(initial int) vango.Component {
    return vango.Func(func() *vango.VNode {
        count := vango.NewSignal(initial)

        return Div(
            H1(Textf("Count: %d", count.Get())),
            Button(OnClick(count.Inc), Text("+")),
            Button(OnClick(count.Dec), Text("-")),
        )
    })
}
```

| Mode | Where `count` lives | Where `OnClick` runs | How DOM updates |
|------|---------------------|---------------------|-----------------|
| Server-Driven | Server memory | Server event loop | Binary patches over WS |
| Hybrid | Server (or client if using `LocalSignal`) | Depends on signal scope | Mixed |
| WASM | Browser WASM memory | Browser WASM | Direct DOM manipulation |

**Start with server-driven mode** and only add client-side execution when you have a specific need (offline support, sub-16ms responsiveness, or heavy client computation).

### 2.7 Understanding Patches and HIDs

When a component re-renders, Vango doesn't send a new HTML document. It computes the **diff** between the old and new VNode trees and sends only the changes as **patches**.

**Example**: A counter incrementing from 5 to 6

```html
<!-- Before -->
<div class="counter">
    <h1 data-hid="h1">Count: 5</h1>
    <button data-hid="h2">+</button>
</div>

<!-- After (only the text changed) -->
<div class="counter">
    <h1 data-hid="h1">Count: 6</h1>  <!-- Just this text -->
    <button data-hid="h2">+</button>
</div>

<!-- Patch sent: -->
[SET_TEXT, hid="h1", "Count: 6"]
```

**HIDs (Hydration IDs)** identify elements for:
1. **Event routing**: When user clicks `data-hid="h2"`, server looks up `handlers["h2"]`
2. **Patch targeting**: When text changes, patch targets `hid="h1"`

**Wire format efficiency**: HIDs are numeric on the wire. `data-hid="h42"` becomes varint `42`. Combined with binary patch encoding, the protocol is extremely compact.

### 2.8 Why Keys Matter for Lists

When rendering dynamic lists, **keys** provide stable identity:

```go
// Good: Stable key based on item identity
Range(tasks, func(task Task, _ int) *vango.VNode {
    return Li(Key(task.ID), Text(task.Title))
})

// Bad: No key (or index-based key)
Range(tasks, func(task Task, i int) *vango.VNode {
    return Li(Key(i), Text(task.Title))  // Index changes when list reorders!
})
```

**Why stable keys matter:**

1. **Correct patching**: When you reorder items, Vango can emit `MOVE_NODE` instead of deleting and recreating elements
2. **Preserved state**: If list items contain component state (signals, form inputs), keys preserve that state across reorders
3. **Efficient diffs**: Less DOM churn means smaller patches and faster updates

**Rule of thumb**: Use a stable, unique identifier from your domain (database ID, UUID) as the key.

### 2.9 Translating Client-Side State to Server Signals

If you're coming from SPAs, here's how to translate your thinking. **Important**: While Vango has hook-order rules similar to React, the execution model is closer to **SolidJS** (fine-grained signal dependencies) than React (component-level re-renders). Signals track dependencies precisely—only affected computations re-run when a signal changes.

#### State Location

**SPA**: State lives in the browser. You architect around client state management (Redux, Zustand, Pinia) and sync with the server via API calls.

**Vango**: State lives on the server by default. There's no synchronization problem because there's only one source of truth. The browser shows a projection of server state.

```
SPA:                              Vango:
┌─────────────────────┐           ┌─────────────────────┐
│ Browser State       │           │ Server State        │
│  ↕ (sync problem)   │           │  ↓ (patches)        │
│ Server State        │           │ Browser (display)   │
└─────────────────────┘           └─────────────────────┘
```

#### Event Handling

**SPA**: Events run JavaScript handlers in the browser. Updates are computed locally. Server is called for mutations.

**Vango**: Events are captured and sent to the server. Handlers run on the server. Patches are sent back. The browser is an input/output device.

```go
// SPA (React)
const handleClick = () => {
  setCount(count + 1);        // Local state update
  api.saveCount(count + 1);   // Server sync
};

// Vango
OnClick(func() {
    count.Inc()  // State update on server; patch sent automatically
})
```

Vango event handlers are type-safe: each `On*` helper supports a small set of function signatures.

```go
// Input/change: shorthand value or full event payload
OnInput(func(value string) { query.Set(value) })
OnInput(func(e vango.InputEvent) { query.Set(e.Value) })

// Click: omit payload or request mouse coordinates/modifiers
OnClick(func() { count.Inc() })
OnClick(func(e vango.MouseEvent) { log.Printf("%d,%d", e.ClientX, e.ClientY) })

// Submit: use FormData when you need fields
OnSubmit(func() { submit.Run() })
OnSubmit(func(data vango.FormData) {
    _ = data.Get("name")
    submit.Run()
})

// Browser behavior and timing are expressed with event modifiers
OnSubmit(vango.PreventDefault(func() { submit.Run() }))
OnInput(vango.Debounce(300*time.Millisecond, func(value string) { query.Set(value) }))
```

#### Data Fetching

**SPA**: Components fetch data, manage loading states, handle caching, deal with stale data and cache invalidation.

**Vango**: Components can access the database directly. There's no API layer for your own UI. Loading states are handled by `Resource`.

```go
// SPA: useQuery, SWR, React Query...
const { data, isLoading, error } = useQuery(['user', id], () => fetchUser(id));

// Vango: Resource wraps async loading with states
user := vango.NewResource(func() (*User, error) {
    return db.Users.FindByID(ctx.StdContext(), userID)
})

// Use directly—no JSON, no API contracts, no cache sync
return user.Match(
    vango.OnLoading(func() *vango.VNode { return Spinner() }),
    vango.OnReady(func(u *User) *vango.VNode { return UserCard(u) }),
    vango.OnError(func(err error) *vango.VNode { return ErrorCard(err) }),
)
```

#### Routing

**SPA**: Client-side router intercepts navigation, manages history, lazy-loads route bundles, calls APIs for route data.

**Vango**: Navigation triggers server-side route handler. Server sends patches (or full page for first load). URLs are still shareable; back button still works.

```go
// Navigation looks like normal links
A(Href("/projects/123"), Text("View Project"))

// Or programmatic
Button(OnClick(func() {
    ctx.Navigate("/projects/123")
}), Text("Go to Project"))
```

#### Offline and Latency Concerns

**SPA**: Works offline once loaded (if you build that). Every interaction is local-first with eventual server sync.

**Vango (Server-Driven)**: Requires connection. Every interaction has network latency (~50-100ms).

**Vango (Hybrid/WASM)**: Can work offline. Can have local-first interactions. You opt in where needed.

**When latency matters**: For most CRUD apps, 50-100ms is imperceptible. For drag-drop, drawing, or games, use client hooks (60fps on client, commit on mouse-up) or WASM islands.

### 2.10 The Single-Writer Session Loop

The most important correctness rule in Vango:

> **All signal writes must happen on the session event loop.**

The session processes one event at a time. This ensures:
1. No race conditions on signal state
2. Predictable render order
3. Atomic commit of patches

**What this means for your code:**

```go
// ✅ Correct: Write in event handler (runs on session loop)
Button(OnClick(func() {
    count.Inc()  // Fine—we're on the session loop
}), Text("+"))

// ✅ Correct: Write via ctx.Dispatch from goroutine
vango.Effect(func() vango.Cleanup {
    ctx := vango.UseCtx()
    id := userID.Get()
    
    go func(id int) {
        user, err := fetchUser(id)
        
        // Marshal back to session loop before writing signals
        ctx.Dispatch(func() {
            if userID.Peek() != id {
                return  // Stale result
            }
            if err != nil {
                errorState.Set(err)
                return
            }
            userData.Set(user)
        })
    }(id)
    
    return nil
})

// ❌ Wrong: Write directly from goroutine
vango.Effect(func() vango.Cleanup {
    go func() {
        user, _ := fetchUser(userID.Get())
        userData.Set(user)  // PANIC: not on session loop!
    }()
    return nil
})
```

**Prefer structured helpers**: `Resource`, `Action`, `Interval`, `Subscribe`, and `GoLatest` handle the dispatch logic correctly. Raw `Effect` + goroutine is the low-level primitive, not the default.

### 2.11 Reactivity Flow

Understanding how changes propagate helps you design efficient components:

```
┌──────────────────────────────────────────────────────────────────┐
│                      REACTIVITY FLOW                             │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. User Action          Button click, input, form submit        │
│         │                                                        │
│         ▼                                                        │
│  2. Event Handler        signal.Set(newValue)                    │
│         │                                                        │
│         ▼                                                        │
│  3. Mark Dirty           Components/memos that read this signal  │
│         │                                                        │
│         ▼                                                        │
│  4. Recompute Memos      Derived values recalculate              │
│         │                                                        │
│         ▼                                                        │
│  5. Re-render Components Dirty components produce new VNodes     │
│         │                                                        │
│         ▼                                                        │
│  6. Diff                 Compare old VNode tree to new           │
│         │                                                        │
│         ▼                                                        │
│  7. Encode Patches       Minimal binary diff                     │
│         │                                                        │
│         ▼                                                        │
│  8. Send to Client       Over WebSocket                          │
│         │                                                        │
│         ▼                                                        │
│  9. Apply to DOM         Thin client updates browser DOM         │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

**Granularity**: Vango tracks dependencies at the component level, not the expression level. A component re-renders if any signal it reads changes. Use memos to isolate expensive computations and avoid unnecessary work.

```go
// Expensive filter runs every time ANY signal in this component changes
func BadExample() vango.Component {
    return vango.Func(func() *vango.VNode {
        filter := filterSignal.Get()
        items := hugeList.Get()  // 10,000 items
        
        // This runs on EVERY render
        filtered := expensiveFilter(items, filter)
        
        return ItemList(filtered)
    })
}

// Good: Memo caches the filtered result
func GoodExample() vango.Component {
    return vango.Func(func() *vango.VNode {
        filtered := vango.NewMemo(func() []Item {
            // Only runs when filterSignal or hugeList changes
            return expensiveFilter(hugeList.Get(), filterSignal.Get())
        })
        
        return ItemList(filtered.Get())
    })
}
```

### 2.12 Summary: The Vango Mental Model

1. **Your code runs on the server**. The browser is a thin display/input layer.

2. **State is server-side by default**. No client-server sync problems.

3. **Events go up, patches come down**. User actions become events; state changes become patches.

4. **One event at a time per session**. The session loop ensures consistency.

5. **Render functions are pure**. No I/O, no side effects, no goroutines.

6. **Use structured primitives**. `Resource` for loading, `Action` for mutations, structured helpers for async work.

7. **Keys provide identity**. Stable keys enable efficient diffs for lists.

8. **Client extensions exist**. Hooks for 60fps, islands for third-party libs, WASM for offline-first.

9. **Progressive enhancement always**. Links/forms work without JS; enhanced when connected.

10. **The spec is authoritative**. When in doubt, consult `VANGO_ARCHITECTURE_AND_GUIDE.md`.

## 3. Install, Requirements, and Tooling

This section covers everything you need to set up a Vango development environment and production build pipeline.

### 3.1 System Requirements

**Go Version**

Vango requires **Go 1.22 or later**. We recommend always using the latest stable Go release.

```bash
# Check your Go version
go version
# go version go1.22.0 darwin/arm64

# Install or update Go: https://go.dev/dl/
```

**Node.js (Not Required)**

Vango does not require Node.js for its core features or for the default Tailwind CSS pipeline. Tailwind is managed via a standalone binary that Vango handles automatically.

Node.js is only needed if you explicitly choose to use other JavaScript-based tooling in your project (like custom PostCSS plugins not supported by the standalone binary).

**Operating System**

Vango works on macOS, Linux, and Windows. For production deployments, we recommend Linux containers.

### 3.2 Installing the Vango CLI

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

### 3.3 Go Module Setup

Vango apps are standard Go modules. Your `go.mod` should import the Vango framework:

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

### 3.4 Tailwind CSS Pipeline (Default Template)

The default template uses Tailwind CSS for styling. Here's how it works:

```
┌─────────────────────────────────────────────────────────────────┐
│                    TAILWIND BUILD PIPELINE                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. Source Files       app/**/*.go (Class("...") calls)        │
│         │                                                        │
│         ▼                                                        │
│  2. Tailwind Scan      Extracts class names from Go files        │
│         │                                                        │
│         ▼                                                        │
│  3. Generate CSS       Only includes classes actually used       │
│         │                                                        │
│         ▼                                                        │
│  4. Output             public/styles.css                         │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Configuration Files:**

```
myapp/
├── vango.json            # Tailwind config is here
└── app/styles/
    └── input.css         # Tailwind source (@import "tailwindcss")
```

**Development:**

```bash
# Tailwind runs automatically with vango dev.
# No Node.js required!
vango dev
```

**Customizing Tailwind:**

Vango uses Tailwind CSS v4. Most configuration is done directly in your CSS using `@theme` blocks. If you need a `tailwind.config.js`, you can still use one, but it is optional.

```javascript
// tailwind.config.js (Optional in v4)
export default {
  theme: {
    extend: {
      colors: {
        primary: '#3b82f6',
      },
    },
  },
}
```

**Scanner Limitations:**

Tailwind's scanner uses regex-based extraction to find class names in your Go files. It looks for complete class name strings, not computed values.

```go
// ✅ WORKS: Full class names in strings
Class("bg-blue-500")                        // Detected
Class("text-red-500", "font-bold")          // Detected
Class(isActive && "bg-blue-500")            // Detected (conditional)

// ❌ WON'T WORK: Dynamic string construction
Class("btn-" + variant)                     // NOT detected
Class(fmt.Sprintf("text-%s-500", color))    // NOT detected
Class("bg-" + getColorName())               // NOT detected
```

**Recommendation**: Always use full class names in your Go code so the scanner can detect them. If you need variant-based styling, use explicit conditionals:

```go
// ✅ CORRECT: Explicit class names
func buttonClass(variant string) string {
    switch variant {
    case "primary":
        return "bg-blue-500 text-white"
    case "danger":
        return "bg-red-500 text-white"
    default:
        return "bg-gray-500 text-white"
    }
}
```

### 3.5 Pure CSS (No Build Step)

If you prefer to use pure CSS without any build step, you can disable Tailwind in `vango.json`:

1. Set `"tailwind": { "enabled": false }` in `vango.json`
2. Write CSS directly in `public/styles.css`
3. No build step for CSS—just edit and refresh

**CSS Organization:**

```css
/* public/css/app.css */

/* Design tokens */
:root {
    --color-primary: #3b82f6;
    --color-secondary: #10b981;
    --spacing-sm: 0.5rem;
    --spacing-md: 1rem;
    --spacing-lg: 2rem;
    --radius: 0.375rem;
}

/* Component styles */
.btn {
    padding: var(--spacing-sm) var(--spacing-md);
    border-radius: var(--radius);
    font-weight: 500;
}

.btn-primary {
    background: var(--color-primary);
    color: white;
}

/* Utility classes you actually use */
.flex { display: flex; }
.flex-col { flex-direction: column; }
.gap-4 { gap: var(--spacing-md); }
```

### 3.6 Development Server (`vango dev`)

The development server provides:

- **Hot reload**: Go file changes trigger recompile and browser refresh
- **Tailwind watch**: CSS rebuilds on class name changes (if using Tailwind)
- **Route regeneration**: New pages automatically register routes
- **DevTools**: Browser extension integration for debugging

```bash
# Start dev server
vango dev

# With options
vango dev --port 3000        # Custom port (default: 8080)
vango dev --host 0.0.0.0     # Listen on all interfaces
vango dev --no-browser       # Don't auto-open browser
```

**What `vango dev` does:**

```
1. Generates route glue (app/routes/routes_gen.go)
2. Starts Tailwind watcher (if configured)
3. Compiles and runs your app
4. Watches for file changes
5. On change: regenerate routes → recompile → restart → notify browser
```

**Development Configuration:**

During development, Vango enables:
- Detailed error pages with stack traces
- Effect strictness warnings (§3.10.4)
- DevTools transaction naming with source locations
- Longer timeouts for debugging

### 3.7 Editor Setup

**VS Code (Recommended):**

Install the Go extension and configure:

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

**GoLand / IntelliJ:**

- Import the project as a Go module
- Enable "Format on Save" with goimports
- Install the Tailwind CSS plugin for class name completion

**Vim / Neovim:**

Use gopls for Go support and the tailwindcss-language-server for Tailwind completion.

### 3.8 Testing Workflow

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

**Test Organization:**

```
app/
├── routes/
│   ├── projects/
│   │   ├── [id].go           # Dynamic route /projects/:id
│   │   └── index.go          # List route /projects
│   │   └── index_test.go     # Route handler tests
├── components/
│   ├── button.go
│   └── button_test.go        # Component tests
└── services/
    ├── tasks.go
    └── tasks_test.go         # Service/logic tests
```

### 3.9 Configuration Management

**Environment Variables:**

```go
// config/config.go
package config

import "os"

type Config struct {
    DatabaseURL   string
    Port          string
    Environment   string  // "development", "staging", "production"
    SessionSecret string
}

func Load() Config {
    return Config{
        DatabaseURL:   getEnv("DATABASE_URL", "postgres://localhost/myapp_dev"),
        Port:          getEnv("PORT", "8080"),
        Environment:   getEnv("ENVIRONMENT", "development"),
        SessionSecret: mustGetEnv("SESSION_SECRET"),
    }
}

func getEnv(key, fallback string) string {
    if val := os.Getenv(key); val != "" {
        return val
    }
    return fallback
}

func mustGetEnv(key string) string {
    if val := os.Getenv(key); val != "" {
        return val
    }
    panic("required environment variable not set: " + key)
}
```

**Development vs Production:**

| Setting | Development | Production |
|---------|-------------|------------|
| Error pages | Detailed with stack traces | Generic error messages |
| Effect strictness | Warn on effect-time writes | Off (unless configured) |
| DevTools | Full transaction names with source locations | Component-level names only |
| Session store | In-memory (lost on restart) | Redis or database |
| Static caching | No caching (reload on every request) | Long-lived with fingerprinting |

### 3.10 CI/CD Setup

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
      
      - name: Generate routes
        run: vango gen routes
      
      - name: Build project (includes Tailwind)
        run: vango build
      
      - name: Run tests
        run: go test -race -v ./...
      
      - name: Build binary
        run: go build -o myapp .

  build:
    needs: test
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v4
      
      # ... build and deploy steps
```

**Reproducible Builds:**

```bash
# Pin tool versions in go.mod with tool directive (Go 1.22+)
# or use a tools.go pattern

# Pin Vango CLI version
go install github.com/vango-go/vango/cmd/vango@v1.2.3
```

---

## 4. Creating a New App

This section walks you through creating and understanding your first Vango application.

### 4.1 Scaffolding a New Project

Use `vango create` to scaffold a new project:

```bash
# Create with default template (Tailwind + example pages)
vango create myapp

# Create in current directory
vango create .

# Create with Tailwind CSS
vango create myapp --with-tailwind

# Create with minimal template (bare bones)
vango create myapp --minimal

# Create with all features (Tailwind, database, auth)
vango create myapp --full
```

**Template Options:**

| Flag | What's Included |
|------|-----------------|
| **(default)** | Standard starter with home, about, health API, navbar, footer, and Tailwind CSS |
| **--with-tailwind** | Explicitly include Tailwind CSS (managed via standalone binary) |
| **--minimal** | Just the essentials: home page, health API |
| **--with-db=postgres** | Include database setup (sqlite or postgres) |
| **--with-auth** | Include admin routes with authentication middleware |
| **--full** | All features: Tailwind, database (sqlite), auth |

### 4.2 Project Structure After Scaffolding

```
myapp/
├── main.go                   # Entry point (at repo root)
├── app/
│   ├── routes/
│   │   ├── routes_gen.go     # Generated route registration
│   │   ├── layout.go         # Root layout
│   │   ├── index.go          # Home page (/)
│   │   ├── about.go          # About page (/about)
│   │   └── api/
│   │       └── health.go     # Health API (/api/health)
│   ├── components/
│   │   ├── shared/
│   │   │   ├── navbar.go     # Navbar component
│   │   │   └── footer.go     # Footer component
│   │   └── ui/
│   │       └── .gitkeep      # Placeholder for UI components
│   ├── middleware/
│   │   └── auth.go           # Auth middleware scaffold
│   └── store/
│       └── .gitkeep          # Placeholder for shared state
├── db/
│   └── .gitkeep              # Placeholder for database code
├── public/
│   ├── styles.css            # CSS (pure CSS by default)
│   └── favicon.ico
├── vango.json                # Vango configuration
├── go.mod
└── .gitignore
```

**With `--with-tailwind` flag:**

```
myapp/
├── ...                       # Same as above, plus:
├── app/styles/
│   └── input.css             # Tailwind source (@import "tailwindcss")
└── tailwind.config.js        # Optional Tailwind configuration
```

### 4.3 Understanding the Entry Point

The entry point (`main.go` at project root) boots your Vango app:

```go
package main

import (
    "log"
    "net/http"
    
    "github.com/vango-go/vango"
    "myapp/app/routes"
)

func main() {
    // Create the Vango app with configuration
    app := vango.New(vango.Config{
        // Session configuration
        Session: vango.SessionConfig{
            ResumeWindow:  30 * time.Second,
            // Store: vango.RedisStore(redisClient), // Production
        },
        
        // Static file serving
        Static: vango.StaticConfig{
            Dir:    "public",
            Prefix: "/",
        },
        
        // Development mode (auto-detected, but can be explicit)
        DevMode: os.Getenv("ENVIRONMENT") != "production",
    })
    
    // Register routes (generated code handles this)
    routes.Register(app)
    
    // Mount into standard http.Handler
    mux := http.NewServeMux()
    mux.Handle("/", app)
    
    // Start server
    log.Println("Server starting on http://localhost:8080")
    log.Fatal(http.ListenAndServe(":8080", mux))
}
```

**Key Points:**

- `vango.New()` creates the app with configuration
- `routes.Register()` is generated and registers all your pages
- The app is a standard `http.Handler`—integrate with any Go router
- Static files are served from `public/`

### 4.4 Your First Page

Pages live in `app/routes/`. The file path Determines the URL:

```go
// app/routes/index.go → /
// app/routes/about.go → /about
// app/routes/projects/index.go → /projects
// app/routes/projects/[id].go → /projects/:id
```

**A Simple Page:**

```go
// app/routes/index.go
package routes

import (
    "github.com/vango-go/vango"
    . "github.com/vango-go/vango/el"
)

// IndexPage is the route handler for /
func IndexPage(ctx vango.Ctx) *vango.VNode {
    return Div(Class("container mx-auto p-8"),
        H1(Class("text-3xl font-bold mb-4"),
            Text("Welcome to Vango"),
        ),
        P(Class("text-gray-600"),
            Text("Build modern web apps with Go."),
        ),
    )
}
```

### 4.5 Your First Layout

Layouts wrap pages and provide shared UI (header, footer, scripts):

```go
// app/routes/layout.go
package routes

import (
    "github.com/vango-go/vango"
    . "github.com/vango-go/vango/el"
)

// Layout wraps all pages in this directory and subdirectories.
func Layout(ctx vango.Ctx, children vango.Slot) *vango.VNode {
    return Html(Lang("en"),
        Head(
            Meta(Charset("utf-8")),
            Meta(Name("viewport"), Content("width=device-width, initial-scale=1")),
            TitleEl(Text("My Vango App")),
            LinkEl(Rel("stylesheet"), Href(ctx.Asset("styles.css"))),
        ),
        Body(Class("bg-gray-50 min-h-screen"),
            Header(Class("bg-white shadow"),
                Nav(Class("container mx-auto px-4 py-3 flex gap-6"),
                    Link("/", Class("font-bold"), Text("Home")),
                    Link("/about", Class("text-gray-600 hover:text-gray-900"), 
                        Text("About")),
                ),
            ),
            Main(Class("container mx-auto px-4 py-8"),
                children,  // Page content goes here
            ),
            Footer(Class("container mx-auto px-4 py-8 text-gray-500 text-sm"),
                Text("© 2024 My App"),
            ),
            // VangoScripts() is automatically injected at the end of Body if not present
        ),
    )
}
```

### 4.6 Your First Interactive Component

Now let's add state and event handling:

```go
// app/routes/counter.go → /counter
package routes

import (
    "github.com/vango-go/vango"
    . "github.com/vango-go/vango/el"
)

func CounterPage(ctx vango.Ctx) *vango.VNode {
    return Div(Class("space-y-4"),
        H1(Class("text-2xl font-bold"), Text("Counter Example")),
        Counter(0),
    )
}

func Counter(initial int) vango.Component {
    return vango.Func(func() *vango.VNode {
        // Create reactive state
        count := vango.NewSignal(initial)
        
        return Div(Class("flex items-center gap-4"),
            Button(
                Class("px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"),
                OnClick(count.Dec),
                Text("-"),
            ),
            Span(Class("text-2xl font-mono w-16 text-center"),
                Textf("%d", count.Get()),
            ),
            Button(
                Class("px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"),
                OnClick(count.Inc),
                Text("+"),
            ),
        )
    })
}
```

**What happens when you click "+":**

1. Thin client captures click on the button (identified by HID)
2. Binary event sent to server: `{type: CLICK, hid: "h3"}`
3. Server finds handler: `count.Inc`
4. `count.Inc()` updates the signal value
5. Component re-renders, producing new VNode tree
6. Diff detects text change: `"0"` → `"1"`
7. Patch sent: `[SET_TEXT, hid="h2", "1"]`
8. Thin client updates the DOM
9. User sees "1" (~50-100ms total)

### 4.7 Running Your App

```bash
# Start development server (recommended)
vango dev

# Or manually:
go run main.go        # Run the app
```

Visit `http://localhost:8080` to see your app.

### 4.8 Choosing a Rendering Mode

Most apps should use **server-driven mode** (the default). Here's when to consider alternatives:

| Mode | When to Use | How to Enable |
|------|-------------|---------------|
| **Server-Driven** | CRUD apps, dashboards, most web apps | Default—no configuration needed |
| **Hybrid** | Apps with some 60fps interactions | Add hooks or WASM islands as needed |
| **Full WASM** | Offline-first, latency-critical apps | Set `"mode": "wasm"` in vango.json |

**Start server-driven** and add client extensions when you have a specific need.

### 4.9 First Architecture Decisions

Before building features, make a few structural decisions:

**Domain Organization:**

```
Option A: Feature Folders (Recommended for most apps)
app/
├── routes/
│   ├── projects/
│   │   ├── index.go
│   │   └── id_/
│   │       └── index.go
│   │   ├── components.go      # Project-specific components
│   │   └── store.go           # Project-specific state
│   └── tasks/
│       └── ...

Option B: Layered Architecture (For larger teams/apps)
app/
├── routes/              # Just routing + page assembly
├── components/          # All UI components
├── services/           # Business logic
├── store/              # All state
└── repositories/       # Data access
```

**Where Database Code Lives:**

```go
// Option A: Direct in route handlers (simple apps)
type Params struct {
    ID int `param:"id"`
}

func Page(ctx vango.Ctx, p Params) *vango.VNode {
    project, err := db.Projects.FindByID(ctx.StdContext(), p.ID)
    // ...
}

// Option B: Service layer (recommended for most apps)
func Page(ctx vango.Ctx, p Params) *vango.VNode {
    project, err := services.Projects.GetByID(ctx.StdContext(), p.ID)
    // ...
}

// Option C: Resource for async loading
func Page(ctx vango.Ctx, p Params) *vango.VNode {
    return ProjectPage(p.ID)
}

func ProjectPage(id int) vango.Component {
    return vango.Func(func() *vango.VNode {
        ctx := vango.UseCtx()
        
        project := vango.NewResource(func() (*Project, error) {
            return services.Projects.GetByID(ctx.StdContext(), id)
        })
        // ...
    })
}
```

---

## 5. Project Layout and Code Organization

This section provides the recommended structure for Vango apps and guidelines for keeping codebases clean as they grow.

### 5.1 Recommended Baseline Layout

```
myapp/
├── cmd/
│   └── server/
│       └── main.go                 # Entry point, config, server startup
│
├── app/
│   ├── routes/                     # File-based routing
│   │   ├── routes_gen.go           # Generated route registration
│   │   ├── layout.go               # Root layout (wraps all pages)
│   │   ├── index.go                # Home page (/)
│   │   ├── about.go                # /about
│   │   ├── projects/
│   │   │   ├── layout.go           # Nested layout for /projects/*
│   │   │   ├── index.go            # /projects (list)
│   │   │   └── [id:int]/
│   │   │       ├── index.go        # /projects/{id}
│   │   │       ├── edit.go         # /projects/{id}/edit
│   │   │       └── tasks.go        # /projects/{id}/tasks
│   │   └── api/
│   │       └── health.go           # /api/health (API endpoint)
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
│   ├── store/                      # Shared state
│   │   ├── auth.go                 # Current user state
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
│   └── config/                     # Configuration
│       └── config.go
│
├── public/                         # Static assets (served directly)
│   ├── styles.css                  # Compiled CSS (output of Tailwind)
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
├── vango.json                      # Vango configuration
├── go.mod
├── go.sum
└── tailwind.config.js              # Optional Tailwind configuration
```

### 5.2 Routing Tree (`app/routes/`)

The `app/routes/` directory uses file-based routing. The file path maps to the URL path:

| File Path | URL | Convention |
|-----------|-----|------------|
| `routes/index.go` | `/` | Home page |
| `routes/about.go` | `/about` | Static path segment |
| `routes/projects/index.go` | `/projects` | List page |
| `routes/projects/[id].go` | `/projects/:id` | Dynamic parameter |
| `routes/projects/id_/edit.go` | `/projects/:id/edit` | Nested under param |
| `routes/blog/[...slug].go` | `/blog/*slug` | Catch-all |
| `routes/api/health.go` | `/api/health` | API endpoint |

**File Naming Conventions:**

| File | Purpose | Signature |
|------|---------|-----------
| `index.go` | Index page handler | `func IndexPage(ctx vango.Ctx) *vango.VNode` |
| `about.go` | Named page | `func AboutPage(ctx vango.Ctx) *vango.VNode` |
| `[id].go` | Dynamic page | `func ShowPage(ctx vango.Ctx, p Params) *vango.VNode` |
| `layout.go` | Layout wrapper | `func Layout(ctx vango.Ctx, children vango.Slot) *vango.VNode` |
| `middleware.go` | Directory middleware | `func Middleware() []router.Middleware` or `var Middleware = []router.Middleware{...}` |
| `api/*.go` | API endpoint | `func ResourceGET(ctx vango.Ctx) (*Response, error)` |

**Go Toolchain Rule (Important):** Avoid route files starting with `_` or `.` (e.g. `_layout.go`). Go ignores them, so the symbol won’t exist at build time.

**Go Import Path Rule (Important):** Avoid route *directories* that contain characters that are invalid in Go import paths (notably `[` and `]`). For nested parameter routes, use the Go-friendly directory form like `id_/index.go` instead of `[id]/index.go`.

**Go-Friendly Param Filenames (Optional):**

- `id_.go` → `:id`
- `slug___.go` → `*slug`

**Parameter Types:**

```go
// String parameter (default)
// routes/users/[username].go
type Params struct {
    Username string `param:"username"`
}
func ShowPage(ctx vango.Ctx, p Params) *vango.VNode

// Integer parameter
// routes/projects/[id:int].go
type Params struct {
    ID int `param:"id"`
}
func ShowPage(ctx vango.Ctx, p Params) *vango.VNode

	// UUID parameter
	// routes/documents/[id:uuid].go
	type Params struct {
	    // UUID parameters are validated by the router, but are represented as strings.
	    ID string `param:"id"`
	}
	func ShowPage(ctx vango.Ctx, p Params) *vango.VNode

// Catch-all parameter
// routes/docs/[...path].go
type Params struct {
    Path []string `param:"path"`
}
func DocPage(ctx vango.Ctx, p Params) *vango.VNode
```

### 5.3 Shared Components (`app/components/`)

Components that are used across multiple pages live in `app/components/`:

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

// Button renders a styled button with variant support.
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
// Button(ButtonDanger, Text("Delete"))(OnClick(handleDelete), Disabled())
```

**Component Organization:**

```
app/components/
├── button.go           # Single component per file
├── card.go
├── modal.go
├── badge.go
├── avatar.go
│
├── form/               # Group related components
│   ├── input.go
│   ├── textarea.go
│   ├── select.go
│   ├── checkbox.go
│   └── field.go        # Field wrapper with label/error
│
├── layout/             # Layout primitives
│   ├── header.go
│   ├── sidebar.go
│   ├── footer.go
│   └── container.go
│
└── data/               # Data display components
    ├── table.go
    ├── pagination.go
    └── empty_state.go
```

**Preventing Circular Dependencies:**

The dependency graph should flow one way:

```
routes → components → (no app dependencies)
routes → store → (no app dependencies)
routes → services → db

Never: components → routes (would be circular)
Never: store → routes (would be circular)
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

### 5.4 Shared State (`app/store/`)

Session-scoped and global state lives in `app/store/`:

```go
// app/store/auth.go
package store

import (
    "github.com/vango-go/vango"
    "github.com/vango-go/vango/pkg/auth"
)

// CurrentUser holds the authenticated user for this session.
// Session-scoped: each browser tab has its own value.
// This signal provides reactive updates for UI components.
var CurrentUser = vango.NewSharedSignal[*User](nil)

// IsAuthenticated is derived state.
var IsAuthenticated = vango.NewSharedMemo(func() bool {
    return CurrentUser.Get() != nil
})

// SetUser stores the user in both the reactive signal and session.
// Use this in login handlers for reactive UI + session persistence.
func SetUser(ctx vango.Ctx, user *User) {
    CurrentUser.Set(user)      // Update reactive UI
    auth.Login(ctx, user)      // Store in session (enables resume)
}

// ClearUser removes the user from signal and session.
func ClearUser(ctx vango.Ctx) {
    CurrentUser.Set(nil)
    auth.Logout(ctx)
}
```

```go
// app/store/notifications.go
package store

import "github.com/vango-go/vango"

type Toast struct {
    ID      string
    Type    string  // "success", "error", "info"
    Message string
}

var Toasts = vango.NewSharedSignal([]Toast{})

func ShowToast(toastType, message string) {
    id := generateID()
    Toasts.Set(append(Toasts.Get(), Toast{ID: id, Type: toastType, Message: message}))
    
    // Auto-dismiss after 5 seconds
    go func() {
        time.Sleep(5 * time.Second)
        // Must use Dispatch to write from goroutine
        vango.UseCtx().Dispatch(func() {
            DismissToast(id)
        })
    }()
}

func DismissToast(id string) {
    Toasts.RemoveWhere(func(t Toast) bool {
        return t.ID == id
    })
}
```

**State Scope Guidelines:**

| Scope | When to Use | Example |
|-------|-------------|---------|
| `NewSignal` (component) | UI state local to one component | Form inputs, accordion open/closed |
| `NewSharedSignal` (session) | State shared across pages for one user | Shopping cart, current user, notifications |
| `NewGlobalSignal` (all users) | Real-time shared state | Online users, live cursors, chat messages |

### 5.5 Static Assets (`public/`)

The `public/` directory is served directly by Vango:

```
public/
├── css/
│   └── app.css              # Compiled CSS (output of Tailwind)
├── js/
│   └── hooks/               # Client hook bundles
│       ├── sortable.js
│       └── tooltip.js
├── images/
│   ├── logo.svg
│   └── hero.png
├── fonts/
│   └── inter.woff2
└── favicon.ico
```

**URL Mapping:**

| File Path | URL |
|-----------|-----|
| `public/css/app.css` | `/css/app.css` |
| `public/images/logo.svg` | `/images/logo.svg` |
| `public/favicon.ico` | `/favicon.ico` |

**Cache Headers:**

In production, Vango applies appropriate caching:
- Fingerprinted assets (`app.abc123.css`): Long-lived cache
- Non-fingerprinted assets: Short cache with revalidation

### 5.6 Internal Packages (`internal/`)

Go's `internal/` directory contains private packages that can't be imported by external code:

```go
// internal/services/projects.go
package services

import (
    "context"
    "myapp/internal/db" // Assuming db package is also internal
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

func (s *ProjectService) GetByID(ctx context.Context, id int) (*db.Project, error) {
    return s.db.GetProject(ctx, id)
}

type CreateProjectInput struct {
    Name        string
    Description string
    OwnerID     int
}

func (s *ProjectService) Create(ctx context.Context, input CreateProjectInput) (*db.Project, error) {
    // Validation
    if input.Name == "" {
        return nil, ErrNameRequired // Define ErrNameRequired elsewhere
    }
    
    // Business logic
    return s.db.CreateProject(ctx, db.CreateProjectParams{
        Name:        input.Name,
        Description: input.Description,
        OwnerID:     input.OwnerID,
    })
}
```

### 5.7 Scaling Patterns

As your app grows, consider these organizational patterns:

**Feature Folders (Domain-Driven):**

Group related routes, components, and state by feature:

```
app/
├── routes/
│   ├── projects/
│   │   ├── index.go
│   │   ├── id_/
│   │   │   └── index.go
│   │   ├── components/      # Project-specific components
│   │   │   ├── project_card.go
│   │   │   └── task_list.go
│   │   └── store/           # Project-specific state
│   │       └── filters.go
│   └── billing/
│       ├── index.go
│       ├── components/
│       └── store/
```

**Shared Components Library:**

For large apps or multi-app organizations, extract a shared component library:

```
# Separate module for shared components
github.com/myorg/ui/
├── button.go
├── card.go
├── modal.go
└── form/
    └── ...

# In your app
import "github.com/myorg/ui"

Button := ui.Button  // Use in your app
```

### 5.8 Naming Conventions

**Files:**

- Use `snake_case` for file names: `project_card.go`, `task_list.go`
- One component per file for major components
- Group small related components in one file

**Types and Functions:**

- `PascalCase` for exported types and functions: `ProjectCard`, `TaskList`
- Component functions return `*vango.VNode` or `vango.Component`
- Stateless components are functions: `func Button(...) *vango.VNode`
- Stateful components return `vango.Component`: `func Counter(...) vango.Component`

**Packages:**

- Short, lowercase package names: `routes`, `components`, `store`
- Avoid stuttering: `store.CartStore` → `store.Cart`

### 5.9 Keeping Page Handlers Pure

> **CRITICAL**: Page handlers are reactive—they execute during the render cycle and **must not** contain blocking I/O. Always use `Resource` for data loading.

Page handlers should:
1. Extract parameters
2. Return a component that uses `Resource` for data loading
3. Remain pure (no blocking I/O, no goroutines)

```go
type Params struct {
    ID int `param:"id"`
}

// ❌ WRONG: Blocking I/O in page handler (violates render purity!)
func ProjectShowPage_BAD(ctx vango.Ctx, p Params) *vango.VNode {
    id := p.ID
    db := getDB()
    // These blocking calls will stall the session loop!
    project, err := db.Query("SELECT * FROM projects WHERE id = ?", id)
    if err != nil {
        return ErrorPage(err)
    }

    tasks, err := db.Query("SELECT * FROM tasks WHERE project_id = ?", id)
    if err != nil {
        return ErrorPage(err)
    }

    // Business logic in render
    var completedCount int
    for _, t := range tasks {
        if t.Done {
            completedCount++
        }
    }

    return Div(/* ... */)
}

// ✅ CORRECT: Page handler delegates to component with Resource
func ProjectShowPage(ctx vango.Ctx, p Params) *vango.VNode {
    return ProjectPage(p.ID)
}

func ProjectPage(id int) vango.Component {
    return vango.Func(func() *vango.VNode {
        ctx := vango.UseCtx()
        
        data := vango.NewResource(func() (*ProjectPageData, error) {
            return services.Projects.GetPageData(ctx.StdContext(), id)
        })
        
        return data.Match(
            vango.OnLoading(func() *vango.VNode {
                return ProjectSkeleton()
            }),
            vango.OnError(func(err error) *vango.VNode {
                return ErrorCard(err)
            }),
            vango.OnReady(func(d *ProjectPageData) *vango.VNode {
                return ProjectView(d)
            }),
        )
    })
}
```

**Benefits:**

- Handlers are easy to test (mock the service)
- Business logic is reusable (call service from multiple routes)
- Clear separation of concerns
- Easier to refactor

## 6. App Entry Point and Configuration

This section covers how Vango apps boot and the full configuration surface.

### 6.1 The `main.go` Entry Point

A Vango app's entry point follows a standard pattern:

```go
// main.go
package main

import (
    "context"
    "log/slog"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

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
    app := vango.New(vango.Config{
        DevMode: cfg.Environment == "development",
        
        Session: vango.SessionConfig{
            ResumeWindow:        30 * time.Second,
            MaxDetachedSessions: 10000,
            // Store: vango.RedisStore(redisClient), // Production
        },
        
        Static: vango.StaticConfig{
            Dir:    "public",
            Prefix: "/",
        },
    })
    
    // 4. Register routes
    routes.Register(app)
    
    // 5. Create HTTP server
    server := &http.Server{
        Addr:              ":" + cfg.Port,
        Handler:           app,
        ReadHeaderTimeout: 5 * time.Second,
        ReadTimeout:       30 * time.Second,
        WriteTimeout:      30 * time.Second,
        IdleTimeout:       60 * time.Second,
    }
    
    // 6. Graceful shutdown
    go func() {
        sigChan := make(chan os.Signal, 1)
        signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
        <-sigChan
        
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        
        slog.Info("shutting down server...")
        if err := server.Shutdown(ctx); err != nil {
            slog.Error("server shutdown error", "error", err)
        }
    }()
    
    // 7. Start server
    slog.Info("server starting", "addr", server.Addr, "env", cfg.Environment)
    if err := server.ListenAndServe(); err != http.ErrServerClosed {
        slog.Error("server error", "error", err)
        os.Exit(1)
    }
}
```

`app.Run()` uses Vango's internal server with HTTP timeout defaults (`ReadHeaderTimeout=5s`, `ReadTimeout=30s`, `WriteTimeout=30s`, `IdleTimeout=60s`). If you need different values, update the server config before starting:

```go
app := vango.New(cfg)
routes.Register(app)

srv := app.Server().Config()
srv.ReadHeaderTimeout = 2 * time.Second
srv.ReadTimeout = 20 * time.Second
srv.WriteTimeout = 20 * time.Second
srv.IdleTimeout = 60 * time.Second

app.Run(":8080")
```

If you embed Vango as an `http.Handler` in your own `http.Server`, you must set equivalent timeouts on that server.

### 6.2 Entry Point Responsibilities

The entry point handles these concerns in order:

| Step | Responsibility | Notes |
|------|----------------|-------|
| **1. Config** | Load and validate configuration | Fail fast on missing required values |
| **2. Dependencies** | Initialize DB, cache, external clients | Pass to routes/services via closures or DI |
| **3. App Creation** | Create Vango app with config | Set mode, session, static, security options |
| **4. Route Registration** | Call generated `routes.Register()` | Generated code handles file-based routing |
| **5. Server Setup** | Configure http.Server | Set HTTP timeouts (especially ReadHeaderTimeout); WebSocket timeouts live in SessionConfig |
| **6. Graceful Shutdown** | Handle SIGINT/SIGTERM | Allow in-flight requests to complete |
| **7. Start** | Listen and serve | Log startup info |

### 6.3 Vango Configuration Reference

#### Core Configuration

```go
vango.Config{
    // Development mode enables detailed errors, strict effect warnings,
    // and source-location transaction names
    DevMode: true,
    
    // Session configuration (see 6.4)
    Session: vango.SessionConfig{...},
    
    // Static file serving (see 6.5)
    Static: vango.StaticConfig{...},
    
    // Security settings (see 6.6)
    Security: vango.SecurityConfig{...},
    
    // Storm budgets for DoS protection (see 6.7)
    StormBudget: vango.StormBudgetConfig{...},
    
    // Logging and observability
    Logger: slog.Default(),
}
```

### 6.4 Session Configuration

```go
vango.SessionConfig{
    // How long a disconnected session is kept in memory
    // Allows refresh/reconnect to restore state
    ResumeWindow: 30 * time.Second,
    
    // Maximum detached sessions in memory (DoS protection)
    MaxDetachedSessions: 10000,
    
    // Maximum sessions per IP (DoS protection)
    MaxSessionsPerIP: 100,

    // Evict the oldest detached session when MaxSessionsPerIP is reached
    EvictOnIPLimit: true,
    
    // What to do when limits are exceeded
    EvictionPolicy: vango.EvictionLRU,
    
    // Optional: Persist sessions for server restarts
    Store: vango.RedisStore(redisClient),
    // Or: Store: vango.PostgresStore(db),
    
    // Storm budgets per session (see §3.10.5 in spec)
    StormBudget: vango.StormBudgetConfig{
        MaxResourceStartsPerSecond: 100,
        MaxActionStartsPerSecond:   50,
        MaxGoLatestStartsPerSecond: 100,
        MaxEffectRunsPerTick:       50,
        WindowDuration:             time.Second,
        OnExceeded:                 vango.BudgetThrottle,
    },
}
```

Important: If TrustedProxies is not configured correctly when running behind a load balancer
(e.g., Nginx or a cloud L7 proxy), RemoteAddr will likely be the balancer's IP. This collapses
all users into a single IP bucket and will trigger MaxSessionsPerIP immediately.

**Session Durability Tiers:**

| Tier | What Survives | Configuration |
|------|---------------|---------------|
| **None** | Nothing (fresh session on every connect) | `ResumeWindow: 0` |
| **In-Memory** | Refresh/reconnect within ResumeWindow | `ResumeWindow: 30*time.Second` |
| **Persistent** | Server restarts, non-sticky deploys | `Store: vango.RedisStore(...)` |

### 6.5 Static File Configuration

```go
vango.StaticConfig{
    // Directory containing static files
    Dir: "public",
    
    // URL prefix (usually "/" for root)
    Prefix: "/",
    
    // Cache control for production
    // Fingerprinted files get immutable; others get short cache
    CacheControl: vango.CacheControlProduction,
    
    // Brotli/gzip compression
    Compression: true,
    
    // Custom headers
    Headers: map[string]string{
        "X-Content-Type-Options": "nosniff",
    },
}
```

**Cache Strategies:**

| Strategy | Behavior | When to Use |
|----------|----------|-------------|
| `CacheControlNone` | No caching headers | Development |
| `CacheControlProduction` | Fingerprinted: immutable; Others: short + revalidate | Production |
| `CacheControlCustom` | You control headers | Special requirements |

### 6.6 Security Configuration

```go
vango.SecurityConfig{
    // CSRF protection (enabled when CSRFSecret is set)
    CSRFSecret: []byte("your-32-byte-secret-key-here!!"),
    
    // Origin checking for WebSocket
    AllowedOrigins: []string{
        "https://myapp.com",
        "https://www.myapp.com",
    },
    // Or allow same-origin automatically:
    AllowSameOrigin: true,
    
    // Trusted reverse proxies (for X-Forwarded-Proto)
    TrustedProxies: []string{"10.0.0.1"},
    
    // Cookie settings
    CookieSecure:   true,  // Requires HTTPS
    CookieHttpOnly: true,  // No JS access
    CookieSameSite: http.SameSiteLaxMode,
    CookieDomain:   "",
}
```

Vango enforces these cookie defaults for `ctx.SetCookie` and `ctx.SetCookieStrict`. When
`CookieSecure` is true and a request is not secure, cookies are dropped and
`SetCookieStrict` returns `server.ErrSecureCookiesRequired`. Use
`vango.WithCookieHTTPOnly(false)` for cookies that must be readable by JS.

### 6.7 Storm Budget Configuration

Storm budgets protect against runaway effects and DoS:

```go
vango.StormBudgetConfig{
    // Per-second limits for async work
    MaxResourceStartsPerSecond: 100,
    MaxActionStartsPerSecond:   50,
    MaxGoLatestStartsPerSecond: 100,
    
    // Per-tick limits
    MaxEffectRunsPerTick: 50,
    
    // Time window for rate limiting
    WindowDuration: time.Second,
    
    // What to do when exceeded
    OnExceeded: vango.BudgetThrottle, // or vango.BudgetTripBreaker
}
```

**Exceeded Behaviors:**

| Mode | Behavior | When to Use |
|------|----------|-------------|
| `BudgetThrottle` | Deny/delay new starts, surface error | Most cases |
| `BudgetTripBreaker` | Terminate/invalidate session | Critical resources |

### 6.8 Environment-Based Configuration

Structure your configuration around environments:

```go
// internal/config/config.go
package config

import (
    "os"
    "time"
)

type Config struct {
    Environment   string
    Port          string
    DatabaseURL   string
    RedisURL      string
    SessionSecret string
    
    // Derived settings
    IsDevelopment bool
    IsProduction  bool
}

func Load() Config {
    env := getEnv("ENVIRONMENT", "development")
    
    cfg := Config{
        Environment:   env,
        Port:          getEnv("PORT", "8080"),
        DatabaseURL:   mustGetEnv("DATABASE_URL"),
        RedisURL:      getEnv("REDIS_URL", ""),
        SessionSecret: mustGetEnv("SESSION_SECRET"),
        
        IsDevelopment: env == "development",
        IsProduction:  env == "production",
    }
    
    cfg.Validate()
    return cfg
}

func (c Config) Validate() {
    if c.IsProduction {
        if c.RedisURL == "" {
            panic("REDIS_URL required in production for session persistence")
        }
        if len(c.SessionSecret) < 32 {
            panic("SESSION_SECRET must be at least 32 characters")
        }
    }
}

func (c Config) VangoConfig() vango.Config {
    cfg := vango.Config{
        DevMode: c.IsDevelopment,
        Session: vango.SessionConfig{
            ResumeWindow: 30 * time.Second,
        },
        Static: vango.StaticConfig{
            Dir:    "public",
            Prefix: "/",
        },
    }
    
    if c.IsProduction {
        cfg.Session.Store = vango.RedisStore(c.RedisURL)
        cfg.Static.CacheControl = vango.CacheControlProduction
        cfg.Security.CookieSecure = true
    }
    
    return cfg
}
```

### 6.9 Configuration Categories

Understand what each configuration affects:

| Category | Settings | Correctness | Performance | Security |
|----------|----------|-------------|-------------|----------|
| **Session** | ResumeWindow, Store | ✓ | ✓ | |
| **Limits** | MaxDetachedSessions, Per-IP limits | | ✓ | ✓ |
| **Storm Budgets** | Resource/Action/Effect limits | | ✓ | ✓ |
| **CSRF** | CSRFSecret, Cookie flags | | | ✓ |
| **Origins** | AllowedOrigins, AllowSameOrigin | | | ✓ |
| **Cookies** | Secure, HttpOnly, SameSite | | | ✓ |
| **Static** | CacheControl, Compression | | ✓ | |
| **DevMode** | Error detail, effect warnings | ✓ | | |

### 6.10 Production Checklist

Before deploying to production:

```go
// ✅ Required for production
vango.Config{
    DevMode: false,  // Disable detailed errors
    
    Session: vango.SessionConfig{
        Store: vango.RedisStore(...),  // Persist sessions
        MaxDetachedSessions: 10000,    // Limit memory
        MaxSessionsPerIP:    100,      // Rate limit
    },
    
    Security: vango.SecurityConfig{
        CSRFSecret:     []byte("your-32-byte-secret-key-here!!"),
        AllowedOrigins: []string{"https://myapp.com"},  // Explicit origins
        CookieSecure:   true,
        CookieHttpOnly: true,
        CookieSameSite: http.SameSiteLaxMode,
    },
    
    Static: vango.StaticConfig{
        CacheControl: vango.CacheControlProduction,
        Compression:  true,
    },
}
```

### 6.11 Integrating with Existing Routers

Vango is a standard `http.Handler`, so it integrates with any Go router:

```go
// With Chi
import "github.com/go-chi/chi/v5"

r := chi.NewRouter()
r.Use(middleware.Logger)
r.Use(middleware.Recoverer)

// Mount Vango app
r.Mount("/", app)

// Or mount at a sub-path.
//
// IMPORTANT: Vango's WebSocket endpoint is rooted at "/_vango/*". If you mount
// the app at a sub-path (e.g. "/app"), you MUST also route "/_vango/*" to the
// same handler (or override the thin client WS URL).
r.Mount("/app", app)
r.Mount("/_vango", app)

// Add non-Vango routes
r.Get("/api/external", externalAPIHandler)
```

```go
// With Gorilla Mux
import "github.com/gorilla/mux"

r := mux.NewRouter()
r.PathPrefix("/").Handler(app)
```

```go
// With standard library
mux := http.NewServeMux()
mux.Handle("/", app)
```

---

## 7. Routing, Layouts, and Middleware

This section is the practical companion to the routing contracts in the spec (§9). It covers everything you need to build complex route structures.

### 7.1 File-Based Routing Overview

Vango uses file-based routing where the file path determines the URL:

```
app/routes/                           URL
├── index.go                         /
├── about.go                         /about
├── blog/
│   ├── index.go                     /blog
│   └── [slug].go                    /blog/:slug
├── projects/
│   ├── layout.go                    (wraps all /projects/* pages)
│   ├── middleware.go                (directory middleware)
│   ├── index.go                     /projects
│   ├── [id:int].go                  /projects/:id
│   ├── [id:int]/                    (nested directory for sub-routes)
│   │   ├── edit.go                  /projects/:id/edit
│   │   └── settings.go              /projects/:id/settings
├── api/
│   └── health.go                    /api/health (Typed API)
└── [...path].go                     /* (catch-all)
```

### 7.2 Route Handler Signatures

**Page Handlers:**

Route handlers receive `vango.Ctx` and a `Params` struct for path parameters.

```go
// No parameters
// routes/index.go
func IndexPage(ctx vango.Ctx) *vango.VNode { ... }

// With path parameters
// routes/projects/[id:int].go
type Params struct {
    ID int `param:"id"`
}

func ShowPage(ctx vango.Ctx, p Params) *vango.VNode { ... }

// Catch-all parameter
// routes/docs/[...path].go
type Params struct {
    Path []string `param:"path"`
}

func DocPage(ctx vango.Ctx, p Params) *vango.VNode { ... }
```

**Layout Handlers:**

```go
// app/routes/layout.go
func Layout(ctx vango.Ctx, children vango.Slot) *vango.VNode { ... }
```

**API Route Handlers:**

API handlers are typed functions, not raw `net/http` handlers.

```go
// routes/api/users.go

func GET(ctx vango.Ctx) ([]User, error) {
    return db.Users.List(), nil
}

type CreateRequest struct {
    Name string `json:"name"`
}

func POST(ctx vango.Ctx, body CreateRequest) (*User, error) {
    return db.Users.Create(body.Name), nil
}
```

### 7.3 Parameter Types

Specify parameter types in filenames using `[name:type]` syntax:

| Syntax | Go Type | Example | Matches |
|--------|---------|---------|---------|
| `[id]` | `string` | `/users/alice` | Any string segment |
| `[id:int]` | `int` | `/projects/123` | Integers only |
| `[id:uuid]` | `string` | `/docs/550e...` | Valid UUIDs only |
| `[...path]` | `string` or `[]string` | `/docs/a/b/c` | Remaining path |

**Type Safety:**

```go
// routes/projects/[id:int].go
type Params struct {
    ID int `param:"id"`
}

func ShowPage(ctx vango.Ctx, p Params) *vango.VNode {
    // p.ID is guaranteed to be a valid int
    // Invalid URLs like /projects/abc return 404 automatically
    
    project := vango.NewResource(func() (*Project, error) {
        return db.Projects.FindByID(ctx.StdContext(), p.ID)
    })
    
    return project.Match(/* ... */)
}
```

**Consistent Parameter Naming:**

```go
// ✅ Good: Parameter name matches domain concept
// routes/projects/[projectID:int]/tasks/[taskID:int].go
type Params struct {
    ProjectID int `param:"projectID"`
    TaskID    int `param:"taskID"`
}
func TaskPage(ctx vango.Ctx, p Params) *vango.VNode

// ❌ Avoid: Generic names are confusing with multiple params
// routes/projects/[id:int]/tasks/[id:int].go  // Which id is which?
```

### 7.4 Layout Composition

Layouts wrap pages and can be nested:

```
Request: GET /projects/123/settings

Layout hierarchy:
1. routes/layout.go              (root layout: html, head, body)
2. routes/projects/layout.go     (project layout: sidebar, nav)
3. routes/projects/[id:int]/settings.go  (page content)

Rendered structure:
<RootLayout>
  <ProjectLayout>
    <SettingsPage />
  </ProjectLayout>
</RootLayout>
```

**Root Layout:**

```go
// routes/layout.go
func Layout(ctx vango.Ctx, children vango.Slot) *vango.VNode {
    return Html(Lang("en"),
        Head(
            Meta(Charset("utf-8")),
            Meta(Name("viewport"), Content("width=device-width, initial-scale=1")),
            TitleEl(Text("My App")),
            LinkEl(Rel("stylesheet"), Href(ctx.Asset("styles.css"))),
            // Preload critical assets
            LinkEl(Rel("preload"), Href(ctx.Asset("fonts/inter.woff2")), As("font"), 
                Type("font/woff2"), Crossorigin("anonymous")),
        ),
        Body(Class("min-h-screen bg-gray-50"),
            children,
            // VangoScripts() is auto-injected if not present
        ),
    )
}
```

**Nested Layout:**

```go
// routes/projects/layout.go
func Layout(ctx vango.Ctx, children vango.Slot) *vango.VNode {
    return Div(Class("flex min-h-screen"),
        // Sidebar navigation
        Aside(Class("w-64 bg-white border-r"),
            Nav(Class("p-4 space-y-2"),
                NavLink(ctx, "/projects", Text("All Projects")),
                NavLink(ctx, "/projects/new", Text("New Project")),
            ),
        ),
        // Main content area
        Main(Class("flex-1 p-8"),
            children,
        ),
    )
}
```

**Layout with Dynamic Data:**

```go
// routes/projects/[id:int]/layout.go
func Layout(ctx vango.Ctx, children vango.Slot) *vango.VNode {
    id := ctx.Param("id") // string
    return Div(
        H2(Text(fmt.Sprintf("Project #%s", id))),
        Nav(
            NavLink(ctx, fmt.Sprintf("/projects/%s", id), Text("Dashboard")),
            NavLink(ctx, fmt.Sprintf("/projects/%s/settings", id), Text("Settings")),
        ),
        children,
    )
}

func ProjectLayoutComponent(id int, children *vango.VNode) vango.Component {
    return vango.Func(func() *vango.VNode {
        ctx := vango.UseCtx()
        
        project := vango.NewResource(func() (*Project, error) {
            return db.Projects.FindByID(ctx.StdContext(), id)
        })
        
        return project.Match(
            vango.OnLoading(func() *vango.VNode {
                return ProjectLayoutSkeleton(children)
            }),
            vango.OnReady(func(p *Project) *vango.VNode {
                return Div(Class("space-y-4"),
                    // Project header
                    Header(Class("border-b pb-4"),
                        H1(Class("text-2xl font-bold"), Text(p.Name)),
                        ProjectTabs(id),
                    ),
                    // Page content
                    children,
                )
            }),
            vango.OnError(func(err error) *vango.VNode {
                return ErrorPage(err)
            }),
        )
    })
}
```

### 7.5 Middleware

Vango supports two middleware types:

**HTTP Middleware** (runs before Vango):

```go
// app/middleware/logging.go
package middleware

import (
    "log/slog"
    "net/http"
    "time"
)

func Logging(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        next.ServeHTTP(w, r)
        
        slog.Info("request",
            "method", r.Method,
            "path", r.URL.Path,
            "duration", time.Since(start),
        )
    })
}
```

**Vango Route Middleware** (runs within Vango):

```go
// routes/admin/middleware.go
package admin

import (
    "github.com/vango-go/vango"
    "github.com/vango-go/vango/pkg/router"
    "myapp/app/store"
)

func Middleware() []router.Middleware {
    return []router.Middleware{
        router.MiddlewareFunc(func(ctx vango.Ctx, next func() error) error {
            user, _ := ctx.User().(*store.User)
            if user == nil {
                ctx.Navigate("/login?redirect="+ctx.Path(), vango.WithReplace())
                return nil // short-circuit
            }
            return next()
        }),
    }
}
```

**Middleware Stacking:**

```
Request: GET /admin/users

1. HTTP Middleware (Logging, Recovery, etc.)
   ↓
2. Vango receives request
   ↓
3. Route matching → /admin/users
   ↓
4. routes/admin/middleware.go (auth check)
 ↓
5. routes/layout.go (root layout)
 ↓
6. routes/admin/layout.go (admin layout)
 ↓
7. routes/admin/users.go (page handler)
```

**What Belongs Where:**

| Concern | HTTP Middleware | Vango Middleware |
|---------|-----------------|------------------|
| Logging | ✓ | |
| Recovery/Panic | ✓ | |
| Request ID | ✓ | |
| Rate Limiting | ✓ | |
| Auth Check (redirect) | | ✓ |
| Permission Check | | ✓ |
| Feature Flags | | ✓ |
| A/B Testing | | ✓ |

### 7.6 API Routes

API routes use typed functions that return data, not raw HTTP handlers.
Files are placed in `app/routes/api/` with descriptive names (e.g., `health.go`, `users.go`).

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

**Named function convention:** `{Resource}{METHOD}` - e.g., `UsersGET`, `UserGET`, `UsersPOST`.

**CRUD Example:**

```go
// app/routes/api/users.go
package api

import (
    "github.com/vango-go/vango"
)

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

**Request bodies (JSON):**
- If an API handler declares a body parameter (e.g. `func UsersPOST(ctx vango.Ctx, input CreateUserInput) ...`), Vango reads and JSON-decodes the HTTP request body into that type.
- By default, missing `Content-Type` is accepted, but an explicit non-JSON `Content-Type` is rejected with `415 Unsupported Media Type`.
- The default maximum body size is **1 MiB**; configure via `vango.Config{ API: vango.APIConfig{ MaxBodyBytes: ... } }`.

**Route registration (generated in routes_gen.go):**

```go
// Code generated by vango. DO NOT EDIT.
func Register(app *vango.App) {
    app.API("GET", "/api/health", api.HealthGET)
    app.API("GET", "/api/users", api.UsersGET)
    app.API("GET", "/api/users/:id", api.UserGET)
    app.API("POST", "/api/users", api.UsersPOST)
    app.API("DELETE", "/api/users/:id", api.UserDELETE)
}
```

### 7.7 Programmatic Navigation

Navigate between pages programmatically:

```go
// Push (creates history entry)
Button(OnClick(func() {
    ctx := vango.UseCtx()
    ctx.Navigate("/projects/123")
}), Text("View Project"))

// Replace (no history entry)
Button(OnClick(func() {
    ctx := vango.UseCtx()
    ctx.Navigate("/dashboard", server.WithReplace())
}), Text("Go to Dashboard"))

// With query parameters
Button(OnClick(func() {
    ctx := vango.UseCtx()
    ctx.Navigate("/search?q=golang&page=1")
}), Text("Search"))

// External navigation
Button(OnClick(func() {
    ctx := vango.UseCtx()
    ctx.NavigateExternal("https://docs.example.com")
}), Text("View Docs"))
```

### 7.8 Context Behavior in SSR vs WebSocket Renders

The `vango.Ctx` provides request information, but behavior differs between SSR and WebSocket renders. This is the **context bridge** pattern—SSR captures HTTP context, and session signals make it available during WS renders.

**During SSR (initial page load):**

The context has access to the full HTTP request:

| Method | SSR Behavior | Guaranteed |
|--------|--------------|------------|
| `ctx.Path()` | Current URL path | ✅ Yes |
| `ctx.Query()` | Query parameters | ✅ Yes |
| `ctx.Method()` | HTTP method (GET, POST, etc.) | ✅ Yes |
| `ctx.Request()` | Full `*http.Request` with headers, cookies, host | ✅ Yes |

**During WebSocket renders (reactive updates):**

The context reflects the current route but uses a **synthetic request** (implementation detail). Only path and query are guaranteed:

| Method | WS Behavior | Guaranteed |
|--------|-------------|------------|
| `ctx.Path()` | Current URL path (from `?path=` or navigation) | ✅ Yes |
| `ctx.Query()` | Query parameters (from `?path=` or navigation) | ✅ Yes |
| `ctx.Method()` | Returns `"GET"` | ❌ Synthetic default |
| `ctx.Request()` | Synthetic—no real headers/cookies/host | ❌ Not meaningful |

> **Key Point**: During WebSocket renders, `ctx.Request()` may exist but is not the original HTTP request—the original HTTP headers and cookies are not re-sent with every WebSocket message. Do not depend on `ctx.Request().Cookies()` or headers for auth decisions inside reactive components. Use session data populated at session start via the context bridge pattern below.

**Context Bridge Pattern—Capturing HTTP Context for WS Access:**

This is the standard pattern for making HTTP-only data (like authenticated user from cookies) available during WebSocket renders:

```go
// In layout during SSR, capture HTTP context into session signals
func Layout(ctx vango.Ctx, children vango.Slot) *vango.VNode {
    // During SSR, we have access to cookies via real HTTP request
    if user := auth.UserFromRequest(ctx.Request()); user != nil {
        store.CurrentUser.Set(user)  // Bridge to session-scoped signal
    }
    // ...
}

// During WS renders, access via signal (not ctx.Request())
func ProfileButton() vango.Component {
    return vango.Func(func() *vango.VNode {
        user := store.CurrentUser.Get()  // ✅ Works in both SSR and WS
        // ❌ Don't: ctx.Request().Cookies()—not available during WS renders
        // ...
    })
}
```

See §17 for the full authentication flow using this pattern.

### 7.9 Routing Style Guide

Follow these conventions for maintainable route structures:

**Naming:**

```go
// ✅ Good: Lowercase, hyphenated for multi-word
routes/user-settings.go          → /user-settings
routes/api/project-members.go    → /api/project-members

// ❌ Avoid: camelCase or underscores
routes/userSettings.go           → /userSettings (inconsistent)
routes/user_settings/index.go    → /user_settings (unconventional)
```

**Grouping:**

```go
// ✅ Good: Group related routes
routes/
├── projects/
│   ├── index.go                // List
│   ├── new.go                  // Create form
│   └── [id:int]/
│       ├── index.go            // Show
│       ├── edit.go             // Edit form
│       └── settings.go         // Settings

// ❌ Avoid: flattening a route group into unrelated filenames
routes/
├── projects-index.go
├── projects-new.go
├── projects-[id].go
```

**API vs Pages:**

```go
// ✅ Good: Clear separation
routes/
├── api/                        // External HTTP APIs
│   └── v1/
│       └── projects/
│           └── route.go
└── projects/                   // Server-driven pages
    └── index.go

// ❌ Avoid: Mixing API and pages
routes/
├── projects/
│   ├── index.go                // Page
│   └── route.go                // API (confusing)
```

### 7.10 Common Routing Pitfalls

**Typed Params (Use Constraints to Reject Bad URLs):**

```go
// ❌ Problem: Untyped params match anything (including unexpected values)
routes/projects/[id].go         // /projects/:id
// /projects/abc matches and flows into your handler

// ✅ Solution: Use typed params to reject invalid matches
routes/projects/[id:int].go     // /projects/:id (int only)
// /projects/abc → 404
```

**Catch-All Gotchas:**

```go
// Catch-alls consume the rest of the path.
// routes/docs/[...path].go matches /docs/* and receives the joined remainder.
// You can't add segments after a catch-all (e.g., /docs/[...path]/edit.go).
```

**Trailing Slashes:**

```go
// Vango canonicalizes URLs by default
// /projects/ redirects to /projects
// /projects/123/ redirects to /projects/123

// Links should use canonical form
A(Href("/projects"), Text("Projects"))      // ✅
A(Href("/projects/"), Text("Projects"))     // Works, but redirects
```

**Canonicalization & Security Notes:**

- Non-canonical paths are redirected with `308 Permanent Redirect` before routing (pages + APIs).
- Invalid paths (`\`, NUL, invalid percent-escapes, `..` escaping root) return `400 Bad Request`.
- `%2F` in non-catch-all params is rejected (404) to prevent path smuggling.

**Route Priority:**

Routes are matched in this order (most specific first):

1. Static segments (`/projects/new`)
2. Typed parameters (`/projects/[id:int]`)
3. String parameters (`/projects/[slug]`)
4. Catch-all (`/docs/[...path]`)

```go
// All coexist correctly:
routes/
├── projects/
│   ├── new.go                  // 1. /projects/new (static)
│   ├── featured.go             // 1. /projects/featured (static)
│   ├── [id:int].go             // 2. /projects/123 (typed)
│   └── [slug].go               // 3. /projects/my-project (string)
```

### 7.11 Route Generation

The `vango gen routes` command generates route registration:

```go
// app/routes/routes_gen.go (generated, do not edit)
package routes

import "github.com/vango-go/vango"

func Register(app *vango.App) {
    // Layouts (hierarchical)
    app.Layout("/", Layout)
    app.Layout("/projects", projects.Layout)

    // Middleware (hierarchical)
    app.Middleware("/admin", admin.Middleware()...)

    // Pages (inherit layouts automatically)
    app.Page("/", IndexPage)
    app.Page("/about", AboutPage)
    app.Page("/projects", projects.IndexPage)
    app.Page("/projects/:id", projectsID.IndexPage)

    // API routes
    app.API("GET", "/api/health", api.HealthGET)
}
```

**When Regeneration Happens:**

- Automatically during `vango dev` when files change
- Manually via `vango gen routes`
- During `vango build`

**Troubleshooting: `routes_gen.go` keeps “reverting”**

- Make sure you’re running the `vango` binary you just built (`which vango`).
- Stop any old `vango dev` processes; an older watcher can keep overwriting generated output.

**Customizing Generation:**

```json
// vango.json
{
    "routes": {
        "dir": "app/routes",
        "output": "app/routes/routes_gen.go",
        "package": "routes"
    }
}
```

## 8. Components and UI Composition

This section turns the component model from the spec (§3) into application patterns:
- Stateless vs stateful components and when to choose each.
- Component boundaries and reusability (props, children, slots).
- Render rules (purity, no I/O in render) and how to enforce them in your app.
- Lists and keys (correctness + patch quality), and patterns for complex list UIs.
- Context usage for dependency injection and app-wide configuration (theme, locale, current user, feature flags).

It will include:
- Recommended folder structure for UI components and “component API design” conventions.
- How to write components that are stable under rerender (hook-order semantics).
- Component testing patterns that assert behavior without coupling to internals.

## 9. State Management in Real Apps

This section is the practical guide to using signals, memos, effects, resources, and actions in production applications. It covers everything from basic patterns to advanced techniques for scaling state in large apps.

### 9.1 State Primitives Overview

Vango provides five core primitives for managing state:

| Primitive | Purpose | When to Use |
|-----------|---------|-------------|
| **Signal** | Reactive container for mutable state | Local UI state, form inputs, toggles |
| **Memo** | Derived state (cached computation) | Filtered lists, computed values, selectors |
| **Effect** | Side effects triggered by state changes | Logging, analytics, external sync |
| **Resource** | Async data loading with states | Fetching data, API calls |
| **Action** | Async mutations with concurrency control | Form submissions, saves, deletes |

```go
// Quick reference
count := vango.NewSignal(0)           // Reactive value
doubled := vango.NewMemo(func() int { // Derived value
    return count.Get() * 2
})
vango.Effect(func() vango.Cleanup {      // Side effect
    log.Printf("Count is now: %d", count.Get())
    return nil
})
data := vango.NewResource(fetchData)  // Async load
save := vango.NewAction(saveData)     // Async mutation
```

### 9.2 Choosing State Scope

State scope determines where state lives and who can access it:

```
┌─────────────────────────────────────────────────────────────────┐
│                        STATE SCOPES                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Component Scope     →  NewSignal() inside component             │
│  (per-instance)         Destroyed when component unmounts        │
│                                                                  │
│  Session Scope       →  NewSharedSignal() at package level       │
│  (per-browser-tab)      Persists across navigation               │
│                                                                  │
│  Global Scope        →  NewGlobalSignal() at package level       │
│  (all users)            Shared in real-time across sessions      │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Component Scope (Default):**

```go
func Counter() vango.Component {
    return vango.Func(func() *vango.VNode {
        // Component-scoped: each Counter instance has its own count
        count := vango.NewSignal(0)
        
        return Div(
            Button(OnClick(count.Dec), Text("-")),
            Text(count.Get()),
            Button(OnClick(count.Inc), Text("+")),
        )
    })
}
```

**Session Scope (Shared Across Pages):**

```go
// store/cart.go
package store

import "github.com/vango-go/vango"

// Session-scoped: each browser tab has its own cart
var Cart = vango.NewSharedSignal([]CartItem{})
var CartTotal = vango.NewSharedMemo(func() float64 {
    total := 0.0
    for _, item := range Cart.Get() {
        total += item.Price * float64(item.Quantity)
    }
    return total
})

func AddToCart(item CartItem) {
    Cart.Set(append(Cart.Get(), item))
}

func ClearCart() {
    Cart.Set([]CartItem{})
}
```

**Global Scope (Real-Time Across Users):**

```go
// store/presence.go
package store

import "github.com/vango-go/vango"

// Global: all users see the same value
var OnlineUsers = vango.NewGlobalSignal([]User{})

func UserOnline(user User) {
    OnlineUsers.Set(append(OnlineUsers.Get(), user))
}

func UserOffline(userID int) {
    OnlineUsers.RemoveWhere(func(u User) bool {
        return u.ID == userID
    })
}
```

**Scope Selection Guide:**

| Question | If Yes → Scope |
|----------|----------------|
| Does this reset when I navigate away? | Component |
| Does this persist across pages for one user? | Session (Shared) |
| Should all users see updates in real-time? | Global |
| Is this UI-only state (open/closed, hover)? | Component |
| Is this user-specific data (cart, preferences)? | Session |
| Is this collaborative data (cursors, chat)? | Global |

### 9.3 Store Patterns for Large Apps

For production apps, organize state into stores by domain:

```go
// store/projects.go
package store

import (
    "github.com/vango-go/vango"
    "myapp/internal/services"
)

// ============ State ============

// Current project being viewed (session-scoped)
var CurrentProject = vango.NewSharedSignal[*Project](nil)

// Filter state (session-scoped)
var ProjectFilter = vango.NewSharedSignal(ProjectFilterState{
    Status:  "all",
    SortBy:  "updated_at",
    SortDir: "desc",
})

// ============ Derived State (Selectors) ============

var CurrentProjectID = vango.NewSharedMemo(func() int {
    if p := CurrentProject.Get(); p != nil {
        return p.ID
    }
    return 0
})

var IsProjectSelected = vango.NewSharedMemo(func() bool {
    return CurrentProject.Get() != nil
})

// ============ Actions ============

func SelectProject(id int) {
    ctx := vango.UseCtx()
    
    project := vango.NewResource(func() (*Project, error) {
        return services.Projects.GetByID(ctx.StdContext(), id)
    })
    
    // When resource completes, update shared state
    vango.Effect(func() vango.Cleanup {
        if project.State() == vango.Ready {
            CurrentProject.Set(project.Data())
        }
        return nil
    })
}

func SetFilter(filter ProjectFilterState) {
    ProjectFilter.Set(filter)
}
```

**Store Module Pattern:**

```
store/
├── auth.go           # CurrentUser, IsAuthenticated, Login(), Logout()
├── projects.go       # CurrentProject, ProjectFilter, SelectProject()
├── tasks.go          # TasksByProject, CreateTask(), ToggleTask()
├── notifications.go  # Toasts, ShowToast(), DismissToast()
└── preferences.go    # Theme, Locale, SetTheme(), SetLocale()
```

### 9.4 Derived State with Memos

Memos compute derived state efficiently:

```go
// Simple derived value
var filteredTasks = vango.NewMemo(func() []Task {
    filter := taskFilter.Get()
    tasks := allTasks.Get()
    
    result := make([]Task, 0)
    for _, t := range tasks {
        if filter.Status == "all" || t.Status == filter.Status {
            result = append(result, t)
        }
    }
    return result
})

// Layered selectors (selector → selector)
var tasksByStatus = vango.NewMemo(func() map[string][]Task {
    grouped := make(map[string][]Task)
    for _, t := range filteredTasks.Get() {  // Depends on filteredTasks
        grouped[t.Status] = append(grouped[t.Status], t)
    }
    return grouped
})

var taskCounts = vango.NewMemo(func() TaskCounts {
    byStatus := tasksByStatus.Get()  // Depends on tasksByStatus
    return TaskCounts{
        Todo:       len(byStatus["todo"]),
        InProgress: len(byStatus["in_progress"]),
        Done:       len(byStatus["done"]),
    }
})
```

**Memo Best Practices:**

```go
// ✅ Good: Memo for expensive computation
var expensiveResult = vango.NewMemo(func() []ProcessedItem {
    items := rawItems.Get()
    return expensiveProcessing(items)  // Only runs when rawItems changes
})

// ✅ Good: Memo for accessing nested data
var currentUserName = vango.NewMemo(func() string {
    if user := currentUser.Get(); user != nil {
        return user.Name
    }
    return "Guest"
})

// ❌ Bad: Memo for trivial computation (overhead not worth it)
var isPositive = vango.NewMemo(func() bool {
    return count.Get() > 0  // Too simple, just inline this
})
```

### 9.5 Transactions and Batching

When updating multiple signals, use transactions to batch updates:

```go
// Without transaction: multiple rerenders
func badUpdate() {
    firstName.Set("John")   // Rerender 1
    lastName.Set("Doe")     // Rerender 2
    email.Set("j@d.com")    // Rerender 3
}

// With transaction: single rerender
func goodUpdate() {
    vango.TxNamed("UpdateUserProfile", func() {
        firstName.Set("John")
        lastName.Set("Doe")
        email.Set("j@d.com")
    })  // Single rerender after all updates
}
```

**Named Transactions:**

Transaction names appear in DevTools and logs:

```go
// Name describes the intent
vango.TxNamed("AddItemToCart", func() {
    cart.Set(append(cart.Get(), item))
    cartCount.Inc()
})

vango.TxNamed("FilterProjectsByStatus", func() {
    filterStatus.Set(status)
    filterPage.Set(1)  // Reset page when filter changes
})

vango.TxNamed("ToggleTaskComplete", func() {
    task := tasks.Get()[index]
    task.Done = !task.Done
    tasks.SetAt(index, task)
})
```

**Automatic Naming:**

Event handlers and effects are automatically named:

```go
// DevTools shows: "onClick@TaskRow:23" (component:line)
Button(OnClick(func() {
    task.Done.Toggle()
}))
```

### 9.6 URL State with URLParam

Use `URLParam` for state that should be shareable and back-button friendly:

For query-state debouncing, use `vango.URLDebounce(d)` to coalesce URL updates. For debouncing event handlers, use the event modifier `vango.Debounce(d, handler)`.

```go
func ProjectList() vango.Component {
    return vango.Func(func() *vango.VNode {
        // URL: /projects?status=active&page=2&sort=name
        status := vango.URLParam("status", "all")
        page := vango.URLParam[int]("page", 1)
        sort := vango.URLParam("sort", "updated_at")
        
        // Setting updates URL without navigation
        return Div(
            // Filter buttons
            FilterButtons(status),
            
            // Sort dropdown
            Select(
                OnChange(func(value string) {
                    sort.Set(value)
                }),
                Option(Value("name"), Text("Name")),
                Option(Value("updated_at"), Text("Last Updated")),
            ),
            
            // Project list
            ProjectListView(status.Get(), page.Get(), sort.Get()),
            
            // Pagination
            Pagination(page, totalPages),
        )
    })
}
```

**When to Use URLParam vs Signal:**

| State | URLParam | Signal |
|-------|----------|--------|
| Filters | ✓ (shareable, back button) | |
| Current page | ✓ (shareable) | |
| Sort order | ✓ (shareable) | |
| Search query | ✓ (shareable) | |
| Modal open/closed | | ✓ (ephemeral) |
| Form input values | | ✓ (ephemeral) |
| Accordion expanded | | ✓ (ephemeral) |
| Hover/focus state | | ✓ (ephemeral) |

### 9.7 Session Durability and Persistence

Understand what state survives and when:

```
┌─────────────────────────────────────────────────────────────────┐
│                    STATE DURABILITY                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Ephemeral          Survives             Survives                │
│  (lost on refresh)  (ResumeWindow)       (persistent store)      │
│                                                                  │
│  Component signals  Shared signals       Shared signals with     │
│  Local UI state     Session state        persistent store        │
│                     within window        + server restarts       │
│                                                                  │
│       ← Less Durable                More Durable →              │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Configuring Durability:**

```go
// In-memory only (lost on server restart)
vango.SessionConfig{
    ResumeWindow: 30 * time.Second,
    // No Store configured
}

// Persistent (survives server restarts)
vango.SessionConfig{
    ResumeWindow: 30 * time.Second,
    Store: vango.RedisStore(redisClient),
}
```

**Stable Persistence Keys:**

When using persistent storage, signal names become storage keys:

```go
// ✅ Good: Stable key won't break existing sessions
var Cart = vango.NewSharedSignal([]CartItem{}, 
    vango.WithKey("user_cart"))

// ❌ Bad: Anonymous signal, key based on creation order
var Cart = vango.NewSharedSignal([]CartItem{})  // Key: "signal_5"
```

### 9.8 Immutable Data Patterns

Keep state updates safe with immutable patterns:

```go
// ❌ Bad: Mutating in place
func badToggle(index int) {
    tasks := taskList.Get()
    tasks[index].Done = !tasks[index].Done  // Mutating shared slice!
    taskList.Set(tasks)  // Same slice, may not trigger update
}

// ✅ Good: Copy-on-write
func goodToggle(index int) {
    tasks := taskList.Get()
    
    // Create new slice
    newTasks := make([]Task, len(tasks))
    copy(newTasks, tasks)
    
    // Create new task with updated field
    newTasks[index] = Task{
        ID:   tasks[index].ID,
        Name: tasks[index].Name,
        Done: !tasks[index].Done,
    }
    
    taskList.Set(newTasks)
}

// ✅ Better: Use signal helper methods
func bestToggle(index int) {
    taskList.UpdateAt(index, func(t Task) Task {
        t.Done = !t.Done
        return t
    })
}
```

**Slice Helpers:**

```go
// Append (creates new slice)
items.Set(append(items.Get(), newItem))

// Remove (creates new slice)
items.RemoveAt(index)
items.RemoveWhere(func(item Item) bool {
    return item.ID == targetID
})

// Update (creates new slice with modified element)
items.UpdateAt(index, func(item Item) Item {
    item.Count++
    return item
})

// Filter (creates new slice)
items.Filter(func(item Item) bool {
    return item.Active
})
```

**Map Helpers:**

```go
// Create map signal
users := vango.NewSignal(map[int]User{})

// Set entry (creates new map)
users.SetEntry(user.ID, user)

// Delete entry (creates new map)
users.DeleteEntry(userID)

// Update entry (creates new map)
users.UpdateEntry(userID, func(u User) User {
    u.LastSeen = time.Now()
    return u
})
```

### 9.9 Avoiding Accidental Dependencies

Be careful what signals your memos depend on:

```go
// ❌ Problem: Hidden dependency
var filteredItems = vango.NewMemo(func() []Item {
    items := allItems.Get()
    filter := getCurrentFilter()  // Calling function, not reading signal!
    // If getCurrentFilter() reads a signal internally,
    // this memo won't track it properly
    return filterItems(items, filter)
})

// ✅ Solution: Make dependencies explicit
var filteredItems = vango.NewMemo(func() []Item {
    items := allItems.Get()        // Tracked dependency
    filter := filterSignal.Get()   // Tracked dependency
    return filterItems(items, filter)
})
```

**Conditional Dependencies:**

```go
// ❌ Problem: Conditional signal read
var displayValue = vango.NewMemo(func() string {
    if showDetails.Get() {
        return detailedValue.Get()  // Only tracked when showDetails is true
    }
    return summary.Get()
})

// ✅ Solution: Read all dependencies unconditionally
var displayValue = vango.NewMemo(func() string {
    show := showDetails.Get()
    detailed := detailedValue.Get()
    sum := summary.Get()
    
    if show {
        return detailed
    }
    return sum
})
```

### 9.10 Debugging State

**Signal Inspection:**

```go
// Log signal value on change
vango.Effect(func() {
    fmt.Printf("Cart items: %+v\n", cart.Get())
})

// In development, signals have debug names
cart := vango.NewSignal([]Item{}, vango.WithDebugName("ShoppingCart"))
```

**DevTools Integration:**

In development mode, Vango DevTools shows:
- All active signals and their values
- Dependency graph between signals and memos
- Transaction history with names
- "Why did this render?" analysis

**Common Debugging Questions:**

| Question | How to Debug |
|----------|--------------|
| "Why isn't my UI updating?" | Check if signal write is inside handler or Dispatch |
| "Why is this re-rendering too much?" | Check memo dependencies, use transaction batching |
| "Why is state lost on navigation?" | Use SharedSignal instead of Signal |
| "Why did my selector fire?" | Check DevTools dependency graph |

### 9.11 State Management Anti-Patterns

**Understanding Hook Order and Signal Creation:**

The real rule is: **Don't create signals conditionally or with variable counts.** Creating signals inside `vango.Func` (and therefore inside page handlers) is correct and required—the prohibition is about conditional/variable hook order, not "signals in render."

```go
// ❌ WRONG: Creating signal outside vango.Func (no render tracking)
func BadComponent() *vango.VNode {
    count := vango.NewSignal(0)  // WRONG: not wrapped in vango.Func
    return Div(Text(count.Get()))
}

// ✅ CORRECT: Creating signal inside vango.Func
func GoodComponent() vango.Component {
    return vango.Func(func() *vango.VNode {
        count := vango.NewSignal(0)  // Created once, tracked by framework
        return Div(Text(count.Get()))
    })
}

// ✅ CORRECT: Creating signal inside a page handler (page handlers are reactive)
func IndexPage(ctx vango.Ctx) *vango.VNode {
    count := vango.NewSignal(0)  // This is correct—page handlers are wrapped in vango.Func
    return Div(Text(count.Get()))
}

// ❌ WRONG: Conditional signal creation (violates hook-order rule)
func BadConditional(showExtra bool) vango.Component {
    return vango.Func(func() *vango.VNode {
        count := vango.NewSignal(0)
        if showExtra {
            extra := vango.NewSignal("")  // WRONG: conditionally created!
            return Div(Text(count.Get()), Text(extra.Get()))
        }
        return Div(Text(count.Get()))
    })
}

// ✅ CORRECT: Always create the same hooks, conditionally use them
func GoodConditional(showExtra bool) vango.Component {
    return vango.Func(func() *vango.VNode {
        count := vango.NewSignal(0)
        extra := vango.NewSignal("")  // Always created
        if showExtra {
            return Div(Text(count.Get()), Text(extra.Get()))
        }
        return Div(Text(count.Get()))  // extra exists but not rendered
    })
}

// ❌ WRONG: Variable-count hooks in a loop
func BadLoop(items []string) vango.Component {
    return vango.Func(func() *vango.VNode {
        // WRONG: Number of signals varies with items length!
        for _, item := range items {
            signal := vango.NewSignal(item)  // Hook count changes!
            _ = signal
        }
        return Div()
    })
}

// ✅ CORRECT: Use a single signal containing the collection
func GoodLoop(items []string) vango.Component {
    return vango.Func(func() *vango.VNode {
        itemsSignal := vango.NewSignal(items)  // One hook, stable count
        return Ul(
            Range(itemsSignal.Get(), func(item string, i int) *vango.VNode {
                return Li(Key(i), Text(item))
            }),
        )
    })
}
```

**Don't: Read signals outside reactive context:**

```go
// ❌ This read won't trigger re-render
func handleClick() {
    value := mySignal.Get()  // Not in component render
    fmt.Println(value)       // OK for logging
    // Don't use this value to compute UI
}
```

**Don't: Forget Dispatch from goroutines:**

```go
// ❌ Will cause undefined behavior
go func() {
    result := fetchData()
    resultSignal.Set(result)  // WRONG: not on session loop
}()

// ✅ Use Dispatch with captured context
ctx := vango.UseCtx()
go func() {
    result := fetchData()
    ctx.Dispatch(func() {
        resultSignal.Set(result)  // Correct: on session loop
    })
}()
```

### 9.12 State Management Checklist

Before shipping, verify:

- [ ] **Component state** doesn't accidentally use shared signals
- [ ] **Session state** uses `NewSharedSignal` for cross-page persistence
- [ ] **Transactions** are named meaningfully for debugging
- [ ] **Memos** don't have hidden dependencies
- [ ] **Goroutines** use `ctx.Dispatch` for signal writes
- [ ] **Persistent state** has stable keys with `WithKey()`
- [ ] **Immutable patterns** are used for slice/map updates
- [ ] **URL state** uses `URLParam` where appropriate

## 10. Data Loading and Caching

This section explains how to load and cache data correctly in a server-driven app.

### 10.1 When to Load Data

> **IMPORTANT**: Page handlers MUST be render-pure (no blocking I/O). Page handlers execute during the render cycle—blocking database or HTTP calls will stall the session loop.

**Where to do blocking I/O:**
- **`Resource`** — Default for most data loading (non-blocking, shows loading states)
- **`Action`** — For user-initiated mutations
- **Route-level prefetch/navigation** — Advanced pattern for nav-blocking data (future feature)

| Pattern | Where | When | Use Case |
|---------|-------|------|----------|
| **Resource** | Inside page handler or component | During render | Default for data loading—shows loading/error/ready states |
| **Action** | Inside component | On user action | Mutations that return data |
| **Route prefetch** | Routing layer | Before navigation | Nav-blocking critical data (advanced) |

```go
// ❌ WRONG: Blocking I/O in page handler (violates render purity!)
func ShowPage(ctx vango.Ctx, p Params) *vango.VNode {
    // DON'T DO THIS—page handlers are in the reactive render path
    project, err := services.Projects.GetByID(ctx.StdContext(), p.ID)
    if err != nil {
        return ErrorPage(err)
    }
    return ProjectView(project)
}

// ✅ CORRECT: Use Resource for data loading (default pattern)
func ShowPage(ctx vango.Ctx, p Params) *vango.VNode {
    return ProjectPage(p.ID)
}
```

```go
func ProjectPage(id int) vango.Component {
    return vango.Func(func() *vango.VNode {
        ctx := vango.UseCtx()
        // Capture StdContext() for async work—this is safe for cancellation/I/O
        stdCtx := ctx.StdContext()

        project := vango.NewResource(func() (*Project, error) {
            // Safe: Using standard context for I/O cancellation
            // Never call ctx methods (Navigate/Dispatch/Session) from here
            return services.Projects.GetByID(stdCtx, id)
        })
        
        return project.Match(
            vango.OnLoading(ProjectSkeleton),
            vango.OnError(ErrorCard),
            vango.OnReady(ProjectView),
        )
    })
}
```

> **Context Safety in Async Work**: Use `ctx.StdContext()` for cancellation and I/O in Resource/Action loaders. Never call `vango.Ctx` methods (`Navigate`, `Dispatch`, session access, etc.) from async work—these are not thread-safe. Vango dispatches Resource results back onto the session loop automatically, so signal updates in `OnReady`/`OnError` handlers are safe.

**Resource is the default for data loading in the render path:**

| Question | Recommendation |
|----------|----------------|
| Want skeleton/loading UI? | Use `Resource` with `OnLoading` matcher |
| Multiple independent data sources? | Use multiple `Resource` instances |
| SEO-critical content? | Use `Resource`—SSR waits for initial data |
| Need data before anything renders? | Use `Resource` with loading state, or route-level prefetch (advanced) |
| Want data cached across re-renders? | Use keyed `Resource` |

> **Note**: Sometimes you want navigation to block until data is ready. This is valid—but that blocking I/O must not happen in the reactive render path. Use the appropriate structured primitive (`Resource`, `Action`) or route-level mechanisms for the kind of work you need.

### 10.2 Resource States

Resources have four states (use `State()` method or matchers):

```go
data := vango.NewResource(fetchData)

// State checks
data.State() == vango.Pending  // Not started
data.State() == vango.Loading  // In progress  
data.State() == vango.Ready    // Success
data.State() == vango.Error    // Failed

// Accessing values
data.Data()       // The loaded value (only valid when Ready)
data.Error()      // The error (only valid when Error)

// Pattern matching (recommended)
switch data.State() {
case vango.Loading:
    return Skeleton()
case vango.Error:
    return ErrorCard(data.Error())
case vango.Ready:
    projects := data.Data()
    if len(projects) == 0 {
        return EmptyState("No projects found")
    }
    return ProjectList(projects)
}
```

**Note:** The spec defines four states (`Pending`, `Loading`, `Ready`, `Error`). Empty-result
handling is application logic (check `len(data.Data()) == 0` when `Ready`).

### 10.3 Keyed Resources

When resource identity depends on a reactive parameter, use `NewResourceKeyed`. The key must come from a **reactive source** (URLParam, signal from parent, or derived memo) to trigger refetches when it changes.

**Pattern 1: Key from URLParam (most common)**

```go
// Route: /projects/:id
func ProjectPage(ctx vango.Ctx, p struct{ ID int }) *vango.VNode {
    // URLParam syncs with the route param and is reactive
    projectID := vango.URLParam[int]("id", p.ID)
    return ProjectView(projectID)
}

func ProjectView(projectID *vango.URLParam[int]) vango.Component {
    return vango.Func(func() *vango.VNode {
        ctx := vango.UseCtx()
        stdCtx := ctx.StdContext()

        // NewResourceKeyed re-fetches when URLParam changes
        project := vango.NewResourceKeyed(projectID, func(id int) (*Project, error) {
            return services.Projects.GetByID(stdCtx, id)
        })

        return project.Match(
            vango.OnLoading(ProjectSkeleton),
            vango.OnError(ErrorCard),
            vango.OnReady(ProjectCard),
        )
    })
}
```

**Pattern 2: Key from parent signal**

```go
// Parent component owns the selection state
func ProjectList(selectedID *vango.Signal[int]) vango.Component {
    return vango.Func(func() *vango.VNode {
        ctx := vango.UseCtx()
        stdCtx := ctx.StdContext()

        // Key is the signal itself—changes trigger refetch
        project := vango.NewResourceKeyed(selectedID, func(id int) (*Project, error) {
            if id == 0 {
                return nil, nil // No selection
            }
            return services.Projects.GetByID(stdCtx, id)
        })

        return project.Match(/* ... */)
    })
}
```

> **Important**: Do NOT use `vango.NewSignal(id)` inside a render function where `id` is a static prop. Each render creates a new signal with the same initial value—it won't react to changes. If the prop changes, pass it as a signal from the parent or derive it from URLParam.

**Key Behaviors:**

| Key Change | Behavior |
|------------|----------|
| Same key | Reuse cached result |
| Different key | Cancel stale work, start new fetch, transition to Loading |
| Unmount | Cancel any in-flight work |

### 10.4 Dependent Queries

Load data that depends on other data:

```go
func TeamProjectsPage(teamID int) vango.Component {
    return vango.Func(func() *vango.VNode {
        ctx := vango.UseCtx()
        
        // First query: load team
        team := vango.NewResource(func() (*Team, error) {
            return services.Teams.GetByID(ctx.StdContext(), teamID)
        })
        
        // Second query: depends on team state
        projects := vango.NewResource(func() ([]Project, error) {
            if team.State() != vango.Ready {
                return nil, nil // Still waiting for parent
            }
            return services.Projects.ListByTeam(ctx.StdContext(), team.Data().ID)
        })
        
        return Div(
            team.Match(
                vango.OnLoading(TeamHeaderSkeleton),
                vango.OnReady(TeamHeader),
            ),
            projects.Match(
                vango.OnLoading(ProjectListSkeleton),
                vango.OnReady(ProjectList),
            ),
        )
    })
}
```

### 10.5 Search-As-You-Type with GoLatest

For debounced search that cancels stale requests, use `GoLatest` inside an Effect:

```go
func SearchPage() vango.Component {
    return vango.Func(func() *vango.VNode {
        query := vango.NewSignal("")
        results := vango.NewSignal[[]SearchResult](nil)
        loading := vango.NewSignal(false)
        err := vango.NewSignal[error](nil)
        
        // GoLatest must be called inside an Effect
        vango.Effect(func() vango.Cleanup {
            q := query.Get()  // Reading query subscribes to changes
            
            if q == "" {
                results.Set(nil)
                return nil
            }
            
            // GoLatest: key coalescing + cancellation
            return vango.GoLatest(
                q,  // Key: same key = cancel previous
                func(ctx context.Context, key string) ([]SearchResult, error) {
                    // Debounce
                    select {
                    case <-time.After(300 * time.Millisecond):
                    case <-ctx.Done():
                        return nil, ctx.Err()
                    }
                    return services.Search.Query(ctx, key)
                },
                func(r []SearchResult, e error) {
                    // Apply callback runs on session loop
                    if e != nil {
                        err.Set(e)
                    } else {
                        results.Set(r)
                        err.Set(nil)
                    }
                    loading.Set(false)
                },
            )
        })
        
        return Div(
            Input(
                Type("search"),
                Placeholder("Search..."),
                Value(query.Get()),
                OnInput(func(value string) {
                    query.Set(value)
                    loading.Set(true)
                }),
            ),
            loading.Get() && SearchResultsSkeleton(),
            err.Get() != nil && ErrorMessage(err.Get().Error()),
            len(results.Get()) == 0 && !loading.Get() && Text("No results"),
            len(results.Get()) > 0 && SearchResultsList(results.Get()),
        )
    })
}
```

**Key GoLatest properties:**
- Returns `Cleanup` (call to cancel)
- Key-based coalescing: new call with same key cancels previous
- Work function receives `context.Context` for cancellation
- Apply callback runs on session loop (safe for signal writes)

### 10.6 Pagination Pattern

```go
func PaginatedList() vango.Component {
    return vango.Func(func() *vango.VNode {
        page := vango.URLParam[int]("page", 1)
        perPage := 20
        
        // Keyed resources re-load automatically when the key signal changes
        data := vango.NewResourceKeyed(page, func(p int) (*PaginatedResponse, error) {
            return services.Items.List(vango.UseCtx().StdContext(), p, perPage)
        })
        
        return Div(
            data.Match(
                vango.OnLoading(ListSkeleton),
                vango.OnReady(func(resp *PaginatedResponse) *vdom.VNode {
                    return Div(
                        ItemList(resp.Items),
                        Pagination(
                            page,
                            resp.TotalPages,
                            func(newPage int) { page.Set(newPage) },
                        ),
                    )
                }),
            ),
        )
    })
}
```

### 10.7 Caching Layers

Cache at the appropriate layer:

```
┌─────────────────────────────────────────────────────────────────┐
│                      CACHING LAYERS                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Browser Cache      →  Static assets (CSS, JS, images)           │
│  (HTTP headers)        Controlled by Cache-Control headers       │
│                                                                  │
│  Resource Cache     →  Per-component resource results            │
│  (keyed resources)     Automatic with same key                   │
│                                                                  │
│  Session Cache      →  Shared across pages for one user          │
│  (SharedSignal)        ShoppingCart, UserPreferences             │
│                                                                  │
│  Application Cache  →  Shared across all users                   │
│  (in-memory/Redis)     Config, static content, rate limits       │
│                                                                  │
│  Database Cache     →  Query result caching                      │
│  (ORM/query layer)     Frequently accessed data                  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Application-Level Caching:**

```go
// Simple in-memory cache with TTL
var configCache = cache.New(5 * time.Minute)

func getConfig(ctx context.Context) (*Config, error) {
    if cached, ok := configCache.Get("app_config"); ok {
        return cached.(*Config), nil
    }
    
    config, err := db.LoadConfig(ctx)
    if err != nil {
        return nil, err
    }
    
    configCache.Set("app_config", config)
    return config, nil
}
```

### 10.8 Preventing Stampedes

When many users request the same data simultaneously:

```go
import "golang.org/x/sync/singleflight"

var group singleflight.Group

func getPopularItem(ctx context.Context, id int) (*Item, error) {
    key := fmt.Sprintf("item:%d", id)
    
    result, err, _ := group.Do(key, func() (interface{}, error) {
        return db.Items.FindByID(ctx, id)
    })
    
    if err != nil {
        return nil, err
    }
    return result.(*Item), nil
}
```

### 10.9 Context and Cancellation

Always respect context cancellation:

```go
type Params struct {
    ID int `param:"id"`
}

func Page(ctx vango.Ctx, p Params) *vango.VNode {
    return ProjectPage(ctx.StdContext(), p.ID)
}

func ProjectPage(ctx context.Context, id int) vango.Component {
    return vango.Func(func() *vango.VNode {
        project := vango.NewResource(func() (*Project, error) {
            // Use the context for cancellation
            return services.Projects.GetByID(ctx, id)
        })
        // ...
    })
}

// In services
func (s *ProjectService) GetByID(ctx context.Context, id int) (*Project, error) {
    // Database query respects context
    return s.db.QueryRowContext(ctx, "SELECT ... WHERE id = $1", id).Scan(...)
}
```

### 10.10 Data Loading Checklist

- [ ] Essential data uses route handlers (blocks navigation)
- [ ] Secondary data uses Resources (shows loading states)
- [ ] Resources have appropriate keys for refetching
- [ ] All four states (loading, error, empty, ready) are handled
- [ ] Context is passed for cancellation
- [ ] Expensive queries use caching
- [ ] Hot paths use singleflight for stampede prevention

---

## 11. Mutations, Background Work, and Side Effects

This section covers changing data and handling asynchronous work.

### 11.1 Actions for Mutations

Use `Action` for user-initiated mutations. The work function receives a `context.Context`
for cancellation; it runs **off the session loop** (in a goroutine), and state transitions
are dispatched back automatically.

```go
func TaskRow(task Task) vango.Component {
    return vango.Func(func() *vango.VNode {
        // Task List signal
        localDone := vango.NewSignal(task.Done)

        // Action[A, R] takes work func(ctx context.Context, a A) (R, error)
        toggleComplete := vango.NewAction(
            func(ctx context.Context, done bool) (bool, error) {
                return done, services.Tasks.SetDone(ctx, task.ID, done)
            },
        )

        // Handle success to update local state
        vango.Effect(func() vango.Cleanup {
            if toggleComplete.State() == vango.ActionSuccess {
                if res, ok := toggleComplete.Result(); ok {
                    localDone.Set(res)
                }
            }
            return nil
        })
        
        return Tr(
            Td(
                Checkbox(
                    Checked(localDone.Get()),
                    OnChange(func() { toggleComplete.Run(!localDone.Get()) }),
                    Disabled(toggleComplete.State() == vango.ActionRunning),
                ),
            ),
            Td(Text(task.Title)),
            Td(
                // Show spinner while saving
                toggleComplete.State() == vango.ActionRunning && Spinner(),
            ),
        )
    })
}
```

**Note:** `Action[A, R]` is generic. For mutations that don't need arguments or results,
use `struct{}` as the type.

### 11.2 Action States

Actions expose their execution state via `State()`:

```go
action := vango.NewAction(doSomething)

// State returns ActionState enum
action.State()  // ActionIdle, ActionRunning, ActionSuccess, ActionError

// Check specific states
action.State() == vango.ActionIdle     // Not started
action.State() == vango.ActionRunning  // Currently running
action.State() == vango.ActionSuccess  // Last run succeeded
action.State() == vango.ActionError    // Last run failed

// Result and Error accessors
result, ok := action.Result()  // (R, true) if Success, (zero, false) otherwise
err := action.Error()          // Error if Error state, nil otherwise

// Run the action (returns true if accepted, false if rejected)
accepted := action.Run(arg)

// Reset clears state back to Idle
action.Reset()

// Pattern: Show different UI based on state
switch action.State() {
case vango.ActionRunning:
    return Button(Disabled(), Spinner(), Text("Saving..."))
case vango.ActionError:
    return Div(Button(OnClick(func() { action.Run(arg) }), Text("Retry")), ErrorText(action.Error()))
default:
    return Button(OnClick(func() { action.Run(arg) }), Text("Save"))
}
```

### 11.3 Concurrency Policies

Control what happens when an action is triggered while already running:

```go
// Default: CancelLatest - Cancel prior in-flight on new Run
vango.NewAction(search)

// Explicitly set CancelLatest (same as default)
vango.NewAction(search, vango.CancelLatest())

// Ignore new calls while running (good for saves)
vango.NewAction(save, vango.DropWhileRunning())

// Queue up to N calls, run sequentially
vango.NewAction(process, vango.Queue(10))
```

**Note:** Options are functions (with parentheses): `CancelLatest()`, `DropWhileRunning()`, `Queue(max)`.

**Policy Selection Guide:**

| Scenario | Policy |
|----------|--------|
| Search/filter | `CancelLatest` |
| Save button | `DropWhileRunning` |
| Add to queue | `Queue` |
| Navigation | `CancelLatest` |
| Delete | `DropWhileRunning` |

### 11.4 The Single-Writer Rule

All signal writes must happen on the session loop. Use `ctx.Dispatch` for goroutines:

```go
// ❌ WRONG: Writing from goroutine
func badExample() {
    go func() {
        result := expensiveFetch()
        resultSignal.Set(result)  // Race condition!
    }()
}

// ✅ CORRECT: Use Dispatch
func goodExample() {
    ctx := vango.UseCtx()
    go func() {
        result := expensiveFetch()
        ctx.Dispatch(func() {
            resultSignal.Set(result)  // Safe: runs on session loop
        })
    }()
}

// ✅ BETTER: Use Action (handles this automatically)
func bestExample() {
    action := vango.NewAction(func() error {
        result := expensiveFetch()
        resultSignal.Set(result)  // Action runs on session loop
        return nil
    })
}
```

### 11.5 Optimistic Updates

Update UI immediately, rollback on failure:

```go
func TaskItem(task Task) vango.Component {
    return vango.Func(func() *vango.VNode {
        ctx := vango.UseCtx()
        localDone := vango.NewSignal(task.Done)
        
        toggle := vango.NewAction(func(ctx context.Context, done bool) (bool, error) {
            // Server call
            err := services.Tasks.ToggleComplete(ctx, task.ID, done)
            return done, err
        })

        // Optimized state
        vango.Effect(func() vango.Cleanup {
            if toggle.State() == vango.ActionSuccess {
                if res, ok := toggle.Result(); ok {
                    localDone.Set(res)
                }
            }
            return nil
        })
        
        return Div(
            Checkbox(
                Checked(localDone.Get()),
                OnChange(func() { toggle.Run(!localDone.Get()) }),
                DataOptimistic(toggle.State() == vango.ActionRunning),  // Visual hint
            ),
            Text(task.Title),
        )
    })
}
```

### 11.6 Structured Effect Helpers

Use structured helpers instead of ad-hoc goroutines:

```go
// Interval: Runs periodically
vango.Interval(5*time.Second, func() {
    notifications := fetchNotifications()
    notificationCount.Set(len(notifications))
})

// Subscribe: External event source (returns Cleanup)
vango.Effect(func() vango.Cleanup {
    return vango.Subscribe(websocketMessages, func(msg Message) {
        messages.Set(append(messages.Get(), msg))
    })
})

// GoLatest: inside Effect, key-based coalescing
vango.Effect(func() vango.Cleanup {
    return vango.GoLatest(
        query.Get(),  // Key
        func(ctx context.Context, key string) ([]Item, error) {
            return search(ctx, key)
        },
        func(items []Item, err error) {
            if err != nil {
                searchError.Set(err)
            } else {
                results.Set(items)
            }
        },
    )
})
```

### 11.7 Effect Cleanup

Effects **return** cleanup functions (not using `vango.OnCleanup`):

```go
vango.Effect(func() vango.Cleanup {
    // Setup: subscribe to external source
    cancel := externalService.Subscribe(handleMessage)
    
    // Return cleanup function - called on unmount or re-run
    return cancel
})
```

**Note:** `vango.Cleanup` is `func()`. Return `nil` if no cleanup needed.
```

### 11.8 Background Jobs

For work that outlives a session:

```go
// Enqueue job (fast, returns immediately)
func Page(ctx vango.Ctx) *vango.VNode {
    enqueueExport := vango.NewAction(func() error {
        jobID, err := jobs.Enqueue("export", ExportParams{...})
        if err != nil {
            return err
        }
        pendingJobID.Set(jobID)
        return nil
    })
    
    return Button(OnClick(enqueueExport.Run), Text("Export Data"))
}

// Poll for job status
func JobProgress(jobID string) vango.Component {
    return vango.Func(func() *vango.VNode {
        status := vango.NewResource(func() (*JobStatus, error) {
            return jobs.GetStatus(vango.UseCtx().StdContext(), jobID)
        })
        
        // Poll every 2 seconds while pending
        vango.Effect(func() {
            if status.State() == vango.Ready && status.Data().State() == vango.ActionRunning {
                vango.Interval(2*time.Second, func() {
                    status.Refetch()
                })
            }
        })
        
        return status.Match(
            vango.OnLoading(Spinner),
            vango.OnReady(func(s *JobStatus) *vango.VNode {
                if s.IsComplete() {
                    return A(Href(s.DownloadURL), Text("Download"))
                }
                return ProgressBar(s.Progress)
            }),
        )
    })
}
```

### 11.9 External System Integration

Handle external systems with clear boundaries:

```go
// Payment processing with Action
func CheckoutButton(amount int) vango.Component {
    return vango.Func(func() *vango.VNode {
        ctx := vango.UseCtx()
        
        processPayment := vango.NewAction(func() error {
            // 1. Create local record
            orderID, err := services.Orders.Create(ctx.StdContext(), amount)
            if err != nil {
                return fmt.Errorf("failed to create order: %w", err)
            }
            
            // 2. Call payment provider
            paymentResult, err := payments.Charge(ctx.StdContext(), amount)
            if err != nil {
                // Mark order as failed
                services.Orders.MarkFailed(ctx.StdContext(), orderID, err.Error())
                return fmt.Errorf("payment failed: %w", err)
            }
            
            // 3. Update local record
            err = services.Orders.MarkPaid(ctx.StdContext(), orderID, paymentResult.TransactionID)
            if err != nil {
                // Log but don't fail - reconcile later
                log.Error("failed to mark order paid", "order", orderID, "error", err)
            }
            
            // 4. Navigate to success
            ctx.Navigate(fmt.Sprintf("/orders/%d/success", orderID))
            return nil
        }, vango.DropWhileRunning)
        
        return Button(
            OnClick(processPayment.Run),
            Disabled(processPayment.State() == vango.ActionRunning),
            processPayment.State() == vango.ActionRunning && Spinner(),
            Text("Pay Now"),
        )
    })
}
```

---

## 12. Forms, Validation, and UX

This section covers form handling patterns for production apps.

### 12.1 Basic Form with Signals

```go
func ContactForm() vango.Component {
    return vango.Func(func() *vango.VNode {
        ctx := vango.UseCtx()
        
        name := vango.NewSignal("")
        email := vango.NewSignal("")
        message := vango.NewSignal("")
        
        submit := vango.NewAction(func() error {
            return services.Contact.Submit(ctx.StdContext(), ContactInput{
                Name:    name.Get(),
                Email:   email.Get(),
                Message: message.Get(),
            })
        })
        
        return Form(
            OnSubmit(vango.PreventDefault(func() {
                submit.Run()
            })),
            
            Field("Name",
                Input(Type("text"), Value(name.Get()), 
                    OnInput(name.Set)),
            ),
            Field("Email",
                Input(Type("email"), Value(email.Get()), 
                    OnInput(email.Set)),
            ),
            Field("Message",
                Textarea(Value(message.Get()), 
                    OnInput(message.Set)),
            ),
            
            Button(
                Type("submit"),
                Disabled(submit.State() == vango.ActionRunning),
                submit.State() == vango.ActionRunning && Text("Sending..."),
                !submit.State() == vango.ActionRunning && Text("Send"),
            ),
            
            submit.Error() != nil && ErrorMessage(submit.Error()),
        )
    })
}
```

### 12.2 UseForm Helper

For complex forms, use the `UseForm` helper:

```go
func ProjectForm(project *Project) vango.Component {
    return vango.Func(func() *vango.VNode {
        ctx := vango.UseCtx()
        
        form := vango.UseForm(ProjectInput{
            Name:        project.Name,
            Description: project.Description,
            Status:      project.Status,
        })
        
        save := vango.NewAction(func() error {
            if !form.Validate() {
                return nil  // Validation errors shown
            }
            return services.Projects.Update(ctx.StdContext(), project.ID, form.Values())
        })
        
        return Form(
            OnSubmit(form.HandleSubmit(save.Run)),
            
            form.Field("Name", Input(
                Type("text"),
                Value(form.Get("Name")),
                OnInput(form.Set("Name")),
            )),
            form.FieldError("Name"),
            
            form.Field("Description", Textarea(
                Value(form.Get("Description")),
                OnInput(form.Set("Description")),
            )),
            
            form.Field("Status", Select(
                Value(form.Get("Status")),
                OnChange(form.Set("Status")),
                Option(Value("active"), Text("Active")),
                Option(Value("archived"), Text("Archived")),
            )),
            
            FormActions(
                Button(Type("button"), OnClick(form.Reset), Text("Reset")),
                Button(Type("submit"), Disabled(save.State() == vango.ActionRunning || !form.IsDirty()),
                    Text("Save")),
            ),
        )
    })
}
```

### 12.3 Validation Strategy

Server-side validation is the source of truth:

```go
type ProjectInput struct {
    Name        string `validate:"required,min=1,max=100"`
    Description string `validate:"max=1000"`
    Status      string `validate:"required,oneof=active archived"`
}

func (p ProjectInput) Validate() []ValidationError {
    var errors []ValidationError
    
    if p.Name == "" {
        errors = append(errors, ValidationError{
            Field:   "Name",
            Message: "Name is required",
        })
    } else if len(p.Name) > 100 {
        errors = append(errors, ValidationError{
            Field:   "Name",
            Message: "Name must be 100 characters or less",
        })
    }
    
    if len(p.Description) > 1000 {
        errors = append(errors, ValidationError{
            Field:   "Description",
            Message: "Description must be 1000 characters or less",
        })
    }
    
    return errors
}
```

**Client hints for better UX:**

```go
func NameField(form *vango.Form) *vango.VNode {
    value := form.Get("Name")
    
    // Instant feedback (client-side hint)
    var hint string
    if len(value) > 100 {
        hint = fmt.Sprintf("%d/100 characters (too long)", len(value))
    } else if len(value) > 0 {
        hint = fmt.Sprintf("%d/100 characters", len(value))
    }
    
    return Div(
        Label(For("name"), Text("Name *")),
        Input(
            ID("name"),
            Type("text"),
            Value(value),
            OnInput(form.Set("Name")),
            MaxLength(100),  // Hard limit
            AriaDescribedby("name-hint"),
        ),
        Span(ID("name-hint"), Class("hint"), Text(hint)),
        form.FieldError("Name"),  // Server validation error
    )
}
```

### 12.4 Error Display and Accessibility

```go
func Field(label string, input *vango.VNode, err string) *vango.VNode {
    fieldID := slugify(label)
    errorID := fieldID + "-error"
    
    return Div(Class("field"),
        Label(For(fieldID), Text(label)),
        // Clone input to add accessibility attributes
        input.With(
            ID(fieldID),
            AriaInvalid(err != ""),
            AriaDescribedby(err != "" && errorID),
        ),
        err != "" && Span(
            ID(errorID),
            Role("alert"),
            Class("error"),
            Text(err),
        ),
    )
}

// Focus management on error
func FormWithFocus() vango.Component {
    return vango.Func(func() *vango.VNode {
        form := vango.UseForm(...)
        
        submit := vango.NewAction(func() error {
            if !form.Validate() {
                // Focus first error field
                vango.UseCtx().FocusElement(form.FirstErrorField())
                return nil
            }
            return doSubmit(form.Values())
        })
        
        return Form(...)
    })
}
```

### 12.5 Preventing Double Submits

```go
func SafeSubmitButton(action *vango.Action, label string) *vango.VNode {
    return Button(
        Type("submit"),
        Disabled(action.State() == vango.ActionRunning),  // Disable while pending
        Class("submit-button"),
        DataLoading(action.State() == vango.ActionRunning),
        
        action.State() == vango.ActionRunning && Span(
            Class("spinner"),
            AriaHidden(true),
        ),
        Span(
            Class(action.State() == vango.ActionRunning && "sr-only"),  // Screen reader only when loading
            Text(label),
        ),
    )
}

// CSS for visual feedback
// .submit-button[data-loading="true"] {
//     opacity: 0.7;
//     cursor: wait;
// }
```

### 12.6 File Uploads

Files use hybrid HTTP + WebSocket:

```go
func FileUploadForm() vango.Component {
    return vango.Func(func() *vango.VNode {
        ctx := vango.UseCtx()
        
        uploadProgress := vango.NewSignal(0)
        uploadedFile := vango.NewSignal[*UploadedFile](nil)
        
        return Div(
            // Standard file input with form action
            Form(
                Method("POST"),
                Action("/api/upload"),
                EncType("multipart/form-data"),
                DataVangoEnhance("upload"),  // Vango intercepts for progress
                
                OnUploadProgress(func(percent int) {
                    uploadProgress.Set(percent)
                }),
                OnUploadComplete(func(file *UploadedFile) {
                    uploadedFile.Set(file)
                }),
                
                Input(Type("file"), Name("file"), Accept("image/*")),
                Button(Type("submit"), Text("Upload")),
            ),
            
            // Progress bar
            uploadProgress.Get() > 0 && uploadProgress.Get() < 100 && (
                ProgressBar(uploadProgress.Get())
            ),
            
            // Show uploaded file
            uploadedFile.Get() != nil && (
                Img(Src(uploadedFile.Get().URL), Alt("Uploaded image"))
            ),
        )
    })
}
```

### 12.7 Array Inputs

Handle dynamic lists of fields:

```go
func TagsInput() vango.Component {
    return vango.Func(func() *vango.VNode {
        tags := vango.NewSignal([]string{""})
        
        addTag := func() {
            tags.Set(append(tags.Get(), ""))
        }
        
        removeTag := func(index int) {
            tags.RemoveAt(index)
        }
        
        updateTag := func(index int, value string) {
            tags.SetAt(index, value)
        }
        
        return Div(
            vango.ForEach(tags.Get(), func(tag string, i int) *vango.VNode {
                return Div(Key(i), Class("tag-input"),
                    Input(
                        Type("text"),
                        Value(tag),
                        OnInput(func(value string) {
                            updateTag(i, value)
                        }),
                        Placeholder("Enter tag"),
                    ),
                    Button(
                        Type("button"),
                        OnClick(func() { removeTag(i) }),
                        Text("×"),
                    ),
                )
            }),
            Button(Type("button"), OnClick(addTag), Text("+ Add Tag")),
        )
    })
}
```

### 12.8 Progressive Enhancement

Forms work without JavaScript:

```go
func EnhancedForm() *vango.VNode {
    return Form(
        Method("POST"),
        Action("/contact"),  // Works without JS
        DataVangoEnhance("form"),  // Enhanced when connected
        
        Input(Name("name"), Type("text")),
        Input(Name("email"), Type("email")),
        Textarea(Name("message")),
        
        Button(Type("submit"), Text("Send")),
    )
}
```

**How it works:**

1. Without WebSocket: Standard form POST to `/contact`
2. With WebSocket: Vango intercepts, sends via WS, updates UI

### 12.9 Toast/Flash Messages

Show feedback after actions:

```go
// store/notifications.go
var Toasts = vango.NewSharedSignal([]Toast{})

func ShowToast(typ, message string) {
    id := uuid.NewString()
    Toasts.Append(Toast{ID: id, Type: typ, Message: message})
    
    // Auto-dismiss
    go func() {
        time.Sleep(5 * time.Second)
        vango.UseCtx().Dispatch(func() {
            Toasts.RemoveWhere(func(t Toast) bool {
                return t.ID == id
            })
        })
    }()
}

// Usage in action
submit := vango.NewAction(func() error {
    err := services.Contact.Submit(...)
    if err != nil {
        store.ShowToast("error", "Failed to send message")
        return err
    }
    store.ShowToast("success", "Message sent!")
    return nil
})

// Toast container in layout
func ToastContainer() vango.Component {
    return vango.Func(func() *vango.VNode {
        return Div(Class("toast-container"),
            vango.ForEach(store.Toasts.Get(), func(toast Toast, i int) *vango.VNode {
                return Div(Key(toast.ID), Class("toast", "toast-"+toast.Type),
                    Text(toast.Message),
                    Button(OnClick(func() {
                        store.Toasts.RemoveAt(i)
                    }), Text("×")),
                )
            }),
        )
    })
}
```

### 12.10 Form Checklist

- [ ] Server-side validation is the source of truth
- [ ] All fields have labels (accessibility)
- [ ] Error messages are associated with fields (aria-describedby)
- [ ] Focus is managed on error
- [ ] Submit button is disabled while pending
- [ ] Double-submit is prevented
- [ ] Forms work without JavaScript (progressive enhancement)
- [ ] Success/error feedback is shown (toasts or inline)

## 13. Navigation, URL State, and Progressive Enhancement

This section covers navigation patterns and building apps that work for everyone.

### 13.1 Link Types

Vango provides link helpers for SPA navigation:

```go
// Native navigation (no interception) - use raw A element
A(Href("/about"), Text("About"))

// SPA navigation (intercepted when WS healthy) - use Link helper
Link("/about", Text("About"))

// SPA navigation with hover prefetch
LinkPrefetch("/about", Text("About"))

// Navigation menu with active state
Nav(
    NavLink(ctx, "/", Text("Home")),       // Adds "active" class when on /
    NavLink(ctx, "/about", Text("About")), // Adds "active" class when on /about
)

// For sub-route matching (e.g., admin section)
NavLinkPrefix(ctx, "/admin", Text("Admin"))  // Active on /admin, /admin/users, etc.

// HTML <link> element (stylesheets, icons) - not navigation
LinkEl(Rel("stylesheet"), Href("/styles.css"))
```

**Note:** `Link`, `LinkPrefetch`, `NavLink`, `NavLinkPrefix` are helpers in `vango/el/helpers.go`.
They add `data-vango-link` automatically for interception.

### 13.2 Programmatic Navigation

Navigate from event handlers:

```go
// Basic navigation (push - adds history entry)
Button(OnClick(func() {
    ctx := vango.UseCtx()
    ctx.Navigate("/projects/123")
}), Text("View"))

// Replace (no history entry) - use server.WithReplace()
ctx.Navigate("/dashboard", server.WithReplace())

// With query parameters
ctx.Navigate("/search?q=golang&page=1")

// Reload current page
ctx.Navigate(ctx.Path(), server.WithReplace())
```

**Note:** External navigation to other origins causes a full page reload.
Use standard `A(Href("https://external.com"), Text("Link"))` for external links.

### 13.3 Query-Only Updates with URLParam

Update URL without navigation:

```go
func FilteredList() vango.Component {
    return vango.Func(func() *vango.VNode {
        // These sync with URL: /items?status=active&sort=name
        status := vango.URLParam("status", "all")
        sort := vango.URLParam("sort", "created_at")
        
        return Div(
            // Filter tabs
            Div(Class("tabs"),
                Tab(status, "all", "All"),
                Tab(status, "active", "Active"),
                Tab(status, "archived", "Archived"),
            ),
            
            // Sort dropdown
            Select(
                Value(sort.Get()),
                OnChange(func(value string) {
                    sort.Set(value)  // Updates URL, no navigation
                }),
                Option(Value("created_at"), Text("Newest")),
                Option(Value("name"), Text("Name")),
            ),
            
            // List (re-renders when URL params change)
            ItemList(status.Get(), sort.Get()),
        )
    })
}

func Tab(param *vango.URLParam, value, label string) *vango.VNode {
    isActive := param.Get() == value
    return Button(
        Class("tab", isActive && "active"),
        OnClick(func() { param.Set(value) }),
        Text(label),
    )
}
```

**URL State Benefits:**

- **Shareable**: Copy URL shares exact view state
- **Back button**: Browser history works naturally
- **Bookmarkable**: Save filter combinations
- **Refresh-safe**: State survives refresh

### 13.4 Navigation Events

Handle navigation lifecycle:

```go
// Before navigation (can cancel)
vango.OnBeforeNavigate(func(to string) bool {
    if form.IsDirty() {
        return confirm("Discard changes?")
    }
    return true  // Allow navigation
})

// After navigation
vango.OnNavigate(func(path string) {
    analytics.TrackPageView(path)
})
```

### 13.5 Self-Heal and Reconnect

When connection is lost or patches fail:

```
┌─────────────────────────────────────────────────────────────────┐
│                    CONNECTION LIFECYCLE                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. Connected      →  Normal operation                          │
│                                                                  │
│  2. Disconnected   →  In ResumeWindow: attempt reconnect         │
│                       Thin client shows reconnecting indicator   │
│                                                                  │
│  3. Reconnected    →  Session state preserved                    │
│                       Continue where left off                    │
│                                                                  │
│  4. Resume Failed  →  Full page reload                          │
│     (after window)   Fresh session, content from SSR             │
│                                                                  │
│  5. Patch Mismatch →  Full page reload                          │
│                       Self-heal: DOM re-syncs from server        │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**User Experience During Disconnection:**

```go
// Layout shows connection status
func Layout(ctx vango.Ctx, children *vango.VNode) *vango.VNode {
    return Html(
        Body(
            ConnectionStatus(),  // Built-in or custom indicator
            children,
        ),
    )
}

// Custom connection status
func ConnectionStatus() vango.Component {
    return vango.Func(func() *vango.VNode {
        status := vango.ConnectionStatus()  // "connected" | "reconnecting" | "disconnected"
        
        if status == "connected" {
            return nil  // No indicator when connected
        }
        
        return Div(Class("connection-banner", "banner-"+status),
            status == "reconnecting" && Text("Reconnecting..."),
            status == "disconnected" && Text("Connection lost. Refresh to continue."),
        )
    })
}
```

### 13.6 Progressive Enhancement

Links and forms work without JavaScript:

```go
// Links: Work as standard <a> tags
A(Href("/projects"), Text("Projects"))
// Without WS: Full page navigation
// With WS: SPA navigation

// Forms: Work as standard POST
Form(
    Method("POST"),
    Action("/contact"),
    DataVangoEnhance("form"),
    
    Input(Name("email"), Type("email")),
    Button(Type("submit"), Text("Send")),
)
// Without WS: Standard form POST
// With WS: Intercepted, sent via WebSocket
```

**Why Progressive Enhancement Matters:**

- Initial load works before WebSocket connects
- Works in no-JS environments (rare but possible)
- SEO: Search engines see functional pages
- Accessibility: Screen readers get HTML forms/links

### 13.7 Prefetching

Prefetch pages before navigation:

```go
// Prefetch on hover
LinkPrefetch("/projects", Text("Projects"))

// Prefetch when in viewport (handled by intersection observer)
LinkPrefetch("/projects/1", Text("Project 1"))

// Manual prefetch
Button(OnMouseEnter(func() {
    // Framework handles prefetch via LinkPrefetch or internal mechanisms
}), OnClick(func() {
    ctx.Navigate("/projects/123")  // Instant if prefetched
}), Text("Hover to Prefetch"))
```

**Prefetch Safety Rules:**

| Rule | Rationale |
|------|-----------|
| Only GET routes | Prefetch must not cause side effects |
| Rate limited | Max N prefetches per second |
| Timeout after 5s | Don't hold stale prefetch data |
| Cancel on navigate | Don't waste work |

### 13.8 Navigation Checklist

- [ ] Internal links use SPA navigation by default
- [ ] External links navigate away properly
- [ ] Filters/sorts use URLParam for shareability
- [ ] Unsaved changes warn before navigation
- [ ] Connection status is visible when disconnected
- [ ] Forms work as standard POST without WebSocket
- [ ] Critical pages work in SSR-only mode

---

## 14. Styling and Design Systems

This section covers styling patterns for Vango apps.

### 14.1 Tailwind CSS (Default)

The default template uses Tailwind CSS:

```go
// Inline classes
Button(
    Class("px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"),
    Text("Click me"),
)

// Dynamic classes based on state
Button(
    Class(
        "px-4 py-2 rounded transition-colors",
        isActive && "bg-blue-500 text-white",
        !isActive && "bg-gray-200 text-gray-700",
    ),
    Text("Toggle"),
)
```

**Tailwind Configuration:**

```javascript
// tailwind.config.js
module.exports = {
    content: ['./app/**/*.go'],  // Scan Go files
    theme: {
        extend: {
            colors: {
                primary: {
                    50: '#eff6ff',
                    500: '#3b82f6',
                    600: '#2563eb',
                    700: '#1d4ed8',
                },
            },
        },
    },
}
```

### 14.2 Pure CSS Workflow

If you prefer no Node.js/Tailwind:

```css
/* public/css/app.css */

/* Design tokens */
:root {
    --color-primary: #3b82f6;
    --color-primary-hover: #2563eb;
    --color-text: #1f2937;
    --color-text-muted: #6b7280;
    --color-bg: #ffffff;
    --color-bg-alt: #f3f4f6;
    
    --spacing-1: 0.25rem;
    --spacing-2: 0.5rem;
    --spacing-4: 1rem;
    --spacing-8: 2rem;
    
    --radius-sm: 0.25rem;
    --radius-md: 0.375rem;
    --radius-lg: 0.5rem;
}

/* Component classes */
.btn {
    padding: var(--spacing-2) var(--spacing-4);
    border-radius: var(--radius-md);
    font-weight: 500;
    cursor: pointer;
    transition: background-color 0.15s;
}

.btn-primary {
    background: var(--color-primary);
    color: white;
}

.btn-primary:hover {
    background: var(--color-primary-hover);
}
```

```go
// Use classes in Go
Button(Class("btn btn-primary"), Text("Click me"))
```

### 14.3 Dark Mode

Support system preference and manual toggle.

Vango apps frequently use **class-based dark mode** (e.g. Tailwind’s `dark:` variant), where the `dark` class is applied to the `<html>` element.

```css
/* CSS approach (class-based) */
:root {
    --color-bg: #ffffff;
    --color-text: #1f2937;
}

/* Dark mode override (enabled by `html.dark`) */
html.dark {
    --color-bg: #111827;
    --color-text: #f9fafb;
}
```

```go
// app/routes/layout.go
//
// Theme initialization: run BEFORE paint to avoid a flash of incorrect theme.
// This is safe to keep as an inline script because it doesn't rely on wiring
// DOM event listeners (which won't re-run under SPA patching).
func Layout(ctx vango.Ctx, children vango.Slot) *vango.VNode {
    return Html(
        Head(
            Script(Raw(`(function(){var t=localStorage.getItem('theme');if(t==='dark'||(!t&&window.matchMedia('(prefers-color-scheme:dark)').matches)){document.documentElement.classList.add('dark')}})();`)),
        ),
        Body(children),
    )
}
```

```go
// app/components/shared/navbar.go
//
// Theme toggle: use a client hook so it survives SPA navigation and DOM patching.
// The built-in hook persists to localStorage and toggles the `dark` class on <html>.
import "github.com/vango-go/vango/pkg/features/hooks"

func ThemeToggleButton() *vango.VNode {
    return Button(
        hooks.Hook("ThemeToggle", map[string]any{"storageKey": "theme"}),
        AriaLabel("Toggle theme"),
        Text("Toggle theme"),
    )
}
```

### 14.4 Dynamic Styling

Apply styles based on reactive state:

```go
// Class-based (preferred)
Div(
    Class(
        "card",
        isSelected && "card-selected",
        isDisabled && "card-disabled",
    ),
)

// Style attribute (for truly dynamic values)
Div(
    Style(fmt.Sprintf("--progress: %d%%", progress)),
    Class("progress-bar"),
)

// With CSS
// .progress-bar::after {
//     width: var(--progress);
// }
```

### 14.5 Component Styling Patterns

Encapsulate styles in components:

```go
// Base + variant pattern
type ButtonVariant string
const (
    ButtonPrimary   ButtonVariant = "primary"
    ButtonSecondary ButtonVariant = "secondary"
    ButtonDanger    ButtonVariant = "danger"
)

type ButtonSize string
const (
    ButtonSm ButtonSize = "sm"
    ButtonMd ButtonSize = "md"
    ButtonLg ButtonSize = "lg"
)

func Button(variant ButtonVariant, size ButtonSize, children ...any) func(...any) *vango.VNode {
    sizeClasses := map[ButtonSize]string{
        ButtonSm: "px-2 py-1 text-sm",
        ButtonMd: "px-4 py-2",
        ButtonLg: "px-6 py-3 text-lg",
    }
    
    variantClasses := map[ButtonVariant]string{
        ButtonPrimary:   "bg-blue-500 text-white hover:bg-blue-600",
        ButtonSecondary: "bg-gray-200 text-gray-800 hover:bg-gray-300",
        ButtonDanger:    "bg-red-500 text-white hover:bg-red-600",
    }
    
    return func(attrs ...any) *vango.VNode {
        return ButtonEl(
            Class("rounded font-medium transition-colors", 
                sizeClasses[size], 
                variantClasses[variant]),
            Group(attrs...),
            Group(children...),
        )
    }
}

// Usage
Button(ButtonPrimary, ButtonMd, Text("Save"))(OnClick(save))
```

### 14.6 Accessibility Checklist

Ensure your design system is accessible:

| Requirement | Implementation |
|-------------|----------------|
| Focus visible | `focus:ring-2 focus:ring-blue-500 focus:outline-none` |
| Color contrast | 4.5:1 for normal text, 3:1 for large text |
| Reduced motion | `motion-reduce:transition-none` |
| Touch targets | Min 44x44px for mobile |
| Focus order | Logical tab order |

```css
/* Reduced motion support */
@media (prefers-reduced-motion: reduce) {
    *, *::before, *::after {
        animation-duration: 0.01ms !important;
        transition-duration: 0.01ms !important;
    }
}
```

---

## 15. Static Assets, Bundles, and the Thin Client

This section covers asset management in Vango apps.

### 15.1 Static File Serving

Files in `public/` are served directly:

```
public/
├── css/
│   └── app.css           →  /css/app.css
├── js/
│   └── hooks/
│       └── sortable.js   →  /js/hooks/sortable.js
├── images/
│   └── logo.svg          →  /images/logo.svg
└── favicon.ico           →  /favicon.ico
```

**Configuration:**

```go
vango.New(vango.Config{
    Static: vango.StaticConfig{
        Dir:    "public",    // Directory to serve
        Prefix: "/",         // URL prefix
    },
})
```

**Security note:** Vango rejects static paths that attempt to escape the static root (e.g., `..` segments or absolute paths) and does not serve directories (no directory listings).

### 15.2 The Thin Client

Vango provides a thin JavaScript client (~12KB) that handles the browser-side of the server-driven architecture.

**Script Injection:**

You should include `VangoScripts()` explicitly in your layout (recommended), and the framework also injects the script tag when rendering full HTML pages via SSR. Both approaches work:

```go
// Explicit (recommended for clarity)
func Layout(ctx vango.Ctx, children vango.Slot) *vango.VNode {
    return Html(
        Head(/* ... */),
        Body(
            children,
            VangoScripts(),  // Explicitly include thin client
        ),
    )
}

// Implicit (framework injects if VangoScripts() not present)
func Layout(ctx vango.Ctx, children vango.Slot) *vango.VNode {
    return Html(
        Head(/* ... */),
        Body(children),
        // VangoScripts() auto-injected before </body>
    )
}
```

**What the thin client does:**

1. Establishes WebSocket connection to `/_vango/live?path=<current-path>`
2. Captures DOM events, sends to server via binary protocol
3. Receives patches, applies to DOM
4. Handles reconnection and self-heal
5. Progressive enhancement for forms/links

**WebSocket URL and Path Parameter:**

The thin client connects to `/_vango/live?path=<pathname+search>` where `path` contains the current browser URL path and query string. This allows the server to immediately mount the correct route and render the component tree when the WebSocket connects—ensuring event handlers exist and patches can be sent.

> **Implementation Detail**: The `?path=` query parameter is required for immediate interactivity after SSR. If `?path=` is absent or invalid, Vango defaults to `/` or triggers a hard-reload per self-heal rules.
>
> **Custom WS URL overrides** (via `data-ws-url` or similar) **must preserve** the `?path=` parameter unless you provide an equivalent initial-route mechanism. Without it, the server won't know which route to mount, resulting in "handler not found" errors.

**Client Bundles:**

The `client/dist/` directory contains the compiled thin client:
- `vango.js` — Development build (readable, with source maps)
- `vango.min.js` — Production build (minified)

> **Warning**: These files are **build output** and must not be edited manually. Both bundles must be generated from the same `client/src/` source and should behave identically aside from minification. If you see behavioral differences between dev and prod, rebuild both from the same source.

### 15.3 Cache Headers

Production cache strategy:

| Asset Type | Cache-Control | Reason |
|------------|---------------|--------|
| HTML pages | `no-cache` | Must always be fresh |
| Thin client | `max-age=31536000, immutable` | Versioned in URL |
| CSS/JS (hashed) | `max-age=31536000, immutable` | Content-addressed |
| CSS/JS (unhashed) | `max-age=3600, must-revalidate` | Shorter cache |
| Images | `max-age=86400` | Medium cache |

**Configuration:**

```go
vango.StaticConfig{
    CacheControl: vango.CacheControlProduction,
    // Adds appropriate headers based on file type and hash presence
}
```

### 15.4 Asset Fingerprinting

For long-term caching, use hashed filenames. Vango provides a context-aware asset resolver that maps source paths to fingerprinted paths via a manifest.

```go
// Layout example
func Layout(ctx vango.Ctx, children vango.Slot) *vango.VNode {
    return Html(
        Head(
            // Use ctx.Asset() for automatic hashing
            LinkEl(Rel("stylesheet"), Href(ctx.Asset("styles.css"))),
        ),
        Body(
            Img(Src(ctx.Asset("images/logo.svg")), Alt("Logo")),
            children,
        ),
    )
}
```

**Production Configuration:**

In production, you load the `manifest.json` generated by `vango build` and pass it to the server.

```go
func main() {
    // Load manifest
    manifest, _ := vango.LoadAssetManifest("dist/manifest.json")
    
    // Create resolver
    // Prefix should match how your static files are served (often "/").
    resolver := vango.NewAssetResolver(manifest, "/")
    
    app := vango.New(vango.Config{
        AssetResolver: resolver,
        // ...
    })
}
```

**Development Configuration:**

In development, use a passthrough resolver that just adds the prefix without hashing.

```go
resolver := vango.NewPassthroughResolver("/")
```

**How it works:**
1. `ctx.Asset("styles.css")` is called.
2. The resolver looks up `"styles.css"` in the manifest.
3. If found, it returns something like `"/styles.a1b2c3d4.css"`.
4. If not found, it returns `"/styles.css"`.

### 15.5 Hook and Island Bundles

Client-side code for hooks and islands:

```
public/
├── js/
│   ├── hooks/                # Optional: custom hooks you add
│   │   └── custom_hook.js
│   └── islands/              # Optional: custom islands you add
│       ├── rich-editor.js
│       └── map.js
```

**Loading Strategy:**

```go
// Built-in hooks ship inside the Vango thin client.
// Hooks initialize when an element with `data-hook="..."` appears in the DOM.
Div(
    vango.Hook("Sortable", map[string]any{
        "handle":    ".drag-handle",
        "animation": 150,
    }),
    vango.OnEvent("reorder", func(e vango.HookEvent) {
        from := e.Int("fromIndex")
        to := e.Int("toIndex")
        db.Tasks.Reorder(from, to)
    }),
)
```

For custom hooks, load your JS and register it with the client before first use:

```js
import { MyHook } from '/js/hooks/custom_hook.js';
window.__vango__?.registerHook('MyHook', MyHook);
```

### 15.6 WASM Assets

WebAssembly islands have their own assets:

```
public/
└── wasm/
    ├── physics.wasm         # Compiled WASM module
    └── physics.js           # JS glue code
```

```go
// WASM island
WASMWidget("physics", map[string]any{
    "gravity":  9.8,
    "friction": 0.1,
})
```

### 15.7 Content Security Policy

Secure script loading:

```go
vango.SecurityConfig{
    CSP: vango.CSPConfig{
        DefaultSrc: []string{"'self'"},
        ScriptSrc:  []string{"'self'"},  // Only own scripts
        StyleSrc:   []string{"'self'", "'unsafe-inline'"},  // Tailwind needs inline
        ImgSrc:     []string{"'self'", "data:", "https:"},
        ConnectSrc: []string{"'self'", "wss:"},  // WebSocket
    },
}
```

### 15.8 Asset Checklist

- [ ] CSS is fingerprinted for cache-busting
- [ ] Static assets have appropriate cache headers
- [ ] Thin client is versioned (automatic)
- [ ] Hooks/islands are lazy-loaded
- [ ] CSP is configured appropriately
- [ ] Assets are compressed (gzip/brotli)

---

## 16. Client Hooks, JavaScript Islands, and WASM

This section covers client extensions for when you need direct browser control.

### 16.1 When to Use Client Extensions

The decision tree:

```
Need client-side behavior?
│
├─ No → Server-driven (default)
│
└─ Yes → What kind?
    │
    ├─ 60fps interaction (drag, resize) → Client Hook
    │
    ├─ Third-party JS library → JavaScript Island
    │
    └─ Compute-heavy client work → WASM Island
```

### 16.2 Client Hooks

Hooks run client-side JavaScript for specific DOM elements:

```go
// Sortable list with drag-and-drop
Ul(
    DataVangoHook("sortable", map[string]any{
        "handle":   ".drag-handle",
        "animation": 150,
    }),
    vango.ForEach(items, func(item Item, i int) *vango.VNode {
        return Li(Key(item.ID),
            Span(Class("drag-handle"), Text("⋮")),
            Text(item.Name),
        )
    }),
)
```

**Hook Definition (JavaScript):**

```javascript
// public/js/hooks/sortable.js
export default {
    mounted(el, config, send) {
        // Initialize Sortable
        this.sortable = new Sortable(el, {
            handle: config.handle,
            animation: config.animation,
            onEnd: (evt) => {
                // Send reorder event to server
                send('reorder', {
                    from: evt.oldIndex,
                    to: evt.newIndex,
                });
            },
        });
    },
    
    updated(el, config) {
        // Called when server updates the element
        // Usually no-op for sortable
    },
    
    destroyed() {
        // Cleanup
        this.sortable.destroy();
    },
};
```

**Handle Hook Events Server-Side:**

```go
func TaskList(tasks []Task) vango.Component {
    return vango.Func(func() *vango.VNode {
        taskList := vango.NewSignal(tasks)
        
        return Ul(
            // Hook handles all drag animation at 60fps
            Hook("Sortable", map[string]any{"handle": ".handle"}),
            
            // Only fires when drag completes
            OnEvent("reorder", func(e vango.HookEvent) {
                from := e.Int("fromIndex")
                to := e.Int("toIndex")
                
                // Reorder locally
                items := taskList.Get()
                item := items[from]
                items = append(items[:from], items[from+1:]...)
                items = append(items[:to], append([]Task{item}, items[to:]...)...)
                taskList.Set(items)
                
                // Persist to server
                services.Tasks.Reorder(from, to)
            }),
            
            vango.ForEach(taskList.Get(), renderTask),
        )
    })
}
```

### 16.3 Standard Hooks

Vango includes common hooks:

| Hook | Purpose |
|------|---------|
| `Sortable` | Drag-and-drop reordering |
| `Draggable` | Drag source behavior |
| `Droppable` | Drop target behavior |
| `Resizable` | Resize handles and resizing |
| `Tooltip` | Tooltips with positioning |
| `Dropdown` | Dropdown menus |
| `Collapsible` | Collapsible sections / accordions |
| `Dialog` | Dialog/modal behavior |
| `Popover` | Popovers anchored to elements |
| `Portal` | Render content into the portal root |
| `FocusTrap` | Trap focus within a subtree (e.g. modal) |
| `ThemeToggle` | App-shell theme toggle (localStorage + `html.dark`) |

```go
// Tooltip example
Button(
    vango.Hook("Tooltip", map[string]any{
        "content":   "Click to save",
        "placement": "top",
    }),
    Text("Save"),
)
```

### 16.4 JavaScript Islands

For third-party libraries that manage their own DOM (opaque subtree):

```go
// Rich text editor island
func RichEditor(content string, onChange func(string)) *vango.VNode {
    return Div(
        ID("editor"),
        JSIsland("rich-editor", map[string]any{
            "content":     content,
            "placeholder": "Write something...",
        }),
    )
}
```

**Island Definition:**

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

**Handle Island Messages:**

```go
vango.OnIslandMessage("rich-editor", func(event string, data map[string]any) {
    if event == "change" {
        content := data["content"].(string)
        contentSignal.Set(content)
    }
})
```

### 16.5 WASM Islands

For compute-heavy client-side work:

```go
// Physics simulation
Div(
    ID("physics-canvas"),
    WASMWidget("physics", map[string]any{
        "gravity":  9.8,
        "friction": 0.1,
    }),
)
```

**When to Use WASM:**

| Use Case | WASM Appropriate? |
|----------|-------------------|
| Image processing | Yes |
| Complex canvas graphics | Yes |
| Realtime physics | Yes |
| Form validation | No (server-side) |
| Simple animations | No (CSS/JS hook) |
| Data display | No (server-driven) |

### 16.6 Island Isolation

Islands are isolated from Vango's DOM management:

```
┌─────────────────────────────────────────────────────────────────┐
│                      VANGO-MANAGED DOM                          │
│                                                                  │
│  ┌──────────────────────┐    ┌──────────────────────┐          │
│  │   ISLAND BOUNDARY    │    │   ISLAND BOUNDARY    │          │
│  │                      │    │                      │          │
│  │   Managed by JS/WASM │    │   Managed by JS/WASM │          │
│  │   Vango won't patch  │    │   Vango won't patch  │          │
│  │                      │    │                      │          │
│  └──────────────────────┘    └──────────────────────┘          │
│                                                                  │
│  Rest of DOM managed by Vango patches                           │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 16.7 Security Considerations

Island messages require validation:

```go
vango.OnIslandMessage("editor", func(event string, data map[string]any) {
    // Validate event type
    if event != "change" && event != "save" {
        return  // Ignore unknown events
    }
    
    // Validate data
    content, ok := data["content"].(string)
    if !ok || len(content) > 100000 {
        return  // Invalid or too large
    }
    
    // Authorization check
    user := store.CurrentUser.Get()
    if user == nil || !user.CanEdit {
        return  // Not authorized
    }
    
    // Process
    contentSignal.Set(content)
})
```

### 16.8 Client Extension Checklist

- [ ] Only use hooks/islands when server-driven won't work
- [ ] Hook bundles are lazy-loaded
- [ ] Islands are properly isolated
- [ ] Island messages are validated server-side
- [ ] WASM only for compute-heavy client work
- [ ] Bundle sizes are budgeted and monitored

## 17. Authentication, Authorization, and Sessions

This section covers identity and access control in Vango apps.

### 17.1 Authentication Flow

Typical authentication in Vango:

```
┌─────────────────────────────────────────────────────────────────┐
│                    AUTHENTICATION FLOW                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. User visits page     →  SSR renders (user from HTTP ctx)    │
│                                                                  │
│  2. User clicks login    →  Redirect to /login or OAuth         │
│                                                                  │
│  3. Auth completes       →  Set HTTP-only cookie                │
│                             Redirect to app                      │
│                                                                  │
│  4. Page load (SSR)      →  HTTP middleware reads cookie        │
│                             Attaches user to request context     │
│                                                                  │
│  5. WebSocket upgrade    →  Cookie sent automatically           │
│                             Session inherits user context        │
│                                                                  │
│  6. Subsequent events    →  User available via ctx.User()       │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

External redirects (OAuth/OIDC providers) must be explicit and allowlisted:

```go
app := vango.New(vango.Config{
    Security: vango.SecurityConfig{
        AllowedRedirectHosts: []string{"accounts.example.com"},
    },
})

// Explicit external redirect (absolute URL).
ctx.RedirectExternal("https://accounts.example.com/login", http.StatusFound)
```

`ctx.Redirect` remains relative-only and rejects absolute URLs by default.

### 17.2 Context Bridge

Move identity from HTTP middleware to Vango session:

```go
// HTTP middleware extracts user from cookie
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        cookie, err := r.Cookie("session")
        if err == nil {
            user, err := auth.ValidateSession(r.Context(), cookie.Value)
            if err == nil {
                // Attach to request context (available to SSR via ctx.User()).
                r = r.WithContext(vango.WithUser(r.Context(), user))
            }
        }
        next.ServeHTTP(w, r)
    })
}

// WebSocket session bridge copies user from HTTP context to the Vango session.
vango.New(vango.Config{
    OnSessionStart: func(httpCtx context.Context, s *vango.Session) {
        if user, ok := vango.UserFromContext(httpCtx).(*User); ok && user != nil {
            auth.Set(s, user) // stores under auth.SessionKey (session-scoped)
        }
    },
    
    // Revalidate auth when session resumes after disconnect
    OnSessionResume: func(httpCtx context.Context, s *vango.Session) error {
        user, err := myauth.ValidateFromRequest(httpCtx)
        if err != nil {
            return err  // Rejects resume
        }
        auth.Set(s, user)  // Rehydrate user
        return nil
    },
})

// Access in components
func Page(ctx vango.Ctx) *vango.VNode {
    user := ctx.User().(*User)  // Type assertion
    if user == nil {
        ctx.Navigate("/login", vango.WithReplace())
        return nil
    }
    return Dashboard(user)
}
```

### 17.3 Type-Safe User Access

Built-in helpers:

```go
import "github.com/vango-go/vango/pkg/auth"

user, ok := auth.Get[*User](ctx)
if !ok {
    // guest
}

user, err := auth.Require[*User](ctx)
if err != nil {
    return err
}

if auth.IsAuthenticated(ctx) {
    // logged in
}
```

Define a typed helper:

```go
// store/auth.go
package store

import "github.com/vango-go/vango"

type User struct {
    ID       int
    Email    string
    Name     string
    Role     string
    TenantID int
}

func CurrentUser() *User {
    ctx := vango.UseCtx()
    if u := ctx.User(); u != nil {
        return u.(*User)
    }
    return nil
}

func IsAuthenticated() bool {
    return CurrentUser() != nil
}

func IsAdmin() bool {
    user := CurrentUser()
    return user != nil && user.Role == "admin"
}

func RequireAuth() bool {
    if !IsAuthenticated() {
        ctx := vango.UseCtx()
        ctx.Navigate("/login?redirect="+ctx.Path(), vango.WithReplace())
        return false
    }
    return true
}
```

### 17.4 Route Guards

Protect routes with middleware:

```go
// routes/admin/middleware.go
package admin

import (
    "github.com/vango-go/vango/pkg/authmw"
    "github.com/vango-go/vango/pkg/router"
    "myapp/app/store"
)

func Middleware() []router.Middleware {
    return []router.Middleware{
        authmw.RequireAuth,
        authmw.RequireRole(func(u *store.User) bool {
            return u.Role == "admin"
        }),
    }
}
```

Additional helpers:
- `authmw.RequirePermission`
- `authmw.RequireAny`
- `authmw.RequireAll`

Custom guards (advanced):

```go
// routes/admin/middleware.go
package admin

import (
    "github.com/vango-go/vango"
    "github.com/vango-go/vango/pkg/router"
    "myapp/app/store"
)

func Middleware() []router.Middleware {
    return []router.Middleware{
        router.MiddlewareFunc(func(ctx vango.Ctx, next func() error) error {
            user, _ := ctx.User().(*store.User)
            if user == nil {
                ctx.Navigate("/login?redirect="+ctx.Path(), vango.WithReplace())
                return nil // short-circuit
            }
            if user.Role != "admin" {
                ctx.Navigate("/forbidden", vango.WithReplace())
                return nil // short-circuit
            }
            return next()
        }),
    }
}
```

### 17.2.1 Auth Freshness (Passive + Active)

To enable auth freshness in long-lived WebSocket sessions, set a principal with an expiry and configure active checks:

```go
OnSessionStart: func(httpCtx context.Context, s *vango.Session) {
    if principal, ok := myProvider.Principal(httpCtx); ok {
        auth.SetPrincipal(s, principal)
    }
}

Session: vango.SessionConfig{
    AuthCheck: &vango.AuthCheckConfig{
        Interval: 2 * time.Minute,
        Check:    myProvider.Verify,
        OnExpired: vango.AuthExpiredConfig{
            Action: vango.ForceReload,
        },
    },
},
```

Runtime-only keys (`auth.SessionKeyPrincipal`, `auth.SessionKeyExpiryUnixMs`) are never persisted; always rehydrate them on start/resume.
`auth.SetPrincipal` also sets `auth.SessionKeyHadAuth` and the legacy presence marker used to enforce resume rehydration.
If `principal.ExpiresAtUnixMs` is zero, `auth.SetPrincipal` will not set the expiry key, so passive checks remain disabled unless you set it explicitly.

For high-value actions, use `ctx.RevalidateAuth()` to force an immediate active check (fail-closed).

Defaults when `AuthCheck` is set:
- `Timeout`: 5 seconds
- `FailureMode`: `FailOpenWithGrace` (bounded by `MaxStale`)
- `MaxStale`: 15 minutes

If you use custom session persistence, skip runtime-only keys explicitly:
`auth.RuntimeOnlySessionKeys` or `auth.SessionKeyPrincipal` + `auth.SessionKeyExpiryUnixMs`.

### 17.2.2 Auth Expiry UX (Multi-Tab + Polling Fallback)

When auth expires during a live WebSocket session, the server sends an auth control
command to the thin client. The client:
- broadcasts `{type, reason}` over `BroadcastChannel` (if available)
- falls back to `storage` events when `BroadcastChannel` is unavailable
- reloads or hard-navigates to re-enter the HTTP pipeline

If the WebSocket handshake is rejected as “not authorized”, the client clears its
resume info and hard reloads automatically to re-authenticate. Handshake errors
may include an auth-expired reason code (for resume rehydrate failures) for logging
and metrics.

Optional polling fallback (only if both BroadcastChannel and storage are unavailable):

```js
import VangoClient from '@vango/client';

const client = new VangoClient({
  authPollUrl: '/auth/status',      // 200 = ok, 401 = expired
  authPollInterval: 30000,
});
```

### 17.2.3 Provider Adapters

Session-first (recommended) lives in core:

```go
import "github.com/vango-go/vango/pkg/auth/sessionauth"

provider := sessionauth.New(store, sessionauth.WithCookieName("session"))
```

Provider modules:
- `github.com/vango-go/vango-clerk` (Go 1.24+)
- `github.com/vango-go/vango-auth0` (Go 1.22+)

Both modules implement `auth.Provider` and plug into the same `AuthCheck` flow.

### 17.5 Per-Action Authorization

Check permissions before mutations:

```go
func DeleteProject(projectID int) vango.Component {
    return vango.Func(func() *vango.VNode {
        ctx := vango.UseCtx()
        user := store.CurrentUser()
        
        deleteAction := vango.NewAction(func() error {
            // Authorization check
            project, err := services.Projects.GetByID(ctx.StdContext(), projectID)
            if err != nil {
                return err
            }
            if project.OwnerID != user.ID && user.Role != "admin" {
                return errors.New("not authorized to delete this project")
            }
            
            return services.Projects.Delete(ctx.StdContext(), projectID)
        })
        
        return Button(
            OnClick(deleteAction.Run),
            Disabled(deleteAction.State() == vango.ActionRunning),
            Text("Delete"),
        )
    })
}
```

### 17.6 Login Flow

```go
// routes/login.go
func Page(ctx vango.Ctx) *vango.VNode {
    return LoginPage()
}

func LoginPage() vango.Component {
    return vango.Func(func() *vango.VNode {
        ctx := vango.UseCtx()
        
        email := vango.NewSignal("")
        password := vango.NewSignal("")
        
        login := vango.NewAction(func() error {
            // Validate credentials against your auth service
            session, err := myauth.ValidateCredentials(ctx.StdContext(), email.Get(), password.Get())
            if err != nil {
                return err
            }
            
            // Store user in Vango session (sets presence flag for resume)
            auth.Login(ctx, session.User)
            
            // Set cookie for subsequent requests
            ctx.SetCookie(&http.Cookie{
                Name:     "session",
                Value:    session.Token,
                Path:     "/",
                HttpOnly: true,
                Secure:   true,
                SameSite: http.SameSiteLaxMode,
                MaxAge:   86400 * 7,  // 7 days
            })
            
            // Redirect to original destination or home
            redirect := ctx.Request().URL.Query().Get("redirect")
            if redirect == "" {
                redirect = "/"
            }
            ctx.Navigate(redirect, vango.Replace)
            return nil
        })
        
        return Form(
            OnSubmit(form.HandleSubmit(login.Run)),
            
            Input(Type("email"), Value(email.Get()), OnInput(email.Set)),
            Input(Type("password"), Value(password.Get()), OnInput(password.Set)),
            
            Button(Type("submit"), Disabled(login.State() == vango.ActionRunning), Text("Log In")),
            
            login.Error() != nil && ErrorMessage("Invalid email or password"),
        )
    })
}
```

Note: `ctx.SetCookie` applies `CookieSecure`, `CookieHttpOnly`, `CookieSameSite`, and
`CookieDomain` defaults from config. Use `ctx.SetCookieStrict` if you want to handle
insecure requests explicitly.

### 17.7 Logout

**Option A: Full page reload (clears cookies)**
```go
func LogoutButton() *vango.VNode {
    return A(
        Href("/logout"),
        DataVangoLink("external"),  // Full page reload
        Text("Log Out"),
    )
}

// routes/logout/route.go
func GET(w http.ResponseWriter, r *http.Request) {
    // Clear session cookie
    http.SetCookie(w, &http.Cookie{
        Name:     "session",
        Value:    "",
        Path:     "/",
        HttpOnly: true,
        Secure:   true,
        MaxAge:   -1,  // Delete
    })
    
    http.Redirect(w, r, "/", http.StatusFound)
}
```

**Option B: In-session logout (SPA-style)**
```go
func LogoutButton() vango.Component {
    return vango.Func(func() *vango.VNode {
        ctx := vango.UseCtx()
        
        handleLogout := func() {
            auth.Logout(ctx)  // Clears user from session
            ctx.Navigate("/")
        }
        
        return Button(OnClick(handleLogout), Text("Log Out"))
    })
}
```

### 17.8 Session Durability

Understand session persistence:

| Scenario | Behavior |
|----------|----------|
| Page refresh | Session preserved (within ResumeWindow, via thin-client resume using `sessionStorage`) |
| Tab close + reopen | Session lost (new session on connect) |
| Auth cookie expired | User becomes nil, redirect to login |
| Server restart | Session lost (unless using store) |

**How refresh resume works (in-memory):**

- The thin client stores the current `sessionId` in `sessionStorage` (per-tab).
- On reload, the next WS handshake includes that `sessionId`, and the server resumes the detached session if it's still within `ResumeWindow`.
- On the server you'll see `session detached` (refresh/disconnect) followed by `session resumed` when the handshake reattaches.

**Auth and Resume:**

User objects are **not serialized** during session persistence (for security). If you have authenticated sessions, configure `OnSessionResume` to rehydrate the user:

```go
vango.Config{
    OnSessionResume: func(httpCtx context.Context, s *vango.Session) error {
        // Revalidate from cookie/token in the HTTP request
        user, err := myauth.ValidateFromRequest(httpCtx)
        if err != nil {
            return err  // Rejects resume, forces new session
        }
        auth.Set(s, user)
        return nil
    },
}
```

If a session was previously authenticated but `OnSessionResume` doesn't rehydrate the user, resume is rejected to prevent "ghost authenticated" state.

**Persistent Sessions:**

```go
vango.SessionConfig{
    ResumeWindow: 30 * time.Second,
    Store: vango.RedisStore(redisClient),  // Survives restarts
}
```

### 17.9 Multi-Tenant Apps

Scope data by tenant:

```go
vango.Config{
    OnSessionStart: func(httpCtx context.Context, s *vango.Session) {
        if user, ok := vango.UserFromContext(httpCtx).(*User); ok && user != nil {
            auth.Set(s, user)
            s.Set("tenantID", user.TenantID)
        }
    },
}

// Access tenant in services
func (s *ProjectService) List(ctx vango.Ctx) ([]Project, error) {
    tenantID := 0
    if sess := ctx.Session(); sess != nil {
        tenantID, _ = sess.Get("tenantID").(int)
    } else if user, ok := ctx.User().(*User); ok && user != nil {
        tenantID = user.TenantID
    }
    return s.db.ListProjectsByTenant(ctx.StdContext(), tenantID)
}

// Global state must be tenant-scoped
var OnlineUsersByTenant = vango.NewGlobalSignal(map[int][]User{})
// NOT: var OnlineUsers = vango.NewGlobalSignal([]User{})
```

### 17.10 Auth Checklist

- [ ] Cookies are HttpOnly, Secure, SameSite
- [ ] Session tokens are cryptographically random
- [ ] Route guards protect sensitive pages
- [ ] Actions check authorization before mutations
- [ ] Logout clears all session state
- [ ] Multi-tenant data is properly scoped

---

## 18. Security Checklist

This section is a production security reference.

### 18.1 CSRF Protection

Vango provides built-in CSRF protection:

```go
vango.SecurityConfig{
    CSRFSecret:     []byte("your-32-byte-secret-key-here!!"),
    CookieSecure:   true,
    CookieHttpOnly: true,
    CookieSameSite: http.SameSiteLaxMode,
    CookieDomain:   "",
    // TrustedProxies: []string{"10.0.0.1"},
}
```

**How it works:**

1. Server generates CSRF token, sets in cookie
2. WebSocket handshake includes token
3. Server validates token matches cookie
4. All subsequent events are trusted

### 18.2 Origin Validation

Restrict WebSocket connections:

```go
vango.SecurityConfig{
    AllowedOrigins: []string{
        "https://myapp.com",
        "https://www.myapp.com",
    },
    // Or for same-origin only:
    AllowSameOrigin: true,
}
```

### 18.3 HTTP Server Timeouts

Protect the pre-upgrade HTTP listener (SSR, API, and WebSocket upgrade) from slowloris-style header drips and connection exhaustion. These are **HTTP** timeouts; WebSocket/session timeouts are configured separately in `vango.SessionConfig`.

- `app.Run()` uses defaults: `ReadHeaderTimeout=5s`, `ReadTimeout=30s`, `WriteTimeout=30s`, `IdleTimeout=60s`.
- If you run your own `http.Server`, set the same timeouts explicitly.

### 18.4 XSS Prevention

Vango escapes content by default:

```go
// Safe: text is escaped
Text(userInput)  // <script> becomes &lt;script&gt;

// Safe: attributes are escaped
Href(userProvidedURL)  // Validates scheme

// DANGEROUS: opt-in to raw HTML
DangerouslySetInnerHTML(sanitizedHTML)  // Only with sanitized input!
```

**Sanitizing User HTML:**

```go
import "github.com/microcosm-cc/bluemonday"

var policy = bluemonday.UGCPolicy()

func SafeHTML(input string) *vango.VNode {
    sanitized := policy.Sanitize(input)
    return DangerouslySetInnerHTML(sanitized)
}
```

### 18.5 Input Validation

Always validate on the server:

```go
// Request limits
vango.Config{
    Limits: vango.LimitsConfig{
        MaxEventPayloadBytes: 64 * 1024,      // 64KB
        MaxFormBytes:         10 * 1024 * 1024, // 10MB
        MaxQueryParams:       50,
        MaxHeaderBytes:       8 * 1024,
    },
}

// Validate in actions
func CreateProject(input ProjectInput) error {
    // Validate length
    if len(input.Name) > 100 {
        return errors.New("name too long")
    }
    
    // Validate format
    if !isValidSlug(input.Slug) {
        return errors.New("invalid slug format")
    }
    
    // Validate authorization
    if !canCreateProjects(currentUser) {
        return errors.New("not authorized")
    }
    
    return db.Projects.Create(input)
}
```

### 18.6 Rate Limiting

Protect against abuse:

```go
// HTTP-level rate limiting (middleware)
func RateLimitMiddleware(next http.Handler) http.Handler {
    limiter := rate.NewLimiter(rate.Limit(100), 200)  // 100 req/s, burst 200
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !limiter.Allow() {
            http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
            return
        }
        next.ServeHTTP(w, r)
    })
}

// Session-level storm budgets
vango.SessionConfig{
    StormBudget: vango.StormBudgetConfig{
        MaxResourceStartsPerSecond: 100,
        MaxActionStartsPerSecond:   50,
        MaxEffectRunsPerTick:       50,
    },
}
```

### 18.7 Cookie Security

```go
http.Cookie{
    Name:     "session",
    Value:    token,
    Path:     "/",
    HttpOnly: true,         // No JavaScript access
    Secure:   true,         // HTTPS only
    SameSite: http.SameSiteLaxMode,  // CSRF protection
    MaxAge:   86400 * 7,    // 7 days
}
```

### 18.8 Secure Headers

```go
func SecurityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        next.ServeHTTP(w, r)
    })
}
```

### 18.9 Security Checklist

| Category | Requirement | Default |
|----------|-------------|---------|
| **CSRF** | Token validation | ✓ Enabled |
| **Origin** | WebSocket origin check | Configure explicitly |
| **XSS** | Content escaping | ✓ Enabled |
| **HTTP Timeouts** | ReadHeader/Read/Write/Idle set | Defaults via app.Run |
| **Cookies** | HttpOnly, Secure, SameSite | Configure explicitly |
| **Headers** | Security headers | Add middleware |
| **Rate Limit** | HTTP and session limits | Configure explicitly |
| **Input** | Server-side validation | Implement in app |
| **Auth** | Proper session management | Implement in app |

---

## 19. Performance and Scaling

This section covers making Vango apps fast and scalable.

### 19.1 Memory Budgeting

Each session consumes memory. Budget accordingly:

```
┌─────────────────────────────────────────────────────────────────┐
│                    MEMORY PER SESSION                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Base session overhead     ~50KB                                │
│  VDOM tree (typical)       10-100KB                              │
│  Signal state              1-50KB                                │
│  Resources (cached data)   0-500KB                               │
│                                                                  │
│  Total typical:            100-500KB per session                │
│                                                                  │
│  10,000 sessions ≈ 1-5GB RAM                                    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Reducing Memory:**

```go
// Don't cache large lists in signals
// ❌ Bad: Entire list in memory
var allProducts = vango.NewSignal([]Product{})  // 10,000 items!

// ✅ Good: Paginated, loaded on demand
func ProductList() vango.Component {
    return vango.Func(func() *vango.VNode {
        page := vango.URLParam[int]("page", 1)
        products := vango.NewResourceKeyed(page, func(p int) ([]Product, error) {
            return db.Products.ListPage(p, 20)  // 20 items
        })
        // ...
    })
}
```

### 19.2 Patch Efficiency

Minimize DOM updates:

```go
// ✅ Use keys for lists
vango.ForEach(items, func(item Item, i int) *vango.VNode {
    return Li(Key(item.ID), Text(item.Name))  // Stable identity
})

// ✅ Use memos for expensive computations
expensiveResult := vango.NewMemo(func() Result {
    return computeExpensive(data.Get())  // Cached
})

// ✅ Batch updates with transactions
vango.TxNamed("UpdateDashboard", func() {
    count.Set(newCount)
    status.Set(newStatus)
    lastUpdated.Set(time.Now())
})  // Single patch instead of three

// ❌ Avoid: Large inline structures that change often
Div(
    Style(fmt.Sprintf("transform: translate(%dpx, %dpx)", x, y)),
    // Every position change = full style attribute update
)
```

### 19.3 Data Access Performance

```go
// ✅ Cache at the right layer
var configCache = cache.New(5 * time.Minute)

func GetConfig(ctx context.Context) (*Config, error) {
    if cached, ok := configCache.Get("config"); ok {
        return cached.(*Config), nil
    }
    // ...
}

// ✅ Use singleflight for hot paths
var group singleflight.Group

func GetPopularItem(ctx context.Context, id int) (*Item, error) {
    key := fmt.Sprintf("item:%d", id)
    result, err, _ := group.Do(key, func() (any, error) {
        return db.Items.FindByID(ctx, id)
    })
    // ...
}

// ❌ Avoid: Query per keystroke
func SearchHandler() vango.Component {
    return vango.Func(func() *vango.VNode {
        query := vango.NewSignal("")
        
        // ❌ Fires on every character
        results := vango.NewResource(func() ([]Item, error) {
            return db.Search(query.Get())  // N queries for N chars!
        })
        
        // ✅ Use GoLatest with debounce
        results := vango.GoLatest(func() ([]Item, error) {
            q := query.Get()
            time.Sleep(300 * time.Millisecond)  // Debounce
            return db.Search(q)
        })
    })
}
```

### 19.4 Horizontal Scaling

Options for multiple servers:

| Approach | Sessions | Global State | Complexity |
|----------|----------|--------------|------------|
| Single server | In-memory | Works | Low |
| Sticky sessions | In-memory | Pub/sub needed | Medium |
| Session store | Redis/DB | Pub/sub needed | Higher |

**With Session Store:**

```go
vango.SessionConfig{
    Store: vango.RedisStore(redisClient),
}

// Global signals need pub/sub
vango.Config{
    GlobalSignalBroadcast: vango.RedisBroadcast(redisClient),
}
```

### 19.5 Storm Budget Configuration

Prevent runaway effects:

```go
vango.SessionConfig{
    StormBudget: vango.StormBudgetConfig{
        MaxResourceStartsPerSecond: 100,
        MaxActionStartsPerSecond:   50,
        MaxGoLatestStartsPerSecond: 100,
        MaxEffectRunsPerTick:       50,
        WindowDuration:             time.Second,
        OnExceeded:                 vango.BudgetThrottle,
    },
}
```

### 19.6 Performance Checklist

- [ ] Session memory is budgeted (~100-500KB typical)
- [ ] Lists use stable keys
- [ ] Expensive computations use memos
- [ ] Multi-update operations use transactions
- [ ] Search/filter uses debouncing
- [ ] Hot queries use caching/singleflight
- [ ] Storm budgets are configured
- [ ] Load testing validates capacity

---

## 20. Observability: Logging, Metrics, Tracing

This section covers monitoring and debugging Vango apps.

### 20.1 Structured Logging

Use structured logging for production:

```go
import "log/slog"

// Configure logger
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))

vango.New(vango.Config{
    Logger: logger,
})
```

**What Vango Logs:**

| Event | Level | Fields |
|-------|-------|--------|
| Session created | Info | session_id, remote_addr |
| Session destroyed | Info | session_id, reason |
| Navigation | Debug | session_id, from, to |
| Event handled | Debug | session_id, event_type, hid |
| Error | Error | session_id, error, stack |
| Patch sent | Debug | session_id, patch_size |

### 20.2 Application Logging

Log application events:

```go
func CreateProject(input ProjectInput) error {
    ctx := vango.UseCtx()
    user := store.CurrentUser()
    
    project, err := services.Projects.Create(ctx.StdContext(), input)
    if err != nil {
        slog.Error("failed to create project",
            "user_id", user.ID,
            "error", err,
        )
        return err
    }
    
    slog.Info("project created",
        "project_id", project.ID,
        "user_id", user.ID,
    )
    return nil
}
```

**Avoid Logging Sensitive Data:**

```go
// ❌ Don't log passwords, tokens, PII
slog.Info("login", "email", email, "password", password)

// ✅ Redact or omit
slog.Info("login", "email", redactEmail(email))
```

### 20.3 Metrics

Expose key metrics:

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    sessionsActive = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "vango_sessions_active",
        Help: "Current number of active sessions",
    })
    
    eventsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
        Name: "vango_events_total",
        Help: "Total events processed",
    }, []string{"type"})
    
    patchSizeBytes = prometheus.NewHistogram(prometheus.HistogramOpts{
        Name:    "vango_patch_size_bytes",
        Help:    "Size of patches sent to clients",
        Buckets: []float64{100, 500, 1000, 5000, 10000, 50000},
    })
)
```

**Key Metrics to Track:**

| Metric | Type | Alert When |
|--------|------|------------|
| `sessions_active` | Gauge | Unusual spikes |
| `sessions_detached` | Gauge | Growing continuously |
| `events_total` | Counter | Error rate high |
| `patch_size_bytes` | Histogram | p99 > 50KB |
| `event_duration_ms` | Histogram | p99 > 500ms |
| `resource_errors_total` | Counter | Increasing |

### 20.4 Distributed Tracing

Propagate trace context:

```go
import "go.opentelemetry.io/otel"

func (s *ProjectService) GetByID(ctx context.Context, id int) (*Project, error) {
    // Create span
    ctx, span := otel.Tracer("projects").Start(ctx, "GetByID")
    defer span.End()
    
    span.SetAttributes(attribute.Int("project_id", id))
    
    // Database query inherits trace context
    return s.db.QueryRowContext(ctx, "SELECT ...")
}

// In route handler
type Params struct {
    ID int `param:"id"`
}

func Page(ctx vango.Ctx, p Params) *vango.VNode {
    // ctx.StdContext() has trace context from request
    project, err := services.Projects.GetByID(ctx.StdContext(), p.ID)
    // ...
}
```

### 20.5 DevTools

In development, Vango DevTools provide:

- Signal values and dependency graph
- Transaction history with names
- Event flow (click → handler → patch)
- Patch contents
- "Why did this render?" analysis

**Enable in development:**

```go
vango.Config{
    DevMode: true,  // Enables DevTools
}
```

### 20.6 Debugging Common Issues

| Symptom | Diagnosis | Tool |
|---------|-----------|------|
| Slow renders | Check memo dependencies | DevTools |
| High memory | Profile per-session state | `pprof` |
| Patch too large | Inspect patch contents | DevTools |
| Events not firing | Check HID assignment | Browser DevTools |
| Stale UI | Signal not updating | DevTools signals |

### 20.7 Incident Runbooks

**Sessions Spiking:**

1. Check if legitimate traffic or attack
2. Verify rate limiting is active
3. Check for session leak (not cleaning up)
4. Scale horizontally if needed

**Patch Mismatch Storms:**

1. Check for unstable keys in lists
2. Look for direct DOM manipulation
3. Check island boundaries
4. Review recent deploys

**CPU Spikes on Effects:**

1. Check storm budget logs
2. Look for infinite effect loops
3. Profile with `pprof`
4. Review recent effect changes

### 20.8 Observability Checklist

- [ ] Structured logging configured
- [ ] Sensitive data redacted
- [ ] Key metrics exposed
- [ ] Tracing propagated to dependencies
- [ ] DevTools available in development
- [ ] Alerts configured for key metrics
- [ ] Incident runbooks documented

## 21. Testing Strategy

This section defines testing patterns for Vango apps.

### 21.1 Testing Pyramid

```
                    ┌─────────┐
                    │  E2E    │  ← Few, critical paths
                    │ (slow)  │
                ┌───┴─────────┴───┐
                │  Integration    │  ← Routes, auth, layouts
                │   (medium)      │
            ┌───┴─────────────────┴───┐
            │     Component Tests      │  ← Render + events
            │        (fast)            │
        ┌───┴─────────────────────────┴───┐
        │          Unit Tests              │  ← Pure logic
        │           (fast)                 │
        └──────────────────────────────────┘
```

### 21.2 Unit Tests

Test pure functions and domain logic:

```go
// internal/services/order_test.go
func TestCalculateTotal(t *testing.T) {
    items := []OrderItem{
        {Price: 1000, Quantity: 2},  // $10 x 2
        {Price: 500, Quantity: 1},   // $5 x 1
    }
    
    total := CalculateTotal(items, 0.10)  // 10% tax
    
    assert.Equal(t, 2750, total)  // $27.50
}

func TestValidateEmail(t *testing.T) {
    tests := []struct {
        email string
        valid bool
    }{
        {"test@example.com", true},
        {"invalid", false},
        {"@example.com", false},
        {"test@", false},
    }
    
    for _, tt := range tests {
        t.Run(tt.email, func(t *testing.T) {
            assert.Equal(t, tt.valid, ValidateEmail(tt.email))
        })
    }
}
```

### 21.3 Component Tests

Test components without a browser:

```go
// app/components/counter_test.go
func TestCounter(t *testing.T) {
    // Mount component
    c := vango.Mount(Counter())
    
    // Query for elements
    button := c.Query("button")
    display := c.Query("[data-count]")
    
    // Assert initial state
    assert.Equal(t, "0", display.Text())
    
    // Simulate event
    button.Click()
    c.Flush()  // Process updates
    
    // Assert updated state
    assert.Equal(t, "1", display.Text())
}

func TestLoginForm(t *testing.T) {
    c := vango.Mount(LoginForm())
    
    // Fill form
    c.Query("input[type=email]").SetValue("test@example.com")
    c.Query("input[type=password]").SetValue("password123")
    c.Flush()
    
    // Submit
    c.Query("form").Submit()
    c.Flush()
    
    // Assert action was triggered
    assert.True(t, c.ActionCalled("login"))
}
```

**What to Test:**

| Test | Assertion |
|------|-----------|
| Initial render | Key elements present |
| State changes | UI updates correctly |
| Event handlers | Correct behavior triggered |
| Loading states | Skeleton/spinner shown |
| Error states | Error message displayed |
| Edge cases | Empty lists, long text |

### 21.4 Integration Tests

Test routes, layouts, and auth:

```go
// app/routes/projects/page_test.go
func TestProjectsPage(t *testing.T) {
    // Setup test server with mock services
    app := vango.NewTestApp(vango.Config{})
    app.MockService("projects", &MockProjectService{
        List: func() []Project {
            return []Project{{ID: 1, Name: "Test"}}
        },
    })
    
    // Navigate to page
    session := app.NewSession()
    session.Navigate("/projects")
    
    // Assert page content
    assert.Contains(t, session.Body(), "Test")
}

func TestProtectedRoute(t *testing.T) {
    app := vango.NewTestApp(vango.Config{})
    
    // Without auth
    session := app.NewSession()
    session.Navigate("/admin")
    assert.Equal(t, "/login", session.CurrentPath())  // Redirected
    
    // With auth
    session.SetUser(&User{Role: "admin"})
    session.Navigate("/admin")
    assert.Equal(t, "/admin", session.CurrentPath())  // Allowed
}
```

### 21.5 E2E Tests (Playwright)

Test critical user journeys:

```typescript
// e2e/checkout.spec.ts
import { test, expect } from '@playwright/test';

test('complete checkout flow', async ({ page }) => {
    // Navigate and add to cart
    await page.goto('/products');
    await page.click('[data-product="1"] button');
    
    // Go to cart
    await page.click('[data-cart-link]');
    await expect(page.locator('.cart-item')).toHaveCount(1);
    
    // Checkout
    await page.click('[data-checkout]');
    await page.fill('[name=email]', 'test@example.com');
    await page.fill('[name=card]', '4242424242424242');
    await page.click('[type=submit]');
    
    // Verify success
    await expect(page).toHaveURL(/\/orders\/\d+\/success/);
    await expect(page.locator('h1')).toContainText('Order Confirmed');
});
```

### 21.6 Testing Session Durability

```go
func TestSessionResume(t *testing.T) {
    app := vango.NewTestApp(vango.Config{
        Session: vango.SessionConfig{
            ResumeWindow: 5 * time.Second,
        },
    })
    
    // Create session with state
    session := app.NewSession()
    session.Navigate("/counter")
    session.Click("button")  // count = 1
    
    // Disconnect
    session.Disconnect()
    
    // Reconnect within window
    session.Reconnect()
    
    // State should be preserved
    assert.Equal(t, "1", session.Query("[data-count]").Text())
}

func TestSessionExpiry(t *testing.T) {
    app := vango.NewTestApp(vango.Config{
        Session: vango.SessionConfig{
            ResumeWindow: 100 * time.Millisecond,
        },
    })
    
    session := app.NewSession()
    session.Navigate("/counter")
    session.Click("button")
    session.Disconnect()
    
    // Wait beyond resume window
    time.Sleep(200 * time.Millisecond)
    
    // Reconnect - should get fresh session
    session.Reconnect()
    assert.Equal(t, "0", session.Query("[data-count]").Text())
}
```

### 21.7 Testing Security

```go
func TestCSRF(t *testing.T) {
    app := vango.NewTestApp(vango.Config{
        Security: vango.SecurityConfig{CSRFSecret: []byte("0123456789abcdef0123456789abcdef")},
    })
    
    // Request without CSRF token should fail
    resp := app.PostJSON("/api/projects", `{"name":"test"}`, nil)
    assert.Equal(t, 403, resp.StatusCode)
    
    // Request with proper token should succeed
    session := app.NewSession()
    token := session.GetCSRFToken()
    resp = app.PostJSON("/api/projects", `{"name":"test"}`, map[string]string{
        "X-CSRF-Token": token,
    })
    assert.Equal(t, 200, resp.StatusCode)
}

func TestOriginCheck(t *testing.T) {
    app := vango.NewTestApp(vango.Config{
        Security: vango.SecurityConfig{
            AllowedOrigins: []string{"https://myapp.com"},
        },
    })
    
    // Wrong origin rejected
    _, err := app.ConnectWebSocket("https://attacker.com")
    assert.Error(t, err)
    
    // Allowed origin works
    _, err = app.ConnectWebSocket("https://myapp.com")
    assert.NoError(t, err)
}
```

### 21.8 Testing Checklist

- [ ] Unit tests for business logic
- [ ] Component tests for interactive UI
- [ ] Integration tests for routes and auth
- [ ] E2E tests for critical paths
- [ ] Session durability tests
- [ ] Security tests (CSRF, auth)
- [ ] Performance budget tests (optional)

---

## 22. Deployment and Operations

This section covers production deployment.

### 22.1 Building for Production

```bash
# Build single binary
go build -o app ./cmd/app

# Or with vango CLI
vango build

# Output structure:
# dist/
# ├── server            # Go binary
# ├── public/           # Static assets (fingerprinted; name comes from vango.json static.dir)
# │   ├── vango.<hash>.min.js
# │   ├── styles.<hash>.css        # if tailwind.enabled = true
# │   └── assets/...
# └── manifest.json     # Asset manifest for ctx.Asset(...)
```

### 22.2 Environment Variables

```bash
# Required
ENVIRONMENT=production
DATABASE_URL=postgres://user:pass@host/db
SESSION_SECRET=<random-64-chars>

# Optional
PORT=8080
LOG_LEVEL=info
REDIS_URL=redis://host:6379

# For session store
SESSION_STORE=redis
SESSION_STORE_URL=redis://host:6379
```

### 22.3 Reverse Proxy Configuration

**Nginx:**

```nginx
upstream vango {
    server 127.0.0.1:8080;
}

server {
    listen 443 ssl http2;
    server_name myapp.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    # WebSocket upgrade
    location / {
        proxy_pass http://vango;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # WebSocket timeouts
        proxy_read_timeout 86400s;
        proxy_send_timeout 86400s;
    }
    
    # Static assets with long cache
    location /css/ {
        proxy_pass http://vango;
        add_header Cache-Control "public, max-age=31536000, immutable";
    }
    
    location /js/ {
        proxy_pass http://vango;
        add_header Cache-Control "public, max-age=31536000, immutable";
    }
}
```

**Caddy:**

```caddyfile
myapp.com {
    reverse_proxy localhost:8080
}
```

### 22.4 Zero-Downtime Deployment

**Strategy:**

1. Deploy new version alongside old
2. New sessions go to new version
3. Old sessions complete on old version
4. Old version drains and stops

**With Session Store:**

```bash
# 1. Deploy new version
docker run -d --name app-new -p 8081:8080 myapp:v2

# 2. Update load balancer to route new connections to v2
# 3. Wait for old sessions to drain (ResumeWindow + buffer)
sleep 60

# 4. Stop old version
docker stop app-old
```

### 22.5 Health Checks

```go
// Health endpoint
mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("ok"))
})

// Readiness endpoint (checks dependencies)
mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
    if err := db.Ping(); err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        return
    }
    w.WriteHeader(http.StatusOK)
})
```

### 22.6 Graceful Shutdown

```go
func main() {
    // ... setup ...
    
    server := &http.Server{Addr: ":8080", Handler: mux}
    
    // Shutdown signal
    go func() {
        sigCh := make(chan os.Signal, 1)
        signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
        <-sigCh
        
        // Give sessions time to complete
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        
        app.Shutdown(ctx)
        server.Shutdown(ctx)
    }()
    
    server.ListenAndServe()
}
```

### 22.7 Deployment Checklist

- [ ] Production build created
- [ ] Environment variables configured
- [ ] Reverse proxy configured for WebSocket
- [ ] SSL/TLS enabled
- [ ] Health checks working
- [ ] Graceful shutdown configured
- [ ] Session store configured (if needed)
- [ ] Logging and metrics ready

---

## 23. Migration and Interop

This section helps teams adopt Vango incrementally.

### 23.1 Strangler Fig Pattern

Migrate route-by-route:

```
┌─────────────────────────────────────────────────────────────────┐
│                    MIGRATION STRATEGY                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Phase 1: Mount Vango alongside existing                        │
│           ┌─────────┐  ┌─────────────────┐                      │
│           │  Vango  │  │  Existing SPA   │                      │
│           │ /new/*  │  │  /old/*         │                      │
│           └─────────┘  └─────────────────┘                      │
│                                                                  │
│  Phase 2: Migrate routes incrementally                          │
│           ┌─────────────────┐  ┌─────────┐                      │
│           │     Vango       │  │   SPA   │                      │
│           │ /new/*, /users/ │  │  /old/* │                      │
│           └─────────────────┘  └─────────┘                      │
│                                                                  │
│  Phase 3: Complete migration                                    │
│           ┌─────────────────────────────┐                       │
│           │          Vango              │                       │
│           │      All routes             │                       │
│           └─────────────────────────────┘                       │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 23.2 Mount into Existing Router

```go
// Existing Go backend with Chi
r := chi.NewRouter()

// Existing routes
r.Get("/api/users", listUsers)
r.Post("/api/users", createUser)

// Mount Vango for UI
vangoApp := vango.New(vango.Config{})
r.Mount("/", vangoApp.Handler())  // Vango handles UI routes
```

### 23.3 Embed Existing SPA as Island

```go
// Wrap existing React component as island
func LegacyDashboard(props map[string]any) *vango.VNode {
    return Div(
        ID("legacy-dashboard"),
        JSIsland("legacy-react", props),
    )
}
```

```javascript
// public/js/islands/legacy-react.js
import ReactDOM from 'react-dom';
import LegacyDashboard from './legacy/Dashboard';

export default {
    mount(container, props) {
        ReactDOM.render(<LegacyDashboard {...props} />, container);
    },
    update(props) {
        // Handle prop updates if needed
    },
    destroy() {
        ReactDOM.unmountComponentAtNode(container);
    },
};
```

### 23.4 Sharing Authentication

```go
// Existing auth middleware
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        user := validateToken(token)
        next.ServeHTTP(w, r.WithContext(vango.WithUser(r.Context(), user)))
    })
}

// Vango copies user from HTTP context into session on WS upgrade
vango.New(vango.Config{
    OnSessionStart: func(httpCtx context.Context, s *vango.Session) {
        if user := vango.UserFromContext(httpCtx); user != nil {
            s.Set(auth.SessionKey, user)
        }
    },
})
```

### 23.5 API Routes for Non-Browser Clients

```go
// routes/api/users/route.go
func GET(w http.ResponseWriter, r *http.Request) {
    users := services.Users.List(r.Context())
    json.NewEncoder(w).Encode(users)
}

func POST(w http.ResponseWriter, r *http.Request) {
    var input UserInput
    json.NewDecoder(r.Body).Decode(&input)
    user := services.Users.Create(r.Context(), input)
    json.NewEncoder(w).Encode(user)
}
```

### 23.6 Migration Checklist

- [ ] Choose migration strategy (strangler fig, big bang)
- [ ] Set up routing to handle both old and new
- [ ] Share authentication between systems
- [ ] Wrap legacy widgets as islands if needed
- [ ] Migrate route by route
- [ ] API routes for mobile/external clients

---

## 24. Recipes and Reference Apps

This section provides complete, copy-paste patterns.

### 24.1 CRUD with Pagination and Search

```go
// Use a struct for resource keys—type-safe and no string parsing
type ListKey struct {
    Search string
    Page   int
}

func UsersPage() vango.Component {
    return vango.Func(func() *vango.VNode {
        ctx := vango.UseCtx()
        stdCtx := ctx.StdContext()

        // URL-driven state (shareable, bookmarkable)
        search := vango.URLParam("search", "")
        page := vango.URLParam[int]("page", 1)

        // Derive key as a struct (comparable type)
        key := vango.NewMemo(func() ListKey {
            return ListKey{Search: search.Get(), Page: page.Get()}
        })

        // Load data—key changes trigger refetch
        users := vango.NewResourceKeyed(key, func(k ListKey) (*PaginatedUsers, error) {
            return services.Users.List(stdCtx, ListParams{
                Search: k.Search,
                Page:   k.Page,
            })
        })

        return Div(Class("users-page"),
            // Search input—updates URL on change
            Input(
                Type("search"),
                Placeholder("Search users..."),
                Value(search.Get()),
                OnChange(func(value string) {
                    search.Set(value)
                    page.Set(1) // Reset to first page
                }),
            ),

            // Table
            users.Match(
                vango.OnLoading(TableSkeleton),
                vango.OnReady(func(data *PaginatedUsers) *vango.VNode {
                    if len(data.Items) == 0 {
                        return EmptyState("No users found")
                    }
                    return Div(
                        Table(
                            Thead(Tr(Th(Text("Name")), Th(Text("Email")), Th(Text("Actions")))),
                            Tbody(
                                vango.ForEach(data.Items, func(u User, i int) *vango.VNode {
                                    return Tr(Key(u.ID),
                                        Td(Text(u.Name)),
                                        Td(Text(u.Email)),
                                        Td(
                                            Link(fmt.Sprintf("/users/%d", u.ID), Text("Edit")),
                                        ),
                                    )
                                }),
                            ),
                        ),
                        Pagination(page, data.TotalPages),
                    )
                }),
            ),
        )
    })
}
```

> **Note**: This example uses `OnChange` (fires on blur/enter) for simplicity. For search-as-you-type with debouncing, use `OnInput` combined with `GoLatest`—see **Section 10.5** for the complete pattern with cancellation and loading states.

### 24.2 Master/Detail with Optimistic Updates

```go
func TasksPage() vango.Component {
    return vango.Func(func() *vango.VNode {
        selectedID := vango.URLParam[int]("id", 0)
        
        return Div(Class("master-detail"),
            Div(Class("master"),
                TaskList(selectedID),
            ),
            Div(Class("detail"),
                selectedID.Get() > 0 && TaskDetail(selectedID.Get()),
                selectedID.Get() == 0 && Placeholder("Select a task"),
            ),
        )
    })
}

func TaskList(selected *vango.URLParam[int]) vango.Component {
    return vango.Func(func() *vango.VNode {
        tasks := vango.NewResource(func() ([]Task, error) {
            return services.Tasks.List(vango.UseCtx().StdContext())
        })
        
        return tasks.Match(
            vango.OnLoading(ListSkeleton),
            vango.OnReady(func(items []Task) *vango.VNode {
                return Ul(
                    vango.ForEach(items, func(t Task, i int) *vango.VNode {
                        return Li(Key(t.ID),
                            Class(selected.Get() == t.ID && "selected"),
                            OnClick(func() { selected.Set(t.ID) }),
                            Text(t.Title),
                        )
                    }),
                )
            }),
        )
    })
}
```

### 24.3 Real-Time Presence

```go
// store/presence.go
var OnlineUsers = vango.NewGlobalSignal(map[int]PresenceInfo{})

func TrackPresence(userID int, name string) func() {
    OnlineUsers.SetEntry(userID, PresenceInfo{
        ID:       userID,
        Name:     name,
        LastSeen: time.Now(),
    })
    
    // Update periodically
    ticker := time.NewTicker(30 * time.Second)
    go func() {
        for range ticker.C {
            vango.UseCtx().Dispatch(func() {
                OnlineUsers.UpdateEntry(userID, func(p PresenceInfo) PresenceInfo {
                    p.LastSeen = time.Now()
                    return p
                })
            })
        }
    }()
    
    // Return cleanup function
    return func() {
        ticker.Stop()
        OnlineUsers.DeleteEntry(userID)
    }
}

// In layout
func PresenceBar() vango.Component {
    return vango.Func(func() *vango.VNode {
        users := store.OnlineUsers.Get()
        return Div(Class("presence-bar"),
            Text(fmt.Sprintf("%d online", len(users))),
            vango.ForEach(maps.Values(users), func(u PresenceInfo, i int) *vango.VNode {
                return Span(Key(u.ID), Class("avatar"), Text(u.Name[:1]))
            }),
        )
    })
}
```

### 24.4 Background Job Progress

```go
func ExportPage() vango.Component {
    return vango.Func(func() *vango.VNode {
        ctx := vango.UseCtx()
        jobID := vango.NewSignal("")
        
        startExport := vango.NewAction(func() error {
            id, err := jobs.Enqueue("export", ExportParams{...})
            if err != nil {
                return err
            }
            jobID.Set(id)
            return nil
        })
        
        return Div(
            jobID.Get() == "" && Button(
                OnClick(startExport.Run),
                Disabled(startExport.State() == vango.ActionRunning),
                Text("Start Export"),
            ),
            jobID.Get() != "" && JobProgress(jobID.Get()),
        )
    })
}

func JobProgress(id string) vango.Component {
    return vango.Func(func() *vango.VNode {
        status := vango.NewResource(func() (*JobStatus, error) {
            return jobs.GetStatus(vango.UseCtx().StdContext(), id)
        })
        
        vango.Effect(func() vango.Cleanup {
            if status.State() == vango.Ready && status.Data().Status == "pending" {
                return vango.Interval(2*time.Second, func() {
                    status.Refetch()
                })
            }
            return nil
        })
        
        return status.Match(
            vango.OnLoading(Spinner),
            vango.OnReady(func(s *JobStatus) *vango.VNode {
                switch s.Status {
                case "pending":
                    return Progress(s.Progress)
                case "complete":
                    return A(Href(s.ResultURL), Text("Download"))
                case "failed":
                    return ErrorMessage(s.Error)
                }
                return nil
            }),
        )
    })
}
```

### 24.5 Reference Apps

| App | Location | Demonstrates |
|-----|----------|--------------|
| `test-vango-app` | `/examples/test-app` | All framework features |
| `trello-clone` | `/examples/trello` | Real-time collaboration |
| `dashboard` | `/examples/dashboard` | Charts, data viz |

---

## 25. Troubleshooting and FAQ

This section addresses common problems and questions.

### 25.1 Hook Order Errors

**Symptom:** `panic: hook call order changed`

**Cause:** Hooks called conditionally or in different order.

```go
// ❌ Wrong: Conditional hook
func Component() vango.Component {
    return vango.Func(func() *vango.VNode {
        if someCondition {
            count := vango.NewSignal(0)  // Not always called!
        }
        return Div()
    })
}

// ✅ Fix: Always call hooks in same order
func Component() vango.Component {
    return vango.Func(func() *vango.VNode {
        count := vango.NewSignal(0)  // Always called
        if someCondition {
            // Use count here
        }
        return Div()
    })
}
```

### 25.2 Signal Write from Wrong Goroutine

**Symptom:** `panic: signal write outside session loop`

**Cause:** Updating signal from goroutine without Dispatch.

```go
// ❌ Wrong
go func() {
    data := fetchData()
    dataSignal.Set(data)  // Wrong thread!
}()

// ✅ Fix
go func() {
    data := fetchData()
    ctx.Dispatch(func() {
        dataSignal.Set(data)  // On session loop
    })
}()
```

### 25.3 Patch Mismatch Reloads

**Symptom:** Page unexpectedly reloads, console shows "patch mismatch"

**Causes:**
1. **Unstable keys** - Using index as key, not stable ID
2. **Direct DOM manipulation** - JS modifying Vango-managed DOM
3. **Island boundary issues** - Island not properly isolated

**Debug:**
1. Check list keys are from data, not index
2. Ensure islands use proper boundaries
3. Check for third-party scripts modifying DOM

### 25.4 Routing Surprises

**Catch-all not matching:**

```
// Routes:
// /users/[id]         →  /users/123 ✓
// /users/[...path]    →  /users/123/settings ✓
// BUT: /users matches neither!

// Fix: Add explicit /users route
// /users/index.go     →  /users ✓
```

**Parameter type coercion:**

```go
// /users/[id:int]
// /users/abc → 404 (not an int)
// /users/123 → id = 123 ✓
```

### 25.5 Performance Regressions

**Effect storms:**

```go
// ❌ Creates infinite loop
vango.Effect(func() {
    count.Set(count.Get() + 1)  // Triggers itself!
})
```

**Resource repeatedly starting:**

```go
// ❌ No key = refetches on every render
resource := vango.NewResource(fetchData)

// ✅ Key prevents refetch if same
resource := vango.NewResourceKeyed(id, fetchData)
```

### 25.6 Diagnostic Workflow

1. **Reproduce**: Get minimal reproduction steps
2. **Isolate**: Remove unrelated code
3. **Instrument**: Enable DevTools, add logging
4. **Fix**: Apply fix, verify in isolation
5. **Test**: Add test to prevent regression

### 25.7 Bug Report Template

```markdown
**Environment:**
- Vango version: 
- Go version: 
- Browser: 

**Steps to reproduce:**
1. 
2. 
3. 

**Expected behavior:**

**Actual behavior:**

**Console output:**

**DevTools state:**
- Transaction name: 
- Signal values: 
- Patch size: 

**Code snippet:**
```

### 25.8 FAQ

**Q: Can I use React/Vue components?**

A: Yes, via JavaScript islands. Wrap them in an island definition.

**Q: Is SEO supported?**

A: Yes. SSR provides full HTML for crawlers.

**Q: How many concurrent users can I handle?**

A: Depends on session memory (~100-500KB) and server resources. 10,000 sessions ≈ 1-5GB RAM.

**Q: Does it work offline?**

A: Partially. SSR HTML displays, but interactivity requires WebSocket.

**Q: Can I use with existing database/ORM?**

A: Yes. Vango doesn't mandate any database layer.

**Q: How do I debug?**

A: DevTools in development, structured logging in production.

---

## 26. Appendices

### 26.1 CLI Reference

**`vango create [name]`**

Creates a new Vango project.

| Flag | Default | Description |
|------|---------|-------------|
| `--minimal` | `false` | Create minimal scaffold (home page, health API only) |
| `--with-tailwind` | `false` | Include Tailwind CSS (uses standalone binary, no Node.js) |
| `--with-db` | `""` | Include database setup: `sqlite` or `postgres` |
| `--with-auth` | `false` | Include admin routes with auth middleware |
| `--full` | `false` | All features: Tailwind, database (sqlite), auth |
| `-d, --description` | `""` | Project description |
| `-y, --yes` | `false` | Skip prompts, use defaults |

**`vango dev`**

Runs development server with hot reload.

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `3000` | Port to listen on |
| `--host` | `localhost` | Host to bind to |

**`vango build`**

Builds production binary and assets.

| Flag | Default | Description |
|------|---------|-------------|
| `--out` | `dist` | Output directory |
| `--fingerprint` | `true` | Hash asset filenames |

**`vango gen routes`**

Regenerates route registration code.

**`vango gen component <name>`**

Scaffolds a new component.

**`vango gen route <path>`**

Scaffolds a new page at the given route.

### 26.2 `vango.json` Reference

```json
{
    "module": "myapp",
    "routes": {
        "dir": "app/routes",
        "output": "app/routes/routes_gen.go"
    },
    "static": {
        "dir": "public",
        "prefix": "/"
    },
    "build": {
        "output": "dist",
        "fingerprint": true
    },
    "tailwind": {
        "enabled": true,
        "config": "tailwind.config.js",
        "input": "app/styles/app.css",
        "output": "public/css/app.css"
    },
    "devtools": {
        "enabled": true
    }
}
```

### 26.3 App Checklists

**Pre-Launch Checklist:**

- [ ] HTTPS configured
- [ ] CSRF protection enabled
- [ ] Allowed origins configured
- [ ] Session secret is random and secure
- [ ] Rate limiting in place
- [ ] Logging configured
- [ ] Metrics exposed
- [ ] Health checks working
- [ ] Error tracking configured
- [ ] Backups configured

**Accessibility Checklist:**

- [ ] All form elements have labels
- [ ] Focus visible on interactive elements
- [ ] Sufficient color contrast
- [ ] Skip links present
- [ ] ARIA attributes where needed
- [ ] Keyboard navigation works
- [ ] Screen reader tested

**Deployment Checklist:**

- [ ] Environment variables set
- [ ] Database migrations run
- [ ] Static assets built
- [ ] Reverse proxy configured
- [ ] WebSocket upgrade working
- [ ] Health checks passing
- [ ] Rollback plan ready

### 26.4 Glossary

| Term | Definition |
|------|------------|
| **Session** | Per-tab server-side state; persists across navigation |
| **Session Loop** | Single-threaded event processor for a session |
| **HID** | Hydration ID; links DOM elements to server-side VDOM |
| **Patch** | Binary-encoded DOM update sent over WebSocket |
| **Signal** | Reactive state container |
| **Memo** | Cached derived state |
| **Effect** | Side effect triggered by state changes |
| **Resource** | Async data loader with loading/error/ready states |
| **Action** | Async mutation with pending state and concurrency control |
| **Hook** | Client-side JavaScript for a DOM element |
| **Island** | Self-contained client-side widget, isolated from Vango |
| **Resume Window** | Time window within which a disconnected session can reconnect |
| **Storm Budget** | Limits on resource/action/effect rates per session |
| **Thin Client** | Minimal JavaScript runtime that applies patches |
| **Context Bridge** | Mechanism to pass HTTP request data into Vango session |
| **Progressive Enhancement** | Strategy where features work without JS, enhanced with it |
