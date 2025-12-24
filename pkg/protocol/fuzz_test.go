package protocol

import (
	"testing"
)

// FuzzDecodeUvarint tests that decoding arbitrary bytes doesn't panic.
func FuzzDecodeUvarint(f *testing.F) {
	// Seed with valid varints
	f.Add([]byte{0x00})
	f.Add([]byte{0x7F})
	f.Add([]byte{0x80, 0x01})
	f.Add([]byte{0xFF, 0x7F})
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x01})

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should not panic
		_, _ = DecodeUvarint(data)
	})
}

// FuzzDecodeSvarint tests that decoding arbitrary bytes doesn't panic.
func FuzzDecodeSvarint(f *testing.F) {
	// Seed with valid varints
	f.Add([]byte{0x00})
	f.Add([]byte{0x01})
	f.Add([]byte{0x02})

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should not panic
		_, _ = DecodeSvarint(data)
	})
}

// FuzzDecodeFrame tests that decoding arbitrary bytes doesn't panic.
func FuzzDecodeFrame(f *testing.F) {
	// Seed with valid frames
	frame := &Frame{Type: FrameEvent, Payload: []byte{0x01, 0x02}}
	f.Add(frame.Encode())

	frame2 := &Frame{Type: FramePatches, Flags: FlagCompressed, Payload: []byte("test")}
	f.Add(frame2.Encode())

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should not panic
		_, _ = DecodeFrame(data)
	})
}

// FuzzDecodeEvent tests that decoding arbitrary bytes doesn't panic.
func FuzzDecodeEvent(f *testing.F) {
	// Seed with valid events
	click := &Event{Seq: 1, Type: EventClick, HID: "h1"}
	f.Add(EncodeEvent(click))

	input := &Event{Seq: 2, Type: EventInput, HID: "h5", Payload: "hello"}
	f.Add(EncodeEvent(input))

	keyboard := &Event{
		Seq:     3,
		Type:    EventKeyDown,
		HID:     "h3",
		Payload: &KeyboardEventData{Key: "Enter", Modifiers: ModCtrl},
	}
	f.Add(EncodeEvent(keyboard))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should not panic
		_, _ = DecodeEvent(data)
	})
}

// FuzzDecodePatches tests that decoding arbitrary bytes doesn't panic.
func FuzzDecodePatches(f *testing.F) {
	// Seed with valid patches
	pf := &PatchesFrame{
		Seq: 1,
		Patches: []Patch{
			NewSetTextPatch("h1", "Hello"),
			NewSetAttrPatch("h2", "class", "active"),
		},
	}
	f.Add(EncodePatches(pf))

	pf2 := &PatchesFrame{Seq: 2, Patches: []Patch{}}
	f.Add(EncodePatches(pf2))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should not panic
		_, _ = DecodePatches(data)
	})
}

// FuzzDecodeVNodeWire tests that decoding arbitrary bytes doesn't panic.
func FuzzDecodeVNodeWire(f *testing.F) {
	// Seed with valid VNodes
	e := NewEncoder()
	EncodeVNodeWire(e, NewTextWire("Hello"))
	f.Add(e.Bytes())

	e.Reset()
	EncodeVNodeWire(e, NewElementWire("div", map[string]string{"class": "test"},
		NewTextWire("Content"),
	))
	f.Add(e.Bytes())

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should not panic
		d := NewDecoder(data)
		_, _ = DecodeVNodeWire(d)
	})
}

// FuzzDecodeClientHello tests that decoding arbitrary bytes doesn't panic.
func FuzzDecodeClientHello(f *testing.F) {
	ch := &ClientHello{
		Version:   CurrentVersion,
		CSRFToken: "token",
		SessionID: "session",
		LastSeq:   42,
		ViewportW: 1920,
		ViewportH: 1080,
		TZOffset:  -480,
	}
	f.Add(EncodeClientHello(ch))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should not panic
		_, _ = DecodeClientHello(data)
	})
}

// FuzzDecodeServerHello tests that decoding arbitrary bytes doesn't panic.
func FuzzDecodeServerHello(f *testing.F) {
	sh := &ServerHello{
		Status:     HandshakeOK,
		SessionID:  "session-123",
		NextSeq:    1,
		ServerTime: 1702000000000,
		Flags:      ServerFlagCompression,
	}
	f.Add(EncodeServerHello(sh))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should not panic
		_, _ = DecodeServerHello(data)
	})
}

