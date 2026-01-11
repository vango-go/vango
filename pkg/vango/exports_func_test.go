package vango

import (
	"testing"

	"github.com/vango-go/vango/pkg/vdom"
)

func TestFuncReExport(t *testing.T) {
	c := Func(func() *vdom.VNode {
		return vdom.Text("ok")
	})
	out := c.Render()
	if out == nil || out.Kind != vdom.KindText || out.Text != "ok" {
		t.Fatalf("Render() = %+v, want text node %q", out, "ok")
	}
}

