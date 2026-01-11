package server

import (
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/vango-go/vango/pkg/protocol"
	"github.com/vango-go/vango/pkg/vdom"
)

type onClickComponent struct{}

func (onClickComponent) Render() *vdom.VNode {
	return &vdom.VNode{
		Kind: vdom.KindElement,
		Tag:  "button",
		Props: map[string]any{
			"onclick": func() {},
		},
	}
}

func TestSession_RebuildHandlers_ErrorsWithoutRoot(t *testing.T) {
	s := NewMockSession()
	if err := s.RebuildHandlers(); err == nil {
		t.Fatal("RebuildHandlers() error=nil, want non-nil")
	}
}

func TestSession_RebuildHandlers_ReassignsHIDsAndHandlers(t *testing.T) {
	s := NewMockSession()
	s.MountRoot(onClickComponent{})
	if s.root == nil || s.currentTree == nil {
		t.Fatal("expected root/currentTree after MountRoot")
	}

	oldHID := s.currentTree.HID
	if oldHID == "" {
		t.Fatal("expected initial tree HID to be assigned")
	}

	if err := s.RebuildHandlers(); err != nil {
		t.Fatalf("RebuildHandlers() error: %v", err)
	}
	if s.currentTree == nil || s.currentTree.HID == "" {
		t.Fatal("expected currentTree with HID after RebuildHandlers")
	}
	if s.currentTree.HID != oldHID {
		// RebuildHandlers resets HID generator, so stable trees should reassign the same HIDs.
		t.Fatalf("tree HID changed from %q to %q; expected stable re-assignment", oldHID, s.currentTree.HID)
	}
	if len(s.handlers) == 0 {
		t.Fatal("expected handlers after RebuildHandlers")
	}
	if len(s.components) == 0 {
		t.Fatal("expected components mapping after RebuildHandlers")
	}
}

func TestSession_collectHandlers_MountsChildComponentsAndRegistersHandlers(t *testing.T) {
	s := NewMockSession()
	s.handlers = make(map[string]Handler)
	s.components = make(map[string]*ComponentInstance)

	child := onClickComponent{}
	rootComp := staticComponent{
		node: &vdom.VNode{
			Kind: vdom.KindElement,
			Tag:  "div",
			Children: []*vdom.VNode{
				{Kind: vdom.KindComponent, Comp: child},
			},
		},
	}

	rootInst := newComponentInstance(rootComp, nil, s)
	s.registerComponent(rootInst)
	tree := rootInst.Render()
	vdom.AssignHIDs(tree, s.hidGen)

	s.collectHandlers(tree, rootInst)

	if len(rootInst.Children) != 1 {
		t.Fatalf("root children=%d, want 1", len(rootInst.Children))
	}
	childInst := rootInst.Children[0]
	if childInst == nil || childInst.LastTree() == nil || childInst.LastTree().HID == "" {
		t.Fatalf("child instance not mounted with HIDs: %+v", childInst)
	}

	key := childInst.LastTree().HID + "_onclick"
	if _, ok := s.handlers[key]; !ok {
		t.Fatalf("missing handler key %q", key)
	}
}

func TestSession_clearComponentHandlers_RemovesOwnedHandlersAndMappings(t *testing.T) {
	s := NewMockSession()
	comp := &ComponentInstance{InstanceID: "c1"}
	s.components = map[string]*ComponentInstance{
		"h1": comp,
		"h2": comp,
		"h3": &ComponentInstance{InstanceID: "c2"},
	}
	s.handlers = map[string]Handler{
		"h1_onclick": func(*Event) {},
		"h2_oninput": func(*Event) {},
		"h3_onclick": func(*Event) {},
	}

	s.clearComponentHandlers(comp)

	if _, ok := s.components["h1"]; ok {
		t.Fatal("h1 still present in components after clearComponentHandlers")
	}
	if _, ok := s.handlers["h1_onclick"]; ok {
		t.Fatal("h1_onclick still present after clearComponentHandlers")
	}
	if _, ok := s.handlers["h2_oninput"]; ok {
		t.Fatal("h2_oninput still present after clearComponentHandlers")
	}
	if _, ok := s.components["h3"]; !ok {
		t.Fatal("h3 unexpectedly removed")
	}
	if _, ok := s.handlers["h3_onclick"]; !ok {
		t.Fatal("h3_onclick unexpectedly removed")
	}
}

