# Phase 6: Server-Side Rendering & Hydration ✅ COMPLETE

> **Render components to HTML on the server, make them interactive on the client**

**Status**: Complete (2024-12-07)

---

## Overview

SSR is the process of rendering Vango components to HTML strings on the server, sending them to the browser, and then "hydrating" them with event handlers via the thin client. This provides:

1. **Fast First Paint**: Users see content immediately (no JS required)
2. **SEO**: Search engines can index the full content
3. **Progressive Enhancement**: Page works without JS
4. **Reduced Client Load**: No component rendering on client

### Design Principles

1. **Streaming by Default**: Render to `io.Writer` for low memory
2. **Minimal Hydration**: Only add `data-hid` to interactive elements
3. **Security First**: All text content escaped by default
4. **Standard HTML5**: Full spec compliance

### Non-Goals

1. Partial hydration (future consideration)
2. Selective SSR (all pages are SSR by default)
3. Client-side rendering fallback

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    SSR Pipeline                                 │
│                                                                 │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────┐  │
│  │   Route     │───▶│  Component  │───▶│    VNode Tree       │  │
│  │   Match     │    │   Render    │    │                     │  │
│  └─────────────┘    └─────────────┘    └──────────┬──────────┘  │
│                                                    │            │
│                                        ┌───────────▼───────────┐│
│                                        │    HTML Renderer      ││
│                                        │  ┌─────────────────┐  ││
│                                        │  │  HID Generator  │  ││
│                                        │  ├─────────────────┤  ││
│                                        │  │  Text Escaper   │  ││
│                                        │  ├─────────────────┤  ││
│                                        │  │  Attr Renderer  │  ││
│                                        │  └─────────────────┘  ││
│                                        └───────────┬───────────┘│
│                                                    │            │
│                                        ┌───────────▼───────────┐│
│                                        │   HTML + Script Tag   ││
│                                        │   (Thin Client)       ││
│                                        └───────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

---

## Core Types

### Renderer

```go
// Renderer handles SSR of VNode trees
type Renderer struct {
    // Configuration
    config RendererConfig

    // HID counter (reset per request)
    hidCounter uint32

    // Handler registry (for session binding)
    handlers map[string]func()
}

type RendererConfig struct {
    // Pretty print HTML (development only)
    Pretty bool

    // Indent string for pretty printing
    Indent string

    // Include CSRF token in output
    IncludeCSRF bool

    // Base path for assets
    AssetPath string

    // Inline critical CSS
    InlineCriticalCSS bool
}

// New creates a new renderer
func NewRenderer(config RendererConfig) *Renderer

// RenderToString renders VNode to complete HTML string
func (r *Renderer) RenderToString(node *VNode) (string, error)

// RenderToWriter streams VNode to writer
func (r *Renderer) RenderToWriter(w io.Writer, node *VNode) error

// RenderPage renders a complete HTML document
func (r *Renderer) RenderPage(w io.Writer, page PageData) error

// GetHandlers returns the handler registry after rendering
func (r *Renderer) GetHandlers() map[string]func()

// Reset resets the renderer state for reuse
func (r *Renderer) Reset()
```

### PageData

```go
// PageData contains all data needed to render a complete page
type PageData struct {
    // The root VNode (body content)
    Body *VNode

    // Document head elements
    Title       string
    Meta        []MetaTag
    Links       []LinkTag
    Scripts     []ScriptTag
    Styles      []string

    // Session info
    SessionID   string
    CSRFToken   string

    // Assets
    ClientScript string  // Path to thin client JS
    StyleSheets  []string
}

type MetaTag struct {
    Name       string
    Content    string
    Property   string  // For OpenGraph
    HTTPEquiv  string
}

type LinkTag struct {
    Rel        string
    Href       string
    Type       string
    Sizes      string
    CrossOrigin string
}

type ScriptTag struct {
    Src        string
    Type       string
    Defer      bool
    Async      bool
    Module     bool
    Inline     string  // For inline scripts
}
```

---

## HTML Rendering

### Main Render Loop

