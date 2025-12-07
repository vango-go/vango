# Phase 4: Server Runtime ✅ COMPLETE

> **The server-side component execution environment**

**Status**: Complete (2024-12-07)

---

## Overview

The server runtime manages WebSocket connections, component state, event handling, and patch generation. It is the core of Vango's server-driven architecture.

### Goals

1. **Efficient session management**: Minimal memory per session
2. **Fast event processing**: < 10ms handler execution
3. **Reliable delivery**: Patches reach client in order
4. **Graceful degradation**: Handle disconnects, reconnects
5. **Scalability**: 10,000+ concurrent sessions per server

### Non-Goals

1. Horizontal scaling (future - requires Redis)
2. Session persistence across restarts (future)
3. Multi-server synchronization (future)

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        HTTP Server                              │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                     Router                                  ││
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          ││
│  │  │ GET /       │  │ GET /about  │  │ WS /_vango  │          ││
│  │  │ SSR Handler │  │ SSR Handler │  │ Live Handler│          ││
│  │  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘          ││
│  └─────────┼────────────────┼────────────────┼─────────────────┘│
│            │                │                │                  │
│            ▼                ▼                ▼                  │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                  Session Manager                            ││
│  │  ┌─────────────────────────────────────────────────────────┐││
│  │  │  Sessions Map: map[string]*Session                      │││
│  │  │  ┌──────────┐ ┌──────────┐ ┌──────────┐                 │││
│  │  │  │Session A │ │Session B │ │Session C │  ...            │││
│  │  │  └──────────┘ └──────────┘ └──────────┘                 │││
│  │  └─────────────────────────────────────────────────────────┘││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

---

## Core Types

### Session

Represents a single WebSocket connection and its state.

```go
type Session struct {
    // Identity
    ID        string
    UserID    string              // From auth (optional)
    CreatedAt time.Time
    LastActive time.Time

    // Connection
    conn      *websocket.Conn
    mu        sync.Mutex          // Protects conn writes
    closed    bool

    // Sequence numbers
    sendSeq   uint64              // Next patch sequence
    recvSeq   uint64              // Last received event sequence
    ackSeq    uint64              // Last acknowledged by client

    // Component state
    root      *ComponentInstance  // Root component
    components map[string]*ComponentInstance  // HID → component
    handlers  map[string]Handler  // HID → event handler

    // Reactive ownership
    owner     *Owner              // Root signal owner

    // Rendering
    currentTree *VNode            // Last rendered tree
    hidGen    *HIDGenerator       // Hydration ID generator

    // Channels
    events    chan *Event         // Incoming events
    done      chan struct{}       // Shutdown signal

    // Configuration
    config    *SessionConfig

    // Metrics
    eventCount   uint64
    patchCount   uint64
    bytesSent    uint64
    bytesRecv    uint64
}

type SessionConfig struct {
    // Timeouts
    ReadTimeout     time.Duration  // Max time waiting for message
    WriteTimeout    time.Duration  // Max time writing message
    IdleTimeout     time.Duration  // Close after inactivity
    HandshakeTimeout time.Duration // Max handshake duration

    // Limits
    MaxMessageSize  int64          // Max incoming message size
    MaxPatchHistory int            // Patches to keep for resync
    MaxEventQueue   int            // Event channel buffer size

    // Features
    EnableCompression bool         // gzip compression
    EnableOptimistic  bool         // Optimistic updates
}

func DefaultSessionConfig() *SessionConfig {
    return &SessionConfig{
        ReadTimeout:      60 * time.Second,
        WriteTimeout:     10 * time.Second,
        IdleTimeout:      5 * time.Minute,
        HandshakeTimeout: 10 * time.Second,
        MaxMessageSize:   64 * 1024,  // 64KB
        MaxPatchHistory:  100,
        MaxEventQueue:    256,
        EnableCompression: true,
        EnableOptimistic:  true,
    }
}
```

### SessionManager

Manages all active sessions.

