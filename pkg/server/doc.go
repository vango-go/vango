// Package server provides the server-side runtime for Vango's server-driven architecture.
//
// The server package manages WebSocket connections, component state, event handling,
// and patch generation. It is the integration layer that brings together the reactive
// system (pkg/vango), virtual DOM (pkg/vdom), and binary protocol (pkg/protocol).
//
// # Architecture
//
// The server runtime consists of several key components:
//
//   - Session: Per-connection state container managing component tree, handlers, and reactive ownership
//   - SessionManager: Manages all active sessions with cleanup and lifecycle hooks
//   - ComponentInstance: A mounted component with its reactive state and render capability
//   - Handler: Event handler functions that respond to client events
//   - Server: HTTP/WebSocket server with handshake and graceful shutdown
//
// # Session Lifecycle
//
// Each WebSocket connection creates a Session that manages:
//   - Component tree and hydration IDs
//   - Event handler registry (HID -> handler mapping)
//   - Reactive ownership for signals and effects
//   - Sequence numbers for reliable delivery
//
// The session runs three goroutines:
//   - ReadLoop: Receives WebSocket frames, decodes events, queues for processing
//   - EventLoop: Processes events, runs handlers, generates patches
//   - WriteLoop: Sends heartbeat pings
//
// # Event Processing
//
// When a client sends an event:
//  1. ReadLoop decodes the binary event frame
//  2. Event is queued for the EventLoop
//  3. Handler is found by HID and executed
//  4. Pending effects are run
//  5. Dirty components are re-rendered
//  6. Diff generates patches
//  7. Patches are encoded and sent to client
//
// # Example Usage
//
//	server := server.New(&server.ServerConfig{
//	    Address: ":8080",
//	})
//
//	server.HandleFunc("/", func(ctx server.Ctx) *vdom.VNode {
//	    count := vango.NewIntSignal(0)
//	    return vdom.Div(
//	        vdom.H1(vdom.Textf("Count: %d", count.Get())),
//	        vdom.Button(
//	            vdom.OnClick(func() { count.Inc() }),
//	            vdom.Text("+"),
//	        ),
//	    )
//	})
//
//	server.Run()
//
// # Thread Safety
//
// The server package is designed for concurrent access:
//   - Session.mu protects WebSocket writes
//   - Events channel serializes event processing
//   - Signal access uses Phase 1 synchronization
//   - SessionManager uses RWMutex for session map
//
// # Performance Targets
//
//   - Memory per session: < 200KB average
//   - Concurrent sessions per GB: 5,000+
//   - Event processing latency: < 10ms
//   - WebSocket reconnect: < 500ms
package server
