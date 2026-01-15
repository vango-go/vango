package hooks

import (
	"github.com/vango-go/vango/pkg/render"
	corevango "github.com/vango-go/vango/pkg/vango"
	"github.com/vango-go/vango/pkg/vdom"
)

// HookEvent represents an event triggered by a client hook.
// It is the canonical type used by the public API: `func(vango.HookEvent)`.
type HookEvent = corevango.HookEvent

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
