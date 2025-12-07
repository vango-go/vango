# Phase 3: Binary Protocol

> **The wire format for events and patches**

---

## Overview

The binary protocol defines how events flow from client to server and patches flow from server to client. It is optimized for minimal bandwidth and fast parsing.

### Goals

1. **Minimal size**: Typical event < 10 bytes, typical patch < 20 bytes
2. **Fast encoding/decoding**: No reflection, direct byte manipulation
3. **Reliable delivery**: Sequence numbers, acknowledgments
4. **Reconnection**: Resync capability after disconnect
5. **Extensible**: Version negotiation, reserved opcodes

### Non-Goals

1. Human readability (use devtools for debugging)
2. Backwards compatibility with V1 (clean break)
3. JSON fallback (binary only)

---

## Message Framing

All messages are prefixed with a frame header:

```
┌────────────────────────────────────────────────────────────┐
│  Frame Header (4 bytes)                                     │
├─────────────┬──────────────┬───────────────────────────────┤
│ Frame Type  │ Flags        │ Payload Length                │
│ (1 byte)    │ (1 byte)     │ (2 bytes, big-endian)         │
└─────────────┴──────────────┴───────────────────────────────┘
│                                                             │
│  Payload (variable length)                                  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Frame Types

```go
type FrameType uint8

const (
    FrameHandshake FrameType = 0x00  // Connection setup
    FrameEvent     FrameType = 0x01  // Client → Server events
    FramePatches   FrameType = 0x02  // Server → Client patches
    FrameControl   FrameType = 0x03  // Control messages (ping, etc.)
    FrameAck       FrameType = 0x04  // Acknowledgment
    FrameError     FrameType = 0x05  // Error message
)
```

### Frame Flags

```go
type FrameFlags uint8

const (
    FlagCompressed FrameFlags = 0x01  // Payload is gzip compressed
    FlagSequenced  FrameFlags = 0x02  // Includes sequence number
    FlagFinal      FrameFlags = 0x04  // Last frame in batch
    FlagPriority   FrameFlags = 0x08  // High priority (skip queue)
)
```

---

## Handshake Protocol

### Client Hello

Sent by client after WebSocket connection established.

```
┌────────────────────────────────────────────────────────────┐
│  Client Hello                                               │
├────────────┬────────────────────────────────────────────────┤
│ Version    │ Protocol version (2 bytes, major.minor)        │
│ CSRF Token │ Length-prefixed string                         │
│ Session ID │ Length-prefixed string (empty if new)          │
│ Last Seq   │ Last seen sequence number (4 bytes)            │
│ Viewport   │ Width (2 bytes) + Height (2 bytes)             │
│ Timezone   │ Offset in minutes (2 bytes, signed)            │
└────────────┴────────────────────────────────────────────────┘
```

### Server Hello

Server response to Client Hello.

```
┌────────────────────────────────────────────────────────────┐
│  Server Hello                                               │
├────────────┬────────────────────────────────────────────────┤
│ Status     │ 0x00 = OK, 0x01 = Version mismatch, etc.       │
│ Session ID │ Length-prefixed string                         │
│ Next Seq   │ Next expected sequence number (4 bytes)        │
│ Server Time│ Unix timestamp in ms (8 bytes)                 │
│ Flags      │ Server capabilities (2 bytes)                  │
└────────────┴────────────────────────────────────────────────┘
```

### Handshake Status Codes

```go
const (
    HandshakeOK              = 0x00
    HandshakeVersionMismatch = 0x01
    HandshakeInvalidCSRF     = 0x02
    HandshakeSessionExpired  = 0x03
    HandshakeServerBusy      = 0x04
    HandshakeUpgradeRequired = 0x05
)
```

---

## Varint Encoding

Variable-length integers for compact encoding of small numbers.

### Unsigned Varint

```go
// Encode unsigned varint (protobuf style)
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

// Decode unsigned varint
func DecodeUvarint(buf []byte) (uint64, int) {
    var v uint64
    var shift uint
    for i, b := range buf {
        v |= uint64(b&0x7F) << shift
        if b < 0x80 {
            return v, i + 1
        }
        shift += 7
        if shift >= 64 {
            return 0, -1 // Overflow
        }
    }
    return 0, -1 // Incomplete
}
```

### Signed Varint (ZigZag)

```go
// Encode signed varint using zigzag encoding
func EncodeSvarint(buf []byte, v int64) int {
    uv := uint64((v << 1) ^ (v >> 63)) // ZigZag encode
    return EncodeUvarint(buf, uv)
}

// Decode signed varint
func DecodeSvarint(buf []byte) (int64, int) {
    uv, n := DecodeUvarint(buf)
    if n < 0 {
        return 0, n
    }
    v := int64(uv >> 1)
    if uv&1 != 0 {
        v = ^v
    }
    return v, n
}
```

### Size Examples

| Value | Bytes | Encoding |
|-------|-------|----------|
| 0 | 1 | `00` |
| 127 | 1 | `7F` |
| 128 | 2 | `80 01` |
| 16383 | 2 | `FF 7F` |
| 16384 | 3 | `80 80 01` |

---

## Event Encoding

### Event Types

```go
type EventType uint8

