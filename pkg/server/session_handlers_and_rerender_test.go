package server

import (
	"testing"

	"github.com/vango-go/vango/pkg/protocol"
	"github.com/vango-go/vango/pkg/vdom"
)

func TestSession_collectHandlersNoMount_OnHookSliceAndLegacyEventHandler(t *testing.T) {
	s := NewMockSession()
	s.handlers = make(map[string]Handler)
	s.components = make(map[string]*ComponentInstance)

	inst := newComponentInstance(onClickComponent{}, nil, s)

	calls := 0
	h1 := func(he HookEvent) { calls++ }
	h2 := func(he HookEvent) { calls++ }

	node := &vdom.VNode{
		Kind: vdom.KindElement,
		Tag:  "div",
		HID:  "h1",
		Props: map[string]any{
			"onhook": []any{h1, h2},
			"onclick": func() {},
			"legacy": vdom.EventHandler{Event: "reorder", Handler: func() {}},
		},
	}

	s.collectHandlersNoMount(node, inst)

	onhook, ok := s.handlers["h1_onhook"]
	if !ok {
		t.Fatal("missing onhook handler")
	}
	onhook(&Event{Payload: &protocol.HookEventData{Name: "evt", Data: map[string]any{}}})
	if calls != 2 {
		t.Fatalf("calls=%d, want 2", calls)
	}

	if _, ok := s.handlers["h1_onclick"]; !ok {
		t.Fatal("missing onclick handler")
	}
	if _, ok := s.handlers["h1_hook_reorder"]; !ok {
		t.Fatal("missing legacy hook handler")
	}
}

type dynamicChildrenComponent struct {
	showSecond *bool
}

func (c dynamicChildrenComponent) Render() *vdom.VNode {
	children := []*vdom.VNode{
		{Kind: vdom.KindComponent, Comp: staticComponent{node: &vdom.VNode{Kind: vdom.KindText, Text: "a"}}},
	}
	if c.showSecond != nil && *c.showSecond {
		children = append(children, &vdom.VNode{Kind: vdom.KindComponent, Comp: staticComponent{node: &vdom.VNode{Kind: vdom.KindText, Text: "b"}}})
	}
	return &vdom.VNode{Kind: vdom.KindElement, Tag: "div", Children: children}
}

func TestSession_rerenderChildren_CreatesReusesAndDisposesInstances(t *testing.T) {
	showSecond := true
	s := NewMockSession()
	s.MountRoot(dynamicChildrenComponent{showSecond: &showSecond})

	if s.root == nil || len(s.root.Children) != 2 {
		t.Fatalf("initial children=%d, want 2", len(s.root.Children))
	}
	oldSecond := s.root.Children[1]
	if oldSecond == nil || oldSecond.Owner == nil {
		t.Fatal("expected old second child instance with owner")
	}

	showSecond = false
	_ = s.renderComponent(s.root)
	if len(s.root.Children) != 1 {
		t.Fatalf("after hide children=%d, want 1", len(s.root.Children))
	}
	if oldSecond.Owner != nil || oldSecond.Component != nil {
		t.Fatal("expected disposed instance to have Owner and Component cleared")
	}

	showSecond = true
	_ = s.renderComponent(s.root)
	if len(s.root.Children) != 2 {
		t.Fatalf("after show children=%d, want 2", len(s.root.Children))
	}
	if s.root.Children[1] == oldSecond {
		t.Fatal("expected new instance for second child after it was disposed")
	}
}

