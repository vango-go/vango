package auth

import (
	"errors"
	"log/slog"
	"reflect"

	"github.com/vango-go/vango/pkg/server"
)

// SessionKey is the standard session key for the authenticated user.
// The Context Bridge should use this key when storing user data.
const SessionKey = server.DefaultAuthSessionKey

// ErrUnauthorized is returned when authentication is required but not present.
// This typically triggers a 401 response or redirect to login.
var ErrUnauthorized = errors.New("unauthorized: authentication required")

// ErrForbidden is returned when authentication is present but insufficient.
// This typically triggers a 403 response.
var ErrForbidden = errors.New("forbidden: insufficient permissions")

// Get retrieves the authenticated user from the session.
// Returns (user, true) if authenticated, (zero, false) otherwise.
//
// In debug mode, logs a warning if a value exists but type assertion fails,
// helping developers catch common value/pointer mismatches.
//
// Example:
//
//	user, ok := auth.Get[*models.User](ctx)
//	if !ok {
//	    // User not authenticated
//	}
func Get[T any](ctx server.Ctx) (T, bool) {
	session := ctx.Session()
	if session == nil {
		var zero T
		return zero, false
	}

	val := session.Get(SessionKey)
	if val == nil {
		var zero T
		return zero, false
	}

	// Fast path: Type assertion succeeds
	if user, ok := val.(T); ok {
		return user, true
	}

	// Slow path: Debug aid for type mismatches
	// Only runs if value exists but assertion failed
	if server.DebugMode {
		storedType := reflect.TypeOf(val)
		var zero T
		requestedType := reflect.TypeOf(zero)

		// Handle the case where T is an interface or pointer
		if requestedType == nil {
			// T is an interface type, get it differently
			requestedType = reflect.TypeOf((*T)(nil)).Elem()
		}

		slog.Warn("vango/auth: type mismatch",
			"stored_type", storedType,
			"requested_type", requestedType,
			"hint", "Did you store a struct (User) but request a pointer (*User)?",
		)
	}

	var zero T
	return zero, false
}

// Require returns the authenticated user or an error.
// Use in components/handlers that require authentication.
//
// Example:
//
//	func Dashboard(ctx vango.Ctx) (vango.Component, error) {
//	    user, err := auth.Require[*models.User](ctx)
//	    if err != nil {
//	        return nil, err // Error boundary handles redirect
//	    }
//	    return renderDashboard(user), nil
//	}
func Require[T any](ctx server.Ctx) (T, error) {
	user, ok := Get[T](ctx)
	if !ok {
		return user, ErrUnauthorized
	}
	return user, nil
}

// Set stores the authenticated user in the session.
// Typically called from OnSessionStart in the Context Bridge.
//
// Example:
//
//	OnSessionStart: func(httpCtx context.Context, session *vango.Session) {
//	    if user := myauth.UserFromContext(httpCtx); user != nil {
//	        auth.Set(session, user)
//	    }
//	}
func Set[T any](session *server.Session, user T) {
	session.Set(SessionKey, user)
}

// Clear removes the authenticated user from the session.
// Call this on logout.
//
// Example:
//
//	func Logout(ctx vango.Ctx) error {
//	    auth.Clear(ctx.Session())
//	    ctx.Redirect("/", http.StatusSeeOther)
//	    return nil
//	}
func Clear(session *server.Session) {
	session.Delete(SessionKey)
}

// IsAuthenticated returns whether the session has an authenticated user.
// This is a quick check that doesn't require knowing the user type.
//
// Example:
//
//	if auth.IsAuthenticated(ctx) {
//	    // Show logged-in UI
//	}
func IsAuthenticated(ctx server.Ctx) bool {
	session := ctx.Session()
	if session == nil {
		return false
	}
	return session.Has(SessionKey)
}

// MustGet is like Get but panics if authentication fails.
// Use sparingly, prefer Require for proper error handling.
func MustGet[T any](ctx server.Ctx) T {
	user, ok := Get[T](ctx)
	if !ok {
		panic("auth.MustGet: user not authenticated")
	}
	return user
}
