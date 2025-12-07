package protocol

import (
	"testing"
)

func TestControlEncodeDecode(t *testing.T) {
	tests := []struct {
		name    string
		ct      ControlType
		payload any
	}{
		{
			name:    "ping",
			ct:      ControlPing,
			payload: &PingPong{Timestamp: 1702000000000},
		},
		{
			name:    "pong",
			ct:      ControlPong,
			payload: &PingPong{Timestamp: 1702000000001},
		},
		{
			name:    "resync_request",
			ct:      ControlResyncRequest,
			payload: &ResyncRequest{LastSeq: 42},
		},
		{
			name: "resync_patches",
			ct:   ControlResyncPatches,
			payload: &ResyncResponse{
				Type:    ControlResyncPatches,
				FromSeq: 43,
				Patches: []Patch{
					NewSetTextPatch("h1", "Updated"),
					NewSetAttrPatch("h2", "class", "active"),
				},
			},
		},
		{
			name: "resync_full",
			ct:   ControlResyncFull,
			payload: &ResyncResponse{
				Type: ControlResyncFull,
				HTML: "<div>Full reload</div>",
			},
		},
		{
			name: "close_normal",
			ct:   ControlClose,
			payload: &CloseMessage{
				Reason:  CloseNormal,
				Message: "",
			},
		},
		{
			name: "close_with_message",
			ct:   ControlClose,
			payload: &CloseMessage{
				Reason:  CloseServerShutdown,
				Message: "Server is restarting",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			encoded := EncodeControl(tc.ct, tc.payload)
			decodedType, decodedPayload, err := DecodeControl(encoded)
			if err != nil {
				t.Fatalf("DecodeControl() error = %v", err)
			}

			if decodedType != tc.ct {
				t.Errorf("Type = %v, want %v", decodedType, tc.ct)
			}

			verifyControlPayload(t, tc.name, decodedPayload, tc.payload)
		})
	}
}

func verifyControlPayload(t *testing.T, _ string, got, want any) {
	t.Helper()

	switch w := want.(type) {
	case *PingPong:
		g, ok := got.(*PingPong)
		if !ok {
			t.Errorf("Payload type = %T, want *PingPong", got)
			return
		}
		if g.Timestamp != w.Timestamp {
			t.Errorf("Timestamp = %d, want %d", g.Timestamp, w.Timestamp)
		}

	case *ResyncRequest:
		g, ok := got.(*ResyncRequest)
		if !ok {
			t.Errorf("Payload type = %T, want *ResyncRequest", got)
			return
		}
		if g.LastSeq != w.LastSeq {
			t.Errorf("LastSeq = %d, want %d", g.LastSeq, w.LastSeq)
		}

	case *ResyncResponse:
		g, ok := got.(*ResyncResponse)
		if !ok {
			t.Errorf("Payload type = %T, want *ResyncResponse", got)
			return
		}
		if g.Type != w.Type {
			t.Errorf("Type = %v, want %v", g.Type, w.Type)
		}
		if g.FromSeq != w.FromSeq {
			t.Errorf("FromSeq = %d, want %d", g.FromSeq, w.FromSeq)
		}
		if g.HTML != w.HTML {
			t.Errorf("HTML = %q, want %q", g.HTML, w.HTML)
		}
		if len(g.Patches) != len(w.Patches) {
			t.Errorf("Patches count = %d, want %d", len(g.Patches), len(w.Patches))
		}

	case *CloseMessage:
		g, ok := got.(*CloseMessage)
		if !ok {
			t.Errorf("Payload type = %T, want *CloseMessage", got)
			return
		}
		if g.Reason != w.Reason {
			t.Errorf("Reason = %v, want %v", g.Reason, w.Reason)
		}
		if g.Message != w.Message {
			t.Errorf("Message = %q, want %q", g.Message, w.Message)
		}
	}
}

