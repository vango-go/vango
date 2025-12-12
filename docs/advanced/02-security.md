# Security

Vango provides security by design.

## XSS Prevention

All text is escaped by default:

```go
Text("<script>")  // Renders as &lt;script&gt;
```

Only use `DangerouslySetInnerHTML` for trusted content.

## CSRF Protection

CSRF tokens are automatically:
- Injected into pages via `VangoScripts()`
- Validated on WebSocket handshake
- Validated on form submissions

## Session Security

```go
vango.Config{
    SessionCookie: http.Cookie{
        Secure:   true,
        HttpOnly: true,
        SameSite: http.SameSiteStrictMode,
    },
}
```

## Authentication

Vango is auth-agnostic. Use any auth system (JWT, cookies, OAuth) via the Context Bridge:

```go
app := vango.New(vango.Config{
    OnSessionStart: func(httpCtx context.Context, session *vango.Session) {
        // Copy user from HTTP middleware to persistent session
        if user := myauth.UserFromContext(httpCtx); user != nil {
            auth.Set(session, user)
        }
    },
})
```

See [Authentication Reference](../reference/09-auth.md) for complete guide.

## Event Handler Safety

Event handlers are server-side function references. The client only sends:

```json
{"hid": "h42"}
```

Users cannot:
- Execute arbitrary functions
- Access other users' handlers
- Modify handler behavior

## Session Data Security

Session data is stored in server RAM (not sent to client). For sensitive data:

```go
// Good: Store user ID, fetch sensitive data as needed
session.Set("user_id", user.ID)

// Also good: Store full user object (stays server-side)
auth.Set(session, user)
```

## Debug Mode Validation

Enable debug mode to catch security issues early:

```go
server.DebugMode = true
```

This will:
- Warn on type mismatches in `auth.Get`
- Panic if storing unserializable types (func, chan)

## Input Validation

Always validate on the server:

```go
func CreateProject(ctx vango.Ctx, input Input) (*Project, error) {
    if err := validate.Struct(input); err != nil {
        return nil, vango.BadRequest(err)
    }
    return db.Projects.Create(input)
}
```

## Environment Variables

Use encrypted environment storage for production secrets:

```go
// In production, use Fly secrets or similar
dbURL := os.Getenv("DATABASE_URL")
```
