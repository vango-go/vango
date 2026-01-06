package vango

import (
	"context"
	"sync/atomic"
	"time"
)

// =============================================================================
// Phase 16: Effect Helpers (SPEC_ADDENDUM.md §A.2)
// =============================================================================

// Interval schedules periodic ticks that execute fn on the session loop.
// It handles cleanup automatically - the returned Cleanup stops future ticks.
//
// By default, the first tick occurs after duration d. Use IntervalImmediate()
// to trigger the first tick immediately.
//
// MUST be called inside an Effect and the returned Cleanup SHOULD be returned
// from that Effect:
//
//	vango.CreateEffect(func() vango.Cleanup {
//	    return vango.Interval(time.Second, func() {
//	        counter.Inc()
//	    })
//	})
//
// See SPEC_ADDENDUM.md §A.2.1.
func Interval(d time.Duration, fn func(), opts ...IntervalOption) Cleanup {
	// Check for prefetch mode (Phase 7: Routing, Section 8.3.2)
	// Interval is forbidden during prefetch - return no-op cleanup
	if !checkPrefetchSideEffect("Interval") {
		return func() {}
	}

	ctx := UseCtx()
	if ctx == nil {
		panic(ErrEffectContext)
	}

	// Apply options
	var cfg intervalConfig
	for _, opt := range opts {
		opt.applyInterval(&cfg)
	}

	// Create done channel for cleanup
	done := make(chan struct{})

	// Wrap fn with transaction naming
	wrappedFn := func() {
		TxNamed(cfg.getTxName(), fn)
	}

	// Start ticker goroutine
	go func() {
		// Handle immediate first tick
		if cfg.immediate {
			select {
			case <-done:
				return
			default:
				ctx.Dispatch(wrappedFn)
			}
		}

		ticker := time.NewTicker(d)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				ctx.Dispatch(wrappedFn)
			case <-done:
				return
			}
		}
	}()

	// Return cleanup function
	return func() {
		close(done)
	}
}

// intervalConfig holds configuration from IntervalOptions.
type intervalConfig struct {
	txName    string
	immediate bool
}

func (c *intervalConfig) getTxName() string {
	if c.txName != "" {
		return "Interval:" + c.txName
	}
	return "Interval"
}

// IntervalOption is an option for configuring Interval.
type IntervalOption interface {
	isIntervalOption()
	applyInterval(cfg *intervalConfig)
}

type intervalOptionFunc func(*intervalConfig)

func (f intervalOptionFunc) isIntervalOption()              {}
func (f intervalOptionFunc) applyInterval(cfg *intervalConfig) { f(cfg) }

// IntervalTxName sets the transaction name for observability.
// Ticks will appear in DevTools as Interval:<name>.
func IntervalTxName(name string) IntervalOption {
	return intervalOptionFunc(func(cfg *intervalConfig) {
		cfg.txName = name
	})
}

// IntervalImmediate causes the first tick to occur immediately instead of
// after the duration.
func IntervalImmediate() IntervalOption {
	return intervalOptionFunc(func(cfg *intervalConfig) {
		cfg.immediate = true
	})
}

// =============================================================================
// Subscribe
// =============================================================================

// Stream is an interface for event streams that support subscription.
// The Subscribe method returns an unsubscribe function.
type Stream[T any] interface {
	Subscribe(handler func(T)) (unsubscribe func())
}

// Subscribe connects to an event stream and invokes fn for each message
// on the session loop. The returned Cleanup unsubscribes from the stream.
//
// MUST be called inside an Effect and the returned Cleanup SHOULD be returned
// from that Effect:
//
//	vango.CreateEffect(func() vango.Cleanup {
//	    return vango.Subscribe(ws.Messages, func(msg Message) {
//	        messages.Append(msg)
//	    })
//	})
//
// See SPEC_ADDENDUM.md §A.2.2.
func Subscribe[T any](stream Stream[T], fn func(T), opts ...SubscribeOption) Cleanup {
	// Check for prefetch mode (Phase 7: Routing, Section 8.3.2)
	// Subscribe is forbidden during prefetch - return no-op cleanup
	if !checkPrefetchSideEffect("Subscribe") {
		return func() {}
	}

	ctx := UseCtx()
	if ctx == nil {
		panic(ErrEffectContext)
	}

	// Apply options
	var cfg subscribeConfig
	for _, opt := range opts {
		opt.applySubscribe(&cfg)
	}

	// Subscribe to stream with dispatch wrapper
	unsubscribe := stream.Subscribe(func(msg T) {
		ctx.Dispatch(func() {
			TxNamed(cfg.getTxName(), func() {
				fn(msg)
			})
		})
	})

	return unsubscribe
}

