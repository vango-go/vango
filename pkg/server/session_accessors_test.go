package server

import (
	"log/slog"
	"testing"

	"github.com/vango-go/vango/pkg/assets"
)

func TestSession_EventsConnAssetResolverAndStormBudget(t *testing.T) {
	clientConn, serverConn := newWebSocketPair(t)
	_ = clientConn

	cfg := DefaultSessionConfig()
	cfg.StormBudget = DefaultStormBudgetConfig()
	sess := newSession(serverConn, "", cfg, slog.Default())

	if sess.Conn() != serverConn {
		t.Fatal("Conn() did not return underlying connection")
	}

	// Events() should expose the same queue used by QueueEvent().
	e := &Event{HID: "h1"}
	if err := sess.QueueEvent(e); err != nil {
		t.Fatalf("QueueEvent error: %v", err)
	}
	select {
	case got := <-sess.Events():
		if got != e {
			t.Fatal("Events() channel did not yield queued event")
		}
	default:
		t.Fatal("expected event to be available on Events()")
	}

	res := assets.NewPassthroughResolver("/public/")
	sess.SetAssetResolver(res)
	c := sess.createRenderContext().(*ctx)
	if got := c.Asset("app.js"); got != "/public/app.js" {
		t.Fatalf("ctx.Asset()=%q, want %q", got, "/public/app.js")
	}

	if sess.StormBudget() == nil {
		t.Fatal("StormBudget()=nil, want non-nil when configured")
	}
}

