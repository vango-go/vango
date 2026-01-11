package vango

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/vango-go/vango/pkg/render"
	"github.com/vango-go/vango/pkg/router"
	"github.com/vango-go/vango/pkg/server"
	"github.com/vango-go/vango/pkg/vdom"
)

// =============================================================================
// App Type
// =============================================================================

// App is the main Vango application entry point.
// It wraps the server, router, and static file serving into a single http.Handler.
//
// Create an App with vango.New():
//
//	app := vango.New(vango.Config{
//	    Session: vango.SessionConfig{ResumeWindow: 30 * time.Second},
//	    Static:  vango.StaticConfig{Dir: "public", Prefix: "/"},
//	    DevMode: os.Getenv("ENV") != "production",
//	})
//
//	routes.Register(app)
//	http.ListenAndServe(":8080", app)
type App struct {
	// Internal components
	server *server.Server
	router *router.Router

	// Static file serving
	staticDir    string
	staticPrefix string
	staticFS     http.FileSystem

	// Configuration
	config Config
	logger *slog.Logger
}

// New creates a new Vango application with the given configuration.
func New(cfg Config) *App {
	// Apply defaults
	if cfg.Session.ResumeWindow == 0 {
		cfg.Session.ResumeWindow = DefaultSessionConfig().ResumeWindow
	}
	if cfg.Static.Prefix == "" {
		cfg.Static.Prefix = "/"
	}
	if cfg.API.MaxBodyBytes == 0 {
		cfg.API.MaxBodyBytes = DefaultAPIConfig().MaxBodyBytes
	}

	// Convert to internal server config
	serverCfg := buildServerConfig(cfg)

	// Set up logger
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	// Create the app
	app := &App{
		server:       server.New(serverCfg),
		router:       router.NewRouter(),
		staticDir:    cfg.Static.Dir,
		staticPrefix: cfg.Static.Prefix,
		config:       cfg,
		logger:       logger,
	}

	// Wire router to server for navigation support
	app.server.SetRouter(router.NewRouterAdapter(app.router))

	// Set up static file system if configured
	if cfg.Static.Dir != "" {
		app.staticFS = http.Dir(cfg.Static.Dir)
	}

	return app
}

// =============================================================================
// http.Handler Implementation
// =============================================================================

// ServeHTTP implements http.Handler.
// It routes requests to static files, the WebSocket endpoint, or page routes.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Check for static files first (if configured)
	if a.staticFS != nil && a.shouldServeStatic(path) {
		a.serveStatic(w, r)
		return
	}

	// WebSocket paths go to the internal server
	if strings.HasPrefix(path, "/_vango/") {
		a.server.ServeHTTP(w, r)
		return
	}

	// Try to match a route
	match, found := a.router.Match(r.Method, path)
	if !found {
		http.NotFound(w, r)
		return
	}

	// Handle page routes (GET only for SSR)
	if match.PageHandler != nil && r.Method == http.MethodGet {
		a.renderPage(w, r, match)
		return
	}

	// Handle API routes
	if match.APIHandler != nil {
		a.handleAPI(w, r, match)
		return
	}

	// No handler found
	http.NotFound(w, r)
}

// renderPage renders a page for SSR (HTTP GET requests).
func (a *App) renderPage(w http.ResponseWriter, r *http.Request, match *router.MatchResult) {
	// Create SSR context
	ctx := newSSRContext(w, r, match.Params, a.config, a.logger)

	var result *vdom.VNode
	ranFinal, mwErr := server.RunRouteMiddleware(ctx, match.GetMiddleware(), func() error {
		// Call page handler to get component
		component := match.PageHandler(ctx, match.Params)
		if component == nil {
			return http.ErrAbortHandler
		}

		// Render component to VNode
		pageNode := component.Render()

		// Determine which layouts to use:
		// - If page has explicit layouts (HasPageLayouts), use those only (no inheritance)
		// - Otherwise, use hierarchical layouts from app.Layout() calls
		layouts := match.Layouts
		if match.HasPageLayouts {
			layouts = match.PageLayouts
		}

		// Apply layouts (innermost to outermost)
		// Layouts[0] is root, Layouts[len-1] is closest to page
		// We apply from end to start: page → inner layout → ... → root layout
		result = pageNode
		for i := len(layouts) - 1; i >= 0; i-- {
			result = layouts[i](ctx, result)
		}
		return nil
	})

	// Middleware/handlers can trigger redirects for SSR via ctx.Navigate()/ctx.Redirect().
	if url, code, ok := ctx.redirectInfo(); ok {
		ctx.applyTo(w)
		http.Redirect(w, r, url, code)
		return
	}

	if mwErr != nil {
		if a.router.ErrorPage() != nil {
			// Prefer the configured error page if present.
			if ctx.status == http.StatusOK {
				ctx.status = http.StatusInternalServerError
			}
			errNode := a.router.ErrorPage()(ctx, mwErr)
			if errNode != nil {
				layouts := match.Layouts
				if match.HasPageLayouts {
					layouts = match.PageLayouts
				}
				for i := len(layouts) - 1; i >= 0; i-- {
					errNode = layouts[i](ctx, errNode)
				}

				renderer := render.NewRenderer(render.RendererConfig{
					Pretty: a.config.DevMode,
				})
				html, err := renderer.RenderToString(errNode)
				if err == nil {
					ctx.applyTo(w)
					if w.Header().Get("Content-Type") == "" {
						w.Header().Set("Content-Type", "text/html; charset=utf-8")
					}
					w.WriteHeader(ctx.status)
					w.Write([]byte("<!DOCTYPE html>\n"))
					w.Write([]byte(html))
					return
				}
			}
		}

		a.logger.Error("middleware failed", "error", mwErr)
		http.Error(w, mwErr.Error(), http.StatusInternalServerError)
		return
	}

	if !ranFinal {
		// Middleware short-circuited without rendering a page. Respect status/cookies/headers.
		ctx.applyTo(w)
		w.WriteHeader(ctx.status)
		return
	}

	if result == nil {
		http.Error(w, "Page returned nil", http.StatusInternalServerError)
		return
	}

	// Render to HTML
	renderer := render.NewRenderer(render.RendererConfig{
		Pretty: a.config.DevMode,
	})

	html, err := renderer.RenderToString(result)
	if err != nil {
		a.logger.Error("render failed", "error", err)
		http.Error(w, "Render error", http.StatusInternalServerError)
		return
	}

	ctx.applyTo(w)
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	}
	w.WriteHeader(ctx.status)
	w.Write([]byte("<!DOCTYPE html>\n"))
	w.Write([]byte(html))
}

