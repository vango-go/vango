# Phase 10: Routing, Middleware & Authentication

> **Production-ready integration with the Go HTTP ecosystem**

---

## Overview

Phase 10 upgrades Vango's routing and middleware architecture to enable seamless integration with the wider Go ecosystem. The core insight is that Vango operates across two distinct lifecycles—and attempting to unify them into a single middleware stack is an anti-pattern.

### Core Philosophy

**Vango owns the rendering loop; the developer owns the infrastructure.**

### Goals

1. **Ecosystem Compatibility**: Integrate with Chi, Gorilla, Echo, stdlib `net/http`
2. **Standard HTTP Handlers**: Expose Vango as a `http.Handler` for maximum flexibility
3. **Dual-Layer Middleware**: Separate infrastructure concerns from application concerns
4. **Auth Agnostic**: Work with any auth system (JWT, session cookies, OAuth)
5. **Zero Performance Regression**: Maintain <100ns event handling hot path

### Non-Goals (Explicit Exclusions)

1. Building a proprietary auth system (users bring their own)
2. Replacing the internal radix tree router (69ns performance)
3. Adding Redis/database-backed sessions (future Phase 11)
4. WebSocket cluster synchronization (future)

---

## The Two-Layer Architecture

Vango operates across two distinct lifecycles. Conflating them causes performance issues and context loss.

### Layer 1: HTTP Stack (Infrastructure)

| Aspect | Details |
|--------|---------|
| **Lifecycle** | Runs ONCE per session (initial GET + WebSocket upgrade) |
| **Standard** | `func(http.Handler) http.Handler` |
| **Responsibility** | Authentication, Logging, CORS, Panic Recovery, Rate Limiting |
| **Ecosystem** | Compatible with chi/middleware, gorilla/handlers, rs/cors |
| **When** | Before WebSocket connection is established |

### Layer 2: Vango Event Stack (Application)

| Aspect | Details |
|--------|---------|
| **Lifecycle** | Runs HUNDREDS of times per session (every Click, Input, Drag) |
| **Standard** | `func(vango.Ctx, next func() error) error` |
| **Responsibility** | Authorization guards (RBAC), Event validation, App logging |
| **Why Separate?** | Running HTTP middleware on 50ns binary events is perf suicide |
| **When** | During WebSocket event loop, on every user interaction |

### Architectural Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         Developer's Application                         │
├─────────────────────────────────────────────────────────────────────────┤
│  main.go                                                                │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  r := chi.NewRouter()                                              │ │
│  │  r.Use(middleware.Logger)                                          │ │
│  │  r.Use(middleware.Recoverer)                                       │ │
│  │  r.Use(cors.Handler(cors.Options{...}))                            │ │
│  │  r.Use(authMiddleware)  // JWT/Cookie validation                   │ │
│  │                                                                    │ │
│  │  // Traditional API routes                                         │ │
│  │  r.Post("/api/webhook", webhookHandler)                            │ │
│  │  r.Get("/api/health", healthHandler)                               │ │
│  │                                                                    │ │
│  │  // Mount Vango (takes over matching for page routes)              │ │
│  │  r.Handle("/*", app.Handler())                                     │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      LAYER 1: HTTP STACK                                │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  1. Request arrives: GET /dashboard                                │ │
│  │  2. Chi Logger: logs request                                       │ │
│  │  3. CORS middleware: adds headers                                  │ │
│  │  4. Auth middleware: validates JWT, sets user in context           │ │
│  │  5. Vango Handler receives request                                 │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      THE CONTEXT BRIDGE                                 │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  OnSessionStart hook runs SYNCHRONOUSLY during WS upgrade          │ │
│  │  1. Extract user from r.Context() (set by Layer 1 auth)            │ │
│  │  2. Copy user to session.Set("user", user)                         │ │
│  │  3. WebSocket connection upgraded                                  │ │
│  │  4. r.Context() is now DEAD (HTTP request complete)                │ │
│  │  5. Session persists with hydrated user data                       │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                  LAYER 2: VANGO EVENT STACK                             │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │  For EVERY user interaction (click, input, drag):                  │ │
│  │  1. Binary event decoded (~26ns)                                   │ │
│  │  2. Vango middleware chain runs                                    │ │
│  │     - RBAC guard: auth.Require[User](ctx)                          │ │
│  │     - Audit logger                                                 │ │
│  │  3. Handler executes                                               │ │
│  │  4. Component re-renders, diff computed                            │ │
│  │  5. Patches sent (~53ns encode)                                    │ │
│  └────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Subsystems

| Subsystem | Purpose | Priority |
|-----------|---------|----------|
| 10.1 HTTP Handler Interface | Standard `http.Handler` for mounting | Critical |
| 10.2 Context Bridge | Hydrate sessions from HTTP context | Critical |
| 10.3 Session State API | Thread-safe Get/Set for session data | Critical |
| 10.4 Auth Package | Type-safe auth helpers with generics | High |
| 10.5 Vango Middleware | Event-loop middleware chain | High |
| 10.6 Toast Notifications | Feedback without HTTP flash cookies | Medium |
| 10.7 File Upload | Hybrid HTTP+WS upload handling | Medium |

---

## 10.1 HTTP Handler Interface

### Goal

Expose Vango as a standard `http.Handler` so developers can mount it in any Go router.

### Current State

The current `Server` struct handles HTTP directly but isn't easily composable with external routers.

### Changes Required

#### File: `pkg/server/server.go`

```go
// Server is the main Vango server.
type Server struct {
    config    *ServerConfig
    manager   *SessionManager
    router    *router.Router
    upgrader  websocket.Upgrader
    logger    *slog.Logger
    metrics   *MetricsCollector
    
    // NEW: Hook for session hydration
    onSessionStart func(ctx context.Context, session *Session)
}

// Handler returns an http.Handler for mounting in external routers.
// This is the primary integration point for ecosystem compatibility.
func (s *Server) Handler() http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Dispatch based on path
        switch {
        case strings.HasPrefix(r.URL.Path, "/_vango/ws"):
            s.handleWebSocket(w, r)
        case strings.HasPrefix(r.URL.Path, "/_vango/"):
            s.handleInternal(w, r)
        default:
            s.handlePage(w, r)
        }
    })
}

// PageHandler returns an http.Handler for page routes only.
// Use when you want to handle /_vango/* separately.
func (s *Server) PageHandler() http.Handler {
    return http.HandlerFunc(s.handlePage)
}

// WebSocketHandler returns an http.Handler for WebSocket upgrade only.
// Use when you want fine-grained control over routing.
func (s *Server) WebSocketHandler() http.Handler {
    return http.HandlerFunc(s.handleWebSocket)
}
```

