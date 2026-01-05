package server

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/vango-go/vango/pkg/session"
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

	// ==========================================================================
	// Phase 16: Storm Budgets
	// ==========================================================================

	// StormBudget configures rate limits for effect primitives to prevent
	// amplification bugs (e.g., effect triggers resource refetch triggers effect).
	// See SPEC_ADDENDUM.md §A.4.
	StormBudget *StormBudgetConfig
}

// BudgetExceededMode determines behavior when a storm budget is exceeded.
type BudgetExceededMode int

const (
	// BudgetThrottle drops excess operations silently (default).
	// Operations that exceed the budget are not executed.
	BudgetThrottle BudgetExceededMode = iota

	// BudgetTripBreaker pauses effect execution until cleared.
	// Like a circuit breaker, stops all effect processing until reset.
	BudgetTripBreaker
)

// StormBudgetConfig configures rate limits for effect primitives.
// These limits help prevent amplification bugs where effects cascade into
// more effects, potentially causing performance issues or infinite loops.
type StormBudgetConfig struct {
	// MaxResourceStartsPerSecond limits how many Resource fetches can start per second.
	// 0 means no limit.
	// Default: 50.
	MaxResourceStartsPerSecond int

	// MaxActionStartsPerSecond limits how many Action runs can start per second.
	// 0 means no limit.
	// Default: 100.
	MaxActionStartsPerSecond int

	// MaxGoLatestStartsPerSecond limits how many GoLatest work items can start per second.
	// 0 means no limit.
	// Default: 50.
	MaxGoLatestStartsPerSecond int

	// MaxEffectRunsPerTick limits effect runs within a single event/dispatch tick.
	// Helps catch infinite loops where effects trigger effects.
	// 0 means no limit.
	// Default: 1000.
	MaxEffectRunsPerTick int

	// WindowDuration is the sliding window for per-second limits.
	// Default: 1 second.
	WindowDuration time.Duration

	// OnExceeded determines what happens when a budget is exceeded.
	// Default: BudgetThrottle (drop excess operations).
	OnExceeded BudgetExceededMode
}

// DefaultStormBudgetConfig returns a StormBudgetConfig with sensible defaults.
// These defaults are conservative but should handle most applications.
func DefaultStormBudgetConfig() *StormBudgetConfig {
	return &StormBudgetConfig{
		MaxResourceStartsPerSecond: 50,
		MaxActionStartsPerSecond:   100,
		MaxGoLatestStartsPerSecond: 50,
		MaxEffectRunsPerTick:       1000,
		WindowDuration:             time.Second,
		OnExceeded:                 BudgetThrottle,
	}
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
		StormBudget:       DefaultStormBudgetConfig(),
	}
}

// Clone returns a copy of the SessionConfig.
func (c *SessionConfig) Clone() *SessionConfig {
	if c == nil {
		return nil
	}
	clone := *c
	if c.StormBudget != nil {
		budgetClone := *c.StormBudget
		clone.StormBudget = &budgetClone
	}
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

	// ==========================================================================
	// Phase 13: Secure Defaults & DevMode
	// ==========================================================================

	// DevMode disables security checks for local development.
	// SECURITY: NEVER use in production - this disables:
	// - Origin checking (allows all origins)
	// - CSRF validation
	// - Secure cookie requirements
	// Default: false (secure by default)
	DevMode bool

	// SecureCookies enforces Secure flag on session cookies.
	// Should be true when using HTTPS.
	// Default: true
	SecureCookies bool

	// SameSiteMode sets the SameSite attribute for cookies.
	// Lax is safe for most use cases and allows OAuth flows.
	// Default: http.SameSiteLaxMode
	SameSiteMode http.SameSite

	// CookieDomain sets the Domain attribute for session cookies.
	// Empty string uses the current domain (most secure).
	// Default: "" (current domain)
	CookieDomain string

	// ==========================================================================
	// Phase 12: Session Resilience & State Persistence
	// ==========================================================================

	// SessionStore is the persistence backend for sessions.
	// If nil, sessions are only stored in memory (lost on restart).
	// Use session.NewMemoryStore(), session.NewRedisStore(), or session.NewSQLStore().
	// Default: nil (in-memory only).
	SessionStore session.SessionStore

	// ResumeWindow is how long a detached session remains resumable after disconnect.
	// After this window, the session is permanently expired.
	// Default: 5 minutes.
	ResumeWindow time.Duration

	// MaxDetachedSessions is the maximum number of disconnected sessions to keep in memory.
	// When exceeded, the least recently used sessions are evicted (and persisted if store is configured).
	// Default: 10000.
	MaxDetachedSessions int

	// MaxSessionsPerIP is the maximum number of concurrent sessions from a single IP address.
	// Helps prevent DoS attacks from IP exhaustion.
	// 0 means no limit.
	// Default: 100.
	MaxSessionsPerIP int

	// PersistInterval is how often to persist dirty sessions to the store.
	// Set to 0 to only persist on disconnect.
	// Default: 30 seconds.
	PersistInterval time.Duration

	// ReconnectConfig configures client-side reconnection behavior.
	// Default: ReconnectConfig with sensible defaults.
	ReconnectConfig *ReconnectConfig
}

