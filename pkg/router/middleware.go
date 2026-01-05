package router

import (
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/vango-go/vango/pkg/server"
)

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

// =============================================================================
// Enhanced Middleware (Phase 10C)
// =============================================================================

// WithValue returns middleware that sets a context value for the request.
// The value is available to subsequent middleware and handlers via ctx.Value().
func WithValue(key, value any) Middleware {
	return MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
		ctx.SetValue(key, value)
		return next()
	})
}

// Logger returns middleware that logs each event with timing information.
// This is for Vango event-loop logging (Layer 2), not HTTP request logging.
func Logger(logger *slog.Logger) Middleware {
	return MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
		start := time.Now()
		err := next()

		session := ctx.Session()
		sessionID := ""
		if session != nil {
			sessionID = session.ID
		}

		if err != nil {
			logger.Error("event error",
				"session", sessionID,
				"path", ctx.Path(),
				"duration", time.Since(start),
				"error", err,
			)
		} else {
			logger.Info("event handled",
				"session", sessionID,
				"path", ctx.Path(),
				"duration", time.Since(start),
			)
		}

		return err
	})
}

// Recover returns middleware that recovers from panics.
// The onPanic callback is called with the recovered value and can log or handle it.
func Recover(onPanic func(ctx server.Ctx, recovered any)) Middleware {
	return MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
		defer func() {
			if r := recover(); r != nil {
				if onPanic != nil {
					onPanic(ctx, r)
				}
			}
		}()
		return next()
	})
}

// ErrRateLimited is returned when the rate limit is exceeded.
var ErrRateLimited = errors.New("rate limit exceeded")

// rateLimiter is a simple token bucket rate limiter.
type rateLimiter struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
}

func newRateLimiter(maxPerSecond int) *rateLimiter {
	return &rateLimiter{
		tokens:     float64(maxPerSecond),
		maxTokens:  float64(maxPerSecond),
		refillRate: float64(maxPerSecond),
		lastRefill: time.Now(),
	}
}

func (l *rateLimiter) allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(l.lastRefill).Seconds()
	l.tokens += elapsed * l.refillRate
	if l.tokens > l.maxTokens {
		l.tokens = l.maxTokens
	}
	l.lastRefill = now

	// Try to consume a token
	if l.tokens >= 1 {
		l.tokens--
		return true
	}

	return false
}

// sessionRateLimitKey is the key for storing rate limiter in session.
const sessionRateLimitKey = "_vango_rate_limiter"

// RateLimit returns middleware that limits events per session.
// maxPerSecond specifies the maximum number of events allowed per second.
// Exceeding the limit returns ErrRateLimited.
func RateLimit(maxPerSecond int) Middleware {
	return MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
		session := ctx.Session()
		if session == nil {
			// No session, allow the request
			return next()
		}

		// Get or create rate limiter for this session
		limiterVal := session.Get(sessionRateLimitKey)
		var limiter *rateLimiter
		if limiterVal != nil {
			limiter = limiterVal.(*rateLimiter)
		} else {
			limiter = newRateLimiter(maxPerSecond)
			session.Set(sessionRateLimitKey, limiter)
		}

		if !limiter.allow() {
			return ErrRateLimited
		}

		return next()
	})
}

// Timeout returns middleware that times out after the specified duration.
// Note: This only affects the handler execution, not the full event processing.
func Timeout(duration time.Duration) Middleware {
	return MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
		done := make(chan error, 1)

		go func() {
			done <- next()
		}()

		select {
		case err := <-done:
			return err
		case <-time.After(duration):
			return errors.New("handler timeout")
		}
	})
}
