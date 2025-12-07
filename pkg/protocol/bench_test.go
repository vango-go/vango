package protocol

import (
	"testing"
)

// Benchmark suite for Phase 3 exit criteria verification.
// Target: < 1μs per event/patch encode/decode

// === Varint Benchmarks ===

func BenchmarkVarint_EncodeSmall(b *testing.B) {
	buf := make([]byte, MaxVarintLen)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EncodeUvarint(buf, 127)
	}
}

func BenchmarkVarint_EncodeLarge(b *testing.B) {
	buf := make([]byte, MaxVarintLen)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EncodeUvarint(buf, 1<<28)
	}
}

func BenchmarkVarint_DecodeSmall(b *testing.B) {
	buf := make([]byte, MaxVarintLen)
	EncodeUvarint(buf, 127)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DecodeUvarint(buf)
	}
}

func BenchmarkVarint_DecodeLarge(b *testing.B) {
	buf := make([]byte, MaxVarintLen)
	n := EncodeUvarint(buf, 1<<28)
	buf = buf[:n]
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DecodeUvarint(buf)
	}
}

// === Encoder/Decoder Benchmarks ===

func BenchmarkEncoder_MixedTypes(b *testing.B) {
	e := NewEncoder()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Reset()
		e.WriteByte(0x42)
		e.WriteUvarint(12345)
		e.WriteSvarint(-9876)
		e.WriteString("hello world")
		e.WriteUint32(0x12345678)
		e.WriteFloat64(3.14159)
	}
}

func BenchmarkDecoder_MixedTypes(b *testing.B) {
	e := NewEncoder()
	e.WriteByte(0x42)
	e.WriteUvarint(12345)
	e.WriteSvarint(-9876)
	e.WriteString("hello world")
	e.WriteUint32(0x12345678)
	e.WriteFloat64(3.14159)
	data := e.Bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d := NewDecoder(data)
		d.ReadByte()
		d.ReadUvarint()
		d.ReadSvarint()
		d.ReadString()
		d.ReadUint32()
		d.ReadFloat64()
	}
}

// === Frame Benchmarks ===

func BenchmarkFrame_EncodeSmall(b *testing.B) {
	f := &Frame{Type: FrameEvent, Payload: []byte{0x01, 0x02, 0x03}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = f.Encode()
	}
}

func BenchmarkFrame_EncodeLarge(b *testing.B) {
	f := &Frame{Type: FramePatches, Payload: make([]byte, 1000)}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = f.Encode()
	}
}

func BenchmarkFrame_DecodeSmall(b *testing.B) {
	f := &Frame{Type: FrameEvent, Payload: []byte{0x01, 0x02, 0x03}}
	data := f.Encode()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeFrame(data)
	}
}

// === Event Benchmarks ===

func BenchmarkEvent_EncodeClick(b *testing.B) {
	e := &Event{Seq: 1, Type: EventClick, HID: "h42"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EncodeEvent(e)
	}
}

func BenchmarkEvent_DecodeClick(b *testing.B) {
	e := &Event{Seq: 1, Type: EventClick, HID: "h42"}
	data := EncodeEvent(e)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeEvent(data)
	}
}

func BenchmarkEvent_EncodeInput(b *testing.B) {
	e := &Event{Seq: 1, Type: EventInput, HID: "h5", Payload: "user input text"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EncodeEvent(e)
	}
}

func BenchmarkEvent_DecodeInput(b *testing.B) {
	e := &Event{Seq: 1, Type: EventInput, HID: "h5", Payload: "user input text"}
	data := EncodeEvent(e)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeEvent(data)
	}
}

func BenchmarkEvent_EncodeSubmit(b *testing.B) {
	e := &Event{
		Seq:  1,
		Type: EventSubmit,
		HID:  "h7",
		Payload: &SubmitEventData{
			Fields: map[string]string{
				"username": "john",
				"password": "secret",
				"email":    "john@example.com",
			},
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EncodeEvent(e)
	}
}

func BenchmarkEvent_DecodeSubmit(b *testing.B) {
	e := &Event{
		Seq:  1,
		Type: EventSubmit,
		HID:  "h7",
		Payload: &SubmitEventData{
			Fields: map[string]string{
				"username": "john",
				"password": "secret",
				"email":    "john@example.com",
			},
		},
	}
	data := EncodeEvent(e)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeEvent(data)
	}
}

func BenchmarkEvent_EncodeKeyboard(b *testing.B) {
	e := &Event{
		Seq:     1,
		Type:    EventKeyDown,
		HID:     "h3",
		Payload: &KeyboardEventData{Key: "Enter", Modifiers: ModCtrl | ModShift},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EncodeEvent(e)
	}
}

