package ui

import (
	. "github.com/vango-dev/vango/v2/pkg/vdom"
)

// TooltipOption configures a Tooltip component.
type TooltipOption func(*tooltipConfig)

type tooltipConfig struct {
	content   string
	side      string
	align     string
	delay     int
	className string
	children  []any
}

func defaultTooltipConfig() tooltipConfig {
	return tooltipConfig{
		side:  "top",
		align: "center",
		delay: 200,
	}
}

// TooltipContent sets the tooltip text content.
func TooltipContent(content string) TooltipOption {
	return func(c *tooltipConfig) {
		c.content = content
	}
}

// TooltipSide sets the tooltip side (top, right, bottom, left).
func TooltipSide(side string) TooltipOption {
	return func(c *tooltipConfig) {
		c.side = side
	}
}

// TooltipAlign sets the alignment (start, center, end).
func TooltipAlign(align string) TooltipOption {
	return func(c *tooltipConfig) {
		c.align = align
	}
}

// TooltipDelay sets the delay before showing (in ms).
func TooltipDelay(delay int) TooltipOption {
	return func(c *tooltipConfig) {
		c.delay = delay
	}
}

// TooltipClass adds additional CSS classes.
func TooltipClass(className string) TooltipOption {
	return func(c *tooltipConfig) {
		c.className = className
	}
}

// TooltipChildren sets the trigger children.
func TooltipChildren(children ...any) TooltipOption {
	return func(c *tooltipConfig) {
		c.children = children
	}
}

// Tooltip renders a tooltip component.
// The tooltip uses the Tooltip client hook for hover behavior.
func Tooltip(opts ...TooltipOption) *VNode {
	cfg := defaultTooltipConfig()
	for _, opt := range opts {
		opt(&cfg)
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

	// Tooltip content (hidden by default, shown by JS hook)
	contentClasses := CN(
		"absolute z-50 hidden overflow-hidden rounded-md border bg-popover px-3 py-1.5 text-sm text-popover-foreground shadow-md animate-in fade-in-0 zoom-in-95",
		sideClasses[cfg.side],
		alignClasses[cfg.side][cfg.align],
		cfg.className,
	)

	tooltipContent := El("span",
		Class(contentClasses),
		Role("tooltip"),
		Data("tooltip-content", "true"),
		Text(cfg.content),
	)

	// Container with hook
	containerAttrs := []any{
		Class("relative inline-block"),
		Hook("Tooltip", map[string]any{
			"side":  cfg.side,
			"delay": cfg.delay,
		}),
	}

	// Add trigger children
	triggerAttrs := []any{
		Class("inline-block"),
		Data("tooltip-trigger", "true"),
	}
	triggerAttrs = append(triggerAttrs, cfg.children...)
	trigger := El("span", triggerAttrs...)

	containerAttrs = append(containerAttrs, trigger, tooltipContent)

	return Div(containerAttrs...)
}
