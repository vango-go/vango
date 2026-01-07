package vango

import (
	"github.com/vango-go/vango/pkg/features/resource"
	"github.com/vango-go/vango/pkg/vdom"
)

// =============================================================================
// Resource State (spec-aligned: vango.Pending, vango.Loading, etc.)
// =============================================================================

// ResourceState represents the current state of a resource.
type ResourceState = resource.State

// State constants for Resource.
// These are spec-aligned exports: use vango.Pending, vango.Loading, etc.
const (
	Pending  ResourceState = resource.Pending  // Initial state, before first fetch
	Loading  ResourceState = resource.Loading  // Fetch in progress
	Ready    ResourceState = resource.Ready    // Data successfully loaded
	Error    ResourceState = resource.Error    // Fetch failed (distinct from vango.Error)
)

// =============================================================================
// Resource Types
// =============================================================================

// Resource manages asynchronous data fetching and state.
// It provides reactive state tracking (Pending → Loading → Ready | Error)
// and methods for matching state in templates.
//
// This is a hook-like API and MUST be called unconditionally during render.
//
// Example:
//
//	users := vango.NewResource(func() ([]User, error) {
//	    return api.FetchUsers(ctx.StdContext())
//	})
//
//	return users.Match(
//	    vango.OnLoading[[]User](func() *vango.VNode {
//	        return Div(Text("Loading..."))
//	    }),
//	    vango.OnError[[]User](func(err error) *vango.VNode {
//	        return Div(Textf("Error: %v", err))
//	    }),
//	    vango.OnReady(func(users []User) *vango.VNode {
//	        return Ul(mapUsers(users)...)
//	    }),
//	)
type Resource[T any] = resource.Resource[T]

// Handler handles a specific resource state in Match().
type Handler[T any] = resource.Handler[T]

// =============================================================================
// Resource Constructors
// =============================================================================

// NewResource creates a new Resource with the given fetcher function.
// The initial fetch is scheduled automatically via Effect.
//
// This is a hook-like API and MUST be called unconditionally during render.
// See §3.1.3 Hook-Order Semantics.
//
// Example:
//
//	users := vango.NewResource(func() ([]User, error) {
//	    return db.Users.All(ctx.StdContext())
//	})
//
//	// Configure with options (chained)
//	users := vango.NewResource(fetchUsers).
//	    StaleTime(5 * time.Minute).
//	    RetryOnError(3, time.Second)
func NewResource[T any](fetcher func() (T, error)) *Resource[T] {
	return resource.New(fetcher)
}

// NewResourceKeyed creates a Resource that automatically refetches when the key changes.
// The key can be a *Signal[K] or a func() K.
//
// This is a hook-like API and MUST be called unconditionally during render.
// See §3.1.3 Hook-Order Semantics.
//
// Example with Signal:
//
//	userId := vango.NewSignal(123)
//	user := vango.NewResourceKeyed(userId, func(id int) (*User, error) {
//	    return api.FetchUser(ctx.StdContext(), id)
//	})
//
// Example with func:
//
//	user := vango.NewResourceKeyed(
//	    func() int { return userId.Get() },
//	    func(id int) (*User, error) {
//	        return api.FetchUser(ctx.StdContext(), id)
//	    },
//	)
func NewResourceKeyed[K comparable, T any](key any, fetcher func(K) (T, error)) *Resource[T] {
	// Convert key to func() K
	var keyFn func() K
	switch k := key.(type) {
	case *Signal[K]:
		keyFn = k.Get
	case func() K:
		keyFn = k
	default:
		panic("vango: NewResourceKeyed key must be *Signal[K] or func() K")
	}
	return resource.NewWithKey(keyFn, fetcher)
}

// =============================================================================
// Match Handlers
// =============================================================================

// OnPending handles the Pending state in Resource.Match().
// Pending is the initial state before the first fetch begins.
//
// Example:
//
//	users.Match(
//	    vango.OnPending[[]User](func() *vango.VNode {
//	        return Div(Text("Initializing..."))
//	    }),
//	    // ...
//	)
func OnPending[T any](fn func() *vdom.VNode) Handler[T] {
	return resource.OnPending[T](fn)
}

// OnLoading handles the Loading state in Resource.Match().
// Loading is when a fetch is in progress.
//
// Example:
//
//	users.Match(
//	    vango.OnLoading[[]User](func() *vango.VNode {
//	        return Div(Class("spinner"), Text("Loading..."))
//	    }),
//	    // ...
//	)
func OnLoading[T any](fn func() *vdom.VNode) Handler[T] {
	return resource.OnLoading[T](fn)
}

// OnReady handles the Ready state in Resource.Match().
// Ready is when data has been successfully loaded.
//
// Example:
//
//	users.Match(
//	    vango.OnReady(func(users []User) *vango.VNode {
//	        return Ul(mapUsers(users)...)
//	    }),
//	)
func OnReady[T any](fn func(T) *vdom.VNode) Handler[T] {
	return resource.OnReady[T](fn)
}

// OnError handles the Error state in Resource.Match().
// Error is when the fetch failed.
//
// Example:
//
//	users.Match(
//	    vango.OnError[[]User](func(err error) *vango.VNode {
//	        return Div(Class("error"), Textf("Failed: %v", err))
//	    }),
//	    // ...
//	)
func OnError[T any](fn func(error) *vdom.VNode) Handler[T] {
	return resource.OnError[T](fn)
}

// OnLoadingOrPending handles both Loading and Pending states in Resource.Match().
// This is a convenience handler for showing a spinner during any waiting state.
//
// Example:
//
//	users.Match(
//	    vango.OnLoadingOrPending[[]User](func() *vango.VNode {
//	        return Spinner() // Show spinner for both pending and loading
//	    }),
//	    vango.OnError[[]User](handleError),
//	    vango.OnReady(renderUsers),
//	)
func OnLoadingOrPending[T any](fn func() *vdom.VNode) Handler[T] {
	return resource.OnLoadingOrPending[T](fn)
}
