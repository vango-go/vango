package server

import (
	"log/slog"
	"testing"
	"time"

	"github.com/vango-go/vango/pkg/protocol"
	"github.com/vango-go/vango/pkg/vdom"
)

func TestSession_handleResyncRequest_Branches(t *testing.T) {
	clientConn, serverConn := newWebSocketPair(t)

	sess := newSession(serverConn, "", DefaultSessionConfig(), slog.Default())
	sess.currentTree = &vdom.VNode{Kind: vdom.KindElement, Tag: "div", Children: []*vdom.VNode{{Kind: vdom.KindText, Text: "x"}}}

	// Up-to-date branch: no write.
	sess.sendSeq.Store(5)
	sess.handleResyncRequest(5)

	// Insufficient history branch => full resync.
	sess.patchHistory = NewPatchHistory(1)
	frame := protocol.NewFrame(protocol.FramePatches, protocol.EncodePatches(&protocol.PatchesFrame{
		Seq:     5,
		Patches: []protocol.Patch{protocol.NewSetTextPatch("h1", "x")},
	})).Encode()
	sess.patchHistory.Add(5, frame)
	sess.handleResyncRequest(0)

	_ = clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := clientConn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage resync full failed: %v", err)
	}
	f, err := protocol.DecodeFrame(msg)
	if err != nil {
		t.Fatalf("DecodeFrame failed: %v", err)
	}
	if f.Type != protocol.FrameControl {
		t.Fatalf("frame type=%v, want %v", f.Type, protocol.FrameControl)
	}
	ct, _, err := protocol.DecodeControl(f.Payload)
	if err != nil {
		t.Fatalf("DecodeControl failed: %v", err)
	}
	if ct != protocol.ControlResyncFull {
		t.Fatalf("control type=%v, want %v", ct, protocol.ControlResyncFull)
	}
}