// === Patch Benchmarks ===

func BenchmarkPatch_EncodeSetText(b *testing.B) {
	pf := &PatchesFrame{
		Seq:     1,
		Patches: []Patch{NewSetTextPatch("h1", "Hello, World!")},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EncodePatches(pf)
	}
}

func BenchmarkPatch_DecodeSetText(b *testing.B) {
	pf := &PatchesFrame{
		Seq:     1,
		Patches: []Patch{NewSetTextPatch("h1", "Hello, World!")},
	}
	data := EncodePatches(pf)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodePatches(data)
	}
}

func BenchmarkPatch_Encode10(b *testing.B) {
	patches := make([]Patch, 10)
	for i := range patches {
		patches[i] = NewSetTextPatch("h1", "test value")
	}
	pf := &PatchesFrame{Seq: 1, Patches: patches}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EncodePatches(pf)
	}
}

func BenchmarkPatch_Decode10(b *testing.B) {
	patches := make([]Patch, 10)
	for i := range patches {
		patches[i] = NewSetTextPatch("h1", "test value")
	}
	pf := &PatchesFrame{Seq: 1, Patches: patches}
	data := EncodePatches(pf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodePatches(data)
	}
}

func BenchmarkPatch_Encode100(b *testing.B) {
	patches := make([]Patch, 100)
	for i := range patches {
		patches[i] = NewSetTextPatch("h1", "test value")
	}
	pf := &PatchesFrame{Seq: 1, Patches: patches}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EncodePatches(pf)
	}
}

func BenchmarkPatch_Decode100(b *testing.B) {
	patches := make([]Patch, 100)
	for i := range patches {
		patches[i] = NewSetTextPatch("h1", "test value")
	}
	pf := &PatchesFrame{Seq: 1, Patches: patches}
	data := EncodePatches(pf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodePatches(data)
	}
}

// === VNode Benchmarks ===

func BenchmarkVNode_EncodeSimple(b *testing.B) {
	node := NewElementWire("div", map[string]string{"class": "container"},
		NewTextWire("Hello"),
	)
	e := NewEncoder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Reset()
		EncodeVNodeWire(e, node)
	}
}

func BenchmarkVNode_DecodeSimple(b *testing.B) {
	node := NewElementWire("div", map[string]string{"class": "container"},
		NewTextWire("Hello"),
	)
	e := NewEncoder()
	EncodeVNodeWire(e, node)
	data := e.Bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d := NewDecoder(data)
		_, _ = DecodeVNodeWire(d)
	}
}

func BenchmarkVNode_EncodeDeep(b *testing.B) {
	// Create a deeply nested structure
	node := NewElementWire("div", map[string]string{"class": "l1"},
		NewElementWire("div", map[string]string{"class": "l2"},
			NewElementWire("div", map[string]string{"class": "l3"},
				NewElementWire("div", map[string]string{"class": "l4"},
					NewTextWire("Deep content"),
				),
			),
		),
	)
	e := NewEncoder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Reset()
		EncodeVNodeWire(e, node)
	}
}

// === Handshake Benchmarks ===

func BenchmarkHandshake_EncodeClientHello(b *testing.B) {
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

func BenchmarkHandshake_DecodeClientHello(b *testing.B) {
	ch := &ClientHello{
		Version:   CurrentVersion,
		CSRFToken: "abc123token",
		SessionID: "session-12345",
		LastSeq:   42,
		ViewportW: 1920,
		ViewportH: 1080,
		TZOffset:  -480,
	}
	data := EncodeClientHello(ch)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeClientHello(data)
	}
}

// === Control Benchmarks ===

func BenchmarkControl_EncodePing(b *testing.B) {
	ct, pp := NewPing(1702000000000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EncodeControl(ct, pp)
	}
}

func BenchmarkControl_DecodePing(b *testing.B) {
	ct, pp := NewPing(1702000000000)
	data := EncodeControl(ct, pp)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = DecodeControl(data)
	}
}

// === Ack Benchmarks ===

func BenchmarkAck_Encode(b *testing.B) {
	ack := NewAck(42, 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EncodeAck(ack)
	}
}

func BenchmarkAck_Decode(b *testing.B) {
	ack := NewAck(42, 100)
	data := EncodeAck(ack)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeAck(data)
	}
}

// === Error Benchmarks ===

func BenchmarkError_Encode(b *testing.B) {
	em := NewError(ErrHandlerNotFound, "No handler for h42")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EncodeErrorMessage(em)
	}
}

func BenchmarkError_Decode(b *testing.B) {
	em := NewError(ErrHandlerNotFound, "No handler for h42")
	data := EncodeErrorMessage(em)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeErrorMessage(data)
	}
}
