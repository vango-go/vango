package resource

import (
	"sync"
	"time"

	"github.com/vango-dev/vango/v2/pkg/vango"
)

// State represents the current state of a resource.
type State int

const (
	Pending State = iota // Initial state, before first fetch
	Loading              // Fetch in progress
	Ready                // Data successfully loaded
	Error                // Fetch failed
)

// Handler handles a specific resource state.
type Handler[T any] interface {
	handle(*Resource[T]) interface{} // Returns *vdom.VNode or nil
}

// Resource manages asynchronous data fetching and state.
type Resource[T any] struct {
	fetcher func() (T, error)
	state   *vango.Signal[State]
	data    *vango.Signal[T]
	err     *vango.Signal[error]

	// Runtime context for Dispatch (captured at creation time)
	ctx vango.Ctx

	// Options
	staleTime  time.Duration
	retryCount int
	retryDelay time.Duration
	onSuccess  func(T)
	onError    func(error)

	// Internal
	lastFetch time.Time
	fetchID   uint64 // For cancelling/ignoring outdated fetches
	mu        sync.Mutex
}

// New creates a new Resource with the given fetcher function.
// The initial fetch is scheduled via Effect (not during render) to avoid
// signal writes during the render phase.
//
// This is a hook-like API and MUST be called unconditionally during render.
// See §3.1.3 Hook-Order Semantics.
func New[T any](fetcher func() (T, error)) *Resource[T] {
	// Track hook call for dev-mode order validation
	vango.TrackHook(vango.HookResource)

	// Use hook slot for stable identity across renders
	slot := vango.UseHookSlot()
	if slot != nil {
		// Subsequent render: return existing instance
		return slot.(*Resource[T])
	}

	// First render: create new instance
	// Capture runtime context for Dispatch calls from goroutines
	ctx := vango.UseCtx()

	r := &Resource[T]{
		fetcher: fetcher,
		state:   vango.NewSignal(Pending),
		data:    vango.NewSignal(*new(T)),
		err:     vango.NewSignal[error](nil),
		ctx:     ctx,
	}

	// Store in hook slot for subsequent renders
	vango.SetHookSlot(r)

	// Schedule initial fetch via Effect (not during render)
	// This avoids signal writes during the render phase
	vango.CreateEffect(func() vango.Cleanup {
		r.Fetch()
		return nil
	})

	return r
}

// NewWithKey creates a Resource that automatically refetches when the key changes.
// The key function is tracked reactively.
//
// This is a hook-like API and MUST be called unconditionally during render.
// See §3.1.3 Hook-Order Semantics.
func NewWithKey[K comparable, T any](key func() K, fetcher func(K) (T, error)) *Resource[T] {
	// Track hook call for dev-mode order validation
	// Note: We track here instead of relying on New() because NewWithKey
	// is semantically a separate hook type (keyed resource vs simple resource)
	vango.TrackHook(vango.HookResource)

	// Use hook slot for stable identity across renders
	slot := vango.UseHookSlot()
	if slot != nil {
		// Subsequent render: return existing instance
		return slot.(*Resource[T])
	}

	// First render: create new instance
	// Capture runtime context for Dispatch calls from goroutines
	ctx := vango.UseCtx()

	// Wrap fetcher to use current key
	wrappedFetcher := func() (T, error) {
		k := key() // Track dependency
		return fetcher(k)
	}

	r := &Resource[T]{
		fetcher: wrappedFetcher,
		state:   vango.NewSignal(Pending),
		data:    vango.NewSignal(*new(T)),
		err:     vango.NewSignal[error](nil),
		ctx:     ctx,
	}

	// Store in hook slot for subsequent renders
	vango.SetHookSlot(r)

	// Setup effect to fetch initially and refetch when key changes
	// Using Effect ensures no signal writes during render
	vango.CreateEffect(func() vango.Cleanup {
		key() // Track dependency
		r.Fetch()
		return nil
	})

	return r
}

// State methods

func (r *Resource[T]) State() State {
	return r.state.Get()
}

func (r *Resource[T]) IsLoading() bool {
	s := r.state.Get()
	return s == Loading || s == Pending
}

func (r *Resource[T]) IsReady() bool {
	return r.state.Get() == Ready
}

func (r *Resource[T]) IsError() bool {
	return r.state.Get() == Error
}

// Data access methods

func (r *Resource[T]) Data() T {
	return r.data.Get()
}

func (r *Resource[T]) DataOr(fallback T) T {
	if r.IsReady() {
		return r.data.Get()
	}
	return fallback
}

func (r *Resource[T]) Error() error {
	return r.err.Get()
}

// Control methods

// Fetch triggers a data fetch. It respects StaleTime if data is already ready.
// To force a fetch, use Refetch().
func (r *Resource[T]) Fetch() {
	r.mu.Lock()
	if r.state.Peek() == Ready && time.Since(r.lastFetch) < r.staleTime {
		r.mu.Unlock()
		return
	}
	r.mu.Unlock()
	r.Refetch()
}

// Refetch forces a data fetch, bypassing cache.
// All signal writes are dispatched via ctx.Dispatch to ensure thread safety.
func (r *Resource[T]) Refetch() {
	r.mu.Lock()
	r.fetchID++
	currentID := r.fetchID
	r.mu.Unlock()

	// Set Loading state via Dispatch (even this is a write from potentially any goroutine!)
	setLoading := func() {
		r.state.Set(Loading)
		r.err.Set(nil)
	}
	if r.ctx != nil {
		r.ctx.Dispatch(setLoading)
	} else {
		setLoading()
	}

	go func() {
		// Retry logic loop
		var result T
		var err error

		maxAttempts := 1 + r.retryCount
		for i := 0; i < maxAttempts; i++ {
			if i > 0 {
				time.Sleep(r.retryDelay)
			}

			// Check if cancelled
			r.mu.Lock()
			if r.fetchID != currentID {
				r.mu.Unlock()
				return
			}
			r.mu.Unlock()

			// Perform fetch
			result, err = r.fetcher()
			if err == nil {
				break
			}
		}

		// Check if cancelled again before updating state
		r.mu.Lock()
		if r.fetchID != currentID {
			r.mu.Unlock()
			return
		}
		r.lastFetch = time.Now()
		r.mu.Unlock()

		// Update signals via Dispatch for thread safety
		updateSignals := func() {
			if err != nil {
				r.err.Set(err)
				r.state.Set(Error)
				if r.onError != nil {
					r.onError(err)
				}
			} else {
				r.data.Set(result)
				r.state.Set(Ready)
				if r.onSuccess != nil {
					r.onSuccess(result)
				}
			}
		}

		if r.ctx != nil {
			r.ctx.Dispatch(updateSignals)
		} else {
			updateSignals()
		}
	}()
}

// Invalidate marks the current data as stale.
func (r *Resource[T]) Invalidate() {
	r.mu.Lock()
	r.lastFetch = time.Time{} // Reset last fetch time
	r.mu.Unlock()
}

// Mutate optimistically updates the local data.
func (r *Resource[T]) Mutate(fn func(T) T) {
	current := r.data.Peek()
	newData := fn(current)
	r.data.Set(newData)
	// Usually one might want to mark as ready or modified?
	// Spec says "Optimistic local update", keeping simple.
}
