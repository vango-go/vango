package server

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/vango-go/vango/pkg/auth"
)

func TestAuthExpiryUnixMs(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  int64
		ok    bool
	}{
		{name: "int64", value: int64(123), want: 123, ok: true},
		{name: "int", value: 456, want: 456, ok: true},
		{name: "float64", value: float64(789), want: 789, ok: true},
		{name: "jsonNumber", value: json.Number("321"), want: 321, ok: true},
		{name: "invalid", value: "nope", want: 0, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := authExpiryUnixMs(tt.value)
			if ok != tt.ok {
				t.Fatalf("ok=%v, want %v", ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Fatalf("got=%d, want %d", got, tt.want)
			}
		})
	}
}

func TestSession_isAuthExpired(t *testing.T) {
	s := NewMockSession()
	if s.isAuthExpired() {
		t.Fatal("expected not expired without expiry key")
	}

	s.Set(auth.SessionKeyExpiryUnixMs, time.Now().Add(-time.Minute).UnixMilli())
	if !s.isAuthExpired() {
		t.Fatal("expected expired when expiry is in the past")
	}

	s.Set(auth.SessionKeyExpiryUnixMs, time.Now().Add(time.Minute).UnixMilli())
	if s.isAuthExpired() {
		t.Fatal("expected not expired when expiry is in the future")
	}
}

func TestSession_runActiveAuthCheck_SuccessUpdatesLastOK(t *testing.T) {
	s := NewMockSession()
	s.config.AuthCheck = &AuthCheckConfig{
		Timeout: 250 * time.Millisecond,
		Check: func(ctx context.Context, p auth.Principal) error {
			return nil
		},
	}
	s.Set(auth.SessionKeyPrincipal, auth.Principal{ID: "user-1"})

	s.runActiveAuthCheck()

	select {
	case fn := <-s.dispatchCh:
		fn()
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for auth check dispatch")
	}

	if s.authLastOK.IsZero() {
		t.Fatal("expected authLastOK to be set after successful check")
	}
	if s.authCheckInFlight.Load() {
		t.Fatal("expected authCheckInFlight to be cleared")
	}
}

func TestSession_runActiveAuthCheck_FailOpenKeepsLastOK(t *testing.T) {
	s := NewMockSession()
	s.config.AuthCheck = &AuthCheckConfig{
		Timeout:     250 * time.Millisecond,
		FailureMode: FailOpenWithGrace,
		MaxStale:    10 * time.Minute,
		Check: func(ctx context.Context, p auth.Principal) error {
			return errors.New("transient failure")
		},
	}
	s.Set(auth.SessionKeyPrincipal, auth.Principal{ID: "user-1"})

	initial := time.Now().Add(-time.Minute)
	s.authLastOK = initial

	s.runActiveAuthCheck()

	select {
	case fn := <-s.dispatchCh:
		fn()
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for auth check dispatch")
	}

	if s.authLastOK != initial {
		t.Fatalf("authLastOK changed on fail-open: got %v want %v", s.authLastOK, initial)
	}
	if s.authCheckInFlight.Load() {
		t.Fatal("expected authCheckInFlight to be cleared")
	}
}