```go
func (r *Renderer) RenderToWriter(w io.Writer, node *VNode) error {
    return r.renderNode(w, node, 0)
}

func (r *Renderer) renderNode(w io.Writer, node *VNode, depth int) error {
    if node == nil {
        return nil
    }

    switch node.Kind {
    case KindElement:
        return r.renderElement(w, node, depth)
    case KindText:
        return r.renderText(w, node)
    case KindFragment:
        return r.renderFragment(w, node, depth)
    case KindComponent:
        return r.renderComponent(w, node, depth)
    default:
        return fmt.Errorf("unknown node kind: %d", node.Kind)
    }
}

func (r *Renderer) renderElement(w io.Writer, node *VNode, depth int) error {
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
        if _, err := fmt.Fprintf(w, ` data-hid="%s"`, hid); err != nil {
            return err
        }
        // Store handler reference
        r.registerHandlers(hid, node)
    }

    // Self-closing check
    if isVoidElement(tag) {
        if _, err := w.Write([]byte(">")); err != nil {
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

    // Newline after opening tag if has children and pretty printing
    if r.config.Pretty && len(node.Children) > 0 && !isInlineElement(tag) {
        w.Write([]byte{'\n'})
    }

    // Render children
    for _, child := range node.Children {
        if err := r.renderNode(w, child, depth+1); err != nil {
            return err
        }
    }

    // Closing tag
    if r.config.Pretty && len(node.Children) > 0 && !isInlineElement(tag) {
        r.writeIndent(w, depth)
    }
    if _, err := fmt.Fprintf(w, "</%s>", tag); err != nil {
        return err
    }
    if r.config.Pretty {
        w.Write([]byte{'\n'})
    }

    return nil
}

func (r *Renderer) renderText(w io.Writer, node *VNode) error {
    escaped := escapeHTML(node.Text)
    _, err := w.Write([]byte(escaped))
    return err
}

func (r *Renderer) renderFragment(w io.Writer, node *VNode, depth int) error {
    for _, child := range node.Children {
        if err := r.renderNode(w, child, depth); err != nil {
            return err
        }
    }
    return nil
}

func (r *Renderer) renderComponent(w io.Writer, node *VNode, depth int) error {
    // Components are pre-rendered to VNode during component execution
    // This should render the component's output VNode
    if node.ComponentOutput != nil {
        return r.renderNode(w, node.ComponentOutput, depth)
    }
    return nil
}
```

### Attribute Rendering

```go
func (r *Renderer) renderAttributes(w io.Writer, node *VNode) error {
    for key, value := range node.Props {
        // Skip internal props
        if strings.HasPrefix(key, "_") {
            continue
        }

        // Skip event handlers (they're not rendered, just registered)
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
            // Handled separately
            continue
        }

        // Boolean attributes
        if isBooleanAttr(key) {
            if boolValue, ok := value.(bool); ok {
                if boolValue {
                    if _, err := fmt.Fprintf(w, ` %s`, key); err != nil {
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
    for key := range node.Props {
        if strings.HasPrefix(key, "on") && isEventHandler(node.Props[key]) {
            eventName := strings.ToLower(key[2:]) // onClick -> click
            if _, err := fmt.Fprintf(w, ` data-on-%s="true"`, eventName); err != nil {
                return err
            }
        }
    }

    return nil
}

// Boolean attributes that don't need a value
var booleanAttrs = map[string]bool{
    "disabled":        true,
    "checked":         true,
    "selected":        true,
    "readonly":        true,
    "required":        true,
    "multiple":        true,
    "autofocus":       true,
    "autoplay":        true,
    "controls":        true,
    "loop":            true,
    "muted":           true,
    "default":         true,
    "defer":           true,
    "async":           true,
    "hidden":          true,
    "open":            true,
    "novalidate":      true,
    "formnovalidate":  true,
    "allowfullscreen": true,
}

func isBooleanAttr(name string) bool {
    return booleanAttrs[name]
}
```

### Void Elements

```go
// Void elements cannot have children and are self-closing
var voidElements = map[string]bool{
    "area":    true,
    "base":    true,
    "br":      true,
    "col":     true,
    "embed":   true,
    "hr":      true,
    "img":     true,
    "input":   true,
    "link":    true,
    "meta":    true,
    "param":   true,
    "source":  true,
    "track":   true,
    "wbr":     true,
}

func isVoidElement(tag string) bool {
    return voidElements[tag]
}

// Inline elements (for pretty printing)
var inlineElements = map[string]bool{
    "a":       true,
    "abbr":    true,
    "b":       true,
    "code":    true,
    "em":      true,
    "i":       true,
    "kbd":     true,
    "span":    true,
    "strong":  true,
    "sub":     true,
    "sup":     true,
}

func isInlineElement(tag string) bool {
    return inlineElements[tag]
}
```

---

## Hydration ID Generation

### HID Strategy

