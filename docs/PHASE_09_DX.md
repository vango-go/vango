# Phase 9: Developer Experience

> **Making Vango a joy to use**

---

## Overview

Developer Experience (DX) is what separates good frameworks from great ones. Phase 9 focuses on the CLI, hot reload, error messages, debugging tools, and documentation that make developers productive from day one.

### Goals

1. **Zero to running app in < 60 seconds**
2. **Instant feedback during development**
3. **Errors that teach, not confuse**
4. **Debugging tools that expose framework internals**
5. **Documentation that answers questions before they're asked**

### Subsystems

| Subsystem | Purpose | Priority |
|-----------|---------|----------|
| 9.1 CLI | Project scaffolding and commands | Critical |
| 9.2 Hot Reload | Instant updates during development | Critical |
| 9.3 Error Messages | Actionable, educational errors | Critical |
| 9.4 DevTools | Browser extension for debugging | High |
| 9.5 Documentation | Guides, API reference, examples | Critical |
| 9.6 IDE Integration | VS Code extension, LSP | Medium |

---

## 9.1 CLI

### Commands

```bash
# Create new project
vango create <name> [--template=<template>]

# Development server with hot reload
vango dev [--port=3000] [--host=localhost]

# Production build
vango build [--output=dist]

# Run tests
vango test [--coverage] [--watch]

# Code generation
vango gen [routes|elements|hooks]

# Version and upgrade
vango version
vango upgrade
```

### `vango create`

Interactive project scaffolding with sensible defaults:

```bash
$ vango create my-app

  ╦  ╦┌─┐┌┐┌┌─┐┌─┐
  ╚╗╔╝├─┤│││├─┤│ │
   ╚╝ ┴ ┴┘└┘┴ ┴└─┘

  Creating a new Vango project...

? Project name: my-app
? Description: My awesome web app
? Include example pages? (Y/n) Y
? Include Tailwind CSS? (Y/n) Y
? Include database setup? (Y/n) n

  Creating project structure...
  ✓ Created my-app/
  ✓ Created my-app/app/routes/
  ✓ Created my-app/app/components/
  ✓ Created my-app/public/
  ✓ Initialized go.mod
  ✓ Installed dependencies
  ✓ Set up Tailwind CSS

  Done! To get started:

    cd my-app
    vango dev

  Your app will be running at http://localhost:3000
```

#### Project Templates

```bash
# Minimal (just the essentials)
vango create my-app --template=minimal

# Full (all features, example pages)
vango create my-app --template=full

# API-only (no UI, just API routes)
vango create my-app --template=api

# SaaS starter (auth, billing, dashboard)
vango create my-app --template=saas
```

#### Generated Structure

```
my-app/
├── app/
│   ├── routes/
│   │   ├── index.go           # Home page
│   │   ├── about.go           # About page
│   │   └── _layout.go         # Root layout
│   └── components/
│       ├── button.go
│       ├── card.go
│       └── navbar.go
├── public/
│   ├── favicon.ico
│   └── styles.css
├── go.mod
├── go.sum
├── main.go
├── vango.json                 # Configuration
├── tailwind.config.js         # If Tailwind enabled
└── README.md
```

#### main.go Template

```go
package main

import (
    "log"
    "my-app/app/routes"
    "vango"
)

func main() {
    app := vango.New()

    // Register routes (auto-generated)
    routes.Register(app)

    // Start server
    log.Println("Server running at http://localhost:3000")
    if err := app.Listen(":3000"); err != nil {
        log.Fatal(err)
    }
}
```

### `vango dev`

Development server with hot reload:

```bash
$ vango dev

  ╦  ╦┌─┐┌┐┌┌─┐┌─┐  ┌┬┐┌─┐┬  ┬
  ╚╗╔╝├─┤│││├─┤│ │   ││├┤ └┐┌┘
   ╚╝ ┴ ┴┘└┘┴ ┴└─┘  ─┴┘└─┘ └┘

  → Server starting on http://localhost:3000
  → Watching for changes...
  → Tailwind CSS compiling...

  Ready in 1.2s

[14:32:15] Changed: app/routes/index.go
[14:32:15] Rebuilding... (48ms)
[14:32:15] ✓ Reloaded 2 connected browsers

[14:32:45] Changed: app/components/button.go
[14:32:45] Rebuilding... (52ms)
[14:32:45] ✓ Reloaded 2 connected browsers
```

#### Dev Server Features

| Feature | Description |
|---------|-------------|
| Hot Reload | Instant updates on file change |
| Incremental Build | Only recompile changed packages |
| Error Overlay | Show errors in browser |
| Tailwind Watch | Auto-compile CSS |
| Open Browser | Auto-open on start (configurable) |
| HTTPS | Optional TLS for testing |
| Proxy | Forward API requests to backend |

