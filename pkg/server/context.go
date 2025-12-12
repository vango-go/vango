package server

import (
	"log/slog"
	"net/http"
	"net/url"
)

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
}

// ctx is the concrete implementation of Ctx.
type ctx struct {
	request *http.Request
	writer  http.ResponseWriter
	session *Session
	params  map[string]string
	user    any
	logger  *slog.Logger
	status  int
	written bool
	values  map[any]any // Request-scoped values (Phase 10)
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
// Note: This is a placeholder implementation. The actual emission happens
// via the protocol layer which sends a custom event patch to the client.
func (c *ctx) Emit(name string, data any) {
	// TODO: Implement via protocol layer
	// For now, log that we would emit
	if c.logger != nil {
		c.logger.Debug("emit custom event", "name", name, "data", data)
	}
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
	}
}
