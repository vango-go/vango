package server

import (
	"errors"
	"fmt"
)

// Sentinel errors for common session and server error conditions.
var (
	// ErrSessionClosed is returned when an operation is attempted on a closed session.
	ErrSessionClosed = errors.New("server: session closed")

	// ErrSessionNotFound is returned when a session ID does not exist.
	ErrSessionNotFound = errors.New("server: session not found")

	// ErrHandlerNotFound is returned when no handler is registered for an HID.
	ErrHandlerNotFound = errors.New("server: handler not found")

	// ErrEventQueueFull is returned when the event queue is full and an event is dropped.
	ErrEventQueueFull = errors.New("server: event queue full")

	// ErrInvalidHandshake is returned when the WebSocket handshake fails.
	ErrInvalidHandshake = errors.New("server: invalid handshake")

	// ErrMaxSessionsReached is returned when the maximum number of sessions is reached.
	ErrMaxSessionsReached = errors.New("server: max sessions reached")

	// ErrInvalidCSRF is returned when CSRF token validation fails.
	ErrInvalidCSRF = errors.New("server: invalid CSRF token")

	// ErrSessionExpired is returned when a session has expired due to inactivity.
	ErrSessionExpired = errors.New("server: session expired")

	// ErrConnectionClosed is returned when the WebSocket connection is closed.
	ErrConnectionClosed = errors.New("server: connection closed")

	// ErrWriteTimeout is returned when a write operation times out.
	ErrWriteTimeout = errors.New("server: write timeout")

	// ErrReadTimeout is returned when a read operation times out.
	ErrReadTimeout = errors.New("server: read timeout")

	// ErrNoConnection is returned when attempting to send on a nil connection.
	ErrNoConnection = errors.New("server: no connection")
)

// SessionError wraps an error with session context for debugging.
type SessionError struct {
	SessionID string
	Op        string // Operation that failed
	Err       error  // Underlying error
}

// Error returns the error message with session context.
func (e *SessionError) Error() string {
	if e.SessionID == "" {
		return fmt.Sprintf("server: %s: %v", e.Op, e.Err)
	}
	return fmt.Sprintf("server: session %s: %s: %v", e.SessionID, e.Op, e.Err)
}

// Unwrap returns the underlying error for errors.Is/As.
func (e *SessionError) Unwrap() error {
	return e.Err
}

// NewSessionError creates a new SessionError.
func NewSessionError(sessionID, op string, err error) *SessionError {
	return &SessionError{
		SessionID: sessionID,
		Op:        op,
		Err:       err,
	}
}

// HandlerError wraps a panic that occurred in an event handler.
type HandlerError struct {
	SessionID string
	HID       string
	EventType string
	Panic     any
	Stack     []byte
}

// Error returns the error message.
func (e *HandlerError) Error() string {
	return fmt.Sprintf("server: handler panic in session %s, HID %s, event %s: %v",
		e.SessionID, e.HID, e.EventType, e.Panic)
}

// NewHandlerError creates a new HandlerError.
func NewHandlerError(sessionID, hid, eventType string, panicVal any, stack []byte) *HandlerError {
	return &HandlerError{
		SessionID: sessionID,
		HID:       hid,
		EventType: eventType,
		Panic:     panicVal,
		Stack:     stack,
	}
}

// ProtocolError represents an error in the binary protocol.
type ProtocolError struct {
	SessionID string
	Op        string
	Message   string
}

// Error returns the error message.
func (e *ProtocolError) Error() string {
	return fmt.Sprintf("server: protocol error in session %s: %s: %s",
		e.SessionID, e.Op, e.Message)
}

// NewProtocolError creates a new ProtocolError.
func NewProtocolError(sessionID, op, message string) *ProtocolError {
	return &ProtocolError{
		SessionID: sessionID,
		Op:        op,
		Message:   message,
	}
}
