package vdom

import "strings"

// attr creates an Attr with the given key and value.
func attr(key string, value any) Attr {
	return Attr{Key: key, Value: value}
}

// Identity attributes

// ID sets the id attribute.
func ID(id string) Attr { return attr("id", id) }

// Class sets the class attribute, joining multiple classes with spaces.
func Class(classes ...string) Attr { return attr("class", strings.Join(classes, " ")) }

// StyleAttr sets the style attribute (named to avoid conflict with Style element).
func StyleAttr(style string) Attr { return attr("style", style) }

// Data attributes

// Data creates a data-* attribute. This is the primary way to add data attributes.
// Example: Data("id", "123") → data-id="123"
func Data(key, value string) Attr { return attr("data-"+key, value) }

// DataAttr is an alias for Data(). Provided for backwards compatibility.
func DataAttr(key, value string) Attr { return Data(key, value) }

// Accessibility attributes

// Role sets the role attribute.
func Role(role string) Attr { return attr("role", role) }

// AriaLabel sets the aria-label attribute.
func AriaLabel(label string) Attr { return attr("aria-label", label) }

// AriaHidden sets the aria-hidden attribute.
func AriaHidden(hidden bool) Attr { return attr("aria-hidden", hidden) }

// AriaExpanded sets the aria-expanded attribute.
func AriaExpanded(expanded bool) Attr { return attr("aria-expanded", expanded) }

// AriaDescribedBy sets the aria-describedby attribute.
func AriaDescribedBy(id string) Attr { return attr("aria-describedby", id) }

// AriaLabelledBy sets the aria-labelledby attribute.
func AriaLabelledBy(id string) Attr { return attr("aria-labelledby", id) }

// AriaLive sets the aria-live attribute.
func AriaLive(mode string) Attr { return attr("aria-live", mode) }

// AriaControls sets the aria-controls attribute.
func AriaControls(id string) Attr { return attr("aria-controls", id) }

// AriaCurrent sets the aria-current attribute.
func AriaCurrent(value string) Attr { return attr("aria-current", value) }

// AriaDisabled sets the aria-disabled attribute.
func AriaDisabled(disabled bool) Attr { return attr("aria-disabled", disabled) }

// AriaPressed sets the aria-pressed attribute.
func AriaPressed(pressed string) Attr { return attr("aria-pressed", pressed) }

// AriaSelected sets the aria-selected attribute.
func AriaSelected(selected bool) Attr { return attr("aria-selected", selected) }

// AriaHasPopup sets the aria-haspopup attribute.
func AriaHasPopup(value string) Attr { return attr("aria-haspopup", value) }

// AriaModal sets the aria-modal attribute.
func AriaModal(modal bool) Attr { return attr("aria-modal", modal) }

// AriaAtomic sets the aria-atomic attribute.
func AriaAtomic(atomic bool) Attr { return attr("aria-atomic", atomic) }

// AriaBusy sets the aria-busy attribute.
func AriaBusy(busy bool) Attr { return attr("aria-busy", busy) }

// AriaValueNow sets the aria-valuenow attribute.
func AriaValueNow(value float64) Attr { return attr("aria-valuenow", value) }

// AriaValueMin sets the aria-valuemin attribute.
func AriaValueMin(value float64) Attr { return attr("aria-valuemin", value) }

// AriaValueMax sets the aria-valuemax attribute.
func AriaValueMax(value float64) Attr { return attr("aria-valuemax", value) }

// Keyboard attributes

// TabIndex sets the tabindex attribute.
func TabIndex(index int) Attr { return attr("tabindex", index) }

// AccessKey sets the accesskey attribute.
func AccessKey(key string) Attr { return attr("accesskey", key) }

// Visibility attributes

// Hidden sets the hidden attribute.
func Hidden() Attr { return attr("hidden", true) }

// TitleAttr sets the title attribute (named to avoid conflict with Title element).
func TitleAttr(title string) Attr { return attr("title", title) }

// Behavior attributes

// ContentEditable sets the contenteditable attribute.
func ContentEditable(editable bool) Attr { return attr("contenteditable", editable) }

// Draggable sets the draggable attribute.
func Draggable() Attr { return attr("draggable", "true") }

