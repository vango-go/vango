package auth

import (
	"github.com/vango-dev/vango/v2/pkg/router"
	"github.com/vango-dev/vango/v2/pkg/server"
)

// RequireAuth is Vango middleware that requires authentication.
// Use on routes that should only be accessible to logged-in users.
//
// Usage in file-based routing:
//
//	// app/routes/dashboard/_layout.go
//	func Middleware() []router.Middleware {
//	    return []router.Middleware{
//	        auth.RequireAuth,
//	    }
//	}
var RequireAuth router.Middleware = router.MiddlewareFunc(
	func(ctx server.Ctx, next func() error) error {
		if !IsAuthenticated(ctx) {
			return ErrUnauthorized
		}
		return next()
	},
)

// RequireRole returns middleware that requires a specific role.
// The check function receives the user and returns true if authorized.
//
// Usage:
//
//	func Middleware() []router.Middleware {
//	    return []router.Middleware{
//	        auth.RequireRole(func(u *models.User) bool {
//	            return u.Role == "admin"
//	        }),
//	    }
//	}
func RequireRole[T any](check func(T) bool) router.Middleware {
	return router.MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
		user, ok := Get[T](ctx)
		if !ok {
			return ErrUnauthorized
		}
		if !check(user) {
			return ErrForbidden
		}
		return next()
	})
}

// RequirePermission returns middleware that checks for a specific permission.
// This is semantically equivalent to RequireRole but communicates intent better
// for permission-based authorization.
//
// Usage:
//
//	auth.RequirePermission(func(u *models.User) bool {
//	    return u.Can("projects.delete")
//	})
func RequirePermission[T any](check func(T) bool) router.Middleware {
	return RequireRole(check) // Same implementation, semantic alias
}

// RequireAny returns middleware that requires at least one of the checks to pass.
//
// Usage:
//
//	auth.RequireAny(
//	    func(u *User) bool { return u.IsAdmin },
//	    func(u *User) bool { return u.IsOwner(resourceID) },
//	)
func RequireAny[T any](checks ...func(T) bool) router.Middleware {
	return router.MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
		user, ok := Get[T](ctx)
		if !ok {
			return ErrUnauthorized
		}

		for _, check := range checks {
			if check(user) {
				return next()
			}
		}

		return ErrForbidden
	})
}

// RequireAll returns middleware that requires all checks to pass.
//
// Usage:
//
//	auth.RequireAll(
//	    func(u *User) bool { return u.IsActive },
//	    func(u *User) bool { return u.EmailVerified },
//	)
func RequireAll[T any](checks ...func(T) bool) router.Middleware {
	return router.MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
		user, ok := Get[T](ctx)
		if !ok {
			return ErrUnauthorized
		}

		for _, check := range checks {
			if !check(user) {
				return ErrForbidden
			}
		}

		return next()
	})
}