const (
    EventClick       EventType = 0x01
    EventDblClick    EventType = 0x02
    EventMouseDown   EventType = 0x03
    EventMouseUp     EventType = 0x04
    EventMouseMove   EventType = 0x05
    EventMouseEnter  EventType = 0x06
    EventMouseLeave  EventType = 0x07
    EventInput       EventType = 0x10
    EventChange      EventType = 0x11
    EventSubmit      EventType = 0x12
    EventFocus       EventType = 0x13
    EventBlur        EventType = 0x14
    EventKeyDown     EventType = 0x20
    EventKeyUp       EventType = 0x21
    EventKeyPress    EventType = 0x22
    EventScroll      EventType = 0x30
    EventResize      EventType = 0x31
    EventTouchStart  EventType = 0x40
    EventTouchMove   EventType = 0x41
    EventTouchEnd    EventType = 0x42
    EventDragStart   EventType = 0x50
    EventDragEnd     EventType = 0x51
    EventDrop        EventType = 0x52
    EventHook        EventType = 0x60  // Client hook event
    EventNavigate    EventType = 0x70  // Navigation request
    EventCustom      EventType = 0xFF  // Custom event
)
```

### Event Format

```
┌────────────────────────────────────────────────────────────┐
│  Event Message                                              │
├────────────┬────────────────────────────────────────────────┤
│ Sequence   │ Sequence number (varint)                       │
│ Event Type │ EventType (1 byte)                             │
│ HID        │ Hydration ID (length-prefixed string)          │
│ Payload    │ Type-specific data (variable)                  │
└────────────┴────────────────────────────────────────────────┘
```

### Event Payloads

#### Click Events (no payload)

```
[empty]
```

#### Input/Change Events

```
┌────────────┬────────────────────────────────────────────────┐
│ Value      │ Length-prefixed UTF-8 string                   │
└────────────┴────────────────────────────────────────────────┘
```

#### Submit Event

```
┌────────────┬────────────────────────────────────────────────┐
│ Field Count│ Number of fields (varint)                      │
├────────────┼────────────────────────────────────────────────┤
│ For each:  │                                                │
│  Key       │ Length-prefixed string                         │
│  Value     │ Length-prefixed string                         │
└────────────┴────────────────────────────────────────────────┘
```

#### Keyboard Events

```
┌────────────┬────────────────────────────────────────────────┐
│ Key Code   │ Key string (length-prefixed)                   │
│ Modifiers  │ Bitmask (1 byte)                               │
│            │   0x01 = Ctrl                                  │
│            │   0x02 = Shift                                 │
│            │   0x04 = Alt                                   │
│            │   0x08 = Meta                                  │
└────────────┴────────────────────────────────────────────────┘
```

#### Mouse Events (with coordinates)

```
┌────────────┬────────────────────────────────────────────────┐
│ ClientX    │ X coordinate (svarint)                         │
│ ClientY    │ Y coordinate (svarint)                         │
│ Button     │ Mouse button (1 byte)                          │
│ Modifiers  │ Bitmask (1 byte)                               │
└────────────┴────────────────────────────────────────────────┘
```

#### Scroll Events

```
┌────────────┬────────────────────────────────────────────────┐
│ ScrollTop  │ Scroll top position (svarint)                  │
│ ScrollLeft │ Scroll left position (svarint)                 │
└────────────┴────────────────────────────────────────────────┘
```

#### Hook Events

```
┌────────────┬────────────────────────────────────────────────┐
│ Event Name │ Length-prefixed string                         │
│ Data Count │ Number of key-value pairs (varint)             │
├────────────┼────────────────────────────────────────────────┤
│ For each:  │                                                │
│  Key       │ Length-prefixed string                         │
│  Type      │ Value type (1 byte)                            │
│  Value     │ Type-encoded value                             │
└────────────┴────────────────────────────────────────────────┘

Value Types:
  0x00 = Null
  0x01 = Boolean (1 byte)
  0x02 = Integer (svarint)
  0x03 = Float (8 bytes, IEEE 754)
  0x04 = String (length-prefixed)
  0x05 = Array (count + values)
```

#### Navigate Event

```
┌────────────┬────────────────────────────────────────────────┐
│ Path       │ Length-prefixed string                         │
│ Replace    │ Boolean (1 byte) - replace vs push             │
└────────────┴────────────────────────────────────────────────┘
```

---

## Patch Encoding

### Patch Types

```go
type PatchOp uint8

