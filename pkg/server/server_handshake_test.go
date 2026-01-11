package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vango-go/vango/pkg/protocol"
	"github.com/vango-go/vango/pkg/vdom"
)

func wsURL(t *testing.T, baseURL, path string) string {
	t.Helper()
	if !strings.HasPrefix(baseURL, "http") {
		t.Fatalf("unexpected base URL: %q", baseURL)
	}
	return "ws" + strings.TrimPrefix(baseURL, "http") + path
}

func dialWS(t *testing.T, url string, header http.Header) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(url, header)
	if err != nil {
		t.Fatalf("Dial(%q) failed: %v", url, err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func writeHandshake(t *testing.T, conn *websocket.Conn, hello *protocol.ClientHello) {
	t.Helper()
	payload := protocol.EncodeClientHello(hello)
	frame := protocol.NewFrame(protocol.FrameHandshake, payload)
	if err := conn.WriteMessage(websocket.BinaryMessage, frame.Encode()); err != nil {
		t.Fatalf("write handshake failed: %v", err)
	}
}

func readServerHello(t *testing.T, conn *websocket.Conn) *protocol.ServerHello {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read handshake response failed: %v", err)
	}
	frame, err := protocol.DecodeFrame(msg)
	if err != nil {
		t.Fatalf("DecodeFrame failed: %v", err)
	}
	if frame.Type != protocol.FrameHandshake {
		t.Fatalf("frame type = %v, want %v", frame.Type, protocol.FrameHandshake)
	}
	hello, err := protocol.DecodeServerHello(frame.Payload)
	if err != nil {
		t.Fatalf("DecodeServerHello failed: %v", err)
	}
	return hello
}

type staticComponent struct {
	node *vdom.VNode
}

func (c staticComponent) Render() *vdom.VNode { return c.node }

type handshakeRouteMatch struct {
	params map[string]string
	page   PageHandler
}

func (m *handshakeRouteMatch) GetParams() map[string]string       { return m.params }
func (m *handshakeRouteMatch) GetPageHandler() PageHandler        { return m.page }
func (m *handshakeRouteMatch) GetLayoutHandlers() []LayoutHandler { return nil }
func (m *handshakeRouteMatch) GetMiddleware() []RouteMiddleware   { return nil }

type recordingRouter struct {
	mu       sync.Mutex
	lastPath string
	routes   map[string]RouteMatch
}

func (r *recordingRouter) Match(method, path string) (RouteMatch, bool) {
	r.mu.Lock()
	r.lastPath = path
	r.mu.Unlock()

	if r.routes == nil {
		return nil, false
	}
	m, ok := r.routes[path]
	return m, ok
}

func (r *recordingRouter) NotFound() PageHandler { return nil }

func (r *recordingRouter) LastPath() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lastPath
}

func TestServer_ServeHTTP_CanonicalRedirectPreservesQuery(t *testing.T) {
	s := New(DefaultServerConfig().WithDevMode())
	s.SetHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/foo//bar/?ignored=1", nil)
	req.URL.Path = "/foo//bar/"
	req.URL.RawQuery = "x=1&y=2"
	rr := httptest.NewRecorder()

	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusPermanentRedirect {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusPermanentRedirect)
	}
	if loc := rr.Header().Get("Location"); loc != "/foo/bar?x=1&y=2" {
		t.Fatalf("Location = %q, want %q", loc, "/foo/bar?x=1&y=2")
	}
}

func TestServer_PageHandler_DoesNotServeInternalRoutes(t *testing.T) {
	s := New(DefaultServerConfig().WithDevMode())
	s.SetHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/_vango/client.js", nil)
	s.PageHandler().ServeHTTP(rr, req)

	if rr.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusTeapot)
	}
}

func TestServer_GenerateCSRFToken_AndValidateCSRF_Signed(t *testing.T) {
	cfg := DefaultServerConfig().WithDevMode().WithCSRFSecret([]byte("0123456789abcdef0123456789abcdef"))
	s := New(cfg)

	token := s.GenerateCSRFToken()
	req := httptest.NewRequest(http.MethodGet, "/_vango/live", nil)
	req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: token})

	if ok := s.validateCSRF(req, token); !ok {
		t.Fatal("validateCSRF() = false, want true")
	}
	if ok := s.validateCSRF(req, token+"x"); ok {
		t.Fatal("validateCSRF() = true for mismatched token, want false")
	}
	if ok := s.validateCSRF(req, "not-base64"); ok {
		t.Fatal("validateCSRF() = true for invalid base64, want false")
	}
}

func TestServer_SetCSRFCookie_SecureHeuristic(t *testing.T) {
	cfg := DefaultServerConfig().WithDevMode().WithAddress("example.com:443").WithCSRFSecret([]byte("0123456789abcdef0123456789abcdef"))
	s := New(cfg)

	token := s.GenerateCSRFToken()
	rr := httptest.NewRecorder()
	s.SetCSRFCookie(rr, token)

	res := rr.Result()
	cookies := res.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
	if cookies[0].Name != CSRFCookieName {
		t.Fatalf("cookie name = %q, want %q", cookies[0].Name, CSRFCookieName)
	}
	if !cookies[0].Secure {
		t.Fatalf("cookie Secure = false, want true")
	}
}

