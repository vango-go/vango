package auth

import (
	"errors"
	"log/slog"
	"net/http"
	"reflect"
)

// Session provides minimal session access needed by auth helpers.
type Session interface {
	Get(key string) any
	Set(key string, value any)
	Delete(key string)
}

// Ctx provides minimal context access needed by auth helpers.
type Ctx interface {
	User() any
	SetUser(user any)
}

type sessionCtx interface {
	AuthSession() Session
}

// DebugMode enables extra validation and logging for development.
var DebugMode bool

func isNilSession(session Session) bool {
	if session == nil {
		return true
	}
	v := reflect.ValueOf(session)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Interface, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

// SessionKey is the standard session key for the authenticated user.
// The Context Bridge should use this key when storing user data.
const SessionKey = "vango_auth_user"

// sessionPresenceKey marks that a session was authenticated.
// This survives serialization (unlike the user object) and allows
// resume logic to detect "was authenticated but needs revalidation."
const sessionPresenceKey = SessionKey + ":present"

// ErrUnauthorized is returned when authentication is required but not present.
// This typically triggers a 401 response or redirect to login.
var ErrUnauthorized = errors.New("unauthorized: authentication required")

// ErrForbidden is returned when authentication is present but insufficient.
// This typically triggers a 403 response.
var ErrForbidden = errors.New("forbidden: insufficient permissions")

// Get retrieves the authenticated user from the context.
// Works in both SSR and WebSocket modes by using ctx.User() as the source of truth.
//
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
func Get[T any](ctx Ctx) (T, bool) {
	// Use ctx.User() as the canonical source - it already chains:
	// c.user → session data → request context (for SSR)
	val := ctx.User()
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
	if DebugMode {
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
// Use in middleware or action handlers that require authentication.
//
// Note: Page handlers in Vango don't return errors. Use authmw.RequireAuth
// middleware to protect routes, then use auth.Get in the handler.
//
// Example (middleware):
//
//	func Middleware() []router.Middleware {
//	    return []router.Middleware{authmw.RequireAuth}
//	}
//
// Example (action handler):
//
//	func DeleteProject(ctx vango.Ctx, id int) error {
//	    user, err := auth.Require[*models.User](ctx)
//	    if err != nil {
//	        return err
//	    }
//	    // ...
//	}
func Require[T any](ctx Ctx) (T, error) {
	user, ok := Get[T](ctx)
	if !ok {
		return user, ErrUnauthorized
	}
	return user, nil
}

// Set stores the authenticated user in the session.
// Also sets an auth presence flag that survives session serialization.
//
// Typically called from OnSessionStart or OnSessionResume in the Context Bridge.
//
// Example:
//
//	OnSessionStart: func(httpCtx context.Context, session *vango.Session) {
//	    if user := myauth.UserFromContext(httpCtx); user != nil {
//	        auth.Set(session, user)
//	    }
//	}
func Set[T any](session Session, user T) {
	if isNilSession(session) {
		return
	}
	session.Set(SessionKey, user)
	session.Set(sessionPresenceKey, true)
	session.Set(SessionKeyHadAuth, true)
}

// Login authenticates the user for both the current request and the session.
// Use this in login handlers to establish authentication.
//
// This is the recommended way to authenticate a user during a session:
// - Sets the user on the current request context
// - Persists to the session (if available) for subsequent requests
// - Sets the presence flag for resume validation
//
// Example:
//
//	func HandleLogin(ctx vango.Ctx, email, password string) error {
//	    user, err := validateCredentials(email, password)
//	    if err != nil {
//	        return err
//	    }
//	    auth.Login(ctx, user)
//	    ctx.Navigate("/dashboard")
//	    return nil
//	}
func Login[T any](ctx Ctx, user T) {
	// Set on current request context
	ctx.SetUser(user)

	// Persist to session if available (WS mode)
	if session := sessionFromCtx(ctx); session != nil {
		Set(session, user)
	}
}

// Clear removes the authenticated user from the session.
// Also clears the auth presence flag.
// Call this on logout.
//
// Example:
//
//	func Logout(ctx vango.Ctx) error {
//	    auth.Clear(ctx.Session())
//	    ctx.Navigate("/login")
//	    return nil
//	}
func Clear(session Session) {
	if isNilSession(session) {
		return
	}
	session.Delete(SessionKey)
	session.Delete(SessionKeyPrincipal)
	session.Delete(SessionKeyExpiryUnixMs)
	session.Delete(sessionPresenceKey)
	session.Delete(SessionKeyHadAuth)
}

// Logout clears authentication from both the request context and session.
// This is the recommended way to log out a user.
//
// Example:
//
//	func HandleLogout(ctx vango.Ctx) error {
//	    auth.Logout(ctx)
//	    ctx.Navigate("/")
//	    return nil
//	}
func Logout(ctx Ctx) {
	ctx.SetUser(nil)
	if session := sessionFromCtx(ctx); session != nil {
		Clear(session)
	}
}

// IsAuthenticated returns whether the context has an authenticated user.
// Works in both SSR and WebSocket modes.
//
// Example:
//
//	if auth.IsAuthenticated(ctx) {
//	    // Show logged-in UI
//	}
func IsAuthenticated(ctx Ctx) bool {
	return ctx.User() != nil
}

// WasAuthenticated checks if the session had authentication before.
// Used internally by resume logic to detect "was authenticated but auth now invalid."
// Returns false if session is nil.
func WasAuthenticated(session Session) bool {
	if isNilSession(session) {
		return false
	}
	if val := session.Get(SessionKeyHadAuth); val != nil {
		if present, ok := val.(bool); ok && present {
			return true
		}
	}

	val := session.Get(sessionPresenceKey)
	if val == nil {
		return false
	}
	present, ok := val.(bool)
	return ok && present
}

// MustGet is like Get but panics if authentication fails.
// Use sparingly, prefer Require for proper error handling.
func MustGet[T any](ctx Ctx) T {
	user, ok := Get[T](ctx)
	if !ok {
		panic("auth.MustGet: user not authenticated")
	}
	return user
}

// StatusCode returns the appropriate HTTP status code for an auth error.
// Returns (statusCode, true) for auth errors, (0, false) otherwise.
//
// Example:
//
//	if code, ok := auth.StatusCode(err); ok {
//	    w.WriteHeader(code)
//	}
func StatusCode(err error) (int, bool) {
	if err == nil {
		return 0, false
	}
	switch {
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized, true
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden, true
	default:
		return 0, false
	}
}

// IsAuthError returns true if the error is an authentication or authorization error.
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrUnauthorized) || errors.Is(err, ErrForbidden)
}

// SessionPresenceKey returns the key used to track auth presence.
// Exported for session serialization to skip the user object but keep the flag.
func SessionPresenceKey() string {
	return sessionPresenceKey
}

func sessionFromCtx(ctx Ctx) Session {
	if ctx == nil {
		return nil
	}
	if sc, ok := ctx.(sessionCtx); ok {
		return sc.AuthSession()
	}
	return nil
}
