package vdom

import "testing"

func TestGlobalAttributes(t *testing.T) {
	tests := []struct {
		name  string
		attr  Attr
		key   string
		value any
	}{
		{"ID", ID("main"), "id", "main"},
		{"Class single", Class("card"), "class", "card"},
		{"Class multiple", Class("card", "active"), "class", "card active"},
		{"StyleAttr", StyleAttr("color: red"), "style", "color: red"},
		{"DataAttr", DataAttr("id", "123"), "data-id", "123"},
		{"Role", Role("button"), "role", "button"},
		{"AriaLabel", AriaLabel("Close"), "aria-label", "Close"},
		{"AriaHidden true", AriaHidden(true), "aria-hidden", true},
		{"AriaHidden false", AriaHidden(false), "aria-hidden", false},
		{"AriaExpanded", AriaExpanded(true), "aria-expanded", true},
		{"TabIndex", TabIndex(0), "tabindex", 0},
		{"TabIndex negative", TabIndex(-1), "tabindex", -1},
		{"Hidden", Hidden(), "hidden", true},
		{"TitleAttr", TitleAttr("Tooltip"), "title", "Tooltip"},
		{"Lang", Lang("en"), "lang", "en"},
		{"Dir", Dir("ltr"), "dir", "ltr"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.attr.Key != tt.key {
				t.Errorf("Key = %v, want %v", tt.attr.Key, tt.key)
			}
			if tt.attr.Value != tt.value {
				t.Errorf("Value = %v, want %v", tt.attr.Value, tt.value)
			}
		})
	}
}

func TestLinkAttributes(t *testing.T) {
	tests := []struct {
		name  string
		attr  Attr
		key   string
		value any
	}{
		{"Href", Href("/page"), "href", "/page"},
		{"Target", Target("_blank"), "target", "_blank"},
		{"Rel", Rel("noopener"), "rel", "noopener"},
		{"Download empty", Download(), "download", true},
		{"Download filename", Download("file.pdf"), "download", "file.pdf"},
		{"Hreflang", Hreflang("en"), "hreflang", "en"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.attr.Key != tt.key {
				t.Errorf("Key = %v, want %v", tt.attr.Key, tt.key)
			}
			if tt.attr.Value != tt.value {
				t.Errorf("Value = %v, want %v", tt.attr.Value, tt.value)
			}
		})
	}
}

func TestFormAttributes(t *testing.T) {
	tests := []struct {
		name  string
		attr  Attr
		key   string
		value any
	}{
		{"Name", Name("email"), "name", "email"},
		{"Value", Value("test"), "value", "test"},
		{"Type", Type("email"), "type", "email"},
		{"Placeholder", Placeholder("Enter..."), "placeholder", "Enter..."},
		{"Disabled", Disabled(), "disabled", true},
		{"Readonly", Readonly(), "readonly", true},
		{"Required", Required(), "required", true},
		{"Checked", Checked(), "checked", true},
		{"Selected", Selected(), "selected", true},
		{"Multiple", Multiple(), "multiple", true},
		{"Autofocus", Autofocus(), "autofocus", true},
		{"Autocomplete", Autocomplete("off"), "autocomplete", "off"},
		{"Pattern", Pattern("[0-9]+"), "pattern", "[0-9]+"},
		{"MinLength", MinLength(3), "minlength", 3},
		{"MaxLength", MaxLength(100), "maxlength", 100},
		{"Min", Min("0"), "min", "0"},
		{"Max", Max("100"), "max", "100"},
		{"Step", Step("0.1"), "step", "0.1"},
		{"Rows", Rows(5), "rows", 5},
		{"Cols", Cols(40), "cols", 40},
		{"For", For("email-input"), "for", "email-input"},
		{"Action", Action("/submit"), "action", "/submit"},
		{"Method", Method("post"), "method", "post"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.attr.Key != tt.key {
				t.Errorf("Key = %v, want %v", tt.attr.Key, tt.key)
			}
			if tt.attr.Value != tt.value {
				t.Errorf("Value = %v, want %v", tt.attr.Value, tt.value)
			}
		})
	}
}

