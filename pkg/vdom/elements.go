package vdom

// voidElements are elements that cannot have children.
var voidElements = map[string]bool{
	"area":   true,
	"base":   true,
	"br":     true,
	"col":    true,
	"embed":  true,
	"hr":     true,
	"img":    true,
	"input":  true,
	"link":   true,
	"meta":   true,
	"param":  true,
	"source": true,
	"track":  true,
	"wbr":    true,
}

// IsVoidElement returns true if the tag is a void element.
func IsVoidElement(tag string) bool {
	return voidElements[tag]
}

// createElement creates a new VNode with the given tag and arguments.
// Arguments can be: nil, Attr, []Attr, *VNode, []*VNode, Component, string, EventHandler.
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
			if v.Key != "" {
				if v.Key == "key" {
					if s, ok := v.Value.(string); ok {
						node.Key = s
					}
				}
				// Special handling for onhook - merge multiple handlers
				if v.Key == "onhook" {
					if existing := node.Props["onhook"]; existing != nil {
						// Merge with existing handlers
						if existingSlice, ok := existing.([]any); ok {
							node.Props["onhook"] = append(existingSlice, v.Value)
						} else {
							// Existing is a single handler, convert to slice
							node.Props["onhook"] = []any{existing, v.Value}
						}
					} else {
						node.Props[v.Key] = v.Value
					}
				} else {
					node.Props[v.Key] = v.Value
				}
			}

		case []Attr:
			// Multiple attributes
			for _, attr := range v {
				if attr.Key != "" {
					if attr.Key == "key" {
						if s, ok := attr.Value.(string); ok {
							node.Key = s
						}
					}
					// Special handling for onhook - merge multiple handlers
					if attr.Key == "onhook" {
						if existing := node.Props["onhook"]; existing != nil {
							if existingSlice, ok := existing.([]any); ok {
								node.Props["onhook"] = append(existingSlice, attr.Value)
							} else {
								node.Props["onhook"] = []any{existing, attr.Value}
							}
						} else {
							node.Props[attr.Key] = attr.Value
						}
					} else {
						node.Props[attr.Key] = attr.Value
					}
				}
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
			// Embedded component - wrap in KindComponent VNode
			node.Children = append(node.Children, &VNode{
				Kind: KindComponent,
				Comp: v,
			})

		case string:
			// Shorthand for text node
			node.Children = append(node.Children, &VNode{
				Kind: KindText,
				Text: v,
			})

		case EventHandler:
			// Event handler
			node.Props[v.Event] = v.Handler
		}
	}

	return node
}

// Document structure elements

func Html(args ...any) *VNode  { return createElement("html", args) }
func Head(args ...any) *VNode  { return createElement("head", args) }
func Body(args ...any) *VNode  { return createElement("body", args) }
func Title(args ...any) *VNode { return createElement("title", args) }
func Meta(args ...any) *VNode  { return createElement("meta", args) }
func Link(args ...any) *VNode  { return createElement("link", args) }
func Base(args ...any) *VNode  { return createElement("base", args) }

// Content sectioning elements

func Header(args ...any) *VNode  { return createElement("header", args) }
func Footer(args ...any) *VNode  { return createElement("footer", args) }
func Main(args ...any) *VNode    { return createElement("main", args) }
func Nav(args ...any) *VNode     { return createElement("nav", args) }
func Section(args ...any) *VNode { return createElement("section", args) }
func Article(args ...any) *VNode { return createElement("article", args) }
func Aside(args ...any) *VNode   { return createElement("aside", args) }
func Address(args ...any) *VNode { return createElement("address", args) }
func H1(args ...any) *VNode      { return createElement("h1", args) }
func H2(args ...any) *VNode      { return createElement("h2", args) }
func H3(args ...any) *VNode      { return createElement("h3", args) }
func H4(args ...any) *VNode      { return createElement("h4", args) }
func H5(args ...any) *VNode      { return createElement("h5", args) }
func H6(args ...any) *VNode      { return createElement("h6", args) }
func Hgroup(args ...any) *VNode  { return createElement("hgroup", args) }

// Text content elements

func Div(args ...any) *VNode        { return createElement("div", args) }
func P(args ...any) *VNode          { return createElement("p", args) }
func Span(args ...any) *VNode       { return createElement("span", args) }
func Pre(args ...any) *VNode        { return createElement("pre", args) }
func Blockquote(args ...any) *VNode { return createElement("blockquote", args) }
func Ul(args ...any) *VNode         { return createElement("ul", args) }
func Ol(args ...any) *VNode         { return createElement("ol", args) }
func Li(args ...any) *VNode         { return createElement("li", args) }
func Dl(args ...any) *VNode         { return createElement("dl", args) }
func Dt(args ...any) *VNode         { return createElement("dt", args) }
func Dd(args ...any) *VNode         { return createElement("dd", args) }
func Hr(args ...any) *VNode         { return createElement("hr", args) }
func Figure(args ...any) *VNode     { return createElement("figure", args) }
func Figcaption(args ...any) *VNode { return createElement("figcaption", args) }

// Inline text semantics

