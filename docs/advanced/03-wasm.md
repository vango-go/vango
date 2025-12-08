# WASM Runtime

> **Status: Experimental**
> This feature is in early development.

## When to Use

- **Offline-first**: PWAs that must work without network
- **Latency-critical**: Drawing, music, games
- **Heavy computation**: Image processing, data viz

## Enabling WASM

**Full WASM mode:**
```json
// vango.json
{ "mode": "wasm" }
```

**Hybrid mode** (specific components only):
```go
func DrawingCanvas() vango.Component {
    return vango.ClientRequired(func() *vango.VNode {
        // Runs in WASM
    })
}
```

## Considerations

- ~300KB bundle size
- Same component API
- State lives in browser memory
- No server connection required
