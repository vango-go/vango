package server

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vango-go/vango/pkg/protocol"
	"github.com/vango-go/vango/pkg/vdom"
)

type bigStaticTreeComponent struct {
	n int
}

func (c bigStaticTreeComponent) Render() *vdom.VNode {
	children := make([]*vdom.VNode, 0, c.n)
	for i := 0; i < c.n; i++ {
		children = append(children, &vdom.VNode{Kind: vdom.KindElement, Tag: "div"})
	}
	return &vdom.VNode{Kind: vdom.KindElement, Tag: "main", Children: children}
}

// This is a regression test for a crash observed when session shutdown races
// the initial MountRoot render that runs after the handshake ServerHello is sent.
func TestServer_SessionShutdownRaceDuringInitialMount_DoesNotPanic(t *testing.T) {
	// Use multiple iterations to make the timing window deterministic enough.
	const iters = 200

	for i := 0; i < iters; i++ {
		s := New(DefaultServerConfig().WithDevMode())
		s.SetRootComponent(func() Component { return bigStaticTreeComponent{n: 500} })

		ts := httptest.NewServer(s.Handler())
		conn := dialWS(t, wsURL(t, ts.URL, "/_vango/live?path=/"), nil)

		writeHandshake(t, conn, protocol.NewClientHello(""))
		_ = readServerHello(t, conn)

		// Trigger shutdown as soon as the handshake response is observed,
		// while the server goroutine is likely still mounting/rendering.
		done := make(chan struct{})
		go func() {
			defer close(done)
			s.Sessions().Shutdown()
		}()

		_ = conn.Close()
		ts.Close()

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatalf("iteration %d: timeout waiting for Sessions().Shutdown()", i)
		}
	}
}

