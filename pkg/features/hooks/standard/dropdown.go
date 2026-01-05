package standard

import (
	"github.com/vango-go/vango/pkg/features/hooks"
	"github.com/vango-go/vango/pkg/vdom"
)

// DropdownConfig configures the Dropdown hook.
type DropdownConfig struct {
	CloseOnEscape bool `json:"closeOnEscape,omitempty"`
	CloseOnClick  bool `json:"closeOnClick,omitempty"`
}

// Dropdown creates a Dropdown hook attribute.
func Dropdown(config DropdownConfig) vdom.Attr {
	m := map[string]any{
		"closeOnEscape": config.CloseOnEscape,
		"closeOnClick":  config.CloseOnClick,
	}
	return hooks.Hook("Dropdown", m)
}
