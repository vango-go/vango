package session

import "testing"

func TestSessionNotFoundError_Error(t *testing.T) {
	err := SessionNotFoundError{SessionID: "abc"}
	if got := err.Error(); got != "session not found: abc" {
		t.Fatalf("Error() got %q", got)
	}
}

func TestErrStoreClosed_Error(t *testing.T) {
	err := ErrStoreClosed{}
	if got := err.Error(); got != "session store is closed" {
		t.Fatalf("Error() got %q", got)
	}
}

