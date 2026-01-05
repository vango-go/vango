package vango

// =============================================================================
// Phase 16: Action Options (SPEC_ADDENDUM.md §A.1.4)
// =============================================================================

// ActionOption is an option for configuring an Action.
type ActionOption interface {
	isActionOption()
	applyAction(a any) // Uses any to avoid generics in interface
}

// actionOptionFunc is a helper for creating ActionOption implementations.
type actionOptionFunc func(a any)

func (f actionOptionFunc) isActionOption() {}
func (f actionOptionFunc) applyAction(a any) {
	f(a)
}

// =============================================================================
// Concurrency Policies
// =============================================================================

// CancelLatest returns an option that cancels prior in-flight work on new Run.
// This is the default concurrency policy.
//
// When Run is called while another operation is running:
//   - The prior operation is cancelled via context cancellation
//   - The new operation starts immediately
//   - Run returns true
func CancelLatest() ActionOption {
	return actionOptionFunc(func(a any) {
		setActionPolicy(a, PolicyCancelLatest)
	})
}

// DropWhileRunning returns an option that ignores Run while an operation is running.
//
// When Run is called while another operation is running:
//   - The call is ignored (no-op)
//   - Run returns false
func DropWhileRunning() ActionOption {
	return actionOptionFunc(func(a any) {
		setActionPolicy(a, PolicyDropWhileRunning)
	})
}

// Queue returns an option that queues Run calls and executes them sequentially.
//
// When Run is called while another operation is running:
//   - If queue has room, the call is queued. Run returns true.
//   - If queue is full, the call is rejected. Run returns false and
//     the Action transitions to ActionError with ErrQueueFull.
//
// maxQueue specifies the maximum number of queued calls (must be > 0).
func Queue(maxQueue int) ActionOption {
	if maxQueue <= 0 {
		maxQueue = 10 // Sensible default
	}
	return actionOptionFunc(func(a any) {
		setActionPolicy(a, PolicyQueue)
		setActionQueueMax(a, maxQueue)
	})
}

// Helper functions to set fields without exposing generics

func setActionPolicy(a any, policy ConcurrencyPolicy) {
	switch action := a.(type) {
	case interface{ setPolicy(ConcurrencyPolicy) }:
		action.setPolicy(policy)
	}
}

func setActionQueueMax(a any, max int) {
	switch action := a.(type) {
	case interface{ setQueueMax(int) }:
		action.setQueueMax(max)
	}
}

// =============================================================================
// Naming and Observability
// =============================================================================

// ActionTxName sets the transaction name for observability.
// State transitions will appear in DevTools as:
//   - Action:<name>:running
//   - Action:<name>:success
//   - Action:<name>:error
//
// Without this option, a default name based on component/file is used.
func ActionTxName(name string) ActionOption {
	return actionOptionFunc(func(a any) {
		setActionTxName(a, name)
	})
}

// OnActionStart registers a callback that runs when the action starts.
// The callback runs on the session loop (inside a Dispatch).
func OnActionStart(fn func()) ActionOption {
	return actionOptionFunc(func(a any) {
		setActionOnStart(a, fn)
	})
}

// OnActionSuccess registers a callback that runs on successful completion.
// The callback runs on the session loop and receives the result.
//
// Note: Due to Go's type system limitations, this version takes func(any).
// For type-safe callbacks, use the OnSuccess method on the Action directly
// or cast the result in the callback.
func OnActionSuccess(fn func(any)) ActionOption {
	return actionOptionFunc(func(a any) {
		setActionOnSuccessAny(a, fn)
	})
}

// OnActionError registers a callback that runs when the action fails.
// The callback runs on the session loop and receives the error.
func OnActionError(fn func(error)) ActionOption {
	return actionOptionFunc(func(a any) {
		setActionOnError(a, fn)
	})
}

// Helper functions for setting callbacks

func setActionTxName(a any, name string) {
	switch action := a.(type) {
	case interface{ setTxName(string) }:
		action.setTxName(name)
	}
}

func setActionOnStart(a any, fn func()) {
	switch action := a.(type) {
	case interface{ setOnStart(func()) }:
		action.setOnStart(fn)
	}
}

func setActionOnSuccessAny(a any, fn func(any)) {
	switch action := a.(type) {
	case interface{ setOnSuccessAny(func(any)) }:
		action.setOnSuccessAny(fn)
	}
}

func setActionOnError(a any, fn func(error)) {
	switch action := a.(type) {
	case interface{ setOnError(func(error)) }:
		action.setOnError(fn)
	}
}