const (
    PatchSetText     PatchOp = 0x01  // Set text content
    PatchSetAttr     PatchOp = 0x02  // Set attribute
    PatchRemoveAttr  PatchOp = 0x03  // Remove attribute
    PatchInsertNode  PatchOp = 0x04  // Insert new node
    PatchRemoveNode  PatchOp = 0x05  // Remove node
    PatchMoveNode    PatchOp = 0x06  // Move node
    PatchReplaceNode PatchOp = 0x07  // Replace entire node
    PatchSetValue    PatchOp = 0x08  // Set input value
    PatchSetChecked  PatchOp = 0x09  // Set checkbox state
    PatchSetSelected PatchOp = 0x0A  // Set selected state
    PatchFocus       PatchOp = 0x0B  // Focus element
    PatchBlur        PatchOp = 0x0C  // Blur element
    PatchScrollTo    PatchOp = 0x0D  // Scroll to position
    PatchAddClass    PatchOp = 0x10  // Add CSS class
    PatchRemoveClass PatchOp = 0x11  // Remove CSS class
    PatchToggleClass PatchOp = 0x12  // Toggle CSS class
    PatchSetStyle    PatchOp = 0x13  // Set style property
    PatchRemoveStyle PatchOp = 0x14  // Remove style property
    PatchSetData     PatchOp = 0x15  // Set data attribute
    PatchDispatch    PatchOp = 0x20  // Dispatch client event
    PatchEval        PatchOp = 0x21  // Eval JS (use sparingly!)
)
```

### Patches Frame

```
┌────────────────────────────────────────────────────────────┐
│  Patches Frame                                              │
├────────────┬────────────────────────────────────────────────┤
│ Sequence   │ Sequence number (varint)                       │
│ Patch Count│ Number of patches (varint)                     │
├────────────┼────────────────────────────────────────────────┤
│ Patches    │ Array of encoded patches                       │
└────────────┴────────────────────────────────────────────────┘
```

### Patch Format

```
┌────────────┬────────────────────────────────────────────────┐
│ Op         │ PatchOp (1 byte)                               │
│ HID        │ Target hydration ID (length-prefixed)          │
│ Payload    │ Op-specific data                               │
└────────────┴────────────────────────────────────────────────┘
```

### Patch Payloads

#### PatchSetText

```
┌────────────┬────────────────────────────────────────────────┐
│ Text       │ Length-prefixed UTF-8 string                   │
└────────────┴────────────────────────────────────────────────┘
```

#### PatchSetAttr / PatchRemoveAttr

```
┌────────────┬────────────────────────────────────────────────┐
│ Key        │ Attribute name (length-prefixed)               │
│ Value      │ Attribute value (length-prefixed) [SetAttr]    │
└────────────┴────────────────────────────────────────────────┘
```

#### PatchInsertNode

```
┌────────────┬────────────────────────────────────────────────┐
│ Parent HID │ Parent's hydration ID (length-prefixed)        │
│ Index      │ Insert position (varint)                       │
│ Node       │ Encoded VNode (see below)                      │
└────────────┴────────────────────────────────────────────────┘
```

#### PatchRemoveNode

```
[empty - HID is sufficient]
```

#### PatchMoveNode

```
┌────────────┬────────────────────────────────────────────────┐
│ Parent HID │ New parent's hydration ID (length-prefixed)    │
│ Index      │ New position (varint)                          │
└────────────┴────────────────────────────────────────────────┘
```

#### PatchReplaceNode

```
┌────────────┬────────────────────────────────────────────────┐
│ Node       │ Encoded VNode (see below)                      │
└────────────┴────────────────────────────────────────────────┘
```

#### PatchSetValue

```
┌────────────┬────────────────────────────────────────────────┐
│ Value      │ Input value (length-prefixed)                  │
└────────────┴────────────────────────────────────────────────┘
```

#### PatchSetChecked / PatchSetSelected

```
┌────────────┬────────────────────────────────────────────────┐
│ State      │ Boolean (1 byte)                               │
└────────────┴────────────────────────────────────────────────┘
```

#### PatchScrollTo

```
┌────────────┬────────────────────────────────────────────────┐
│ X          │ Scroll X position (svarint)                    │
│ Y          │ Scroll Y position (svarint)                    │
│ Behavior   │ 0=instant, 1=smooth (1 byte)                   │
└────────────┴────────────────────────────────────────────────┘
```

#### PatchAddClass / PatchRemoveClass

```
┌────────────┬────────────────────────────────────────────────┐
│ Class      │ Class name (length-prefixed)                   │
└────────────┴────────────────────────────────────────────────┘
```

#### PatchSetStyle / PatchRemoveStyle

```
┌────────────┬────────────────────────────────────────────────┐
│ Property   │ CSS property name (length-prefixed)            │
│ Value      │ CSS value (length-prefixed) [SetStyle only]    │
└────────────┴────────────────────────────────────────────────┘
```

---

## VNode Encoding

For InsertNode and ReplaceNode patches.

### VNode Format

```
┌────────────┬────────────────────────────────────────────────┐
│ Kind       │ VKind (1 byte)                                 │
│ Payload    │ Kind-specific data                             │
└────────────┴────────────────────────────────────────────────┘
```

### Element Node

```
┌────────────┬────────────────────────────────────────────────┐
│ Kind       │ 0x01 (KindElement)                             │
│ Tag        │ Tag name (length-prefixed)                     │
│ HID        │ Hydration ID (length-prefixed, may be empty)   │
│ Attr Count │ Number of attributes (varint)                  │
├────────────┼────────────────────────────────────────────────┤
│ Attributes │ For each attribute:                            │
│            │   Key (length-prefixed)                        │
│            │   Value (length-prefixed)                      │
├────────────┼────────────────────────────────────────────────┤
│ Child Count│ Number of children (varint)                    │
│ Children   │ Recursively encoded VNodes                     │
└────────────┴────────────────────────────────────────────────┘
```

### Text Node

```
┌────────────┬────────────────────────────────────────────────┐
│ Kind       │ 0x02 (KindText)                                │
│ Text       │ Text content (length-prefixed)                 │
└────────────┴────────────────────────────────────────────────┘
```

### Fragment Node

```
┌────────────┬────────────────────────────────────────────────┐
│ Kind       │ 0x03 (KindFragment)                            │
│ Child Count│ Number of children (varint)                    │
│ Children   │ Recursively encoded VNodes                     │
└────────────┴────────────────────────────────────────────────┘
```

### Raw HTML Node

```
┌────────────┬────────────────────────────────────────────────┐
│ Kind       │ 0x04 (KindRaw)                                 │
│ HTML       │ Raw HTML content (length-prefixed)             │
└────────────┴────────────────────────────────────────────────┘
```

---

## Control Messages

### Ping/Pong (Heartbeat)

```
┌────────────┬────────────────────────────────────────────────┐
│ Subtype    │ 0x01 = Ping, 0x02 = Pong                       │
│ Timestamp  │ Unix timestamp in ms (8 bytes)                 │
└────────────┴────────────────────────────────────────────────┘
```

### Resync Request

Sent by client after reconnect to request missed patches.

```
┌────────────┬────────────────────────────────────────────────┐
│ Subtype    │ 0x10 = Resync Request                          │
│ Last Seq   │ Last received sequence number (varint)         │
└────────────┴────────────────────────────────────────────────┘
```

### Resync Response

Server sends missed patches or full state.

```
┌────────────┬────────────────────────────────────────────────┐
│ Subtype    │ 0x11 = Resync Patches, 0x12 = Full Reload      │
│ From Seq   │ Starting sequence number (varint)              │
│ Patches    │ Array of patches (if 0x11)                     │
│ HTML       │ Full HTML to replace (if 0x12)                 │
└────────────┴────────────────────────────────────────────────┘
```

### Close Session

```
┌────────────┬────────────────────────────────────────────────┐
│ Subtype    │ 0x20 = Close                                   │
│ Reason     │ Close reason code (1 byte)                     │
│ Message    │ Optional message (length-prefixed)             │
└────────────┴────────────────────────────────────────────────┘
```

---

## Acknowledgment

### Ack Frame

Sent periodically by client to acknowledge received patches.

```
┌────────────┬────────────────────────────────────────────────┐
│ Last Seq   │ Last received sequence number (varint)         │
│ Window     │ Receive window size (varint)                   │
└────────────┴────────────────────────────────────────────────┘
```

Server uses this for:
1. Garbage collection of patch history
2. Flow control (don't send faster than client can process)
3. Detecting client lag

---

## Error Messages

```
┌────────────┬────────────────────────────────────────────────┐
│ Code       │ Error code (2 bytes)                           │
│ Message    │ Error message (length-prefixed)                │
│ Fatal      │ Boolean - should close connection? (1 byte)    │
└────────────┴────────────────────────────────────────────────┘
```

### Error Codes

```go
const (
    ErrUnknown         ErrorCode = 0x0000
    ErrInvalidFrame    ErrorCode = 0x0001
    ErrInvalidEvent    ErrorCode = 0x0002
    ErrHandlerNotFound ErrorCode = 0x0003
    ErrHandlerPanic    ErrorCode = 0x0004
    ErrSessionExpired  ErrorCode = 0x0005
    ErrRateLimited     ErrorCode = 0x0006
    ErrNotAuthorized   ErrorCode = 0x0007
    ErrNotFound        ErrorCode = 0x0008
    ErrValidation      ErrorCode = 0x0009
    ErrServerError     ErrorCode = 0x0100
)
```

---

## Implementation

### Encoder

```go
type Encoder struct {
    buf []byte
}