// handleAPI handles API routes, returning JSON responses.
func (a *App) handleAPI(w http.ResponseWriter, r *http.Request, match *router.MatchResult) {
	ctx := newSSRContext(w, r, match.Params, a.config, a.logger)

	body := apiRawBody{
		ContentType:           r.Header.Get("Content-Type"),
		StrictJSONContentType: a.config.API.RequireJSONContentType,
	}

	var out any
	ranFinal, mwErr := server.RunRouteMiddleware(ctx, match.GetMiddleware(), func() error {
		if shouldReadAPIRequestBody(r) {
			raw, err := readAPIRequestBody(w, r, a.config.API.MaxBodyBytes)
			if err != nil {
				return err
			}
			body.Bytes = raw
		}

		var err error
		out, err = match.APIHandler(ctx, match.Params, body)
		return err
	})

	if url, code, ok := ctx.redirectInfo(); ok {
		ctx.applyTo(w)
		http.Redirect(w, r, url, code)
		return
	}

	if mwErr != nil {
		ctx.applyTo(w)
		if sc, ok := mwErr.(interface{ StatusCode() int }); ok {
			ctx.status = sc.StatusCode()
		}
		w.Header().Set("Content-Type", "application/json")
		if ctx.status == http.StatusOK {
			ctx.status = http.StatusInternalServerError
		}
		w.WriteHeader(ctx.status)
		json.NewEncoder(w).Encode(map[string]string{"error": mwErr.Error()})
		return
	}

	if !ranFinal {
		// Middleware short-circuited without producing a response body.
		ctx.applyTo(w)
		if ctx.status == http.StatusOK {
			ctx.status = http.StatusNoContent
		}
		w.WriteHeader(ctx.status)
		return
	}

	ctx.applyTo(w)
	if outWriter, ok := out.(interface {
		Write(http.ResponseWriter) error
	}); ok {
		if err := outWriter.Write(w); err != nil {
			a.logger.Error("API response write failed", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	if ctx.status == http.StatusNoContent {
		w.WriteHeader(ctx.status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(ctx.status)
	json.NewEncoder(w).Encode(out)
}

func shouldReadAPIRequestBody(r *http.Request) bool {
	if r == nil || r.Body == nil {
		return false
	}
	if r.ContentLength > 0 {
		return true
	}
	if r.Header.Get("Transfer-Encoding") != "" {
		return true
	}
	switch r.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func readAPIRequestBody(w http.ResponseWriter, r *http.Request, maxBytes int64) ([]byte, error) {
	if r == nil || r.Body == nil {
		return nil, nil
	}
	if maxBytes <= 0 {
		maxBytes = DefaultAPIConfig().MaxBodyBytes
	}

	limited := http.MaxBytesReader(w, r.Body, maxBytes)
	raw, err := io.ReadAll(limited)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return nil, &HTTPError{
				Code:    http.StatusRequestEntityTooLarge,
				Message: "request body too large",
				Err:     err,
			}
		}
		return nil, &HTTPError{
			Code:    http.StatusBadRequest,
			Message: "invalid request body",
			Err:     err,
		}
	}

	// Restore body for downstream consumers (middleware/handlers reading ctx.Request().Body).
	r.Body = io.NopCloser(bytes.NewReader(raw))
	return raw, nil
}

// Handler returns the App as an http.Handler.
// This is useful for explicit type conversion or middleware wrapping.
func (a *App) Handler() http.Handler {
	return a
}

// =============================================================================
// Route Registration
// =============================================================================

// Page registers a page handler with optional layouts.
// The handler is called when a user navigates to the path.
//
// Two handler signatures are supported:
//
//	// Static page (no route parameters)
//	func IndexPage(ctx vango.Ctx) *vango.VNode
//
//	// Dynamic page (with typed parameters)
//	type ShowParams struct {
//	    ID int `param:"id"`
//	}
//	func ShowPage(ctx vango.Ctx, p ShowParams) *vango.VNode
//
// Layouts wrap the page content. They are applied in order (root to leaf):
//
//	app.Page("/projects/:id", projects.ShowPage, RootLayout, ProjectsLayout)
func (a *App) Page(path string, handler PageHandler, layouts ...LayoutHandler) {
	wrappedHandler := wrapPageHandler(handler)

	if len(layouts) > 0 {
		// Page has explicit layouts - store them separately (no inheritance)
		wrappedLayouts := make([]router.LayoutHandler, 0, len(layouts))
		for _, layout := range layouts {
			if layout == nil {
				continue
			}
			wrappedLayouts = append(wrappedLayouts, wrapLayoutHandler(layout))
		}
		a.router.Page(path, wrappedHandler, router.WithPageLayouts(wrappedLayouts...))
	} else {
		// No explicit layouts - will use hierarchical layouts from app.Layout()
		a.router.Page(path, wrappedHandler)
	}
}

// API registers an API handler for the given HTTP method and path.
// API handlers return JSON responses.
//
// Multiple handler signatures are supported:
//
//	// Simple handler (no params or body)
//	func HealthGET(ctx vango.Ctx) (*HealthResponse, error)
//
//	// With route parameters
//	func UserGET(ctx vango.Ctx, p UserParams) (*User, error)
//
//	// With request body
//	func UserPOST(ctx vango.Ctx, body CreateUserRequest) (*User, error)
//
//	// With both parameters and body
//	func UserPUT(ctx vango.Ctx, p UserParams, body UpdateRequest) (*User, error)
func (a *App) API(method, path string, handler APIHandler) {
	a.router.API(method, path, wrapAPIHandler(handler))
}

// Layout registers a layout handler for a path.
// Layouts registered separately apply to all pages under that path.
//
//	app.Layout("/admin", AdminLayout)
//	app.Page("/admin/dashboard", DashboardPage) // Uses AdminLayout
//	app.Page("/admin/users", UsersPage)         // Uses AdminLayout
func (a *App) Layout(path string, handler LayoutHandler) {
	a.router.Layout(path, wrapLayoutHandler(handler))
}

// Middleware registers route middleware for a path.
// Middleware runs before the page handler and can:
//
//   - Redirect (e.g., authentication checks)
//
//   - Add data to context
//
//   - Short-circuit the request
//
//     app.Middleware("/admin", authMiddleware)
func (a *App) Middleware(path string, mw ...RouteMiddleware) {
	a.router.Middleware(path, mw...)
}

// Use adds global middleware that applies to all routes.
//
//	app.Use(loggingMiddleware, rateLimitMiddleware)
func (a *App) Use(mw ...RouteMiddleware) {
	a.router.Use(mw...)
}

// SetNotFound sets the handler for 404 pages.
//
//	app.SetNotFound(func(ctx vango.Ctx) *vango.VNode {
//	    return Div(H1(Text("Page Not Found")))
//	})
func (a *App) SetNotFound(handler PageHandler) {
	a.router.SetNotFound(wrapPageHandler(handler))
}

// SetErrorPage sets the handler for error pages.
//
//	app.SetErrorPage(func(ctx vango.Ctx, err error) *vango.VNode {
//	    return Div(H1(Textf("Error: %v", err)))
//	})
func (a *App) SetErrorPage(handler func(Ctx, error) *VNode) {
	a.router.SetErrorPage(func(ctx server.Ctx, err error) *vdom.VNode {
		return handler(ctx, err)
	})
}

// =============================================================================
// Server Access
// =============================================================================

// Server returns the underlying server for advanced configuration.
// Most apps won't need this.
func (a *App) Server() *server.Server {
	return a.server
}

// Router returns the underlying router for advanced configuration.
// Most apps won't need this.
func (a *App) Router() *router.Router {
	return a.router
}

// Config returns the app configuration.
func (a *App) Config() Config {
	return a.config
}

// Run starts the server and blocks until shutdown.
// This is a convenience method equivalent to http.ListenAndServe with graceful shutdown.
//
//	app := vango.New(cfg)
//	routes.Register(app)
//	app.Run(":8080")
func (a *App) Run(addr string) error {
	a.server.Config().Address = addr
	return a.server.Run()
}
