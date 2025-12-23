# Vango V2: Hybrid Runtime & Offline Architecture

This document serves as the comprehensive design specification for two advanced capabilities in Vango V2: **WASM Islands** (seamless client-side code execution) and **Full Offline Mode** (portable server architecture).

These designs are based on the core thesis that Vango is a **Server-Driven Framework** that treats "the client" not as a separate codebase, but as a deployment target for Go logic.

---

## Part 1: WASM Islands

**Objective:** Enable developers to run Go component logic directly in the browser for high-frequency interaction, heavy computation, or offline parity, without sacrificing the benefits of Server-Side Rendering (SSR).

### 1.1 Architecture: "The Progressive Island"

Standard "Island Architectures" (like Astro) often require hydration that blocks interactivity. Vango uses a **Progressive Upgrade** pattern ensures the app is usable immediately via server roundtrips, then seamlessly upgrades to local WASM execution.

#### The Four Phases
1.  **Server-Active (Phase 1)**: The component loads as standard SSR HTML. Interactions work immediately via the Vango Thin Client (WebSocket roundtrips).
2.  **Background Loading (Phase 2)**: The browser lazily downloads the `islands.wasm` bundle (containing *all* island logic) during network idle time.
3.  **Silent Handoff (Phase 3)**:
    *   WASM runtime boots.
    *   It scans the DOM for `[data-island]` elements.
    *   For each island, it deserializes the `data-state` (maintained by the server).
    *   It attaches local event listeners.
4.  **WASM-Active (Phase 4)**: The next user interaction triggers the WASM listener. This listener calls `event.stopPropagation()`, preventing the event from bubbling to the Thin Client. The server is effectively disconnected for that specific interaction loop.

### 1.2 Developer Experience (DX)

The API is designed to look like a standard Vango component wrapper. The developer writes a single Go function that handles both Server and Client execution contexts.

#### The `vango.Island` Primitive
```go
// pkg/components/graph.go

// 1. The Wrapper (Public API)
func KnowledgeGraph(nodes []Node) *vdom.VNode {
    // "KnowledgeGraph" -> Stable ID for registry
    // renderGraph      -> The logic function
    // nodes            -> Props (auto-serialized)
    return vango.Island("KnowledgeGraph", renderGraph, nodes)
}

// 2. The Implementation (Shared Logic)
func renderGraph(ctx *vango.IslandCtx, nodes []Node) *vdom.VNode {
    // Client-side local state
    // On Server: Tracks state for SSR
    // On Client: Real reactive Signal
    zoom := ctx.Signal(1.0)
    
    return vdom.Canvas(
        vdom.OnWheel(func(e vdom.WheelEvent) {
             // Runs on Server (Phase 1) OR Client (Phase 4)
             zoom.Set(zoom.Get() + e.DeltaY)
        }),
    )
}
```

### 1.3 Implementation Strategy

#### Handoff Mechanics: Event Interception
Rationale: Avoids complex "pause/resume" coordination protocols between JS and WASM.

1.  **Thin Client**: Listens on `document` (Event Delegation).
2.  **WASM Island**: Attaches listener to the `div`.
3.  **Action**: WASM handler calls `e.stopPropagation()`.
    *   *Result*: The event never bubbles to `document`. The Thin Client never sees it. The server never hears about it.
    *   *Safety*: The Thin Client is modified to check `if (target.closest('.wasm-active')) return;` when processing incoming server patches, preventing race conditions where old server patches overwrite new client state.

#### Build System Integration (`vango gen`)
To support Go's static linking, we generate a registry entrypoint.

*   **Command**: `vango gen islands` (runs automatically during `vango dev`).
*   **Behavior**: Scans codebase for `vango.Island` calls.
*   **Output**: `cmd/islands/main.go`.

```go
// cmd/islands/main.go
package main
import (
    "github.com/vango/client"
    "myapp/pkg/components"
)
func main() {
    client.Register("KnowledgeGraph", components.RenderGraph)
    client.Hydrate()
}
```

### 1.4 State Synchronization
*   **Server -> Client**: The server renders `data-state` attributes on the Island container. When WASM boots, it reads this initial state.
*   **Real-time Sync**: `vango.SharedSignal` allows the WASM Island to subscribe to server pushes (e.g., for chat apps). `ctx.SendToServer()` allows the Island to invoke server actions cleanly.

---

## Part 2: Full Offline Mode

**Thesis**: "The Portable Server"
Most frameworks build a separate "Offline Client" that syncs with an API. Vango simply moves the **Go Server** to the client device.

### 2.1 Web Implementation (PWA)

*   **Mechanism**: **Service Worker Interception**.
*   **Execution**: The entire Go application (Router, Handlers, Database Logic) is compiled to WASM.
*   **Storage**: **SQLite** running on **OPFS** (Origin Private File System). This provides a high-performance, file-backed SQL database in the browser.
*   **Flow**:
    1.  Browser makes HTTP request (`/todos`).
    2.  Service Worker intercepts request.
    3.  Service Worker passes request to WASM Server instance.
    4.  WASM Server routes request, queries SQLite (OPFS), renders HTML.
    5.  Response returned to main thread.

### 2.2 Native Implementation (Mobile/Desktop)

*   **Targets**: iOS, Android, macOS, Windows, Linux.
*   **Mechanism**: **Embedded Sidecar**.
    *   **Mobile**: Go code compiled as a shared library (`.framework`/`.aar`) via `gomobile`. Runs a local HTTP server on a random port.
    *   **Desktop**: Go binary runs local server + webview (Tauri/Wails pattern).
*   **Storage**: Standard SQLite on device filesystem.

### 2.3 The "Universal Sync" Primitive

To solve the data synchronization problem between "Local Server" and "Cloud Server", Vango provides an opinionated sync primitive.

#### `vango.SyncModel`
```go
type Todo struct {
    ID        string `vango:"pk"`
    Title     string
    UpdatedAt int64  `vango:"version"`
}

func init() {
    // Automatically adds Sync logic to this model
    vango.RegisterSyncModel(&Todo{})
}
```

*   **Behavior**:
    *   **Change Data Capture (CDC)**: Vango's ORM layer records a "Mutation Log" (Create, Update, Delete) locally.
    *   **Replication**: When online, the Local Server pushes the Mutation Log to the Cloud Server.
    *   **Conflict Resolution**: Last-Write-Wins (LWW) based on `UpdatedAt` timestamp, or custom resolution logic.

### 2.4 Build Targets

The Vango CLI manages the complexity of targeting these environments:

| Command | Target | Storage | Runtime |
|---------|--------|---------|---------|
| `vango build` | Cloud Server | Postgres | Linux/AMD64 |
| `vango build --pwa` | Browser | SQLite (OPFS) | WASM (Service Worker) |
| `vango build --ios` | iPhone | SQLite (FS) | ARM64 Native Lib |
| `vango build --desktop` | Mac/PC | SQLite (FS) | Native Binary |

---

## Summary of Benefits

1.  **Unified Codebase**: Logic, validation, and rendering code is identical across Cloud, Client, and Mobile.
2.  **Zero Latency Upgrade**: WASM Islands provide instant interactivity without blocking initial load.
3.  **True Offline**: Not just "cached API responses", but full application logic availability offline.
4.  **Go Everywhere**: Leveraging standard Go features (and the standard Go compiler) simplifies the mental model significantly compared to splitting logic between Go backend and JS frontend.
