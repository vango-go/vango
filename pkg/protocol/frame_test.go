package protocol

import (
	"bytes"
	"io"
	"testing"
)

func TestFrameEncodeDecode(t *testing.T) {
	tests := []struct {
		name    string
		frame   Frame
		wantLen int // expected total length including header
	}{
		{
			name: "empty_payload",
			frame: Frame{
				Type:    FrameEvent,
				Flags:   0,
				Payload: []byte{},
			},
			wantLen: FrameHeaderSize,
		},
		{
			name: "with_payload",
			frame: Frame{
				Type:    FramePatches,
				Flags:   FlagSequenced,
				Payload: []byte{0x01, 0x02, 0x03},
			},
			wantLen: FrameHeaderSize + 3,
		},
		{
			name: "with_flags",
			frame: Frame{
				Type:    FrameControl,
				Flags:   FlagCompressed | FlagPriority,
				Payload: []byte("test"),
			},
			wantLen: FrameHeaderSize + 4,
		},
		{
			name: "handshake",
			frame: Frame{
				Type:    FrameHandshake,
				Flags:   FlagFinal,
				Payload: []byte{0x02, 0x00, 0x00}, // Version 2.0
			},
			wantLen: FrameHeaderSize + 3,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Encode
			encoded := tc.frame.Encode()
			if len(encoded) != tc.wantLen {
				t.Errorf("Encode() length = %d, want %d", len(encoded), tc.wantLen)
			}

			// Verify header
			if FrameType(encoded[0]) != tc.frame.Type {
				t.Errorf("Encoded type = %v, want %v", FrameType(encoded[0]), tc.frame.Type)
			}
			if FrameFlags(encoded[1]) != tc.frame.Flags {
				t.Errorf("Encoded flags = %v, want %v", FrameFlags(encoded[1]), tc.frame.Flags)
			}

			// Decode
			decoded, err := DecodeFrame(encoded)
			if err != nil {
				t.Fatalf("DecodeFrame() error = %v", err)
			}

			if decoded.Type != tc.frame.Type {
				t.Errorf("Decoded type = %v, want %v", decoded.Type, tc.frame.Type)
			}
			if decoded.Flags != tc.frame.Flags {
				t.Errorf("Decoded flags = %v, want %v", decoded.Flags, tc.frame.Flags)
			}
			if !bytes.Equal(decoded.Payload, tc.frame.Payload) {
				t.Errorf("Decoded payload = %v, want %v", decoded.Payload, tc.frame.Payload)
			}
		})
	}
}

func TestFrameEncodeTo(t *testing.T) {
	f := &Frame{
		Type:    FrameEvent,
		Flags:   FlagSequenced,
		Payload: []byte{0x01, 0x02, 0x03},
	}

	e := NewEncoder()
	f.EncodeTo(e)

	direct := f.Encode()
	if !bytes.Equal(e.Bytes(), direct) {
		t.Errorf("EncodeTo() = %v, want %v", e.Bytes(), direct)
	}
}

func TestDecodeFrameHeader(t *testing.T) {
	data := []byte{0x02, 0x03, 0x00, 0x10} // FramePatches, FlagCompressed|FlagSequenced, length 16

	ft, flags, length, err := DecodeFrameHeader(data)
	if err != nil {
		t.Fatalf("DecodeFrameHeader() error = %v", err)
	}
	if ft != FramePatches {
		t.Errorf("Type = %v, want FramePatches", ft)
	}
	if flags != FlagCompressed|FlagSequenced {
		t.Errorf("Flags = %v, want FlagCompressed|FlagSequenced", flags)
	}
	if length != 16 {
		t.Errorf("Length = %d, want 16", length)
	}
}

func TestDecodeFrameErrors(t *testing.T) {
	// Short header
	_, err := DecodeFrame([]byte{0x00, 0x00})
	if err != io.ErrUnexpectedEOF {
		t.Errorf("Short header: got %v, want io.ErrUnexpectedEOF", err)
	}

	// Short payload
	_, err = DecodeFrame([]byte{0x01, 0x00, 0x00, 0x10}) // Claims 16 bytes, has 0
	if err != io.ErrUnexpectedEOF {
		t.Errorf("Short payload: got %v, want io.ErrUnexpectedEOF", err)
	}
}

func TestReadWriteFrame(t *testing.T) {
	original := &Frame{
		Type:    FrameEvent,
		Flags:   FlagSequenced | FlagFinal,
		Payload: []byte("hello world"),
	}

	// Write to buffer
	var buf bytes.Buffer
	if err := WriteFrame(&buf, original); err != nil {
		t.Fatalf("WriteFrame() error = %v", err)
	}

	// Read back
	decoded, err := ReadFrame(&buf)
	if err != nil {
		t.Fatalf("ReadFrame() error = %v", err)
	}

	if decoded.Type != original.Type {
		t.Errorf("Type = %v, want %v", decoded.Type, original.Type)
	}
	if decoded.Flags != original.Flags {
		t.Errorf("Flags = %v, want %v", decoded.Flags, original.Flags)
	}
	if !bytes.Equal(decoded.Payload, original.Payload) {
		t.Errorf("Payload = %v, want %v", decoded.Payload, original.Payload)
	}
}

