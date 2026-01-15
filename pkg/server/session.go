package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"reflect"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vango-go/vango/pkg/assets"
	"github.com/vango-go/vango/pkg/features/store"
	"github.com/vango-go/vango/pkg/protocol"
	"github.com/vango-go/vango/pkg/render"
	"github.com/vango-go/vango/pkg/routepath"
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

	// Connection lifecycle (ResumeWindow)
	// A session becomes detached when the WebSocket drops. While detached, we keep
	// signal/component state in memory so the client can resume within ResumeWindow.
	detached   atomic.Bool
	DetachedAt time.Time // Time when the session detached (per-IP eviction ordering)

	// Loop lifecycle guards (prevent duplicate goroutines and allow resume to restart IO).
	readLoopRunning  atomic.Bool
	writeLoopRunning atomic.Bool
	eventLoopRunning atomic.Bool

	// Sequence numbers for reliable delivery
	sendSeq atomic.Uint64 // Next patch sequence to send
	recvSeq atomic.Uint64 // Last received event sequence
	ackSeq  atomic.Uint64 // Last acknowledged by client

	// Patch history for resync (Phase 2)
	// Stores recently sent patch frames for replay on client reconnection.
	patchHistory *PatchHistory

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
	events     chan *Event   // Incoming events
	renderCh   chan struct{} // Signal for re-render
	dispatchCh chan func()   // Functions to run on event loop (ctx.Dispatch)
	done       chan struct{} // Shutdown signal

	// Configuration
	config *SessionConfig

	// Logger
	logger *slog.Logger

	// Hooks
	onDetach func(*Session)

	// Metrics
	eventCount atomic.Uint64
	patchCount atomic.Uint64
	bytesSent  atomic.Uint64
	bytesRecv  atomic.Uint64

	// Lifecycle coordination
	//
	// Close() can be called concurrently with MountRoot()/flush()/handler execution.
	// We split close into:
	//   - beginClose: signal goroutines + close the websocket (best-effort)
	//   - finalizeClose: dispose owner/component tree + clear maps (runs once)
	//
	// finalizeClose is deferred until there is no in-flight session work, preventing
	// shutdown from racing renders (which previously caused nil Owner panics).
	inFlight     atomic.Int32
	finalizeOnce sync.Once

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

	// Route navigation (Phase 7: Routing)
	// navigator handles route-based navigation for this session
	navigator *RouteNavigator

	// Prefetch system (Phase 7: Routing, Section 8)
	// Per Section 8.2: Cache result per session, keyed by canonical path
	prefetchCache     *PrefetchCache
	prefetchLimiter   *PrefetchRateLimiter
	prefetchSemaphore *PrefetchSemaphore
	prefetchConfig    *PrefetchConfig

	// Asset resolver for fingerprinted asset paths (DX Improvements)
	assetResolver assets.Resolver
}

// IsDetached reports whether the session currently has no active WebSocket
// connection but is still kept in memory for ResumeWindow.
func (s *Session) IsDetached() bool {
	return s != nil && s.detached.Load()
}

func (s *Session) setOnDetach(fn func(*Session)) {
	s.onDetach = fn
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
		patchHistory:  NewPatchHistory(config.MaxPatchHistory),
	}

	// Initialize session-scoped store for SharedSignal support.
	// This enables store.Shared[T] signals to work without manual context setup.
	sessionStore := store.NewSessionStore()
	s.owner.SetValue(store.SessionKey, sessionStore)
	// Also set vango.SessionSignalStoreKey for vango.NewSharedSignal support.
	s.owner.SetValue(vango.SessionSignalStoreKey, sessionStore)

	// Initialize URL navigator for URLParam support.
	// URLParam.Set() will queue patches via queueURLPatch, which are then
	// sent along with DOM patches in renderDirty().
	navigator := urlparam.NewNavigator(s.queueURLPatch)
	s.owner.SetValue(urlparam.NavigatorKey, navigator)

	// Initialize prefetch system (Phase 7: Routing, Section 8)
	// Per Section 8.2: Cache result per session with TTL and LRU eviction
	// Per Section 8.5: Rate limit 5 requests/second per session
	s.prefetchConfig = DefaultPrefetchConfig()
	s.prefetchCache = NewPrefetchCache(s.prefetchConfig)
	s.prefetchLimiter = NewPrefetchRateLimiter(s.prefetchConfig.RateLimit)
	s.prefetchSemaphore = NewPrefetchSemaphore(s.prefetchConfig.SessionConcurrency)

	return s
}

