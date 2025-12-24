package vango

// SignalOption is a functional option for configuring signals.
type SignalOption func(*signalOptions)

// signalOptions holds configuration for signal behavior.
type signalOptions struct {
	// transient signals are not persisted to session store.
	transient bool

	// persistKey is the explicit key for serialization.
	// If empty, an auto-generated key is used based on component/position.
	persistKey string
}

// Transient marks a signal as non-persistent.
// Transient signals are not saved to the session store on disconnect or server restart.
// Use this for ephemeral state like cursor positions, hover states, or temporary UI state.
//
// Example:
//
//	cursor := vango.Signal(Point{0, 0}, vango.Transient())    // Not persisted
//	formData := vango.Signal(Form{})                          // Persisted (default)
func Transient() SignalOption {
	return func(o *signalOptions) {
		o.transient = true
	}
}

// PersistKey sets an explicit key for signal serialization.
// By default, signals use an auto-generated key based on component path and position.
// Use PersistKey when you need a stable, predictable key for:
// - Migrations between code versions
// - Debugging serialized session data
// - Sharing signal state between different components
//
// Example:
//
//	userID := vango.Signal(0, vango.PersistKey("user_id"))
func PersistKey(key string) SignalOption {
	return func(o *signalOptions) {
		o.persistKey = key
	}
}

// applyOptions applies the given options and returns the resulting config.
func applyOptions(opts []SignalOption) signalOptions {
	var options signalOptions
	for _, opt := range opts {
		opt(&options)
	}
	return options
}

// PersistableSignal is an interface for signals that can be persisted.
// This is implemented by Signal[T] when created with persistence options.
type PersistableSignal interface {
	// IsTransient returns true if the signal should not be persisted.
	IsTransient() bool

	// PersistKey returns the explicit persistence key, or empty string for auto-key.
	PersistKey() string

	// GetAny returns the current value as an interface{}.
	GetAny() any

	// SetAny sets the value from an interface{}.
	// Returns an error if the type doesn't match.
	SetAny(value any) error
}
