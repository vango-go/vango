package vtest

import (
	"strings"
	"testing"

	"github.com/vango-go/vango/pkg/auth"
	"github.com/vango-go/vango/pkg/render"
	"github.com/vango-go/vango/pkg/server"
	"github.com/vango-go/vango/pkg/vdom"
)

// CtxBuilder allows fluent construction of test contexts.
type CtxBuilder struct {
	session *server.Session
	ctx     server.Ctx
	params  map[string]string
}

// NewCtx creates a new context builder for testing.
//
// Example:
//
//	ctx := vtest.NewCtx().
//	    WithUser(&User{ID: "123"}).
//	    WithData("theme", "dark").
//	    Build()
func NewCtx() *CtxBuilder {
	s := server.NewMockSession()
	return &CtxBuilder{
		session: s,
		ctx:     server.NewTestContext(s),
		params:  make(map[string]string),
	}
}

// WithUser injects an authenticated user into the session.
// Uses the auth package's SessionKey for storage.
//
// Example:
//
//	ctx := vtest.NewCtx().WithUser(&User{Role: "admin"}).Build()
func (b *CtxBuilder) WithUser(user any) *CtxBuilder {
	auth.Set(b.session, user)
	return b
}

// WithData injects arbitrary data into the session.
//
// Example:
//
//	ctx := vtest.NewCtx().WithData("cart_count", 5).Build()
func (b *CtxBuilder) WithData(key string, val any) *CtxBuilder {
	b.session.Set(key, val)
	return b
}

// WithParam sets a route parameter.
//
// Example:
//
//	ctx := vtest.NewCtx().WithParam("id", "123").Build()
func (b *CtxBuilder) WithParam(key, value string) *CtxBuilder {
	b.params[key] = value
	return b
}

// Build returns the final context for use in tests.
func (b *CtxBuilder) Build() server.Ctx {
	return b.ctx
}

// CtxWithUser is a shorthand for NewCtx().WithUser(user).Build()
//
// Example:
//
//	ctx := vtest.CtxWithUser(&User{ID: "123"})
func CtxWithUser(user any) server.Ctx {
	return NewCtx().WithUser(user).Build()
}

// RenderToString renders a VNode and returns the HTML string.
// This is useful for asserting on rendered output.
//
// Example:
//
//	html := vtest.RenderToString(MyComponent())
//	if !strings.Contains(html, "expected text") {
//	    t.Error("missing expected text")
//	}
func RenderToString(node *vdom.VNode) string {
	r := render.NewRenderer(render.RendererConfig{})
	html, err := r.RenderToString(node)
	if err != nil {
		return ""
	}
	return html
}

// ExpectContains asserts that rendered output contains expected substring.
//
// Example:
//
//	vtest.ExpectContains(t, comp.Render(), "Welcome Admin")
func ExpectContains(t *testing.T, node *vdom.VNode, expected string) {
	t.Helper()
	html := RenderToString(node)
	if !strings.Contains(html, expected) {
		t.Errorf("expected rendered output to contain %q, got:\n%s", expected, truncate(html, 500))
	}
}

// ExpectNotContains asserts that rendered output does not contain substring.
//
// Example:
//
//	vtest.ExpectNotContains(t, comp.Render(), "Error")
func ExpectNotContains(t *testing.T, node *vdom.VNode, unexpected string) {
	t.Helper()
	html := RenderToString(node)
	if strings.Contains(html, unexpected) {
		t.Errorf("expected rendered output to NOT contain %q, got:\n%s", unexpected, truncate(html, 500))
	}
}

// ExpectElement asserts that rendered output contains a specific tag.
//
// Example:
//
//	vtest.ExpectElement(t, comp.Render(), "button")
func ExpectElement(t *testing.T, node *vdom.VNode, tag string) {
	t.Helper()
	html := RenderToString(node)
	if !strings.Contains(html, "<"+tag) {
		t.Errorf("expected rendered output to contain <%s> element, got:\n%s", tag, truncate(html, 500))
	}
}

// ExpectAttribute asserts that rendered output contains an attribute value.
//
// Example:
//
//	vtest.ExpectAttribute(t, comp.Render(), "class", "btn-primary")
func ExpectAttribute(t *testing.T, node *vdom.VNode, attr, value string) {
	t.Helper()
	html := RenderToString(node)
	needle := attr + `="` + value + `"`
	if !strings.Contains(html, needle) {
		t.Errorf("expected attribute %s=%q not found, got:\n%s", attr, value, truncate(html, 500))
	}
}

// truncate truncates a string to max length with ellipsis.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
