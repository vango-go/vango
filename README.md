# Vango V2

**A server-driven web framework for Go** — Build interactive web applications without writing JavaScript.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

---

## What is Vango?

Vango is a **server-driven UI framework** inspired by Phoenix LiveView. Components run on the server, and the browser receives binary patches over WebSocket — no JavaScript required.

```go
func Counter() vango.Component {
    return vango.Func(func() *vango.VNode {
        count := vango.Signal(0)
        
        return Div(
            H1(Textf("Count: %d", count())),
            Button(OnClick(count.Inc), Text("+")),
            Button(OnClick(count.Dec), Text("-")),
        )
    })
}
```

### Key Features

- **🚀 One Language** — Go from database to DOM
- **📦 Tiny Client** — 9.56 KB gzipped (no React, no Vue)
- **⚡ Sub-millisecond Updates** — Binary protocol + efficient diffing
- **🔒 Secure by Default** — State lives on the server
- **🔄 Real-time** — WebSocket patches, not full page reloads

> **Note**: WASM client-side mode is planned but **not yet implemented**. All components currently run on the server.

---

## Quick Start

```bash
# Install CLI
go install vango.dev/cli/vango@latest

# Create project
vango create my-app
cd my-app

# Run dev server
vango dev
```

Open http://localhost:3000

### Project Structure

```
my-app/
├── app/
│   ├── routes/       # File-based routing
│   │   └── index.go  # → /
│   └── components/   # Shared components
├── public/           # Static assets
└── vango.json        # Config
```

---

## Performance

Vango V2 exceeds performance targets by **10-50x** in most categories.

### Bundle Size

| Framework | JS Bundle (gzip) |
|-----------|------------------|
| **Vango V2** | **9.56 KB** |
| htmx | ~14 KB |
| React + ReactDOM | ~45 KB |
| Next.js runtime | ~80+ KB |

### Server Performance

| Metric | Result |
|--------|--------|
| Memory per session | **24.76 KB** (target: <50 KB) |
| 10K concurrent sessions | ~247 MB |
| Event encode | ~50 ns |
| Diff 1000 nodes | ~25 µs |
| SSR 1000 list items | 241 µs |

### End-to-End Latency

| Metric | Target | Actual |
|--------|--------|--------|
| P50 RTT | <50 ms | **1.2 ms** |
| P95 RTT | <75 ms | **5.1 ms** |
| P99 RTT | <100 ms | **8.2 ms** |

*Measured on localhost. Real-world adds network latency (~30ms).*

### Comparison

| Feature | Vango | Next.js | Phoenix LiveView |
|---------|-------|---------|------------------|
| Server-rendered | ✅ | ✅ | ✅ |
| Server-driven updates | ✅ | ❌ | ✅ |
| Binary protocol | ✅ | ❌ | ❌ (JSON) |
| Client JS required | ~10 KB | ~80+ KB | ~30 KB |
| Direct DB access | ✅ | API layer | ✅ |
| Language | Go | JavaScript | Elixir |

---

## Documentation

- [Getting Started](docs/getting-started/01-introduction.md)
- [Concepts](docs/concepts/01-philosophy.md)
- [API Reference](docs/reference/)
- [Full Benchmark Report](build_docs/BENCHMARK_REPORT.md)

---

## Architecture

```
┌─────────────────────────────────────┐
│           BROWSER                   │
│  ┌───────────────────────────────┐  │
│  │  Thin Client (9.56 KB)        │  │
│  │  Event Capture → Patch Apply  │  │
│  └───────────────┬───────────────┘  │
│                  │ WebSocket        │
└──────────────────┼──────────────────┘
                   ▼
┌──────────────────────────────────────┐
│           SERVER                     │
│  Session → Signals → Diff → Patches  │
│              ↓                       │
│  Direct: Database, Cache, Services   │
└──────────────────────────────────────┘
```

---

## Status

| Feature | Status |
|---------|--------|
| Server-driven mode | ✅ Production-ready |
| Binary protocol | ✅ Complete |
| SSR + Hydration | ✅ Complete |
| Routing | ✅ Complete |
| State (Signals) | ✅ Complete |
| Forms & Validation | ✅ Complete |
| Client Hooks | ✅ Complete |
| VangoUI Components | ✅ Available |
| **WASM client mode** | ⏳ **Not implemented** |

---

## License

MIT
