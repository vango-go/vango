package router

import (
	"strings"

	"github.com/vango-go/vango/pkg/server"
)

// Router manages route matching and handler dispatch.
type Router struct {
	root       *RouteNode
	notFound   PageHandler
	errorPage  ErrorHandler
	middleware []Middleware
}

// NewRouter creates a new router.
func NewRouter() *Router {
	return &Router{
		root: newRouteNode(""),
	}
}

// AddPage registers a page handler for a path.
func (r *Router) AddPage(path string, handler PageHandler) {
	node := r.root.insertRoute(path)
	node.pageHandler = handler
}

// AddLayout registers a layout handler for a path.
func (r *Router) AddLayout(path string, handler LayoutHandler) {
	node := r.root.insertRoute(path)
	node.layoutHandler = handler
}

// AddAPI registers an API handler for a path and method.
func (r *Router) AddAPI(path, method string, handler APIHandler) {
	node := r.root.insertRoute(path)
	if node.apiHandlers == nil {
		node.apiHandlers = make(map[string]APIHandler)
	}
	node.apiHandlers[method] = handler
}

// AddMiddleware adds middleware to a specific path.
func (r *Router) AddMiddleware(path string, mw ...Middleware) {
	node := r.root.insertRoute(path)
	node.middleware = append(node.middleware, mw...)
}

// =============================================================================
// Spec-compliant aliases (per VANGO_ARCHITECTURE_AND_GUIDE.md §9.1.4)
// =============================================================================

// Page registers a page handler for a path.
// This is the spec-compliant alias for AddPage.
// Optional RouteOption arguments can specify parameter type constraints.
//
// Example:
//
//	r.Page("/users/:id", users.ShowPage)
//	r.Page("/users/:id", users.ShowPage, router.WithParamType("id", "int"))
func (r *Router) Page(path string, handler PageHandler, opts ...RouteOption) {
	node := r.root.insertRoute(path)
	node.pageHandler = handler
	r.applyRouteOptions(path, opts)
}

// Layout registers a layout handler for a path.
// This is the spec-compliant alias for AddLayout.
func (r *Router) Layout(path string, handler LayoutHandler) {
	r.AddLayout(path, handler)
}

// API registers an API handler for a method and path.
// This is the spec-compliant alias for AddAPI.
// Note: method comes first to match natural reading order ("GET /api/health").
//
// Example:
//
//	r.API("GET", "/api/health", api.HealthGET)
//	r.API("POST", "/api/users", api.UsersPOST)
func (r *Router) API(method, path string, handler APIHandler) {
	r.AddAPI(path, method, handler)
}

// Middleware adds middleware to a specific path.
// This is the spec-compliant alias for AddMiddleware.
func (r *Router) Middleware(path string, mw ...Middleware) {
	r.AddMiddleware(path, mw...)
}

// RouteOption configures route registration.
// Used for param type constraints, etc.
type RouteOption func(*routeOptions)

type routeOptions struct {
	paramTypes map[string]string
}

// WithParamType specifies the type constraint for a route parameter.
// This allows specifying the type separately from the path pattern.
//
// Example:
//
//	r.Page("/users/:id", users.ShowPage, router.WithParamType("id", "int"))
//
// This is equivalent to using the inline type syntax:
//
//	r.Page("/users/:id:int", users.ShowPage)
func WithParamType(param, typ string) RouteOption {
	return func(o *routeOptions) {
		if o.paramTypes == nil {
			o.paramTypes = make(map[string]string)
		}
		o.paramTypes[param] = typ
	}
}

// applyRouteOptions applies route options to the route tree.
// It traverses the path from the root to find and update parameter nodes.
func (r *Router) applyRouteOptions(path string, opts []RouteOption) {
	if len(opts) == 0 {
		return
	}

	// Collect options
	var options routeOptions
	for _, opt := range opts {
		opt(&options)
	}

	if len(options.paramTypes) == 0 {
		return
	}

	// Traverse the tree following the path to find param nodes
	segments := splitPath(path)
	current := r.root

	for _, seg := range segments {
		if strings.HasPrefix(seg, "*") {
			// Catch-all - check if we have a type (though catch-all is always []string)
			if current.catchAllChild != nil {
				name := seg[1:]
				if typ, ok := options.paramTypes[name]; ok {
					current.catchAllChild.paramType = typ
				}
			}
			break
		} else if strings.HasPrefix(seg, ":") {
			// Parameter segment - this is where we apply the type
			if current.paramChild != nil {
				// Extract param name (without type suffix if present)
				name := seg[1:]
				if idx := strings.Index(name, ":"); idx != -1 {
					name = name[:idx]
				}
				if typ, ok := options.paramTypes[name]; ok {
					current.paramChild.paramType = typ
				}
				current = current.paramChild
			}
		} else {
			// Static segment
			if child := current.findChild(seg); child != nil {
				current = child
			}
		}
	}
}

