# Phase 16: Unified Context & Platform Capabilities

> **Streaming, universal adaptation, and offline-first data synchronization**

---

## Overview

Phase 16 completes the Vango V2.1 vision by expanding `vango.Ctx` into a full platform API. This phase enables intelligent streaming for slow data fetchers, capability negotiation for cross-platform delivery (Web, iOS, Android), and offline-first data synchronization for mobile deployments.

### Core Philosophy

**One Go codebase. Multiple platforms. Zero compromise on user experience.**

### Goals

1. **`ctx.Async`**: Intelligent streaming with fallbacks for slow components
2. **`ctx.Client`**: Capability negotiation for universal rendering
3. **`ctx.Sync`**: Offline-first data with conflict resolution
4. **Session Improvements**: Robust state interface on ctx.Session

### Non-Goals (Explicit Exclusions)

1. Full WASM mode (separate track)
2. Native SDKs for iOS/Android (players are separate projects)
3. Peer-to-peer sync (server is always source of truth)

### Scope Decision

This phase is marked as **"V2.1 Platform"** scope. Teams focused on "web-first, server-driven" deployments may defer parts of this phase (especially `ctx.Sync`) to a later milestone.

---

## Subsystems

| Subsystem | Purpose | Priority |
|-----------|---------|----------|
| 16.1 Unified Ctx Interface | Refactor Ctx to expose platform APIs | Critical |
| 16.2 Intelligent Streaming | `ctx.Async` for out-of-order rendering | High |
| 16.3 Universal Adaptation | `ctx.Client` for cross-platform | Medium |
| 16.4 Offline Sync | `ctx.Sync` for offline-first data | Medium |
| 16.5 Session Interface | Robust session state API | High |
| 16.6 NATS JetStream Store | Distributed session storage | Low |

---

## 16.1 Unified Ctx Interface

### Current vs New Interface

```go
// BEFORE: Limited interface
type Ctx interface {
    Session() *Session
    Route() string
    Redirect(path string)
    Emit(event string, data any)
    // ... internal methods
}

// AFTER: Platform interface
type Ctx interface {
    // Core (existing)
    Session() Session      // Note: Now returns interface, not pointer
    Route() string
    Redirect(path string)
    Emit(event string, data any)
    
    // Platform APIs (new)
    Async() AsyncCtx       // Streaming & concurrent fetching
    Client() ClientCtx     // Platform capabilities
    Sync() SyncCtx         // Offline data synchronization
    
    // Standard library integration
    StdContext() context.Context
    WithStdContext(context.Context) Ctx
    
    // Internal
    event() *protocol.Event
    patchBuffer() *patchBuffer
}
```

### Implementation Strategy

The new interfaces are implemented as lazy accessors:

```go
// pkg/vango/ctx_impl.go

type ctxImpl struct {
    session    *sessionImpl
    route      string
    stdCtx     context.Context
    
    // Lazy-initialized platform interfaces
    asyncCtx   *asyncCtxImpl
    clientCtx  *clientCtxImpl
    syncCtx    *syncCtxImpl
}

func (c *ctxImpl) Async() AsyncCtx {
    if c.asyncCtx == nil {
        c.asyncCtx = newAsyncCtx(c)
    }
    return c.asyncCtx
}

func (c *ctxImpl) Client() ClientCtx {
    if c.clientCtx == nil {
        c.clientCtx = newClientCtx(c)
    }
    return c.clientCtx
}

func (c *ctxImpl) Sync() SyncCtx {
    if c.syncCtx == nil {
        c.syncCtx = newSyncCtx(c)
    }
    return c.syncCtx
}
```

---

## 16.2 Intelligent Streaming (`ctx.Async`)

### Problem

Some components depend on slow data (external APIs, complex queries). Currently, the entire page waits for all data before any rendering begins.

### Solution

`ctx.Async` allows components to render immediately with fallbacks, then stream in real content when ready.

### Interface

```go
// pkg/vango/async.go

type AsyncCtx interface {
    // Render renders a component asynchronously with a fallback.
    // The fallback is shown immediately, real content streams when ready.
    Render(fallback *VNode, component func() *VNode) *VNode
    
    // RenderWithError handles both success and error cases.
    RenderWithError(
        fallback *VNode,
        component func() (*VNode, error),
        errorFn func(error) *VNode,
    ) *VNode
    
    // Batch groups multiple async operations for efficient streaming.
    Batch(name string, fns ...func()) Batch
}

type Batch interface {
    // OnComplete registers a callback when all batch items complete.
    OnComplete(fn func())
    
    // Wait blocks until all batch items complete (for testing).
    Wait()
}
```