// Spellcheck sets the spellcheck attribute.
func Spellcheck(check bool) Attr { return attr("spellcheck", check) }

// Language attributes

// Lang sets the lang attribute.
func Lang(lang string) Attr { return attr("lang", lang) }

// Dir sets the dir attribute.
func Dir(dir string) Attr { return attr("dir", dir) }

// Link attributes

// Href sets the href attribute.
func Href(url string) Attr { return attr("href", url) }

// Target sets the target attribute.
func Target(target string) Attr { return attr("target", target) }

// Rel sets the rel attribute.
func Rel(rel string) Attr { return attr("rel", rel) }

// Download sets the download attribute.
func Download(filename ...string) Attr {
	if len(filename) > 0 {
		return attr("download", filename[0])
	}
	return attr("download", true)
}

// Hreflang sets the hreflang attribute.
func Hreflang(lang string) Attr { return attr("hreflang", lang) }

// Form input attributes

// Name sets the name attribute.
func Name(name string) Attr { return attr("name", name) }

// Value sets the value attribute.
func Value(value string) Attr { return attr("value", value) }

// Type sets the type attribute.
func Type(t string) Attr { return attr("type", t) }

// Placeholder sets the placeholder attribute.
func Placeholder(text string) Attr { return attr("placeholder", text) }

// Form state attributes

// Disabled sets the disabled attribute.
func Disabled() Attr { return attr("disabled", true) }

// Readonly sets the readonly attribute.
func Readonly() Attr { return attr("readonly", true) }

// Required sets the required attribute.
func Required() Attr { return attr("required", true) }

// Checked sets the checked attribute.
func Checked() Attr { return attr("checked", true) }

// Selected sets the selected attribute.
func Selected() Attr { return attr("selected", true) }

// Multiple sets the multiple attribute.
func Multiple() Attr { return attr("multiple", true) }

// Autofocus sets the autofocus attribute.
func Autofocus() Attr { return attr("autofocus", true) }

// Autocomplete sets the autocomplete attribute.
func Autocomplete(value string) Attr { return attr("autocomplete", value) }

// Form validation attributes

// Pattern sets the pattern attribute.
func Pattern(pattern string) Attr { return attr("pattern", pattern) }

// MinLength sets the minlength attribute.
func MinLength(n int) Attr { return attr("minlength", n) }

// MaxLength sets the maxlength attribute.
func MaxLength(n int) Attr { return attr("maxlength", n) }

// Min sets the min attribute.
func Min(value string) Attr { return attr("min", value) }

// Max sets the max attribute.
func Max(value string) Attr { return attr("max", value) }

// Step sets the step attribute.
func Step(value string) Attr { return attr("step", value) }

// File input attributes

// Accept sets the accept attribute.
func Accept(types string) Attr { return attr("accept", types) }

// Capture sets the capture attribute.
func Capture(mode string) Attr { return attr("capture", mode) }

// Textarea attributes

// Rows sets the rows attribute.
func Rows(n int) Attr { return attr("rows", n) }

// Cols sets the cols attribute.
func Cols(n int) Attr { return attr("cols", n) }

// Wrap sets the wrap attribute.
func Wrap(mode string) Attr { return attr("wrap", mode) }

// Form element attributes

// Action sets the action attribute.
func Action(url string) Attr { return attr("action", url) }

// Method sets the method attribute.
func Method(method string) Attr { return attr("method", method) }

// Enctype sets the enctype attribute.
func Enctype(enctype string) Attr { return attr("enctype", enctype) }

// Novalidate sets the novalidate attribute.
func Novalidate() Attr { return attr("novalidate", true) }

// For sets the for attribute (for labels).
func For(id string) Attr { return attr("for", id) }

// FormAttr sets the form attribute (to associate with a form by id).
func FormAttr(id string) Attr { return attr("form", id) }

// Media attributes

// Src sets the src attribute.
func Src(url string) Attr { return attr("src", url) }

// Alt sets the alt attribute.
func Alt(text string) Attr { return attr("alt", text) }

// Width sets the width attribute.
func Width(w int) Attr { return attr("width", w) }

