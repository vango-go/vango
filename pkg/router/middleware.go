package router

import "github.com/vango-dev/vango/v2/pkg/server"

// ComposeMiddleware builds a handler chain from middleware and a final handler.
// Middleware is executed in order (first to last), with the handler at the end.
func ComposeMiddleware(ctx server.Ctx, mw []Middleware, handler func() error) error {
	if len(mw) == 0 {
		return handler()
	}

	// Build chain from end to start
	var chain func() error
	chain = handler

	for i := len(mw) - 1; i >= 0; i-- {
		m := mw[i]
		next := chain
		chain = func() error {
			return m.Handle(ctx, next)
		}
	}

	return chain()
}

// Chain creates a middleware that combines multiple middleware in order.
func Chain(middleware ...Middleware) Middleware {
	return MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
		return ComposeMiddleware(ctx, middleware, next)
	})
}

// Skip is a middleware that skips to the next middleware based on a condition.
func Skip(condition func(ctx server.Ctx) bool, mw Middleware) Middleware {
	return MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
		if condition(ctx) {
			return next()
		}
		return mw.Handle(ctx, next)
	})
}

// Only is a middleware that runs only if a condition is true.
func Only(condition func(ctx server.Ctx) bool, mw Middleware) Middleware {
	return MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
		if !condition(ctx) {
			return next()
		}
		return mw.Handle(ctx, next)
	})
}