```go
type SessionManager struct {
    sessions  map[string]*Session
    mu        sync.RWMutex
    config    *SessionConfig

    // Cleanup
    cleanupInterval time.Duration
    cleanupTicker   *time.Ticker
    done            chan struct{}

    // Metrics
    totalCreated  uint64
    totalClosed   uint64
    peakSessions  int

    // Callbacks
    onSessionCreate func(*Session)
    onSessionClose  func(*Session)
}

func NewSessionManager(config *SessionConfig) *SessionManager {
    sm := &SessionManager{
        sessions:        make(map[string]*Session),
        config:          config,
        cleanupInterval: 30 * time.Second,
        done:            make(chan struct{}),
    }
    go sm.cleanupLoop()
    return sm
}

func (sm *SessionManager) Create(conn *websocket.Conn, userID string) *Session {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    id := generateSessionID()
    session := &Session{
        ID:         id,
        UserID:     userID,
        CreatedAt:  time.Now(),
        LastActive: time.Now(),
        conn:       conn,
        handlers:   make(map[string]Handler),
        components: make(map[string]*ComponentInstance),
        events:     make(chan *Event, sm.config.MaxEventQueue),
        done:       make(chan struct{}),
        config:     sm.config,
        hidGen:     NewHIDGenerator(),
        owner:      NewOwner(nil),
    }

    sm.sessions[id] = session
    atomic.AddUint64(&sm.totalCreated, 1)

    if len(sm.sessions) > sm.peakSessions {
        sm.peakSessions = len(sm.sessions)
    }

    if sm.onSessionCreate != nil {
        sm.onSessionCreate(session)
    }

    return session
}

func (sm *SessionManager) Get(id string) *Session {
    sm.mu.RLock()
    defer sm.mu.RUnlock()
    return sm.sessions[id]
}

func (sm *SessionManager) Close(id string) {
    sm.mu.Lock()
    session, exists := sm.sessions[id]
    if exists {
        delete(sm.sessions, id)
    }
    sm.mu.Unlock()

    if exists {
        session.Close()
        atomic.AddUint64(&sm.totalClosed, 1)
        if sm.onSessionClose != nil {
            sm.onSessionClose(session)
        }
    }
}

func (sm *SessionManager) cleanupLoop() {
    sm.cleanupTicker = time.NewTicker(sm.cleanupInterval)
    defer sm.cleanupTicker.Stop()

    for {
        select {
        case <-sm.cleanupTicker.C:
            sm.cleanupExpired()
        case <-sm.done:
            return
        }
    }
}

func (sm *SessionManager) cleanupExpired() {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    now := time.Now()
    for id, session := range sm.sessions {
        if now.Sub(session.LastActive) > sm.config.IdleTimeout {
            delete(sm.sessions, id)
            go session.Close()
            atomic.AddUint64(&sm.totalClosed, 1)
        }
    }
}

func (sm *SessionManager) Shutdown() {
    close(sm.done)

    sm.mu.Lock()
    sessions := make([]*Session, 0, len(sm.sessions))
    for _, s := range sm.sessions {
        sessions = append(sessions, s)
    }
    sm.sessions = make(map[string]*Session)
    sm.mu.Unlock()

    // Close all sessions
    var wg sync.WaitGroup
    for _, s := range sessions {
        wg.Add(1)
        go func(session *Session) {
            defer wg.Done()
            session.Close()
        }(s)
    }
    wg.Wait()
}

func (sm *SessionManager) Stats() SessionStats {
    sm.mu.RLock()
    defer sm.mu.RUnlock()

    return SessionStats{
        Active:       len(sm.sessions),
        TotalCreated: atomic.LoadUint64(&sm.totalCreated),
        TotalClosed:  atomic.LoadUint64(&sm.totalClosed),
        Peak:         sm.peakSessions,
    }
}
```

### ComponentInstance

A mounted component with its state.

```go
type ComponentInstance struct {
    ID        string              // Unique instance ID
    Component Component           // The component
    HID       string              // Root element's hydration ID
    Owner     *Owner              // Signal ownership
    Parent    *ComponentInstance  // Parent instance
    Children  []*ComponentInstance
    Props     map[string]any      // Current props
    dirty     bool                // Needs re-render
    session   *Session            // Owning session
}

func (c *ComponentInstance) Render() *VNode {
    // Set up tracking context
    ctx := &TrackingContext{
        currentOwner: c.Owner,
    }
    setTrackingContext(ctx)
    defer clearTrackingContext()

    // Render component
    tree := c.Component.Render()

    return tree
}

func (c *ComponentInstance) MarkDirty() {
    if !c.dirty {
        c.dirty = true
        c.session.scheduleRender(c)
    }
}
```