### New File: `pkg/server/config.go` (additions)

```go
type ServerConfig struct {
    // Existing fields...
    
    // OnSessionStart is called during WebSocket upgrade.
    // Use this to copy data from the HTTP context to the Vango session.
    // This runs SYNCHRONOUSLY before the upgrade completes.
    OnSessionStart func(httpCtx context.Context, session *Session)
    
    // TrustedProxies lists trusted reverse proxy IPs for X-Forwarded-* headers.
    TrustedProxies []string
    
    // CSRFTokenFunc generates/validates CSRF tokens.
    // If nil, uses default secure token generation.
    CSRFTokenFunc func(r *http.Request) (token string, valid bool)
}
```

### Usage Example

```go
// main.go
func main() {
    // Create Vango app
    app := vango.New(vango.Config{
        OnSessionStart: func(ctx context.Context, session *vango.Session) {
            // Copy user from HTTP context to session
            if user := auth.UserFromContext(ctx); user != nil {
                session.Set("vango_auth_user", user)
            }
        },
    })
    
    // Register routes
    routes.Register(app)
    
    // Create Chi router with HTTP middleware
    r := chi.NewRouter()
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    r.Use(cors.Handler(cors.Options{
        AllowedOrigins: []string{"https://example.com"},
    }))
    r.Use(auth.Middleware) // Your JWT/cookie validation
    
    // API routes (traditional HTTP)
    r.Post("/api/webhook", handleWebhook)
    r.Get("/api/health", handleHealth)
    
    // Mount Vango (handles all other routes)
    r.Handle("/*", app.Handler())
    
    // Start server
    http.ListenAndServe(":3000", r)
}
```

### Exit Criteria

- [ ] `app.Handler()` returns `http.Handler`
- [ ] `app.PageHandler()` returns page-only handler
- [ ] `app.WebSocketHandler()` returns WS-only handler
- [ ] Unit test: handler can be mounted in Chi router
- [ ] Unit test: handler can be mounted in stdlib mux
- [ ] Integration test: Chi middleware chain executes before Vango

---

## 10.2 Context Bridge

### Goal

Solve the "Dead Context" problem where data from HTTP middleware is lost when WebSocket upgrades.

### The Problem

```
HTTP Request → Auth Middleware sets r.Context() → Handler → WebSocket Upgrade
                                                                    ↓
                                                   r.Context() is CANCELED
                                                   All HTTP middleware data is LOST
```

### The Solution

The Context Bridge runs synchronously during the upgrade handshake, before the HTTP context dies.

### Implementation

#### File: `pkg/server/websocket.go` (modify)

```go
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
    // Validate origin, CSRF, etc.
    if !s.validateUpgrade(r) {
        http.Error(w, "Forbidden", http.StatusForbidden)
        return
    }
    
    // Upgrade connection
    conn, err := s.upgrader.Upgrade(w, r, nil)
    if err != nil {
        s.logger.Error("websocket upgrade failed", "error", err)
        return
    }
    
    // Create session
    session := s.manager.Create(conn, "")
    
    // ═══════════════════════════════════════════════════════════════
    // THE CONTEXT BRIDGE: Copy data from dying HTTP context to session
    // ═══════════════════════════════════════════════════════════════
    if s.config.OnSessionStart != nil {
        s.config.OnSessionStart(r.Context(), session)
    }
    
    // Perform handshake
    if err := s.performHandshake(conn, session, r); err != nil {
        s.logger.Error("handshake failed", "error", err)
        session.Close()
        return
    }
    
    // Start session event loops
    session.Start()
}
```

### Flow Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│ HTTP Request: GET /_vango/ws                                    │
│ Headers: Cookie: session=abc123                                 │
└──────────────────────────────┬──────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│ Layer 1: HTTP Middleware (Chi/Gorilla/Echo)                     │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │ 1. Logger: logs request                                     │ │
│ │ 2. Auth Middleware:                                         │ │
│ │    - Reads "session=abc123" cookie                          │ │
│ │    - Validates session in Redis/DB                          │ │
│ │    - Sets user in context:                                  │ │
│ │      ctx = context.WithValue(ctx, "user", &models.User{...})│ │
│ └─────────────────────────────────────────────────────────────┘ │
│ r.Context() now contains: {"user": &User{ID: 123, Email: ...}}  │
└──────────────────────────────┬──────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│ Vango WebSocket Handler                                         │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │ 1. Upgrade HTTP → WebSocket                                 │ │
│ │ 2. Create Vango Session                                     │ │
│ │ 3. THE BRIDGE: OnSessionStart(r.Context(), session)         │ │
│ │    - Developer code:                                        │ │
│ │      user := r.Context().Value("user").(*User)              │ │
│ │      session.Set("vango_auth_user", user)                   │ │
│ │ 4. Complete handshake                                       │ │
│ │ 5. HTTP context is now DEAD (request complete)              │ │
│ └─────────────────────────────────────────────────────────────┘ │
│ Session.data now contains: {"vango_auth_user": &User{...}}      │
└──────────────────────────────┬──────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│ WebSocket Event Loop (runs for minutes/hours)                   │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │ For each event:                                             │ │
│ │   user, _ := auth.Get[*User](ctx)  // From session.Get()    │ │
│ │   if user == nil { return ErrUnauthorized }                 │ │
│ │   // Handle event with authenticated user                   │ │
│ └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### Exit Criteria

- [ ] `OnSessionStart` callback added to `ServerConfig`
- [ ] Callback invoked synchronously before handshake completes
- [ ] Unit test: data survives from HTTP context to session
- [ ] Unit test: callback errors prevent session creation

