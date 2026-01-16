package vango

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuildServerConfig_AllowedOrigins(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Security.AllowedOrigins = []string{"https://allowed.example.com"}
	cfg.Security.AllowSameOrigin = false

	serverCfg := buildServerConfig(cfg)
	if serverCfg.CheckOrigin == nil {
		t.Fatal("expected CheckOrigin to be configured")
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.Header.Set("Origin", "https://allowed.example.com")
	if !serverCfg.CheckOrigin(req) {
		t.Fatal("expected allowed origin to pass")
	}

	req.Header.Set("Origin", "https://other.example.com")
	if serverCfg.CheckOrigin(req) {
		t.Fatal("expected non-allowed origin to fail")
	}
}

func TestBuildServerConfig_AllowSameOrigin(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Security.AllowedOrigins = nil
	cfg.Security.AllowSameOrigin = true

	serverCfg := buildServerConfig(cfg)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.Host = "example.com"
	req.Header.Set("Origin", "https://example.com")
	if !serverCfg.CheckOrigin(req) {
		t.Fatal("expected same-origin request to pass")
	}

	req.Header.Set("Origin", "https://evil.example.com")
	if serverCfg.CheckOrigin(req) {
		t.Fatal("expected cross-origin request to fail")
	}
}

func TestBuildServerConfig_TrustedProxiesCopied(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Security.TrustedProxies = []string{"203.0.113.10"}

	serverCfg := buildServerConfig(cfg)
	cfg.Security.TrustedProxies[0] = "198.51.100.2"

	if len(serverCfg.TrustedProxies) != 1 {
		t.Fatalf("TrustedProxies len = %d, want 1", len(serverCfg.TrustedProxies))
	}
	if serverCfg.TrustedProxies[0] != "203.0.113.10" {
		t.Fatalf("TrustedProxies[0] = %q, want %q", serverCfg.TrustedProxies[0], "203.0.113.10")
	}
}

func TestBuildServerConfig_CookieSettings(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Security.CookieSameSite = http.SameSiteStrictMode
	cfg.Security.CookieDomain = ".example.com"

	serverCfg := buildServerConfig(cfg)
	if serverCfg.SameSiteMode != http.SameSiteStrictMode {
		t.Fatalf("SameSiteMode = %v, want %v", serverCfg.SameSiteMode, http.SameSiteStrictMode)
	}
	if serverCfg.CookieDomain != ".example.com" {
		t.Fatalf("CookieDomain = %q, want %q", serverCfg.CookieDomain, ".example.com")
	}
}

func TestBuildServerConfig_AuthCheckNormalizationDoesNotMutateInput(t *testing.T) {
	checkCfg := &AuthCheckConfig{
		Interval:    0,
		Timeout:     0,
		MaxStale:    0,
		FailureMode: AuthFailureMode(99),
	}
	cfg := DefaultConfig()
	cfg.Session.AuthCheck = checkCfg

	serverCfg := buildServerConfig(cfg)
	if serverCfg.SessionConfig.AuthCheck == nil {
		t.Fatal("expected AuthCheck to be configured")
	}

	if serverCfg.SessionConfig.AuthCheck.Timeout <= 0 {
		t.Fatalf("Timeout = %v, want > 0", serverCfg.SessionConfig.AuthCheck.Timeout)
	}
	if serverCfg.SessionConfig.AuthCheck.MaxStale <= 0 {
		t.Fatalf("MaxStale = %v, want > 0", serverCfg.SessionConfig.AuthCheck.MaxStale)
	}
	if serverCfg.SessionConfig.AuthCheck.FailureMode != FailOpenWithGrace {
		t.Fatalf("FailureMode = %v, want %v", serverCfg.SessionConfig.AuthCheck.FailureMode, FailOpenWithGrace)
	}

	if checkCfg.Timeout != 0 || checkCfg.MaxStale != 0 {
		t.Fatalf("input AuthCheck mutated: timeout=%v maxStale=%v", checkCfg.Timeout, checkCfg.MaxStale)
	}
	if checkCfg.FailureMode != AuthFailureMode(99) {
		t.Fatalf("input FailureMode mutated: %v", checkCfg.FailureMode)
	}
	if checkCfg.Interval != 0 {
		t.Fatalf("input Interval mutated: %v", checkCfg.Interval)
	}
}