func TestSession_SendResyncFull_AndWebsocketResyncReplay(t *testing.T) {
	clientConn, serverConn := newWebSocketPair(t)

	sess := newSession(serverConn, "", DefaultSessionConfig(), slog.Default())
	sess.MountRoot(staticComponent{node: &vdom.VNode{Kind: vdom.KindElement, Tag: "div", Children: []*vdom.VNode{
		{Kind: vdom.KindText, Text: "hello"},
	}}})

	// ResyncFull requires a tree and a live connection.
	if err := sess.SendResyncFull(); err != nil {
		t.Fatalf("SendResyncFull() error: %v", err)
	}

	_ = clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := clientConn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}
	frame, err := protocol.DecodeFrame(msg)
	if err != nil {
		t.Fatalf("DecodeFrame failed: %v", err)
	}
	if frame.Type != protocol.FrameControl {
		t.Fatalf("frame type=%v, want %v", frame.Type, protocol.FrameControl)
	}
	ct, data, err := protocol.DecodeControl(frame.Payload)
	if err != nil {
		t.Fatalf("DecodeControl failed: %v", err)
	}
	if ct != protocol.ControlResyncFull {
		t.Fatalf("control type=%v, want %v", ct, protocol.ControlResyncFull)
	}
	rf, ok := data.(*protocol.ResyncResponse)
	if !ok || rf == nil || rf.Type != protocol.ControlResyncFull || rf.HTML == "" {
		t.Fatalf("resyncFull=%T %+v, want non-empty HTML", data, data)
	}

	// Also cover the patch replay path: write a few patch frames to history and request resync.
	f1 := protocol.NewFrame(protocol.FramePatches, protocol.EncodePatches(&protocol.PatchesFrame{
		Seq: 1,
		Patches: []protocol.Patch{
			protocol.NewSetTextPatch("h1", "a"),
		},
	})).Encode()
	f2 := protocol.NewFrame(protocol.FramePatches, protocol.EncodePatches(&protocol.PatchesFrame{
		Seq: 2,
		Patches: []protocol.Patch{
			protocol.NewSetTextPatch("h1", "b"),
		},
	})).Encode()

	sess.patchHistory = NewPatchHistory(10)
	sess.patchHistory.Add(1, f1)
	sess.patchHistory.Add(2, f2)
	sess.sendSeq.Store(2)

	sess.handleResyncRequest(0)
	_ = clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, got1, err := clientConn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage replay1 failed: %v", err)
	}
	_, got2, err := clientConn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage replay2 failed: %v", err)
	}
	if string(got1) != string(f1) || string(got2) != string(f2) {
		t.Fatalf("replayed frames differ: got1=%d got2=%d want1=%d want2=%d", len(got1), len(got2), len(f1), len(f2))
	}
}

func TestSession_sendPatches_sendPing_SendClose(t *testing.T) {
	clientConn, serverConn := newWebSocketPair(t)

	sess := newSession(serverConn, "", DefaultSessionConfig(), slog.Default())
	sess.patchHistory = NewPatchHistory(10)

	// sendPing should succeed with a live connection.
	if err := sess.sendPing(); err != nil {
		t.Fatalf("sendPing() error: %v", err)
	}
	_ = clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := clientConn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage ping failed: %v", err)
	}
	frame, err := protocol.DecodeFrame(msg)
	if err != nil {
		t.Fatalf("DecodeFrame ping failed: %v", err)
	}
	if frame.Type != protocol.FrameControl {
		t.Fatalf("frame type=%v, want %v", frame.Type, protocol.FrameControl)
	}

	// sendPatches should encode vdom patches, store in history, and deliver to client.
	sess.sendPatches([]vdom.Patch{
		{Op: vdom.PatchSetText, HID: "h1", Value: "hello"},
		{Op: vdom.PatchInsertNode, ParentID: "h1", Index: 0, Node: &vdom.VNode{Kind: vdom.KindText, Text: "x"}},
	})
	_ = clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err = clientConn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage patches failed: %v", err)
	}
	frame, err = protocol.DecodeFrame(msg)
	if err != nil {
		t.Fatalf("DecodeFrame patches failed: %v", err)
	}
	if frame.Type != protocol.FramePatches {
		t.Fatalf("frame type=%v, want %v", frame.Type, protocol.FramePatches)
	}
	pf, err := protocol.DecodePatches(frame.Payload)
	if err != nil {
		t.Fatalf("DecodePatches failed: %v", err)
	}
	if len(pf.Patches) != 2 {
		t.Fatalf("patch count=%d, want 2", len(pf.Patches))
	}
	if sess.patchHistory.MaxSeq() == 0 {
		t.Fatal("expected patchHistory to record sent frame")
	}

	// SendClose should emit a close control frame.
	sess.SendClose(protocol.CloseNormal, "bye")
	_ = clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err = clientConn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage close failed: %v", err)
	}
	frame, err = protocol.DecodeFrame(msg)
	if err != nil {
		t.Fatalf("DecodeFrame close failed: %v", err)
	}
	if frame.Type != protocol.FrameControl {
		t.Fatalf("frame type=%v, want %v", frame.Type, protocol.FrameControl)
	}
	ct, _, err := protocol.DecodeControl(frame.Payload)
	if err != nil {
		t.Fatalf("DecodeControl close failed: %v", err)
	}
	if ct != protocol.ControlClose {
		t.Fatalf("control type=%v, want %v", ct, protocol.ControlClose)
	}
}

func TestSession_SendResyncFull_ErrorsWithoutTreeOrConn(t *testing.T) {
	s := NewMockSession()
	if err := s.SendResyncFull(); err == nil {
		t.Fatal("SendResyncFull() error=nil, want non-nil without tree")
	}

	s.currentTree = &vdom.VNode{Kind: vdom.KindElement, Tag: "div"}
	if err := s.SendResyncFull(); !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("SendResyncFull() error=%v, want %v", err, ErrSessionClosed)
	}
}

func TestSession_sendPing_ErrorsWithoutConn(t *testing.T) {
	s := NewMockSession()
	if err := s.sendPing(); !errors.Is(err, ErrNoConnection) {
		t.Fatalf("sendPing() error=%v, want %v", err, ErrNoConnection)
	}
}