---

## 10.3 Session State API

### Goal

Provide thread-safe Get/Set methods on Session for storing arbitrary data.

### Current State (`pkg/server/session.go`)

The Session struct has `ID`, `UserID`, and component state, but no general-purpose data storage.

### Changes Required

#### File: `pkg/server/session.go` (modify)

```go
type Session struct {
    // Existing fields...
    
    // NEW: General-purpose session data storage
    data   map[string]any
    dataMu sync.RWMutex
}

// Get retrieves a value from session data.
// Returns nil if key doesn't exist.
func (s *Session) Get(key string) any {
    s.dataMu.RLock()
    defer s.dataMu.RUnlock()
    return s.data[key]
}

// Set stores a value in session data.
// Value must be safe to access concurrently (immutable or properly synchronized).
//
// WARNING: For future Redis/distributed session support, stored values should
// be JSON-serializable. Avoid storing functions, channels, or complex structs.
// In debug mode, this will panic on obviously unserializable types.
func (s *Session) Set(key string, value any) {
    s.dataMu.Lock()
    defer s.dataMu.Unlock()
    if s.data == nil {
        s.data = make(map[string]any)
    }
    
    // Debug mode: check for unserializable types
    if debugMode {
        t := reflect.TypeOf(value)
        if t != nil {
            switch t.Kind() {
            case reflect.Func, reflect.Chan:
                panic(fmt.Sprintf("Session.Set: cannot store %s (unserializable)", t.Kind()))
            }
        }
    }
    
    s.data[key] = value
}

// SetString stores a string value (always serializable).
func (s *Session) SetString(key string, value string) {
    s.Set(key, value)
}

// SetInt stores an int value (always serializable).
func (s *Session) SetInt(key string, value int) {
    s.Set(key, value)
}

// SetJSON stores a JSON-serializable struct.
// Panics in debug mode if value cannot be marshaled.
func (s *Session) SetJSON(key string, value any) {
    if debugMode {
        if _, err := json.Marshal(value); err != nil {
            panic(fmt.Sprintf("Session.SetJSON: value not serializable: %v", err))
        }
    }
    s.Set(key, value)
}

// Delete removes a key from session data.
func (s *Session) Delete(key string) {
    s.dataMu.Lock()
    defer s.dataMu.Unlock()
    delete(s.data, key)
}

// GetString is a convenience method that returns value as string.
func (s *Session) GetString(key string) string {
    if v, ok := s.Get(key).(string); ok {
        return v
    }
    return ""
}

// GetInt is a convenience method that returns value as int.
func (s *Session) GetInt(key string) int {
    switch v := s.Get(key).(type) {
    case int:
        return v
    case int64:
        return int(v)
    case float64:
        return int(v)
    default:
        return 0
    }
}

// Has returns whether a key exists in session data.
func (s *Session) Has(key string) bool {
    s.dataMu.RLock()
    defer s.dataMu.RUnlock()
    _, ok := s.data[key]
    return ok
}
```

### Exit Criteria

- [ ] `Session.Get(key)` returns any value
- [ ] `Session.Set(key, value)` stores value with debug validation
- [ ] `Session.SetString(key, value)` stores string
- [ ] `Session.SetInt(key, value)` stores int
- [ ] `Session.SetJSON(key, value)` stores with serialization check
- [ ] `Session.Delete(key)` removes value
- [ ] Convenience getters: `GetString`, `GetInt`, `Has`
- [ ] Thread-safe (protected by RWMutex)
- [ ] Debug mode panics on func/chan storage
- [ ] Unit tests for concurrent access
- [ ] Benchmark: Get/Set under contention

---

## 10.4 Auth Package (`vango/auth`)

### Goal

Provide type-safe helpers for accessing authenticated user data from sessions.

### New Package: `pkg/auth/`

#### File: `pkg/auth/auth.go`

```go
// Package auth provides type-safe authentication helpers for Vango.
//
// The auth package is designed to be auth-system agnostic. It doesn't
// validate tokens or manage sessions—that's the responsibility of HTTP
// middleware in Layer 1. Instead, it provides type-safe access to the
// user data that was hydrated via the Context Bridge.
//
// Basic usage:
//
//     func Dashboard(ctx vango.Ctx) (vango.Component, error) {
//         user, err := auth.Require[*models.User](ctx)
//         if err != nil {
//             return nil, err // Returns 401, handled by error boundary
//         }
//         return renderDashboard(user), nil
//     }
//
// For optional auth (guest allowed):
//
//     func HomePage(ctx vango.Ctx) vango.Component {
//         user, ok := auth.Get[*models.User](ctx)
//         if ok {
//             return renderLoggedInHome(user)
//         }
//         return renderGuestHome()
//     }
package auth

import (
    "errors"
    "log/slog"
    "reflect"
    
    "github.com/vango-dev/vango/v2/pkg/server"
)

// Standard session key for auth user
const SessionKey = "vango_auth_user"

// ErrUnauthorized is returned when auth is required but not present.
var ErrUnauthorized = errors.New("unauthorized: authentication required")

// ErrForbidden is returned when auth is present but insufficient.
var ErrForbidden = errors.New("forbidden: insufficient permissions")

// Get retrieves the authenticated user from the session.
// Returns (user, true) if authenticated, (zero, false) otherwise.
//
// In debug mode, logs a warning if a value exists but type assertion fails,
// helping developers catch common value/pointer mismatches.
func Get[T any](ctx server.Ctx) (T, bool) {
    val := ctx.Session().Get(SessionKey)
    if val == nil {
        var zero T
        return zero, false
    }
    
    // Fast path: Type assertion succeeds
    if user, ok := val.(T); ok {
        return user, true
    }
    
    // Slow path: Debug aid for type mismatches
    // Only runs if value exists but assertion failed
    if server.DebugMode {
        storedType := reflect.TypeOf(val)
        requestedType := reflect.TypeOf(*new(T))
        
        slog.Warn("vango/auth: type mismatch",
            "stored_type", storedType,
            "requested_type", requestedType,
            "hint", "Did you store a struct (User) but request a pointer (*User)?",
        )
    }
    
    var zero T
    return zero, false
}

// Require returns the authenticated user or an error.
// Use in components/handlers that require authentication.
func Require[T any](ctx server.Ctx) (T, error) {
    user, ok := Get[T](ctx)
    if !ok {
        return user, ErrUnauthorized
    }
    return user, nil
}

// Set stores the authenticated user in the session.
// Typically called from OnSessionStart in the Context Bridge.
func Set[T any](session *server.Session, user T) {
    session.Set(SessionKey, user)
}

// Clear removes the authenticated user from the session.
// Call this on logout.
func Clear(session *server.Session) {
    session.Delete(SessionKey)
}

// IsAuthenticated returns whether the session has an authenticated user.
func IsAuthenticated(ctx server.Ctx) bool {
    return ctx.Session().Has(SessionKey)
}
```

