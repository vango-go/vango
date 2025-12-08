# Architecture

This document explains Vango's high-level system design.

## The Three Modes

Vango supports three rendering modes:

| Mode | Description | Client Size | Requires Connection |
|------|-------------|-------------|---------------------|
| **Server-Driven** (Default) | Components run on server | ~12KB | Yes |
| **Hybrid** | Most on server, some on client | 12KB + partial WASM | Yes |
| **WASM** | All in browser | ~300KB | No |

## Server-Driven Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                           BROWSER                                │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │                    Thin Client (12KB)                      │  │
│  │  ┌─────────────┐  ┌──────────────┐  ┌─────────────────┐    │  │
│  │  │   Event     │  │   Patch      │  │   Optimistic    │    │  │
│  │  │   Capture   │──│   Applier    │──│   Updates       │    │  │
│  │  └─────────────┘  └──────────────┘  └─────────────────┘    │  │
│  └──────────────────────────┬─────────────────────────────────┘  │
│                             │ WebSocket (Binary)                 │
└─────────────────────────────┼────────────────────────────────────┘
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                           SERVER                                │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    Vango Runtime                        │    │
│  │  Session Manager → Component Tree → Diff Engine         │    │
│  │       ↓                  ↓                ↓             │    │
│  │  Signal Store      Event Router      Patch Encoder      │    │
│  └─────────────────────────────────────────────────────────┘    │
│              ↓                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │        Direct Access: Database, Cache, Services         │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

## Request Lifecycle

**Initial Page Load:**
1. Browser requests `GET /projects/123`
2. Server matches route, renders component to HTML (SSR)
3. HTML sent immediately (user sees content)
4. Thin client JS loads (~12KB)
5. WebSocket connection established
6. Page is now interactive

**User Interaction:**
1. User clicks a button
2. Thin client sends binary event: `{type: CLICK, hid: "h42"}`
3. Server finds handler, runs it
4. Signals update → component re-renders
5. Diff engine creates patches
6. Patches sent: `[{SET_TEXT, "h17", "Done"}]`
7. Client applies patches to DOM
8. Total time: ~50-80ms

## VDOM & Hydration IDs

Vango uses a custom Virtual DOM that mirrors the browser's DOM on the server.

### Node Types (VKind)
The VKind enum aligns with the client's binary protocol:
- **Element (1)**: HTML tags (`<div>`, `<button>`)
- **Text (2)**: Text content
- **Fragment (3)**: Layout wrappers

### Universal HIDs
To support dynamic updates (insertions, reordering), **all element nodes** are assigned a unique Hydration ID (`hid`) during SSR. This ensures that any element can be targeted as a parent for patches.

- **Assignment**: `h1`, `h2`, `h3`... generated sequentially.
- **Stability**: HIDs are preserved across re-renders for the same component instance, preventing handler mismatches.

## Same Components, Different Modes

The same code works in all modes:

```go
func Counter(initial int) vango.Component {
    return vango.Func(func() *vango.VNode {
        count := vango.Signal(initial)
        return Div(
            H1(Textf("Count: %d", count())),
            Button(OnClick(count.Inc), Text("+")),
        )
    })
}
```

| Mode | Signal Location | Event Handling | DOM Updates |
|------|-----------------|----------------|-------------|
| Server-Driven | Server memory | Server | Binary patches |
| WASM | Browser WASM | Browser WASM | Direct DOM |
| Hybrid | Mixed | Mixed | Mixed |
