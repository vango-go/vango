# Security

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

## Event Handler Safety

Event handlers are server-side function references. The client only sends:

```json
{"hid": "h42"}
```

Users cannot:
- Execute arbitrary functions
- Access other users' handlers
- Modify handler behavior

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
