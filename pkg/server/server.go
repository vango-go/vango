package server

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vango-go/vango/pkg/auth"
	"github.com/vango-go/vango/pkg/protocol"
	"github.com/vango-go/vango/pkg/routepath"
)

// Server is the main HTTP/WebSocket server for Vango.
type Server struct {
	// Session management
	sessions *SessionManager

	// HTTP handler (router placeholder for Phase 7)
	handler http.Handler

	// Root component factory
	rootComponent func() Component

	// Router for route-based navigation (Phase 7: Routing)
	// When set, this router is passed to all new sessions to enable
	// client-side navigation via EventNavigate and ctx.Navigate().
	router Router

	// Configuration
	config *ServerConfig

	// Trusted proxy matcher for forwarded headers
	trustedProxies *proxyMatcher

	// Cookie policy helper
	cookiePolicy *CookiePolicy

	// WebSocket upgrader
	upgrader websocket.Upgrader

	// Middleware
	middleware []Middleware

	// Authentication
	authFunc func(*http.Request) (any, error)

	// CSRF
	csrfSecret []byte

	// HTTP server
	httpServer *http.Server

	// Logger
	logger *slog.Logger
}

// Middleware is a function that wraps an HTTP handler.
type Middleware func(http.Handler) http.Handler

// New creates a new Server with the given configuration.
func New(config *ServerConfig) *Server {
	if config == nil {
		config = DefaultServerConfig()
	} else {
		// Fill in defaults for any unset fields
		defaults := DefaultServerConfig()
		if config.Address == "" {
			config.Address = defaults.Address
		}
		if config.ReadBufferSize == 0 {
			config.ReadBufferSize = defaults.ReadBufferSize
		}
		if config.WriteBufferSize == 0 {
			config.WriteBufferSize = defaults.WriteBufferSize
		}
		if config.CheckOrigin == nil {
			config.CheckOrigin = defaults.CheckOrigin
		}
		if config.SessionConfig == nil {
			config.SessionConfig = defaults.SessionConfig
		}
		if config.ShutdownTimeout == 0 {
			config.ShutdownTimeout = defaults.ShutdownTimeout
		}
		if config.ReadHeaderTimeout == 0 {
			config.ReadHeaderTimeout = defaults.ReadHeaderTimeout
		}
		if config.ReadTimeout == 0 {
			config.ReadTimeout = defaults.ReadTimeout
		}
		if config.WriteTimeout == 0 {
			config.WriteTimeout = defaults.WriteTimeout
		}
		if config.IdleTimeout == 0 {
			config.IdleTimeout = defaults.IdleTimeout
		}
	}

	normalizeAuthCheckConfig(config.SessionConfig.AuthCheck)

	// Apply debug flags before initialization to keep logging consistent.
	DebugMode = config.DebugMode
	auth.DebugMode = config.DebugMode

	logger := slog.Default().With("component", "server")

	for _, warning := range config.GetConfigWarnings() {
		logger.Warn("config warning", "warning", warning)
	}
	if err := config.ValidateConfig(); err != nil {
		logger.Error("config validation failed", "error", err)
	}

	// Build session limits from config (use defaults for unset values)
	limits := DefaultSessionLimits()
	if config.SessionLimits != nil {
		limits = &SessionLimits{
			MaxSessions:         config.SessionLimits.MaxSessions,
			MaxMemoryPerSession: config.SessionLimits.MaxMemoryPerSession,
			MaxTotalMemory:      config.SessionLimits.MaxTotalMemory,
		}
	} else {
		if config.MaxSessions > 0 {
			limits.MaxSessions = config.MaxSessions
		}
		if config.MaxMemoryPerSession > 0 {
			limits.MaxMemoryPerSession = config.MaxMemoryPerSession
		}
	}

	// Phase 12: Build persistence options
	var persistOpts *SessionManagerOptions
	if config.SessionStore != nil || config.ResumeWindow > 0 || config.MaxDetachedSessions > 0 || config.MaxSessionsPerIP > 0 {
		persistOpts = &SessionManagerOptions{
			SessionStore:        config.SessionStore,
			ResumeWindow:        config.ResumeWindow,
			MaxDetachedSessions: config.MaxDetachedSessions,
			MaxSessionsPerIP:    config.MaxSessionsPerIP,
			EvictOnIPLimit:      config.EvictOnIPLimit,
			PersistInterval:     config.PersistInterval,
		}
	}

	trustedProxies := newProxyMatcher(config.TrustedProxies, logger)
	s := &Server{
		sessions:       NewSessionManagerWithOptions(config.SessionConfig, limits, logger, persistOpts),
		config:         config,
		trustedProxies: trustedProxies,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  config.ReadBufferSize,
			WriteBufferSize: config.WriteBufferSize,
			CheckOrigin:     config.CheckOrigin,
		},
		csrfSecret:   config.CSRFSecret,
		cookiePolicy: newCookiePolicy(config, trustedProxies, logger),
		logger:       logger,
	}

	return s
}

