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
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vango-dev/vango/v2/pkg/protocol"
)

// Server is the main HTTP/WebSocket server for Vango.
type Server struct {
	// Session management
	sessions *SessionManager

	// HTTP handler (router placeholder for Phase 7)
	handler http.Handler

	// Root component factory
	rootComponent func() Component

	// Configuration
	config *ServerConfig

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
	}

	logger := slog.Default().With("component", "server")

	// SECURITY WARNING: Log if CSRF protection is disabled
	if config.CSRFSecret == nil {
		logger.Warn("CSRF protection is DISABLED. Set CSRFSecret for production use. " +
			"This will become a hard requirement in Vango v3.0.")
	}

	s := &Server{
		sessions: NewSessionManager(config.SessionConfig, DefaultSessionLimits(), logger),
		config:   config,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  config.ReadBufferSize,
			WriteBufferSize: config.WriteBufferSize,
			CheckOrigin:     config.CheckOrigin,
		},
		csrfSecret: config.CSRFSecret,
		logger:     logger,
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

	// Parse client hello
	hello, err := protocol.DecodeClientHello(msg)
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

	// Check for session resume
	// NOTE: After a page refresh, SSR renders a new page with different HIDs/handlers.
	// If we resume the old session, its handler map is stale and won't match.
	// For now, we disable session resume to ensure fresh handlers on each page load.
	// A production fix would remount the component on resume.
	// TODO: Implement proper session resume with component remount
	var session *Session
	if false && hello.SessionID != "" { // Disabled: stale handlers cause click failures
		session = s.sessions.Get(hello.SessionID)
		if session != nil {
			// Resume existing session
			session.Resume(conn, uint64(hello.LastSeq))
			s.sendServerHello(conn, session)
			session.Start()
			return
		}
	}

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
	session, err = s.sessions.Create(conn, userID)
	if err != nil {
		if err == ErrMaxSessionsReached {
			s.sendHandshakeError(conn, protocol.HandshakeServerBusy)
		} else {
			s.sendHandshakeError(conn, protocol.HandshakeInternalError)
		}
		conn.Close()
		return
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

	// Send server hello
	s.sendServerHello(conn, session)

	// Mount root component if factory is set
	if s.rootComponent != nil {
		session.MountRoot(s.rootComponent())
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
// 1. Set as a cookie with path=/, SameSite=Strict, Secure (in prod)
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
func (s *Server) SetCSRFCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false, // Must be readable by JS for Double Submit
		SameSite: http.SameSiteStrictMode,
		Secure:   s.isSecure(),
	})
}

// isSecure returns true if the server should use secure cookies.
func (s *Server) isSecure() bool {
	// Check if address starts with https or if TLS is configured
	// For now, check if not localhost for simple heuristic
	addr := s.config.Address
	return addr != ":8080" && addr != "localhost:8080" && addr != "127.0.0.1:8080"
}

// Run starts the server and blocks until shutdown.
func (s *Server) Run() error {
	s.httpServer = &http.Server{
		Addr:    s.config.Address,
		Handler: s,
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

// Logger returns the server logger.
func (s *Server) Logger() *slog.Logger {
	return s.logger
}

// SetLogger sets the server logger.
func (s *Server) SetLogger(logger *slog.Logger) {
	s.logger = logger
}
