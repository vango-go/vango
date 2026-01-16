package server

import (
	"crypto/tls"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCookiePolicyApplyDefaults(t *testing.T) {
	cfg := DefaultServerConfig()
	cfg.SameSiteMode = http.SameSiteStrictMode
	cfg.CookieDomain = "example.com"
	cfg.CookieHTTPOnly = true
	policy := newCookiePolicy(cfg, nil, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	req.TLS = &tls.ConnectionState{}
	cookie := &http.Cookie{
		Name:  "session",
		Value: "token",
	}

	updated, err := policy.Apply(req, cookie)
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	if !updated.Secure {
		t.Fatalf("cookie Secure = false, want true")
	}
	if !updated.HttpOnly {
		t.Fatalf("cookie HttpOnly = false, want true")
	}
	if updated.SameSite != http.SameSiteStrictMode {
		t.Fatalf("cookie SameSite = %v, want %v", updated.SameSite, http.SameSiteStrictMode)
	}
	if updated.Domain != "example.com" {
		t.Fatalf("cookie Domain = %q, want %q", updated.Domain, "example.com")
	}
}

func TestCookiePolicyApplySameSiteDefaultMode(t *testing.T) {
	cfg := DefaultServerConfig()
	cfg.SameSiteMode = http.SameSiteStrictMode
	policy := newCookiePolicy(cfg, nil, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	req.TLS = &tls.ConnectionState{}
	cookie := &http.Cookie{
		Name:     "session",
		Value:    "token",
		SameSite: http.SameSiteDefaultMode,
	}

	updated, err := policy.Apply(req, cookie)
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	if updated.SameSite != http.SameSiteStrictMode {
		t.Fatalf("cookie SameSite = %v, want %v", updated.SameSite, http.SameSiteStrictMode)
	}
}

func TestCookiePolicyApplyOverrideHTTPOnly(t *testing.T) {
	cfg := DefaultServerConfig()
	cfg.CookieHTTPOnly = true
	policy := newCookiePolicy(cfg, nil, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	req.TLS = &tls.ConnectionState{}
	cookie := &http.Cookie{
		Name:  "session",
		Value: "token",
	}

	updated, err := policy.Apply(req, cookie, WithCookieHTTPOnly(false))
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	if updated.HttpOnly {
		t.Fatalf("cookie HttpOnly = true, want false")
	}
}

func TestCookiePolicyApplyAllowsInsecureWhenSecureCookiesDisabled(t *testing.T) {
	cfg := DefaultServerConfig()
	cfg.SecureCookies = false
	policy := newCookiePolicy(cfg, nil, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	cookie := &http.Cookie{
		Name:  "session",
		Value: "token",
	}

	updated, err := policy.Apply(req, cookie)
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	if updated.Secure {
		t.Fatalf("cookie Secure = true, want false")
	}
}

func TestCtxSetCookieStrictErrorsWhenWriterMissing(t *testing.T) {
	ctx := NewTestContext(nil)
	if err := ctx.SetCookieStrict(&http.Cookie{Name: "a", Value: "b"}); err == nil {
		t.Fatalf("expected error for missing response writer")
	}
}

func TestCtxSetCookieStrictErrorsWhenPolicyMissing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	req.TLS = &tls.ConnectionState{}
	w := httptest.NewRecorder()
	ctx := newCtx(w, req, slog.Default())
	ctx.cookiePolicy = nil

	if err := ctx.SetCookieStrict(&http.Cookie{Name: "a", Value: "b"}); err == nil {
		t.Fatalf("expected error for missing cookie policy")
	}
}

func TestCtxSetCookieDropsInsecureRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	w := httptest.NewRecorder()
	ctx := newCtx(w, req, slog.Default())

	ctx.SetCookie(&http.Cookie{Name: "a", Value: "b"})

	if len(w.Result().Cookies()) != 0 {
		t.Fatalf("expected no cookies for insecure request")
	}
}

func TestCtxSetCookieStrictAllowsInsecureWhenSecureCookiesDisabled(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	w := httptest.NewRecorder()
	ctx := newCtx(w, req, slog.Default())

	cfg := DefaultServerConfig()
	cfg.SecureCookies = false
	ctx.cookiePolicy = newCookiePolicy(cfg, nil, slog.Default())

	if err := ctx.SetCookieStrict(&http.Cookie{Name: "a", Value: "b"}); err != nil {
		t.Fatalf("SetCookieStrict error: %v", err)
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
	if cookies[0].Secure {
		t.Fatalf("cookie Secure = true, want false")
	}
}
