//go:build vangoui

package ui

import (
	"github.com/vango-dev/vango/v2/pkg/vdom"
)

// SwitchOption configures a Switch component.
type SwitchOption func(*switchConfig)

type switchConfig struct {
	name      string
	checked   bool
	disabled  bool
	className string
	onChange  func(bool)
	label     string
}

// SwitchName sets the switch name.
func SwitchName(name string) SwitchOption {
	return func(c *switchConfig) {
		c.name = name
	}
}

// SwitchChecked sets the checked state.
func SwitchChecked(checked bool) SwitchOption {
	return func(c *switchConfig) {
		c.checked = checked
	}
}

// SwitchDisabled sets the disabled state.
func SwitchDisabled(disabled bool) SwitchOption {
	return func(c *switchConfig) {
		c.disabled = disabled
	}
}

// SwitchClass adds additional CSS classes.
func SwitchClass(className string) SwitchOption {
	return func(c *switchConfig) {
		c.className = className
	}
}

// SwitchOnChange sets the change event handler.
func SwitchOnChange(handler func(bool)) SwitchOption {
	return func(c *switchConfig) {
		c.onChange = handler
	}
}

// SwitchLabel sets a label for the switch.
func SwitchLabel(label string) SwitchOption {
	return func(c *switchConfig) {
		c.label = label
	}
}

// Switch renders a toggle switch component.
func Switch(opts ...SwitchOption) *VNode {
	cfg := switchConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	// Track container classes
	containerClasses := "peer inline-flex h-6 w-11 shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:cursor-not-allowed disabled:opacity-50"

	if cfg.checked {
		containerClasses = CN(containerClasses, "bg-primary")
	} else {
		containerClasses = CN(containerClasses, "bg-input")
	}

	containerClasses = CN(containerClasses, cfg.className)

	// Thumb classes
	thumbClasses := "pointer-events-none block h-5 w-5 rounded-full bg-background shadow-lg ring-0 transition-transform"
	if cfg.checked {
		thumbClasses = CN(thumbClasses, "translate-x-5")
	} else {
		thumbClasses = CN(thumbClasses, "translate-x-0")
	}

	containerAttrs := []any{
		Role("switch"),
		Class(containerClasses),
		AriaLabel(cfg.label),
	}

	if cfg.checked {
		containerAttrs = append(containerAttrs, AriaPressed("true"))
		containerAttrs = append(containerAttrs, Data("state", "checked"))
	} else {
		containerAttrs = append(containerAttrs, AriaPressed("false"))
		containerAttrs = append(containerAttrs, Data("state", "unchecked"))
	}

	if cfg.disabled {
		containerAttrs = append(containerAttrs, AriaDisabled(true))
	}

	if cfg.onChange != nil {
		containerAttrs = append(containerAttrs, OnClick(func() {
			cfg.onChange(!cfg.checked)
		}))
	}

	// Hidden input for form submission
	hiddenInput := El("input",
		Type("hidden"),
		Name(cfg.name),
		Value(func() string {
			if cfg.checked {
				return "on"
			}
			return ""
		}()),
	)

	thumb := El("span", Class(thumbClasses))

	containerAttrs = append(containerAttrs, thumb)

	switchEl := El("button", containerAttrs...)

	// If there's a label, wrap in a label element
	if cfg.label != "" {
		return Div(
			Class("flex items-center space-x-2"),
			hiddenInput,
			switchEl,
			El("label",
				Class("text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"),
				Text(cfg.label),
			),
		)
	}

	return Fragment(hiddenInput, switchEl)
}
