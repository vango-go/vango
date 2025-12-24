# Vango V2 Build Roadmap

> **Production-Ready Server-Driven Web Framework for Go**

---

## Executive Summary

Vango V2 is a ground-up rewrite informed by V1 lessons. The architecture prioritizes:

1. **Server-Driven First**: Components run on server by default, 12KB thin client
2. **Binary Protocol**: Minimal bandwidth, optimized for real-time updates
3. **Progressive Enhancement**: Works without JS, enhanced with WebSocket
4. **Type Safety**: Compile-time guarantees, no runtime surprises
5. **Production Ready**: Built for 28+ companies waiting to deploy

### Target Metrics

| Metric | Target |
|--------|--------|
| Thin client size | < 15KB gzipped |
| Initial page load (SSR) | < 100ms TTFB |
| Interaction latency | < 100ms (with optimistic: 0ms perceived) |
| Memory per session | < 200KB average |
| Concurrent sessions per GB | 5,000+ |
| WebSocket reconnect | < 500ms |

---

## Phase Overview

```
=== V2.0 Core (Complete) ===

Phase 1: Reactive Core          ████████████  [Foundation] ✅ COMPLETE
    │
    ▼
Phase 2: Virtual DOM            ████████████  [Foundation] ✅ COMPLETE
    │
    ▼
Phase 3: Binary Protocol        ████████████  [Foundation] ✅ COMPLETE
    │
    ▼
Phase 4: Server Runtime         ████████████  [Integration] ✅ COMPLETE
    │
    ▼
Phase 5: Thin Client            ████████████  [Integration] ✅ COMPLETE
    │
    ▼
Phase 6: SSR & Hydration        ████████████  [Integration] ✅ COMPLETE
    │
    ▼
Phase 7: Routing                ████████████  [Features] ✅ COMPLETE
    │
    ▼
Phase 8: Higher-Level Features  ████████████  [Features] ✅ COMPLETE
    │
    ▼
Phase 9: Developer Experience   ████████████  [Polish] ✅ COMPLETE
    │
    ▼
Phase 10: Middleware & Auth     ████████████  [Features] ✅ COMPLETE
    │
    ▼
Phase 11: Security Hardening    ████████████  [Security] ✅ COMPLETE

=== V2.1 Production Release Track ===

    │
    ▼
Phase 12: Session Resilience    ████████████  [Resilience] ✅ COMPLETE
    │
    ▼
Phase 13: Observability         ████████████  [Production] ✅ COMPLETE
    │
    ▼
Phase 14: CLI & Scaffold        ████████████  [DX] ✅ COMPLETE
    │
    ▼
Phase 15: VangoUI               ░░░░░░░░░░░░  [Components] ◀── NEXT
    │
    ▼
Phase 16: Platform APIs         ░░░░░░░░░░░░  [Platform]
```

### V2.1 Release Track Summary

| Phase | Focus | Key Features | Status |
|-------|-------|--------------|--------|
| 12 | Resilience | SessionStore, LRU eviction, reconnect UX, URLParam 2.0 | ✅ Complete |
| 13 | Observability | OpenTelemetry middleware, Prometheus metrics, fuzz testing | ✅ Complete |
| 14 | DX | `vango create`, `vango gen`, `vango dev`, route generator | ✅ Complete |
| 15 | Components | VangoUI, functional options API, client hooks | **Next** |
| 16 | Platform | `ctx.Async`, `ctx.Client`, `ctx.Sync`, NATS JetStream | Planned |

---

## Phase 1: Reactive Core ✅ COMPLETE

**Goal**: Build the reactive primitives that power all state management.

**Duration**: Foundation phase

**Status**: Complete (2024-12-06)

**Deliverables**:
- [x] `Signal[T]` - Reactive value with automatic dependency tracking
- [x] `Memo[T]` - Cached derived computation
- [x] `Effect` - Side effect with cleanup
- [x] `Batch` - Group updates for single re-render
- [x] `Untracked` - Read without subscribing
- [x] Ownership model (component-scoped signals)
- [x] Unit tests with 91.6% coverage
- [x] Type-specific wrappers (IntSignal, BoolSignal, SliceSignal, MapSignal)
- [x] Thread-safe concurrent access (race detector passes)

**Dependencies**: None (foundation)

**Key Decisions**:
- Goroutine-local tracking context via sync.Map
- Separate locks for values vs. subscribers (RWMutex)
- Copy-before-notify pattern to prevent deadlocks
- Idiomatic Go API (`NewSignal`, `NewMemo`, `CreateEffect`) - see API note below

**API Note (Go Constraints)**:
The spec's idealized syntax (`Signal(0)`, `count()`) conflicts with Go's type system.
We use idiomatic Go constructors instead:

| Spec | Implementation | Reason |
|------|----------------|--------|
| `Signal(0)` | `NewSignal(0)` | Type/function name conflict |
| `Memo(fn)` | `NewMemo(fn)` | Type/function name conflict |
| `Effect(fn)` | `CreateEffect(fn)` | Type/function name conflict |
| `count()` | `count.Get()` | No operator overloading in Go |

**Exit Criteria** (verified working):
```go
count := vango.NewSignal(0)
doubled := vango.NewMemo(func() int { return count.Get() * 2 })

vango.CreateEffect(func() vango.Cleanup {
    fmt.Println("Count is:", count.Get())
    return func() { fmt.Println("Cleanup") }
})

count.Set(5) // Triggers effect, memo recomputes
```

**Detailed Spec**: [PHASE_01_CORE.md](./PHASE_01_CORE.md)

---

## Phase 2: Virtual DOM ✅ COMPLETE

**Goal**: Build the VNode representation and diffing algorithm.

**Duration**: Foundation phase

**Status**: Complete (2024-12-06)

