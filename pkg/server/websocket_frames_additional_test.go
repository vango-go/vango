package server

import (
	"log/slog"
	"testing"
	"time"

	"github.com/vango-go/vango/pkg/protocol"
)

func TestSession_WebSocketFrameHandlers(t *testing.T) {
	clientConn, serverConn := newWebSocketPair(t)

	sess := newSession(serverConn, "", DefaultSessionConfig(), slog.Default())

	// 1) Invalid event payload should trigger an error frame.
	sess.handleEventFrame([]byte{0x01}) // too short to decode as event

	_ = clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := clientConn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage error frame failed: %v", err)
	}
	frame, err := protocol.DecodeFrame(msg)
	if err != nil {
		t.Fatalf("DecodeFrame error frame failed: %v", err)
	}
	if frame.Type != protocol.FrameError {
		t.Fatalf("frame type=%v, want %v", frame.Type, protocol.FrameError)
	}
	em, err := protocol.DecodeErrorMessage(frame.Payload)
	if err != nil {
		t.Fatalf("DecodeErrorMessage failed: %v", err)
	}
	if em.Code != protocol.ErrInvalidEvent {
		t.Fatalf("error code=%v, want %v", em.Code, protocol.ErrInvalidEvent)
	}

	// 2) ControlPing should respond with a pong.
	ct, ping := protocol.NewPing(123)
	controlPayload := protocol.EncodeControl(ct, ping)
	sess.handleControlFrame(controlPayload)

	_ = clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err = clientConn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage pong failed: %v", err)
	}
	frame, err = protocol.DecodeFrame(msg)
	if err != nil {
		t.Fatalf("DecodeFrame pong failed: %v", err)
	}
	if frame.Type != protocol.FrameControl {
		t.Fatalf("pong frame type=%v, want %v", frame.Type, protocol.FrameControl)
	}
	ct2, data, err := protocol.DecodeControl(frame.Payload)
	if err != nil {
		t.Fatalf("DecodeControl pong failed: %v", err)
	}
	if ct2 != protocol.ControlPong {
		t.Fatalf("control type=%v, want %v", ct2, protocol.ControlPong)
	}
	pp, ok := data.(*protocol.PingPong)
	if !ok || pp.Timestamp != 123 {
		t.Fatalf("pong payload=%T %+v, want timestamp=123", data, data)
	}

	// 3) ACK updates session ack sequence.
	ackPayload := protocol.EncodeAck(protocol.NewAck(42, 100))
	sess.handleAckFrame(ackPayload)
	if sess.ackSeq.Load() != 42 {
		t.Fatalf("ackSeq=%d, want %d", sess.ackSeq.Load(), 42)
	}
}