#### File: `pkg/auth/middleware.go`

```go
package auth

import (
    "github.com/vango-dev/vango/v2/pkg/router"
    "github.com/vango-dev/vango/v2/pkg/server"
)

// RequireAuth is Vango middleware that requires authentication.
// Use on routes that should only be accessible to logged-in users.
//
// Usage:
//     func Middleware() []router.Middleware {
//         return []router.Middleware{
//             auth.RequireAuth,
//         }
//     }
var RequireAuth router.Middleware = router.MiddlewareFunc(
    func(ctx server.Ctx, next func() error) error {
        if !IsAuthenticated(ctx) {
            return ErrUnauthorized
        }
        return next()
    },
)

// RequireRole returns middleware that requires a specific role.
// The check function receives the user and returns true if authorized.
//
// Usage:
//     func Middleware() []router.Middleware {
//         return []router.Middleware{
//             auth.RequireRole(func(u *models.User) bool {
//                 return u.Role == "admin"
//             }),
//         }
//     }
func RequireRole[T any](check func(T) bool) router.Middleware {
    return router.MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
        user, ok := Get[T](ctx)
        if !ok {
            return ErrUnauthorized
        }
        if !check(user) {
            return ErrForbidden
        }
        return next()
    })
}

// RequirePermission returns middleware that checks for a specific permission.
//
// Usage:
//     auth.RequirePermission[*models.User](func(u *models.User) bool {
//         return u.Can("projects.delete")
//     })
func RequirePermission[T any](check func(T) bool) router.Middleware {
    return RequireRole(check) // Same implementation, semantic alias
}
```

### Usage Examples

```go
// In component
func Dashboard(ctx vango.Ctx) (vango.Component, error) {
    user, err := auth.Require[*models.User](ctx)
    if err != nil {
        return nil, err // Error boundary handles redirect
    }
    
    return vango.Func(func() *vdom.VNode {
        return Div(
            H1(Textf("Welcome, %s", user.Name)),
            // ...
        )
    }), nil
}

// In route middleware (file-based routing)
// app/routes/admin/_layout.go
func Middleware() []router.Middleware {
    return []router.Middleware{
        auth.RequireRole(func(u *models.User) bool {
            return u.IsAdmin
        }),
    }
}

// In Context Bridge (main.go)
app := vango.New(vango.Config{
    OnSessionStart: func(ctx context.Context, session *vango.Session) {
        // Your auth middleware already validated and set user in context
        if user := myauth.UserFromContext(ctx); user != nil {
            auth.Set(session, user)
        }
    },
})
```

### Exit Criteria

- [ ] `auth.Get[T](ctx)` returns typed user
- [ ] `auth.Require[T](ctx)` returns user or error
- [ ] `auth.Set[T](session, user)` stores user
- [ ] `auth.Clear(session)` removes user
- [ ] `auth.IsAuthenticated(ctx)` returns bool
- [ ] `auth.RequireAuth` middleware
- [ ] `auth.RequireRole[T](check)` middleware
- [ ] Unit tests with mock session
- [ ] Example integration with Goth OAuth

---

## 10.5 Vango Middleware Improvements

### Goal

Formalize and document the Vango event middleware layer (already partially exists in `pkg/router/middleware.go`).

### Current State

The middleware interface exists:
```go
type Middleware interface {
    Handle(ctx server.Ctx, next func() error) error
}
```

### Enhancements

#### File: `pkg/router/middleware.go` (additions)

```go
// WithValue returns middleware that sets a context value for the request.
func WithValue(key, value any) Middleware {
    return MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
        ctx.SetValue(key, value)
        return next()
    })
}

// Logger returns middleware that logs each event.
func Logger(logger *slog.Logger) Middleware {
    return MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
        start := time.Now()
        err := next()
        
        logger.Info("event handled",
            "session", ctx.Session().ID,
            "event", ctx.Event().Type,
            "hid", ctx.Event().HID,
            "duration", time.Since(start),
            "error", err,
        )
        
        return err
    })
}

// Recover returns middleware that recovers from panics.
func Recover(onPanic func(ctx server.Ctx, recovered any)) Middleware {
    return MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
        defer func() {
            if r := recover(); r != nil {
                if onPanic != nil {
                    onPanic(ctx, r)
                }
            }
        }()
        return next()
    })
}

// RateLimit returns middleware that limits events per session.
func RateLimit(maxPerSecond int) Middleware {
    // Uses session-stored limiter
    return MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
        limiter := ctx.Session().Get("_rate_limiter")
        if limiter == nil {
            limiter = rate.NewLimiter(rate.Limit(maxPerSecond), maxPerSecond)
            ctx.Session().Set("_rate_limiter", limiter)
        }
        
        if !limiter.(*rate.Limiter).Allow() {
            return errors.New("rate limit exceeded")
        }
        
        return next()
    })
}
```

### Context Interface Updates

#### File: `pkg/server/context.go` (additions)

```go
// Ctx is the context interface for Vango handlers.
type Ctx interface {
    // Existing methods...
    
    // Session returns the current session.
    Session() *Session
    
    // Event returns the current event being processed.
    Event() *Event
    
    // NEW: SetValue stores a request-scoped value.
    SetValue(key, value any)
    
    // NEW: Value retrieves a request-scoped value.
    Value(key any) any
}
```

