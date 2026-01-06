# Routing Runtime Contracts

This document defines the authoritative contracts for Vango's routing system. All implementation code MUST conform to these specifications.

---

## 1. Route Registration Contract

### 1.1 File-Based Route Discovery

Routes are discovered from `app/routes/` using these conventions:

| File Pattern | URL Pattern | Type |
|-------------|-------------|------|
| `index.go` | `/` | Page |
| `about.go` | `/about` | Page |
| `projects/index.go` | `/projects` | Page |
| `projects/[id].go` | `/projects/:id` | Page (dynamic) |
| `projects/_id_.go` | `/projects/:id` | Page (Go-friendly syntax) |
| `blog/[...slug].go` | `/blog/*slug` | Page (catch-all) |
| `blog/_slug___.go` | `/blog/*slug` | Page (catch-all, Go-friendly) |
| `_layout.go` | (directory scope) | Layout |
| `_middleware.go` | (directory scope) | Middleware |
| `api/health.go` | `/api/health` | API |

**Catch-all capture semantics (Normative):**

Catch-all segments (`[...slug]` or `_slug___`) capture the remainder of the path:

- Value includes slashes: `/blog/a/b/c` → `Params["slug"] = "a/b/c"`
- Value MUST be non-empty: `/blog/` does NOT match `/blog/*slug`
- Leading slash is NOT included: captures `a/b/c`, not `/a/b/c`

**URL decoding:** Path segments are URL-decoded at match time, per-segment. The raw `Params["slug"]` contains the decoded path remainder with literal `/` separators.

```go
// blog/[...slug].go
type Params struct {
    Slug string `param:"slug"`  // "2024/01/my-post" for /blog/2024/01/my-post
}

// Or as segments (splits on /)
type Params struct {
    Slug []string `param:"slug"`  // ["2024", "01", "my-post"]
}
```

When the Params field is `[]string`, the catch-all value is split on `/` (not `,`). Each segment is URL-decoded individually.

### 1.2 Path Canonicalization (Normative)

All paths are canonicalized before matching. This is critical for security, caching, and predictable behavior.

#### 1.2.1 Canonical Path Form

- Canonical paths MUST NOT end in `/` except for root `/`
- Formally: if `path == "/"`, canonical is `/`; else canonical is `TrimRight(path, "/")`

#### 1.2.2 Normalization Rules

Before route matching, the router applies these transformations:

| Input | Canonical | Rule |
|-------|-----------|------|
| `/blog/` | `/blog` | Remove trailing slash |
| `/blog//post` | `/blog/post` | Collapse multiple slashes |
| `/blog/./post` | `/blog/post` | Remove `.` segments |
| `/blog/../other` | `/other` | Resolve `..` segments |
| `/blog` | `/blog` | Already canonical |

