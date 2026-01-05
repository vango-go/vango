package router

import (
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
