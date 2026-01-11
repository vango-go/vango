package server

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vango-go/vango/pkg/assets"
	"github.com/vango-go/vango/pkg/protocol"
)

func newWebSocketPair(t *testing.T) (client *websocket.Conn, server *websocket.Conn) {
	t.Helper()

	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	serverConnCh := make(chan *websocket.Conn, 1)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("Upgrade failed: %v", err)
			return
		}
		serverConnCh <- c
	}))
	t.Cleanup(ts.Close)

	client = dialWS(t, wsURL(t, ts.URL, "/ws"), nil)
	server = <-serverConnCh
	t.Cleanup(func() { _ = server.Close() })
	return client, server
}

type testAssetResolver struct{}

func (testAssetResolver) Asset(source string) string {
	return "/public/" + strings.TrimPrefix(source, "/")
}

func TestCtx_Emit_SendsDispatchPatch(t *testing.T) {
	clientConn, serverConn := newWebSocketPair(t)

	sess := newSession(serverConn, "", DefaultSessionConfig(), slog.Default())
	c := NewTestContext(sess).(*ctx)

	c.Emit("toast", map[string]any{"message": "hello"})

	_ = clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := clientConn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}
	frame, err := protocol.DecodeFrame(msg)
	if err != nil {
		t.Fatalf("DecodeFrame failed: %v", err)
	}
	if frame.Type != protocol.FramePatches {
		t.Fatalf("frame type=%v, want %v", frame.Type, protocol.FramePatches)
	}
	pf, err := protocol.DecodePatches(frame.Payload)
	if err != nil {
		t.Fatalf("DecodePatches failed: %v", err)
	}
	if len(pf.Patches) != 1 {
		t.Fatalf("patch count=%d, want 1", len(pf.Patches))
	}
	p := pf.Patches[0]
	if p.Op != protocol.PatchDispatch || p.Key != "toast" || p.Value == "" {
		t.Fatalf("dispatch patch = %+v, want op=Dispatch key=toast value non-empty", p)
	}
}

func TestCtx_Mode_RenderMode_AndAsset(t *testing.T) {
	c := &ctx{}
	if c.Mode() != int(ModeNormal) || c.RenderMode() != ModeNormal {
		t.Fatalf("default mode=%v renderMode=%v, want normal", c.Mode(), c.RenderMode())
	}

	c.setMode(ModePrefetch)
	if c.Mode() != int(ModePrefetch) || c.RenderMode() != ModePrefetch || !c.IsPrefetch() {
		t.Fatalf("mode=%v renderMode=%v isPrefetch=%v, want prefetch/true", c.Mode(), c.RenderMode(), c.IsPrefetch())
	}

	if got := c.Asset("app.js"); got != "app.js" {
		t.Fatalf("Asset()=%q without resolver, want passthrough", got)
	}

	c.setAssetResolver(assets.Resolver(testAssetResolver{}))
	if got := c.Asset("app.js"); got != "/public/app.js" {
		t.Fatalf("Asset()=%q, want %q", got, "/public/app.js")
	}
}

func TestCtx_StormBudget_NilWithoutSession(t *testing.T) {
	c := &ctx{session: nil}
	if c.StormBudget() != nil {
		t.Fatal("StormBudget() != nil, want nil without session")
	}
}

func TestNavigateOptions_WithoutScrollAndParams(t *testing.T) {
	applied := ApplyNavigateOptions(WithoutScroll(), WithNavigateParams(map[string]any{"q": "x"}))
	if applied.Scroll {
		t.Fatal("Scroll=true, want false")
	}
	if applied.Params["q"] != "x" {
		t.Fatalf("Params[q]=%v, want %q", applied.Params["q"], "x")
	}
}

