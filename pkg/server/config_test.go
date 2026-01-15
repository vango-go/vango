package server

import (
	"net/http"
	"testing"
	"time"
)

func TestDefaultSessionConfig(t *testing.T) {
	config := DefaultSessionConfig()

	if config.MaxMessageSize <= 0 {
		t.Error("MaxMessageSize should be positive")
	}
	if config.MaxEventQueue <= 0 {
		t.Error("MaxEventQueue should be positive")
	}
	if config.HandshakeTimeout <= 0 {
		t.Error("HandshakeTimeout should be positive")
	}
	if config.ReadTimeout <= 0 {
		t.Error("ReadTimeout should be positive")
	}
	if config.WriteTimeout <= 0 {
		t.Error("WriteTimeout should be positive")
	}
	if config.IdleTimeout <= 0 {
		t.Error("IdleTimeout should be positive")
	}
	if config.HeartbeatInterval <= 0 {
		t.Error("HeartbeatInterval should be positive")
	}
}

func TestDefaultServerConfig(t *testing.T) {
	config := DefaultServerConfig()

	if config.Address == "" {
		t.Error("Address should not be empty")
	}
	if config.SessionConfig == nil {
		t.Error("SessionConfig should not be nil")
	}
	if config.ReadBufferSize <= 0 {
		t.Error("ReadBufferSize should be positive")
	}
	if config.WriteBufferSize <= 0 {
		t.Error("WriteBufferSize should be positive")
	}
	if config.ShutdownTimeout <= 0 {
		t.Error("ShutdownTimeout should be positive")
	}
	if config.ReadHeaderTimeout <= 0 {
		t.Error("ReadHeaderTimeout should be positive")
	}
	if config.ReadTimeout <= 0 {
		t.Error("ReadTimeout should be positive")
	}
	if config.WriteTimeout <= 0 {
		t.Error("WriteTimeout should be positive")
	}
	if config.IdleTimeout <= 0 {
		t.Error("IdleTimeout should be positive")
	}
	if config.CheckOrigin == nil {
		t.Error("CheckOrigin should not be nil")
	}
}

func TestDefaultSessionLimits(t *testing.T) {
	limits := DefaultSessionLimits()

	if limits.MaxSessions <= 0 {
		t.Error("MaxSessions should be positive")
	}
	if limits.MaxMemoryPerSession <= 0 {
		t.Error("MaxMemoryPerSession should be positive")
	}
	if limits.MaxTotalMemory <= 0 {
		t.Error("MaxTotalMemory should be positive")
	}
}

func TestServerConfigBuilder(t *testing.T) {
	config := DefaultServerConfig().
		WithAddress(":9090").
		WithMaxSessions(1000).
		WithCSRFSecret([]byte("secret"))

	if config.Address != ":9090" {
		t.Errorf("Address = %s, want :9090", config.Address)
	}
	if config.MaxSessions != 1000 {
		t.Errorf("MaxSessions = %d, want 1000", config.MaxSessions)
	}
	if string(config.CSRFSecret) != "secret" {
		t.Error("CSRFSecret not set correctly")
	}
}

func TestServerConfigWithSessionConfig(t *testing.T) {
	sessionConfig := DefaultSessionConfig()
	sessionConfig.MaxMessageSize = 1024 * 1024

	config := DefaultServerConfig().WithSessionConfig(sessionConfig)

	if config.SessionConfig.MaxMessageSize != 1024*1024 {
		t.Errorf("SessionConfig.MaxMessageSize = %d, want 1MB", config.SessionConfig.MaxMessageSize)
	}
}

func TestServerConfigWithCSRFSecret(t *testing.T) {
	secret := []byte("test-secret-key")
	config := DefaultServerConfig().WithCSRFSecret(secret)

	if string(config.CSRFSecret) != string(secret) {
		t.Error("CSRFSecret not set correctly")
	}
}

func TestSessionConfigValidation(t *testing.T) {
	// Validate default config is sane
	config := DefaultSessionConfig()

	// Heartbeat should be less than idle timeout
	if config.HeartbeatInterval >= config.IdleTimeout {
		t.Error("HeartbeatInterval should be less than IdleTimeout")
	}

	// Read timeout should be reasonable
	if config.ReadTimeout < config.HeartbeatInterval {
		t.Error("ReadTimeout should be at least HeartbeatInterval")
	}
}