Every element that needs client-side interaction gets a unique hydration ID:

```go
func (r *Renderer) needsHID(node *VNode) bool {
    if node.Kind != KindElement {
        return false
    }

    // Check for event handlers
    for key := range node.Props {
        if strings.HasPrefix(key, "on") && isEventHandler(node.Props[key]) {
            return true
        }
    }

    // Check for hooks
    if _, hasHook := node.Props["_hook"]; hasHook {
        return true
    }

    // Check for refs
    if _, hasRef := node.Props["_ref"]; hasRef {
        return true
    }

    // Check for optimistic updates
    if _, hasOptimistic := node.Props["_optimistic"]; hasOptimistic {
        return true
    }

    return false
}

func (r *Renderer) nextHID() string {
    r.hidCounter++
    return fmt.Sprintf("h%d", r.hidCounter)
}
```

### Handler Registration

```go
func (r *Renderer) registerHandlers(hid string, node *VNode) {
    for key, value := range node.Props {
        if strings.HasPrefix(key, "on") {
            if handler, ok := value.(func()); ok {
                r.handlers[hid+"_"+key] = handler
            }
            // Also handle handlers with event parameter
            if handler, ok := value.(func(Event)); ok {
                r.handlers[hid+"_"+key] = func() {
                    // Event will be populated at runtime
                    handler(Event{})
                }
            }
        }
    }
}
```

---

## Text Escaping

### HTML Escaping

```go
// escapeHTML escapes text for safe inclusion in HTML
func escapeHTML(s string) string {
    var buf strings.Builder
    buf.Grow(len(s))

    for _, r := range s {
        switch r {
        case '&':
            buf.WriteString("&amp;")
        case '<':
            buf.WriteString("&lt;")
        case '>':
            buf.WriteString("&gt;")
        case '"':
            buf.WriteString("&quot;")
        case '\'':
            buf.WriteString("&#39;")
        default:
            buf.WriteRune(r)
        }
    }

    return buf.String()
}

// escapeAttr escapes text for safe inclusion in attribute values
func escapeAttr(s string) string {
    // Same as escapeHTML but also handles line breaks
    var buf strings.Builder
    buf.Grow(len(s))

    for _, r := range s {
        switch r {
        case '&':
            buf.WriteString("&amp;")
        case '<':
            buf.WriteString("&lt;")
        case '>':
            buf.WriteString("&gt;")
        case '"':
            buf.WriteString("&quot;")
        case '\'':
            buf.WriteString("&#39;")
        case '\n':
            buf.WriteString("&#10;")
        case '\r':
            buf.WriteString("&#13;")
        case '\t':
            buf.WriteString("&#9;")
        default:
            buf.WriteRune(r)
        }
    }

    return buf.String()
}
```

### Raw HTML (Dangerous)

```go
func (r *Renderer) renderElement(w io.Writer, node *VNode, depth int) error {
    // ... existing code ...

    // Handle dangerouslySetInnerHTML
    if rawHTML, ok := node.Props["dangerouslySetInnerHTML"].(string); ok {
        // Write the raw HTML without escaping
        if _, err := w.Write([]byte(rawHTML)); err != nil {
            return err
        }
    } else {
        // Render children normally
        for _, child := range node.Children {
            if err := r.renderNode(w, child, depth+1); err != nil {
                return err
            }
        }
    }

    // ... closing tag ...
}
```

---

## Full Document Rendering

### Page Template

```go
func (r *Renderer) RenderPage(w io.Writer, page PageData) error {
    // DOCTYPE
    if _, err := w.Write([]byte("<!DOCTYPE html>\n")); err != nil {
        return err
    }

    // HTML tag with lang
    if _, err := w.Write([]byte(`<html lang="en">` + "\n")); err != nil {
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

func (r *Renderer) renderMetaTag(w io.Writer, meta MetaTag) error {
    if _, err := w.Write([]byte("  <meta")); err != nil {
        return err
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

    if _, err := w.Write([]byte(">\n")); err != nil {
        return err
    }

    return nil
}

func (r *Renderer) renderScriptTag(w io.Writer, script ScriptTag) error {
    if _, err := w.Write([]byte("  <script")); err != nil {
        return err
    }

    if script.Src != "" {
        if _, err := fmt.Fprintf(w, ` src="%s"`, escapeAttr(script.Src)); err != nil {
            return err
        }
    }

    if script.Type != "" {
        if _, err := fmt.Fprintf(w, ` type="%s"`, escapeAttr(script.Type)); err != nil {
            return err
        }
    }

    if script.Module {
        if _, err := w.Write([]byte(` type="module"`)); err != nil {
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
```

