package vango

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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

// =============================================================================
// Request Accessor Tests
// =============================================================================

func TestSSRContext_Method_ReturnsHTTPMethod(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		method string
	}{
		{http.MethodGet},
		{http.MethodPost},
		{http.MethodPut},
		{http.MethodDelete},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "http://example.com/test", nil)
			ctx, _ := newTestSSRContext(t, cfg, req)

			if got := ctx.Method(); got != tt.method {
				t.Errorf("Method() = %q, want %q", got, tt.method)
			}
		})
	}
}

func TestSSRContext_Query_ReturnsQueryParams(t *testing.T) {
	cfg := DefaultConfig()
	req := httptest.NewRequest(http.MethodGet, "http://example.com/test?foo=bar&baz=qux&multi=a&multi=b", nil)
	ctx, _ := newTestSSRContext(t, cfg, req)

	query := ctx.Query()

	if got := query.Get("foo"); got != "bar" {
		t.Errorf("Query().Get(\"foo\") = %q, want %q", got, "bar")
	}
	if got := query.Get("baz"); got != "qux" {
		t.Errorf("Query().Get(\"baz\") = %q, want %q", got, "qux")
	}
	if got := query["multi"]; len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("Query()[\"multi\"] = %v, want [\"a\", \"b\"]", got)
	}
}

func TestSSRContext_QueryParam_ReturnsSingleParam(t *testing.T) {
	cfg := DefaultConfig()
	req := httptest.NewRequest(http.MethodGet, "http://example.com/test?search=hello&page=5", nil)
	ctx, _ := newTestSSRContext(t, cfg, req)

	if got := ctx.QueryParam("search"); got != "hello" {
		t.Errorf("QueryParam(\"search\") = %q, want %q", got, "hello")
	}
	if got := ctx.QueryParam("page"); got != "5" {
		t.Errorf("QueryParam(\"page\") = %q, want %q", got, "5")
	}
	if got := ctx.QueryParam("missing"); got != "" {
		t.Errorf("QueryParam(\"missing\") = %q, want empty", got)
	}
}

func TestSSRContext_Header_ReturnsHeaderValue(t *testing.T) {
	cfg := DefaultConfig()
	req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
	req.Header.Set("X-Custom-Header", "custom-value")
	req.Header.Set("Authorization", "Bearer token123")
	ctx, _ := newTestSSRContext(t, cfg, req)

	if got := ctx.Header("X-Custom-Header"); got != "custom-value" {
		t.Errorf("Header(\"X-Custom-Header\") = %q, want %q", got, "custom-value")
	}
	if got := ctx.Header("Authorization"); got != "Bearer token123" {
		t.Errorf("Header(\"Authorization\") = %q, want %q", got, "Bearer token123")
	}
	if got := ctx.Header("Missing-Header"); got != "" {
		t.Errorf("Header(\"Missing-Header\") = %q, want empty", got)
	}
}

func TestSSRContext_Cookie_ReturnsRequestCookie(t *testing.T) {
	cfg := DefaultConfig()
	req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "abc123"})
	req.AddCookie(&http.Cookie{Name: "prefs", Value: "dark"})
	ctx, _ := newTestSSRContext(t, cfg, req)

	cookie, err := ctx.Cookie("session")
	if err != nil {
		t.Fatalf("Cookie(\"session\") error: %v", err)
	}
	if cookie.Value != "abc123" {
		t.Errorf("Cookie(\"session\").Value = %q, want %q", cookie.Value, "abc123")
	}

	cookie, err = ctx.Cookie("prefs")
	if err != nil {
		t.Fatalf("Cookie(\"prefs\") error: %v", err)
	}
	if cookie.Value != "dark" {
		t.Errorf("Cookie(\"prefs\").Value = %q, want %q", cookie.Value, "dark")
	}
}

func TestSSRContext_Cookie_ReturnsErrorForMissingCookie(t *testing.T) {
	cfg := DefaultConfig()
	ctx, _ := newTestSSRContext(t, cfg, nil)

	_, err := ctx.Cookie("nonexistent")
	if err == nil {
		t.Error("expected error for missing cookie")
	}
}

