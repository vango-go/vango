package urlstate

import (
	"fmt"
	"time"

	"github.com/vango-dev/vango/v2/pkg/vango"
)

// URLState represents a reactive state synced with a URL parameter.
type URLState[T any] struct {
	key          string
	defaultValue T
	signal       *vango.Signal[T]
	serializer   func(T) string
	deserializer func(string) T
	debounce     time.Duration
	replace      bool

	// Internal
	lastUpdate time.Time
	timer      *time.Timer
}

// Use creates a new URLState bound to the given query parameter key.
func Use[T any](key string, defaultValue T) *URLState[T] {
	// Initialize signal with default value
	// In a real implementation, we would try to read the initial value from the URL here.
	// Since we don't have direct access to Router/Context in this isolated package implementation yet,
	// we start with defaultValue.
	// TODO: Integrate with Router/Window context to read initial value.

	u := &URLState[T]{
		key:          key,
		defaultValue: defaultValue,
		signal:       vango.NewSignal(defaultValue),
		serializer:   DefaultSerializer(defaultValue),
		deserializer: DefaultDeserializer(defaultValue),
	}

	return u
}

// Get returns the current value.
func (u *URLState[T]) Get() T {
	return u.signal.Get()
}

// Set updates the value and the URL.
func (u *URLState[T]) Set(value T) {
	u.signal.Set(value)
	u.updateURL(value)
}

// Replace updates the value and replaces the current URL history entry.
func (u *URLState[T]) Replace(value T) {
	u.replace = true
	u.Set(value)
	u.replace = false // Reset for next valid Set? Or should Replace be persistent option?
	// API spec says Replace(value) is a method.
}

// Reset resets the value to the default.
func (u *URLState[T]) Reset() {
	u.Set(u.defaultValue)
}

// IsSet returns true if the current value is different from the default.
func (u *URLState[T]) IsSet() bool {
	// Simple equality check. For slices/maps might need deeper check.
	// basic equality for now.
	return fmt.Sprintf("%v", u.Get()) != fmt.Sprintf("%v", u.defaultValue)
}

// Debounce sets the debounce duration for URL updates.
func (u *URLState[T]) Debounce(d time.Duration) *URLState[T] {
	u.debounce = d
	return u
}

// Serialize sets a custom serializer.
func (u *URLState[T]) Serialize(fn func(T) string) *URLState[T] {
	u.serializer = fn
	return u
}

// Deserialize sets a custom deserializer.
func (u *URLState[T]) Deserialize(fn func(string) T) *URLState[T] {
	u.deserializer = fn
	return u
}

// Internal update logic
func (u *URLState[T]) updateURL(value T) {
	str := u.serializer(value)

	// Debounce logic
	if u.debounce > 0 {
		if u.timer != nil {
			u.timer.Stop()
		}
		u.timer = time.AfterFunc(u.debounce, func() {
			u.performNavigation(str)
		})
		return
	}

	u.performNavigation(str)
}

func (u *URLState[T]) performNavigation(value string) {
	// TODO: integration with Router/Navigator
	// fmt.Printf("Navigating to ?%s=%s (replace=%v)\n", u.key, value, u.replace)
}