// MountRoot mounts the root component for this session.
func (s *Session) MountRoot(component Component) {
	s.beginWork()
	defer s.endWork()

	// Create root component instance
	s.root = newComponentInstance(component, nil, s)
	s.root.InstanceID = "root"

	// Register root component for dirty tracking
	s.registerComponent(s.root)

	// Render the component tree, expanding any nested vdom.KindComponent nodes into
	// their rendered output before assigning HIDs.
	//
	// This is required for SSR/WS alignment: SSR rendering expands nested components
	// inline during traversal, so WS must assign HIDs in the same structural order
	// or client events will reference HIDs the session didn't register.
	tree := s.rerenderTree(s.root)

	// Assign hydration IDs to match SSR order.
	vdom.AssignHIDs(tree, s.hidGen)

	// Collect handlers from all mounted component instances.
	s.handlers = make(map[string]Handler)
	s.components = make(map[string]*ComponentInstance)
	s.collectHandlersFromInstances(s.root)

	// Store the tree
	s.currentTree = tree
	s.root.HID = tree.HID
	s.root.SetLastTree(tree)

	s.logger.Info("mounted root component",
		"handlers", len(s.handlers),
		"components", len(s.components),
		"hid_counter", s.hidGen.Current())

	// Run mount effects after initial commit
	ctx := s.createRenderContext()
	vango.WithCtx(ctx, func() {
		s.flush()
	})
}

// collectHandlersFromInstances collects event handlers from the mounted component
// instance tree.
//
// This intentionally walks component instances (not vdom.KindComponent markers)
// so handler ownership can remain correct even when nested components are expanded
// into the parent VNode tree during mount/rebuild.
func (s *Session) collectHandlersFromInstances(instance *ComponentInstance) {
	if instance == nil {
		return
	}

	// Collect from this instance's rendered tree first.
	s.collectHandlersNoMount(instance.LastTree(), instance)

	// Then recurse into children so deeper instances override ownership mapping.
	for _, child := range instance.Children {
		s.collectHandlersFromInstances(child)
	}
}

