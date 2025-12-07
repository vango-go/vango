# V1 Lessons Learned

> **What we learned from Vango V1 to inform V2 architecture**

---

## Executive Summary

Vango V1 was an ambitious alpha (~5,200 lines of Go) that proved the core concepts work but revealed significant integration gaps and architectural tensions. This document captures the lessons learned to ensure V2 doesn't repeat mistakes.

---

## What Worked Well

### 1. Single VNode Architecture

**Decision**: One VNode structure serves SSR, hydration, and client rendering.

**Why it worked**:
- Eliminated SSR/client parity bugs
- Simplified diff algorithm (one source of truth)
- Same appliers (HTML and DOM) consume the same patch stream

**Carry forward to V2**: Yes, this is the right approach.

### 2. Cooperative Scheduler with Fibers

**Decision**: Single-threaded fiber scheduler instead of goroutines.

**Why it worked**:
- TinyGo WASM can't create OS threads anyway
- Lower memory overhead (4KB per fiber vs 2MB per goroutine)
- Predictable scheduling (no race conditions from concurrent renders)
- Panic recovery per-fiber prevents cascade failures

**Carry forward to V2**: Yes, but V2 is server-driven first, so we use goroutines per session instead. The fiber concept informs our component execution model.

### 3. Binary Protocol Design

**Decision**: Live protocol uses binary frames, not JSON.

**Why it worked**:
- 5-10x smaller payloads (43 bytes JSON vs 6 bytes binary for typical patch)
- Lower latency for real-time updates
- Efficient varint encoding for common small values

**Carry forward to V2**: Yes, with improvements (see Protocol Issues below).

### 4. Structured Logging with slog

**Decision**: Use Go's slog package throughout.

**Why it worked**:
- Consistent log format
- Structured fields for debugging
- Easy to integrate with observability tools

**Carry forward to V2**: Yes.

### 5. Interface-Based Design

**Decision**: Key abstractions defined as interfaces (`Ctx`, `Middleware`, `Component`).

**Why it worked**:
- Easy to mock in tests
- Allows multiple implementations
- Clear contracts between subsystems

**Carry forward to V2**: Yes.

---

## What Didn't Work

### 1. Context Type Confusion

**Problem**: Two different "Context" types that serve overlapping purposes.

```go
// V1 had both:
type vango.Context struct {
    Props      map[string]interface{}
    Params     map[string]string
    Query      map[string]string
    Fiber      *scheduler.Fiber
    Scheduler  *scheduler.Scheduler
    Mode       RenderMode
    SessionID  string
    Data       map[string]interface{}
}

type server.Ctx interface {
    Request() *http.Request
    Path() string
    Method() string
    Query() url.Values
    Param(key string) string
    // ... HTTP response methods
    Session() Session
}
```

**Why it failed**:
- Routes receive `server.Ctx`, components receive `vango.Context`
- Mapping between them is unclear and lossy
- Developers confused about which to use where
- Data passed awkwardly between layers

**V2 Fix**: Single unified `vango.Ctx` that:
- Wraps `*http.Request` and `http.ResponseWriter`
- Contains session, signals, component scope
- Flows through entire request→render→response pipeline
- One type to learn, one type to pass

### 2. Protocol Frame Type Bug

**Problem**: Critical bug where client checked wrong frame type constant.

```javascript
// V1 Bug: Client was checking 0x02 instead of 0x00 for patches
if (frameType === 0x02) { // WRONG!
    applyPatches(data);
}

// Should have been:
if (frameType === 0x00) { // FramePatches
    applyPatches(data);
}
```

**Why it happened**:
- Constants defined in Go, manually mirrored in JS
- No automated validation between Go and JS constants
- No integration test that verified end-to-end message flow

**V2 Fix**:
- Generate JS constants from Go definitions
- Integration tests that verify actual wire bytes
- Fuzz testing on protocol decoder
- Protocol version negotiation in handshake

