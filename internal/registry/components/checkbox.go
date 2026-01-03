//go:build vangoui

package ui

import (
	"github.com/vango-dev/vango/v2/pkg/vdom"
)

// CheckboxOption configures a Checkbox component.
type CheckboxOption func(*checkboxConfig)

type checkboxConfig struct {
	name      string
	checked   bool
	disabled  bool
	className string
	onChange  func(bool)
	label     string
}

// CheckboxName sets the checkbox name.
func CheckboxName(name string) CheckboxOption {
	return func(c *checkboxConfig) {
		c.name = name
	}
}

// CheckboxChecked sets the checked state.
func CheckboxChecked(checked bool) CheckboxOption {
	return func(c *checkboxConfig) {
		c.checked = checked
	}
}

// CheckboxDisabled sets the disabled state.
func CheckboxDisabled(disabled bool) CheckboxOption {
	return func(c *checkboxConfig) {
		c.disabled = disabled
	}
}

// CheckboxClass adds additional CSS classes.
func CheckboxClass(className string) CheckboxOption {
	return func(c *checkboxConfig) {
		c.className = className
	}
}

// CheckboxOnChange sets the change event handler.
func CheckboxOnChange(handler func(bool)) CheckboxOption {
	return func(c *checkboxConfig) {
		c.onChange = handler
	}
}

// CheckboxLabel sets a label for the checkbox.
func CheckboxLabel(label string) CheckboxOption {
	return func(c *checkboxConfig) {
		c.label = label
	}
}

// Checkbox renders a checkbox input.
func Checkbox(opts ...CheckboxOption) *VNode {
	cfg := checkboxConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	classes := CN(
		"peer h-4 w-4 shrink-0 rounded-sm border border-primary ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 data-[state=checked]:bg-primary data-[state=checked]:text-primary-foreground",
		cfg.className,
	)

	attrs := []any{
		Type("checkbox"),
		Class(classes),
	}

	if cfg.name != "" {
		attrs = append(attrs, Name(cfg.name))
	}
	if cfg.checked {
		attrs = append(attrs, Checked())
		attrs = append(attrs, Data("state", "checked"))
	} else {
		attrs = append(attrs, Data("state", "unchecked"))
	}
	if cfg.disabled {
		attrs = append(attrs, Disabled())
	}
	if cfg.onChange != nil {
		attrs = append(attrs, OnChange(cfg.onChange))
	}

	input := El("input", attrs...)

	// If there's a label, wrap in a label element
	if cfg.label != "" {
		return El("label",
			Class("flex items-center space-x-2"),
			input,
			El("span",
				Class("text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"),
				Text(cfg.label),
			),
		)
	}

	return input
}
