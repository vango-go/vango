package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vango-go/vango/pkg/auth"
	"github.com/vango-go/vango/pkg/protocol"
	"github.com/vango-go/vango/pkg/vdom"
)

// setupServerWithRouter creates a test server with a minimal router that mounts
// a simple page component. This is needed for resume tests because RebuildHandlers
// requires a root component to be mounted.
func setupServerWithRouter(cfg *ServerConfig) *Server {
	s := New(cfg)
	s.SetRouter(&authTestRouter{})
	return s
}

// authTestRouter is a minimal router for auth resume tests.
type authTestRouter struct{}

func (r *authTestRouter) Match(method, path string) (RouteMatch, bool) {
	return &authTestRouteMatch{params: map[string]string{}}, true
}

func (r *authTestRouter) NotFound() PageHandler { return nil }

type authTestRouteMatch struct {
	params map[string]string
}

func (m *authTestRouteMatch) GetParams() map[string]string { return m.params }
func (m *authTestRouteMatch) GetPageHandler() PageHandler {
	return func(ctx Ctx, params any) Component {
		return staticComponent{node: &vdom.VNode{Kind: vdom.KindElement, Tag: "div"}}
	}
}
func (m *authTestRouteMatch) GetLayoutHandlers() []LayoutHandler { return nil }
func (m *authTestRouteMatch) GetMiddleware() []RouteMiddleware   { return nil }

// =============================================================================
// Resume Auth Revalidation Tests
// =============================================================================

// TestResumeRejectsWhenAuthFuncFailsForAuthenticatedSession tests that when a
// session was previously authenticated (has presence flag) but authFunc now
// fails, the resume is rejected with HandshakeNotAuthorized.
func TestResumeRejectsWhenAuthFuncFailsForAuthenticatedSession(t *testing.T) {
	// Start with authFunc that succeeds
	var authShouldFail atomic.Bool
	cfg := DefaultServerConfig().WithDevMode().WithResumeWindow(5 * time.Second)
	cfg.OnSessionStart = func(httpCtx context.Context, session *Session) {
		// Set auth presence flag (simulates auth.Set)
		session.Set(DefaultAuthSessionKey, "test-user")
		session.Set(DefaultAuthSessionKey+":present", true)
	}
	s := setupServerWithRouter(cfg)
	s.SetAuthFunc(func(r *http.Request) (any, error) {
		if authShouldFail.Load() {
			return nil, errors.New("auth expired")
		}
		return "test-user", nil
	})

	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	t.Cleanup(func() { s.Sessions().Shutdown() })

	// First connection - should succeed
	c1 := dialWS(t, wsURL(t, ts.URL, "/_vango/live?path=/"), nil)
	writeHandshake(t, c1, protocol.NewClientHello(""))
	h1 := readServerHello(t, c1)
	if h1.Status != protocol.HandshakeOK {
		t.Fatalf("initial handshake status=%v, want %v", h1.Status, protocol.HandshakeOK)
	}
	sessionID := h1.SessionID

	// Wait for session to be ready
	sess := getSessionEventually(t, s.Sessions(), sessionID)
	waitForEventLoopStarted(t, sess)

	// Close connection to trigger detach
	_ = c1.Close()
	waitForDetached(t, sess)

	// Now make authFunc fail
	authShouldFail.Store(true)

	// Try to resume - should be rejected because session was authenticated
	// but auth is no longer valid
	c2 := dialWS(t, wsURL(t, ts.URL, "/_vango/live?path=/"), nil)
	resumeHello := protocol.NewClientHello("")
	resumeHello.SessionID = sessionID
	writeHandshake(t, c2, resumeHello)
	h2 := readServerHello(t, c2)

	if h2.Status != protocol.HandshakeNotAuthorized {
		t.Fatalf("resume handshake status=%v, want %v (HandshakeNotAuthorized)", h2.Status, protocol.HandshakeNotAuthorized)
	}
	if !h2.AuthReasonSet {
		t.Fatal("expected auth reason to be set for resume rejection")
	}
	if h2.AuthReason != uint8(AuthExpiredResumeRehydrateFailed) {
		t.Fatalf("auth reason=%v, want %v", h2.AuthReason, uint8(AuthExpiredResumeRehydrateFailed))
	}
}

