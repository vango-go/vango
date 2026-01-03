# Phase 13: Production Hardening & Observability

> **Security gates, protocol defense, and production-grade monitoring**

---

## Overview

Phase 13 transforms Vango from a "works in development" framework to a "production-grade platform." This phase implements the observability infrastructure required for debugging distributed systems and hardens the protocol against sophisticated attacks. No production deployment should proceed without completing this phase.

### Core Philosophy

**Secure by default. Observable by design. Production-ready out of the box.**

### Goals

1. **Observability**: OpenTelemetry middleware for distributed tracing
2. **Protocol Defense**: Allocation limits, depth limits, fuzz testing
3. **Session Security**: Per-IP limits, CSRF enforcement, secure cookies
4. **Metrics**: Prometheus-compatible metrics for dashboards

### Non-Goals (Explicit Exclusions)

1. Custom logging framework (use standard `log/slog`)
2. APM vendor lock-in (OTel is vendor-neutral)
3. Proprietary security scanner (use external tools)

### Decision: No `ctx.Trace()`

After analysis, we explicitly reject adding `ctx.Trace()` as an API. Instead, all tracing happens via middleware injection. This keeps the component API clean and follows OTel best practices.

---

## Subsystems

| Subsystem | Purpose | Priority |
|-----------|---------|----------|
| 13.1 OpenTelemetry Middleware | Distributed tracing for every event | Critical |
| 13.2 Context Propagation | Trace IDs flow to DB calls | Critical |
| 13.3 Protocol Depth Limits | Prevent stack overflow attacks | Critical |
| 13.4 Protocol Allocation Audit | Verify all paths are capped | Critical |
| 13.5 Fuzz Testing Suite | Automated protocol fuzzing | High |
| 13.6 Secure Defaults Enforcement | CSRF, Origin, Cookie settings | High |
| 13.7 Prometheus Metrics | Runtime observability | Medium |

---

## 13.1 OpenTelemetry Middleware

### Problem

Debugging production issues requires understanding the flow of requests through the system. Without tracing, correlating client events with server-side errors is impossible.

### Solution

Middleware-first OpenTelemetry integration that automatically traces every WebSocket event.

### Philosophy: Middleware-First

```
❌ REJECTED: ctx.Trace("operation-name")
✅ ADOPTED:  Middleware automatically starts span for every event
```

**Why Middleware-First?**
1. Zero boilerplate in component code
2. Consistent span naming and attributes
3. Automatic context propagation
4. No API surface to learn

### Middleware Implementation

```go
// pkg/middleware/otel.go

package middleware

import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
    "go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("vango")

// OpenTelemetry creates a middleware that traces every Vango event.
func OpenTelemetry(opts ...OTelOption) vango.Middleware {
    config := defaultOTelConfig()
    for _, opt := range opts {
        opt(&config)
    }
    
    return func(ctx vango.Ctx, next func() error) error {
        event := ctx.Event()
        
        // Create span with event details
        spanCtx, span := tracer.Start(
            ctx.StdContext(),
            formatSpanName(event),
            trace.WithSpanKind(trace.SpanKindServer),
            trace.WithAttributes(
                attribute.String("vango.session_id", ctx.Session().ID),
                attribute.String("vango.event_type", event.Type),
                attribute.String("vango.event_target", event.TargetHID),
                attribute.String("vango.route", ctx.Route()),
            ),
        )
        defer span.End()
        
        // Inject trace context for downstream calls
        ctx = ctx.WithStdContext(spanCtx)
        
        // Execute handler
        err := next()
        
        // Record result
        if err != nil {
            span.RecordError(err)
            span.SetStatus(codes.Error, err.Error())
        } else {
            span.SetStatus(codes.Ok, "")
        }
        
        // Record patch count
        span.SetAttributes(
            attribute.Int("vango.patch_count", ctx.PatchCount()),
        )
        
        return err
    }
}

func formatSpanName(event *protocol.Event) string {
    switch event.Type {
    case protocol.EventClick:
        return "vango.click"
    case protocol.EventInput:
        return "vango.input"
    case protocol.EventNavigation:
        return "vango.navigate"
    case protocol.EventHook:
        return fmt.Sprintf("vango.hook.%s", event.HookName)
    default:
        return fmt.Sprintf("vango.event.%d", event.Type)
    }
}

type OTelConfig struct {
    // Include user ID in traces (may be sensitive)
    IncludeUserID bool
    
    // Custom attribute extractor
    AttributeExtractor func(vango.Ctx) []attribute.KeyValue
    
    // Filter events that shouldn't be traced
    Filter func(vango.Ctx) bool
}

func defaultOTelConfig() OTelConfig {
    return OTelConfig{
        IncludeUserID: false,
        Filter:        func(ctx vango.Ctx) bool { return true },
    }
}
```

