# Phase 12: Session Resilience & State Persistence

> **Making Vango applications survive disconnects, restarts, and page refreshes**

**Status: COMPLETE** | Completed: 2024-12-24 | All 39 tests passing

---

## Overview

Phase 12 addresses the critical "Day 2 Operations" gap: what happens when a server restarts, a user refreshes, or a network blip occurs? Currently, all in-memory session state is lost. This phase introduces a pluggable persistence layer, memory protection against DoS attacks, and user-facing reconnection feedback.

### Core Philosophy

**Sessions are durable by default. State loss should be exceptional, not expected.**

### Goals

1. **Session Serialization**: Persist session state across server restarts
2. **Memory Protection**: Prevent OOM from excessive detached sessions
3. **Reconnection UX**: Visual feedback during connection interruptions
4. **URLParam 2.0**: Enhanced URL state management with debouncing and encoding options
5. **Preference Sync**: Cross-device/cross-tab preference consistency

### Non-Goals (Explicit Exclusions)

1. Multi-server session sharing (Phase 16: Distributed Vango)
2. Offline-first data sync (Phase 16: `ctx.Sync`)
3. Full WASM mode state management (separate concern)

---

## Subsystems

| Subsystem | Purpose | Priority |
|-----------|---------|----------|
| 12.1 SessionStore Interface | Pluggable persistence backends | Critical |
| 12.2 Session Serialization | Convert session state to bytes | Critical |
| 12.3 Memory Protection | LRU eviction, per-IP limits | Critical |
| 12.4 Reconnection UX | CSS classes, optional toast | High |
| 12.5 URLParam 2.0 | Push/Replace modes, debounce, encoding | High |
| 12.6 Preference Sync | Cross-device merge strategies | Medium |
| 12.7 Testing Infrastructure | Session lifecycle simulation | High |

---

## 12.1 SessionStore Interface

### Problem

Session state is stored in memory only. Server restarts lose all user state, forcing users to start over.

### Solution

Define a `SessionStore` interface that backends can implement to persist session state.

### Interface Definition

```go
// pkg/session/store.go

package session

import (
    "context"
    "time"
)

// SessionStore defines the interface for session persistence backends.
type SessionStore interface {
    // Save persists session state. Called periodically and on graceful shutdown.
    Save(ctx context.Context, sessionID string, data []byte, expiresAt time.Time) error
    
    // Load retrieves session state. Returns nil, nil if session doesn't exist.
    Load(ctx context.Context, sessionID string) ([]byte, error)
    
    // Delete removes a session. Called on explicit logout or expiration.
    Delete(ctx context.Context, sessionID string) error
    
    // Touch updates the expiration time without loading full state.
    Touch(ctx context.Context, sessionID string, expiresAt time.Time) error
    
    // SaveAll persists multiple sessions atomically. Used during graceful shutdown.
    SaveAll(ctx context.Context, sessions map[string][]byte, expiresAt time.Time) error
}
```

### Built-in Implementations

#### MemoryStore (Default)

```go
// pkg/session/memory.go

type MemoryStore struct {
    mu       sync.RWMutex
    sessions map[string]*storedSession
}

type storedSession struct {
    data      []byte
    expiresAt time.Time
}

func NewMemoryStore() *MemoryStore {
    store := &MemoryStore{
        sessions: make(map[string]*storedSession),
    }
    go store.cleanupLoop()
    return store
}

func (m *MemoryStore) Save(ctx context.Context, sessionID string, data []byte, expiresAt time.Time) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.sessions[sessionID] = &storedSession{data: data, expiresAt: expiresAt}
    return nil
}

func (m *MemoryStore) Load(ctx context.Context, sessionID string) ([]byte, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    if s, ok := m.sessions[sessionID]; ok && time.Now().Before(s.expiresAt) {
        return s.data, nil
    }
    return nil, nil
}
```

#### RedisStore (Production)

