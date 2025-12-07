package protocol

// MaxVarintLen is the maximum number of bytes a varint can occupy.
// A uint64 requires at most 10 bytes in varint encoding.
const MaxVarintLen = 10

// EncodeUvarint encodes an unsigned integer as a varint into buf.
// Returns the number of bytes written.
// buf must have at least MaxVarintLen bytes available.
// Uses protobuf-style encoding: 7 bits of data per byte, MSB indicates continuation.
func EncodeUvarint(buf []byte, v uint64) int {
	i := 0
	for v >= 0x80 {
		buf[i] = byte(v) | 0x80
		v >>= 7
		i++
	}
	buf[i] = byte(v)
	return i + 1
}

// DecodeUvarint decodes an unsigned varint from buf.
// Returns (value, bytesRead). If bytesRead < 0, decoding failed:
//   - -1: buffer too short (incomplete varint)
//   - -2: varint overflow (more than 10 bytes)
func DecodeUvarint(buf []byte) (uint64, int) {
	var v uint64
	var shift uint

	for i, b := range buf {
		if i >= MaxVarintLen {
			return 0, -2 // Overflow
		}
		v |= uint64(b&0x7F) << shift
		if b < 0x80 {
			return v, i + 1
		}
		shift += 7
	}
	return 0, -1 // Incomplete
}

// EncodeSvarint encodes a signed integer as a varint using ZigZag encoding.
// Returns the number of bytes written.
// ZigZag maps signed integers to unsigned: 0->0, -1->1, 1->2, -2->3, 2->4, etc.
func EncodeSvarint(buf []byte, v int64) int {
	// ZigZag encode: (v << 1) ^ (v >> 63)
	// Positive v: v << 1 (0->0, 1->2, 2->4)
	// Negative v: (-v << 1) - 1 (-1->1, -2->3, -3->5)
	uv := uint64((v << 1) ^ (v >> 63))
	return EncodeUvarint(buf, uv)
}

// DecodeSvarint decodes a signed varint using ZigZag decoding.
// Returns (value, bytesRead). Negative bytesRead indicates error (see DecodeUvarint).
func DecodeSvarint(buf []byte) (int64, int) {
	uv, n := DecodeUvarint(buf)
	if n < 0 {
		return 0, n
	}
	// ZigZag decode: (uv >> 1) ^ -(uv & 1)
	v := int64(uv >> 1)
	if uv&1 != 0 {
		v = ^v
	}
	return v, n
}

// UvarintLen returns the number of bytes needed to encode v as a varint.
func UvarintLen(v uint64) int {
	n := 1
	for v >= 0x80 {
		n++
		v >>= 7
	}
	return n
}

// SvarintLen returns the number of bytes needed to encode v as a signed varint.
func SvarintLen(v int64) int {
	uv := uint64((v << 1) ^ (v >> 63))
	return UvarintLen(uv)
}