### 3. Component Instance Not in Context

**Problem**: Component instance wasn't stored in context before render, causing handler lookup failures.

**Why it happened**:
- Order of operations wasn't clear
- No explicit lifecycle documentation
- Missing assertions in debug mode

**V2 Fix**:
- Explicit component lifecycle with documented phases
- Debug assertions that verify state at each phase
- Integration tests for full request cycle

### 4. Double-Increment Bug

**Problem**: Both WASM and server handlers were firing for the same event.

**Why it happened**:
- Hybrid mode tried to run handlers in both places
- No clear ownership model for who handles what
- Event routing logic was ad-hoc

**V2 Fix**:
- V2 is server-driven first (no WASM by default)
- Clear ownership: server handles all events
- Client hooks are explicit opt-in for 60fps interactions
- WASM mode (if added later) is completely separate path

### 5. Props Comparison with fmt.Sprintf

**Problem**: Props equality used `fmt.Sprintf("%v", ...)` for comparison.

```go
// V1 approach (crude):
func propsEqual(a, b any) bool {
    return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}
```

**Why it failed**:
- Slow (allocates strings for every comparison)
- Doesn't handle all types correctly
- Deep equality for slices/maps is wrong

**V2 Fix**:
- Type-specific equality functions
- Use `reflect.DeepEqual` only for complex types
- Allow custom equality via `.Equals()` method
- Consider using hash-based comparison for large props

### 6. Incomplete Reactive System

**Problem**: `State[T]` and basic `Computed[T]` worked, but missing:
- `GlobalSignal[T]` for cross-session state
- `Resource[T]` with proper loading states
- Persistence layer
- Explicit dependency tracking for Computed

**Why it happened**:
- Phase 0 focused on primitives, Phase 1 on integration
- Advanced reactive features deferred but never built

**V2 Fix**:
- Complete reactive system in Phase 1
- All signal types specified upfront
- Resource is a first-class concept, not an afterthought

### 7. No Component Lifecycle Hooks

**Problem**: Components had no `OnMount`, `OnUnmount`, `OnUpdate` hooks.

**Why it failed**:
- Couldn't manage subscriptions/cleanup
- No way to run side effects on state changes
- Effects were half-implemented

**V2 Fix**:
- `Effect(fn)` with cleanup return
- `OnMount(fn)` convenience wrapper
- `OnUnmount(fn)` convenience wrapper
- Clear lifecycle documentation

### 8. Router Code Generator Missing

**Problem**: Route scanner/generator was planned but never implemented.

**Why it failed**:
- File-based routing requires code generation
- Without it, routes had to be registered manually
- Manual registration defeats the purpose

**V2 Fix**:
- `vango gen routes` command
- Scans `app/routes/**/*.go`
- Generates type-safe route tree
- Runs automatically in dev mode

### 9. Template Codegen Untested

**Problem**: VEX template parser existed but code generation was never verified.

```go
//vango:template
<div class="card">
  <h1>{{.Title}}</h1>
  {{#if len(.Items) > 0}}
    <ul>{{#for item in .Items}}<li>{{item}}</li>{{/for}}</ul>
  {{/if}}
</div>
```

**Why it failed**:
- Parser grammar was complete
- But output never tested against real components
- Event handler wiring unclear
- Nested component support unknown

**V2 Fix**:
- For V2, skip templates initially
- Pure Go components are the primary API
- Templates can be added later if there's demand
- Focus on making Go syntax ergonomic

### 10. Styling System Incomplete

**Problem**: Scoped CSS extraction and class hashing never implemented.

**Why it failed**:
- Build-time CSS extraction requires AST parsing
- Class name hashing requires coordination between Go and CSS
- Tailwind integration was partial

**V2 Fix**:
- Tailwind is the recommended styling approach
- Global CSS works out of the box
- Scoped styles can be added later
- Critical CSS extraction deferred

---

## Integration Gaps

### Phase 0 → Phase 1 Disconnect