### Exit Criteria

- [ ] `router.Logger(logger)` middleware
- [ ] `router.Recover(handler)` middleware
- [ ] `router.RateLimit(n)` middleware
- [ ] `ctx.SetValue()` / `ctx.Value()` for request scoping
- [ ] Unit tests for each middleware
- [ ] Benchmark: middleware chain overhead

---

## 10.6 Toast Notifications (`vango/toast`)

### Goal

Provide feedback mechanism since HTTP flash cookies don't work with persistent WebSocket.

### Approach: Zero-Protocol with `ctx.Emit`

> **Design Decision**: Instead of adding a new protocol opcode (`PatchShowToast`), we use
> the existing Custom Event mechanism (`ctx.Emit`). This keeps the thin client at 9.56KB
> and allows users to choose their own toast UI library.

This approach:
- **Zero changes** to `pkg/protocol/`
- **Zero changes** to `client/src/`
- Users can use Toastify, Sonner, vanilla JS, or any library they prefer

### New Package: `pkg/toast/`

#### File: `pkg/toast/toast.go`

```go
// Package toast provides feedback notifications for Vango applications.
//
// Since Vango uses persistent WebSocket connections, traditional HTTP
// flash cookies don't work. Instead, toasts use the generic ctx.Emit()
// mechanism to dispatch events to the client.
//
// The client-side handler is user-defined, allowing integration with
// any toast library (Toastify, Sonner, vanilla JS, etc.).
//
// Usage:
//     toast.Success(ctx, "Changes saved!")
//     toast.Error(ctx, "Failed to delete item")
package toast

import "github.com/vango-dev/vango/v2/pkg/server"

// EventName is the event dispatched for toasts.
const EventName = "vango:toast"

// Type represents the toast notification type.
type Type string

const (
    TypeSuccess Type = "success"
    TypeError   Type = "error"
    TypeWarning Type = "warning"
    TypeInfo    Type = "info"
)

// Show displays a toast notification to the user.
// Uses ctx.Emit to send a custom event to the client.
func Show(ctx server.Ctx, level Type, message string) {
    ctx.Emit(EventName, map[string]any{
        "level":   string(level),
        "message": message,
    })
}

// Success shows a success toast.
func Success(ctx server.Ctx, message string) {
    Show(ctx, TypeSuccess, message)
}

// Error shows an error toast.
func Error(ctx server.Ctx, message string) {
    Show(ctx, TypeError, message)
}

// Warning shows a warning toast.
func Warning(ctx server.Ctx, message string) {
    Show(ctx, TypeWarning, message)
}

// Info shows an info toast.
func Info(ctx server.Ctx, message string) {
    Show(ctx, TypeInfo, message)
}

// WithTitle shows a toast with a title and message.
func WithTitle(ctx server.Ctx, level Type, title, message string) {
    ctx.Emit(EventName, map[string]any{
        "level":   string(level),
        "title":   title,
        "message": message,
    })
}
```

### Client-Side Handler (User Land)

The client listens for the `vango:toast` custom event and renders using their preferred library:

```javascript
// user/app.js - User provides this, NOT framework core
window.addEventListener("vango:toast", (e) => {
    const { level, message, title } = e.detail;
    
    // Option 1: Use a library like Toastify
    Toastify({ text: message, className: level }).showToast();
    
    // Option 2: Use Sonner
    toast[level](message);
    
    // Option 3: Vanilla JS
    showCustomToast(level, message);
});
```

### Usage in Components

```go
func DeleteProject(ctx vango.Ctx, id int) error {
    if err := db.Projects.Delete(id); err != nil {
        toast.Error(ctx, "Failed to delete project")
        return err
    }
    
    toast.Success(ctx, "Project deleted")
    ctx.Navigate("/projects")
    return nil
}

// With title
func SaveSettings(ctx vango.Ctx) {
    toast.WithTitle(ctx, toast.TypeSuccess, "Settings", "Your changes have been saved.")
}
```

### Exit Criteria

- [ ] `toast.Success(ctx, msg)` calls `ctx.Emit`
- [ ] `toast.Error(ctx, msg)` / `toast.Warning` / `toast.Info`
- [ ] `toast.WithTitle(ctx, level, title, msg)` for custom titles
- [ ] **No protocol changes required** (uses existing Custom Event)
- [ ] **No client changes required** (user provides handler)
- [ ] Unit test: verify `ctx.Emit` called with correct payload
- [ ] Example: toast with Sonner/Toastify

---

## 10.7 File Upload (`vango/upload`)

### Goal

Handle file uploads efficiently despite WebSocket limitations.

### The Problem

Large file uploads over WebSocket block the heartbeat and event loop.

### Solution

Hybrid approach: HTTP POST for upload, WebSocket for processing.

### Flow

```
1. User selects file in <input type="file">
2. Client performs HTTP POST to /upload endpoint (traditional)
3. Server streams to temp storage (S3/disk), returns temp_id
4. Client includes temp_id in form submission via WebSocket
5. Vango handler calls upload.Claim(temp_id) to finalize
```

### New Package: `pkg/upload/`

#### File: `pkg/upload/upload.go`

```go
// Package upload provides file upload handling for Vango.
//
// Since WebSocket connections are poor at handling large binary uploads
// (blocking heartbeats), this package uses a hybrid HTTP+WebSocket approach.
package upload

import (
    "io"
    "net/http"
    "time"
)

// Store is the interface for upload storage backends.
type Store interface {
    // Save stores the uploaded file and returns a temp ID.
    Save(filename string, contentType string, r io.Reader) (tempID string, err error)
    
    // Claim retrieves and removes a temp file, returning a permanent handle.
    Claim(tempID string) (*File, error)
    
    // Cleanup removes expired temp files.
    Cleanup(maxAge time.Duration) error
}

// File represents an uploaded file.
type File struct {
    ID          string
    Filename    string
    ContentType string
    Size        int64
    Path        string    // For disk storage
    URL         string    // For S3/CDN storage
    Reader      io.Reader // For streaming
}

// Handler returns an http.Handler for file uploads.
// Mount this on your router: r.Post("/upload", upload.Handler(store))
func Handler(store Store) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
            return
        }
        
        file, header, err := r.FormFile("file")
        if err != nil {
            http.Error(w, "No file provided", http.StatusBadRequest)
            return
        }
        defer file.Close()
        
        tempID, err := store.Save(header.Filename, header.Header.Get("Content-Type"), file)
        if err != nil {
            http.Error(w, "Upload failed", http.StatusInternalServerError)
            return
        }
        
        // Return temp ID as JSON
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]string{
            "temp_id": tempID,
        })
    })
}

// Claim retrieves a temp file by ID.
// Call this in your Vango handler after receiving the temp_id.
func Claim(store Store, tempID string) (*File, error) {
    return store.Claim(tempID)
}
```