```go
// pkg/session/redis.go

type RedisStore struct {
    client *redis.Client
    prefix string
}

func NewRedisStore(client *redis.Client, opts ...RedisOption) *RedisStore {
    store := &RedisStore{
        client: client,
        prefix: "vango:session:",
    }
    for _, opt := range opts {
        opt(store)
    }
    return store
}

func (r *RedisStore) Save(ctx context.Context, sessionID string, data []byte, expiresAt time.Time) error {
    ttl := time.Until(expiresAt)
    if ttl <= 0 {
        return r.Delete(ctx, sessionID)
    }
    return r.client.Set(ctx, r.prefix+sessionID, data, ttl).Err()
}

func (r *RedisStore) Load(ctx context.Context, sessionID string) ([]byte, error) {
    data, err := r.client.Get(ctx, r.prefix+sessionID).Bytes()
    if err == redis.Nil {
        return nil, nil
    }
    return data, err
}

func (r *RedisStore) SaveAll(ctx context.Context, sessions map[string][]byte, expiresAt time.Time) error {
    pipe := r.client.Pipeline()
    ttl := time.Until(expiresAt)
    for id, data := range sessions {
        pipe.Set(ctx, r.prefix+id, data, ttl)
    }
    _, err := pipe.Exec(ctx)
    return err
}
```

#### SQLStore (PostgreSQL/MySQL)

```go
// pkg/session/sql.go

type SQLStore struct {
    db        *sql.DB
    tableName string
}

func NewSQLStore(db *sql.DB, opts ...SQLOption) *SQLStore {
    store := &SQLStore{
        db:        db,
        tableName: "vango_sessions",
    }
    for _, opt := range opts {
        opt(store)
    }
    return store
}

// Requires table:
// CREATE TABLE vango_sessions (
//     id VARCHAR(64) PRIMARY KEY,
//     data BYTEA NOT NULL,
//     expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
//     created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
//     updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
// );
// CREATE INDEX idx_vango_sessions_expires ON vango_sessions(expires_at);

func (s *SQLStore) Save(ctx context.Context, sessionID string, data []byte, expiresAt time.Time) error {
    query := fmt.Sprintf(`
        INSERT INTO %s (id, data, expires_at, updated_at)
        VALUES ($1, $2, $3, NOW())
        ON CONFLICT (id) DO UPDATE SET
            data = EXCLUDED.data,
            expires_at = EXCLUDED.expires_at,
            updated_at = NOW()
    `, s.tableName)
    _, err := s.db.ExecContext(ctx, query, sessionID, data, expiresAt)
    return err
}
```

### Server Configuration

```go
// pkg/server/config.go

type Config struct {
    // ... existing fields ...
    
    // Session persistence
    SessionStore session.SessionStore // nil = MemoryStore (default)
    
    // How often to persist dirty sessions (0 = only on disconnect)
    PersistInterval time.Duration // default: 30s
    
    // Grace period for graceful shutdown session save
    ShutdownTimeout time.Duration // default: 10s
}
```

---

## 12.2 Session Serialization

### Problem

Not all session state should be persisted. Some state is transient (cursors, hover states), while other state is critical (form data, user context).

### Solution

Implement a `Serialize()` / `Deserialize()` mechanism that respects signal options.

### Signal Persistence Options

```go
// pkg/vango/signal.go

type SignalOption func(*signalConfig)

// Transient marks a signal as non-persistent (not saved to store).
func Transient() SignalOption {
    return func(c *signalConfig) {
        c.transient = true
    }
}

// PersistKey sets an explicit key for serialization (default: auto-generated).
func PersistKey(key string) SignalOption {
    return func(c *signalConfig) {
        c.persistKey = key
    }
}

// Usage
cursor := vango.Signal(Point{0, 0}, vango.Transient())     // Not persisted
userID := vango.Signal(0, vango.PersistKey("user_id"))    // Persisted with key
formData := vango.Signal(Form{})                           // Persisted with auto-key
```

### SerializableSession

```go
// pkg/session/serialize.go

type SerializableSession struct {
    ID        string                 `json:"id"`
    UserID    string                 `json:"user_id,omitempty"`
    CreatedAt time.Time              `json:"created_at"`
    Values    map[string]any         `json:"values"`      // Session.Get/Set values
    Signals   map[string]json.RawMessage `json:"signals"` // Persisted signals
    Route     string                 `json:"route"`       // Current page
}

func (s *Session) Serialize() ([]byte, error) {
    ss := SerializableSession{
        ID:        s.ID,
        UserID:    s.userID,
        CreatedAt: s.createdAt,
        Values:    s.values,
        Signals:   make(map[string]json.RawMessage),
        Route:     s.currentRoute,
    }
    
    for key, sig := range s.signals {
        if sig.IsTransient() {
            continue // Skip transient signals
        }
        data, err := json.Marshal(sig.Get())
        if err != nil {
            return nil, fmt.Errorf("serialize signal %s: %w", key, err)
        }
        ss.Signals[key] = data
    }
    
    return json.Marshal(ss)
}

func (s *Session) Deserialize(data []byte) error {
    var ss SerializableSession
    if err := json.Unmarshal(data, &ss); err != nil {
        return err
    }
    
    s.ID = ss.ID
    s.userID = ss.UserID
    s.createdAt = ss.CreatedAt
    s.values = ss.Values
    s.currentRoute = ss.Route
    
    // Signals are restored on-demand when components re-render
    s.persistedSignals = ss.Signals
    return nil
}
```

