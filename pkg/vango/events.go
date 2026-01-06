package vango

// =============================================================================
// Event Types - Public API
// =============================================================================
//
// These event types are the public API for handling DOM events in Vango.
// They match the spec section 3.9.3 and provide full access to event data.
//
// Usage:
//
//	OnClick(func(e vango.MouseEvent) {
//	    fmt.Println(e.ClientX, e.ClientY)
//	})
//
//	OnKeyDown(func(e vango.KeyboardEvent) {
//	    if e.Key == vango.KeyEnter {
//	        submit()
//	    }
//	})

// MouseEvent represents a mouse event with position and modifiers.
// Per spec section 3.9.3, lines 1059-1075.
type MouseEvent struct {
	// Position relative to viewport
	ClientX int
	ClientY int

	// Position relative to document
	PageX int
	PageY int

	// Position relative to target element
	OffsetX int
	OffsetY int

	// Button that triggered the event (0=left, 1=middle, 2=right)
	Button int

	// Bitmask of currently pressed buttons
	Buttons int

	// Modifier keys
	CtrlKey  bool
	ShiftKey bool
	AltKey   bool
	MetaKey  bool
}

// KeyboardEvent represents a keyboard event with key and modifiers.
// Per spec section 3.9.3, lines 1091-1103.
type KeyboardEvent struct {
	// The key value (e.g., "Enter", "a", "Escape")
	Key string

	// The physical key code (e.g., "Enter", "KeyA", "Escape")
	Code string

	// Modifier keys
	CtrlKey  bool
	ShiftKey bool
	AltKey   bool
	MetaKey  bool

	// True if key is being held down (auto-repeat)
	Repeat bool

	// Key location: 0=standard, 1=left, 2=right, 3=numpad
	Location int
}

// InputEvent represents an input field change event.
// Per spec section 3.9.3, lines 1131-1133.
type InputEvent struct {
	// Current value of the input
	Value string

	// Type of input change (e.g., "insertText", "deleteContentBackward")
	InputType string

	// Data being inserted (if any)
	Data string
}

// WheelEvent represents a mouse wheel event.
// Per spec section 3.9.3, line 1056.
type WheelEvent struct {
	// Scroll amounts
	DeltaX float64
	DeltaY float64
	DeltaZ float64

	// Delta mode: 0=pixels, 1=lines, 2=pages
	DeltaMode int

	// Position relative to viewport
	ClientX int
	ClientY int

	// Modifier keys
	CtrlKey  bool
	ShiftKey bool
	AltKey   bool
	MetaKey  bool
}

// DragEvent represents a drag-and-drop event.
// Per spec section 3.9.3, lines 1181-1200.
type DragEvent struct {
	// Position relative to viewport
	ClientX int
	ClientY int

	// Modifier keys
	CtrlKey  bool
	ShiftKey bool
	AltKey   bool
	MetaKey  bool

	// Internal data transfer storage
	dataTransfer map[string]string
}

// SetData sets data for the drag operation.
func (d *DragEvent) SetData(format, data string) {
	if d.dataTransfer == nil {
		d.dataTransfer = make(map[string]string)
	}
	d.dataTransfer[format] = data
}

// GetData gets data from the drag operation.
func (d *DragEvent) GetData(format string) string {
	if d.dataTransfer == nil {
		return ""
	}
	return d.dataTransfer[format]
}

// HasData returns true if the format exists in the data transfer.
func (d *DragEvent) HasData(format string) bool {
	if d.dataTransfer == nil {
		return false
	}
	_, ok := d.dataTransfer[format]
	return ok
}

// DropEvent is an alias for DragEvent used on drop targets.
// Per spec section 3.9.3, lines 1197-1199.
type DropEvent = DragEvent

// Touch represents a single touch point.
// Per spec section 3.9.3, lines 1219-1225.
type Touch struct {
	// Unique identifier for the touch point
	Identifier int

	// Position relative to viewport
	ClientX int
	ClientY int

	// Position relative to document
	PageX int
	PageY int
}

// TouchEvent represents a touch event.
// Per spec section 3.9.3, lines 1212-1217.
type TouchEvent struct {
	// All current touches on the screen
	Touches []Touch

	// Touches that started on this element
	TargetTouches []Touch

	// Touches that changed in this event
	ChangedTouches []Touch
}

// AnimationEvent represents a CSS animation event.
// Per spec section 3.9.3, lines 1230-1234.
type AnimationEvent struct {
	// Name of the CSS animation
	AnimationName string

	// Time in seconds since the animation started
	ElapsedTime float64

	// Pseudo-element the animation runs on (e.g., "::before")
	PseudoElement string
}

