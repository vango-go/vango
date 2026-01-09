package server

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/vango-go/vango/pkg/protocol"
	"github.com/vango-go/vango/pkg/vango"
	"github.com/vango-go/vango/pkg/vdom"
)

func TestSharedSignalWorksInEventHandler(t *testing.T) {
	counter := vango.NewSharedSignal(0)
	var sawSignalInHandler bool

	session := newSession(nil, "", DefaultSessionConfig(), slog.Default())

	comp := FuncComponent(func() *vdom.VNode {
		return vdom.Div(
			vdom.Button(
				vdom.OnClick(func() {
					// This used to be nil because event handlers did not set an Owner.
					sawSignalInHandler = counter.Signal() != nil
					counter.Set(counter.Get() + 1)
				}),
				vdom.Text("inc"),
			),
			vdom.Span(vdom.Textf("%d", counter.Get())),
		)
	})

	session.MountRoot(comp)

	var hid string
	for k := range session.handlers {
		if strings.HasSuffix(k, "_onclick") {
			hid = strings.TrimSuffix(k, "_onclick")
			break
		}
	}
	if hid == "" {
		t.Fatal("expected an onclick handler to be registered")
	}

	session.handleEvent(&Event{
		Seq:  1,
		Type: protocol.EventClick,
		HID:  hid,
	})

	if !sawSignalInHandler {
		t.Fatal("expected SharedSignalDef.Signal() to be non-nil inside handler")
	}

	instance := session.components[hid]
	if instance == nil || instance.Owner == nil {
		t.Fatal("expected component instance owner to be recorded for HID")
	}

	var got int
	vango.WithOwner(instance.Owner, func() {
		got = counter.Get()
	})
	if got != 1 {
		t.Fatalf("counter = %d, want 1", got)
	}
}

