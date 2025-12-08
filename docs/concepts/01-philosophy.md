# Philosophy

## The Problem with Modern Web Development

Modern web development is fragmented:

- **Two languages**: JavaScript on frontend, another language on backend
- **Two data models**: DTOs, JSON serialization, API contracts
- **Two state systems**: Client state, server state, synchronization hell
- **Heavy bundles**: 200KB+ of JavaScript before interactivity
- **Complex toolchains**: Webpack, Babel, TypeScript, bundlers, transpilers

What if we could write web applications in a single language, with direct access to the database, instant interactivity, and no bundle size concerns?

## The Vango Approach

Vango is a **server-driven web framework** where:

1. **Components run on the server** by default
2. **UI updates flow as binary patches** over WebSocket
3. **The client is a thin renderer** (~12KB)
4. **You write Go everywhere** — no JavaScript required
5. **WASM is available** for offline or latency-sensitive features

This is similar to Phoenix LiveView (Elixir) or Laravel Livewire (PHP), but with Go's performance, type safety, and concurrency model.

## Design Principles

| Principle | Meaning |
|-----------|---------|
| **Server-First** | Most code runs on the server. Client is minimal. |
| **One Language** | Go from database to DOM. No context switching. |
| **Type-Safe** | Compiler catches errors. No runtime surprises. |
| **Instant Interactive** | SSR means no waiting for bundles. |
| **Progressive Enhancement** | Works without JS, enhanced with WebSocket. |
| **Escape Hatches** | WASM and JS islands when you need them. |

## When to Use Vango

**Ideal for:**
- CRUD applications (admin panels, dashboards)
- Collaborative apps (project management, documents)
- Data-heavy interfaces (analytics, reporting)
- Real-time features (chat, notifications, live updates)
- Internal tools (where Go backend teams own the frontend)

**Consider alternatives for:**
- Offline-first applications (use WASM mode or different framework)
- Extremely latency-sensitive UIs (drawing apps, games)
- Static content sites (use a static site generator)
