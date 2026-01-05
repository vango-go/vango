package router

import (
	"errors"
	"testing"

	"github.com/vango-go/vango/pkg/server"
)

// mockCtx is a minimal implementation for testing
type mockCtx struct{}

func (m mockCtx) Request() interface{}                    { return nil }
func (m mockCtx) Path() string                            { return "" }
func (m mockCtx) Method() string                          { return "" }
func (m mockCtx) Query() interface{}                      { return nil }
func (m mockCtx) Param(key string) string                 { return "" }
func (m mockCtx) Header(key string) string                { return "" }
func (m mockCtx) Cookie(name string) (interface{}, error) { return nil, nil }
func (m mockCtx) Status(code int)                         {}
func (m mockCtx) Redirect(url string, code int)           {}
func (m mockCtx) SetHeader(key, value string)             {}
func (m mockCtx) SetCookie(cookie interface{})            {}
func (m mockCtx) Session() interface{}                    { return nil }
func (m mockCtx) User() interface{}                       { return nil }
func (m mockCtx) SetUser(user interface{})                {}
func (m mockCtx) Logger() interface{}                     { return nil }
func (m mockCtx) Done() <-chan struct{}                   { return nil }

func TestMiddlewareFuncHandle(t *testing.T) {
	called := false
	mw := MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
		called = true
		return next()
	})

	err := mw.Handle(nil, func() error { return nil })
	if err != nil {
		t.Errorf("Handle() error = %v", err)
	}
	if !called {
		t.Error("middleware was not called")
	}
}

func TestComposeMiddlewareEmpty(t *testing.T) {
	called := false
	handler := func() error {
		called = true
		return nil
	}

	err := ComposeMiddleware(nil, nil, handler)
	if err != nil {
		t.Errorf("ComposeMiddleware() error = %v", err)
	}
	if !called {
		t.Error("handler was not called")
	}
}

func TestComposeMiddlewareOrder(t *testing.T) {
	var order []string

	mw1 := MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
		order = append(order, "mw1-before")
		err := next()
		order = append(order, "mw1-after")
		return err
	})

	mw2 := MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
		order = append(order, "mw2-before")
		err := next()
		order = append(order, "mw2-after")
		return err
	})

	handler := func() error {
		order = append(order, "handler")
		return nil
	}

	err := ComposeMiddleware(nil, []Middleware{mw1, mw2}, handler)
	if err != nil {
		t.Errorf("ComposeMiddleware() error = %v", err)
	}

	expected := []string{
		"mw1-before",
		"mw2-before",
		"handler",
		"mw2-after",
		"mw1-after",
	}

	if len(order) != len(expected) {
		t.Fatalf("order = %v, want %v", order, expected)
	}
	for i := range order {
		if order[i] != expected[i] {
			t.Errorf("order[%d] = %q, want %q", i, order[i], expected[i])
		}
	}
}

func TestComposeMiddlewareShortCircuit(t *testing.T) {
	var order []string
	testErr := errors.New("short circuit")

	mw1 := MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
		order = append(order, "mw1-before")
		// Don't call next, return error
		return testErr
	})

	mw2 := MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
		order = append(order, "mw2-before")
		return next()
	})

	handler := func() error {
		order = append(order, "handler")
		return nil
	}

	err := ComposeMiddleware(nil, []Middleware{mw1, mw2}, handler)
	if err != testErr {
		t.Errorf("ComposeMiddleware() error = %v, want %v", err, testErr)
	}

	// mw2 and handler should not be called
	if len(order) != 1 || order[0] != "mw1-before" {
		t.Errorf("order = %v, want [mw1-before]", order)
	}
}

func TestChain(t *testing.T) {
	var order []string

	mw1 := MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
		order = append(order, "mw1")
		return next()
	})

	mw2 := MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
		order = append(order, "mw2")
		return next()
	})

	chain := Chain(mw1, mw2)

	err := chain.Handle(nil, func() error {
		order = append(order, "handler")
		return nil
	})

	if err != nil {
		t.Errorf("Chain.Handle() error = %v", err)
	}

	if len(order) != 3 {
		t.Errorf("order = %v, want [mw1 mw2 handler]", order)
	}
}

func TestSkip(t *testing.T) {
	mwCalled := false
	mw := MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
		mwCalled = true
		return next()
	})

	// Skip when condition is true
	skipMw := Skip(func(ctx server.Ctx) bool { return true }, mw)
	handlerCalled := false
	err := skipMw.Handle(nil, func() error {
		handlerCalled = true
		return nil
	})

	if err != nil {
		t.Errorf("Skip.Handle() error = %v", err)
	}
	if mwCalled {
		t.Error("middleware should have been skipped")
	}
	if !handlerCalled {
		t.Error("handler should have been called")
	}

	// Don't skip when condition is false
	mwCalled = false
	handlerCalled = false
	dontSkipMw := Skip(func(ctx server.Ctx) bool { return false }, mw)
	err = dontSkipMw.Handle(nil, func() error {
		handlerCalled = true
		return nil
	})

	if err != nil {
		t.Errorf("Skip.Handle() error = %v", err)
	}
	if !mwCalled {
		t.Error("middleware should have been called")
	}
	if !handlerCalled {
		t.Error("handler should have been called")
	}
}

func TestOnly(t *testing.T) {
	mwCalled := false
	mw := MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
		mwCalled = true
		return next()
	})

	// Run when condition is true
	onlyMw := Only(func(ctx server.Ctx) bool { return true }, mw)
	err := onlyMw.Handle(nil, func() error { return nil })

	if err != nil {
		t.Errorf("Only.Handle() error = %v", err)
	}
	if !mwCalled {
		t.Error("middleware should have been called")
	}

	// Skip when condition is false
	mwCalled = false
	dontRunMw := Only(func(ctx server.Ctx) bool { return false }, mw)
	err = dontRunMw.Handle(nil, func() error { return nil })

	if err != nil {
		t.Errorf("Only.Handle() error = %v", err)
	}
	if mwCalled {
		t.Error("middleware should not have been called")
	}
}
