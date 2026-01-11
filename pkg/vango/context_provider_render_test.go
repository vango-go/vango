package vango

import (
	"testing"

	"github.com/vango-go/vango/pkg/vdom"
)

func TestContextProvider_CreatesComponentVNode(t *testing.T) {
	ctx := CreateContext("default")
	node := ctx.Provider("value", vdom.Text("child"))
	if node == nil || node.Kind != vdom.KindComponent || node.Comp == nil {
		t.Fatalf("Provider() should return a component VNode, got %+v", node)
	}
}

func TestContextProviderComponent_Render_NoOwnerReturnsFragment(t *testing.T) {
	ctx := CreateContext("default")
	node := ctx.Provider("value", vdom.Text("a"), vdom.Text("b"))
	provider, ok := node.Comp.(*contextProviderComponent[string])
	if !ok {
		t.Fatalf("expected provider comp type, got %T", node.Comp)
	}

	old := setCurrentOwner(nil)
	defer setCurrentOwner(old)

	out := provider.Render()
	if out == nil || out.Kind != vdom.KindFragment {
		t.Fatalf("Render() without owner should return Fragment, got %+v", out)
	}
	if len(out.Children) != 2 {
		t.Fatalf("Fragment children = %d, want %d", len(out.Children), 2)
	}
}

func TestContextProviderComponent_Render_UpdatesSignalQuietly(t *testing.T) {
	ctx := CreateContext("default")
	root := NewOwner(nil)

	node := ctx.Provider("v1", vdom.Text("child"))
	provider := node.Comp.(*contextProviderComponent[string])

	root.StartRender()
	WithOwner(root, func() {
		_ = provider.Render()
	})
	root.EndRender()

	valAny := root.GetValueLocal(ctx.key)
	cv, ok := valAny.(*contextValue[string])
	if !ok || cv == nil || cv.signal == nil {
		t.Fatalf("expected contextValue stored in owner, got %T", valAny)
	}
	if got := cv.signal.Peek(); got != "v1" {
		t.Fatalf("initial provider signal value = %q, want %q", got, "v1")
	}
	if !cv.signal.transient {
		t.Fatalf("context provider signals should be transient")
	}

	// Subscribe a listener, then re-render provider: setQuietly MUST NOT notify.
	l := &mockListener{id: nextID()}
	WithListener(l, func() {
		_ = cv.signal.Get()
	})
	if l.dirty {
		t.Fatalf("listener should not be dirty after subscribing")
	}

	provider2 := ctx.Provider("v2", vdom.Text("child")).Comp.(*contextProviderComponent[string])
	root.StartRender()
	WithOwner(root, func() {
		_ = provider2.Render()
	})
	root.EndRender()

	if got := cv.signal.Peek(); got != "v2" {
		t.Fatalf("updated provider signal value = %q, want %q", got, "v2")
	}
	if l.dirty {
		t.Fatalf("provider rerender should update quietly without notifying subscribers")
	}
}