### Handler

Event handler function.

```go
// Handler is called when an event targets an HID
type Handler func(event *Event)

// Event contains parsed event data
type Event struct {
    Sequence uint64
    Type     EventType
    HID      string
    Data     EventData
}

// EventData is a union of possible event payloads
type EventData interface {
    eventData()
}

type ClickEventData struct{}
func (ClickEventData) eventData() {}

type InputEventData struct {
    Value string
}
func (InputEventData) eventData() {}

type KeyboardEventData struct {
    Key       string
    Modifiers byte
}
func (KeyboardEventData) eventData() {}

type MouseEventData struct {
    ClientX   int
    ClientY   int
    Button    byte
    Modifiers byte
}
func (MouseEventData) eventData() {}

type SubmitEventData struct {
    Fields map[string]string
}
func (SubmitEventData) eventData() {}

type HookEventData struct {
    Name string
    Data map[string]any
}
func (HookEventData) eventData() {}

type NavigateEventData struct {
    Path    string
    Replace bool
}
func (NavigateEventData) eventData() {}
```

---

## Session Lifecycle

### Connection Establishment

```go
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
    // Upgrade to WebSocket
    conn, err := s.upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Printf("WebSocket upgrade failed: %v", err)
        return
    }

    // Set connection options
    conn.SetReadLimit(s.config.MaxMessageSize)
    conn.SetReadDeadline(time.Now().Add(s.config.HandshakeTimeout))

    // Wait for handshake
    _, msg, err := conn.ReadMessage()
    if err != nil {
        conn.Close()
        return
    }

    // Parse client hello
    hello, err := protocol.DecodeClientHello(msg)
    if err != nil {
        s.sendError(conn, protocol.ErrInvalidFrame, "Invalid handshake")
        conn.Close()
        return
    }

    // Validate CSRF
    if !s.validateCSRF(r, hello.CSRFToken) {
        s.sendError(conn, protocol.ErrInvalidCSRF, "Invalid CSRF token")
        conn.Close()
        return
    }

    // Check for session resume
    var session *Session
    if hello.SessionID != "" {
        session = s.sessions.Get(hello.SessionID)
        if session != nil {
            // Resume existing session
            session.Resume(conn, hello.LastSeq)
        }
    }

    // Create new session if needed
    if session == nil {
        userID := s.getUserID(r)
        session = s.sessions.Create(conn, userID)
    }

    // Send server hello
    serverHello := protocol.ServerHello{
        Status:     protocol.HandshakeOK,
        SessionID:  session.ID,
        NextSeq:    session.sendSeq,
        ServerTime: time.Now().UnixMilli(),
    }
    conn.SetWriteDeadline(time.Now().Add(s.config.WriteTimeout))
    conn.WriteMessage(websocket.BinaryMessage, protocol.EncodeServerHello(serverHello))

    // Mount root component
    session.MountRoot(s.rootComponent)

    // Start session loops
    go session.ReadLoop()
    go session.WriteLoop()
    go session.EventLoop()
}
```

### Read Loop

```go
func (s *Session) ReadLoop() {
    defer s.Close()

    for {
        s.conn.SetReadDeadline(time.Now().Add(s.config.ReadTimeout))
        _, msg, err := s.conn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                log.Printf("Session %s read error: %v", s.ID, err)
            }
            return
        }

        s.LastActive = time.Now()
        atomic.AddUint64(&s.bytesRecv, uint64(len(msg)))

        // Parse frame header
        frameType, _, payload, err := protocol.DecodeFrame(msg)
        if err != nil {
            log.Printf("Session %s frame decode error: %v", s.ID, err)
            continue
        }

        switch frameType {
        case protocol.FrameEvent:
            event, err := protocol.DecodeEvent(payload)
            if err != nil {
                log.Printf("Session %s event decode error: %v", s.ID, err)
                continue
            }

            // Queue event for processing
            select {
            case s.events <- event:
                // Queued successfully
            default:
                // Queue full, drop event
                log.Printf("Session %s event queue full, dropping event", s.ID)
            }

        case protocol.FrameControl:
            s.handleControl(payload)

        case protocol.FrameAck:
            s.handleAck(payload)
        }
    }
}
```

