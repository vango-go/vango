package render

// voidElements are elements that cannot have children and have no closing tag.
// These are self-closing in HTML5.
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

// isVoidElement returns true if the tag is a void element.
func isVoidElement(tag string) bool {
	return voidElements[tag]
}

// inlineElements are elements that are typically rendered inline
// and don't need newlines in pretty-printed output.
var inlineElements = map[string]bool{
	"a":      true,
	"abbr":   true,
	"b":      true,
	"bdi":    true,
	"bdo":    true,
	"br":     true,
	"cite":   true,
	"code":   true,
	"data":   true,
	"dfn":    true,
	"em":     true,
	"i":      true,
	"kbd":    true,
	"mark":   true,
	"q":      true,
	"rb":     true,
	"rp":     true,
	"rt":     true,
	"rtc":    true,
	"ruby":   true,
	"s":      true,
	"samp":   true,
	"small":  true,
	"span":   true,
	"strong": true,
	"sub":    true,
	"sup":    true,
	"time":   true,
	"u":      true,
	"var":    true,
	"wbr":    true,
}

// isInlineElement returns true if the tag is an inline element.
func isInlineElement(tag string) bool {
	return inlineElements[tag]
}

// booleanAttrs are attributes that don't need a value.
// When true, they're rendered as just the attribute name.
var booleanAttrs = map[string]bool{
	"allowfullscreen": true,
	"async":           true,
	"autofocus":       true,
	"autoplay":        true,
	"checked":         true,
	"controls":        true,
	"default":         true,
	"defer":           true,
	"disabled":        true,
	"formnovalidate":  true,
	"hidden":          true,
	"ismap":           true,
	"itemscope":       true,
	"loop":            true,
	"multiple":        true,
	"muted":           true,
	"nomodule":        true,
	"novalidate":      true,
	"open":            true,
	"playsinline":     true,
	"readonly":        true,
	"required":        true,
	"reversed":        true,
	"selected":        true,
}

// isBooleanAttr returns true if the attribute is a boolean attribute.
func isBooleanAttr(name string) bool {
	return booleanAttrs[name]
}