// TestResumeSucceedsWhenAuthFuncSucceedsForAuthenticatedSession tests that
// resume succeeds when authFunc still returns a valid user.
func TestResumeSucceedsWhenAuthFuncSucceedsForAuthenticatedSession(t *testing.T) {
	cfg := DefaultServerConfig().WithDevMode().WithResumeWindow(5 * time.Second)
	cfg.OnSessionStart = func(httpCtx context.Context, session *Session) {
		session.Set(DefaultAuthSessionKey, "test-user")
		session.Set(DefaultAuthSessionKey+":present", true)
	}
	s := setupServerWithRouter(cfg)
	s.SetAuthFunc(func(r *http.Request) (any, error) {
		return "test-user", nil
	})

	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	t.Cleanup(func() { s.Sessions().Shutdown() })

	// First connection
	c1 := dialWS(t, wsURL(t, ts.URL, "/_vango/live?path=/"), nil)
	writeHandshake(t, c1, protocol.NewClientHello(""))
	h1 := readServerHello(t, c1)
	if h1.Status != protocol.HandshakeOK {
		t.Fatalf("initial handshake status=%v, want %v", h1.Status, protocol.HandshakeOK)
	}
	sessionID := h1.SessionID

	sess := getSessionEventually(t, s.Sessions(), sessionID)
	waitForEventLoopStarted(t, sess)
	_ = c1.Close()
	waitForDetached(t, sess)

	// Resume - should succeed because authFunc still returns valid user
	c2 := dialWS(t, wsURL(t, ts.URL, "/_vango/live?path=/"), nil)
	resumeHello := protocol.NewClientHello("")
	resumeHello.SessionID = sessionID
	writeHandshake(t, c2, resumeHello)
	h2 := readServerHello(t, c2)

	if h2.Status != protocol.HandshakeOK {
		t.Fatalf("resume handshake status=%v, want %v", h2.Status, protocol.HandshakeOK)
	}
	if h2.SessionID != sessionID {
		t.Fatalf("resume sessionID=%q, want %q (same session)", h2.SessionID, sessionID)
	}
}

// TestResumeSucceedsForGuestSessionWhenAuthFuncFails tests that guest sessions
// (without auth presence flag) can still resume even if authFunc fails.
func TestResumeSucceedsForGuestSessionWhenAuthFuncFails(t *testing.T) {
	var authShouldFail atomic.Bool

	cfg := DefaultServerConfig().WithDevMode().WithResumeWindow(5 * time.Second)
	// No OnSessionStart - guest session
	s := setupServerWithRouter(cfg)
	s.SetAuthFunc(func(r *http.Request) (any, error) {
		if authShouldFail.Load() {
			return nil, errors.New("no auth")
		}
		return nil, nil // Guest - no user
	})

	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	t.Cleanup(func() { s.Sessions().Shutdown() })

	// First connection as guest
	c1 := dialWS(t, wsURL(t, ts.URL, "/_vango/live?path=/"), nil)
	writeHandshake(t, c1, protocol.NewClientHello(""))
	h1 := readServerHello(t, c1)
	if h1.Status != protocol.HandshakeOK {
		t.Fatalf("initial handshake status=%v, want %v", h1.Status, protocol.HandshakeOK)
	}
	sessionID := h1.SessionID

	sess := getSessionEventually(t, s.Sessions(), sessionID)
	waitForEventLoopStarted(t, sess)
	_ = c1.Close()
	waitForDetached(t, sess)

	// Make authFunc fail
	authShouldFail.Store(true)

	// Resume - should succeed because session was guest (no auth to revalidate)
	c2 := dialWS(t, wsURL(t, ts.URL, "/_vango/live?path=/"), nil)
	resumeHello := protocol.NewClientHello("")
	resumeHello.SessionID = sessionID
	writeHandshake(t, c2, resumeHello)
	h2 := readServerHello(t, c2)

	if h2.Status != protocol.HandshakeOK {
		t.Fatalf("resume handshake status=%v, want %v (guest resume should succeed)", h2.Status, protocol.HandshakeOK)
	}
}

// =============================================================================
// Auto-Hydration from authFunc Tests
// =============================================================================

