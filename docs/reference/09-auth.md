# Authentication

Vango provides type-safe authentication helpers that integrate with any auth system.

## The Two-Layer Architecture

Vango uses a dual-layer approach:

1. **Layer 1 (HTTP)**: Traditional middleware validates JWT/cookies during initial request
2. **Context Bridge**: Data transfers from HTTP context to persistent Vango session
3. **Layer 2 (WebSocket)**: Vango middleware guards event handlers

```go
// main.go - Complete auth integration
app := vango.New(vango.Config{
    OnSessionStart: func(httpCtx context.Context, session *vango.Session) {
        // THE CONTEXT BRIDGE: Copy user from dying HTTP context
        if user := myauth.UserFromContext(httpCtx); user != nil {
            auth.Set(session, user)
        }
    },
})

r := chi.NewRouter()
r.Use(myauth.JWTMiddleware)  // Your HTTP auth middleware
r.Handle("/*", app.Handler())
```

## Type-Safe User Access

```go
import "github.com/vango-dev/vango/v2/pkg/auth"

// Get user if authenticated (returns zero value if not)
user, ok := auth.Get[*models.User](ctx)
if !ok {
    return renderGuestView()
}

// Require user or return error (use for protected pages)
user, err := auth.Require[*models.User](ctx)
if err != nil {
    return nil, err  // Error boundary handles 401
}

// Quick check without knowing type
if auth.IsAuthenticated(ctx) {
    // Show logout button
}
```

## Route Middleware

```go
// app/routes/admin/_layout.go
func Middleware() []router.Middleware {
    return []router.Middleware{
        auth.RequireAuth,  // Must be logged in
        auth.RequireRole(func(u *models.User) bool {
            return u.Role == "admin"
        }),
    }
}
```

## Logout

```go
func Logout(ctx vango.Ctx) error {
    auth.Clear(ctx.Session())
    ctx.Navigate("/")
    return nil
}
```

## Debug Mode

Enable `server.DebugMode = true` to get warnings when:
- You store `User` but request `*User` (common mistake)
- You store unserializable types (func, chan)
