package standard

import (
	"github.com/vango-dev/vango/v2/pkg/features/hooks"
	"github.com/vango-dev/vango/v2/pkg/vdom"
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
func Sortable(config SortableConfig) vdom.Attr {
	// Convert struct to map for Hook function
	// Manual conversion or just JSON marshaling inside Hook?
	// Hook takes map[string]any.
	// We can convert struct to map.
	// OR modify Hook to accept 'any' and marshal it.
	// Since Hook is in hooks package, changing signiture is possible but breaks spec?
	// Spec said map[string]any.
	// Let's convert struct to map manually or via generic helper.

	m := map[string]any{
		"group":      config.Group,
		"animation":  config.Animation,
		"ghostClass": config.GhostClass,
		"handle":     config.Handle,
		"disabled":   config.Disabled,
	}
	return hooks.Hook("Sortable", m)
}
