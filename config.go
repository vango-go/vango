package vango

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/vango-go/vango/pkg/server"
	"github.com/vango-go/vango/pkg/session"
)

// =============================================================================
// Configuration Types
// =============================================================================

// Config is the main application configuration.
// This is the user-friendly entry point for configuring a Vango app.
type Config struct {
	// Session configures session behavior including durability and limits.
	Session SessionConfig

	// Static configures static file serving.
	Static StaticConfig

	// API configures JSON API routes (app.API).
	API APIConfig

	// Security configures security features (CSRF, origin checking, cookies).
	Security SecurityConfig

	// DevMode enables development mode which disables security checks.
	// SECURITY: NEVER use in production - this disables:
	//   - Origin checking (allows all origins)
	//   - CSRF validation
	//   - Secure cookie requirements
	DevMode bool

	// Logger is the structured logger for the application.
	// If nil, slog.Default() is used.
	Logger *slog.Logger

	// OnSessionStart is called when a new WebSocket session is established.
	// Use this to transfer data from the HTTP context (e.g., authenticated user)
	// to the Vango session before the handshake completes.
	//
	// Example:
	//   OnSessionStart: func(httpCtx context.Context, s *vango.Session) {
	//       if user := myauth.UserFromContext(httpCtx); user != nil {
	//           auth.Set(s, user)  // Use auth.Set to set presence flag
	//       }
	//   }
	OnSessionStart func(httpCtx context.Context, s *Session)

	// OnSessionResume is called when resuming an existing WebSocket session.
	// Use this to rehydrate session data from the HTTP context (e.g., re-validate auth).
	//
	// Unlike OnSessionStart, this is called when resuming after disconnect.
	// Return nil to allow the resume, or an error to reject it.
	//
	// Example:
	//   OnSessionResume: func(httpCtx context.Context, s *vango.Session) error {
	//       user, err := myauth.ValidateFromContext(httpCtx)
	//       if err != nil {
	//           return err  // Reject resume if previously authenticated
	//       }
	//       if user != nil {
	//           auth.Set(s, user)
	//       }
	//       return nil
	//   }
	OnSessionResume func(httpCtx context.Context, s *Session) error
}

// SessionConfig configures session behavior.
type SessionConfig struct {
	// ResumeWindow is how long a disconnected session remains resumable.
	// Within this window, a reconnecting client restores full session state.
	// After this window, the session is permanently expired.
	// Default: 5 minutes.
	ResumeWindow time.Duration

	// Store is the persistence backend for sessions.
	// If nil, sessions are only stored in memory (lost on server restart).
	// Use vango.NewMemoryStore(), vango.NewRedisStore(), or vango.NewSQLStore().
	Store SessionStore

	// MaxDetachedSessions is the maximum number of disconnected sessions
	// to keep in memory. When exceeded, least recently used sessions are evicted.
	// Default: 10000.
	MaxDetachedSessions int

	// MaxSessionsPerIP is the maximum concurrent sessions from a single IP.
	// Helps prevent DoS attacks. 0 means no limit.
	// Default: 100.
	MaxSessionsPerIP int

	// EvictOnIPLimit controls whether hitting MaxSessionsPerIP evicts the oldest
	// detached session for that IP instead of rejecting the new session.
	// Default: true when MaxSessionsPerIP > 0.
	EvictOnIPLimit bool

	// StormBudget configures rate limits for async primitives to prevent
	// amplification bugs (e.g., effect triggers resource refetch triggers effect).
	StormBudget *StormBudgetConfig

	// AuthCheck configures authentication freshness checks for the session.
	// When nil, no passive or active auth checks are performed.
	AuthCheck *AuthCheckConfig
}

// StaticConfig configures static file serving.
type StaticConfig struct {
	// Dir is the directory containing static files (e.g., "public").
	// Files in this directory are served at the URL prefix.
	Dir string

	// Prefix is the URL path prefix for static files (e.g., "/").
	// A file at public/styles.css with Prefix="/" is served at /styles.css.
	// Default: "/".
	Prefix string

	// CacheControl determines caching behavior for static files.
	// Default: CacheControlNone (no caching headers).
	CacheControl CacheControlStrategy

	// Compression enables gzip/brotli compression for static files.
	// Default: false.
	Compression bool

	// Headers are custom headers to add to all static file responses.
	Headers map[string]string
}

