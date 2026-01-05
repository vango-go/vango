package server

import (
	"fmt"
	"runtime/debug"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vango-dev/vango/v2/pkg/protocol"
	"github.com/vango-dev/vango/v2/pkg/vango"
)

// ReadLoop continuously reads messages from the WebSocket connection.
// It decodes frames, processes control messages, and queues events.
// This method blocks until the connection is closed or an error occurs.
func (s *Session) ReadLoop() {
	defer s.Close()

	for {
		// Set read deadline
		s.conn.SetReadDeadline(time.Now().Add(s.config.ReadTimeout))

		// Read message
		_, msg, err := s.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
				websocket.CloseNormalClosure) {
				s.logger.Error("read error", "error", err)
			}
			return
		}

		// Update activity
		s.UpdateLastActive()
		s.BytesReceived(len(msg))

		// Decode frame
		frame, err := protocol.DecodeFrame(msg)
		if err != nil {
			s.logger.Error("frame decode error", "error", err)
			continue
		}

		// Handle based on frame type
		switch frame.Type {
		case protocol.FrameEvent:
			s.handleEventFrame(frame.Payload)

		case protocol.FrameControl:
			s.handleControlFrame(frame.Payload)

		case protocol.FrameAck:
			s.handleAckFrame(frame.Payload)

		default:
			s.logger.Warn("unknown frame type", "type", frame.Type)
		}
	}
}

// handleEventFrame decodes and queues an event from the client.
func (s *Session) handleEventFrame(payload []byte) {
	if DebugMode {
		fmt.Printf("[WS] Received event frame, %d bytes\n", len(payload))
	}

	// Decode event
	pe, err := protocol.DecodeEvent(payload)
	if err != nil {
		s.logger.Error("event decode error", "error", err)
		s.sendErrorMessage(protocol.ErrInvalidEvent, "Invalid event format")
		return
	}

	if DebugMode {
		fmt.Printf("[WS] Decoded event: HID=%s Type=%v\n", pe.HID, pe.Type)
	}

	// Convert to server event
	event := eventFromProtocol(pe, s)

	// Queue for processing
	if err := s.QueueEvent(event); err != nil {
		s.sendErrorMessage(protocol.ErrRateLimited, "Event queue full")
	}
}

// handleControlFrame handles control messages (ping, pong, resync, close).
func (s *Session) handleControlFrame(payload []byte) {
	ct, data, err := protocol.DecodeControl(payload)
	if err != nil {
		s.logger.Error("control decode error", "error", err)
		return
	}

	switch ct {
	case protocol.ControlPing:
		// Respond with pong
		if pp, ok := data.(*protocol.PingPong); ok {
			s.sendPong(pp.Timestamp)
		}

	case protocol.ControlPong:
		// Client responded to our ping - update latency metrics if needed
		s.logger.Debug("received pong")

	case protocol.ControlResyncRequest:
		// Client requests missed patches
		if rr, ok := data.(*protocol.ResyncRequest); ok {
			s.handleResyncRequest(rr.LastSeq)
		}

	case protocol.ControlClose:
		// Client is closing
		if cm, ok := data.(*protocol.CloseMessage); ok {
			s.logger.Info("client closing", "reason", cm.Reason, "message", cm.Message)
		}
		s.Close()
	}
}

// handleAckFrame handles acknowledgment messages.
func (s *Session) handleAckFrame(payload []byte) {
	ack, err := protocol.DecodeAck(payload)
	if err != nil {
		s.logger.Error("ack decode error", "error", err)
		return
	}

	// Update acknowledged sequence
	s.ackSeq.Store(ack.LastSeq)
	s.logger.Debug("received ack", "seq", ack.LastSeq)
}

// handleResyncRequest handles a client request for missed patches.
// For now, we send a full resync since we don't store patch history.
func (s *Session) handleResyncRequest(lastSeq uint64) {
	s.logger.Info("resync requested", "last_seq", lastSeq)

	// For now, we don't have patch history, so we'd need to do a full reload
	// In production, this would send missed patches from a buffer
	// For now, just log it
	s.logger.Warn("patch history not implemented, client should reload")
}

// sendPong sends a pong response.
func (s *Session) sendPong(timestamp uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed.Load() {
		return
	}

	ct, pp := protocol.NewPong(timestamp)
	payload := protocol.EncodeControl(ct, pp)
	frame := protocol.NewFrame(protocol.FrameControl, payload)

	s.conn.SetWriteDeadline(time.Now().Add(s.config.WriteTimeout))
	if err := s.conn.WriteMessage(websocket.BinaryMessage, frame.Encode()); err != nil {
		s.logger.Error("pong error", "error", err)
	}
}

// WriteLoop handles periodic tasks like heartbeats.
// It runs until the session is closed.
func (s *Session) WriteLoop() {
	ticker := time.NewTicker(s.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.sendPing(); err != nil {
				return
			}

		case <-s.done:
			return
		}
	}
}