// Height sets the height attribute.
func Height(h int) Attr { return attr("height", h) }

// Loading sets the loading attribute.
func Loading(mode string) Attr { return attr("loading", mode) }

// Decoding sets the decoding attribute.
func Decoding(mode string) Attr { return attr("decoding", mode) }

// Srcset sets the srcset attribute.
func Srcset(srcset string) Attr { return attr("srcset", srcset) }

// Sizes sets the sizes attribute.
func SizesAttr(sizes string) Attr { return attr("sizes", sizes) }

// Video/Audio attributes

// Controls sets the controls attribute.
func Controls() Attr { return attr("controls", true) }

// Autoplay sets the autoplay attribute.
func Autoplay() Attr { return attr("autoplay", true) }

// Loop sets the loop attribute.
func Loop() Attr { return attr("loop", true) }

// Muted sets the muted attribute.
func MutedAttr() Attr { return attr("muted", true) }

// Preload sets the preload attribute.
func Preload(mode string) Attr { return attr("preload", mode) }

// Poster sets the poster attribute.
func Poster(url string) Attr { return attr("poster", url) }

// Playsinline sets the playsinline attribute.
func Playsinline() Attr { return attr("playsinline", true) }

// Iframe attributes

// Sandbox sets the sandbox attribute.
func Sandbox(value string) Attr { return attr("sandbox", value) }

// Allow sets the allow attribute.
func Allow(value string) Attr { return attr("allow", value) }

// Allowfullscreen sets the allowfullscreen attribute.
func Allowfullscreen() Attr { return attr("allowfullscreen", true) }

// Table attributes

// Colspan sets the colspan attribute.
func Colspan(n int) Attr { return attr("colspan", n) }

// Rowspan sets the rowspan attribute.
func Rowspan(n int) Attr { return attr("rowspan", n) }

// Scope sets the scope attribute.
func Scope(scope string) Attr { return attr("scope", scope) }

// Headers sets the headers attribute.
func HeadersAttr(ids string) Attr { return attr("headers", ids) }

// Meta/Link attributes

// Charset sets the charset attribute.
func Charset(charset string) Attr { return attr("charset", charset) }

// Content sets the content attribute.
func Content(content string) Attr { return attr("content", content) }

// HttpEquiv sets the http-equiv attribute.
func HttpEquiv(value string) Attr { return attr("http-equiv", value) }

// Conditional attributes

// ClassIf adds a class conditionally.
func ClassIf(condition bool, class string) Attr {
	if condition {
		return attr("class", class)
	}
	return Attr{} // Empty attr, will be ignored
}

// AttrIf adds any attribute conditionally.
func AttrIf(condition bool, a Attr) Attr {
	if condition {
		return a
	}
	return Attr{}
}

// Classes merges multiple class values.
// Accepts string, []string, and map[string]bool.
func Classes(classes ...any) Attr {
	var result []string
	for _, c := range classes {
		switch v := c.(type) {
		case string:
			if v != "" {
				result = append(result, v)
			}
		case []string:
			for _, s := range v {
				if s != "" {
					result = append(result, s)
				}
			}
		case map[string]bool:
			for class, include := range v {
				if include && class != "" {
					result = append(result, class)
				}
			}
		}
	}
	return attr("class", strings.Join(result, " "))
}

// Open sets the open attribute (for details, dialog).
func Open() Attr { return attr("open", true) }

// Defer_ sets the defer attribute for script elements.
func Defer_() Attr { return attr("defer", true) }

// Async sets the async attribute for script elements.
func Async() Attr { return attr("async", true) }

// Crossorigin sets the crossorigin attribute.
func Crossorigin(value string) Attr { return attr("crossorigin", value) }

// Integrity sets the integrity attribute for subresource integrity.
func Integrity(value string) Attr { return attr("integrity", value) }

// List sets the list attribute (for input with datalist).
func List(id string) Attr { return attr("list", id) }

// Inputmode sets the inputmode attribute.
func Inputmode(mode string) Attr { return attr("inputmode", mode) }

// Enterkeyhint sets the enterkeyhint attribute.
func Enterkeyhint(hint string) Attr { return attr("enterkeyhint", hint) }