### Implementation

```go
// pkg/vango/async_impl.go

type asyncCtxImpl struct {
    ctx      *ctxImpl
    pending  []*asyncJob
    mu       sync.Mutex
}

type asyncJob struct {
    id        string
    fallback  *VNode
    component func() *VNode
    result    *VNode
    done      chan struct{}
}

func (a *asyncCtxImpl) Render(fallback *VNode, component func() *VNode) *VNode {
    // Generate placeholder ID
    id := a.ctx.session.nextAsyncID()
    
    // Create job
    job := &asyncJob{
        id:        id,
        fallback:  fallback,
        component: component,
        done:      make(chan struct{}),
    }
    
    a.mu.Lock()
    a.pending = append(a.pending, job)
    a.mu.Unlock()
    
    // Start goroutine to fetch content
    go func() {
        defer close(job.done)
        defer recoverPanic(a.ctx)
        
        // Execute component function
        result := component()
        job.result = result
        
        // Send patch to replace placeholder
        a.ctx.session.sendAsyncPatch(id, result)
    }()
    
    // Return placeholder with fallback content
    return Div(
        DataAttr("async-id", id),
        Class("vango-async-placeholder"),
        fallback,
    )
}

func (a *asyncCtxImpl) RenderWithError(
    fallback *VNode,
    component func() (*VNode, error),
    errorFn func(error) *VNode,
) *VNode {
    return a.Render(fallback, func() *VNode {
        result, err := component()
        if err != nil {
            return errorFn(err)
        }
        return result
    })
}
```

### Session Patch Integration

```go
// pkg/server/session.go

func (s *Session) sendAsyncPatch(asyncID string, content *VNode) {
    // Render content to VNode tree
    tree := render.ToVTree(content)
    
    // Create REPLACE_NODE patch targeting the async placeholder
    patch := protocol.NewReplacePatch(
        fmt.Sprintf("[data-async-id=\"%s\"]", asyncID),
        tree,
    )
    
    // Queue for sending
    s.queuePatch(patch)
}
```

### Protocol Extension

```go
// New patch type for async content replacement
const (
    PatchAsyncReplace = 0x40 // Replace async placeholder with content
)

type AsyncReplacePatch struct {
    AsyncID string
    Content *VNode
}
```

### Client Handling

```javascript
// client/src/async.js

class AsyncHandler {
    handleAsyncReplace(patch) {
        const placeholder = document.querySelector(
            `[data-async-id="${patch.asyncId}"]`
        );
        
        if (!placeholder) return;
        
        // Render new content
        const newElement = this.renderer.createDOM(patch.content);
        
        // Replace placeholder with smooth transition
        placeholder.classList.add('vango-async-replacing');
        requestAnimationFrame(() => {
            placeholder.replaceWith(newElement);
        });
    }
}
```

### CSS for Async

```css
/* Async placeholder styles */
.vango-async-placeholder {
    min-height: 100px;
    position: relative;
}

.vango-async-replacing {
    animation: vango-fade-out 0.15s ease-out forwards;
}

@keyframes vango-fade-out {
    to { opacity: 0; }
}
```

### Usage Example

```go
func Dashboard(ctx vango.Ctx) *vango.VNode {
    return Div(
        Class("grid grid-cols-3 gap-4"),
        
        // Fast local data - renders immediately
        QuickStats(ctx),
        
        // Slow API call - streams when ready
        ctx.Async().Render(
            Skeleton(Class("h-48")),  // Fallback
            func() *vango.VNode {
                data, _ := slowExternalAPI.Fetch(ctx.StdContext())
                return AnalyticsChart(data)
            },
        ),
        
        // Another slow component with error handling
        ctx.Async().RenderWithError(
            Skeleton(Class("h-48")),  // Fallback
            func() (*vango.VNode, error) {
                users, err := userService.GetRecent(ctx.StdContext())
                if err != nil {
                    return nil, err
                }
                return UserList(users), nil
            },
            func(err error) *vango.VNode {
                return ErrorCard("Failed to load users", err.Error())
            },
        ),
    )
}
```

---

## 16.3 Universal Adaptation (`ctx.Client`)

### Problem

The same Vango app should render to web browsers, iOS apps, and Android apps. Each platform has different capabilities (touch, hover, native components).

### Solution

`ctx.Client` provides capability negotiation so components can adapt their output based on the client platform.

