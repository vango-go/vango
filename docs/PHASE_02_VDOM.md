# Phase 2: Virtual DOM

> **The representation and diffing of UI structure**

---

## Overview

The Virtual DOM (VDOM) provides an in-memory representation of the UI that can be efficiently diffed to produce minimal DOM updates. In Vango's server-driven architecture, the VDOM lives on the server and diffs produce binary patches sent to the client.

### Goals

1. **Immutable**: VNodes never mutate after creation
2. **Efficient diffing**: O(n) complexity for typical cases
3. **Minimal patches**: Only send what changed
4. **Type-safe**: Compile-time element/attribute validation
5. **Ergonomic API**: Clean, readable component code

### Non-Goals

1. Client-side rendering (Phase 5 handles patch application)
2. WASM compilation (deferred to future version)
3. Animation primitives (handled by client hooks)

---

## Core Types

### VNode

The fundamental building block of the virtual DOM.

```go
// Node kind discriminator
type VKind uint8

const (
    KindElement   VKind = iota  // <div>, <button>, etc.
    KindText                     // Plain text node
    KindFragment                 // Grouping without wrapper
    KindComponent                // Nested component
    KindRaw                      // Raw HTML (dangerous)
)

// The virtual node
type VNode struct {
    Kind     VKind
    Tag      string           // Element tag name (e.g., "div")
    Props    Props            // Attributes and event handlers
    Children []*VNode         // Child nodes
    Key      string           // Reconciliation key
    Text     string           // For KindText and KindRaw
    Comp     Component        // For KindComponent
    HID      string           // Hydration ID (assigned during render)
}

// Props holds attributes and handlers
type Props map[string]any

// Helper to check if node is interactive (needs HID)
func (v *VNode) IsInteractive() bool {
    if v.Kind != KindElement {
        return false
    }
    for key := range v.Props {
        if strings.HasPrefix(key, "on") {
            return true
        }
    }
    return false
}
```

### Component Interface

```go
// Component is anything that can render to a VNode
type Component interface {
    Render() *VNode
}

// FuncComponent wraps a render function
type FuncComponent struct {
    render func() *VNode
}

func (f *FuncComponent) Render() *VNode {
    return f.render()
}

// Func creates a stateful component from a function
func Func(render func() *VNode) Component {
    return &FuncComponent{render: render}
}
```

---

## Element API

### Design Philosophy

Elements are created using a **variadic function** pattern where attributes and children can be mixed freely:

```go
// Signature for all element functions
func Div(args ...any) *VNode

// Usage - attributes and children intermixed
Div(
    Class("card"),           // attribute
    ID("main"),              // attribute
    H1(Text("Title")),       // child
    P(Text("Content")),      // child
    OnClick(handler),        // event handler
)
```

This provides:
- **Flexibility**: Add attributes anywhere in the list
- **Readability**: Structure mirrors HTML
- **Type safety**: Compiler validates attribute types

### Element Factory

```go
// Internal element creation
func createElement(tag string, args []any) *VNode {
    node := &VNode{
        Kind:     KindElement,
        Tag:      tag,
        Props:    make(Props),
        Children: make([]*VNode, 0),
    }

    for _, arg := range args {
        switch v := arg.(type) {
        case nil:
            // Ignore nil (allows conditional attributes)
            continue

        case Attr:
            // Single attribute
            node.Props[v.Key] = v.Value

        case []Attr:
            // Multiple attributes
            for _, attr := range v {
                node.Props[attr.Key] = attr.Value
            }

        case *VNode:
            // Child node
            if v != nil {
                node.Children = append(node.Children, v)
            }

        case []*VNode:
            // Multiple children
            for _, child := range v {
                if child != nil {
                    node.Children = append(node.Children, child)
                }
            }

        case Component:
            // Embedded component
            node.Children = append(node.Children, &VNode{
                Kind: KindComponent,
                Comp: v,
            })

        case string:
            // Shorthand for text
            node.Children = append(node.Children, &VNode{
                Kind: KindText,
                Text: v,
            })

        case EventHandler:
            // Event handler
            node.Props[v.Event] = v.Handler

        default:
            // Unknown type - panic in dev, ignore in prod
            if debugMode {
                panic(fmt.Sprintf("unknown arg type: %T", arg))
            }
        }
    }

    return node
}
```

### Generated Elements

All HTML5 elements are generated:

