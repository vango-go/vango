package ui

import (
	. "github.com/vango-dev/vango/v2/pkg/vdom"
)

// SeparatorOption configures a Separator component.
type SeparatorOption func(*separatorConfig)

type separatorConfig struct {
	orientation string
	decorative  bool
	className   string
}

func defaultSeparatorConfig() separatorConfig {
	return separatorConfig{
		orientation: "horizontal",
		decorative:  true,
	}
}

// SeparatorHorizontal sets horizontal orientation (default).
func SeparatorHorizontal() SeparatorOption {
	return func(c *separatorConfig) {
		c.orientation = "horizontal"
	}
}

// SeparatorVertical sets vertical orientation.
func SeparatorVertical() SeparatorOption {
	return func(c *separatorConfig) {
		c.orientation = "vertical"
	}
}

// SeparatorDecorative sets whether the separator is decorative.
func SeparatorDecorative(decorative bool) SeparatorOption {
	return func(c *separatorConfig) {
		c.decorative = decorative
	}
}

// SeparatorClass adds additional CSS classes.
func SeparatorClass(className string) SeparatorOption {
	return func(c *separatorConfig) {
		c.className = className
	}
}

// Separator renders a visual separator line.
func Separator(opts ...SeparatorOption) *VNode {
	cfg := defaultSeparatorConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	orientationClasses := ""
	if cfg.orientation == "horizontal" {
		orientationClasses = "h-[1px] w-full"
	} else {
		orientationClasses = "h-full w-[1px]"
	}

	classes := CN(
		"shrink-0 bg-border",
		orientationClasses,
		cfg.className,
	)

	attrs := []any{
		Class(classes),
		Role("separator"),
		Data("orientation", cfg.orientation),
	}

	if cfg.decorative {
		attrs = append(attrs, AriaHidden(true))
	}

	return Div(attrs...)
}
