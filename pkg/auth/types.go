package auth

import (
	"context"
	"errors"
	"net/http"
)

// Session keys — presence of SessionKeyExpiryUnixMs enables automatic passive checks.
const (
	SessionKeyPrincipal = "vango:auth:principal"
	// SessionKeyExpiryUnixMs is the hard expiry timestamp in unix milliseconds.
	// Stored as int64 to remain robust even if an app accidentally serializes it.
	SessionKeyExpiryUnixMs = "vango:auth:expiry_unix_ms"

	// SessionKeyHadAuth is a non-authoritative marker indicating the session
	// previously had authenticated state (useful for logs/telemetry and resume UX).
	// This key MAY be persisted (it's a simple boolean), but it MUST NOT be an authority source.
	SessionKeyHadAuth = "vango:auth:had_auth"
)

var (
	// RuntimeOnlySessionKeys lists keys that MUST NOT be persisted by session serializers.
	RuntimeOnlySessionKeys = []string{
		SessionKeyPrincipal,
		SessionKeyExpiryUnixMs,
	}

	// ErrSessionExpired indicates the session is no longer valid due to expiry.
	ErrSessionExpired = errors.New("session expired")

	// ErrSessionRevoked indicates the session is no longer valid due to revocation.
	ErrSessionRevoked = errors.New("session revoked")
)

// Principal represents the authenticated identity.
// Intentionally minimal — no catch-all Claims map to prevent leakage.
type Principal struct {
	// User identity
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`

	// Authorization
	Roles    []string `json:"roles,omitempty"`
	TenantID string   `json:"tenant_id,omitempty"`

	// Provider session (for active verification)
	SessionID string `json:"session_id,omitempty"`

	// Expiration
	ExpiresAtUnixMs int64 `json:"expires_at_unix_ms"`
	AuthVersion     int   `json:"auth_version,omitempty"`
}

// Provider adapts an identity provider to Vango.
type Provider interface {
	// Middleware validates HTTP requests and populates context.
	Middleware() func(http.Handler) http.Handler

	// Principal extracts identity from validated request context.
	Principal(ctx context.Context) (Principal, bool)

	// Verify checks if session is still valid (for active revalidation).
	Verify(ctx context.Context, p Principal) error
}

// SetPrincipal stores the principal and expiry keys on the session.
// This marks the session as previously authenticated.
func SetPrincipal(session Session, principal Principal) {
	if isNilSession(session) {
		return
	}
	session.Set(SessionKeyPrincipal, principal)
	session.Set(SessionKeyExpiryUnixMs, principal.ExpiresAtUnixMs)
	session.Set(SessionKeyHadAuth, true)
	session.Set(sessionPresenceKey, true)
}