#### File: `pkg/upload/disk.go`

```go
package upload

// DiskStore stores uploads on the local filesystem.
type DiskStore struct {
    dir     string
    maxSize int64
}

func NewDiskStore(dir string, maxSize int64) *DiskStore {
    return &DiskStore{dir: dir, maxSize: maxSize}
}

func (s *DiskStore) Save(filename, contentType string, r io.Reader) (string, error) {
    tempID := generateTempID()
    path := filepath.Join(s.dir, tempID)
    
    f, err := os.Create(path)
    if err != nil {
        return "", err
    }
    defer f.Close()
    
    // Limit read to maxSize
    limited := io.LimitReader(r, s.maxSize)
    size, err := io.Copy(f, limited)
    if err != nil {
        os.Remove(path)
        return "", err
    }
    
    // Store metadata
    meta := &uploadMeta{
        Filename:    filename,
        ContentType: contentType,
        Size:        size,
        CreatedAt:   time.Now(),
    }
    s.saveMeta(tempID, meta)
    
    return tempID, nil
}

func (s *DiskStore) Claim(tempID string) (*File, error) {
    meta, err := s.loadMeta(tempID)
    if err != nil {
        return nil, ErrNotFound
    }
    
    path := filepath.Join(s.dir, tempID)
    
    // Remove from temp after claim
    defer os.Remove(path)
    defer s.deleteMeta(tempID)
    
    return &File{
        ID:          tempID,
        Filename:    meta.Filename,
        ContentType: meta.ContentType,
        Size:        meta.Size,
        Path:        path,
    }, nil
}
```

### Usage

```go
// main.go - Mount upload handler
r.Post("/upload", upload.Handler(uploadStore))

// component.go - Handle uploaded file
func CreatePost(ctx vango.Ctx, formData server.FormData) error {
    tempID := formData.Get("attachment_temp_id")
    
    var attachment *upload.File
    if tempID != "" {
        file, err := upload.Claim(uploadStore, tempID)
        if err != nil {
            return err
        }
        attachment = file
    }
    
    // Create post with attachment
    db.Posts.Create(Post{
        Title:         formData.Get("title"),
        AttachmentURL: attachment.URL,
    })
    
    toast.Success(ctx, "Post created!")
    return nil
}
```

### Exit Criteria

- [ ] `upload.Handler(store)` returns `http.Handler`
- [ ] `upload.Claim(store, tempID)` returns file
- [ ] `DiskStore` implementation
- [ ] S3 store example in docs
- [ ] Temp file cleanup
- [ ] Unit test: upload flow
- [ ] Max file size enforcement

---

## File Structure

### New Files

```
pkg/
├── auth/
│   ├── doc.go           # Package documentation
│   ├── auth.go          # Get, Require, Set, Clear, IsAuthenticated
│   ├── middleware.go    # RequireAuth, RequireRole, RequirePermission
│   └── auth_test.go     # Unit tests
├── toast/
│   ├── doc.go           # Package documentation
│   ├── toast.go         # Show, Success, Error, Warning, Info
│   └── toast_test.go    # Unit tests
├── upload/
│   ├── doc.go           # Package documentation
│   ├── upload.go        # Store interface, Handler, Claim
│   ├── disk.go          # DiskStore implementation
│   └── upload_test.go   # Unit tests
├── vtest/
│   ├── doc.go           # Package documentation
│   ├── vtest.go         # NewCtx, CtxBuilder, ExpectContains, RenderToString
│   └── vtest_test.go    # Unit tests
```

### Modified Files

```
pkg/server/
├── server.go            # Add Handler(), PageHandler(), WebSocketHandler()
├── config.go            # Add OnSessionStart, TrustedProxies, CSRFTokenFunc
├── session.go           # Add Get, Set, SetString, SetInt, SetJSON, Delete, Has
├── websocket.go         # Call OnSessionStart during upgrade
├── context.go           # Add SetValue, Value, Emit for request scoping

pkg/router/
├── middleware.go        # Add Logger, Recover, RateLimit, WithValue
```

> **Note**: No protocol or client changes required for Phase 10. Toast uses the
> existing Custom Event (`ctx.Emit`) mechanism.

---

## Implementation Phases

### Phase 10A: Core Infrastructure (Critical)

**Duration**: 2-3 days

1. Add `OnSessionStart` to `ServerConfig`
2. Implement `Session.Get/Set/Delete`
3. Implement `Server.Handler()` returning `http.Handler`
4. Wire up Context Bridge in `handleWebSocket`

**Exit Criteria**:
- Chi middleware can authenticate, Vango session receives user
- Integration test passes

### Phase 10B: Auth Package (High)

**Duration**: 1-2 days

1. Create `pkg/auth/` package
2. Implement `Get`, `Require`, `Set`, `Clear`
3. Implement `RequireAuth`, `RequireRole` middleware
4. Write unit tests

**Exit Criteria**:
- `auth.Require[*User](ctx)` works in components
- Middleware blocks unauthenticated requests

### Phase 10C: Enhanced Middleware (High)

**Duration**: 1 day

1. Add `Logger`, `Recover`, `RateLimit` to `pkg/router/`
2. Add `ctx.SetValue()` / `ctx.Value()` to Ctx interface
3. Write unit tests and benchmarks

**Exit Criteria**:
- All middleware functions work
- No performance regression on hot path

### Phase 10D: Toast & Upload (Medium)

