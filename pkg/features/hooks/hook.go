package hooks

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/vango-dev/vango/v2/pkg/vdom"
)

// Hook creates a hook attribute for element.
// The config map is serialized to JSON and sent to the client.
func Hook(name string, config any) vdom.Attr {
	// We serialize the config immediately to ensure it's valid JSON.
	// In a real implementation, this might be handled by the renderer,
	// but here we pack it into the attribute value.
	// Format: "HookName:{\"config\":\"values\"}"

	b, _ := json.Marshal(config)
	value := fmt.Sprintf("%s:%s", name, string(b))

	return vdom.Attr{
		Key:   "v-hook",
		Value: value,
	}
}

// OnEvent creates an event handler attribute for a hook event.
func OnEvent(name string, handler func(HookEvent)) vdom.EventHandler {
	return vdom.EventHandler{
		Event:   name,
		Handler: handler,
	}
}

// HookEvent represents an event triggered by a client hook.
type HookEvent struct {
	Name string
	Data map[string]any
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
// This is typically handled by sending a command back to the client.
func (e HookEvent) Revert() {
	// In the server-first model, reverting means invalidating the optimistic state
	// or sending a specific command.
	// For this implementation, we can't easily signal back without context.
	// We'll leave it as a placeholder or dependent on how handlers are invoked.
}
