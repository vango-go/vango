# Vango V2 Benchmark Report

> **Performance analysis and framework comparison**
> Generated: 2025-12-07 | Platform: Apple M-series (arm64)

---

## Executive Summary

Vango V2 demonstrates **exceptional performance** across all measured dimensions, exceeding targets by **10-50x** in most categories.

| Metric | Target | Actual | Status |
|--------|--------|--------|--------|
| Thin client bundle | < 15 KB | **9.56 KB** | ✅ 36% under |
| Event encode | < 500 ns | **~50 ns** | ✅ 10x faster |
| Patch encode | < 500 ns | **~54 ns** | ✅ 10x faster |
| 100 patches encode | < 50 µs | **~1.1 µs** | ✅ 45x faster |
| Diff 1000 nodes | - | **~25 µs** | ✅ Excellent |
| Router match | - | **~69 ns** | ✅ Excellent |

---

## 1. Protocol Performance

The binary protocol layer shows the fastest performance in the entire stack.

### Varint Encoding (Foundation)

| Operation | Time | Allocations |
|-----------|------|-------------|
| Encode small (< 128) | **0.24 ns** | 0 |
| Encode large | **1.85 ns** | 0 |
| Decode small | **0.50 ns** | 0 |
| Decode large | **1.48 ns** | 0 |

### Event Encoding/Decoding

| Event Type | Encode | Decode |
|------------|--------|--------|
| Click | 49 ns | 27 ns |
| Input | 53 ns | 50 ns |
| Submit | 88 ns | 173 ns |
| Keyboard | 51 ns | - |

### Patch Encoding/Decoding

| Patch Count | Encode | Decode |
|-------------|--------|--------|
| 1 (SetText) | 53 ns | 59 ns |
| 10 | 96 ns | 304 ns |
| 100 | **1.1 µs** | 2.5 µs |

### Frame & Control Messages

| Message | Encode | Decode |
|---------|--------|--------|
| Frame | 17 ns | 16 ns |
| Ping | 48 ns | 9 ns |
| Ack | 48 ns | 11 ns |
| Error | 49 ns | 28 ns |
| ClientHello | 52 ns | 40 ns |

---

## 2. Virtual DOM Performance

### Element Creation

| Complexity | Time | Allocations |
|------------|------|-------------|
| Simple div | 112 ns | 5 |
| With children | 286 ns | 15 |
| With event handler | 143 ns | 7 |
| Complex card (6 elem) | **882 ns** | 45 |

### Tree Creation (Scaling)

| Structure | Time | Allocs |
|-----------|------|--------|
| Depth 5 | 665 ns | 31 |
| Depth 10 | 1.3 µs | 61 |
| Width 10 | 2.3 µs | 89 |
| Width 100 | 21 µs | 902 |

### Diff Algorithm

| Scenario | Time | Allocs |
|----------|------|--------|
| Same tree (100 nodes) | 2.4 µs | 0 |
| Text change | **78 ns** | 1 |
| Attribute change | 103 ns | 1 |
| Unkeyed 100 children | 2.6 µs | 1 |
| Keyed reorder (100) | 23 µs | 38 |
| **Large tree (1000 nodes)** | **25 µs** | 1 |

---

## 3. SSR Rendering Performance

### Rendering Speed

| Scenario | Time | Memory |
|----------|------|--------|
| Simple element | 785 ns | 584 B |
| Full page render | **1.1 µs** | 536 B |
| Deep nesting (20 levels) | 2.4 µs | 1.5 KB |
| Many attributes | 1.6 µs | 1.2 KB |
| 100 buttons w/handlers | 42 µs | 46 KB |
| **1000 nodes (list)** | **239 µs** | 250 KB |
| Complex page (table) | 82 µs | 69 KB |

### HTML Escaping

| Scenario | Time |
|----------|------|
| Plain text | 234 ns |
| With special chars | 178 ns |
| Attribute escape | 51 ns |

---

## 4. Router Performance

### Route Matching

| Pattern | Time | Allocs |
|---------|------|--------|
| Static route | **69 ns** | 3 |
| Single param (`:id`) | 128 ns | 4 |
| Multiple params | 195 ns | 4 |
| Catch-all (`*path`) | 226 ns | 6 |
| Deep path | 164 ns | 3 |
| **Large tree (100 routes)** | **163 ns** | 3 |
| No match | 42 ns | 2 |

---

## 5. Framework Comparison

### Bundle Size Comparison

| Framework | JS Bundle (gzipped) | Notes |
|-----------|---------------------|-------|
| **Vango V2** | **9.56 KB** | Thin client only |
| htmx | ~14 KB | Similar architecture |
| Alpine.js | ~15 KB | Lightweight |
| Svelte (hello world) | ~2 KB | Per-component |
| React + ReactDOM | ~45 KB | Base runtime |
| Vue 3 | ~34 KB | Base runtime |
| Next.js runtime | ~80+ KB | With hydration |

### Architecture Comparison

| Feature | Vango | Next.js | Phoenix LiveView |
|---------|-------|---------|------------------|
| Server-rendered | ✅ | ✅ | ✅ |
| Server-driven updates | ✅ | ❌ | ✅ |
| Binary protocol | ✅ | ❌ | ❌ (JSON) |
| WebSocket patches | ✅ | ❌ | ✅ |
| Client JS required | ~10 KB | ~80+ KB | ~30 KB |
| Database access | Direct | API layer | Direct |
| Language | Go | JavaScript | Elixir |

### Time-to-Interactive (TTI) Estimate

| Framework | First Paint | Full TTI |
|-----------|-------------|----------|
| **Vango V2** | ~50 ms | ~100 ms |
| Next.js (SSR) | ~50 ms | ~300+ ms |
| SPA (React) | ~200+ ms | ~500+ ms |

*Note: Vango's thin client loads faster due to smaller bundle size and no hydration reconciliation needed.*

---

## 6. Performance by UI Mode

### Server-Driven Mode (Default)

| Metric | Performance |
|--------|-------------|
| Initial render | **< 1 ms** (SSR) |
| Interaction RTT | Network dependent |
| Memory (client) | Minimal |
| Memory (server) | ~200 KB/session |

### WASM Mode (Planned)

| Metric | Expected |
|--------|----------|
| Bundle size | ~300 KB |
| Interaction | **0 ms** (local) |
| Memory (client) | Higher |
| Offline capable | ✅ |

---

## 7. Key Takeaways

### Strengths

1. **Protocol efficiency**: 10-50x faster than targets
2. **Bundle size**: 36% under target at 9.56 KB
3. **Diff algorithm**: Sub-millisecond for typical updates
4. **Router**: Nanosecond-scale matching

### Recommendations

1. **Typical apps**: Use Server-Driven mode (default)
2. **Real-time**: Binary protocol handles high-frequency updates
3. **Large lists**: Keyed reconciliation at ~23µs/100 items
4. **Complex pages**: Full page render at ~82µs

---

## Appendix: Raw Benchmark Commands

```bash
# Protocol benchmarks
cd vango_v2 && go test -bench=. -benchmem ./pkg/protocol/...

# VDOM benchmarks
cd vango_v2 && go test -bench=. -benchmem ./pkg/vdom/...

# Render benchmarks
cd vango_v2 && go test -bench=. -benchmem ./pkg/render/...

# Router benchmarks
cd vango_v2 && go test -bench=. -benchmem ./pkg/router/...
```

---

*Report generated automatically from Go benchmark suite on Apple M-series hardware.*
