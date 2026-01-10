// Package router implements file-based routing for Vango.
//
// The router provides:
//   - File-system based route discovery from app/routes/
//   - Radix tree for efficient route matching
//   - Parameter extraction with type coercion
//   - Layout composition and middleware chains
//   - Code generation for compile-time route registration
//
// # File Structure Convention
//
// Routes are defined by Go files in the app/routes/ directory:
//
//	app/routes/
//	├── index.go           → GET /
//	├── about.go           → GET /about
//	├── layout.go          → Layout for all routes
//	├── projects/
//	│   ├── index.go       → GET /projects
//	│   ├── [id].go        → GET /projects/:id
//	│   └── layout.go      → Layout for /projects/*
//	└── api/
//	    └── users.go       → GET/POST /api/users
//
// # Parameters
//
// Dynamic route segments are defined with brackets:
//
//	[id].go        → :id (string by default)
//	[id:int].go    → :id (parsed as int)
//	[...slug].go   → *slug (catch-all, []string)
//
// # Route Files
//
// Each route file can export specific functions:
//
//	func Page(ctx server.Ctx, params Params) vdom.Component  // Page handler
//	func Layout(ctx server.Ctx, children Slot) *vdom.VNode   // Layout wrapper
//	func Meta(ctx server.Ctx, params Params) PageMeta        // Page metadata
//	func Middleware() []Middleware                           // Route middleware
//	func GET(ctx server.Ctx, params Params) (any, error)     // API handlers
//	func POST(ctx server.Ctx, params Params, body T) (any, error)
//
// # Usage
//
//	scanner := router.NewScanner("app/routes")
//	routes, err := scanner.Scan()
//
//	r := router.NewRouter()
//	for _, route := range routes {
//	    r.AddPage(route.Path, pageHandler)
//	}
//
//	result, ok := r.Match("GET", "/projects/123")
//	if ok {
//	    // result.Params["id"] == "123"
//	    // result.PageHandler, result.Layouts, result.Middleware available
//	}
package router
