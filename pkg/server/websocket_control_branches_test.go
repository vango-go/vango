package server

import (
	"log/slog"
	"testing"
	"time"

	"github.com/vango-go/vango/pkg/protocol"
	"github.com/vango-go/vango/pkg/vdom"
)

func TestSession_handleControlFrame_ResyncRequestAndClose(t *testing.T) {
	clientConn, serverConn := newWebSocketPair(t)

	sess := newSession(serverConn, "", DefaultSessionConfig(), slog.Default())
	sess.currentTree = &vdom.VNode{Kind: vdom.KindElement, Tag: "div", Children: []*vdom.VNode{{Kind: vdom.KindText, Text: "x"}}}
	sess.sendSeq.Store(10)
	sess.patchHistory = nil // force full resync path

	ct, rr := protocol.NewResyncRequest(0)
	payload := protocol.EncodeControl(ct, rr)
	sess.handleControlFrame(payload)

	_ = clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := clientConn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage resync response failed: %v", err)
	}
	frame, err := protocol.DecodeFrame(msg)
	if err != nil {
		t.Fatalf("DecodeFrame failed: %v", err)
	}
	if frame.Type != protocol.FrameControl {
		t.Fatalf("frame type=%v, want %v", frame.Type, protocol.FrameControl)
	}
	ct2, _, err := protocol.DecodeControl(frame.Payload)
	if err != nil {
		t.Fatalf("DecodeControl failed: %v", err)
	}
	if ct2 != protocol.ControlResyncFull {
		t.Fatalf("control type=%v, want %v", ct2, protocol.ControlResyncFull)
	}

	ct, cm := protocol.NewClose(protocol.CloseNormal, "bye")
	sess.handleControlFrame(protocol.EncodeControl(ct, cm))
	if !sess.IsClosed() {
		t.Fatal("expected session to be closed after ControlClose")
	}
}

