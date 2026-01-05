//go:build vangoui

package ui

import (
	"github.com/vango-go/vango/pkg/vdom"
)

// LabelOption configures a Label component.
type LabelOption func(*labelConfig)

type labelConfig struct {
	forID     string
	required  bool
	disabled  bool
	className string
	text      string
	children  []any
}

// LabelFor sets the for attribute (associates with input by ID).
func LabelFor(id string) LabelOption {
	return func(c *labelConfig) {
		c.forID = id
	}
}

// LabelRequired shows a required indicator.
func LabelRequired(required bool) LabelOption {
	return func(c *labelConfig) {
		c.required = required
	}
}

// LabelDisabled shows disabled styling.
func LabelDisabled(disabled bool) LabelOption {
	return func(c *labelConfig) {
		c.disabled = disabled
	}
}

// LabelClass adds additional CSS classes.
func LabelClass(className string) LabelOption {
	return func(c *labelConfig) {
		c.className = className
	}
}

// LabelText sets the label text.
func LabelText(text string) LabelOption {
	return func(c *labelConfig) {
		c.text = text
	}
}

// LabelChildren sets the label children.
func LabelChildren(children ...any) LabelOption {
	return func(c *labelConfig) {
		c.children = children
	}
}

// Label renders a label element.
func Label(opts ...LabelOption) *VNode {
	cfg := labelConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	baseClasses := "text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"

	disabledClasses := ""
	if cfg.disabled {
		disabledClasses = "cursor-not-allowed opacity-70"
	}

	classes := CN(
		baseClasses,
		disabledClasses,
		cfg.className,
	)

	attrs := []any{Class(classes)}

	if cfg.forID != "" {
		attrs = append(attrs, For(cfg.forID))
	}

	if cfg.text != "" {
		attrs = append(attrs, Text(cfg.text))
	}

	attrs = append(attrs, cfg.children...)

	// Add required indicator
	if cfg.required {
		attrs = append(attrs,
			El("span",
				Class("text-destructive ml-1"),
				Text("*"),
			),
		)
	}

	return El("label", attrs...)
}
