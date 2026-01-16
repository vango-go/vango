package sessionauth

import (
	"context"
	"net/http"
	"time"

	"github.com/vango-go/vango/pkg/auth"
)

// StoredSession represents a validated session from a backing store.
type StoredSession struct {
	ID          string
	UserID      string
	Email       string
	Name        string
	Roles       []string
	TenantID    string
	ExpiresAt   time.Time
	AuthVersion int
}

// SessionStore is the backing store for session-first auth.
type SessionStore interface {
	Get(ctx context.Context, sessionID string) (*StoredSession, error)
	Validate(ctx context.Context, session *StoredSession) error
}

// Provider adapts a session store to Vango's auth.Provider interface.
type Provider struct {
	store      SessionStore
	cookieName string
	cookiePolicy CookiePolicy
}

// Option configures a Provider.
type Option func(*Provider)

// WithCookieName sets the cookie name used to load session IDs.
func WithCookieName(name string) Option {
	return func(p *Provider) {
		if name != "" {
			p.cookieName = name
		}
	}
}

// CookiePolicy applies security defaults to cookies set by the provider.
type CookiePolicy interface {
	ApplyCookiePolicy(r *http.Request, cookie *http.Cookie) (*http.Cookie, error)
}

// WithCookiePolicy applies a cookie policy for provider-managed cookies.
func WithCookiePolicy(policy CookiePolicy) Option {
	return func(p *Provider) {
		p.cookiePolicy = policy
	}
}

// SetCookiePolicy updates the cookie policy after provider creation.
func (p *Provider) SetCookiePolicy(policy CookiePolicy) {
	p.cookiePolicy = policy
}

// New creates a session-first auth provider.
func New(store SessionStore, opts ...Option) *Provider {
	p := &Provider{
		store:      store,
		cookieName: "session",
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Middleware validates the session cookie and injects the stored session into context.
func (p *Provider) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(p.cookieName)
			if err != nil || cookie.Value == "" {
				next.ServeHTTP(w, r)
				return
			}

			stored, err := p.store.Get(r.Context(), cookie.Value)
			if err != nil {
				p.clearCookie(w, r)
				next.ServeHTTP(w, r)
				return
			}

			if err := p.store.Validate(r.Context(), stored); err != nil {
				p.clearCookie(w, r)
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), sessionContextKey{}, stored)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Principal extracts the auth.Principal from a validated request context.
func (p *Provider) Principal(ctx context.Context) (auth.Principal, bool) {
	stored, ok := SessionFromContext(ctx)
	if !ok || stored == nil {
		return auth.Principal{}, false
	}

	return auth.Principal{
		ID:              stored.UserID,
		Email:           stored.Email,
		Name:            stored.Name,
		Roles:           stored.Roles,
		TenantID:        stored.TenantID,
		SessionID:       stored.ID,
		ExpiresAtUnixMs: stored.ExpiresAt.UnixMilli(),
		AuthVersion:     stored.AuthVersion,
	}, true
}

// Verify checks whether the session is still valid for active revalidation.
func (p *Provider) Verify(ctx context.Context, principal auth.Principal) error {
	if principal.SessionID == "" {
		return nil
	}
	stored, err := p.store.Get(ctx, principal.SessionID)
	if err != nil {
		return err
	}
	return p.store.Validate(ctx, stored)
}

func (p *Provider) clearCookie(w http.ResponseWriter, r *http.Request) {
	cookie := &http.Cookie{
		Name:     p.cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   r != nil && r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	}
	if p.cookiePolicy != nil {
		updated, err := p.cookiePolicy.ApplyCookiePolicy(r, cookie)
		if err != nil {
			return
		}
		cookie = updated
	}
	http.SetCookie(w, cookie)
}

// SessionFromContext returns the stored session from context.
func SessionFromContext(ctx context.Context) (*StoredSession, bool) {
	if ctx == nil {
		return nil, false
	}
	stored, ok := ctx.Value(sessionContextKey{}).(*StoredSession)
	return stored, ok && stored != nil
}

type sessionContextKey struct{}