### Graceful Shutdown

```go
// pkg/server/server.go

func (s *Server) Shutdown(ctx context.Context) error {
    // Stop accepting new connections
    s.listener.Close()
    
    // Collect all active sessions
    sessions := make(map[string][]byte)
    s.sessions.Range(func(id string, sess *Session) bool {
        if data, err := sess.Serialize(); err == nil {
            sessions[id] = data
        }
        return true
    })
    
    // Persist all sessions
    if s.config.SessionStore != nil {
        expiresAt := time.Now().Add(s.config.ResumeWindow)
        if err := s.config.SessionStore.SaveAll(ctx, sessions, expiresAt); err != nil {
            log.Printf("WARN: Failed to save sessions on shutdown: %v", err)
        } else {
            log.Printf("INFO: Saved %d sessions to store", len(sessions))
        }
    }
    
    // Close WebSocket connections gracefully
    return s.closeAllConnections(ctx)
}
```

---

## 12.3 Memory Protection

### Problem

A malicious actor could open thousands of connections from different IPs, then disconnect them all. Without limits, these "detached" sessions consume memory indefinitely until `ResumeWindow` expires.

### Solution

Implement LRU eviction and per-IP session limits.

### Configuration

```go
// pkg/server/config.go

type Config struct {
    // ... existing fields ...
    
    // Maximum detached sessions before LRU eviction starts
    MaxDetachedSessions int // default: 10000
    
    // Maximum active sessions per IP address
    MaxSessionsPerIP int // default: 100
    
    // Eviction policy for when limits are exceeded
    EvictionPolicy EvictionPolicy // default: LRU
}

type EvictionPolicy int

const (
    EvictionLRU EvictionPolicy = iota // Evict least recently active
    EvictionOldest                     // Evict by creation time
    EvictionRandom                     // Random eviction (fast but unfair)
)
```

### LRU Session Manager

```go
// pkg/session/manager.go

type Manager struct {
    mu              sync.RWMutex
    sessions        map[string]*Session
    detachedQueue   *list.List                // LRU queue for detached sessions
    detachedIndex   map[string]*list.Element  // Fast lookup into queue
    sessionsByIP    map[string]int            // Count per IP
    config          ManagerConfig
}

type ManagerConfig struct {
    MaxDetached   int
    MaxPerIP      int
    ResumeWindow  time.Duration
    Store         SessionStore
}

func (m *Manager) OnDisconnect(sess *Session) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    sess.disconnectedAt = time.Now()
    
    // Add to detached queue
    elem := m.detachedQueue.PushFront(sess.ID)
    m.detachedIndex[sess.ID] = elem
    
    // Evict if over limit
    for m.detachedQueue.Len() > m.config.MaxDetached {
        oldest := m.detachedQueue.Back()
        if oldest == nil {
            break
        }
        m.evictSession(oldest.Value.(string))
    }
}

func (m *Manager) OnReconnect(sessionID string) *Session {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    // Remove from detached queue
    if elem, ok := m.detachedIndex[sessionID]; ok {
        m.detachedQueue.Remove(elem)
        delete(m.detachedIndex, sessionID)
    }
    
    return m.sessions[sessionID]
}

func (m *Manager) CheckIPLimit(ip string) error {
    m.mu.RLock()
    count := m.sessionsByIP[ip]
    m.mu.RUnlock()
    
    if count >= m.config.MaxPerIP {
        return ErrTooManySessionsFromIP
    }
    return nil
}

func (m *Manager) evictSession(sessionID string) {
    sess := m.sessions[sessionID]
    if sess == nil {
        return
    }
    
    // Persist before eviction if store is configured
    if m.config.Store != nil {
        if data, err := sess.Serialize(); err == nil {
            ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
            m.config.Store.Save(ctx, sessionID, data, time.Now().Add(m.config.ResumeWindow))
            cancel()
        }
    }
    
    // Remove from memory
    delete(m.sessions, sessionID)
    if elem, ok := m.detachedIndex[sessionID]; ok {
        m.detachedQueue.Remove(elem)
        delete(m.detachedIndex, sessionID)
    }
    m.sessionsByIP[sess.ip]--
}
```

