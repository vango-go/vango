package server

import (
	"log"
	"time"

	"github.com/vango-dev/vango/v2/pkg/features/hooks"
	"github.com/vango-dev/vango/v2/pkg/protocol"
)

// Handler is the internal event handler function type.
// It receives a decoded event and processes it.
type Handler func(event *Event)

// Event represents a decoded event from the client with runtime context.
type Event struct {
	// Seq is the sequence number of the event.
	Seq uint64

	// Type is the type of event (click, input, submit, etc.).
	Type protocol.EventType

	// HID is the hydration ID of the target element.
	HID string

	// Payload contains type-specific event data.
	Payload any

	// Session is the session that received the event.
	Session *Session

	// Time is when the event was received by the server.
	Time time.Time
}

// TypeString returns the string representation of the event type.
// Used for logging and tracing.
func (e *Event) TypeString() string {
	return e.Type.String()
}

// MouseEvent represents a mouse event with position and modifiers.
type MouseEvent struct {
	ClientX  int
	ClientY  int
	Button   int
	CtrlKey  bool
	ShiftKey bool
	AltKey   bool
	MetaKey  bool
}

// KeyboardEvent represents a keyboard event with key and modifiers.
type KeyboardEvent struct {
	Key      string
	CtrlKey  bool
	ShiftKey bool
	AltKey   bool
	MetaKey  bool
}

// FormData represents submitted form data.
type FormData struct {
	values map[string]string
}

// Get returns the value for a form field.
func (f FormData) Get(key string) string {
	return f.values[key]
}

// Has returns whether a form field exists.
func (f FormData) Has(key string) bool {
	_, ok := f.values[key]
	return ok
}

// All returns all form fields.
func (f FormData) All() map[string]string {
	result := make(map[string]string, len(f.values))
	for k, v := range f.values {
		result[k] = v
	}
	return result
}

