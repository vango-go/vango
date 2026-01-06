package vango

import (
	"context"
	"sync"
	"sync/atomic"
)

// =============================================================================
// Phase 16: Action API (SPEC_ADDENDUM.md §A.1)
// =============================================================================

// ActionState represents the current state of an Action.
type ActionState int

const (
	// ActionIdle is the initial state before any Run call.
	ActionIdle ActionState = iota

	// ActionRunning indicates an async operation is in progress.
	ActionRunning

	// ActionSuccess indicates the last operation completed successfully.
	ActionSuccess

	// ActionError indicates the last operation failed.
	ActionError
)

// String returns a human-readable name for the action state.
func (s ActionState) String() string {
	switch s {
	case ActionIdle:
		return "idle"
	case ActionRunning:
		return "running"
	case ActionSuccess:
		return "success"
	case ActionError:
		return "error"
	default:
		return "unknown"
	}
}

// ConcurrencyPolicy defines how an Action handles concurrent Run calls.
type ConcurrencyPolicy int

const (
	// PolicyCancelLatest cancels prior in-flight work when Run is called again.
	// This is the default policy.
	PolicyCancelLatest ConcurrencyPolicy = iota

	// PolicyDropWhileRunning ignores Run calls while work is in progress.
	PolicyDropWhileRunning

	// PolicyQueue queues Run calls and executes them sequentially.
	PolicyQueue
)

// Action is the structured primitive for async mutations.
// It bundles loading/error/success state, cancellation, and dispatch into
// a standard, auditable unit.
//
// Action uses hook-order semantics: it MUST be created unconditionally during
// render and returns a stable pointer that persists across re-renders.
//
// See SPEC_ADDENDUM.md §A.1 for full specification.
type Action[A any, R any] struct {
	// The async work function
	do func(ctx context.Context, arg A) (R, error)

	// Current state
	state *Signal[ActionState]

	// Last successful result
	result *Signal[R]

	// Last error
	err *Signal[error]

	// Runtime context for Dispatch
	ctx Ctx

	// Concurrency policy
	policy ConcurrencyPolicy

	// Queue for PolicyQueue (max size)
	queueMax  int
	queue     []A
	queueMu   sync.Mutex
	queueCond *sync.Cond

	// Cancellation for current operation
	cancel     context.CancelFunc
	cancelMu   sync.Mutex
	currentSeq uint64

	// Options
	txName    string
	onStart   func()
	onSuccess func(R)
	onError   func(error)

	// Atomic counter for sequence numbers
	seq atomic.Uint64
}

// NewAction creates a new Action with the given work function.
// This is a hook-like API and MUST be called unconditionally during render.
//
// The do function executes off the session event loop. All state transitions
// are applied on the session loop via ctx.Dispatch().
//
// Options:
//   - CancelLatest() - Cancel prior in-flight on new Run (default)
//   - DropWhileRunning() - Ignore Run while Running
//   - Queue(max) - Buffer up to max, execute sequentially
//   - ActionTxName(name) - Set transaction name for observability
//   - OnActionStart(fn) - Called when action starts
//   - OnActionSuccess(fn) - Called on success with result
//   - OnActionError(fn) - Called on error
//
// Example:
//
//	save := vango.NewAction(
//	    func(ctx context.Context, p Profile) (Profile, error) {
//	        return api.SaveProfile(ctx, p)
//	    },
//	    vango.CancelLatest(),
//	    vango.ActionTxName("profile:save"),
//	)
func NewAction[A any, R any](
	do func(ctx context.Context, arg A) (R, error),
	opts ...ActionOption,
) *Action[A, R] {
	owner := getCurrentOwner()
	inRender := owner != nil && isInRender()

	// Track hook call for dev-mode order validation
	TrackHook(HookAction)

	var a *Action[A, R]
	first := true
	if inRender {
		// Use hook slot for stable identity across renders
		if slot := owner.UseHookSlot(); slot != nil {
			existing, ok := slot.(*Action[A, R])
			if !ok {
				panic("vango: hook slot type mismatch for Action")
			}
			a = existing
			first = false
		}
	}

	if a == nil {
		a = &Action[A, R]{
			policy: PolicyCancelLatest, // Default
		}
		if inRender {
			owner.SetHookSlot(a)
		}
	}

	// Always update do function in case closures changed
	a.do = do

	// Signals are hook-slot stabilized when called during render
	a.state = NewSignal(ActionIdle)
	a.result = NewSignal(*new(R))
	a.err = NewSignal[error](nil)

	if a.ctx == nil {
		ctx := UseCtx()
		if ctx == nil {
			panic("NewAction: UseCtx() returned nil - must be called during render or effect")
		}
		a.ctx = ctx
	}

	if first {
		// Apply options only on first creation
		for _, opt := range opts {
			opt.applyAction(a)
		}

		// Initialize queue condition if using queue policy
		if a.policy == PolicyQueue && a.queueMax > 0 {
			a.queueCond = sync.NewCond(&a.queueMu)
		}
	}

	return a
}