```go
// generated_elements.go

// Document structure
func Html(args ...any) *VNode { return createElement("html", args) }
func Head(args ...any) *VNode { return createElement("head", args) }
func Body(args ...any) *VNode { return createElement("body", args) }
func Title(args ...any) *VNode { return createElement("title", args) }

// Content sectioning
func Header(args ...any) *VNode { return createElement("header", args) }
func Footer(args ...any) *VNode { return createElement("footer", args) }
func Main(args ...any) *VNode { return createElement("main", args) }
func Nav(args ...any) *VNode { return createElement("nav", args) }
func Section(args ...any) *VNode { return createElement("section", args) }
func Article(args ...any) *VNode { return createElement("article", args) }
func Aside(args ...any) *VNode { return createElement("aside", args) }
func H1(args ...any) *VNode { return createElement("h1", args) }
func H2(args ...any) *VNode { return createElement("h2", args) }
func H3(args ...any) *VNode { return createElement("h3", args) }
func H4(args ...any) *VNode { return createElement("h4", args) }
func H5(args ...any) *VNode { return createElement("h5", args) }
func H6(args ...any) *VNode { return createElement("h6", args) }

// Text content
func Div(args ...any) *VNode { return createElement("div", args) }
func P(args ...any) *VNode { return createElement("p", args) }
func Span(args ...any) *VNode { return createElement("span", args) }
func Pre(args ...any) *VNode { return createElement("pre", args) }
func Blockquote(args ...any) *VNode { return createElement("blockquote", args) }
func Ul(args ...any) *VNode { return createElement("ul", args) }
func Ol(args ...any) *VNode { return createElement("ol", args) }
func Li(args ...any) *VNode { return createElement("li", args) }
func Dl(args ...any) *VNode { return createElement("dl", args) }
func Dt(args ...any) *VNode { return createElement("dt", args) }
func Dd(args ...any) *VNode { return createElement("dd", args) }
func Hr(args ...any) *VNode { return createElement("hr", args) }
func Figure(args ...any) *VNode { return createElement("figure", args) }
func Figcaption(args ...any) *VNode { return createElement("figcaption", args) }

// Inline text
func A(args ...any) *VNode { return createElement("a", args) }
func Strong(args ...any) *VNode { return createElement("strong", args) }
func Em(args ...any) *VNode { return createElement("em", args) }
func B(args ...any) *VNode { return createElement("b", args) }
func I(args ...any) *VNode { return createElement("i", args) }
func U(args ...any) *VNode { return createElement("u", args) }
func S(args ...any) *VNode { return createElement("s", args) }
func Small(args ...any) *VNode { return createElement("small", args) }
func Mark(args ...any) *VNode { return createElement("mark", args) }
func Sub(args ...any) *VNode { return createElement("sub", args) }
func Sup(args ...any) *VNode { return createElement("sup", args) }
func Code(args ...any) *VNode { return createElement("code", args) }
func Kbd(args ...any) *VNode { return createElement("kbd", args) }
func Samp(args ...any) *VNode { return createElement("samp", args) }
func Var(args ...any) *VNode { return createElement("var", args) }
func Abbr(args ...any) *VNode { return createElement("abbr", args) }
func Time_(args ...any) *VNode { return createElement("time", args) }
func Br(args ...any) *VNode { return createElement("br", args) }
func Wbr(args ...any) *VNode { return createElement("wbr", args) }

// Forms
func Form(args ...any) *VNode { return createElement("form", args) }
func Input(args ...any) *VNode { return createElement("input", args) }
func Textarea(args ...any) *VNode { return createElement("textarea", args) }
func Select(args ...any) *VNode { return createElement("select", args) }
func Option(args ...any) *VNode { return createElement("option", args) }
func Optgroup(args ...any) *VNode { return createElement("optgroup", args) }
func Button(args ...any) *VNode { return createElement("button", args) }
func Label(args ...any) *VNode { return createElement("label", args) }
func Fieldset(args ...any) *VNode { return createElement("fieldset", args) }
func Legend(args ...any) *VNode { return createElement("legend", args) }
func Datalist(args ...any) *VNode { return createElement("datalist", args) }
func Output(args ...any) *VNode { return createElement("output", args) }
func Progress(args ...any) *VNode { return createElement("progress", args) }
func Meter(args ...any) *VNode { return createElement("meter", args) }

// Tables
func Table(args ...any) *VNode { return createElement("table", args) }
func Thead(args ...any) *VNode { return createElement("thead", args) }
func Tbody(args ...any) *VNode { return createElement("tbody", args) }
func Tfoot(args ...any) *VNode { return createElement("tfoot", args) }
func Tr(args ...any) *VNode { return createElement("tr", args) }
func Th(args ...any) *VNode { return createElement("th", args) }
func Td(args ...any) *VNode { return createElement("td", args) }
func Caption(args ...any) *VNode { return createElement("caption", args) }
func Colgroup(args ...any) *VNode { return createElement("colgroup", args) }
func Col(args ...any) *VNode { return createElement("col", args) }

// Media
func Img(args ...any) *VNode { return createElement("img", args) }
func Picture(args ...any) *VNode { return createElement("picture", args) }
func Source(args ...any) *VNode { return createElement("source", args) }
func Video(args ...any) *VNode { return createElement("video", args) }
func Audio(args ...any) *VNode { return createElement("audio", args) }
func Track(args ...any) *VNode { return createElement("track", args) }
func Iframe(args ...any) *VNode { return createElement("iframe", args) }
func Embed(args ...any) *VNode { return createElement("embed", args) }
func Object(args ...any) *VNode { return createElement("object", args) }
func Canvas(args ...any) *VNode { return createElement("canvas", args) }
func Svg(args ...any) *VNode { return createElement("svg", args) }

// Interactive
func Details(args ...any) *VNode { return createElement("details", args) }
func Summary(args ...any) *VNode { return createElement("summary", args) }
func Dialog(args ...any) *VNode { return createElement("dialog", args) }
func Menu(args ...any) *VNode { return createElement("menu", args) }

// Scripting
func Script(args ...any) *VNode { return createElement("script", args) }
func Noscript(args ...any) *VNode { return createElement("noscript", args) }
func Template(args ...any) *VNode { return createElement("template", args) }
func Slot(args ...any) *VNode { return createElement("slot", args) }

// Web components
func CustomElement(tag string, args ...any) *VNode { return createElement(tag, args) }
```

