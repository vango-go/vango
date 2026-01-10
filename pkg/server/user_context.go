package server

import "context"

// DefaultAuthSessionKey is the conventional session key for an authenticated user.
// The vango/pkg/auth helpers use this key, and server.Ctx.User() falls back to it.
const DefaultAuthSessionKey = "vango_auth_user"

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