**Deliverables**:
- [x] `VNode` struct (Element, Text, Fragment, Component, Raw)
- [x] Element factory functions (95 HTML5 elements)
- [x] Attribute functions (80+ including ARIA, form, media)
- [x] Event handler functions (70+ including mouse, keyboard, touch, media)
- [x] Props handling (merge, override, conditional)
- [x] Child normalization (flatten, filter nil)
- [x] Helper functions (Text, Fragment, If/IfElse/When/Unless, Switch, Range, Key)
- [x] Diff algorithm with keyed reconciliation
- [x] Patch generation (10 patch types)
- [x] Hydration ID generation and assignment
- [x] Unit tests with 96.8% coverage
- [x] Diff algorithm benchmarks

**Dependencies**: Phase 1 (signals referenced in event handlers)

**Key Decisions**:
- Immutable VNodes (never mutate after creation)
- Hydration IDs assigned during SSR render phase (not at VNode creation)
- Event handlers stored in Props with "on" prefix (onclick, oninput)
- Keys required for efficient list reconciliation (unkeyed uses positional matching)
- Children as `[]*VNode` pointers to allow nil filtering for conditionals
- Variadic element API: `Div(Class("card"), H1(Text("Title")), OnClick(fn))`

**File Structure** (`pkg/vdom/`):
| File | Lines | Description |
|------|-------|-------------|
| `vnode.go` | 97 | VNode, VKind, Props, Attr, EventHandler, Component |
| `patch.go` | 55 | PatchOp enum (10 operations), Patch struct |
| `elements.go` | 234 | createElement + 95 HTML5 elements |
| `attributes.go` | 310 | 80+ attribute functions |
| `events.go` | 270 | 70+ event handler functions |
| `helpers.go` | 210 | Text, Fragment, If, Range, Switch, etc. |
| `diff.go` | 280 | Diff algorithm (keyed + unkeyed) |
| `hydration.go` | 170 | HIDGenerator, AssignHIDs |

**Benchmark Highlights**:
| Operation | Time | Allocs |
|-----------|------|--------|
| Simple Div creation | 114 ns | 5 |
| Complex card (6 elements) | 930 ns | 45 |
| Diff same tree (100 nodes) | 2.4 µs | 0 |
| Diff text change | 79 ns | 1 |
| Diff keyed reorder (100 items) | 23 µs | 38 |
| Diff large tree (1000 nodes) | 26 µs | 1 |

**Exit Criteria** (verified working):
```go
// Element creation with variadic API
node := Div(Class("card"), ID("main"),
    H1(Text("Hello")),
    Button(OnClick(handler), Text("Click")),
)

// Conditional rendering
If(showHeader, Header(Text("Welcome")))

// List rendering with keys
Ul(Range(items, func(item Item, i int) *VNode {
    return Li(Key(item.ID), Text(item.Name))
})...)

// Diffing produces minimal patches
patches := Diff(oldTree, newTree)
// patches = [{SetText, "h1", "new text"}, ...]
```

**Detailed Spec**: [PHASE_02_VDOM.md](./PHASE_02_VDOM.md)

---

## Phase 3: Binary Protocol ✅ COMPLETE

**Goal**: Define and implement the wire protocol for events and patches.

**Duration**: Foundation phase

**Status**: Complete (2024-12-07)

**Deliverables**:
- [x] Varint encoding/decoding (protobuf-style unsigned, ZigZag signed)
- [x] Encoder/Decoder with all primitive types
- [x] Frame transport (4-byte header with type, flags, length)
- [x] Event types and encoding (25+ event types)
- [x] Patch types and encoding (20+ patch operations)
- [x] VNode serialization for InsertNode/ReplaceNode
- [x] Handshake message format (ClientHello/ServerHello)
- [x] Control messages (Ping, Pong, Resync, Close)
- [x] Acknowledgment for reliable delivery
- [x] Error messages with codes
- [x] Protocol version negotiation
- [x] Fuzz tests for decoder robustness (12 fuzz targets)
- [x] Benchmarks (all targets exceeded)
- [x] Package documentation (doc.go)

**Dependencies**: Phase 2 (VNode structure for serialization)

**Key Decisions**:
- Varint for small numbers (most HIDs, lengths, sequence numbers)
- ZigZag encoding for signed integers (negative values efficient)
- Big-endian for fixed-width integers (network byte order)
- Length-prefixed strings (varint length + UTF-8 bytes)
- No reflection (direct byte manipulation for performance)
- Sequence numbers for reliable delivery
- Resync capability on reconnect
- Protocol PatchOp is superset of vdom.PatchOp (0x01-0x0B shared, extends to 0x21)
- VNodeWire strips event handlers for serialization

**File Structure** (`pkg/protocol/`):
| File | Lines | Description |
|------|-------|-------------|
| `varint.go` | 75 | Varint encoding/decoding, ZigZag |
| `encoder.go` | 132 | Binary encoder with all Write methods |
| `decoder.go` | 185 | Binary decoder with all Read methods |
| `frame.go` | 180 | Frame types, flags, transport |
| `event.go` | 385 | 25+ event types with payloads |
| `patch.go` | 355 | 20+ patch operations |
| `vnode.go` | 175 | VNodeWire format for serialization |
| `handshake.go` | 175 | ClientHello/ServerHello |
| `control.go` | 220 | Ping/Pong, Resync, Close |
| `ack.go` | 50 | Acknowledgment |
| `error.go` | 100 | Error messages with codes |
| `doc.go` | 135 | Package documentation |

**Benchmark Highlights**:
| Operation | Target | Actual |
|-----------|--------|--------|
| Event encode (click) | < 500ns | ~52ns |
| Event decode (click) | < 500ns | ~26ns |
| Patch encode (SetText) | < 500ns | ~53ns |
| Patch decode (SetText) | < 500ns | ~56ns |
| 100 patches encode | < 50μs | ~1.1μs |
| 100 patches decode | < 50μs | ~2.4μs |
| Varint decode | < 10ns | ~1.3ns |

