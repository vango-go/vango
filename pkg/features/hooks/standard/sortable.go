package standard

import (
	"github.com/vango-go/vango/pkg/features/hooks"
	"github.com/vango-go/vango/pkg/vdom"
)

// SortableConfig configures the Sortable hook.
type SortableConfig struct {
	Group      string `json:"group,omitempty"`
	Animation  int    `json:"animation,omitempty"`
	GhostClass string `json:"ghostClass,omitempty"`
	Handle     string `json:"handle,omitempty"`
	Disabled   bool   `json:"disabled,omitempty"`
}

// Sortable creates a Sortable hook attribute.
func Sortable(config any) vdom.Attr {
	// We pass the config directly, relying on hooks.Hook to marshal it.
	// This supports both structs and maps.
	return hooks.Hook("Sortable", config)
}
