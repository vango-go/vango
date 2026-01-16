// Package auth provides type-safe authentication helpers for Vango.
//
// The auth package is designed to be auth-system agnostic. It doesn't
// validate tokens or manage sessions—that's the responsibility of HTTP
// middleware in Layer 1 (the infrastructure layer). Instead, it provides
// type-safe access to the user data that was hydrated via the Context Bridge.
//
// # SSR and WebSocket Compatibility
//
// The auth package works seamlessly in both SSR and WebSocket modes:
//
//   - auth.Get and auth.IsAuthenticated use ctx.User() as the source of truth
//   - ctx.User() checks: per-request user (SetUser) → session data → HTTP request context (SSR)
//   - This means the same component code works in both modes
//
// # The Two-Layer Architecture
//
// Vango operates across two distinct lifecycles:
//
//   - Layer 1 (HTTP): Runs ONCE per session during initial GET + WebSocket upgrade.
//     Standard HTTP middleware (Chi, Gorilla, etc.) handles authentication here.
//
//   - Layer 2 (WebSocket): Runs HUNDREDS of times per session for each event.
//     The auth package operates here, providing type-safe access to user data.
//
// # The Context Bridge
//
// Data flows from Layer 1 to Layer 2 via the Context Bridge:
//
//	OnSessionStart: func(httpCtx context.Context, session *vango.Session) {
//	    if user := myauth.UserFromContext(httpCtx); user != nil {
//	        auth.Set(session, user)  // Copy to session
//	    }
//	}
//
// After this runs, r.Context() is dead (HTTP request complete), but the
// session persists with the hydrated user data.
//
// # Session Resume
//
// When a client reconnects (e.g., after a page refresh within ResumeWindow),
// auth data is revalidated via OnSessionResume:
//
//	OnSessionResume: func(httpCtx context.Context, session *vango.Session) error {
//	    user, err := myauth.ValidateFromContext(httpCtx)
//	    if err != nil {
//	        return err  // Reject resume if previously authenticated
//	    }
//	    if user != nil {
//	        auth.Set(session, user)
//	    }
//	    return nil
//	}
//
// Important: User objects are NOT serialized to session storage. They must be
// rehydrated from cookies/headers on each handshake and resume. This ensures
// auth state is always validated against the current request.
//
// Principal + expiry keys (SessionKeyPrincipal, SessionKeyExpiryUnixMs) are
// runtime-only and must also be rehydrated on start/resume. Session persistence
// MUST skip these keys; use auth.RuntimeOnlySessionKeys as the allowlist.
//
// To enable auth freshness checks, set a Principal with an explicit expiry:
//
//	auth.SetPrincipal(session, auth.Principal{
//	    ID:              user.ID,
//	    Email:           user.Email,
//	    ExpiresAtUnixMs: expiresAt.UnixMilli(),
//	})
//
// If ExpiresAtUnixMs is zero, SetPrincipal will not set SessionKeyExpiryUnixMs,
// so passive expiry checks remain disabled unless you set the key explicitly.
//
// # Auth Freshness (Passive + Active)
//
// Passive expiry is enforced on every WebSocket event when
// SessionKeyExpiryUnixMs is present.
//
// Active revalidation is configured via SessionConfig.AuthCheck:
//
//	app := vango.New(vango.Config{
//	    Session: vango.SessionConfig{
//	        AuthCheck: &vango.AuthCheckConfig{
//	            Interval: 2 * time.Minute,
//	            Check:    myProvider.Verify,
//	            OnExpired: vango.AuthExpiredConfig{
//	                Action: vango.ForceReload,
//	            },
//	        },
//	    },
//	})
//
// For high-value operations, use ctx.RevalidateAuth() to force an immediate
// active check (fail-closed).
//
// # Session-First Adapter
//
// The sessionauth package provides a reference adapter for session stores:
//
//	provider := sessionauth.New(store)
//	r.Use(provider.Middleware())
//	principal, ok := provider.Principal(r.Context())
//
// # Basic Usage
//
// Use middleware to protect routes that require authentication:
//
//	// In app/routes/dashboard/middleware.go
//	func Middleware() []router.Middleware {
//	    return []router.Middleware{authmw.RequireAuth}
//	}
//
//	// In app/routes/dashboard/page.go
//	func Dashboard(ctx vango.Ctx) vango.Component {
//	    // Middleware guarantees user is authenticated
//	    user, _ := auth.Get[*models.User](ctx)
//	    return renderDashboard(user)
//	}
//
// For routes where auth is optional (guest allowed):
//
//	func HomePage(ctx vango.Ctx) vango.Component {
//	    user, ok := auth.Get[*models.User](ctx)
//	    if ok {
//	        return renderLoggedInHome(user)
//	    }
//	    return renderGuestHome()
//	}
//
// # Login and Logout
//
// Use auth.Login to authenticate a user during the session:
//
//	func HandleLogin(ctx vango.Ctx, email, password string) error {
//	    user, err := validateCredentials(email, password)
//	    if err != nil {
//	        return err
//	    }
//	    auth.Login(ctx, user)  // Sets both request context and session
//	    ctx.Navigate("/dashboard")
//	    return nil
//	}
//
// Use auth.Logout to clear authentication:
//
//	func HandleLogout(ctx vango.Ctx) error {
//	    auth.Logout(ctx)
//	    ctx.Navigate("/")
//	    return nil
//	}
//
// # Middleware
//
// Use authmw middleware to protect entire route segments:
//
//	// In app/routes/admin/middleware.go
//	func Middleware() []router.Middleware {
//	    return []router.Middleware{
//	        authmw.RequireAuth,
//	        authmw.RequireRole(func(u *models.User) bool {
//	            return u.IsAdmin
//	        }),
//	    }
//	}
//
// # Error Handling
//
// Auth errors are mapped to appropriate HTTP status codes:
//
//   - auth.ErrUnauthorized → 401 Unauthorized
//   - auth.ErrForbidden → 403 Forbidden
//
// In SSR and API routes, these errors produce the correct HTTP status.
// In WebSocket navigation, they produce protocol.ErrNotAuthorized.
//
// Use auth.Require in action handlers for explicit error handling:
//
//	func DeleteProject(ctx vango.Ctx, id int) error {
//	    user, err := auth.Require[*models.User](ctx)
//	    if err != nil {
//	        return err  // Returns 401/ErrNotAuthorized
//	    }
//	    // ... delete logic
//	}
//
// # Type Safety
//
// The auth package uses Go generics for type-safe user retrieval:
//
//   - auth.Get[*User](ctx) returns (*User, bool)
//   - auth.Require[*User](ctx) returns (*User, error)
//
// In debug mode (ServerConfig.DebugMode = true), the package logs warnings
// when type assertions fail, helping catch common mistakes like storing
// a value type but requesting a pointer type.
package auth
