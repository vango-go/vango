package render

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/vango-dev/vango/v2/pkg/vdom"
)

// PageData contains all data needed to render a complete HTML page.
type PageData struct {
	// Body is the root VNode for the page content
	Body *vdom.VNode

	// Title is the page title
	Title string

	// Meta contains meta tags for the page
	Meta []MetaTag

	// Links contains link tags (stylesheets, favicon, etc.)
	Links []LinkTag

	// Scripts contains script tags to include
	Scripts []ScriptTag

	// Styles contains inline CSS styles
	Styles []string

	// SessionID is the session identifier for WebSocket reconnection
	SessionID string

	// CSRFToken is the CSRF token for form submissions and WebSocket
	CSRFToken string

	// ClientScript is the path to the thin client JavaScript
	// Defaults to "/_vango/client.js" if not specified
	ClientScript string

	// StyleSheets contains paths to external stylesheets
	StyleSheets []string

	// Lang is the language attribute for the html element
	// Defaults to "en" if not specified
	Lang string
}

// MetaTag represents a meta element in the document head.
type MetaTag struct {
	Name      string // name attribute
	Content   string // content attribute
	Property  string // property attribute (for OpenGraph)
	HTTPEquiv string // http-equiv attribute
	Charset   string // charset attribute
}

// LinkTag represents a link element in the document head.
type LinkTag struct {
	Rel         string // rel attribute
	Href        string // href attribute
	Type        string // type attribute
	Sizes       string // sizes attribute
	CrossOrigin string // crossorigin attribute
	Media       string // media attribute
}

// ScriptTag represents a script element.
type ScriptTag struct {
	Src    string // src attribute
	Type   string // type attribute
	Defer  bool   // defer attribute
	Async  bool   // async attribute
	Module bool   // type="module"
	Inline string // inline script content
}

// HookConfig contains configuration for a client-side hook.
type HookConfig struct {
	Name   string         // Hook name (e.g., "Sortable", "Tooltip")
	Config map[string]any // Hook-specific configuration
}

// OptimisticConfig contains configuration for optimistic updates.
type OptimisticConfig struct {
	Class string // Class to add/remove optimistically
	Text  string // Text to show optimistically
	Attr  string // Attribute to modify optimistically
	Value string // Attribute value
}

// RenderPage renders a complete HTML document to the given writer.
func (r *Renderer) RenderPage(w io.Writer, page PageData) error {
	// Set default language
	lang := page.Lang
	if lang == "" {
		lang = "en"
	}

	// DOCTYPE
	if _, err := w.Write([]byte("<!DOCTYPE html>\n")); err != nil {
		return err
	}

	// HTML tag with lang
	if _, err := fmt.Fprintf(w, `<html lang="%s">`+"\n", escapeAttr(lang)); err != nil {
		return err
	}

	// Head
	if err := r.renderHead(w, page); err != nil {
		return err
	}

	// Body
	if _, err := w.Write([]byte("<body>\n")); err != nil {
		return err
	}

	// Main content
	if err := r.RenderToWriter(w, page.Body); err != nil {
		return err
	}

	// Inject Vango client script
	if err := r.renderClientScript(w, page); err != nil {
		return err
	}

	// Close body and html
	if _, err := w.Write([]byte("</body>\n</html>\n")); err != nil {
		return err
	}

	return nil
}

