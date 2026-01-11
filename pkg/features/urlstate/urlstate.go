package urlstate

import (
	"reflect"
	"sync"
	"time"

	"github.com/vango-go/vango/pkg/urlparam"
	"github.com/vango-go/vango/pkg/vango"
)

// URLState represents a reactive state synced with a URL parameter.
type URLState[T any] struct {
	key          string
	defaultValue T
	signal       *vango.Signal[T]
	serializer   func(T) string
	deserializer func(string) T
	debounce     time.Duration

	// Internal
	timerMu sync.Mutex
	timer   *time.Timer

	navigate func(params map[string]string, mode urlparam.URLMode)
}

// Use creates a new URLState bound to the given query parameter key.
func Use[T any](key string, defaultValue T) *URLState[T] {
	vango.TrackHook(vango.HookURLParam)

	// Hook slot stabilization (matches URLParam behavior).
	slot := vango.UseHookSlot()
	var u *URLState[T]
	first := false
	if slot != nil {
		existing, ok := slot.(*URLState[T])
		if !ok {
			panic("vango: hook slot type mismatch for URLState")
		}
		u = existing
	} else {
		first = true
		u = &URLState[T]{}
		vango.SetHookSlot(u)
	}

	if first {
		u.key = key
		u.defaultValue = defaultValue
		u.serializer = DefaultSerializer(defaultValue)
		u.deserializer = DefaultDeserializer(defaultValue)

		if navCtx := vango.GetContext(urlparam.NavigatorKey); navCtx != nil {
			if nav, ok := navCtx.(*urlparam.Navigator); ok {
				u.navigate = nav.Navigate
			}
		}
	}

	// Determine initial value from URL params (first render only).
	initial := defaultValue
	if first {
		if initialCtx := vango.GetContext(urlparam.InitialParamsKey); initialCtx != nil {
			if state, ok := initialCtx.(*urlparam.InitialURLState); ok && state.Params != nil && u.key != "" {
				if raw, ok := state.Params[u.key]; ok {
					initial = u.deserializer(raw)
				}
			}
		}
	}

	// Must be called on every render to preserve hook slot order.
	u.signal = vango.NewSignal(initial)

	return u
}

// Get returns the current value.
func (u *URLState[T]) Get() T {
	return u.signal.Get()
}

// Set updates the value and the URL.
func (u *URLState[T]) Set(value T) {
	u.signal.Set(value)
	u.updateURL(value, urlparam.ModePush)
}

// Replace updates the value and replaces the current URL history entry.
func (u *URLState[T]) Replace(value T) {
	u.signal.Set(value)
	u.updateURL(value, urlparam.ModeReplace)
}

// Reset resets the value to the default.
func (u *URLState[T]) Reset() {
	u.Set(u.defaultValue)
}

// IsSet returns true if the current value is different from the default.
func (u *URLState[T]) IsSet() bool {
	return !reflect.DeepEqual(u.Get(), u.defaultValue)
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
func (u *URLState[T]) updateURL(value T, mode urlparam.URLMode) {
	if u.key == "" {
		return
	}

	str := u.serializer(value)

	// Debounce logic
	if u.debounce > 0 {
		u.timerMu.Lock()
		defer u.timerMu.Unlock()

		if u.timer != nil {
			u.timer.Stop()
		}

		u.timer = time.AfterFunc(u.debounce, func() {
			u.performNavigation(str, mode)
		})
		return
	}

	u.performNavigation(str, mode)
}

func (u *URLState[T]) performNavigation(value string, mode urlparam.URLMode) {
	if u.navigate == nil {
		if navCtx := vango.GetContext(urlparam.NavigatorKey); navCtx != nil {
			if nav, ok := navCtx.(*urlparam.Navigator); ok {
				u.navigate = nav.Navigate
			}
		}
	}
	if u.navigate == nil {
		return
	}

	u.navigate(map[string]string{u.key: value}, mode)
}
