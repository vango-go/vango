package vango

import (
	"errors"
	"fmt"
)

// =============================================================================
// Phase 16: Sentinel Errors for Structured Side Effects
// =============================================================================

// ErrBudgetExceeded is returned when a storm budget limit is exceeded.
// This happens when too many operations (Resource fetches, Action runs, etc.)
// occur within the configured time window.
//
// Applications should handle this by:
// - Logging the event for debugging
// - Optionally showing user feedback about rate limiting
// - Reducing the frequency of operations if possible
//
// See SPEC_ADDENDUM.md §A.4 for storm budget configuration.
var ErrBudgetExceeded = errors.New("vango: storm budget exceeded")

// ErrQueueFull is returned when an Action's queue is full and cannot accept
// more work items. This applies to Actions with ConcurrencyQueue policy.
//
// Applications should handle this by:
// - Informing the user their action was not queued
// - Waiting before retrying
// - Using a different concurrency policy if appropriate
//
// See SPEC_ADDENDUM.md §A.1.5 for concurrency policies.
var ErrQueueFull = errors.New("vango: action queue full")

// ErrActionRunning is returned when attempting to run an Action that is
// already in the Running state and the concurrency policy is DropWhileRunning.
//
// Applications can safely ignore this error as it's expected behavior
// for de-duplicating rapid user actions.
var ErrActionRunning = errors.New("vango: action already running")

// ErrEffectContext is returned when an effect helper (Interval, Subscribe,
// GoLatest) is called outside of an effect body or render context.
//
// These helpers require access to the runtime context (Ctx) and must be
// called within CreateEffect or during component render.
var ErrEffectContext = errors.New("vango: effect helper called outside effect/render context")

// ErrGoLatestContext is returned when GoLatest is called outside an effect body.
// GoLatest requires effect-local storage and must be called from within CreateEffect.
var ErrGoLatestContext = errors.New("vango: GoLatest must be called inside an Effect")

// =============================================================================
// HTTP Error Helpers for API Routes
// =============================================================================

// HTTPError represents an HTTP error with a status code and message.
// It implements the error interface and can be returned from API handlers
// to send appropriate HTTP status codes to clients.
type HTTPError struct {
	Code    int    // HTTP status code (e.g., 400, 403, 404, 500)
	Message string // Error message to return to client
	Err     error  // Optional underlying error
}

// Error implements the error interface.
func (e *HTTPError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap returns the underlying error for errors.Is/As support.
func (e *HTTPError) Unwrap() error {
	return e.Err
}

// StatusCode returns the HTTP status code for this error.
func (e *HTTPError) StatusCode() int {
	return e.Code
}

// BadRequest creates a 400 Bad Request error.
// Use this when the client sent invalid data.
//
// Example:
//
//	if err := validate(input); err != nil {
//	    return nil, vango.BadRequest(err)
//	}
func BadRequest(err error) *HTTPError {
	msg := "bad request"
	if err != nil {
		msg = err.Error()
	}
	return &HTTPError{Code: 400, Message: msg, Err: err}
}

// BadRequestf creates a 400 Bad Request error with a formatted message.
func BadRequestf(format string, args ...any) *HTTPError {
	return &HTTPError{Code: 400, Message: fmt.Sprintf(format, args...)}
}

// Unauthorized creates a 401 Unauthorized error.
// Use this when authentication is required but not provided.
func Unauthorized(message ...string) *HTTPError {
	msg := "unauthorized"
	if len(message) > 0 {
		msg = message[0]
	}
	return &HTTPError{Code: 401, Message: msg}
}

// Forbidden creates a 403 Forbidden error.
// Use this when the user is authenticated but lacks permission.
//
// Example:
//
//	if !user.HasRole("admin") {
//	    return nil, vango.Forbidden()
//	}
func Forbidden(message ...string) *HTTPError {
	msg := "forbidden"
	if len(message) > 0 {
		msg = message[0]
	}
	return &HTTPError{Code: 403, Message: msg}
}

// NotFound creates a 404 Not Found error.
// Use this when the requested resource doesn't exist.
func NotFound(message ...string) *HTTPError {
	msg := "not found"
	if len(message) > 0 {
		msg = message[0]
	}
	return &HTTPError{Code: 404, Message: msg}
}

// Conflict creates a 409 Conflict error.
// Use this when the request conflicts with the current state.
func Conflict(message ...string) *HTTPError {
	msg := "conflict"
	if len(message) > 0 {
		msg = message[0]
	}
	return &HTTPError{Code: 409, Message: msg}
}

// UnprocessableEntity creates a 422 Unprocessable Entity error.
// Use this for validation errors on semantically correct but invalid data.
func UnprocessableEntity(message ...string) *HTTPError {
	msg := "unprocessable entity"
	if len(message) > 0 {
		msg = message[0]
	}
	return &HTTPError{Code: 422, Message: msg}
}

// InternalError creates a 500 Internal Server Error.
// Use this for unexpected server errors. Consider logging the underlying error.
func InternalError(err error) *HTTPError {
	return &HTTPError{Code: 500, Message: "internal server error", Err: err}
}

// ServiceUnavailable creates a 503 Service Unavailable error.
// Use this when the server is temporarily unable to handle requests.
func ServiceUnavailable(message ...string) *HTTPError {
	msg := "service unavailable"
	if len(message) > 0 {
		msg = message[0]
	}
	return &HTTPError{Code: 503, Message: msg}
}
