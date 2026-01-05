package resource

import (
	"github.com/vango-go/vango/pkg/vdom"
)

// Match renders different content based on the resource state.
func (r *Resource[T]) Match(handlers ...Handler[T]) *vdom.VNode {
	for _, h := range handlers {
		if node := h.handle(r); node != nil {
			// Type assertion to handle different return types if we generalize later
			// For now assuming *vdom.VNode
			if vnode, ok := node.(*vdom.VNode); ok {
				return vnode
			}
		}
	}
	return nil
}

// Handler implementations

type pendingHandler[T any] struct {
	fn func() *vdom.VNode
}

func (h pendingHandler[T]) handle(r *Resource[T]) interface{} {
	if r.State() == Pending {
		return h.fn()
	}
	return nil
}

type loadingHandler[T any] struct {
	fn func() *vdom.VNode
}

func (h loadingHandler[T]) handle(r *Resource[T]) interface{} {
	if r.State() == Loading {
		return h.fn()
	}
	return nil
}

type errorHandler[T any] struct {
	fn func(error) *vdom.VNode
}

func (h errorHandler[T]) handle(r *Resource[T]) interface{} {
	if r.State() == Error {
		return h.fn(r.Error())
	}
	return nil
}

type readyHandler[T any] struct {
	fn func(T) *vdom.VNode
}

func (h readyHandler[T]) handle(r *Resource[T]) interface{} {
	if r.State() == Ready {
		return h.fn(r.Data())
	}
	return nil
}

type loadingOrPendingHandler[T any] struct {
	fn func() *vdom.VNode
}

func (h loadingOrPendingHandler[T]) handle(r *Resource[T]) interface{} {
	state := r.State()
	if state == Loading || state == Pending {
		return h.fn()
	}
	return nil
}

// Constructors

// OnPending handles the Pending state.
func OnPending[T any](fn func() *vdom.VNode) Handler[T] {
	return pendingHandler[T]{fn: fn}
}

// OnLoading handles the Loading state.
func OnLoading[T any](fn func() *vdom.VNode) Handler[T] {
	return loadingHandler[T]{fn: fn}
}

// OnError handles the Error state.
func OnError[T any](fn func(error) *vdom.VNode) Handler[T] {
	return errorHandler[T]{fn: fn}
}

// OnReady handles the Ready state.
func OnReady[T any](fn func(T) *vdom.VNode) Handler[T] {
	return readyHandler[T]{fn: fn}
}

// OnLoadingOrPending handles both Loading and Pending states.
func OnLoadingOrPending[T any](fn func() *vdom.VNode) Handler[T] {
	return loadingOrPendingHandler[T]{fn: fn}
}