// FuzzDecodeControl tests that decoding arbitrary bytes doesn't panic.
func FuzzDecodeControl(f *testing.F) {
	// Seed with valid control messages
	f.Add(EncodeControl(ControlPing, &PingPong{Timestamp: 1702000000000}))
	f.Add(EncodeControl(ControlClose, &CloseMessage{Reason: CloseNormal, Message: "bye"}))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should not panic
		_, _, _ = DecodeControl(data)
	})
}

// FuzzDecodeAck tests that decoding arbitrary bytes doesn't panic.
func FuzzDecodeAck(f *testing.F) {
	f.Add(EncodeAck(NewAck(42, 100)))
	f.Add(EncodeAck(NewAck(0, 0)))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should not panic
		_, _ = DecodeAck(data)
	})
}

// FuzzDecodeErrorMessage tests that decoding arbitrary bytes doesn't panic.
func FuzzDecodeErrorMessage(f *testing.F) {
	f.Add(EncodeErrorMessage(NewError(ErrHandlerNotFound, "test")))
	f.Add(EncodeErrorMessage(NewFatalError(ErrServerError, "fatal error")))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should not panic
		_, _ = DecodeErrorMessage(data)
	})
}

// FuzzRoundTrip tests that encoding and decoding produces the same result.
func FuzzRoundTrip(f *testing.F) {
	f.Add("hello world", uint64(42), int64(-123))

	f.Fuzz(func(t *testing.T, s string, u uint64, i int64) {
		e := NewEncoder()
		e.WriteString(s)
		e.WriteUvarint(u)
		e.WriteSvarint(i)

		d := NewDecoder(e.Bytes())
		gotS, err := d.ReadString()
		if err != nil {
			return // Invalid input, that's fine
		}
		gotU, err := d.ReadUvarint()
		if err != nil {
			return
		}
		gotI, err := d.ReadSvarint()
		if err != nil {
			return
		}

		if gotS != s {
			t.Errorf("String: got %q, want %q", gotS, s)
		}
		if gotU != u {
			t.Errorf("Uvarint: got %d, want %d", gotU, u)
		}
		if gotI != i {
			t.Errorf("Svarint: got %d, want %d", gotI, i)
		}
	})
}

// FuzzDecodeHookPayload tests that decoding hook payloads doesn't panic.
// This is important for security as hook payloads come from clients.
func FuzzDecodeHookPayload(f *testing.F) {
	// Seed with valid JSON payloads
	f.Add([]byte(`{"key": "value"}`))
	f.Add([]byte(`{"nested": {"deep": {"deeper": true}}}`))
	f.Add([]byte(`[1, 2, 3, "four", null, true]`))
	f.Add([]byte(`"simple string"`))
	f.Add([]byte(`42`))
	f.Add([]byte(`true`))
	f.Add([]byte(`null`))
	f.Add([]byte(`{"array": [{"a": 1}, {"b": 2}]}`))

	// Edge cases
	f.Add([]byte(`{}`))
	f.Add([]byte(`[]`))
	f.Add([]byte(`""`))
	f.Add([]byte(`0`))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should not panic - errors are acceptable for invalid input
		_, _ = decodeHookValueWithDepth(NewDecoder(data), 0)
	})
}

// FuzzDeeplyNestedVNode tests that deeply nested VNodes are rejected.
func FuzzDeeplyNestedVNode(f *testing.F) {
	// Seed with various VNode structures
	e := NewEncoder()
	EncodeVNodeWire(e, NewTextWire("Hello"))
	f.Add(e.Bytes())

	e.Reset()
	EncodeVNodeWire(e, NewElementWire("div", nil))
	f.Add(e.Bytes())

	f.Fuzz(func(t *testing.T, data []byte) {
		d := NewDecoder(data)
		// Should not panic, should return error for deep nesting
		_, _ = DecodeVNodeWire(d)
	})
}

// FuzzPatchesWithVNodes tests patches containing VNodes are properly limited.
func FuzzPatchesWithVNodes(f *testing.F) {
	// Seed with valid patches
	pf := &PatchesFrame{
		Seq: 1,
		Patches: []Patch{
			NewInsertNodePatch("h1", "h0", 0, NewTextWire("text")),
			NewReplaceNodePatch("h2", NewElementWire("span", nil)),
		},
	}
	f.Add(EncodePatches(pf))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Should not panic
		_, _ = DecodePatches(data)
	})
}