// TestResumeAutoHydratesUserFromAuthFunc tests that when authFunc returns a
// user on resume and the session user is nil (e.g., restored from persistence),
// the user is automatically set in the session.
func TestResumeAutoHydratesUserFromAuthFunc(t *testing.T) {
	cfg := DefaultServerConfig().WithDevMode().WithResumeWindow(5 * time.Second)
	cfg.OnSessionStart = func(httpCtx context.Context, session *Session) {
		// Set presence flag but NOT the user (simulates persistence restore)
		session.Set(DefaultAuthSessionKey+":present", true)
		// Deliberately don't set DefaultAuthSessionKey to simulate restored session
	}
	s := setupServerWithRouter(cfg)
	s.SetAuthFunc(func(r *http.Request) (any, error) {
		return "hydrated-user", nil
	})

	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	t.Cleanup(func() { s.Sessions().Shutdown() })

	// First connection
	c1 := dialWS(t, wsURL(t, ts.URL, "/_vango/live?path=/"), nil)
	writeHandshake(t, c1, protocol.NewClientHello(""))
	h1 := readServerHello(t, c1)
	if h1.Status != protocol.HandshakeOK {
		t.Fatalf("initial handshake status=%v, want %v", h1.Status, protocol.HandshakeOK)
	}

	sess := getSessionEventually(t, s.Sessions(), h1.SessionID)
	waitForEventLoopStarted(t, sess)

	// Clear the user to simulate persistence restore scenario
	sess.Delete(DefaultAuthSessionKey)
	if sess.Get(DefaultAuthSessionKey) != nil {
		t.Fatal("user should be nil before resume")
	}

	_ = c1.Close()
	waitForDetached(t, sess)

	// Resume - authFunc should auto-hydrate the user
	c2 := dialWS(t, wsURL(t, ts.URL, "/_vango/live?path=/"), nil)
	resumeHello := protocol.NewClientHello("")
	resumeHello.SessionID = h1.SessionID
	writeHandshake(t, c2, resumeHello)
	h2 := readServerHello(t, c2)

	if h2.Status != protocol.HandshakeOK {
		t.Fatalf("resume handshake status=%v, want %v", h2.Status, protocol.HandshakeOK)
	}

	// Verify user was hydrated
	sess2 := s.Sessions().Get(h1.SessionID)
	if sess2 == nil {
		t.Fatal("session not found after resume")
	}
	user := sess2.Get(DefaultAuthSessionKey)
	if user != "hydrated-user" {
		t.Fatalf("session user=%v, want 'hydrated-user' (auto-hydrated from authFunc)", user)
	}
}

// =============================================================================
// OnSessionResume Hook Tests
// =============================================================================

// TestOnSessionResumeHookIsCalled tests that the OnSessionResume hook is called
// during resume.
func TestOnSessionResumeHookIsCalled(t *testing.T) {
	var hookCalled atomic.Bool
	var hookSessionID string

	cfg := DefaultServerConfig().WithDevMode().WithResumeWindow(5 * time.Second)
	cfg.OnSessionResume = func(httpCtx context.Context, session *Session) error {
		hookCalled.Store(true)
		hookSessionID = session.ID
		return nil
	}
	s := setupServerWithRouter(cfg)

	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	t.Cleanup(func() { s.Sessions().Shutdown() })

	// First connection
	c1 := dialWS(t, wsURL(t, ts.URL, "/_vango/live?path=/"), nil)
	writeHandshake(t, c1, protocol.NewClientHello(""))
	h1 := readServerHello(t, c1)
	if h1.Status != protocol.HandshakeOK {
		t.Fatalf("initial handshake status=%v, want %v", h1.Status, protocol.HandshakeOK)
	}

	sess := getSessionEventually(t, s.Sessions(), h1.SessionID)
	waitForEventLoopStarted(t, sess)
	_ = c1.Close()
	waitForDetached(t, sess)

	// Reset hook flag
	hookCalled.Store(false)

	// Resume
	c2 := dialWS(t, wsURL(t, ts.URL, "/_vango/live?path=/"), nil)
	resumeHello := protocol.NewClientHello("")
	resumeHello.SessionID = h1.SessionID
	writeHandshake(t, c2, resumeHello)
	h2 := readServerHello(t, c2)

	if h2.Status != protocol.HandshakeOK {
		t.Fatalf("resume handshake status=%v, want %v", h2.Status, protocol.HandshakeOK)
	}

	if !hookCalled.Load() {
		t.Fatal("OnSessionResume hook was not called")
	}
	if hookSessionID != h1.SessionID {
		t.Fatalf("OnSessionResume sessionID=%q, want %q", hookSessionID, h1.SessionID)
	}
}

