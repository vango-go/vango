package ui

import (
	. "github.com/vango-dev/vango/v2/pkg/vdom"
)

// DialogOption configures a Dialog component.
type DialogOption func(*dialogConfig)

type dialogConfig struct {
	open             bool
	onOpenChange     func(bool)
	title            string
	description      string
	trigger          *VNode
	content          *VNode
	footer           *VNode
	closeOnEscape    bool
	closeOnOverlay   bool
	showCloseButton  bool
	className        string
	overlayClassName string
}

func defaultDialogConfig() dialogConfig {
	return dialogConfig{
		closeOnEscape:   true,
		closeOnOverlay:  true,
		showCloseButton: true,
	}
}

// DialogOpen sets the open state.
func DialogOpen(open bool) DialogOption {
	return func(c *dialogConfig) {
		c.open = open
	}
}

// DialogOnOpenChange sets the open change handler.
func DialogOnOpenChange(handler func(bool)) DialogOption {
	return func(c *dialogConfig) {
		c.onOpenChange = handler
	}
}

// DialogTitle sets the dialog title.
func DialogTitle(title string) DialogOption {
	return func(c *dialogConfig) {
		c.title = title
	}
}

// DialogDescription sets the dialog description.
func DialogDescription(description string) DialogOption {
	return func(c *dialogConfig) {
		c.description = description
	}
}

// DialogTrigger sets the trigger element.
func DialogTrigger(trigger *VNode) DialogOption {
	return func(c *dialogConfig) {
		c.trigger = trigger
	}
}

// DialogContent sets the dialog content.
func DialogContent(content *VNode) DialogOption {
	return func(c *dialogConfig) {
		c.content = content
	}
}

// DialogFooter sets the dialog footer.
func DialogFooter(footer *VNode) DialogOption {
	return func(c *dialogConfig) {
		c.footer = footer
	}
}

// DialogCloseOnEscape enables/disables close on Escape key.
func DialogCloseOnEscape(close bool) DialogOption {
	return func(c *dialogConfig) {
		c.closeOnEscape = close
	}
}

// DialogCloseOnOverlay enables/disables close on overlay click.
func DialogCloseOnOverlay(close bool) DialogOption {
	return func(c *dialogConfig) {
		c.closeOnOverlay = close
	}
}

// DialogShowCloseButton shows/hides the close button.
func DialogShowCloseButton(show bool) DialogOption {
	return func(c *dialogConfig) {
		c.showCloseButton = show
	}
}

// DialogClass adds additional CSS classes to the dialog content.
func DialogClass(className string) DialogOption {
	return func(c *dialogConfig) {
		c.className = className
	}
}

// DialogOverlayClass adds additional CSS classes to the overlay.
func DialogOverlayClass(className string) DialogOption {
	return func(c *dialogConfig) {
		c.overlayClassName = className
	}
}

// Dialog renders a modal dialog component.
// This component uses the FocusTrap hook for accessibility.
func Dialog(opts ...DialogOption) *VNode {
	cfg := defaultDialogConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	// Build trigger with open handler
	var triggerEl *VNode
	if cfg.trigger != nil {
		triggerEl = Div(
			OnClick(func() {
				if cfg.onOpenChange != nil {
					cfg.onOpenChange(true)
				}
			}),
			cfg.trigger,
		)
	}

	// If not open, just return trigger
	if !cfg.open {
		if triggerEl != nil {
			return triggerEl
		}
		return Fragment()
	}

	// Close handler
	handleClose := func() {
		if cfg.onOpenChange != nil {
			cfg.onOpenChange(false)
		}
	}

	// Overlay
	overlayClasses := CN(
		"fixed inset-0 z-50 bg-black/80 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0",
		cfg.overlayClassName,
	)

	overlayAttrs := []any{
		Class(overlayClasses),
		Data("state", "open"),
	}
	if cfg.closeOnOverlay {
		overlayAttrs = append(overlayAttrs, OnClick(handleClose))
	}

	overlay := Div(overlayAttrs...)

	// Content
	contentClasses := CN(
		"fixed left-[50%] top-[50%] z-50 grid w-full max-w-lg translate-x-[-50%] translate-y-[-50%] gap-4 border bg-background p-6 shadow-lg duration-200 data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[state=closed]:slide-out-to-left-1/2 data-[state=closed]:slide-out-to-top-[48%] data-[state=open]:slide-in-from-left-1/2 data-[state=open]:slide-in-from-top-[48%] sm:rounded-lg",
		cfg.className,
	)

	contentAttrs := []any{
		Role("dialog"),
		Class(contentClasses),
		Data("state", "open"),
		AriaModal(true),
		Hook("FocusTrap", map[string]any{
			"active":        true,
			"closeOnEscape": cfg.closeOnEscape,
		}),
	}

	// Header with title and close button
	var headerEl *VNode
	if cfg.title != "" || cfg.showCloseButton {
		headerChildren := []any{Class("flex flex-col space-y-1.5 text-center sm:text-left")}

		if cfg.title != "" {
			headerChildren = append(headerChildren,
				El("h2",
					Class("text-lg font-semibold leading-none tracking-tight"),
					Text(cfg.title),
				),
			)
		}

		if cfg.description != "" {
			headerChildren = append(headerChildren,
				El("p",
					Class("text-sm text-muted-foreground"),
					Text(cfg.description),
				),
			)
		}

		headerEl = Div(headerChildren...)
	}

	// Close button
	var closeButtonEl *VNode
	if cfg.showCloseButton {
		closeButtonEl = El("button",
			Class("absolute right-4 top-4 rounded-sm opacity-70 ring-offset-background transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:pointer-events-none data-[state=open]:bg-accent data-[state=open]:text-muted-foreground"),
			OnClick(handleClose),
			El("svg",
				Class("h-4 w-4"),
				Attr{Key: "xmlns", Value: "http://www.w3.org/2000/svg"},
				Attr{Key: "viewBox", Value: "0 0 24 24"},
				Attr{Key: "fill", Value: "none"},
				Attr{Key: "stroke", Value: "currentColor"},
				Attr{Key: "stroke-width", Value: "2"},
				El("line", Attr{Key: "x1", Value: "18"}, Attr{Key: "y1", Value: "6"}, Attr{Key: "x2", Value: "6"}, Attr{Key: "y2", Value: "18"}),
				El("line", Attr{Key: "x1", Value: "6"}, Attr{Key: "y1", Value: "6"}, Attr{Key: "x2", Value: "18"}, Attr{Key: "y2", Value: "18"}),
			),
			El("span", Class("sr-only"), Text("Close")),
		)
	}

	// Build content children
	if headerEl != nil {
		contentAttrs = append(contentAttrs, headerEl)
	}
	if cfg.content != nil {
		contentAttrs = append(contentAttrs, cfg.content)
	}
	if cfg.footer != nil {
		contentAttrs = append(contentAttrs,
			Div(
				Class("flex flex-col-reverse sm:flex-row sm:justify-end sm:space-x-2"),
				cfg.footer,
			),
		)
	}
	if closeButtonEl != nil {
		contentAttrs = append(contentAttrs, closeButtonEl)
	}

	dialogContent := Div(contentAttrs...)

	// Portal container (rendered at document body level in practice)
	portal := Fragment(overlay, dialogContent)

	// Return trigger + portal
	if triggerEl != nil {
		return Fragment(triggerEl, portal)
	}

	return portal
}