**Problem**: Phase 0 built isolated primitives that didn't connect.

| Component | Status | Integration |
|-----------|--------|-------------|
| Diff engine | Working | No applier wired |
| Scheduler | Working | No component lifecycle |
| Live protocol | Types defined | Bridge incomplete |
| Router | Basic tree | Code generator missing |
| Templates | Parser works | Codegen untested |

**V2 Fix**:
- Milestone checkpoints require end-to-end tests
- M1 (Hello World) must work before moving to Phase 2
- Each phase has integration tests, not just unit tests

### Testing Gaps

**Problem**: Unit tests existed but integration coverage was poor.

| Area | Unit Tests | Integration Tests |
|------|------------|-------------------|
| VNode diff | ✓ | ✗ |
| Reactive signals | ✓ | ✗ |
| Scheduler | ✓ | ✗ |
| Live protocol | ✗ | ✗ |
| Routing | Partial | ✗ |
| Templates | ✓ | ✗ |
| Hydration | ✗ | ✗ |

**V2 Fix**:
- Integration tests are mandatory for each phase
- E2E tests (Playwright) for user flows
- Fuzz tests for parsers and protocol
- Coverage requirements: 80% unit, key paths E2E

---

## Architectural Tensions

### 1. SSR vs Client-First

**Tension**: V1 tried to support all modes equally:
- SSR-only (zero client JS)
- SSR + hydration (partial interactivity)
- Client-only WASM (SPA mode)
- Server-driven live patches

**Result**: All implemented in skeleton form, none well-integrated.