### Server Integration

```go
// main.go

func main() {
    // Initialize OpenTelemetry (standard setup)
    tp := initTracerProvider()
    defer tp.Shutdown(context.Background())
    
    app := vango.NewApp(
        vango.WithMiddleware(
            middleware.OpenTelemetry(
                middleware.IncludeUserID(true),
            ),
            middleware.Logger(),
            middleware.Recover(),
        ),
    )
    
    // ...
}

func initTracerProvider() *sdktrace.TracerProvider {
    exporter, _ := otlptracehttp.New(context.Background())
    
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceName("my-vango-app"),
            semconv.ServiceVersion("1.0.0"),
        )),
    )
    
    otel.SetTracerProvider(tp)
    return tp
}
```

---

## 13.2 Context Propagation

### Problem

Database queries made during event handling should inherit the trace context, allowing correlation between Vango events and slow queries.

### Solution

`ctx.StdContext()` returns a `context.Context` with trace information, compatible with all Go database drivers.

### Implementation

```go
// pkg/vango/ctx.go

// StdContext returns the standard library context with trace propagation.
// Use this when calling external services or database drivers.
func (c *Ctx) StdContext() context.Context {
    return c.stdCtx
}

// WithStdContext returns a new Ctx with an updated standard context.
// Used by middleware to inject trace spans.
func (c *Ctx) WithStdContext(stdCtx context.Context) vango.Ctx {
    newCtx := *c
    newCtx.stdCtx = stdCtx
    return &newCtx
}
```

### Usage in Components

```go
func UserProfile(ctx vango.Ctx, userID int) *vango.VNode {
    // Database call inherits trace context
    user, err := db.Users.FindByID(ctx.StdContext(), userID)
    if err != nil {
        // Error is automatically recorded in span
        return ErrorComponent(err)
    }
    
    // External API call also inherits trace context
    avatar, _ := avatarService.Get(ctx.StdContext(), user.AvatarID)
    
    return Div(
        // ...
    )
}
```

### Database Driver Integration

```go
// Most database drivers automatically propagate context:

// database/sql
row := db.QueryRowContext(ctx.StdContext(), "SELECT * FROM users WHERE id = $1", userID)

// pgx
row := pool.QueryRow(ctx.StdContext(), "SELECT * FROM users WHERE id = $1", userID)

// gorm
db.WithContext(ctx.StdContext()).First(&user, userID)

// ent
client.User.Query().Where(user.ID(userID)).First(ctx.StdContext())
```

### Trace Correlation

With proper OTel setup, you'll see traces like:

```
vango.click (2.3ms)
├── db.query (1.8ms) - SELECT * FROM users WHERE id = $1
├── http.client (0.4ms) - GET /api/avatar/123
└── vango.render (0.1ms)
```

---

## 13.3 Protocol Depth Limits

### Problem

Deeply nested JSON in hook payloads can cause stack overflow during decode. Phase 11 added `MaxHookDepth`, but we need comprehensive depth limiting across all protocol decoding.

### Solution

Add depth tracking to all recursive decode functions.

### Constants

```go
// pkg/protocol/limits.go

const (
    // MaxHookDepth limits nesting in hook event payloads
    MaxHookDepth = 64
    
    // MaxVNodeDepth limits nesting in VNode trees
    MaxVNodeDepth = 256
    
    // MaxPatchDepth limits nesting in patch structures
    MaxPatchDepth = 128
)

var (
    ErrMaxDepthExceeded = errors.New("protocol: maximum nesting depth exceeded")
)
```

### VNode Decode Depth Tracking

```go
// pkg/protocol/vnode.go

func DecodeVNode(dec *Decoder) (*VNode, error) {
    return decodeVNodeWithDepth(dec, 0)
}

func decodeVNodeWithDepth(dec *Decoder, depth int) (*VNode, error) {
    if depth > MaxVNodeDepth {
        return nil, ErrMaxDepthExceeded
    }
    
    node := &VNode{}
    
    // Decode tag
    tag, err := dec.ReadString()
    if err != nil {
        return nil, err
    }
    node.Tag = tag
    
    // Decode children count with limit
    childCount, err := dec.ReadCollectionCount()
    if err != nil {
        return nil, err
    }
    
    // Decode children recursively
    node.Children = make([]*VNode, 0, childCount)
    for i := 0; i < childCount; i++ {
        child, err := decodeVNodeWithDepth(dec, depth+1)
        if err != nil {
            return nil, err
        }
        node.Children = append(node.Children, child)
    }
    
    return node, nil
}
```

