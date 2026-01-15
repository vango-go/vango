package protocol

import (
	"testing"
)

func TestClientHelloEncodeDecode(t *testing.T) {
	tests := []struct {
		name  string
		hello *ClientHello
	}{
		{
			name: "new_session",
			hello: &ClientHello{
				Version:   CurrentVersion,
				CSRFToken: "abc123token",
				SessionID: "",
				LastSeq:   0,
				ViewportW: 1920,
				ViewportH: 1080,
				TZOffset:  -480, // UTC-8
			},
		},
		{
			name: "reconnect",
			hello: &ClientHello{
				Version:   ProtocolVersion{Major: 2, Minor: 1},
				CSRFToken: "xyz789token",
				SessionID: "session-12345",
				LastSeq:   42,
				ViewportW: 1280,
				ViewportH: 720,
				TZOffset:  60, // UTC+1
			},
		},
		{
			name: "minimal",
			hello: &ClientHello{
				Version:   CurrentVersion,
				CSRFToken: "",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			encoded := EncodeClientHello(tc.hello)
			decoded, err := DecodeClientHello(encoded)
			if err != nil {
				t.Fatalf("DecodeClientHello() error = %v", err)
			}

			if decoded.Version != tc.hello.Version {
				t.Errorf("Version = %v, want %v", decoded.Version, tc.hello.Version)
			}
			if decoded.CSRFToken != tc.hello.CSRFToken {
				t.Errorf("CSRFToken = %q, want %q", decoded.CSRFToken, tc.hello.CSRFToken)
			}
			if decoded.SessionID != tc.hello.SessionID {
				t.Errorf("SessionID = %q, want %q", decoded.SessionID, tc.hello.SessionID)
			}
			if decoded.LastSeq != tc.hello.LastSeq {
				t.Errorf("LastSeq = %d, want %d", decoded.LastSeq, tc.hello.LastSeq)
			}
			if decoded.ViewportW != tc.hello.ViewportW {
				t.Errorf("ViewportW = %d, want %d", decoded.ViewportW, tc.hello.ViewportW)
			}
			if decoded.ViewportH != tc.hello.ViewportH {
				t.Errorf("ViewportH = %d, want %d", decoded.ViewportH, tc.hello.ViewportH)
			}
			if decoded.TZOffset != tc.hello.TZOffset {
				t.Errorf("TZOffset = %d, want %d", decoded.TZOffset, tc.hello.TZOffset)
			}
		})
	}
}

func TestServerHelloEncodeDecode(t *testing.T) {
	tests := []struct {
		name  string
		hello *ServerHello
	}{
		{
			name: "success",
			hello: &ServerHello{
				Status:     HandshakeOK,
				SessionID:  "new-session-id",
				NextSeq:    1,
				ServerTime: 1702000000000,
				Flags:      ServerFlagCompression | ServerFlagStreaming,
			},
		},
		{
			name: "version_mismatch",
			hello: &ServerHello{
				Status: HandshakeVersionMismatch,
			},
		},
		{
			name: "session_expired",
			hello: &ServerHello{
				Status:    HandshakeSessionExpired,
				SessionID: "new-session-after-expiry",
				NextSeq:   1,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			encoded := EncodeServerHello(tc.hello)
			decoded, err := DecodeServerHello(encoded)
			if err != nil {
				t.Fatalf("DecodeServerHello() error = %v", err)
			}

			if decoded.Status != tc.hello.Status {
				t.Errorf("Status = %v, want %v", decoded.Status, tc.hello.Status)
			}
			if decoded.SessionID != tc.hello.SessionID {
				t.Errorf("SessionID = %q, want %q", decoded.SessionID, tc.hello.SessionID)
			}
			if decoded.NextSeq != tc.hello.NextSeq {
				t.Errorf("NextSeq = %d, want %d", decoded.NextSeq, tc.hello.NextSeq)
			}
			if decoded.ServerTime != tc.hello.ServerTime {
				t.Errorf("ServerTime = %d, want %d", decoded.ServerTime, tc.hello.ServerTime)
			}
			if decoded.Flags != tc.hello.Flags {
				t.Errorf("Flags = %x, want %x", decoded.Flags, tc.hello.Flags)
			}
		})
	}
}

func TestHandshakeStatusString(t *testing.T) {
	tests := []struct {
		status HandshakeStatus
		want   string
	}{
		{HandshakeOK, "OK"},
		{HandshakeVersionMismatch, "VersionMismatch"},
		{HandshakeInvalidCSRF, "InvalidCSRF"},
		{HandshakeSessionExpired, "SessionExpired"},
		{HandshakeServerBusy, "ServerBusy"},
		{HandshakeUpgradeRequired, "UpgradeRequired"},
		{HandshakeLimitExceeded, "LimitExceeded"},
		{HandshakeStatus(0xFF), "Unknown"},
	}

	for _, tc := range tests {
		if got := tc.status.String(); got != tc.want {
			t.Errorf("HandshakeStatus(%d).String() = %q, want %q", tc.status, got, tc.want)
		}
	}
}

func TestNewClientHello(t *testing.T) {
	ch := NewClientHello("my-csrf-token")

	if ch.Version != CurrentVersion {
		t.Errorf("Version = %v, want %v", ch.Version, CurrentVersion)
	}
	if ch.CSRFToken != "my-csrf-token" {
		t.Errorf("CSRFToken = %q, want %q", ch.CSRFToken, "my-csrf-token")
	}
}

func TestNewServerHello(t *testing.T) {
	sh := NewServerHello("session-123", 42, 1702000000000)

	if sh.Status != HandshakeOK {
		t.Errorf("Status = %v, want OK", sh.Status)
	}
	if sh.SessionID != "session-123" {
		t.Errorf("SessionID = %q, want %q", sh.SessionID, "session-123")
	}
	if sh.NextSeq != 42 {
		t.Errorf("NextSeq = %d, want 42", sh.NextSeq)
	}
	if sh.ServerTime != 1702000000000 {
		t.Errorf("ServerTime = %d, want 1702000000000", sh.ServerTime)
	}
}

func TestNewServerHelloError(t *testing.T) {
	sh := NewServerHelloError(HandshakeServerBusy)

	if sh.Status != HandshakeServerBusy {
		t.Errorf("Status = %v, want ServerBusy", sh.Status)
	}
}

func BenchmarkEncodeClientHello(b *testing.B) {
	ch := &ClientHello{
		Version:   CurrentVersion,
		CSRFToken: "abc123token",
		SessionID: "session-12345",
		LastSeq:   42,
		ViewportW: 1920,
		ViewportH: 1080,
		TZOffset:  -480,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EncodeClientHello(ch)
	}
}

func BenchmarkDecodeClientHello(b *testing.B) {
	ch := &ClientHello{
		Version:   CurrentVersion,
		CSRFToken: "abc123token",
		SessionID: "session-12345",
		LastSeq:   42,
		ViewportW: 1920,
		ViewportH: 1080,
		TZOffset:  -480,
	}
	encoded := EncodeClientHello(ch)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeClientHello(encoded)
	}
}