**Duration**: 1-2 days

1. Create `pkg/toast/` package using `ctx.Emit` (no protocol changes)
2. Create `pkg/upload/` package with `DiskStore`
3. Write integration tests

**Exit Criteria**:
- `toast.Success(ctx, "Saved!")` calls `ctx.Emit("vango:toast", ...)`
- File upload flow works end-to-end

### Phase 10E: Documentation & Examples (Medium)

**Duration**: 1-2 days

1. Write integration guide (Chi + Gorilla Sessions)
2. Create example: Vango + Goth OAuth
3. Update API documentation
4. Add to vango.dev/docs

**Exit Criteria**:
- Developer can set up auth with external docs
- Example runs successfully

### Phase 10F: Test Kit (vtest)

**Duration**: 1 day

**Goal**: Reduce testing boilerplate for authenticated components to a single line.

1. Create `pkg/vtest/` package
2. Implement fluent context builder (`NewCtx().WithUser().Build()`)
3. Add render assertion helpers (`ExpectContains`, `RenderToString`)
4. Write usage examples

#### File: `pkg/vtest/vtest.go`

```go
// Package vtest provides testing helpers for Vango components.
//
// The vtest package reduces boilerplate when testing authenticated
// components by providing fluent context builders and render assertions.
//
// Usage:
//     func TestDashboard(t *testing.T) {
//         ctx := vtest.NewCtx().WithUser(&User{Role: "admin"}).Build()
//         comp, err := Dashboard(ctx)
//         vtest.ExpectContains(t, comp, "Welcome Admin")
//     }
package vtest

import (
    "strings"
    "testing"
    
    "github.com/vango-dev/vango/v2"
    "github.com/vango-dev/vango/v2/pkg/auth"
    "github.com/vango-dev/vango/v2/pkg/render"
    "github.com/vango-dev/vango/v2/pkg/server"
)

// CtxBuilder allows fluent construction of test contexts.
type CtxBuilder struct {
    session *server.Session
    ctx     server.Ctx
}

// NewCtx creates a new context builder for testing.
func NewCtx() *CtxBuilder {
    s := server.NewMockSession()
    return &CtxBuilder{
        session: s,
        ctx:     server.NewTestContext(s),
    }
}

// WithUser injects an authenticated user into the session.
func (b *CtxBuilder) WithUser(user any) *CtxBuilder {
    auth.Set(b.session, user)
    return b
}

// WithData injects arbitrary data into the session.
func (b *CtxBuilder) WithData(key string, val any) *CtxBuilder {
    b.session.Set(key, val)
    return b
}

// Build returns the final context for use in tests.
func (b *CtxBuilder) Build() server.Ctx {
    return b.ctx
}

// CtxWithUser is a shorthand for NewCtx().WithUser(user).Build()
func CtxWithUser(user any) server.Ctx {
    return NewCtx().WithUser(user).Build()
}

// RenderToString renders a component and returns the HTML string.
func RenderToString(comp vango.Component) string {
    return render.RenderToString(comp)
}

// ExpectContains asserts that rendered output contains expected substring.
func ExpectContains(t *testing.T, comp vango.Component, expected string) {
    t.Helper()
    html := RenderToString(comp)
    if !strings.Contains(html, expected) {
        t.Errorf("expected rendered output to contain %q, got:\n%s", expected, html)
    }
}

// ExpectNotContains asserts that rendered output does not contain substring.
func ExpectNotContains(t *testing.T, comp vango.Component, unexpected string) {
    t.Helper()
    html := RenderToString(comp)
    if strings.Contains(html, unexpected) {
        t.Errorf("expected rendered output to NOT contain %q, got:\n%s", unexpected, html)
    }
}
```

#### Usage Examples

```go
func TestDashboard_Authenticated(t *testing.T) {
    // Setup: Fluent context with authenticated user
    ctx := vtest.NewCtx().
        WithUser(&User{ID: "123", Role: "admin"}).
        WithData("theme", "dark").
        Build()

    // Execute
    comp, err := Dashboard(ctx)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    
    // Assert
    vtest.ExpectContains(t, comp, "Welcome")
    vtest.ExpectContains(t, comp, "admin")
}

func TestDashboard_Unauthenticated(t *testing.T) {
    // No user = unauthenticated
    ctx := vtest.NewCtx().Build()
    
    _, err := Dashboard(ctx)
    if err != auth.ErrUnauthorized {
        t.Errorf("expected ErrUnauthorized, got %v", err)
    }
}
```

**Exit Criteria**:
- `vtest.NewCtx().WithUser(u).Build()` works
- `vtest.ExpectContains(t, comp, "text")` works
- Zero-dependency on external test frameworks
- Unit tests for vtest helpers

---

## Verification Plan

### Unit Tests

```bash
# Run all Phase 10 tests
go test ./pkg/auth/... ./pkg/toast/... ./pkg/upload/... -v

# Run session state tests
go test ./pkg/server/... -run TestSession -v

# Run middleware tests
go test ./pkg/router/... -run TestMiddleware -v
```

### Integration Tests

```bash
# Test Chi integration
go test ./test/integration/chi_test.go -v

# Test full auth flow
go test ./test/integration/auth_flow_test.go -v
```

### Benchmarks

```bash
# Verify no performance regression
go test ./pkg/server/... -bench=. -benchmem

# Expected results:
# BenchmarkSessionGet-8      50000000    23.5 ns/op    0 B/op    0 allocs/op
# BenchmarkSessionSet-8      30000000    45.2 ns/op    0 B/op    0 allocs/op
# BenchmarkMiddlewareChain-8 10000000    125 ns/op     0 B/op    0 allocs/op
```

### Manual Testing

1. Create new Vango app with Chi router
2. Add JWT auth middleware
3. Verify user available in Vango components
4. Test toast notifications appear
5. Test file upload flow

---

## Exit Criteria Summary