### Patch Decode Depth Tracking

```go
// pkg/protocol/patch.go

func DecodePatch(dec *Decoder) (*Patch, error) {
    return decodePatchWithDepth(dec, 0)
}

func decodePatchWithDepth(dec *Decoder, depth int) (*Patch, error) {
    if depth > MaxPatchDepth {
        return nil, ErrMaxDepthExceeded
    }
    
    patchType, err := dec.ReadByte()
    if err != nil {
        return nil, err
    }
    
    switch patchType {
    case PatchInsertNode:
        // Decode node with depth tracking
        node, err := decodeVNodeWithDepth(dec, depth+1)
        if err != nil {
            return nil, err
        }
        return &Patch{Type: PatchInsertNode, Node: node}, nil
        
    // ... other cases
    }
}
```

---

## 13.4 Protocol Allocation Audit

### Problem

Phase 11 added allocation limits, but we need to verify ALL code paths are protected. A single missed path is a DoS vector.

### Audit Checklist

| Location | Protected | Method |
|----------|-----------|--------|
| `decoder.ReadString()` | ✅ | Checks `length <= DefaultMaxAllocation` |
| `decoder.ReadLenBytes()` | ✅ | Checks `length <= DefaultMaxAllocation` |
| `event.DecodeEvent()` | ✅ | Uses `ReadCollectionCount()` for all collections |
| `patch.DecodePatch()` | ✅ | Uses `ReadCollectionCount()` for patch arrays |
| `vnode.DecodeVNode()` | ✅ | Uses `ReadCollectionCount()` for children/attrs |
| `control.DecodeControl()` | ✅ | Uses `ReadCollectionCount()` for resync patches |
| Hook payload JSON | ✅ | Uses `MaxHookDepth` in recursive decode |

### Test Cases

```go
// pkg/protocol/security_test.go

func TestAllocationLimits(t *testing.T) {
    tests := []struct {
        name    string
        payload []byte
        wantErr error
    }{
        {
            name:    "string too large",
            payload: makeOversizedString(5 * 1024 * 1024), // 5MB
            wantErr: ErrAllocationTooLarge,
        },
        {
            name:    "collection too large",
            payload: makeOversizedCollection(200_000),
            wantErr: ErrCollectionTooLarge,
        },
        {
            name:    "depth too deep",
            payload: makeDeeplyNested(300),
            wantErr: ErrMaxDepthExceeded,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            dec := NewDecoder(tt.payload)
            _, err := DecodeEvent(dec)
            assert.ErrorIs(t, err, tt.wantErr)
        })
    }
}
```

---

## 13.5 Fuzz Testing Suite

### Problem

Manual testing can't cover all edge cases. Protocol parsers are notorious for having subtle bugs that fuzzing can find.

### Solution

Go's native fuzzing for all protocol decode functions.

### Fuzz Targets

```go
// pkg/protocol/fuzz_test.go

func FuzzDecodeEvent(f *testing.F) {
    // Seed corpus with valid events
    f.Add(validClickEvent)
    f.Add(validInputEvent)
    f.Add(validHookEvent)
    
    f.Fuzz(func(t *testing.T, data []byte) {
        dec := NewDecoder(data)
        
        // Should not panic
        _, err := DecodeEvent(dec)
        
        // Should return error for invalid data, not panic
        if err != nil {
            return // Expected for random data
        }
    })
}

func FuzzDecodePatch(f *testing.F) {
    f.Add(validInsertPatch)
    f.Add(validRemovePatch)
    f.Add(validAttrPatch)
    
    f.Fuzz(func(t *testing.T, data []byte) {
        dec := NewDecoder(data)
        _, _ = DecodePatch(dec)
    })
}

func FuzzDecodeVNode(f *testing.F) {
    f.Add(validVNodeBytes)
    
    f.Fuzz(func(t *testing.T, data []byte) {
        dec := NewDecoder(data)
        _, _ = DecodeVNode(dec)
    })
}

func FuzzDecodeHookPayload(f *testing.F) {
    f.Add([]byte(`{"key": "value"}`))
    f.Add([]byte(`{"nested": {"deep": {"deeper": true}}}`))
    
    f.Fuzz(func(t *testing.T, data []byte) {
        _, _ = decodeHookValue(data, 0)
    })
}
```

### CI Integration

