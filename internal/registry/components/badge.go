//go:build vangoui

package ui

import (
	"github.com/vango-dev/vango/v2/pkg/vdom"
)

// BadgeOption configures a Badge component.
type BadgeOption func(*badgeConfig)

type badgeConfig struct {
	variant   Variant
	className string
	text      string
	children  []any
}

func defaultBadgeConfig() badgeConfig {
	return badgeConfig{
		variant: VariantDefault,
	}
}

// BadgeVariant sets the badge variant.
func BadgeVariant(v Variant) BadgeOption {
	return func(c *badgeConfig) {
		c.variant = v
	}
}

// BadgeSecondary sets the badge to secondary variant.
func BadgeSecondary() BadgeOption {
	return func(c *badgeConfig) {
		c.variant = VariantSecondary
	}
}

// BadgeDestructive sets the badge to destructive variant.
func BadgeDestructive() BadgeOption {
	return func(c *badgeConfig) {
		c.variant = VariantDestructive
	}
}

// BadgeOutline sets the badge to outline variant.
func BadgeOutline() BadgeOption {
	return func(c *badgeConfig) {
		c.variant = VariantOutline
	}
}

// BadgeSuccess sets the badge to success variant.
func BadgeSuccess() BadgeOption {
	return func(c *badgeConfig) {
		c.variant = VariantSuccess
	}
}

// BadgeWarning sets the badge to warning variant.
func BadgeWarning() BadgeOption {
	return func(c *badgeConfig) {
		c.variant = VariantWarning
	}
}

// BadgeClass adds additional CSS classes.
func BadgeClass(className string) BadgeOption {
	return func(c *badgeConfig) {
		c.className = className
	}
}

// BadgeText sets the badge text.
func BadgeText(text string) BadgeOption {
	return func(c *badgeConfig) {
		c.text = text
	}
}

// BadgeChildren sets the badge children.
func BadgeChildren(children ...any) BadgeOption {
	return func(c *badgeConfig) {
		c.children = children
	}
}

// Badge renders a badge/tag element.
func Badge(opts ...BadgeOption) *VNode {
	cfg := defaultBadgeConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	// Base classes
	baseClasses := "inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"

	// Variant classes
	variantClasses := map[Variant]string{
		VariantDefault:     "border-transparent bg-primary text-primary-foreground hover:bg-primary/80",
		VariantPrimary:     "border-transparent bg-primary text-primary-foreground hover:bg-primary/80",
		VariantSecondary:   "border-transparent bg-secondary text-secondary-foreground hover:bg-secondary/80",
		VariantDestructive: "border-transparent bg-destructive text-destructive-foreground hover:bg-destructive/80",
		VariantOutline:     "text-foreground",
		VariantSuccess:     "border-transparent bg-success text-success-foreground hover:bg-success/80",
		VariantWarning:     "border-transparent bg-warning text-warning-foreground hover:bg-warning/80",
	}

	classes := CN(
		baseClasses,
		variantClasses[cfg.variant],
		cfg.className,
	)

	attrs := []any{Class(classes)}
	if cfg.text != "" {
		attrs = append(attrs, Text(cfg.text))
	}
	attrs = append(attrs, cfg.children...)

	return El("span", attrs...)
}