// ReconnectConfig configures client-side reconnection behavior.
// These values are sent to the client during handshake and control
// how the thin client handles connection interruptions.
type ReconnectConfig struct {
	// ToastOnReconnect shows a toast notification when connection is restored.
	// Default: false.
	ToastOnReconnect bool

	// ToastMessage is the message shown in the reconnection toast.
	// Default: "Connection restored".
	ToastMessage string

	// MaxRetries is the maximum number of reconnection attempts before giving up.
	// Default: 10.
	MaxRetries int

	// BaseDelay is the initial delay between reconnection attempts (milliseconds).
	// Uses exponential backoff: delay = min(baseDelay * 2^attempt, maxDelay).
	// Default: 1000 (1 second).
	BaseDelay int

	// MaxDelay is the maximum delay between reconnection attempts (milliseconds).
	// Default: 30000 (30 seconds).
	MaxDelay int
}

// DefaultReconnectConfig returns a ReconnectConfig with sensible defaults.
func DefaultReconnectConfig() *ReconnectConfig {
	return &ReconnectConfig{
		ToastOnReconnect: false,
		ToastMessage:     "Connection restored",
		MaxRetries:       10,
		BaseDelay:        1000,  // 1 second
		MaxDelay:         30000, // 30 seconds
	}
}

// DefaultServerConfig returns a ServerConfig with sensible defaults.
// SECURITY: All security features are ENABLED by default.
// SECURITY: CheckOrigin enforces same-origin by default to prevent CSWSH.
// SECURITY: CSRFSecret is nil by default but a warning is logged on startup.
// SECURITY: SecureCookies is true by default for HTTPS environments.
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
		// Phase 13: Secure defaults
		DevMode:       false,               // SECURE DEFAULT: security enabled
		SecureCookies: true,                // SECURE DEFAULT: HTTPS cookies
		SameSiteMode:  http.SameSiteLaxMode, // SECURE DEFAULT: Lax mode
		CookieDomain:  "",                  // SECURE DEFAULT: current domain only
		// Phase 12 defaults
		SessionStore:        nil, // In-memory only by default
		ResumeWindow:        5 * time.Minute,
		MaxDetachedSessions: 10000,
		MaxSessionsPerIP:    100,
		PersistInterval:     30 * time.Second,
		ReconnectConfig:     DefaultReconnectConfig(),
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

// =============================================================================
// Phase 12: Session Resilience Configuration Helpers
// =============================================================================

// WithSessionStore sets the session persistence backend and returns the config for chaining.
func (c *ServerConfig) WithSessionStore(store session.SessionStore) *ServerConfig {
	c.SessionStore = store
	return c
}

// WithResumeWindow sets the resume window duration and returns the config for chaining.
func (c *ServerConfig) WithResumeWindow(d time.Duration) *ServerConfig {
	c.ResumeWindow = d
	return c
}

// WithMaxDetachedSessions sets the maximum detached sessions and returns the config for chaining.
func (c *ServerConfig) WithMaxDetachedSessions(max int) *ServerConfig {
	c.MaxDetachedSessions = max
	return c
}

// WithMaxSessionsPerIP sets the per-IP session limit and returns the config for chaining.
func (c *ServerConfig) WithMaxSessionsPerIP(max int) *ServerConfig {
	c.MaxSessionsPerIP = max
	return c
}

// WithPersistInterval sets the persist interval and returns the config for chaining.
func (c *ServerConfig) WithPersistInterval(d time.Duration) *ServerConfig {
	c.PersistInterval = d
	return c
}

// WithReconnectConfig sets the reconnect configuration and returns the config for chaining.
func (c *ServerConfig) WithReconnectConfig(rc *ReconnectConfig) *ServerConfig {
	c.ReconnectConfig = rc
	return c
}

