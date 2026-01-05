package router

import (
	"github.com/vango-go/vango/pkg/server"
	"github.com/vango-go/vango/pkg/vdom"
)

// Slot represents the child content passed to a layout.
// It is the rendered VNode tree of the wrapped page or nested layout.
type Slot = *vdom.VNode

// PageHandler handles a page request, returning a component to render.
type PageHandler func(ctx server.Ctx, params any) vdom.Component

// LayoutHandler wraps child content in a layout.
type LayoutHandler func(ctx server.Ctx, children Slot) *vdom.VNode

// APIHandler handles an API request, returning data or an error.
// The params parameter is a typed params struct, body is the decoded request body.
type APIHandler func(ctx server.Ctx, params any, body any) (any, error)

// ErrorHandler handles error pages.
type ErrorHandler func(ctx server.Ctx, err error) *vdom.VNode

// PageMeta contains page metadata for SEO.
type PageMeta struct {
	Title       string
	Description string
	Keywords    []string
	OGImage     string
	OGTitle     string
	OGDesc      string
	Canonical   string
	Robots      string
}

// ScannedRoute represents a route discovered by the scanner.
type ScannedRoute struct {
	// Path is the URL pattern (e.g., "/projects/:id")
	Path string

	// FilePath is the source file path
	FilePath string

	// Package is the Go package name
	Package string

	// Params are the route parameters
	Params []ParamDef

	// HasPage indicates the file exports a page handler function (e.g., IndexPage)
	HasPage bool

	// HandlerName is the actual name of the page handler function (e.g., "IndexPage", "ShowPage")
	HandlerName string

	// HasLayout indicates the file exports a Layout function
	HasLayout bool

	// HasMeta indicates the file exports a Meta function
	HasMeta bool

	// HasMiddleware indicates the file exports a Middleware variable or function
	HasMiddleware bool

	// Methods lists HTTP methods for API routes (GET, POST, etc.)
	Methods []string

	// IsAPI indicates this is an API route (returns JSON)
	IsAPI bool

	// IsCatchAll indicates this is a catch-all route ([...slug])
	IsCatchAll bool
}

// ParamDef defines a route parameter.
type ParamDef struct {
	// Name is the parameter name (e.g., "id")
	Name string

	// Type is the parameter type (e.g., "int", "string", "uuid")
	Type string

	// Segment is the original segment (e.g., "[id]", "[id:int]")
	Segment string
}

// MatchResult contains the result of matching a path against the router.
type MatchResult struct {
	// PageHandler is the handler for page routes
	PageHandler PageHandler

	// APIHandler is the handler for API routes
	APIHandler APIHandler

	// Layouts are the layout handlers in order (root to leaf)
	Layouts []LayoutHandler

	// Middleware is the combined middleware chain
	Middleware []Middleware

	// Params are the extracted route parameters
	Params map[string]string

	// Route is the matched route definition
	Route *ScannedRoute
}

// Middleware processes requests before they reach the handler.
type Middleware interface {
	// Handle processes the request and optionally calls next.
	// Return an error to stop the chain and report an error.
	// Return nil without calling next to stop the chain without error.
	Handle(ctx server.Ctx, next func() error) error
}

// MiddlewareFunc is a function adapter for Middleware.
type MiddlewareFunc func(ctx server.Ctx, next func() error) error

// Handle implements Middleware.
func (f MiddlewareFunc) Handle(ctx server.Ctx, next func() error) error {
	return f(ctx, next)
}
