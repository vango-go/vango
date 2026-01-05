package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vango-go/vango/pkg/features/store"
	"github.com/vango-go/vango/pkg/protocol"
	"github.com/vango-go/vango/pkg/render"
	"github.com/vango-go/vango/pkg/session"
	"github.com/vango-go/vango/pkg/urlparam"
	"github.com/vango-go/vango/pkg/vango"
	"github.com/vango-go/vango/pkg/vdom"
)

// DebugMode enables extra validation and logging for development.
// When true:
// - Session.Set() panics on unserializable types (func, chan)
// - auth.Get() logs warnings on type mismatches
// Set via ServerConfig.DebugMode or directly for testing.
var DebugMode bool

// checkSerializable panics if value is an obviously unserializable type.
// Only called when DebugMode is true.
func checkSerializable(key string, value any) {
	t := reflect.TypeOf(value)
	if t == nil {
		return
	}
	switch t.Kind() {
	case reflect.Func:
		panic(fmt.Sprintf("Session.Set(%q): cannot store func (unserializable for distributed sessions)", key))
	case reflect.Chan:
		panic(fmt.Sprintf("Session.Set(%q): cannot store chan (unserializable for distributed sessions)", key))
	}
}

// Session represents a single WebSocket connection and its state.
// Each session has its own component tree, reactive ownership, and handler registry.
type Session struct {
	// Identity
	ID         string
	UserID     string
	CreatedAt  time.Time
	LastActive time.Time

	// Phase 12: Session persistence fields
	IP           string // Client IP address for per-IP limits
	CurrentRoute string // Current page route for restoration

	// Connection
	conn   *websocket.Conn
	mu     sync.Mutex // Protects conn writes
	closed atomic.Bool

	// Sequence numbers for reliable delivery
	sendSeq atomic.Uint64 // Next patch sequence to send
	recvSeq atomic.Uint64 // Last received event sequence
	ackSeq  atomic.Uint64 // Last acknowledged by client

	// Component state
	root          *ComponentInstance              // Root component
	allComponents map[*ComponentInstance]struct{} // ALL mounted components (for dirty checking)
	components    map[string]*ComponentInstance   // HID -> component that owns element
	handlers      map[string]Handler              // HID_eventType -> event handler

	// Reactive ownership
	owner *vango.Owner

	// Rendering
	currentTree *vdom.VNode        // Last rendered tree
	hidGen      *vdom.HIDGenerator // Hydration ID generator

	// Channels
	events     chan *Event    // Incoming events
	renderCh   chan struct{}  // Signal for re-render
	dispatchCh chan func()    // Functions to run on event loop (ctx.Dispatch)
	done       chan struct{}  // Shutdown signal

	// Configuration
	config *SessionConfig

	// Logger
	logger *slog.Logger

	// Metrics
	eventCount atomic.Uint64
	patchCount atomic.Uint64
	bytesSent  atomic.Uint64
	bytesRecv  atomic.Uint64

	// General-purpose session data storage (Phase 10)
	// Use Get/Set/Delete to access. Protected by dataMu.
	data   map[string]any
	dataMu sync.RWMutex

	// URL patch buffering (Phase 12: URLParam 2.0)
	// URL patches are queued here and sent along with DOM patches.
	pendingURLPatches []protocol.Patch
	urlPatchMu        sync.Mutex

	// Storm budget tracker (Phase 16)
	stormBudget *vango.StormBudgetTracker
}

// generateSessionID generates a cryptographically random session ID.
func generateSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// SECURITY: Fatal on entropy failure - weak IDs are dangerous
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}
	return hex.EncodeToString(b)
}

