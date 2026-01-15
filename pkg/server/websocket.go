package server

import (
	"fmt"
	"runtime/debug"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vango-go/vango/pkg/protocol"
	"github.com/vango-go/vango/pkg/vango"
)

// ReadLoop continuously reads messages from the WebSocket connection.
// It decodes frames, processes control messages, and queues events.
// This method blocks until the connection is closed or an error occurs.
func (s *Session) ReadLoop() {
	if s.readLoopRunning.Swap(true) {
		// Already running
		return
	}
	defer s.readLoopRunning.Store(false)

	for {
		// If the session is fully closed, exit.
		if s.closed.Load() {
			return
		}

		// Set read deadline
		s.mu.Lock()
		conn := s.conn
		s.mu.Unlock()
		if conn == nil {
			// Detached: nothing to read.
			return
		}
		conn.SetReadDeadline(time.Now().Add(s.config.ReadTimeout))

		// Read message
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
				websocket.CloseNormalClosure) {
				s.logger.Error("read error", "error", err)
			}
			s.detach("read", err)
			return
		}

		// Update activity
		s.UpdateLastActive()
		s.detached.Store(false)
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
// ACKs are used for garbage collection of the patch history buffer.
func (s *Session) handleAckFrame(payload []byte) {
	ack, err := protocol.DecodeAck(payload)
	if err != nil {
		s.logger.Error("ack decode error", "error", err)
		return
	}

	// Update acknowledged sequence
	s.ackSeq.Store(ack.LastSeq)

	// Garbage collect old patch history
	// Patches with seq <= ackSeq are no longer needed for resync
	if s.patchHistory != nil {
		s.patchHistory.GarbageCollect(ack.LastSeq)
	}

	s.logger.Debug("received ack", "seq", ack.LastSeq, "window", ack.Window)
}

// handleResyncRequest handles a client request for missed patches.
// If the requested patches are still in the history buffer, they are replayed.
// Otherwise, a full resync (complete HTML) is sent.
func (s *Session) handleResyncRequest(lastSeq uint64) {
	currentSeq := s.sendSeq.Load()
	s.logger.Info("resync requested",
		"last_seq", lastSeq,
		"current_seq", currentSeq)

	// Nothing to resync if client is up to date
	if lastSeq >= currentSeq {
		s.logger.Debug("resync not needed, client up to date")
		return
	}

	// Check if we can recover from patch history
	if s.patchHistory == nil {
		s.logger.Warn("resync: no patch history available, sending full resync")
		if err := s.SendResyncFull(); err != nil {
			s.logger.Error("resync full failed", "error", err)
		}
		return
	}

	if !s.patchHistory.CanRecover(lastSeq) {
		s.logger.Warn("resync: patch history insufficient",
			"requested", lastSeq,
			"min_available", s.patchHistory.MinSeq(),
			"max_available", s.patchHistory.MaxSeq())
		if err := s.SendResyncFull(); err != nil {
			s.logger.Error("resync full failed", "error", err)
		}
		return
	}

	// Get missed frames and replay them
	frames := s.patchHistory.GetFrames(lastSeq, currentSeq)
	if frames == nil {
		s.logger.Warn("resync: failed to get frames from history, sending full resync")
		if err := s.SendResyncFull(); err != nil {
			s.logger.Error("resync full failed", "error", err)
		}
		return
	}

	// Replay the original FramePatches frames
	// Client will process them as normal patch frames
	s.replayPatchFrames(frames)
}

