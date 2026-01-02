package ui

import (
	. "github.com/vango-dev/vango/v2/pkg/vdom"
)

// SelectOption configures a Select component.
type SelectOption func(*selectConfig)

type selectConfig struct {
	name        string
	value       string
	placeholder string
	disabled    bool
	required    bool
	className   string
	options     []SelectItem
	onChange    func(string)
}

// SelectItem represents an option in the select.
type SelectItem struct {
	Value    string
	Label    string
	Disabled bool
	Group    string
}

// SelectName sets the select name.
func SelectName(name string) SelectOption {
	return func(c *selectConfig) {
		c.name = name
	}
}

// SelectValue sets the selected value.
func SelectValue(value string) SelectOption {
	return func(c *selectConfig) {
		c.value = value
	}
}

// SelectPlaceholder sets the placeholder text.
func SelectPlaceholder(placeholder string) SelectOption {
	return func(c *selectConfig) {
		c.placeholder = placeholder
	}
}

// SelectDisabled sets the disabled state.
func SelectDisabled(disabled bool) SelectOption {
	return func(c *selectConfig) {
		c.disabled = disabled
	}
}

// SelectRequired sets the required state.
func SelectRequired(required bool) SelectOption {
	return func(c *selectConfig) {
		c.required = required
	}
}

// SelectClass adds additional CSS classes.
func SelectClass(className string) SelectOption {
	return func(c *selectConfig) {
		c.className = className
	}
}

// SelectOptions sets the select options.
func SelectOptions(options ...SelectItem) SelectOption {
	return func(c *selectConfig) {
		c.options = options
	}
}

// SelectOnChange sets the change event handler.
func SelectOnChange(handler func(string)) SelectOption {
	return func(c *selectConfig) {
		c.onChange = handler
	}
}

// Select renders a native select element.
func Select(opts ...SelectOption) *VNode {
	cfg := selectConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	classes := CN(
		"flex h-10 w-full items-center justify-between rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50",
		cfg.className,
	)

	attrs := []any{
		Class(classes),
	}

	if cfg.name != "" {
		attrs = append(attrs, Name(cfg.name))
	}
	if cfg.disabled {
		attrs = append(attrs, Disabled())
	}
	if cfg.required {
		attrs = append(attrs, Required())
	}
	if cfg.onChange != nil {
		attrs = append(attrs, OnChange(cfg.onChange))
	}

	// Add placeholder option
	if cfg.placeholder != "" {
		placeholderAttrs := []any{
			Value(""),
			Disabled(),
		}
		if cfg.value == "" {
			placeholderAttrs = append(placeholderAttrs, Selected())
		}
		placeholderAttrs = append(placeholderAttrs, Text(cfg.placeholder))
		attrs = append(attrs, El("option", placeholderAttrs...))
	}

	// Group options by group name
	groupedOptions := make(map[string][]SelectItem)
	var ungrouped []SelectItem
	for _, opt := range cfg.options {
		if opt.Group != "" {
			groupedOptions[opt.Group] = append(groupedOptions[opt.Group], opt)
		} else {
			ungrouped = append(ungrouped, opt)
		}
	}

	// Add ungrouped options
	for _, opt := range ungrouped {
		optAttrs := []any{
			Value(opt.Value),
		}
		if opt.Disabled {
			optAttrs = append(optAttrs, Disabled())
		}
		if opt.Value == cfg.value {
			optAttrs = append(optAttrs, Selected())
		}
		optAttrs = append(optAttrs, Text(opt.Label))
		attrs = append(attrs, El("option", optAttrs...))
	}

	// Add grouped options
	for group, opts := range groupedOptions {
		groupAttrs := []any{
			Attr{Key: "label", Value: group},
		}
		for _, opt := range opts {
			optAttrs := []any{
				Value(opt.Value),
			}
			if opt.Disabled {
				optAttrs = append(optAttrs, Disabled())
			}
			if opt.Value == cfg.value {
				optAttrs = append(optAttrs, Selected())
			}
			optAttrs = append(optAttrs, Text(opt.Label))
			groupAttrs = append(groupAttrs, El("option", optAttrs...))
		}
		attrs = append(attrs, El("optgroup", groupAttrs...))
	}

	return El("select", attrs...)
}