// collectHandlersNoMount walks a VNode tree and collects handlers without mounting
// or rendering nested component nodes.
func (s *Session) collectHandlersNoMount(node *vdom.VNode, instance *ComponentInstance) {
	if node == nil {
		return
	}

	if node.HID != "" {
		// Track that this component owns this HID (for cleanup and lookup).
		s.components[node.HID] = instance

		for key, value := range node.Props {
			if value == nil {
				continue
			}

			// Case 1: onhook handlers (from hooks.OnEvent, can be single or slice)
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

	for _, child := range node.Children {
		s.collectHandlersNoMount(child, instance)
	}
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
	// Reset per-tick storm budget counters at the start of each event tick
	if s.stormBudget != nil {
		s.stormBudget.ResetTick()
	}

	// Update sequence tracking
	s.recvSeq.Store(event.Seq)
	s.eventCount.Add(1)
	s.LastActive = time.Now()

	if DebugMode {
		fmt.Printf("[EVENT] Received: HID=%s Type=%v Seq=%d\n", event.HID, event.Type, event.Seq)
	}

	// Special handling for EventNavigate (0x70)
	// Per Section 4.2, navigation events trigger route matching, remount, and
	// NAV_* + DOM patches in ONE transaction (no client roundtrip).
	if event.Type == protocol.EventNavigate {
		s.handleEventNavigate(event)
		return
	}

	// Special handling for EventCustom (0xFF) - check for prefetch events
	// Per Section 8.1, prefetch events come as CUSTOM with name="prefetch"
	if event.Type == protocol.EventCustom {
		if customData, ok := event.Payload.(*protocol.CustomEventData); ok && customData != nil {
			if customData.Name == "prefetch" {
				s.handlePrefetch(customData.Data)
				return
			}
		}
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

	// Execute handler and effects with context and owner set.
	// Shared signals and other context-bound primitives rely on a current Owner.
	// We prefer the owner of the component that owns this HID.
	owner := s.owner
	if instance := s.components[event.HID]; instance != nil && instance.Owner != nil {
		owner = instance.Owner
	}
	vango.WithCtx(ctx, func() {
		vango.WithOwner(owner, func() {
			// Execute handler with panic recovery
			s.safeExecute(handler, event)

			// Commit render + effects after handler
			s.flush()
		})
	})
}

// handleEventNavigate handles client navigation events.
// Per Section 4.5 (Link Click Navigation) and 4.6 (Back/Forward Navigation),
// the client sends EventNavigate and the server responds with NAV_* + DOM patches.
func (s *Session) handleEventNavigate(event *Event) {
	navData, ok := event.Payload.(*protocol.NavigateEventData)
	if !ok || navData == nil {
		s.logger.Warn("invalid navigate event payload")
		s.sendErrorMessage(protocol.ErrInvalidEvent, "Invalid navigate event")
		return
	}

	if DebugMode {
		fmt.Printf("[EVENT] Navigate: path=%s replace=%v\n", navData.Path, navData.Replace)
	}

	// Navigation events are their own "tick": we must establish a runtime context
	// so mount effects run and reactive subscriptions are wired after the remount.
	ctx := s.createEventContext(event)
	vango.WithCtx(ctx, func() {
		// Use the session owner so context-bound primitives (SharedSignal/GetContext)
		// resolve correctly during any immediate effect work.
		vango.WithOwner(s.owner, func() {
			// Handle the navigation (matches route, remounts page, sends patches)
			if err := s.HandleNavigate(navData.Path, navData.Replace); err != nil {
				// Error already logged and sent to client
				return
			}

			// Run mount effects (and any follow-up renders) after sending NAV + DOM patches.
			s.flush()
		})
	})
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

// hasDirtyComponents returns true if any component is marked dirty.
func (s *Session) hasDirtyComponents() bool {
	for comp := range s.allComponents {
		if comp.IsDirty() {
			return true
		}
	}

	if s.root != nil && s.root.IsDirty() {
		return true
	}

	return false
}

// flush runs render/effect cycles until the system is stable or a safety cap is reached.
// Assumes a valid runtime context has been set via vango.WithCtx.
//
// Per Section 4.4 (Programmatic Navigation), flush FIRST checks for pending navigation.
// If ctx.Navigate() was called during handler execution, the navigation is processed
// here so NAV_* + DOM patches are sent in ONE transaction.
func (s *Session) flush() {
	const maxCycles = 10

	// Check for pending navigation FIRST
	// Per Section 4.4: ctx.Navigate() sets a pending navigation, which is processed
	// at flush time to ensure NAV_* + DOM patches are sent together.
	s.processPendingNavigation()

	for i := 0; i < maxCycles; i++ {
		// Commit current dirty state
		s.renderDirty()

		// Run pending effects after commit (pass storm budget for per-tick limiting)
		if s.owner != nil {
			s.owner.RunPendingEffects(s.stormBudget)
		}

		// Continue if effects or signal writes created new work
		if s.owner == nil || (!s.owner.HasPendingEffects() && !s.hasDirtyComponents()) {
			return
		}
	}

	s.logger.Warn("flush exceeded max cycles", "max", maxCycles)
}

// processPendingNavigation checks for and processes any pending navigation from ctx.Navigate().
// This ensures navigation happens at flush time so NAV_* + DOM patches are sent together.
func (s *Session) processPendingNavigation() {
	// Get current context via UseCtx (set by vango.WithCtx)
	vangoCtx := vango.UseCtx()
	if vangoCtx == nil {
		return
	}

	// Type assert to our *ctx to access pending navigation
	c, ok := vangoCtx.(*ctx)
	if !ok {
		return
	}

	// Check for pending navigation
	path, replace, has := c.PendingNavigation()
	if !has {
		return
	}

	// Clear pending to prevent double-processing
	c.ClearPendingNavigation()

	if DebugMode {
		fmt.Printf("[FLUSH] Processing pending navigation: path=%s replace=%v\n", path, replace)
	}

	// Process the navigation (this handles route matching, remount, and sending patches)
	if err := s.HandleNavigate(path, replace); err != nil {
		s.logger.Error("pending navigation failed", "path", path, "error", err)
	}
}

// renderComponent re-renders a single component and returns patches.
func (s *Session) renderComponent(comp *ComponentInstance) []vdom.Patch {
	// Get old tree
	oldTree := comp.LastTree()

	// Render new tree
	newTree := comp.Render()

	// Expand nested component nodes (KindComponent) into their rendered output.
	// SSR does this inline during HTML generation; WS must do the same so:
	//   - HIDs are assigned in the same structural order
	//   - diffing operates on the real element tree
	//   - event handlers are collected for nested components
	s.rerenderChildren(newTree, comp)

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
	s.clearSubtreeHandlers(comp)
	s.collectHandlersFromInstances(comp)

	return patches
}

func (s *Session) clearSubtreeHandlers(root *ComponentInstance) {
	if root == nil {
		return
	}

	subtree := make(map[*ComponentInstance]struct{})
	var collect func(*ComponentInstance)
	collect = func(n *ComponentInstance) {
		if n == nil {
			return
		}
		if _, ok := subtree[n]; ok {
			return
		}
		subtree[n] = struct{}{}
		for _, ch := range n.Children {
			collect(ch)
		}
	}
	collect(root)

	for hid, inst := range s.components {
		if _, ok := subtree[inst]; !ok {
			continue
		}
		delete(s.components, hid)
		prefix := hid + "_"
		for key := range s.handlers {
			if strings.HasPrefix(key, prefix) {
				delete(s.handlers, key)
			}
		}
	}
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
	s.collectHandlersFromInstances(s.root)

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

	oldChildren := parent.Children
	usedChildren := make([]*ComponentInstance, 0, len(oldChildren))
	slot := 0

	var walk func(*vdom.VNode)
	walk = func(n *vdom.VNode) {
		if n == nil {
			return
		}

		for i, child := range n.Children {
			if child.Kind == vdom.KindComponent && child.Comp != nil {
				var childInstance *ComponentInstance
				if slot < len(oldChildren) {
					childInstance = oldChildren[slot]
					if childInstance == nil {
						childInstance = newComponentInstance(child.Comp, parent, s)
						s.registerComponent(childInstance)
					}
					// Update component implementation each render so "props via closure"
					// patterns (e.g., Counter(initial)) work while preserving owner state.
					childInstance.Component = child.Comp
				} else {
					childInstance = newComponentInstance(child.Comp, parent, s)
					s.registerComponent(childInstance)
				}

				// Ensure correct parent pointer (defensive; should already be set).
				childInstance.Parent = parent

				usedChildren = append(usedChildren, childInstance)
				slot++

				rendered := s.rerenderTree(childInstance)
				n.Children[i] = rendered
				continue
			}

			// Recurse into regular nodes; any nested components discovered are
			// still mounted as children of the same parent component instance.
			walk(child)
		}
	}

	walk(node)

	// Dispose any instances that are no longer present at this component slot sequence.
	for i := len(usedChildren); i < len(oldChildren); i++ {
		s.disposeInstanceTree(oldChildren[i])
	}

	parent.Children = usedChildren
}

func (s *Session) disposeInstanceTree(instance *ComponentInstance) {
	if instance == nil {
		return
	}

	// Unregister descendants first.
	for _, ch := range instance.Children {
		s.disposeInstanceTree(ch)
	}

	s.clearSubtreeHandlers(instance)
	s.unregisterComponent(instance)
	instance.Dispose()
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

	if s.closed.Load() {
		s.mu.Unlock()
		return
	}

	// Guard against nil connection (can happen in tests or edge cases)
	if s.conn == nil {
		s.logger.Warn("sendPatches: no connection available")
		s.mu.Unlock()
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
		s.mu.Unlock()
		s.Close()
		return
	}

	// Store frame in patch history AFTER successful write
	// This enables resync if client misses this frame
	if s.patchHistory != nil {
		s.patchHistory.Add(seq, frameData)
	}

	// Update metrics
	s.bytesSent.Add(uint64(len(frameData)))
	s.patchCount.Add(uint64(len(protocolPatches)))

	s.logger.Debug("sent patches",
		"seq", seq,
		"count", len(protocolPatches),
		"bytes", len(frameData))

	s.mu.Unlock()
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

// SendHookRevert requests the client to revert a hook's optimistic UI change.
// The client will invoke the revert callback registered for the given HID.
func (s *Session) SendHookRevert(hid string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed.Load() {
		return
	}
	if s.conn == nil {
		return
	}

	ct, hr := protocol.NewHookRevert(hid)
	payload := protocol.EncodeControl(ct, hr)
	frame := protocol.NewFrame(protocol.FrameControl, payload)

	s.conn.SetWriteDeadline(time.Now().Add(s.config.WriteTimeout))
	if err := s.conn.WriteMessage(websocket.BinaryMessage, frame.Encode()); err != nil {
		s.logger.Error("hook revert send error", "error", err)
	}
}

// Close gracefully closes the session.
func (s *Session) Close() {
	if s.closed.Swap(true) {
		// Already closed
		return
	}

	s.beginClose()

	// If nothing is currently running, we can finalize immediately.
	if s.inFlight.Load() == 0 {
		s.finalizeClose()
	}
}

// beginWork marks the beginning of a session "work unit" that must not race
// finalizeClose (tree/owner disposal). Work units include MountRoot, event ticks,
// dispatched callbacks, and render ticks.
func (s *Session) beginWork() {
	s.inFlight.Add(1)
}

func (s *Session) endWork() {
	if s.inFlight.Add(-1) == 0 && s.closed.Load() {
		s.finalizeClose()
	}
}

// beginClose performs the non-destructive part of closing:
//   - signals goroutines via done
//   - closes the websocket connection (best effort)
//
// It does NOT dispose the reactive owner or component tree; that happens in
// finalizeClose once no work is in-flight.
func (s *Session) beginClose() {
	// Signal shutdown to goroutines
	select {
	case <-s.done:
		// Already closed
	default:
		close(s.done)
	}

	// Send close message and close WebSocket
	s.mu.Lock()
	conn := s.conn
	if conn != nil {
		_ = conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(time.Second),
		)
		_ = conn.Close()
		s.conn = nil
	}
	s.mu.Unlock()
}

// finalizeClose performs the destructive portion of closing and is guaranteed
// to run at most once. It must not run while session work is in-flight.
func (s *Session) finalizeClose() {
	s.finalizeOnce.Do(func() {
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

		s.logger.Info("session closed",
			"events", s.eventCount.Load(),
			"patches", s.patchCount.Load(),
			"bytes_sent", s.bytesSent.Load(),
			"bytes_recv", s.bytesRecv.Load())
	})
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
	size += EstimateMapMemory(len(s.handlers), 32, 32)

	// Components map
	size += EstimateMapMemory(len(s.components), 16, 8)
	size += EstimateMapMemory(len(s.allComponents), 8, 8)
	for _, comp := range s.components {
		size += comp.MemoryUsage()
	}

	// Current tree
	if s.currentTree != nil {
		size += estimateVNodeSize(s.currentTree)
	}

	// Events channel buffer (estimate)
	size += EstimateSliceMemory(cap(s.events), 64)
	size += EstimateSliceMemory(cap(s.dispatchCh), 64)
	size += EstimateSliceMemory(cap(s.renderCh), 8)

	if s.patchHistory != nil {
		size += s.patchHistory.MemoryUsage()
	}

	if s.prefetchCache != nil {
		size += s.prefetchCache.MemoryUsage()
	}

	if s.owner != nil {
		size += s.owner.MemoryUsage()
	}

	s.dataMu.RLock()
	if len(s.data) > 0 {
		size += EstimateMapMemory(len(s.data), 32, 32)
		for k, v := range s.data {
			size += EstimateStringMemory(k)
			size += EstimateAnyMemory(v)
		}
	}
	s.dataMu.RUnlock()

	s.urlPatchMu.Lock()
	if len(s.pendingURLPatches) > 0 {
		size += EstimateSliceMemory(len(s.pendingURLPatches), 64)
		for i := range s.pendingURLPatches {
			size += EstimateAnyMemory(s.pendingURLPatches[i])
		}
	}
	s.urlPatchMu.Unlock()

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
		session:       s,
		event:         event,
		logger:        s.logger,
		stdCtx:        context.Background(),
		assetResolver: s.assetResolver,
	}
}

// createRenderContext creates a Ctx for component rendering.
// This context is set via vango.WithCtx so UseCtx() works during render.
func (s *Session) createRenderContext() Ctx {
	return &ctx{
		session:       s,
		logger:        s.logger,
		stdCtx:        context.Background(),
		assetResolver: s.assetResolver,
	}
}

// SetAssetResolver sets the asset resolver for this session.
// This is called by the server when a session is created or resumed.
func (s *Session) SetAssetResolver(r assets.Resolver) {
	s.assetResolver = r
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
// Route Navigation (Phase 7: Routing)
// =============================================================================

// SetRouter sets the router for this session and creates a navigator.
// This enables route-based navigation via ctx.Navigate() and EventNavigate.
// The router must implement the Router interface (defined in navigation.go).
func (s *Session) SetRouter(r Router) {
	s.navigator = NewRouteNavigator(s, r)
}

// Navigator returns the route navigator for this session.
// Returns nil if no router has been set.
func (s *Session) Navigator() *RouteNavigator {
	return s.navigator
}

// HandleNavigate processes a navigation request and returns patches.
// This is called from handleEvent when an EventNavigate is received,
// or from flush() when ctx.Navigate() was called during event handling.
//
// Per Section 4.4 (Programmatic Navigation), navigation is ONE transaction:
// NAV_* patch + DOM patches are sent together in a single frame.
//
// Parameters:
//   - path: The target path (may include query string)
//   - replace: If true, use NAV_REPLACE instead of NAV_PUSH
//
// Returns:
//   - error if navigation failed (no route matched and no NotFound handler)
func (s *Session) HandleNavigate(path string, replace bool) error {
	if s.navigator == nil {
		// No router configured - fall back to sending just NAV_* patch
		// The client will need to request a full page load
		s.logger.Warn("navigation without router, sending NAV patch only", "path", path)
		var patch protocol.Patch
		if replace {
			patch = protocol.NewNavReplacePatch(path)
		} else {
			patch = protocol.NewNavPushPatch(path)
		}
		s.SendPatches([]protocol.Patch{patch})
		return nil
	}

	// Use the route navigator
	result := s.navigator.Navigate(path, replace)

	if result.Error != nil {
		s.logger.Error("navigation error", "path", path, "error", result.Error)
		s.sendErrorMessage(protocol.ErrRouteError, result.Error.Error())
		return result.Error
	}

	if !result.Matched && s.navigator.router.NotFound() == nil {
		s.logger.Warn("no route matched", "path", path)
		s.sendErrorMessage(protocol.ErrNotFound, "Route not found: "+path)
		return errors.New("route not found")
	}

	// Send NAV_* patch + DOM patches in one frame
	// Convert vdom.Patch to protocol.Patch
	allPatches := make([]protocol.Patch, 0, len(result.Patches)+1)

	// NAV patch first
	allPatches = append(allPatches, result.NavPatch)

	// Then DOM patches
	for _, p := range result.Patches {
		allPatches = append(allPatches, protocol.Patch{
			Op:       protocol.PatchOp(p.Op),
			HID:      p.HID,
			Key:      p.Key,
			Value:    p.Value,
			ParentID: p.ParentID,
			Index:    p.Index,
		})
		if p.Node != nil {
			allPatches[len(allPatches)-1].Node = protocol.VNodeToWire(p.Node)
		}
	}

	s.SendPatches(allPatches)

	s.logger.Debug("navigation completed",
		"path", result.Path,
		"matched", result.Matched,
		"dom_patches", len(result.Patches))

	return nil
}

// =============================================================================
// Prefetch System (Phase 7: Routing, Section 8)
// =============================================================================

// prefetchPayload is the JSON structure for prefetch events.
// Per Section 8.1: Data: JSON-encoded `{ "path": "/target/path" }`
type prefetchPayload struct {
	Path string `json:"path"`
}

// handlePrefetch handles a prefetch custom event.
// Per Section 8.1-8.5:
//   - Parse JSON path from event data
//   - Check rate limit (5/sec per session)
//   - Check concurrency limits (2 per session, 50 global)
//   - Execute route in prefetch mode with timeout
//   - Cache result if successful
func (s *Session) handlePrefetch(data []byte) {
	// Parse the prefetch payload
	var payload prefetchPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		s.logger.Warn("invalid prefetch payload", "error", err)
		return
	}

	if payload.Path == "" {
		s.logger.Warn("prefetch path is empty")
		return
	}

	if DebugMode {
		fmt.Printf("[PREFETCH] Received request for path: %s\n", payload.Path)
	}

	// Check rate limit (Section 8.5: 5 requests/sec per session)
	if !s.prefetchLimiter.Allow() {
		if DebugMode {
			fmt.Printf("[PREFETCH] Rate limited, dropping request for: %s\n", payload.Path)
		}
		return // Silently drop
	}

	// Check session concurrency limit (Section 8.3.3: 2 per session)
	if !s.prefetchSemaphore.Acquire() {
		if DebugMode {
			fmt.Printf("[PREFETCH] Session concurrency limit reached, dropping: %s\n", payload.Path)
		}
		return // Silently drop
	}
	defer s.prefetchSemaphore.Release()

	// Check global concurrency limit (Section 8.3.3: 50 global)
	if !GlobalPrefetchSemaphore().Acquire() {
		if DebugMode {
			fmt.Printf("[PREFETCH] Global concurrency limit reached, dropping: %s\n", payload.Path)
		}
		return // Silently drop
	}
	defer GlobalPrefetchSemaphore().Release()

	// Execute prefetch with timeout
	s.executePrefetch(payload.Path)
}

// executePrefetch renders a route in prefetch mode and caches the result.
// Per Section 8.2:
//   - Route match the prefetch path (using canonical path)
//   - Execute page handler in "prefetch mode" (ctx.Mode() == ModePrefetch)
//   - Cache result per session, keyed by canonical path
//
// Per Section 8.3.3:
//   - Timeout: 100ms (abort if handler takes too long)
func (s *Session) executePrefetch(path string) {
	// Canonicalize the path
	canonFullPath, err := routepath.CanonicalizeAndValidateNavPath(path)
	if err != nil {
		s.logger.Warn("prefetch path canonicalization failed", "path", path, "error", err)
		return
	}
	canonPath, query := routepath.SplitPathAndQuery(canonFullPath)

	// Check if already cached
	if s.prefetchCache.Get(canonPath) != nil {
		if DebugMode {
			fmt.Printf("[PREFETCH] Already cached: %s\n", canonPath)
		}
		return
	}

	// Check if we have a router
	if s.navigator == nil || s.navigator.router == nil {
		s.logger.Warn("prefetch without router", "path", canonPath)
		return
	}

	// Match the route
	match, ok := s.navigator.router.Match("GET", canonPath)
	if !ok {
		s.logger.Debug("prefetch: no route matched", "path", canonPath)
		return
	}

	// Get page handler
	pageHandler := match.GetPageHandler()
	if pageHandler == nil {
		s.logger.Debug("prefetch: no page handler", "path", canonPath)
		return
	}

	// Create a prefetch context with timeout
	done := make(chan *vdom.VNode, 1)
	timeout := time.NewTimer(s.prefetchConfig.Timeout)
	defer timeout.Stop()

	// Execute in goroutine with timeout
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Warn("prefetch panic recovered", "path", canonPath, "panic", r)
				done <- nil
			}
		}()

		// Create a render context in prefetch mode
		renderCtx := s.createPrefetchContext()
		if ctxImpl, ok := renderCtx.(*ctx); ok {
			ctxImpl.setParams(match.GetParams())
			if ctxImpl.request == nil {
				ctxImpl.request = &http.Request{
					Method: http.MethodGet,
					URL: &url.URL{
						Path:     canonPath,
						RawQuery: query,
					},
				}
			}
		}

		var tree *vdom.VNode

		// Render within vango.WithCtx for proper reactive context
		vango.WithCtx(renderCtx, func() {
			ranFinal, mwErr := RunRouteMiddleware(renderCtx, match.GetMiddleware(), func() error {
				// Call the page handler to get the component
				comp := pageHandler(renderCtx, match.GetParams())
				if comp == nil {
					return nil
				}

				// Render the component to VNode
				tree = comp.Render()

				// Apply layouts root to leaf (reverse order so outermost is first)
				layouts := match.GetLayoutHandlers()
				for i := len(layouts) - 1; i >= 0; i-- {
					layout := layouts[i]
					tree = layout(renderCtx, tree)
				}

				return nil
			})
			if mwErr != nil || !ranFinal {
				tree = nil
			}
		})

		done <- tree
	}()

	// Wait for result or timeout
	select {
	case tree := <-done:
		if tree != nil {
			// Cache the result
			s.prefetchCache.Set(canonPath, tree)
			if DebugMode {
				fmt.Printf("[PREFETCH] Cached result for: %s\n", canonPath)
			}
		}

	case <-timeout.C:
		// Timeout - discard result
		s.logger.Debug("prefetch timeout", "path", canonPath)
		if DebugMode {
			fmt.Printf("[PREFETCH] Timeout for: %s\n", canonPath)
		}
	}
}