// TestOnSessionResumeCanRejectResume tests that OnSessionResume can reject
// resume by returning an error when session was previously authenticated.
func TestOnSessionResumeCanRejectResume(t *testing.T) {
	var shouldReject atomic.Bool

	cfg := DefaultServerConfig().WithDevMode().WithResumeWindow(5 * time.Second)
	cfg.OnSessionStart = func(httpCtx context.Context, session *Session) {
		// Mark as authenticated
		session.Set(DefaultAuthSessionKey+":present", true)
	}
	cfg.OnSessionResume = func(httpCtx context.Context, session *Session) error {
		if shouldReject.Load() {
			return errors.New("auth validation failed")
		}
		return nil
	}
	s := setupServerWithRouter(cfg)

	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	t.Cleanup(func() { s.Sessions().Shutdown() })

	// First connection
	c1 := dialWS(t, wsURL(t, ts.URL, "/_vango/live?path=/"), nil)
	writeHandshake(t, c1, protocol.NewClientHello(""))
	h1 := readServerHello(t, c1)
	if h1.Status != protocol.HandshakeOK {
		t.Fatalf("initial handshake status=%v, want %v", h1.Status, protocol.HandshakeOK)
	}

	sess := getSessionEventually(t, s.Sessions(), h1.SessionID)
	waitForEventLoopStarted(t, sess)
	_ = c1.Close()
	waitForDetached(t, sess)

	// Make hook reject
	shouldReject.Store(true)

	// Resume - should be rejected
	c2 := dialWS(t, wsURL(t, ts.URL, "/_vango/live?path=/"), nil)
	resumeHello := protocol.NewClientHello("")
	resumeHello.SessionID = h1.SessionID
	writeHandshake(t, c2, resumeHello)
	h2 := readServerHello(t, c2)

	if h2.Status != protocol.HandshakeNotAuthorized {
		t.Fatalf("resume handshake status=%v, want %v (OnSessionResume rejected)", h2.Status, protocol.HandshakeNotAuthorized)
	}
	if !h2.AuthReasonSet {
		t.Fatal("expected auth reason to be set for resume rejection")
	}
	if h2.AuthReason != uint8(AuthExpiredResumeRehydrateFailed) {
		t.Fatalf("auth reason=%v, want %v", h2.AuthReason, uint8(AuthExpiredResumeRehydrateFailed))
	}
}

// TestOnSessionResumeCanRehydrateUser tests that OnSessionResume can rehydrate
// the user into the session.
func TestOnSessionResumeCanRehydrateUser(t *testing.T) {
	cfg := DefaultServerConfig().WithDevMode().WithResumeWindow(5 * time.Second)
	cfg.OnSessionResume = func(httpCtx context.Context, session *Session) error {
		// Rehydrate user
		session.Set(DefaultAuthSessionKey, "rehydrated-user")
		session.Set(DefaultAuthSessionKey+":present", true)
		return nil
	}
	s := setupServerWithRouter(cfg)

	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	t.Cleanup(func() { s.Sessions().Shutdown() })

	// First connection
	c1 := dialWS(t, wsURL(t, ts.URL, "/_vango/live?path=/"), nil)
	writeHandshake(t, c1, protocol.NewClientHello(""))
	h1 := readServerHello(t, c1)
	if h1.Status != protocol.HandshakeOK {
		t.Fatalf("initial handshake status=%v, want %v", h1.Status, protocol.HandshakeOK)
	}

	sess := getSessionEventually(t, s.Sessions(), h1.SessionID)
	waitForEventLoopStarted(t, sess)
	_ = c1.Close()
	waitForDetached(t, sess)

	// Resume
	c2 := dialWS(t, wsURL(t, ts.URL, "/_vango/live?path=/"), nil)
	resumeHello := protocol.NewClientHello("")
	resumeHello.SessionID = h1.SessionID
	writeHandshake(t, c2, resumeHello)
	h2 := readServerHello(t, c2)

	if h2.Status != protocol.HandshakeOK {
		t.Fatalf("resume handshake status=%v, want %v", h2.Status, protocol.HandshakeOK)
	}

	// Verify user was rehydrated
	sess2 := s.Sessions().Get(h1.SessionID)
	user := sess2.Get(DefaultAuthSessionKey)
	if user != "rehydrated-user" {
		t.Fatalf("session user=%v, want 'rehydrated-user'", user)
	}
}

