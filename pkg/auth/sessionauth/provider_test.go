package sessionauth

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

type testCookiePolicy struct{}

func (p testCookiePolicy) ApplyCookiePolicy(r *http.Request, cookie *http.Cookie) (*http.Cookie, error) {
	cookie.Domain = "example.com"
	cookie.SameSite = http.SameSiteStrictMode
	return cookie, nil
}

func TestProviderClearCookieAppliesPolicy(t *testing.T) {
	provider := New(nil, WithCookiePolicy(testCookiePolicy{}))
	req := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	req.TLS = &tls.ConnectionState{}
	rec := httptest.NewRecorder()

	provider.clearCookie(rec, req)

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Domain != "example.com" {
		t.Fatalf("cookie Domain = %q, want %q", cookie.Domain, "example.com")
	}
	if cookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("cookie SameSite = %v, want %v", cookie.SameSite, http.SameSiteStrictMode)
	}
	if !cookie.Secure {
		t.Fatalf("cookie Secure = false, want true")
	}
}