func TestServer_HandleWebSocket_HandshakeErrors(t *testing.T) {
	s := New(DefaultServerConfig().WithDevMode())
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)

	conn := dialWS(t, wsURL(t, ts.URL, "/_vango/live?path=/"), nil)

	// Too short to include FrameHeaderSize => server should reply with handshake error then close.
	if err := conn.WriteMessage(websocket.BinaryMessage, []byte{0x01, 0x02}); err != nil {
		t.Fatalf("write invalid handshake failed: %v", err)
	}

	hello := readServerHello(t, conn)
	if hello.Status != protocol.HandshakeInvalidFormat {
		t.Fatalf("status = %v, want %v", hello.Status, protocol.HandshakeInvalidFormat)
	}
}

func TestServer_HandleWebSocket_AuthErrorRejectsHandshake(t *testing.T) {
	s := New(DefaultServerConfig().WithDevMode())
	s.SetAuthFunc(func(r *http.Request) (any, error) {
		return nil, http.ErrNoCookie
	})
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)

	conn := dialWS(t, wsURL(t, ts.URL, "/_vango/live?path=/"), nil)
	writeHandshake(t, conn, protocol.NewClientHello(""))
	hello := readServerHello(t, conn)

	if hello.Status != protocol.HandshakeNotAuthorized {
		t.Fatalf("status = %v, want %v", hello.Status, protocol.HandshakeNotAuthorized)
	}
}

func TestServer_HandleWebSocket_MaxSessionsReturnsServerBusy(t *testing.T) {
	cfg := DefaultServerConfig().WithDevMode().WithMaxSessions(1)
	s := New(cfg)
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	t.Cleanup(func() { s.Sessions().Shutdown() })

	c1 := dialWS(t, wsURL(t, ts.URL, "/_vango/live?path=/"), nil)
	writeHandshake(t, c1, protocol.NewClientHello(""))
	h1 := readServerHello(t, c1)
	if h1.Status != protocol.HandshakeOK {
		t.Fatalf("first handshake status = %v, want %v", h1.Status, protocol.HandshakeOK)
	}

	c2 := dialWS(t, wsURL(t, ts.URL, "/_vango/live?path=/"), nil)
	writeHandshake(t, c2, protocol.NewClientHello(""))
	h2 := readServerHello(t, c2)
	if h2.Status != protocol.HandshakeServerBusy {
		t.Fatalf("second handshake status = %v, want %v", h2.Status, protocol.HandshakeServerBusy)
	}
}

func TestServer_HandleWebSocket_ResumeExpiredFallsBackToNewSession(t *testing.T) {
	cfg := DefaultServerConfig().WithDevMode().WithResumeWindow(10 * time.Millisecond)
	s := New(cfg)
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	t.Cleanup(func() { s.Sessions().Shutdown() })

	c1 := dialWS(t, wsURL(t, ts.URL, "/_vango/live?path=/"), nil)
	writeHandshake(t, c1, protocol.NewClientHello(""))
	h1 := readServerHello(t, c1)
	if h1.Status != protocol.HandshakeOK || h1.SessionID == "" {
		t.Fatalf("handshake1 status=%v sessionID=%q, want OK and non-empty", h1.Status, h1.SessionID)
	}

	sess1 := getSessionEventually(t, s.Sessions(), h1.SessionID)
	waitForEventLoopStarted(t, sess1)
	_ = c1.Close()
	waitForDetached(t, sess1)

	// Allow the resume window to expire.
	time.Sleep(50 * time.Millisecond)

	c2 := dialWS(t, wsURL(t, ts.URL, "/_vango/live?path=/"), nil)
	h2hello := protocol.NewClientHello("")
	h2hello.SessionID = h1.SessionID
	writeHandshake(t, c2, h2hello)
	h2 := readServerHello(t, c2)

	if h2.Status != protocol.HandshakeOK {
		t.Fatalf("handshake2 status=%v, want %v", h2.Status, protocol.HandshakeOK)
	}
	if h2.SessionID == "" || h2.SessionID == h1.SessionID {
		t.Fatalf("handshake2 sessionID=%q, want new non-empty session ID", h2.SessionID)
	}
}

func TestServer_HandleWebSocket_RouterMountsInitialPathAndSanitizesInternal(t *testing.T) {
	cfg := DefaultServerConfig().WithDevMode()
	s := New(cfg)

	r := &recordingRouter{
		routes: map[string]RouteMatch{
			"/": &handshakeRouteMatch{
				params: map[string]string{},
				page: func(ctx Ctx, params any) Component {
					return staticComponent{node: &vdom.VNode{Kind: vdom.KindElement, Tag: "div"}}
				},
			},
		},
	}
	s.SetRouter(r)

	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	t.Cleanup(func() { s.Sessions().Shutdown() })

	// The server should never attempt to mount internal endpoints as page routes.
	conn := dialWS(t, wsURL(t, ts.URL, "/_vango/live?path=/_vango/ws"), nil)
	writeHandshake(t, conn, protocol.NewClientHello(""))
	hello := readServerHello(t, conn)
	if hello.Status != protocol.HandshakeOK {
		t.Fatalf("handshake status=%v, want %v", hello.Status, protocol.HandshakeOK)
	}

	sess := getSessionEventually(t, s.Sessions(), hello.SessionID)
	waitForEventLoopStarted(t, sess)
	if sess.root == nil || sess.currentTree == nil {
		t.Fatal("expected root component to be mounted")
	}
	if sess.CurrentRoute != "/" {
		t.Fatalf("CurrentRoute=%q, want %q", sess.CurrentRoute, "/")
	}
	if got := r.LastPath(); got != "/" {
		t.Fatalf("router Match path=%q, want %q", got, "/")
	}
}