#### Configuration (vango.json)

```json
{
  "dev": {
    "port": 3000,
    "host": "localhost",
    "openBrowser": true,
    "https": false,
    "proxy": {
      "/api/external": "https://api.example.com"
    }
  },
  "build": {
    "output": "dist",
    "minify": true,
    "sourceMaps": false
  },
  "tailwind": {
    "enabled": true,
    "config": "./tailwind.config.js"
  },
  "hooks": "./public/js/hooks.js"
}
```

### `vango build`

Production build:

```bash
$ vango build

  Building for production...

  ✓ Compiling Go (1.8s)
  ✓ Generating routes
  ✓ Bundling thin client (12.4 KB gzipped)
  ✓ Compiling Tailwind CSS (234 KB → 18 KB)
  ✓ Copying static assets
  ✓ Generating manifest

  Build complete in 3.2s

  Output:
    dist/
    ├── server          # Go binary (14.2 MB)
    ├── public/
    │   ├── vango.min.js
    │   ├── styles.css
    │   └── ...
    └── manifest.json

  To run:
    ./dist/server
```

#### Build Output

```
dist/
├── server                    # Compiled Go binary
├── public/
│   ├── vango.min.js          # Thin client (12 KB gzipped)
│   ├── vango.min.js.map      # Source map (if enabled)
│   ├── styles.css            # Compiled CSS
│   ├── styles.css.map
│   └── assets/               # Static files with hashes
│       ├── logo.a1b2c3.png
│       └── ...
└── manifest.json             # Asset manifest for cache busting
```

### `vango gen`

Code generation commands:

```bash
# Regenerate route registration code
vango gen routes

# Regenerate element functions (from HTML spec)
vango gen elements

# Regenerate hook bindings
vango gen hooks
```

### CLI Implementation

```go
// cmd/vango/main.go
package main

import (
    "os"

    "github.com/spf13/cobra"
)

func main() {
    root := &cobra.Command{
        Use:   "vango",
        Short: "The Go framework for modern web applications",
    }

    root.AddCommand(
        createCmd(),
        devCmd(),
        buildCmd(),
        testCmd(),
        genCmd(),
        versionCmd(),
        upgradeCmd(),
    )

    if err := root.Execute(); err != nil {
        os.Exit(1)
    }
}

// cmd/vango/create.go
func createCmd() *cobra.Command {
    var template string

    cmd := &cobra.Command{
        Use:   "create <name>",
        Short: "Create a new Vango project",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            name := args[0]
            return runCreate(name, template)
        },
    }

    cmd.Flags().StringVar(&template, "template", "full", "Project template")

    return cmd
}

func runCreate(name, template string) error {
    // Interactive prompts using bubbletea or similar
    config := promptForConfig(name)

    // Create directory structure
    if err := createProjectDir(name); err != nil {
        return err
    }

    // Copy template files
    if err := copyTemplate(template, name, config); err != nil {
        return err
    }

    // Initialize go.mod
    if err := initGoMod(name); err != nil {
        return err
    }

    // Install dependencies
    if err := runGoModTidy(name); err != nil {
        return err
    }

    // Setup Tailwind if enabled
    if config.Tailwind {
        if err := setupTailwind(name); err != nil {
            return err
        }
    }

    printSuccess(name)
    return nil
}
```

---

## 9.2 Hot Reload

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Development Server                       │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │ File        │  │ Go          │  │ HTTP/WebSocket      │  │
│  │ Watcher     │─▶│ Compiler    │─▶│ Server              │  │
│  └─────────────┘  └─────────────┘  └──────────┬──────────┘  │
│         │                                      │             │
│         │ File change                          │ Reload msg  │
│         ▼                                      ▼             │
│  ┌─────────────┐                    ┌─────────────────────┐  │
│  │ Tailwind    │                    │ Connected           │  │
│  │ Watcher     │                    │ Browsers            │  │
│  └─────────────┘                    └─────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### File Watcher

```go
// internal/dev/watcher.go
type Watcher struct {
    paths     []string
    ignore    []string
    debounce  time.Duration
    onChange  func(path string)
}

func NewWatcher(paths []string) *Watcher {
    return &Watcher{
        paths:    paths,
        ignore:   defaultIgnore,
        debounce: 100 * time.Millisecond,
    }
}

var defaultIgnore = []string{
    "*.test.go",
    "*_test.go",
    ".git",
    "node_modules",
    "dist",
    "tmp",
}

func (w *Watcher) Start(ctx context.Context) error {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return err
    }
    defer watcher.Close()

    // Add all paths recursively
    for _, path := range w.paths {
        if err := w.addRecursive(watcher, path); err != nil {
            return err
        }
    }

    // Debounce timer
    var timer *time.Timer

    for {
        select {
        case <-ctx.Done():
            return nil

        case event := <-watcher.Events:
            if w.shouldIgnore(event.Name) {
                continue
            }

            if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
                if timer != nil {
                    timer.Stop()
                }
                timer = time.AfterFunc(w.debounce, func() {
                    w.onChange(event.Name)
                })
            }

        case err := <-watcher.Errors:
            log.Printf("Watcher error: %v", err)
        }
    }
}
```

