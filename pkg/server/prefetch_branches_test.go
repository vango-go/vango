package server

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/vango-go/vango/pkg/vdom"
)

func TestSession_handlePrefetch_Branches(t *testing.T) {
	t.Run("invalid JSON dropped", func(t *testing.T) {
		s := NewMockSession()
		s.handlePrefetch([]byte("{not-json"))
		if s.PrefetchCache().Len() != 0 {
			t.Fatal("expected no cache entries")
		}
	})

	t.Run("empty path dropped", func(t *testing.T) {
		s := NewMockSession()
		b, _ := json.Marshal(map[string]any{"path": ""})
		s.handlePrefetch(b)
		if s.PrefetchCache().Len() != 0 {
			t.Fatal("expected no cache entries")
		}
	})

	t.Run("rate limited dropped", func(t *testing.T) {
		s := NewMockSession()
		s.prefetchLimiter.tokens = 0
		s.prefetchLimiter.lastRefill = time.Now()

		b, _ := json.Marshal(map[string]any{"path": "/x"})
		s.handlePrefetch(b)
		if s.PrefetchCache().Len() != 0 {
			t.Fatal("expected no cache entries when rate limited")
		}
	})

	t.Run("already cached returns early", func(t *testing.T) {
		s := NewMockSession()
		s.PrefetchCache().Set("/a", &vdom.VNode{Kind: vdom.KindText, Text: "cached"})

		called := false
		s.SetRouter(&testRouter{
			routes: map[string]RouteMatch{
				"/a": &testRouteMatch{
					page: func(c Ctx, params any) Component {
						called = true
						return staticComponent{node: &vdom.VNode{Kind: vdom.KindElement, Tag: "div"}}
					},
				},
			},
		})

		s.executePrefetch("/a")
		if called {
			t.Fatal("page handler should not run when already cached")
		}
	})

	t.Run("panic in handler does not cache", func(t *testing.T) {
		s := NewMockSession()
		s.prefetchConfig.Timeout = 100 * time.Millisecond
		s.SetRouter(&testRouter{
			routes: map[string]RouteMatch{
				"/panic": &testRouteMatch{
					page: func(c Ctx, params any) Component {
						panic("boom")
					},
				},
			},
		})

		s.executePrefetch("/panic")
		if s.PrefetchCache().Get("/panic") != nil {
			t.Fatal("expected no cached entry after panic")
		}
	})

	t.Run("timeout does not cache", func(t *testing.T) {
		s := NewMockSession()
		s.prefetchConfig.Timeout = 10 * time.Millisecond
		s.SetRouter(&testRouter{
			routes: map[string]RouteMatch{
				"/slow": &testRouteMatch{
					page: func(c Ctx, params any) Component {
						time.Sleep(50 * time.Millisecond)
						return staticComponent{node: &vdom.VNode{Kind: vdom.KindElement, Tag: "div"}}
					},
				},
			},
		})

		s.executePrefetch("/slow")
		if s.PrefetchCache().Get("/slow") != nil {
			t.Fatal("expected no cached entry after timeout")
		}
	})
}

