# Phase 14: Vango CLI Scaffold Specification

> **Status**: FINAL  
> **Version**: 1.0.1  
> **Date**: December 2024  
> **Ready for**: Engineering Implementation

---

## Table of Contents

1. [Overview](#1-overview)
2. [Locked Design Decisions](#2-locked-design-decisions)
3. [Directory Structure](#3-directory-structure)
4. [Scaffolded File Contents](#4-scaffolded-file-contents)
5. [CLI Command Reference](#5-cli-command-reference)
6. [Route Generator Logic](#6-route-generator-logic)
7. [Navigation Semantics](#7-navigation-semantics)
8. [API Route Semantics](#8-api-route-semantics)
9. [Error Handling Contract](#9-error-handling-contract)
10. [Middleware Specification](#10-middleware-specification)
11. [Package Collision Handling](#11-package-collision-handling)
12. [Configuration Schema](#12-configuration-schema)
13. [Implementation Checklist](#13-implementation-checklist)

---

## 1. Overview

This document specifies exactly what the Vango CLI should generate. All design tensions have been resolved through technical review. This spec is ready for engineering implementation.

### Design Principles

- **No magic**: Explicit over implicit; generated code is visible and debuggable
- **Go-idiomatic**: Respect Go's package system, naming conventions, and idioms
- **Scalable defaults**: Choices that work for small apps and grow to large codebases
- **Escape hatches**: Framework opinions with explicit opt-outs for advanced cases

### Critical Issues Resolved

| Issue | Problem | Resolution |
|-------|---------|------------|
| Package naming | Docs showed `package routes` in nested directories—violates Go's "one directory = one package" rule | Directory name = package name. Generated glue imports all subpackages. |
| Replace naming collision | `vango.Replace()` for navigation and `vango.Replace` as URLParam option cannot coexist | Move navigation to `ctx.Navigate().Replace()`. URLParam keeps `vango.Replace`. |
| Route registration via init() | `import _ "my-app/app/routes"` relies on init() side effects—non-deterministic, hard to test | Generated glue with explicit `routes.Register(r)` call. |
| ctx.Redirect() in WebSocket | Spec implied HTTP 302/303 responses, but WS context has no HTTP response to write | Redirect is a "hard navigation command." `.Status()` only applies in HTTP context. |

---

## 2. Locked Design Decisions

| Decision | Resolution | Rationale |
|----------|------------|-----------|
| **Package naming** | Directory name = package name | Go-idiomatic; avoids toolchain conflicts |
| **Route registration** | Generated glue via `routes.Register(*router.Router)` | Deterministic, testable, no `init()` side effects |
| **Middleware location** | `_middleware.go` (separate from `_layout.go`) | Separates request logic from UI composition |
| **Middleware signature** | `var Middleware` or `func Middleware()` (no ctx) | Avoids registration circularity |
| **Layout location** | `_layout.go` in route directories only | Colocation; no separate `layouts/` directory |
| **API location** | `app/routes/api/` | One mental model; file-based routing for everything |
| **Navigation** | `ctx.Navigate(path)` with fluent modifiers | Contextual side effect; no global state |
| **Redirect** | `ctx.Redirect(url)` as hard navigation command | `.Status()` only applies in HTTP context |
| **URL timing** | History-first (URL updates before patches) | Predictable back-button behavior |
| **API signatures** | Simple returns + `vango.Response[T]` for customization | Declarative; works with codegen |
| **Response body** | `*T` pointer (nil = no body) | Unambiguous nil semantics |
| **Error mapping** | `HTTPError` interface + typed constructors | Flexible yet convenient |
| **URLParam options** | `vango.Replace`, `vango.Push` as values | No collision with navigation |
| **Domain layer** | Not enforced | Stay minimal; `store/` and `db/` suffice |
| **CLI scaffold** | `vango create <name>` | Matches Architecture Guide |
| **CSS location** | `public/styles.css` | Single static assets directory |
| **Static serving** | `public/` is served at `/` (site root) | `/public/*` is NOT used as URL prefix |
| **Glue file** | `app/routes/routes_gen.go` | Single location, stable naming |

---

## 3. Directory Structure

The standard scaffold creates a minimal, immediately-compilable project:

```
my-app/
├── app/
│   ├── routes/                        # File-based routing
│   │   ├── routes_gen.go              # GENERATED — route registration
│   │   ├── _layout.go                 # package routes — Root UI wrapper
│   │   ├── index.go                   # package routes — Home page (/)
│   │   ├── about.go                   # package routes — About page (/about)
│   │   └── api/
│   │       └── health.go              # package api — GET /api/health
│   ├── components/                    # Shared UI components
│   │   ├── shared/                    # package shared — Cross-domain
│   │   │   ├── navbar.go
│   │   │   └── footer.go
│   │   └── ui/                        # package ui — VangoUI primitives
│   │       └── .gitkeep               # Populated by `vango add`
│   ├── store/                         # Shared state
│   │   └── .gitkeep                   # package store
│   └── middleware/                    # Middleware definitions
│       └── auth.go                    # package middleware
├── db/                                # Data access layer
│   └── .gitkeep                       # package db
├── public/                            # Static assets (served at /)
│   ├── favicon.ico
│   └── styles.css
├── main.go                            # Entry point
├── go.mod
├── go.sum
└── vango.json                         # Vango configuration
```

### Static Asset Serving Contract

**Critical**: The `public/` directory is served at the **site root** (`/`).

| File Path | URL |
|-----------|-----|
| `public/styles.css` | `/styles.css` |
| `public/favicon.ico` | `/favicon.ico` |
| `public/images/logo.png` | `/images/logo.png` |

**Note**: The URL prefix `/public/*` is NOT used. This convention matches most static site generators and keeps URLs clean.

---

## 4. Scaffolded File Contents

### 4.1 `main.go`

```go
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/vango-dev/vango"
	"my-app/app/routes"
)

func main() {
	app := vango.New(vango.Config{
		Dev: true, // Disable in production

		Session: vango.SessionConfig{
			ResumeWindow: 30 * time.Second,
			// --- Production Configuration (uncomment when ready) ---
			// Store: vango.RedisStore(redisClient),
			// MaxDetachedSessions: 10000,
			// MaxSessionsPerIP:    100,
		},

		// --- Protocol Limits (production hardening) ---
		// MaxAllocation:      4 * 1024 * 1024,  // 4MB max message
		// MaxCollectionCount: 100_000,
		// MaxVNodeDepth:      256,
	})

	// --- Observability (uncomment for production) ---
	// app.Use(middleware.OpenTelemetry())
	// app.Use(middleware.Prometheus())

	// Standard middleware
	app.Use(vango.Logger())
	app.Use(vango.Recover())

	// Register routes from generated glue
	routes.Register(app.Router())

	// Serve static files from public/ at site root /
	app.Static("/", "public")

	log.Println("🚀 Vango server starting on http://localhost:3000")
	if err := http.ListenAndServe(":3000", app.Handler()); err != nil {
		log.Fatal(err)
	}
}
```

### 4.2 `app/routes/routes_gen.go`

> **Note:** `routes_gen.go` is committed to git. The generator produces deterministic output (sorted imports, consistent formatting). Running the generator twice produces identical files.

```go
// Code generated by vango. DO NOT EDIT.

package routes

import (
	"github.com/vango-dev/vango/router"

	"my-app/app/routes/api"
)

// Register adds all routes to the router.
// Generated by `vango dev` or `vango gen routes`.
func Register(r *router.Router) {
	// Root routes (package routes)
	r.Page("/", IndexPage, Layout)
	r.Page("/about", AboutPage, Layout)

	// API routes (package api)
	r.API("GET", "/api/health", api.HealthGET)
}
```

### 4.3 `app/routes/_layout.go`

```go
package routes

import (
	"github.com/vango-dev/vango"
	. "github.com/vango-dev/vango/el"

	"my-app/app/components/shared"
)

// Layout wraps all routes in this directory and subdirectories.
func Layout(ctx vango.Ctx, children vango.Slot) *vango.VNode {
	return Html(Lang("en"),
		Head(
			Meta(Charset("utf-8")),
			Meta(Name("viewport"), Content("width=device-width, initial-scale=1")),
			Title(Text("My Vango App")),
			Link(Rel("stylesheet"), Href("/styles.css")),
			Link(Rel("icon"), Href("/favicon.ico")),
		),
		Body(
			shared.Navbar(),
			Main(Class("container"), children),
			shared.Footer(),
			vango.Scripts(), // Thin client (~12KB)
		),
	)
}
```

### 4.4 `app/routes/index.go`

```go
package routes

import (
	"github.com/vango-dev/vango"
	. "github.com/vango-dev/vango/el"
)

// IndexPage is the home page component.
func IndexPage(ctx vango.Ctx) vango.Component {
	return vango.Func(func() *vango.VNode {
		count := vango.Signal(0)

		return Div(Class("hero"),
			H1(Text("Welcome to Vango")),
			P(Text("Server-driven UI for Go")),

			Div(Class("counter"),
				Button(OnClick(count.Dec), Text("-")),
				Span(Textf(" %d ", count())),
				Button(OnClick(count.Inc), Text("+")),
			),
		)
	})
}
```

### 4.5 `app/routes/about.go`

```go
package routes

import (
	"github.com/vango-dev/vango"
	. "github.com/vango-dev/vango/el"
)

// AboutPage is the about page (stateless).
func AboutPage(ctx vango.Ctx) *vango.VNode {
	return Div(Class("about"),
		H1(Text("About")),
		P(Text("Built with Vango — the Go framework for modern web apps.")),
		Ul(
			Li(Text("Server-driven architecture")),
			Li(Text("~12KB client runtime")),
			Li(Text("Direct database access")),
			Li(Text("Type-safe components")),
		),
	)
}
```

### 4.6 `app/routes/api/health.go`

```go
package api

import "github.com/vango-dev/vango"

// HealthResponse is the JSON response for health checks.
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// HealthGET handles GET /api/health.
func HealthGET(ctx vango.Ctx) (*HealthResponse, error) {
	return &HealthResponse{
		Status:  "ok",
		Version: "1.0.0",
	}, nil
}
```

### 4.7 `app/components/shared/navbar.go`

```go
package shared

import (
	"github.com/vango-dev/vango"
	. "github.com/vango-dev/vango/el"
)

// Navbar renders the site navigation.
func Navbar() *vango.VNode {
	return Nav(Class("navbar"),
		A(Href("/"), Class("logo"), Text("MyApp")),
		Div(Class("nav-links"),
			A(Href("/"), Text("Home")),
			A(Href("/about"), Text("About")),
		),
	)
}
```

### 4.8 `app/components/shared/footer.go`

```go
package shared

import (
	"github.com/vango-dev/vango"
	. "github.com/vango-dev/vango/el"
)

// Footer renders the site footer.
func Footer() *vango.VNode {
	return Footer(Class("footer"),
		P(Text("Built with Vango")),
	)
}
```

### 4.9 `app/middleware/auth.go`

```go
package middleware

import (
	"github.com/vango-dev/vango"
	"github.com/vango-dev/vango/router"
)

// RequireAuth redirects unauthenticated users to login.
func RequireAuth(next router.Handler) router.Handler {
	return func(ctx vango.Ctx) error {
		if ctx.User() == nil {
			ctx.Redirect("/login")
			return nil
		}
		return next(ctx)
	}
}

// RequireRole checks if user has a specific role.
func RequireRole(role string) router.Middleware {
	return func(next router.Handler) router.Handler {
		return func(ctx vango.Ctx) error {
			user := ctx.User()
			if user == nil || user.Role != role {
				return vango.Forbidden("insufficient permissions")
			}
			return next(ctx)
		}
	}
}
```

### 4.10 `vango.json`

```json
{
  "name": "my-app",
  "version": "0.1.0",
  "port": 3000,

  "paths": {
    "routes": "app/routes",
    "components": "app/components",
    "ui": "app/components/ui",
    "store": "app/store",
    "middleware": "app/middleware"
  },

  "static": {
    "dir": "public",
    "prefix": "/"
  },

  "dev": {
    "watch": ["app", "db", "public"],
    "hotReload": true
  },

  "tailwind": {
    "enabled": false,
    "config": "tailwind.config.js",
    "input": "public/styles.css"
  }
}
```

### 4.11 `public/styles.css`

```css
/* Base styles — replace with Tailwind via `vango create --with-tailwind` */
:root {
  --background: #ffffff;
  --foreground: #171717;
  --primary: #2563eb;
  --muted: #f5f5f5;
  --border: #e5e5e5;
}

* {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

body {
  font-family: system-ui, -apple-system, sans-serif;
  background: var(--background);
  color: var(--foreground);
  line-height: 1.6;
}

.container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 2rem;
}

/* Navbar */
.navbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 2rem;
  border-bottom: 1px solid var(--border);
}

.navbar .logo {
  font-weight: 700;
  font-size: 1.25rem;
  text-decoration: none;
  color: inherit;
}

.nav-links {
  display: flex;
  gap: 1.5rem;
}

.nav-links a {
  text-decoration: none;
  color: inherit;
}

.nav-links a:hover {
  color: var(--primary);
}

/* Hero */
.hero {
  text-align: center;
  padding: 4rem 0;
}

.hero h1 {
  font-size: 3rem;
  margin-bottom: 0.5rem;
}

.hero p {
  color: #666;
  margin-bottom: 2rem;
}

/* Counter */
.counter {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
}

.counter button {
  width: 2.5rem;
  height: 2.5rem;
  font-size: 1.25rem;
  border: 1px solid var(--border);
  border-radius: 0.375rem;
  background: var(--background);
  cursor: pointer;
}

.counter button:hover {
  background: var(--muted);
}

.counter span {
  min-width: 3rem;
  text-align: center;
  font-size: 1.5rem;
  font-weight: 600;
}

/* Footer */
.footer {
  text-align: center;
  padding: 2rem;
  color: #666;
  font-size: 0.875rem;
}

/* About */
.about h1 {
  margin-bottom: 1rem;
}

.about ul {
  margin-top: 1rem;
  margin-left: 1.5rem;
}

.about li {
  margin-bottom: 0.5rem;
}

/* Admin Layout (for --with-auth scaffold) */
.admin-layout {
  display: flex;
  min-height: calc(100vh - 120px);
}

.admin-sidebar {
  width: 240px;
  background: var(--muted);
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.admin-sidebar a {
  padding: 0.5rem 1rem;
  text-decoration: none;
  color: inherit;
  border-radius: 0.375rem;
}

.admin-sidebar a:hover {
  background: var(--border);
}

.admin-content {
  flex: 1;
  padding: 1rem 2rem;
}

/* Connection state indicators (Phase 12 reconnection UX) */
.vango-reconnecting::after {
  content: "Reconnecting...";
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  padding: 0.5rem;
  background: #fbbf24;
  text-align: center;
  font-size: 0.875rem;
  z-index: 9999;
}

.vango-offline::after {
  content: "Connection lost";
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  padding: 0.5rem;
  background: #ef4444;
  color: white;
  text-align: center;
  font-size: 0.875rem;
  z-index: 9999;
}
```

### 4.12 `.gitignore`

```gitignore
# Binaries
/my-app
/dist/
*.exe
*.dll
*.so
*.dylib

# Dependencies
/vendor/

# IDE
.idea/
.vscode/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db

# Test
coverage.out
*.cover

# Vango
.vango/
```

### 4.13 `README.md`

```markdown
# my-app

A web application built with [Vango](https://vango.dev).

## Quick Start

\`\`\`bash
# Development server with hot reload
vango dev

# Open http://localhost:3000
\`\`\`

## Commands

\`\`\`bash
vango dev                    # Start dev server
vango build                  # Production build
vango test                   # Run tests

vango gen route users/[id]   # Generate route
vango gen component Card     # Generate component
vango gen api products       # Generate API route

vango add button             # Add VangoUI component
\`\`\`

## Project Structure

\`\`\`
app/
├── routes/         # File-based routing (pages, layouts, middleware)
├── components/     # Shared UI components
├── store/          # Shared state (SharedSignal, GlobalSignal)
└── middleware/     # Middleware definitions
db/                 # Database layer
public/             # Static assets (served at /)
\`\`\`

## Production Checklist

Before deploying to production:

1. Set \`Dev: false\` in \`main.go\`
2. Configure \`SessionStore\` (Redis recommended)
3. Set session limits (\`MaxDetachedSessions\`, \`MaxSessionsPerIP\`)
4. Enable observability (\`middleware.OpenTelemetry()\`)
5. Configure CSRF secret

See the [Production Guide](https://docs.vango.dev/production) for details.

## Learn More

- [Documentation](https://docs.vango.dev)
- [API Reference](https://pkg.go.dev/github.com/vango-dev/vango)
- [Examples](https://github.com/vango-dev/examples)
```

---

## 5. CLI Command Reference

### 5.1 Project Scaffolding

```bash
vango create <name>                    # Standard scaffold
vango create <name> --minimal          # Minimal (home page only)
vango create <name> --with-tailwind    # Include Tailwind CSS
vango create <name> --with-db=sqlite   # Include SQLite setup
vango create <name> --with-db=postgres # Include PostgreSQL setup
vango create <name> --with-auth        # Include admin routes with auth
vango create <name> --full             # All features
```

### 5.2 Generators

```bash
# Routes
vango gen route <path>                 # Generate page route
vango gen route projects/[id]          # → app/routes/projects/[id].go
vango gen route admin/users/[id]/edit  # → app/routes/admin/users/[id]/edit.go

# API Routes
vango gen api <path>                   # Generate API route
vango gen api users                    # → app/routes/api/users.go
vango gen api admin/reports            # → app/routes/api/admin/reports.go

# Components
vango gen component <path>             # Generate component
vango gen component Card               # → app/components/card.go
vango gen component marketing/Hero     # → app/components/marketing/hero.go

# Store
vango gen store <name>                 # Generate store file
vango gen store cart                   # → app/store/cart.go

# Middleware
vango gen middleware <name>            # Generate middleware
vango gen middleware ratelimit         # → app/middleware/ratelimit.go

# Route glue (usually auto-run by dev)
vango gen routes                       # Regenerate routes_gen.go
```

### 5.3 VangoUI

```bash
vango add init                         # Initialize VangoUI config
vango add init --path=lib/ui           # Set custom UI path

vango add <component>                  # Download to configured path
vango add button                       # → app/components/ui/button.go
vango add card dialog toast            # Multiple at once
vango add button --path=design/atoms   # Override path (updates config)

vango add --list                       # List available components
vango add --check                      # Check for updates
```

### 5.4 Development

```bash
vango dev                              # Start dev server + hot reload
vango dev --port=8080                  # Custom port

vango build                            # Production build
vango build --output=./bin             # Custom output

vango test                             # Run tests
vango test --coverage                  # With coverage report
```

### 5.5 Route Regeneration Behavior

`vango dev` regenerates `routes_gen.go` when **any `.go` file** in `app/routes/` changes (create, delete, rename, or modify).

However, the regeneration is **smart**:
1. On any change, re-parse the AST of all route files
2. Compute the new route manifest (routes, handlers, middleware)
3. Compare with the current `routes_gen.go` content
4. **Only rewrite if the manifest changed** (deterministic output means minimal churn)

This means:
- Renaming `IndexPage` to `HomePage` triggers regeneration (symbol changed)
- Editing the body of `IndexPage` does NOT regenerate (symbol unchanged)
- Adding/removing files always regenerates

**Build Failure Recovery**: If a developer renames a handler and the build fails with "undefined" errors, `vango dev` automatically repairs `routes_gen.go` with the new symbol names.

### 5.6 Scaffold Variants

| Command | Contents |
|---------|----------|
| `vango create my-app` | Minimal: index, about, health API. Compiles immediately. |
| `vango create my-app --with-db=sqlite` | Adds: db/ with SQLite setup, projects routes, users API |
| `vango create my-app --with-db=postgres` | Adds: db/ with PostgreSQL setup, projects routes, users API |
| `vango create my-app --with-auth` | Adds: admin routes with middleware, login page |
| `vango create my-app --full` | All of the above |

**Standard scaffold goal:** `vango create && vango dev` works in <30 seconds with zero configuration.

---

## 6. Route Generator Logic

### 6.1 File to Route Mapping

| File Path | Package | Route | Handler Symbol |
|-----------|---------|-------|----------------|
| `routes/index.go` | `routes` | `/` | `IndexPage` |
| `routes/about.go` | `routes` | `/about` | `AboutPage` |
| `routes/admin/index.go` | `admin` | `/admin` | `IndexPage` |
| `routes/admin/users.go` | `admin` | `/admin/users` | `UsersPage` |
| `routes/projects/[id].go` | `projects` | `/projects/:id` | `ShowPage` |
| `routes/api/users.go` | `api` | `/api/users` | `UsersGET`, `UsersPOST`, etc. |

### 6.2 Param Type Inference

| Filename Pattern | Generated Type | Rationale |
|------------------|----------------|-----------|
| `[id].go` | `int` | Convention for numeric IDs |
| `[slug].go` | `string` | Default for named params |
| `[uuid].go` | `string` | UUIDs are strings |
| `[...path].go` | `[]string` | Catch-all route |

### 6.3 Generated Params Struct

```go
// For routes/projects/[id].go
type Params struct {
    ID int `param:"id"`
}

// For routes/blog/[year]/[month]/[slug].go
type Params struct {
    Year  int    `param:"year"`
    Month int    `param:"month"`
    Slug  string `param:"slug"`
}

// For routes/files/[...path].go
type Params struct {
    Path []string `param:"path"`
}
```

### 6.4 Special Files

| File | Purpose | Export Symbol |
|------|---------|---------------|
| `_layout.go` | UI wrapper for directory | `func Layout(ctx, children) *VNode` |
| `_middleware.go` | Request guards for directory | `var Middleware` or `func Middleware()` |
| `index.go` | Index route for directory | `func IndexPage(ctx) ...` |
| `about.go` | About page | `func AboutPage(ctx) ...` |
| `new.go` | New/create page | `func NewPage(ctx) ...` |
| `[id].go` | Detail/show page | `func ShowPage(ctx, p Params) ...` |
| `[id]/edit.go` | Edit page | `func EditPage(ctx, p Params) ...` |

### 6.5 Handler Symbol Naming Convention

The route generator produces deterministic handler symbols based on filenames:

| Filename | Generated Symbol | Pattern |
|----------|------------------|---------|
| `index.go` | `IndexPage` | "Index" + "Page" |
| `about.go` | `AboutPage` | PascalCase(filename) + "Page" |
| `new.go` | `NewPage` | PascalCase(filename) + "Page" |
| `[id].go` | `ShowPage` | "Show" + "Page" (detail view) |
| `[slug].go` | `ShowPage` | "Show" + "Page" (detail view) |
| `edit.go` | `EditPage` | PascalCase(filename) + "Page" |
| `settings.go` | `SettingsPage` | PascalCase(filename) + "Page" |

**API routes follow a different pattern:**

| Filename | HTTP Method | Generated Symbol |
|----------|-------------|------------------|
| `users.go` | GET | `UsersGET` |
| `users.go` | POST | `UsersPOST` |
| `users.go` | PUT | `UsersPUT` |
| `users.go` | DELETE | `UsersDELETE` |
| `health.go` | GET | `HealthGET` |

**Rules:**
1. Generator always uses file-based symbol names
2. If the expected symbol is not exported, generator fails with clear error
3. No inference or fallback — explicit exports required

---

## 7. Navigation Semantics

### 7.1 `ctx.Navigate()` — Soft Navigation

**Purpose:** SPA-like navigation within the app. Uses WebSocket patches; no full page reload.

**API:**

```go
ctx.Navigate("/dashboard")                    // Basic navigation
ctx.Navigate("/login").Replace()              // No history entry
ctx.Navigate("/search").Query("q", term)      // Add query param
ctx.Navigate("/search").Query("q", term).Replace()  // Combined
```

**Execution Sequence (History-First):**

1. Server matches new route
2. Server renders new component tree
3. Server computes diff against current tree
4. Server sends: `[URL_UPDATE, patches...]`
5. **Client updates URL** via `history.pushState()` (or `replaceState()`)
6. Client applies DOM patches
7. Old component state torn down; new route mounts

**`.Query()` Behavior:**

- Merges with existing query string
- Overwrites keys that are provided
- Preserves keys that are not mentioned

**Failure Recovery:**

If patch application fails (DOM shape mismatch, client error, hook exception), the client performs a **hard reload to the new URL**. This self-healing behavior prevents "URL says X, DOM shows Y" split-brain states.

The client logs the failure to console in dev mode for debugging.

### 7.2 `ctx.Redirect()` — Hard Navigation

**Purpose:** Full page navigation. Breaks WebSocket; causes full reload.

**API:**

```go
ctx.Redirect("/login")                        // Hard navigation (302 in HTTP)
ctx.Redirect("/new-url").Status(301)          // Permanent redirect
ctx.Redirect("https://external.com")          // External URL
```

**Transport-Dependent Behavior:**

| Context | Mechanism | `.Status()` Effect |
|---------|-----------|-------------------|
| **WebSocket** | Sends `HARD_NAV` command → client executes `window.location.href = url` | Ignored |
| **HTTP (SSR)** | HTTP 302 response (or specified status) | Applied |
| **HTTP (API)** | HTTP 302 response (or specified status) | Applied |

**Documentation Note:** "In WebSocket context, `.Status()` is ignored. The client performs a full page navigation regardless of status code."

**API Route Restriction:**

In API routes (under `app/routes/api/`), use `vango.Response[T]` for redirects, not `ctx.Redirect()`:

```go
// Correct: Use Response for API redirects
func UsersPOST(ctx vango.Ctx, input CreateInput) (vango.Response[any], error) {
    resource := createResource(input)
    return vango.SeeOther("/api/resources/" + resource.ID), nil
}

// Incorrect: Don't use ctx.Redirect in API routes
func UsersPOST(ctx vango.Ctx, input CreateInput) (vango.Response[any], error) {
    ctx.Redirect("/api/resources/" + resource.ID)  // DON'T DO THIS
    return vango.Response[any]{}, nil
}
```

`ctx.Redirect()` is reserved for **page routes and SSR contexts** where full-page navigation is intended.

### 7.3 URL Update Protocol Distinction

**Important**: Navigation and URLParam use **distinct protocol mechanisms** to avoid semantic confusion.

| Mechanism | Protocol | Purpose | URL Change |
|-----------|----------|---------|------------|
| **Navigation** | `ControlNav` envelope | Change route (full component swap) | Entire path + query |
| **URLParam** | `PatchURLPush` / `PatchURLReplace` | Update query params on current page | Query params only |

**Navigation Envelope (`ControlNav`):**

```
ControlNav {
    TargetURL:  string          // Full URL path
    Mode:       push | replace  // History mode
    Patches:    []Patch         // Component diff
}
```

**URLParam Patches (from Phase 12):**

```
PatchURLPush    = 0x30  // Update query, push history
PatchURLReplace = 0x31  // Update query, replace history

// Only contains query param changes, not full URL
URLPatch {
    Mode:   push | replace
    Params: map[string]string  // Query params to set/clear
}
```

**Why Separate?**

1. **Semantic clarity**: Navigation = route change; URLParam = state change on current route
2. **Patching scope**: Navigation includes component patches; URLParam is URL-only
3. **Implementation simplicity**: URLParam patches are small and frequent (search filters); Navigation is rare and large

---

## 8. API Route Semantics

### 8.1 Handler Signatures

```go
// Simple GET — returns 200 with JSON body
func GET(ctx vango.Ctx) (*User, error)
func GET(ctx vango.Ctx) ([]*User, error)

// With path params
func GET(ctx vango.Ctx, p Params) (*User, error)

// With request body
func POST(ctx vango.Ctx, input CreateUserInput) (*User, error)

// Custom response (status, headers)
func POST(ctx vango.Ctx, input CreateUserInput) (vango.Response[User], error)

// No content response
func DELETE(ctx vango.Ctx, p Params) error
func DELETE(ctx vango.Ctx, p Params) (vango.Response[any], error)
```

### 8.2 Response Type

```go
package vango

// Response allows custom status codes and headers.
type Response[T any] struct {
    Status  int               // HTTP status (default: 200 if Body != nil, 204 if nil)
    Headers map[string]string // Additional response headers
    Body    *T                // Response body (nil = no body)
}
```

### 8.3 Status Code Defaults

| Condition | Default Status |
|-----------|----------------|
| `Body != nil`, `Status == 0` | `200 OK` |
| `Body == nil`, `Status == 0` | `204 No Content` |
| `Status` specified | Use specified status |

### 8.4 Invalid Combinations (Enforced)

- `Status == 204` with `Body != nil` → Error
- `Status == 304` with `Body != nil` → Error

### 8.5 Convenience Constructors

Constructors accept value and take address internally for ergonomics:

```go
// 200 OK with body
func OK[T any](body T) Response[T] {
    return Response[T]{Status: 200, Body: &body}
}

// 201 Created with Location header
func Created[T any](body T, location string) Response[T] {
    return Response[T]{
        Status:  201,
        Headers: map[string]string{"Location": location},
        Body:    &body,
    }
}

// 204 No Content
func NoContent() Response[any] {
    return Response[any]{Status: 204, Body: nil}
}

// 202 Accepted
func Accepted[T any](body T) Response[T] {
    return Response[T]{Status: 202, Body: &body}
}

// 303 See Other (for POST-redirect-GET in APIs)
func SeeOther(location string) Response[any] {
    return Response[any]{
        Status:  303,
        Headers: map[string]string{"Location": location},
        Body:    nil,
    }
}
```

### 8.6 Escape Hatch: Raw Handler

```go
// For webhooks, legacy handlers, or maximum flexibility
r.RawAPI("/api/webhook/stripe", stripeWebhookHandler)  // http.Handler
```

---

## 9. Error Handling Contract

### 9.1 HTTPError Interface

```go
package vango

// HTTPError is implemented by errors that map to HTTP status codes.
type HTTPError interface {
    error
    StatusCode() int
    PublicMessage() string  // Safe to show to clients
}
```

### 9.2 Typed Error Constructors

```go
// 400 Bad Request
func BadRequest(msg string) error
func BadRequestf(format string, args ...any) error

// 401 Unauthorized
func Unauthorized(msg string) error

// 403 Forbidden
func Forbidden(msg string) error

// 404 Not Found
func NotFound(msg string) error

// 409 Conflict
func Conflict(msg string) error

// 422 Unprocessable Entity (validation errors)
func ValidationError(fields map[string]string) error

// 429 Too Many Requests
func TooManyRequests(msg string) error

// 500 Internal Server Error
func InternalError(msg string) error
```

### 9.3 Error Mapping Precedence

1. **`HTTPError` interface** → Use `StatusCode()` + `PublicMessage()`
2. **`ValidationError`** → 422 with structured field errors
3. **Any other `error`** → 500 Internal Server Error
   - In production: generic message ("internal server error")
   - In dev mode: actual error message

### 9.4 Validation Error Response Format

```json
{
  "error": "validation_error",
  "message": "Validation failed",
  "fields": {
    "email": "must be a valid email address",
    "name": "is required"
  }
}
```

---

## 10. Middleware Specification

### 10.1 Middleware Type

```go
package router

type Handler func(ctx vango.Ctx) error
type Middleware func(next Handler) Handler
```

### 10.2 Export Options (in `_middleware.go`)

**Option A: Variable (recommended)**

```go
package admin

var Middleware = []router.Middleware{
    middleware.RequireAuth,
    middleware.RequireRole("admin"),
}
```

**Option B: Function (for programmatic composition)**

```go
package admin

func Middleware() []router.Middleware {
    mw := []router.Middleware{
        middleware.RequireAuth,
    }
    
    if config.StrictMode {
        mw = append(mw, middleware.StrictCSP)
    }
    
    return mw
}
```

### 10.3 Per-Request Logic

Per-request decisions happen **inside** middleware functions, not in the Middleware slice construction:

```go
func RequireTenant(next router.Handler) router.Handler {
    return func(ctx vango.Ctx) error {
        tenant := ctx.Param("tenant")
        
        // Per-request: check tenant-specific rate limits
        limits := tenantService.GetRateLimits(tenant)
        if !limits.Allow(ctx.IP()) {
            return vango.TooManyRequests("rate limit exceeded")
        }
        
        return next(ctx)
    }
}
```

### 10.4 Middleware Inheritance

Middleware in a directory applies to all routes in that directory **and its subdirectories**.

```
routes/
├── _middleware.go      # Applies to /, /about, /admin/*, /projects/*
├── admin/
│   ├── _middleware.go  # Applies to /admin/* (stacks with parent)
│   └── index.go
```

**Execution order:** Parent middleware runs before child middleware.

---

## 11. Package Collision Handling

### 11.1 The Collision

- `app/routes/admin/` → `package admin`
- `app/components/admin/` → `package admin`

Both are valid Go packages with the same name.

### 11.2 Solution: Import Aliasing

```go
// app/routes/admin/index.go
package admin

import (
    "github.com/vango-dev/vango"
    . "github.com/vango-dev/vango/el"

    // Alias the components package
    adminui "my-app/app/components/admin"
)

func IndexPage(ctx vango.Ctx) vango.Component {
    return vango.Func(func() *vango.VNode {
        return Div(
            H1(Text("Admin Dashboard")),
            adminui.DataTable(users),  // Use aliased import
            adminui.Sidebar(),
        )
    })
}
```

### 11.3 CLI Guidance

When `vango gen` detects a potential collision, it should output:

```
Note: 'admin' package exists in both routes and components.
Use import aliasing in routes/admin/*.go:

    import adminui "my-app/app/components/admin"
```

---

## 12. Configuration Schema

### 12.1 `vango.json` Full Schema

```json
{
  "name": "my-app",
  "version": "0.1.0",
  "port": 3000,

  "paths": {
    "routes": "app/routes",
    "components": "app/components",
    "ui": "app/components/ui",
    "store": "app/store",
    "middleware": "app/middleware"
  },

  "static": {
    "dir": "public",
    "prefix": "/"
  },

  "dev": {
    "watch": ["app", "db", "public"],
    "hotReload": true,
    "openBrowser": false
  },

  "build": {
    "output": "./dist",
    "minifyAssets": true,
    "stripSymbols": true
  },

  "tailwind": {
    "enabled": false,
    "config": "tailwind.config.js",
    "input": "public/styles.css",
    "output": "public/styles.css"
  },

  "session": {
    "resumeWindow": "30s"
  }
}
```

**Build options:**
- `minifyAssets`: Minify CSS, JS hooks bundle
- `stripSymbols`: Go binary uses `-ldflags="-s -w"` (smaller binary)

### 12.2 Path Configuration

All paths are relative to project root. The CLI respects these when generating files:

```bash
vango gen component Card
# Reads paths.components from vango.json
# Creates: {paths.components}/card.go

vango add button
# Reads paths.ui from vango.json
# Creates: {paths.ui}/button.go
```

---

## 13. Implementation Checklist

### Phase 14.1: Core CLI

- [ ] `vango create <name>` — Full scaffold generation
- [ ] `vango create --minimal` — Minimal scaffold
- [ ] `vango dev` — Dev server with hot reload
- [ ] `vango build` — Production build
- [ ] File watcher integration with smart route regeneration

### Phase 14.2: Route Generator

- [ ] `vango gen route <path>` — Route with param inference
- [ ] `vango gen api <path>` — API route generation
- [ ] `vango gen routes` — Glue file generation with AST parsing
- [ ] Support for `_layout.go` detection
- [ ] Support for `_middleware.go` detection
- [ ] Params struct generation from `[param]` patterns
- [ ] Build error recovery (symbol rename detection)

### Phase 14.3: Component Generator

- [ ] `vango gen component <path>` — Component generation
- [ ] `vango gen store <name>` — Store generation
- [ ] `vango gen middleware <name>` — Middleware generation
- [ ] Package collision detection and warning

### Phase 14.4: VangoUI Integration

- [ ] `vango add init` — Initialize UI config
- [ ] `vango add <component>` — Download from registry
- [ ] Component dependency resolution
- [ ] `vango.json` path persistence
- [ ] `--path` override flag

### Phase 14.5: Enhancements

- [ ] `vango create --with-tailwind`
- [ ] `vango create --with-db=*`
- [ ] `vango create --with-auth`
- [ ] `vango create --full`
- [ ] `vango test`
- [ ] OpenAPI generation from typed API routes
- [ ] VS Code extension

---

## Appendix A: Framework Types Reference

```go
package vango

// --- Core Types ---

type Ctx interface {
    // User & Auth
    User() *User
    SetUser(u *User)
    
    // Navigation
    Navigate(path string) *NavigateBuilder
    Redirect(url string) *RedirectBuilder
    
    // Request
    Param(name string) string
    Query(name string) string
    Header(name string) string
    
    // Response (API routes)
    SetHeader(name, value string)
    
    // Session
    Session() *Session
    
    // Standard library context (for DB drivers, OTel)
    StdContext() context.Context
}

type NavigateBuilder struct{}
func (n *NavigateBuilder) Replace() *NavigateBuilder
func (n *NavigateBuilder) Query(key, value string) *NavigateBuilder
func (n *NavigateBuilder) State(data any) *NavigateBuilder

type RedirectBuilder struct{}
func (r *RedirectBuilder) Status(code int) *RedirectBuilder

// --- Response Types ---

type Response[T any] struct {
    Status  int
    Headers map[string]string
    Body    *T
}

func OK[T any](body T) Response[T]
func Created[T any](body T, location string) Response[T]
func NoContent() Response[any]
func Accepted[T any](body T) Response[T]
func SeeOther(location string) Response[any]

// --- Error Types ---

type HTTPError interface {
    error
    StatusCode() int
    PublicMessage() string
}

func BadRequest(msg string) error
func Unauthorized(msg string) error
func Forbidden(msg string) error
func NotFound(msg string) error
func Conflict(msg string) error
func ValidationError(fields map[string]string) error
func TooManyRequests(msg string) error
func InternalError(msg string) error

// --- URLParam Options (Phase 12) ---

var (
    Push    URLParamOption  // Default: creates history entry
    Replace URLParamOption  // No history entry
)
```

---

## Appendix B: Extended Scaffold Files (--with-db, --with-auth)

The following files are included when using `--with-db` or `--with-auth` flags.

### B.1 `app/routes/admin/_middleware.go` (--with-auth)

```go
package admin

import (
	"github.com/vango-dev/vango/router"

	"my-app/app/middleware"
)

// Middleware applies to all routes in /admin/*.
var Middleware = []router.Middleware{
	middleware.RequireAuth,
}
```

### B.2 `app/routes/admin/_layout.go` (--with-auth)

```go
package admin

import (
	"github.com/vango-dev/vango"
	. "github.com/vango-dev/vango/el"
)

// Layout wraps all admin routes.
func Layout(ctx vango.Ctx, children vango.Slot) *vango.VNode {
	return Div(Class("admin-layout"),
		Nav(Class("admin-sidebar"),
			A(Href("/admin"), Text("Dashboard")),
			A(Href("/admin/users"), Text("Users")),
			A(Href("/admin/settings"), Text("Settings")),
		),
		Div(Class("admin-content"),
			children,
		),
	)
}
```

### B.3 `app/routes/admin/index.go` (--with-auth)

```go
package admin

import (
	"github.com/vango-dev/vango"
	. "github.com/vango-dev/vango/el"
)

// IndexPage is the admin dashboard.
func IndexPage(ctx vango.Ctx) *vango.VNode {
	user := ctx.User()
	
	return Div(
		H1(Text("Admin Dashboard")),
		P(Textf("Welcome, %s", user.Name)),
	)
}
```

### B.4 `app/routes/projects/[id].go` (--with-db)

```go
package projects

import (
	"github.com/vango-dev/vango"
	. "github.com/vango-dev/vango/el"

	"my-app/db"
)

// Params is auto-generated from filename [id].go.
type Params struct {
	ID int `param:"id"`
}

// ShowPage handles /projects/:id.
func ShowPage(ctx vango.Ctx, p Params) vango.Component {
	return vango.Func(func() *vango.VNode {
		project := vango.Signal[*db.Project](nil)

		vango.Effect(func() vango.Cleanup {
			proj, _ := db.Projects.Find(p.ID)
			project.Set(proj)
			return nil
		})

		if project() == nil {
			return Div(Text("Loading..."))
		}

		return Div(
			H1(Text(project().Name)),
			P(Text(project().Description)),
		)
	})
}
```

### B.5 `app/routes/api/users.go` (--with-db)

```go
package api

import (
	"github.com/vango-dev/vango"

	"my-app/db"
)

// --- Request/Response Types ---

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type CreateUserInput struct {
	Email string `json:"email" validate:"required,email"`
	Name  string `json:"name" validate:"required"`
}

// --- Handlers ---

// UsersGET handles GET /api/users.
func UsersGET(ctx vango.Ctx) ([]*User, error) {
	return db.Users.All()
}

// UsersPOST handles POST /api/users.
// Returns 201 Created with Location header.
func UsersPOST(ctx vango.Ctx, input CreateUserInput) (vango.Response[User], error) {
	user, err := db.Users.Create(input)
	if err != nil {
		return vango.Response[User]{}, err
	}

	return vango.Response[User]{
		Status: 201,
		Headers: map[string]string{
			"Location": "/api/users/" + user.ID,
		},
		Body: &user,
	}, nil
}
```

### B.6 Extended `routes_gen.go` (--with-db --with-auth)

```go
// Code generated by vango. DO NOT EDIT.

package routes

import (
	"github.com/vango-dev/vango/router"

	"my-app/app/routes/admin"
	"my-app/app/routes/api"
	"my-app/app/routes/projects"
)

// Register adds all routes to the router.
// Generated by `vango dev` or `vango gen routes`.
func Register(r *router.Router) {
	// Root routes (package routes)
	r.Page("/", IndexPage, Layout)
	r.Page("/about", AboutPage, Layout)

	// Admin routes (package admin)
	r.Page("/admin", admin.IndexPage, admin.Layout, admin.Middleware...)

	// Project routes (package projects)
	r.Page("/projects", projects.IndexPage, Layout)
	r.Page("/projects/:id", projects.ShowPage, Layout)

	// API routes (package api)
	r.API("GET", "/api/health", api.HealthGET)
	r.API("GET", "/api/users", api.UsersGET)
	r.API("POST", "/api/users", api.UsersPOST)
}
```

---

## Appendix C: Cross-References to V2.1 Production Features

This scaffold connects to the production hardening features defined in other phases:

| Feature | Phase | Scaffold Integration |
|---------|-------|---------------------|
| SessionStore | Phase 12 | Commented stub in `main.go` |
| MaxDetachedSessions | Phase 12 | Commented stub in `main.go` |
| MaxSessionsPerIP | Phase 12 | Commented stub in `main.go` |
| Protocol limits | Phase 13 | Commented stub in `main.go` |
| OpenTelemetry | Phase 13 | Commented stub in `main.go` |
| Prometheus | Phase 13 | Commented stub in `main.go` |
| ctx.StdContext() | Phase 13 | Referenced in Ctx interface |
| Reconnection CSS | Phase 12 | Included in `public/styles.css` |

### SessionStore Type Mapping

**Important**: `SessionConfig.Store` is **exactly** the `session.SessionStore` interface defined in Phase 12. There is no adapter layer.

```go
// pkg/vango/config.go
import "github.com/vango-dev/vango/session"

type SessionConfig struct {
    Store session.SessionStore  // Direct interface from Phase 12
    // ...
}

// Usage with built-in implementations
app := vango.New(vango.Config{
    Session: vango.SessionConfig{
        Store: session.NewRedisStore(redisClient),   // Implements session.SessionStore
        // OR
        Store: session.NewSQLStore(db),              // Implements session.SessionStore  
        // OR
        Store: session.NewMemoryStore(),             // Default, implements session.SessionStore
    },
})
```

### Connection State CSS Classes (Phase 12 Alignment)

The scaffold CSS uses these class names, which match Phase 12 exactly:

| Class | Meaning | Applied When |
|-------|---------|--------------|
| `.vango-connected` | WebSocket open and healthy | Normal operation |
| `.vango-reconnecting` | Actively attempting to reconnect | After disconnect, during backoff |
| `.vango-offline` | Gave up reconnecting | After `maxRetries` exhausted |

### Default Configuration Values

These defaults are consistent across Phase 12 and the scaffold:

| Setting | Default Value | Phase |
|---------|---------------|-------|
| `MaxDetachedSessions` | 10,000 | Phase 12 |
| `MaxSessionsPerIP` | 100 | Phase 12 |
| `ResumeWindow` | 30 seconds | Phase 12 |
| `MaxRetries` (reconnect) | 10 | Phase 12 |
| `BaseDelay` (reconnect) | 1000ms | Phase 12 |

Users following the golden path will see these stubs and know where to enable production features.

---

*End of Specification*