// HookEvent represents a client hook event.
type HookEvent struct {
	Name string
	Data map[string]any
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

// GetBool returns a bool value from the hook data.
func (h HookEvent) GetBool(key string) bool {
	if v, ok := h.Data[key].(bool); ok {
		return v
	}
	return false
}

// ScrollEvent represents a scroll event with position.
type ScrollEvent struct {
	ScrollTop  int
	ScrollLeft int
}

// ResizeEvent represents a resize event with dimensions.
type ResizeEvent struct {
	Width  int
	Height int
}

// TouchPoint represents a single touch point.
type TouchPoint struct {
	ID      int
	ClientX int
	ClientY int
}

// TouchEvent represents a touch event with touch points.
type TouchEvent struct {
	Touches []TouchPoint
}

// NavigateEvent represents a navigation request.
type NavigateEvent struct {
	Path    string
	Replace bool
}

// wrapHandler converts a user-provided handler to the internal Handler type.
// It supports various function signatures for different event types.
func wrapHandler(value any) Handler {
	switch h := value.(type) {
	// Simple click handler - no arguments
	case func():
		return func(e *Event) { h() }

	// Click handler with event
	case func(*Event):
		return h

	// Input/Change handler - string value
	case func(string):
		return func(e *Event) {
			if s, ok := e.Payload.(string); ok {
				h(s)
			}
		}

	// Mouse event handler
	case func(MouseEvent):
		return func(e *Event) {
			if data, ok := e.Payload.(*protocol.MouseEventData); ok {
				h(MouseEvent{
					ClientX:  data.ClientX,
					ClientY:  data.ClientY,
					Button:   int(data.Button),
					CtrlKey:  data.Modifiers.Has(protocol.ModCtrl),
					ShiftKey: data.Modifiers.Has(protocol.ModShift),
					AltKey:   data.Modifiers.Has(protocol.ModAlt),
					MetaKey:  data.Modifiers.Has(protocol.ModMeta),
				})
			}
		}

	// Keyboard event handler
	case func(KeyboardEvent):
		return func(e *Event) {
			if data, ok := e.Payload.(*protocol.KeyboardEventData); ok {
				h(KeyboardEvent{
					Key:      data.Key,
					CtrlKey:  data.Modifiers.Has(protocol.ModCtrl),
					ShiftKey: data.Modifiers.Has(protocol.ModShift),
					AltKey:   data.Modifiers.Has(protocol.ModAlt),
					MetaKey:  data.Modifiers.Has(protocol.ModMeta),
				})
			}
		}

	// Form submit handler
	case func(FormData):
		return func(e *Event) {
			if data, ok := e.Payload.(*protocol.SubmitEventData); ok {
				h(FormData{values: data.Fields})
			}
		}

	// Hook event handler (internal server type)
	case func(HookEvent):
		return func(e *Event) {
			if data, ok := e.Payload.(*protocol.HookEventData); ok {
				h(HookEvent{Name: data.Name, Data: data.Data})
			} else {
				// Debug log for type mismatch
				// fmt.Printf("[DEBUG] Hook handler payload type mismatch: %T\n", e.Payload)
			}
		}

	// Hook event handler (public hooks package type)
	case func(hooks.HookEvent):
		return func(e *Event) {
			if data, ok := e.Payload.(*protocol.HookEventData); ok {
				h(hooks.HookEvent{Name: data.Name, Data: data.Data})
			}
		}

	// Scroll event handler
	case func(ScrollEvent):
		return func(e *Event) {
			if data, ok := e.Payload.(*protocol.ScrollEventData); ok {
				h(ScrollEvent{
					ScrollTop:  data.ScrollTop,
					ScrollLeft: data.ScrollLeft,
				})
			}
		}

	// Resize event handler
	case func(ResizeEvent):
		return func(e *Event) {
			if data, ok := e.Payload.(*protocol.ResizeEventData); ok {
				h(ResizeEvent{
					Width:  data.Width,
					Height: data.Height,
				})
			}
		}

	// Touch event handler
	case func(TouchEvent):
		return func(e *Event) {
			if data, ok := e.Payload.(*protocol.TouchEventData); ok {
				touches := make([]TouchPoint, len(data.Touches))
				for i, t := range data.Touches {
					touches[i] = TouchPoint{
						ID:      t.ID,
						ClientX: t.ClientX,
						ClientY: t.ClientY,
					}
				}
				h(TouchEvent{Touches: touches})
			}
		}

	// Navigate event handler
	case func(NavigateEvent):
		return func(e *Event) {
			if data, ok := e.Payload.(*protocol.NavigateEventData); ok {
				h(NavigateEvent{
					Path:    data.Path,
					Replace: data.Replace,
				})
			}
		}

	default:
		// Unknown handler type - warn developer and return no-op handler
		log.Printf("[WARN] wrapHandler: Unrecognized handler type %T. "+
			"Handler will NOT be called. Supported types: func(), func(*Event), "+
			"func(string), func(hooks.HookEvent), func(FormData), etc.", value)
		return func(e *Event) {}
	}
}

// eventFromProtocol converts a protocol.Event to a server.Event.
func eventFromProtocol(pe *protocol.Event, session *Session) *Event {
	return &Event{
		Seq:     pe.Seq,
		Type:    pe.Type,
		HID:     pe.HID,
		Payload: pe.Payload,
		Session: session,
		Time:    time.Now(),
	}
}

// isClickLike returns true if the event type is a simple click-like event.
func isClickLike(et protocol.EventType) bool {
	switch et {
	case protocol.EventClick, protocol.EventDblClick,
		protocol.EventFocus, protocol.EventBlur,
		protocol.EventMouseEnter, protocol.EventMouseLeave:
		return true
	default:
		return false
	}
}

// isInputLike returns true if the event type has a string value payload.
func isInputLike(et protocol.EventType) bool {
	switch et {
	case protocol.EventInput, protocol.EventChange:
		return true
	default:
		return false
	}
}

// isMouseEvent returns true if the event type is a mouse event with position.
func isMouseEvent(et protocol.EventType) bool {
	switch et {
	case protocol.EventMouseDown, protocol.EventMouseUp, protocol.EventMouseMove,
		protocol.EventDragStart, protocol.EventDragEnd, protocol.EventDrop:
		return true
	default:
		return false
	}
}

// isKeyboardEvent returns true if the event type is a keyboard event.
func isKeyboardEvent(et protocol.EventType) bool {
	switch et {
	case protocol.EventKeyDown, protocol.EventKeyUp, protocol.EventKeyPress:
		return true
	default:
		return false
	}
}
