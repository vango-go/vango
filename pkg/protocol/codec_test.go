package protocol

import (
	"io"
	"math"
	"testing"
)

func TestEncoderDecoder(t *testing.T) {
	e := NewEncoder()

	// Write various types
	e.WriteByte(0x42)
	e.WriteBytes([]byte{0x01, 0x02, 0x03})
	e.WriteUvarint(12345)
	e.WriteSvarint(-9876)
	e.WriteString("hello world")
	e.WriteLenBytes([]byte{0xDE, 0xAD, 0xBE, 0xEF})
	e.WriteBool(true)
	e.WriteBool(false)
	e.WriteUint16(0x1234)
	e.WriteUint32(0x12345678)
	e.WriteUint64(0x123456789ABCDEF0)
	e.WriteInt16(-1234)
	e.WriteInt32(-12345678)
	e.WriteInt64(-123456789012345)
	e.WriteFloat32(3.14159)
	e.WriteFloat64(2.718281828459045)

	// Decode and verify
	d := NewDecoder(e.Bytes())

	// Byte
	b, err := d.ReadByte()
	if err != nil || b != 0x42 {
		t.Errorf("ReadByte() = %x, %v; want 0x42, nil", b, err)
	}

	// Bytes
	bs, err := d.ReadBytes(3)
	if err != nil || string(bs) != "\x01\x02\x03" {
		t.Errorf("ReadBytes(3) = %v, %v; want [1 2 3], nil", bs, err)
	}

	// Uvarint
	uv, err := d.ReadUvarint()
	if err != nil || uv != 12345 {
		t.Errorf("ReadUvarint() = %d, %v; want 12345, nil", uv, err)
	}

	// Svarint
	sv, err := d.ReadSvarint()
	if err != nil || sv != -9876 {
		t.Errorf("ReadSvarint() = %d, %v; want -9876, nil", sv, err)
	}

	// String
	s, err := d.ReadString()
	if err != nil || s != "hello world" {
		t.Errorf("ReadString() = %q, %v; want \"hello world\", nil", s, err)
	}

	// LenBytes
	lb, err := d.ReadLenBytes()
	if err != nil || len(lb) != 4 || lb[0] != 0xDE {
		t.Errorf("ReadLenBytes() = %v, %v; want [DE AD BE EF], nil", lb, err)
	}

	// Bools
	bt, err := d.ReadBool()
	if err != nil || bt != true {
		t.Errorf("ReadBool() = %v, %v; want true, nil", bt, err)
	}
	bf, err := d.ReadBool()
	if err != nil || bf != false {
		t.Errorf("ReadBool() = %v, %v; want false, nil", bf, err)
	}

	// Uint16
	u16, err := d.ReadUint16()
	if err != nil || u16 != 0x1234 {
		t.Errorf("ReadUint16() = %x, %v; want 0x1234, nil", u16, err)
	}

	// Uint32
	u32, err := d.ReadUint32()
	if err != nil || u32 != 0x12345678 {
		t.Errorf("ReadUint32() = %x, %v; want 0x12345678, nil", u32, err)
	}

	// Uint64
	u64, err := d.ReadUint64()
	if err != nil || u64 != 0x123456789ABCDEF0 {
		t.Errorf("ReadUint64() = %x, %v; want 0x123456789ABCDEF0, nil", u64, err)
	}

	// Int16
	i16, err := d.ReadInt16()
	if err != nil || i16 != -1234 {
		t.Errorf("ReadInt16() = %d, %v; want -1234, nil", i16, err)
	}

	// Int32
	i32, err := d.ReadInt32()
	if err != nil || i32 != -12345678 {
		t.Errorf("ReadInt32() = %d, %v; want -12345678, nil", i32, err)
	}

	// Int64
	i64, err := d.ReadInt64()
	if err != nil || i64 != -123456789012345 {
		t.Errorf("ReadInt64() = %d, %v; want -123456789012345, nil", i64, err)
	}

	// Float32
	f32, err := d.ReadFloat32()
	if err != nil || math.Abs(float64(f32)-3.14159) > 0.00001 {
		t.Errorf("ReadFloat32() = %v, %v; want ~3.14159, nil", f32, err)
	}

	// Float64
	f64, err := d.ReadFloat64()
	if err != nil || math.Abs(f64-2.718281828459045) > 0.0000001 {
		t.Errorf("ReadFloat64() = %v, %v; want ~2.718281828459045, nil", f64, err)
	}

	// Should be at EOF
	if !d.EOF() {
		t.Errorf("Expected EOF, but %d bytes remaining", d.Remaining())
	}
}