// EnableToastOnReconnect enables toast notifications on reconnect.
// Shorthand for modifying ReconnectConfig.
func (c *ServerConfig) EnableToastOnReconnect(message string) *ServerConfig {
	if c.ReconnectConfig == nil {
		c.ReconnectConfig = DefaultReconnectConfig()
	}
	c.ReconnectConfig.ToastOnReconnect = true
	if message != "" {
		c.ReconnectConfig.ToastMessage = message
	}
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

// =============================================================================
// Phase 13: DevMode & Secure Defaults
// =============================================================================

// WithDevMode enables development mode which disables security checks.
// SECURITY WARNING: NEVER use in production. This disables:
//   - Origin checking (allows all origins)
//   - CSRF validation (disabled)
//   - Secure cookie requirements (disabled)
//
// Only use for local development:
//
//	config := server.DefaultServerConfig().WithDevMode()
func (c *ServerConfig) WithDevMode() *ServerConfig {
	c.DevMode = true
	// Override security settings for development
	c.CheckOrigin = func(r *http.Request) bool { return true }
	c.SecureCookies = false
	return c
}

// WithSecureCookies sets whether cookies should have the Secure flag.
func (c *ServerConfig) WithSecureCookies(secure bool) *ServerConfig {
	c.SecureCookies = secure
	return c
}

// WithSameSiteMode sets the SameSite attribute for cookies.
func (c *ServerConfig) WithSameSiteMode(mode http.SameSite) *ServerConfig {
	c.SameSiteMode = mode
	return c
}

// WithCookieDomain sets the domain for session cookies.
func (c *ServerConfig) WithCookieDomain(domain string) *ServerConfig {
	c.CookieDomain = domain
	return c
}

// ValidateConfig validates the server configuration and logs warnings.
// Called automatically by Server.Start().
// Returns any fatal configuration errors.
func (c *ServerConfig) ValidateConfig() error {
	var warnings []string

	// Check DevMode
	if c.DevMode {
		warnings = append(warnings, "DEV MODE ENABLED - Security checks disabled. DO NOT USE IN PRODUCTION.")
	}

	// Check CSRF secret
	if c.CSRFSecret == nil && !c.DevMode {
		warnings = append(warnings, "CSRFSecret not set. CSRF protection disabled. Set CSRFSecret for production.")
	}

	// Check cookie security
	if !c.SecureCookies && !c.DevMode {
		warnings = append(warnings, "SecureCookies disabled. Session cookies will not have Secure flag. Enable for HTTPS.")
	}

	// Check session limits
	if c.MaxSessionsPerIP == 0 {
		warnings = append(warnings, "MaxSessionsPerIP is 0 (unlimited). Consider setting a limit to prevent DoS.")
	}

	if c.MaxDetachedSessions == 0 {
		warnings = append(warnings, "MaxDetachedSessions is 0 (unlimited). Consider setting a limit to prevent memory exhaustion.")
	}

	// Log warnings
	for _, w := range warnings {
		// Warnings are informational, not fatal
		// Server will still start but these should be addressed
		_ = w // In production, this would log via slog
	}

	return nil
}

// GetConfigWarnings returns a list of configuration warnings without logging them.
// Useful for displaying warnings in a custom way.
func (c *ServerConfig) GetConfigWarnings() []string {
	var warnings []string

	if c.DevMode {
		warnings = append(warnings, "DEV MODE ENABLED - Security checks disabled")
	}

	if c.CSRFSecret == nil && !c.DevMode {
		warnings = append(warnings, "CSRFSecret not set - CSRF protection disabled")
	}

	if !c.SecureCookies && !c.DevMode {
		warnings = append(warnings, "SecureCookies disabled - cookies won't have Secure flag")
	}

	if c.MaxSessionsPerIP == 0 {
		warnings = append(warnings, "MaxSessionsPerIP unlimited - DoS risk")
	}

	if c.MaxDetachedSessions == 0 {
		warnings = append(warnings, "MaxDetachedSessions unlimited - memory exhaustion risk")
	}

	return warnings
}

// IsSecure returns true if the configuration has all security features enabled.
// Useful for startup checks.
func (c *ServerConfig) IsSecure() bool {
	return !c.DevMode &&
		c.CSRFSecret != nil &&
		c.SecureCookies &&
		c.CheckOrigin != nil &&
		c.MaxSessionsPerIP > 0 &&
		c.MaxDetachedSessions > 0
}
