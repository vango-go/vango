package server

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
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
	}

	logger := slog.Default().With("component", "server")

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

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check for WebSocket upgrade
	if r.URL.Path == "/_vango/ws" {
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
	var session *Session
	if hello.SessionID != "" {
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

// validateCSRF validates a CSRF token from the request.
func (s *Server) validateCSRF(r *http.Request, token string) bool {
	if s.csrfSecret == nil {
		return true // CSRF validation disabled
	}

	if token == "" {
		return false
	}

	// For now, simple validation - in production use proper CSRF handling
	// The token should be HMAC of session info with the secret
	expected := s.generateCSRF(r)
	return hmac.Equal([]byte(token), []byte(expected))
}

// generateCSRF generates a CSRF token for the request.
func (s *Server) generateCSRF(r *http.Request) string {
	h := hmac.New(sha256.New, s.csrfSecret)
	h.Write([]byte(r.RemoteAddr))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// GenerateCSRFToken generates a new CSRF token.
// This should be embedded in the initial HTML page.
func (s *Server) GenerateCSRFToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
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
