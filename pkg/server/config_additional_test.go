package server

import (
	"net/http"
	"testing"
	"time"

	"github.com/vango-go/vango/pkg/session"
)

func TestServerConfig_Phase12ChainingAndValidation(t *testing.T) {
	store := session.NewMemoryStore()

	cfg := DefaultServerConfig().
		WithSessionStore(store).
		WithResumeWindow(2*time.Minute).
		WithMaxDetachedSessions(10).
		WithMaxSessionsPerIP(5).
		WithPersistInterval(123*time.Millisecond).
		WithReconnectConfig(&ReconnectConfig{ToastOnReconnect: false}).
		EnableToastOnReconnect("welcome back")

	if cfg.SessionStore != store {
		t.Fatal("WithSessionStore did not set SessionStore")
	}
	if cfg.ResumeWindow != 2*time.Minute {
		t.Fatalf("ResumeWindow=%v, want %v", cfg.ResumeWindow, 2*time.Minute)
	}
	if cfg.MaxDetachedSessions != 10 || cfg.MaxSessionsPerIP != 5 {
		t.Fatalf("limits mismatch: detached=%d perIP=%d", cfg.MaxDetachedSessions, cfg.MaxSessionsPerIP)
	}
	if cfg.PersistInterval != 123*time.Millisecond {
		t.Fatalf("PersistInterval=%v, want %v", cfg.PersistInterval, 123*time.Millisecond)
	}
	if cfg.ReconnectConfig == nil || !cfg.ReconnectConfig.ToastOnReconnect || cfg.ReconnectConfig.ToastMessage != "welcome back" {
		t.Fatalf("ReconnectConfig=%+v, want toast enabled with message", cfg.ReconnectConfig)
	}

	if err := cfg.ValidateConfig(); err != nil {
		t.Fatalf("ValidateConfig() error: %v", err)
	}
}

func TestServerConfig_IsSecureRequiresAllSecuritySettings(t *testing.T) {
	cfg := DefaultServerConfig()
	if cfg.IsSecure() {
		t.Fatal("IsSecure()=true by default, want false")
	}

	cfg.DevMode = false
	cfg.CSRFSecret = []byte("0123456789abcdef0123456789abcdef")
	cfg.SecureCookies = true
	cfg.CheckOrigin = func(r *http.Request) bool { return true }
	cfg.MaxSessionsPerIP = 1
	cfg.MaxDetachedSessions = 1

	if !cfg.IsSecure() {
		t.Fatal("IsSecure()=false, want true when all security settings present")
	}
}