---

## 12.4 Reconnection UX

### Problem

Users have no visual feedback when their connection drops. The UI appears frozen until the connection is restored or times out.

### Solution

Add automatic CSS classes and optional toast notifications for connection state.

### CSS Classes (Automatic)

The thin client automatically adds/removes these classes on `document.body`:

```css
/* Connection state classes */
.vango-connected { }        /* WebSocket open and healthy */
.vango-reconnecting { }     /* Attempting to reconnect */
.vango-offline { }          /* Failed to reconnect, giving up */
```

### Default Styles (Optional)

```css
/* Included in vango base CSS, can be overridden */

body.vango-reconnecting::after {
    content: '';
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    height: 3px;
    background: linear-gradient(90deg, transparent, var(--primary), transparent);
    animation: vango-reconnect-pulse 1.5s ease-in-out infinite;
    z-index: 9999;
}

@keyframes vango-reconnect-pulse {
    0%, 100% { opacity: 0.3; }
    50% { opacity: 1; }
}

body.vango-offline {
    filter: grayscale(0.3);
}

body.vango-offline::before {
    content: 'Connection lost. Attempting to reconnect...';
    position: fixed;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    background: hsl(var(--background));
    border: 1px solid hsl(var(--border));
    padding: 1rem 2rem;
    border-radius: var(--radius);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
    z-index: 10000;
}
```

### Client Implementation

```javascript
// client/src/connection.js

class ConnectionManager {
    constructor(options) {
        this.state = 'connected';
        this.retryCount = 0;
        this.maxRetries = options.maxRetries ?? 10;
        this.baseDelay = options.baseDelay ?? 1000;
    }
    
    setState(newState) {
        const oldState = this.state;
        this.state = newState;
        
        // Update body classes
        document.body.classList.remove('vango-connected', 'vango-reconnecting', 'vango-offline');
        document.body.classList.add(`vango-${newState}`);
        
        // Dispatch custom event
        document.dispatchEvent(new CustomEvent('vango:connection', {
            detail: { state: newState, previousState: oldState }
        }));
        
        // Optional toast notification
        if (window.__VANGO_TOAST_ON_RECONNECT__ && newState === 'connected' && oldState !== 'connected') {
            this.showToast('Connection restored');
        }
    }
    
    onDisconnect() {
        this.setState('reconnecting');
        this.scheduleReconnect();
    }
    
    scheduleReconnect() {
        if (this.retryCount >= this.maxRetries) {
            this.setState('offline');
            return;
        }
        
        const delay = Math.min(
            this.baseDelay * Math.pow(2, this.retryCount),
            30000 // Max 30s
        );
        
        setTimeout(() => this.attemptReconnect(), delay);
        this.retryCount++;
    }
    
    onReconnect() {
        this.retryCount = 0;
        this.setState('connected');
    }
}
```

### Server Configuration

```go
// pkg/server/config.go

type ReconnectConfig struct {
    // Show toast when connection is restored
    ToastOnReconnect bool // default: false
    
    // Custom message for the toast
    ToastMessage string // default: "Connection restored"
    
    // Maximum reconnection attempts before showing offline state
    MaxRetries int // default: 10
    
    // Base delay for exponential backoff (ms)
    BaseDelay int // default: 1000
}
```

---

## 12.5 URLParam 2.0

### Problem

Current `URLParam` has limitations:
1. Every update creates a new history entry (annoying back button behavior)
2. No support for complex types (arrays, structs)
3. No debouncing (search inputs spam history)

### Solution

Enhanced `URLParam` with modes, encodings, and debouncing.

### Naming Decision

**Important**: URLParam options use **values** (`vango.Replace`, `vango.Push`) not functions. This avoids collision with navigation which uses `ctx.Navigate().Replace()` as a method.

| API | Type | Usage |
|-----|------|-------|
| `vango.Replace` | Value (URLParamOption) | `vango.URLParam("q", "", vango.Replace)` |
| `vango.Push` | Value (URLParamOption) | `vango.URLParam("q", "", vango.Push)` |
| `ctx.Navigate().Replace()` | Method | `ctx.Navigate("/path").Replace()` |

### New API

