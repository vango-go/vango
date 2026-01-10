package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vango-go/vango/pkg/protocol"
	"github.com/vango-go/vango/pkg/vdom"
)

func TestSessionDetachesOnDisconnectAndResumesIO(t *testing.T) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	newConnPair := func() (*websocket.Conn, *websocket.Conn, func()) {
		serverConnCh := make(chan *websocket.Conn, 1)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				t.Errorf("upgrade: %v", err)
				return
			}
			serverConnCh <- c
		}))

		wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
		clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			srv.Close()
			t.Fatalf("dial: %v", err)
		}

		serverConn := <-serverConnCh

		cleanup := func() {
			_ = clientConn.Close()
			_ = serverConn.Close()
			srv.Close()
		}

		return clientConn, serverConn, cleanup
	}

	client1, server1, cleanup1 := newConnPair()
	defer cleanup1()

	sess := newSession(server1, "", DefaultSessionConfig(), testLogger())
	sess.MountRoot(FuncComponent(func() *vdom.VNode { return vdom.Div(vdom.Text("ok")) }))
	sess.Start()

	if sess.IsDetached() {
		t.Fatal("expected session to start attached")
	}

	// Simulate a browser refresh / connection drop.
	_ = client1.Close()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if sess.IsDetached() {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !sess.IsDetached() {
		t.Fatal("expected session to detach after connection closes")
	}
	if sess.IsClosed() {
		t.Fatal("expected detached session to remain open (resumable)")
	}
	if sess.Owner() == nil || sess.Owner().IsDisposed() {
		t.Fatal("expected detached session owner to remain alive for resume")
	}

	// Resume on a new WebSocket connection.
	client2, server2, cleanup2 := newConnPair()
	defer cleanup2()

	sess.Resume(server2, 0)
	if sess.IsDetached() {
		t.Fatal("expected session to be marked attached after Resume")
	}

	if sess.NeedsRestart() {
		sess.Start()
	}

	// Verify the resumed ReadLoop is active by sending a protocol ping and
	// reading a protocol pong response.
	ct, pp := protocol.NewPing(uint64(time.Now().UnixMilli()))
	payload := protocol.EncodeControl(ct, pp)
	frame := protocol.NewFrame(protocol.FrameControl, payload)

	_ = client2.SetWriteDeadline(time.Now().Add(1 * time.Second))
	if err := client2.WriteMessage(websocket.BinaryMessage, frame.Encode()); err != nil {
		t.Fatalf("write ping frame: %v", err)
	}

	_ = client2.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := client2.ReadMessage()
	if err != nil {
		t.Fatalf("read pong frame: %v", err)
	}

	gotFrame, err := protocol.DecodeFrame(msg)
	if err != nil {
		t.Fatalf("decode frame: %v", err)
	}
	if gotFrame.Type != protocol.FrameControl {
		t.Fatalf("frame type = %v, want %v", gotFrame.Type, protocol.FrameControl)
	}

	gotType, gotData, err := protocol.DecodeControl(gotFrame.Payload)
	if err != nil {
		t.Fatalf("decode control: %v", err)
	}
	if gotType != protocol.ControlPong {
		t.Fatalf("control type = %v, want %v", gotType, protocol.ControlPong)
	}
	if _, ok := gotData.(*protocol.PingPong); !ok {
		t.Fatalf("control payload type = %T, want *protocol.PingPong", gotData)
	}
}