func NewEncoder() *Encoder {
    return &Encoder{
        buf: make([]byte, 0, 256),
    }
}

func NewEncoderWithCap(cap int) *Encoder {
    return &Encoder{
        buf: make([]byte, 0, cap),
    }
}

func (e *Encoder) Reset() {
    e.buf = e.buf[:0]
}

func (e *Encoder) Bytes() []byte {
    return e.buf
}

// Write raw bytes
func (e *Encoder) WriteBytes(b []byte) {
    e.buf = append(e.buf, b...)
}

// Write single byte
func (e *Encoder) WriteByte(b byte) {
    e.buf = append(e.buf, b)
}

// Write unsigned varint
func (e *Encoder) WriteUvarint(v uint64) {
    for v >= 0x80 {
        e.buf = append(e.buf, byte(v)|0x80)
        v >>= 7
    }
    e.buf = append(e.buf, byte(v))
}

// Write signed varint (zigzag)
func (e *Encoder) WriteSvarint(v int64) {
    uv := uint64((v << 1) ^ (v >> 63))
    e.WriteUvarint(uv)
}

// Write length-prefixed string
func (e *Encoder) WriteString(s string) {
    e.WriteUvarint(uint64(len(s)))
    e.buf = append(e.buf, s...)
}

// Write length-prefixed bytes
func (e *Encoder) WriteLenBytes(b []byte) {
    e.WriteUvarint(uint64(len(b)))
    e.buf = append(e.buf, b...)
}