// SetRootComponent sets the root component factory.
func (s *Server) SetRootComponent(factory func() Component) {
	s.rootComponent = factory
}

// SetHandler sets the HTTP handler for non-WebSocket requests.
func (s *Server) SetHandler(h http.Handler) {
	s.handler = h
}

// SetAuthFunc sets the authentication function.
func (s *Server) SetAuthFunc(fn func(*http.Request) (any, error)) {
	s.authFunc = fn
}

// SetRouter sets the router for route-based navigation.
// When set, this router is passed to all new sessions to enable:
//   - Client-side navigation via EventNavigate (link clicks, popstate)
//   - Programmatic navigation via ctx.Navigate()
//
// The router must implement the Router interface (defined in navigation.go).
// Use router.NewRouterAdapter to wrap a router.Router for use with Server.
//
// Example:
//
//	r := router.NewRouter()
//	r.AddPage("/", index.IndexPage)
//	r.AddPage("/about", about.AboutPage)
//	r.AddPage("/projects/:id", projects.ShowPage)
//	r.SetNotFound(notfound.NotFoundPage)
//
//	app.SetRouter(router.NewRouterAdapter(r))
func (s *Server) SetRouter(r Router) {
	s.router = r
}

// Router returns the current router, or nil if none is set.
func (s *Server) Router() Router {
	return s.router
}

// Use adds middleware to the server.
func (s *Server) Use(mw Middleware) {
	s.middleware = append(s.middleware, mw)
}

// =============================================================================
// HTTP Handler Interface (Phase 10)
// =============================================================================

// Handler returns an http.Handler for mounting in external routers.
// This is the primary integration point for ecosystem compatibility with
// Chi, Gorilla, Echo, stdlib mux, etc.
//
// The handler dispatches based on path:
//   - /_vango/ws, /_vango/live → WebSocket upgrade
//   - /_vango/* → Internal routes (future: CSRF, assets)
//   - /* → Page routes via SSR/handler
//
// Example:
//
//	r := chi.NewRouter()
//	r.Use(middleware.Logger)
//	r.Use(authMiddleware)
//	r.Handle("/*", app.Handler())
//	http.ListenAndServe(":3000", r)
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.ServeHTTP(w, r)
	})
}

// PageHandler returns an http.Handler for page routes only.
// Use when you want to handle /_vango/* routes separately.
func (s *Server) PageHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Apply middleware and serve
		handler := s.handler
		if handler == nil {
			handler = http.NotFoundHandler()
		}

		for i := len(s.middleware) - 1; i >= 0; i-- {
			handler = s.middleware[i](handler)
		}

		handler.ServeHTTP(w, r)
	})
}

// WebSocketHandler returns an http.Handler for WebSocket upgrade only.
// Use when you want fine-grained control over routing.
func (s *Server) WebSocketHandler() http.Handler {
	return http.HandlerFunc(s.HandleWebSocket)
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check for WebSocket upgrade
	if r.URL.Path == "/_vango/ws" || r.URL.Path == "/_vango/live" {
		s.HandleWebSocket(w, r)
		return
	}

	// Internal assets
	if r.URL.Path == "/_vango/client.js" {
		s.serveThinClient(w, r)
		return
	}

	// Per Section 1.2.4 (Path Canonicalization):
	// HTTP requests with non-canonical paths should redirect with 308 Permanent Redirect.
	// This ensures consistent URL handling and prevents duplicate content issues.
	rawPath := r.URL.EscapedPath()
	input := rawPath
	if r.URL.RawQuery != "" {
		input = rawPath + "?" + r.URL.RawQuery
	}
	if result, err := routepath.CanonicalizePath(input); err != nil {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	} else if result.Changed {
		// Build canonical URL.
		canonURL := result.Path
		if result.Query != "" {
			canonURL = result.Path + "?" + result.Query
		} else if r.URL.RawQuery != "" {
			canonURL = result.Path + "?" + r.URL.RawQuery
		}

		// 308 Permanent Redirect preserves the HTTP method (unlike 301).
		// This is important for POST/PUT/DELETE requests.
		http.Redirect(w, r, canonURL, http.StatusPermanentRedirect)
		return
	}

	// Apply middleware and serve
	handler := s.handler
	if handler == nil {
		handler = http.NotFoundHandler()
	}

	for i := len(s.middleware) - 1; i >= 0; i-- {
		handler = s.middleware[i](handler)
	}

	handler.ServeHTTP(w, r)
}

