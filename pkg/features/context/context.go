package context

import (
	"github.com/vango-go/vango/pkg/vango"
	"github.com/vango-go/vango/pkg/vdom"
)

// Context provides a way to pass data through the component tree.
type Context[T any] struct {
	defaultValue T
}

// Create creates a new Context with a default value.
func Create[T any](defaultValue T) *Context[T] {
	return &Context[T]{
		defaultValue: defaultValue,
	}
}

// Provider creates a component that provides a value to its children.
func (c *Context[T]) Provider(value T, children ...any) *vdom.VNode {
	// We wrap the children in a component to create a new Owner scope,
	// and set the context value on that scope.
	comp := vdom.Func(func() *vdom.VNode {
		vango.SetContext(c, value)
		return vdom.Fragment(children...)
	})

	return &vdom.VNode{
		Kind: vdom.KindComponent,
		Comp: comp,
	}
}

// Use retrieves the current context value.
// If no provider is found, returns the default value.
func (c *Context[T]) Use() T {
	val := vango.GetContext(c)
	if val == nil {
		return c.defaultValue
	}
	if tVal, ok := val.(T); ok {
		return tVal
	}
	return c.defaultValue
}