// replayPatchFrames re-sends previously sent patch frames to the client.
// This is used for resync when a client misses some patches.
func (s *Session) replayPatchFrames(frames [][]byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed.Load() || s.conn == nil {
		return
	}

	for i, frame := range frames {
		s.conn.SetWriteDeadline(time.Now().Add(s.config.WriteTimeout))
		if err := s.conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
			s.logger.Error("replay patch frame failed",
				"frame_index", i,
				"total_frames", len(frames),
				"error", err)
			return
		}
	}

	s.logger.Info("replayed patch frames", "count", len(frames))
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
	if s.writeLoopRunning.Swap(true) {
		// Already running
		return
	}
	defer s.writeLoopRunning.Store(false)

	ticker := time.NewTicker(s.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.sendPing(); err != nil {
				// If we have no connection, we're detached. Let ReadLoop handle detach.
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
	if s.eventLoopRunning.Swap(true) {
		// Already running
		return
	}
	defer s.eventLoopRunning.Store(false)

	var authTicker *time.Ticker
	var authTickerC <-chan time.Time
	if cfg := s.config.AuthCheck; cfg != nil && cfg.Interval > 0 {
		authTicker = time.NewTicker(cfg.Interval)
		authTickerC = authTicker.C
		defer authTicker.Stop()
	}

	for {
		select {
		case event := <-s.events:
			func() {
				s.beginWork()
				defer s.endWork()
				if s.isAuthExpired() {
					s.triggerAuthExpired(AuthExpiredPassiveExpiry)
					return
				}
				s.handleEvent(event)
			}()

		case fn := <-s.dispatchCh:
			// Execute dispatched function on the event loop
			func() {
				s.beginWork()
				defer s.endWork()
				s.executeDispatch(fn)
			}()

		case <-authTickerC:
			func() {
				s.beginWork()
				defer s.endWork()
				if s.hasAuthPrincipal() {
					s.runActiveAuthCheck()
				}
			}()

		case <-s.renderCh:
			func() {
				s.beginWork()
				defer s.endWork()

				ctx := s.createRenderContext()
				vango.WithCtx(ctx, func() {
					s.flush()
				})
			}()

		case <-s.done:
			return
		}
	}
}

// executeDispatch runs a dispatched function with proper cleanup.
// It handles panic recovery and runs the commit cycle (render, effects, rerender as needed).
//
// IMPORTANT: Wraps execution with vango.WithCtx so that UseCtx() is valid
// inside dispatched callbacks. This is required by SPEC_ADDENDUM.md:90 which
// states that UseCtx() MUST be valid during callbacks invoked on the session
// loop via ctx.Dispatch(...).
func (s *Session) executeDispatch(fn func()) {
	// Reset per-tick storm budget counters at the start of each dispatch tick
	if s.stormBudget != nil {
		s.stormBudget.ResetTick()
	}

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

		// Commit render + effects after dispatch
		s.flush()
	})
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
	s.DetachedAt = time.Time{}
	s.detached.Store(false)

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
	// Any of the loop goroutines may have exited due to a detach/reconnect cycle.
	// Start() is safe to call again; the loops guard against duplicates.
	return !s.readLoopRunning.Load() || !s.writeLoopRunning.Load() || !s.eventLoopRunning.Load()
}

// detach transitions a session into the "detached" state: the WebSocket is gone,
// but the session (signals/components) is kept in memory for ResumeWindow.
//
// This MUST NOT dispose the owner/component tree, otherwise resume cannot restore state.
func (s *Session) detach(source string, err error) {
	s.mu.Lock()

	if s.closed.Load() {
		s.mu.Unlock()
		return
	}

	// Close and nil out the connection.
	if s.conn != nil {
		_ = s.conn.Close()
		s.conn = nil
	}

	// Mark detached and update last active so ResumeWindow starts now.
	wasDetached := s.detached.Load()
	now := time.Now()
	s.detached.Store(true)
	s.DetachedAt = now
	s.LastActive = now
	onDetach := s.onDetach
	s.mu.Unlock()

	if err != nil {
		s.logger.Info("session detached", "source", source, "error", err)
	} else {
		s.logger.Info("session detached", "source", source)
	}

	if !wasDetached && onDetach != nil {
		onDetach(s)
	}
}

// SendPatches is a public wrapper for sendPatches.
func (s *Session) SendPatches(patches []protocol.Patch) {
	s.mu.Lock()

	if s.closed.Load() {
		s.mu.Unlock()
		return
	}

	// Guard against nil connection (can happen in tests or edge cases)
	if s.conn == nil {
		s.logger.Warn("SendPatches: no connection available")
		s.mu.Unlock()
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
		s.mu.Unlock()
		s.Close()
		return
	}

	s.bytesSent.Add(uint64(len(frameData)))
	s.patchCount.Add(uint64(len(patches)))

	s.mu.Unlock()
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
