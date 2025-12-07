package protocol

import "math"

// Encoder is a binary encoder that appends data to an internal buffer.
// It is designed for efficient encoding without allocations in the hot path.
type Encoder struct {
	buf []byte
}

// NewEncoder creates a new encoder with a default initial capacity.
func NewEncoder() *Encoder {
	return &Encoder{
		buf: make([]byte, 0, 256),
	}
}

// NewEncoderWithCap creates a new encoder with the specified initial capacity.
func NewEncoderWithCap(cap int) *Encoder {
	return &Encoder{
		buf: make([]byte, 0, cap),
	}
}

// Reset resets the encoder to empty state, reusing the underlying buffer.
func (e *Encoder) Reset() {
	e.buf = e.buf[:0]
}

// Bytes returns the encoded bytes. The returned slice is valid until
// the next call to Reset or any Write method.
func (e *Encoder) Bytes() []byte {
	return e.buf
}

// Len returns the number of bytes currently encoded.
func (e *Encoder) Len() int {
	return len(e.buf)
}

// WriteByte appends a single byte.
// Note: This intentionally doesn't return error (unlike io.ByteWriter)
// because our buffer is unbounded and can always append.
func (e *Encoder) WriteByte(b byte) {
	e.buf = append(e.buf, b)
}

// WriteBytes appends raw bytes.
func (e *Encoder) WriteBytes(b []byte) {
	e.buf = append(e.buf, b...)
}

// WriteUvarint appends an unsigned varint.
func (e *Encoder) WriteUvarint(v uint64) {
	for v >= 0x80 {
		e.buf = append(e.buf, byte(v)|0x80)
		v >>= 7
	}
	e.buf = append(e.buf, byte(v))
}

// WriteSvarint appends a signed varint using ZigZag encoding.
func (e *Encoder) WriteSvarint(v int64) {
	uv := uint64((v << 1) ^ (v >> 63))
	e.WriteUvarint(uv)
}

// WriteString appends a length-prefixed UTF-8 string.
// Format: varint length + string bytes
func (e *Encoder) WriteString(s string) {
	e.WriteUvarint(uint64(len(s)))
	e.buf = append(e.buf, s...)
}

// WriteLenBytes appends length-prefixed bytes.
// Format: varint length + bytes
func (e *Encoder) WriteLenBytes(b []byte) {
	e.WriteUvarint(uint64(len(b)))
	e.buf = append(e.buf, b...)
}

// WriteBool appends a boolean as a single byte (0x00 or 0x01).
func (e *Encoder) WriteBool(b bool) {
	if b {
		e.buf = append(e.buf, 0x01)
	} else {
		e.buf = append(e.buf, 0x00)
	}
}

// WriteUint16 appends a uint16 in big-endian byte order.
func (e *Encoder) WriteUint16(v uint16) {
	e.buf = append(e.buf, byte(v>>8), byte(v))
}

// WriteUint32 appends a uint32 in big-endian byte order.
func (e *Encoder) WriteUint32(v uint32) {
	e.buf = append(e.buf, byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}

// WriteUint64 appends a uint64 in big-endian byte order.
func (e *Encoder) WriteUint64(v uint64) {
	e.buf = append(e.buf,
		byte(v>>56), byte(v>>48), byte(v>>40), byte(v>>32),
		byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}

// WriteInt16 appends an int16 in big-endian byte order.
func (e *Encoder) WriteInt16(v int16) {
	e.WriteUint16(uint16(v))
}

// WriteInt32 appends an int32 in big-endian byte order.
func (e *Encoder) WriteInt32(v int32) {
	e.WriteUint32(uint32(v))
}

// WriteInt64 appends an int64 in big-endian byte order.
func (e *Encoder) WriteInt64(v int64) {
	e.WriteUint64(uint64(v))
}

// WriteFloat32 appends a float32 in IEEE 754 format (big-endian).
func (e *Encoder) WriteFloat32(v float32) {
	e.WriteUint32(math.Float32bits(v))
}

// WriteFloat64 appends a float64 in IEEE 754 format (big-endian).
func (e *Encoder) WriteFloat64(v float64) {
	e.WriteUint64(math.Float64bits(v))
}
