package vango

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/vango-go/vango/pkg/server"
	"github.com/vango-go/vango/pkg/vango"
)

// ssrContext implements server.Ctx for SSR (no WebSocket session).
// It provides read-only access to request data and no-ops for session operations.
type ssrContext struct {
	request *http.Request
	params  map[string]string
	config  Config
	logger  *slog.Logger
	values  map[any]any
}

func newSSRContext(r *http.Request, params map[string]string, cfg Config, logger *slog.Logger) *ssrContext {
	if params == nil {
		params = make(map[string]string)
	}
	return &ssrContext{
		request: r,
		params:  params,
		config:  cfg,
		logger:  logger,
		values:  make(map[any]any),
	}
}

// Request info
func (c *ssrContext) Request() *http.Request       { return c.request }
func (c *ssrContext) Path() string                 { return c.request.URL.Path }
func (c *ssrContext) Method() string               { return c.request.Method }
func (c *ssrContext) Query() url.Values            { return c.request.URL.Query() }
func (c *ssrContext) QueryParam(key string) string { return c.request.URL.Query().Get(key) }
func (c *ssrContext) Param(key string) string      { return c.params[key] }
func (c *ssrContext) Header(key string) string     { return c.request.Header.Get(key) }
func (c *ssrContext) Cookie(name string) (*http.Cookie, error) {
	return c.request.Cookie(name)
}

// Response control (no-ops for SSR - response is handled by App)
func (c *ssrContext) Status(code int)               {}
func (c *ssrContext) Redirect(url string, code int) {}
func (c *ssrContext) SetHeader(key, value string)   {}
func (c *ssrContext) SetCookie(cookie *http.Cookie) {}

// Session (nil for SSR - no WebSocket session)
func (c *ssrContext) Session() *server.Session { return nil }
func (c *ssrContext) User() any                { return nil }
func (c *ssrContext) SetUser(user any)         {}

// Logging
func (c *ssrContext) Logger() *slog.Logger {
	if c.logger != nil {
		return c.logger
	}
	return slog.Default()
}

// Lifecycle
func (c *ssrContext) Done() <-chan struct{} {
	return c.request.Context().Done()
}

// Request-scoped values
func (c *ssrContext) SetValue(key, value any) { c.values[key] = value }
func (c *ssrContext) Value(key any) any       { return c.values[key] }

// Events (no-op for SSR - no WebSocket to send to)
func (c *ssrContext) Emit(name string, data any) {}

// Dispatch executes immediately for SSR (no event loop)
func (c *ssrContext) Dispatch(fn func()) { fn() }

// Navigation (no-op for SSR)
func (c *ssrContext) Navigate(path string, opts ...server.NavigateOption) {}

// Context propagation
func (c *ssrContext) StdContext() context.Context { return c.request.Context() }
func (c *ssrContext) WithStdContext(ctx context.Context) server.Ctx {
	clone := *c
	clone.request = c.request.WithContext(ctx)
	return &clone
}

// Event & Metrics (SSR defaults)
func (c *ssrContext) Event() *server.Event    { return nil }
func (c *ssrContext) PatchCount() int         { return 0 }
func (c *ssrContext) AddPatchCount(count int) {}

// Storm budget (nil for SSR)
func (c *ssrContext) StormBudget() vango.StormBudgetChecker { return nil }

// Render mode (0 = ModeNormal for SSR)
func (c *ssrContext) Mode() int { return 0 }

// Asset resolution
func (c *ssrContext) Asset(source string) string {
	// For SSR, just prefix with static prefix
	prefix := c.config.Static.Prefix
	if prefix == "" {
		prefix = "/"
	}
	if prefix[len(prefix)-1] != '/' {
		prefix += "/"
	}
	return prefix + source
}