### Incremental Compilation

```go
// internal/dev/compiler.go
type Compiler struct {
    projectPath string
    lastBuild   map[string]time.Time  // package -> last modified
    binary      string
    mu          sync.Mutex
}

func (c *Compiler) Build() (time.Duration, error) {
    start := time.Now()

    c.mu.Lock()
    defer c.mu.Unlock()

    // Use go build with cache
    cmd := exec.Command("go", "build",
        "-o", c.binary,
        "./...",
    )
    cmd.Dir = c.projectPath
    cmd.Env = append(os.Environ(),
        "GOCACHE="+filepath.Join(c.projectPath, ".vango", "cache"),
    )

    output, err := cmd.CombinedOutput()
    if err != nil {
        return 0, &BuildError{
            Output: string(output),
            Err:    err,
        }
    }

    return time.Since(start), nil
}

func (c *Compiler) Restart() error {
    // Kill existing process
    if c.process != nil {
        c.process.Kill()
        c.process.Wait()
    }

    // Start new process
    cmd := exec.Command(c.binary)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    cmd.Env = append(os.Environ(),
        "VANGO_DEV=1",
    )

    if err := cmd.Start(); err != nil {
        return err
    }

    c.process = cmd.Process
    return nil
}
```

### Reload Protocol

When files change, the dev server notifies connected browsers:

```go
// internal/dev/reload.go
type ReloadServer struct {
    clients map[*websocket.Conn]bool
    mu      sync.RWMutex
}

func (r *ReloadServer) NotifyReload() {
    r.mu.RLock()
    defer r.mu.RUnlock()

    message := []byte(`{"type":"reload"}`)

    for client := range r.clients {
        client.WriteMessage(websocket.TextMessage, message)
    }
}

func (r *ReloadServer) NotifyError(err *BuildError) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    message, _ := json.Marshal(map[string]any{
        "type":  "error",
        "error": err.Format(),
    })

    for client := range r.clients {
        client.WriteMessage(websocket.TextMessage, message)
    }
}
```

### Browser Client (Dev Only)

```javascript
// Injected only in development mode
(function() {
    const ws = new WebSocket(`ws://${location.host}/_vango/reload`);

    ws.onmessage = (e) => {
        const msg = JSON.parse(e.data);

        if (msg.type === 'reload') {
            location.reload();
        }

        if (msg.type === 'error') {
            showErrorOverlay(msg.error);
        }
    };

    ws.onclose = () => {
        // Reconnect after delay
        setTimeout(() => location.reload(), 1000);
    };

    function showErrorOverlay(error) {
        const overlay = document.createElement('div');
        overlay.id = 'vango-error-overlay';
        overlay.innerHTML = `
            <div class="vango-error">
                <h2>Build Error</h2>
                <pre>${escapeHtml(error)}</pre>
                <p>Fix the error and save to reload.</p>
            </div>
        `;
        document.body.appendChild(overlay);
    }
})();
```

### State Preservation (Future)

For advanced hot reload that preserves component state:

```go
// Future: HMR-style state preservation
type StateSnapshot struct {
    Signals map[string]any
    URL     string
    Scroll  int
}

func (s *Session) CaptureState() StateSnapshot {
    // Serialize current signal values
}

func (s *Session) RestoreState(snap StateSnapshot) {
    // Restore signal values after reload
}
```

---

## 9.3 Error Messages

### Design Principles

1. **Show the exact location** (file, line, column)
2. **Explain what went wrong** in plain language
3. **Suggest how to fix it** with code examples
4. **Link to documentation** for deeper understanding

### Error Categories

| Category | Example |
|----------|---------|
| Compile-time | Type mismatch, missing import |
| Runtime | Signal read outside component, nil pointer |
| Hydration | Server/client mismatch |
| Protocol | Invalid message, connection lost |
| Validation | Form validation, route parameter |

### Error Format

```go
// internal/errors/error.go
type VangoError struct {
    Code       string       // e.g., "E001"
    Category   string       // e.g., "runtime", "compile"
    Message    string       // Short description
    Detail     string       // Longer explanation
    Location   *Location    // File, line, column
    Suggestion string       // How to fix
    DocURL     string       // Link to docs
    Context    []string     // Surrounding code lines
}

