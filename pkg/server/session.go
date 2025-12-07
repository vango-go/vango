package server

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vango-dev/vango/v2/pkg/protocol"
	"github.com/vango-dev/vango/v2/pkg/vango"
	"github.com/vango-dev/vango/v2/pkg/vdom"
)

// Session represents a single WebSocket connection and its state.
// Each session has its own component tree, reactive ownership, and handler registry.
type Session struct {
	// Identity
	ID         string
	UserID     string
	CreatedAt  time.Time
	LastActive time.Time

	// Connection
	conn   *websocket.Conn
	mu     sync.Mutex // Protects conn writes
	closed atomic.Bool

	// Sequence numbers for reliable delivery
	sendSeq atomic.Uint64 // Next patch sequence to send
	recvSeq atomic.Uint64 // Last received event sequence
	ackSeq  atomic.Uint64 // Last acknowledged by client

	// Component state
	root       *ComponentInstance            // Root component
	components map[string]*ComponentInstance // HID -> component that owns element
	handlers   map[string]Handler            // HID -> event handler

	// Reactive ownership
	owner *vango.Owner

	// Rendering
	currentTree *vdom.VNode        // Last rendered tree
	hidGen      *vdom.HIDGenerator // Hydration ID generator

	// Channels
	events chan *Event   // Incoming events
	done   chan struct{} // Shutdown signal

	// Configuration
	config *SessionConfig

	// Logger
	logger *slog.Logger

	// Metrics
	eventCount atomic.Uint64
	patchCount atomic.Uint64
	bytesSent  atomic.Uint64
	bytesRecv  atomic.Uint64
}

// generateSessionID generates a cryptographically random session ID.
func generateSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// newSession creates a new session with the given connection.
func newSession(conn *websocket.Conn, userID string, config *SessionConfig, logger *slog.Logger) *Session {
	now := time.Now()
	id := generateSessionID()

	s := &Session{
		ID:         id,
		UserID:     userID,
		CreatedAt:  now,
		LastActive: now,
		conn:       conn,
		handlers:   make(map[string]Handler),
		components: make(map[string]*ComponentInstance),
		owner:      vango.NewOwner(nil),
		hidGen:     vdom.NewHIDGenerator(),
		events:     make(chan *Event, config.MaxEventQueue),
		done:       make(chan struct{}),
		config:     config,
		logger:     logger.With("session_id", id),
	}

	return s
}

// MountRoot mounts the root component for this session.
func (s *Session) MountRoot(component Component) {
	// Create root component instance
	s.root = newComponentInstance(component, nil, s)
	s.root.InstanceID = "root"

	// Render the component tree
	tree := s.root.Render()

	// Assign hydration IDs to interactive elements
	vdom.AssignHIDs(tree, s.hidGen)

	// Collect handlers from the tree
	s.collectHandlers(tree, s.root)

	// Store the tree
	s.currentTree = tree
	s.root.HID = tree.HID
	s.root.SetLastTree(tree)

	s.logger.Debug("mounted root component",
		"handlers", len(s.handlers),
		"components", len(s.components))
}

// collectHandlers walks the VNode tree and collects event handlers.
func (s *Session) collectHandlers(node *vdom.VNode, instance *ComponentInstance) {
	if node == nil {
		return
	}

	// If this node has an HID and event handlers, register them
	if node.HID != "" {
		for key, value := range node.Props {
			if strings.HasPrefix(key, "on") && value != nil {
				handler := wrapHandler(value)
				s.handlers[node.HID] = handler
				s.components[node.HID] = instance
			}
		}
	}

	// Recurse to children
	for _, child := range node.Children {
		if child.Kind == vdom.KindComponent && child.Comp != nil {
			// Mount child component
			childInstance := newComponentInstance(child.Comp, instance, s)
			instance.AddChild(childInstance)

			// Render child and collect its handlers
			childTree := childInstance.Render()
			vdom.AssignHIDs(childTree, s.hidGen)
			s.collectHandlers(childTree, childInstance)
		} else {
			s.collectHandlers(child, instance)
		}
	}
}

// handleEvent processes a single event from the client.
func (s *Session) handleEvent(event *Event) {
	// Update sequence tracking
	s.recvSeq.Store(event.Seq)
	s.eventCount.Add(1)
	s.LastActive = time.Now()

	// Find handler for this HID
	handler, exists := s.handlers[event.HID]
	if !exists {
		s.logger.Warn("handler not found", "hid", event.HID, "type", event.Type)
		s.sendErrorMessage(protocol.ErrHandlerNotFound, "Handler not found for HID: "+event.HID)
		return
	}

	// Execute handler with panic recovery
	s.safeExecute(handler, event)

	// Run pending effects (scheduled by signal updates)
	s.owner.RunPendingEffects()

	// Re-render dirty components
	s.renderDirty()
}