func TestResumeSucceedsWhenPrincipalRehydrated(t *testing.T) {
	cfg := DefaultServerConfig().WithDevMode().WithResumeWindow(5 * time.Second)
	cfg.OnSessionStart = func(httpCtx context.Context, session *Session) {
		auth.SetPrincipal(session, auth.Principal{
			ID:              "user-123",
			ExpiresAtUnixMs: time.Now().Add(time.Hour).UnixMilli(),
		})
	}
	cfg.OnSessionResume = func(httpCtx context.Context, session *Session) error {
		auth.SetPrincipal(session, auth.Principal{
			ID:              "user-123",
			ExpiresAtUnixMs: time.Now().Add(time.Hour).UnixMilli(),
		})
		return nil
	}
	s := setupServerWithRouter(cfg)

	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	t.Cleanup(func() { s.Sessions().Shutdown() })

	c1 := dialWS(t, wsURL(t, ts.URL, "/_vango/live?path=/"), nil)
	writeHandshake(t, c1, protocol.NewClientHello(""))
	h1 := readServerHello(t, c1)
	if h1.Status != protocol.HandshakeOK {
		t.Fatalf("initial handshake status=%v, want %v", h1.Status, protocol.HandshakeOK)
	}

	sess := getSessionEventually(t, s.Sessions(), h1.SessionID)
	waitForEventLoopStarted(t, sess)
	_ = c1.Close()
	waitForDetached(t, sess)

	c2 := dialWS(t, wsURL(t, ts.URL, "/_vango/live?path=/"), nil)
	resumeHello := protocol.NewClientHello("")
	resumeHello.SessionID = h1.SessionID
	writeHandshake(t, c2, resumeHello)
	h2 := readServerHello(t, c2)

	if h2.Status != protocol.HandshakeOK {
		t.Fatalf("resume handshake status=%v, want %v", h2.Status, protocol.HandshakeOK)
	}
}

// =============================================================================
// Session Serialization Tests
// =============================================================================

// TestSessionSerializeSkipsRuntimeAuthKeys tests that session serialization does not
// include runtime-only auth keys or the user object, but preserves markers.
func TestSessionSerializeSkipsRuntimeAuthKeys(t *testing.T) {
	s := NewMockSession()
	s.ID = "test-session"

	// Set user and presence flag
	s.Set(DefaultAuthSessionKey, map[string]any{"id": "123", "name": "Test User"})
	s.Set(DefaultAuthSessionKey+":present", true)
	s.Set(auth.SessionKeyHadAuth, true)
	s.Set(auth.SessionKeyPrincipal, auth.Principal{ID: "123"})
	s.Set(auth.SessionKeyExpiryUnixMs, int64(123))
	s.Set("other-data", "should-be-serialized")

	// Serialize
	data, err := s.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error: %v", err)
	}

	// Deserialize into new session
	s2 := NewMockSession()
	if err := s2.Deserialize(data); err != nil {
		t.Fatalf("Deserialize() error: %v", err)
	}

	// User object should NOT be present (skipped during serialization)
	if s2.Get(DefaultAuthSessionKey) != nil {
		t.Fatal("DefaultAuthSessionKey should not be serialized")
	}

	// Presence flag SHOULD be present (not skipped)
	if s2.Get(DefaultAuthSessionKey+":present") != true {
		t.Fatal("presence flag should be serialized")
	}
	if s2.Get(auth.SessionKeyHadAuth) != true {
		t.Fatal("had_auth should be serialized")
	}

	// Runtime-only keys should NOT be present
	if s2.Get(auth.SessionKeyPrincipal) != nil {
		t.Fatal("SessionKeyPrincipal should not be serialized")
	}
	if s2.Get(auth.SessionKeyExpiryUnixMs) != nil {
		t.Fatal("SessionKeyExpiryUnixMs should not be serialized")
	}

	// Other data should be present
	if s2.Get("other-data") != "should-be-serialized" {
		t.Fatal("other-data should be serialized")
	}
}

// =============================================================================
// Ghost Auth State Rejection Tests
// =============================================================================

