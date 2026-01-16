package vango

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/vango-go/vango/pkg/auth"
	"github.com/vango-go/vango/pkg/routepath"
	"github.com/vango-go/vango/pkg/server"
	"github.com/vango-go/vango/pkg/vango"
)

// ssrContext implements server.Ctx for SSR (no WebSocket session).
// It provides read-only access to request data and no-ops for session operations.
type ssrContext struct {
	request *http.Request
	writer  http.ResponseWriter
	params  map[string]string
	config  Config
	logger  *slog.Logger
	values  map[any]any
	policy  *server.CookiePolicy

	user any

	status       int
	headers      http.Header
	cookies      []*http.Cookie
	redirectURL  string
	redirectCode int
	redirected   bool
}

func newSSRContext(w http.ResponseWriter, r *http.Request, params map[string]string, cfg Config, logger *slog.Logger, policy *server.CookiePolicy) *ssrContext {
	if params == nil {
		params = make(map[string]string)
	}
	user := server.UserFromContext(r.Context())
	return &ssrContext{
		request: r,
		writer:  w,
		params:  params,
		config:  cfg,
		logger:  logger,
		values:  make(map[any]any),
		policy:  policy,
		user:    user,
		status:  http.StatusOK,
		headers: make(http.Header),
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

// Response control (captured for App to apply)
func (c *ssrContext) Status(code int) {
	c.status = code
}

func (c *ssrContext) Redirect(url string, code int) {
	canon, err := routepath.CanonicalizeAndValidateNavPath(url)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("invalid redirect path (must be relative)", "path", url, "error", err)
		}
		return
	}
	c.redirected = true
	c.redirectURL = canon
	c.redirectCode = code
}

func (c *ssrContext) RedirectExternal(url string, code int) {
	canon, ok := server.ValidateExternalRedirectURL(url, c.config.Security.AllowedRedirectHosts)
	if !ok {
		if c.logger != nil {
			c.logger.Error("external redirect rejected", "url", url)
		}
		return
	}
	c.redirected = true
	c.redirectURL = canon
	c.redirectCode = code
}

func (c *ssrContext) SetHeader(key, value string) {
	c.headers.Set(key, value)
}

func (c *ssrContext) SetCookie(cookie *http.Cookie) {
	if err := c.SetCookieStrict(cookie); err != nil {
		if c.config.DevMode && c.Logger() != nil {
			c.Logger().Warn("cookie dropped", "error", err)
		}
	}
}

func (c *ssrContext) SetCookieStrict(cookie *http.Cookie, opts ...server.CookieOption) error {
	if cookie == nil {
		return nil
	}
	if c.policy == nil {
		return errors.New("server: cookie policy unavailable")
	}
	updated, err := c.policy.Apply(c.request, cookie, opts...)
	if err != nil {
		return err
	}
	c.cookies = append(c.cookies, updated)
	return nil
}

// Session (nil for SSR - no WebSocket session)
func (c *ssrContext) Session() *server.Session { return nil }
func (c *ssrContext) AuthSession() auth.Session {
	return nil
}
func (c *ssrContext) User() any                { return c.user }
func (c *ssrContext) SetUser(user any)         { c.user = user }
func (c *ssrContext) Principal() (auth.Principal, bool) {
	return auth.Principal{}, false
}
func (c *ssrContext) MustPrincipal() auth.Principal {
	panic("MustPrincipal called without authenticated principal")
}
func (c *ssrContext) RevalidateAuth() error { return nil }

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

func (c *ssrContext) Navigate(path string, opts ...server.NavigateOption) {
	fullPath, applied := server.BuildNavigateURL(path, opts...)
	if fullPath == "" {
		return
	}

	code := http.StatusFound // 302
	if applied.Replace {
		code = http.StatusSeeOther
	}
	c.Redirect(fullPath, code)
}

func (c *ssrContext) applyTo(w http.ResponseWriter) {
	if c == nil || w == nil {
		return
	}
	for key, values := range c.headers {
		if len(values) == 0 {
			continue
		}
		// Preserve multi-value headers when explicitly set.
		w.Header().Del(key)
		for _, v := range values {
			w.Header().Add(key, v)
		}
	}
	for _, cookie := range c.cookies {
		http.SetCookie(w, cookie)
	}
}

func (c *ssrContext) redirectInfo() (url string, code int, ok bool) {
	if !c.redirected {
		return "", 0, false
	}
	if c.redirectCode == 0 {
		return c.redirectURL, http.StatusFound, true
	}
	return c.redirectURL, c.redirectCode, true
}

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
