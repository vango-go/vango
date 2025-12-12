package server

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// SessionConfig holds configuration for individual sessions.
type SessionConfig struct {
	// Timeouts

	// ReadTimeout is the maximum time to wait for a message from the client.
	// Default: 60 seconds.
	ReadTimeout time.Duration

	// WriteTimeout is the maximum time to wait when sending a message.
	// Default: 10 seconds.
	WriteTimeout time.Duration

	// IdleTimeout is the time after which an inactive session is closed.
	// Default: 5 minutes.
	IdleTimeout time.Duration

	// HandshakeTimeout is the maximum time for the initial handshake.
	// Default: 10 seconds.
	HandshakeTimeout time.Duration

	// HeartbeatInterval is the time between heartbeat pings.
	// Default: 30 seconds.
	HeartbeatInterval time.Duration

	// Limits

	// MaxMessageSize is the maximum size of an incoming WebSocket message.
	// Default: 64KB.
	MaxMessageSize int64

	// MaxPatchHistory is the number of recent patches to keep for resync.
	// Default: 100.
	MaxPatchHistory int

	// MaxEventQueue is the size of the event channel buffer.
	// Default: 256.
	MaxEventQueue int

	// Features

	// EnableCompression enables WebSocket compression.
	// Default: true.
	EnableCompression bool

	// EnableOptimistic enables optimistic updates on the client.
	// Default: true.
	EnableOptimistic bool
}

// DefaultSessionConfig returns a SessionConfig with sensible defaults.
func DefaultSessionConfig() *SessionConfig {
	return &SessionConfig{
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       5 * time.Minute,
		HandshakeTimeout:  10 * time.Second,
		HeartbeatInterval: 30 * time.Second,
		MaxMessageSize:    64 * 1024, // 64KB
		MaxPatchHistory:   100,
		MaxEventQueue:     256,
		EnableCompression: true,
		EnableOptimistic:  true,
	}
}

// Clone returns a copy of the SessionConfig.
func (c *SessionConfig) Clone() *SessionConfig {
	if c == nil {
		return nil
	}
	clone := *c
	return &clone
}

// ServerConfig holds configuration for the HTTP/WebSocket server.
type ServerConfig struct {
	// Address is the address to listen on (e.g., ":8080" or "localhost:3000").
	// Default: ":8080".
	Address string

	// WebSocket buffer sizes

	// ReadBufferSize is the WebSocket read buffer size.
	// Default: 4096.
	ReadBufferSize int

	// WriteBufferSize is the WebSocket write buffer size.
	// Default: 4096.
	WriteBufferSize int

	// CheckOrigin is called to validate the request origin.
	// Default: allows all origins (not recommended for production).
	CheckOrigin func(r *http.Request) bool

	// Session configuration

	// SessionConfig is the configuration for individual sessions.
	// Default: DefaultSessionConfig().
	SessionConfig *SessionConfig

	// Server lifecycle

	// ShutdownTimeout is the maximum time to wait for graceful shutdown.
	// Default: 30 seconds.
	ShutdownTimeout time.Duration

	// Limits

	// MaxSessions is the maximum number of concurrent sessions.
	// 0 means no limit.
	// Default: 0 (no limit).
	MaxSessions int

	// MaxMemoryPerSession is the approximate memory limit per session.
	// Sessions exceeding this may be evicted under memory pressure.
	// Default: 200KB.
	MaxMemoryPerSession int64

	// Security

	// CSRFSecret is the secret key for CSRF token generation.
	// If nil, CSRF validation is disabled (not recommended for production).
	CSRFSecret []byte

	// Cleanup

	// CleanupInterval is the interval for the session cleanup loop.
	// Default: 30 seconds.
	CleanupInterval time.Duration

	// Context Bridge

	// OnSessionStart is called during WebSocket upgrade, BEFORE the handshake completes.
	// Use this to copy data from the HTTP context (e.g., authenticated user) to the Vango session.
	// This runs SYNCHRONOUSLY before the WebSocket upgrade completes, while r.Context() is still alive.
	// After this callback returns, the HTTP context is dead and cannot be accessed.
	//
	// Example:
	//     OnSessionStart: func(httpCtx context.Context, session *Session) {
	//         if user := auth.UserFromContext(httpCtx); user != nil {
	//             session.Set("vango_auth_user", user)
	//         }
	//     }
	OnSessionStart func(httpCtx context.Context, session *Session)

	// TrustedProxies lists trusted reverse proxy IPs for X-Forwarded-* headers.
	// If set, the server will trust X-Forwarded-For, X-Real-IP, etc. from these IPs.
	// Default: nil (don't trust proxy headers).
	TrustedProxies []string

	// DebugMode enables extra validation and logging for development.
	// When true:
	// - Session.Set() panics on unserializable types (func, chan)
	// - auth.Get() logs warnings on type mismatches
	// Default: false.
	DebugMode bool
}