// Use adds global middleware that applies to all routes.
func (r *Router) Use(mw ...Middleware) {
	r.middleware = append(r.middleware, mw...)
}

// SetNotFound sets the 404 handler.
func (r *Router) SetNotFound(handler PageHandler) {
	r.notFound = handler
}

// SetErrorPage sets the error page handler.
func (r *Router) SetErrorPage(handler ErrorHandler) {
	r.errorPage = handler
}

// Match finds the handler for a path.
func (r *Router) Match(method, path string) (*MatchResult, bool) {
	params := make(map[string]string)

	// Initialize match context with global middleware
	ctx := &matchContext{
		layouts:    nil,
		middleware: append([]Middleware{}, r.middleware...), // Copy global middleware
	}

	// Collect root layout and middleware if present
	if r.root.layoutHandler != nil {
		ctx.layouts = append(ctx.layouts, r.root.layoutHandler)
	}
	if len(r.root.middleware) > 0 {
		ctx.middleware = append(ctx.middleware, r.root.middleware...)
	}

	// Match against tree
	node, matchCtx, ok := r.root.match(splitPath(path), params, ctx)
	if !ok {
		return nil, false
	}

	result := &MatchResult{
		Params:     params,
		Layouts:    matchCtx.layouts,
		Middleware: matchCtx.middleware,
	}

	// Check for API handler
	if node.apiHandlers != nil {
		if handler, exists := node.apiHandlers[method]; exists {
			result.APIHandler = handler
			return result, true
		}
		// Method exists but not this one - could return 405
		// For now, return no match
		return nil, false
	}

	// Check for page handler
	if node.pageHandler != nil {
		result.PageHandler = node.pageHandler
		return result, true
	}

	return nil, false
}

// NotFound returns the 404 handler.
func (r *Router) NotFound() PageHandler {
	return r.notFound
}

// ErrorPage returns the error page handler.
func (r *Router) ErrorPage() ErrorHandler {
	return r.errorPage
}

// RouterAdapter wraps Router to implement server.Router interface.
// This adapter is needed because the server package defines its own
// PageHandler type to avoid import cycles.
type RouterAdapter struct {
	*Router
}

// NewRouterAdapter creates a new adapter for use with server.Session.SetRouter().
func NewRouterAdapter(r *Router) *RouterAdapter {
	return &RouterAdapter{Router: r}
}

// Match implements server.Router interface.
func (a *RouterAdapter) Match(method, path string) (server.RouteMatch, bool) {
	match, ok := a.Router.Match(method, path)
	if !ok {
		return nil, false
	}
	return match, true
}

// NotFound implements server.Router interface.
func (a *RouterAdapter) NotFound() server.PageHandler {
	if a.Router.notFound == nil {
		return nil
	}
	return func(ctx server.Ctx, params any) server.Component {
		paramsMap, _ := params.(map[string]string)
		return a.Router.notFound(ctx, paramsMap)
	}
}

// ServeHTTP implements http.Handler for the router.
// This provides basic HTTP routing without WebSocket features.
func (r *Router) ServeHTTP(ctx server.Ctx) (*MatchResult, bool) {
	return r.Match(ctx.Method(), ctx.Path())
}

// BuildFromScanned builds the router from scanned routes and a handler registry.
// The registry maps file paths to handler functions.
type HandlerRegistry struct {
	Pages   map[string]PageHandler
	Layouts map[string]LayoutHandler
	APIs    map[string]map[string]APIHandler // path -> method -> handler
	MW      map[string][]Middleware
}

// BuildFromScanned populates the router from scanned routes.
func (r *Router) BuildFromScanned(routes []ScannedRoute, registry *HandlerRegistry) {
	for _, route := range routes {
		if route.HasLayout && registry.Layouts != nil {
			if handler, ok := registry.Layouts[route.Path]; ok {
				r.AddLayout(route.Path, handler)
			}
		}

		if route.HasPage && registry.Pages != nil {
			if handler, ok := registry.Pages[route.Path]; ok {
				r.AddPage(route.Path, handler)
			}
		}

		if len(route.Methods) > 0 && registry.APIs != nil {
			if handlers, ok := registry.APIs[route.Path]; ok {
				for method, handler := range handlers {
					r.AddAPI(route.Path, method, handler)
				}
			}
		}

		if route.HasMiddleware && registry.MW != nil {
			if mw, ok := registry.MW[route.Path]; ok {
				r.AddMiddleware(route.Path, mw...)
			}
		}
	}
}
