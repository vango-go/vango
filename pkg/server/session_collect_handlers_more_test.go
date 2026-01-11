package server

import (
	"testing"

	"github.com/vango-go/vango/pkg/protocol"
	"github.com/vango-go/vango/pkg/vdom"
)

func TestSession_collectHandlers_CoversOnHookAndLegacyAndMountsChild(t *testing.T) {
	s := NewMockSession()
	s.handlers = make(map[string]Handler)
	s.components = make(map[string]*ComponentInstance)

	rootInst := newComponentInstance(staticComponent{node: &vdom.VNode{Kind: vdom.KindElement, Tag: "div"}}, nil, s)
	s.registerComponent(rootInst)

	hookCalls := 0
	rootTree := &vdom.VNode{
		Kind: vdom.KindElement,
		Tag:  "div",
		HID:  "h1",
		Props: map[string]any{
			"onhook": []any{
				func(he HookEvent) { hookCalls++ },
				func(he HookEvent) { hookCalls++ },
			},
			"legacy": vdom.EventHandler{Event: "reorder", Handler: func() {}},
		},
		Children: []*vdom.VNode{
			{Kind: vdom.KindComponent, Comp: onClickComponent{}},
		},
	}

	s.collectHandlers(rootTree, rootInst)

	combined, ok := s.handlers["h1_onhook"]
	if !ok {
		t.Fatal("missing onhook handler")
	}
	combined(&Event{Payload: &protocol.HookEventData{Name: "evt", Data: map[string]any{}}})
	if hookCalls != 2 {
		t.Fatalf("hookCalls=%d, want 2", hookCalls)
	}
	if _, ok := s.handlers["h1_hook_reorder"]; !ok {
		t.Fatal("missing legacy handler")
	}
	if len(rootInst.Children) != 1 {
		t.Fatalf("children=%d, want 1", len(rootInst.Children))
	}
	childInst := rootInst.Children[0]
	if childInst == nil || childInst.LastTree() == nil || childInst.LastTree().HID == "" {
		t.Fatalf("expected mounted child instance with HIDs: %+v", childInst)
	}
	if _, ok := s.handlers[childInst.LastTree().HID+"_onclick"]; !ok {
		t.Fatal("missing child onclick handler")
	}
}

func TestSession_collectHandlersPreserving_CoversOnHookAndLegacy(t *testing.T) {
	s := NewMockSession()
	s.handlers = make(map[string]Handler)
	s.components = make(map[string]*ComponentInstance)

	parent := newComponentInstance(staticComponent{node: &vdom.VNode{Kind: vdom.KindElement, Tag: "div"}}, nil, s)
	childComp := onClickComponent{}
	child := newComponentInstance(childComp, parent, s)
	parent.Children = []*ComponentInstance{child}

	childTree := child.Render()
	vdom.AssignHIDs(childTree, s.hidGen)

	parentTree := &vdom.VNode{
		Kind: vdom.KindElement,
		Tag:  "div",
		HID:  "h1",
		Props: map[string]any{
			"onhook": []any{func(he HookEvent) {}},
			"legacy": vdom.EventHandler{Event: "reorder", Handler: func() {}},
		},
		Children: []*vdom.VNode{
			{Kind: vdom.KindComponent, Comp: childComp},
		},
	}

	s.collectHandlersPreserving(parentTree, parent)

	if _, ok := s.handlers["h1_onhook"]; !ok {
		t.Fatal("missing preserved onhook handler")
	}
	if _, ok := s.handlers["h1_hook_reorder"]; !ok {
		t.Fatal("missing preserved legacy handler")
	}
	if _, ok := s.handlers[childTree.HID+"_onclick"]; !ok {
		t.Fatal("missing preserved child onclick handler")
	}
}