// APIConfig configures JSON API routes registered via app.API.
type APIConfig struct {
	// MaxBodyBytes is the maximum number of bytes read from the HTTP request body
	// when an API handler declares a typed body parameter.
	//
	// Default: 1 MiB.
	MaxBodyBytes int64

	// RequireJSONContentType enforces that requests with a non-empty body specify a
	// JSON Content-Type (application/json or application/*+json) when an API handler
	// declares a structured (non-[]byte, non-string) body parameter.
	//
	// When false (default), missing Content-Type is accepted, but explicit non-JSON
	// Content-Type is rejected.
	RequireJSONContentType bool
}

// SecurityConfig configures security features.
type SecurityConfig struct {
	// CSRFSecret is the secret key for CSRF token generation.
	// If nil and DevMode is false, CSRF protection uses a random secret
	// (tokens won't survive server restarts).
	// Required for production: generate with crypto/rand and store securely.
	CSRFSecret []byte

	// AllowedOrigins lists domains allowed for WebSocket connections.
	// If empty and AllowSameOrigin is true, only same-origin requests are allowed.
	// Example: []string{"https://myapp.com", "https://www.myapp.com"}
	AllowedOrigins []string

	// AllowedRedirectHosts lists hostnames (and optional ports) allowed for external redirects.
	// When empty, external redirects are rejected.
	// Example: []string{"accounts.google.com", "auth.mycompany.com", "auth.mycompany.com:8443"}
	AllowedRedirectHosts []string

	// AllowSameOrigin enables automatic same-origin validation.
	// When true and AllowedOrigins is empty, validates that Origin header
	// matches the request Host header.
	// Default: true.
	AllowSameOrigin bool

	// TrustedProxies lists reverse proxy IPs trusted for X-Forwarded-* headers.
	// When set, forwarded proto headers are honored for secure cookie decisions.
	// Default: nil (do not trust forwarded headers).
	TrustedProxies []string

	// CookieSecure sets the Secure flag on cookies set by the server.
	// Should be true when using HTTPS.
	// Default: true.
	CookieSecure bool

	// CookieHttpOnly sets the HttpOnly flag on cookies set by the server.
	// Prevents JavaScript access to cookies that should not be read by JS.
	// Default: true (except for CSRF cookies which need JS access).
	CookieHttpOnly bool

	// CookieSameSite sets the SameSite attribute for cookies set by the server.
	// Lax is safe for most use cases and allows OAuth redirect flows.
	// Default: http.SameSiteLaxMode.
	CookieSameSite http.SameSite

	// CookieDomain sets the Domain attribute for cookies.
	// Empty string uses the current domain (most secure).
	// Default: "".
	CookieDomain string
}

// CacheControlStrategy determines caching behavior for static files.
type CacheControlStrategy int

const (
	// CacheControlNone adds no caching headers.
	// Use in development for instant updates.
	CacheControlNone CacheControlStrategy = iota

	// CacheControlProduction uses appropriate caching:
	// - Fingerprinted files (*.abc123.css): immutable, 1 year max-age
	// - Other files: short cache with revalidation
	CacheControlProduction
)

// StormBudgetConfig configures rate limits for async primitives.
// These limits help prevent amplification bugs where effects cascade
// into more effects, potentially causing performance issues.
type StormBudgetConfig = server.StormBudgetConfig

// AuthCheckConfig configures periodic active revalidation.
type AuthCheckConfig = server.AuthCheckConfig

// AuthFailureMode controls what happens when active checks fail.
type AuthFailureMode = server.AuthFailureMode

// AuthExpiredConfig defines behavior when auth expires.
type AuthExpiredConfig = server.AuthExpiredConfig

// AuthExpiredAction defines the action type for auth expiry.
type AuthExpiredAction = server.AuthExpiredAction

// AuthExpiredReason provides structured context for auth expiry.
type AuthExpiredReason = server.AuthExpiredReason

const (
	// Failure modes
	FailOpenWithGrace = server.FailOpenWithGrace
	FailClosed        = server.FailClosed

	// Expiry actions
	ForceReload = server.ForceReload
	NavigateTo  = server.NavigateTo
	AuthCustom  = server.Custom

	// Expiry reasons
	AuthExpiredUnknown                  = server.AuthExpiredUnknown
	AuthExpiredPassiveExpiry            = server.AuthExpiredPassiveExpiry
	AuthExpiredResumeRehydrateFailed    = server.AuthExpiredResumeRehydrateFailed
	AuthExpiredActiveRevalidateFailed   = server.AuthExpiredActiveRevalidateFailed
	AuthExpiredOnDemandRevalidateFailed = server.AuthExpiredOnDemandRevalidateFailed
)

