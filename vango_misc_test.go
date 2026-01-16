package vango

import (
	"context"
	"testing"

	"github.com/vango-go/vango/pkg/assets"
	"github.com/vango-go/vango/pkg/vdom"
)

func TestOnEventFiltersByName(t *testing.T) {
	calls := 0
	attr := OnEvent("save", func(e HookEvent) {
		calls++
	})

	if attr.Key != "onhook" {
		t.Fatalf("attr.Key = %q, want %q", attr.Key, "onhook")
	}

	handler, ok := attr.Value.(func(HookEvent))
	if !ok || handler == nil {
		t.Fatalf("attr.Value type = %T, want func(HookEvent)", attr.Value)
	}

	handler(HookEvent{Name: "other"})
	handler(HookEvent{Name: "save"})
	handler(HookEvent{Name: "save"})

	if calls != 2 {
		t.Fatalf("calls = %d, want %d", calls, 2)
	}
}

func TestWithUserRoundTrip(t *testing.T) {
	base := context.Background()
	ctx := WithUser(base, "alice")

	if got := UserFromContext(base); got != nil {
		t.Fatalf("base context user = %v, want nil", got)
	}

	if got := UserFromContext(ctx); got != "alice" {
		t.Fatalf("user = %v, want %q", got, "alice")
	}
}

func TestAssetResolvers(t *testing.T) {
	manifest := assets.NewManifest()
	manifest.Set("app.js", "app.abc123.js")

	resolver := NewAssetResolver(manifest, "/public/")
	if got := resolver.Asset("app.js"); got != "/public/app.abc123.js" {
		t.Fatalf("resolved = %q, want %q", got, "/public/app.abc123.js")
	}
	if got := resolver.Asset("other.js"); got != "/public/other.js" {
		t.Fatalf("resolved = %q, want %q", got, "/public/other.js")
	}

	passthrough := NewPassthroughResolver("/public/")
	if got := passthrough.Asset("app.js"); got != "/public/app.js" {
		t.Fatalf("resolved = %q, want %q", got, "/public/app.js")
	}
}

func TestComponentTypesAreCompatible(t *testing.T) {
	node := vdom.Text("ok")
	var comp Component = vdom.Func(func() *vdom.VNode { return node })

	if got := comp.Render(); got != node {
		t.Fatalf("Render() = %#v, want %#v", got, node)
	}
}