func TestControlTypeString(t *testing.T) {
	tests := []struct {
		ct   ControlType
		want string
	}{
		{ControlPing, "Ping"},
		{ControlPong, "Pong"},
		{ControlResyncRequest, "ResyncRequest"},
		{ControlResyncPatches, "ResyncPatches"},
		{ControlResyncFull, "ResyncFull"},
		{ControlClose, "Close"},
		{ControlType(0xFF), "Unknown"},
	}

	for _, tc := range tests {
		if got := tc.ct.String(); got != tc.want {
			t.Errorf("ControlType(%d).String() = %q, want %q", tc.ct, got, tc.want)
		}
	}
}

func TestCloseReasonString(t *testing.T) {
	tests := []struct {
		cr   CloseReason
		want string
	}{
		{CloseNormal, "Normal"},
		{CloseGoingAway, "GoingAway"},
		{CloseSessionExpired, "SessionExpired"},
		{CloseServerShutdown, "ServerShutdown"},
		{CloseError, "Error"},
		{CloseReason(0xFF), "Unknown"},
	}

	for _, tc := range tests {
		if got := tc.cr.String(); got != tc.want {
			t.Errorf("CloseReason(%d).String() = %q, want %q", tc.cr, got, tc.want)
		}
	}
}

func TestNewControlHelpers(t *testing.T) {
	// Test NewPing
	ct, pp := NewPing(1702000000000)
	if ct != ControlPing {
		t.Errorf("NewPing type = %v, want Ping", ct)
	}
	if pp.Timestamp != 1702000000000 {
		t.Errorf("NewPing timestamp = %d, want 1702000000000", pp.Timestamp)
	}

	// Test NewPong
	ct, pp = NewPong(1702000000001)
	if ct != ControlPong {
		t.Errorf("NewPong type = %v, want Pong", ct)
	}

	// Test NewResyncRequest
	ct, rr := NewResyncRequest(42)
	if ct != ControlResyncRequest {
		t.Errorf("NewResyncRequest type = %v, want ResyncRequest", ct)
	}
	if rr.LastSeq != 42 {
		t.Errorf("NewResyncRequest LastSeq = %d, want 42", rr.LastSeq)
	}

	// Test NewResyncPatches
	patches := []Patch{NewSetTextPatch("h1", "test")}
	ct, resp := NewResyncPatches(10, patches)
	if ct != ControlResyncPatches {
		t.Errorf("NewResyncPatches type = %v, want ResyncPatches", ct)
	}
	if resp.FromSeq != 10 {
		t.Errorf("NewResyncPatches FromSeq = %d, want 10", resp.FromSeq)
	}
	if len(resp.Patches) != 1 {
		t.Errorf("NewResyncPatches Patches count = %d, want 1", len(resp.Patches))
	}

	// Test NewResyncFull
	ct, resp = NewResyncFull("<html>...</html>")
	if ct != ControlResyncFull {
		t.Errorf("NewResyncFull type = %v, want ResyncFull", ct)
	}
	if resp.HTML != "<html>...</html>" {
		t.Errorf("NewResyncFull HTML = %q, want %q", resp.HTML, "<html>...</html>")
	}

	// Test NewClose
	ct, cm := NewClose(CloseGoingAway, "Bye")
	if ct != ControlClose {
		t.Errorf("NewClose type = %v, want Close", ct)
	}
	if cm.Reason != CloseGoingAway {
		t.Errorf("NewClose Reason = %v, want GoingAway", cm.Reason)
	}
	if cm.Message != "Bye" {
		t.Errorf("NewClose Message = %q, want %q", cm.Message, "Bye")
	}
}

func BenchmarkEncodePing(b *testing.B) {
	ct, pp := NewPing(1702000000000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EncodeControl(ct, pp)
	}
}

func BenchmarkDecodePing(b *testing.B) {
	ct, pp := NewPing(1702000000000)
	encoded := EncodeControl(ct, pp)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = DecodeControl(encoded)
	}
}
