package server

import (
	"errors"
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{"ErrSessionClosed", ErrSessionClosed, "server: session closed"},
		{"ErrSessionNotFound", ErrSessionNotFound, "server: session not found"},
		{"ErrHandlerNotFound", ErrHandlerNotFound, "server: handler not found"},
		{"ErrEventQueueFull", ErrEventQueueFull, "server: event queue full"},
		{"ErrMaxSessionsReached", ErrMaxSessionsReached, "server: max sessions reached"},
		{"ErrInvalidHandshake", ErrInvalidHandshake, "server: invalid handshake"},
		{"ErrInvalidCSRF", ErrInvalidCSRF, "server: invalid CSRF token"},
		{"ErrSecureCookiesRequired", ErrSecureCookiesRequired, "server: secure cookies require HTTPS or trusted proxy headers"},
		{"ErrSessionExpired", ErrSessionExpired, "server: session expired"},
		{"ErrConnectionClosed", ErrConnectionClosed, "server: connection closed"},
		{"ErrWriteTimeout", ErrWriteTimeout, "server: write timeout"},
		{"ErrReadTimeout", ErrReadTimeout, "server: read timeout"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.msg {
				t.Errorf("Error message = %q, want %q", tt.err.Error(), tt.msg)
			}
		})
	}
}

func TestSessionError(t *testing.T) {
	cause := errors.New("connection reset")
	err := &SessionError{
		SessionID: "test-session-123",
		Op:        "write",
		Err:       cause,
	}

	// Test error message format
	expected := "server: session test-session-123: write: connection reset"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}

	// Test Unwrap
	if !errors.Is(err, cause) {
		t.Error("Unwrap should return the cause error")
	}
}

func TestSessionErrorWithoutSessionID(t *testing.T) {
	cause := errors.New("some error")
	err := &SessionError{
		Op:  "close",
		Err: cause,
	}

	expected := "server: close: some error"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestNewSessionError(t *testing.T) {
	cause := errors.New("test error")
	err := NewSessionError("session-1", "read", cause)

	if err.SessionID != "session-1" {
		t.Errorf("SessionID = %s, want session-1", err.SessionID)
	}
	if err.Op != "read" {
		t.Errorf("Op = %s, want read", err.Op)
	}
	if err.Err != cause {
		t.Error("Err should be the cause")
	}
}

func TestHandlerError(t *testing.T) {
	err := &HandlerError{
		SessionID: "sess-1",
		HID:       "hid-123",
		EventType: "click",
		Panic:     "runtime error: index out of range",
		Stack:     []byte("stack trace here"),
	}

	// Test error message format
	expected := "server: handler panic in session sess-1, HID hid-123, event click: runtime error: index out of range"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestNewHandlerError(t *testing.T) {
	stack := []byte("test stack")
	err := NewHandlerError("sess-2", "hid-456", "submit", "panic!", stack)

	if err.SessionID != "sess-2" {
		t.Errorf("SessionID = %s, want sess-2", err.SessionID)
	}
	if err.HID != "hid-456" {
		t.Errorf("HID = %s, want hid-456", err.HID)
	}
	if err.EventType != "submit" {
		t.Errorf("EventType = %s, want submit", err.EventType)
	}
	if err.Panic != "panic!" {
		t.Errorf("Panic = %v, want panic!", err.Panic)
	}
	if string(err.Stack) != string(stack) {
		t.Error("Stack should be preserved")
	}
}

func TestProtocolError(t *testing.T) {
	err := &ProtocolError{
		SessionID: "sess-3",
		Op:        "decode",
		Message:   "invalid frame type",
	}

	expected := "server: protocol error in session sess-3: decode: invalid frame type"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestNewProtocolError(t *testing.T) {
	err := NewProtocolError("sess-4", "encode", "buffer overflow")

	if err.SessionID != "sess-4" {
		t.Errorf("SessionID = %s, want sess-4", err.SessionID)
	}
	if err.Op != "encode" {
		t.Errorf("Op = %s, want encode", err.Op)
	}
	if err.Message != "buffer overflow" {
		t.Errorf("Message = %s, want buffer overflow", err.Message)
	}
}

func TestErrorWrapping(t *testing.T) {
	// Test that errors can be properly wrapped and unwrapped
	baseErr := errors.New("base error")
	sessionErr := &SessionError{
		SessionID: "s1",
		Op:        "read",
		Err:       baseErr,
	}

	// errors.Is should work
	if !errors.Is(sessionErr, baseErr) {
		t.Error("errors.Is should find the base error")
	}

	// errors.As should work
	var se *SessionError
	if !errors.As(sessionErr, &se) {
		t.Error("errors.As should work with SessionError")
	}
	if se.SessionID != "s1" {
		t.Error("errors.As should preserve fields")
	}
}

func TestErrorsAreDistinct(t *testing.T) {
	// Verify all sentinel errors are distinct
	sentinels := []error{
		ErrSessionClosed,
		ErrSessionNotFound,
		ErrHandlerNotFound,
		ErrEventQueueFull,
		ErrMaxSessionsReached,
		ErrInvalidHandshake,
		ErrInvalidCSRF,
		ErrSecureCookiesRequired,
		ErrSessionExpired,
		ErrConnectionClosed,
		ErrWriteTimeout,
		ErrReadTimeout,
	}

	for i, err1 := range sentinels {
		for j, err2 := range sentinels {
			if i != j && errors.Is(err1, err2) {
				t.Errorf("errors %d and %d should not be equal", i, j)
			}
		}
	}
}
