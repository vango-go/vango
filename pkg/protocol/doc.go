// Package protocol implements the binary wire protocol for Vango V2.
//
// The protocol is optimized for minimal bandwidth and fast encoding/decoding.
// It defines how events flow from client to server and patches flow from
// server to client over WebSocket connections.
//
// # Design Goals
//
//   - Minimal size: Typical event < 10 bytes, typical patch < 20 bytes
//   - Fast encoding/decoding: No reflection, direct byte manipulation
//   - Reliable delivery: Sequence numbers, acknowledgments
//   - Reconnection: Resync capability after disconnect
//   - Extensible: Version negotiation, reserved opcodes
//
// # Wire Format
//
// All messages are framed with a 4-byte header:
//
//	┌─────────────┬──────────────┬───────────────────────────────┐
//	│ Frame Type  │ Flags        │ Payload Length                │
//	│ (1 byte)    │ (1 byte)     │ (2 bytes, big-endian)         │
//	└─────────────┴──────────────┴───────────────────────────────┘
//
// # Frame Types
//
//   - FrameHandshake (0x00): Connection setup
//   - FrameEvent (0x01): Client → Server events
//   - FramePatches (0x02): Server → Client patches
//   - FrameControl (0x03): Control messages (ping, resync)
//   - FrameAck (0x04): Acknowledgment
//   - FrameError (0x05): Error message
//
// # Encoding
//
// The protocol uses several encoding strategies:
//
//   - Varint: Compact encoding for small integers (protobuf-style)
//   - ZigZag: Signed integers encoded as unsigned varints
//   - Length-prefixed: Strings and byte arrays prefixed with varint length
//   - Big-endian: Fixed-width integers (uint16, uint32, uint64)
//
// # Events
//
// Events are sent from client to server when user interactions occur.
// Each event includes a sequence number, event type, hydration ID (HID),
// and type-specific payload.
//
// Example click event encoding:
//
//	[Seq: varint][Type: 0x01][HID: len-prefixed string]
//	Total: ~5 bytes for "h1"
//
// # Patches
//
// Patches are sent from server to client to update the DOM.
// Each patch includes an operation type, target HID, and operation-specific data.
//
// Example SetText patch encoding:
//
//	[Op: 0x01][HID: len-prefixed][Value: len-prefixed]
//	Total: ~15 bytes for updating "h1" with "Hello"
//
// # Handshake
//
// Connection establishment uses ClientHello and ServerHello messages:
//
//	Client                          Server
//	  │                                │
//	  │──── ClientHello ─────────────>│
//	  │     (version, csrf, session)  │
//	  │                                │
//	  │<──── ServerHello ─────────────│
//	  │     (status, session, time)   │
//	  │                                │
//
// # Control Messages
//
//   - Ping/Pong: Heartbeat for connection health
//   - ResyncRequest: Client requests missed patches after reconnect
//   - ResyncPatches/ResyncFull: Server response with missed data
//   - Close: Graceful session termination
//
// # Usage Example
//
//	// Encode an event
//	event := &Event{
//	    Seq:     1,
//	    Type:    EventClick,
//	    HID:     "h42",
//	}
//	data := EncodeEvent(event)
//
//	// Decode an event
//	decoded, err := DecodeEvent(data)
//	if err != nil {
//	    // Handle error
//	}
//
//	// Encode patches
//	pf := &PatchesFrame{
//	    Seq: 1,
//	    Patches: []Patch{
//	        NewSetTextPatch("h1", "Hello, World!"),
//	        NewSetAttrPatch("h2", "class", "active"),
//	    },
//	}
//	data = EncodePatches(pf)
//
//	// Decode patches
//	decoded, err := DecodePatches(data)
//
// # Performance
//
// Target metrics:
//   - Event encode/decode: < 500ns
//   - Patch encode/decode: < 500ns
//   - 100 patches encode: < 50μs
//   - 100 patches decode: < 50μs
//
// # File Structure
//
// The package is organized as follows:
//
//   - varint.go: Varint encoding/decoding
//   - encoder.go: Binary encoder
//   - decoder.go: Binary decoder
//   - frame.go: Frame types and transport
//   - event.go: Event types and encoding
//   - patch.go: Patch types and encoding
//   - vnode.go: VNode wire format
//   - handshake.go: Handshake protocol
//   - control.go: Control messages
//   - ack.go: Acknowledgment
//   - error.go: Error messages
package protocol