// Run starts the async operation with the given argument.
// Returns true if the call was accepted (started or queued), false if rejected.
//
// Behavior depends on concurrency policy:
//   - CancelLatest: Cancels any in-flight work, starts new. Returns true.
//   - DropWhileRunning: Ignores call if already running. Returns false if dropped.
//   - Queue: Queues the call if running. Returns false if queue is full.
func (a *Action[A, R]) Run(arg A) bool {
	switch a.policy {
	case PolicyCancelLatest:
		return a.runCancelLatest(arg)
	case PolicyDropWhileRunning:
		return a.runDropWhileRunning(arg)
	case PolicyQueue:
		return a.runQueue(arg)
	default:
		return a.runCancelLatest(arg)
	}
}

func (a *Action[A, R]) runCancelLatest(arg A) bool {
	// Cancel any existing operation
	a.cancelMu.Lock()
	if a.cancel != nil {
		a.cancel()
	}
	a.cancelMu.Unlock()

	return a.startWork(arg)
}

func (a *Action[A, R]) runDropWhileRunning(arg A) bool {
	// Check if already running
	if a.state.Peek() == ActionRunning {
		return false
	}

	return a.startWork(arg)
}

func (a *Action[A, R]) runQueue(arg A) bool {
	a.queueMu.Lock()
	defer a.queueMu.Unlock()

	// If not running, start immediately
	if a.state.Peek() != ActionRunning {
		return a.startWork(arg)
	}

	// Check queue capacity
	if len(a.queue) >= a.queueMax {
		// Queue full - set error state
		a.ctx.Dispatch(func() {
			a.err.Set(ErrQueueFull)
			a.state.Set(ActionError)
		})
		return false
	}

	// Add to queue
	a.queue = append(a.queue, arg)
	return true
}

func (a *Action[A, R]) startWork(arg A) bool {
	// Check storm budget before starting work
	if budget := a.ctx.StormBudget(); budget != nil {
		if err := budget.CheckAction(); err != nil {
			// Budget exceeded - transition to Error state
			a.ctx.Dispatch(func() {
				TxNamed(a.getTxName("budget_exceeded"), func() {
					a.err.Set(ErrBudgetExceeded)
					a.state.Set(ActionError)
					if a.onError != nil {
						a.onError(ErrBudgetExceeded)
					}
				})
			})
			return false
		}
	}

	// Get new sequence number
	seq := a.seq.Add(1)

	// Create cancellable context
	stdCtx := a.ctx.StdContext()
	if stdCtx == nil {
		stdCtx = context.Background()
	}
	workCtx, cancel := context.WithCancel(stdCtx)

	a.cancelMu.Lock()
	a.cancel = cancel
	a.currentSeq = seq
	a.cancelMu.Unlock()

	// Set Running state on session loop
	a.ctx.Dispatch(func() {
		TxNamed(a.getTxName("running"), func() {
			a.state.Set(ActionRunning)
			a.err.Set(nil)
			if a.onStart != nil {
				a.onStart()
			}
		})
	})

	// Execute work off the session loop
	go func() {
		result, err := a.do(workCtx, arg)

		// Check if cancelled
		if workCtx.Err() != nil {
			return
		}

		// Apply result on session loop
		a.ctx.Dispatch(func() {
			// Check if stale (newer operation started)
			a.cancelMu.Lock()
			if a.currentSeq != seq {
				a.cancelMu.Unlock()
				return
			}
			a.cancelMu.Unlock()

			if err != nil {
				TxNamed(a.getTxName("error"), func() {
					a.err.Set(err)
					a.state.Set(ActionError)
					if a.onError != nil {
						a.onError(err)
					}
				})
			} else {
				TxNamed(a.getTxName("success"), func() {
					a.result.Set(result)
					a.state.Set(ActionSuccess)
					if a.onSuccess != nil {
						a.onSuccess(result)
					}
				})
			}

			// Process queue if using queue policy
			if a.policy == PolicyQueue {
				a.processQueue()
			}
		})
	}()

	return true
}