### Client Script Injection

```go
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

    if _, err := fmt.Fprintf(w, `  <script src="%s" defer></script>`+"\n",
        escapeAttr(clientPath)); err != nil {
        return err
    }

    return nil
}
```

---

## Streaming Renderer

For large pages, we can stream HTML as it's generated:

```go
// StreamingRenderer wraps Renderer with chunked output
type StreamingRenderer struct {
    *Renderer
    flusher http.Flusher
    w       io.Writer
}

func NewStreamingRenderer(w http.ResponseWriter, config RendererConfig) *StreamingRenderer {
    flusher, _ := w.(http.Flusher)
    return &StreamingRenderer{
        Renderer: NewRenderer(config),
        flusher:  flusher,
        w:        w,
    }
}

func (s *StreamingRenderer) RenderPage(page PageData) error {
    // Set headers for streaming
    if s.flusher != nil {
        // No content-length, enable chunked transfer
    }

    // Render head immediately
    if err := s.renderHead(s.w, page); err != nil {
        return err
    }

    // Flush head
    if s.flusher != nil {
        s.flusher.Flush()
    }

    // Render body
    if _, err := s.w.Write([]byte("<body>\n")); err != nil {
        return err
    }

    // Render content (could flush periodically for very large pages)
    if err := s.RenderToWriter(s.w, page.Body); err != nil {
        return err
    }

    // Final flush
    if err := s.renderClientScript(s.w, page); err != nil {
        return err
    }

    if _, err := s.w.Write([]byte("</body>\n</html>\n")); err != nil {
        return err
    }

    return nil
}
```

---

## Component Rendering Flow

### Rendering a Component

```go
// RenderComponent executes a component and returns its VNode output
func RenderComponent(ctx *Context, component Component) (*VNode, map[string]func(), error) {
    // Create owner for this component
    owner := newOwner(ctx.parentOwner)
    ctx.currentOwner = owner

    // Set up tracking context
    trackingCtx := newTrackingContext(owner)

    // Execute the component function
    var output *VNode
    withContext(trackingCtx, func() {
        output = component.Render(ctx)
    })

    // Run initial effects
    owner.runPendingEffects()

    // Create renderer and render to HTML
    renderer := NewRenderer(RendererConfig{})
    html, err := renderer.RenderToString(output)
    if err != nil {
        return nil, nil, err
    }

    return output, renderer.GetHandlers(), nil
}
```

### Full Page Request Flow

```go
func HandlePageRequest(w http.ResponseWriter, r *http.Request) {
    // Match route
    route, params := router.Match(r.URL.Path)
    if route == nil {
        handle404(w, r)
        return
    }

    // Create session
    session := sessionManager.Create(r)

    // Create request context
    ctx := &Context{
        Request:    r,
        Response:   w,
        Params:     params,
        Session:    session,
        SessionID:  session.ID,
        CSRFToken:  session.CSRFToken,
    }

    // Execute layout and page components
    body, handlers, err := renderPage(ctx, route)
    if err != nil {
        handle500(w, r, err)
        return
    }

    // Store handlers in session for event handling
    session.Handlers = handlers

    // Prepare page data
    page := PageData{
        Body:        body,
        Title:       ctx.Title,
        Meta:        ctx.Meta,
        SessionID:   session.ID,
        CSRFToken:   session.CSRFToken,
        StyleSheets: ctx.StyleSheets,
    }

    // Render complete page
    renderer := NewRenderer(RendererConfig{
        Pretty: isDevelopment(),
    })

    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    if err := renderer.RenderPage(w, page); err != nil {
        log.Printf("Error rendering page: %v", err)
    }
}

func renderPage(ctx *Context, route *Route) (*VNode, map[string]func(), error) {
    allHandlers := make(map[string]func())

    // Render layout if exists
    var layoutOutput *VNode
    if route.Layout != nil {
        layout, handlers, err := RenderComponent(ctx, route.Layout)
        if err != nil {
            return nil, nil, err
        }
        layoutOutput = layout
        for k, v := range handlers {
            allHandlers[k] = v
        }
    }

    // Render page component
    page, handlers, err := RenderComponent(ctx, route.Page)
    if err != nil {
        return nil, nil, err
    }
    for k, v := range handlers {
        allHandlers[k] = v
    }

    // Compose layout with page
    if layoutOutput != nil {
        // Find slot and insert page
        insertPageIntoLayout(layoutOutput, page)
        return layoutOutput, allHandlers, nil
    }

    return page, allHandlers, nil
}
```