// Write boolean
func (e *Encoder) WriteBool(b bool) {
    if b {
        e.WriteByte(1)
    } else {
        e.WriteByte(0)
    }
}

// Write uint16 (big-endian)
func (e *Encoder) WriteUint16(v uint16) {
    e.buf = append(e.buf, byte(v>>8), byte(v))
}

// Write uint32 (big-endian)
func (e *Encoder) WriteUint32(v uint32) {
    e.buf = append(e.buf, byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}

// Write uint64 (big-endian)
func (e *Encoder) WriteUint64(v uint64) {
    e.buf = append(e.buf,
        byte(v>>56), byte(v>>48), byte(v>>40), byte(v>>32),
        byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}

// Write float64 (IEEE 754)
func (e *Encoder) WriteFloat64(v float64) {
    e.WriteUint64(math.Float64bits(v))
}
```

### Decoder

```go
type Decoder struct {
    buf []byte
    pos int
}

func NewDecoder(buf []byte) *Decoder {
    return &Decoder{buf: buf}
}

func (d *Decoder) Remaining() int {
    return len(d.buf) - d.pos
}

func (d *Decoder) EOF() bool {
    return d.pos >= len(d.buf)
}

// Read single byte
func (d *Decoder) ReadByte() (byte, error) {
    if d.pos >= len(d.buf) {
        return 0, io.ErrUnexpectedEOF
    }
    b := d.buf[d.pos]
    d.pos++
    return b, nil
}

// Read unsigned varint
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
            return 0, errors.New("varint overflow")
        }
    }
}

// Read signed varint (zigzag)
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

// Read length-prefixed string
func (d *Decoder) ReadString() (string, error) {
    length, err := d.ReadUvarint()
    if err != nil {
        return "", err
    }
    if d.pos+int(length) > len(d.buf) {
        return "", io.ErrUnexpectedEOF
    }
    s := string(d.buf[d.pos : d.pos+int(length)])
    d.pos += int(length)
    return s, nil
}

// Read length-prefixed bytes
func (d *Decoder) ReadLenBytes() ([]byte, error) {
    length, err := d.ReadUvarint()
    if err != nil {
        return nil, err
    }
    if d.pos+int(length) > len(d.buf) {
        return nil, io.ErrUnexpectedEOF
    }
    b := d.buf[d.pos : d.pos+int(length)]
    d.pos += int(length)
    return b, nil
}

// Read exactly n bytes
func (d *Decoder) ReadBytes(n int) ([]byte, error) {
    if d.pos+n > len(d.buf) {
        return nil, io.ErrUnexpectedEOF
    }
    b := d.buf[d.pos : d.pos+n]
    d.pos += n
    return b, nil
}

// Read boolean
func (d *Decoder) ReadBool() (bool, error) {
    b, err := d.ReadByte()
    return b != 0, err
}

// Read uint16 (big-endian)
func (d *Decoder) ReadUint16() (uint16, error) {
    if d.pos+2 > len(d.buf) {
        return 0, io.ErrUnexpectedEOF
    }
    v := uint16(d.buf[d.pos])<<8 | uint16(d.buf[d.pos+1])
    d.pos += 2
    return v, nil
}

// Read uint32 (big-endian)
func (d *Decoder) ReadUint32() (uint32, error) {
    if d.pos+4 > len(d.buf) {
        return 0, io.ErrUnexpectedEOF
    }
    v := uint32(d.buf[d.pos])<<24 | uint32(d.buf[d.pos+1])<<16 |
        uint32(d.buf[d.pos+2])<<8 | uint32(d.buf[d.pos+3])
    d.pos += 4
    return v, nil
}

// Read uint64 (big-endian)
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

// Read float64
func (d *Decoder) ReadFloat64() (float64, error) {
    v, err := d.ReadUint64()
    if err != nil {
        return 0, err
    }
    return math.Float64frombits(v), nil
}
```

### Event Encoding

```go
// Event represents a client event.
type Event struct {
    Seq     uint64      // Sequence number
    Type    EventType   // Event type
    HID     string      // Target element's hydration ID
    Payload any         // Type-specific payload (nil, string, or typed struct)
}

// Additional payload types beyond those in the spec:
type ResizeEventData struct {
    Width  int
    Height int
}

type TouchPoint struct {
    ID      int
    ClientX int
    ClientY int
}

type TouchEventData struct {
    Touches []TouchPoint
}

type DragEventData struct {
    ClientX int
    ClientY int
}

func EncodeEvent(event *Event) []byte {
    e := NewEncoder()
    EncodeEventTo(e, event)
    return e.Bytes()
}