// =============================================================================
// Auth/Session Tests (SSR defaults)
// =============================================================================

func TestSSRContext_Session_ReturnsNilForSSR(t *testing.T) {
	cfg := DefaultConfig()
	ctx, _ := newTestSSRContext(t, cfg, nil)

	if got := ctx.Session(); got != nil {
		t.Errorf("Session() = %v, want nil for SSR", got)
	}
}

func TestSSRContext_AuthSession_ReturnsNilForSSR(t *testing.T) {
	cfg := DefaultConfig()
	ctx, _ := newTestSSRContext(t, cfg, nil)

	if got := ctx.AuthSession(); got != nil {
		t.Errorf("AuthSession() = %v, want nil for SSR", got)
	}
}

func TestSSRContext_User_ReturnsUserFromContext(t *testing.T) {
	cfg := DefaultConfig()
	// Create request with user in context
	baseReq := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
	reqWithUser := baseReq.WithContext(WithUser(baseReq.Context(), "alice"))
	ctx, _ := newTestSSRContext(t, cfg, reqWithUser)

	if got := ctx.User(); got != "alice" {
		t.Errorf("User() = %v, want %q", got, "alice")
	}
}

func TestSSRContext_SetUser_UpdatesUser(t *testing.T) {
	cfg := DefaultConfig()
	ctx, _ := newTestSSRContext(t, cfg, nil)

	// Initially nil
	if got := ctx.User(); got != nil {
		t.Errorf("User() before SetUser = %v, want nil", got)
	}

	ctx.SetUser("bob")

	if got := ctx.User(); got != "bob" {
		t.Errorf("User() after SetUser = %v, want %q", got, "bob")
	}
}

func TestSSRContext_Principal_ReturnsFalseForSSR(t *testing.T) {
	cfg := DefaultConfig()
	ctx, _ := newTestSSRContext(t, cfg, nil)

	_, ok := ctx.Principal()
	if ok {
		t.Error("Principal() should return false for SSR")
	}
}

func TestSSRContext_MustPrincipal_PanicsForSSR(t *testing.T) {
	cfg := DefaultConfig()
	ctx, _ := newTestSSRContext(t, cfg, nil)

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustPrincipal() should panic for SSR")
		}
	}()

	ctx.MustPrincipal()
}

func TestSSRContext_RevalidateAuth_NoOpForSSR(t *testing.T) {
	cfg := DefaultConfig()
	ctx, _ := newTestSSRContext(t, cfg, nil)

	// Should not error or panic
	err := ctx.RevalidateAuth()
	if err != nil {
		t.Errorf("RevalidateAuth() error = %v, want nil", err)
	}
}

// =============================================================================
// Context Methods Tests
// =============================================================================

func TestSSRContext_Logger_ReturnsConfiguredLogger(t *testing.T) {
	cfg := DefaultConfig()
	customLogger := slog.New(slog.NewTextHandler(nil, nil))
	cfg.Logger = customLogger

	app := New(cfg)
	req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
	rr := httptest.NewRecorder()
	ctx := newSSRContext(rr, req, nil, cfg, customLogger, app.server.CookiePolicy())

	if got := ctx.Logger(); got != customLogger {
		t.Error("Logger() should return the configured logger")
	}
}

func TestSSRContext_Logger_DefaultsToSlogDefault(t *testing.T) {
	cfg := DefaultConfig()
	app := New(cfg)
	req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
	rr := httptest.NewRecorder()
	ctx := newSSRContext(rr, req, nil, cfg, nil, app.server.CookiePolicy())

	// When logger is nil, should return slog.Default()
	if got := ctx.Logger(); got == nil {
		t.Error("Logger() should not return nil")
	}
}

func TestSSRContext_Done_ReturnsRequestContextDone(t *testing.T) {
	cfg := DefaultConfig()
	baseCtx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
	req = req.WithContext(baseCtx)
	ctx, _ := newTestSSRContext(t, cfg, req)

	done := ctx.Done()

	// Before cancel, should not be closed
	select {
	case <-done:
		t.Error("Done() channel should not be closed before cancel")
	default:
		// Expected
	}

	cancel()

	// After cancel, should be closed
	select {
	case <-done:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("Done() channel should be closed after cancel")
	}
}