// TransitionEvent represents a CSS transition event.
// Per spec section 3.9.3, lines 1236-1239.
type TransitionEvent struct {
	// Name of the CSS property being transitioned
	PropertyName string

	// Time in seconds since the transition started
	ElapsedTime float64

	// Pseudo-element the transition runs on
	PseudoElement string
}

// ScrollEvent represents a scroll event.
// Per spec section 3.9.3, lines 1269-1271.
type ScrollEvent struct {
	// Scroll position from top
	ScrollTop int

	// Scroll position from left
	ScrollLeft int
}

// ResizeEvent represents a resize event.
// Per spec section 3.9.3, line 1291.
type ResizeEvent struct {
	// New width in pixels
	Width int

	// New height in pixels
	Height int
}

// FormData represents submitted form data.
// Per spec section 3.9.3, lines 1169-1179.
type FormData struct {
	values map[string][]string
}

// NewFormData creates a new FormData from a map of values.
func NewFormData(values map[string][]string) FormData {
	if values == nil {
		values = make(map[string][]string)
	}
	return FormData{values: values}
}

// NewFormDataFromSingle creates FormData from a single-value map.
// This is for backward compatibility with the old map[string]string format.
func NewFormDataFromSingle(values map[string]string) FormData {
	multi := make(map[string][]string, len(values))
	for k, v := range values {
		multi[k] = []string{v}
	}
	return FormData{values: multi}
}

// Get returns the first value for a form field.
func (f FormData) Get(key string) string {
	if vals, ok := f.values[key]; ok && len(vals) > 0 {
		return vals[0]
	}
	return ""
}

// GetAll returns all values for a form field.
func (f FormData) GetAll(key string) []string {
	if vals, ok := f.values[key]; ok {
		// Return a copy to prevent mutation
		result := make([]string, len(vals))
		copy(result, vals)
		return result
	}
	return nil
}

// Has returns whether a form field exists.
func (f FormData) Has(key string) bool {
	_, ok := f.values[key]
	return ok
}

// Keys returns all form field names.
func (f FormData) Keys() []string {
	keys := make([]string, 0, len(f.values))
	for k := range f.values {
		keys = append(keys, k)
	}
	return keys
}

// All returns all form fields as a map.
// For fields with multiple values, only the first value is returned.
func (f FormData) All() map[string]string {
	result := make(map[string]string, len(f.values))
	for k, vals := range f.values {
		if len(vals) > 0 {
			result[k] = vals[0]
		}
	}
	return result
}

// HookEvent represents a client hook event.
// Per spec section 3.9.3, line 1018.
type HookEvent struct {
	// Name of the hook event
	Name string

	// Arbitrary data from the client
	Data map[string]any

	// Internal: HID of the element
	hid string

	// Internal: dispatch function for Revert
	dispatch func(name string, payload any)
}

// Get returns a value from the hook data.
func (h HookEvent) Get(key string) any {
	return h.Data[key]
}

// GetString returns a string value from the hook data.
func (h HookEvent) GetString(key string) string {
	if v, ok := h.Data[key].(string); ok {
		return v
	}
	return ""
}

// GetInt returns an int value from the hook data.
func (h HookEvent) GetInt(key string) int {
	switch v := h.Data[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

// GetFloat returns a float64 value from the hook data.
func (h HookEvent) GetFloat(key string) float64 {
	switch v := h.Data[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}

// GetBool returns a bool value from the hook data.
func (h HookEvent) GetBool(key string) bool {
	if v, ok := h.Data[key].(bool); ok {
		return v
	}
	return false
}

// GetStrings returns a []string value from the hook data.
func (h HookEvent) GetStrings(key string) []string {
	if v, ok := h.Data[key].([]any); ok {
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

// Revert sends a revert signal to the client hook.
// Used for optimistic update rollback.
func (h HookEvent) Revert() {
	if h.dispatch != nil {
		h.dispatch("revert", nil)
	}
}

// SetContext sets the internal context for the hook event.
// This is called by the runtime when dispatching hook events.
func (h *HookEvent) SetContext(hid string, dispatch func(name string, payload any)) {
	h.hid = hid
	h.dispatch = dispatch
}

// NavigateEvent represents a navigation request event.
type NavigateEvent struct {
	// The path being navigated to
	Path string

	// Whether this is a replace (vs push) navigation
	Replace bool
}