### Event Loop

```go
func (s *Session) EventLoop() {
    for {
        select {
        case event := <-s.events:
            s.handleEvent(event)

        case <-s.done:
            return
        }
    }
}

func (s *Session) handleEvent(event *Event) {
    // Update sequence
    s.recvSeq = event.Sequence
    atomic.AddUint64(&s.eventCount, 1)

    // Find handler
    handler, exists := s.handlers[event.HID]
    if !exists {
        log.Printf("Session %s: no handler for HID %s", s.ID, event.HID)
        return
    }

    // Execute handler with panic recovery
    func() {
        defer func() {
            if r := recover(); r != nil {
                log.Printf("Session %s: handler panic: %v\n%s", s.ID, r, debug.Stack())
                s.sendError(protocol.ErrHandlerPanic, fmt.Sprintf("Handler panic: %v", r))
            }
        }()

        handler(event)
    }()

    // Run pending effects
    s.owner.runPendingEffects()

    // Re-render dirty components
    s.renderDirty()
}

func (s *Session) renderDirty() {
    // Collect dirty components
    var dirty []*ComponentInstance
    for _, comp := range s.components {
        if comp.dirty {
            dirty = append(dirty, comp)
            comp.dirty = false
        }
    }

    if len(dirty) == 0 {
        return
    }

    // Re-render each dirty component
    for _, comp := range dirty {
        newTree := comp.Render()

        // Find component's subtree in current tree
        oldTree := s.findSubtree(comp.HID)

        // Diff and collect patches
        patches := Diff(oldTree, newTree)

        // Update stored tree
        s.updateSubtree(comp.HID, newTree)

        // Send patches
        if len(patches) > 0 {
            s.sendPatches(patches)
        }
    }
}
```

### Write Loop

```go
func (s *Session) WriteLoop() {
    // Heartbeat ticker
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            s.sendPing()

        case <-s.done:
            return
        }
    }
}

func (s *Session) sendPatches(patches []Patch) {
    s.mu.Lock()
    defer s.mu.Unlock()

    if s.closed {
        return
    }

    s.sendSeq++
    encoded := protocol.EncodePatches(s.sendSeq, patches)

    frame := protocol.EncodeFrame(protocol.FramePatches, 0, encoded)

    s.conn.SetWriteDeadline(time.Now().Add(s.config.WriteTimeout))
    err := s.conn.WriteMessage(websocket.BinaryMessage, frame)
    if err != nil {
        log.Printf("Session %s write error: %v", s.ID, err)
        s.closed = true
        close(s.done)
        return
    }

    atomic.AddUint64(&s.bytesSent, uint64(len(frame)))
    atomic.AddUint64(&s.patchCount, uint64(len(patches)))
}

func (s *Session) sendPing() {
    s.mu.Lock()
    defer s.mu.Unlock()

    if s.closed {
        return
    }

    ping := protocol.EncodePing(time.Now().UnixMilli())
    frame := protocol.EncodeFrame(protocol.FrameControl, 0, ping)

    s.conn.SetWriteDeadline(time.Now().Add(s.config.WriteTimeout))
    err := s.conn.WriteMessage(websocket.BinaryMessage, frame)
    if err != nil {
        log.Printf("Session %s ping error: %v", s.ID, err)
    }
}

func (s *Session) sendError(code uint16, message string) {
    s.mu.Lock()
    defer s.mu.Unlock()

    if s.closed {
        return
    }

    errMsg := protocol.EncodeError(code, message, false)
    frame := protocol.EncodeFrame(protocol.FrameError, 0, errMsg)

    s.conn.SetWriteDeadline(time.Now().Add(s.config.WriteTimeout))
    s.conn.WriteMessage(websocket.BinaryMessage, frame)
}
```

### Session Close

