package server

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vango-go/vango/pkg/protocol"
	"github.com/vango-go/vango/pkg/vdom"
)

func TestServer_MiddlewareOrderAndLoggerAccessors(t *testing.T) {
	s := New(DefaultServerConfig().WithDevMode())
	s.SetHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	s.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-MW", "1")
			next.ServeHTTP(w, r)
		})
	})
	s.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-MW", "2")
			next.ServeHTTP(w, r)
		})
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/any", nil)
	s.ServeHTTP(rr, req)

	got := rr.Header().Values("X-MW")
	if len(got) != 2 || got[0] != "1" || got[1] != "2" {
		t.Fatalf("middleware order=%v, want [1 2]", got)
	}

	if s.Config() == nil || s.Sessions() == nil || s.Logger() == nil {
		t.Fatal("expected non-nil Config/Sessions/Logger")
	}

	custom := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
	s.SetLogger(custom)
	if s.Logger() != custom {
		t.Fatal("SetLogger did not take effect")
	}
}

func TestServer_HandleWebSocket_SetRootComponentMountsRoot(t *testing.T) {
	s := New(DefaultServerConfig().WithDevMode())
	s.SetRootComponent(func() Component {
		return staticComponent{node: &vdom.VNode{Kind: vdom.KindElement, Tag: "main"}}
	})
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	t.Cleanup(func() { s.Sessions().Shutdown() })

	conn := dialWS(t, wsURL(t, ts.URL, "/_vango/live?path=/"), nil)
	writeHandshake(t, conn, protocol.NewClientHello(""))
	hello := readServerHello(t, conn)
	if hello.Status != protocol.HandshakeOK {
		t.Fatalf("handshake status=%v, want %v", hello.Status, protocol.HandshakeOK)
	}

	sess := getSessionEventually(t, s.Sessions(), hello.SessionID)
	waitForEventLoopStarted(t, sess)

	if sess.root == nil || sess.currentTree == nil {
		t.Fatal("expected mounted session root after handshake")
	}
	if sess.currentTree.Tag != "main" {
		t.Fatalf("root tag=%q, want %q", sess.currentTree.Tag, "main")
	}
}

func TestServer_Run_ReturnsListenError(t *testing.T) {
	cfg := DefaultServerConfig().WithDevMode().WithAddress("127.0.0.1:-1")
	s := New(cfg)
	if err := s.Run(); err == nil {
		t.Fatal("Run() error=nil, want non-nil listen error")
	}
}