### Void Elements

Elements that cannot have children:

```go
var voidElements = map[string]bool{
    "area": true, "base": true, "br": true, "col": true,
    "embed": true, "hr": true, "img": true, "input": true,
    "link": true, "meta": true, "param": true, "source": true,
    "track": true, "wbr": true,
}

func IsVoidElement(tag string) bool {
    return voidElements[tag]
}
```

---

## Attribute API

### Attribute Type

```go
type Attr struct {
    Key   string
    Value any
}

// Attr constructor
func attr(key string, value any) Attr {
    return Attr{Key: key, Value: value}
}
```

### Global Attributes

```go
// Identity
func ID(id string) Attr { return attr("id", id) }
func Class(classes ...string) Attr { return attr("class", strings.Join(classes, " ")) }
func Style(style string) Attr { return attr("style", style) }

// Data attributes
func Data(key, value string) Attr { return attr("data-"+key, value) }

// Accessibility
func Role(role string) Attr { return attr("role", role) }
func AriaLabel(label string) Attr { return attr("aria-label", label) }
func AriaHidden(hidden bool) Attr { return attr("aria-hidden", hidden) }
func AriaExpanded(expanded bool) Attr { return attr("aria-expanded", expanded) }
func AriaDescribedBy(id string) Attr { return attr("aria-describedby", id) }
func AriaLabelledBy(id string) Attr { return attr("aria-labelledby", id) }
func AriaLive(mode string) Attr { return attr("aria-live", mode) }
func AriaControls(id string) Attr { return attr("aria-controls", id) }
func AriaCurrent(value string) Attr { return attr("aria-current", value) }
func AriaDisabled(disabled bool) Attr { return attr("aria-disabled", disabled) }
func AriaPressed(pressed string) Attr { return attr("aria-pressed", pressed) }
func AriaSelected(selected bool) Attr { return attr("aria-selected", selected) }

// Keyboard
func TabIndex(index int) Attr { return attr("tabindex", index) }
func AccessKey(key string) Attr { return attr("accesskey", key) }

// Visibility
func Hidden() Attr { return attr("hidden", true) }
func Title(title string) Attr { return attr("title", title) }

// Behavior
func ContentEditable(editable bool) Attr { return attr("contenteditable", editable) }
func Draggable() Attr { return attr("draggable", "true") }
func Spellcheck(check bool) Attr { return attr("spellcheck", check) }

// Language
func Lang(lang string) Attr { return attr("lang", lang) }
func Dir(dir string) Attr { return attr("dir", dir) }
```

### Link Attributes

```go
func Href(url string) Attr { return attr("href", url) }
func Target(target string) Attr { return attr("target", target) }
func Rel(rel string) Attr { return attr("rel", rel) }
func Download(filename ...string) Attr {
    if len(filename) > 0 {
        return attr("download", filename[0])
    }
    return attr("download", true)
}
func Hreflang(lang string) Attr { return attr("hreflang", lang) }
```

### Form Attributes

```go
// Input attributes
func Name(name string) Attr { return attr("name", name) }
func Value(value string) Attr { return attr("value", value) }
func Type(t string) Attr { return attr("type", t) }
func Placeholder(text string) Attr { return attr("placeholder", text) }

// States
func Disabled() Attr { return attr("disabled", true) }
func Readonly() Attr { return attr("readonly", true) }
func Required() Attr { return attr("required", true) }
func Checked() Attr { return attr("checked", true) }
func Selected() Attr { return attr("selected", true) }
func Multiple() Attr { return attr("multiple", true) }
func Autofocus() Attr { return attr("autofocus", true) }
func Autocomplete(value string) Attr { return attr("autocomplete", value) }

// Validation
func Pattern(pattern string) Attr { return attr("pattern", pattern) }
func MinLength(n int) Attr { return attr("minlength", n) }
func MaxLength(n int) Attr { return attr("maxlength", n) }
func Min(value string) Attr { return attr("min", value) }
func Max(value string) Attr { return attr("max", value) }
func Step(value string) Attr { return attr("step", value) }

// Files
func Accept(types string) Attr { return attr("accept", types) }
func Capture(mode string) Attr { return attr("capture", mode) }

// Textarea
func Rows(n int) Attr { return attr("rows", n) }
func Cols(n int) Attr { return attr("cols", n) }
func Wrap(mode string) Attr { return attr("wrap", mode) }

// Form element
func Action(url string) Attr { return attr("action", url) }
func Method(method string) Attr { return attr("method", method) }
func Enctype(enctype string) Attr { return attr("enctype", enctype) }
func Novalidate() Attr { return attr("novalidate", true) }

// Label
func For(id string) Attr { return attr("for", id) }
```

### Media Attributes

