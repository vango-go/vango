package render

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/vango-dev/vango/v2/pkg/vdom"
)

// RendererConfig configures the HTML renderer.
type RendererConfig struct {
	// Pretty enables pretty-printed HTML output with indentation.
	// Should only be used in development as it increases output size.
	Pretty bool

	// Indent is the string used for each indentation level in pretty mode.
	// Defaults to two spaces if not specified.
	Indent string

	// IncludeCSRF indicates whether to include CSRF token in the output.
	IncludeCSRF bool

	// AssetPath is the base path for assets.
	AssetPath string

	// InlineCriticalCSS indicates whether to inline critical CSS.
	InlineCriticalCSS bool
}

// Renderer handles server-side rendering of VNode trees to HTML.
type Renderer struct {
	config     RendererConfig
	hidCounter uint32
	handlers   map[string]any
}

// NewRenderer creates a new Renderer with the given configuration.
func NewRenderer(config RendererConfig) *Renderer {
	if config.Indent == "" {
		config.Indent = "  "
	}
	return &Renderer{
		config:   config,
		handlers: make(map[string]any),
	}
}

// RenderToString renders a VNode tree to a complete HTML string.
func (r *Renderer) RenderToString(node *vdom.VNode) (string, error) {
	var buf bytes.Buffer
	if err := r.RenderToWriter(&buf, node); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderToWriter streams a VNode tree to the given writer.
func (r *Renderer) RenderToWriter(w io.Writer, node *vdom.VNode) error {
	return r.renderNode(w, node, 0)
}

// GetHandlers returns the handler registry collected during rendering.
// The map keys are in the format "hid_eventname" (e.g., "h1_onclick").
func (r *Renderer) GetHandlers() map[string]any {
	return r.handlers
}

// Reset resets the renderer state for reuse.
// This clears the HID counter and handler registry.
func (r *Renderer) Reset() {
	r.hidCounter = 0
	r.handlers = make(map[string]any)
}

// renderNode dispatches rendering based on node kind.
func (r *Renderer) renderNode(w io.Writer, node *vdom.VNode, depth int) error {
	if node == nil {
		return nil
	}

	switch node.Kind {
	case vdom.KindElement:
		return r.renderElement(w, node, depth)
	case vdom.KindText:
		return r.renderText(w, node)
	case vdom.KindFragment:
		return r.renderFragment(w, node, depth)
	case vdom.KindComponent:
		return r.renderComponent(w, node, depth)
	case vdom.KindRaw:
		return r.renderRaw(w, node)
	default:
		return fmt.Errorf("unknown node kind: %d", node.Kind)
	}
}

// renderElement renders an HTML element with its attributes and children.
func (r *Renderer) renderElement(w io.Writer, node *vdom.VNode, depth int) error {
	tag := node.Tag

	// Indentation (if pretty printing)
	if r.config.Pretty && depth > 0 {
		r.writeIndent(w, depth)
	}

	// Opening tag
	if _, err := w.Write([]byte{'<'}); err != nil {
		return err
	}
	if _, err := w.Write([]byte(tag)); err != nil {
		return err
	}

	// Render attributes
	if err := r.renderAttributes(w, node); err != nil {
		return err
	}

	// Check if this element needs a hydration ID
	if r.needsHID(node) {
		hid := r.nextHID()
		node.HID = hid
		// Debug: trace HID assignment with tag
		fmt.Printf("[SSR HID] %s -> %s\n", hid, tag)
		if _, err := fmt.Fprintf(w, ` data-hid="%s"`, hid); err != nil {
			return err
		}
		// Store handler references
		r.registerHandlers(hid, node)
	}

	// Self-closing check for void elements
	if isVoidElement(tag) {
		if _, err := w.Write([]byte{'>'}); err != nil {
			return err
		}
		if r.config.Pretty {
			w.Write([]byte{'\n'})
		}
		return nil
	}

	if _, err := w.Write([]byte{'>'}); err != nil {
		return err
	}

	// Handle dangerouslySetInnerHTML
	if rawHTML, ok := node.Props["dangerouslySetInnerHTML"].(string); ok {
		if _, err := w.Write([]byte(rawHTML)); err != nil {
			return err
		}
	} else {
		// Newline after opening tag if has children and pretty printing
		hasBlockChildren := len(node.Children) > 0 && !isInlineElement(tag)
		if r.config.Pretty && hasBlockChildren {
			w.Write([]byte{'\n'})
		}

		// Render children
		for _, child := range node.Children {
			if err := r.renderNode(w, child, depth+1); err != nil {
				return err
			}
		}

		// Closing tag indentation
		if r.config.Pretty && hasBlockChildren {
			r.writeIndent(w, depth)
		}
	}

	// Closing tag
	if _, err := fmt.Fprintf(w, "</%s>", tag); err != nil {
		return err
	}
	if r.config.Pretty {
		w.Write([]byte{'\n'})
	}

	return nil
}

// renderText renders a text node with HTML escaping.
func (r *Renderer) renderText(w io.Writer, node *vdom.VNode) error {
	escaped := escapeHTML(node.Text)
	_, err := w.Write([]byte(escaped))
	return err
}

// renderFragment renders a fragment's children without a wrapper element.
func (r *Renderer) renderFragment(w io.Writer, node *vdom.VNode, depth int) error {
	for _, child := range node.Children {
		if err := r.renderNode(w, child, depth); err != nil {
			return err
		}
	}
	return nil
}

// renderComponent renders a component by rendering its output VNode.
func (r *Renderer) renderComponent(w io.Writer, node *vdom.VNode, depth int) error {
	// If the component has already been rendered to a VNode, render that
	if node.Comp != nil {
		output := node.Comp.Render()
		return r.renderNode(w, output, depth)
	}
	return nil
}

// renderRaw renders raw HTML without escaping.
func (r *Renderer) renderRaw(w io.Writer, node *vdom.VNode) error {
	_, err := w.Write([]byte(node.Text))
	return err
}

// renderAttributes renders all attributes for an element.
func (r *Renderer) renderAttributes(w io.Writer, node *vdom.VNode) error {
	if node.Props == nil {
		return nil
	}

	// Sort keys for deterministic output
	keys := make([]string, 0, len(node.Props))
	for key := range node.Props {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := node.Props[key]

		// Skip internal props
		if strings.HasPrefix(key, "_") {
			continue
		}

		// Skip event handlers (they're registered, not rendered as attributes)
		if strings.HasPrefix(key, "on") && isEventHandler(value) {
			continue
		}

		// Handle special attributes
		switch key {
		case "className":
			key = "class"
		case "htmlFor":
			key = "for"
		case "dangerouslySetInnerHTML":
			// Handled separately in renderElement
			continue
		case "key":
			// Key is internal, not rendered
			continue
		}

		// Boolean attributes
		if isBooleanAttr(key) {
			if boolValue, ok := value.(bool); ok {
				if boolValue {
					if _, err := fmt.Fprintf(w, " %s", key); err != nil {
						return err
					}
				}
				continue
			}
		}

		// Regular attributes
		strValue := attrToString(value)
		if strValue != "" {
			escaped := escapeAttr(strValue)
			if _, err := fmt.Fprintf(w, ` %s="%s"`, key, escaped); err != nil {
				return err
			}
		}
	}

	// Add event marker attributes (for client-side binding)
	for _, key := range keys {
		if strings.HasPrefix(key, "on") && isEventHandler(node.Props[key]) {
			eventName := strings.ToLower(key[2:]) // onclick -> click
			if _, err := fmt.Fprintf(w, ` data-on-%s="true"`, eventName); err != nil {
				return err
			}
		}
	}

	return nil
}

// needsHID returns true if the element needs a hydration ID.
// This must match the logic in vdom.AssignHIDs to ensure consistency
// between SSR-rendered HTML and WebSocket session handlers.
func (r *Renderer) needsHID(node *vdom.VNode) bool {
	if node.Kind != vdom.KindElement {
		return false
	}

	// Check for event handlers - matches vdom.IsInteractive()
	// Just check key prefix, not value type, for consistency with WebSocket
	// Assign HIDs to all elements to match vdom.AssignHIDs behavior.
	// This ensures consistency between SSR and WebSocket logic.
	return true
}

// nextHID generates the next sequential hydration ID.
func (r *Renderer) nextHID() string {
	r.hidCounter++
	hid := fmt.Sprintf("h%d", r.hidCounter)
	// Debug: trace HID assignment
	fmt.Printf("[SSR HID] %s\n", hid)
	return hid
}

// registerHandlers stores handler references for the given HID.
func (r *Renderer) registerHandlers(hid string, node *vdom.VNode) {
	for key, value := range node.Props {
		if strings.HasPrefix(key, "on") && isEventHandler(value) {
			r.handlers[hid+"_"+key] = value
		}
	}
}

// isEventHandler returns true if the value looks like an event handler.
func isEventHandler(value any) bool {
	if value == nil {
		return false
	}
	// Check for common handler types
	switch value.(type) {
	case func():
		return true
	case func(any):
		return true
	case vdom.EventHandler:
		return true
	default:
		// Use reflection to check for function types
		return strings.HasPrefix(fmt.Sprintf("%T", value), "func")
	}
}

// attrToString converts an attribute value to a string.
func attrToString(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%g", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// writeIndent writes indentation for pretty printing.
func (r *Renderer) writeIndent(w io.Writer, depth int) {
	for i := 0; i < depth; i++ {
		w.Write([]byte(r.config.Indent))
	}
}