func (a *Action[A, R]) processQueue() {
	a.queueMu.Lock()
	if len(a.queue) == 0 {
		a.queueMu.Unlock()
		return
	}
	arg := a.queue[0]
	a.queue = a.queue[1:]
	a.queueMu.Unlock()

	a.startWork(arg)
}

func (a *Action[A, R]) getTxName(state string) string {
	if a.txName != "" {
		return "Action:" + a.txName + ":" + state
	}
	// Default name (could include file:line in dev mode)
	return "Action:" + state
}

// State returns the current ActionState.
// This is reactive - reading it inside an Effect will subscribe to changes.
func (a *Action[A, R]) State() ActionState {
	return a.state.Get()
}

// Result returns the last successful result and true, or zero value and false
// if no success has occurred yet.
func (a *Action[A, R]) Result() (R, bool) {
	if a.state.Get() == ActionSuccess {
		return a.result.Get(), true
	}
	return *new(R), false
}

// Error returns the last error, or nil if no error occurred.
func (a *Action[A, R]) Error() error {
	return a.err.Get()
}

// Reset sets the Action back to ActionIdle and clears stored result/error.
func (a *Action[A, R]) Reset() {
	// Cancel any in-flight work
	a.cancelMu.Lock()
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}
	a.cancelMu.Unlock()

	// Clear queue
	if a.policy == PolicyQueue {
		a.queueMu.Lock()
		a.queue = nil
		a.queueMu.Unlock()
	}

	// Reset state
	a.state.Set(ActionIdle)
	a.result.Set(*new(R))
	a.err.Set(nil)
}

// IsIdle returns true if the action is in the Idle state.
func (a *Action[A, R]) IsIdle() bool {
	return a.state.Get() == ActionIdle
}

// IsRunning returns true if the action is in the Running state.
func (a *Action[A, R]) IsRunning() bool {
	return a.state.Get() == ActionRunning
}

// IsSuccess returns true if the action is in the Success state.
func (a *Action[A, R]) IsSuccess() bool {
	return a.state.Get() == ActionSuccess
}

// IsError returns true if the action is in the Error state.
func (a *Action[A, R]) IsError() bool {
	return a.state.Get() == ActionError
}

// =============================================================================
// Option setters (called by ActionOption implementations)
// =============================================================================

func (a *Action[A, R]) setPolicy(p ConcurrencyPolicy) {
	a.policy = p
}

func (a *Action[A, R]) setQueueMax(max int) {
	a.queueMax = max
}

func (a *Action[A, R]) setTxName(name string) {
	a.txName = name
}

func (a *Action[A, R]) setOnStart(fn func()) {
	a.onStart = fn
}

func (a *Action[A, R]) setOnSuccessAny(fn func(any)) {
	// Wrap the any callback to call with the typed result
	a.onSuccess = func(r R) {
		fn(r)
	}
}

func (a *Action[A, R]) setOnError(fn func(error)) {
	a.onError = fn
}
