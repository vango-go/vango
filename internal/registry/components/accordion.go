//go:build vangoui

package ui

import (
	"github.com/vango-go/vango/pkg/vdom"
)

// AccordionOption configures an Accordion component.
type AccordionOption func(*accordionConfig)

type accordionConfig struct {
	openItems []string
	onToggle  func(string, bool)
	items     []AccordionItem
	multiple  bool
	className string
}

// AccordionItem represents an accordion section.
type AccordionItem struct {
	Value   string
	Trigger string
	Content *VNode
}

// AccordionOpenItems sets the currently open items.
func AccordionOpenItems(items ...string) AccordionOption {
	return func(c *accordionConfig) {
		c.openItems = items
	}
}

// AccordionOnToggle sets the toggle handler.
func AccordionOnToggle(handler func(string, bool)) AccordionOption {
	return func(c *accordionConfig) {
		c.onToggle = handler
	}
}

// AccordionItems sets the accordion items.
func AccordionItems(items ...AccordionItem) AccordionOption {
	return func(c *accordionConfig) {
		c.items = items
	}
}

// AccordionMultiple allows multiple items to be open.
func AccordionMultiple(multiple bool) AccordionOption {
	return func(c *accordionConfig) {
		c.multiple = multiple
	}
}

// AccordionClass adds additional CSS classes.
func AccordionClass(className string) AccordionOption {
	return func(c *accordionConfig) {
		c.className = className
	}
}

// Accordion renders an accordion component.
func Accordion(opts ...AccordionOption) *VNode {
	cfg := accordionConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	// Create set of open items for quick lookup
	openSet := make(map[string]bool)
	for _, item := range cfg.openItems {
		openSet[item] = true
	}

	containerAttrs := []any{
		Class(CN("w-full", cfg.className)),
	}

	for _, item := range cfg.items {
		isOpen := openSet[item.Value]

		// Item container
		itemClasses := "border-b"

		// Trigger
		triggerClasses := "flex flex-1 items-center justify-between py-4 font-medium transition-all hover:underline [&[data-state=open]>svg]:rotate-180"

		triggerAttrs := []any{
			Class(triggerClasses),
			AriaExpanded(isOpen),
			Data("state", func() string {
				if isOpen {
					return "open"
				}
				return "closed"
			}()),
		}

		if cfg.onToggle != nil {
			triggerAttrs = append(triggerAttrs, OnClick(func() {
				cfg.onToggle(item.Value, !isOpen)
			}))
		}

		// Chevron icon
		chevron := El("svg",
			Class("h-4 w-4 shrink-0 transition-transform duration-200"),
			Attr{Key: "xmlns", Value: "http://www.w3.org/2000/svg"},
			Attr{Key: "viewBox", Value: "0 0 24 24"},
			Attr{Key: "fill", Value: "none"},
			Attr{Key: "stroke", Value: "currentColor"},
			Attr{Key: "stroke-width", Value: "2"},
			El("polyline", Attr{Key: "points", Value: "6 9 12 15 18 9"}),
		)

		triggerAttrs = append(triggerAttrs, Text(item.Trigger), chevron)

		trigger := El("button", triggerAttrs...)
		header := El("h3", Class("flex"), trigger)

		// Content
		var contentEl *VNode
		if isOpen && item.Content != nil {
			contentEl = Div(
				Class("overflow-hidden text-sm transition-all data-[state=closed]:animate-accordion-up data-[state=open]:animate-accordion-down"),
				Data("state", "open"),
				Div(
					Class("pb-4 pt-0"),
					item.Content,
				),
			)
		}

		itemEl := Div(
			Class(itemClasses),
			Data("state", func() string {
				if isOpen {
					return "open"
				}
				return "closed"
			}()),
			header,
			contentEl,
		)

		containerAttrs = append(containerAttrs, itemEl)
	}

	return Div(containerAttrs...)
}
