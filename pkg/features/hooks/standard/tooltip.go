package standard

import (
	"github.com/vango-go/vango/pkg/features/hooks"
	"github.com/vango-go/vango/pkg/vdom"
)

// TooltipConfig configures the Tooltip hook.
type TooltipConfig struct {
	Content   string `json:"content"`
	Placement string `json:"placement,omitempty"` // top, bottom, left, right
	Delay     int    `json:"delay,omitempty"`     // ms
	Trigger   string `json:"trigger,omitempty"`   // hover, click, focus
}

// Tooltip creates a Tooltip hook attribute.
func Tooltip(config TooltipConfig) vdom.Attr {
	m := map[string]any{
		"content":   config.Content,
		"placement": config.Placement,
		"delay":     config.Delay,
		"trigger":   config.Trigger,
	}
	return hooks.Hook("Tooltip", m)
}