func TestEncoderReset(t *testing.T) {
	e := NewEncoder()
	e.WriteString("test")
	if e.Len() == 0 {
		t.Error("Encoder should have data after write")
	}

	e.Reset()
	if e.Len() != 0 {
		t.Error("Encoder should be empty after reset")
	}

	e.WriteString("new data")
	if e.Len() == 0 {
		t.Error("Encoder should have data after write following reset")
	}
}

func TestEncoderWithCap(t *testing.T) {
	e := NewEncoderWithCap(1024)
	if cap(e.Bytes()) < 1024 {
		t.Errorf("Expected capacity >= 1024, got %d", cap(e.Bytes()))
	}
}

func TestDecoderErrors(t *testing.T) {
	// Empty decoder
	d := NewDecoder([]byte{})

	_, err := d.ReadByte()
	if err != io.ErrUnexpectedEOF {
		t.Errorf("ReadByte on empty = %v, want io.ErrUnexpectedEOF", err)
	}

	_, err = d.ReadUint16()
	if err != io.ErrUnexpectedEOF {
		t.Errorf("ReadUint16 on empty = %v, want io.ErrUnexpectedEOF", err)
	}

	_, err = d.ReadUint32()
	if err != io.ErrUnexpectedEOF {
		t.Errorf("ReadUint32 on empty = %v, want io.ErrUnexpectedEOF", err)
	}

	_, err = d.ReadUint64()
	if err != io.ErrUnexpectedEOF {
		t.Errorf("ReadUint64 on empty = %v, want io.ErrUnexpectedEOF", err)
	}

	_, err = d.ReadFloat32()
	if err != io.ErrUnexpectedEOF {
		t.Errorf("ReadFloat32 on empty = %v, want io.ErrUnexpectedEOF", err)
	}

	_, err = d.ReadFloat64()
	if err != io.ErrUnexpectedEOF {
		t.Errorf("ReadFloat64 on empty = %v, want io.ErrUnexpectedEOF", err)
	}

	// Short buffer for string
	d = NewDecoder([]byte{10}) // Length 10, but no data
	_, err = d.ReadString()
	if err != io.ErrUnexpectedEOF {
		t.Errorf("ReadString on short = %v, want io.ErrUnexpectedEOF", err)
	}
}

func TestDecoderSkip(t *testing.T) {
	d := NewDecoder([]byte{1, 2, 3, 4, 5})

	if err := d.Skip(2); err != nil {
		t.Errorf("Skip(2) = %v, want nil", err)
	}
	if d.Position() != 2 {
		t.Errorf("Position after Skip(2) = %d, want 2", d.Position())
	}

	b, _ := d.ReadByte()
	if b != 3 {
		t.Errorf("ReadByte after Skip = %d, want 3", b)
	}

	if err := d.Skip(10); err != io.ErrUnexpectedEOF {
		t.Errorf("Skip(10) on short buffer = %v, want io.ErrUnexpectedEOF", err)
	}
}

func TestDecoderRemaining(t *testing.T) {
	d := NewDecoder([]byte{1, 2, 3, 4, 5})

	if d.Remaining() != 5 {
		t.Errorf("Initial Remaining() = %d, want 5", d.Remaining())
	}

	d.ReadByte()
	if d.Remaining() != 4 {
		t.Errorf("Remaining() after ReadByte = %d, want 4", d.Remaining())
	}

	d.ReadBytes(2)
	if d.Remaining() != 2 {
		t.Errorf("Remaining() after ReadBytes(2) = %d, want 2", d.Remaining())
	}
}

func TestEmptyString(t *testing.T) {
	e := NewEncoder()
	e.WriteString("")

	d := NewDecoder(e.Bytes())
	s, err := d.ReadString()
	if err != nil || s != "" {
		t.Errorf("ReadString() = %q, %v; want \"\", nil", s, err)
	}
}

func TestBinaryData(t *testing.T) {
	// Test that binary data with null bytes and special characters works
	original := []byte{0x00, 0xFF, 0x7F, 0x80, 0x01}

	e := NewEncoder()
	e.WriteLenBytes(original)

	d := NewDecoder(e.Bytes())
	decoded, err := d.ReadLenBytes()
	if err != nil {
		t.Errorf("ReadLenBytes() error: %v", err)
	}
	if len(decoded) != len(original) {
		t.Errorf("Length mismatch: got %d, want %d", len(decoded), len(original))
	}
	for i, b := range original {
		if decoded[i] != b {
			t.Errorf("Byte %d mismatch: got %x, want %x", i, decoded[i], b)
		}
	}
}