**V2 Resolution**:
- **Server-driven is the default and primary mode**
- SSR happens automatically (it's how server-driven works)
- WASM is explicitly out of scope for V2
- Client hooks provide 60fps when needed
- JS Islands integrate third-party libraries

### 2. Single Binary vs Module System

**Tension**: Should components be compiled into one binary or loaded as modules?

**V1 Approach**: Single binary (Go's model)

**Implication**:
- Hot reload requires full recompilation
- But Go compiles fast, so this is acceptable
- Plugin system not needed for V2

**V2 Resolution**: Single binary is fine. Incremental compilation makes hot reload fast enough.

### 3. Type Safety vs Flexibility

**Tension**: Strict types vs dynamic props

**V1 Approach**: `Props map[string]any` for flexibility

**Problems**:
- No compile-time checking of prop names
- Easy to typo attribute names
- Runtime errors instead of compile errors

**V2 Resolution**:
- Element functions accept typed attributes
- `Div(Class("foo"), ID("bar"))` not `Div(Props{"class": "foo"})`
- Event handlers are typed: `OnClick(func())` not `Props{"onclick": fn}`
- Props map still exists internally, but API is typed

---

## Developer Experience Lessons

### 1. First Run Must Be Perfect

**Learning**: `vango create` must produce a working app in < 60 seconds with zero manual steps.

**V1 Issues**:
- Sometimes required manual `go mod tidy`
- Tailwind setup was flaky
- Missing files caused confusing errors

**V2 Fix**:
- Extensive testing of create flow
- All dependencies pre-resolved
- Smoke test runs automatically after create

### 2. Errors Are Teaching Moments

**Learning**: Error messages should explain what went wrong AND how to fix it.

**V1 Issues**:
- Generic Go errors ("nil pointer dereference")
- No context about which component or signal
- No suggestions for fixes

**V2 Fix**:
- Custom error types with error codes
- Source location in every error
- Suggested fixes with code examples
- Links to documentation

### 3. Examples > Documentation

**Learning**: Nobody reads docs; everyone copies examples.

**V1 Issues**:
- Docs existed but were out of date
- Examples were minimal
- Cookbook missing

**V2 Fix**:
- Examples are first-class (tested, versioned)
- Every feature has a working example
- Cookbook covers common patterns
- Examples linked from API docs

### 4. Styling Must Be Zero-Config

**Learning**: Developers expect Tailwind to "just work".

**V1 Issues**:
- Required manual Tailwind setup
- Config file needed editing
- Watch mode was separate process

**V2 Fix**:
- `vango create` sets up Tailwind automatically
- `vango dev` runs Tailwind watch internally
- Zero configuration needed for basic use

---

## Platform-Specific Issues

### macOS
- File watcher ulimit too low (need `ulimit -n 2048`)
- Tailwind via Homebrew has different binary paths
- Port 5173 conflicts with Vite (use 3000)

### Linux
- TinyGo requires manual installation
- Some distros need CGO_ENABLED=1

### Windows
- Path separators break glob patterns
- File watching is unreliable on network drives
- Line endings cause issues in templates

**V2 Fix**:
- Document platform requirements
- CLI checks prerequisites
- Path handling uses `filepath` consistently
- Tests run on all platforms in CI

---

## Performance Observations

### Memory Per Session (V1)

| App Complexity | Observed Memory |
|----------------|-----------------|
| Counter | 15 KB |
| Todo list | 45 KB |
| Dashboard | 180 KB |
| Complex form | 250 KB |

These numbers are acceptable. With 32 GB RAM, can handle 100k+ concurrent users.

### Latency (V1)

| Operation | Observed |
|-----------|----------|
| Event → Handler | 1-2ms |
| Diff (simple) | < 1ms |
| Diff (complex) | 2-5ms |
| WebSocket send | 1ms |
| Total round-trip | 50-80ms |

Acceptable for most use cases. Optimistic updates make it feel instant.

### Compilation Speed

| Change Type | Rebuild Time |
|-------------|--------------|
| Single file | 40-80ms |
| Multiple files | 100-200ms |
| Full rebuild | 1-3s |

Fast enough for good hot reload experience.

---

## Key Decisions for V2

Based on V1 lessons, these decisions are locked for V2:

| Decision | Rationale |
|----------|-----------|
| Server-driven first | Simpler model, WASM adds complexity |
| Single VNode type | Proven to work, eliminates parity bugs |
| Binary protocol | Performance validated in V1 |
| Unified Ctx type | Fixes V1's biggest pain point |
| Go syntax for components | Templates add complexity, Go is fine |
| Tailwind for styling | Zero-config, widely adopted |
| Goroutines per session | Simpler than fibers for server |
| No WASM in V2 | Focus on server-driven excellence |
| Integration tests mandatory | Prevents V1's integration gaps |

---

## What to Build First

Based on V1 lessons, the build order for V2:

1. **Reactive core** - Get signals/memos/effects rock solid
2. **VNode + diff** - With proper equality handling
3. **Protocol** - With generated constants and fuzz tests
4. **Server runtime** - Session management, event loop
5. **Thin client** - With protocol verification tests
6. **SSR** - HTML rendering with hydration IDs
7. **Routing** - With code generator from day one
8. **Higher features** - Forms, resources, hooks
9. **DX** - CLI, hot reload, error messages
10. **Production** - Security, observability, deployment

Each phase must have:
- Unit tests (80%+ coverage)
- Integration tests (key paths)
- Documentation (API + examples)
- Working example app

---

## Conclusion

V1 proved that server-driven UI in Go is viable and performant. The core concepts (single VNode, binary protocol, reactive signals) are sound. The failures were in integration, lifecycle management, and developer experience.

V2 will succeed by:
1. **Finishing what V1 started** - No half-implemented features
2. **Testing integration points** - Unit tests aren't enough
3. **Unifying concepts** - One Ctx, one component model
4. **Prioritizing DX** - Errors, docs, examples
5. **Shipping server-driven only** - WASM is a distraction

The 28 companies waiting for Vango deserve a framework that works out of the box. V2 will deliver that.

---

*V1 Lessons Document - Version 1.0*
*Based on analysis of Vango V1 codebase at /Users/collinshill/Documents/vango/vango/*