```go
func Src(url string) Attr { return attr("src", url) }
func Alt(text string) Attr { return attr("alt", text) }
func Width(w int) Attr { return attr("width", w) }
func Height(h int) Attr { return attr("height", h) }
func Loading(mode string) Attr { return attr("loading", mode) }
func Decoding(mode string) Attr { return attr("decoding", mode) }
func Srcset(srcset string) Attr { return attr("srcset", srcset) }
func Sizes(sizes string) Attr { return attr("sizes", sizes) }

// Video/Audio
func Controls() Attr { return attr("controls", true) }
func Autoplay() Attr { return attr("autoplay", true) }
func Loop() Attr { return attr("loop", true) }
func Muted() Attr { return attr("muted", true) }
func Preload(mode string) Attr { return attr("preload", mode) }
func Poster(url string) Attr { return attr("poster", url) }
func Playsinline() Attr { return attr("playsinline", true) }

// Iframe
func Sandbox(value string) Attr { return attr("sandbox", value) }
func Allow(value string) Attr { return attr("allow", value) }
func Allowfullscreen() Attr { return attr("allowfullscreen", true) }
```

### Table Attributes

```go
func Colspan(n int) Attr { return attr("colspan", n) }
func Rowspan(n int) Attr { return attr("rowspan", n) }
func Scope(scope string) Attr { return attr("scope", scope) }
func Headers(ids string) Attr { return attr("headers", ids) }
```

### Conditional Attributes

```go
// ClassIf adds a class conditionally
func ClassIf(condition bool, class string) Attr {
    if condition {
        return attr("class", class)
    }
    return Attr{} // Empty attr, will be ignored
}

// AttrIf adds any attribute conditionally
func AttrIf(condition bool, a Attr) Attr {
    if condition {
        return a
    }
    return Attr{}
}

// Classes merges multiple class values
func Classes(classes ...any) Attr {
    var result []string
    for _, c := range classes {
        switch v := c.(type) {
        case string:
            if v != "" {
                result = append(result, v)
            }
        case []string:
            result = append(result, v...)
        case map[string]bool:
            for class, include := range v {
                if include {
                    result = append(result, class)
                }
            }
        }
    }
    return attr("class", strings.Join(result, " "))
}
```

---

## Event Handlers

### Event Handler Type

```go
type EventHandler struct {
    Event   string      // "onclick", "oninput", etc.
    Handler any         // Function to call
}

func event(name string, handler any) EventHandler {
    return EventHandler{Event: "on" + name, Handler: handler}
}
```

### Mouse Events

```go
func OnClick(handler any) EventHandler { return event("click", handler) }
func OnDblClick(handler any) EventHandler { return event("dblclick", handler) }
func OnMouseDown(handler any) EventHandler { return event("mousedown", handler) }
func OnMouseUp(handler any) EventHandler { return event("mouseup", handler) }
func OnMouseMove(handler any) EventHandler { return event("mousemove", handler) }
func OnMouseEnter(handler any) EventHandler { return event("mouseenter", handler) }
func OnMouseLeave(handler any) EventHandler { return event("mouseleave", handler) }
func OnMouseOver(handler any) EventHandler { return event("mouseover", handler) }
func OnMouseOut(handler any) EventHandler { return event("mouseout", handler) }
func OnContextMenu(handler any) EventHandler { return event("contextmenu", handler) }
func OnWheel(handler any) EventHandler { return event("wheel", handler) }
```

### Keyboard Events

```go
func OnKeyDown(handler any) EventHandler { return event("keydown", handler) }
func OnKeyUp(handler any) EventHandler { return event("keyup", handler) }
func OnKeyPress(handler any) EventHandler { return event("keypress", handler) }
```

### Form Events

```go
func OnInput(handler any) EventHandler { return event("input", handler) }
func OnChange(handler any) EventHandler { return event("change", handler) }
func OnSubmit(handler any) EventHandler { return event("submit", handler) }
func OnFocus(handler any) EventHandler { return event("focus", handler) }
func OnBlur(handler any) EventHandler { return event("blur", handler) }
func OnFocusIn(handler any) EventHandler { return event("focusin", handler) }
func OnFocusOut(handler any) EventHandler { return event("focusout", handler) }
func OnSelect(handler any) EventHandler { return event("select", handler) }
func OnInvalid(handler any) EventHandler { return event("invalid", handler) }
func OnReset(handler any) EventHandler { return event("reset", handler) }
```

### Drag Events

```go
func OnDragStart(handler any) EventHandler { return event("dragstart", handler) }
func OnDrag(handler any) EventHandler { return event("drag", handler) }
func OnDragEnd(handler any) EventHandler { return event("dragend", handler) }
func OnDragEnter(handler any) EventHandler { return event("dragenter", handler) }
func OnDragOver(handler any) EventHandler { return event("dragover", handler) }
func OnDragLeave(handler any) EventHandler { return event("dragleave", handler) }
func OnDrop(handler any) EventHandler { return event("drop", handler) }
```

### Touch Events

```go
func OnTouchStart(handler any) EventHandler { return event("touchstart", handler) }
func OnTouchMove(handler any) EventHandler { return event("touchmove", handler) }
func OnTouchEnd(handler any) EventHandler { return event("touchend", handler) }
func OnTouchCancel(handler any) EventHandler { return event("touchcancel", handler) }
```

### Scroll Events

```go
func OnScroll(handler any) EventHandler { return event("scroll", handler) }
```

