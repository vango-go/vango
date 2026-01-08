package vango

import (
	"fmt"
	"sync/atomic"
	"testing"

	corevango "github.com/vango-go/vango/pkg/vango"
	"github.com/vango-go/vango/pkg/vdom"
)

type dirtyListener struct {
	id      uint64
	dirtyCt atomic.Uint64
}

func (l *dirtyListener) ID() uint64 { return l.id }
func (l *dirtyListener) MarkDirty() { l.dirtyCt.Add(1) }

func renderWithOwner(owner *corevango.Owner, listener corevango.Listener, comp vdom.Component) *vdom.VNode {
	var out *vdom.VNode
	corevango.WithOwner(owner, func() {
		owner.StartRender()
		defer owner.EndRender()
		corevango.WithListener(listener, func() {
			out = comp.Render()
		})
	})
	return out
}

func TestWrapPageHandler_SignalUpdatesMarkDirty(t *testing.T) {
	var renderCalls atomic.Uint64
	var sig *Signal[int]

	page := func(ctx Ctx) *VNode {
		renderCalls.Add(1)
		sig = NewSignal(0)
		return &vdom.VNode{
			Kind: vdom.KindElement,
			Tag:  "div",
			Children: []*vdom.VNode{
				{Kind: vdom.KindText, Text: fmt.Sprintf("%d", sig.Get())},
			},
		}
	}

	internal := wrapPageHandler(page)
	comp := internal(nil, nil)

	owner := corevango.NewOwner(nil)
	listener := &dirtyListener{id: owner.ID()}

	n1 := renderWithOwner(owner, listener, comp)
	if n1 == nil {
		t.Fatal("expected rendered node")
	}
	if sig == nil {
		t.Fatal("expected signal to be created during render")
	}
	if got := renderCalls.Load(); got != 1 {
		t.Fatalf("renderCalls = %d, want 1", got)
	}
	if got := listener.dirtyCt.Load(); got != 0 {
		t.Fatalf("dirtyCt after initial render = %d, want 0", got)
	}

	sig.Inc()
	if got := listener.dirtyCt.Load(); got == 0 {
		t.Fatal("expected signal update to mark listener dirty")
	}

	n2 := renderWithOwner(owner, listener, comp)
	if got := renderCalls.Load(); got != 2 {
		t.Fatalf("renderCalls after second render = %d, want 2", got)
	}
	if n2 == nil || len(n2.Children) == 0 || n2.Children[0].Kind != vdom.KindText {
		t.Fatal("expected text child in rendered node")
	}
	if got := n2.Children[0].Text; got != "1" {
		t.Fatalf("rendered text = %q, want %q", got, "1")
	}
}

