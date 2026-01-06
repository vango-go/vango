package vango

import "time"

// =============================================================================
// Event Modifiers
// =============================================================================
//
// Event modifiers wrap handlers to modify their behavior. They can be composed.
// Per spec section 3.9.3, lines 1299-1365.
//
// Usage:
//
//	// Prevent default browser behavior
//	OnClick(vango.PreventDefault(func() {
//	    handleClick()
//	}))
//
//	// Compose modifiers
//	OnClick(vango.PreventDefault(vango.StopPropagation(func() {
//	    handleClick()
//	})))
//
//	// Debounce input
//	OnInput(vango.Debounce(300*time.Millisecond, func(value string) {
//	    search(value)
//	}))
//
//	// Hotkey handling
//	OnKeyDown(vango.KeyWithModifiers("s", vango.Ctrl, func() {
//	    save() // Ctrl+S
//	}))

// KeyMod represents keyboard modifier flags for KeyWithModifiers.
type KeyMod uint8

const (
	// Ctrl modifier (Control key)
	Ctrl KeyMod = 0x01

	// Shift modifier
	Shift KeyMod = 0x02

	// Alt modifier (Option on Mac)
	Alt KeyMod = 0x04

	// Meta modifier (Cmd on Mac, Windows key on Windows)
	Meta KeyMod = 0x08
)

// Has returns true if the specified modifier is set.
func (k KeyMod) Has(mod KeyMod) bool {
	return k&mod != 0
}

// String returns a human-readable representation of the modifiers.
func (k KeyMod) String() string {
	var parts []string
	if k.Has(Ctrl) {
		parts = append(parts, "Ctrl")
	}
	if k.Has(Shift) {
		parts = append(parts, "Shift")
	}
	if k.Has(Alt) {
		parts = append(parts, "Alt")
	}
	if k.Has(Meta) {
		parts = append(parts, "Meta")
	}
	if len(parts) == 0 {
		return "none"
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += "+" + parts[i]
	}
	return result
}

// ModifiedHandler wraps a handler with modifier flags.
// The runtime recognizes this wrapper and applies the appropriate behavior:
// - Client-side: PreventDefault, StopPropagation, Self, Passive, Capture, Once (removal from DOM)
// - Server-side: Once (handler removal), key filtering, Debounce/Throttle timing
type ModifiedHandler struct {
	// The wrapped handler function
	Handler any

	// Client-side modifiers
	PreventDefault  bool // Prevent default browser behavior
	StopPropagation bool // Stop event bubbling
	Self            bool // Only fire if target is the exact element
	Once            bool // Remove handler after first trigger
	Passive         bool // Passive listener (cannot preventDefault)
	Capture         bool // Fire during capture phase

	// Timing modifiers (client-side implementation)
	Debounce time.Duration // Debounce delay
	Throttle time.Duration // Throttle interval

	// Key filtering (server-side implementation)
	KeyFilter    string   // Single key filter (for Hotkey)
	KeysFilter   []string // Multiple keys filter (for Keys)
	KeyModifiers KeyMod   // Required modifiers (for KeyWithModifiers)
}

// Unwrap returns the innermost handler, unwrapping any nested ModifiedHandlers.
func (m ModifiedHandler) Unwrap() any {
	if inner, ok := m.Handler.(ModifiedHandler); ok {
		return inner.Unwrap()
	}
	return m.Handler
}

// merge combines this ModifiedHandler with another, inheriting all flags.
func (m ModifiedHandler) merge(other ModifiedHandler) ModifiedHandler {
	// Copy flags from other
	if other.PreventDefault {
		m.PreventDefault = true
	}
	if other.StopPropagation {
		m.StopPropagation = true
	}
	if other.Self {
		m.Self = true
	}
	if other.Once {
		m.Once = true
	}
	if other.Passive {
		m.Passive = true
	}
	if other.Capture {
		m.Capture = true
	}
	if other.Debounce > 0 {
		m.Debounce = other.Debounce
	}
	if other.Throttle > 0 {
		m.Throttle = other.Throttle
	}
	if other.KeyFilter != "" {
		m.KeyFilter = other.KeyFilter
	}
	if len(other.KeysFilter) > 0 {
		m.KeysFilter = other.KeysFilter
	}
	if other.KeyModifiers != 0 {
		m.KeyModifiers = other.KeyModifiers
	}

	// Preserve the innermost handler
	m.Handler = other.Unwrap()

	return m
}

// PreventDefault wraps a handler to prevent the default browser behavior.
// Per spec section 3.9.3, lines 1304-1307.
//
// Example:
//
//	OnClick(vango.PreventDefault(func() {
//	    // Click handled, default prevented
//	}))
func PreventDefault(handler any) ModifiedHandler {
	if mh, ok := handler.(ModifiedHandler); ok {
		result := mh
		result.PreventDefault = true
		return result
	}
	return ModifiedHandler{Handler: handler, PreventDefault: true}
}

// StopPropagation wraps a handler to stop event bubbling.
// Per spec section 3.9.3, lines 1309-1312.
//
// Example:
//
//	OnClick(vango.StopPropagation(func() {
//	    // Click won't bubble up
//	}))
func StopPropagation(handler any) ModifiedHandler {
	if mh, ok := handler.(ModifiedHandler); ok {
		result := mh
		result.StopPropagation = true
		return result
	}
	return ModifiedHandler{Handler: handler, StopPropagation: true}
}

