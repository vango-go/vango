package server

// RunRouteMiddleware executes a route middleware chain and then calls final.
//
// Middleware can short-circuit by returning nil without calling next.
// In that case ranFinal will be false and err will be nil.
func RunRouteMiddleware(ctx Ctx, middleware []RouteMiddleware, final func() error) (ranFinal bool, err error) {
	if final == nil {
		return false, nil
	}

	ran := false
	wrappedFinal := func() error {
		ran = true
		return final()
	}

	if len(middleware) == 0 {
		return true, wrappedFinal()
	}

	index := 0
	var next func() error
	next = func() error {
		if index >= len(middleware) {
			return wrappedFinal()
		}

		mw := middleware[index]
		index++
		if mw == nil {
			return next()
		}

		return mw.Handle(ctx, next)
	}

	err = next()
	return ran, err
}
