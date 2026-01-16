package vango

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vango-go/vango/pkg/router"
	"github.com/vango-go/vango/pkg/server"
	"github.com/vango-go/vango/pkg/vdom"
)

// =============================================================================
// App Getter Methods Tests
// =============================================================================

func TestApp_Handler_ReturnsSelf(t *testing.T) {
	app := New(DefaultConfig())

	handler := app.Handler()

	if handler == nil {
		t.Fatal("Handler() returned nil")
	}
	// Handler() should return the App itself as http.Handler
	if handler != http.Handler(app) {
		t.Fatal("Handler() should return the App itself")
	}
}

func TestApp_Server_ReturnsNonNil(t *testing.T) {
	app := New(DefaultConfig())

	srv := app.Server()

	if srv == nil {
		t.Fatal("Server() returned nil")
	}
}

func TestApp_Router_ReturnsNonNil(t *testing.T) {
	app := New(DefaultConfig())

	rtr := app.Router()

	if rtr == nil {
		t.Fatal("Router() returned nil")
	}
}

func TestApp_Config_ReturnsConfiguration(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DevMode = true
	cfg.API.MaxBodyBytes = 12345

	app := New(cfg)
	got := app.Config()

	if !got.DevMode {
		t.Error("Config().DevMode should be true")
	}
	if got.API.MaxBodyBytes != 12345 {
		t.Errorf("Config().API.MaxBodyBytes = %d, want %d", got.API.MaxBodyBytes, 12345)
	}
}

func TestApp_SetNotFound_HandlesUnknownPaths(t *testing.T) {
	app := New(DefaultConfig())
	app.SetNotFound(func(ctx Ctx) *VNode {
		ctx.Status(http.StatusNotFound)
		return vdom.Div(vdom.Text("Custom 404"))
	})
	app.Page("/exists", func(ctx Ctx) *VNode {
		return vdom.Text("exists")
	})

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	rr := httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	// SetNotFound should handle the 404 page
	// Note: The default behavior without SetNotFound is http.NotFound which returns 404
	// With SetNotFound, the custom handler should be called
	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestApp_SetErrorPage_HandlesErrors(t *testing.T) {
	app := New(DefaultConfig())
	app.SetErrorPage(func(ctx Ctx, err error) *VNode {
		return vdom.Div(vdom.Text("Error: " + err.Error()))
	})

	// Verify SetErrorPage was called without panicking
	// The error page is tested more thoroughly in app_render_test.go
}

func TestApp_Page_RegistersRoute(t *testing.T) {
	app := New(DefaultConfig())
	app.Page("/test", func(ctx Ctx) *VNode {
		return vdom.Text("test page")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestApp_API_RegistersRoute(t *testing.T) {
	app := New(DefaultConfig())
	app.API(http.MethodGet, "/api/health", func(ctx Ctx) (any, error) {
		return map[string]string{"status": "ok"}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rr := httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestApp_Layout_RegistersLayout(t *testing.T) {
	app := New(DefaultConfig())
	layoutCalled := false
	app.Layout("/", func(ctx Ctx, children Slot) *VNode {
		layoutCalled = true
		return vdom.Div(children)
	})
	app.Page("/test", func(ctx Ctx) *VNode {
		return vdom.Text("content")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	if !layoutCalled {
		t.Error("Layout handler was not called")
	}
}

func TestApp_Use_AddsGlobalMiddleware(t *testing.T) {
	app := New(DefaultConfig())
	middlewareCalled := false
	app.Use(router.MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
		middlewareCalled = true
		return next()
	}))
	app.Page("/test", func(ctx Ctx) *VNode {
		return vdom.Text("content")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	if !middlewareCalled {
		t.Error("Global middleware was not called")
	}
}

func TestApp_Middleware_AddsPathMiddleware(t *testing.T) {
	app := New(DefaultConfig())
	middlewareCalled := false
	app.Middleware("/admin", router.MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
		middlewareCalled = true
		return next()
	}))
	app.Page("/admin/dashboard", func(ctx Ctx) *VNode {
		return vdom.Text("dashboard")
	})
	app.Page("/public", func(ctx Ctx) *VNode {
		return vdom.Text("public")
	})

	// Request to /admin/dashboard should trigger middleware
	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard", nil)
	rr := httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	if !middlewareCalled {
		t.Error("Path middleware was not called for /admin/dashboard")
	}

	// Request to /public should NOT trigger the /admin middleware
	middlewareCalled = false
	req = httptest.NewRequest(http.MethodGet, "/public", nil)
	rr = httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	if middlewareCalled {
		t.Error("Path middleware should not be called for /public")
	}
}
