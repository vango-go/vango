//go:build vangoui

package ui

import (
	"github.com/vango-dev/vango/v2/pkg/vdom"
)

// CardOption configures a Card component.
type CardOption func(*cardConfig)

type cardConfig struct {
	className string
	children  []any
}

// CardClass adds additional CSS classes to the card.
func CardClass(className string) CardOption {
	return func(c *cardConfig) {
		c.className = className
	}
}

// CardChildren sets the card children.
func CardChildren(children ...any) CardOption {
	return func(c *cardConfig) {
		c.children = children
	}
}

// Card renders a card container.
func Card(opts ...CardOption) *VNode {
	cfg := cardConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	classes := CN(
		"rounded-lg border bg-card text-card-foreground shadow-sm",
		cfg.className,
	)

	attrs := []any{Class(classes)}
	attrs = append(attrs, cfg.children...)

	return Div(attrs...)
}

// CardHeaderOption configures a CardHeader component.
type CardHeaderOption func(*cardHeaderConfig)

type cardHeaderConfig struct {
	className string
	children  []any
}

// CardHeaderClass adds additional CSS classes.
func CardHeaderClass(className string) CardHeaderOption {
	return func(c *cardHeaderConfig) {
		c.className = className
	}
}

// CardHeaderChildren sets the children.
func CardHeaderChildren(children ...any) CardHeaderOption {
	return func(c *cardHeaderConfig) {
		c.children = children
	}
}

// CardHeader renders the card header section.
func CardHeader(opts ...CardHeaderOption) *VNode {
	cfg := cardHeaderConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	classes := CN(
		"flex flex-col space-y-1.5 p-6",
		cfg.className,
	)

	attrs := []any{Class(classes)}
	attrs = append(attrs, cfg.children...)

	return Div(attrs...)
}

// CardTitleOption configures a CardTitle component.
type CardTitleOption func(*cardTitleConfig)

type cardTitleConfig struct {
	className string
	text      string
	children  []any
}

// CardTitleClass adds additional CSS classes.
func CardTitleClass(className string) CardTitleOption {
	return func(c *cardTitleConfig) {
		c.className = className
	}
}

// CardTitleText sets the title text.
func CardTitleText(text string) CardTitleOption {
	return func(c *cardTitleConfig) {
		c.text = text
	}
}

// CardTitleChildren sets the children.
func CardTitleChildren(children ...any) CardTitleOption {
	return func(c *cardTitleConfig) {
		c.children = children
	}
}

// CardTitle renders the card title.
func CardTitle(opts ...CardTitleOption) *VNode {
	cfg := cardTitleConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	classes := CN(
		"text-2xl font-semibold leading-none tracking-tight",
		cfg.className,
	)

	attrs := []any{Class(classes)}
	if cfg.text != "" {
		attrs = append(attrs, Text(cfg.text))
	}
	attrs = append(attrs, cfg.children...)

	return El("h3", attrs...)
}

// CardDescriptionOption configures a CardDescription component.
type CardDescriptionOption func(*cardDescriptionConfig)

type cardDescriptionConfig struct {
	className string
	text      string
	children  []any
}

// CardDescriptionClass adds additional CSS classes.
func CardDescriptionClass(className string) CardDescriptionOption {
	return func(c *cardDescriptionConfig) {
		c.className = className
	}
}

// CardDescriptionText sets the description text.
func CardDescriptionText(text string) CardDescriptionOption {
	return func(c *cardDescriptionConfig) {
		c.text = text
	}
}

// CardDescriptionChildren sets the children.
func CardDescriptionChildren(children ...any) CardDescriptionOption {
	return func(c *cardDescriptionConfig) {
		c.children = children
	}
}

// CardDescription renders the card description.
func CardDescription(opts ...CardDescriptionOption) *VNode {
	cfg := cardDescriptionConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	classes := CN(
		"text-sm text-muted-foreground",
		cfg.className,
	)

	attrs := []any{Class(classes)}
	if cfg.text != "" {
		attrs = append(attrs, Text(cfg.text))
	}
	attrs = append(attrs, cfg.children...)

	return El("p", attrs...)
}

// CardContentOption configures a CardContent component.
type CardContentOption func(*cardContentConfig)

type cardContentConfig struct {
	className string
	children  []any
}

// CardContentClass adds additional CSS classes.
func CardContentClass(className string) CardContentOption {
	return func(c *cardContentConfig) {
		c.className = className
	}
}

// CardContentChildren sets the children.
func CardContentChildren(children ...any) CardContentOption {
	return func(c *cardContentConfig) {
		c.children = children
	}
}

// CardContent renders the card content section.
func CardContent(opts ...CardContentOption) *VNode {
	cfg := cardContentConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	classes := CN(
		"p-6 pt-0",
		cfg.className,
	)

	attrs := []any{Class(classes)}
	attrs = append(attrs, cfg.children...)

	return Div(attrs...)
}

// CardFooterOption configures a CardFooter component.
type CardFooterOption func(*cardFooterConfig)

type cardFooterConfig struct {
	className string
	children  []any
}

// CardFooterClass adds additional CSS classes.
func CardFooterClass(className string) CardFooterOption {
	return func(c *cardFooterConfig) {
		c.className = className
	}
}

// CardFooterChildren sets the children.
func CardFooterChildren(children ...any) CardFooterOption {
	return func(c *cardFooterConfig) {
		c.children = children
	}
}

// CardFooter renders the card footer section.
func CardFooter(opts ...CardFooterOption) *VNode {
	cfg := cardFooterConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	classes := CN(
		"flex items-center p-6 pt-0",
		cfg.className,
	)

	attrs := []any{Class(classes)}
	attrs = append(attrs, cfg.children...)

	return Div(attrs...)
}
