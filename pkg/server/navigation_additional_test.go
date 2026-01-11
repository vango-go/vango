package server

import (
	"testing"

	"github.com/vango-go/vango/pkg/protocol"
	"github.com/vango-go/vango/pkg/vdom"
)

func TestRedirectError_ErrorString(t *testing.T) {
	err := redirectError{path: "/to", replace: true}
	if got := err.Error(); got == "" || got[0] != 'r' {
		t.Fatalf("Error()=%q, want non-empty message", got)
	}
}

func TestRouteNavigator_Navigate_CanonicalizationForcesReplace(t *testing.T) {
	sess := NewMockSession()
	r := &testRouter{
		routes: map[string]RouteMatch{
			"/x": &testRouteMatch{
				params: map[string]string{},
				page: func(c Ctx, params any) Component {
					return staticComponent{node: &vdom.VNode{Kind: vdom.KindElement, Tag: "div"}}
				},
			},
		},
	}

	nav := NewRouteNavigator(sess, r)
	res := nav.Navigate("/x/", false) // trailing slash -> canonicalized to /x
	if res.Error != nil {
		t.Fatalf("Navigate error: %v", res.Error)
	}
	if res.Path != "/x" {
		t.Fatalf("Path=%q, want %q", res.Path, "/x")
	}
	if res.NavPatch.Op != protocol.PatchNavReplace {
		t.Fatalf("NavPatch.Op=%v, want %v", res.NavPatch.Op, protocol.PatchNavReplace)
	}
}

func TestRouteNavigator_Navigate_NoRouteAndNoNotFoundReturnsNavOnly(t *testing.T) {
	sess := NewMockSession()
	r := &testRouter{routes: map[string]RouteMatch{}}

	nav := NewRouteNavigator(sess, r)
	res := nav.Navigate("/missing", false)
	if res.Error != nil {
		t.Fatalf("Navigate error: %v", res.Error)
	}
	if res.Matched {
		t.Fatal("Matched=true, want false")
	}
	if res.NavPatch.Op != protocol.PatchNavPush || res.NavPatch.Path != "/missing" {
		t.Fatalf("NavPatch=%+v, want NavPush(/missing)", res.NavPatch)
	}
}

func TestRouteNavigator_Navigate_RedirectViaCtxNavigate(t *testing.T) {
	sess := NewMockSession()
	r := &testRouter{
		routes: map[string]RouteMatch{
			"/from": &testRouteMatch{
				params: map[string]string{},
				page: func(c Ctx, params any) Component {
					c.Navigate("/to")
					return staticComponent{node: &vdom.VNode{Kind: vdom.KindElement, Tag: "div"}}
				},
			},
			"/to": &testRouteMatch{
				params: map[string]string{},
				page: func(c Ctx, params any) Component {
					return staticComponent{node: &vdom.VNode{Kind: vdom.KindElement, Tag: "span"}}
				},
			},
		},
	}

	nav := NewRouteNavigator(sess, r)
	res := nav.Navigate("/from", false)
	if res.Error != nil {
		t.Fatalf("Navigate error: %v", res.Error)
	}
	if res.Path != "/to" {
		t.Fatalf("Path=%q, want %q", res.Path, "/to")
	}
	if res.NavPatch.Op != protocol.PatchNavReplace {
		t.Fatalf("NavPatch.Op=%v, want %v (redirects should replace)", res.NavPatch.Op, protocol.PatchNavReplace)
	}
}

func TestRouteNavigator_collectHandlersFromTree_AndExpandComponents(t *testing.T) {
	sess := NewMockSession()
	sess.handlers = make(map[string]Handler)

	nav := NewRouteNavigator(sess, &testRouter{routes: map[string]RouteMatch{}})

	tree := &vdom.VNode{
		Kind: vdom.KindElement,
		Tag:  "div",
		HID:  "h1",
		Props: map[string]any{
			"onclick": func() {},
		},
		Children: []*vdom.VNode{
			{Kind: vdom.KindText, Text: "x"},
		},
	}
	nav.collectHandlersFromTree(tree)
	if _, ok := sess.handlers["h1_onclick"]; !ok {
		t.Fatal("expected handler collected from tree")
	}

	expanded := expandComponents(&vdom.VNode{
		Kind: vdom.KindElement,
		Tag:  "div",
		Children: []*vdom.VNode{
			{Kind: vdom.KindComponent, Comp: staticComponent{node: &vdom.VNode{Kind: vdom.KindText, Text: "ok"}}},
			nil,
		},
	})
	if expanded == nil || len(expanded.Children) != 1 || expanded.Children[0].Text != "ok" {
		t.Fatalf("expandComponents result=%+v, want div with single text child", expanded)
	}
}

