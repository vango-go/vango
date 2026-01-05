package vango

import (
	"sync"
	"time"
)

// =============================================================================
// Phase 16: Storm Budgets (SPEC_ADDENDUM.md §A.4)
// =============================================================================

// StormBudgetTracker tracks rate limits for effect primitives.
// It provides protection against amplification bugs where effects cascade
// into more effects, potentially causing performance issues or infinite loops.
type StormBudgetTracker struct {
	// Configuration
	maxResourceStarts int
	maxActionStarts   int
	maxGoLatestStarts int
	maxEffectRuns     int
	windowDuration    time.Duration
	onExceeded        BudgetExceededMode

	// Windows for per-second limits
	resourceWindow *slidingWindow
	actionWindow   *slidingWindow
	goLatestWindow *slidingWindow

	// Per-tick counter for effect runs
	effectRunsThisTick int

	mu sync.Mutex
}

// BudgetExceededMode determines behavior when a storm budget is exceeded.
// (Re-exported from server/config.go for convenience)
type BudgetExceededMode int

const (
	// BudgetModeThrottle drops excess operations silently (default).
	BudgetModeThrottle BudgetExceededMode = iota

	// BudgetModeTripBreaker pauses effect execution until cleared.
	BudgetModeTripBreaker
)

// slidingWindow tracks events within a time window for rate limiting.
type slidingWindow struct {
	events     []time.Time
	windowSize time.Duration
	maxEvents  int
	mu         sync.Mutex
}

func newSlidingWindow(windowSize time.Duration, maxEvents int) *slidingWindow {
	return &slidingWindow{
		windowSize: windowSize,
		maxEvents:  maxEvents,
	}
}

// tryAdd attempts to add an event to the window.
// Returns true if allowed (under limit), false if rate limited.
func (w *slidingWindow) tryAdd() bool {
	if w.maxEvents == 0 {
		return true // No limit
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-w.windowSize)

	// Remove old events outside the window
	validIdx := 0
	for _, t := range w.events {
		if t.After(cutoff) {
			w.events[validIdx] = t
			validIdx++
		}
	}
	w.events = w.events[:validIdx]

	// Check if under limit
	if len(w.events) >= w.maxEvents {
		return false
	}

	// Add new event
	w.events = append(w.events, now)
	return true
}

// count returns the current number of events in the window.
func (w *slidingWindow) count() int {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-w.windowSize)

	count := 0
	for _, t := range w.events {
		if t.After(cutoff) {
			count++
		}
	}
	return count
}

// NewStormBudgetTracker creates a new storm budget tracker with the given configuration.
func NewStormBudgetTracker(cfg *StormBudgetConfig) *StormBudgetTracker {
	if cfg == nil {
		return nil
	}

	windowDuration := cfg.WindowDuration
	if windowDuration == 0 {
		windowDuration = time.Second
	}

	return &StormBudgetTracker{
		maxResourceStarts: cfg.MaxResourceStartsPerSecond,
		maxActionStarts:   cfg.MaxActionStartsPerSecond,
		maxGoLatestStarts: cfg.MaxGoLatestStartsPerSecond,
		maxEffectRuns:     cfg.MaxEffectRunsPerTick,
		windowDuration:    windowDuration,
		onExceeded:        cfg.OnExceeded,
		resourceWindow:    newSlidingWindow(windowDuration, cfg.MaxResourceStartsPerSecond),
		actionWindow:      newSlidingWindow(windowDuration, cfg.MaxActionStartsPerSecond),
		goLatestWindow:    newSlidingWindow(windowDuration, cfg.MaxGoLatestStartsPerSecond),
	}
}

// StormBudgetConfig holds configuration for storm budgets.
// This mirrors the server/config.StormBudgetConfig for use in the vango package.
type StormBudgetConfig struct {
	MaxResourceStartsPerSecond int
	MaxActionStartsPerSecond   int
	MaxGoLatestStartsPerSecond int
	MaxEffectRunsPerTick       int
	WindowDuration             time.Duration
	OnExceeded                 BudgetExceededMode
}

// CheckResource checks if a Resource fetch can start.
// Returns nil if allowed, ErrBudgetExceeded if rate limited.
func (t *StormBudgetTracker) CheckResource() error {
	if t == nil || t.maxResourceStarts == 0 {
		return nil
	}

	if !t.resourceWindow.tryAdd() {
		if Debug.LogStormBudget {
			println("Storm budget exceeded: Resource starts")
		}
		return ErrBudgetExceeded
	}
	return nil
}

// CheckAction checks if an Action can start.
// Returns nil if allowed, ErrBudgetExceeded if rate limited.
func (t *StormBudgetTracker) CheckAction() error {
	if t == nil || t.maxActionStarts == 0 {
		return nil
	}

	if !t.actionWindow.tryAdd() {
		if Debug.LogStormBudget {
			println("Storm budget exceeded: Action starts")
		}
		return ErrBudgetExceeded
	}
	return nil
}

// CheckGoLatest checks if GoLatest work can start.
// Returns nil if allowed, ErrBudgetExceeded if rate limited.
func (t *StormBudgetTracker) CheckGoLatest() error {
	if t == nil || t.maxGoLatestStarts == 0 {
		return nil
	}

	if !t.goLatestWindow.tryAdd() {
		if Debug.LogStormBudget {
			println("Storm budget exceeded: GoLatest starts")
		}
		return ErrBudgetExceeded
	}
	return nil
}

// CheckEffectRun checks if another effect can run this tick.
// Returns nil if allowed, ErrBudgetExceeded if limit reached.
func (t *StormBudgetTracker) CheckEffectRun() error {
	if t == nil || t.maxEffectRuns == 0 {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.effectRunsThisTick >= t.maxEffectRuns {
		if Debug.LogStormBudget {
			println("Storm budget exceeded: Effect runs per tick")
		}
		return ErrBudgetExceeded
	}

	t.effectRunsThisTick++
	return nil
}

// ResetTick resets the per-tick counters.
// Should be called at the start of each event/dispatch processing.
func (t *StormBudgetTracker) ResetTick() {
	if t == nil {
		return
	}

	t.mu.Lock()
	t.effectRunsThisTick = 0
	t.mu.Unlock()
}

// GetOnExceeded returns the configured behavior when budget is exceeded.
func (t *StormBudgetTracker) GetOnExceeded() BudgetExceededMode {
	if t == nil {
		return BudgetModeThrottle
	}
	return t.onExceeded
}

// Stats returns current budget usage statistics.
type BudgetStats struct {
	ResourceStartsInWindow int
	ActionStartsInWindow   int
	GoLatestStartsInWindow int
	EffectRunsThisTick     int
}

func (t *StormBudgetTracker) Stats() BudgetStats {
	if t == nil {
		return BudgetStats{}
	}

	t.mu.Lock()
	effectRuns := t.effectRunsThisTick
	t.mu.Unlock()

	return BudgetStats{
		ResourceStartsInWindow: t.resourceWindow.count(),
		ActionStartsInWindow:   t.actionWindow.count(),
		GoLatestStartsInWindow: t.goLatestWindow.count(),
		EffectRunsThisTick:     effectRuns,
	}
}

// StormBudgetChecker is the interface exposed to primitives for budget checking.
// This is implemented by Session and exposed via Ctx.
type StormBudgetChecker interface {
	CheckResource() error
	CheckAction() error
	CheckGoLatest() error
	CheckEffectRun() error
	ResetTick()
}