### Interface

```go
// pkg/vango/client.go

type ClientCtx interface {
    // Platform returns the client platform.
    Platform() Platform
    
    // Capabilities returns available client capabilities.
    Capabilities() Capabilities
    
    // HasCapability checks if a specific capability is available.
    HasCapability(cap string) bool
    
    // SendNative sends a native/platform-specific patch.
    // For web, this is a no-op. For native, this triggers native components.
    SendNative(component string, props any)
    
    // IsNative returns true if the client is a native app.
    IsNative() bool
    
    // ScreenSize returns the client's screen size category.
    ScreenSize() ScreenSize
}

type Platform string

const (
    PlatformWeb     Platform = "web"
    PlatformIOS     Platform = "ios"
    PlatformAndroid Platform = "android"
    PlatformDesktop Platform = "desktop"  // Electron, Tauri
)

type ScreenSize string

const (
    ScreenXS ScreenSize = "xs"  // < 640px
    ScreenSM ScreenSize = "sm"  // 640-768px
    ScreenMD ScreenSize = "md"  // 768-1024px
    ScreenLG ScreenSize = "lg"  // 1024-1280px
    ScreenXL ScreenSize = "xl"  // > 1280px
)

type Capabilities struct {
    Touch       bool
    Hover       bool
    Keyboard    bool
    Haptics     bool
    Camera      bool
    GPS         bool
    Push        bool
    Biometrics  bool
    NativeShare bool
}
```

### Capability Detection

Capabilities are negotiated during WebSocket handshake:

```go
// pkg/server/handshake.go

type HandshakeMessage struct {
    Platform     string            `json:"platform"`
    Version      string            `json:"version"`
    Capabilities map[string]bool   `json:"capabilities"`
    Screen       ScreenInfo        `json:"screen"`
}

type ScreenInfo struct {
    Width  int `json:"width"`
    Height int `json:"height"`
}

func (s *Server) handleHandshake(conn *websocket.Conn, msg []byte) (*Session, error) {
    var hs HandshakeMessage
    if err := json.Unmarshal(msg, &hs); err != nil {
        return nil, err
    }
    
    sess := s.createSession()
    sess.platform = Platform(hs.Platform)
    sess.capabilities = parseCapabilities(hs.Capabilities)
    sess.screenSize = categorizeScreen(hs.Screen.Width)
    
    return sess, nil
}
```

### Usage Example

```go
func ProductCard(ctx vango.Ctx, product Product) *vango.VNode {
    client := ctx.Client()
    
    // Platform-specific rendering
    if client.IsNative() {
        // Use native image loading for better performance
        client.SendNative("FastImage", map[string]any{
            "source": product.ImageURL,
            "cacheKey": product.ID,
        })
    }
    
    // Adaptive interactions
    var addToCartTrigger *vango.VNode
    if client.HasCapability("touch") {
        addToCartTrigger = SwipeToAdd(product)
    } else {
        addToCartTrigger = ButtonAdd(product)
    }
    
    // Screen size adaptation
    var imageClass string
    switch client.ScreenSize() {
    case ScreenXS, ScreenSM:
        imageClass = "w-full h-48"
    default:
        imageClass = "w-64 h-64"
    }
    
    return Card(
        Img(Src(product.ImageURL), Class(imageClass)),
        H3(Text(product.Name)),
        addToCartTrigger,
    )
}
```

### Native Protocol Extension

For native clients, special patch types trigger native component rendering:

```go
// Native patch types
const (
    PatchNativeView     = 0x50  // Render native view
    PatchNativeNavigate = 0x51  // Native navigation (e.g., push screen)
    PatchNativeHaptic   = 0x52  // Trigger haptic feedback
    PatchNativeShare    = 0x53  // Native share sheet
)
```

---

## 16.4 Offline Sync (`ctx.Sync`)

### Problem

Mobile apps need offline-first data access. Users should be able to view and edit data without network, with automatic sync when connectivity returns.

### Solution

`ctx.Sync` provides a `SyncResource` that abstracts local storage + background sync.

### Interface