func EncodeEventTo(e *Encoder, event *Event) {
    e.WriteUvarint(event.Seq)
    e.WriteByte(byte(event.Type))
    e.WriteString(event.HID)

    switch event.Type {
    case EventClick, EventDblClick, EventFocus, EventBlur:
        // No payload

    case EventInput, EventChange:
        e.WriteString(event.Payload.(string))

    case EventSubmit:
        data := event.Payload.(*SubmitEventData)
        e.WriteUvarint(uint64(len(data.Fields)))
        for k, v := range data.Fields {
            e.WriteString(k)
            e.WriteString(v)
        }

    case EventKeyDown, EventKeyUp, EventKeyPress:
        ev := event.Payload.(*KeyboardEventData)
        e.WriteString(ev.Key)
        e.WriteByte(byte(ev.Modifiers))

    case EventMouseMove, EventMouseDown, EventMouseUp:
        ev := event.Payload.(*MouseEventData)
        e.WriteSvarint(int64(ev.ClientX))
        e.WriteSvarint(int64(ev.ClientY))
        e.WriteByte(ev.Button)
        e.WriteByte(byte(ev.Modifiers))

    case EventScroll:
        ev := event.Payload.(*ScrollEventData)
        e.WriteSvarint(int64(ev.ScrollTop))
        e.WriteSvarint(int64(ev.ScrollLeft))

    case EventResize:
        ev := event.Payload.(*ResizeEventData)
        e.WriteSvarint(int64(ev.Width))
        e.WriteSvarint(int64(ev.Height))

    case EventTouchStart, EventTouchMove, EventTouchEnd:
        ev := event.Payload.(*TouchEventData)
        e.WriteUvarint(uint64(len(ev.Touches)))
        for _, t := range ev.Touches {
            e.WriteSvarint(int64(t.ID))
            e.WriteSvarint(int64(t.ClientX))
            e.WriteSvarint(int64(t.ClientY))
        }

    case EventHook:
        ev := event.Payload.(*HookEventData)
        e.WriteString(ev.Name)
        encodeHookData(e, ev.Data)

    case EventNavigate:
        ev := event.Payload.(*NavigateEventData)
        e.WriteString(ev.Path)
        e.WriteBool(ev.Replace)

    case EventCustom:
        ev := event.Payload.(*CustomEventData)
        e.WriteString(ev.Name)
        e.WriteString(ev.Data)
    }
}
```

### Patch Encoding

```go
// Patch represents a single DOM operation.
type Patch struct {
    Op       PatchOp        // Operation type
    HID      string         // Target element's hydration ID
    Key      string         // Attribute/style/class key
    Value    string         // Value for text/attr/style/class
    ParentID string         // Parent HID for InsertNode/MoveNode
    Index    int            // Insert/Move position
    Node     *VNodeWire     // For InsertNode/ReplaceNode
    Bool     bool           // For SetChecked/SetSelected
    X        int            // For ScrollTo
    Y        int            // For ScrollTo
    Behavior ScrollBehavior // For ScrollTo (0=instant, 1=smooth)
}

// PatchesFrame represents a batch of patches with sequence number.
type PatchesFrame struct {
    Seq     uint64
    Patches []Patch
}

func EncodePatches(pf *PatchesFrame) []byte {
    e := NewEncoder()
    EncodePatchesTo(e, pf)
    return e.Bytes()
}

func EncodePatchesTo(e *Encoder, pf *PatchesFrame) {
    e.WriteUvarint(pf.Seq)
    e.WriteUvarint(uint64(len(pf.Patches)))

    for i := range pf.Patches {
        p := &pf.Patches[i]
        e.WriteByte(byte(p.Op))
        e.WriteString(p.HID)

        switch p.Op {
        case PatchSetText:
            e.WriteString(p.Value)

        case PatchSetAttr:
            e.WriteString(p.Key)
            e.WriteString(p.Value)

        case PatchRemoveAttr:
            e.WriteString(p.Key)

        case PatchInsertNode:
            e.WriteString(p.ParentID)
            e.WriteUvarint(uint64(p.Index))
            EncodeVNodeWire(e, p.Node)

        case PatchRemoveNode:
            // No additional data

        case PatchMoveNode:
            e.WriteString(p.ParentID)
            e.WriteUvarint(uint64(p.Index))

        case PatchReplaceNode:
            EncodeVNodeWire(e, p.Node)

        case PatchSetValue:
            e.WriteString(p.Value)

        case PatchSetChecked, PatchSetSelected:
            e.WriteBool(p.Bool)

        case PatchFocus, PatchBlur:
            // No additional data

        case PatchScrollTo:
            e.WriteSvarint(int64(p.X))
            e.WriteSvarint(int64(p.Y))
            e.WriteByte(byte(p.Behavior))

        case PatchAddClass, PatchRemoveClass, PatchToggleClass:
            e.WriteString(p.Value)

        case PatchSetStyle:
            e.WriteString(p.Key)
            e.WriteString(p.Value)

        case PatchRemoveStyle:
            e.WriteString(p.Key)

        case PatchSetData:
            e.WriteString(p.Key)
            e.WriteString(p.Value)

        case PatchDispatch:
            e.WriteString(p.Value) // Event name

        case PatchEval:
            e.WriteString(p.Value) // JS code
        }
    }
}

// VNodeWire is the wire format for VNodes.
// It contains only serializable data (no event handlers or components).
type VNodeWire struct {
    Kind     vdom.VKind        // Node type
    Tag      string            // Element tag name
    HID      string            // Hydration ID
    Attrs    map[string]string // String attributes only (no handlers)
    Children []*VNodeWire      // Child nodes
    Text     string            // For Text and Raw nodes
}