// createPrefetchContext creates a Ctx for prefetch rendering.
// The context is in ModePrefetch to enforce read-only behavior.
func (s *Session) createPrefetchContext() Ctx {
	c := &ctx{
		session: s,
		logger:  s.logger,
		stdCtx:  context.Background(),
		mode:    ModePrefetch,
	}
	return c
}

// PrefetchCache returns the prefetch cache for this session.
// Returns nil if prefetch is not initialized.
func (s *Session) PrefetchCache() *PrefetchCache {
	return s.prefetchCache
}

// =============================================================================
// Test Helpers (Phase 10F)
// =============================================================================

// NewMockSession creates a session without a WebSocket connection for testing.
// The session has all fields initialized except conn.
func NewMockSession() *Session {
	prefetchConfig := DefaultPrefetchConfig()
	return &Session{
		ID:                "test-session-id",
		UserID:            "",
		CreatedAt:         time.Now(),
		LastActive:        time.Now(),
		allComponents:     make(map[*ComponentInstance]struct{}),
		handlers:          make(map[string]Handler),
		components:        make(map[string]*ComponentInstance),
		owner:             vango.NewOwner(nil),
		hidGen:            vdom.NewHIDGenerator(),
		events:            make(chan *Event, 256),
		renderCh:          make(chan struct{}, 1),
		dispatchCh:        make(chan func(), 256),
		done:              make(chan struct{}),
		config:            DefaultSessionConfig(),
		logger:            slog.Default().With("session_id", "test-session-id"),
		data:              make(map[string]any),
		prefetchConfig:    prefetchConfig,
		prefetchCache:     NewPrefetchCache(prefetchConfig),
		prefetchLimiter:   NewPrefetchRateLimiter(prefetchConfig.RateLimit),
		prefetchSemaphore: NewPrefetchSemaphore(prefetchConfig.SessionConcurrency),
	}
}