```yaml
# .github/workflows/fuzz.yml

name: Fuzz Testing

on:
  schedule:
    - cron: '0 0 * * *'  # Daily
  workflow_dispatch:

jobs:
  fuzz:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      
      - name: Fuzz DecodeEvent
        run: go test -fuzz=FuzzDecodeEvent -fuzztime=5m ./pkg/protocol/
      
      - name: Fuzz DecodePatch
        run: go test -fuzz=FuzzDecodePatch -fuzztime=5m ./pkg/protocol/
      
      - name: Fuzz DecodeVNode
        run: go test -fuzz=FuzzDecodeVNode -fuzztime=5m ./pkg/protocol/
      
      - name: Upload crashers
        if: failure()
        uses: actions/upload-artifact@v4
        with:
          name: fuzz-crashers
          path: testdata/fuzz/
```

### Fuzz Corpus Management

```
pkg/protocol/testdata/fuzz/
├── FuzzDecodeEvent/
│   ├── corpus/           # Seed inputs
│   │   ├── click_event
│   │   ├── input_event
│   │   └── hook_event
│   └── crashers/         # Failing inputs (gitignored)
├── FuzzDecodePatch/
│   └── corpus/
└── FuzzDecodeVNode/
    └── corpus/
```

---

## 13.6 Secure Defaults Enforcement

### Problem

Developers may forget to configure security settings. Insecure defaults lead to vulnerable applications.

### Solution

Secure defaults with explicit opt-out for development.

### Default Configuration

```go
// pkg/server/config.go

func defaultConfig() Config {
    return Config{
        // Security: ENABLED by default
        CheckOrigin:    SameOriginCheck,        // Reject cross-origin WS
        CSRFProtection: true,                    // Require CSRF tokens
        SecureCookies:  true,                    // HTTPS-only cookies
        
        // Session limits
        MaxSessionsPerIP:     100,
        MaxDetachedSessions:  10000,
        
        // Protocol limits
        MaxAllocation:        4 * 1024 * 1024,   // 4MB
        MaxCollectionCount:   100_000,
        MaxVNodeDepth:        256,
        
        // Development mode (must be explicitly enabled)
        DevMode: false,
    }
}
```

### Development Mode

```go
// For development only - explicitly disables security checks
app := vango.NewApp(
    vango.WithDevMode(), // Logs warning on startup
)

// This does:
// - Sets CheckOrigin to allow all
// - Disables CSRF validation
// - Allows insecure cookies
// - Enables debug logging
```

### Startup Validation

```go
// pkg/server/server.go

func (s *Server) validateConfig() error {
    if !s.config.DevMode {
        // Production mode validations
        
        if s.config.CSRFSecret == nil {
            log.Warn("CSRF protection enabled but CSRFSecret not set. Generate with: vango secret")
            // Generate a random secret for this instance
            s.config.CSRFSecret = generateRandomSecret()
        }
        
        if !s.config.SecureCookies && s.isTLS() {
            log.Warn("TLS enabled but SecureCookies disabled. Enabling SecureCookies.")
            s.config.SecureCookies = true
        }
    } else {
        log.Warn("⚠️  DEV MODE ENABLED - Security checks disabled. DO NOT USE IN PRODUCTION.")
    }
    
    return nil
}
```

### Cookie Configuration

```go
// pkg/server/cookies.go

func (s *Server) sessionCookie(sessionID string) *http.Cookie {
    cookie := &http.Cookie{
        Name:     "__vango_session",
        Value:    sessionID,
        Path:     "/",
        HttpOnly: true,  // Always prevent JS access
        SameSite: http.SameSiteLaxMode,
    }
    
    if s.config.SecureCookies {
        cookie.Secure = true
    }
    
    if s.config.CookieDomain != "" {
        cookie.Domain = s.config.CookieDomain
    }
    
    return cookie
}
```

---

## 13.7 Prometheus Metrics

### Problem

Operations teams need visibility into Vango's runtime behavior: connection counts, event rates, memory usage.

### Solution

Optional Prometheus metrics middleware.

### Metrics Definition

```go
// pkg/middleware/metrics.go

package middleware

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    activeSessions = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "vango_active_sessions",
        Help: "Number of active WebSocket sessions",
    })
    
    detachedSessions = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "vango_detached_sessions",
        Help: "Number of detached (disconnected but resumable) sessions",
    })
    
    eventsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "vango_events_total",
        Help: "Total number of events processed",
    }, []string{"type"})
    
    eventDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "vango_event_duration_seconds",
        Help:    "Event processing duration in seconds",
        Buckets: prometheus.DefBuckets,
    }, []string{"type"})
    
    patchesSent = promauto.NewCounter(prometheus.CounterOpts{
        Name: "vango_patches_sent_total",
        Help: "Total number of patches sent to clients",
    })
    
    sessionMemoryBytes = promauto.NewHistogram(prometheus.HistogramOpts{
        Name:    "vango_session_memory_bytes",
        Help:    "Estimated memory usage per session",
        Buckets: []float64{1024, 10240, 102400, 1048576, 10485760},
    })
    
    wsErrors = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "vango_websocket_errors_total",
        Help: "Total WebSocket errors by type",
    }, []string{"type"})
)
```

