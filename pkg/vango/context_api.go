package vango

import (
	"github.com/vango-go/vango/pkg/vdom"
)

// Context provides dependency injection through the component tree.
// Create a context with CreateContext, provide values with Provider,
// and consume values with Use.
//
// Provider scoping (normative): Provider() creates a new owner scope (component
// boundary) so the provided value is only visible to descendants of the Provider,
// not to sibling components.
//
// Reactivity (normative): Use() is a reactive hook that subscribes the calling
// component to the context value. When a Provider re-renders with a new value,
// all components that called Use() on that context will be re-rendered.
// See VANGO_ARCHITECTURE_AND_GUIDE.md §3.1.3 and §3.9.10.
//
// Example:
//
//	var ThemeContext = vango.CreateContext("light")
//
//	func App() vango.Component {
//	    return vango.Func(func() *vango.VNode {
//	        theme := vango.NewSignal("dark")
//	        return ThemeContext.Provider(theme.Get(),
//	            Header(),
//	            Main(),
//	        )
//	    })
//	}
//
//	func Button() *vango.VNode {
//	    theme := ThemeContext.Use()  // Subscribes to changes
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

// contextValue wraps a signal to provide reactive context values.
// This is stored in the Provider's owner scope.
type contextValue[T any] struct {
	signal *Signal[T]
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
// Provider creates a new component boundary (owner scope) so the context
// value is only visible to descendants, not siblings. This is the proper
// scoping behavior per the spec.
//
// The provider is reactive: when the value changes and the parent component
// re-renders, all descendants that called Use() will also re-render.
//
// Example:
//
//	func App() vango.Component {
//	    return vango.Func(func() *vango.VNode {
//	        theme := vango.NewSignal("dark")
//	        return ThemeContext.Provider(theme.Get(),
//	            Header(),
//	            Main(),
//	            Footer(),
//	        )
//	    })
//	}
func (c *Context[T]) Provider(value T, children ...any) *vdom.VNode {
	// Create a provider component that establishes a new owner scope.
	// This ensures the context value is scoped to descendants only.
	providerComp := &contextProviderComponent[T]{
		ctx:      c,
		value:    value,
		children: children,
	}

	return &vdom.VNode{
		Kind: vdom.KindComponent,
		Comp: providerComp,
	}
}

// contextProviderComponent is the internal component that creates the
// owner scope for context providers.
type contextProviderComponent[T any] struct {
	ctx      *Context[T]
	value    T
	children []any
}

// Render implements vdom.Component.
// This creates a new owner scope and stores a reactive contextValue.
func (p *contextProviderComponent[T]) Render() *vdom.VNode {
	owner := getCurrentOwner()
	if owner == nil {
		// No owner context, just return children as fragment
		return vdom.Fragment(p.children...)
	}

	// Look for existing contextValue in this owner's slot.
	// We use the owner's values map directly (not hook slots) because
	// the Provider is the component that owns the contextValue.
	existingValue := owner.GetValueLocal(p.ctx.key)

	if existingValue != nil {
		// Re-render: update the existing signal value quietly.
		// We use setQuietly to avoid violating the "no signal writes during
		// render" rule (§3.1.2). The children will read the new value when
		// they render as part of this same render cycle.
		//
		// Note: subscribers are NOT notified here. If external code needs to
		// react to context changes, they should depend on the signal that
		// drives the Provider's value, not the context signal itself.
		if cv, ok := existingValue.(*contextValue[T]); ok {
			cv.signal.setQuietly(p.value)
		}
	} else {
		// First render: create new contextValue with signal
		// The signal is created outside hook-slot semantics because
		// this is the Provider's own state, not a hook.
		sig := &Signal[T]{
			base: signalBase{
				id: nextID(),
			},
			value:     p.value,
			transient: true, // Context values are ephemeral, not persisted
		}
		cv := &contextValue[T]{signal: sig}
		owner.SetValue(p.ctx.key, cv)
	}

	return vdom.Fragment(p.children...)
}

// Use retrieves the context value from the nearest Provider ancestor.
// If no Provider is found, returns the default value.
//
// This is a reactive hook: it subscribes the calling component to the
// context value. When the Provider re-renders with a new value, this
// component will also re-render.
//
// This MUST be called unconditionally during render (hook-order semantics).
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

	// Look up the contextValue in the owner hierarchy
	owner := getCurrentOwner()
	if owner != nil {
		if value := owner.GetValue(c.key); value != nil {
			if cv, ok := value.(*contextValue[T]); ok {
				// Get the value via the signal's Get() method.
				// This subscribes the current listener (the component
				// being rendered) to the signal, making Use() reactive.
				return cv.signal.Get()
			}
		}
	}

	return c.defaultValue
}

// Default returns the default value for this context.
func (c *Context[T]) Default() T {
	return c.defaultValue
}