func TestEdgeCases(t *testing.T) {
	// Max values
	e := NewEncoder()
	e.WriteUint16(math.MaxUint16)
	e.WriteUint32(math.MaxUint32)
	e.WriteUint64(math.MaxUint64)
	e.WriteInt16(math.MaxInt16)
	e.WriteInt16(math.MinInt16)
	e.WriteInt32(math.MaxInt32)
	e.WriteInt32(math.MinInt32)
	e.WriteInt64(math.MaxInt64)
	e.WriteInt64(math.MinInt64)
	e.WriteFloat32(math.MaxFloat32)
	e.WriteFloat32(math.SmallestNonzeroFloat32)
	e.WriteFloat64(math.MaxFloat64)
	e.WriteFloat64(math.SmallestNonzeroFloat64)

	d := NewDecoder(e.Bytes())

	u16, _ := d.ReadUint16()
	if u16 != math.MaxUint16 {
		t.Errorf("MaxUint16: got %d, want %d", u16, uint16(math.MaxUint16))
	}

	u32, _ := d.ReadUint32()
	if u32 != math.MaxUint32 {
		t.Errorf("MaxUint32: got %d, want %d", u32, uint32(math.MaxUint32))
	}

	u64, _ := d.ReadUint64()
	if u64 != math.MaxUint64 {
		t.Errorf("MaxUint64: got %d, want %d", u64, uint64(math.MaxUint64))
	}

	i16max, _ := d.ReadInt16()
	if i16max != math.MaxInt16 {
		t.Errorf("MaxInt16: got %d, want %d", i16max, int16(math.MaxInt16))
	}

	i16min, _ := d.ReadInt16()
	if i16min != math.MinInt16 {
		t.Errorf("MinInt16: got %d, want %d", i16min, int16(math.MinInt16))
	}

	i32max, _ := d.ReadInt32()
	if i32max != math.MaxInt32 {
		t.Errorf("MaxInt32: got %d, want %d", i32max, int32(math.MaxInt32))
	}

	i32min, _ := d.ReadInt32()
	if i32min != math.MinInt32 {
		t.Errorf("MinInt32: got %d, want %d", i32min, int32(math.MinInt32))
	}

	i64max, _ := d.ReadInt64()
	if i64max != math.MaxInt64 {
		t.Errorf("MaxInt64: got %d, want %d", i64max, int64(math.MaxInt64))
	}

	i64min, _ := d.ReadInt64()
	if i64min != math.MinInt64 {
		t.Errorf("MinInt64: got %d, want %d", i64min, int64(math.MinInt64))
	}

	f32max, _ := d.ReadFloat32()
	if f32max != math.MaxFloat32 {
		t.Errorf("MaxFloat32: got %v, want %v", f32max, math.MaxFloat32)
	}

	f32min, _ := d.ReadFloat32()
	if f32min != math.SmallestNonzeroFloat32 {
		t.Errorf("SmallestFloat32: got %v, want %v", f32min, math.SmallestNonzeroFloat32)
	}

	f64max, _ := d.ReadFloat64()
	if f64max != math.MaxFloat64 {
		t.Errorf("MaxFloat64: got %v, want %v", f64max, math.MaxFloat64)
	}

	f64min, _ := d.ReadFloat64()
	if f64min != math.SmallestNonzeroFloat64 {
		t.Errorf("SmallestFloat64: got %v, want %v", f64min, math.SmallestNonzeroFloat64)
	}
}

func BenchmarkEncoderWrite(b *testing.B) {
	e := NewEncoder()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Reset()
		e.WriteByte(0x42)
		e.WriteUvarint(12345)
		e.WriteString("hello world")
		e.WriteUint32(0x12345678)
		e.WriteFloat64(3.14159)
	}
}

func BenchmarkDecoderRead(b *testing.B) {
	e := NewEncoder()
	e.WriteByte(0x42)
	e.WriteUvarint(12345)
	e.WriteString("hello world")
	e.WriteUint32(0x12345678)
	e.WriteFloat64(3.14159)
	data := e.Bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d := NewDecoder(data)
		d.ReadByte()
		d.ReadUvarint()
		d.ReadString()
		d.ReadUint32()
		d.ReadFloat64()
	}
}