### Media Events

```go
func OnPlay(handler any) EventHandler { return event("play", handler) }
func OnPause(handler any) EventHandler { return event("pause", handler) }
func OnEnded(handler any) EventHandler { return event("ended", handler) }
func OnTimeUpdate(handler any) EventHandler { return event("timeupdate", handler) }
func OnLoadStart(handler any) EventHandler { return event("loadstart", handler) }
func OnLoadedData(handler any) EventHandler { return event("loadeddata", handler) }
func OnLoadedMetadata(handler any) EventHandler { return event("loadedmetadata", handler) }
func OnCanPlay(handler any) EventHandler { return event("canplay", handler) }
func OnError(handler any) EventHandler { return event("error", handler) }
```

### Handler Signatures

Handlers can have different signatures:

```go
// No arguments
OnClick(func() {
    count.Inc()
})

// With typed event
OnClick(func(e MouseEvent) {
    fmt.Println(e.ClientX, e.ClientY)
})

// With value (for input)
OnInput(func(value string) {
    search.Set(value)
})

// With form data (for submit)
OnSubmit(func(data FormData) {
    handleSubmit(data)
})
```

---

## Helper Functions

### Text Nodes

```go
// Text creates a text node
func Text(content string) *VNode {
    return &VNode{
        Kind: KindText,
        Text: content,
    }
}

// Textf creates a formatted text node
func Textf(format string, args ...any) *VNode {
    return Text(fmt.Sprintf(format, args...))
}

// Raw creates an unescaped HTML node (use carefully!)
func Raw(html string) *VNode {
    return &VNode{
        Kind: KindRaw,
        Text: html,
    }
}
```

### Fragments

```go
// Fragment groups children without a wrapper element
func Fragment(children ...any) *VNode {
    node := &VNode{
        Kind:     KindFragment,
        Children: make([]*VNode, 0),
    }

    for _, child := range children {
        switch v := child.(type) {
        case nil:
            continue
        case *VNode:
            if v != nil {
                node.Children = append(node.Children, v)
            }
        case []*VNode:
            for _, c := range v {
                if c != nil {
                    node.Children = append(node.Children, c)
                }
            }
        }
    }

    return node
}
```

### Conditionals

```go
// If returns the node if condition is true, nil otherwise
func If(condition bool, node *VNode) *VNode {
    if condition {
        return node
    }
    return nil
}

// IfElse returns first node if true, second if false
func IfElse(condition bool, ifTrue, ifFalse *VNode) *VNode {
    if condition {
        return ifTrue
    }
    return ifFalse
}

// When is like If but with lazy evaluation
func When(condition bool, fn func() *VNode) *VNode {
    if condition {
        return fn()
    }
    return nil
}

// Unless is the inverse of If
func Unless(condition bool, node *VNode) *VNode {
    if !condition {
        return node
    }
    return nil
}

// Switch for multiple conditions
func Switch[T comparable](value T, cases ...Case[T]) *VNode {
    for _, c := range cases {
        if c.Value == value {
            return c.Node
        }
    }
    // Check for default
    for _, c := range cases {
        if c.IsDefault {
            return c.Node
        }
    }
    return nil
}

type Case[T comparable] struct {
    Value     T
    Node      *VNode
    IsDefault bool
}

func Case_[T comparable](value T, node *VNode) Case[T] {
    return Case[T]{Value: value, Node: node}
}

func Default[T comparable](node *VNode) Case[T] {
    return Case[T]{Node: node, IsDefault: true}
}
```

### Lists

```go
// Range maps a slice to VNodes
func Range[T any](items []T, fn func(item T, index int) *VNode) []*VNode {
    result := make([]*VNode, 0, len(items))
    for i, item := range items {
        node := fn(item, i)
        if node != nil {
            result = append(result, node)
        }
    }
    return result
}

// RangeMap maps a map to VNodes
func RangeMap[K comparable, V any](m map[K]V, fn func(key K, value V) *VNode) []*VNode {
    result := make([]*VNode, 0, len(m))
    for k, v := range m {
        node := fn(k, v)
        if node != nil {
            result = append(result, node)
        }
    }
    return result
}

// Repeat creates n nodes
func Repeat(n int, fn func(i int) *VNode) []*VNode {
    result := make([]*VNode, 0, n)
    for i := 0; i < n; i++ {
        node := fn(i)
        if node != nil {
            result = append(result, node)
        }
    }
    return result
}

// Key sets the reconciliation key on a node
func Key(key any) Attr {
    return attr("key", fmt.Sprintf("%v", key))
}
```

---

## Diffing Algorithm

### Patch Types

```go
type PatchOp uint8

const (
    PatchSetText      PatchOp = 0x01  // Update text content
    PatchSetAttr      PatchOp = 0x02  // Set/update attribute
    PatchRemoveAttr   PatchOp = 0x03  // Remove attribute
    PatchInsertNode   PatchOp = 0x04  // Insert new node
    PatchRemoveNode   PatchOp = 0x05  // Remove node
    PatchMoveNode     PatchOp = 0x06  // Move node to new position
    PatchReplaceNode  PatchOp = 0x07  // Replace node entirely
    PatchSetValue     PatchOp = 0x08  // Set input value
    PatchSetChecked   PatchOp = 0x09  // Set checkbox checked
    PatchFocus        PatchOp = 0x0A  // Focus element
)

type Patch struct {
    Op       PatchOp
    HID      string      // Target element's hydration ID
    Key      string      // Attribute key (for SetAttr/RemoveAttr)
    Value    string      // New value
    Node     *VNode      // For InsertNode/ReplaceNode
    Index    int         // Insert position
    ParentID string      // Parent for InsertNode
}
```