// subscribeConfig holds configuration from SubscribeOptions.
type subscribeConfig struct {
	txName string
}

func (c *subscribeConfig) getTxName() string {
	if c.txName != "" {
		return "Subscribe:" + c.txName
	}
	return "Subscribe"
}

// SubscribeOption is an option for configuring Subscribe.
type SubscribeOption interface {
	isSubscribeOption()
	applySubscribe(cfg *subscribeConfig)
}

type subscribeOptionFunc func(*subscribeConfig)

func (f subscribeOptionFunc) isSubscribeOption()                 {}
func (f subscribeOptionFunc) applySubscribe(cfg *subscribeConfig) { f(cfg) }

// SubscribeTxName sets the transaction name for observability.
// Messages will appear in DevTools as Subscribe:<name>.
func SubscribeTxName(name string) SubscribeOption {
	return subscribeOptionFunc(func(cfg *subscribeConfig) {
		cfg.txName = name
	})
}

// =============================================================================
// GoLatest
// =============================================================================

// goLatestState holds per-call-site state for GoLatest.
// This is stored in Effect.callSiteData for persistence across effect reruns.
type goLatestState[K comparable] struct {
	lastKey         K
	initialized     bool
	cancel          context.CancelFunc
	seq             uint64
	lastBudgetError time.Time // Track when we last reported budget error
}

// GoLatest is the standard helper for async integration work inside Effect.
// It handles key coalescing, stale suppression, cancellation, and dispatch.
//
// Key semantics:
//   - Same key as previous call: No new work starts (existing work continues)
//   - Different key: Cancels prior work, starts new work
//   - Use GoLatestForceRestart() to restart even with same key
//
// MUST be called inside an Effect:
//
//	vango.CreateEffect(func() vango.Cleanup {
//	    q := query.Get()
//	    return vango.GoLatest(q,
//	        func(ctx context.Context, q string) ([]User, error) {
//	            return api.SearchUsers(ctx, q)
//	        },
//	        func(users []User, err error) {
//	            results.Set(users)
//	        },
//	    )
//	})
//
// See SPEC_ADDENDUM.md §A.2.3.
func GoLatest[K comparable, R any](
	key K,
	work func(ctx context.Context, key K) (R, error),
	apply func(result R, err error),
	opts ...GoLatestOption,
) Cleanup {
	// Check for prefetch mode (Phase 7: Routing, Section 8.3.2)
	// GoLatest is forbidden during prefetch - return no-op cleanup
	if !checkPrefetchSideEffect("GoLatest") {
		return func() {}
	}

	ctx := UseCtx()
	if ctx == nil {
		panic(ErrGoLatestContext)
	}

	// Get effect-local state for this call site
	state := GetEffectCallSiteState(func() *goLatestState[K] {
		return &goLatestState[K]{}
	})
	if state == nil {
		panic(ErrGoLatestContext)
	}

	// Apply options
	var cfg goLatestConfig
	for _, opt := range opts {
		opt.applyGoLatest(&cfg)
	}

	// Key coalescing: if same key and no force restart, don't start new work
	if state.initialized && state.lastKey == key && !cfg.forceRestart {
		// Same key, work may be in flight - don't cancel, don't restart
		// Return a cleanup that cancels on unmount (not on effect rerun)
		return func() {
			if state.cancel != nil {
				state.cancel()
			}
		}
	}

	// Different key or first call or force restart: cancel old work if any
	if state.cancel != nil {
		state.cancel()
	}

	// Update state
	state.initialized = true
	state.lastKey = key
	state.seq++
	mySeq := state.seq

	// Create cancellable context
	stdCtx := ctx.StdContext()
	if stdCtx == nil {
		stdCtx = context.Background()
	}
	workCtx, cancel := context.WithCancel(stdCtx)
	state.cancel = cancel

	// Check storm budget before spawning work
	if budget := ctx.StormBudget(); budget != nil {
		if err := budget.CheckGoLatest(); err != nil {
			// Budget exceeded - invoke apply(zero, err) at most once per window
			now := time.Now()
			if now.Sub(state.lastBudgetError) >= time.Second {
				state.lastBudgetError = now
				ctx.Dispatch(func() {
					TxNamed(cfg.getTxName(), func() {
						var zero R
						apply(zero, ErrBudgetExceeded)
					})
				})
			}
			// Return no-op cleanup since no work was started
			return func() {}
		}
	}

	// Execute work off the session loop
	go func() {
		result, err := work(workCtx, key)

		// Check if cancelled
		if workCtx.Err() != nil {
			return
		}

		// Apply result on session loop
		ctx.Dispatch(func() {
			// Check if stale (newer work started)
			if state.seq != mySeq {
				// Emit telemetry for stale-ignored result
				// (privacy-safe: don't log actual key)
				if Debug.LogStormBudget {
					// Log stale event
				}
				return
			}

			TxNamed(cfg.getTxName(), func() {
				apply(result, err)
			})
		})
	}()

	// Return cleanup that cancels work
	return func() {
		cancel()
	}
}