**Exit Criteria** (verified working):
```go
// Encode event
event := &Event{Seq: 1, Type: EventClick, HID: "h42"}
data := EncodeEvent(event)
// data = ~5 bytes

// Decode event
decoded, err := DecodeEvent(data)
// decoded.Type == EventClick, decoded.HID == "h42"

// Encode patches
pf := &PatchesFrame{
    Seq: 1,
    Patches: []Patch{
        NewSetTextPatch("h1", "Hello, World!"),
        NewSetAttrPatch("h2", "class", "active"),
    },
}
data = EncodePatches(pf)

// Decode patches
decoded, err := DecodePatches(data)
// decoded.Patches = [{SetText, "h1", "Hello, World!"}, {SetAttr, "h2", "class", "active"}]
```

**Detailed Spec**: [PHASE_03_PROTOCOL.md](./PHASE_03_PROTOCOL.md)

---

## Phase 4: Server Runtime ✅ COMPLETE

**Goal**: Build the server-side component execution environment.

**Duration**: Integration phase

**Status**: Complete (2024-12-07)

**Deliverables**:
- [x] `Session` - Per-connection state container
- [x] `SessionManager` - Create, retrieve, cleanup sessions
- [x] `ComponentInstance` - Mounted component with signals
- [x] `HandlerRegistry` - Map HID to handler functions (via session.handlers)
- [x] Event loop (receive event → run handler → diff → send patches)
- [x] WebSocket upgrade and connection management
- [x] Heartbeat and timeout handling
- [x] Graceful shutdown
- [x] Session memory limits and eviction (LRU)
- [x] Unit tests for all components
- [x] Three-goroutine architecture (ReadLoop, WriteLoop, EventLoop)
- [x] Metrics collection (MetricsCollector)
- [x] Memory monitoring (MemoryMonitor)

**Dependencies**:
- Phase 1 (signals for component state)
- Phase 2 (VNode for component output)
- Phase 3 (protocol for wire format)

**Key Decisions**:
- Three goroutines per session (read, write, event processing)
- Session state in memory (not Redis, for V2)
- Configurable timeouts and limits
- Handler re-registration on each render
- vango.Owner per component for reactive ownership
- Patch conversion from vdom.Patch to protocol.Patch at send time

**File Structure** (`pkg/server/`):
| File | Lines | Description |
|------|-------|-------------|
| `doc.go` | 75 | Package documentation |
| `errors.go` | 120 | Error types and sentinels |
| `config.go` | 216 | SessionConfig, ServerConfig, SessionLimits |
| `context.go` | 200 | Ctx interface for request context |
| `handler.go` | 325 | Handler type, event types, wrappers |
| `component.go` | 246 | ComponentInstance with Owner |
| `session.go` | 550 | Session struct and lifecycle |
| `manager.go` | 366 | SessionManager with cleanup |
| `websocket.go` | 275 | ReadLoop, WriteLoop, EventLoop |
| `server.go` | 335 | HTTP server, WebSocket upgrade |
| `metrics.go` | 222 | MetricsCollector with latency tracking |
| `memory.go` | 290 | MemoryMonitor, byte size utilities |

**Exit Criteria** (verified working):
```go
// Session creation on WS connect
session, err := manager.Create(conn, userID)

// Mount root component
session.MountRoot(Counter())
// Internally: renders, assigns HIDs, collects handlers

// Start session loops
session.Start()
// Three goroutines: ReadLoop, WriteLoop, EventLoop

// Event handling (internal)
// ReadLoop → decode event → queue
// EventLoop → find handler → run → run effects → diff → send patches

// Cleanup
session.Close() // Disposes owner, clears handlers, closes connection
```

**Detailed Spec**: [PHASE_04_RUNTIME.md](./PHASE_04_RUNTIME.md)

---

## Phase 5: Thin Client ✅ COMPLETE

**Goal**: Build the minimal JavaScript client that renders patches.

**Duration**: Integration phase

**Status**: Complete (2024-12-07)

**Deliverables**:
- [x] WebSocket connection with auto-reconnect
- [x] Binary message parsing
- [x] Event capture (click, input, submit, focus, blur, keydown, scroll)
- [x] Event encoding and sending
- [x] Patch application (all patch types)
- [x] DOM node creation from VNode encoding
- [x] Hydration ID tracking (`data-hid` → DOM node map)
- [x] Optimistic update support
- [x] Client hooks (Sortable, Tooltip, Dropdown, Draggable)
- [x] Minified bundle < 15KB gzipped (actual: **9.56 KB**)
- [ ] Browser compatibility testing (pending)

**Dependencies**:
- Phase 3 (protocol for message format)
- Phase 4 (server to test against)

**Key Decisions**:
- Vanilla JavaScript (no framework dependencies)
- Single file output (no module splitting)
- esbuild for bundling
- ESM modules for development, IIFE for production

**File Structure** (`client/`):
| File | Lines | Description |
|------|-------|-------------|
| `src/index.js` | 275 | VangoClient entry point |
| `src/codec.js` | 540 | Binary protocol encoding/decoding |
| `src/websocket.js` | 160 | Connection management |
| `src/events.js` | 295 | Event capture and delegation |
| `src/patches.js` | 295 | DOM patch application |
| `src/optimistic.js` | 135 | Optimistic update handling |
| `src/hooks/manager.js` | 130 | Hook lifecycle |
| `src/hooks/*.js` | 450 | 4 standard hooks |

**Bundle Size**:
| Build | Size |
|-------|------|
| Raw | 35.87 KB |
| Minified | 35.87 KB |
| **Gzipped** | **9.56 KB** |

