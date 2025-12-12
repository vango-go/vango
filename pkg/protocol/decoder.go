package protocol

import (
	"errors"
	"io"
	"math"
)

// Allocation limits to prevent DoS attacks via malicious length prefixes.
const (
	// DefaultMaxAllocation is the default maximum allocation size (4MB).
	// This is sufficient for normal binary patches and events.
	DefaultMaxAllocation = 4 * 1024 * 1024

	// HardMaxAllocation is the absolute ceiling for allocations (16MB).
	// Even if configured higher, allocations are capped at this limit.
	HardMaxAllocation = 16 * 1024 * 1024

	// MaxCollectionCount is the maximum number of items in a collection (array/map).
	// This prevents OOM from huge counts with small per-item overhead.
	MaxCollectionCount = 100_000
)

// Common decoding errors.
var (
	ErrBufferTooShort     = errors.New("protocol: buffer too short")
	ErrVarintOverflow     = errors.New("protocol: varint overflow")
	ErrInvalidBool        = errors.New("protocol: invalid boolean value")
	ErrAllocationTooLarge = errors.New("protocol: allocation size exceeds limit")
	ErrCollectionTooLarge = errors.New("protocol: collection count exceeds limit")
)

// Decoder is a binary decoder that reads from a byte buffer.
type Decoder struct {
	buf []byte
	pos int
}

// NewDecoder creates a new decoder from the given byte slice.
func NewDecoder(buf []byte) *Decoder {
	return &Decoder{buf: buf}
}

// Remaining returns the number of unread bytes.
func (d *Decoder) Remaining() int {
	return len(d.buf) - d.pos
}

// EOF returns true if all bytes have been read.
func (d *Decoder) EOF() bool {
	return d.pos >= len(d.buf)
}

// Position returns the current read position.
func (d *Decoder) Position() int {
	return d.pos
}

// Skip advances the position by n bytes.
func (d *Decoder) Skip(n int) error {
	if d.pos+n > len(d.buf) {
		return io.ErrUnexpectedEOF
	}
	d.pos += n
	return nil
}

// ReadByte reads a single byte.
func (d *Decoder) ReadByte() (byte, error) {
	if d.pos >= len(d.buf) {
		return 0, io.ErrUnexpectedEOF
	}
	b := d.buf[d.pos]
	d.pos++
	return b, nil
}

// ReadBytes reads exactly n bytes and returns them.
// The returned slice references the decoder's buffer; do not modify.
func (d *Decoder) ReadBytes(n int) ([]byte, error) {
	if d.pos+n > len(d.buf) {
		return nil, io.ErrUnexpectedEOF
	}
	b := d.buf[d.pos : d.pos+n]
	d.pos += n
	return b, nil
}

// ReadUvarint reads an unsigned varint.
func (d *Decoder) ReadUvarint() (uint64, error) {
	var v uint64
	var shift uint

	for {
		if d.pos >= len(d.buf) {
			return 0, io.ErrUnexpectedEOF
		}
		b := d.buf[d.pos]
		d.pos++
		v |= uint64(b&0x7F) << shift
		if b < 0x80 {
			return v, nil
		}
		shift += 7
		if shift >= 64 {
			return 0, ErrVarintOverflow
		}
	}
}

// ReadSvarint reads a signed varint using ZigZag decoding.
func (d *Decoder) ReadSvarint() (int64, error) {
	uv, err := d.ReadUvarint()
	if err != nil {
		return 0, err
	}
	v := int64(uv >> 1)
	if uv&1 != 0 {
		v = ^v
	}
	return v, nil
}

// ReadString reads a length-prefixed UTF-8 string.
// Returns ErrAllocationTooLarge if the string exceeds DefaultMaxAllocation.
func (d *Decoder) ReadString() (string, error) {
	length, err := d.ReadUvarint()
	if err != nil {
		return "", err
	}
	// Bounds check: length must fit in remaining buffer
	if length > uint64(d.Remaining()) {
		return "", io.ErrUnexpectedEOF
	}
	// Allocation limit check: prevent DoS via huge length prefix
	if length > DefaultMaxAllocation {
		return "", ErrAllocationTooLarge
	}
	n := int(length)
	s := string(d.buf[d.pos : d.pos+n])
	d.pos += n
	return s, nil
}