// Self wraps a handler to only fire if the event target is the exact element.
// Per spec section 3.9.3, lines 1319-1322.
//
// Example:
//
//	OnClick(vango.Self(func() {
//	    // Only fires if clicked element is this exact element
//	}))
func Self(handler any) ModifiedHandler {
	if mh, ok := handler.(ModifiedHandler); ok {
		result := mh
		result.Self = true
		return result
	}
	return ModifiedHandler{Handler: handler, Self: true}
}

// Once wraps a handler to remove it after the first trigger.
// Per spec section 3.9.3, lines 1324-1327.
//
// Example:
//
//	OnClick(vango.Once(func() {
//	    // Only fires once
//	}))
func Once(handler any) ModifiedHandler {
	if mh, ok := handler.(ModifiedHandler); ok {
		result := mh
		result.Once = true
		return result
	}
	return ModifiedHandler{Handler: handler, Once: true}
}

// Passive wraps a handler as a passive listener for scroll performance.
// Passive handlers cannot call preventDefault.
// Per spec section 3.9.3, lines 1329-1332.
//
// Example:
//
//	OnScroll(vango.Passive(func(e vango.ScrollEvent) {
//	    // Cannot call preventDefault
//	}))
func Passive(handler any) ModifiedHandler {
	if mh, ok := handler.(ModifiedHandler); ok {
		result := mh
		result.Passive = true
		return result
	}
	return ModifiedHandler{Handler: handler, Passive: true}
}

// Capture wraps a handler to fire during the capture phase.
// Per spec section 3.9.3, lines 1334-1337.
//
// Note: Due to Vango's event delegation architecture (all events are captured
// at the document level), per-element capture vs bubble phase control has
// limitations. Some events (focus, blur, scroll, mouseenter, mouseleave) are
// always captured, while others (click, input, keydown) bubble. This flag is
// available for documentation and potential future use.
//
// Example:
//
//	OnClick(vango.Capture(func() {
//	    // Fires during capture phase
//	}))
func Capture(handler any) ModifiedHandler {
	if mh, ok := handler.(ModifiedHandler); ok {
		result := mh
		result.Capture = true
		return result
	}
	return ModifiedHandler{Handler: handler, Capture: true}
}

// Debounce wraps a handler with debounce behavior.
// The handler will only be called after the specified duration has passed
// since the last event.
// Per spec section 3.9.3, lines 1339-1342.
//
// Example:
//
//	OnInput(vango.Debounce(300*time.Millisecond, func(value string) {
//	    search(value)
//	}))
func Debounce(duration time.Duration, handler any) ModifiedHandler {
	if mh, ok := handler.(ModifiedHandler); ok {
		result := mh
		result.Debounce = duration
		return result
	}
	return ModifiedHandler{Handler: handler, Debounce: duration}
}

// Throttle wraps a handler with throttle behavior.
// The handler will be called at most once per specified duration.
// Per spec section 3.9.3, lines 1344-1347.
//
// Example:
//
//	OnMouseMove(vango.Throttle(100*time.Millisecond, func(e vango.MouseEvent) {
//	    updatePosition(e.ClientX, e.ClientY)
//	}))
func Throttle(duration time.Duration, handler any) ModifiedHandler {
	if mh, ok := handler.(ModifiedHandler); ok {
		result := mh
		result.Throttle = duration
		return result
	}
	return ModifiedHandler{Handler: handler, Throttle: duration}
}

// Hotkey wraps a handler to only fire on a specific key.
// Per spec section 3.9.3, lines 1349-1352.
//
// Example:
//
//	OnKeyDown(vango.Hotkey("Enter", func() {
//	    submit()
//	}))
//
//	OnKeyDown(vango.Hotkey(vango.KeyEscape, func() {
//	    closeModal()
//	}))
func Hotkey(key string, handler any) ModifiedHandler {
	if mh, ok := handler.(ModifiedHandler); ok {
		result := mh
		result.KeyFilter = key
		return result
	}
	return ModifiedHandler{Handler: handler, KeyFilter: key}
}

// Keys wraps a handler to fire on any of the specified keys.
// Per spec section 3.9.3, lines 1354-1356.
//
// Example:
//
//	OnKeyDown(vango.Keys([]string{"Enter", "NumpadEnter"}, func() {
//	    submit()
//	}))
func Keys(keys []string, handler any) ModifiedHandler {
	if mh, ok := handler.(ModifiedHandler); ok {
		result := mh
		result.KeysFilter = keys
		return result
	}
	return ModifiedHandler{Handler: handler, KeysFilter: keys}
}

// KeyWithModifiers wraps a handler to fire on a key with specific modifiers.
// Per spec section 3.9.3, lines 1358-1364.
//
// Example:
//
//	OnKeyDown(vango.KeyWithModifiers("s", vango.Ctrl, func() {
//	    save() // Ctrl+S
//	}))
//
//	OnKeyDown(vango.KeyWithModifiers("s", vango.Ctrl|vango.Shift, func() {
//	    saveAs() // Ctrl+Shift+S
//	}))
func KeyWithModifiers(key string, mods KeyMod, handler any) ModifiedHandler {
	if mh, ok := handler.(ModifiedHandler); ok {
		result := mh
		result.KeyFilter = key
		result.KeyModifiers = mods
		return result
	}
	return ModifiedHandler{Handler: handler, KeyFilter: key, KeyModifiers: mods}
}