// HandleWebSocket handles WebSocket upgrade and connection.
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade to WebSocket
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("websocket upgrade failed", "error", err)
		return
	}

	// Set connection options
	conn.SetReadLimit(s.config.SessionConfig.MaxMessageSize)
	conn.SetReadDeadline(time.Now().Add(s.config.SessionConfig.HandshakeTimeout))

	// Wait for handshake
	_, msg, err := conn.ReadMessage()
	if err != nil {
		s.logger.Error("handshake read failed", "error", err)
		conn.Close()
		return
	}

	// Decode frame header first (consistent framing per spec)
	// Frame format: [type:1][flags:1][len:2][payload...]
	if len(msg) < protocol.FrameHeaderSize {
		s.sendHandshakeError(conn, protocol.HandshakeInvalidFormat)
		conn.Close()
		return
	}
	frameType := protocol.FrameType(msg[0])
	if frameType != protocol.FrameHandshake {
		s.logger.Error("handshake frame type mismatch", "got", frameType, "expected", protocol.FrameHandshake)
		s.sendHandshakeError(conn, protocol.HandshakeInvalidFormat)
		conn.Close()
		return
	}
	payloadLen := int(msg[2])<<8 | int(msg[3])
	if len(msg) < protocol.FrameHeaderSize+payloadLen {
		s.sendHandshakeError(conn, protocol.HandshakeInvalidFormat)
		conn.Close()
		return
	}
	payload := msg[protocol.FrameHeaderSize : protocol.FrameHeaderSize+payloadLen]

	// Parse client hello from payload
	hello, err := protocol.DecodeClientHello(payload)
	if err != nil {
		s.sendHandshakeError(conn, protocol.HandshakeInvalidFormat)
		conn.Close()
		return
	}

	// Validate CSRF if configured
	if s.csrfSecret != nil && !s.validateCSRF(r, hello.CSRFToken) {
		s.sendHandshakeError(conn, protocol.HandshakeInvalidCSRF)
		conn.Close()
		return
	}

	clientIP := s.clientIP(r)

	// ═══════════════════════════════════════════════════════════════════════════
	// SESSION RESUME (Phase 5)
	// Check if client has an existing session to resume. Uses soft remount
	// (RebuildHandlers) to preserve signal state while regenerating HIDs.
	// ═══════════════════════════════════════════════════════════════════════════
	var session *Session
	var isResume bool

	if hello.SessionID != "" {
		// Try active sessions first
		session = s.sessions.Get(hello.SessionID)
		if session != nil && !session.IsClosed() {
			isResume = true
		} else if s.sessions.HasPersistence() {
			// Try persistence store (server restart scenario)
			if restored, ok := s.sessions.OnSessionReconnect(hello.SessionID); ok {
				session = restored
				isResume = true
			}
		}

		// Validate resume window
		if isResume && session != nil {
			if time.Since(session.LastActive) > s.sessions.ResumeWindow() {
				s.logger.Info("session resume rejected: expired",
					"session_id", hello.SessionID,
					"last_active", session.LastActive)
				session.Close()
				s.sessions.Close(session.ID)
				session = nil
				isResume = false
				// Metrics: RecordResumeFailed("expired") can be called via hooks
			}
		}
	}

	if isResume && session != nil {
		if err := s.sessions.UpdateSessionIP(session, clientIP); err != nil {
			if err == ErrTooManySessionsFromIP {
				s.sendHandshakeError(conn, protocol.HandshakeLimitExceeded)
			} else {
				s.sendHandshakeError(conn, protocol.HandshakeInternalError)
			}
			conn.Close()
			return
		}

		// ═══════════════════════════════════════════════════════════════════════════
		// RESUME AUTH REVALIDATION
		// Check if auth is still valid. If the session was previously authenticated
		// but auth is now invalid, reject the resume.
		// ═══════════════════════════════════════════════════════════════════════════
		wasAuthenticated := auth.WasAuthenticated(session) || session.UserID != ""

		// First, run authFunc if set (validates auth cookie/token)
		var authValid bool
		if s.authFunc != nil {
			user, err := s.authFunc(r)
			if err == nil && user != nil {
				authValid = true
				// Auto-hydrate: set user into session if not already present
				// This handles the case where session was restored from persistence
				// (user object not serialized) and apps rely on authFunc without OnSessionResume.
				if session.Get(DefaultAuthSessionKey) == nil {
					auth.Set(session, user)
				}
			}
		} else {
			// No authFunc means we rely on OnSessionResume
			authValid = true
		}

		// Then run OnSessionResume hook to rehydrate auth data
		var resumeErr error
		if s.config.OnSessionResume != nil {
			resumeErr = s.config.OnSessionResume(r.Context(), session)
		}

		// Warn in dev mode if session was authenticated but no revalidation hooks are configured
		if wasAuthenticated && s.authFunc == nil && s.config.OnSessionResume == nil {
			if s.config.DevMode {
				s.logger.Warn("session resume without auth revalidation hooks configured",
					"session_id", hello.SessionID,
					"hint", "Configure authFunc or OnSessionResume to revalidate auth on resume")
			}
		}

		// If the session was previously authenticated but auth is now invalid, reject resume
		if wasAuthenticated && (!authValid || resumeErr != nil) {
			s.logger.Warn("session resume rejected: auth no longer valid",
				"session_id", hello.SessionID,
				"was_authenticated", wasAuthenticated,
				"auth_valid", authValid,
				"resume_error", resumeErr)
			s.sendHandshakeErrorWithReason(conn, protocol.HandshakeNotAuthorized, AuthExpiredResumeRehydrateFailed)
			conn.Close()
			// Clean up the session since auth failed
			session.Close()
			s.sessions.Close(session.ID)
			return
		}

		// If there was a resume error but session was not authenticated, log it but continue
		if resumeErr != nil {
			s.logger.Debug("session resume hook returned error for guest session",
				"session_id", hello.SessionID,
				"error", resumeErr)
		}

		// Final guard: if session was authenticated but user wasn't rehydrated, reject.
		// This prevents "ghost authenticated" state where presence flag is true but user is nil.
		authRehydrated := session.Get(DefaultAuthSessionKey) != nil || session.Get(auth.SessionKeyPrincipal) != nil
		if wasAuthenticated && !authRehydrated {
			s.logger.Warn("session resume rejected: auth not rehydrated",
				"session_id", hello.SessionID,
				"hint", "OnSessionResume or authFunc must call auth.Set or auth.SetPrincipal to rehydrate auth")
			s.sendHandshakeErrorWithReason(conn, protocol.HandshakeNotAuthorized, AuthExpiredResumeRehydrateFailed)
			conn.Close()
			session.Close()
			s.sessions.Close(session.ID)
			return
		}

		if session.Get(auth.SessionKeyPrincipal) != nil {
			session.authLastOK = time.Now()
		}

		// Resume existing session with soft remount
		session.Resume(conn, uint64(hello.LastSeq))

		// Set asset resolver if configured
		if s.config.AssetResolver != nil {
			session.SetAssetResolver(s.config.AssetResolver)
		}

		// Rebuild handlers (soft remount - preserves signal state)
		if err := session.RebuildHandlers(); err != nil {
			s.logger.Error("rebuild handlers failed", "error", err)
			s.sendHandshakeError(conn, protocol.HandshakeInternalError)
			conn.Close()
			// Metrics: RecordResumeFailed("rebuild_error") can be called via hooks
			return
		}

		s.sendServerHello(conn, session)

		// Send ResyncFull to ensure client DOM matches (fallback for HID mismatch)
		if err := session.SendResyncFull(); err != nil {
			s.logger.Warn("resync full failed", "error", err)
			// Not fatal - may still work if HIDs align
		}

		// Only restart goroutines if they were stopped
		if session.NeedsRestart() {
			session.Start()
		}

		// Metrics: RecordResume() and RecordSessionReattach() can be called via hooks
		s.logger.Info("session resumed",
			"session_id", session.ID,
			"user_id", session.UserID)
		return
	}

	// ═══════════════════════════════════════════════════════════════════════════
	// NEW SESSION
	// ═══════════════════════════════════════════════════════════════════════════

	// Authenticate if auth function is set
	var userID string
	if s.authFunc != nil {
		user, err := s.authFunc(r)
		if err != nil {
			s.sendHandshakeError(conn, protocol.HandshakeNotAuthorized)
			conn.Close()
			return
		}
		if id, ok := user.(string); ok {
			userID = id
		}
	}

	// Create new session
	session, err = s.sessions.Create(conn, userID, clientIP)
	if err != nil {
		if err == ErrMaxSessionsReached {
			s.sendHandshakeError(conn, protocol.HandshakeServerBusy)
		} else if err == ErrTooManySessionsFromIP {
			s.sendHandshakeError(conn, protocol.HandshakeLimitExceeded)
		} else {
			s.sendHandshakeError(conn, protocol.HandshakeInternalError)
		}
		conn.Close()
		return
	}

	// Wire router to session for route-based navigation (Phase 7: Routing)
	// This enables EventNavigate and ctx.Navigate() to work
	if s.router != nil {
		session.SetRouter(s.router)
	}

	// Set asset resolver if configured (DX Improvements)
	// This enables ctx.Asset() to resolve fingerprinted paths
	if s.config.AssetResolver != nil {
		session.SetAssetResolver(s.config.AssetResolver)
	}

	// ═══════════════════════════════════════════════════════════════════════════
	// THE CONTEXT BRIDGE (Phase 10)
	// Copy data from dying HTTP context to living session.
	// This runs SYNCHRONOUSLY before the WebSocket upgrade completes.
	// After this callback returns, r.Context() is dead.
	// ═══════════════════════════════════════════════════════════════════════════
	if s.config.OnSessionStart != nil {
		s.config.OnSessionStart(r.Context(), session)
	}

	if session.Get(auth.SessionKeyPrincipal) != nil {
		session.authLastOK = time.Now()
	}

	// Send server hello
	s.sendServerHello(conn, session)

	// Mount root component. Prefer an explicit root factory, otherwise mount the current route.
	if s.rootComponent != nil {
		session.MountRoot(s.rootComponent())
	} else if s.router != nil {
		initialPath := r.URL.Query().Get("path")
		if initialPath == "" {
			initialPath = "/"
		}
		// Never treat internal endpoints as page routes.
		if strings.HasPrefix(initialPath, "/_vango/") {
			initialPath = "/"
		}

		root, canonicalPath, err := newRouteRootComponent(session, s.router, initialPath)
		if err != nil {
			s.logger.Warn("initial route mount failed", "path", initialPath, "error", err)
		} else {
			session.CurrentRoute = canonicalPath
			session.MountRoot(root)
		}
	}

	// Start session loops
	session.Start()
}