**Exit Criteria** (verified working):
```javascript
// Auto-connects on page load ✓
// Captures events on [data-hid] elements ✓
// Applies patches from server ✓
// Reconnects on disconnect ✓
// 9.56 KB gzipped (< 15KB target) ✓
```

**Detailed Spec**: [PHASE_05_CLIENT.md](./PHASE_05_CLIENT.md)

---

## Phase 6: SSR & Hydration ✅ COMPLETE

**Goal**: Render components to HTML and hydrate on client.

**Duration**: Integration phase

**Status**: Complete (2024-12-07)

**Deliverables**:
- [x] `RenderToString(VNode) string` - Full HTML output
- [x] `RenderToWriter(io.Writer, VNode)` - Streaming output
- [x] Hydration ID generation during render
- [x] `data-hid` attribute injection
- [x] Void element handling (`<input>`, `<br>`, etc.)
- [x] Boolean attribute handling (`disabled`, `checked`, etc.)
- [x] Text escaping (XSS prevention)
- [x] Full page rendering with head/body/scripts
- [x] Document structure (`<!DOCTYPE>`, `<html>`, etc.)
- [x] Integration with thin client script injection
- [x] CSRF token and session ID injection
- [x] StreamingRenderer for large pages
- [x] Benchmark tests
- [x] Unit tests (69.2% coverage)

**Dependencies**:
- Phase 2 (VNode to render)
- Phase 5 (thin client to inject)

**Key Decisions**:
- Streaming by default (low memory)
- Hydration IDs only on interactive elements
- No client-side re-render (patches only)
- Full HTML5 spec compliance
- Sorted attribute output for deterministic HTML

**File Structure** (`pkg/render/`):
| File | Lines | Description |
|------|-------|--------------|
| `doc.go` | 55 | Package documentation |
| `escape.go` | 60 | HTML/attribute escaping |
| `elements.go` | 95 | Void/inline/boolean element lists |
| `renderer.go` | 320 | Main Renderer type |
| `page.go` | 340 | Full page rendering, PageData |
| `streaming.go` | 95 | StreamingRenderer |

**Benchmark Highlights**:
| Operation | Time |
|-----------|------|
| Simple element render | ~600ns |
| Large tree (1000 nodes) | ~154µs |
| With handlers (100 buttons) | ~43µs |
| Full page render | ~1µs |
| Escape HTML | ~170-230ns |

**Exit Criteria** (verified working):
```go
html := renderer.RenderToString(Div(Class("app"),
    H1(Text("Hello")),
    Button(OnClick(handler), Text("Click")),
))
// Output:
// <div class="app"><h1>Hello</h1><button data-hid="h1" data-on-click="true">Click</button></div>
```

**Detailed Spec**: [PHASE_06_SSR.md](./PHASE_06_SSR.md)

---

## Phase 7: Routing

**Goal**: File-based routing with layouts and navigation.

**Duration**: Features phase

**Status**: In Progress (14/15 exit criteria complete, 2024-12-07)

**Deliverables**:
- [x] Route scanner (read `app/routes/**/*.go`)
- [x] Route tree builder (radix tree)
- [x] Parameter extraction (`[id]` → `:id`)
- [x] Parameter type coercion (int, string, uuid)
- [x] Catch-all routes (`[...slug]`)
- [x] Layout detection and composition (`_layout.go`)
- [x] Middleware chain
- [x] `Navigate(path)` for programmatic navigation
- [x] Link prefetching
- [x] 404/500 error pages (handlers registered)
- [x] Route code generator (compile-time)
- [ ] Hot reload support (deferred to Phase 9: DX)

**Dependencies**:
- Phase 4 (server for HTTP handling)
- Phase 6 (SSR for page rendering)

**Key Decisions**:
- File-based is the default (no manual registration)
- Code generation for performance
- Layouts wrap children automatically
- Navigation via WebSocket (no full page reload)

**File Structure** (`pkg/router/`):
| File | Lines | Description |
|------|-------|-------------|
| `doc.go` | 53 | Package documentation |
| `types.go` | 119 | Handler types, ScannedRoute, MatchResult |
| `tree.go` | 195 | Radix tree implementation |
| `scanner.go` | 230 | Go file scanner |
| `params.go` | 145 | Parameter parsing |
| `middleware.go` | 50 | Middleware composition |
| `router.go` | 160 | Main Router type |
| `codegen.go` | 215 | Code generator |
| `navigate.go` | 140 | Navigator interface |
| `link.go` | 60 | Link components |

**Benchmark Highlights**:
| Operation | Time | Allocs |
|-----------|------|--------|
| Static match | 64 ns | 3 |
| Param match | 118 ns | 4 |
| Catch-all match | 214 ns | 6 |
| Large tree (100 routes) | 152 ns | 3 |

**Exit Criteria** (verified working):
```
app/routes/
├── index.go           → GET /
├── about.go           → GET /about
├── projects/
│   ├── index.go       → GET /projects
│   ├── [id].go        → GET /projects/:id
│   └── _layout.go     → Wraps all /projects/* routes
```

**Detailed Spec**: [PHASE_07_ROUTING.md](./PHASE_07_ROUTING.md)

---

## Phase 8: Higher-Level Features ✅ COMPLETE

**Goal**: Build the APIs that make Vango productive.

**Duration**: Features phase

**Status**: Complete (2024-12-07)

**Deliverables**:

### 8.1 Forms & Validation
- [x] `UseForm(struct)` hook
- [x] Field binding with error display
- [x] Built-in validators (Required, Email, MinLength, etc.)
- [x] Custom validators
- [x] Form arrays
- [x] Async validation

### 8.2 Resources
- [x] `Resource[T]` for async data loading
- [x] Loading/Error/Ready states
- [x] `Match()` helper for state handling
- [x] Refetch capability
- [x] Stale time configuration
- [x] Initial data support

