package context

import (
	"github.com/vango-go/vango/pkg/vango"
	"github.com/vango-go/vango/pkg/vdom"
)

// Context provides a way to pass data through the component tree.
//
// This is an alternative implementation to vango.CreateContext that lives in
// the features package. It uses vango.CreateContext under the hood.
//
// For new code, prefer using vango.CreateContext directly.
type Context[T any] struct {
	inner *vango.Context[T]
}

// Create creates a new Context with a default value.
func Create[T any](defaultValue T) *Context[T] {
	return &Context[T]{
		inner: vango.CreateContext(defaultValue),
	}
}

// Provider creates a component that provides a value to its children.
// The provided value is scoped to descendants only (siblings won't see it).
// When the provider re-renders with a new value, all descendants that called
// Use() will be re-rendered (reactive).
func (c *Context[T]) Provider(value T, children ...any) *vdom.VNode {
	// Delegate to the vango.Context implementation which handles scoping
	// and reactivity correctly.
	return c.inner.Provider(value, children...)
}

// Use retrieves the current context value.
// If no provider is found, returns the default value.
//
// This is a reactive hook: it subscribes the calling component to the context
// value. When the Provider re-renders with a new value, this component will
// also re-render.
func (c *Context[T]) Use() T {
	return c.inner.Use()
}
