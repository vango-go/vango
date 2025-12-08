# Performance

## Server Resource Usage

| App Type | Memory / Session | 10k Users |
|----------|------------------|-----------|
| Simple (blog) | ~10 KB | 100 MB |
| Dashboard | ~100 KB | 1 GB |
| Complex Tool | ~500 KB | 5 GB |

## Optimization Strategies

1. **Stateless pages**: Don't use signals for read-only content
2. **Session timeout**: Configure cleanup for inactive sessions
3. **External stores**: Use Redis for large datasets

```go
vango.Config{
    SessionTimeout: 5 * time.Minute,
    SessionMaxAge:  1 * time.Hour,
}
```

## Latency Guidelines

| Round-trip | User Experience |
|------------|-----------------|
| < 50ms | Feels instant |
| < 100ms | Noticeable but good |
| > 150ms | Use optimistic updates |

**Best practice**: Deploy servers close to users.

## Bundle Size

| Mode | Size (gzipped) |
|------|----------------|
| Server-Driven | ~12 KB |
| + Hooks | ~15 KB |
| Full WASM | ~300 KB |
