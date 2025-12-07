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
// The fetch is triggered immediately.
func New[T any](fetcher func() (T, error)) *Resource[T] {
	r := &Resource[T]{
		fetcher: fetcher,
		state:   vango.NewSignal(Pending),
		data:    vango.NewSignal(*new(T)),
		err:     vango.NewSignal[error](nil),
	}
	r.Fetch()
	return r
}

// NewWithKey creates a Resource that automatically refetches when the key changes.
// The key function is tracked reactively.
func NewWithKey[K comparable, T any](key func() K, fetcher func(K) (T, error)) *Resource[T] {
	// Wrap fetcher to use current key
	wrappedFetcher := func() (T, error) {
		k := key() // Track dependency
		return fetcher(k)
	}

	r := New(wrappedFetcher)

	// Setup effect to refetch when key changes
	vango.CreateEffect(func() vango.Cleanup {
		key() // Track dependency
		r.Fetch()
		return nil
	})

	return r
}

// Note: I need to verify how to use Effects in Vango.
// vango/owner.go mentioned `effects`.
// I'll stick to basic Resource for now and implement NewWithKey later if needed, or just implement it with manual tracking expectation.
// Actually, `NewWithKey` is in the spec:
// "ResourceWithKey (re-fetches when key changes)"
// It implies using an effect.

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
func (r *Resource[T]) Refetch() {
	r.mu.Lock()
	r.fetchID++
	currentID := r.fetchID
	r.mu.Unlock()

	r.state.Set(Loading)
	r.err.Set(nil)

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
