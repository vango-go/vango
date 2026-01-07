// Package features provides higher-level abstractions for building Vango applications.
//
// This package contains the productive APIs that developers interact with daily,
// built on top of the foundation provided by the vango, vdom, and server packages.
//
// # Subsystems
//
// The features package is organized into several subsystems:
//
//   - form: Type-safe form binding with validation
//   - resource: Async data loading with loading/error/success states
//   - context: Dependency injection through the component tree
//   - hooks: Client-side 60fps interactions with server state
//   - shared: Session-scoped and global shared state
//   - optimistic: Instant visual feedback for interactions
//   - islands: Third-party JavaScript library integration
//
// Note: For URL query state, use the urlparam package (vango.URLParam).
//
// # Usage
//
// Each subsystem is in its own sub-package and can be imported independently:
//
//	import "vango_v2/pkg/features/form"
//	import "vango_v2/pkg/features/resource"
//	import "vango_v2/pkg/features/context"
//
// See the individual package documentation for detailed usage examples.
package features
