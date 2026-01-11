package server

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/vango-go/vango/pkg/protocol"
	"github.com/vango-go/vango/pkg/vdom"
)

func TestSession_handleEventNavigate_ValidAndInvalidPayload(t *testing.T) {
	s := NewMockSession()

	// Invalid payload should send an error message (no conn in mock, so just cover branch).
	s.handleEvent(&Event{Type: protocol.EventNavigate, Payload: "not-a-navigate-payload"})

	// Valid payload should call HandleNavigate (with no router it falls back to NAV patch).
	s.handleEvent(&Event{
		Type: protocol.EventNavigate,
		Payload: &protocol.NavigateEventData{
			Path:    "/test",
			Replace: true,
		},
	})
}

func TestSession_Prefetch_HandlePrefetchAndExecutePrefetch_CachesInPrefetchMode(t *testing.T) {
	s := NewMockSession()
	s.prefetchConfig.Timeout = 2 * time.Second

	var sawPrefetchMode bool
	r := &testRouter{
		routes: map[string]RouteMatch{
			"/a": &testRouteMatch{
				params: map[string]string{"id": "1"},
				page: func(c Ctx, params any) Component {
					if c.Mode() != int(ModePrefetch) {
						t.Fatalf("ctx mode=%v, want prefetch", c.Mode())
					}
					if impl, ok := c.(*ctx); ok {
						if impl.RenderMode() != ModePrefetch {
							t.Fatalf("RenderMode=%v, want %v", impl.RenderMode(), ModePrefetch)
						}
					}
					sawPrefetchMode = true
					return staticComponent{node: &vdom.VNode{Kind: vdom.KindElement, Tag: "div"}}
				},
			},
		},
	}
	s.SetRouter(r)

	payloadBytes, err := json.Marshal(map[string]any{"path": "/a?x=1"})
	if err != nil {
		t.Fatalf("Marshal prefetch payload failed: %v", err)
	}

	s.handlePrefetch(payloadBytes)

	if !sawPrefetchMode {
		t.Fatal("prefetch handler did not run in ModePrefetch")
	}

	// Cache is keyed by canonical path (query stripped by CanonicalizePath's returned canonPath).
	if got := s.PrefetchCache().Get("/a"); got == nil || got.Tree == nil {
		t.Fatal("expected cached prefetch result for /a")
	}
}

func TestSession_collectHandlersPreserving_UsesChildInstanceLastTree(t *testing.T) {
	s := NewMockSession()
	s.handlers = make(map[string]Handler)
	s.components = make(map[string]*ComponentInstance)

	childComp := onClickComponent{}
	parentInst := newComponentInstance(staticComponent{node: &vdom.VNode{Kind: vdom.KindElement, Tag: "div"}}, nil, s)
	childInst := newComponentInstance(childComp, parentInst, s)
	parentInst.Children = []*ComponentInstance{childInst}

	// Child's last tree must exist for preserving logic to descend.
	childTree := childInst.Render()
	vdom.AssignHIDs(childTree, s.hidGen)
	if childInst.LastTree() == nil || childInst.LastTree().HID == "" {
		t.Fatal("expected child lastTree with HID")
	}

	// Parent's "last tree" includes a KindComponent marker for the child.
	parentTree := &vdom.VNode{
		Kind: vdom.KindElement,
		Tag:  "div",
		Children: []*vdom.VNode{
			{Kind: vdom.KindComponent, Comp: childComp},
		},
	}
	vdom.AssignHIDs(parentTree, s.hidGen)

	s.collectHandlersPreserving(parentTree, parentInst)
	key := childTree.HID + "_onclick"
	if _, ok := s.handlers[key]; !ok {
		t.Fatalf("missing preserved handler key %q", key)
	}
}