// ReadLenBytes reads length-prefixed bytes.
// Returns a copy of the bytes (safe to retain).
// Returns ErrAllocationTooLarge if the byte slice exceeds DefaultMaxAllocation.
func (d *Decoder) ReadLenBytes() ([]byte, error) {
	length, err := d.ReadUvarint()
	if err != nil {
		return nil, err
	}
	// Bounds check: length must fit in remaining buffer
	if length > uint64(d.Remaining()) {
		return nil, io.ErrUnexpectedEOF
	}
	// Allocation limit check: prevent DoS via huge length prefix
	if length > DefaultMaxAllocation {
		return nil, ErrAllocationTooLarge
	}
	n := int(length)
	b := make([]byte, n)
	copy(b, d.buf[d.pos:d.pos+n])
	d.pos += n
	return b, nil
}

// ReadBool reads a boolean (single byte: 0x00=false, 0x01=true).
func (d *Decoder) ReadBool() (bool, error) {
	b, err := d.ReadByte()
	if err != nil {
		return false, err
	}
	switch b {
	case 0x00:
		return false, nil
	case 0x01:
		return true, nil
	default:
		// Be lenient: any non-zero is true
		return true, nil
	}
}

// ReadUint16 reads a uint16 in big-endian byte order.
func (d *Decoder) ReadUint16() (uint16, error) {
	if d.pos+2 > len(d.buf) {
		return 0, io.ErrUnexpectedEOF
	}
	v := uint16(d.buf[d.pos])<<8 | uint16(d.buf[d.pos+1])
	d.pos += 2
	return v, nil
}

// ReadUint32 reads a uint32 in big-endian byte order.
func (d *Decoder) ReadUint32() (uint32, error) {
	if d.pos+4 > len(d.buf) {
		return 0, io.ErrUnexpectedEOF
	}
	v := uint32(d.buf[d.pos])<<24 | uint32(d.buf[d.pos+1])<<16 |
		uint32(d.buf[d.pos+2])<<8 | uint32(d.buf[d.pos+3])
	d.pos += 4
	return v, nil
}

// ReadUint64 reads a uint64 in big-endian byte order.
func (d *Decoder) ReadUint64() (uint64, error) {
	if d.pos+8 > len(d.buf) {
		return 0, io.ErrUnexpectedEOF
	}
	v := uint64(d.buf[d.pos])<<56 | uint64(d.buf[d.pos+1])<<48 |
		uint64(d.buf[d.pos+2])<<40 | uint64(d.buf[d.pos+3])<<32 |
		uint64(d.buf[d.pos+4])<<24 | uint64(d.buf[d.pos+5])<<16 |
		uint64(d.buf[d.pos+6])<<8 | uint64(d.buf[d.pos+7])
	d.pos += 8
	return v, nil
}

// ReadInt16 reads an int16 in big-endian byte order.
func (d *Decoder) ReadInt16() (int16, error) {
	v, err := d.ReadUint16()
	return int16(v), err
}

// ReadInt32 reads an int32 in big-endian byte order.
func (d *Decoder) ReadInt32() (int32, error) {
	v, err := d.ReadUint32()
	return int32(v), err
}

// ReadInt64 reads an int64 in big-endian byte order.
func (d *Decoder) ReadInt64() (int64, error) {
	v, err := d.ReadUint64()
	return int64(v), err
}

// ReadFloat32 reads a float32 in IEEE 754 format (big-endian).
func (d *Decoder) ReadFloat32() (float32, error) {
	v, err := d.ReadUint32()
	if err != nil {
		return 0, err
	}
	return math.Float32frombits(v), nil
}

// ReadFloat64 reads a float64 in IEEE 754 format (big-endian).
func (d *Decoder) ReadFloat64() (float64, error) {
	v, err := d.ReadUint64()
	if err != nil {
		return 0, err
	}
	return math.Float64frombits(v), nil
}

// ReadCollectionCount reads a varint count and validates it against limits.
// Returns ErrCollectionTooLarge if count exceeds MaxCollectionCount.
// This should be used when reading the size of arrays, maps, or other collections.
func (d *Decoder) ReadCollectionCount() (int, error) {
	count, err := d.ReadUvarint()
	if err != nil {
		return 0, err
	}
	if count > MaxCollectionCount {
		return 0, ErrCollectionTooLarge
	}
	// Also check that count is reasonable for remaining bytes
	// (at minimum 1 byte per item for the smallest possible items)
	if count > uint64(d.Remaining()) {
		return 0, io.ErrUnexpectedEOF
	}
	return int(count), nil
}
