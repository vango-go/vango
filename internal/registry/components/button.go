//go:build vangoui

package ui

import (
	"github.com/vango-go/vango/pkg/vdom"
)

// ButtonOption configures a Button component.
type ButtonOption func(*buttonConfig)

type buttonConfig struct {
	variant   Variant
	size      Size
	disabled  bool
	loading   bool
	asChild   bool
	className string
	children  []any
	onClick   func()
	attrs     map[string]string
}

func defaultButtonConfig() buttonConfig {
	return buttonConfig{
		variant: VariantDefault,
		size:    SizeMd,
	}
}

// Variant options for Button

// WithVariant sets the button variant.
func WithVariant(v Variant) ButtonOption {
	return func(c *buttonConfig) {
		c.variant = v
	}
}

// Primary sets the button to primary variant.
func Primary() ButtonOption {
	return func(c *buttonConfig) {
		c.variant = VariantPrimary
	}
}

// Secondary sets the button to secondary variant.
func Secondary() ButtonOption {
	return func(c *buttonConfig) {
		c.variant = VariantSecondary
	}
}

// Destructive sets the button to destructive variant.
func Destructive() ButtonOption {
	return func(c *buttonConfig) {
		c.variant = VariantDestructive
	}
}

// Outline sets the button to outline variant.
func Outline() ButtonOption {
	return func(c *buttonConfig) {
		c.variant = VariantOutline
	}
}

// Ghost sets the button to ghost variant.
func Ghost() ButtonOption {
	return func(c *buttonConfig) {
		c.variant = VariantGhost
	}
}

// Link sets the button to link variant.
func Link() ButtonOption {
	return func(c *buttonConfig) {
		c.variant = VariantLink
	}
}

// Size options for Button

// WithSize sets the button size.
func WithSize(s Size) ButtonOption {
	return func(c *buttonConfig) {
		c.size = s
	}
}

// Sm sets the button to small size.
func Sm() ButtonOption {
	return func(c *buttonConfig) {
		c.size = SizeSm
	}
}

// Lg sets the button to large size.
func Lg() ButtonOption {
	return func(c *buttonConfig) {
		c.size = SizeLg
	}
}

// Icon sets the button to icon size.
func Icon() ButtonOption {
	return func(c *buttonConfig) {
		c.size = SizeIcon
	}
}

// Behavior options

// WithDisabled sets the disabled state.
func WithDisabled(d bool) ButtonOption {
	return func(c *buttonConfig) {
		c.disabled = d
	}
}

// WithLoading sets the loading state.
func WithLoading(l bool) ButtonOption {
	return func(c *buttonConfig) {
		c.loading = l
	}
}

// WithOnClick sets the click handler.
func WithOnClick(handler func()) ButtonOption {
	return func(c *buttonConfig) {
		c.onClick = handler
	}
}

// WithChildren sets the button children.
func WithChildren(children ...any) ButtonOption {
	return func(c *buttonConfig) {
		c.children = children
	}
}

// WithClass adds additional CSS classes.
func WithClass(className string) ButtonOption {
	return func(c *buttonConfig) {
		c.className = className
	}
}

// WithAttr adds a custom attribute.
func WithAttr(name, value string) ButtonOption {
	return func(c *buttonConfig) {
		if c.attrs == nil {
			c.attrs = make(map[string]string)
		}
		c.attrs[name] = value
	}
}

// Button renders a button element with the configured options.
func Button(opts ...ButtonOption) *vdom.VNode {
	cfg := defaultButtonConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	// Base classes
	baseClasses := "inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50"

	// Variant classes
	variantClasses := map[Variant]string{
		VariantDefault:     "bg-primary text-primary-foreground hover:bg-primary/90",
		VariantPrimary:     "bg-primary text-primary-foreground hover:bg-primary/90",
		VariantDestructive: "bg-destructive text-destructive-foreground hover:bg-destructive/90",
		VariantOutline:     "border border-input bg-background hover:bg-accent hover:text-accent-foreground",
		VariantSecondary:   "bg-secondary text-secondary-foreground hover:bg-secondary/80",
		VariantGhost:       "hover:bg-accent hover:text-accent-foreground",
		VariantLink:        "text-primary underline-offset-4 hover:underline",
	}

	// Size classes
	sizeClasses := map[Size]string{
		SizeXS:   "h-7 rounded px-2 text-xs",
		SizeSm:   "h-9 rounded-md px-3",
		SizeMd:   "h-10 px-4 py-2",
		SizeLg:   "h-11 rounded-md px-8",
		SizeXL:   "h-12 rounded-md px-10 text-base",
		SizeIcon: "h-10 w-10",
	}

	classes := vdom.CN(
		baseClasses,
		variantClasses[cfg.variant],
		sizeClasses[cfg.size],
		cfg.className,
	)

	// Build children
	var children []any
	if cfg.loading {
		children = append(children, spinnerIcon())
	}
	children = append(children, cfg.children...)

	// Build attributes
	attrs := []any{
		vdom.Class(classes),
	}

	if cfg.disabled || cfg.loading {
		attrs = append(attrs, vdom.Disabled())
	}

	if cfg.onClick != nil && !cfg.disabled && !cfg.loading {
		attrs = append(attrs, vdom.OnClick(cfg.onClick))
	}

	for name, value := range cfg.attrs {
		attrs = append(attrs, vdom.Data(name, value))
	}

	// Append children to attrs
	attrs = append(attrs, children...)

	return vdom.El("button", attrs...)
}

// spinnerIcon returns an SVG spinner icon for loading state.
func spinnerIcon() *vdom.VNode {
	return vdom.El("svg",
		vdom.Class("mr-2 h-4 w-4 animate-spin"),
		vdom.Attr{Key: "xmlns", Value: "http://www.w3.org/2000/svg"},
		vdom.Attr{Key: "fill", Value: "none"},
		vdom.Attr{Key: "viewBox", Value: "0 0 24 24"},
		vdom.El("circle",
			vdom.Class("opacity-25"),
			vdom.Attr{Key: "cx", Value: "12"},
			vdom.Attr{Key: "cy", Value: "12"},
			vdom.Attr{Key: "r", Value: "10"},
			vdom.Attr{Key: "stroke", Value: "currentColor"},
			vdom.Attr{Key: "stroke-width", Value: "4"},
		),
		vdom.El("path",
			vdom.Class("opacity-75"),
			vdom.Attr{Key: "fill", Value: "currentColor"},
			vdom.Attr{Key: "d", Value: "M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"},
		),
	)
}