### Metrics Middleware

```go
// pkg/middleware/metrics.go

func Prometheus() vango.Middleware {
    return func(ctx vango.Ctx, next func() error) error {
        eventType := ctx.Event().TypeString()
        
        // Record event
        eventsTotal.WithLabelValues(eventType).Inc()
        
        // Time execution
        start := time.Now()
        err := next()
        duration := time.Since(start).Seconds()
        
        eventDuration.WithLabelValues(eventType).Observe(duration)
        
        // Record patches
        patchesSent.Add(float64(ctx.PatchCount()))
        
        return err
    }
}
```

### Server Metrics Hooks

```go
// pkg/server/server.go

func (s *Server) onSessionCreate(sess *Session) {
    activeSessions.Inc()
    metrics.RecordSessionCreate(sess.IP)
}

func (s *Server) onSessionDestroy(sess *Session) {
    activeSessions.Dec()
    sessionMemoryBytes.Observe(float64(sess.EstimatedMemory()))
}

func (s *Server) onSessionDetach(sess *Session) {
    activeSessions.Dec()
    detachedSessions.Inc()
}

func (s *Server) onSessionReattach(sess *Session) {
    activeSessions.Inc()
    detachedSessions.Dec()
}
```

### Metrics Endpoint

```go
// main.go

import (
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
    mux := http.NewServeMux()
    
    // Metrics endpoint (typically behind auth or internal network)
    mux.Handle("/metrics", promhttp.Handler())
    
    // Vango app
    mux.Handle("/", app.Handler())
    
    http.ListenAndServe(":8080", mux)
}
```

### Grafana Dashboard

A sample Grafana dashboard JSON is provided in `examples/monitoring/grafana-dashboard.json` with panels for:

1. **Active Sessions** - Real-time connection count
2. **Event Rate** - Events per second by type
3. **Event Latency** - P50/P95/P99 event duration
4. **Memory Usage** - Per-session memory distribution
5. **Patch Rate** - Patches sent per second
6. **Error Rate** - WebSocket errors by type

---

## Exit Criteria

### 13.1 OpenTelemetry Middleware
- [x] `middleware.OpenTelemetry()` implemented
- [x] Every event creates a span
- [x] Span includes session ID, event type, target HID, route
- [x] Errors are recorded with `span.RecordError()`
- [x] Patch count recorded as attribute

### 13.2 Context Propagation
- [x] `ctx.StdContext()` returns traced context
- [x] `ctx.WithStdContext()` allows middleware to inject spans
- [x] Database calls inherit trace context
- [x] HTTP client calls inherit trace context

### 13.3 Protocol Depth Limits
- [x] `MaxVNodeDepth` enforced in VNode decode
- [x] `MaxPatchDepth` enforced in Patch decode
- [x] All recursive decode functions track depth
- [x] `ErrMaxDepthExceeded` returned for violations

### 13.4 Protocol Allocation Audit
- [x] All decode paths verified with checklist
- [x] No `make([]T, untrustedCount)` without limit
- [x] No `make([]byte, untrustedLength)` without limit

### 13.5 Fuzz Testing Suite
- [x] `FuzzDecodeEvent` implemented
- [x] `FuzzDecodePatch` implemented
- [x] `FuzzDecodeVNode` implemented
- [x] `FuzzDecodeHookPayload` implemented
- [x] CI job runs fuzz tests nightly
- [x] Crash artifacts are captured

### 13.6 Secure Defaults Enforcement
- [x] `CheckOrigin` defaults to `SameOriginCheck`
- [x] `CSRFProtection` defaults to `true`
- [x] `SecureCookies` defaults to `true`
- [x] `DevMode` must be explicitly enabled
- [x] Startup warns if security settings are insecure

### 13.7 Prometheus Metrics
- [x] `vango_active_sessions` gauge
- [x] `vango_events_total` counter with labels
- [x] `vango_event_duration_seconds` histogram
- [x] `vango_patches_sent_total` counter
- [x] Sample Grafana dashboard provided

---

## Files Changed

