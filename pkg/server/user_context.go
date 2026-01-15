package server

import (
	"context"
	"errors"
)

// DefaultAuthSessionKey is the conventional session key for an authenticated user.
// The vango/pkg/auth helpers use this key, and server.Ctx.User() falls back to it.
const DefaultAuthSessionKey = "vango_auth_user"

// ErrUnauthorized is returned when authentication is required but not present.
// The auth package re-exports this as auth.ErrUnauthorized.
var ErrUnauthorized = errors.New("unauthorized: authentication required")

// ErrForbidden is returned when authentication is present but insufficient.
// The auth package re-exports this as auth.ErrForbidden.
var ErrForbidden = errors.New("forbidden: insufficient permissions")

// IsAuthError returns true if the error is an authentication or authorization error.
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrUnauthorized) || errors.Is(err, ErrForbidden)
}

type userContextKey struct{}

// WithUser stores an authenticated user in a standard library context.
// This is useful for bridging from HTTP middleware into Vango SSR and OnSessionStart.
func WithUser(ctx context.Context, user any) context.Context {
	return context.WithValue(ctx, userContextKey{}, user)
}

// UserFromContext retrieves the authenticated user stored by WithUser.
func UserFromContext(ctx context.Context) any {
	if ctx == nil {
		return nil
	}
	return ctx.Value(userContextKey{})
}