type Location struct {
    File   string
    Line   int
    Column int
}

func (e *VangoError) Error() string {
    return e.Message
}

func (e *VangoError) Format() string {
    var b strings.Builder

    // Header
    fmt.Fprintf(&b, "\n%s %s\n\n",
        color.Red("ERROR"),
        color.White(e.Code+": "+e.Message),
    )

    // Location
    if e.Location != nil {
        fmt.Fprintf(&b, "  %s:%d:%d\n\n",
            e.Location.File,
            e.Location.Line,
            e.Location.Column,
        )

        // Context with arrow
        for i, line := range e.Context {
            lineNum := e.Location.Line - len(e.Context)/2 + i
            if lineNum == e.Location.Line {
                fmt.Fprintf(&b, "  %s %d │ %s\n",
                    color.Red("→"),
                    lineNum,
                    line,
                )
                // Arrow pointing to column
                fmt.Fprintf(&b, "      │ %s%s\n",
                    strings.Repeat(" ", e.Location.Column-1),
                    color.Red("^"),
                )
            } else {
                fmt.Fprintf(&b, "    %d │ %s\n", lineNum, line)
            }
        }
        fmt.Fprintln(&b)
    }

    // Detail
    if e.Detail != "" {
        fmt.Fprintf(&b, "  %s\n\n", e.Detail)
    }

    // Suggestion
    if e.Suggestion != "" {
        fmt.Fprintf(&b, "  %s %s\n\n",
            color.Cyan("Hint:"),
            e.Suggestion,
        )
    }

    // Doc link
    if e.DocURL != "" {
        fmt.Fprintf(&b, "  %s %s\n",
            color.Gray("Learn more:"),
            e.DocURL,
        )
    }

    return b.String()
}
```

### Example Errors

#### Signal Outside Component

```
ERROR E001: Signal read outside component context

  app/routes/index.go:15:12

    13 │ func HomePage() *vango.VNode {
    14 │     count := vango.Signal(0)
  → 15 │     value := count()
       │             ^
    16 │     return Div(Text(fmt.Sprintf("%d", value)))
    17 │ }

  Signals must be read inside a component's render function,
  wrapped with vango.Func().

  Hint: Wrap your component logic in vango.Func(func() *vango.VNode { ... })

  Example:
    func HomePage() vango.Component {
        return vango.Func(func() *vango.VNode {
            count := vango.Signal(0)
            return Div(Text(fmt.Sprintf("%d", count())))
        })
    }

  Learn more: https://vango.dev/docs/errors/E001
```

#### Hydration Mismatch

```
ERROR E042: Hydration mismatch detected

  app/components/status.go:8

  Server rendered:
    <div class="status">Offline</div>

  Client expected:
    <div class="status">Online</div>

  The component reads browser-only state (navigator.onLine)
  during render. Server doesn't have access to this value.

  Hint: Use an Effect to read browser state after mount:

    func StatusIndicator() vango.Component {
        return vango.Func(func() *vango.VNode {
            status := vango.Signal("unknown")

            vango.OnMount(func() {
                status.Set(getOnlineStatus())
            })

            return Div(Class("status"), Text(status()))
        })
    }

  Learn more: https://vango.dev/docs/errors/E042