func TestSessionConfigClone(t *testing.T) {
	original := DefaultSessionConfig()
	original.MaxMessageSize = 123456

	clone := original.Clone()

	if clone.MaxMessageSize != original.MaxMessageSize {
		t.Error("Clone should copy MaxMessageSize")
	}

	// Modify clone and verify original unchanged
	clone.MaxMessageSize = 999
	if original.MaxMessageSize == clone.MaxMessageSize {
		t.Error("Modifying clone should not affect original")
	}
}

func TestSessionConfigCloneNil(t *testing.T) {
	var config *SessionConfig
	clone := config.Clone()
	if clone != nil {
		t.Error("Clone of nil should return nil")
	}
}

func TestServerConfigClone(t *testing.T) {
	original := DefaultServerConfig()
	original.Address = ":9999"
	original.CSRFSecret = []byte("secret")

	clone := original.Clone()

	if clone.Address != original.Address {
		t.Error("Clone should copy Address")
	}
	if string(clone.CSRFSecret) != string(original.CSRFSecret) {
		t.Error("Clone should copy CSRFSecret")
	}

	// Modify clone and verify original unchanged
	clone.Address = ":1111"
	clone.CSRFSecret[0] = 'X'
	if original.Address == clone.Address {
		t.Error("Modifying clone should not affect original")
	}
}

func TestServerConfigCloneNil(t *testing.T) {
	var config *ServerConfig
	clone := config.Clone()
	if clone != nil {
		t.Error("Clone of nil should return nil")
	}
}

func TestCheckOriginDefault(t *testing.T) {
	config := DefaultServerConfig()

	// Default allows all origins
	req, _ := http.NewRequest("GET", "http://evil.com", nil)
	if !config.CheckOrigin(req) {
		t.Error("Default CheckOrigin should allow all")
	}
}

func TestServerConfigShutdownTimeout(t *testing.T) {
	config := DefaultServerConfig()

	// Modify directly since we don't have a builder method
	config.ShutdownTimeout = 60 * time.Second

	if config.ShutdownTimeout != 60*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 60s", config.ShutdownTimeout)
	}
}

// =============================================================================
// Phase 13: Secure Defaults Tests
// =============================================================================

func TestDefaultServerConfigSecureDefaults(t *testing.T) {
	config := DefaultServerConfig()

	t.Run("DevMode is false by default", func(t *testing.T) {
		if config.DevMode {
			t.Error("DevMode should be false by default")
		}
	})

	t.Run("SecureCookies is true by default", func(t *testing.T) {
		if !config.SecureCookies {
			t.Error("SecureCookies should be true by default")
		}
	})

	t.Run("SameSiteMode is Lax by default", func(t *testing.T) {
		if config.SameSiteMode != http.SameSiteLaxMode {
			t.Errorf("SameSiteMode = %v, want %v", config.SameSiteMode, http.SameSiteLaxMode)
		}
	})

	t.Run("MaxSessionsPerIP is set by default", func(t *testing.T) {
		if config.MaxSessionsPerIP == 0 {
			t.Error("MaxSessionsPerIP should be set by default")
		}
	})

	t.Run("MaxDetachedSessions is set by default", func(t *testing.T) {
		if config.MaxDetachedSessions == 0 {
			t.Error("MaxDetachedSessions should be set by default")
		}
	})
}

func TestWithDevMode(t *testing.T) {
	config := DefaultServerConfig().WithDevMode()

	t.Run("DevMode is enabled", func(t *testing.T) {
		if !config.DevMode {
			t.Error("DevMode should be true after WithDevMode()")
		}
	})

	t.Run("SecureCookies is disabled", func(t *testing.T) {
		if config.SecureCookies {
			t.Error("SecureCookies should be false in DevMode")
		}
	})

	t.Run("CheckOrigin allows all", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "http://localhost", nil)
		req.Header.Set("Origin", "http://malicious.com")
		if !config.CheckOrigin(req) {
			t.Error("CheckOrigin should allow all origins in DevMode")
		}
	})
}