// SessionStore is the interface for session persistence backends.
type SessionStore = session.SessionStore

// Session represents a client session.
type Session = server.Session

// =============================================================================
// Default Configurations
// =============================================================================

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Session: DefaultSessionConfig(),
		Static:  DefaultStaticConfig(),
		API:     DefaultAPIConfig(),
		Security: SecurityConfig{
			AllowSameOrigin: true,
			CookieSecure:    true,
			CookieHttpOnly:  true,
			CookieSameSite:  http.SameSiteLaxMode,
			CookieDomain:    "",
		},
		DevMode: false,
	}
}

// DefaultSessionConfig returns a SessionConfig with sensible defaults.
func DefaultSessionConfig() SessionConfig {
	return SessionConfig{
		ResumeWindow:        5 * time.Minute,
		MaxDetachedSessions: 10000,
		MaxSessionsPerIP:    100,
		EvictOnIPLimit:      true,
		AuthCheck:           nil,
	}
}

// DefaultStaticConfig returns a StaticConfig with sensible defaults.
func DefaultStaticConfig() StaticConfig {
	return StaticConfig{
		Prefix:       "/",
		CacheControl: CacheControlNone,
	}
}

// DefaultAPIConfig returns an APIConfig with sensible defaults.
func DefaultAPIConfig() APIConfig {
	return APIConfig{
		MaxBodyBytes:           1 << 20, // 1 MiB
		RequireJSONContentType: false,
	}
}

// =============================================================================
// Config to ServerConfig Translation
// =============================================================================

// buildServerConfig converts user-friendly vango.Config to internal server.ServerConfig.
func buildServerConfig(cfg Config) *server.ServerConfig {
	serverCfg := server.DefaultServerConfig()

	// Session settings
	if cfg.Session.ResumeWindow > 0 {
		serverCfg.ResumeWindow = cfg.Session.ResumeWindow
	}
	if cfg.Session.Store != nil {
		serverCfg.SessionStore = cfg.Session.Store
	}
	if cfg.Session.MaxDetachedSessions > 0 {
		serverCfg.MaxDetachedSessions = cfg.Session.MaxDetachedSessions
	}
	if cfg.Session.MaxSessionsPerIP > 0 {
		serverCfg.MaxSessionsPerIP = cfg.Session.MaxSessionsPerIP
		serverCfg.EvictOnIPLimit = cfg.Session.EvictOnIPLimit
	}
	if cfg.Session.StormBudget != nil {
		serverCfg.SessionConfig.StormBudget = cfg.Session.StormBudget
	}
	if cfg.Session.AuthCheck != nil {
		authCheck := *cfg.Session.AuthCheck
		server.NormalizeAuthCheckConfig(&authCheck)
		serverCfg.SessionConfig.AuthCheck = &authCheck
	}

	// Security settings
	if cfg.Security.CSRFSecret != nil {
		serverCfg.CSRFSecret = cfg.Security.CSRFSecret
	}
	if len(cfg.Security.AllowedOrigins) > 0 {
		origins := make(map[string]bool)
		for _, o := range cfg.Security.AllowedOrigins {
			origins[o] = true
		}
		serverCfg.CheckOrigin = func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true // No origin header (same-origin or non-browser)
			}
			return origins[origin]
		}
	} else if cfg.Security.AllowSameOrigin {
		serverCfg.CheckOrigin = server.SameOriginCheck
	}
	if len(cfg.Security.TrustedProxies) > 0 {
		serverCfg.TrustedProxies = append([]string(nil), cfg.Security.TrustedProxies...)
	}
	serverCfg.SecureCookies = cfg.Security.CookieSecure
	if cfg.Security.CookieSameSite != 0 {
		serverCfg.SameSiteMode = cfg.Security.CookieSameSite
	}
	if cfg.Security.CookieDomain != "" {
		serverCfg.CookieDomain = cfg.Security.CookieDomain
	}

	// DevMode
	if cfg.DevMode {
		serverCfg = serverCfg.WithDevMode()
	}

	// Context bridge
	if cfg.OnSessionStart != nil {
		serverCfg.OnSessionStart = cfg.OnSessionStart
	}
	if cfg.OnSessionResume != nil {
		serverCfg.OnSessionResume = cfg.OnSessionResume
	}

	return serverCfg
}