// VNodeToWire converts a vdom.VNode to wire format.
// Event handlers are stripped; only string attributes are included.
func VNodeToWire(node *vdom.VNode) *VNodeWire

// EncodeVNodeWire encodes a VNodeWire to the encoder.
func EncodeVNodeWire(e *Encoder, node *VNodeWire) {
    if node == nil {
        e.WriteByte(0) // Nil marker
        return
    }
    e.WriteByte(byte(node.Kind))

    switch node.Kind {
    case vdom.KindElement:
        e.WriteString(node.Tag)
        e.WriteString(node.HID)
        e.WriteUvarint(uint64(len(node.Attrs)))
        for k, v := range node.Attrs {
            e.WriteString(k)
            e.WriteString(v)
        }
        e.WriteUvarint(uint64(len(node.Children)))
        for _, child := range node.Children {
            EncodeVNodeWire(e, child)
        }

    case vdom.KindText:
        e.WriteString(node.Text)

    case vdom.KindFragment:
        e.WriteUvarint(uint64(len(node.Children)))
        for _, child := range node.Children {
            EncodeVNodeWire(e, child)
        }

    case vdom.KindRaw:
        e.WriteString(node.Text)
    }
}

// DecodeVNodeWire decodes a VNodeWire from the decoder.
func DecodeVNodeWire(d *Decoder) (*VNodeWire, error)
```

---

## Size Comparisons

### Example: Click Event

**Binary (Vango V2):**
```
01          # Event type: Click
02 68 31    # HID: "h1" (length 2)
```
**Total: 4 bytes**

**JSON equivalent:**
```json
{"type":"click","hid":"h1"}
```
**Total: 26 bytes**

**Savings: 85%**

### Example: SetText Patch

**Binary (Vango V2):**
```
01          # Patch op: SetText
02 68 31    # HID: "h1"
0C 48 65 6C 6C 6F 2C 20 77 6F 72 6C 64  # Text: "Hello, world"
```
**Total: 16 bytes**

**JSON equivalent:**
```json
{"op":"setText","hid":"h1","value":"Hello, world"}
```
**Total: 46 bytes**

**Savings: 65%**

---

## Benchmark Results

Performance benchmarks run on Apple M-series (arm64). All targets exceeded.

### Core Operations

| Operation | Target | Actual | Notes |
|-----------|--------|--------|-------|
| Event encode (click) | < 500ns | **~52ns** | 10x faster than target |
| Event decode (click) | < 500ns | **~26ns** | 20x faster than target |
| Event encode (input) | < 500ns | **~52ns** | With string payload |
| Event decode (input) | < 500ns | **~48ns** | With string payload |
| Event encode (submit) | < 500ns | **~86ns** | With form fields map |
| Event decode (submit) | < 500ns | **~168ns** | With form fields map |
| Patch encode (SetText) | < 500ns | **~53ns** | 10x faster than target |
| Patch decode (SetText) | < 500ns | **~56ns** | 10x faster than target |

### Batch Operations

| Operation | Target | Actual | Notes |
|-----------|--------|--------|-------|
| 10 patches encode | < 5μs | **~94ns** | 50x faster than target |
| 10 patches decode | < 5μs | **~291ns** | 17x faster than target |
| 100 patches encode | < 50μs | **~1.1μs** | 45x faster than target |
| 100 patches decode | < 50μs | **~2.4μs** | 20x faster than target |

### Low-Level Operations

| Operation | Actual | Allocations |
|-----------|--------|-------------|
| Varint encode (small) | ~0.23ns | 0 |
| Varint encode (large) | ~1.7ns | 0 |
| Varint decode (small) | ~0.47ns | 0 |
| Varint decode (large) | ~1.5ns | 0 |
| Encoder mixed types | ~6.5ns | 0 |
| Decoder mixed types | ~16ns | 1 (string) |
| Frame encode (small) | ~8ns | 1 |
| Frame decode (small) | ~7ns | 1 |

### Protocol Messages

| Operation | Actual | Allocations |
|-----------|--------|-------------|
| ClientHello encode | ~52ns | 1 |
| ClientHello decode | ~40ns | 3 |
| Ping encode | ~49ns | 1 |
| Ping decode | ~9ns | 1 |
| Ack encode | ~47ns | 1 |
| Ack decode | ~11ns | 1 |
| Error encode | ~50ns | 1 |
| Error decode | ~27ns | 2 |

### VNode Serialization

| Operation | Actual | Notes |
|-----------|--------|-------|
| VNodeToWire (simple) | ~211ns | Converts vdom.VNode to wire format |
| VNodeWire encode (simple) | ~38ns | Div with text child |
| VNodeWire decode (simple) | ~164ns | Div with text child |
| VNodeWire encode (deep) | ~149ns | 4-level nested divs |

**Benchmark command:** `go test ./pkg/protocol/... -bench=. -benchmem`

---

## Testing Strategy

### Unit Tests

```go
func TestVarintRoundTrip(t *testing.T) {
    values := []uint64{0, 1, 127, 128, 16383, 16384, math.MaxUint64}

    for _, v := range values {
        t.Run(fmt.Sprintf("value_%d", v), func(t *testing.T) {
            e := NewEncoder()
            e.WriteUvarint(v)

            d := NewDecoder(e.Bytes())
            got, err := d.ReadUvarint()

            assert.NoError(t, err)
            assert.Equal(t, v, got)
        })
    }
}