```go
// pkg/vango/sync.go

type SyncCtx interface {
    // Resource creates or retrieves a synced resource.
    Resource(key string, opts ...SyncOption) *SyncResource
    
    // Status returns the overall sync status.
    Status() SyncStatus
    
    // ForceSync triggers an immediate sync attempt.
    ForceSync() error
}

type SyncResource struct {
    // Current returns the current local value.
    Current() any
    
    // Update updates the local value and queues for sync.
    Update(value any) error
    
    // Delete marks for deletion and queues for sync.
    Delete() error
    
    // Status returns this resource's sync status.
    Status() ResourceStatus
    
    // Conflicts returns unresolved conflicts, if any.
    Conflicts() []Conflict
    
    // ResolveConflict resolves a conflict with the given strategy.
    ResolveConflict(conflictID string, resolution Resolution) error
}

type SyncStatus struct {
    Online     bool
    Syncing    bool
    Pending    int   // Number of pending changes
    LastSync   time.Time
    LastError  error
}

type ResourceStatus int

const (
    StatusSynced ResourceStatus = iota
    StatusPending
    StatusConflict
    StatusError
)

type Conflict struct {
    ID          string
    LocalValue  any
    RemoteValue any
    LocalTime   time.Time
    RemoteTime  time.Time
}

type Resolution int

const (
    ResolveLocalWins Resolution = iota
    ResolveRemoteWins
    ResolveMerge
)
```

### Last-Write-Wins Implementation

```go
// pkg/sync/lww.go

type LWWValue struct {
    Value     any       `json:"value"`
    Timestamp time.Time `json:"timestamp"`
    ClientID  string    `json:"client_id"`
}

func (s *Syncer) resolveConflict(local, remote LWWValue) (any, bool) {
    // Conflict if timestamps are within threshold
    threshold := 1 * time.Second
    timeDiff := local.Timestamp.Sub(remote.Timestamp).Abs()
    
    if timeDiff < threshold && local.ClientID != remote.ClientID {
        // True conflict - needs user resolution
        return nil, true
    }
    
    // LWW: Later timestamp wins
    if local.Timestamp.After(remote.Timestamp) {
        return local.Value, false
    }
    return remote.Value, false
}
```

### Local Storage (SQLite for Mobile)

```go
// pkg/sync/sqlite.go

type SQLiteStore struct {
    db *sql.DB
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
    db, err := sql.Open("sqlite3", path)
    if err != nil {
        return nil, err
    }
    
    // Create schema
    _, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS sync_data (
            key TEXT PRIMARY KEY,
            value BLOB,
            timestamp INTEGER,
            status INTEGER,
            version INTEGER
        );
        
        CREATE TABLE IF NOT EXISTS sync_queue (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            key TEXT,
            operation INTEGER,
            value BLOB,
            timestamp INTEGER,
            attempts INTEGER DEFAULT 0
        );
        
        CREATE TABLE IF NOT EXISTS sync_conflicts (
            id TEXT PRIMARY KEY,
            key TEXT,
            local_value BLOB,
            remote_value BLOB,
            local_time INTEGER,
            remote_time INTEGER
        );
    `)
    
    return &SQLiteStore{db: db}, err
}

func (s *SQLiteStore) Get(key string) (any, error) {
    var value []byte
    err := s.db.QueryRow(
        "SELECT value FROM sync_data WHERE key = ?",
        key,
    ).Scan(&value)
    
    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }
    
    var result any
    json.Unmarshal(value, &result)
    return result, nil
}

func (s *SQLiteStore) Set(key string, value any) error {
    data, _ := json.Marshal(value)
    timestamp := time.Now().UnixMilli()
    
    // Update local data
    _, err := s.db.Exec(`
        INSERT OR REPLACE INTO sync_data (key, value, timestamp, status, version)
        VALUES (?, ?, ?, 1, COALESCE((SELECT version FROM sync_data WHERE key = ?), 0) + 1)
    `, key, data, timestamp, key)
    if err != nil {
        return err
    }
    
    // Queue for sync
    _, err = s.db.Exec(`
        INSERT INTO sync_queue (key, operation, value, timestamp)
        VALUES (?, 1, ?, ?)
    `, key, data, timestamp)
    
    return err
}
```

### Background Syncer

```go
// pkg/sync/syncer.go

type Syncer struct {
    store    *SQLiteStore
    api      SyncAPI
    interval time.Duration
    stop     chan struct{}
}

func (s *Syncer) Start() {
    go s.syncLoop()
}

func (s *Syncer) syncLoop() {
    ticker := time.NewTicker(s.interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            s.sync()
        case <-s.stop:
            return
        }
    }
}