```go
// pkg/vango/urlparam.go

type URLParamOption interface {
    applyURLParam(*urlParamConfig)
}

// Mode options as values (not functions) to avoid collision with navigation
var (
    // Push creates a new history entry (default behavior)
    Push URLParamOption = urlModeOption{mode: URLModePush}
    
    // Replace updates URL without creating history entry (use for filters, search)
    Replace URLParamOption = urlModeOption{mode: URLModeReplace}
)

type urlModeOption struct {
    mode URLMode
}

func (o urlModeOption) applyURLParam(c *urlParamConfig) {
    c.mode = o.mode
}

// Debounce delays URL updates by the specified duration.
func Debounce(d time.Duration) URLParamOption {
    return debounceOption{d: d}
}

type debounceOption struct {
    d time.Duration
}

func (o debounceOption) applyURLParam(c *urlParamConfig) {
    c.debounce = o.d
}

// Encoding specifies how complex types are serialized to URLs.
func Encoding(e URLEncoding) URLParamOption {
    return encodingOption{e: e}
}

type encodingOption struct {
    e URLEncoding
}

func (o encodingOption) applyURLParam(c *urlParamConfig) {
    c.encoding = o.e
}

type URLEncoding int

const (
    // URLEncodingFlat serializes structs as flat params: ?cat=tech&sort=asc
    URLEncodingFlat URLEncoding = iota
    
    // URLEncodingJSON serializes as compressed JSON: ?filter=eyJjYXQiOiJ0ZWNoIn0
    URLEncodingJSON
    
    // URLEncodingComma serializes arrays as comma-separated: ?tags=go,web,api
    URLEncodingComma
)

type URLMode int

const (
    URLModePush URLMode = iota    // Default: adds history entry
    URLModeReplace                 // Replaces current history entry
)
```

### Usage Examples

```go
// Search input - replaces history, debounced
// Note: vango.Replace is a value, not a function call
searchQuery := vango.URLParam("q", "", vango.Replace, vango.Debounce(300*time.Millisecond))

// Filter struct - flat encoding
type Filters struct {
    Category string `url:"cat"`
    SortBy   string `url:"sort"`
    Page     int    `url:"page"`
}
filters := vango.URLParam("", Filters{}, vango.Encoding(vango.URLEncodingFlat))
// URL: /products?cat=electronics&sort=price&page=1

// Tag array - comma encoding
tags := vango.URLParam("tags", []string{}, vango.Encoding(vango.URLEncodingComma))
// URL: /products?tags=go,web,api

// Explicit push (creates history entry - this is the default)
page := vango.URLParam("page", 1, vango.Push)
```

### Protocol Extension

**Important**: These patches are for **URLParam updates only** (query param changes on the current page). Navigation URL updates use a distinct `ControlNav` envelope. See Phase 14 section 7.3 for the full distinction.

```go
// New patch type for URL updates (query params only, not navigation)
const (
    PatchURLPush    = 0x30 // Update query params, push to history
    PatchURLReplace = 0x31 // Update query params, replace current entry
)

type URLPatch struct {
    Mode   URLMode
    Params map[string]string  // Query params to set/clear
}
```

**Scope**: URLParam patches only modify query parameters. They do NOT change the path or trigger component re-rendering. For route changes, use `ctx.Navigate()` which sends a `ControlNav` envelope with both URL and component patches.

### Client Implementation

```javascript
// client/src/url.js

class URLManager {
    constructor() {
        this.pending = new Map(); // key -> {value, timer}
    }
    
    applyPatch(patch) {
        const url = new URL(window.location);
        
        for (const [key, value] of Object.entries(patch.params)) {
            if (value === '') {
                url.searchParams.delete(key);
            } else {
                url.searchParams.set(key, value);
            }
        }
        
        if (patch.mode === 'push') {
            history.pushState(null, '', url);
        } else {
            history.replaceState(null, '', url);
        }
    }
}
```

---

## 12.6 Preference Sync

### Problem

User preferences (theme, sidebar state, notification settings) need to:
1. Work for anonymous users (LocalStorage)
2. Sync when user logs in (merge with DB)
3. Stay consistent across tabs/devices

### Solution

`Pref` primitive with configurable merge strategies.

### Merge Strategies

