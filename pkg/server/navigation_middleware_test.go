package server_test

import (
	"errors"
	"testing"

	"github.com/vango-go/vango/pkg/protocol"
	"github.com/vango-go/vango/pkg/server"
	"github.com/vango-go/vango/pkg/router"
	"github.com/vango-go/vango/pkg/vdom"
)

func TestRouteNavigatorNavigate_MiddlewareRedirect(t *testing.T) {
	s := server.NewMockSession()

	r := router.NewRouter()
	r.Page("/login", func(ctx server.Ctx, params any) vdom.Component {
		return vdom.Func(func() *vdom.VNode {
			return &vdom.VNode{Kind: vdom.KindText, Text: "login"}
		})
	})
	r.Page("/admin", func(ctx server.Ctx, params any) vdom.Component {
		return vdom.Func(func() *vdom.VNode {
			return &vdom.VNode{Kind: vdom.KindText, Text: "admin"}
		})
	})
	r.Middleware("/admin", router.MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
		ctx.Navigate("/login", server.WithReplace())
		return nil // short-circuit
	}))

	s.SetRouter(router.NewRouterAdapter(r))

	res := s.Navigator().Navigate("/admin", false)
	if res.Error != nil {
		t.Fatalf("Navigate error = %v", res.Error)
	}
	if got, want := res.Path, "/login"; got != want {
		t.Fatalf("Path = %q, want %q", got, want)
	}
	if got, want := res.NavPatch.Op, protocol.PatchNavReplace; got != want {
		t.Fatalf("NavPatch.Op = %v, want %v", got, want)
	}
}

func TestRouteNavigatorNavigate_MiddlewareSeesRequest(t *testing.T) {
	s := server.NewMockSession()

	r := router.NewRouter()
	r.Page("/admin", func(ctx server.Ctx, params any) vdom.Component {
		return vdom.Func(func() *vdom.VNode {
			return &vdom.VNode{Kind: vdom.KindText, Text: "admin"}
		})
	})

	r.Middleware("/admin", router.MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
		req := ctx.Request()
		if req == nil || req.URL == nil || req.URL.Path != "/admin" {
			return errors.New("missing or incorrect request in navigation middleware")
		}
		return next()
	}))

	s.SetRouter(router.NewRouterAdapter(r))

	res := s.Navigator().Navigate("/admin", false)
	if res.Error != nil {
		t.Fatalf("Navigate error = %v", res.Error)
	}
}
