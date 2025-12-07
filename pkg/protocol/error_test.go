package protocol

import (
	"testing"
)

func TestErrorMessageEncodeDecode(t *testing.T) {
	tests := []struct {
		name string
		em   *ErrorMessage
	}{
		{
			name: "simple_error",
			em: &ErrorMessage{
				Code:    ErrHandlerNotFound,
				Message: "No handler for h42",
				Fatal:   false,
			},
		},
		{
			name: "fatal_error",
			em: &ErrorMessage{
				Code:    ErrSessionExpired,
				Message: "Session has expired",
				Fatal:   true,
			},
		},
		{
			name: "empty_message",
			em: &ErrorMessage{
				Code:    ErrUnknown,
				Message: "",
				Fatal:   false,
			},
		},
		{
			name: "server_error",
			em: &ErrorMessage{
				Code:    ErrServerError,
				Message: "Internal server error: database connection failed",
				Fatal:   true,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			encoded := EncodeErrorMessage(tc.em)
			decoded, err := DecodeErrorMessage(encoded)
			if err != nil {
				t.Fatalf("DecodeErrorMessage() error = %v", err)
			}

			if decoded.Code != tc.em.Code {
				t.Errorf("Code = %v, want %v", decoded.Code, tc.em.Code)
			}
			if decoded.Message != tc.em.Message {
				t.Errorf("Message = %q, want %q", decoded.Message, tc.em.Message)
			}
			if decoded.Fatal != tc.em.Fatal {
				t.Errorf("Fatal = %v, want %v", decoded.Fatal, tc.em.Fatal)
			}
		})
	}
}

func TestErrorCodeString(t *testing.T) {
	tests := []struct {
		code ErrorCode
		want string
	}{
		{ErrUnknown, "Unknown"},
		{ErrInvalidFrame, "InvalidFrame"},
		{ErrInvalidEvent, "InvalidEvent"},
		{ErrHandlerNotFound, "HandlerNotFound"},
		{ErrHandlerPanic, "HandlerPanic"},
		{ErrSessionExpired, "SessionExpired"},
		{ErrRateLimited, "RateLimited"},
		{ErrServerError, "ServerError"},
		{ErrNotAuthorized, "NotAuthorized"},
		{ErrNotFound, "NotFound"},
		{ErrValidation, "Validation"},
		{ErrorCode(0xFFFF), "Unknown"},
	}

	for _, tc := range tests {
		if got := tc.code.String(); got != tc.want {
			t.Errorf("ErrorCode(%d).String() = %q, want %q", tc.code, got, tc.want)
		}
	}
}

func TestNewError(t *testing.T) {
	em := NewError(ErrHandlerNotFound, "Handler not found")

	if em.Code != ErrHandlerNotFound {
		t.Errorf("Code = %v, want HandlerNotFound", em.Code)
	}
	if em.Message != "Handler not found" {
		t.Errorf("Message = %q, want %q", em.Message, "Handler not found")
	}
	if em.Fatal {
		t.Error("Fatal = true, want false")
	}
}

func TestNewFatalError(t *testing.T) {
	em := NewFatalError(ErrSessionExpired, "Session expired")

	if em.Code != ErrSessionExpired {
		t.Errorf("Code = %v, want SessionExpired", em.Code)
	}
	if em.Message != "Session expired" {
		t.Errorf("Message = %q, want %q", em.Message, "Session expired")
	}
	if !em.Fatal {
		t.Error("Fatal = false, want true")
	}
}

func TestErrorMessageError(t *testing.T) {
	// Non-fatal error
	em := NewError(ErrHandlerNotFound, "No handler")
	got := em.Error()
	want := "HandlerNotFound: No handler"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}

	// Fatal error
	em = NewFatalError(ErrSessionExpired, "Expired")
	got = em.Error()
	want = "fatal: SessionExpired: Expired"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestErrorMessageIsFatal(t *testing.T) {
	em := NewError(ErrUnknown, "test")
	if em.IsFatal() {
		t.Error("IsFatal() = true, want false")
	}

	em = NewFatalError(ErrUnknown, "test")
	if !em.IsFatal() {
		t.Error("IsFatal() = false, want true")
	}
}

func BenchmarkEncodeErrorMessage(b *testing.B) {
	em := NewError(ErrHandlerNotFound, "No handler for h42")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EncodeErrorMessage(em)
	}
}

func BenchmarkDecodeErrorMessage(b *testing.B) {
	em := NewError(ErrHandlerNotFound, "No handler for h42")
	encoded := EncodeErrorMessage(em)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeErrorMessage(encoded)
	}
}
