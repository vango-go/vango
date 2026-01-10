package server

import (
	"log/slog"
	"testing"

	"github.com/vango-go/vango/pkg/vango"
	"github.com/vango-go/vango/pkg/vdom"
)

type testRouteMatch struct {
	params     map[string]string
	page       PageHandler
	layouts    []LayoutHandler
	middleware []RouteMiddleware
}

func (m *testRouteMatch) GetParams() map[string]string       { return m.params }
func (m *testRouteMatch) GetPageHandler() PageHandler        { return m.page }
func (m *testRouteMatch) GetLayoutHandlers() []LayoutHandler { return m.layouts }
func (m *testRouteMatch) GetMiddleware() []RouteMiddleware   { return m.middleware }

type testRouter struct {
	routes map[string]RouteMatch
}

func (r *testRouter) Match(method, path string) (RouteMatch, bool) {
	if r.routes == nil {
		return nil, false
	}
	m, ok := r.routes[path]
	return m, ok
}

func (r *testRouter) NotFound() PageHandler { return nil }

func TestSharedSignalSurvivesRouteNavigationAndRemainsReactive(t *testing.T) {
	counter := vango.NewSharedSignal(0)

	var sawIndexOwner bool
	var sawAboutOwner bool
	var renderedIndex int
	var renderedAbout int

	indexPage := func(ctx Ctx, params any) Component {
		return FuncComponent(func() *vdom.VNode {
			sawIndexOwner = counter.Signal() != nil
			renderedIndex = counter.Get()
			return vdom.Div(
				vdom.Span(vdom.Textf("%d", counter.Get())),
			)
		})
	}

	aboutPage := func(ctx Ctx, params any) Component {
		return FuncComponent(func() *vdom.VNode {
			sawAboutOwner = counter.Signal() != nil
			renderedAbout = counter.Get()
			return vdom.Div(
				vdom.Span(vdom.Textf("%d", counter.Get())),
			)
		})
	}

	r := &testRouter{
		routes: map[string]RouteMatch{
			"/": &testRouteMatch{
				params: map[string]string{},
				page:   indexPage,
			},
			"/about": &testRouteMatch{
				params: map[string]string{},
				page:   aboutPage,
			},
		},
	}

	session := newSession(nil, "", DefaultSessionConfig(), slog.Default())
	session.SetRouter(r)

	root, _, err := newRouteRootComponent(session, r, "/")
	if err != nil {
		t.Fatalf("newRouteRootComponent(/): %v", err)
	}
	session.MountRoot(root)

	if !sawIndexOwner {
		t.Fatal("expected SharedSignalDef.Signal() to be non-nil during initial route render")
	}

	// Simulate some session-scoped state before navigation.
	vango.WithOwner(session.root.Owner, func() {
		counter.Set(7)
	})

	if err := session.HandleNavigate("/about", false); err != nil {
		t.Fatalf("HandleNavigate(/about): %v", err)
	}

	if !sawAboutOwner {
		t.Fatal("expected SharedSignalDef.Signal() to be non-nil during navigation render")
	}
	if renderedIndex != 0 {
		// renderedIndex is from the first render before we set counter=7.
		// Keep this assertion to ensure we didn't accidentally re-render the index page
		// as part of navigation setup.
		t.Fatalf("renderedIndex = %d, want 0", renderedIndex)
	}
	if renderedAbout != 7 {
		t.Fatalf("renderedAbout = %d, want 7", renderedAbout)
	}

	// Verify that the new page render established a reactive subscription:
	// updating the shared signal should mark the mounted root dirty.
	session.root.ClearDirty()
	vango.WithOwner(session.root.Owner, func() {
		counter.Set(8)
	})
	if !session.root.IsDirty() {
		t.Fatal("expected route root to become dirty after SharedSignal update post-navigation")
	}
}

