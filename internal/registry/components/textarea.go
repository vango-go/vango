//go:build vangoui

package ui

import (
	"github.com/vango-dev/vango/v2/pkg/vdom"
)

// TextareaOption configures a Textarea component.
type TextareaOption func(*textareaConfig)

type textareaConfig struct {
	name        string
	placeholder string
	value       string
	rows        int
	disabled    bool
	required    bool
	readonly    bool
	className   string
	onInput     func(string)
	onChange    func(string)
}

func defaultTextareaConfig() textareaConfig {
	return textareaConfig{
		rows: 3,
	}
}

// TextareaName sets the textarea name.
func TextareaName(name string) TextareaOption {
	return func(c *textareaConfig) {
		c.name = name
	}
}

// TextareaPlaceholder sets the placeholder text.
func TextareaPlaceholder(placeholder string) TextareaOption {
	return func(c *textareaConfig) {
		c.placeholder = placeholder
	}
}

// TextareaValue sets the textarea value.
func TextareaValue(value string) TextareaOption {
	return func(c *textareaConfig) {
		c.value = value
	}
}

// TextareaRows sets the number of visible rows.
func TextareaRows(rows int) TextareaOption {
	return func(c *textareaConfig) {
		c.rows = rows
	}
}

// TextareaDisabled sets the disabled state.
func TextareaDisabled(disabled bool) TextareaOption {
	return func(c *textareaConfig) {
		c.disabled = disabled
	}
}

// TextareaRequired sets the required state.
func TextareaRequired(required bool) TextareaOption {
	return func(c *textareaConfig) {
		c.required = required
	}
}

// TextareaReadonly sets the readonly state.
func TextareaReadonly(readonly bool) TextareaOption {
	return func(c *textareaConfig) {
		c.readonly = readonly
	}
}

// TextareaClass adds additional CSS classes.
func TextareaClass(className string) TextareaOption {
	return func(c *textareaConfig) {
		c.className = className
	}
}

// TextareaOnInput sets the input event handler.
func TextareaOnInput(handler func(string)) TextareaOption {
	return func(c *textareaConfig) {
		c.onInput = handler
	}
}

// TextareaOnChange sets the change event handler.
func TextareaOnChange(handler func(string)) TextareaOption {
	return func(c *textareaConfig) {
		c.onChange = handler
	}
}

// Textarea renders a textarea element.
func Textarea(opts ...TextareaOption) *VNode {
	cfg := defaultTextareaConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	classes := CN(
		"flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50",
		cfg.className,
	)

	attrs := []any{
		Class(classes),
		Rows(cfg.rows),
	}

	if cfg.name != "" {
		attrs = append(attrs, Name(cfg.name))
	}
	if cfg.placeholder != "" {
		attrs = append(attrs, Placeholder(cfg.placeholder))
	}
	if cfg.disabled {
		attrs = append(attrs, Disabled())
	}
	if cfg.required {
		attrs = append(attrs, Required())
	}
	if cfg.readonly {
		attrs = append(attrs, Readonly())
	}
	if cfg.onInput != nil {
		attrs = append(attrs, OnInput(cfg.onInput))
	}
	if cfg.onChange != nil {
		attrs = append(attrs, OnChange(cfg.onChange))
	}

	// Value is content for textarea
	if cfg.value != "" {
		attrs = append(attrs, Text(cfg.value))
	}

	return El("textarea", attrs...)
}