// EventLoop processes queued events, dispatch callbacks, and render signals.
// It runs handlers, schedules effects, and triggers re-renders.
func (s *Session) EventLoop() {
	for {
		select {
		case event := <-s.events:
			s.handleEvent(event)

		case fn := <-s.dispatchCh:
			// Execute dispatched function on the event loop
			s.executeDispatch(fn)

		case <-s.renderCh:
			s.renderDirty()

		case <-s.done:
			return
		}
	}
}

// executeDispatch runs a dispatched function with proper cleanup.
// It handles panic recovery, runs pending effects, and re-renders dirty components.
//
// IMPORTANT: Wraps execution with vango.WithCtx so that UseCtx() is valid
// inside dispatched callbacks. This is required by SPEC_ADDENDUM.md:90 which
// states that UseCtx() MUST be valid during callbacks invoked on the session
// loop via ctx.Dispatch(...).
func (s *Session) executeDispatch(fn func()) {
	// Panic recovery
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			s.logger.Error("dispatch panic",
				"panic", r,
				"stack", string(stack))
		}
	}()

	// Create context so UseCtx() works in dispatched callbacks
	ctx := s.createRenderContext()

	// Execute with context set, matching handleEvent pattern
	vango.WithCtx(ctx, func() {
		// Execute the dispatched function
		fn()

		// Run pending effects (scheduled by signal updates)
		s.owner.RunPendingEffects()
	})

	// Re-render dirty components (outside WithCtx, same as handleEvent)
	s.renderDirty()
}

// Start starts all session loops.
// This should be called after the handshake is complete.
func (s *Session) Start() {
	go s.ReadLoop()
	go s.WriteLoop()
	go s.EventLoop()
}

// Resume resumes a session after reconnect.
// It swaps the WebSocket connection, resets sequence numbers, and reinitializes
// channels if the session was previously closed. Call NeedsRestart() after
// Resume() to check if Start() should be called to restart goroutines.
func (s *Session) Resume(conn *websocket.Conn, lastSeq uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Swap connection
	oldConn := s.conn
	s.conn = conn

	// Close old connection if exists
	if oldConn != nil {
		oldConn.Close()
	}

	// Update activity
	s.LastActive = time.Now()

	// Reset closed flag if it was set
	s.closed.Store(false)

	// Check if goroutines need restarting (done channel closed)
	needsRestart := false
	select {
	case <-s.done:
		// Done was closed, need to reinitialize channels
		s.done = make(chan struct{})
		s.events = make(chan *Event, s.config.MaxEventQueue)
		s.renderCh = make(chan struct{}, 1)
		s.dispatchCh = make(chan func(), s.config.MaxEventQueue)
		needsRestart = true
	default:
		// Goroutines still running, just swapped connection
	}

	// Reset send sequence for fresh patches after RebuildHandlers()
	s.sendSeq.Store(0)
	// Track client's last received sequence
	s.recvSeq.Store(lastSeq)

	s.logger.Info("session resumed",
		"last_seq", lastSeq,
		"needs_restart", needsRestart)
}

// NeedsRestart returns true if session goroutines need to be restarted.
// This should be checked after Resume() to decide whether to call Start().
// If the session's done channel was closed, goroutines have exited and
// need to be restarted for the session to function.
func (s *Session) NeedsRestart() bool {
	select {
	case <-s.done:
		return true
	default:
		return false
	}
}

// SendPatches is a public wrapper for sendPatches.
func (s *Session) SendPatches(patches []protocol.Patch) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed.Load() {
		return
	}

	seq := s.sendSeq.Add(1)

	pf := &protocol.PatchesFrame{
		Seq:     seq,
		Patches: patches,
	}

	payload := protocol.EncodePatches(pf)
	frame := protocol.NewFrame(protocol.FramePatches, payload)
	frameData := frame.Encode()

	s.conn.SetWriteDeadline(time.Now().Add(s.config.WriteTimeout))

	if err := s.conn.WriteMessage(websocket.BinaryMessage, frameData); err != nil {
		s.logger.Error("write error", "error", err)
		s.closeInternal()
		return
	}

	s.bytesSent.Add(uint64(len(frameData)))
	s.patchCount.Add(uint64(len(patches)))
}

// SendClose sends a close control message to the client.
func (s *Session) SendClose(reason protocol.CloseReason, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed.Load() {
		return
	}

	ct, cm := protocol.NewClose(reason, message)
	payload := protocol.EncodeControl(ct, cm)
	frame := protocol.NewFrame(protocol.FrameControl, payload)

	s.conn.SetWriteDeadline(time.Now().Add(s.config.WriteTimeout))
	s.conn.WriteMessage(websocket.BinaryMessage, frame.Encode())
}