// sendHandshakeError sends a handshake error response.
func (s *Server) sendHandshakeError(conn *websocket.Conn, status protocol.HandshakeStatus) {
	hello := protocol.NewServerHelloError(status)
	payload := protocol.EncodeServerHello(hello)
	frame := protocol.NewFrame(protocol.FrameHandshake, payload)

	conn.SetWriteDeadline(time.Now().Add(s.config.SessionConfig.WriteTimeout))
	conn.WriteMessage(websocket.BinaryMessage, frame.Encode())
}

func (s *Server) sendHandshakeErrorWithReason(conn *websocket.Conn, status protocol.HandshakeStatus, reason AuthExpiredReason) {
	hello := protocol.NewServerHelloErrorWithReason(status, uint8(reason))
	payload := protocol.EncodeServerHello(hello)
	frame := protocol.NewFrame(protocol.FrameHandshake, payload)

	conn.SetWriteDeadline(time.Now().Add(s.config.SessionConfig.WriteTimeout))
	conn.WriteMessage(websocket.BinaryMessage, frame.Encode())
}

// sendServerHello sends a successful handshake response.
func (s *Server) sendServerHello(conn *websocket.Conn, session *Session) {
	hello := protocol.NewServerHello(
		session.ID,
		uint32(session.sendSeq.Load()),
		uint64(time.Now().UnixMilli()),
	)
	payload := protocol.EncodeServerHello(hello)
	frame := protocol.NewFrame(protocol.FrameHandshake, payload)

	conn.SetWriteDeadline(time.Now().Add(s.config.SessionConfig.WriteTimeout))
	conn.WriteMessage(websocket.BinaryMessage, frame.Encode())
}