### 8.3 Context API
- [x] `CreateContext[T](default)`
- [x] `Provider(value, children...)`
- [x] `Use()` to consume

### 8.4 URL State
- [x] `UseURLState(key, default)`
- [x] Sync with query parameters
- [x] History push/replace
- [x] Debounce option

### 8.5 Client Hooks
- [x] `Hook(name, config)` attribute
- [x] `OnEvent(name, handler)` attribute
- [x] Standard hooks: Sortable, Draggable, Tooltip, Dropdown
- [x] Custom hook registration
- [x] Hook JavaScript bundling

### 8.6 Shared State
- [x] `SharedSignal[T]` - Session-scoped
- [x] `GlobalSignal[T]` - Cross-session
- [x] `SharedMemo[T]` and `GlobalMemo[T]`
- [x] Persistence backends (LocalStorage, Database)

### 8.7 Optimistic Updates
- [x] `optimistic.Class(class, add/remove)`
- [x] `optimistic.Text(text)`
- [x] `optimistic.Attr(name, value)`
- [x] Revert on server error

### 8.8 JS Islands
- [x] `Island(id, children)` container
- [x] Client-side hydration support
- [x] Island JavaScript bundling

**Test Coverage**:
| Package | Coverage |
|---------|----------|
| context | 100.0% |
| form | 80.1% |
| hooks | 100.0% |
| hooks/standard | 100.0% |
| islands | 100.0% |
| optimistic | 100.0% |
| resource | 85.8% |
| store | 96.4% |
| urlstate | 92.7% |

**File Structure** (`pkg/features/`):
| File | Purpose |
|------|---------|
| `context/context.go` | Context API implementation |
| `form/form.go` | Form state management |
| `form/validators.go` | Built-in validators |
| `form/array.go` | Dynamic form arrays |
| `hooks/hooks.go` | Client hooks system |
| `hooks/standard/*.go` | Standard hook implementations |
| `islands/islands.go` | JS Islands support |
| `optimistic/optimistic.go` | Optimistic updates |
| `resource/resource.go` | Async data loading |
| `store/store.go` | Shared state management |
| `urlstate/urlstate.go` | URL state sync |
| `integration_test.go` | Cross-package workflow tests |

**Dependencies**: All previous phases

**Detailed Spec**: [PHASE_08_FEATURES.md](./PHASE_08_FEATURES.md)

---

## Phase 9: Developer Experience ✅ COMPLETE

**Goal**: Make Vango a joy to use.

**Duration**: Polish phase

**Status**: Complete (2024-12-07)

**Deliverables**:

### 9.1 CLI
- [x] `vango create <name>` - Project scaffolding with templates (minimal, full, api)
- [x] `vango dev` - Development server with hot reload
- [x] `vango build` - Production build with minification
- [x] `vango test` - Test runner wrapper
- [x] `vango gen` - Code generation (routes, components)
- [x] `vango add` - VangoUI component registry
- [x] `vango version` - Version information

### 9.2 Hot Reload
- [x] File watcher with debouncing
- [x] Incremental Go compilation
- [x] WebSocket push to browsers
- [x] Error overlay in browser
- [x] CSS-only reload support
- [ ] State preservation across reloads (deferred to Phase 10)

### 9.3 Error Messages
- [x] VangoError type with source location
- [x] Suggested fixes with examples
- [x] Error registry with 50+ error codes
- [x] Colored terminal output
- [x] JSON and compact output formats

### 9.4 DevTools
- [ ] Browser extension (deferred to Phase 10)
- [x] Development mode logging
- [ ] Network panel (deferred to Phase 10)
- [ ] Component tree viewer (deferred to Phase 10)

### 9.5 Documentation
- [x] Project templates with README
- [x] API documentation structure defined
- [ ] Getting started guide (deferred to Phase 10)
- [ ] Video tutorials (deferred to Phase 10)

### 9.6 Component Registry (VangoUI)
- [x] `vango add init` - Initialize UI components
- [x] `vango add [components...]` - Install components
- [x] `vango add upgrade` - Upgrade installed components
- [x] Dependency resolution (topological sort)
- [x] Local modification detection
- [x] Component headers with version/checksum

**File Structure** (`cmd/vango/`):
| File | Description |
|------|-------------|
| `main.go` | CLI entry point with cobra |
| `create.go` | Project scaffolding |
| `dev.go` | Development server |
| `build.go` | Production build |
| `gen.go` | Code generation |
| `add.go` | Component registry |
| `test.go` | Test runner |
| `version.go` | Version info |

**File Structure** (`internal/`):
| Package | Description |
|---------|-------------|
| `errors/` | Structured error messages |
| `config/` | vango.json parsing |
| `dev/` | Watcher, compiler, reload server |
| `build/` | Production builder |
| `templates/` | Project templates |
| `registry/` | VangoUI component management |

**Exit Criteria** (verified working):
```bash
# Create new project
vango create my-app

# Start development
cd my-app && vango dev

# Build for production
vango build

# Add components
vango add init
vango add button dialog
```

**Dependencies**: All previous phases

**Detailed Spec**: [PHASE_09_DX.md](./PHASE_09_DX.md)

---

## Phase 10: Production Hardening

**Goal**: Make Vango production-ready.

**Duration**: Release phase

**Deliverables**:

### 10.1 Security
- [ ] CSRF token generation and validation
- [ ] XSS prevention audit
- [ ] CSP header middleware
- [ ] Session encryption
- [ ] Rate limiting

### 10.2 Performance
- [ ] Connection pooling
- [ ] Session memory optimization
- [ ] Patch batching
- [ ] Compression (gzip/brotli)
- [ ] CDN-friendly static assets

