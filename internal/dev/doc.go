// Package dev provides the development server and hot reload functionality.
//
// This package implements:
//   - File watching for Go, CSS, and asset changes
//   - Incremental Go compilation
//   - WebSocket-based browser refresh
//   - Tailwind CSS compilation
//   - Error overlay in browser
//
// # Architecture
//
// The development server consists of several components:
//
//   - Watcher: Monitors file system for changes
//   - Compiler: Builds Go code incrementally
//   - Server: Serves the application and static files
//   - ReloadServer: Notifies browsers of changes via WebSocket
//   - TailwindRunner: Compiles Tailwind CSS on change
//
// # Usage
//
//	srv := dev.NewServer(config, dev.Options{
//	    Port:        3000,
//	    OpenBrowser: true,
//	})
//
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//
//	if err := srv.Start(ctx); err != nil {
//	    log.Fatal(err)
//	}
//
// # Configuration
//
// Hot reload can be disabled via vango.json (dev.hotReload=false).
// Watch paths are derived from project config (routes, components, static, etc.)
// plus any entries in dev.watch.
//
// # Hot Reload Protocol
//
// The browser connects to /_vango/reload via WebSocket.
// Messages are JSON-encoded:
//
//	{"type": "reload"}              // Triggers full page reload
//	{"type": "css"}                 // Triggers CSS-only reload
//	{"type": "error", "error": "..."} // Shows error overlay
//	{"type": "clear"}               // Clears error overlay
package dev
