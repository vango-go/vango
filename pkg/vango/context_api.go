package vango

import (
	"github.com/vango-go/vango/pkg/vdom"
)

// Context provides dependency injection through the component tree.
// Create a context with CreateContext, provide values with Provider,
// and consume values with Use.
//
// Example:
//
//	var ThemeContext = vango.CreateContext("light")
//
//	func App() vango.Component {
//	    return vango.Func(func() *vango.VNode {
//	        return ThemeContext.Provider("dark",
//	            Header(),
//	            Main(),
//	        )
//	    })
//	}
//
//	func Button() *vango.VNode {
//	    theme := ThemeContext.Use()
//	    return vdom.Button(vdom.Class("btn-" + theme))
//	}
type Context[T any] struct {
	// key uniquely identifies this context in the owner value map
	key any

	// defaultValue is returned when no provider is found
	defaultValue T
}

// contextKey wraps Context to create a unique key type
type contextKey[T any] struct {
	ctx *Context[T]
}

// CreateContext creates a new context with the given default value.
// The default value is returned by Use() when no Provider is found
// in the component tree.
//
// Example:
//
//	var ThemeContext = vango.CreateContext("light")
//	var UserContext = vango.CreateContext[*User](nil)
func CreateContext[T any](defaultValue T) *Context[T] {
	ctx := &Context[T]{
		defaultValue: defaultValue,
	}
	// Use the context pointer itself as the key to ensure uniqueness
	ctx.key = contextKey[T]{ctx: ctx}
	return ctx
}

// Provider wraps children with this context's value.
// Descendant components can access the value via Use().
//
// Example:
//
//	func App() vango.Component {
//	    return vango.Func(func() *vango.VNode {
//	        return ThemeContext.Provider("dark",
//	            Header(),
//	            Main(),
//	            Footer(),
//	        )
//	    })
//	}
func (c *Context[T]) Provider(value T, children ...any) *vdom.VNode {
	// Store the value in the current owner's context
	owner := getCurrentOwner()
	if owner != nil {
		owner.SetValue(c.key, value)
	}

	// Return a fragment containing the children
	return vdom.Fragment(children...)
}

// Use retrieves the context value from the nearest Provider ancestor.
// If no Provider is found, returns the default value.
//
// This is a hook-like API and MUST be called unconditionally during render.
// See §3.1.3 Hook-Order Semantics.
//
// Example:
//
//	func Button() *vango.VNode {
//	    theme := ThemeContext.Use()
//	    return vdom.Button(vdom.Class("btn-" + theme))
//	}
func (c *Context[T]) Use() T {
	// Track hook call for dev-mode order validation
	TrackHook(HookContext)

	// Look up the value in the owner hierarchy
	owner := getCurrentOwner()
	if owner != nil {
		if value := owner.GetValue(c.key); value != nil {
			if typed, ok := value.(T); ok {
				return typed
			}
		}
	}

	return c.defaultValue
}

// Default returns the default value for this context.
func (c *Context[T]) Default() T {
	return c.defaultValue
}