// CSRFCookieName is the name of the CSRF cookie.
const CSRFCookieName = "__vango_csrf"

// validateCSRF validates a CSRF token using Double Submit Cookie pattern.
// If CSRFSecret is set, also validates the HMAC signature.
func (s *Server) validateCSRF(r *http.Request, token string) bool {
	if s.csrfSecret == nil {
		return true // CSRF validation disabled
	}

	if token == "" {
		return false
	}

	// Double Submit Cookie: compare token from handshake with cookie value
	cookie, err := r.Cookie(CSRFCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}

	// First check: handshake token must match cookie (Double Submit)
	if !hmac.Equal([]byte(token), []byte(cookie.Value)) {
		return false
	}

	// Second check: if we have a secret, validate the HMAC signature
	if s.config.CSRFSecret != nil {
		decoded, err := base64.URLEncoding.DecodeString(token)
		if err != nil {
			return false
		}

		// Token format: 16-byte nonce + 32-byte HMAC-SHA256 signature
		if len(decoded) != 48 {
			return false
		}

		nonce := decoded[:16]
		providedSig := decoded[16:]

		// Recompute expected signature
		h := hmac.New(sha256.New, s.config.CSRFSecret)
		h.Write(nonce)
		expectedSig := h.Sum(nil)

		// Constant-time comparison
		if !hmac.Equal(providedSig, expectedSig) {
			return false
		}
	}

	return true
}