func TestConfigValidation(t *testing.T) {
	t.Run("secure config has no warnings", func(t *testing.T) {
		config := DefaultServerConfig()
		config.CSRFSecret = []byte("secret-key-for-csrf")

		warnings := config.GetConfigWarnings()
		if len(warnings) > 0 {
			t.Errorf("secure config should have no warnings, got: %v", warnings)
		}
	})

	t.Run("missing CSRF secret warns", func(t *testing.T) {
		config := DefaultServerConfig()
		config.CSRFSecret = nil

		warnings := config.GetConfigWarnings()
		found := false
		for _, w := range warnings {
			if containsString(w, "CSRF") {
				found = true
				break
			}
		}
		if !found {
			t.Error("should warn about missing CSRFSecret")
		}
	})

	t.Run("DevMode warns", func(t *testing.T) {
		config := DefaultServerConfig().WithDevMode()

		warnings := config.GetConfigWarnings()
		found := false
		for _, w := range warnings {
			if containsString(w, "DEV MODE") {
				found = true
				break
			}
		}
		if !found {
			t.Error("should warn about DevMode")
		}
	})

	t.Run("disabled SecureCookies warns", func(t *testing.T) {
		config := DefaultServerConfig()
		config.SecureCookies = false

		warnings := config.GetConfigWarnings()
		found := false
		for _, w := range warnings {
			if containsString(w, "SecureCookies") {
				found = true
				break
			}
		}
		if !found {
			t.Error("should warn about disabled SecureCookies")
		}
	})
}

func TestIsSecure(t *testing.T) {
	t.Run("default config is not fully secure", func(t *testing.T) {
		config := DefaultServerConfig()
		// Default config doesn't have CSRFSecret
		if config.IsSecure() {
			t.Error("default config should not be fully secure (missing CSRFSecret)")
		}
	})

	t.Run("fully configured is secure", func(t *testing.T) {
		config := DefaultServerConfig()
		config.CSRFSecret = []byte("secret")

		if !config.IsSecure() {
			t.Error("fully configured config should be secure")
		}
	})

	t.Run("DevMode is not secure", func(t *testing.T) {
		config := DefaultServerConfig().WithDevMode()
		config.CSRFSecret = []byte("secret")

		if config.IsSecure() {
			t.Error("DevMode config should not be secure")
		}
	})
}

func TestConfigHelpers(t *testing.T) {
	t.Run("WithSecureCookies", func(t *testing.T) {
		config := DefaultServerConfig().WithSecureCookies(false)
		if config.SecureCookies {
			t.Error("SecureCookies should be false")
		}

		config = config.WithSecureCookies(true)
		if !config.SecureCookies {
			t.Error("SecureCookies should be true")
		}
	})

	t.Run("WithSameSiteMode", func(t *testing.T) {
		config := DefaultServerConfig().WithSameSiteMode(http.SameSiteStrictMode)
		if config.SameSiteMode != http.SameSiteStrictMode {
			t.Errorf("SameSiteMode = %v, want %v", config.SameSiteMode, http.SameSiteStrictMode)
		}
	})

	t.Run("WithCookieDomain", func(t *testing.T) {
		config := DefaultServerConfig().WithCookieDomain(".example.com")
		if config.CookieDomain != ".example.com" {
			t.Errorf("CookieDomain = %q, want %q", config.CookieDomain, ".example.com")
		}
	})
}

func TestSameOriginCheck(t *testing.T) {
	tests := []struct {
		name   string
		host   string
		origin string
		want   bool
	}{
		{
			name:   "no origin header",
			host:   "localhost:8080",
			origin: "",
			want:   true,
		},
		{
			name:   "same origin",
			host:   "localhost:8080",
			origin: "http://localhost:8080",
			want:   true,
		},
		{
			name:   "different origin",
			host:   "localhost:8080",
			origin: "http://malicious.com",
			want:   false,
		},
		{
			name:   "different port",
			host:   "localhost:8080",
			origin: "http://localhost:3000",
			want:   false,
		},
		{
			name:   "https origin matching https host",
			host:   "example.com",
			origin: "https://example.com",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "http://"+tt.host, nil)
			req.Host = tt.host
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			got := SameOriginCheck(req)
			if got != tt.want {
				t.Errorf("SameOriginCheck() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function for string containment check
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
