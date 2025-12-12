# Middleware Reference

Vango has two layers of middleware for different concerns.

## Layer 1: HTTP Middleware

Standard `func(http.Handler) http.Handler` middleware runs once per session (during connection):

```go
r := chi.NewRouter()
r.Use(middleware.Logger)       // Chi's logger
r.Use(middleware.Recoverer)    // Chi's panic recovery
r.Use(cors.Handler(...))       // CORS for cross-origin
r.Use(myauth.JWTMiddleware)    // Your auth validation
r.Handle("/*", app.Handler())
```

## Layer 2: Vango Event Middleware

Vango middleware runs on **every event** (clicks, inputs, etc). Use for:
- Authorization guards
- Request-scoped logging
- Rate limiting

### Route-Level Middleware

```go
// app/routes/admin/_layout.go
func Middleware() []router.Middleware {
    return []router.Middleware{
        auth.RequireAuth,
        router.Logger(slog.Default()),
    }
}
```

### Built-in Middleware

#### Logger

```go
router.Logger(slog.Default())
```

Logs each event with session ID, path, duration, and errors.

#### Recover

```go
router.Recover(func(ctx server.Ctx, recovered any) {
    slog.Error("panic in handler", "panic", recovered)
    toast.Error(ctx, "Something went wrong")
})
```

Recovers from panics in the event loop.

#### RateLimit

```go
router.RateLimit(10)  // 10 events per second per session
```

Prevents abuse from rapid-fire events.

#### Timeout

```go
router.Timeout(5 * time.Second)
```

Limits handler execution time.

#### WithValue

```go
router.WithValue("tenant", tenant)
```

Sets request-scoped values accessible via `ctx.Value("tenant")`.

### Custom Middleware

```go
func AuditLog() router.Middleware {
    return router.MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
        start := time.Now()
        err := next()
        
        user, _ := auth.Get[*User](ctx)
        slog.Info("audit",
            "user", user.Email,
            "path", ctx.Path(),
            "duration", time.Since(start),
        )
        
        return err
    })
}
```

### Composition

```go
// Chain multiple middleware
mw := router.Chain(
    auth.RequireAuth,
    router.Logger(log),
    router.Recover(onPanic),
)

// Skip middleware for specific routes
mw := router.Skip(auth.RequireAuth, "/public", "/health")

// Apply middleware only to specific routes
mw := router.Only(auth.RequireAuth, "/admin/*", "/settings/*")
```