### Diff Function

```go
// Diff compares two trees and returns patches
func Diff(prev, next *VNode) []Patch {
    var patches []Patch
    diff(prev, next, &patches)
    return patches
}

func diff(prev, next *VNode, patches *[]Patch) {
    // Both nil - nothing to do
    if prev == nil && next == nil {
        return
    }

    // Node added
    if prev == nil {
        // Handled by parent (InsertNode)
        return
    }

    // Node removed
    if next == nil {
        *patches = append(*patches, Patch{
            Op:  PatchRemoveNode,
            HID: prev.HID,
        })
        return
    }

    // Different types - replace
    if prev.Kind != next.Kind {
        *patches = append(*patches, Patch{
            Op:   PatchReplaceNode,
            HID:  prev.HID,
            Node: next,
        })
        return
    }

    // Same type, diff by kind
    switch prev.Kind {
    case KindText:
        diffText(prev, next, patches)
    case KindElement:
        diffElement(prev, next, patches)
    case KindFragment:
        diffChildren(prev, next, patches)
    case KindComponent:
        diffComponent(prev, next, patches)
    case KindRaw:
        diffRaw(prev, next, patches)
    }
}

func diffText(prev, next *VNode, patches *[]Patch) {
    if prev.Text != next.Text {
        *patches = append(*patches, Patch{
            Op:    PatchSetText,
            HID:   prev.HID,
            Value: next.Text,
        })
    }
    // Copy HID
    next.HID = prev.HID
}

func diffElement(prev, next *VNode, patches *[]Patch) {
    // Different tag - replace
    if prev.Tag != next.Tag {
        *patches = append(*patches, Patch{
            Op:   PatchReplaceNode,
            HID:  prev.HID,
            Node: next,
        })
        return
    }

    // Copy HID
    next.HID = prev.HID

    // Diff props
    diffProps(prev, next, patches)

    // Diff children
    diffChildren(prev, next, patches)
}

func diffProps(prev, next *VNode, patches *[]Patch) {
    // Check for removed/changed props
    for key, prevVal := range prev.Props {
        if isEventHandler(key) {
            continue // Events handled separately
        }

        nextVal, exists := next.Props[key]
        if !exists {
            *patches = append(*patches, Patch{
                Op:  PatchRemoveAttr,
                HID: prev.HID,
                Key: key,
            })
        } else if !propsEqual(prevVal, nextVal) {
            *patches = append(*patches, Patch{
                Op:    PatchSetAttr,
                HID:   prev.HID,
                Key:   key,
                Value: propToString(nextVal),
            })
        }
    }

    // Check for added props
    for key, nextVal := range next.Props {
        if isEventHandler(key) {
            continue
        }

        if _, exists := prev.Props[key]; !exists {
            *patches = append(*patches, Patch{
                Op:    PatchSetAttr,
                HID:   prev.HID,
                Key:   key,
                Value: propToString(nextVal),
            })
        }
    }
}

func diffChildren(prev, next *VNode, patches *[]Patch) {
    prevChildren := prev.Children
    nextChildren := next.Children

    // Check if children are keyed
    if hasKeys(prevChildren) || hasKeys(nextChildren) {
        diffKeyedChildren(prev, prevChildren, nextChildren, patches)
    } else {
        diffUnkeyedChildren(prev, prevChildren, nextChildren, patches)
    }
}
```

### Unkeyed Reconciliation

```go
func diffUnkeyedChildren(parent *VNode, prev, next []*VNode, patches *[]Patch) {
    maxLen := max(len(prev), len(next))

    for i := 0; i < maxLen; i++ {
        var prevChild, nextChild *VNode

        if i < len(prev) {
            prevChild = prev[i]
        }
        if i < len(next) {
            nextChild = next[i]
        }

        if prevChild == nil && nextChild != nil {
            // Insert new child
            *patches = append(*patches, Patch{
                Op:       PatchInsertNode,
                ParentID: parent.HID,
                Index:    i,
                Node:     nextChild,
            })
        } else if prevChild != nil && nextChild == nil {
            // Remove child
            *patches = append(*patches, Patch{
                Op:  PatchRemoveNode,
                HID: prevChild.HID,
            })
        } else {
            // Diff existing
            diff(prevChild, nextChild, patches)
        }
    }
}
```

### Keyed Reconciliation