### 10.3 Reliability
- [ ] Graceful shutdown
- [ ] Health checks
- [ ] Circuit breakers
- [ ] Retry logic

### 10.4 Observability
- [ ] Structured logging (slog)
- [ ] Prometheus metrics
- [ ] OpenTelemetry tracing
- [ ] Error tracking integration

### 10.5 Deployment
- [ ] Docker image
- [ ] Kubernetes manifests
- [ ] Systemd service
- [ ] Cloud provider guides (AWS, GCP, Fly.io)

### 10.6 Testing
- [x] Unit test coverage > 80%
- [x] Integration test suite
- [x] E2E tests (Playwright)
- [ ] Performance benchmarks
- [ ] Chaos testing

**Dependencies**: All previous phases

**Detailed Spec**: [PHASE_10.md](./PHASE_10.md)

---

## Phase 11: Security Hardening ✅ COMPLETE

**Goal**: Ensure Vango is secure by default.

**Duration**: Security phase

**Status**: Complete (2024-12-10)

**Deliverables**:
- [x] Protocol decoder hardening (allocation limits, bounds checking)
- [x] Secure server defaults (Same-origin WebSocket, CSRF)
- [x] PatchEval removal (eliminate RCE vector)
- [x] Attribute sanitization (XSS prevention)
- [x] Double Submit Cookie pattern for CSRF
- [x] Security audit documentation

**Detailed Spec**: [PHASE_11.md](./PHASE_11.md)

---

## Phase 12: Session Resilience & State Persistence ✅ COMPLETE

**Goal**: Make Vango applications survive disconnects, restarts, and page refreshes.

**Duration**: Resilience phase

**Status**: In Progress

**Deliverables**:

### 12.1 SessionStore Interface
- [x] `SessionStore` interface with Save/Load/Delete/Touch/SaveAll
- [x] `MemoryStore` - Default in-memory implementation
- [x] `RedisStore` - Production Redis backend
- [x] `SQLStore` - PostgreSQL/MySQL backend

### 12.2 Session Serialization
- [ ] `Transient()` signal option (skip persistence)
- [ ] `PersistKey(key)` signal option
- [ ] `Session.Serialize()` / `Deserialize()`
- [ ] Graceful `Server.Shutdown()` saves all sessions

### 12.3 Memory Protection
- [ ] `MaxDetachedSessions` with LRU eviction
- [ ] `MaxSessionsPerIP` limit
- [ ] Configurable eviction policies

### 12.4 Reconnection UX
- [x] Automatic CSS classes (`.vango-connected`, `.vango-reconnecting`, `.vango-offline`)
- [x] Optional toast notifications
- [x] Exponential backoff with configurable retry limits

### 12.5 URLParam 2.0
- [x] `vango.Replace` / `vango.Push` mode options
- [x] `Debounce(duration)` option
- [x] Complex type encoding (Flat, JSON, Comma)

### 12.6 Preference Sync
- [x] `Pref` primitive with merge strategies
- [x] Cross-tab sync via BroadcastChannel
- [x] Cross-device sync via database

### 12.7 Testing Infrastructure
- [x] `vtest.NewTestSession` with lifecycle simulation
- [x] `SimulateDisconnect()`, `SimulateReconnect()`, `SimulateRefresh()`

**Dependencies**: Phases 1-11

**Detailed Spec**: [PHASE_12.md](./PHASE_12.md)

---

## Phase 13: Production Hardening & Observability ✅ COMPLETE

**Goal**: Production-grade monitoring, security gates, and protocol defense.

**Duration**: Production phase

**Status**: Complete (2024-12-24)

**Deliverables**:

### 13.1 OpenTelemetry Integration
- [x] Middleware-first tracing (no `ctx.Trace()` API)
- [x] Distributed trace propagation via `ctx.StdContext()`
- [x] Span creation for WebSocket events

### 13.2 Protocol Hardening
- [x] `MaxVNodeDepth`, `MaxPatchDepth`, `MaxHookDepth` limits
- [x] Protocol allocation audit
- [x] Fuzz testing suite (Go native fuzzing)

### 13.3 Secure Defaults
- [x] `CheckOrigin` defaults to SameOrigin
- [x] `CSRFProtection` defaults to true
- [x] `SecureCookies` defaults to true
- [x] Explicit `DevMode` opt-out

### 13.4 Prometheus Metrics
- [x] `active_sessions` gauge
- [x] `events_total` counter
- [x] `event_duration_seconds` histogram
- [x] `patches_sent_total` counter
- [x] `session_memory_bytes` gauge
- [x] `ws_errors` counter
- [x] Grafana dashboard (`examples/monitoring/grafana-dashboard.json`)

**File Structure** (`pkg/middleware/`):
| File | Lines | Description |
|------|-------|-------------|
| `doc.go` | 60 | Package documentation |
| `otel.go` | 250 | OpenTelemetry tracing middleware |
| `metrics.go` | 315 | Prometheus metrics middleware |
| `middleware_test.go` | 275 | Comprehensive tests |

**File Structure** (`pkg/protocol/` additions):
| File | Lines | Description |
|------|-------|-------------|
| `limits.go` | 78 | Depth limit constants and helpers |
| `security_test.go` | 330 | Allocation and depth limit tests |
| `fuzz_test.go` (expanded) | 200 | Native Go fuzz tests |

**Dependencies**: Phase 12

**Detailed Spec**: [PHASE_13.md](./PHASE_13.md)

---

## Phase 14: CLI & Scaffold ✅ COMPLETE

**Goal**: Official Vango CLI with project scaffolding and code generation.

**Duration**: DX phase

**Status**: Complete (2024-12-24)

**Deliverables**:

