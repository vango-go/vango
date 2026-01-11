package server

import (
	"log/slog"
	"testing"
	"time"

	"github.com/vango-go/vango/pkg/protocol"
)

func TestSession_handleEventFrame_SuccessAndQueueFull(t *testing.T) {
	t.Run("success queues decoded event", func(t *testing.T) {
		_, serverConn := newWebSocketPair(t)
		sess := newSession(serverConn, "", DefaultSessionConfig(), slog.Default())
		sess.events = make(chan *Event, 1)

		payload := protocol.EncodeEvent(&protocol.Event{
			Seq:  1,
			Type: protocol.EventClick,
			HID:  "h1",
		})
		sess.handleEventFrame(payload)

		select {
		case e := <-sess.events:
			if e == nil || e.Seq != 1 || e.HID != "h1" || e.Type != protocol.EventClick {
				t.Fatalf("queued event=%+v, want click seq=1 hid=h1", e)
			}
		case <-time.After(1 * time.Second):
			t.Fatal("expected event to be queued")
		}
	})

	t.Run("queue full sends rate-limited error", func(t *testing.T) {
		clientConn, serverConn := newWebSocketPair(t)
		sess := newSession(serverConn, "", DefaultSessionConfig(), slog.Default())

		// Make event queue always "full".
		sess.events = make(chan *Event)

		payload := protocol.EncodeEvent(&protocol.Event{
			Seq:  1,
			Type: protocol.EventClick,
			HID:  "h1",
		})
		sess.handleEventFrame(payload)

		_ = clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, msg, err := clientConn.ReadMessage()
		if err != nil {
			t.Fatalf("ReadMessage failed: %v", err)
		}
		frame, err := protocol.DecodeFrame(msg)
		if err != nil {
			t.Fatalf("DecodeFrame failed: %v", err)
		}
		if frame.Type != protocol.FrameError {
			t.Fatalf("frame type=%v, want %v", frame.Type, protocol.FrameError)
		}
		em, err := protocol.DecodeErrorMessage(frame.Payload)
		if err != nil {
			t.Fatalf("DecodeErrorMessage failed: %v", err)
		}
		if em.Code != protocol.ErrRateLimited {
			t.Fatalf("error code=%v, want %v", em.Code, protocol.ErrRateLimited)
		}
	})
}