```go
// pkg/vango/pref.go

type MergeStrategy int

const (
    // MergeDBWins uses server value, discards local
    MergeDBWins MergeStrategy = iota
    
    // MergeLocalWins uses local value, updates server
    MergeLocalWins
    
    // MergePrompt notifies user to choose
    MergePrompt
    
    // MergeLWW uses last-write-wins with timestamps
    MergeLWW
)

type PrefOption func(*prefConfig)

func MergeWith(strategy MergeStrategy) PrefOption {
    return func(c *prefConfig) {
        c.mergeStrategy = strategy
    }
}

func OnConflict(handler func(local, remote any) any) PrefOption {
    return func(c *prefConfig) {
        c.conflictHandler = handler
    }
}
```

### Cross-Tab Sync

```go
// Prefs use BroadcastChannel for cross-tab sync
type Pref[T any] struct {
    key       string
    value     T
    updatedAt time.Time
    config    prefConfig
}

func (p *Pref[T]) Set(value T) {
    p.value = value
    p.updatedAt = time.Now()
    
    // Broadcast to other tabs
    ctx.BroadcastPref(p.key, value, p.updatedAt)
    
    // Persist to backend if logged in
    if ctx.Session().IsAuthenticated() {
        ctx.PersistPref(p.key, value, p.updatedAt)
    }
}
```

### Client BroadcastChannel

```javascript
// client/src/prefs.js

class PrefSync {
    constructor() {
        this.channel = new BroadcastChannel('vango:prefs');
        this.channel.onmessage = (e) => this.onRemoteUpdate(e.data);
    }
    
    broadcast(key, value, timestamp) {
        this.channel.postMessage({ key, value, timestamp });
    }
    
    onRemoteUpdate({ key, value, timestamp }) {
        // LWW: only apply if newer
        const local = this.timestamps.get(key) ?? 0;
        if (timestamp > local) {
            this.applyRemote(key, value);
            this.timestamps.set(key, timestamp);
        }
    }
}
```

---

## 12.7 Testing Infrastructure

### Problem

Testing session lifecycle (disconnect, reconnect, server restart) is difficult without proper test utilities.

### Solution

`vango.NewTestSession` with lifecycle simulation methods.

### TestSession API

```go
// pkg/vtest/session.go

package vtest

type TestSession struct {
    *vango.Session
    app     *vango.App
    store   session.SessionStore
}

// NewTestSession creates a session for testing with optional store.
func NewTestSession(app *vango.App, opts ...TestOption) *TestSession {
    return &TestSession{
        Session: app.NewSession(),
        app:     app,
        store:   session.NewMemoryStore(),
    }
}

// SimulateDisconnect simulates a WebSocket disconnect.
func (t *TestSession) SimulateDisconnect() {
    t.app.OnDisconnect(t.Session)
}

// SimulateReconnect simulates a WebSocket reconnect within ResumeWindow.
func (t *TestSession) SimulateReconnect() error {
    restored := t.app.OnReconnect(t.Session.ID)
    if restored == nil {
        return ErrSessionExpired
    }
    t.Session = restored
    return nil
}

// SimulateRefresh simulates a full page refresh (new WS connection, same session ID cookie).
func (t *TestSession) SimulateRefresh() error {
    // Serialize current state
    data, err := t.Session.Serialize()
    if err != nil {
        return err
    }
    
    // Store to backend
    t.store.Save(context.Background(), t.Session.ID, data, time.Now().Add(time.Hour))
    
    // Create new session from stored data
    newSession := t.app.NewSession()
    storedData, _ := t.store.Load(context.Background(), t.Session.ID)
    if err := newSession.Deserialize(storedData); err != nil {
        return err
    }
    
    t.Session = newSession
    return nil
}

// SimulateServerRestart simulates a full server restart.
func (t *TestSession) SimulateServerRestart() error {
    // Save all sessions
    data, _ := t.Session.Serialize()
    t.store.Save(context.Background(), t.Session.ID, data, time.Now().Add(time.Hour))
    
    // Create new app instance (simulates restart)
    newApp := vango.NewApp(t.app.Config())
    
    // Restore session from store
    storedData, _ := t.store.Load(context.Background(), t.Session.ID)
    newSession := newApp.RestoreSession(storedData)
    
    t.app = newApp
    t.Session = newSession
    return nil
}

// GetSignal returns the current value of a signal by key.
func (t *TestSession) GetSignal(key string) any {
    return t.Session.SignalByKey(key).Get()
}

// SetSignal updates a signal value for testing.
func (t *TestSession) SetSignal(key string, value any) {
    t.Session.SignalByKey(key).SetAny(value)
}
```

### Usage Example

