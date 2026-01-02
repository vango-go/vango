package ui

import (
	. "github.com/vango-dev/vango/v2/pkg/vdom"
)

// DropdownOption configures a Dropdown component.
type DropdownOption func(*dropdownConfig)

type dropdownConfig struct {
	open         bool
	onOpenChange func(bool)
	trigger      *VNode
	items        []DropdownItem
	align        string
	className    string
}

// DropdownItem represents a menu item.
type DropdownItem struct {
	Label    string
	Icon     *VNode
	Shortcut string
	Disabled bool
	OnSelect func()
	Href     string
	Variant  Variant
}

func defaultDropdownConfig() dropdownConfig {
	return dropdownConfig{
		align: "start",
	}
}

// DropdownOpen sets the open state.
func DropdownOpen(open bool) DropdownOption {
	return func(c *dropdownConfig) {
		c.open = open
	}
}

// DropdownOnOpenChange sets the open change handler.
func DropdownOnOpenChange(handler func(bool)) DropdownOption {
	return func(c *dropdownConfig) {
		c.onOpenChange = handler
	}
}

// DropdownTrigger sets the trigger element.
func DropdownTrigger(trigger *VNode) DropdownOption {
	return func(c *dropdownConfig) {
		c.trigger = trigger
	}
}

// DropdownItems sets the menu items.
func DropdownItems(items ...DropdownItem) DropdownOption {
	return func(c *dropdownConfig) {
		c.items = items
	}
}

// DropdownAlign sets the menu alignment (start, center, end).
func DropdownAlign(align string) DropdownOption {
	return func(c *dropdownConfig) {
		c.align = align
	}
}

// DropdownClass adds additional CSS classes.
func DropdownClass(className string) DropdownOption {
	return func(c *dropdownConfig) {
		c.className = className
	}
}

// Dropdown renders a dropdown menu component.
func Dropdown(opts ...DropdownOption) *VNode {
	cfg := defaultDropdownConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	handleOpen := func() {
		if cfg.onOpenChange != nil {
			cfg.onOpenChange(true)
		}
	}

	handleClose := func() {
		if cfg.onOpenChange != nil {
			cfg.onOpenChange(false)
		}
	}

	// Trigger
	triggerEl := Div(
		Class("inline-block"),
		Data("dropdown-trigger", "true"),
		OnClick(handleOpen),
		cfg.trigger,
	)

	// If not open, just return trigger
	if !cfg.open {
		return triggerEl
	}

	// Alignment classes
	alignClasses := map[string]string{
		"start":  "left-0",
		"center": "left-1/2 -translate-x-1/2",
		"end":    "right-0",
	}

	// Menu content
	menuClasses := CN(
		"absolute z-50 min-w-[8rem] overflow-hidden rounded-md border bg-popover p-1 text-popover-foreground shadow-md data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95",
		alignClasses[cfg.align],
		cfg.className,
	)

	menuAttrs := []any{
		Role("menu"),
		Class(menuClasses),
		Data("state", "open"),
		Hook("Dropdown", map[string]any{
			"closeOnEscape":  true,
			"closeOnOutside": true,
		}),
	}

	// Build menu items
	for _, item := range cfg.items {
		itemClasses := "relative flex cursor-pointer select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none transition-colors focus:bg-accent focus:text-accent-foreground hover:bg-accent hover:text-accent-foreground"

		if item.Disabled {
			itemClasses = CN(itemClasses, "pointer-events-none opacity-50")
		}

		if item.Variant == VariantDestructive {
			itemClasses = CN(itemClasses, "text-destructive focus:text-destructive")
		}

		itemAttrs := []any{
			Role("menuitem"),
			Class(itemClasses),
		}

		if item.Disabled {
			itemAttrs = append(itemAttrs, AriaDisabled(true))
		}

		if item.OnSelect != nil && !item.Disabled {
			itemAttrs = append(itemAttrs, OnClick(func() {
				item.OnSelect()
				handleClose()
			}))
		}

		// Icon
		if item.Icon != nil {
			itemAttrs = append(itemAttrs,
				El("span", Class("mr-2 h-4 w-4"), item.Icon),
			)
		}

		// Label
		itemAttrs = append(itemAttrs, El("span", Text(item.Label)))

		// Shortcut
		if item.Shortcut != "" {
			itemAttrs = append(itemAttrs,
				El("span",
					Class("ml-auto text-xs tracking-widest opacity-60"),
					Text(item.Shortcut),
				),
			)
		}

		// Build item (button or link)
		var itemEl *VNode
		if item.Href != "" {
			itemAttrs = append([]any{Href(item.Href)}, itemAttrs...)
			itemEl = El("a", itemAttrs...)
		} else {
			itemEl = Div(itemAttrs...)
		}

		menuAttrs = append(menuAttrs, itemEl)
	}

	menu := Div(menuAttrs...)

	// Container with relative positioning
	return Div(
		Class("relative inline-block"),
		triggerEl,
		menu,
	)
}

// DropdownSeparator returns a separator item (visual only).
func DropdownSeparator() *VNode {
	return Div(Class("-mx-1 my-1 h-px bg-muted"))
}

// DropdownLabel returns a label item (non-interactive).
func DropdownLabel(text string) *VNode {
	return Div(
		Class("px-2 py-1.5 text-sm font-semibold"),
		Text(text),
	)
}