```go
func (s *Session) Close() {
    s.mu.Lock()
    if s.closed {
        s.mu.Unlock()
        return
    }
    s.closed = true
    s.mu.Unlock()

    // Signal shutdown
    close(s.done)

    // Dispose reactive owner (cleanup effects)
    s.owner.dispose()

    // Clear handlers
    s.handlers = nil

    // Close WebSocket
    s.conn.WriteControl(
        websocket.CloseMessage,
        websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
        time.Now().Add(time.Second),
    )
    s.conn.Close()
}
```

---

## Component Mounting

### Mount Root

```go
func (s *Session) MountRoot(component Component) {
    // Create component instance
    instance := &ComponentInstance{
        ID:        "root",
        Component: component,
        Owner:     s.owner,
        session:   s,
    }

    s.root = instance

    // Render component
    tree := instance.Render()

    // Assign hydration IDs
    AssignHIDs(tree, s.hidGen)

    // Collect handlers
    s.collectHandlers(tree, instance)

    // Store tree
    s.currentTree = tree
    instance.HID = tree.HID
}

func (s *Session) collectHandlers(node *VNode, instance *ComponentInstance) {
    if node == nil {
        return
    }

    // Collect event handlers from props
    if node.HID != "" {
        for key, value := range node.Props {
            if strings.HasPrefix(key, "on") && value != nil {
                if handler, ok := value.(func()); ok {
                    s.handlers[node.HID] = func(event *Event) {
                        handler()
                    }
                } else if handlerWithEvent, ok := value.(func(*Event)); ok {
                    s.handlers[node.HID] = handlerWithEvent
                }
                // Store component reference for the handler
                s.components[node.HID] = instance
            }
        }
    }

    // Recurse to children
    for _, child := range node.Children {
        if child.Kind == KindComponent {
            // Mount child component
            childInstance := s.mountComponent(child.Comp, instance)
            s.collectHandlers(child, childInstance)
        } else {
            s.collectHandlers(child, instance)
        }
    }
}

func (s *Session) mountComponent(comp Component, parent *ComponentInstance) *ComponentInstance {
    instance := &ComponentInstance{
        ID:        generateComponentID(),
        Component: comp,
        Owner:     NewOwner(parent.Owner),
        Parent:    parent,
        session:   s,
    }

    parent.Children = append(parent.Children, instance)

    return instance
}
```

---

## Handler Registration

Handlers are registered during render by wrapping event functions:

```go
// During element creation, wrap handlers
func wrapHandler(s *Session, hid string, handler any) {
    switch h := handler.(type) {
    case func():
        s.handlers[hid] = func(e *Event) { h() }

    case func(string):
        s.handlers[hid] = func(e *Event) {
            if data, ok := e.Data.(InputEventData); ok {
                h(data.Value)
            }
        }

    case func(MouseEvent):
        s.handlers[hid] = func(e *Event) {
            if data, ok := e.Data.(MouseEventData); ok {
                h(MouseEvent{
                    ClientX:  data.ClientX,
                    ClientY:  data.ClientY,
                    Button:   int(data.Button),
                    CtrlKey:  data.Modifiers&0x01 != 0,
                    ShiftKey: data.Modifiers&0x02 != 0,
                    AltKey:   data.Modifiers&0x04 != 0,
                    MetaKey:  data.Modifiers&0x08 != 0,
                })
            }
        }

    case func(KeyboardEvent):
        s.handlers[hid] = func(e *Event) {
            if data, ok := e.Data.(KeyboardEventData); ok {
                h(KeyboardEvent{
                    Key:      data.Key,
                    CtrlKey:  data.Modifiers&0x01 != 0,
                    ShiftKey: data.Modifiers&0x02 != 0,
                    AltKey:   data.Modifiers&0x04 != 0,
                    MetaKey:  data.Modifiers&0x08 != 0,
                })
            }
        }

    case func(FormData):
        s.handlers[hid] = func(e *Event) {
            if data, ok := e.Data.(SubmitEventData); ok {
                h(FormData{values: data.Fields})
            }
        }

    case func(HookEvent):
        s.handlers[hid] = func(e *Event) {
            if data, ok := e.Data.(HookEventData); ok {
                h(HookEvent{Name: data.Name, Data: data.Data})
            }
        }
    }
}
```

