//go:build vangoui

package ui

import (
	"github.com/vango-dev/vango/v2/pkg/vdom"
)

// PopoverOption configures a Popover component.
type PopoverOption func(*popoverConfig)

type popoverConfig struct {
	open         bool
	onOpenChange func(bool)
	trigger      *VNode
	content      *VNode
	side         string
	align        string
	className    string
}

func defaultPopoverConfig() popoverConfig {
	return popoverConfig{
		side:  "bottom",
		align: "center",
	}
}

// PopoverOpen sets the open state.
func PopoverOpen(open bool) PopoverOption {
	return func(c *popoverConfig) {
		c.open = open
	}
}

// PopoverOnOpenChange sets the open change handler.
func PopoverOnOpenChange(handler func(bool)) PopoverOption {
	return func(c *popoverConfig) {
		c.onOpenChange = handler
	}
}

// PopoverTrigger sets the trigger element.
func PopoverTrigger(trigger *VNode) PopoverOption {
	return func(c *popoverConfig) {
		c.trigger = trigger
	}
}

// PopoverContent sets the popover content.
func PopoverContent(content *VNode) PopoverOption {
	return func(c *popoverConfig) {
		c.content = content
	}
}

// PopoverSide sets the popover side (top, right, bottom, left).
func PopoverSide(side string) PopoverOption {
	return func(c *popoverConfig) {
		c.side = side
	}
}

// PopoverAlign sets the alignment (start, center, end).
func PopoverAlign(align string) PopoverOption {
	return func(c *popoverConfig) {
		c.align = align
	}
}

// PopoverClass adds additional CSS classes.
func PopoverClass(className string) PopoverOption {
	return func(c *popoverConfig) {
		c.className = className
	}
}

// Popover renders a popover component.
func Popover(opts ...PopoverOption) *VNode {
	cfg := defaultPopoverConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	handleToggle := func() {
		if cfg.onOpenChange != nil {
			cfg.onOpenChange(!cfg.open)
		}
	}

	// Trigger
	triggerEl := Div(
		Class("inline-block"),
		Data("popover-trigger", "true"),
		OnClick(handleToggle),
		cfg.trigger,
	)

	// If not open, just return trigger
	if !cfg.open {
		return Div(
			Class("relative inline-block"),
			triggerEl,
		)
	}

	// Position classes based on side
	sideClasses := map[string]string{
		"top":    "bottom-full mb-2",
		"right":  "left-full ml-2",
		"bottom": "top-full mt-2",
		"left":   "right-full mr-2",
	}

	// Alignment classes
	alignClasses := map[string]map[string]string{
		"top": {
			"start":  "left-0",
			"center": "left-1/2 -translate-x-1/2",
			"end":    "right-0",
		},
		"bottom": {
			"start":  "left-0",
			"center": "left-1/2 -translate-x-1/2",
			"end":    "right-0",
		},
		"left": {
			"start":  "top-0",
			"center": "top-1/2 -translate-y-1/2",
			"end":    "bottom-0",
		},
		"right": {
			"start":  "top-0",
			"center": "top-1/2 -translate-y-1/2",
			"end":    "bottom-0",
		},
	}

	// Content
	contentClasses := CN(
		"absolute z-50 w-72 rounded-md border bg-popover p-4 text-popover-foreground shadow-md outline-none data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95",
		sideClasses[cfg.side],
		alignClasses[cfg.side][cfg.align],
		cfg.className,
	)

	contentEl := Div(
		Class(contentClasses),
		Data("state", "open"),
		Data("side", cfg.side),
		Data("popover-content", "true"),
		Hook("Popover", map[string]any{
			"closeOnEscape":  true,
			"closeOnOutside": true,
		}),
		cfg.content,
	)

	// Container with relative positioning
	return Div(
		Class("relative inline-block"),
		triggerEl,
		contentEl,
	)
}
