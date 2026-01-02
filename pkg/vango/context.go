package vango

import "context"

// Ctx is the runtime context interface available during render, effects, and event handlers.
// It provides access to the current request, session, and utility methods.
//
// Use UseCtx() to obtain the current context within a component.
type Ctx interface {
	// Dispatch queues a function to run on the session's event loop.
	// This is safe to call from any goroutine and is the correct way to
	// update signals from asynchronous operations.
	//
	// Example:
	//     go func() {
	//         user, err := db.Users.FindByID(ctx.StdContext(), id)
	//         ctx.Dispatch(func() {
	//             if err != nil { errorSignal.Set(err) } else { userSignal.Set(user) }
	//         })
	//     }()
	Dispatch(fn func())

	// StdContext returns the standard library context with trace propagation.
	// Use this when calling external services or database drivers.
	//
	// Example:
	//     row := db.QueryRowContext(ctx.StdContext(), "SELECT * FROM users WHERE id = $1", userID)
	StdContext() context.Context
}

// UseCtx returns the current runtime context for the active session tick.
// It MUST only be called during a component render, effect, or event handler.
//
// Returns nil if called outside of a render/effect/handler context.
//
// Example:
//
//	func MyComponent() vango.Component {
//	    return vango.Func(func() *vango.VNode {
//	        ctx := vango.UseCtx()
//	        user := vango.NewSignal[*User](nil)
//
//	        vango.Effect(func() vango.Cleanup {
//	            cctx, cancel := context.WithCancel(ctx.StdContext())
//	            go func() {
//	                u, err := db.Users.FindByID(cctx, userID)
//	                ctx.Dispatch(func() {
//	                    if err == nil { user.Set(u) }
//	                })
//	            }()
//	            return cancel
//	        })
//
//	        return Div(Text(user.Get().Name))
//	    })
//	}
func UseCtx() Ctx {
	c := getCurrentCtx()
	if c == nil {
		return nil
	}
	if ctx, ok := c.(Ctx); ok {
		return ctx
	}
	return nil
}

// SetContext sets a context value for the current component scope.
// This value will be available to all descendants via GetContext.
func SetContext(key, value any) {
	owner := getCurrentOwner()
	if owner != nil {
		owner.SetValue(key, value)
	}
}

// GetContext retrieves a context value from the nearest provider in the hierarchy.
// Returns nil if no value is found.
func GetContext(key any) any {
	owner := getCurrentOwner()
	if owner != nil {
		return owner.GetValue(key)
	}
	return nil
}

// SetValue sets a value on this Owner.
func (o *Owner) SetValue(key, value any) {
	o.valuesMu.Lock()
	defer o.valuesMu.Unlock()

	if o.values == nil {
		o.values = make(map[any]any)
	}
	o.values[key] = value
}

// GetValue retrieves a value from this Owner or its parents.
func (o *Owner) GetValue(key any) any {
	// Check self
	o.valuesMu.RLock()
	if o.values != nil {
		if val, ok := o.values[key]; ok {
			o.valuesMu.RUnlock()
			return val
		}
	}
	o.valuesMu.RUnlock()

	// Check parent
	if o.parent != nil {
		return o.parent.GetValue(key)
	}

	return nil
}
