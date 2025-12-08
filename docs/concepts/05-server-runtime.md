# Server Runtime

The Vango server runtime manages sessions, events, and DOM patching.

## Sessions

Each browser tab creates a WebSocket connection with its own session:

```go
type Session struct {
    ID         string
    Conn       *websocket.Conn
    Signals    map[uint32]*SignalBase
    Components map[uint32]*ComponentInst
    LastTree   *vdom.VNode
    Handlers   map[string]func()
}
```

**Lifecycle:**
1. WebSocket handshake (validates CSRF, creates session)
2. Initial render (components mount, effects run)
3. Event loop (events → updates → patches)
4. Disconnect (cleanup, session evicted after timeout)

## Event Loop

```go
for event := range session.events {
    handler := session.Handlers[event.HID]
    handler()

    session.renderDirtyComponents()
    patches := vdom.Diff(session.LastTree, session.CurrentTree)
    session.sendPatches(patches)
}
```

## Binary Protocol

Events and patches use a compact binary format.

**Client → Server (Events):**
```
[Type: 1 byte] [HID: varint] [Payload: variable]
```

**Server → Client (Patches):**
```
[Count: varint] [Patch 1] [Patch 2] ...

Each Patch:
[Type: 1 byte] [HID: varint] [Payload: variable]
```

Patch types: `SET_TEXT`, `SET_ATTR`, `INSERT_NODE`, `REMOVE_NODE`, etc.

## Hydration IDs

Every interactive element gets a `data-hid` attribute during SSR:

```html
<button data-hid="h42">Click me</button>
```

This maps the DOM element to its server-side handler.