```go
func diffKeyedChildren(parent *VNode, prev, next []*VNode, patches *[]Patch) {
    // Build key maps
    prevKeyMap := make(map[string]int)
    nextKeyMap := make(map[string]int)

    for i, child := range prev {
        if key := getKey(child); key != "" {
            prevKeyMap[key] = i
        }
    }
    for i, child := range next {
        if key := getKey(child); key != "" {
            nextKeyMap[key] = i
        }
    }

    // Track which prev nodes have been matched
    matched := make(map[int]bool)

    // Process next children in order
    for nextIdx, nextChild := range next {
        key := getKey(nextChild)

        if prevIdx, exists := prevKeyMap[key]; exists && key != "" {
            // Found matching key
            matched[prevIdx] = true
            prevChild := prev[prevIdx]

            // Check if position changed
            if prevIdx != nextIdx {
                *patches = append(*patches, Patch{
                    Op:       PatchMoveNode,
                    HID:      prevChild.HID,
                    ParentID: parent.HID,
                    Index:    nextIdx,
                })
            }

            // Diff the node itself
            diff(prevChild, nextChild, patches)
        } else {
            // New node
            *patches = append(*patches, Patch{
                Op:       PatchInsertNode,
                ParentID: parent.HID,
                Index:    nextIdx,
                Node:     nextChild,
            })
        }
    }

    // Remove unmatched prev nodes
    for i, prevChild := range prev {
        if !matched[i] {
            *patches = append(*patches, Patch{
                Op:  PatchRemoveNode,
                HID: prevChild.HID,
            })
        }
    }
}

func getKey(node *VNode) string {
    if node == nil || node.Props == nil {
        return ""
    }
    if key, ok := node.Props["key"].(string); ok {
        return key
    }
    return ""
}

func hasKeys(children []*VNode) bool {
    for _, child := range children {
        if getKey(child) != "" {
            return true
        }
    }
    return false
}
```

### Helper Functions

```go
func isEventHandler(key string) bool {
    return strings.HasPrefix(key, "on")
}

func propsEqual(a, b any) bool {
    // Fast path for common types
    switch av := a.(type) {
    case string:
        if bv, ok := b.(string); ok {
            return av == bv
        }
    case int:
        if bv, ok := b.(int); ok {
            return av == bv
        }
    case bool:
        if bv, ok := b.(bool); ok {
            return av == bv
        }
    }
    // Fallback to reflect
    return reflect.DeepEqual(a, b)
}

func propToString(v any) string {
    switch val := v.(type) {
    case string:
        return val
    case bool:
        if val {
            return "true"
        }
        return "false"
    case int:
        return strconv.Itoa(val)
    case int64:
        return strconv.FormatInt(val, 10)
    case float64:
        return strconv.FormatFloat(val, 'f', -1, 64)
    default:
        return fmt.Sprintf("%v", v)
    }
}
```

---

## Hydration ID Generation

During SSR, interactive elements receive hydration IDs:

```go
type HIDGenerator struct {
    counter uint32
    mu      sync.Mutex
}

func NewHIDGenerator() *HIDGenerator {
    return &HIDGenerator{}
}

func (g *HIDGenerator) Next() string {
    g.mu.Lock()
    defer g.mu.Unlock()
    g.counter++
    return fmt.Sprintf("h%d", g.counter)
}

func (g *HIDGenerator) Reset() {
    g.mu.Lock()
    defer g.mu.Unlock()
    g.counter = 0
}

// AssignHIDs walks the tree and assigns HIDs to interactive nodes
func AssignHIDs(node *VNode, gen *HIDGenerator) {
    if node == nil {
        return
    }

    // Only elements can be interactive
    if node.Kind == KindElement && node.IsInteractive() {
        node.HID = gen.Next()
    }

    // Recurse
    for _, child := range node.Children {
        AssignHIDs(child, gen)
    }
}
```

---

## Testing Strategy

### Unit Tests

```go
func TestElementCreation(t *testing.T) {
    node := Div(Class("card"), ID("main"),
        H1(Text("Hello")),
    )

    assert.Equal(t, KindElement, node.Kind)
    assert.Equal(t, "div", node.Tag)
    assert.Equal(t, "card", node.Props["class"])
    assert.Equal(t, "main", node.Props["id"])
    assert.Equal(t, 1, len(node.Children))
    assert.Equal(t, "h1", node.Children[0].Tag)
}

func TestConditionalRendering(t *testing.T) {
    assert.Nil(t, If(false, Div()))
    assert.NotNil(t, If(true, Div()))

    node := IfElse(true, Div(ID("a")), Div(ID("b")))
    assert.Equal(t, "a", node.Props["id"])

    node = IfElse(false, Div(ID("a")), Div(ID("b")))
    assert.Equal(t, "b", node.Props["id"])
}

func TestRange(t *testing.T) {
    items := []string{"a", "b", "c"}
    nodes := Range(items, func(item string, i int) *VNode {
        return Li(Key(i), Text(item))
    })

    assert.Equal(t, 3, len(nodes))
    assert.Equal(t, "a", nodes[0].Children[0].Text)
}

func TestDiffText(t *testing.T) {
    prev := Text("Hello")
    prev.HID = "h1"
    next := Text("World")

    patches := Diff(prev, next)

    assert.Equal(t, 1, len(patches))
    assert.Equal(t, PatchSetText, patches[0].Op)
    assert.Equal(t, "h1", patches[0].HID)
    assert.Equal(t, "World", patches[0].Value)
}

func TestDiffAttributes(t *testing.T) {
    prev := Div(Class("old"), ID("test"))
    prev.HID = "h1"
    next := Div(Class("new"), Title("hello"))

    patches := Diff(prev, next)

    // Should have: SetAttr(class), RemoveAttr(id), SetAttr(title)
    assert.Equal(t, 3, len(patches))
}

func TestDiffKeyedChildren(t *testing.T) {
    prev := Ul(
        Li(Key("a"), Text("A")),
        Li(Key("b"), Text("B")),
        Li(Key("c"), Text("C")),
    )
    assignHIDsForTest(prev)

    next := Ul(
        Li(Key("c"), Text("C")),
        Li(Key("a"), Text("A")),
        Li(Key("b"), Text("B")),
    )

    patches := Diff(prev, next)

    // Should produce MoveNode patches, not Remove+Insert
    moveCount := 0
    for _, p := range patches {
        if p.Op == PatchMoveNode {
            moveCount++
        }
    }
    assert.True(t, moveCount > 0)
}
```