---

## Request Context

For accessing request data within components.

```go
type Ctx interface {
    // Request info
    Request() *http.Request
    Path() string
    Method() string
    Query() url.Values
    Param(key string) string

    // Response control
    Status(code int)
    Redirect(url string, code int)

    // Session
    Session() *Session
    User() any  // User from auth

    // Logging
    Logger() *slog.Logger

    // Lifecycle
    Done() <-chan struct{}
}

type ctx struct {
    request  *http.Request
    writer   http.ResponseWriter
    session  *Session
    params   map[string]string
    user     any
    logger   *slog.Logger
    status   int
}

func (c *ctx) Request() *http.Request { return c.request }
func (c *ctx) Path() string { return c.request.URL.Path }
func (c *ctx) Method() string { return c.request.Method }
func (c *ctx) Query() url.Values { return c.request.URL.Query() }

func (c *ctx) Param(key string) string {
    return c.params[key]
}

func (c *ctx) Status(code int) {
    c.status = code
}

func (c *ctx) Redirect(url string, code int) {
    http.Redirect(c.writer, c.request, url, code)
}

func (c *ctx) Session() *Session {
    return c.session
}

func (c *ctx) User() any {
    return c.user
}

func (c *ctx) Logger() *slog.Logger {
    return c.logger
}

func (c *ctx) Done() <-chan struct{} {
    return c.request.Context().Done()
}
```

---

## WebSocket Server

```go
type Server struct {
    sessions *SessionManager
    router   *Router
    config   *ServerConfig
    upgrader websocket.Upgrader

    // Middleware
    middleware []Middleware

    // Auth
    authFunc func(*http.Request) (any, error)

    // CSRF
    csrfSecret []byte
}

type ServerConfig struct {
    // WebSocket
    ReadBufferSize  int
    WriteBufferSize int
    CheckOrigin     func(r *http.Request) bool

    // Session
    SessionConfig *SessionConfig

    // Server
    Address         string
    ShutdownTimeout time.Duration
}

func NewServer(config *ServerConfig) *Server {
    if config == nil {
        config = DefaultServerConfig()
    }

    return &Server{
        sessions: NewSessionManager(config.SessionConfig),
        router:   NewRouter(),
        config:   config,
        upgrader: websocket.Upgrader{
            ReadBufferSize:  config.ReadBufferSize,
            WriteBufferSize: config.WriteBufferSize,
            CheckOrigin:     config.CheckOrigin,
        },
    }
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Check for WebSocket upgrade
    if r.URL.Path == "/_vango/ws" {
        s.HandleWebSocket(w, r)
        return
    }

    // Regular HTTP request
    s.router.ServeHTTP(w, r)
}

func (s *Server) Run() error {
    httpServer := &http.Server{
        Addr:    s.config.Address,
        Handler: s,
    }

    // Graceful shutdown
    shutdown := make(chan os.Signal, 1)
    signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

    go func() {
        <-shutdown
        log.Println("Shutting down...")

        ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
        defer cancel()

        // Close all sessions
        s.sessions.Shutdown()

        // Shutdown HTTP server
        httpServer.Shutdown(ctx)
    }()

    log.Printf("Server starting on %s", s.config.Address)
    return httpServer.ListenAndServe()
}
```

---

## Memory Management

### Session Limits

```go
type SessionLimits struct {
    MaxSessions     int   // Maximum concurrent sessions
    MaxMemoryPerSession int64 // Approximate memory limit per session
}

func (sm *SessionManager) canCreate() bool {
    sm.mu.RLock()
    defer sm.mu.RUnlock()
    return len(sm.sessions) < sm.limits.MaxSessions
}

// Estimate session memory usage
func (s *Session) MemoryUsage() int64 {
    var size int64

    // Handlers map
    size += int64(len(s.handlers)) * 64

    // Components
    size += int64(len(s.components)) * 128

    // Tree size (estimate)
    size += estimateTreeSize(s.currentTree)

    // Owner/signals
    size += s.owner.MemoryUsage()

    return size
}

func estimateTreeSize(node *VNode) int64 {
    if node == nil {
        return 0
    }

    size := int64(64) // Base VNode size
    size += int64(len(node.Tag))
    size += int64(len(node.Text))
    size += int64(len(node.HID))

    for k, v := range node.Props {
        size += int64(len(k))
        if s, ok := v.(string); ok {
            size += int64(len(s))
        } else {
            size += 16 // Estimate for other types
        }
    }

    for _, child := range node.Children {
        size += estimateTreeSize(child)
    }

    return size
}
```

