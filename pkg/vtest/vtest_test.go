package vtest_test

import (
	"testing"

	"github.com/vango-go/vango/pkg/auth"
	"github.com/vango-go/vango/pkg/vdom"
	"github.com/vango-go/vango/pkg/vtest"
)

type TestUser struct {
	ID   string
	Role string
}

func TestNewCtx(t *testing.T) {
	ctx := vtest.NewCtx().Build()

	if ctx == nil {
		t.Fatal("expected non-nil context")
	}

	// Should have a session
	if ctx.Session() == nil {
		t.Error("expected session to be set")
	}
}

func TestNewCtx_WithUser(t *testing.T) {
	user := &TestUser{ID: "123", Role: "admin"}
	ctx := vtest.NewCtx().WithUser(user).Build()

	// Should be able to retrieve user via auth package
	retrieved, ok := auth.Get[*TestUser](ctx)
	if !ok {
		t.Fatal("expected user to be authenticated")
	}

	if retrieved.ID != "123" {
		t.Errorf("expected ID 123, got %s", retrieved.ID)
	}
	if retrieved.Role != "admin" {
		t.Errorf("expected role admin, got %s", retrieved.Role)
	}
}

func TestNewCtx_WithData(t *testing.T) {
	ctx := vtest.NewCtx().
		WithData("theme", "dark").
		WithData("count", 42).
		Build()

	session := ctx.Session()

	theme := session.GetString("theme")
	if theme != "dark" {
		t.Errorf("expected theme dark, got %s", theme)
	}

	count := session.GetInt("count")
	if count != 42 {
		t.Errorf("expected count 42, got %d", count)
	}
}

func TestCtxWithUser(t *testing.T) {
	user := &TestUser{ID: "456"}
	ctx := vtest.CtxWithUser(user)

	retrieved, ok := auth.Get[*TestUser](ctx)
	if !ok {
		t.Fatal("expected authenticated")
	}
	if retrieved.ID != "456" {
		t.Errorf("expected ID 456, got %s", retrieved.ID)
	}
}

func TestRenderToString(t *testing.T) {
	node := vdom.Div(
		vdom.Class("container"),
		vdom.H1(vdom.Text("Hello")),
		vdom.P(vdom.Text("World")),
	)

	html := vtest.RenderToString(node)

	if html == "" {
		t.Error("expected non-empty HTML")
	}

	// Should contain the elements
	if !contains(html, "container") {
		t.Error("expected class container")
	}
	if !contains(html, "Hello") {
		t.Error("expected Hello")
	}
	if !contains(html, "World") {
		t.Error("expected World")
	}
}

func TestExpectContains_Pass(t *testing.T) {
	node := vdom.Div(vdom.Text("Hello World"))

	// This should pass (no error)
	mockT := &testing.T{}
	vtest.ExpectContains(mockT, node, "Hello")

	if mockT.Failed() {
		t.Error("ExpectContains should have passed")
	}
}

func TestExpectNotContains_Pass(t *testing.T) {
	node := vdom.Div(vdom.Text("Hello World"))

	mockT := &testing.T{}
	vtest.ExpectNotContains(mockT, node, "Goodbye")

	if mockT.Failed() {
		t.Error("ExpectNotContains should have passed")
	}
}

func TestChainedBuilder(t *testing.T) {
	user := &TestUser{ID: "chain"}
	ctx := vtest.NewCtx().
		WithUser(user).
		WithData("a", 1).
		WithData("b", 2).
		WithParam("id", "test").
		Build()

	// Verify all was set
	retrieved, _ := auth.Get[*TestUser](ctx)
	if retrieved.ID != "chain" {
		t.Error("user not set")
	}

	session := ctx.Session()
	if session.GetInt("a") != 1 || session.GetInt("b") != 2 {
		t.Error("data not set")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