// safeExecute runs a handler with panic recovery.
func (s *Session) safeExecute(handler Handler, event *Event) {
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			s.logger.Error("handler panic",
				"panic", r,
				"hid", event.HID,
				"type", event.Type,
				"stack", string(stack))

			// Create handler error for logging/metrics
			_ = NewHandlerError(s.ID, event.HID, event.Type.String(), r, stack)

			// Send error to client
			s.sendErrorMessage(protocol.ErrHandlerPanic, "Internal error")
		}
	}()

	handler(event)
}

// renderDirty re-renders all dirty components and sends patches.
func (s *Session) renderDirty() {
	// Collect dirty components
	var dirty []*ComponentInstance
	for _, comp := range s.components {
		if comp.IsDirty() {
			dirty = append(dirty, comp)
			comp.ClearDirty()
		}
	}

	// Also check root
	if s.root != nil && s.root.IsDirty() {
		dirty = append(dirty, s.root)
		s.root.ClearDirty()
	}

	if len(dirty) == 0 {
		return
	}

	// Re-render each dirty component
	var allPatches []vdom.Patch
	for _, comp := range dirty {
		patches := s.renderComponent(comp)
		allPatches = append(allPatches, patches...)
	}

	// Send all patches
	if len(allPatches) > 0 {
		s.sendPatches(allPatches)
	}
}

// renderComponent re-renders a single component and returns patches.
func (s *Session) renderComponent(comp *ComponentInstance) []vdom.Patch {
	// Get old tree
	oldTree := comp.LastTree()

	// Render new tree
	newTree := comp.Render()

	// Assign HIDs to new nodes
	vdom.AssignHIDs(newTree, s.hidGen)

	// Diff old and new
	patches := vdom.Diff(oldTree, newTree)

	// Update stored tree
	comp.SetLastTree(newTree)

	// Re-collect handlers (they may have changed)
	s.clearComponentHandlers(comp)
	s.collectHandlers(newTree, comp)

	return patches
}

// clearComponentHandlers removes handlers for a component.
func (s *Session) clearComponentHandlers(comp *ComponentInstance) {
	for hid, c := range s.components {
		if c == comp {
			delete(s.handlers, hid)
			delete(s.components, hid)
		}
	}
}

// scheduleRender is called when a component marks itself dirty.
// For now this is a no-op since renderDirty() iterates all components
// checking their dirty flags. In the future, this could be optimized
// to maintain a set of dirty components for more efficient re-rendering.
func (s *Session) scheduleRender(comp *ComponentInstance) {
	// The component has already marked itself dirty.
	// renderDirty() will pick it up during the next render pass.
	// Future optimization: maintain a dirty set for O(1) lookup.
}

// sendPatches encodes and sends patches to the client.
func (s *Session) sendPatches(vdomPatches []vdom.Patch) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed.Load() {
		return
	}

	// Increment sequence number
	seq := s.sendSeq.Add(1)

	// Convert vdom patches to protocol patches
	protocolPatches := s.convertPatches(vdomPatches)

	// Create patches frame
	pf := &protocol.PatchesFrame{
		Seq:     seq,
		Patches: protocolPatches,
	}

	// Encode payload
	payload := protocol.EncodePatches(pf)

	// Create frame
	frame := protocol.NewFrame(protocol.FramePatches, payload)

	// Encode once for sending
	frameData := frame.Encode()

	// Set write deadline
	s.conn.SetWriteDeadline(time.Now().Add(s.config.WriteTimeout))

	// Write to WebSocket
	err := s.conn.WriteMessage(websocket.BinaryMessage, frameData)
	if err != nil {
		s.logger.Error("write error", "error", err)
		s.closeInternal()
		return
	}

	// Update metrics
	s.bytesSent.Add(uint64(len(frameData)))
	s.patchCount.Add(uint64(len(protocolPatches)))

	s.logger.Debug("sent patches",
		"seq", seq,
		"count", len(protocolPatches),
		"bytes", len(frameData))
}

// convertPatches converts vdom.Patch to protocol.Patch.
func (s *Session) convertPatches(vdomPatches []vdom.Patch) []protocol.Patch {
	result := make([]protocol.Patch, len(vdomPatches))

	for i, p := range vdomPatches {
		result[i] = protocol.Patch{
			Op:       protocol.PatchOp(p.Op),
			HID:      p.HID,
			Key:      p.Key,
			Value:    p.Value,
			ParentID: p.ParentID,
			Index:    p.Index,
		}

		// Convert VNode to wire format for InsertNode/ReplaceNode
		if p.Node != nil {
			result[i].Node = protocol.VNodeToWire(p.Node)
		}
	}

	return result
}