func (s *Syncer) sync() error {
    // Get pending queue items
    rows, err := s.store.db.Query(`
        SELECT id, key, operation, value, timestamp
        FROM sync_queue
        WHERE attempts < 5
        ORDER BY timestamp ASC
        LIMIT 100
    `)
    if err != nil {
        return err
    }
    defer rows.Close()
    
    for rows.Next() {
        var id int64
        var key string
        var op int
        var value []byte
        var ts int64
        rows.Scan(&id, &key, &op, &value, &ts)
        
        // Try to sync
        err := s.api.Sync(key, op, value, ts)
        if err != nil {
            // Increment attempts
            s.store.db.Exec("UPDATE sync_queue SET attempts = attempts + 1 WHERE id = ?", id)
            continue
        }
        
        // Success - remove from queue
        s.store.db.Exec("DELETE FROM sync_queue WHERE id = ?", id)
        s.store.db.Exec("UPDATE sync_data SET status = 0 WHERE key = ?", key)
    }
    
    return nil
}
```

### Usage Example

```go
func TodoList(ctx vango.Ctx) *vango.VNode {
    sync := ctx.Sync()
    
    // Get synced todos
    todosResource := sync.Resource("todos", 
        vango.SyncWithDefault([]Todo{}),
        vango.SyncConflictStrategy(vango.LWW),
    )
    
    todos := todosResource.Current().([]Todo)
    
    // Show sync status
    status := sync.Status()
    var statusBadge *vango.VNode
    if !status.Online {
        statusBadge = Badge(BadgeOutline, Children(Text("Offline")))
    } else if status.Pending > 0 {
        statusBadge = Badge(BadgeSecondary, Children(Textf("Syncing %d...", status.Pending)))
    }
    
    // Check for conflicts
    if conflicts := todosResource.Conflicts(); len(conflicts) > 0 {
        return ConflictResolver(ctx, conflicts, todosResource)
    }
    
    return Div(
        Header(
            H1(Text("My Todos")),
            statusBadge,
        ),
        Range(todos, func(todo Todo, i int) *vango.VNode {
            return TodoItem(ctx, todo, func(updated Todo) {
                // Update triggers sync
                newTodos := updateTodo(todos, updated)
                todosResource.Update(newTodos)
            })
        }),
    )
}
```

---

## 16.5 Session Interface

### Enhanced Session API

```go
// pkg/vango/session.go

type Session interface {
    // Identity
    ID() string
    
    // Key-value storage
    Get(key string) any
    Set(key string, value any)
    Delete(key string)
    
    // User identity (from auth middleware)
    UserID() string
    SetUserID(id string)
    IsAuthenticated() bool
    
    // Signals
    SignalByKey(key string) Signal[any]
    
    // Lifecycle
    CreatedAt() time.Time
    LastActiveAt() time.Time
    
    // Metadata
    IP() string
    UserAgent() string
}
```

### Implementation

```go
// pkg/server/session_impl.go

type sessionImpl struct {
    id           string
    values       map[string]any
    signals      map[string]signalEntry
    userID       string
    createdAt    time.Time
    lastActiveAt time.Time
    ip           string
    userAgent    string
    
    mu sync.RWMutex
}

func (s *sessionImpl) Get(key string) any {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.values[key]
}

func (s *sessionImpl) Set(key string, value any) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.values[key] = value
}

func (s *sessionImpl) IsAuthenticated() bool {
    return s.userID != ""
}
```

---

## 16.6 NATS JetStream Store

### Problem

For horizontally scaled deployments, sessions need to be sharable across multiple server instances.

### Solution

NATS JetStream provides durable, distributed key-value storage.

### Implementation

```go
// pkg/session/nats.go

type NatsStore struct {
    kv   nats.KeyValue
    conn *nats.Conn
}

func NewNatsStore(url string, opts ...NatsOption) (*NatsStore, error) {
    nc, err := nats.Connect(url)
    if err != nil {
        return nil, err
    }
    
    js, err := nc.JetStream()
    if err != nil {
        return nil, err
    }
    
    kv, err := js.CreateKeyValue(&nats.KeyValueConfig{
        Bucket:  "vango_sessions",
        TTL:     time.Hour,
        Storage: nats.FileStorage,
    })
    if err != nil {
        // Bucket might already exist
        kv, err = js.KeyValue("vango_sessions")
        if err != nil {
            return nil, err
        }
    }
    
    return &NatsStore{kv: kv, conn: nc}, nil
}

func (n *NatsStore) Save(ctx context.Context, sessionID string, data []byte, expiresAt time.Time) error {
    _, err := n.kv.Put(sessionID, data)
    return err
}