---

## Hooks and Special Rendering

### Client Hooks

```go
func (r *Renderer) renderAttributes(w io.Writer, node *VNode) error {
    // ... existing attribute rendering ...

    // Render hook configuration
    if hookConfig, ok := node.Props["_hook"].(HookConfig); ok {
        if _, err := fmt.Fprintf(w, ` data-hook="%s"`, escapeAttr(hookConfig.Name)); err != nil {
            return err
        }
        if hookConfig.Config != nil {
            configJSON, _ := json.Marshal(hookConfig.Config)
            if _, err := fmt.Fprintf(w, ` data-hook-config='%s'`, string(configJSON)); err != nil {
                return err
            }
        }
    }

    // Render optimistic update configuration
    if optimistic, ok := node.Props["_optimistic"].(OptimisticConfig); ok {
        if optimistic.Class != "" {
            if _, err := fmt.Fprintf(w, ` data-optimistic-class="%s"`, escapeAttr(optimistic.Class)); err != nil {
                return err
            }
        }
        if optimistic.Text != "" {
            if _, err := fmt.Fprintf(w, ` data-optimistic-text="%s"`, escapeAttr(optimistic.Text)); err != nil {
                return err
            }
        }
    }

    return nil
}
```

### Refs

```go
// Refs are not rendered to HTML, but the HID is stored for later binding
func (r *Renderer) needsHID(node *VNode) bool {
    // ... existing checks ...

    // Check for refs
    if _, hasRef := node.Props["_ref"]; hasRef {
        return true
    }

    return false
}
```

---

## Testing

### Unit Tests

```go
func TestRenderText(t *testing.T) {
    renderer := NewRenderer(RendererConfig{})

    node := Text("Hello, World!")
    html, err := renderer.RenderToString(node)

    require.NoError(t, err)
    assert.Equal(t, "Hello, World!", html)
}

func TestRenderElement(t *testing.T) {
    renderer := NewRenderer(RendererConfig{})

    node := Div(Class("container"),
        H1(Text("Title")),
        P(Text("Content")),
    )
    html, err := renderer.RenderToString(node)

    require.NoError(t, err)
    assert.Contains(t, html, `<div class="container">`)
    assert.Contains(t, html, `<h1>Title</h1>`)
    assert.Contains(t, html, `<p>Content</p>`)
}

func TestEscapeHTML(t *testing.T) {
    renderer := NewRenderer(RendererConfig{})

    node := Div(Text("<script>alert('xss')</script>"))
    html, err := renderer.RenderToString(node)

    require.NoError(t, err)
    assert.NotContains(t, html, "<script>")
    assert.Contains(t, html, "&lt;script&gt;")
}

func TestHydrationID(t *testing.T) {
    renderer := NewRenderer(RendererConfig{})

    handler := func() {}
    node := Button(OnClick(handler), Text("Click"))
    html, err := renderer.RenderToString(node)

    require.NoError(t, err)
    assert.Contains(t, html, `data-hid="h1"`)
    assert.Contains(t, html, `data-on-click="true"`)
}

func TestVoidElements(t *testing.T) {
    renderer := NewRenderer(RendererConfig{})

    node := Input(Type("text"), Name("email"))
    html, err := renderer.RenderToString(node)

    require.NoError(t, err)
    assert.Contains(t, html, `<input type="text" name="email">`)
    assert.NotContains(t, html, `</input>`)
}

func TestBooleanAttributes(t *testing.T) {
    renderer := NewRenderer(RendererConfig{})

    node := Input(Type("checkbox"), Checked(true), Disabled(true))
    html, err := renderer.RenderToString(node)

    require.NoError(t, err)
    assert.Contains(t, html, ` checked`)
    assert.Contains(t, html, ` disabled`)
    assert.NotContains(t, html, `checked="true"`)
}

func TestFragment(t *testing.T) {
    renderer := NewRenderer(RendererConfig{})

    node := Fragment(
        Div(Text("One")),
        Div(Text("Two")),
    )
    html, err := renderer.RenderToString(node)

    require.NoError(t, err)
    assert.Equal(t, "<div>One</div><div>Two</div>", html)
}
```

### Benchmark Tests

