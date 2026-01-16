package vango

import (
	"crypto/tls"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestSSRContext(t *testing.T, cfg Config, req *http.Request) (*ssrContext, *httptest.ResponseRecorder) {
	t.Helper()

	if req == nil {
		req = httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
	}
	rr := httptest.NewRecorder()
	app := New(cfg)
	ctx := newSSRContext(rr, req, nil, cfg, slog.Default(), app.server.CookiePolicy())
	return ctx, rr
}

func TestSSRContext_RedirectRejectsAbsoluteURL(t *testing.T) {
	cfg := DefaultConfig()
	ctx, _ := newTestSSRContext(t, cfg, nil)

	ctx.Redirect("https://evil.example.com", http.StatusFound)

	if _, _, ok := ctx.redirectInfo(); ok {
		t.Fatal("expected redirect to be rejected for absolute URL")
	}
}

func TestSSRContext_RedirectExternalAllowlist(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Security.AllowedRedirectHosts = []string{"accounts.example.com"}
	ctx, _ := newTestSSRContext(t, cfg, nil)

	ctx.RedirectExternal("https://accounts.example.com/login", http.StatusFound)

	url, code, ok := ctx.redirectInfo()
	if !ok {
		t.Fatal("expected external redirect to be allowed")
	}
	if code != http.StatusFound {
		t.Fatalf("status = %d, want %d", code, http.StatusFound)
	}
	if url != "https://accounts.example.com/login" {
		t.Fatalf("url = %q, want %q", url, "https://accounts.example.com/login")
	}
}

func TestSSRContext_RedirectExternalRejectsDisallowedHost(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Security.AllowedRedirectHosts = []string{"accounts.example.com"}
	ctx, _ := newTestSSRContext(t, cfg, nil)

	ctx.RedirectExternal("https://evil.example.com/login", http.StatusFound)

	if _, _, ok := ctx.redirectInfo(); ok {
		t.Fatal("expected external redirect to be rejected")
	}
}

func TestSSRContext_NavigateSetsRedirect(t *testing.T) {
	cfg := DefaultConfig()
	ctx, _ := newTestSSRContext(t, cfg, nil)

	ctx.Navigate("/dest")

	url, code, ok := ctx.redirectInfo()
	if !ok {
		t.Fatal("expected navigate to set redirect")
	}
	if code != http.StatusFound {
		t.Fatalf("status = %d, want %d", code, http.StatusFound)
	}
	if url != "/dest" {
		t.Fatalf("url = %q, want %q", url, "/dest")
	}
}

func TestSSRContext_NavigateReplaceUsesSeeOther(t *testing.T) {
	cfg := DefaultConfig()
	ctx, _ := newTestSSRContext(t, cfg, nil)

	ctx.Navigate("/dest", WithReplace())

	url, code, ok := ctx.redirectInfo()
	if !ok {
		t.Fatal("expected navigate to set redirect")
	}
	if code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", code, http.StatusSeeOther)
	}
	if url != "/dest" {
		t.Fatalf("url = %q, want %q", url, "/dest")
	}
}

func TestSSRContext_NavigateRejectsAbsoluteURL(t *testing.T) {
	cfg := DefaultConfig()
	ctx, _ := newTestSSRContext(t, cfg, nil)

	ctx.Navigate("https://evil.example.com")

	if _, _, ok := ctx.redirectInfo(); ok {
		t.Fatal("expected navigate to reject absolute URL")
	}
}

func TestSSRContext_ApplyToCopiesHeadersAndCookies(t *testing.T) {
	cfg := DefaultConfig()
	req := httptest.NewRequest(http.MethodGet, "https://example.com/test", nil)
	req.TLS = &tls.ConnectionState{}
	ctx, rr := newTestSSRContext(t, cfg, req)

	ctx.headers.Add("X-Multi", "a")
	ctx.headers.Add("X-Multi", "b")
	ctx.headers.Set("X-Single", "one")

	if err := ctx.SetCookieStrict(&http.Cookie{
		Name:  "session",
		Value: "abc123",
		Path:  "/",
	}); err != nil {
		t.Fatalf("SetCookieStrict error: %v", err)
	}
	if err := ctx.SetCookieStrict(nil); err != nil {
		t.Fatalf("SetCookieStrict(nil) error: %v", err)
	}

	rr.Header().Set("X-Multi", "old")
	ctx.applyTo(rr)

	if got := rr.Header()["X-Multi"]; len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("X-Multi = %#v, want [\"a\" \"b\"]", got)
	}
	if got := rr.Header().Get("X-Single"); got != "one" {
		t.Fatalf("X-Single = %q, want %q", got, "one")
	}

	cookies := rr.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
	if cookies[0].Name != "session" {
		t.Fatalf("cookie name = %q, want %q", cookies[0].Name, "session")
	}
}

func TestSSRContext_AssetPrefixHandling(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Static.Prefix = "/static"
	ctx, _ := newTestSSRContext(t, cfg, nil)

	if got := ctx.Asset("app.js"); got != "/static/app.js" {
		t.Fatalf("Asset = %q, want %q", got, "/static/app.js")
	}

	cfg.Static.Prefix = "/static/"
	ctx, _ = newTestSSRContext(t, cfg, nil)
	if got := ctx.Asset("app.js"); got != "/static/app.js" {
		t.Fatalf("Asset = %q, want %q", got, "/static/app.js")
	}

	cfg.Static.Prefix = ""
	ctx, _ = newTestSSRContext(t, cfg, nil)
	if got := ctx.Asset("app.js"); got != "/app.js" {
		t.Fatalf("Asset = %q, want %q", got, "/app.js")
	}
}