### New Files
- `pkg/middleware/otel.go` - OpenTelemetry middleware (~250 lines)
- `pkg/middleware/metrics.go` - Prometheus middleware (~315 lines)
- `pkg/middleware/middleware_test.go` - Comprehensive tests (~275 lines)
- `pkg/protocol/limits.go` - Depth limit constants and helpers (~78 lines)
- `pkg/protocol/security_test.go` - Allocation and depth limit tests (~330 lines)
- `pkg/protocol/fuzz_test.go` - Native Go fuzz tests (15 fuzz targets)
- `examples/monitoring/grafana-dashboard.json` - Sample Grafana dashboard
- `.github/workflows/fuzz.yml` - CI fuzz job (nightly)

### Modified Files
- `pkg/server/context.go` - Added `Event()`, `PatchCount()`, `AddPatchCount()`, `StdContext()`, `WithStdContext()`
- `pkg/server/handler.go` - Added `TypeString()` method to Event
- `pkg/protocol/vnode.go` - Depth tracking via `decodeVNodeWireWithDepth()`
- `pkg/protocol/patch.go` - Depth tracking via `decodePatchesFromWithDepth()`
- `pkg/protocol/event.go` - `MaxHookDepth` enforcement in hook decode
- `pkg/protocol/decoder.go` - Allocation limit checks (order fixed)
- `pkg/server/config.go` - Secure defaults (DevMode, SecureCookies, SameSiteMode, CheckOrigin)

---

## Dependencies

- `go.opentelemetry.io/otel` - OpenTelemetry API
- `go.opentelemetry.io/otel/sdk` - OTel SDK
- `go.opentelemetry.io/otel/exporters/otlp/otlptracehttp` - OTLP exporter
- `github.com/prometheus/client_golang` - Prometheus client

---

## Security Considerations

### Trace Data Sensitivity

Traces may contain sensitive information:

```go
// ❌ BAD: User ID in every trace
middleware.OpenTelemetry(middleware.IncludeUserID(true))

// ✅ GOOD: Only in secure environments
if os.Getenv("TRACE_USER_IDS") == "true" {
    opts = append(opts, middleware.IncludeUserID(true))
}
```

### Metrics Endpoint Security

The `/metrics` endpoint should NOT be publicly accessible:

```go
// ❌ BAD: Metrics exposed to internet
mux.Handle("/metrics", promhttp.Handler())

// ✅ GOOD: Metrics behind auth or internal network
internalMux := http.NewServeMux()
internalMux.Handle("/metrics", promhttp.Handler())
go http.ListenAndServe(":9090", internalMux) // Internal port
```

---

## Migration Guide

### Enabling Tracing

1. Add OpenTelemetry dependencies
2. Initialize tracer provider in `main.go`
3. Add `middleware.OpenTelemetry()` to app middleware stack

### Enabling Metrics

1. Add Prometheus dependency
2. Add `middleware.Prometheus()` to middleware stack
3. Expose `/metrics` endpoint on internal port
4. Import Grafana dashboard

---

## Completion Summary

**Status**: Complete
**Completed**: 2024-12-24
**Audited**: 2026-01-02 (spec alignment verification)
**All Tests Passing**: Yes

### Implementation Highlights

1. **OpenTelemetry Middleware** (`pkg/middleware/otel.go`)
   - Traces every WebSocket event with span attributes: session_id, event_type, event_target, route, patch_count
   - Configurable options: `WithTracerName()`, `WithIncludeUserID()`, `WithEventFilter()`
   - Context propagation via `ctx.StdContext()` for database/HTTP calls

2. **Prometheus Metrics** (`pkg/middleware/metrics.go`)
   - 8 metrics: active_sessions, detached_sessions, events_total, event_duration_seconds, patches_sent_total, session_memory_bytes, websocket_errors_total, reconnects_total
   - Global record functions for server hooks: `RecordPatches()`, `RecordSessionCreate()`, `RecordReconnect()`, etc.

3. **Protocol Security**
   - 15 fuzz test targets covering all decode functions
   - Depth limits enforced: MaxVNodeDepth=256, MaxPatchDepth=128, MaxHookDepth=64
   - Allocation limits with proper check ordering (limit before bounds)

4. **Secure Defaults** (`pkg/server/config.go`)
   - DevMode=false, SecureCookies=true, SameSiteMode=Lax, CheckOrigin=SameOrigin
   - **Design Note**: SameSiteLaxMode (not Strict) is intentional - it allows OAuth flows and payment redirects while still providing CSRF protection for top-level navigations. StrictMode would break legitimate cross-site flows.