### LRU Eviction

```go
// When memory pressure is high, evict least recently used sessions
func (sm *SessionManager) evictLRU(count int) {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    // Build list sorted by LastActive
    sessions := make([]*Session, 0, len(sm.sessions))
    for _, s := range sm.sessions {
        sessions = append(sessions, s)
    }

    sort.Slice(sessions, func(i, j int) bool {
        return sessions[i].LastActive.Before(sessions[j].LastActive)
    })

    // Evict oldest
    for i := 0; i < count && i < len(sessions); i++ {
        id := sessions[i].ID
        delete(sm.sessions, id)
        go sessions[i].Close()
    }
}
```

---

## Error Handling

```go
// Recover from panics in handlers
func (s *Session) safeHandle(event *Event) {
    defer func() {
        if r := recover(); r != nil {
            stack := debug.Stack()
            log.Printf("Session %s handler panic: %v\n%s", s.ID, r, stack)

            // Send error to client
            s.sendError(protocol.ErrHandlerPanic, fmt.Sprintf("Internal error"))

            // Optionally close session on severe errors
            if isSevere(r) {
                s.Close()
            }
        }
    }()

    s.handleEvent(event)
}

func isSevere(r any) bool {
    // Check if error is severe enough to close session
    if err, ok := r.(error); ok {
        return errors.Is(err, context.Canceled) ||
               errors.Is(err, context.DeadlineExceeded)
    }
    return false
}
```

---

## Metrics

```go
type ServerMetrics struct {
    // Sessions
    ActiveSessions   int64
    TotalSessions    int64
    SessionCreates   int64
    SessionCloses    int64

    // Events
    EventsReceived   int64
    EventsProcessed  int64
    EventsDropped    int64

    // Patches
    PatchesSent      int64
    PatchBytes       int64

    // Errors
    HandlerPanics    int64
    WriteErrors      int64
    ReadErrors       int64

    // Latency
    EventLatencyP50  time.Duration
    EventLatencyP99  time.Duration
}

func (s *Server) Metrics() *ServerMetrics {
    stats := s.sessions.Stats()

    return &ServerMetrics{
        ActiveSessions: int64(stats.Active),
        TotalSessions:  int64(stats.TotalCreated),
        SessionCreates: int64(stats.TotalCreated),
        SessionCloses:  int64(stats.TotalClosed),
        // ... other metrics
    }
}

// Prometheus integration
func (s *Server) PrometheusHandler() http.Handler {
    return promhttp.Handler()
}
```

---

## Testing Strategy

### Unit Tests

```go
func TestSessionCreation(t *testing.T) {
    sm := NewSessionManager(DefaultSessionConfig())
    defer sm.Shutdown()

    conn := &mockWebSocketConn{}
    session := sm.Create(conn, "user123")

    assert.NotEmpty(t, session.ID)
    assert.Equal(t, "user123", session.UserID)
    assert.Equal(t, 1, len(sm.sessions))
}

func TestSessionClose(t *testing.T) {
    sm := NewSessionManager(DefaultSessionConfig())
    defer sm.Shutdown()

    conn := &mockWebSocketConn{}
    session := sm.Create(conn, "")

    sm.Close(session.ID)

    assert.Equal(t, 0, len(sm.sessions))
    assert.True(t, session.closed)
}

func TestEventHandling(t *testing.T) {
    session := createTestSession()

    called := false
    session.handlers["h1"] = func(e *Event) {
        called = true
    }

    event := &Event{
        Sequence: 1,
        Type:     protocol.EventClick,
        HID:      "h1",
    }

    session.handleEvent(event)

    assert.True(t, called)
}

func TestHandlerPanicRecovery(t *testing.T) {
    session := createTestSession()

    session.handlers["h1"] = func(e *Event) {
        panic("test panic")
    }

    event := &Event{
        Sequence: 1,
        Type:     protocol.EventClick,
        HID:      "h1",
    }

    // Should not panic
    assert.NotPanics(t, func() {
        session.safeHandle(event)
    })
}

func TestIdleTimeout(t *testing.T) {
    config := DefaultSessionConfig()
    config.IdleTimeout = 100 * time.Millisecond

    sm := NewSessionManager(config)
    sm.cleanupInterval = 50 * time.Millisecond
    defer sm.Shutdown()

    conn := &mockWebSocketConn{}
    session := sm.Create(conn, "")

    // Wait for cleanup
    time.Sleep(200 * time.Millisecond)

    assert.Nil(t, sm.Get(session.ID))
}
```