// TestResumeRejectsGhostAuthState tests that resume is rejected when session
// was authenticated (presence flag set) but OnSessionResume doesn't rehydrate
// the user. This prevents "ghost authenticated" state where the presence flag
// is true but there's no actual user object.
func TestResumeRejectsGhostAuthState(t *testing.T) {
	cfg := DefaultServerConfig().WithDevMode().WithResumeWindow(5 * time.Second)
	cfg.OnSessionStart = func(httpCtx context.Context, session *Session) {
		// Mark as authenticated with presence flag
		session.Set(DefaultAuthSessionKey+":present", true)
		session.Set(DefaultAuthSessionKey, "original-user")
	}
	cfg.OnSessionResume = func(httpCtx context.Context, session *Session) error {
		// Deliberately DON'T rehydrate the user - simulates bug in app code
		// This should cause resume to be rejected
		return nil
	}
	s := setupServerWithRouter(cfg)

	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	t.Cleanup(func() { s.Sessions().Shutdown() })

	// First connection
	c1 := dialWS(t, wsURL(t, ts.URL, "/_vango/live?path=/"), nil)
	writeHandshake(t, c1, protocol.NewClientHello(""))
	h1 := readServerHello(t, c1)
	if h1.Status != protocol.HandshakeOK {
		t.Fatalf("initial handshake status=%v, want %v", h1.Status, protocol.HandshakeOK)
	}

	sess := getSessionEventually(t, s.Sessions(), h1.SessionID)
	waitForEventLoopStarted(t, sess)

	// Clear the user to simulate what would happen if user wasn't rehydrated
	// (In real scenario, this happens because user object isn't serialized)
	sess.Delete(DefaultAuthSessionKey)

	_ = c1.Close()
	waitForDetached(t, sess)

	// Resume - OnSessionResume doesn't set user, so resume should be rejected
	c2 := dialWS(t, wsURL(t, ts.URL, "/_vango/live?path=/"), nil)
	resumeHello := protocol.NewClientHello("")
	resumeHello.SessionID = h1.SessionID
	writeHandshake(t, c2, resumeHello)
	h2 := readServerHello(t, c2)

	if h2.Status != protocol.HandshakeNotAuthorized {
		t.Fatalf("resume handshake status=%v, want %v (ghost auth should be rejected)", h2.Status, protocol.HandshakeNotAuthorized)
	}
	if !h2.AuthReasonSet {
		t.Fatal("expected auth reason to be set for resume rejection")
	}
	if h2.AuthReason != uint8(AuthExpiredResumeRehydrateFailed) {
		t.Fatalf("auth reason=%v, want %v", h2.AuthReason, uint8(AuthExpiredResumeRehydrateFailed))
	}
}

// TestResumeSucceedsWhenOnSessionResumeRehydratesUser tests that resume succeeds
// when OnSessionResume properly rehydrates the user.
func TestResumeSucceedsWhenOnSessionResumeRehydratesUser(t *testing.T) {
	cfg := DefaultServerConfig().WithDevMode().WithResumeWindow(5 * time.Second)
	cfg.OnSessionStart = func(httpCtx context.Context, session *Session) {
		session.Set(DefaultAuthSessionKey+":present", true)
		session.Set(DefaultAuthSessionKey, "original-user")
	}
	cfg.OnSessionResume = func(httpCtx context.Context, session *Session) error {
		// Properly rehydrate the user
		session.Set(DefaultAuthSessionKey, "rehydrated-user")
		return nil
	}
	s := setupServerWithRouter(cfg)

	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	t.Cleanup(func() { s.Sessions().Shutdown() })

	// First connection
	c1 := dialWS(t, wsURL(t, ts.URL, "/_vango/live?path=/"), nil)
	writeHandshake(t, c1, protocol.NewClientHello(""))
	h1 := readServerHello(t, c1)
	if h1.Status != protocol.HandshakeOK {
		t.Fatalf("initial handshake status=%v, want %v", h1.Status, protocol.HandshakeOK)
	}

	sess := getSessionEventually(t, s.Sessions(), h1.SessionID)
	waitForEventLoopStarted(t, sess)

	// Clear the user to simulate persistence restore scenario
	sess.Delete(DefaultAuthSessionKey)

	_ = c1.Close()
	waitForDetached(t, sess)

	// Resume - OnSessionResume rehydrates user, so resume should succeed
	c2 := dialWS(t, wsURL(t, ts.URL, "/_vango/live?path=/"), nil)
	resumeHello := protocol.NewClientHello("")
	resumeHello.SessionID = h1.SessionID
	writeHandshake(t, c2, resumeHello)
	h2 := readServerHello(t, c2)

	if h2.Status != protocol.HandshakeOK {
		t.Fatalf("resume handshake status=%v, want %v", h2.Status, protocol.HandshakeOK)
	}

	// Verify user was rehydrated
	sess2 := s.Sessions().Get(h1.SessionID)
	if sess2 == nil {
		t.Fatal("session not found after resume")
	}
	user := sess2.Get(DefaultAuthSessionKey)
	if user != "rehydrated-user" {
		t.Fatalf("session user=%v, want 'rehydrated-user'", user)
	}
}