func TestEventEncoding(t *testing.T) {
    cases := []struct {
        name    string
        event   EventType
        hid     string
        payload any
    }{
        {"click", EventClick, "h1", nil},
        {"input", EventInput, "h5", "hello"},
        {"keydown", EventKeyDown, "h3", KeyboardEventData{Key: "Enter", Modifiers: 0x01}},
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            encoded := EncodeEvent(1, tc.event, tc.hid, tc.payload)
            assert.NotEmpty(t, encoded)

            // Verify decode
            decoded, err := DecodeEvent(encoded)
            assert.NoError(t, err)
            assert.Equal(t, tc.event, decoded.Type)
            assert.Equal(t, tc.hid, decoded.HID)
        })
    }
}

func TestPatchEncoding(t *testing.T) {
    patches := []Patch{
        {Op: PatchSetText, HID: "h1", Value: "Hello"},
        {Op: PatchSetAttr, HID: "h2", Key: "class", Value: "active"},
        {Op: PatchRemoveNode, HID: "h3"},
    }

    encoded := EncodePatches(1, patches)
    assert.NotEmpty(t, encoded)

    decoded, err := DecodePatches(encoded)
    assert.NoError(t, err)
    assert.Equal(t, len(patches), len(decoded))
}
```

### Fuzz Tests

```go
func FuzzDecoder(f *testing.F) {
    // Seed with valid messages
    f.Add(EncodeEvent(1, EventClick, "h1", nil))
    f.Add(EncodePatches(1, []Patch{{Op: PatchSetText, HID: "h1", Value: "test"}}))

    f.Fuzz(func(t *testing.T, data []byte) {
        // Should not panic
        defer func() {
            if r := recover(); r != nil {
                t.Errorf("panic on input: %x", data)
            }
        }()

        d := NewDecoder(data)
        // Try to decode as event
        _, _ = DecodeEvent(data)
        // Try to decode as patches
        _, _ = DecodePatches(data)
    })
}
```

### Benchmark Tests

```go
func BenchmarkEncodeEvent(b *testing.B) {
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = EncodeEvent(uint64(i), EventClick, "h42", nil)
    }
}

func BenchmarkDecodeEvent(b *testing.B) {
    encoded := EncodeEvent(1, EventInput, "h5", "hello world")
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = DecodeEvent(encoded)
    }
}

func BenchmarkEncodePatches(b *testing.B) {
    patches := make([]Patch, 100)
    for i := range patches {
        patches[i] = Patch{Op: PatchSetText, HID: fmt.Sprintf("h%d", i), Value: "test"}
    }
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = EncodePatches(uint64(i), patches)
    }
}

func BenchmarkDecodePatches(b *testing.B) {
    patches := make([]Patch, 100)
    for i := range patches {
        patches[i] = Patch{Op: PatchSetText, HID: fmt.Sprintf("h%d", i), Value: "test"}
    }
    encoded := EncodePatches(1, patches)
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = DecodePatches(encoded)
    }
}
```

---

## File Structure

```
pkg/protocol/
├── doc.go            # Package documentation
├── varint.go         # Varint encoding
├── varint_test.go
├── encoder.go        # Encoder implementation
├── decoder.go        # Decoder implementation
├── codec_test.go     # Round-trip tests
├── frame.go          # Frame types and header
├── frame_test.go
├── event.go          # Event types and encoding
├── event_test.go
├── patch.go          # Patch types and encoding
├── patch_test.go
├── vnode.go          # VNode wire format
├── vnode_test.go
├── handshake.go      # Handshake protocol
├── handshake_test.go
├── control.go        # Control messages
├── control_test.go
├── ack.go            # Acknowledgment
├── ack_test.go
├── error.go          # Error types
├── error_test.go
├── fuzz_test.go      # Fuzz testing
└── bench_test.go     # Benchmarks
```

**Total: 24 files** (12 implementation + 12 test)

---

## Exit Criteria

Phase 3 is complete when:

1. [x] All frame types defined and documented
2. [x] Varint encoding/decoding with tests
3. [x] All event types encoded/decoded (25+ types)
4. [x] All patch types encoded/decoded (20+ operations)
5. [x] VNode serialization working
6. [x] Handshake protocol defined
7. [x] Control messages implemented
8. [x] Fuzz tests pass without panics (12 fuzz targets)
9. [x] Benchmarks show < 1μs per event/patch (achieved ~50ns)
10. [x] Wire format documented (doc.go)

**All exit criteria verified complete on 2024-12-07.**

---

## Dependencies

- **Requires**: Phase 2 (VNode structure for serialization)
- **Required by**: Phase 4 (server uses for event decoding), Phase 5 (client uses for patch decoding)

---

*Phase 3 Specification - Version 1.1 (Updated 2024-12-07)*
*Implementation complete - spec updated to match actual implementation*