### 14.1 Project Scaffolding ✅ COMPLETE
- [x] `vango create <name>` - Production-ready scaffold
- [x] `--minimal`, `--with-tailwind`, `--with-db`, `--with-auth` flags
- [x] Static serving (`public/` at `/`)
- [x] Four templates: minimal, standard, full, api
- [x] File watcher integration with smart route regeneration

### 14.2 Route Generator ✅ COMPLETE
- [x] `vango gen route <path>` - Page route with param type inference
- [x] `vango gen api <path>` - API route generation
- [x] `routes_gen.go` glue file generation with deterministic output
- [x] AST-based scanner for route detection
- [x] Parameter type inference from naming conventions ([id] → int, [slug] → string)
- [x] `_layout.go` and `_middleware.go` detection
- [x] **Build error recovery** (symbol rename detection) - Auto-regenerates routes_gen.go

### 14.3 Component Generator ✅ COMPLETE
- [x] `vango gen component <name>` - Component generation
- [x] `vango gen store <name>` - Store generation
- [x] `vango gen middleware <name>` - Middleware generation
- [x] Package collision detection and warning

### 14.4 VangoUI Integration ✅ COMPLETE
- [x] `vango add init` - Initialize VangoUI
- [x] `vango add <component>` - Install components with dependency resolution
- [x] `vango add list` - List available components
- [x] `vango add upgrade` - Upgrade installed components
- [x] `--path` flag for custom component path

### 14.5 Build System & Enhancements ✅ COMPLETE
- [x] `vango build` for production
- [x] Build configuration in vango.json (MinifyAssets, StripSymbols)
- [x] `vango test` - Test runner wrapper
- [x] `vango gen openapi` - OpenAPI 3.0 generation from typed API routes
- [ ] VS Code extension - SEPARATE PROJECT (not part of CLI)

**File Structure** (`cmd/vango/`):
| File | Lines | Description |
|------|-------|-------------|
| `main.go` | 70 | CLI entry point with cobra |
| `create.go` | 200 | Project scaffolding with variants |
| `dev.go` | 130 | Development server |
| `build.go` | 70 | Production build |
| `gen.go` | 450 | Code generation (route, api, component, store, middleware) |
| `add.go` | 360 | VangoUI component registry |
| `test.go` | 50 | Test runner wrapper |
| `version.go` | 30 | Version info |

**File Structure** (`internal/templates/`):
| File | Lines | Description |
|------|-------|-------------|
| `templates.go` | 1200 | Phase 14 scaffold templates (minimal, standard, full, api) |

**Test Coverage**:
| Package | Coverage |
|---------|----------|
| internal/config | 89.3% |
| internal/templates | 85.2% |
| pkg/router | 78.9% |

**Dependencies**: Phase 13

**Detailed Spec**: [PHASE_14.md](./PHASE_14.md)

---

## Phase 15: VangoUI Component System

**Goal**: CLI-distributed, server-first component library.

**Duration**: Components phase

**Status**: Planned

**Deliverables**:

### 15.1 CLI Integration
- [ ] `vango add init` - Initialize VangoUI
- [ ] `vango add <component>` - Download components
- [ ] Dependency resolution
- [ ] VS Code Tailwind IntelliSense config

### 15.2 Component Registry
- [ ] Component manifest with versions
- [ ] Hook dependencies
- [ ] `vango add --check` for updates

### 15.3 Styling System
- [ ] CSS variables for theming
- [ ] `CN()` utility for class merging
- [ ] Dark mode support

### 15.4 Functional Options API
- [ ] Type-safe component configuration
- [ ] Shared types (Variant, Size)
- [ ] Compile-time validation

### 15.5 Client Hook Protocol
- [ ] Standard hooks: Dialog, Popover, Sortable, Combobox
- [ ] Hook loader (on-demand)
- [ ] `Hook()` and `OnEvent()` vdom API

### 15.6 Component Library
- [ ] Primitives: Button, Badge, Label, Separator, Skeleton
- [ ] Form: Input, Textarea, Checkbox, Switch
- [ ] Layout: Card, Accordion, Tabs
- [ ] Interactive: Dialog, Dropdown, Popover
- [ ] Data: DataTable, Avatar, Progress

**Dependencies**: Phase 14

**Detailed Spec**: [PHASE_15.md](./PHASE_15.md)

---

## Phase 16: Unified Context & Platform Capabilities

**Goal**: Streaming, universal adaptation, and offline-first data.

**Duration**: Platform phase

**Status**: Planned (V2.1 Platform scope - may be deferred)

**Deliverables**:

### 16.1 Intelligent Streaming (`ctx.Async`)
- [ ] `ctx.Async().Render(fallback, component)` for slow data
- [ ] Async placeholder with streaming replacement
- [ ] Batch API for grouped operations

### 16.2 Universal Adaptation (`ctx.Client`)
- [ ] `ctx.Client().Platform()` - Web, iOS, Android, Desktop
- [ ] `ctx.Client().Capabilities()` - Touch, hover, haptics, etc.
- [ ] Capability negotiation in handshake
- [ ] Native component support

### 16.3 Offline Sync (`ctx.Sync`)
- [ ] `ctx.Sync().Resource(key)` - Offline-first data
- [ ] SQLite local storage (mobile)
- [ ] Background syncer with LWW conflict resolution
- [ ] Conflict UI support

### 16.4 Session Interface
- [ ] Enhanced `Session` interface (not pointer)
- [ ] `UserID`, `IsAuthenticated` helpers
- [ ] Metadata access (IP, UserAgent)

### 16.5 NATS JetStream Store
- [ ] Distributed session storage
- [ ] Horizontal scaling support

**Dependencies**: Phase 15

**Detailed Spec**: [PHASE_16.md](./PHASE_16.md)

## Milestone Checkpoints