func TestReadFrameErrors(t *testing.T) {
	// EOF on header
	_, err := ReadFrame(bytes.NewReader([]byte{}))
	if err != io.EOF {
		t.Errorf("Empty reader: got %v, want io.EOF", err)
	}

	// Short header
	_, err = ReadFrame(bytes.NewReader([]byte{0x00, 0x00}))
	if err != io.ErrUnexpectedEOF {
		t.Errorf("Short header: got %v, want io.ErrUnexpectedEOF", err)
	}

	// Short payload
	_, err = ReadFrame(bytes.NewReader([]byte{0x01, 0x00, 0x00, 0x05, 0x01, 0x02}))
	if err != io.ErrUnexpectedEOF {
		t.Errorf("Short payload: got %v, want io.ErrUnexpectedEOF", err)
	}
}

func TestWriteFrameTooLarge(t *testing.T) {
	f := &Frame{
		Type:    FramePatches,
		Payload: make([]byte, MaxPayloadSize+1),
	}

	var buf bytes.Buffer
	err := WriteFrame(&buf, f)
	if err != ErrFrameTooLarge {
		t.Errorf("WriteFrame() = %v, want ErrFrameTooLarge", err)
	}
}

func TestFrameTypeString(t *testing.T) {
	tests := []struct {
		ft   FrameType
		want string
	}{
		{FrameHandshake, "Handshake"},
		{FrameEvent, "Event"},
		{FramePatches, "Patches"},
		{FrameControl, "Control"},
		{FrameAck, "Ack"},
		{FrameError, "Error"},
		{FrameType(0xFF), "Unknown"},
	}

	for _, tc := range tests {
		if got := tc.ft.String(); got != tc.want {
			t.Errorf("FrameType(%d).String() = %q, want %q", tc.ft, got, tc.want)
		}
	}
}

func TestFrameFlagsHas(t *testing.T) {
	flags := FlagCompressed | FlagSequenced

	if !flags.Has(FlagCompressed) {
		t.Error("Has(FlagCompressed) = false, want true")
	}
	if !flags.Has(FlagSequenced) {
		t.Error("Has(FlagSequenced) = false, want true")
	}
	if flags.Has(FlagFinal) {
		t.Error("Has(FlagFinal) = true, want false")
	}
	if flags.Has(FlagPriority) {
		t.Error("Has(FlagPriority) = true, want false")
	}
}

func TestNewFrame(t *testing.T) {
	payload := []byte{1, 2, 3}
	f := NewFrame(FrameEvent, payload)

	if f.Type != FrameEvent {
		t.Errorf("Type = %v, want FrameEvent", f.Type)
	}
	if f.Flags != 0 {
		t.Errorf("Flags = %v, want 0", f.Flags)
	}
	if !bytes.Equal(f.Payload, payload) {
		t.Errorf("Payload = %v, want %v", f.Payload, payload)
	}
}

func TestNewFrameWithFlags(t *testing.T) {
	payload := []byte{1, 2, 3}
	flags := FlagCompressed | FlagPriority
	f := NewFrameWithFlags(FramePatches, flags, payload)

	if f.Type != FramePatches {
		t.Errorf("Type = %v, want FramePatches", f.Type)
	}
	if f.Flags != flags {
		t.Errorf("Flags = %v, want %v", f.Flags, flags)
	}
	if !bytes.Equal(f.Payload, payload) {
		t.Errorf("Payload = %v, want %v", f.Payload, payload)
	}
}

func TestMultipleFrames(t *testing.T) {
	// Write multiple frames
	var buf bytes.Buffer

	frames := []*Frame{
		{Type: FrameEvent, Payload: []byte("frame1")},
		{Type: FramePatches, Payload: []byte("frame2")},
		{Type: FrameControl, Payload: []byte("frame3")},
	}

	for _, f := range frames {
		if err := WriteFrame(&buf, f); err != nil {
			t.Fatalf("WriteFrame() error = %v", err)
		}
	}

	// Read them back
	reader := bytes.NewReader(buf.Bytes())
	for i, original := range frames {
		decoded, err := ReadFrame(reader)
		if err != nil {
			t.Fatalf("ReadFrame(%d) error = %v", i, err)
		}
		if decoded.Type != original.Type {
			t.Errorf("Frame %d: Type = %v, want %v", i, decoded.Type, original.Type)
		}
		if !bytes.Equal(decoded.Payload, original.Payload) {
			t.Errorf("Frame %d: Payload mismatch", i)
		}
	}
}

func BenchmarkFrameEncode(b *testing.B) {
	f := &Frame{
		Type:    FrameEvent,
		Flags:   FlagSequenced,
		Payload: make([]byte, 100),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = f.Encode()
	}
}

func BenchmarkFrameDecode(b *testing.B) {
	f := &Frame{
		Type:    FrameEvent,
		Flags:   FlagSequenced,
		Payload: make([]byte, 100),
	}
	data := f.Encode()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeFrame(data)
	}
}

func BenchmarkFrameReadWrite(b *testing.B) {
	f := &Frame{
		Type:    FrameEvent,
		Flags:   FlagSequenced,
		Payload: make([]byte, 100),
	}
	var buf bytes.Buffer
	data := f.Encode()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		buf.Write(data)
		_, _ = ReadFrame(&buf)
	}
}