```

#### Route Parameter Type Error

```
ERROR E023: Route parameter type mismatch

  app/routes/users/[id].go:12

    10 │ type Params struct {
    11 │     ID int `param:"id"`
  → 12 │ }
       │
    13 │
    14 │ func Page(ctx vango.Ctx, p Params) vango.Component {

  Route parameter "id" is defined as int but received "abc".
  URL: /users/abc

  Hint: Either validate the parameter or use string type:

    type Params struct {
        ID string `param:"id"`
    }

  Or add a route constraint:

    app/routes/users/[id:int].go

  Learn more: https://vango.dev/docs/errors/E023
```

### Error Registry

```go
// internal/errors/registry.go
var errors = map[string]ErrorTemplate{
    "E001": {
        Message:  "Signal read outside component context",
        Category: "runtime",
        Detail:   "Signals must be read inside a component's render function, wrapped with vango.Func().",
        DocURL:   "https://vango.dev/docs/errors/E001",
    },
    "E002": {
        Message:  "Effect created outside component context",
        Category: "runtime",
        Detail:   "Effects must be created inside a component's render function.",
        DocURL:   "https://vango.dev/docs/errors/E002",
    },
    // ... more errors
}

func NewError(code string) *VangoError {
    template := errors[code]
    return &VangoError{
        Code:     code,
        Category: template.Category,
        Message:  template.Message,
        Detail:   template.Detail,
        DocURL:   template.DocURL,
    }
}

func (e *VangoError) WithLocation(file string, line, col int) *VangoError {
    e.Location = &Location{File: file, Line: line, Column: col}
    e.Context = readContextLines(file, line, 5)
    return e
}

func (e *VangoError) WithSuggestion(s string) *VangoError {
    e.Suggestion = s
    return e
}
```

---

## 9.4 DevTools

### Browser Extension

A Chrome/Firefox extension for debugging Vango applications:

```
┌─────────────────────────────────────────────────────────────┐
│ Vango DevTools                                     [Signals]│
├─────────────────────────────────────────────────────────────┤
│                                                             │
│ Components                    │ Signals                     │
│ ────────────                  │ ────────                    │
│ ▼ App                         │ count: 5                    │
│   ▼ Layout                    │ user: {id: 1, name: "John"} │
│     ▶ Header                  │ items: [...] (3 items)      │
│     ▼ Main                    │                             │
│       ▼ Counter ◉             │ ─────────────────────────   │
│         Button                │ History                     │
│         Button                │ ─────────────────────────   │
│     ▶ Footer                  │ 14:32:15 count: 4 → 5       │
│                               │ 14:32:10 count: 3 → 4       │
│                               │ 14:32:05 count: 2 → 3       │
│                               │                             │
├─────────────────────────────────────────────────────────────┤
│ Network                                                     │
│ ──────────────────────────────────────────────────────────  │
│ → Event: Click (h42)                               2ms      │
│ ← Patches: 2 (SetText, SetClass)                  48ms      │
│ → Event: Input (h15)                               1ms      │
│ ← Patches: 1 (SetText)                            52ms      │
└─────────────────────────────────────────────────────────────┘
```

### Features

| Feature | Description |
|---------|-------------|
| Component Tree | Hierarchical view of mounted components |
| Signal Inspector | View and edit signal values |
| Signal History | Time-travel through state changes |
| Network Panel | WebSocket events and patches |
| Performance | Render timing and bottlenecks |
| Console | Log signal changes and effects |

### DevTools Protocol

The app exposes a debug endpoint in development mode:

```go
// Only in dev mode
if os.Getenv("VANGO_DEV") == "1" {
    app.Get("/_vango/devtools", devtools.Handler)
    app.WS("/_vango/devtools/ws", devtools.WebSocketHandler)
}
```

```go
// internal/devtools/handler.go
type DevToolsServer struct {
    sessions *server.SessionManager
}

func (d *DevToolsServer) HandleWebSocket(conn *websocket.Conn) {
    for {
        var msg DevToolsMessage
        if err := conn.ReadJSON(&msg); err != nil {
            return
        }

        switch msg.Type {
        case "getComponentTree":
            tree := d.sessions.Get(msg.SessionID).GetComponentTree()
            conn.WriteJSON(map[string]any{
                "type": "componentTree",
                "data": tree,
            })

        case "getSignals":
            signals := d.sessions.Get(msg.SessionID).GetSignals()
            conn.WriteJSON(map[string]any{
                "type": "signals",
                "data": signals,
            })

        case "setSignal":
            d.sessions.Get(msg.SessionID).SetSignal(msg.Name, msg.Value)

        case "subscribe":
            // Subscribe to real-time updates
            d.subscribeToSession(conn, msg.SessionID)
        }
    }
}
```

### Console Logging

In development, signal changes are logged:

```go
// Automatically enabled in dev mode
func (s *Signal[T]) Set(value T) {
    if devMode {
        log.Printf("[Signal] %s: %v → %v", s.name, s.value, value)
    }
    // ... normal set logic
}
```

Output:
```
[14:32:15.123] [Signal] count: 4 → 5
               Source: app/routes/counter.go:24
               Subscribers: [Counter, Header.CartBadge]
[14:32:15.125] [Render] Counter (2 patches, 3ms)
```

---

## 9.5 Documentation

### Documentation Structure

```
docs.vango.dev/
├── Getting Started
│   ├── Installation
│   ├── Quick Start
│   ├── Project Structure
│   └── Your First Component
├── Core Concepts
│   ├── Components
│   ├── Signals & Reactivity
│   ├── Effects & Lifecycle
│   ├── Server-Driven Architecture
│   └── How It Works
├── Guides
│   ├── Routing
│   ├── Forms & Validation
│   ├── Data Loading
│   ├── State Management
│   ├── Styling
│   ├── Client Hooks
│   ├── JavaScript Islands
│   ├── Authentication
│   ├── Testing
│   └── Deployment
├── API Reference
│   ├── vango
│   ├── vango/el
│   ├── vango/server
│   └── vango/router
├── Examples
│   ├── Counter
│   ├── Todo App
│   ├── Real-time Chat
│   ├── Dashboard
│   └── E-commerce
├── Cookbook
│   ├── Pagination
│   ├── Infinite Scroll
│   ├── Modal Dialogs
│   ├── Drag and Drop
│   ├── File Upload
│   └── Keyboard Shortcuts
└── Error Reference
    └── E001 - E100
```

### Quick Start Guide

```markdown
# Quick Start

Get a Vango app running in under 60 seconds.

## Prerequisites

- Go 1.21 or later
- Node.js 18+ (for Tailwind CSS, optional)

## Create Your App

```bash
# Install Vango CLI
go install vango.dev/cmd/vango@latest

# Create a new project
vango create my-app

# Start development server
cd my-app
vango dev
```

Open http://localhost:3000 to see your app.

## Your First Component

Edit `app/routes/index.go`:

```go
package routes

import (
    . "vango/el"
    "vango"
)

func Page(ctx vango.Ctx) vango.Component {
    return vango.Func(func() *vango.VNode {
        count := vango.Signal(0)

        return Div(Class("p-8"),
            H1(Class("text-2xl font-bold"), Text("Hello Vango!")),
            P(Textf("Count: %d", count())),
            Button(
                Class("bg-blue-500 text-white px-4 py-2 rounded"),
                OnClick(count.Inc),
                Text("Increment"),
            ),
        )
    })
}
```

Save the file and watch your browser update instantly!

## What Just Happened?

1. **Signal**: `count := vango.Signal(0)` creates reactive state
2. **Reading**: `count()` reads the current value
3. **Click Handler**: `OnClick(count.Inc)` increments on click
4. **Auto-Update**: The UI updates automatically when `count` changes

The magic? Your component runs on the **server**. When you click:
1. Click event sent to server via WebSocket
2. Server runs the handler, updates the signal
3. Server diffs the component, sends minimal patches
4. Browser applies patches (just the text change!)

No JavaScript bundle. No state sync. No hydration bugs.

## Next Steps

- [Core Concepts](/docs/core-concepts) - Understand how Vango works
- [Routing](/docs/guides/routing) - Add more pages
- [Forms](/docs/guides/forms) - Handle user input
- [Examples](/docs/examples) - See full applications
```

### API Documentation (Generated)

```go
// generate-docs.go
// Scans source code and generates API reference

func GenerateAPIDocs() {
    pkgs := []string{
        "vango",
        "vango/el",
        "vango/server",
        "vango/router",
    }

    for _, pkg := range pkgs {
        doc := extractPackageDoc(pkg)
        markdown := renderToMarkdown(doc)
        writeFile(fmt.Sprintf("docs/api/%s.md", pkg), markdown)
    }
}

type PackageDoc struct {
    Name      string
    Overview  string
    Functions []FunctionDoc
    Types     []TypeDoc
}

type FunctionDoc struct {
    Name       string
    Signature  string
    Doc        string
    Example    string
    Parameters []ParamDoc
    Returns    []ReturnDoc
}
```

Example output:

```markdown
# vango

The core Vango framework package.

## Functions

### Signal

```go
func Signal[T any](initial T) *Signal[T]
```

Creates a reactive signal with the given initial value.

**Parameters:**
- `initial T` - The initial value of the signal

**Returns:**
- `*Signal[T]` - A pointer to the new signal

**Example:**

```go
count := vango.Signal(0)
name := vango.Signal("Alice")
items := vango.Signal([]Item{})
```

**See also:** [Signals Guide](/docs/core-concepts/signals)

---

### Memo

```go
func Memo[T any](compute func() T) *Memo[T]
```

Creates a memoized computation that updates when its dependencies change.

...
```

---

## 9.6 IDE Integration

### VS Code Extension

```json
// package.json
{
  "name": "vango-vscode",
  "displayName": "Vango",
  "description": "Vango framework support for VS Code",
  "version": "0.1.0",
  "engines": {
    "vscode": "^1.80.0"
  },
  "categories": ["Programming Languages", "Snippets"],
  "activationEvents": [
    "onLanguage:go",
    "workspaceContains:vango.json"
  ],
  "main": "./out/extension.js",
  "contributes": {
    "languages": [{
      "id": "vango",
      "extensions": [".vango"],
      "configuration": "./language-configuration.json"
    }],
    "snippets": [{
      "language": "go",
      "path": "./snippets/go.json"
    }],
    "commands": [{
      "command": "vango.createComponent",
      "title": "Vango: Create Component"
    }]
  }
}
```

### Snippets

```json
// snippets/go.json
{
  "Vango Component": {
    "prefix": "vcomp",
    "body": [
      "func ${1:ComponentName}(${2:props}) vango.Component {",
      "\treturn vango.Func(func() *vango.VNode {",
      "\t\t${3:// signals here}",
      "\t\t",
      "\t\treturn Div(",
      "\t\t\t${0}",
      "\t\t)",
      "\t})",
      "}"
    ],
    "description": "Create a Vango component"
  },
  "Vango Signal": {
    "prefix": "vsig",
    "body": "${1:name} := vango.Signal(${2:initialValue})",
    "description": "Create a Vango signal"
  },
  "Vango Effect": {
    "prefix": "veff",
    "body": [
      "vango.Effect(func() vango.Cleanup {",
      "\t${1:// effect logic}",
      "\t",
      "\treturn func() {",
      "\t\t${2:// cleanup}",
      "\t}",
      "})"
    ],
    "description": "Create a Vango effect"
  },
  "Vango Resource": {
    "prefix": "vres",
    "body": [
      "${1:data} := vango.Resource(func() (${2:Type}, error) {",
      "\treturn ${3:fetchData()}",
      "})"
    ],
    "description": "Create a Vango resource"
  }
}
```

### Go to Definition

The extension provides enhanced Go to Definition for:
- Component references
- Route files
- Signal sources

### Autocomplete

Context-aware autocomplete for:
- Element attributes
- Event handlers
- CSS classes (Tailwind integration)

---

## 9.7 Component Registry & Add Command

> **VangoUI Amendment**: Implements the "Copy-Paste Ownership" distribution model for VangoUI components.

Vango adopts a "copy-paste ownership" model for UI components. We do not distribute UI components as a compiled binary library. Instead, we distribute them as source code that developers add to their projects and own.

### 9.7.1 The `vango add` Command

**Syntax**:
```bash
vango add [command]
vango add [component_names...]
```

#### Sub-commands

##### `vango add init`

Initializes the VangoUI environment in a user's project.

**Actions**:
1. **Config Check**: Verifies `tailwind.config.js` exists
2. **Utils Generation**: Creates `app/components/ui/utils.go` containing the `CN` utility
3. **Base Generation**: Creates `app/components/ui/base.go` containing `BaseConfig`, `ConfigProvider` interface, and generic options (`Class`, `Attr`, `Child`)
4. **Version Pin**: Adds `"ui": {"version": "..."}` to `vango.json`

```bash
$ vango add init

  Initializing VangoUI...

  ✓ Created app/components/ui/utils.go
  ✓ Created app/components/ui/base.go
  ✓ Updated vango.json with ui version
  
  Ready! Add components with:
    vango add button card dialog
```

##### `vango add [component]`

Fetches and installs specific components.

**Workflow**:
1. **Fetch Manifest**: Downloads `registry.json` from configured registry
2. **Resolve Dependencies**: Performs a topological sort of dependencies
   - Example: `card` → `text` → `utils`
   - Install order: `utils`, then `text`, then `card`
3. **Download & Write**:
   - For each component:
     - Check against `vango.json` version
     - If file exists locally:
       - Calculate SHA256 of local file
       - Compare with header checksum
       - If different, prompt: `[Diff] / [Overwrite] / [Skip]`
     - Write file with metadata header
4. **Format**: Run `go fmt` on the output directory

```bash
$ vango add button dialog

  Resolving dependencies...
    button → [base, utils]
    dialog → [base, utils, focustrap, portal]

  Installing 4 components:
    ✓ utils.go (already installed)
    ✓ base.go (already installed)
    ✓ button.go
    ✓ dialog.go

  Done! Components installed to app/components/ui/
```

##### `vango add upgrade`

Upgrades installed components to latest versions.

**Logic**:
1. Read `vango.json` to find current pinned version
2. Check registry for latest version
3. For each installed component:
   - **Clean**: Local hash matches old version hash → Safe upgrade
   - **Dirty**: Local hash differs → Show diff → Prompt user

```bash
$ vango add upgrade

  Checking for updates...
    Current: v1.0.2
    Latest:  v1.1.0

  Components to update:
    button.go: clean → upgrading
    dialog.go: modified → [D]iff / [O]verwrite / [S]kip?
```

##### `vango gen component [Name]`

Scaffolds a new custom component following the VangoUI pattern.

**Generates** `[name].go` with:
- `BaseConfig` embedding
- `GetBase()` implementation
- Standard option types
- Component function skeleton

```bash
$ vango gen component Card

  Created app/components/ui/card.go

  Next steps:
    1. Define your CardConfig options
    2. Implement the Card() function
    3. Add CSS styles as needed
```

### 9.7.2 Registry Manifest Format

```json
{
  "manifestVersion": 1,
  "version": "1.0.2",
  "registry": "https://vango.dev/registry",
  "components": {
    "button": {
      "files": ["button.go"],
      "dependsOn": ["utils", "base"]
    },
    "dialog": {
      "files": ["dialog.go"],
      "dependsOn": ["utils", "base", "focustrap", "portal"]
    },
    "focustrap": {
      "files": ["focustrap.go"],
      "dependsOn": ["base"],
      "internal": true
    },
    "portal": {
      "files": ["portal.go"],
      "dependsOn": ["base"],
      "internal": true
    }
  }
}
```

### 9.7.3 Source File Headers

All distributed component files include metadata for version tracking and 3-way diffs:

```go
// Source: vango.dev/ui/button
// Version: 1.0.2
// Checksum: sha256:a1b2c3d4e5f6...

package ui

// ... component code
```

### 9.7.4 Project Configuration (`vango.json`)

```json
{
  "ui": {
    "version": "1.0.2",
    "registry": "https://vango.dev/registry.json",
    "installed": ["button", "dialog", "card"]
  }
}
```

---

## Testing

### CLI Tests

```go
func TestCreateCommand(t *testing.T) {
    tmpDir := t.TempDir()
    projectPath := filepath.Join(tmpDir, "test-app")

    err := runCreate(projectPath, "minimal")
    require.NoError(t, err)

    // Check structure
    assert.FileExists(t, filepath.Join(projectPath, "go.mod"))
    assert.FileExists(t, filepath.Join(projectPath, "main.go"))
    assert.DirExists(t, filepath.Join(projectPath, "app/routes"))

    // Check go.mod content
    goMod, _ := os.ReadFile(filepath.Join(projectPath, "go.mod"))
    assert.Contains(t, string(goMod), "module test-app")
    assert.Contains(t, string(goMod), "vango.dev/vango")
}

func TestDevServer(t *testing.T) {
    // Start dev server
    server := startDevServer(t, testProject)
    defer server.Stop()

    // Wait for ready
    waitForReady(t, server)

    // Make request
    resp, err := http.Get(server.URL)
    require.NoError(t, err)
    assert.Equal(t, 200, resp.StatusCode)
}

func TestHotReload(t *testing.T) {
    server := startDevServer(t, testProject)
    defer server.Stop()

    // Connect to reload WebSocket
    ws := connectReloadWS(t, server)

    // Modify a file
    appendToFile(t, filepath.Join(testProject, "app/routes/index.go"), "\n// comment")

    // Wait for reload message
    msg := readWSMessage(t, ws, 5*time.Second)
    assert.Equal(t, "reload", msg.Type)
}
```

### Error Message Tests

```go
func TestErrorFormatting(t *testing.T) {
    err := NewError("E001").
        WithLocation("app/routes/index.go", 15, 12).
        WithSuggestion("Wrap your component logic in vango.Func()")

    formatted := err.Format()

    assert.Contains(t, formatted, "E001")
    assert.Contains(t, formatted, "Signal read outside component context")
    assert.Contains(t, formatted, "app/routes/index.go:15:12")
    assert.Contains(t, formatted, "Hint:")
    assert.Contains(t, formatted, "vango.Func()")
}
```

---

## File Structure

```
cmd/vango/
├── main.go
├── create.go
├── dev.go
├── build.go
├── test.go
├── gen.go
├── version.go
└── internal/
    ├── dev/
    │   ├── watcher.go
    │   ├── compiler.go
    │   ├── reload.go
    │   └── tailwind.go
    ├── build/
    │   ├── builder.go
    │   ├── bundler.go
    │   └── manifest.go
    ├── templates/
    │   ├── minimal/
    │   ├── full/
    │   └── saas/
    └── devtools/
        ├── handler.go
        └── protocol.go

internal/errors/
├── error.go
├── registry.go
├── format.go
└── codes.go

docs/
├── content/          # Markdown documentation
├── generator/        # API doc generator
└── site/            # Documentation site

vscode-extension/
├── package.json
├── src/
│   └── extension.ts
└── snippets/
    └── go.json
```

---

## Exit Criteria

Phase 9 is complete when:

1. [ ] CLI: `create`, `dev`, `build`, `test`, `gen` commands working
2. [ ] Hot Reload: < 100ms rebuild, instant browser refresh
3. [ ] Error Messages: All error codes documented with suggestions
4. [ ] DevTools: Browser extension with component tree and signal inspector
5. [ ] Documentation: Getting started, guides, API reference, examples
6. [ ] IDE: VS Code extension with snippets and autocomplete
7. [ ] First-run experience tested (< 60 seconds to running app)
8. [ ] All CLI commands have help text and examples
9. [ ] Error messages tested with user studies

---

## Dependencies

- **Requires**: Phases 1-8 (all core functionality)
- **Required by**: Phase 10 (Production Hardening)

---

*Phase 9 Specification - Version 1.0*
