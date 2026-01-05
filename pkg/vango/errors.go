package vango

import "errors"

// =============================================================================
// Phase 16: Sentinel Errors for Structured Side Effects
// =============================================================================

// ErrBudgetExceeded is returned when a storm budget limit is exceeded.
// This happens when too many operations (Resource fetches, Action runs, etc.)
// occur within the configured time window.
//
// Applications should handle this by:
// - Logging the event for debugging
// - Optionally showing user feedback about rate limiting
// - Reducing the frequency of operations if possible
//
// See SPEC_ADDENDUM.md §A.4 for storm budget configuration.
var ErrBudgetExceeded = errors.New("vango: storm budget exceeded")

// ErrQueueFull is returned when an Action's queue is full and cannot accept
// more work items. This applies to Actions with ConcurrencyQueue policy.
//
// Applications should handle this by:
// - Informing the user their action was not queued
// - Waiting before retrying
// - Using a different concurrency policy if appropriate
//
// See SPEC_ADDENDUM.md §A.1.5 for concurrency policies.
var ErrQueueFull = errors.New("vango: action queue full")

// ErrActionRunning is returned when attempting to run an Action that is
// already in the Running state and the concurrency policy is DropWhileRunning.
//
// Applications can safely ignore this error as it's expected behavior
// for de-duplicating rapid user actions.
var ErrActionRunning = errors.New("vango: action already running")

// ErrEffectContext is returned when an effect helper (Interval, Subscribe,
// GoLatest) is called outside of an effect body or render context.
//
// These helpers require access to the runtime context (Ctx) and must be
// called within CreateEffect or during component render.
var ErrEffectContext = errors.New("vango: effect helper called outside effect/render context")

// ErrGoLatestContext is returned when GoLatest is called outside an effect body.
// GoLatest requires effect-local storage and must be called from within CreateEffect.
var ErrGoLatestContext = errors.New("vango: GoLatest must be called inside an Effect")
