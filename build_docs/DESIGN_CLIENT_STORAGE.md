# Design Deep Dive: Client-Side Persistence in Vango

> **Problem**: How do we persist user preferences (Theme, Sidebar State) across sessions in a framework where state lives on the server?

---

## 1. The Context: How Vango Works

Vango is a **Server-Driven UI** framework. This means:
1.  **State is on the Server**: All `Signal[T]` values live in process memory (RAM) on the Go server.
2.  **UI is on the Client**: The browser is a "Thin Client" that just renders DOM updates sent by the server.
3.  **The Bridge is Volatile**: The connection is a WebSocket. If the user refreshes the page, the WebSocket disconnects, the server-side session is destroyed, and all state in RAM is lost.

### The Challenge
Users expect certain UI states to survive a refresh:
- "Dark Mode"
- "Sidebar Collapsed"
- "Table Sort Order"
- "Language Preference"

In a traditional SPA (React), these are stored in `localStorage`.
In Vango, the server (where the logic lives) cannot synchronously read `localStorage` (where the data lives).

---

## 2. The Initial Idea: `.Persist()`

The initial spec proposed a "Magic" API inspired by client-side reactivity libraries:

```go
// The aspirational API
var SidebarOpen = vango.Signal(false).Persist(vango.LocalStorage, "sidebar")
```

### Why it was attractive
- **DX**: One liner. Looks just like a standard signal.
- **Familiarity**: Similar to MobX or Zustand middlewares.

### Why it is Flawed (The "Context Gap")

This API implies that a global or component-level variable can "know" the value of `localStorage` at initialization time.

**Fail 1: Timing (The Lie)**
When `vango.Signal(false)` executes on the server, the client hasn't connected yet.
- Server: "I am initializing this component."
- Client: *Still downloading `vango.js`...*
- Result: The signal *must* initialize with `false`. Later, when the client connects, it *might* update to `true`. This causes logic bugs where the code assumes the stored value is available immediately.

**Fail 2: Context (The Void)**
```go
// Global definition
var Theme = vango.Signal("light").Persist(LocalStorage, "theme")
```
Whose localStorage? User A's? User B's?
Without a `vango.Ctx` reference, the server doesn't know *which* connection to talk to.

**Fail 3: Performance (The Flood)**
If `.Persist()` works by sending every `.Set()` to the client, and we have 100 persisted signals, we risk spamming the WebSocket channel with minor state updates that might not even be needed.

---

## 3. The Architecture Constraint

We are bound by the laws of physics:
1.  **Server cannot read Client synchronousy.**
2.  **Cookies** are the only data sent *before* the WebSocket connects (via HTTP headers).
3.  **Handshake** is the first moment the Client can send arbitrary data (via WebSocket).

### Option A: Cookies
Use HTTP Cookies for everything.
- **Pros**: Available on first render (Server Side Rendering works perfectly).
- **Cons**: Adds overhead to *every* HTTP request. 4KB size limit. Clunky API for simple UI state.

### Option B: Lazy Fetch
Wait for component to mount, then ask client.
- **Pros**: Simple server logic.
- **Cons**: Massive "Pop-in". UI renders "Light", then asks client, then swaps to "Dark" 100ms later. Bad UX.

### Option C: Sync-on-Connect (The Winner)
Send specific `localStorage` keys *during the WebSocket handshake*.

---

## 4. The Proposed Solution: "Sync-on-Connect"

This design acknowledges the boundary and bridges it explicitly.

### Part 1: Configuration (The Allowlist)
We must treat the client as untrusted. We don't want to accept *all* localStorage (which could be megabytes). We whitelist keys.

```go
// main.go
app := vango.New(vango.Config{
    // "Please send me these keys when you connect"
    ClientStorageKeys: []string{"theme", "sidebar", "lang"},
})
```

### Part 2: The Handshake
The Thin Client (`vango.js`) sees logic like this:

```javascript
// Client-side
const payload = { type: 'hello' };
if (config.storageKeys) {
    payload.storage = {};
    for (const key of config.storageKeys) {
        payload.storage[key] = localStorage.getItem(key);
    }
}
ws.send(JSON.stringify(payload));
```

The Server receives this and stores it in the `Session` struct *before* any component renders.

### Part 3: The `UseLocalStorage` Hook
Now, inside a component, the data is **already there in memory**.

```go
func Sidebar(ctx vango.Ctx) vango.Component {
    // 1. Init: Reads from Session memory (Instant! No network trip)
    // 2. Write: Sets up Effect to send updates back to client
    isOpen := vango.UseLocalStorage(ctx, "sidebar", false)
    
    return Div(...)
}
```

### Part 4: Developer Experience (Safety)
What if the dev forgets to add the key to the config?
The hook panics in development mode:

```go
if vango.IsDev() && !ctx.IsAllowedKey("sidebar") {
    panic("Key 'sidebar' not in Config.ClientStorageKeys. Add it to main.go!")
}
```

---

## 5. Solving the Corner Cases

### The "Flash of Unstyled Content" (FOUC)
**Problem**: The very first HTML response (SSR) comes before the WebSocket handshake. The server guesses "Light Mode".
**Solution**: A tiny script helper.

```go
Head(
    // Injects <script> that reads localStorage('theme') and sets 
    // document.documentElement.className BEFORE the body renders.
    vango.ScriptInitTheme("theme", "light", "dark"),
)
```
This bypasses the framework state for the critical milliseconds before JS loads.

### Complex Types
The Hook handles JSON serialization automatically.

```go
type Filters struct { Search string; Sort string }

// vango handles JSON.marshal/unmarshal automatically
filters := vango.UseLocalStorage(ctx, "filters", Filters{}) 
```

---

## 6. Final Verdict

This design honors Vango's pillars:

1.  **Functionality**: It actually works. It handles the user-specific context correctly.
2.  **Performance**: We only send allowlisted keys. Data is in-memory for reads.
3.  **DX**: The Hook API is clean, typed, and the runtime panic prevents "silent failures."
4.  **Security**: The server defines the contract (Allowlist). We don't blindly trust client storage blobs.

This replaces the "Magical" `.Persist()` with an "Explicit" architecture that scales.