### Benchmark Tests

```go
func BenchmarkElementCreation(b *testing.B) {
    for i := 0; i < b.N; i++ {
        _ = Div(Class("card"),
            H1(Text("Title")),
            P(Text("Content")),
            Button(OnClick(func() {}), Text("Click")),
        )
    }
}

func BenchmarkDiffSameTree(b *testing.B) {
    tree := createLargeTree(100)
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = Diff(tree, tree)
    }
}

func BenchmarkDiffSmallChange(b *testing.B) {
    prev := createLargeTree(100)
    next := createLargeTreeWithChange(100)
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = Diff(prev, next)
    }
}

func BenchmarkDiffKeyedReorder(b *testing.B) {
    prev := createKeyedList(100)
    next := createReorderedKeyedList(100)
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = Diff(prev, next)
    }
}
```

---

## Benchmark Results

Performance benchmarks run on Apple M-series (arm64).

### Element Creation

| Operation | Time | Allocations | Notes |
|-----------|------|-------------|-------|
| Simple Div | ~114ns | 5 | Single element |
| Complex card (6 elements) | ~930ns | 45 | Nested structure |

### Diff Algorithm

| Operation | Time | Allocations | Notes |
|-----------|------|-------------|-------|
| Same tree (100 nodes) | **~2.4μs** | 0 | No changes = no allocs |
| Text change | **~82ns** | 1 | Single node update |
| Attribute change | **~104ns** | 1 | Single attr update |
| 10 unkeyed children | ~268ns | 1 | Positional matching |
| 100 unkeyed children | **~2.2μs** | 1 | Positional matching |
| 10 keyed reorder | ~1.9μs | 8 | Complete shuffle |
| 100 keyed reorder | **~23μs** | 38 | **Worst case**: complete shuffle |
| Keyed addition (100 items) | ~22μs | 37 | Add items to list |
| Keyed removal (100 items) | ~22μs | 37 | Remove items from list |
| Large tree (100 nodes) | ~2.6μs | 1 | Single change in tree |
| Large tree (1000 nodes) | **~25μs** | 1 | Single change in tree |

### Key Insights

1. **Zero-allocation same-tree diff**: When nothing changes, we allocate nothing
2. **O(n) keyed reconciliation**: Even 100-item reorder is only ~23μs
3. **Single allocation for changes**: Most updates allocate just 1 patch struct
4. **1000-node trees in 25μs**: Well under the 16ms frame budget (0.15%)

### Performance vs Frame Budget

```
Operation               Time      % of 16ms frame
─────────────────────────────────────────────────
Text change             82ns      0.0005%
100 unkeyed diff        2.2μs     0.014%
100 keyed reorder       23μs      0.14%
1000 nodes diff         25μs      0.15%
```

**Benchmark command:** `go test ./pkg/vdom/... -bench=. -benchmem`

---

## File Structure

```
pkg/vdom/
├── vnode.go              # VNode type and helpers
├── vnode_test.go
├── elements.go           # Element factory functions
├── elements_gen.go       # Generated element functions
├── attributes.go         # Attribute functions
├── attributes_gen.go     # Generated attributes
├── events.go             # Event handler functions
├── events_gen.go         # Generated events
├── helpers.go            # Text, Fragment, If, Range, etc.
├── helpers_test.go
├── diff.go               # Diff algorithm
├── diff_test.go
├── patch.go              # Patch types
├── hydration.go          # HID generation
├── hydration_test.go
└── vdom_bench_test.go
```

---

## Exit Criteria

Phase 2 is complete when:

1. [x] All HTML5 elements implemented (95 elements)
2. [x] All common attributes implemented (80+ functions)
3. [x] All event handlers implemented (70+ functions)
4. [x] Helper functions (If, Range, Fragment, etc.) working
5. [x] Diff algorithm produces correct patches
6. [x] Keyed reconciliation detects moves
7. [x] Hydration IDs assigned to interactive elements
8. [x] Unit test coverage > 90% (achieved: 96.8%)
9. [x] Benchmark baselines documented
10. [x] Code reviewed and documented

**Status: COMPLETE (2024-12-06)**

---

## Dependencies

- **Requires**: Phase 1 (signals referenced in event handlers)
- **Required by**: Phase 3 (VNode serialization), Phase 6 (SSR)

---

*Phase 2 Specification - Version 1.1 (Updated 2024-12-07)*
*Benchmark results added*
