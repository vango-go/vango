package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/vango-dev/vango/v2/pkg/protocol"
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

	// Query returns the URL query parameters.
	Query() url.Values

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
	return c.request.URL.Path
}

// Method returns the HTTP method.
func (c *ctx) Method() string {
	return c.request.Method
}

// Query returns the URL query parameters.
func (c *ctx) Query() url.Values {
	return c.request.URL.Query()
}

// Param returns a route parameter by key.
func (c *ctx) Param(key string) string {
	return c.params[key]
}

// Header returns a request header value.
func (c *ctx) Header(key string) string {
	return c.request.Header.Get(key)
}

// Cookie returns a cookie by name.
func (c *ctx) Cookie(name string) (*http.Cookie, error) {
	return c.request.Cookie(name)
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
	return c.user
}

// SetUser sets the authenticated user.
func (c *ctx) SetUser(user any) {
	c.user = user
}

// Logger returns the request-scoped logger.
func (c *ctx) Logger() *slog.Logger {
	return c.logger
}

// Done returns a channel that's closed when the request is canceled.
func (c *ctx) Done() <-chan struct{} {
	return c.request.Context().Done()
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
// For WebSocket sessions, this sends a navigate event to the client which
// triggers client-side navigation. For HTTP-only requests, this falls back
// to an HTTP redirect.
func (c *ctx) Navigate(path string, opts ...NavigateOption) {
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

	// If no session (SSR-only), fall back to HTTP redirect
	if c.session == nil {
		code := 302 // Found (temporary redirect)
		if options.Replace {
			code = 303 // See Other (for POST -> GET)
		}
		c.Redirect(fullPath, code)
		return
	}

	// Build navigation event data
	navData := struct {
		Path    string `json:"path"`
		Replace bool   `json:"replace,omitempty"`
		Scroll  bool   `json:"scroll"`
	}{
		Path:    fullPath,
		Replace: options.Replace,
		Scroll:  options.Scroll,
	}

	// Encode to JSON
	detail, err := encodeJSON(navData)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("failed to encode navigation data", "error", err)
		}
		return
	}

	// Send as a special dispatch event that the client handles
	// The client will intercept "vango:navigate" and perform navigation
	patch := protocol.NewDispatchPatch("", "vango:navigate", detail)
	c.session.SendPatches([]protocol.Patch{patch})

	if c.logger != nil {
		c.logger.Debug("queued navigation", "path", fullPath, "replace", options.Replace)
	}
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