```go
func BenchmarkRenderSimple(b *testing.B) {
    renderer := NewRenderer(RendererConfig{})
    node := Div(Class("card"),
        H1(Text("Title")),
        P(Text("Content")),
    )

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        renderer.Reset()
        renderer.RenderToString(node)
    }
}

func BenchmarkRenderLargeTree(b *testing.B) {
    renderer := NewRenderer(RendererConfig{})

    // Build a tree with 1000 elements
    var items []any
    for i := 0; i < 1000; i++ {
        items = append(items, Li(Text(fmt.Sprintf("Item %d", i))))
    }
    node := Ul(items...)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        renderer.Reset()
        renderer.RenderToString(node)
    }
}

func BenchmarkRenderWithHandlers(b *testing.B) {
    renderer := NewRenderer(RendererConfig{})
    handler := func() {}

    var buttons []any
    for i := 0; i < 100; i++ {
        buttons = append(buttons, Button(OnClick(handler), Text(fmt.Sprintf("Button %d", i))))
    }
    node := Div(buttons...)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        renderer.Reset()
        renderer.RenderToString(node)
    }
}

func BenchmarkRenderToWriter(b *testing.B) {
    renderer := NewRenderer(RendererConfig{})
    node := Div(Class("card"),
        H1(Text("Title")),
        P(Text("Content")),
    )

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        renderer.Reset()
        renderer.RenderToWriter(io.Discard, node)
    }
}
```

---

## File Structure

```
pkg/render/
├── renderer.go        # Main Renderer type and RenderToString/RenderToWriter
├── renderer_test.go
├── html.go            # HTML-specific rendering (elements, attributes)
├── html_test.go
├── escape.go          # HTML/attribute escaping
├── escape_test.go
├── page.go            # Full page rendering (DOCTYPE, head, body)
├── page_test.go
├── streaming.go       # StreamingRenderer for large pages
├── streaming_test.go
├── elements.go        # Void elements, inline elements, boolean attrs
└── bench_test.go      # Benchmarks
```

---

## Exit Criteria

Phase 6 is complete when:

1. [x] `RenderToString(VNode)` produces correct HTML
2. [x] `RenderToWriter(io.Writer, VNode)` streams HTML
3. [x] All HTML5 elements render correctly
4. [x] Void elements handled (no closing tag)
5. [x] Boolean attributes handled (no value)
6. [x] Text content properly escaped (XSS prevention)
7. [x] Hydration IDs generated for interactive elements
8. [x] Event handler markers added (`data-on-click`, etc.)
9. [x] Hook configuration rendered as data attributes
10. [x] Full page rendering with DOCTYPE, head, body
11. [x] Thin client script injected correctly
12. [x] CSRF token injected for WebSocket
13. [x] Unit tests for all cases (55 tests, 69.2% coverage)
14. [x] Benchmarks documented (10 benchmarks)
15. [x] Streaming renderer for large pages

---

## Dependencies

- **Requires**: Phase 2 (VNode structure), Phase 5 (Client script path)
- **Required by**: Phase 7 (Routing renders pages)

---

## Implementation Notes (2024-12-07)

### Actual File Structure

```
pkg/render/
├── doc.go              # Package documentation (55 lines)
├── escape.go           # HTML/attribute escaping (60 lines)
├── escape_test.go      # Escaping tests
├── elements.go         # Void/inline/boolean element lists (95 lines)
├── renderer.go         # Main Renderer + HTML rendering (385 lines)
├── renderer_test.go    # Renderer tests
├── page.go             # Full page rendering, PageData (428 lines)
├── page_test.go        # Page tests
├── streaming.go        # StreamingRenderer (95 lines)
├── streaming_test.go   # Streaming tests
└── bench_test.go       # Benchmarks
```

> **Note**: `html.go` was merged into `renderer.go` for simplicity. All specified functionality is present.

### Benchmark Results

| Operation | Time | Allocations |
|-----------|------|-------------|
| Simple element render | ~600ns | 24 allocs |
| Large tree (1000 nodes) | ~154µs | 6017 allocs |
| With handlers (100 buttons) | ~43µs | 1125 allocs |
| Full page render | ~1µs | 36 allocs |
| Deep nesting (20 levels) | ~2.4µs | 92 allocs |
| Complex page (50 table rows) | ~63µs | 1981 allocs |
| Escape HTML | ~170-230ns | 1-2 allocs |

### Test Coverage

- **55 tests** passing
- **69.2% coverage**
- All exit criteria verified through tests

---

*Phase 6 Specification - Version 1.1 (Completed)*