| Subsystem | Critical Exit Criteria |
|-----------|------------------------|
| 10.1 HTTP Handler | `app.Handler()` returns `http.Handler` that works with Chi |
| 10.2 Context Bridge | `OnSessionStart` callback hydrates session from HTTP context |
| 10.3 Session State | `session.Get/Set` are thread-safe and work |
| 10.4 Auth Package | `auth.Require[T](ctx)` returns typed user or error |
| 10.5 Middleware | `router.Logger`, `Recover`, `RateLimit` work |
| 10.6 Toast | `toast.Success(ctx, msg)` emits via `ctx.Emit` |
| 10.7 Upload | HTTP upload + WS claim flow works |
| 10F Test Kit | `vtest.NewCtx().WithUser().Build()` reduces test boilerplate |

---

## Risks & Mitigations

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Breaking existing API | High | Medium | Version `Server.Handler()` as v2, maintain compatibility |
| Performance regression | High | Low | Benchmark before/after, keep hot path unchanged |
| Auth complexity | Medium | Medium | Keep package minimal, defer to external auth |
| Context race conditions | Medium | Low | RWMutex on session data, immutable user objects |

---

## Open Questions

1. **Session Serialization**: Should session data be serializable for Redis backends?
   - **Recommendation**: Defer. Phase 11 for distributed sessions.

2. **Multiple Auth Types**: Support both JWT and cookie sessions?
   - **Recommendation**: Yes. `OnSessionStart` is generic, works with any auth.

3. **Built-in OAuth**: Include OAuth providers?
   - **Recommendation**: No. Document Goth integration instead.

4. **Upload Middleware**: Should upload handler use Vango middleware?
   - **Recommendation**: No. It's pure HTTP, use standard middleware.

---

## The Golden Artifact: Complete Integration Example

This is the canonical `main.go` that ties **Chi**, **Auth**, **Context Bridge**, and **Vango** together. It proves the architecture works without magic.

```go
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	
	// Vango packages
	"github.com/vango-dev/vango/v2"
	"github.com/vango-dev/vango/v2/pkg/auth"
	"github.com/vango-dev/vango/v2/pkg/toast"
	"github.com/vango-dev/vango/v2/pkg/vdom"
)

// ═══════════════════════════════════════════════════════════════════════════
// 1. DOMAIN TYPES
// ═══════════════════════════════════════════════════════════════════════════

type User struct {
	ID    string
	Email string
	Role  string
}

// ═══════════════════════════════════════════════════════════════════════════
// 2. HTTP AUTH MIDDLEWARE (Layer 1)
// This is standard ecosystem middleware - Chi, Goth, yourauth, etc.
// ═══════════════════════════════════════════════════════════════════════════

type contextKey string
const userContextKey contextKey = "http_user"

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate checking a cookie or JWT
		cookie, err := r.Cookie("session_token")
		if err == nil && cookie.Value == "valid_token" {
			// Success: Put User into HTTP Context
			user := User{ID: "u_123", Email: "alice@example.com", Role: "admin"}
			ctx := context.WithValue(r.Context(), userContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		// Continue even if anon (pages might handle it differently)
		next.ServeHTTP(w, r)
	})
}

func main() {
	// ═══════════════════════════════════════════════════════════════════════
	// 3. VANGO CONFIGURATION (The Context Bridge)
	// ═══════════════════════════════════════════════════════════════════════
	
	app := vango.New(vango.Config{
		// This runs ONCE during WebSocket Upgrade
		OnSessionStart: func(httpCtx context.Context, s *vango.Session) {
			// ═══════════════════════════════════════════════════════════════
			// THE BRIDGE: Copy data from dying HTTP context to living session
			// ═══════════════════════════════════════════════════════════════
			if val := httpCtx.Value(userContextKey); val != nil {
				// Use the auth package helper for type-safe storage
				auth.Set(s, val.(User))
			}
			
			// Hydrate other infrastructure data
			if reqID := middleware.GetReqID(httpCtx); reqID != "" {
				s.SetString("request_id", reqID)
			}
		},
	})

	// ═══════════════════════════════════════════════════════════════════════
	// 4. THE ROUTER (Chi)
	// ═══════════════════════════════════════════════════════════════════════
	
	r := chi.NewRouter()

	// Stack 1: Infrastructure Middleware (runs ONCE per session)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(AuthMiddleware) // Populates http_user in context

	// Traditional API Routes (standard HTTP)
	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})
	
	r.HandleFunc("/static/*", http.StripPrefix("/static/", 
		http.FileServer(http.Dir("public"))).ServeHTTP)

	// Mount Vango (handles all page routes via WebSocket)
	r.Handle("/*", app.Handler())

	log.Println("Listening on :3000")
	http.ListenAndServe(":3000", r)
}

// ═══════════════════════════════════════════════════════════════════════════
// 5. A VANGO PAGE COMPONENT
// Note: Returns error, allowing ErrorBoundary to handle 401s
// ═══════════════════════════════════════════════════════════════════════════

func Dashboard(ctx vango.Ctx) (vango.Component, error) {
	// A. Guard Clause (Uses Vango Session, NOT the dead HTTP context)
	user, err := auth.Require[User](ctx)
	if err != nil {
		// vango.ErrUnauthorized triggers redirect in ErrorBoundary
		return nil, err 
	}

	// B. Render
	return vango.Func(func() *vdom.VNode {
		// Interactive state
		count := vango.Signal(0)

		return vdom.Div(
			vdom.H1(vdom.Text("Welcome, " + user.Email)),
			vdom.P(vdom.Textf("Role: %s", user.Role)),
			vdom.Button(
				vdom.OnClick(func() { 
					count.Inc()
					// Toast using generic event bus (no protocol changes!)
					toast.Success(ctx, "Counted!")
				}),
				vdom.Textf("Clicks: %d", count()),
			),
		)
	}), nil
}
```

### What This Proves

1. **Ecosystem Compatibility**: Standard Chi router, standard HTTP middleware
2. **Context Bridge Works**: User data flows from HTTP → Session → Components
3. **Type Safety**: `auth.Require[User](ctx)` returns typed user or error
4. **Zero Magic**: Every step is explicit and visible
5. **Toast Works**: `toast.Success` uses `ctx.Emit`, no protocol changes

---

*Last Updated: 2024-12-10*
*Status: Proposed*
*Dependencies: Phases 1-9 Complete*
