package optimistic

import (
	"fmt"
	"strings"

	"github.com/vango-dev/vango/v2/pkg/render"
	"github.com/vango-dev/vango/v2/pkg/vdom"
)

// OptimisticBuilder combines multiple optimistic options into one config.
// Use the Optimistic() function to create a builder, then chain methods.
//
// Example:
//
//	Button(
//	    Text("Submit"),
//	    OnClick(handleSubmit),
//	    Optimistic().
//	        Class("submitting", true).
//	        Text("Submitting...").
//	        Build(),
//	)
type OptimisticBuilder struct {
	class string // "classname:action" format
	text  string
	attr  string
	value string
}

// Optimistic creates a new builder for chaining optimistic updates.
func Optimistic() *OptimisticBuilder {
	return &OptimisticBuilder{}
}

// Class adds a class modification to the builder.
func (b *OptimisticBuilder) Class(class string, add bool) *OptimisticBuilder {
	action := "remove"
	if add {
		action = "add"
	}
	b.class = fmt.Sprintf("%s:%s", class, action)
	return b
}

// ClassToggle adds a class toggle to the builder.
func (b *OptimisticBuilder) ClassToggle(class string) *OptimisticBuilder {
	b.class = fmt.Sprintf("%s:toggle", class)
	return b
}

// Text sets the optimistic text replacement.
func (b *OptimisticBuilder) Text(text string) *OptimisticBuilder {
	b.text = text
	return b
}

// Attr sets an attribute modification.
func (b *OptimisticBuilder) Attr(name, value string) *OptimisticBuilder {
	b.attr = name
	b.value = value
	return b
}

// AttrRemove sets an attribute for removal.
func (b *OptimisticBuilder) AttrRemove(name string) *OptimisticBuilder {
	b.attr = name
	b.value = "" // Empty value signals removal
	return b
}

// Build returns a vdom.Attr with _optimistic prop containing OptimisticConfig.
func (b *OptimisticBuilder) Build() vdom.Attr {
	return vdom.Attr{
		Key: "_optimistic",
		Value: render.OptimisticConfig{
			Class: b.class,
			Text:  b.text,
			Attr:  b.attr,
			Value: b.value,
		},
	}
}

// Action represents the type of optimistic change.
type Action string

const (
	ActionAdd    Action = "add"
	ActionRemove Action = "remove"
	ActionSet    Action = "set"
	ActionToggle Action = "toggle"
)

// Class creates an optimistic class toggle attribute.
// When the element is clicked, the class is immediately added or removed
// based on the addNotRemove parameter.
//
// Example:
//
//	Button(
//	    Class("like-btn"),
//	    OnClick(handleLike),
//	    optimistic.Class("liked", true),  // Add "liked" class on click
//	)
func Class(class string, addNotRemove bool) vdom.Attr {
	action := ActionRemove
	if addNotRemove {
		action = ActionAdd
	}
	return vdom.Attr{
		Key: "_optimistic",
		Value: render.OptimisticConfig{
			Class: fmt.Sprintf("%s:%s", class, action),
		},
	}
}

// ClassToggle creates an optimistic class toggle attribute.
// When the element is clicked, the class is toggled (added if absent, removed if present).
//
// Example:
//
//	Button(
//	    Class("menu-toggle"),
//	    OnClick(handleToggle),
//	    optimistic.ClassToggle("active"),
//	)
func ClassToggle(class string) vdom.Attr {
	return vdom.Attr{
		Key: "_optimistic",
		Value: render.OptimisticConfig{
			Class: fmt.Sprintf("%s:%s", class, ActionToggle),
		},
	}
}

// Text creates an optimistic text change attribute.
// When the element is clicked, the text content is immediately changed
// to the specified value.
//
// Example:
//
//	Button(
//	    Text("Save"),
//	    OnClick(handleSave),
//	    optimistic.Text("Saving..."),
//	)
func Text(newText string) vdom.Attr {
	return vdom.Attr{
		Key: "_optimistic",
		Value: render.OptimisticConfig{
			Text: newText,
		},
	}
}

// Attr creates an optimistic attribute change.
// When the element is clicked, the attribute is immediately set to the new value.
//
// Example:
//
//	Input(
//	    Type("checkbox"),
//	    Checked(isChecked),
//	    OnChange(handleChange),
//	    optimistic.Attr("checked", "true"),
//	)
func Attr(name, attrValue string) vdom.Attr {
	return vdom.Attr{
		Key: "_optimistic",
		Value: render.OptimisticConfig{
			Attr:  name,
			Value: attrValue,
		},
	}
}

// AttrRemove creates an optimistic attribute removal.
// When the element is clicked, the attribute is immediately removed.
//
// Example:
//
//	Button(
//	    Disabled(true),
//	    OnClick(handleEnable),
//	    optimistic.AttrRemove("disabled"),
//	)
func AttrRemove(name string) vdom.Attr {
	return vdom.Attr{
		Key: "_optimistic",
		Value: render.OptimisticConfig{
			Attr:  name,
			Value: "", // Empty value signals removal
		},
	}
}