```go
func TestCartSurvivesRestart(t *testing.T) {
    app := vango.NewApp()
    sess := vtest.NewTestSession(app)
    
    // Add item to cart
    sess.Navigate("/cart")
    sess.Click("#add-item-1")
    
    // Verify item added
    cart := sess.GetSignal("cart").([]CartItem)
    assert.Len(t, cart, 1)
    
    // Simulate server restart
    err := sess.SimulateServerRestart()
    require.NoError(t, err)
    
    // Verify cart survived
    cart = sess.GetSignal("cart").([]CartItem)
    assert.Len(t, cart, 1)
    assert.Equal(t, "item-1", cart[0].ID)
}
```

---

## Exit Criteria

**Status: COMPLETE** (2024-12-24)

### 12.1 SessionStore Interface
- [x] `SessionStore` interface defined with Save/Load/Delete/Touch/SaveAll
- [x] `MemoryStore` implemented with cleanup goroutine
- [x] `RedisStore` implemented with TTL support
- [x] `SQLStore` implemented with PostgreSQL support
- [x] Server config accepts `SessionStore` option

### 12.2 Session Serialization
- [x] `Transient()` signal option prevents persistence
- [x] `PersistKey()` allows explicit key naming
- [x] `Session.Serialize()` produces valid JSON
- [x] `Session.Deserialize()` restores full state
- [x] `Server.Shutdown()` saves all active sessions

### 12.3 Memory Protection
- [x] `MaxDetachedSessions` config enforced
- [x] `MaxSessionsPerIP` config enforced
- [x] LRU eviction implemented
- [x] Evicted sessions persisted before removal
- [x] `ErrTooManySessionsFromIP` returned when limit exceeded

### 12.4 Reconnection UX
- [x] `.vango-connecting`, `.vango-connected`, `.vango-reconnecting`, `.vango-disconnected` CSS classes added
- [x] Classes applied to `<html>` element (document.documentElement)
- [x] Exponential backoff reconnection logic
- [x] Optional toast on reconnect
- [x] `vango:connection` custom event dispatched

### 12.5 URLParam 2.0
- [x] `Replace` option prevents history spam
- [x] `Debounce()` option implemented
- [x] `EncodingFlat` for struct params
- [x] `EncodingComma` for array params
- [x] `PatchURLPush` and `PatchURLReplace` protocol messages

### 12.6 Preference Sync
- [x] `Pref` primitive implemented
- [x] `DBWins`, `LocalWins`, `LWW` strategies
- [x] `BroadcastChannel` cross-tab sync
- [x] Conflict handler callback support

### 12.7 Testing Infrastructure
- [x] `vtest.NewTestSession` implemented
- [x] `SimulateDisconnect()` works correctly
- [x] `SimulateReconnect()` restores session
- [x] `SimulateRefresh()` persists and restores
- [x] `SimulateServerRestart()` creates new app instance

---

## Files Changed

### New Files
- `pkg/session/store.go` - SessionStore interface and SessionData type
- `pkg/session/memory.go` - MemoryStore implementation with cleanup goroutine
- `pkg/session/redis.go` - RedisStore implementation with TTL and pipelining
- `pkg/session/sql.go` - SQLStore implementation for PostgreSQL/MySQL
- `pkg/session/manager.go` - LRU manager, IP limits, detached session tracking
- `pkg/session/serialize.go` - SerializableSession, Serialize/Deserialize functions
- `pkg/session/doc.go` - Package documentation
- `pkg/urlparam/urlparam.go` - URLParam 2.0 with modes, debounce, encodings
- `pkg/pref/pref.go` - Pref primitive with merge strategies
- `pkg/vtest/session.go` - TestSession with lifecycle simulation
- `client/src/connection.js` - ConnectionManager with CSS classes, toast, backoff
- `client/src/url.js` - URLManager for push/replace patches
- `client/src/prefs.js` - PrefManager with BroadcastChannel sync

### Modified Files
- `pkg/server/config.go` - Added Phase 12 config fields (SessionStore, ResumeWindow, MaxDetachedSessions, MaxSessionsPerIP, PersistInterval, ReconnectConfig)
- `pkg/server/server.go` - Integration with SessionManagerOptions for persistence
- `pkg/server/manager.go` - Added persistence integration (OnSessionDisconnect, OnSessionReconnect, CheckIPLimit, ShutdownWithContext)
- `pkg/server/session.go` - Added IP, CurrentRoute fields, GetAllData(), RestoreData() methods
- `pkg/vango/signal.go` - Added IsTransient(), PersistKey() methods
- `pkg/vango/signal_options.go` - Added Transient(), PersistKey() options
- `pkg/protocol/patch.go` - Added PatchURLPush (0x30) and PatchURLReplace (0x31)
- `pkg/vtest/session.go` - Improved serialization to include session data values
- `client/src/connection.js` - Fixed CSS classes per spec (vango-connecting, vango-disconnected), applied to html element
- `client/src/index.js` - ConnectionManager and PrefManager integration