### Integration Tests

```go
func TestWebSocketConnection(t *testing.T) {
    server := NewServer(nil)
    server.router.Handle("/", func(ctx Ctx) (*VNode, error) {
        return Div(Text("Hello")), nil
    })

    ts := httptest.NewServer(server)
    defer ts.Close()

    // Connect WebSocket
    wsURL := "ws" + ts.URL[4:] + "/_vango/ws"
    conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
    require.NoError(t, err)
    defer conn.Close()

    // Send handshake
    hello := protocol.EncodeClientHello(protocol.ClientHello{
        Version: 1,
    })
    err = conn.WriteMessage(websocket.BinaryMessage, hello)
    require.NoError(t, err)

    // Receive server hello
    _, msg, err := conn.ReadMessage()
    require.NoError(t, err)

    serverHello, err := protocol.DecodeServerHello(msg)
    require.NoError(t, err)
    assert.Equal(t, protocol.HandshakeOK, serverHello.Status)
}

func TestEventRoundTrip(t *testing.T) {
    clicked := make(chan bool, 1)

    server := NewServer(nil)
    server.router.Handle("/", func(ctx Ctx) (*VNode, error) {
        return Button(OnClick(func() {
            clicked <- true
        }), Text("Click")), nil
    })

    // ... set up WebSocket connection ...

    // Send click event
    event := protocol.EncodeEvent(1, protocol.EventClick, "h1", nil)
    conn.WriteMessage(websocket.BinaryMessage, event)

    // Verify handler was called
    select {
    case <-clicked:
        // Success
    case <-time.After(time.Second):
        t.Fatal("Handler not called")
    }
}
```

---

## File Structure

```
pkg/server/
├── doc.go               # Package documentation
├── errors.go            # Error types and sentinels
├── errors_test.go
├── config.go            # SessionConfig, ServerConfig, SessionLimits
├── config_test.go
├── context.go           # Request context (Ctx interface)
├── handler.go           # Handler types, event types, wrappers
├── handler_test.go
├── component.go         # ComponentInstance, Component interface
├── component_test.go
├── session.go           # Session type and lifecycle
├── manager.go           # SessionManager
├── manager_test.go
├── websocket.go         # ReadLoop, WriteLoop, EventLoop
├── server.go            # HTTP/WebSocket server
├── metrics.go           # MetricsCollector
├── metrics_test.go
├── memory.go            # MemoryMonitor, byte size utilities
└── memory_test.go
```

---

## Exit Criteria

Phase 4 is complete when:

1. [x] Session creation and cleanup working
2. [x] WebSocket connection established
3. [x] Handshake protocol implemented
4. [x] Event decoding and routing working
5. [x] Handler execution with panic recovery
6. [x] Patch generation and sending working
7. [x] Heartbeat/ping implemented
8. [x] Idle timeout and eviction working
9. [x] Graceful shutdown working
10. [x] Unit tests for all components
11. [x] Integration with Phases 1-3
12. [x] Memory monitoring and estimation

---

## Dependencies

- **Requires**: Phase 1 (signals), Phase 2 (VNode), Phase 3 (protocol)
- **Required by**: Phase 5 (client connects to server), Phase 6 (SSR uses session)

---

*Phase 4 Specification - Version 1.0 - COMPLETE (2024-12-07)*
