package vango

// =============================================================================
// Development Mode (Phase 7: Prefetch)
// =============================================================================

// DevMode enables development-time checks and panics for invalid operations.
// When true:
//   - Signal writes in prefetch mode panic
//   - Effect/Interval/Timeout in prefetch mode panic
//   - More verbose logging and error messages
//
// When false (production):
//   - Signal writes in prefetch mode are silently dropped
//   - Effect/Interval/Timeout in prefetch mode are no-ops
//   - Minimal overhead
//
// Set this at application startup:
//
//	func main() {
//	    vango.DevMode = os.Getenv("VANGO_DEV") == "1"
//	    // ...
//	}
var DevMode = false

// =============================================================================
// Prefetch Mode Detection (Phase 7: Routing, Section 8.3.2)
// =============================================================================

// PrefetchModeChecker is implemented by contexts that support prefetch mode.
// Used by primitives to check if side effects should be suppressed.
type PrefetchModeChecker interface {
	// Mode returns the current render mode.
	// 0 = normal, 1 = prefetch
	Mode() int
}

// IsPrefetchMode returns true if the current context is in prefetch mode.
// This is used by Signal.Set(), Effect(), Interval(), Timeout(), and Action()
// to enforce read-only behavior during prefetch.
//
// Per Section 8.3.2 of the Routing Spec:
//   - Dev mode: Operations panic with descriptive message
//   - Prod mode: Operations are silently dropped (no-op)
func IsPrefetchMode() bool {
	ctx := getCurrentCtx()
	if ctx == nil {
		return false
	}
	if checker, ok := ctx.(PrefetchModeChecker); ok {
		return checker.Mode() == 1 // 1 = ModePrefetch
	}
	return false
}

// checkPrefetchWrite checks if a write operation is allowed.
// Returns true if the write should proceed, false if it should be dropped.
// Panics in dev mode if in prefetch mode.
func checkPrefetchWrite(operation string) bool {
	if !IsPrefetchMode() {
		return true // Write allowed
	}
	if DevMode {
		panic("vango: " + operation + " is forbidden in prefetch mode")
	}
	// Production: silently drop
	return false
}

// checkPrefetchSideEffect checks if a side effect operation is allowed.
// Returns true if the operation should proceed, false if it should be a no-op.
// Panics in dev mode if in prefetch mode.
func checkPrefetchSideEffect(operation string) bool {
	if !IsPrefetchMode() {
		return true // Operation allowed
	}
	if DevMode {
		panic("vango: " + operation + " is forbidden in prefetch mode")
	}
	// Production: no-op
	return false
}

// =============================================================================
// Phase 16: Configuration Types for Structured Side Effects
// =============================================================================

// StrictEffectMode controls how effect-time signal writes are handled.
// This helps catch bugs where effects modify signals during their synchronous
// body, which can cause unexpected re-renders and cascading effects.
//
// See SPEC_ADDENDUM.md §A.3 for effect enforcement details.
type StrictEffectMode int

const (
	// StrictEffectOff disables effect-time write detection.
	// No warnings or errors for signal writes during effects.
	StrictEffectOff StrictEffectMode = iota

	// StrictEffectWarn logs a warning when an effect writes to a signal
	// without the AllowWrites() option. This is the recommended mode for
	// development to catch bugs without breaking existing code.
	StrictEffectWarn

	// StrictEffectPanic panics when an effect writes to a signal without
	// the AllowWrites() option. Use this mode to strictly enforce the rule
	// during testing or in strict development environments.
	StrictEffectPanic
)

// EffectStrictMode controls global effect-time write detection.
// Set this in your application initialization based on build mode.
//
// Example:
//
//	func init() {
//	    if os.Getenv("VANGO_DEV") == "1" {
//	        vango.EffectStrictMode = vango.StrictEffectWarn
//	    }
//	}
var EffectStrictMode = StrictEffectOff

// DebugConfig controls debugging features for development.
// These settings affect logging and error messages.
type DebugConfig struct {
	// IncludeSourceLocations includes file:line in debug messages.
	// Useful for tracing signal/effect creation locations.
	// Default: false (for performance).
	IncludeSourceLocations bool

	// LogRawKeys logs signal persist keys and internal identifiers.
	// Useful for debugging state persistence issues.
	// Default: false.
	LogRawKeys bool

	// LogEffectRuns logs each effect run with timing information.
	// Useful for debugging performance issues.
	// Default: false.
	LogEffectRuns bool

	// LogStormBudget logs when storm budgets are checked or exceeded.
	// Useful for tuning budget limits.
	// Default: false.
	LogStormBudget bool
}

// DefaultDebugConfig returns a DebugConfig with all debugging disabled.
// Enable individual options as needed for development.
func DefaultDebugConfig() DebugConfig {
	return DebugConfig{
		IncludeSourceLocations: false,
		LogRawKeys:             false,
		LogEffectRuns:          false,
		LogStormBudget:         false,
	}
}

// Debug is the global debug configuration.
// Modify this at application startup to enable debugging features.
var Debug = DefaultDebugConfig()

// Note: TxNamed is defined in batch.go and wraps function execution with
// a transaction name for observability. See SPEC_ADDENDUM.md §A.5 for
// transaction naming conventions.