// goLatestConfig holds configuration from GoLatestOptions.
type goLatestConfig struct {
	txName       string
	forceRestart bool
}

func (c *goLatestConfig) getTxName() string {
	if c.txName != "" {
		return "GoLatest:" + c.txName
	}
	return "GoLatest"
}

// GoLatestOption is an option for configuring GoLatest.
type GoLatestOption interface {
	isGoLatestOption()
	applyGoLatest(cfg *goLatestConfig)
}

type goLatestOptionFunc func(*goLatestConfig)

func (f goLatestOptionFunc) isGoLatestOption()                {}
func (f goLatestOptionFunc) applyGoLatest(cfg *goLatestConfig) { f(cfg) }

// GoLatestTxName sets the transaction name for observability.
// Work will appear in DevTools as GoLatest:<name>.
func GoLatestTxName(name string) GoLatestOption {
	return goLatestOptionFunc(func(cfg *goLatestConfig) {
		cfg.txName = name
	})
}

// GoLatestForceRestart causes work to restart even when the key is unchanged.
// By default, same key = no new work (existing work continues).
func GoLatestForceRestart() GoLatestOption {
	return goLatestOptionFunc(func(cfg *goLatestConfig) {
		cfg.forceRestart = true
	})
}

// =============================================================================
// Timeout - Additional helper for deadline-based work
// =============================================================================

// Timeout creates a one-shot timer that executes fn after duration d.
// Returns a Cleanup that cancels the timer if called before it fires.
//
// This is a simpler alternative to Interval for single delayed operations:
//
//	vango.CreateEffect(func() vango.Cleanup {
//	    return vango.Timeout(5*time.Second, func() {
//	        showTooltip.Set(true)
//	    })
//	})
func Timeout(d time.Duration, fn func(), opts ...TimeoutOption) Cleanup {
	// Check for prefetch mode (Phase 7: Routing, Section 8.3.2)
	// Timeout is forbidden during prefetch - return no-op cleanup
	if !checkPrefetchSideEffect("Timeout") {
		return func() {}
	}

	ctx := UseCtx()
	if ctx == nil {
		panic(ErrEffectContext)
	}

	// Apply options
	var cfg timeoutConfig
	for _, opt := range opts {
		opt.applyTimeout(&cfg)
	}

	// Use atomic to prevent double-fire after cancel
	var fired atomic.Bool
	timer := time.AfterFunc(d, func() {
		if fired.CompareAndSwap(false, true) {
			ctx.Dispatch(func() {
				TxNamed(cfg.getTxName(), fn)
			})
		}
	})

	return func() {
		fired.Store(true)
		timer.Stop()
	}
}

// timeoutConfig holds configuration from TimeoutOptions.
type timeoutConfig struct {
	txName string
}

func (c *timeoutConfig) getTxName() string {
	if c.txName != "" {
		return "Timeout:" + c.txName
	}
	return "Timeout"
}

// TimeoutOption is an option for configuring Timeout.
type TimeoutOption interface {
	isTimeoutOption()
	applyTimeout(cfg *timeoutConfig)
}

type timeoutOptionFunc func(*timeoutConfig)

func (f timeoutOptionFunc) isTimeoutOption()               {}
func (f timeoutOptionFunc) applyTimeout(cfg *timeoutConfig) { f(cfg) }

// TimeoutTxName sets the transaction name for observability.
// The timeout will appear in DevTools as Timeout:<name>.
func TimeoutTxName(name string) TimeoutOption {
	return timeoutOptionFunc(func(cfg *timeoutConfig) {
		cfg.txName = name
	})
}