func TestSSRContext_SetValue_Value_RoundTrip(t *testing.T) {
	cfg := DefaultConfig()
	ctx, _ := newTestSSRContext(t, cfg, nil)

	type ctxKey string
	key1 := ctxKey("key1")
	key2 := ctxKey("key2")

	// Set values
	ctx.SetValue(key1, "value1")
	ctx.SetValue(key2, 42)

	// Get values
	if got := ctx.Value(key1); got != "value1" {
		t.Errorf("Value(key1) = %v, want %q", got, "value1")
	}
	if got := ctx.Value(key2); got != 42 {
		t.Errorf("Value(key2) = %v, want %d", got, 42)
	}
	if got := ctx.Value(ctxKey("missing")); got != nil {
		t.Errorf("Value(missing) = %v, want nil", got)
	}
}

func TestSSRContext_Emit_NoOpForSSR(t *testing.T) {
	cfg := DefaultConfig()
	ctx, _ := newTestSSRContext(t, cfg, nil)

	// Should not panic
	ctx.Emit("test-event", map[string]string{"foo": "bar"})
}

func TestSSRContext_Dispatch_ExecutesImmediately(t *testing.T) {
	cfg := DefaultConfig()
	ctx, _ := newTestSSRContext(t, cfg, nil)

	executed := false
	ctx.Dispatch(func() {
		executed = true
	})

	if !executed {
		t.Error("Dispatch() should execute function immediately for SSR")
	}
}

// =============================================================================
// Event/Perf Tests (SSR defaults)
// =============================================================================

func TestSSRContext_Event_ReturnsNil(t *testing.T) {
	cfg := DefaultConfig()
	ctx, _ := newTestSSRContext(t, cfg, nil)

	if got := ctx.Event(); got != nil {
		t.Errorf("Event() = %v, want nil for SSR", got)
	}
}

func TestSSRContext_PatchCount_ReturnsZero(t *testing.T) {
	cfg := DefaultConfig()
	ctx, _ := newTestSSRContext(t, cfg, nil)

	if got := ctx.PatchCount(); got != 0 {
		t.Errorf("PatchCount() = %d, want 0 for SSR", got)
	}
}

func TestSSRContext_AddPatchCount_NoOp(t *testing.T) {
	cfg := DefaultConfig()
	ctx, _ := newTestSSRContext(t, cfg, nil)

	// Should not panic and count should remain 0
	ctx.AddPatchCount(10)
	if got := ctx.PatchCount(); got != 0 {
		t.Errorf("PatchCount() after AddPatchCount = %d, want 0 for SSR", got)
	}
}

func TestSSRContext_StormBudget_ReturnsNil(t *testing.T) {
	cfg := DefaultConfig()
	ctx, _ := newTestSSRContext(t, cfg, nil)

	if got := ctx.StormBudget(); got != nil {
		t.Errorf("StormBudget() = %v, want nil for SSR", got)
	}
}

func TestSSRContext_Mode_ReturnsZero(t *testing.T) {
	cfg := DefaultConfig()
	ctx, _ := newTestSSRContext(t, cfg, nil)

	if got := ctx.Mode(); got != 0 {
		t.Errorf("Mode() = %d, want 0 for SSR", got)
	}
}

func TestSSRContext_StdContext_ReturnsRequestContext(t *testing.T) {
	cfg := DefaultConfig()
	type ctxKey string
	key := ctxKey("test")
	baseCtx := context.WithValue(context.Background(), key, "value")
	req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
	req = req.WithContext(baseCtx)
	ctx, _ := newTestSSRContext(t, cfg, req)

	stdCtx := ctx.StdContext()

	if got := stdCtx.Value(key); got != "value" {
		t.Errorf("StdContext().Value(key) = %v, want %q", got, "value")
	}
}