func TestMediaAttributes(t *testing.T) {
	tests := []struct {
		name  string
		attr  Attr
		key   string
		value any
	}{
		{"Src", Src("/image.png"), "src", "/image.png"},
		{"Alt", Alt("An image"), "alt", "An image"},
		{"Width", Width(100), "width", 100},
		{"Height", Height(200), "height", 200},
		{"Loading", Loading("lazy"), "loading", "lazy"},
		{"Controls", Controls(), "controls", true},
		{"Autoplay", Autoplay(), "autoplay", true},
		{"Loop", Loop(), "loop", true},
		{"MutedAttr", MutedAttr(), "muted", true},
		{"Poster", Poster("/thumb.jpg"), "poster", "/thumb.jpg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.attr.Key != tt.key {
				t.Errorf("Key = %v, want %v", tt.attr.Key, tt.key)
			}
			if tt.attr.Value != tt.value {
				t.Errorf("Value = %v, want %v", tt.attr.Value, tt.value)
			}
		})
	}
}

func TestTableAttributes(t *testing.T) {
	tests := []struct {
		name  string
		attr  Attr
		key   string
		value any
	}{
		{"Colspan", Colspan(2), "colspan", 2},
		{"Rowspan", Rowspan(3), "rowspan", 3},
		{"Scope", Scope("col"), "scope", "col"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.attr.Key != tt.key {
				t.Errorf("Key = %v, want %v", tt.attr.Key, tt.key)
			}
			if tt.attr.Value != tt.value {
				t.Errorf("Value = %v, want %v", tt.attr.Value, tt.value)
			}
		})
	}
}

func TestConditionalAttributes(t *testing.T) {
	t.Run("ClassIf true", func(t *testing.T) {
		attr := ClassIf(true, "active")
		if attr.Key != "class" {
			t.Errorf("Key = %v, want class", attr.Key)
		}
		if attr.Value != "active" {
			t.Errorf("Value = %v, want active", attr.Value)
		}
	})

	t.Run("ClassIf false", func(t *testing.T) {
		attr := ClassIf(false, "active")
		if !attr.IsEmpty() {
			t.Error("Expected empty attr when condition is false")
		}
	})

	t.Run("AttrIf true", func(t *testing.T) {
		attr := AttrIf(true, Disabled())
		if attr.Key != "disabled" {
			t.Errorf("Key = %v, want disabled", attr.Key)
		}
	})

	t.Run("AttrIf false", func(t *testing.T) {
		attr := AttrIf(false, Disabled())
		if !attr.IsEmpty() {
			t.Error("Expected empty attr when condition is false")
		}
	})

	t.Run("Classes with strings", func(t *testing.T) {
		attr := Classes("card", "active", "large")
		if attr.Value != "card active large" {
			t.Errorf("Value = %v, want 'card active large'", attr.Value)
		}
	})

	t.Run("Classes with slice", func(t *testing.T) {
		attr := Classes([]string{"card", "active"})
		if attr.Value != "card active" {
			t.Errorf("Value = %v, want 'card active'", attr.Value)
		}
	})

	t.Run("Classes with map", func(t *testing.T) {
		attr := Classes(map[string]bool{
			"card":     true,
			"active":   true,
			"disabled": false,
		})
		// Map order is not guaranteed, so we check length
		val := attr.Value.(string)
		if len(val) == 0 {
			t.Error("Expected non-empty class value")
		}
	})

	t.Run("Classes with empty strings filtered", func(t *testing.T) {
		attr := Classes("card", "", "active")
		if attr.Value != "card active" {
			t.Errorf("Value = %v, want 'card active'", attr.Value)
		}
	})

	t.Run("Classes mixed", func(t *testing.T) {
		attr := Classes("base", []string{"one", "two"}, map[string]bool{"three": true, "four": false})
		val := attr.Value.(string)
		if val == "" {
			t.Error("Expected non-empty class value")
		}
	})
}

func TestEmptyAttrIgnored(t *testing.T) {
	node := Div(ClassIf(false, "hidden"), Class("visible"))
	if node.Props["class"] != "visible" {
		t.Errorf("class = %v, want visible", node.Props["class"])
	}
}

