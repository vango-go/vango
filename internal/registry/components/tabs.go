//go:build vangoui

package ui

import (
	"github.com/vango-dev/vango/v2/pkg/vdom"
)

// TabsOption configures a Tabs component.
type TabsOption func(*tabsConfig)

type tabsConfig struct {
	value    string
	onChange func(string)
	tabs     []TabItem
	variant  string
	className string
}

// TabItem represents a tab with its content.
type TabItem struct {
	Value    string
	Label    string
	Icon     *VNode
	Disabled bool
	Content  *VNode
}

func defaultTabsConfig() tabsConfig {
	return tabsConfig{
		variant: "default",
	}
}

// TabsValue sets the active tab value.
func TabsValue(value string) TabsOption {
	return func(c *tabsConfig) {
		c.value = value
	}
}

// TabsOnChange sets the tab change handler.
func TabsOnChange(handler func(string)) TabsOption {
	return func(c *tabsConfig) {
		c.onChange = handler
	}
}

// TabsItems sets the tab items.
func TabsItems(tabs ...TabItem) TabsOption {
	return func(c *tabsConfig) {
		c.tabs = tabs
	}
}

// TabsVariant sets the tabs variant (default, pills).
func TabsVariant(variant string) TabsOption {
	return func(c *tabsConfig) {
		c.variant = variant
	}
}

// TabsClass adds additional CSS classes.
func TabsClass(className string) TabsOption {
	return func(c *tabsConfig) {
		c.className = className
	}
}

// Tabs renders a tabs component.
func Tabs(opts ...TabsOption) *VNode {
	cfg := defaultTabsConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	// Default to first tab if no value
	activeValue := cfg.value
	if activeValue == "" && len(cfg.tabs) > 0 {
		activeValue = cfg.tabs[0].Value
	}

	// Tab list classes based on variant
	listClasses := "inline-flex h-10 items-center justify-center rounded-md bg-muted p-1 text-muted-foreground"
	if cfg.variant == "pills" {
		listClasses = "inline-flex h-10 items-center justify-center gap-2"
	}

	// Build tab triggers
	var triggers []any
	triggers = append(triggers, Class(listClasses), Role("tablist"))

	for _, tab := range cfg.tabs {
		isActive := tab.Value == activeValue

		// Trigger classes based on variant
		triggerClasses := "inline-flex items-center justify-center whitespace-nowrap rounded-sm px-3 py-1.5 text-sm font-medium ring-offset-background transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50"

		if cfg.variant == "pills" {
			if isActive {
				triggerClasses = CN(triggerClasses, "bg-primary text-primary-foreground")
			} else {
				triggerClasses = CN(triggerClasses, "hover:bg-muted")
			}
		} else {
			if isActive {
				triggerClasses = CN(triggerClasses, "bg-background text-foreground shadow-sm")
			}
		}

		triggerAttrs := []any{
			Role("tab"),
			Class(triggerClasses),
			Data("state", func() string {
				if isActive {
					return "active"
				}
				return "inactive"
			}()),
			AriaSelected(isActive),
		}

		if tab.Disabled {
			triggerAttrs = append(triggerAttrs, AriaDisabled(true))
		} else {
			triggerAttrs = append(triggerAttrs, OnClick(func() {
				if cfg.onChange != nil {
					cfg.onChange(tab.Value)
				}
			}))
		}

		// Icon
		if tab.Icon != nil {
			triggerAttrs = append(triggerAttrs, El("span", Class("mr-2"), tab.Icon))
		}

		// Label
		triggerAttrs = append(triggerAttrs, Text(tab.Label))

		triggers = append(triggers, El("button", triggerAttrs...))
	}

	tabList := Div(triggers...)

	// Find active content
	var activeContent *VNode
	for _, tab := range cfg.tabs {
		if tab.Value == activeValue && tab.Content != nil {
			activeContent = Div(
				Role("tabpanel"),
				Class("mt-2 ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"),
				Data("state", "active"),
				tab.Content,
			)
			break
		}
	}

	return Div(
		Class(CN("w-full", cfg.className)),
		tabList,
		activeContent,
	)
}
