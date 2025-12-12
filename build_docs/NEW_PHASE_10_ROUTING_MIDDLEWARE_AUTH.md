# Vango V2 Architecture Definition

**Focus Area: Routing, Middleware, and Authentication**


-----

## 1\. Executive Summary

Vango V2 will adopt a **"Dual-Layer, Standard-Compliant"** architecture.

  * **Routing:** We retain the custom high-performance Radix router (69ns match time) for internal dispatch but expose standard `net/http` handlers for external integration.
  * **Middleware:** We separate the stack into **HTTP Middleware** (connection initialization) and **Vango Middleware** (event loop processing).
  * **Auth:** We provide a generic "Context Bridge" to hydrate WebSocket sessions from standard HTTP authentication contexts, ensuring compatibility with the wider Go ecosystem (Chi, Gorilla, Goth, etc.).

**Core Philosophy:** *Vango owns the rendering loop; the developer owns the infrastructure.*

-----

## 2\. The "Two-Layer" Middleware Architecture

Vango operates across two distinct lifecycles. Attempting to unify them into a single middleware stack is an anti-pattern that leads to performance degradation and context loss.

### Layer 1: The HTTP Stack (Infrastructure)

  * **Lifecycle:** Runs ONCE per user session (during the initial GET request and WebSocket Upgrade).
  * **Standard:** `func(http.Handler) http.Handler`
  * **Responsibility:** Authentication, Logging, CORS, Panic Recovery, Session Cookie validation.
  * **Ecosystem:** Compatible with `chi/middleware`, `gorilla/handlers`, `rs/cors`.

### Layer 2: The Vango Event Stack (Application)

  * **Lifecycle:** Runs HUNDREDS of times per session (on every Click, Input, Drag).
  * **Standard:** `func(vango.Ctx, vango.Next)`
  * **Responsibility:** Authorization guards (RBAC), Event validation, Application-level logging.
  * **Why Separate?** Running a full HTTP middleware chain on a 50-nanosecond binary event patch is performance suicide. The Vango stack is zero-allocation and type-safe.

-----

## 3\. The Context Bridge (Critical Mechanism)

The "Context Bridge" is the mechanism that allows Vango to be agnostic about how a user is authenticated. It solves the "Dead Context" problem where data attached to the HTTP request is lost once the connection upgrades to a WebSocket.

### 3.1 The Configuration Hook

We introduce `OnSessionStart` in the Vango configuration. This hook runs **synchronously** during the upgrade handshake.

```go
// vango/core.go

type Config struct {
    // OnSessionStart runs during the WebSocket upgrade.
    // Use this to copy data from the dying HTTP context to the new Vango Session.
    OnSessionStart func(httpCtx context.Context, session *Session)
}
```

### 3.2 Implementation Strategy

Vango does not parse JWTs or check cookies. It trusts that the **Layer 1 HTTP Middleware** has already done this and populated the `http.Request.Context`.

**The Handshake Flow:**

1.  **HTTP Req:** `GET /_vango/ws` arrives.
2.  **Layer 1 Middleware:** Validates Auth Cookie, sets `user` object in `r.Context()`.
3.  **Vango Handler:** Accepts request.
4.  **The Bridge:** Vango calls `config.OnSessionStart(r.Context(), newSession)`.
5.  **Data Copy:** Developer code copies `user` from Context → Session.
6.  **Upgrade:** Connection becomes WebSocket. `r.Context()` is canceled.
7.  **Event Loop:** Vango Session persists with the copied User data.

-----

## 4\. Routing Strategy

We reject the proposal to use **Fiber** (due to `fasthttp` incompatibility) or **Chi** (due to performance overhead on the hot path).

### 4.1 Internal Router (The Engine)

We keep the custom Radix Tree implementation detailed in the Benchmark Report.

  * **Performance:** \~69ns matching.
  * **Features:** Optimized for matching `HID` (Handler IDs) and WebSocket event types.
  * **Isolation:** This router is internal and not exposed directly for HTTP definitions.

### 4.2 External Router (The Interface)

Vango exposes itself as a standard `http.Handler`. This allows the developer to mount Vango inside *any* Go router.