// DefaultServerConfig returns a ServerConfig with sensible defaults.
// SECURITY: CheckOrigin enforces same-origin by default to prevent CSWSH.
// SECURITY: CSRFSecret is nil by default but a warning is logged on startup.
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Address:             ":8080",
		ReadBufferSize:      4096,
		WriteBufferSize:     4096,
		CheckOrigin:         SameOriginCheck, // SECURE DEFAULT: reject cross-origin
		SessionConfig:       DefaultSessionConfig(),
		ShutdownTimeout:     30 * time.Second,
		MaxSessions:         0,          // No limit
		MaxMemoryPerSession: 200 * 1024, // 200KB
		CSRFSecret:          nil,        // Warning logged on startup if nil
		CleanupInterval:     30 * time.Second,
	}
}

// SameOriginCheck validates that the WebSocket request origin matches the host.
// This is the secure default for CheckOrigin.
// SECURITY: Uses proper URL parsing to avoid edge cases with string manipulation.
func SameOriginCheck(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		// No Origin header (e.g., same-origin request or curl)
		return true
	}

	// Parse origin as URL for robust comparison
	originURL, err := url.Parse(origin)
	if err != nil {
		return false
	}

	host := r.Host
	if host == "" {
		return false
	}

	// Compare the host portion (includes port if present)
	return originURL.Host == host
}

// Clone returns a copy of the ServerConfig.
func (c *ServerConfig) Clone() *ServerConfig {
	if c == nil {
		return nil
	}
	clone := *c
	if c.SessionConfig != nil {
		clone.SessionConfig = c.SessionConfig.Clone()
	}
	if c.CSRFSecret != nil {
		clone.CSRFSecret = make([]byte, len(c.CSRFSecret))
		copy(clone.CSRFSecret, c.CSRFSecret)
	}
	return &clone
}

// WithAddress sets the server address and returns the config for chaining.
func (c *ServerConfig) WithAddress(addr string) *ServerConfig {
	c.Address = addr
	return c
}

// WithSessionConfig sets the session configuration and returns the config for chaining.
func (c *ServerConfig) WithSessionConfig(sc *SessionConfig) *ServerConfig {
	c.SessionConfig = sc
	return c
}

// WithMaxSessions sets the maximum sessions and returns the config for chaining.
func (c *ServerConfig) WithMaxSessions(max int) *ServerConfig {
	c.MaxSessions = max
	return c
}

// WithCSRFSecret sets the CSRF secret and returns the config for chaining.
func (c *ServerConfig) WithCSRFSecret(secret []byte) *ServerConfig {
	c.CSRFSecret = secret
	return c
}

// SessionLimits defines limits for session management.
type SessionLimits struct {
	// MaxSessions is the maximum number of concurrent sessions.
	MaxSessions int

	// MaxMemoryPerSession is the approximate memory limit per session.
	MaxMemoryPerSession int64

	// MaxTotalMemory is the total memory budget for all sessions.
	// If exceeded, least recently used sessions are evicted.
	MaxTotalMemory int64
}

// DefaultSessionLimits returns default session limits.
func DefaultSessionLimits() *SessionLimits {
	return &SessionLimits{
		MaxSessions:         10000,
		MaxMemoryPerSession: 200 * 1024,             // 200KB
		MaxTotalMemory:      1 * 1024 * 1024 * 1024, // 1GB
	}
}