// GenerateCSRFToken generates a new cryptographically secure CSRF token.
// If CSRFSecret is set, the token is HMAC-signed for additional security.
// This should be:
// 1. Set as a cookie with path=/, SameSite from config, Secure when required
// 2. Embedded in the initial HTML page for the client to send in handshake
func (s *Server) GenerateCSRFToken() string {
	// Generate random nonce
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		// SECURITY: Fatal on entropy failure - weak tokens are dangerous
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}

	// If no secret, just return the nonce (backward compatible)
	if s.config.CSRFSecret == nil {
		return base64.URLEncoding.EncodeToString(nonce)
	}

	// HMAC-sign the nonce with the secret
	h := hmac.New(sha256.New, s.config.CSRFSecret)
	h.Write(nonce)
	sig := h.Sum(nil)

	// Token = nonce + signature (both base64 encoded together)
	token := make([]byte, len(nonce)+len(sig))
	copy(token[:len(nonce)], nonce)
	copy(token[len(nonce):], sig)

	return base64.URLEncoding.EncodeToString(token)
}

// SetCSRFCookie sets the CSRF cookie on the response.
// Call this when rendering the initial page.
// Uses request TLS or trusted proxy headers to decide the Secure flag.
// Returns ErrSecureCookiesRequired if secure cookies are enabled and the request is not secure.
func (s *Server) SetCSRFCookie(w http.ResponseWriter, r *http.Request, token string) error {
	cookie, err := s.csrfCookie(r, token)
	if err != nil {
		return err
	}
	http.SetCookie(w, cookie)
	return nil
}

// Run starts the server and blocks until shutdown.
func (s *Server) Run() error {
	if err := s.config.ValidateConfig(); err != nil {
		return err
	}

	s.httpServer = &http.Server{
		Addr:              s.config.Address,
		Handler:           s,
		ReadHeaderTimeout: s.config.ReadHeaderTimeout,
		ReadTimeout:       s.config.ReadTimeout,
		WriteTimeout:      s.config.WriteTimeout,
		IdleTimeout:       s.config.IdleTimeout,
	}

	// Set up graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Error channel for ListenAndServe
	errCh := make(chan error, 1)

	go func() {
		s.logger.Info("server starting", "address", s.config.Address)
		errCh <- s.httpServer.ListenAndServe()
	}()

	// Wait for shutdown signal or error
	select {
	case err := <-errCh:
		if err != http.ErrServerClosed {
			return err
		}
		return nil

	case <-shutdown:
		s.logger.Info("shutting down...")
		return s.Shutdown(context.Background())
	}
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	// Create timeout context
	ctx, cancel := context.WithTimeout(ctx, s.config.ShutdownTimeout)
	defer cancel()

	// Close all sessions first
	s.sessions.Shutdown()

	// Shutdown HTTP server
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			s.logger.Error("shutdown error", "error", err)
			return err
		}
	}

	s.logger.Info("server shutdown complete")
	return nil
}

// Sessions returns the session manager.
func (s *Server) Sessions() *SessionManager {
	return s.sessions
}

// Config returns the server configuration.
func (s *Server) Config() *ServerConfig {
	return s.config
}

// CookiePolicy returns the server cookie policy helper.
func (s *Server) CookiePolicy() *CookiePolicy {
	return s.cookiePolicy
}

// Logger returns the server logger.
func (s *Server) Logger() *slog.Logger {
	return s.logger
}

// SetLogger sets the server logger.
func (s *Server) SetLogger(logger *slog.Logger) {
	s.logger = logger
}
