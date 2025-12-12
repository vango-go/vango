// Package vtest provides testing helpers for Vango components.
//
// The vtest package reduces boilerplate when testing authenticated
// components by providing fluent context builders and render assertions.
//
// # Quick Start
//
//	func TestDashboard_Authenticated(t *testing.T) {
//	    ctx := vtest.NewCtx().WithUser(&User{Role: "admin"}).Build()
//	    comp, err := Dashboard(ctx)
//	    if err != nil {
//	        t.Fatalf("unexpected error: %v", err)
//	    }
//	    vtest.ExpectContains(t, comp, "Welcome")
//	}
//
// # Fluent Context Builder
//
// The context builder allows chaining multiple setup operations:
//
//	ctx := vtest.NewCtx().
//	    WithUser(&User{ID: "123", Role: "admin"}).
//	    WithData("theme", "dark").
//	    WithParam("id", "456").
//	    Build()
//
// # One-Liner Shorthand
//
// For simple auth tests, use the shorthand:
//
//	ctx := vtest.CtxWithUser(&User{ID: "123"})
//
// # Render Assertions
//
// Assert on rendered HTML output:
//
//	vtest.ExpectContains(t, comp, "Welcome Admin")
//	vtest.ExpectNotContains(t, comp, "Login")
//
// # Integration with Auth Package
//
// The vtest package integrates with the auth package for authenticated tests:
//
//	func TestProtectedRoute(t *testing.T) {
//	    // Without auth - should fail
//	    ctx := vtest.NewCtx().Build()
//	    _, err := ProtectedPage(ctx)
//	    if err != auth.ErrUnauthorized {
//	        t.Error("expected unauthorized")
//	    }
//
//	    // With auth - should succeed
//	    ctx = vtest.NewCtx().WithUser(&User{}).Build()
//	    _, err = ProtectedPage(ctx)
//	    if err != nil {
//	        t.Errorf("unexpected error: %v", err)
//	    }
//	}
package vtest
