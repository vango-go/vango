package server

import (
	"net"
	"net/http/httptest"
	"testing"
)

func TestClientIPFromRequest_UntrustedProxyIgnoresForwarded(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com", nil)
	req.RemoteAddr = "198.51.100.10:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.5")

	trusted := newProxyMatcher([]string{"203.0.113.1"}, nil)
	got := clientIPFromRequest(req, trusted)
	want := net.ParseIP("198.51.100.10")

	if got == nil || !got.Equal(want) {
		t.Fatalf("clientIP=%v, want %v", got, want)
	}
}

func TestClientIPFromRequest_TrustedProxyRightMostUntrusted(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	req.Header.Set("X-Forwarded-For", "198.51.100.1, 203.0.113.11, 192.0.2.20")

	trusted := newProxyMatcher([]string{"203.0.113.10", "203.0.113.11"}, nil)
	got := clientIPFromRequest(req, trusted)
	want := net.ParseIP("192.0.2.20")

	if got == nil || !got.Equal(want) {
		t.Fatalf("clientIP=%v, want %v", got, want)
	}
}

func TestClientIPFromRequest_AllTrustedUsesLeftmost(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	req.Header.Set("Forwarded", `for=192.0.2.1, for=192.0.2.2`)

	trusted := newProxyMatcher([]string{"203.0.113.10", "192.0.2.1", "192.0.2.2"}, nil)
	got := clientIPFromRequest(req, trusted)
	want := net.ParseIP("192.0.2.1")

	if got == nil || !got.Equal(want) {
		t.Fatalf("clientIP=%v, want %v", got, want)
	}
}
