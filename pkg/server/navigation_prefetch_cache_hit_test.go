package server

import (
	"testing"

	"github.com/vango-go/vango/pkg/vdom"
)

func TestRouteNavigator_Navigate_PrefetchCacheHitCallsUseCachedTree(t *testing.T) {
	sess := NewMockSession()
	sess.PrefetchCache().Set("/hit", &vdom.VNode{Kind: vdom.KindText, Text: "cached"})

	r := &testRouter{
		routes: map[string]RouteMatch{
			"/hit": &testRouteMatch{
				params: map[string]string{},
				page: func(c Ctx, params any) Component {
					return staticComponent{node: &vdom.VNode{Kind: vdom.KindElement, Tag: "div"}}
				},
			},
		},
	}

	nav := NewRouteNavigator(sess, r)
	res := nav.Navigate("/hit", false)
	if res.Error != nil {
		t.Fatalf("Navigate error: %v", res.Error)
	}
}

