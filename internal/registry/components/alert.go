//go:build vangoui

package ui

import (
	"github.com/vango-dev/vango/v2/pkg/vdom"
)

// AlertOption configures an Alert component.
type AlertOption func(*alertConfig)

type alertConfig struct {
	variant     Variant
	title       string
	description string
	icon        *VNode
	className   string
	children    []any
}

func defaultAlertConfig() alertConfig {
	return alertConfig{
		variant: VariantDefault,
	}
}

// AlertVariant sets the alert variant.
func AlertVariant(v Variant) AlertOption {
	return func(c *alertConfig) {
		c.variant = v
	}
}

// AlertDestructive sets the alert to destructive variant.
func AlertDestructive() AlertOption {
	return func(c *alertConfig) {
		c.variant = VariantDestructive
	}
}

// AlertSuccess sets the alert to success variant.
func AlertSuccess() AlertOption {
	return func(c *alertConfig) {
		c.variant = VariantSuccess
	}
}

// AlertWarning sets the alert to warning variant.
func AlertWarning() AlertOption {
	return func(c *alertConfig) {
		c.variant = VariantWarning
	}
}

// AlertTitle sets the alert title.
func AlertTitle(title string) AlertOption {
	return func(c *alertConfig) {
		c.title = title
	}
}

// AlertDescription sets the alert description.
func AlertDescription(description string) AlertOption {
	return func(c *alertConfig) {
		c.description = description
	}
}

// AlertIcon sets a custom icon.
func AlertIcon(icon *VNode) AlertOption {
	return func(c *alertConfig) {
		c.icon = icon
	}
}

// AlertClass adds additional CSS classes.
func AlertClass(className string) AlertOption {
	return func(c *alertConfig) {
		c.className = className
	}
}

// AlertChildren sets the alert children.
func AlertChildren(children ...any) AlertOption {
	return func(c *alertConfig) {
		c.children = children
	}
}

// Alert renders an alert component.
func Alert(opts ...AlertOption) *VNode {
	cfg := defaultAlertConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	// Variant classes
	variantClasses := map[Variant]string{
		VariantDefault:     "bg-background text-foreground",
		VariantDestructive: "border-destructive/50 text-destructive dark:border-destructive [&>svg]:text-destructive",
		VariantSuccess:     "border-success/50 text-success dark:border-success [&>svg]:text-success",
		VariantWarning:     "border-warning/50 text-warning dark:border-warning [&>svg]:text-warning",
	}

	classes := CN(
		"relative w-full rounded-lg border p-4 [&>svg~*]:pl-7 [&>svg+div]:translate-y-[-3px] [&>svg]:absolute [&>svg]:left-4 [&>svg]:top-4 [&>svg]:text-foreground",
		variantClasses[cfg.variant],
		cfg.className,
	)

	attrs := []any{
		Role("alert"),
		Class(classes),
	}

	// Add icon if provided
	if cfg.icon != nil {
		attrs = append(attrs, cfg.icon)
	}

	// Add title if provided
	if cfg.title != "" {
		attrs = append(attrs,
			El("h5",
				Class("mb-1 font-medium leading-none tracking-tight"),
				Text(cfg.title),
			),
		)
	}

	// Add description if provided
	if cfg.description != "" {
		attrs = append(attrs,
			Div(
				Class("text-sm [&_p]:leading-relaxed"),
				Text(cfg.description),
			),
		)
	}

	attrs = append(attrs, cfg.children...)

	return Div(attrs...)
}
