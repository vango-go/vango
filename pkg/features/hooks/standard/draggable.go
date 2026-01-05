package standard

import (
	"github.com/vango-go/vango/pkg/features/hooks"
	"github.com/vango-go/vango/pkg/vdom"
)

// DraggableConfig configures the Draggable hook.
type DraggableConfig struct {
	Axis   string `json:"axis,omitempty"` // "x", "y", or "both"
	Handle string `json:"handle,omitempty"`
	Revert bool   `json:"revert,omitempty"`
}

// Draggable creates a Draggable hook attribute.
func Draggable(config DraggableConfig) vdom.Attr {
	m := map[string]any{
		"axis":   config.Axis,
		"handle": config.Handle,
		"revert": config.Revert,
	}
	return hooks.Hook("Draggable", m)
}
