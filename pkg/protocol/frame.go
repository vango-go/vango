package protocol

import (
	"errors"
	"io"
)

// Frame constants.
const (
	// FrameHeaderSize is the size of the frame header in bytes.
	FrameHeaderSize = 4

	// MaxPayloadSize is the maximum payload size (2^16 - 1 bytes).
	MaxPayloadSize = 65535
)

// FrameType identifies the type of frame.
type FrameType uint8

const (
	FrameHandshake FrameType = 0x00 // Connection setup
	FrameEvent     FrameType = 0x01 // Client → Server events
	FramePatches   FrameType = 0x02 // Server → Client patches
	FrameControl   FrameType = 0x03 // Control messages (ping, etc.)
	FrameAck       FrameType = 0x04 // Acknowledgment
	FrameError     FrameType = 0x05 // Error message
)

// String returns the string representation of the frame type.
func (ft FrameType) String() string {
	switch ft {
	case FrameHandshake:
		return "Handshake"
	case FrameEvent:
		return "Event"
	case FramePatches:
		return "Patches"
	case FrameControl:
		return "Control"
	case FrameAck:
		return "Ack"
	case FrameError:
		return "Error"
	default:
		return "Unknown"
	}
}

// FrameFlags are optional flags for frame processing.
type FrameFlags uint8

const (
	FlagCompressed FrameFlags = 0x01 // Payload is gzip compressed
	FlagSequenced  FrameFlags = 0x02 // Includes sequence number
	FlagFinal      FrameFlags = 0x04 // Last frame in batch
	FlagPriority   FrameFlags = 0x08 // High priority (skip queue)
)

// Has returns true if the flags contain the specified flag.
func (ff FrameFlags) Has(flag FrameFlags) bool {
	return ff&flag != 0
}

// Frame errors.
var (
	ErrFrameTooLarge    = errors.New("protocol: frame payload too large")
	ErrInvalidFrameType = errors.New("protocol: invalid frame type")
)

// Frame represents a protocol frame with header and payload.
//
// Wire format (4 bytes header + variable payload):
//
//	┌─────────────┬──────────────┬───────────────────────────────┐
//	│ Frame Type  │ Flags        │ Payload Length                │
//	│ (1 byte)    │ (1 byte)     │ (2 bytes, big-endian)         │
//	└─────────────┴──────────────┴───────────────────────────────┘
//	│                                                             │
//	│  Payload (variable length)                                  │
//	│                                                             │
//	└─────────────────────────────────────────────────────────────┘
type Frame struct {
	Type    FrameType
	Flags   FrameFlags
	Payload []byte
}

// Encode encodes the frame to bytes including the header.
func (f *Frame) Encode() []byte {
	length := len(f.Payload)
	buf := make([]byte, FrameHeaderSize+length)
	buf[0] = byte(f.Type)
	buf[1] = byte(f.Flags)
	buf[2] = byte(length >> 8)
	buf[3] = byte(length)
	copy(buf[FrameHeaderSize:], f.Payload)
	return buf
}

// EncodeTo encodes the frame using the provided encoder.
func (f *Frame) EncodeTo(e *Encoder) {
	e.WriteByte(byte(f.Type))
	e.WriteByte(byte(f.Flags))
	e.WriteUint16(uint16(len(f.Payload)))
	e.WriteBytes(f.Payload)
}

// DecodeFrame decodes a frame from bytes.
// The input must contain at least the header (4 bytes) and full payload.
func DecodeFrame(data []byte) (*Frame, error) {
	if len(data) < FrameHeaderSize {
		return nil, io.ErrUnexpectedEOF
	}

	ft := FrameType(data[0])
	flags := FrameFlags(data[1])
	length := int(data[2])<<8 | int(data[3])

	if len(data) < FrameHeaderSize+length {
		return nil, io.ErrUnexpectedEOF
	}

	payload := make([]byte, length)
	copy(payload, data[FrameHeaderSize:FrameHeaderSize+length])

	return &Frame{
		Type:    ft,
		Flags:   flags,
		Payload: payload,
	}, nil
}

// DecodeFrameHeader decodes just the frame header, returning type, flags, and payload length.
func DecodeFrameHeader(data []byte) (FrameType, FrameFlags, int, error) {
	if len(data) < FrameHeaderSize {
		return 0, 0, 0, io.ErrUnexpectedEOF
	}

	ft := FrameType(data[0])
	flags := FrameFlags(data[1])
	length := int(data[2])<<8 | int(data[3])

	return ft, flags, length, nil
}

// ReadFrame reads a complete frame from an io.Reader.
func ReadFrame(r io.Reader) (*Frame, error) {
	header := make([]byte, FrameHeaderSize)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}

	ft := FrameType(header[0])
	flags := FrameFlags(header[1])
	length := int(header[2])<<8 | int(header[3])

	if length > MaxPayloadSize {
		return nil, ErrFrameTooLarge
	}

	payload := make([]byte, length)
	if length > 0 {
		if _, err := io.ReadFull(r, payload); err != nil {
			return nil, err
		}
	}

	return &Frame{
		Type:    ft,
		Flags:   flags,
		Payload: payload,
	}, nil
}

// WriteFrame writes a complete frame to an io.Writer.
func WriteFrame(w io.Writer, f *Frame) error {
	if len(f.Payload) > MaxPayloadSize {
		return ErrFrameTooLarge
	}

	data := f.Encode()
	_, err := w.Write(data)
	return err
}

// NewFrame creates a new frame with the given type and payload.
func NewFrame(ft FrameType, payload []byte) *Frame {
	return &Frame{
		Type:    ft,
		Flags:   0,
		Payload: payload,
	}
}

// NewFrameWithFlags creates a new frame with the given type, flags, and payload.
func NewFrameWithFlags(ft FrameType, flags FrameFlags, payload []byte) *Frame {
	return &Frame{
		Type:    ft,
		Flags:   flags,
		Payload: payload,
	}
}
