package resource

import "time"

// StaleTime sets the duration before data is considered stale.
func (r *Resource[T]) StaleTime(d time.Duration) *Resource[T] {
	r.mu.Lock()
	r.staleTime = d
	r.mu.Unlock()
	return r
}

// RetryOnError sets the number of retries and delay between them.
func (r *Resource[T]) RetryOnError(count int, delay time.Duration) *Resource[T] {
	r.mu.Lock()
	r.retryCount = count
	r.retryDelay = delay
	r.mu.Unlock()
	return r
}

// OnSuccess registers a callback to be called when data is successfully loaded.
func (r *Resource[T]) OnSuccess(fn func(T)) *Resource[T] {
	r.mu.Lock()
	r.onSuccess = fn
	r.mu.Unlock()
	return r
}

// OnError registers a callback to be called when data loading fails.
func (r *Resource[T]) OnError(fn func(error)) *Resource[T] {
	r.mu.Lock()
	r.onError = fn
	r.mu.Unlock()
	return r
}
