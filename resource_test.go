package vango

import (
	"testing"
	"time"

	corevango "github.com/vango-go/vango/pkg/vango"
	"github.com/vango-go/vango/pkg/vdom"
)

func TestNewResourceKeyed_WithSignalKey(t *testing.T) {
	key := NewSignal(42)
	got := make(chan int, 1)

	comp := vdom.Func(func() *vdom.VNode {
		NewResourceKeyed(key, func(k int) (int, error) {
			got <- k
			return k, nil
		})
		return vdom.Text("ok")
	})

	owner := corevango.NewOwner(nil)
	corevango.WithOwner(owner, func() {
		owner.StartRender()
		defer owner.EndRender()
		comp.Render()
	})
	owner.RunPendingEffects(nil)

	select {
	case value := <-got:
		if value != 42 {
			t.Fatalf("key = %d, want %d", value, 42)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for fetcher to receive key")
	}
}

func TestNewResourceKeyed_WithFuncKey(t *testing.T) {
	got := make(chan string, 1)

	comp := vdom.Func(func() *vdom.VNode {
		NewResourceKeyed(func() string { return "alpha" }, func(k string) (string, error) {
			got <- k
			return k, nil
		})
		return vdom.Text("ok")
	})

	owner := corevango.NewOwner(nil)
	corevango.WithOwner(owner, func() {
		owner.StartRender()
		defer owner.EndRender()
		comp.Render()
	})
	owner.RunPendingEffects(nil)

	select {
	case value := <-got:
		if value != "alpha" {
			t.Fatalf("key = %q, want %q", value, "alpha")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for fetcher to receive key")
	}
}

func TestNewResourceKeyed_InvalidKeyPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for invalid key type")
		}
	}()

	NewResourceKeyed(123, func(k int) (int, error) {
		return k, nil
	})
}