func (n *NatsStore) Load(ctx context.Context, sessionID string) ([]byte, error) {
    entry, err := n.kv.Get(sessionID)
    if err == nats.ErrKeyNotFound {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }
    return entry.Value(), nil
}

func (n *NatsStore) Delete(ctx context.Context, sessionID string) error {
    return n.kv.Delete(sessionID)
}

func (n *NatsStore) Touch(ctx context.Context, sessionID string, expiresAt time.Time) error {
    // NATS JetStream doesn't support touch, so we re-save
    entry, err := n.kv.Get(sessionID)
    if err != nil {
        return err
    }
    _, err = n.kv.Put(sessionID, entry.Value())
    return err
}

func (n *NatsStore) SaveAll(ctx context.Context, sessions map[string][]byte, expiresAt time.Time) error {
    for id, data := range sessions {
        if _, err := n.kv.Put(id, data); err != nil {
            return err
        }
    }
    return nil
}
```

---

## Exit Criteria

### 16.1 Unified Ctx Interface
- [ ] `Ctx` interface includes `Async()`, `Client()`, `Sync()`
- [ ] Lazy initialization of platform interfaces
- [ ] Backward compatible with existing code

### 16.2 Intelligent Streaming
- [ ] `ctx.Async().Render()` returns placeholder immediately
- [ ] Goroutine fetches content in background
- [ ] `PatchAsyncReplace` sent when content ready
- [ ] Client replaces placeholder with smooth transition
- [ ] Panic recovery in async goroutines
- [ ] Batch API for grouping operations

### 16.3 Universal Adaptation
- [ ] `ctx.Client().Platform()` returns correct platform
- [ ] `ctx.Client().Capabilities()` reflects client features
- [ ] Capability negotiation in WebSocket handshake
- [ ] `SendNative()` triggers native component on native clients

### 16.4 Offline Sync
- [ ] `ctx.Sync().Resource()` creates synced resource
- [ ] Local SQLite storage works on mobile
- [ ] Background syncer runs on interval
- [ ] LWW conflict resolution implemented
- [ ] Conflict UI can be rendered
- [ ] Sync status observable

### 16.5 Session Interface
- [ ] `Session` is interface, not pointer
- [ ] `Get`/`Set`/`Delete` work correctly
- [ ] `UserID` and `IsAuthenticated` work
- [ ] Metadata (IP, UserAgent) accessible

### 16.6 NATS JetStream Store
- [ ] `NatsStore` implements `SessionStore`
- [ ] Save/Load/Delete work correctly
- [ ] SaveAll works for graceful shutdown
- [ ] Automatic bucket creation

---

## Files Changed

### New Files
- `pkg/vango/async.go` - AsyncCtx interface
- `pkg/vango/async_impl.go` - AsyncCtx implementation
- `pkg/vango/client.go` - ClientCtx interface
- `pkg/vango/client_impl.go` - ClientCtx implementation
- `pkg/vango/sync.go` - SyncCtx interface
- `pkg/vango/sync_impl.go` - SyncCtx implementation
- `pkg/sync/lww.go` - Last-Write-Wins logic
- `pkg/sync/sqlite.go` - SQLite local storage
- `pkg/sync/syncer.go` - Background sync
- `pkg/session/nats.go` - NATS JetStream store
- `client/src/async.js` - Async patch handling

### Modified Files
- `pkg/vango/ctx.go` - Interface updates
- `pkg/vango/ctx_impl.go` - Implementation
- `pkg/server/session.go` - Session interface
- `pkg/server/handshake.go` - Capability negotiation
- `pkg/protocol/patch.go` - New patch types
- `client/src/patches.js` - New patch handlers

---

## Dependencies

- `github.com/nats-io/nats.go` - NATS client
- `github.com/mattn/go-sqlite3` - SQLite driver (for mobile)

---

## Migration Guide

### Accessing New APIs

```go
// All new APIs are backward compatible additions

// Streaming
ctx.Async().Render(fallback, component)

// Platform detection
if ctx.Client().IsNative() {
    // Native-specific code
}

// Offline data (when enabled)
resource := ctx.Sync().Resource("key")
```

### Session Interface Change

```go
// BEFORE: Session was a pointer
func handler(ctx vango.Ctx) {
    sess := ctx.Session()  // *Session
    user := sess.Get("user")
}

// AFTER: Session is an interface (same usage)
func handler(ctx vango.Ctx) {
    sess := ctx.Session()  // Session (interface)
    user := sess.Get("user")
}
```

The change from pointer to interface is backward compatible for all typical usage patterns.
