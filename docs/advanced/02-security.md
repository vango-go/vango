# Security

Vango provides security by design with secure defaults.

## Secure Defaults (v2.1+)

Vango ships with **secure defaults** that prevent common vulnerabilities:

| Setting | Default | Notes |
|---------|---------|-------|
| `CheckOrigin` | Same-origin only | Cross-origin WS rejected |
| CSRF | Warning if disabled | Required in v3.0 |
| `on*` attributes | Stripped unless handler | Prevents XSS injection |

## XSS Prevention

### Text Escaping

All text is escaped by default:

```go
Text("<script>")  // Renders as &lt;script&gt;
```

Only use `DangerouslySetInnerHTML` for trusted content.

### Attribute Sanitization

Event handler attributes (`onclick`, `onmouseover`, etc.) are automatically filtered:

```go
// This is BLOCKED - attribute stripped during render
Attr("onclick", "alert(1)")

// This is SAFE - uses internal event handler
OnClick(myHandler)
```

> **Note**: The filter is case-insensitive. `ONCLICK`, `onClick`, and `onclick` are all blocked.

## CSRF Protection

Enable CSRF protection in production:

```go
vango.Config{
    CSRFSecret: []byte("your-32-byte-secret-key-here!!"),
}
```

CSRF tokens are automatically:
- Generated via `server.GenerateCSRFToken()`
- Set as cookie via `server.SetCSRFCookie(w, token)`
- Validated on WebSocket handshake

> **Warning**: If `CSRFSecret` is nil, a warning is logged on startup. This will become a hard error in v3.0.

## WebSocket Origin Validation

By default, Vango rejects cross-origin WebSocket connections:

```go
// Default behavior - same-origin only
config := server.DefaultServerConfig()
// config.CheckOrigin = SameOriginCheck (secure default)

// Explicit cross-origin (dev only!)
config.CheckOrigin = func(r *http.Request) bool { return true }
```

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

Vango is auth-agnostic. Use any auth system via the Context Bridge:

```go
app := vango.New(vango.Config{
    OnSessionStart: func(httpCtx context.Context, session *vango.Session) {
        if user := myauth.UserFromContext(httpCtx); user != nil {
            auth.Set(session, user)
        }
    },
})
```

See [Authentication Reference](../reference/09-auth.md) for complete guide.

## Event Handler Safety

Event handlers are server-side function references. The client only sends:

```
{hid: "h42", type: 0x01}  // Binary event
```

Users cannot:
- Execute arbitrary functions
- Access other users' handlers
- Inject JavaScript

## Protocol Security

The binary protocol includes allocation limits to prevent DoS:

| Limit | Value | Purpose |
|-------|-------|---------|
| Max string/bytes | 4MB | Prevent OOM |
| Max collection | 100K items | Prevent CPU exhaustion |
| Max hook depth | 64 levels | Prevent stack overflow |
| Hard cap | 16MB | Absolute ceiling |

### Upload DoS Prevention

Uploads are protected at the HTTP layer:

```go
// Request body limited BEFORE parsing
config := &upload.Config{
    MaxFileSize: 10 << 20, // 10MB limit
}
http.Handle("/upload", upload.HandlerWithConfig(store, config))
```

## Debug Mode

Enable debug mode to catch security issues early and get verbose logging:

```go
server.DebugMode = true
```

This will:
- Log handler registrations, event processing, and render cycles
- Warn on type mismatches in `auth.Get`
- Panic if storing unserializable types (func, chan)

> **Note**: Debug logs are completely silent when `DebugMode = false` (default).

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

