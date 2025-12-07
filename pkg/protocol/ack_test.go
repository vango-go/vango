package protocol

import (
	"testing"
)

func TestAckEncodeDecode(t *testing.T) {
	tests := []struct {
		name string
		ack  *Ack
	}{
		{
			name: "zero",
			ack:  &Ack{LastSeq: 0, Window: 0},
		},
		{
			name: "typical",
			ack:  &Ack{LastSeq: 42, Window: 100},
		},
		{
			name: "large_values",
			ack:  &Ack{LastSeq: 1000000, Window: 1000},
		},
		{
			name: "max_values",
			ack:  &Ack{LastSeq: ^uint64(0), Window: ^uint64(0)},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			encoded := EncodeAck(tc.ack)
			decoded, err := DecodeAck(encoded)
			if err != nil {
				t.Fatalf("DecodeAck() error = %v", err)
			}

			if decoded.LastSeq != tc.ack.LastSeq {
				t.Errorf("LastSeq = %d, want %d", decoded.LastSeq, tc.ack.LastSeq)
			}
			if decoded.Window != tc.ack.Window {
				t.Errorf("Window = %d, want %d", decoded.Window, tc.ack.Window)
			}
		})
	}
}

func TestNewAck(t *testing.T) {
	ack := NewAck(42, 100)

	if ack.LastSeq != 42 {
		t.Errorf("LastSeq = %d, want 42", ack.LastSeq)
	}
	if ack.Window != 100 {
		t.Errorf("Window = %d, want 100", ack.Window)
	}
}

func TestDefaultWindow(t *testing.T) {
	if DefaultWindow != 100 {
		t.Errorf("DefaultWindow = %d, want 100", DefaultWindow)
	}
}

func TestAckEncodingSize(t *testing.T) {
	// Small values should encode compactly
	ack := NewAck(10, 100)
	encoded := EncodeAck(ack)

	// 10 encodes in 1 byte, 100 encodes in 1 byte
	// Total: 2 bytes
	if len(encoded) > 4 {
		t.Errorf("Ack encoding size = %d bytes, want <= 4", len(encoded))
	}
}

func BenchmarkEncodeAck(b *testing.B) {
	ack := NewAck(42, 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EncodeAck(ack)
	}
}

func BenchmarkDecodeAck(b *testing.B) {
	ack := NewAck(42, 100)
	encoded := EncodeAck(ack)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeAck(encoded)
	}
}
