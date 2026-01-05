package vango

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