**Rejection rules (return 400 Bad Request):**
- Paths containing `\` (backslash)
- Paths containing NUL (`%00`)
- Invalid percent-escapes (e.g., `%GG`, `%2`)
- `..` that would escape root (e.g., `/../secret`)

#### 1.2.3 Percent-Decoding and Slash Handling

**For non-catch-all params:**
- If percent-decoding produces `/` (i.e., `%2F`), the match FAILS (404)
- This prevents path smuggling attacks

**For catch-all params:**
- Split path into segments FIRST, then decode each segment
- `%2F` within a segment becomes a literal `/` character in the decoded value
- Example: `/blog/a%2Fb/c` → segments `["a/b", "c"]` → `Params["slug"] = "a/b/c"`

> **Ambiguity note:** The string form `"a/b/c"` is ambiguous—it could mean `["a","b","c"]` or `["a/b","c"]`. If you need to preserve original segment boundaries (e.g., when `%2F` was used), use `[]string` instead of `string` for your Params field.

#### 1.2.4 Canonicalization Behavior

**Input string specification:**

| Context | Input Source | Encoding |
|---------|-------------|----------|
| HTTP | `r.URL.EscapedPath()` + `r.URL.RawQuery` | Percent-encoded |
| WS link click | `href` attribute from anchor | May be percent-encoded |
| WS popstate | `location.pathname` + `location.search` | Decoded by browser |
| WS ctx.Navigate | Argument string | Application-provided |

For WS navigation, the client SHOULD send the raw `href` attribute for link clicks (preserving percent-escapes). For `popstate`, `location.pathname` is already decoded by the browser, so strict percent-escape validation is not possible—canonicalization proceeds on the decoded path.

**WS percent-escape validation rule:** Invalid percent-escape rejection (e.g., `%GG`, `%2`) ONLY applies when the input string contains `%` characters. If the input has no `%`, skip percent-escape validation entirely. This avoids false rejections on browser-decoded paths from `popstate`.

**HTTP requests (SSR):**
- Canonicalize from `r.URL.EscapedPath()` (preserves percent-encoding)
- If `rawPath != canonicalPath`, respond with `308 Permanent Redirect` to canonical URL
- Location header includes original query string
- This ensures one canonical URL per route (SEO, caching)

**WebSocket navigation (EventNavigate / ctx.Navigate):**
- Canonicalize the target path
- Proceed with matching using canonical path
- If canonicalization changed the path, emit `NAV_REPLACE` (not `NAV_PUSH`) to avoid history duplication
- If path was already canonical, preserve the original push/replace intent from the navigation request

#### 1.2.5 Trailing Slash and Catch-All Interaction

Given catch-all rule "value MUST be non-empty":
- `/blog/` → canonical `/blog` → does NOT match `/blog/*slug`
- `/blog/a/` → canonical `/blog/a` → matches with `slug="a"`

### 1.3 Parameter Type Annotations

Dynamic segments support explicit type annotations using **bracket notation only**:

```
[id]        → string (default)
[id:int]    → int (validated, 404 on invalid)
[id:uuid]   → string (UUID format validated)
```

**Go-friendly underscore notation is always untyped:**
```
_id_        → string (equivalent to [id])
_slug___    → catch-all string (equivalent to [...slug])
```

Type annotations MUST use bracket notation. Underscore notation is provided only for Go identifier compatibility (avoiding `[` in filenames) and does not support type suffixes.

> **Deliberate change from prior spec:** Earlier spec text (§9.2) implied `[id].go` → `ID int` "automatically typed from filename." This contract supersedes that: **no automatic type inference**. All params default to `string` unless explicitly annotated with `:int` or `:uuid`. This makes the contract explicit and avoids magic based on param names.

**Type annotation vs Params struct compatibility (Normative):**

If a route segment has an explicit type annotation (`:int`, `:uuid`, etc.), the dispatcher MUST validate/coerce accordingly. If the handler's `Params` struct field type is incompatible with the annotation, the generator MUST error at build time:

```
ERROR: Type mismatch for route parameter 'id'
  File: /projects/[id:int].go
  Segment annotation: int
  Params struct field: string

Either change the annotation to [id] or change the field type to int.
```

**Compatibility matrix:**

| Annotation | Compatible Go Types |
|------------|---------------------|
| (none) or underscore | `string`, `[]string` (catch-all only) |
| `:int` | `int`, `int64`, `int32`, etc. |
| `:uuid` | `string` (runtime UUID validation) |

**Generator responsibility:** To enforce this contract, the generator MUST parse the route file's `type Params struct{...}` AST and map `param` tags to Go field types. If no `Params` struct is exported, the generator assumes no typed params and skips validation.

### 1.4 Registration API

Routes are registered via explicit method calls (no init() side effects):

```go
// Generated routes_gen.go calls:
func Register(r *router.Router) {
    r.Layout("/", root.Layout)
    r.Layout("/projects", projects.Layout)
    r.Middleware("/api", api.Middleware()...)
    r.Page("/", index.IndexPage)
    r.Page("/about", about.AboutPage)
    r.Page("/projects/:id", projects.ShowPage)
    r.API("GET", "/api/health", api.HealthGET)
    r.API("POST", "/api/users", api.UsersPOST)
}
```

### 1.5 Registration Order

Generator MUST produce deterministic output that reflects match priority (not just alphabetical). This ensures "first registered wins" aligns with specificity.

**Registration order (most specific first):**
1. Layouts: root to leaf (alphabetical within each level)
2. Middleware: root to leaf (alphabetical within each level)
3. Pages, ordered by specificity:
   - Static routes first (more static segments = higher priority, then longer paths)
   - Typed parameter routes (`:id:int`, `:id:uuid`)
   - Plain parameter routes (`:id`)
   - Catch-all routes (shorter prefix first)
   - Alphabetical tie-breaker within each tier
4. APIs: same specificity ordering as pages, then by method (GET < HEAD < OPTIONS < POST < PUT < PATCH < DELETE)

**Example ordering:**
```
/users/settings      # static (2 segments)
/users/profile       # static (2 segments, alphabetical after settings)
/users/:id:int       # typed param
/users/:id           # plain param
/users/*path         # catch-all
```

This eliminates "registration order" as a hidden footgun—routes register in the order they would naturally match.

**Relationship between registration order and runtime priority:**

The router's radix tree structure enforces priority (static > typed param > plain param > catch-all) regardless of registration order. Registration order only affects ties within the same match class. Since duplicates are forbidden (§1.6), ties should not occur in practice. The generator's deterministic ordering is primarily for:
1. Reproducible builds (same input → same output)
2. Code review clarity (predictable generated code)
3. Debugging (registration order matches expected match order)

### 1.6 Duplicate Route Detection (Normative)

Generator MUST error on duplicate URL patterns that resolve to the same method+path+kind (Page/API/Layout/Middleware).

**Go-friendly syntax is an alternative encoding, not an alias:**
- `[id].go` and `_id_.go` resolve to the same URL pattern
- These forms MUST NOT coexist in the same directory
- Generator MUST fail with a clear message if both exist

```
ERROR: Duplicate route detected
  /projects/[id].go → /projects/:id
  /projects/_id_.go → /projects/:id

Remove one of these files. Go-friendly syntax (_id_) is an alternative
to bracket syntax ([id]), not an alias.
```

**Same-path different-type collisions:**
- A Page and API at the same path are allowed (different HTTP semantics)
- Two Pages at the same path are NOT allowed
- Two APIs with the same method+path are NOT allowed

---

## 2. Handler Signature Contract

**Type bridge:** `vango.Ctx` is the public API type for route handlers. Internally, `vango.Ctx` is defined as:
```go
type Ctx = server.Ctx  // or interface embedding server.Ctx
```
This allows route handlers to use the clean `vango.Ctx` name while the runtime uses `server.Ctx`.

### 2.1 Page Handlers

Page handlers are called **once per navigation**. Blocking I/O is allowed.

**Canonical form:**
```go
func ShowPage(ctx vango.Ctx, p Params) *vdom.VNode {
    // Blocking I/O allowed here
    project, err := db.GetProject(p.ID)
    if err != nil {
        return ErrorView(err)
    }
    return ProjectView(project)
}

type Params struct {
    ID int `param:"id"`
}
```

**Supported variants:**
```go
// Without params
func IndexPage(ctx vango.Ctx) *vdom.VNode

// Returning Component (Render() called once)
func AboutPage(ctx vango.Ctx) vdom.Component

// Component with params
func ShowPage(ctx vango.Ctx, p Params) vdom.Component
```

**Reactivity rules:**

The page handler itself MUST be non-reactive and run once per navigation. It MAY return a component whose render function uses signals, effects, resources, and actions normally.

```go
// CORRECT: Page handler does I/O, returns reactive component
func ShowPage(ctx vango.Ctx, p Params) *vdom.VNode {
    project := db.GetProject(p.ID)  // Blocking I/O in handler: OK
    return ProjectView(project)      // Component can be reactive internally
}

// INCORRECT: Creating signals directly in page handler
func ShowPage(ctx vango.Ctx, p Params) *vdom.VNode {
    count := vango.NewSignal(0)  // WRONG: handler runs once, signal orphaned
    return Div(Text(count.Get()))
}
```

**NOT allowed in page handlers (the handler function body itself):**
- `vango.NewSignal()` — signals must be created inside `vango.Func` render
- `vango.Effect()` — effects must be inside reactive components
- Relying on re-execution — use `vango.Func` for reactive behavior

### 2.2 Layout Handlers

Layouts wrap page content. The `Slot` is the pre-rendered child VNode tree (not a component).

```go
func Layout(ctx vango.Ctx, children vango.Slot) *vdom.VNode {
    return Div(Class("layout"),
        Header(Nav(...)),
        Main(children),  // Insert pre-rendered page content
        Footer(...),
    )
}
```

Note: `vango.Slot` is a type alias for `*vdom.VNode`. The child content is already rendered before being passed to the layout.

Layouts are applied **root to leaf**. For `/projects/123`:
1. `app/routes/_layout.go` (root layout)
2. `app/routes/projects/_layout.go` (projects layout)
3. `app/routes/projects/[id].go` (page)

### 2.3 API Handlers

API handlers return data that is JSON-encoded.

**Canonical form:**
```go
func HealthGET(ctx vango.Ctx) (HealthResponse, error) {
    return HealthResponse{Status: "ok"}, nil
}

// Or with params
func UserGET(ctx vango.Ctx, p Params) (User, error) {
    return db.GetUser(p.ID)
}

// Or with request body
func UsersPOST(ctx vango.Ctx, body CreateUserRequest) (User, error) {
    return db.CreateUser(body)
}
```

**Handler name conventions:**
- File exports `func GET(ctx)` → registered as-is
- File exports `func HealthGET(ctx)` → registered as-is
- Priority: exact method name (`GET`) > resource+method (`HealthGET`)

**Collision rules (Normative):**

Generator MUST error on ambiguous exports **per HTTP method** within a file:

```
ERROR: Ambiguous API handler exports in /api/health.go
  Both GET() and HealthGET() are exported for GET method.

Export only one handler per HTTP method.
```

Note: Ambiguity is checked per method, not per file. Having both `GET()` and `POST()` in the same file is fine. Having both `GET()` and `HealthGET()` is an error (two handlers for GET).

**Supported HTTP methods:**

GET, HEAD, OPTIONS, POST, PUT, PATCH, DELETE

Method ordering for registration: GET < HEAD < OPTIONS < POST < PUT < PATCH < DELETE (alphabetical fallback for unlisted methods).

**Multiple methods per file:**

A single API file MAY export handlers for multiple HTTP methods:

```go
// api/users.go - exports handlers for multiple methods
func GET(ctx vango.Ctx) ([]User, error) { ... }      // GET /api/users
func POST(ctx vango.Ctx, body CreateUser) (User, error) { ... }  // POST /api/users
```

Each method is registered independently. The naming scheme (exact method vs resource+method) must be consistent within a file—mixing `GET()` with `UsersPOST()` is allowed but not recommended.

### 2.4 Middleware

```go
// Function returning middleware
func Middleware() []router.Middleware {
    return []router.Middleware{
        auth.RequireAuth(),
        ratelimit.New(100),
    }
}

// Or as a variable
var Middleware = []router.Middleware{
    logger.New(),
}
```

---

## 3. Route Matching Contract

### 3.1 Match Priority

When multiple routes could match, priority is:
1. **Exact static match** (highest priority)
2. **Parameter with type constraint** (`:id:int`, `:id:uuid`)
3. **Plain parameter** (`:id`)
4. **Catch-all match** (`*slug`) (lowest priority)

Within the same priority, **first registered wins**.

**Typed param collision rule (Normative):**

Routes with the same path but different param constraints (e.g., `[id:int]` and `[id]`) in the same directory are illegal. Generator MUST error:

```
ERROR: Conflicting parameter constraints at /users/
  [id:int].go → /users/:id (typed)
  [id].go     → /users/:id (untyped)

A route cannot have both typed and untyped variants. Choose one.
```

This prevents surprising behavior where `/users/abc` matches the untyped route but `/users/123` matches the typed route.

**Router internal representation (Normative):**

Typed constraints are stored separately from the route pattern string:
- Pattern string remains `/users/:id` (no type suffix in pattern)
- Type constraints are stored per-node in the router tree
- Matching checks type constraints during param edge traversal

**Generator responsibility:** The generator MUST pass typed constraint metadata into route registration calls. Since the pattern string is normalized (no type suffix), the runtime needs explicit constraint information:

```go
// Generator emits constraint metadata alongside registration
r.Page("/users/:id", users.ShowPage, router.WithParamType("id", "int"))
// Or via route options struct
r.Page("/users/:id", users.ShowPage, router.RouteOpts{
    ParamTypes: map[string]string{"id": "int"},
})
```

This allows the radix tree structure to remain simple while supporting typed/untyped priority tiers.

### 3.2 Match Result

A successful match produces:
```go
type MatchResult struct {
    PageHandler PageHandler         // nil for API routes
    APIHandler  APIHandler          // nil for page routes
    Layouts     []LayoutHandler     // Root to leaf order
    Middleware  []Middleware        // Combined chain
    Params      map[string]string   // Raw string params
    Route       *ScannedRoute       // Route metadata
}
```

### 3.3 Parameter Parsing

At dispatch time, `map[string]string` params are parsed into typed structs:

```go
type Params struct {
    ID   int    `param:"id"`
    Slug string `param:"slug"`
}
```

**Type coercion rules:**
| Go Type | Parsing | Invalid Value Behavior |
|---------|---------|----------------------|
| `string` | As-is | Never fails |
| `int`, `int64`, etc. | `strconv.ParseInt` | 404 for pages, 400 for APIs |
| `uint`, `uint64`, etc. | `strconv.ParseUint` | 404 for pages, 400 for APIs |
| `float32`, `float64` | `strconv.ParseFloat` | 404 for pages, 400 for APIs |
| `bool` | `strconv.ParseBool` | 404 for pages, 400 for APIs |
| `[]string` | Split by `/` for catch-all params | Never fails |

**Validation:**
- UUID format: `/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i`
- If param type annotation specifies `:uuid` and value doesn't match → 404/400

---

## 4. Navigation Contract

### 4.1 Navigation Types

There are two distinct navigation mechanisms:

| Type | Patch Ops | Remounts Page? | Changes Path? |
|------|-----------|----------------|---------------|
| **Full Navigation** | NAV_PUSH (0x32), NAV_REPLACE (0x33) | Yes | Yes |
| **Query-Only Update** | URL_PUSH (0x30), URL_REPLACE (0x31) | No | No |

### 4.2 Full Navigation (NAV_PUSH / NAV_REPLACE)

Used when changing the route path. Server remounts the component tree.

**Wire format:**
```
NAV_PUSH:    [0x32][path:string]
NAV_REPLACE: [0x33][path:string]
```

Where `path` is the canonical path including query string (e.g., `/projects/123?tab=details`).

**Security constraint (Normative):** NAV_* payloads MUST be relative paths only:
- MUST start with `/`
- MUST NOT be a full URL (no `http://`, `https://`, `//`)
- Server MUST reject or sanitize absolute URLs to prevent open-redirect attacks
- Client MUST validate path starts with `/` before calling `pushState`/`replaceState`

**Client behavior:**
1. Receive NAV_* patch
2. Update history via `pushState`/`replaceState` with provided path
3. Apply subsequent DOM patches
4. Do NOT send EventNavigate back to server

**Server behavior:**
1. Receive EventNavigate from client (or ctx.Navigate() call)
2. Match new route
3. Remount page component tree (with layouts)
4. Diff old tree vs new tree
5. Send: [NAV_* patch] + [DOM patches]

### 4.3 Query-Only Updates (URL_PUSH / URL_REPLACE)

Used for updating query parameters without changing the route.

**Wire format:**
```
URL_PUSH:    [0x30][count:varint][key:string, value:string]...
URL_REPLACE: [0x31][count:varint][key:string, value:string]...
```

**Client behavior:**
1. Receive URL_* patch
2. Update only query params via URLSearchParams
3. Update history
4. Do NOT remount the route/component tree
5. Apply any DOM patches that arrive in the same frame (if present)

**Server behavior:**
- Used by `URLParam.Set()` to sync URL state
- URL patches are queued and sent alongside DOM patches in the same frame
- Does NOT trigger route remount (page handler is not re-invoked)

### 4.4 Programmatic Navigation (ctx.Navigate)

```go
// Push navigation (adds history entry)
ctx.Navigate("/projects/123")

// Replace navigation (replaces current entry)
ctx.Navigate("/projects/123", server.WithReplace())

// With query params
ctx.Navigate("/projects?filter=active")
```

**Implementation contract:**
- `ctx.Navigate()` sets a pending navigation on the event context
- At flush/commit, if pending navigation exists:
  1. Match new route
  2. Remount page tree
  3. Send NAV_* patch + DOM patches
- This is ONE transaction (no client roundtrip)

### 4.5 Link Click Navigation

When client intercepts a link click:
1. Do NOT update history immediately
2. Send `EventNavigate { path, replace }`
3. Wait for server response with NAV_* + DOM patches
4. Apply patches (URL update + DOM update)

### 4.6 Back/Forward Navigation (popstate)

1. Browser fires `popstate` event
2. Client sends `EventNavigate { path: location.pathname + location.search, replace: true }`
3. Server remounts and sends DOM patches
4. Server MAY omit NAV_REPLACE since URL already changed

---

## 5. Progressive Enhancement Contract

### 5.1 Link Interception Rules

A link click is intercepted by Vango ONLY when ALL conditions are true:
1. Element has explicit Vango marker: `data-vango-link` attribute
2. WebSocket connection is healthy
3. Link is same-origin
4. No modifier keys pressed (ctrl, meta, shift, alt)
5. No `target` attribute (or `target="_self"`)
6. No `download` attribute

If ANY condition fails → browser performs native navigation.

> **Deprecation notice:** Early implementations may have used `data-link` as a shorthand. This is now **deprecated**. Use `data-vango-link` exclusively. The `data-link` attribute will be ignored in future versions. The `vango.Link()` helper MUST emit `data-vango-link` only (not `data-link`).

### 5.2 Plain Anchors

```go
// This does native navigation (no interception)
A(Href("/about"), Text("About"))

// This uses SPA navigation (intercepted) - presence is sufficient
A(Href("/about"), Attr("data-vango-link", "true"), Text("About"))
A(Href("/about"), Attr("data-vango-link", ""), Text("About"))  // Also works

// Helper (recommended)
vango.Link("/about", Text("About"))  // Adds data-vango-link automatically
```

**`vango.Link` helper (Normative):**

Location: `vango/pkg/el/link.go` (or `vango/el` in the public API)

```go
// Link creates an anchor element with SPA navigation enabled.
// Sets: href, data-vango-link, and optionally data-prefetch.
func Link(href string, children ...vdom.Child) *vdom.VNode {
    return A(Href(href), Attr("data-vango-link", ""), children...)
}

// LinkPrefetch creates a Link that also triggers prefetch on hover.
func LinkPrefetch(href string, children ...vdom.Child) *vdom.VNode {
    return A(Href(href), Attr("data-vango-link", ""), Attr("data-prefetch", ""), children...)
}
```

**Marker detection:** The client checks for attribute presence, not value. Any truthy or empty value on `data-vango-link` enables interception.

### 5.3 No-JS / Failed-WS Fallback

When WebSocket is not connected:
- All links do native navigation
- Forms submit normally
- Initial page is SSR HTML (fully functional)

---

## 6. Self-Heal Contract

### 6.1 Patch Application Failure

If applying a DOM patch fails (target node missing, operation invalid):

1. Log error with details (HID, operation, expected state)
2. If navigation was in progress → `location.assign(targetURL)`
3. Else → `location.reload()` or request `RESYNC_FULL`

**Rationale:** A patch mismatch indicates client/server DOM drift. The safest recovery is full page reload to re-sync state.

### 6.2 Handler Not Found

If server returns `ErrHandlerNotFound` for a navigation:
1. Client logs error
2. Client performs hard navigation: `location.assign(targetPath)`

**Error wire format (reference):**
```
[uint16: code][varint-len string: message][bool: fatal]
```
Routing uses `protocol.ErrNotFound` (0x0102) for 404 responses. See `vango/pkg/protocol/error.go` for the complete error code registry.

### 6.3 Connection Loss During Navigation

If WebSocket closes while awaiting navigation response:
1. Client detects pending navigation
2. Client performs: `location.assign(pendingPath)`

---

## 7. Error Handling Contract

### 7.1 Error Page Hook

```go
router.SetErrorPage(func(ctx vango.Ctx, err error) *vdom.VNode {
    return Div(Class("error"),
        H1(Text("Error")),
        P(Text(err.Error())),
    )
})
```

Signature: `func(ctx vango.Ctx, err error) *vdom.VNode`

### 7.2 Not Found Hook

```go
router.SetNotFound(func(ctx vango.Ctx) *vdom.VNode {
    return Div(Class("not-found"),
        H1(Text("404 - Page Not Found")),
    )
})
```

Signature: `func(ctx vango.Ctx) *vdom.VNode`

Both hooks return `*vdom.VNode` (consistent with page handlers). The NotFound hook does not receive params since the route didn't match.

### 7.3 API Error Responses

> **Note:** API error response format is defined by the API subsystem contract, not routing. This section is informational only and does not block routing implementation.

API handlers that return errors produce JSON with appropriate HTTP status codes. The exact shape is defined by the API middleware/encoder configuration.

---

## 8. Prefetch Contract

### 8.1 Prefetch Trigger

Client sends `EventType.CUSTOM` (0xFF) with:
- Name: `"prefetch"`
- Data: JSON-encoded `{ "path": "/target/path" }`

Triggers:
- `mouseenter` on links with `data-prefetch` attribute
- Programmatic `vango.prefetch(path)` call

**Wire format:**
```
[0xFF (CUSTOM)][name: "prefetch" (varint-len string)][data: JSON bytes (varint-len)]

Example payload bytes:
  name = "prefetch" (8 bytes + length prefix)
  data = {"path":"/projects/123"} (JSON, length-prefixed)
```

> **Note:** If a dedicated `PREFETCH` event type (e.g., 0x0F) is added in future, this contract will be updated. For now, use CUSTOM with name="prefetch".

> **Client implementation note:** The current `events.js` passes an object for `data`, but the wire format requires JSON bytes. Client code MUST `JSON.stringify()` the data object before encoding. Example: `sendCustom("prefetch", JSON.stringify({ path: href }))`.

### 8.2 Server Behavior

1. Route match the prefetch path (using canonical path)
2. Execute page handler in "prefetch mode" (`ctx.Mode() == Prefetch`)
3. Cache result per session, keyed by canonical path
4. TTL: 30 seconds (configurable)
5. Max entries per session: 10 (LRU eviction)

### 8.3 Prefetch Execution Model (Normative)

Prefetch uses **bounded I/O**: synchronous work is allowed, but no async work may outlive the prefetch call.

**Core invariant:** Prefetch MUST be referentially transparent with respect to session state—no signal writes, no auth changes, no navigation, no durable side effects.

#### 8.3.1 Allowed Operations

| Operation | Allowed? | Notes |
|-----------|----------|-------|
| Synchronous DB queries | Yes | Must complete within timeout |
| Cache reads | Yes | |
| HTTP calls (blocking) | Yes | Must complete within timeout |
| Reading session/user/params | Yes | |
| Constructing VDOM tree | Yes | Primary purpose |
| Pure computation | Yes | |

#### 8.3.2 Forbidden Operations

| Operation | Dev Mode | Prod Mode |
|-----------|----------|-----------|
| `vango.Interval()` | panic | no-op + warn |
| `vango.Timeout()` | panic | no-op + warn |
| `vango.Subscribe()` | panic | no-op + warn |
| `vango.GoLatest()` | panic | no-op + warn |
| `vango.Action()` scheduling | panic | no-op + warn |
| Signal writes (`signal.Set()`) | panic | drop + warn |
| `ctx.SetUser()` | panic | panic |
| `ctx.Navigate()` | ignore | ignore |

**Enforcement:** All async-capable APIs check `ctx.Mode()` and fail immediately if in prefetch mode. This makes violations loud in development and harmless in production.

#### 8.3.3 Resource Bounds

To prevent DoS and tail-latency amplification:

| Bound | Default | Purpose |
|-------|---------|---------|
| Timeout | 100ms | Abort prefetch if handler takes too long |
| Concurrency per session | 2 | Max simultaneous prefetch evaluations |
| Concurrency per server | 50 | Global prefetch limit |

If prefetch exceeds timeout or is cancelled, discard result (do not cache partial work).

### 8.4 Navigation with Prefetch Hit

When `EventNavigate` arrives for a prefetched path:
1. Check prefetch cache (using canonical path as key)
2. If hit and not stale → reuse rendered tree and cached data
3. Continue with diff and patch generation (skip page handler I/O)

### 8.5 Rate Limiting

Prefetch events are rate-limited per session:
- Max 5 prefetch requests per second
- Excess requests are silently dropped

---

## 9. Context Bridge Contract

### 9.1 HTTP → WebSocket Transition

For initial page load:
1. HTTP request arrives
2. SSR renders page with full context (user, session, params)
3. HTML includes hydration data
4. Client connects WebSocket
5. Server creates Session with same context

The `vango.Ctx` interface is consistent across both modes:
- `ctx.Path()` → current route path
- `ctx.Param(key)` → route param value
- `ctx.Query(key)` → query param value
- `ctx.User()` → authenticated user (if any)
- `ctx.Session()` → WebSocket session (nil during SSR)

### 9.2 Session Route State

```go
type Session struct {
    currentPath string           // Current route path
    currentRoute *MatchResult    // Current route match
    rootComponent vdom.Component // Route root (re-mountable)
    // ...
}
```

On navigation, `currentPath` and `currentRoute` are updated, and `rootComponent` is remounted.

---

## Summary: Patch Type Reference

| Op Code | Name | Payload | Purpose |
|---------|------|---------|---------|
| 0x30 | URL_PUSH | `map[string]string` params | Update query params, push history |
| 0x31 | URL_REPLACE | `map[string]string` params | Update query params, replace history |
| 0x32 | NAV_PUSH | `string` path | Full navigation, push history |
| 0x33 | NAV_REPLACE | `string` path | Full navigation, replace history |

---

## Implementation Checklist

The following files need updates to conform to this contract:

### Routing Core (Go)
- `vango/pkg/router/scanner.go` - discovery rules, symbol detection, typed annotations, duplicates/conflicts
- `vango/pkg/router/codegen.go` - generate Register(*router.Router), deterministic ordering, duplicate detection errors
- `vango/pkg/router/router.go` - add Page/Layout/Middleware/API public methods; canonicalization-aware matching
- `vango/pkg/router/tree.go` - match priority tiers, catch-all non-empty rule
- `vango/pkg/router/params.go` - per-segment URL-decoding, %2F handling, typed parsing
- `vango/pkg/router/types.go` - type constraints metadata
- `vango/pkg/router/canonicalize.go` - path canonicalization functions (NEW)

### HTTP SSR + Runtime Integration
- `vango/pkg/server/server.go` - HTTP request path canonicalization + 308 redirect
- `vango/pkg/server/session.go` - EventNavigate handling, session route state, remount + diff
- `vango/pkg/server/context.go` - ctx.Navigate() using NAV_* patches

### Protocol (Go)
- `vango/pkg/protocol/patch.go` - NAV_PUSH=0x32, NAV_REPLACE=0x33

### Thin Client (JS)
- `vango/client/src/codec.js` - decode NAV_* patches
- `vango/client/src/patches.js` - apply NAV_* by updating history
- `vango/client/src/events.js` - link interception with data-vango-link, prefetch JSON bytes

### Public API Helpers
- `vango/pkg/router/link.go` - vango.Link() emits data-vango-link

---

*This document is the authoritative contract. Implementation MUST conform to these specifications.*