```go
// The developer's main.go
r := chi.NewRouter()
r.Use(middleware.Logger)

// Mount Vango
app := vango.New(...)
r.Handle("/*", app.Handler()) // Vango takes over matching for page routes
```

### 4.3 Handling "Page vs. API"

  * **Pages:** Defined via Vango's file-based routing (`app/routes/`).
  * **APIs:** Developers can define standard API routes alongside Vango in their chosen router.
    ```go
    r.Post("/api/webhook", webhookHandler) // Standard Go handler
    r.Handle("/*", app.Handler())          // Vango pages
    ```

-----

## 5\. Authentication & State (`vango/auth`)

We abstract Authentication into a helper package that relies on the Session being hydrated.

### 5.1 The Interface

We provide a generic helper to avoid type assertions in user code.

```go
// vango/auth/auth.go

// Get retrieves a typed value from the session
func Get[T any](ctx vango.Ctx) (T, bool) {
    val := ctx.Session().Get("vango_auth_user")
    if user, ok := val.(T); ok {
        return user, true
    }
    var zero T
    return zero, false
}

// Require acts as a Guard Clause
func Require[T any](ctx vango.Ctx) (T, error) {
    user, ok := Get[T](ctx)
    if !ok {
        return user, vango.ErrUnauthorized
    }
    return user, nil
}
```

### 5.2 Usage in Components

```go
func Dashboard(ctx vango.Ctx) (vango.Component, error) {
    // 1. Guard Clause (Type-Safe)
    user, err := auth.Require[models.User](ctx)
    if err != nil {
        return nil, err // Bubbles up to Layout ErrorBoundary
    }

    // 2. Render
    return vango.Div(vango.Text("Welcome " + user.Email)), nil
}
```

-----

## 6\. Gap Fillers: Toast & Upload

To complete the framework experience without bloating the core, we introduce two standard packages.

### 6.1 `vango/toast` (Feedback)

Since we cannot use HTTP Flash cookies in a persistent WebSocket connection, we use a signal-based approach.

  * **Server:** `toast.Success(ctx, "Saved!")` sends a specific event opcode.
  * **Client:** The Thin Client listens for this opcode and dispatches a CustomEvent to a standard `<vango-toast>` web component.

### 6.2 `vango/upload` (Binary Data)

WebSockets are poor at handling large file uploads (blocking the heartbeat).

  * **Architecture:** Hybrid approach.
  * **Step 1:** Client performs standard HTTP POST to `app.UploadHandler`.
  * **Step 2:** Server streams file to temp storage (S3/Disk) and returns a `temp_id`.
  * **Step 3:** Client sends `temp_id` via WebSocket as a string value in the form submission.
  * **Step 4:** Vango Handler uses `upload.Claim(temp_id)` to finalize the file.

-----

## 7\. Implementation Roadmap

1.  **Core Update:** Modify `vango.Config` to add `OnSessionStart`.
2.  **Session Logic:** Implement thread-safe `Session.Get/Set`.
3.  **Auth Package:** Create `vango/auth` with Generics support.
4.  **Refactor Handler:** Ensure `app.Handler()` is fully `net/http` compliant.
5.  **Documentation:** Write the "Integration Guide" showing how to use Vango with Chi + Gorilla Sessions.

## 8\. Final Architecture Diagram

```mermaid
graph TD
    UserRequest[Browser Request] --> LB[Load Balancer]
    LB --> GoApp[Go Binary]
    
    subgraph "Layer 1: HTTP Stack (net/http)"
        GoApp --> Router[Router (Chi/Mux)]
        Router --> Middleware[Middleware Chain]
        Middleware --> AuthMW[Auth Middleware]
        AuthMW --> Upgrade[Upgrade Handler]
    end
    
    subgraph "The Context Bridge"
        Upgrade -- "1. Extract User from Context" --> Hydrator[OnSessionStart]
        Hydrator -- "2. Inject User to Session" --> SessionStore[Vango Session]
    end
    
    subgraph "Layer 2: Vango Runtime"
        SessionStore --> EventLoop[Event Loop]
        EventLoop --> VangoMW[Vango Middleware]
        VangoMW --> RouterInt[Internal Radix Router]
        RouterInt --> Component[Component Logic]
    end
    
    Component -- "auth.Require[User]" --> SessionStore
```