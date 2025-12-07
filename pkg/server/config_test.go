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
