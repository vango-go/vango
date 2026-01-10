package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/vango-go/vango/pkg/assets"
	"github.com/vango-go/vango/pkg/protocol"
	"github.com/vango-go/vango/pkg/vango"
)

// encodeJSON encodes a value to JSON string.
func encodeJSON(v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// =============================================================================
// Render Mode (Phase 7: Prefetch)
// =============================================================================

// RenderMode indicates the rendering context for a request.
// This is used by the prefetch system to enforce read-only behavior.
type RenderMode int

const (
	// ModeNormal is the default rendering mode for regular requests.
	// All operations are allowed.
	ModeNormal RenderMode = iota

	// ModePrefetch is used when prefetching a route for caching.
	// In this mode, side effects are forbidden:
	//   - Signal.Set() panics in dev / drops in prod
	//   - Effect/Interval/Timeout panic in dev / no-op in prod
	//   - SetUser() panics
	//   - Navigate() is ignored
	//
	// Per Routing Spec Section 8.3.1, prefetch uses "bounded I/O":
	// synchronous work is allowed, but no async work may outlive the prefetch.
	ModePrefetch
)

// String returns a human-readable name for the render mode.
func (m RenderMode) String() string {
	switch m {
	case ModeNormal:
		return "normal"
	case ModePrefetch:
		return "prefetch"
	default:
		return "unknown"
	}
}

// =============================================================================
// Navigation Options (Phase 7: Routing)
// =============================================================================

// NavigateOption is a functional option for Navigate.
type NavigateOption func(*navigateOptions)

// navigateOptions holds navigation configuration.
type navigateOptions struct {
	Replace bool           // Replace current history entry instead of pushing
	Params  map[string]any // Query parameters to add to the URL
	Scroll  bool           // Scroll to top after navigation (default: true)
}

// WithReplace replaces the current history entry instead of pushing.
func WithReplace() NavigateOption {
	return func(o *navigateOptions) {
		o.Replace = true
	}
}

// WithNavigateParams adds query parameters to the navigation URL.
func WithNavigateParams(params map[string]any) NavigateOption {
	return func(o *navigateOptions) {
		o.Params = params
	}
}

// WithoutScroll disables scrolling to top after navigation.
func WithoutScroll() NavigateOption {
	return func(o *navigateOptions) {
		o.Scroll = false
	}
}

// Ctx provides access to request data within components.
// It is passed to route handlers and can be used to access the request,
// session, and response control methods.
type Ctx interface {
	// Request info

	// Request returns the underlying HTTP request.
	Request() *http.Request

	// Path returns the URL path.
	Path() string

	// Method returns the HTTP method.
	Method() string

	// Query returns the URL query parameters as url.Values.
	Query() url.Values

	// QueryParam returns a single query parameter value by key.
	// Returns an empty string if the key is not present.
	QueryParam(key string) string

	// Param returns a route parameter by key.
	Param(key string) string

	// Header returns a request header value.
	Header(key string) string

	// Cookie returns a cookie by name.
	Cookie(name string) (*http.Cookie, error)

	// Response control

	// Status sets the HTTP response status code.
	Status(code int)

	// Redirect redirects to the given URL with the given status code.
	Redirect(url string, code int)

	// SetHeader sets a response header.
	SetHeader(key, value string)

	// SetCookie sets a response cookie.
	SetCookie(cookie *http.Cookie)

	// Session

	// Session returns the WebSocket session (nil for SSR-only requests).
	Session() *Session

	// User returns the authenticated user (set by auth middleware).
	User() any

	// SetUser sets the authenticated user.
	SetUser(user any)

	// Logging

	// Logger returns the request-scoped logger.
	Logger() *slog.Logger

	// Lifecycle

	// Done returns a channel that's closed when the request is canceled.
	Done() <-chan struct{}

	// Request-scoped values (Phase 10)

	// SetValue stores a request-scoped value.
	// These values are only available for the duration of the current event/request.
	SetValue(key, value any)

	// Value retrieves a request-scoped value.
	Value(key any) any

	// Custom events (Phase 10)

	// Emit dispatches a custom event to the client.
	// The event will be dispatched as a CustomEvent with the given name and detail.
	// Use this for notifications, toast messages, analytics, etc.
	Emit(name string, data any)

	// Dispatch (Phase 12: Spec compliance)

	// Dispatch queues a function to run on the session's event loop.
	// This is safe to call from any goroutine and is the correct way to
	// update signals from asynchronous operations (database calls, timers, etc.).
	//
	// The function will be executed synchronously on the event loop, ensuring
	// signal writes are properly serialized. After the function completes,
	// pending effects will run and dirty components will re-render.
	//
	// Example:
	//
	//     go func() {
	//         user, err := db.Users.FindByID(ctx.StdContext(), id)
	//         ctx.Dispatch(func() {
	//             if err != nil {
	//                 errorSignal.Set(err)
	//             } else {
	//                 userSignal.Set(user)
	//             }
	//         })
	//     }()
	//
	// IMPORTANT: This method MUST be safe to call from any goroutine.
	Dispatch(fn func())

	// ==========================================================================
	// Navigation (Phase 7: Routing)
	// ==========================================================================

	// Navigate performs a client-side navigation to the given path.
	// Unlike Redirect(), this updates the browser URL and re-renders the page
	// without a full HTTP redirect. The navigation is queued and processed
	// after the current handler completes.
	//
	// Options can be provided to customize behavior:
	//   - WithReplace() - replace current history entry instead of pushing
	//   - WithParams(map[string]any) - add query parameters to the URL
	//   - WithoutScroll() - disable scrolling to top after navigation
	//
	// Example:
	//
	//     func handleSave(ctx vango.Ctx) {
	//         project := saveProject()
	//         ctx.Navigate("/projects/" + project.ID)
	//     }
	//
	// For HTTP-only requests (SSR without WebSocket), this falls back to
	// an HTTP redirect.
	Navigate(path string, opts ...NavigateOption)

	// ==========================================================================
	// Context propagation (Phase 13: Production Hardening)
	// ==========================================================================

	// StdContext returns the standard library context with trace propagation.
	// Use this when calling external services or database drivers.
	//
	// Example:
	//     row := db.QueryRowContext(ctx.StdContext(), "SELECT * FROM users WHERE id = $1", userID)
	//     req, _ := http.NewRequestWithContext(ctx.StdContext(), "GET", url, nil)
	//
	// The context includes any trace spans injected by middleware (e.g., OpenTelemetry).
	StdContext() context.Context

	// WithStdContext returns a new Ctx with an updated standard context.
	// Used by middleware to inject trace spans.
	//
	// This is typically called by observability middleware:
	//     spanCtx, span := tracer.Start(ctx.StdContext(), "operation")
	//     defer span.End()
	//     ctx = ctx.WithStdContext(spanCtx)
	//     return next()
	WithStdContext(stdCtx context.Context) Ctx

	// ==========================================================================
	// Event & Metrics (Phase 13: Production Hardening)
	// ==========================================================================

	// Event returns the current WebSocket event being processed.
	// Returns nil for HTTP-only requests (SSR).
	//
	// Example:
	//     if event := ctx.Event(); event != nil {
	//         log.Printf("Processing %s event on %s", event.Type, event.HID)
	//     }
	Event() *Event

	// PatchCount returns the number of patches sent during this request.
	// This is updated after each render cycle.
	PatchCount() int

	// AddPatchCount increments the patch count for this request.
	// Called internally by the render system.
	AddPatchCount(count int)

	// ==========================================================================
	// Storm Budgets (Phase 16)
	// ==========================================================================

	// StormBudget returns the storm budget checker for this session.
	// Used by primitives (Action, Resource, GoLatest) to check rate limits.
	// Returns nil if storm budgets are not configured.
	//
	// See SPEC_ADDENDUM.md §A.4 for storm budget configuration.
	StormBudget() vango.StormBudgetChecker

	// ==========================================================================
	// Render Mode (Phase 7: Prefetch)
	// ==========================================================================

	// Mode returns the current render mode as int.
	// Returns 0 (ModeNormal) for regular requests, 1 (ModePrefetch) during prefetch.
	//
	// Primitives like Effect, Interval, and signal.Set() check this to
	// enforce read-only behavior during prefetch:
	//   - ModePrefetch (1): Signal writes are dropped, effects are no-ops
	//   - ModeNormal (0): All operations proceed normally
	//
	// This method implements vango.PrefetchModeChecker interface.
	//
	// Example (for primitive implementations):
	//     if ctx := UseCtx(); ctx != nil && ctx.Mode() == 1 {
	//         if DevMode { panic("signal write forbidden in prefetch") }
	//         return // drop in prod
	//     }
	Mode() int

	// ==========================================================================
	// Asset Resolution (DX Improvements)
	// ==========================================================================

	// Asset resolves a source asset path to its fingerprinted path.
	// Returns the original path if no manifest is configured or the asset is not found.
	//
	// This enables cache-busting via content-hashed filenames while keeping
	// templates simple:
	//
	// Example:
	//
	//     <script src={ctx.Asset("vango.js")}></script>
	//     // In dev: "/public/vango.js"
	//     // In prod: "/public/vango.a1b2c3d4.min.js"
	Asset(source string) string
}

// ctx is the concrete implementation of Ctx.
type ctx struct {
	request    *http.Request
	writer     http.ResponseWriter
	session    *Session
	params     map[string]string
	user       any
	logger     *slog.Logger
	status     int
	written    bool
	values     map[any]any     // Request-scoped values (Phase 10)
	stdCtx     context.Context // Standard context with trace propagation (Phase 13)
	event      *Event          // Current WebSocket event (Phase 13)
	patchCount int             // Number of patches sent (Phase 13)
	mode       RenderMode      // Render mode (Phase 7: Prefetch)

	// Asset resolver for fingerprinted asset paths (DX Improvements)
	assetResolver assets.Resolver

	// pendingNavigation is set by ctx.Navigate() and processed by flush().
	// Per Section 4.4 (Programmatic Navigation), navigation is processed
	// at flush time so NAV_* + DOM patches are sent in ONE transaction.
	pendingNavigation *pendingNav
}

// pendingNav holds a pending navigation request.
type pendingNav struct {
	Path    string
	Replace bool
}

// newCtx creates a new context for a request.
func newCtx(w http.ResponseWriter, r *http.Request, logger *slog.Logger) *ctx {
	return &ctx{
		request: r,
		writer:  w,
		params:  make(map[string]string),
		logger:  logger,
		status:  http.StatusOK,
	}
}

// Request returns the underlying HTTP request.
func (c *ctx) Request() *http.Request {
	return c.request
}

// Path returns the URL path.
func (c *ctx) Path() string {
	if c.request != nil {
		return c.request.URL.Path
	}
	if c.session != nil {
		return c.session.CurrentRoute
	}
	return ""
}

// Method returns the HTTP method.
func (c *ctx) Method() string {
	if c.request != nil {
		return c.request.Method
	}
	return http.MethodGet
}

// Query returns the URL query parameters.
func (c *ctx) Query() url.Values {
	if c.request != nil {
		return c.request.URL.Query()
	}
	return nil
}

// QueryParam returns a single query parameter value by key.
// Returns an empty string if the key is not present.
func (c *ctx) QueryParam(key string) string {
	if c.request != nil {
		return c.request.URL.Query().Get(key)
	}
	return ""
}

// Param returns a route parameter by key.
func (c *ctx) Param(key string) string {
	return c.params[key]
}

// Header returns a request header value.
func (c *ctx) Header(key string) string {
	if c.request != nil {
		return c.request.Header.Get(key)
	}
	return ""
}

// Cookie returns a cookie by name.
func (c *ctx) Cookie(name string) (*http.Cookie, error) {
	if c.request != nil {
		return c.request.Cookie(name)
	}
	return nil, http.ErrNoCookie
}

// Status sets the HTTP response status code.
func (c *ctx) Status(code int) {
	c.status = code
}

// Redirect redirects to the given URL with the given status code.
func (c *ctx) Redirect(url string, code int) {
	http.Redirect(c.writer, c.request, url, code)
	c.written = true
}

// SetHeader sets a response header.
func (c *ctx) SetHeader(key, value string) {
	c.writer.Header().Set(key, value)
}

// SetCookie sets a response cookie.
func (c *ctx) SetCookie(cookie *http.Cookie) {
	http.SetCookie(c.writer, cookie)
}

// Session returns the WebSocket session.
func (c *ctx) Session() *Session {
	return c.session
}

// User returns the authenticated user.
func (c *ctx) User() any {
	if c.user != nil {
		return c.user
	}
	if c.session != nil {
		if val := c.session.Get(DefaultAuthSessionKey); val != nil {
			return val
		}
	}
	if c.request != nil {
		if val := UserFromContext(c.request.Context()); val != nil {
			return val
		}
	}
	return nil
}

// SetUser sets the authenticated user.
// Per Section 8.3.2 of the Routing Spec, SetUser is forbidden during prefetch
// and panics in BOTH dev and production modes (unlike other operations that
// are silently dropped in production). This is because authentication changes
// must never occur during prefetch.
func (c *ctx) SetUser(user any) {
	// Check for prefetch mode - SetUser panics in BOTH modes per spec
	if c.mode == ModePrefetch {
		panic("vango: ctx.SetUser() is forbidden in prefetch mode")
	}
	c.user = user
}

// Logger returns the request-scoped logger.
func (c *ctx) Logger() *slog.Logger {
	return c.logger
}

// Done returns a channel that's closed when the request is canceled.
func (c *ctx) Done() <-chan struct{} {
	if c.request != nil {
		return c.request.Context().Done()
	}
	if c.stdCtx != nil {
		return c.stdCtx.Done()
	}
	return nil
}

// setSession sets the session for WebSocket requests.
func (c *ctx) setSession(s *Session) {
	c.session = s
}

// setParams sets the route parameters.
func (c *ctx) setParams(params map[string]string) {
	c.params = params
}

// getStatus returns the current status code.
func (c *ctx) getStatus() int {
	return c.status
}

// isWritten returns whether the response has been written.
func (c *ctx) isWritten() bool {
	return c.written
}

// write writes the response status code.
func (c *ctx) writeStatus() {
	if !c.written {
		c.writer.WriteHeader(c.status)
	}
}

// ResponseWriter returns the underlying response writer.
// Use with caution - prefer the Ctx methods when possible.
func (c *ctx) ResponseWriter() http.ResponseWriter {
	return c.writer
}

// WithLogger returns a new context with the given logger.
func (c *ctx) WithLogger(logger *slog.Logger) *ctx {
	clone := *c
	clone.logger = logger
	return &clone
}

// =============================================================================
// Request-scoped values (Phase 10)
// =============================================================================

// SetValue stores a request-scoped value.
func (c *ctx) SetValue(key, value any) {
	if c.values == nil {
		c.values = make(map[any]any)
	}
	c.values[key] = value
}

// Value retrieves a request-scoped value.
func (c *ctx) Value(key any) any {
	if c.values == nil {
		return nil
	}
	return c.values[key]
}

// =============================================================================
// Custom events (Phase 10)
// =============================================================================

// Emit dispatches a custom event to the client.
// The event will be dispatched as a CustomEvent with the given name and detail.
// Use this for notifications, toast messages, analytics, etc.
//
// The event is sent as a Dispatch patch which the client interprets as a
// CustomEvent to dispatch on the target element or document.
func (c *ctx) Emit(name string, data any) {
	if c.session == nil {
		if c.logger != nil {
			c.logger.Warn("cannot emit event: no session", "name", name)
		}
		return
	}

	// Convert data to JSON string for the patch
	var detail string
	if data != nil {
		if s, ok := data.(string); ok {
			detail = s
		} else {
			// Try to JSON-encode the data
			if encoded, err := encodeJSON(data); err == nil {
				detail = encoded
			} else {
				detail = fmt.Sprintf("%v", data)
			}
		}
	}

	// Send dispatch patch to client
	// The client will dispatch this as a CustomEvent on the document
	patch := protocol.NewDispatchPatch("", name, detail)
	c.session.SendPatches([]protocol.Patch{patch})

	if c.logger != nil {
		c.logger.Debug("emitted custom event", "name", name)
	}
}

// =============================================================================
// Dispatch (Phase 12: Spec compliance)
// =============================================================================

// Dispatch queues a function to run on the session's event loop.
// This is safe to call from any goroutine.
func (c *ctx) Dispatch(fn func()) {
	if c.session == nil {
		// No session (SSR-only request), execute immediately but warn
		if c.logger != nil {
			c.logger.Warn("dispatch called without session, executing inline")
		}
		fn()
		return
	}
	c.session.Dispatch(fn)
}

// =============================================================================
// Navigation (Phase 7: Routing)
// =============================================================================

// Navigate performs a client-side navigation to the given path.
//
// Per the Navigation Contract (Section 4.4), this:
//   - Sets a pending navigation on the context
//   - At flush() time, the pending navigation triggers:
//     1. Route matching
//     2. Page remount
//     3. NAV_* patch + DOM patches sent in ONE transaction
//   - For HTTP-only requests (SSR): Falls back to HTTP redirect
//
// The path must be a relative path starting with "/". It may include query
// parameters (e.g., "/projects/123?tab=details").
//
// Options can be provided to customize behavior:
//   - WithReplace() - replace current history entry instead of pushing
//   - WithNavigateParams(map[string]any) - add query parameters to the URL
//   - WithoutScroll() - disable scrolling to top after navigation (TODO)
func (c *ctx) Navigate(path string, opts ...NavigateOption) {
	// Per Section 8.3.2 of the Routing Spec, Navigate is ignored during prefetch.
	// Prefetch should be referentially transparent - no navigation side effects.
	if c.mode == ModePrefetch {
		if c.logger != nil {
			c.logger.Debug("ctx.Navigate() ignored during prefetch", "path", path)
		}
		return
	}

	// Apply options with defaults
	options := navigateOptions{
		Scroll: true, // Default to scrolling to top
	}
	for _, opt := range opts {
		opt(&options)
	}

	// Build full URL with query params
	fullPath := path
	if len(options.Params) > 0 {
		q := url.Values{}
		for k, v := range options.Params {
			q.Set(k, fmt.Sprintf("%v", v))
		}
		if len(q) > 0 {
			fullPath = path + "?" + q.Encode()
		}
	}

	// SECURITY: Validate that path is relative (starts with "/")
	// This prevents open-redirect attacks per Navigation Contract Section 4.2
	if !isRelativePath(fullPath) {
		if c.logger != nil {
			c.logger.Error("invalid navigation path (must be relative)", "path", fullPath)
		}
		return
	}

	// If no session (SSR-only), fall back to HTTP redirect
	if c.session == nil {
		code := 302 // Found (temporary redirect)
		if options.Replace {
			code = 303 // See Other (for POST -> GET)
		}
		c.Redirect(fullPath, code)
		return
	}

	// Set pending navigation to be processed by flush()
	// Per Section 4.4 (Programmatic Navigation):
	// "ctx.Navigate() sets a pending navigation on the event context.
	// At flush/commit, if pending navigation exists:
	//   1. Match new route
	//   2. Remount page tree
	//   3. Send NAV_* patch + DOM patches
	// This is ONE transaction (no client roundtrip)"
	c.pendingNavigation = &pendingNav{
		Path:    fullPath,
		Replace: options.Replace,
	}

	if c.logger != nil {
		c.logger.Debug("pending navigation set",
			"path", fullPath,
			"replace", options.Replace)
	}
}

// PendingNavigation returns the pending navigation, if any.
// This is used by Session.flush() to check for and process pending navigations.
func (c *ctx) PendingNavigation() (path string, replace bool, has bool) {
	if c.pendingNavigation == nil {
		return "", false, false
	}
	return c.pendingNavigation.Path, c.pendingNavigation.Replace, true
}

// ClearPendingNavigation clears the pending navigation.
// This is called after the navigation has been processed.
func (c *ctx) ClearPendingNavigation() {
	c.pendingNavigation = nil
}

// isRelativePath checks if a path is a valid relative path for navigation.
// Returns true if the path starts with "/" and is not an absolute URL.
func isRelativePath(path string) bool {
	// Must start with /
	if len(path) == 0 || path[0] != '/' {
		return false
	}
	// Must not be protocol-relative URL (//example.com)
	if len(path) >= 2 && path[1] == '/' {
		return false
	}
	// Must not contain protocol
	// Check for common protocols
	lowerPath := strings.ToLower(path)
	if strings.HasPrefix(lowerPath, "/http:") ||
		strings.HasPrefix(lowerPath, "/https:") ||
		strings.HasPrefix(lowerPath, "/javascript:") {
		return false
	}
	return true
}

// =============================================================================
// Context Propagation (Phase 13)
// =============================================================================

// StdContext returns the standard library context with trace propagation.
// If no custom context was set via WithStdContext, returns the request's context.
func (c *ctx) StdContext() context.Context {
	if c.stdCtx != nil {
		return c.stdCtx
	}
	if c.request != nil {
		return c.request.Context()
	}
	return context.Background()
}

// WithStdContext returns a new Ctx with an updated standard context.
// Used by middleware to inject trace spans.
func (c *ctx) WithStdContext(stdCtx context.Context) Ctx {
	clone := *c
	clone.stdCtx = stdCtx
	return &clone
}

// =============================================================================
// Event & Metrics (Phase 13)
// =============================================================================

// Event returns the current WebSocket event being processed.
// Returns nil for HTTP-only requests (SSR).
func (c *ctx) Event() *Event {
	return c.event
}

// setEvent sets the current event for WebSocket requests.
func (c *ctx) setEvent(e *Event) {
	c.event = e
}

// PatchCount returns the number of patches sent during this request.
func (c *ctx) PatchCount() int {
	return c.patchCount
}

// AddPatchCount increments the patch count for this request.
func (c *ctx) AddPatchCount(count int) {
	c.patchCount += count
}

// =============================================================================
// Storm Budgets (Phase 16)
// =============================================================================

// StormBudget returns the storm budget checker for this session.
// Returns nil if no session or storm budgets not configured.
func (c *ctx) StormBudget() vango.StormBudgetChecker {
	if c.session == nil {
		return nil
	}
	return c.session.StormBudget()
}

// =============================================================================
// Render Mode (Phase 7: Prefetch)
// =============================================================================

// Mode returns the current render mode.
// Returns ModeNormal for regular requests, ModePrefetch during prefetch.
// This implements vango.PrefetchModeChecker by returning int.
func (c *ctx) Mode() int {
	return int(c.mode)
}

// RenderMode returns the current render mode as RenderMode type.
// Use Mode() for interface compatibility with vango.PrefetchModeChecker.
func (c *ctx) RenderMode() RenderMode {
	return c.mode
}

// setMode sets the render mode for this context.
// This is called internally when creating a prefetch context.
func (c *ctx) setMode(mode RenderMode) {
	c.mode = mode
}

// IsPrefetch returns true if the context is in prefetch mode.
// This is a convenience method for checking if side effects should be suppressed.
func (c *ctx) IsPrefetch() bool {
	return c.mode == ModePrefetch
}

// =============================================================================
// Asset Resolution (DX Improvements)
// =============================================================================

// Asset resolves a source asset path to its fingerprinted path.
// Returns the original path if no resolver is configured or the asset is not found.
func (c *ctx) Asset(source string) string {
	if c.assetResolver == nil {
		return source
	}
	return c.assetResolver.Asset(source)
}

// setAssetResolver sets the asset resolver for this context.
// This is called internally when creating a context from a server with a configured resolver.
func (c *ctx) setAssetResolver(r assets.Resolver) {
	c.assetResolver = r
}

// =============================================================================
// Test Helpers (Phase 10F)
// =============================================================================

// NewTestContext creates a context for testing with the given session.
// This allows testing components that require a valid context.
func NewTestContext(s *Session) Ctx {
	return &ctx{
		request: nil,
		writer:  nil,
		session: s,
		params:  make(map[string]string),
		logger:  slog.Default(),
		status:  http.StatusOK,
		values:  make(map[any]any),
		stdCtx:  context.Background(),
	}
}
