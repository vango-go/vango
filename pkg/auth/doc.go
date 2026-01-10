// Package auth provides type-safe authentication helpers for Vango.
//
// The auth package is designed to be auth-system agnostic. It doesn't
// validate tokens or manage sessions—that's the responsibility of HTTP
// middleware in Layer 1 (the infrastructure layer). Instead, it provides
// type-safe access to the user data that was hydrated via the Context Bridge.
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
// # Basic Usage
//
// For routes that require authentication:
//
//	func Dashboard(ctx vango.Ctx) (vango.Component, error) {
//	    user, err := auth.Require[*models.User](ctx)
//	    if err != nil {
//	        return nil, err // Returns 401, handled by error boundary
//	    }
//	    return renderDashboard(user), nil
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
// # Middleware
//
// Use auth middleware to protect entire route segments:
//
//	// In app/routes/admin/middleware.go
//	func Middleware() []router.Middleware {
//	    return []router.Middleware{
//	        auth.RequireAuth,
//	        auth.RequireRole(func(u *models.User) bool {
//	            return u.IsAdmin
//	        }),
//	    }
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