// newSession creates a new session with the given connection.
func newSession(conn *websocket.Conn, userID string, config *SessionConfig, logger *slog.Logger) *Session {
	now := time.Now()
	id := generateSessionID()

	s := &Session{
		ID:            id,
		UserID:        userID,
		CreatedAt:     now,
		LastActive:    now,
		conn:          conn,
		allComponents: make(map[*ComponentInstance]struct{}),
		handlers:      make(map[string]Handler),
		components:    make(map[string]*ComponentInstance),
		owner:         vango.NewOwner(nil),
		hidGen:        vdom.NewHIDGenerator(),
		events:        make(chan *Event, config.MaxEventQueue),
		renderCh:      make(chan struct{}, 1),
		dispatchCh:    make(chan func(), config.MaxEventQueue),
		done:          make(chan struct{}),
		config:        config,
		logger:        logger.With("session_id", id),
		stormBudget:   createStormBudgetTracker(config.StormBudget),
	}

	// Initialize session-scoped store for SharedSignal support.
	// This enables store.Shared[T] signals to work without manual context setup.
	s.owner.SetValue(store.SessionKey, store.NewSessionStore())

	// Initialize URL navigator for URLParam support.
	// URLParam.Set() will queue patches via queueURLPatch, which are then
	// sent along with DOM patches in renderDirty().
	navigator := urlparam.NewNavigator(s.queueURLPatch)
	s.owner.SetValue(urlparam.NavigatorKey, navigator)

	return s
}

// MountRoot mounts the root component for this session.
func (s *Session) MountRoot(component Component) {
	// Create root component instance
	s.root = newComponentInstance(component, nil, s)
	s.root.InstanceID = "root"

	// Register root component for dirty tracking
	s.registerComponent(s.root)

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

	s.logger.Info("mounted root component",
		"handlers", len(s.handlers),
		"components", len(s.components),
		"hid_counter", s.hidGen.Current())
}