### M1: Hello World (Phases 1-6)
**Definition**: A counter component that increments on click.
- Server renders initial HTML
- Client connects via WebSocket
- Click sends event to server
- Server updates signal, diffs, sends patch
- Client applies patch

**Validation**:
```go
func Counter() vango.Component {
    return vango.Func(func() *vango.VNode {
        count := vango.NewIntSignal(0)
        return Div(
            H1(Textf("Count: %d", count.Get())),
            Button(OnClick(func() { count.Inc() }), Text("+")),
        )
    })
}
```
This works end-to-end with < 15KB client.

### M2: Todo App (Phases 7-8.2)
**Definition**: Full CRUD todo list with routing.
- Multiple routes (`/`, `/todos`, `/todos/:id`)
- List rendering with keys
- Form for new todos
- Resource for data loading
- Persistence to database

### M3: Real-time Collaboration (Phase 8.6)
**Definition**: Multiple users see live updates.
- GlobalSignal for shared state
- Cursor presence indicators
- Optimistic updates with revert

### M4: Production Deployment (Phase 10)
**Definition**: Deployed to production with full observability.
- Handles 1000+ concurrent users
- < 100ms p99 latency
- Zero downtime deploys
- Full metrics and alerting

---

## Risk Register

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Binary protocol complexity | High | Medium | Extensive fuzz testing, fallback to JSON in dev |
| WebSocket scaling | High | Low | Horizontal scaling with sticky sessions |
| Memory per session | Medium | Medium | Session limits, LRU eviction |
| Browser compatibility | Medium | Low | Polyfills, feature detection |
| TinyGo limitations (if WASM) | High | N/A | V2 is server-driven only initially |

---

## Open Questions

1. **WASM Mode**: Should V2 include WASM compilation, or defer to V3?
   - **Recommendation**: Defer. Focus on server-driven excellence.

2. **Database Integration**: Should Vango include a query builder?
   - **Recommendation**: No. Integrate with sqlc, GORM, or raw sql.

3. **Authentication**: Built-in auth or bring-your-own?
   - **Recommendation**: Bring-your-own with session helpers.

4. **JavaScript Islands**: Include in V2 or defer?
   - **Recommendation**: Include. Critical for third-party libraries.

5. **Client Hooks**: Bundle SortableJS or require user installation?
   - **Recommendation**: Bundle standard hooks (~3KB).

---

## Success Criteria

Vango V2 is complete when:

1. **All 28 waiting companies** can deploy production apps
2. **Documentation** enables self-service onboarding
3. **Performance** meets or exceeds targets
4. **Security** passes third-party audit
5. **Community** can contribute (clear architecture, good tests)

---

## Appendix: File Structure

```
vango_v2/
├── cmd/
│   └── vango/              # CLI tool
│       ├── main.go         # CLI entry point
│       ├── create.go       # Project scaffolding
│       ├── dev.go          # Development server
│       ├── build.go        # Production build
│       ├── gen.go          # Code generation
│       ├── add.go          # VangoUI components
│       ├── test.go         # Test runner
│       └── version.go      # Version info
├── internal/               # CLI internals
│   ├── errors/             # Structured error messages
│   │   ├── error.go        # VangoError type
│   │   ├── format.go       # Terminal formatting
│   │   └── registry.go     # Error code registry
│   ├── config/             # Configuration
│   │   └── config.go       # vango.json parsing
│   ├── dev/                # Development server
│   │   ├── watcher.go      # File watching
│   │   ├── compiler.go     # Go compilation
│   │   ├── reload.go       # Browser reload
│   │   ├── tailwind.go     # CSS processing
│   │   └── server.go       # Dev server
│   ├── build/              # Production build
│   │   └── builder.go      # Build pipeline
│   ├── templates/          # Project templates
│   │   └── templates.go    # Scaffolding
│   └── registry/           # Component registry
│       └── registry.go     # VangoUI management
├── pkg/
│   ├── vango/              # Core framework
│   │   ├── signal.go       # Reactive primitives
│   │   ├── memo.go
│   │   ├── effect.go
│   │   ├── component.go
│   │   ├── context.go
│   │   └── vango.go        # Public API
│   ├── vdom/               # Virtual DOM
│   │   ├── vnode.go
│   │   ├── diff.go
│   │   ├── patch.go
│   │   └── elements.go     # Generated
│   ├── protocol/           # Binary protocol
│   │   ├── event.go
│   │   ├── patch.go
│   │   ├── varint.go
│   │   └── codec.go
│   ├── server/             # Server runtime
│   │   ├── session.go
│   │   ├── manager.go
│   │   ├── handler.go
│   │   └── websocket.go
│   ├── render/             # SSR
│   │   ├── html.go
│   │   └── hydration.go
│   ├── router/             # Routing
│   │   ├── tree.go
│   │   ├── match.go
│   │   └── codegen.go
│   └── features/           # Higher-level
│       ├── form.go
│       ├── resource.go
│       ├── context.go
│       ├── urlstate.go
│       └── hooks.go
├── client/                 # Thin client (JS)
│   ├── src/
│   │   ├── index.js
│   │   ├── websocket.js
│   │   ├── events.js
│   │   ├── patches.js
│   │   └── hooks/
│   │       ├── sortable.js
│   │       └── tooltip.js
│   └── dist/
│       └── vango.min.js
├── docs/                   # Build documentation
│   ├── BUILD_ROADMAP.md
│   ├── PHASE_*.md
│   ├── DECISIONS.md
│   └── V1_LESSONS.md
├── examples/               # Example apps
│   ├── counter/
│   ├── todo/
│   └── realtime/
├── test/                   # Test suites
│   ├── integration/
│   └── e2e/
├── go.mod
├── go.sum
└── README.md
```

---

*Last Updated: 2024-12-24*
*Version: 2.1-draft*
*Phases 1-14 Complete, Phase 15 Next*