5. **Grafana Dashboard** (`examples/monitoring/grafana-dashboard.json`)
   - 7 panels: Sessions, Detached Sessions, Sessions Over Time, Reconnect Rate, Event Rate by Path, Event Latency (P50/P95/P99), Patch Rate, Memory Distribution, Errors

### Audit Notes (2026-01-02)

- **Transaction (Tx) Tracing**: The spec (Section 7.8.1) mentions Tx-level OTel attributes like `vango.tx.id`, `vango.tx.writes.count`. These are explicitly v2.2+ features and will be implemented when the Transaction system is built.
- **Reconnect Metric**: Added `vango_reconnects_total` counter per spec requirement (Section 14.6).
- **Grafana Dashboard Labels**: Fixed to use `{{path}}` instead of `{{type}}` to match actual metric labels.

---

## Spec Verification Matrix

This section annotates every Phase 13 spec requirement against the implementation source code.

### 14.6 Observability (VANGO_ARCHITECTURE_AND_GUIDE.md)

| Requirement | Status | Source Location |
|-------------|--------|-----------------|
| No `ctx.Trace()` API (middleware-first) | ✅ VERIFIED | No such API exists in `pkg/server/context.go` |
| OpenTelemetry: middleware starts spans for each event | ✅ VERIFIED | `pkg/middleware/otel.go:179` - `tracer.Start()` |
| Spans for click/input/nav events | ✅ VERIFIED | `pkg/middleware/otel.go:152-158` - `event.TypeString()` |
| Records patch counts | ✅ VERIFIED | `pkg/middleware/otel.go:206` - `vango.patch_count` attribute |
| Records errors | ✅ VERIFIED | `pkg/middleware/otel.go:200-203` - `span.RecordError(err)` |
| Context propagation via `ctx.StdContext()` | ✅ VERIFIED | `pkg/server/context.go:535-545` |
| `ctx.WithStdContext()` for middleware injection | ✅ VERIFIED | `pkg/server/context.go:547-553` |
| Prometheus: session counts | ✅ VERIFIED | `pkg/middleware/metrics.go` - `activeSessions`, `detachedSessions` gauges |
| Prometheus: event rates | ✅ VERIFIED | `pkg/middleware/metrics.go` - `eventsTotal` counter |
| Prometheus: patch rates | ✅ VERIFIED | `pkg/middleware/metrics.go` - `patchesSent` counter |
| Prometheus: reconnects | ✅ VERIFIED | `pkg/middleware/metrics.go:346-352` - `reconnectsTotal` counter (added 2026-01-02) |

### 15.1 Secure Defaults (VANGO_ARCHITECTURE_AND_GUIDE.md)

| Setting | Default | Status | Source Location |
|---------|---------|--------|-----------------|
| `CheckOrigin` same-origin only | `SameOriginCheck` | ✅ VERIFIED | `pkg/server/config.go:279` |
| CSRF warning if disabled | Warning logged | ✅ VERIFIED | `pkg/server/server.go:87-89` |
| `on*` attributes stripped unless handler | Stripped | ✅ VERIFIED | `pkg/render/renderer.go:258-264` - `isEventHandlerKey()` |
| Protocol limits 4MB max allocation | 4MB | ✅ VERIFIED | `pkg/protocol/decoder.go:13` - `DefaultMaxAllocation` |

### 15.3 CSRF Protection (VANGO_ARCHITECTURE_AND_GUIDE.md)

| Requirement | Status | Source Location |
|-------------|--------|-----------------|
| Double Submit Cookie pattern | ✅ VERIFIED | `pkg/server/server.go:348-397` - `validateCSRF()` |
| `server.GenerateCSRFToken()` | ✅ VERIFIED | `pkg/server/server.go:404-428` |
| `server.SetCSRFCookie()` | ✅ VERIFIED | `pkg/server/server.go:430-441` |
| Cookie name `__vango_csrf` | ✅ VERIFIED | `pkg/server/server.go:346` - `CSRFCookieName` |
| Token embedded as `window.__VANGO_CSRF__` | ✅ VERIFIED | `pkg/render/page.go:357` |
| Client reads from cookie/global | ✅ VERIFIED | `client/src/websocket.js:83-91` |
| Warning if CSRFSecret nil | ✅ VERIFIED | `pkg/server/server.go:87-89` |

### 15.4 WebSocket Origin Validation (VANGO_ARCHITECTURE_AND_GUIDE.md)

