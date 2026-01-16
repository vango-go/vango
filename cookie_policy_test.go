package vango

import (
	"crypto/tls"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vango-go/vango/pkg/server"
)

func TestSSRContextSetCookieStrictRejectsInsecureRequest(t *testing.T) {
	cfg := DefaultConfig()
	app := New(cfg)

	req := httptest.NewRequest("GET", "http://example.com/", nil)
	rr := httptest.NewRecorder()
	ctx := newSSRContext(rr, req, nil, cfg, slog.Default(), app.server.CookiePolicy())

	err := ctx.SetCookieStrict(&http.Cookie{
		Name:  "session",
		Value: "abc123",
		Path:  "/",
	})
	if !errors.Is(err, server.ErrSecureCookiesRequired) {
		t.Fatalf("SetCookieStrict error = %v, want %v", err, server.ErrSecureCookiesRequired)
	}
	if len(ctx.cookies) != 0 {
		t.Fatalf("cookies = %d, want 0 when request is insecure", len(ctx.cookies))
	}
}

func TestSSRContextSetCookieStrictTrustsForwardedProto(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Security.TrustedProxies = []string{"203.0.113.10"}
	app := New(cfg)

	req := httptest.NewRequest("GET", "http://example.com/", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	req.Header.Set("X-Forwarded-Proto", "https")
	rr := httptest.NewRecorder()
	ctx := newSSRContext(rr, req, nil, cfg, slog.Default(), app.server.CookiePolicy())

	if err := ctx.SetCookieStrict(&http.Cookie{
		Name:  "session",
		Value: "abc123",
		Path:  "/",
	}); err != nil {
		t.Fatalf("SetCookieStrict error: %v", err)
	}

	ctx.applyTo(rr)
	cookies := rr.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
	if !cookies[0].Secure {
		t.Fatalf("cookie Secure = false, want true")
	}
}

func TestAppAPISetCookieAppliesDefaults(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Security.CookieDomain = ".example.com"
	app := New(cfg)

	type params struct{}
	app.API(http.MethodGet, "/cookie", func(ctx Ctx, params params, body []byte) (any, error) {
		ctx.SetCookie(&http.Cookie{
			Name:  "session",
			Value: "token",
			Path:  "/",
		})
		return map[string]string{"ok": "1"}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "https://example.com/cookie", nil)
	req.TLS = &tls.ConnectionState{}
	rr := httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	cookies := rr.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
	cookie := cookies[0]
	if !cookie.Secure {
		t.Fatalf("cookie Secure = false, want true")
	}
	if !cookie.HttpOnly {
		t.Fatalf("cookie HttpOnly = false, want true")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie SameSite = %v, want %v", cookie.SameSite, http.SameSiteLaxMode)
	}
	if cookie.Domain != "example.com" {
		t.Fatalf("cookie Domain = %q, want %q", cookie.Domain, "example.com")
	}
}

func TestAppAPISetCookieHTTPOnlyOverride(t *testing.T) {
	cfg := DefaultConfig()
	app := New(cfg)

	type params struct{}
	app.API(http.MethodGet, "/cookie", func(ctx Ctx, params params, body []byte) (any, error) {
		if err := ctx.SetCookieStrict(&http.Cookie{
			Name:  "session",
			Value: "token",
			Path:  "/",
		}, WithCookieHTTPOnly(false)); err != nil {
			return nil, err
		}
		return map[string]string{"ok": "1"}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "https://example.com/cookie", nil)
	req.TLS = &tls.ConnectionState{}
	rr := httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	cookies := rr.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
	if cookies[0].HttpOnly {
		t.Fatalf("cookie HttpOnly = true, want false")
	}
}
