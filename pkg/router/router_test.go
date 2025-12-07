package router

import (
	"testing"

	"github.com/vango-dev/vango/v2/pkg/server"
	"github.com/vango-dev/vango/v2/pkg/vdom"
)

func TestRouterAddPageAndMatch(t *testing.T) {
	r := NewRouter()

	called := false
	r.AddPage("/users", func(ctx server.Ctx, params any) vdom.Component {
		called = true
		return nil
	})

	result, ok := r.Match("GET", "/users")
	if !ok {
		t.Fatal("expected match for /users")
	}
	if result.PageHandler == nil {
		t.Fatal("expected PageHandler")
	}

	// Call handler to verify it's the right one
	result.PageHandler(nil, nil)
	if !called {
		t.Error("handler was not called")
	}
}

func TestRouterMatchParams(t *testing.T) {
	r := NewRouter()

	r.AddPage("/users/:id", func(ctx server.Ctx, params any) vdom.Component {
		return nil
	})

	result, ok := r.Match("GET", "/users/123")
	if !ok {
		t.Fatal("expected match for /users/123")
	}
	if result.Params["id"] != "123" {
		t.Errorf("params[id] = %q, want %q", result.Params["id"], "123")
	}
}

func TestRouterMatchCatchAll(t *testing.T) {
	r := NewRouter()

	r.AddPage("/files/*path", func(ctx server.Ctx, params any) vdom.Component {
		return nil
	})

	result, ok := r.Match("GET", "/files/a/b/c")
	if !ok {
		t.Fatal("expected match for /files/a/b/c")
	}
	if result.Params["path"] != "a/b/c" {
		t.Errorf("params[path] = %q, want %q", result.Params["path"], "a/b/c")
	}
}

func TestRouterNoMatch(t *testing.T) {
	r := NewRouter()

	r.AddPage("/users", func(ctx server.Ctx, params any) vdom.Component {
		return nil
	})

	_, ok := r.Match("GET", "/projects")
	if ok {
		t.Error("should not match /projects")
	}
}

func TestRouterAddLayout(t *testing.T) {
	r := NewRouter()

	// Add root layout
	r.AddLayout("/", func(ctx server.Ctx, children Slot) *vdom.VNode {
		return nil
	})

	// Add page
	r.AddPage("/users", func(ctx server.Ctx, params any) vdom.Component {
		return nil
	})

	result, ok := r.Match("GET", "/users")
	if !ok {
		t.Fatal("expected match for /users")
	}

	// Should include root layout AND users layout (from "/" and "/users" paths)
	// The tree collects layouts at each node visited
	if len(result.Layouts) < 1 {
		t.Errorf("len(Layouts) = %d, want at least 1", len(result.Layouts))
	}
}

func TestRouterNestedLayouts(t *testing.T) {
	r := NewRouter()

	// Add root layout
	r.AddLayout("/", func(ctx server.Ctx, children Slot) *vdom.VNode {
		return nil
	})

	// Add nested layout
	r.AddLayout("/users", func(ctx server.Ctx, children Slot) *vdom.VNode {
		return nil
	})

	// Add page
	r.AddPage("/users/list", func(ctx server.Ctx, params any) vdom.Component {
		return nil
	})

	result, ok := r.Match("GET", "/users/list")
	if !ok {
		t.Fatal("expected match for /users/list")
	}

	// Should include both root and users layouts (+ potentially intermediate)
	if len(result.Layouts) < 2 {
		t.Errorf("len(Layouts) = %d, want at least 2", len(result.Layouts))
	}
}

func TestRouterAddAPI(t *testing.T) {
	r := NewRouter()

	getCalled := false
	postCalled := false

	r.AddAPI("/users", "GET", func(ctx server.Ctx, params any, body any) (any, error) {
		getCalled = true
		return nil, nil
	})

	r.AddAPI("/users", "POST", func(ctx server.Ctx, params any, body any) (any, error) {
		postCalled = true
		return nil, nil
	})

	// Test GET
	result, ok := r.Match("GET", "/users")
	if !ok {
		t.Fatal("expected match for GET /users")
	}
	if result.APIHandler == nil {
		t.Fatal("expected APIHandler for GET")
	}
	result.APIHandler(nil, nil, nil)
	if !getCalled {
		t.Error("GET handler was not called")
	}

	// Test POST
	result, ok = r.Match("POST", "/users")
	if !ok {
		t.Fatal("expected match for POST /users")
	}
	if result.APIHandler == nil {
		t.Fatal("expected APIHandler for POST")
	}
	result.APIHandler(nil, nil, nil)
	if !postCalled {
		t.Error("POST handler was not called")
	}

	// Test unsupported method
	_, ok = r.Match("DELETE", "/users")
	if ok {
		t.Error("should not match DELETE /users")
	}
}