| Requirement | Status | Source Location |
|-------------|--------|-----------------|
| `SameOriginCheck` as default | ✅ VERIFIED | `pkg/server/config.go:279` |
| Proper URL parsing (not string manipulation) | ✅ VERIFIED | `pkg/server/config.go:304-343` - uses `url.Parse()` |
| Tests for edge cases | ✅ VERIFIED | `pkg/server/config_test.go:380-432` |

### 15.5 Session Security (VANGO_ARCHITECTURE_AND_GUIDE.md)

| Setting | Status | Source Location | Notes |
|---------|--------|-----------------|-------|
| HttpOnly | ✅ VERIFIED | `pkg/server/server.go:437` | Set to `false` for CSRF cookie (required for Double Submit) |
| Secure | ✅ VERIFIED | `pkg/server/server.go:439` - `s.isSecure()` | Conditional on environment |
| SameSite | ✅ VERIFIED | `pkg/server/config.go:289` | Uses `Lax` (not Strict) - allows OAuth flows |

### 15.6 Protocol Defense (VANGO_ARCHITECTURE_AND_GUIDE.md)

| Limit | Spec Value | Impl Value | Status | Source Location |
|-------|------------|------------|--------|-----------------|
| Max string/bytes | 4MB | 4MB | ✅ VERIFIED | `pkg/protocol/decoder.go:13` |
| Max collection | 100K items | 100,000 | ✅ VERIFIED | `pkg/protocol/decoder.go:21` |
| Max VNode depth | 256 | 256 | ✅ VERIFIED | `pkg/protocol/limits.go:9` |
| Max patch depth | 128 | 128 | ✅ VERIFIED | `pkg/protocol/limits.go:14` |
| Hard cap | 16MB | 16MB | ✅ VERIFIED | `pkg/protocol/decoder.go:17` |
| Fuzz testing for decoders | Required | 15 targets | ✅ VERIFIED | `pkg/protocol/fuzz_test.go` |

### Fuzz Test Targets (15 total)

| Target | Status | Source Location |
|--------|--------|-----------------|
| `FuzzDecodeUvarint` | ✅ VERIFIED | `pkg/protocol/fuzz_test.go:8` |
| `FuzzDecodeSvarint` | ✅ VERIFIED | `pkg/protocol/fuzz_test.go:22` |
| `FuzzDecodeFrame` | ✅ VERIFIED | `pkg/protocol/fuzz_test.go:35` |
| `FuzzDecodeEvent` | ✅ VERIFIED | `pkg/protocol/fuzz_test.go:50` |
| `FuzzDecodePatches` | ✅ VERIFIED | `pkg/protocol/fuzz_test.go:74` |
| `FuzzDecodeVNodeWire` | ✅ VERIFIED | `pkg/protocol/fuzz_test.go:94` |
| `FuzzDecodeClientHello` | ✅ VERIFIED | `pkg/protocol/fuzz_test.go:114` |
| `FuzzDecodeServerHello` | ✅ VERIFIED | `pkg/protocol/fuzz_test.go:133` |
| `FuzzDecodeControl` | ✅ VERIFIED | `pkg/protocol/fuzz_test.go:150` |
| `FuzzDecodeAck` | ✅ VERIFIED | `pkg/protocol/fuzz_test.go:162` |
| `FuzzDecodeErrorMessage` | ✅ VERIFIED | `pkg/protocol/fuzz_test.go:173` |
| `FuzzRoundTrip` | ✅ VERIFIED | `pkg/protocol/fuzz_test.go:184` |
| `FuzzDecodeHookPayload` | ✅ VERIFIED | `pkg/protocol/fuzz_test.go:220` |
| `FuzzDeeplyNestedVNode` | ✅ VERIFIED | `pkg/protocol/fuzz_test.go:245` |
| `FuzzPatchesWithVNodes` | ✅ VERIFIED | `pkg/protocol/fuzz_test.go:263` |

### v2.2+ Features (NOT in Phase 13 scope)

| Feature | Spec Reference | Status |
|---------|----------------|--------|
| Transaction (Tx) OTel attributes | Section 7.8.1 | DEFERRED to v2.2 |
| `vango.tx.id`, `vango.tx.name`, etc. | Section 7.8.1 | DEFERRED to v2.2 |

### Ctx Interface Additions (Phase 13)

```go
// Event returns the current WebSocket event being processed.
Event() *Event

// PatchCount returns the number of patches sent during this request.
PatchCount() int

// AddPatchCount increments the patch count for this request.
AddPatchCount(count int)

// StdContext returns context.Context with trace propagation.
StdContext() context.Context

// WithStdContext returns a new Ctx with updated standard context.
WithStdContext(stdCtx context.Context) Ctx
```

---

*Last Updated: 2026-01-02*
