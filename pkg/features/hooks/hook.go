package hooks

import (
	"fmt"
	"strconv"

	"github.com/vango-dev/vango/v2/pkg/render"
	"github.com/vango-dev/vango/v2/pkg/vdom"
)

// Hook creates a hook attribute for an element.
// The config is passed to the client-side hook on mount.
// Config can be a struct, map, or any JSON-serializable value.
func Hook(name string, config any) vdom.Attr {
	return vdom.Attr{
		Key: "_hook",
		Value: render.HookConfig{
			Name:   name,
			Config: config,
		},
	}
}

// OnEvent creates an event handler attribute for a hook event.
// The handler is wrapped to filter by the specific event name,
// allowing multiple hook event handlers on the same element.
func OnEvent(name string, handler func(HookEvent)) vdom.Attr {
	// Wrap handler to filter by event name
	wrapped := func(e HookEvent) {
		if e.Name == name {
			handler(e)
		}
	}
	return vdom.Attr{
		Key:   "onhook",
		Value: wrapped,
	}
}

// HookEvent represents an event triggered by a client hook.
type HookEvent struct {
	Name string
	Data map[string]any

	// Internal fields for Revert() support - set by session before invoking handler
	hid  string
	emit func(name string, data any)
}

// SetContext injects the HID and emit function for Revert() support.
// This is called internally by the session before invoking the handler.
func (e *HookEvent) SetContext(hid string, emit func(string, any)) {
	e.hid = hid
	e.emit = emit
}

// Accessors

func (e HookEvent) String(key string) string {
	if v, ok := e.Data[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func (e HookEvent) Int(key string) int {
	if v, ok := e.Data[key]; ok {
		switch val := v.(type) {
		case int:
			return val
		case float64:
			return int(val)
		case string:
			i, _ := strconv.Atoi(val)
			return i
		}
	}
	return 0
}

func (e HookEvent) Float(key string) float64 {
	if v, ok := e.Data[key]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case int:
			return float64(val)
		case string:
			f, _ := strconv.ParseFloat(val, 64)
			return f
		}
	}
	return 0.0
}

func (e HookEvent) Bool(key string) bool {
	if v, ok := e.Data[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
		s := fmt.Sprintf("%v", v)
		b, _ := strconv.ParseBool(s)
		return b
	}
	return false
}

func (e HookEvent) Strings(key string) []string {
	if v, ok := e.Data[key]; ok {
		// Handle []interface{} from JSON
		if list, ok := v.([]any); ok {
			strs := make([]string, len(list))
			for i, item := range list {
				strs[i] = fmt.Sprintf("%v", item)
			}
			return strs
		}
		if list, ok := v.([]string); ok {
			return list
		}
	}
	return nil
}

func (e HookEvent) Raw(key string) any {
	return e.Data[key]
}

// Revert requests the client to revert the optimistic change.
// This sends a "vango:hook-revert" event to the client with the target HID.
// The client's HookManager listens for this event and calls the revert callback
// that was provided when the hook event was sent.
//
// This requires the hook to have registered a revert callback via pushEvent.
// Standard hooks (Sortable, Draggable) provide revert callbacks automatically.
func (e HookEvent) Revert() {
	if e.emit != nil {
		e.emit("vango:hook-revert", map[string]any{
			"hid": e.hid,
		})
	}
}