// sendErrorMessage sends an error frame to the client.
func (s *Session) sendErrorMessage(code protocol.ErrorCode, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed.Load() {
		return
	}

	errMsg := protocol.NewError(code, message)
	payload := protocol.EncodeErrorMessage(errMsg)
	frame := protocol.NewFrame(protocol.FrameError, payload)

	s.conn.SetWriteDeadline(time.Now().Add(s.config.WriteTimeout))
	s.conn.WriteMessage(websocket.BinaryMessage, frame.Encode())
}

// sendPing sends a heartbeat ping to the client.
func (s *Session) sendPing() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed.Load() {
		return ErrSessionClosed
	}

	ct, pp := protocol.NewPing(uint64(time.Now().UnixMilli()))
	payload := protocol.EncodeControl(ct, pp)
	frame := protocol.NewFrame(protocol.FrameControl, payload)

	s.conn.SetWriteDeadline(time.Now().Add(s.config.WriteTimeout))
	err := s.conn.WriteMessage(websocket.BinaryMessage, frame.Encode())
	if err != nil {
		s.logger.Error("ping error", "error", err)
		return err
	}

	return nil
}

// Close gracefully closes the session.
func (s *Session) Close() {
	if s.closed.Swap(true) {
		// Already closed
		return
	}

	s.closeInternal()
}

// closeInternal performs the actual close operations.
func (s *Session) closeInternal() {
	// Signal shutdown to goroutines
	select {
	case <-s.done:
		// Already closed
	default:
		close(s.done)
	}

	// Dispose reactive owner (cleans up effects and signals)
	if s.owner != nil {
		s.owner.Dispose()
	}

	// Dispose root component
	if s.root != nil {
		s.root.Dispose()
	}

	// Clear handlers
	s.handlers = nil
	s.components = nil

	// Send close message and close WebSocket
	if s.conn != nil {
		s.conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(time.Second),
		)
		s.conn.Close()
	}

	s.logger.Info("session closed",
		"events", s.eventCount.Load(),
		"patches", s.patchCount.Load(),
		"bytes_sent", s.bytesSent.Load(),
		"bytes_recv", s.bytesRecv.Load())
}

// IsClosed returns whether the session is closed.
func (s *Session) IsClosed() bool {
	return s.closed.Load()
}

// Done returns a channel that's closed when the session is done.
func (s *Session) Done() <-chan struct{} {
	return s.done
}

// Events returns the events channel for the event loop.
func (s *Session) Events() <-chan *Event {
	return s.events
}

// QueueEvent queues an event for processing.
func (s *Session) QueueEvent(event *Event) error {
	select {
	case s.events <- event:
		return nil
	default:
		s.logger.Warn("event queue full, dropping event", "hid", event.HID)
		return ErrEventQueueFull
	}
}

// UpdateLastActive updates the last activity timestamp.
func (s *Session) UpdateLastActive() {
	s.LastActive = time.Now()
}

// Stats returns session statistics.
func (s *Session) Stats() SessionStats {
	return SessionStats{
		ID:             s.ID,
		UserID:         s.UserID,
		CreatedAt:      s.CreatedAt,
		LastActive:     s.LastActive,
		EventCount:     s.eventCount.Load(),
		PatchCount:     s.patchCount.Load(),
		BytesSent:      s.bytesSent.Load(),
		BytesRecv:      s.bytesRecv.Load(),
		HandlerCount:   len(s.handlers),
		ComponentCount: len(s.components),
	}
}

// SessionStats contains session statistics.
type SessionStats struct {
	ID             string
	UserID         string
	CreatedAt      time.Time
	LastActive     time.Time
	EventCount     uint64
	PatchCount     uint64
	BytesSent      uint64
	BytesRecv      uint64
	HandlerCount   int
	ComponentCount int
}

// MemoryUsage estimates the memory used by this session.
func (s *Session) MemoryUsage() int64 {
	var size int64 = 512 // Base struct size

	// Handlers map
	size += int64(len(s.handlers)) * 64

	// Components map
	for _, comp := range s.components {
		size += comp.MemoryUsage()
	}

	// Current tree
	if s.currentTree != nil {
		size += estimateVNodeSize(s.currentTree)
	}

	// Events channel buffer (estimate)
	size += int64(cap(s.events)) * 64

	return size
}

// Conn returns the underlying WebSocket connection.
// Use with caution - prefer session methods when possible.
func (s *Session) Conn() *websocket.Conn {
	return s.conn
}

// Config returns the session configuration.
func (s *Session) Config() *SessionConfig {
	return s.config
}

// Logger returns the session logger.
func (s *Session) Logger() *slog.Logger {
	return s.logger
}

// Owner returns the reactive owner for this session.
func (s *Session) Owner() *vango.Owner {
	return s.owner
}

// BytesReceived adds to the bytes received counter.
func (s *Session) BytesReceived(n int) {
	s.bytesRecv.Add(uint64(n))
}