// collectHandlers walks the VNode tree and collects event handlers.
func (s *Session) collectHandlers(node *vdom.VNode, instance *ComponentInstance) {
	if node == nil {
		return
	}

	// If this node has an HID, check for event handlers
	if node.HID != "" {
		// Track that this component owns this HID (for cleanup and lookup)
		s.components[node.HID] = instance

		for key, value := range node.Props {
			if value == nil {
				continue
			}

			// Case 1: onhook handlers (from hooks.OnEvent, can be single or slice)
			if key == "onhook" {
				handlerKey := node.HID + "_onhook"

				// Handle both single handler and slice of handlers
				var handlers []Handler
				if slice, ok := value.([]any); ok {
					// Multiple merged handlers
					for _, h := range slice {
						handlers = append(handlers, wrapHandler(h))
					}
				} else {
					// Single handler
					handlers = append(handlers, wrapHandler(value))
				}

				// Create a combined handler that calls all wrapped handlers
				combinedHandler := func(e *Event) {
					for _, h := range handlers {
						h(e)
					}
				}
				s.handlers[handlerKey] = combinedHandler

				if DebugMode {
					fmt.Printf("[HANDLER] Registered onhook on %s (%s) -> key=%s (%d handlers)\n", node.HID, node.Tag, handlerKey, len(handlers))
				}
				continue
			}

			// Case 2: Standard on* handlers (onclick, oninput, etc.)
			if strings.HasPrefix(key, "on") {
				handler := wrapHandler(value)
				// Use compound key: HID_eventType (e.g., "h1_onclick")
				eventType := strings.ToLower(key) // "onclick", "onmouseenter"
				handlerKey := node.HID + "_" + eventType
				s.handlers[handlerKey] = handler
				if DebugMode {
					fmt.Printf("[HANDLER] Registered %s on %s (%s) -> key=%s\n", key, node.HID, node.Tag, handlerKey)
				}
				continue
			}

			// Case 3: EventHandler struct (DEPRECATED - legacy support for vdom.EventHandler)
			if eh, ok := value.(vdom.EventHandler); ok {
				handler := wrapHandler(eh.Handler)
				// Hook events use key: HID_hook_eventName (e.g., "h1_hook_reorder")
				handlerKey := node.HID + "_hook_" + eh.Event
				s.handlers[handlerKey] = handler
				if DebugMode {
					fmt.Printf("[HANDLER] Registered hook event %s on %s (%s) -> key=%s\n", eh.Event, node.HID, node.Tag, handlerKey)
				}
			}
		}
	}

	// Recurse to children
	for _, child := range node.Children {
		if child.Kind == vdom.KindComponent && child.Comp != nil {
			// Mount child component
			childInstance := newComponentInstance(child.Comp, instance, s)
			instance.AddChild(childInstance)

			// Register for dirty tracking
			s.registerComponent(childInstance)

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

	if DebugMode {
		fmt.Printf("[EVENT] Received: HID=%s Type=%v Seq=%d\n", event.HID, event.Type, event.Seq)
	}

	// Build handler key based on event type
	var handlerKey string
	if _, ok := event.Payload.(*protocol.HookEventData); ok {
		// Hook events use new unified key: HID_onhook
		// The wrapped handlers filter by event name internally
		handlerKey = event.HID + "_onhook"
	} else {
		// Standard events keyed by: HID_oneventtype (e.g., "h1_onclick")
		eventType := "on" + strings.ToLower(event.Type.String())
		handlerKey = event.HID + "_" + eventType
	}

	// Find handler using compound key
	handler, exists := s.handlers[handlerKey]
	if !exists {
		s.logger.Warn("handler not found", "hid", event.HID, "type", event.Type, "key", handlerKey)
		s.sendErrorMessage(protocol.ErrHandlerNotFound, "Handler not found: "+handlerKey)
		return
	}

	if DebugMode {
		fmt.Printf("[EVENT] Handler found for %s (key=%s), executing...\n", event.HID, handlerKey)
	}

	// Create context for this event so UseCtx() works in handlers
	ctx := s.createEventContext(event)

	// Execute handler and effects with context set
	vango.WithCtx(ctx, func() {
		// Execute handler with panic recovery
		s.safeExecute(handler, event)

		// Run pending effects (scheduled by signal updates)
		s.owner.RunPendingEffects()
	})

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

// registerComponent adds a component to the allComponents set for dirty tracking.
func (s *Session) registerComponent(comp *ComponentInstance) {
	s.allComponents[comp] = struct{}{}
}

// unregisterComponent removes a component from the allComponents set.
func (s *Session) unregisterComponent(comp *ComponentInstance) {
	delete(s.allComponents, comp)
}

// renderDirty re-renders all dirty components and sends patches.
func (s *Session) renderDirty() {
	// Collect dirty components from ALL components (not just those with handlers)
	var dirty []*ComponentInstance
	for comp := range s.allComponents {
		if comp.IsDirty() {
			dirty = append(dirty, comp)
			comp.ClearDirty()
		}
	}

	// Also check root if not in allComponents (edge case)
	if s.root != nil && s.root.IsDirty() {
		dirty = append(dirty, s.root)
		s.root.ClearDirty()
	}

	if DebugMode && len(dirty) == 0 {
		fmt.Println("[DEBUG] renderDirty: no dirty components")
	}

	if DebugMode && len(dirty) > 0 {
		fmt.Printf("[DEBUG] renderDirty: %d dirty components\n", len(dirty))
	}

	// Re-render each dirty component
	var allPatches []vdom.Patch
	for _, comp := range dirty {
		patches := s.renderComponent(comp)
		if DebugMode {
			fmt.Printf("[DEBUG] renderComponent returned %d patches\n", len(patches))
		}
		allPatches = append(allPatches, patches...)
	}

	// Drain any pending URL patches
	urlPatches := s.drainURLPatches()

	// Send all patches (DOM + URL)
	if DebugMode {
		fmt.Printf("[DEBUG] renderDirty: sending %d DOM patches, %d URL patches\n", len(allPatches), len(urlPatches))
	}
	if len(allPatches) > 0 || len(urlPatches) > 0 {
		s.sendPatchesWithURL(allPatches, urlPatches)
	}
}

// renderComponent re-renders a single component and returns patches.
func (s *Session) renderComponent(comp *ComponentInstance) []vdom.Patch {
	// Get old tree
	oldTree := comp.LastTree()

	// Render new tree
	newTree := comp.Render()

	// Try to copy HIDs from old tree to preserve them
	// If structure changed significantly, this will return false for some nodes
	// and we'll assign new HIDs to those
	if oldTree != nil {
		vdom.CopyHIDs(oldTree, newTree)
	}

	// Assign HIDs to any new nodes that didn't get one from CopyHIDs
	// This handles new elements added to the tree
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
// This clears all handlers with keys prefixed by the component's HIDs.
func (s *Session) clearComponentHandlers(comp *ComponentInstance) {
	// First, collect all HIDs owned by this component
	var hidsToRemove []string
	for hid, c := range s.components {
		if c == comp {
			hidsToRemove = append(hidsToRemove, hid)
		}
	}

	// Then remove the HIDs and all associated handlers
	for _, hid := range hidsToRemove {
		delete(s.components, hid)
		// Delete ALL handlers with this HID as prefix (e.g., "h1_onclick", "h1_hook_reorder")
		prefix := hid + "_"
		for key := range s.handlers {
			if strings.HasPrefix(key, prefix) {
				delete(s.handlers, key)
			}
		}
	}
}

// =============================================================================
// Session Resume: Soft Remount (Phase 5)
// =============================================================================

// RebuildHandlers clears and rebuilds the handler map with fresh HIDs.
// This is called on session resume to match SSR-rendered DOM.
//
// Unlike a full remount, this preserves:
//   - Owner (signals stay alive with their values)
//   - Component instances (state preserved)
//
// It regenerates:
//   - HID assignments (reset to h1, h2... matching SSR)
//   - Handler map (rebuilt from fresh render)
//   - Component ownership map
func (s *Session) RebuildHandlers() error {
	if s.root == nil {
		return fmt.Errorf("no root component to rebuild")
	}

	// 1. Clear handlers and component mappings (NOT owner!)
	s.handlers = make(map[string]Handler)
	s.components = make(map[string]*ComponentInstance)

	// 2. Reset HID generator to 0 (will produce h1, h2... matching SSR)
	s.hidGen.Reset()

	// 3. Re-render existing component tree (signals still alive, so same values)
	tree := s.rerenderTree(s.root)

	// 4. Assign fresh HIDs
	vdom.AssignHIDs(tree, s.hidGen)

	// 5. Collect handlers from fresh render (use existing instances)
	s.collectHandlersPreserving(tree, s.root)

	// 6. Store new tree
	s.currentTree = tree
	s.root.HID = tree.HID
	s.root.SetLastTree(tree)

	s.logger.Info("handlers rebuilt",
		"handlers", len(s.handlers),
		"components", len(s.components),
		"hid_counter", s.hidGen.Current())

	return nil
}

// rerenderTree recursively re-renders all components in the tree.
// This uses existing component instances (preserving their state).
func (s *Session) rerenderTree(instance *ComponentInstance) *vdom.VNode {
	tree := instance.Render()

	// Recursively handle child components in the rendered tree
	s.rerenderChildren(tree, instance)

	return tree
}

// rerenderChildren walks the tree and re-renders child component instances.
// When it finds a KindComponent node, it looks up the existing child instance
// and renders it instead of creating a new one.
func (s *Session) rerenderChildren(node *vdom.VNode, parent *ComponentInstance) {
	if node == nil {
		return
	}

	for i, child := range node.Children {
		if child.Kind == vdom.KindComponent && child.Comp != nil {
			// Find existing child instance that matches this component
			var existingChild *ComponentInstance
			for _, existing := range parent.Children {
				// Match by component type (pointer equality or interface match)
				if existing.Component == child.Comp {
					existingChild = existing
					break
				}
			}

			if existingChild != nil {
				// Re-render existing instance (preserving its state)
				rendered := s.rerenderTree(existingChild)
				node.Children[i] = rendered
			} else {
				// No existing instance found - this is a new component
				// For soft remount, we should ideally skip this or handle gracefully
				// For now, log a warning and render it fresh
				s.logger.Warn("no existing child instance found during rebuild",
					"parent", parent.InstanceID,
					"component_type", fmt.Sprintf("%T", child.Comp))

				// Create new instance for the new component
				childInstance := newComponentInstance(child.Comp, parent, s)
				parent.AddChild(childInstance)
				s.registerComponent(childInstance)
				rendered := s.rerenderTree(childInstance)
				node.Children[i] = rendered
			}
		} else {
			// Recurse into regular elements
			s.rerenderChildren(child, parent)
		}
	}
}

// collectHandlersPreserving walks the tree and collects handlers,
// using existing component instances instead of creating new ones.
// This is similar to collectHandlers but for the resume path.
func (s *Session) collectHandlersPreserving(node *vdom.VNode, instance *ComponentInstance) {
	if node == nil {
		return
	}

	// If this node has an HID, check for event handlers
	if node.HID != "" {
		// Track that this component owns this HID (for cleanup and lookup)
		s.components[node.HID] = instance

		for key, value := range node.Props {
			if value == nil {
				continue
			}

			// Case 1: onhook handlers (from hooks.OnEvent)
			if key == "onhook" {
				handlerKey := node.HID + "_onhook"

				var handlers []Handler
				if slice, ok := value.([]any); ok {
					for _, h := range slice {
						handlers = append(handlers, wrapHandler(h))
					}
				} else {
					handlers = append(handlers, wrapHandler(value))
				}

				combinedHandler := func(e *Event) {
					for _, h := range handlers {
						h(e)
					}
				}
				s.handlers[handlerKey] = combinedHandler
				continue
			}

			// Case 2: Standard on* handlers (onclick, oninput, etc.)
			if strings.HasPrefix(key, "on") {
				handler := wrapHandler(value)
				eventType := strings.ToLower(key)
				handlerKey := node.HID + "_" + eventType
				s.handlers[handlerKey] = handler
				continue
			}

			// Case 3: EventHandler struct (DEPRECATED - legacy support)
			if eh, ok := value.(vdom.EventHandler); ok {
				handler := wrapHandler(eh.Handler)
				handlerKey := node.HID + "_hook_" + eh.Event
				s.handlers[handlerKey] = handler
			}
		}
	}

	// Recurse to children - but don't create new component instances
	for _, child := range node.Children {
		// For component nodes, we need to find the right instance
		if child.Kind == vdom.KindComponent && child.Comp != nil {
			// Find the child instance that owns this subtree
			for _, childInstance := range instance.Children {
				if childInstance.Component == child.Comp {
					// Collect handlers from the child's last rendered tree
					if childInstance.LastTree() != nil {
						s.collectHandlersPreserving(childInstance.LastTree(), childInstance)
					}
					break
				}
			}
		} else {
			s.collectHandlersPreserving(child, instance)
		}
	}
}

// SendResyncFull sends the full HTML tree to the client.
// This is used as a fallback when HIDs may not align between SSR and remount.
// The client will replace its entire body content with this HTML.
func (s *Session) SendResyncFull() error {
	if s.currentTree == nil {
		return errors.New("no tree to send")
	}

	// Render tree to HTML using the render package
	// The tree already has HIDs assigned from RebuildHandlers()
	renderer := render.NewRenderer(render.RendererConfig{})
	html, err := renderer.RenderToString(s.currentTree)
	if err != nil {
		return fmt.Errorf("render tree to HTML: %w", err)
	}

	// Send via ResyncFull control message (protocol type 0x12)
	ct, rr := protocol.NewResyncFull(html)
	payload := protocol.EncodeControl(ct, rr)
	frame := protocol.NewFrame(protocol.FrameControl, payload)

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed.Load() || s.conn == nil {
		return ErrSessionClosed
	}

	s.conn.SetWriteDeadline(time.Now().Add(s.config.WriteTimeout))
	if err := s.conn.WriteMessage(websocket.BinaryMessage, frame.Encode()); err != nil {
		return fmt.Errorf("write resync full: %w", err)
	}

	s.logger.Debug("sent ResyncFull",
		"html_size", len(html),
		"frame_size", len(frame.Encode()))

	return nil
}

// scheduleRender is called when a component marks itself dirty.
// For now this is a no-op since renderDirty() iterates all components
// checking their dirty flags. In the future, this could be optimized
// to maintain a set of dirty components for more efficient re-rendering.
func (s *Session) scheduleRender(comp *ComponentInstance) {
	// The component has already marked itself dirty.
	// We notify the event loop to run a render pass.
	select {
	case s.renderCh <- struct{}{}:
	default:
		// Already scheduled
	}
}

// sendPatches encodes and sends patches to the client.
func (s *Session) sendPatches(vdomPatches []vdom.Patch) {
	s.sendPatchesWithURL(vdomPatches, nil)
}

// sendPatchesWithURL encodes and sends DOM and URL patches to the client.
// URL patches (if any) are appended after DOM patches in the same frame.
func (s *Session) sendPatchesWithURL(vdomPatches []vdom.Patch, urlPatches []protocol.Patch) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed.Load() {
		return
	}

	// Guard against nil connection (can happen in tests or edge cases)
	if s.conn == nil {
		s.logger.Warn("sendPatches: no connection available")
		return
	}

	// Increment sequence number
	seq := s.sendSeq.Add(1)

	// Convert vdom patches to protocol patches
	protocolPatches := s.convertPatches(vdomPatches)

	// Append any URL patches
	if len(urlPatches) > 0 {
		protocolPatches = append(protocolPatches, urlPatches...)
	}

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

	// Guard against nil connection (can happen in tests or edge cases)
	if s.conn == nil {
		s.logger.Warn("sendErrorMessage: no connection available",
			"code", code,
			"message", message)
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

	// Guard against nil connection
	if s.conn == nil {
		return ErrNoConnection
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

	// Clear handlers and components
	s.handlers = nil
	s.components = nil
	s.allComponents = nil

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

// Dispatch queues a function to run on the session's event loop.
// This is safe to call from any goroutine and is the correct way to
// update signals from asynchronous operations (database calls, timers, etc.).
//
// The function will be executed synchronously on the event loop, ensuring
// signal writes are properly serialized. After the function completes,
// pending effects will run and dirty components will re-render.
//
// Example:
//
//	go func() {
//	    user, err := db.Users.FindByID(ctx.StdContext(), id)
//	    ctx.Dispatch(func() {
//	        if err != nil {
//	            errorSignal.Set(err)
//	        } else {
//	            userSignal.Set(user)
//	        }
//	    })
//	}()
func (s *Session) Dispatch(fn func()) {
	if s.closed.Load() {
		return
	}
	select {
	case s.dispatchCh <- fn:
		// Successfully queued
	case <-s.done:
		// Session is closing, discard
	default:
		// Queue full - this shouldn't happen normally, but log it
		s.logger.Warn("dispatch queue full, discarding callback")
	}
}

// UpdateLastActive updates the last activity timestamp.
func (s *Session) UpdateLastActive() {
	s.LastActive = time.Now()
}

// =============================================================================
// URLParam Support (Phase 12: URLParam 2.0)
// =============================================================================

// queueURLPatch adds a URL patch to the pending buffer.
// URL patches are sent along with DOM patches in renderDirty().
// This is called by the Navigator when URLParam.Set() is invoked.
func (s *Session) queueURLPatch(patch protocol.Patch) {
	s.urlPatchMu.Lock()
	defer s.urlPatchMu.Unlock()
	s.pendingURLPatches = append(s.pendingURLPatches, patch)
}

// drainURLPatches returns and clears the pending URL patches.
// This is called by sendPatches to include URL updates with DOM patches.
func (s *Session) drainURLPatches() []protocol.Patch {
	s.urlPatchMu.Lock()
	defer s.urlPatchMu.Unlock()
	patches := s.pendingURLPatches
	s.pendingURLPatches = nil
	return patches
}

// SetInitialURL sets the initial URL parameters from the client handshake.
// This is called when processing the initial handshake message from the client.
// The URLParam hook will consume these parameters once on first render.
func (s *Session) SetInitialURL(path string, params map[string]string) {
	s.owner.SetValue(urlparam.InitialParamsKey, &urlparam.InitialURLState{
		Path:   path,
		Params: params,
	})
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

// createEventContext creates a Ctx for event handling.
// This context is set via vango.WithCtx so UseCtx() works in handlers.
func (s *Session) createEventContext(event *Event) Ctx {
	return &ctx{
		session: s,
		event:   event,
		logger:  s.logger,
		stdCtx:  context.Background(),
	}
}

// createRenderContext creates a Ctx for component rendering.
// This context is set via vango.WithCtx so UseCtx() works during render.
func (s *Session) createRenderContext() Ctx {
	return &ctx{
		session: s,
		logger:  s.logger,
		stdCtx:  context.Background(),
	}
}

// =============================================================================
// Session State API (Phase 10)
// =============================================================================

// Get retrieves a value from session data.
// Returns nil if key doesn't exist.
// This is thread-safe and can be called from any goroutine.
func (s *Session) Get(key string) any {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()
	return s.data[key]
}

// Set stores a value in session data.
// Value must be safe to access concurrently (immutable or properly synchronized).
//
// WARNING: For future Redis/distributed session support, stored values should
// be JSON-serializable. Avoid storing functions, channels, or complex structs.
// In debug mode (ServerConfig.DebugMode = true), this will panic on obviously
// unserializable types like func and chan.
func (s *Session) Set(key string, value any) {
	s.dataMu.Lock()
	defer s.dataMu.Unlock()
	if s.data == nil {
		s.data = make(map[string]any)
	}

	// Debug mode: check for obviously unserializable types
	if DebugMode && value != nil {
		checkSerializable(key, value)
	}

	s.data[key] = value
}

// SetString stores a string value (always serializable).
func (s *Session) SetString(key string, value string) {
	s.Set(key, value)
}

// SetInt stores an int value (always serializable).
func (s *Session) SetInt(key string, value int) {
	s.Set(key, value)
}

// SetJSON stores a JSON-serializable struct.
// This is equivalent to Set() but documents the intent that the value
// should be JSON-serializable for future distributed session support.
func (s *Session) SetJSON(key string, value any) {
	s.Set(key, value)
}

// Delete removes a key from session data.
func (s *Session) Delete(key string) {
	s.dataMu.Lock()
	defer s.dataMu.Unlock()
	delete(s.data, key)
}

// GetString is a convenience method that returns value as string.
// Returns empty string if key doesn't exist or value is not a string.
func (s *Session) GetString(key string) string {
	if v, ok := s.Get(key).(string); ok {
		return v
	}
	return ""
}

// GetInt is a convenience method that returns value as int.
// Returns 0 if key doesn't exist or value is not numeric.
// Handles int, int64, and float64 conversions.
func (s *Session) GetInt(key string) int {
	switch v := s.Get(key).(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

// =============================================================================
// Storm Budgets (Phase 16)
// =============================================================================

// createStormBudgetTracker creates a storm budget tracker from server config.
// Returns nil if no storm budget config is provided.
func createStormBudgetTracker(cfg *StormBudgetConfig) *vango.StormBudgetTracker {
	if cfg == nil {
		return nil
	}
	return vango.NewStormBudgetTracker(&vango.StormBudgetConfig{
		MaxResourceStartsPerSecond: cfg.MaxResourceStartsPerSecond,
		MaxActionStartsPerSecond:   cfg.MaxActionStartsPerSecond,
		MaxGoLatestStartsPerSecond: cfg.MaxGoLatestStartsPerSecond,
		MaxEffectRunsPerTick:       cfg.MaxEffectRunsPerTick,
		WindowDuration:             cfg.WindowDuration,
		OnExceeded:                 vango.BudgetExceededMode(cfg.OnExceeded),
	})
}

// StormBudget returns the storm budget checker for this session.
// Returns nil if storm budgets are not configured.
func (s *Session) StormBudget() vango.StormBudgetChecker {
	return s.stormBudget
}

// Has returns whether a key exists in session data.
func (s *Session) Has(key string) bool {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()
	_, ok := s.data[key]
	return ok
}

// GetAllData returns a copy of all session data for serialization.
// This is used during session persistence to save all key-value pairs.
// Returns nil if no data has been set.
func (s *Session) GetAllData() map[string]any {
	s.dataMu.RLock()
	defer s.dataMu.RUnlock()

	if s.data == nil {
		return nil
	}

	// Return a copy to prevent external mutations
	dataCopy := make(map[string]any, len(s.data))
	for k, v := range s.data {
		dataCopy[k] = v
	}
	return dataCopy
}

// RestoreData restores session data from serialized values.
// This is used during session restoration after server restart or reconnection.
// Values are merged into existing data (doesn't clear existing keys).
func (s *Session) RestoreData(values map[string]any) {
	if values == nil {
		return
	}

	s.dataMu.Lock()
	defer s.dataMu.Unlock()

	if s.data == nil {
		s.data = make(map[string]any)
	}
	for k, v := range values {
		s.data[k] = v
	}
}

// =============================================================================
// Session Serialization (Phase 12)
// =============================================================================

// Serialize converts the session state to bytes for persistence.
// This is called during graceful shutdown, disconnect, and periodic saves.
//
// The serialized state includes:
//   - Session ID and user ID
//   - Creation and last active timestamps
//   - Current route for page restoration
//   - All session data values (from Get/Set)
//
// Note: Signals are persisted separately when they have PersistKey options.
// Transient signals are not serialized.
func (s *Session) Serialize() ([]byte, error) {
	// Convert session data to JSON-friendly format
	var values map[string]json.RawMessage
	if data := s.GetAllData(); data != nil {
		values = make(map[string]json.RawMessage, len(data))
		for k, v := range data {
			b, err := json.Marshal(v)
			if err != nil {
				// Skip unserializable values
				continue
			}
			values[k] = b
		}
	}

	ss := &session.SerializableSession{
		ID:         s.ID,
		UserID:     s.UserID,
		CreatedAt:  s.CreatedAt,
		LastActive: s.LastActive,
		Values:     values,
		Route:      s.CurrentRoute,
	}

	return session.Serialize(ss)
}

// Deserialize restores session state from bytes.
// This is called when resuming a session after server restart or reconnection.
//
// The deserialized state is merged into the current session:
//   - ID is restored (should match existing ID from cookie)
//   - User ID is restored
//   - Timestamps are restored
//   - Current route is restored for page navigation
//   - All session data values are restored
//
// Note: Signals are restored on-demand when components re-render and
// call NewSignal with PersistKey options. The restored signal values
// are matched by their persist keys.
func (s *Session) Deserialize(data []byte) error {
	ss, err := session.Deserialize(data)
	if err != nil {
		return err
	}

	// Restore identity fields
	s.ID = ss.ID
	s.UserID = ss.UserID
	s.CreatedAt = ss.CreatedAt
	s.LastActive = ss.LastActive
	s.CurrentRoute = ss.Route

	// Restore session data values (converting json.RawMessage to any)
	if ss.Values != nil {
		values := make(map[string]any, len(ss.Values))
		for k, v := range ss.Values {
			var val any
			if err := json.Unmarshal(v, &val); err == nil {
				values[k] = val
			}
		}
		s.RestoreData(values)
	}

	return nil
}

// =============================================================================
// Test Helpers (Phase 10F)
// =============================================================================

// NewMockSession creates a session without a WebSocket connection for testing.
// The session has all fields initialized except conn.
func NewMockSession() *Session {
	return &Session{
		ID:            "test-session-id",
		UserID:        "",
		CreatedAt:     time.Now(),
		LastActive:    time.Now(),
		allComponents: make(map[*ComponentInstance]struct{}),
		handlers:      make(map[string]Handler),
		components:    make(map[string]*ComponentInstance),
		owner:         vango.NewOwner(nil),
		hidGen:        vdom.NewHIDGenerator(),
		events:        make(chan *Event, 256),
		renderCh:      make(chan struct{}, 1),
		dispatchCh:    make(chan func(), 256),
		done:          make(chan struct{}),
		config:        DefaultSessionConfig(),
		logger:        slog.Default().With("session_id", "test-session-id"),
		data:          make(map[string]any),
	}
}