func TestRouterAddMiddleware(t *testing.T) {
	r := NewRouter()

	globalMw := MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
		return next()
	})

	r.Use(globalMw)
	r.AddPage("/users", func(ctx server.Ctx, params any) vdom.Component {
		return nil
	})

	result, ok := r.Match("GET", "/users")
	if !ok {
		t.Fatal("expected match for /users")
	}

	if len(result.Middleware) != 1 {
		t.Errorf("len(Middleware) = %d, want 1", len(result.Middleware))
	}
}

func TestRouterSetNotFound(t *testing.T) {
	r := NewRouter()

	r.SetNotFound(func(ctx server.Ctx, params any) vdom.Component {
		return nil
	})

	if r.NotFound() == nil {
		t.Error("NotFound() should not be nil")
	}
}

func TestRouterSetErrorPage(t *testing.T) {
	r := NewRouter()

	r.SetErrorPage(func(ctx server.Ctx, err error) *vdom.VNode {
		return nil
	})

	if r.ErrorPage() == nil {
		t.Error("ErrorPage() should not be nil")
	}
}

func TestRouterBuildFromScanned(t *testing.T) {
	r := NewRouter()

	routes := []ScannedRoute{
		{Path: "/", HasPage: true},
		{Path: "/about", HasPage: true},
		{Path: "/users/:id", HasPage: true, Params: []ParamDef{{Name: "id", Type: "int"}}},
	}

	pageCalled := make(map[string]bool)
	registry := &HandlerRegistry{
		Pages: map[string]PageHandler{
			"/": func(ctx server.Ctx, params any) vdom.Component {
				pageCalled["/"] = true
				return nil
			},
			"/about": func(ctx server.Ctx, params any) vdom.Component {
				pageCalled["/about"] = true
				return nil
			},
			"/users/:id": func(ctx server.Ctx, params any) vdom.Component {
				pageCalled["/users/:id"] = true
				return nil
			},
		},
	}

	r.BuildFromScanned(routes, registry)

	// Test each route
	for _, path := range []string{"/", "/about"} {
		result, ok := r.Match("GET", path)
		if !ok {
			t.Errorf("expected match for %s", path)
			continue
		}
		result.PageHandler(nil, nil)
		if !pageCalled[path] {
			t.Errorf("page handler for %s was not called", path)
		}
	}

	// Test parameterized route
	result, ok := r.Match("GET", "/users/123")
	if !ok {
		t.Fatal("expected match for /users/123")
	}
	result.PageHandler(nil, nil)
	if !pageCalled["/users/:id"] {
		t.Error("page handler for /users/:id was not called")
	}
	if result.Params["id"] != "123" {
		t.Errorf("params[id] = %q, want %q", result.Params["id"], "123")
	}
}

func TestRouterMultipleParams(t *testing.T) {
	r := NewRouter()

	r.AddPage("/users/:userId/posts/:postId", func(ctx server.Ctx, params any) vdom.Component {
		return nil
	})

	result, ok := r.Match("GET", "/users/42/posts/100")
	if !ok {
		t.Fatal("expected match")
	}

	if result.Params["userId"] != "42" {
		t.Errorf("params[userId] = %q, want %q", result.Params["userId"], "42")
	}
	if result.Params["postId"] != "100" {
		t.Errorf("params[postId] = %q, want %q", result.Params["postId"], "100")
	}
}

func TestRouterRootPath(t *testing.T) {
	r := NewRouter()

	r.AddPage("/", func(ctx server.Ctx, params any) vdom.Component {
		return nil
	})

	result, ok := r.Match("GET", "/")
	if !ok {
		t.Fatal("expected match for /")
	}
	if result.PageHandler == nil {
		t.Error("expected PageHandler")
	}
}