// ParentClass creates an optimistic class toggle on the parent element.
// Useful for dropdown menus, accordions, and similar patterns.
//
// Example:
//
//	Button(
//	    Text("Toggle Menu"),
//	    OnClick(handleMenu),
//	    OptimisticParentClass("open", true),
//	)
func ParentClass(class string, addNotRemove bool) vdom.Attr {
	action := ActionRemove
	if addNotRemove {
		action = ActionAdd
	}
	return vdom.Attr{
		Key:   "data-optimistic-parent-class",
		Value: fmt.Sprintf("%s:%s", class, action),
	}
}

// ParentClassToggle creates an optimistic class toggle on the parent element.
//
// Example:
//
//	Button(
//	    Text("Expand"),
//	    OnClick(handleExpand),
//	    OptimisticParentClassToggle("expanded"),
//	)
func ParentClassToggle(class string) vdom.Attr {
	return vdom.Attr{
		Key:   "data-optimistic-parent-class",
		Value: fmt.Sprintf("%s:%s", class, ActionToggle),
	}
}

// Disable creates an optimistic disable attribute.
// When the element is clicked, it's immediately disabled to prevent double-clicks.
// The server response will restore the correct state.
//
// Example:
//
//	Button(
//	    Text("Submit"),
//	    OnClick(handleSubmit),
//	    OptimisticDisable(),
//	)
func Disable() vdom.Attr {
	return vdom.Attr{
		Key:   "data-optimistic-disable",
		Value: "true",
	}
}

// Hide creates an optimistic hide attribute.
// When the element is clicked, it's immediately hidden (display: none).
//
// Example:
//
//	Button(
//	    Text("Dismiss"),
//	    OnClick(handleDismiss),
//	    OptimisticHide(),
//	)
func Hide() vdom.Attr {
	return vdom.Attr{
		Key:   "data-optimistic-hide",
		Value: "true",
	}
}

// Show creates an optimistic show attribute.
// When the element is clicked, it removes display: none.
//
// Example:
//
//	Button(
//	    Text("Show Details"),
//	    OnClick(handleShow),
//	    OptimisticShow(),
//	)
func Show() vdom.Attr {
	return vdom.Attr{
		Key:   "data-optimistic-show",
		Value: "true",
	}
}

// Loading creates a loading state optimistic update.
// Adds a "loading" class and optionally disables the element.
//
// Example:
//
//	Button(
//	    Text("Load More"),
//	    OnClick(handleLoad),
//	    OptimisticLoading(true),
//	)
func Loading(disable bool) vdom.Attr {
	value := "class"
	if disable {
		value = "class,disable"
	}
	return vdom.Attr{
		Key:   "data-optimistic-loading",
		Value: value,
	}
}

// Multiple combines multiple optimistic updates into a single attribute.
// This is useful when you need to apply several changes at once.
//
// Example:
//
//	Button(
//	    Text("Submit"),
//	    OnClick(handleSubmit),
//	    OptimisticMultiple(
//	        OptimisticClass("submitting", true),
//	        OptimisticText("Submitting..."),
//	        OptimisticDisable(),
//	    ),
//	)
func Multiple(attrs ...vdom.Attr) vdom.Attr {
	var parts []string
	for _, attr := range attrs {
		if attr.Key != "" && attr.Value != nil {
			parts = append(parts, fmt.Sprintf("%s=%v", attr.Key, attr.Value))
		}
	}
	return vdom.Attr{
		Key:   "data-optimistic-multi",
		Value: strings.Join(parts, ";"),
	}
}

// Target specifies which element should receive the optimistic update.
// By default, updates apply to the clicked element. This allows targeting
// a different element by its data-hid or a CSS selector.
//
// Example:
//
//	Button(
//	    Text("Update Header"),
//	    OnClick(handleUpdate),
//	    OptimisticTarget("#header"),
//	    OptimisticClass("updated", true),
//	)
func Target(selector string) vdom.Attr {
	return vdom.Attr{
		Key:   "data-optimistic-target",
		Value: selector,
	}
}

// Swap creates an optimistic content swap.
// The inner HTML is replaced with the specified content.
// Use with caution as this can introduce XSS if content is user-provided.
//
// Example:
//
//	Div(
//	    ID("content"),
//	    OnClick(handleSwap),
//	    OptimisticSwap("<span class='loading'>Loading...</span>"),
//	)
func Swap(html string) vdom.Attr {
	return vdom.Attr{
		Key:   "data-optimistic-swap",
		Value: html,
	}
}

// ParseClassAction parses a class action string like "liked:add" or "active:toggle".
// Returns the class name, action, and whether parsing succeeded.
func ParseClassAction(value string) (class string, action Action, ok bool) {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	class = parts[0]
	switch parts[1] {
	case "add":
		action = ActionAdd
	case "remove":
		action = ActionRemove
	case "toggle":
		action = ActionToggle
	case "set":
		action = ActionSet
	default:
		return "", "", false
	}
	return class, action, true
}

// ParseAttrAction parses an attribute action string like "disabled:true".
// Returns the attribute name, value, and whether parsing succeeded.
func ParseAttrAction(value string) (name, attrValue string, ok bool) {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// ParseMultiple parses a multiple action string.
// Returns a slice of key-value pairs.
func ParseMultiple(value string) []struct{ Key, Value string } {
	var result []struct{ Key, Value string }
	parts := strings.Split(value, ";")
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			result = append(result, struct{ Key, Value string }{kv[0], kv[1]})
		}
	}
	return result
}