func A(args ...any) *VNode      { return createElement("a", args) }
func Strong(args ...any) *VNode { return createElement("strong", args) }
func Em(args ...any) *VNode     { return createElement("em", args) }
func B(args ...any) *VNode      { return createElement("b", args) }
func I(args ...any) *VNode      { return createElement("i", args) }
func U(args ...any) *VNode      { return createElement("u", args) }
func S(args ...any) *VNode      { return createElement("s", args) }
func Small(args ...any) *VNode  { return createElement("small", args) }
func Mark(args ...any) *VNode   { return createElement("mark", args) }
func Sub(args ...any) *VNode    { return createElement("sub", args) }
func Sup(args ...any) *VNode    { return createElement("sup", args) }
func Code(args ...any) *VNode   { return createElement("code", args) }
func Kbd(args ...any) *VNode    { return createElement("kbd", args) }
func Samp(args ...any) *VNode   { return createElement("samp", args) }
func Var(args ...any) *VNode    { return createElement("var", args) }
func Abbr(args ...any) *VNode   { return createElement("abbr", args) }
func Time_(args ...any) *VNode  { return createElement("time", args) }
func Cite(args ...any) *VNode   { return createElement("cite", args) }
func Q(args ...any) *VNode      { return createElement("q", args) }
func Dfn(args ...any) *VNode    { return createElement("dfn", args) }
func Ruby(args ...any) *VNode   { return createElement("ruby", args) }
func Rt(args ...any) *VNode     { return createElement("rt", args) }
func Rp(args ...any) *VNode     { return createElement("rp", args) }
func Bdi(args ...any) *VNode    { return createElement("bdi", args) }
func Bdo(args ...any) *VNode    { return createElement("bdo", args) }

// DataElement creates a <data> HTML element.
// Note: For data-* attributes, use Data(key, value) from attributes.go instead.
func DataElement(args ...any) *VNode { return createElement("data", args) }
func Br(args ...any) *VNode          { return createElement("br", args) }
func Wbr(args ...any) *VNode         { return createElement("wbr", args) }

// Form elements

func Form(args ...any) *VNode     { return createElement("form", args) }
func Input(args ...any) *VNode    { return createElement("input", args) }
func Textarea(args ...any) *VNode { return createElement("textarea", args) }
func Select(args ...any) *VNode   { return createElement("select", args) }
func Option(args ...any) *VNode   { return createElement("option", args) }
func Optgroup(args ...any) *VNode { return createElement("optgroup", args) }
func Button(args ...any) *VNode   { return createElement("button", args) }
func Label(args ...any) *VNode    { return createElement("label", args) }
func Fieldset(args ...any) *VNode { return createElement("fieldset", args) }
func Legend(args ...any) *VNode   { return createElement("legend", args) }
func Datalist(args ...any) *VNode { return createElement("datalist", args) }
func Output(args ...any) *VNode   { return createElement("output", args) }
func Progress(args ...any) *VNode { return createElement("progress", args) }
func Meter(args ...any) *VNode    { return createElement("meter", args) }

// Table elements

func Table(args ...any) *VNode    { return createElement("table", args) }
func Thead(args ...any) *VNode    { return createElement("thead", args) }
func Tbody(args ...any) *VNode    { return createElement("tbody", args) }
func Tfoot(args ...any) *VNode    { return createElement("tfoot", args) }
func Tr(args ...any) *VNode       { return createElement("tr", args) }
func Th(args ...any) *VNode       { return createElement("th", args) }
func Td(args ...any) *VNode       { return createElement("td", args) }
func Caption(args ...any) *VNode  { return createElement("caption", args) }
func Colgroup(args ...any) *VNode { return createElement("colgroup", args) }
func Col(args ...any) *VNode      { return createElement("col", args) }

// Media elements

func Img(args ...any) *VNode     { return createElement("img", args) }
func Picture(args ...any) *VNode { return createElement("picture", args) }
func Source(args ...any) *VNode  { return createElement("source", args) }
func Video(args ...any) *VNode   { return createElement("video", args) }
func Audio(args ...any) *VNode   { return createElement("audio", args) }
func Track(args ...any) *VNode   { return createElement("track", args) }
func Iframe(args ...any) *VNode  { return createElement("iframe", args) }
func Embed(args ...any) *VNode   { return createElement("embed", args) }
func Object(args ...any) *VNode  { return createElement("object", args) }
func Param(args ...any) *VNode   { return createElement("param", args) }
func Canvas(args ...any) *VNode  { return createElement("canvas", args) }
func Svg(args ...any) *VNode     { return createElement("svg", args) }
func Math(args ...any) *VNode    { return createElement("math", args) }
func Map_(args ...any) *VNode    { return createElement("map", args) }
func Area(args ...any) *VNode    { return createElement("area", args) }

// Interactive elements

func Details(args ...any) *VNode { return createElement("details", args) }
func Summary(args ...any) *VNode { return createElement("summary", args) }
func Dialog(args ...any) *VNode  { return createElement("dialog", args) }
func Menu(args ...any) *VNode    { return createElement("menu", args) }

// Scripting elements

func Script(args ...any) *VNode   { return createElement("script", args) }
func Noscript(args ...any) *VNode { return createElement("noscript", args) }
func Template(args ...any) *VNode { return createElement("template", args) }
func Slot(args ...any) *VNode     { return createElement("slot", args) }
func Style(args ...any) *VNode    { return createElement("style", args) }

// CustomElement creates an element with a custom tag name.
func CustomElement(tag string, args ...any) *VNode {
	return createElement(tag, args)
}