func TestSSRContext_WithStdContext_CreatesNewContext(t *testing.T) {
	cfg := DefaultConfig()
	ctx, _ := newTestSSRContext(t, cfg, nil)

	type ctxKey string
	key := ctxKey("new-key")
	newStdCtx := context.WithValue(context.Background(), key, "new-value")

	newCtx := ctx.WithStdContext(newStdCtx)

	// Original context should not have the new value
	if got := ctx.StdContext().Value(key); got != nil {
		t.Errorf("original StdContext().Value(key) = %v, want nil", got)
	}

	// New context should have the new value
	if got := newCtx.StdContext().Value(key); got != "new-value" {
		t.Errorf("new StdContext().Value(key) = %v, want %q", got, "new-value")
	}
}

// =============================================================================
// Response Control Tests
// =============================================================================

func TestSSRContext_Status_SetsStatusCode(t *testing.T) {
	cfg := DefaultConfig()
	ctx, _ := newTestSSRContext(t, cfg, nil)

	ctx.Status(http.StatusCreated)

	// Check internal status field
	if ctx.status != http.StatusCreated {
		t.Errorf("status = %d, want %d", ctx.status, http.StatusCreated)
	}
}

func TestSSRContext_SetHeader_SetsResponseHeader(t *testing.T) {
	cfg := DefaultConfig()
	ctx, rr := newTestSSRContext(t, cfg, nil)

	ctx.SetHeader("X-Custom", "value")
	ctx.applyTo(rr)

	if got := rr.Header().Get("X-Custom"); got != "value" {
		t.Errorf("header X-Custom = %q, want %q", got, "value")
	}
}

func TestSSRContext_SetCookie_SetsCookieViaPolicy(t *testing.T) {
	cfg := DefaultConfig()
	req := httptest.NewRequest(http.MethodGet, "https://example.com/test", nil)
	req.TLS = &tls.ConnectionState{} // HTTPS required for Secure cookies
	ctx, rr := newTestSSRContext(t, cfg, req)

	ctx.SetCookie(&http.Cookie{
		Name:  "test",
		Value: "value",
		Path:  "/",
	})
	ctx.applyTo(rr)

	cookies := rr.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "test" {
			found = true
			if c.Value != "value" {
				t.Errorf("cookie value = %q, want %q", c.Value, "value")
			}
			break
		}
	}
	if !found {
		t.Error("expected cookie 'test' to be set")
	}
}

func TestSSRContext_Request_ReturnsOriginalRequest(t *testing.T) {
	cfg := DefaultConfig()
	req := httptest.NewRequest(http.MethodPost, "http://example.com/test", nil)
	ctx, _ := newTestSSRContext(t, cfg, req)

	got := ctx.Request()

	if got.Method != http.MethodPost {
		t.Errorf("Request().Method = %q, want %q", got.Method, http.MethodPost)
	}
	if got.URL.Path != "/test" {
		t.Errorf("Request().URL.Path = %q, want %q", got.URL.Path, "/test")
	}
}

func TestSSRContext_Path_ReturnsURLPath(t *testing.T) {
	cfg := DefaultConfig()
	req := httptest.NewRequest(http.MethodGet, "http://example.com/users/123", nil)
	ctx, _ := newTestSSRContext(t, cfg, req)

	if got := ctx.Path(); got != "/users/123" {
		t.Errorf("Path() = %q, want %q", got, "/users/123")
	}
}

func TestSSRContext_Param_ReturnsRouteParam(t *testing.T) {
	cfg := DefaultConfig()
	app := New(cfg)
	req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
	rr := httptest.NewRecorder()
	params := map[string]string{"id": "123", "slug": "hello-world"}
	ctx := newSSRContext(rr, req, params, cfg, slog.Default(), app.server.CookiePolicy())

	if got := ctx.Param("id"); got != "123" {
		t.Errorf("Param(\"id\") = %q, want %q", got, "123")
	}
	if got := ctx.Param("slug"); got != "hello-world" {
		t.Errorf("Param(\"slug\") = %q, want %q", got, "hello-world")
	}
	if got := ctx.Param("missing"); got != "" {
		t.Errorf("Param(\"missing\") = %q, want empty", got)
	}
}
