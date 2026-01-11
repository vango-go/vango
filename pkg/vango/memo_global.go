package vango

// =============================================================================
// Global Memo
// =============================================================================

// GlobalMemo wraps a Memo that is shared across ALL sessions.
// It embeds *Memo[T] so Memo methods are directly available.
//
// Global memos should only depend on global signals/memos (or other process-wide
// state). Depending on per-session signals inside a GlobalMemo would produce
// incorrect cross-session caching.
type GlobalMemo[T any] struct {
	*Memo[T]
}

// NewGlobalMemo creates a memo shared across ALL sessions.
// The returned memo is initialized immediately and persists for the lifetime
// of the application.
func NewGlobalMemo[T any](compute func() T) *GlobalMemo[T] {
	return &GlobalMemo[T]{Memo: NewMemo(compute)}
}

// WithEquals configures the memo with a custom equality function and returns
// the GlobalMemo for chaining.
func (m *GlobalMemo[T]) WithEquals(fn func(T, T) bool) *GlobalMemo[T] {
	m.Memo.WithEquals(fn)
	return m
}