func TestMoreAttributes(t *testing.T) {
	tests := []struct {
		name  string
		attr  Attr
		key   string
		value any
	}{
		// More ARIA attributes
		{"AriaDescribedBy", AriaDescribedBy("desc"), "aria-describedby", "desc"},
		{"AriaLabelledBy", AriaLabelledBy("label"), "aria-labelledby", "label"},
		{"AriaLive", AriaLive("polite"), "aria-live", "polite"},
		{"AriaControls", AriaControls("menu"), "aria-controls", "menu"},
		{"AriaCurrent", AriaCurrent("page"), "aria-current", "page"},
		{"AriaDisabled", AriaDisabled(true), "aria-disabled", true},
		{"AriaPressed", AriaPressed("true"), "aria-pressed", "true"},
		{"AriaSelected", AriaSelected(true), "aria-selected", true},
		{"AriaHasPopup", AriaHasPopup("menu"), "aria-haspopup", "menu"},
		{"AriaModal", AriaModal(true), "aria-modal", true},
		{"AriaAtomic", AriaAtomic(true), "aria-atomic", true},
		{"AriaBusy", AriaBusy(false), "aria-busy", false},
		{"AriaValueNow", AriaValueNow(50), "aria-valuenow", 50.0},
		{"AriaValueMin", AriaValueMin(0), "aria-valuemin", 0.0},
		{"AriaValueMax", AriaValueMax(100), "aria-valuemax", 100.0},

		// Keyboard
		{"AccessKey", AccessKey("s"), "accesskey", "s"},

		// Behavior
		{"ContentEditable", ContentEditable(true), "contenteditable", true},
		{"Draggable", Draggable(), "draggable", "true"},
		{"Spellcheck", Spellcheck(false), "spellcheck", false},

		// File input
		{"Accept", Accept("image/*"), "accept", "image/*"},
		{"Capture", Capture("user"), "capture", "user"},

		// Textarea
		{"Wrap", Wrap("soft"), "wrap", "soft"},

		// Form element
		{"Enctype", Enctype("multipart/form-data"), "enctype", "multipart/form-data"},
		{"Novalidate", Novalidate(), "novalidate", true},
		{"FormAttr", FormAttr("myform"), "form", "myform"},

		// Media
		{"Decoding", Decoding("async"), "decoding", "async"},
		{"Srcset", Srcset("img-320.jpg 320w, img-640.jpg 640w"), "srcset", "img-320.jpg 320w, img-640.jpg 640w"},
		{"SizesAttr", SizesAttr("(max-width: 320px) 280px"), "sizes", "(max-width: 320px) 280px"},
		{"Preload", Preload("auto"), "preload", "auto"},
		{"Playsinline", Playsinline(), "playsinline", true},

		// Iframe
		{"Sandbox", Sandbox("allow-scripts"), "sandbox", "allow-scripts"},
		{"Allow", Allow("fullscreen"), "allow", "fullscreen"},
		{"Allowfullscreen", Allowfullscreen(), "allowfullscreen", true},

		// Table
		{"HeadersAttr", HeadersAttr("col1 col2"), "headers", "col1 col2"},

		// Meta/Link
		{"Charset", Charset("utf-8"), "charset", "utf-8"},
		{"Content", Content("description"), "content", "description"},
		{"HttpEquiv", HttpEquiv("refresh"), "http-equiv", "refresh"},

		// More attributes
		{"Open", Open(), "open", true},
		{"Defer_", Defer_(), "defer", true},
		{"Async", Async(), "async", true},
		{"Crossorigin", Crossorigin("anonymous"), "crossorigin", "anonymous"},
		{"Integrity", Integrity("sha384-abc"), "integrity", "sha384-abc"},
		{"List", List("datalist1"), "list", "datalist1"},
		{"Inputmode", Inputmode("numeric"), "inputmode", "numeric"},
		{"Enterkeyhint", Enterkeyhint("search"), "enterkeyhint", "search"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.attr.Key != tt.key {
				t.Errorf("Key = %v, want %v", tt.attr.Key, tt.key)
			}
			if tt.attr.Value != tt.value {
				t.Errorf("Value = %v, want %v", tt.attr.Value, tt.value)
			}
		})
	}
}