// renderHead renders the document head section.
func (r *Renderer) renderHead(w io.Writer, page PageData) error {
	if _, err := w.Write([]byte("<head>\n")); err != nil {
		return err
	}

	// Charset
	if _, err := w.Write([]byte(`  <meta charset="utf-8">` + "\n")); err != nil {
		return err
	}

	// Viewport
	if _, err := w.Write([]byte(`  <meta name="viewport" content="width=device-width, initial-scale=1">` + "\n")); err != nil {
		return err
	}

	// Title
	if page.Title != "" {
		if _, err := fmt.Fprintf(w, "  <title>%s</title>\n", escapeHTML(page.Title)); err != nil {
			return err
		}
	}

	// Meta tags
	for _, meta := range page.Meta {
		if err := r.renderMetaTag(w, meta); err != nil {
			return err
		}
	}

	// Link tags (stylesheets, favicon, etc.)
	for _, link := range page.Links {
		if err := r.renderLinkTag(w, link); err != nil {
			return err
		}
	}

	// Stylesheets
	for _, href := range page.StyleSheets {
		if _, err := fmt.Fprintf(w, `  <link rel="stylesheet" href="%s">`+"\n", escapeAttr(href)); err != nil {
			return err
		}
	}

	// Inline styles
	for _, style := range page.Styles {
		if _, err := fmt.Fprintf(w, "  <style>%s</style>\n", style); err != nil {
			return err
		}
	}

	// Scripts in head (defer/async)
	for _, script := range page.Scripts {
		if script.Defer || script.Async {
			if err := r.renderScriptTag(w, script); err != nil {
				return err
			}
		}
	}

	if _, err := w.Write([]byte("</head>\n")); err != nil {
		return err
	}

	return nil
}

// renderMetaTag renders a meta element.
func (r *Renderer) renderMetaTag(w io.Writer, meta MetaTag) error {
	if _, err := w.Write([]byte("  <meta")); err != nil {
		return err
	}

	if meta.Charset != "" {
		if _, err := fmt.Fprintf(w, ` charset="%s"`, escapeAttr(meta.Charset)); err != nil {
			return err
		}
	}

	if meta.Name != "" {
		if _, err := fmt.Fprintf(w, ` name="%s"`, escapeAttr(meta.Name)); err != nil {
			return err
		}
	}

	if meta.Property != "" {
		if _, err := fmt.Fprintf(w, ` property="%s"`, escapeAttr(meta.Property)); err != nil {
			return err
		}
	}

	if meta.HTTPEquiv != "" {
		if _, err := fmt.Fprintf(w, ` http-equiv="%s"`, escapeAttr(meta.HTTPEquiv)); err != nil {
			return err
		}
	}

	if meta.Content != "" {
		if _, err := fmt.Fprintf(w, ` content="%s"`, escapeAttr(meta.Content)); err != nil {
			return err
		}
	}

	if _, err := w.Write([]byte(">\n")); err != nil {
		return err
	}

	return nil
}

// renderLinkTag renders a link element.
func (r *Renderer) renderLinkTag(w io.Writer, link LinkTag) error {
	if _, err := w.Write([]byte("  <link")); err != nil {
		return err
	}

	if link.Rel != "" {
		if _, err := fmt.Fprintf(w, ` rel="%s"`, escapeAttr(link.Rel)); err != nil {
			return err
		}
	}

	if link.Href != "" {
		if _, err := fmt.Fprintf(w, ` href="%s"`, escapeAttr(link.Href)); err != nil {
			return err
		}
	}

	if link.Type != "" {
		if _, err := fmt.Fprintf(w, ` type="%s"`, escapeAttr(link.Type)); err != nil {
			return err
		}
	}

	if link.Sizes != "" {
		if _, err := fmt.Fprintf(w, ` sizes="%s"`, escapeAttr(link.Sizes)); err != nil {
			return err
		}
	}

	if link.CrossOrigin != "" {
		if _, err := fmt.Fprintf(w, ` crossorigin="%s"`, escapeAttr(link.CrossOrigin)); err != nil {
			return err
		}
	}

	if link.Media != "" {
		if _, err := fmt.Fprintf(w, ` media="%s"`, escapeAttr(link.Media)); err != nil {
			return err
		}
	}

	if _, err := w.Write([]byte(">\n")); err != nil {
		return err
	}

	return nil
}