### Test Files
- `pkg/session/store_test.go` - MemoryStore tests
- `pkg/session/manager_test.go` - Manager tests (LRU, IP limits, shutdown)
- `pkg/urlparam/urlparam_test.go` - URLParam tests
- `pkg/pref/pref_test.go` - Pref tests (merge strategies, concurrency)
- `pkg/server/manager_test.go` - Added Phase 12 integration tests

---

## Dependencies

- `github.com/redis/go-redis/v9` (optional, for RedisStore)
- `database/sql` (stdlib, for SQLStore)

---

## Migration Guide

### From V2.0 to V2.1

1. **No breaking changes** - all new features are opt-in
2. **Recommended**: Add `SessionStore` for production deployments
3. **Recommended**: Add `Transient()` to ephemeral signals (cursors, hover states)
4. **Optional**: Migrate `URLParam` calls to use `Replace()` for filters/search

---

## Implementation Notes

### Architecture Decisions

1. **Two-Layer Session Management**: The implementation uses two complementary managers:
   - `server.SessionManager` - Manages active WebSocket connections
   - `session.Manager` - Handles persistence, LRU eviction, and IP limits

   The server.SessionManager delegates persistence operations to session.Manager when a SessionStore is configured.

2. **Packages Structure**: Phase 12 features are organized into separate packages:
   - `pkg/session/` - SessionStore interface and persistence infrastructure
   - `pkg/urlparam/` - URLParam 2.0 (separate from pkg/vango for modularity)
   - `pkg/pref/` - Preference primitive with merge strategies
   - `pkg/vtest/` - Testing utilities (extended with TestSession)

3. **Backwards Compatibility**: All Phase 12 features are opt-in:
   - Without `SessionStore`, sessions remain in-memory only
   - URLParam defaults to `Push` mode (existing behavior)
   - Connection CSS classes are always added but have no visual effect without CSS

### Usage Examples

```go
// Enable session persistence with Redis
config := server.DefaultServerConfig().
    WithSessionStore(session.NewRedisStore(redisClient)).
    WithResumeWindow(10 * time.Minute).
    WithMaxDetachedSessions(5000).
    WithMaxSessionsPerIP(50)

// URLParam with Replace mode (no history spam)
searchQuery := urlparam.Param("q", "", urlparam.Replace, urlparam.Debounce(300*time.Millisecond))

// Preference with Last-Write-Wins
theme := pref.New("theme", "light", pref.MergeWith(pref.LWW))

// Testing session lifecycle
sess := vtest.NewTestSession(manager)
sess.SimulateDisconnect()
err := sess.SimulateReconnect()
```

### Test Coverage

All Phase 12 subsystems have comprehensive tests:
- `pkg/session/` - 9 tests (store, manager, LRU, IP limits)
- `pkg/urlparam/` - 10 tests (modes, encodings, debounce)
- `pkg/pref/` - 9 tests (merge strategies, concurrency)
- `pkg/vtest/` - 8 tests (lifecycle simulation)
- `pkg/server/` - 3 new tests (persistence integration)

Total: 39 new tests, all passing.

### Spec Audit (2026-01-02)

Following audit against VANGO_ARCHITECTURE_AND_GUIDE.md, the following corrections were made:

1. **Connection CSS Classes**: Fixed to match spec Section 5.4:
   - Added `vango-connecting` state for initial connection
   - Renamed `vango-offline` to `vango-disconnected`
   - Changed element from `<body>` to `<html>` (document.documentElement)
   - All four classes: `vango-connecting`, `vango-connected`, `vango-reconnecting`, `vango-disconnected`

2. **URLParam Package Location**: Per spec, URLParam is accessed via `urlparam.Param()` from the `pkg/urlparam` package, not re-exported from `pkg/vango` (to avoid import cycle since urlparam depends on vango.Signal).

3. **Session Serialization**: Added `GetAllData()` and `RestoreData()` methods to `server.Session` for proper serialization of session values.

4. **vtest Improvements**: Updated `serializeSession()` and `deserializeSession()` to properly persist and restore session data values.