// renderScriptTag renders a script element.
func (r *Renderer) renderScriptTag(w io.Writer, script ScriptTag) error {
	if _, err := w.Write([]byte("  <script")); err != nil {
		return err
	}

	if script.Src != "" {
		if _, err := fmt.Fprintf(w, ` src="%s"`, escapeAttr(script.Src)); err != nil {
			return err
		}
	}

	if script.Module {
		if _, err := w.Write([]byte(` type="module"`)); err != nil {
			return err
		}
	} else if script.Type != "" {
		if _, err := fmt.Fprintf(w, ` type="%s"`, escapeAttr(script.Type)); err != nil {
			return err
		}
	}

	if script.Defer {
		if _, err := w.Write([]byte(" defer")); err != nil {
			return err
		}
	}

	if script.Async {
		if _, err := w.Write([]byte(" async")); err != nil {
			return err
		}
	}

	if _, err := w.Write([]byte(">")); err != nil {
		return err
	}

	if script.Inline != "" {
		if _, err := w.Write([]byte(script.Inline)); err != nil {
			return err
		}
	}

	if _, err := w.Write([]byte("</script>\n")); err != nil {
		return err
	}

	return nil
}

// renderClientScript injects the Vango thin client and configuration.
func (r *Renderer) renderClientScript(w io.Writer, page PageData) error {
	// CSRF token for WebSocket handshake
	if page.CSRFToken != "" {
		if _, err := fmt.Fprintf(w, `  <script>window.__VANGO_CSRF__="%s";</script>`+"\n",
			escapeAttr(page.CSRFToken)); err != nil {
			return err
		}
	}

	// Session ID for reconnection
	if page.SessionID != "" {
		if _, err := fmt.Fprintf(w, `  <script>window.__VANGO_SESSION__="%s";</script>`+"\n",
			escapeAttr(page.SessionID)); err != nil {
			return err
		}
	}

	// Thin client script
	clientPath := page.ClientScript
	if clientPath == "" {
		clientPath = "/_vango/client.js"
	}

	// Enable debug mode for development
	if _, err := fmt.Fprintf(w, `  <script src="%s" data-debug="true" defer></script>`+"\n",
		escapeAttr(clientPath)); err != nil {
		return err
	}

	return nil
}

// renderHookConfig renders hook configuration as data attributes.
// This is called internally by renderAttributes when _hook prop is present.
func renderHookConfig(w io.Writer, hookConfig HookConfig) error {
	if _, err := fmt.Fprintf(w, ` data-hook="%s"`, escapeAttr(hookConfig.Name)); err != nil {
		return err
	}

	if len(hookConfig.Config) > 0 {
		configJSON, err := json.Marshal(hookConfig.Config)
		if err != nil {
			return fmt.Errorf("failed to marshal hook config: %w", err)
		}
		if _, err := fmt.Fprintf(w, ` data-hook-config='%s'`, string(configJSON)); err != nil {
			return err
		}
	}

	return nil
}

// renderOptimisticConfig renders optimistic update configuration as data attributes.
// This is called internally by renderAttributes when _optimistic prop is present.
func renderOptimisticConfig(w io.Writer, config OptimisticConfig) error {
	if config.Class != "" {
		if _, err := fmt.Fprintf(w, ` data-optimistic-class="%s"`, escapeAttr(config.Class)); err != nil {
			return err
		}
	}

	if config.Text != "" {
		if _, err := fmt.Fprintf(w, ` data-optimistic-text="%s"`, escapeAttr(config.Text)); err != nil {
			return err
		}
	}

	if config.Attr != "" {
		if _, err := fmt.Fprintf(w, ` data-optimistic-attr="%s"`, escapeAttr(config.Attr)); err != nil {
			return err
		}
		if config.Value != "" {
			if _, err := fmt.Fprintf(w, ` data-optimistic-value="%s"`, escapeAttr(config.Value)); err != nil {
				return err
			}
		}
	}

	return nil
}
