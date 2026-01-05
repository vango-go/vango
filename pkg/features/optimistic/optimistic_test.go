package optimistic

import (
	"strings"
	"testing"

	"github.com/vango-dev/vango/v2/pkg/render"
)

// contains is a helper for checking substring presence
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestClass(t *testing.T) {
	tests := []struct {
		name         string
		class        string
		addNotRemove bool
		wantClass    string // Expected Class field in OptimisticConfig
	}{
		{
			name:         "add class",
			class:        "liked",
			addNotRemove: true,
			wantClass:    "liked:add",
		},
		{
			name:         "remove class",
			class:        "liked",
			addNotRemove: false,
			wantClass:    "liked:remove",
		},
		{
			name:         "add class with hyphen",
			class:        "is-active",
			addNotRemove: true,
			wantClass:    "is-active:add",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attr := Class(tt.class, tt.addNotRemove)
			if attr.Key != "_optimistic" {
				t.Errorf("Class() key = %q, want %q", attr.Key, "_optimistic")
			}
			config, ok := attr.Value.(render.OptimisticConfig)
			if !ok {
				t.Fatalf("Class() value is not render.OptimisticConfig, got %T", attr.Value)
			}
			if config.Class != tt.wantClass {
				t.Errorf("Class() config.Class = %q, want %q", config.Class, tt.wantClass)
			}
		})
	}
}

func TestClassToggle(t *testing.T) {
	attr := ClassToggle("active")
	if attr.Key != "_optimistic" {
		t.Errorf("ClassToggle() key = %q, want %q", attr.Key, "_optimistic")
	}
	config, ok := attr.Value.(render.OptimisticConfig)
	if !ok {
		t.Fatalf("ClassToggle() value is not render.OptimisticConfig, got %T", attr.Value)
	}
	if config.Class != "active:toggle" {
		t.Errorf("ClassToggle() config.Class = %q, want %q", config.Class, "active:toggle")
	}
}

func TestText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		wantText string
	}{
		{
			name:     "simple text",
			text:     "Saving...",
			wantText: "Saving...",
		},
		{
			name:     "number text",
			text:     "42",
			wantText: "42",
		},
		{
			name:     "empty text",
			text:     "",
			wantText: "",
		},
		{
			name:     "text with special chars",
			text:     "Hello <World>",
			wantText: "Hello <World>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attr := Text(tt.text)
			if attr.Key != "_optimistic" {
				t.Errorf("Text() key = %q, want %q", attr.Key, "_optimistic")
			}
			config, ok := attr.Value.(render.OptimisticConfig)
			if !ok {
				t.Fatalf("Text() value is not render.OptimisticConfig, got %T", attr.Value)
			}
			if config.Text != tt.wantText {
				t.Errorf("Text() config.Text = %q, want %q", config.Text, tt.wantText)
			}
		})
	}
}

func TestAttr(t *testing.T) {
	tests := []struct {
		name          string
		attrName      string
		attrValue     string
		wantAttrName  string
		wantAttrValue string
	}{
		{
			name:          "checked attribute",
			attrName:      "checked",
			attrValue:     "true",
			wantAttrName:  "checked",
			wantAttrValue: "true",
		},
		{
			name:          "aria attribute",
			attrName:      "aria-expanded",
			attrValue:     "true",
			wantAttrName:  "aria-expanded",
			wantAttrValue: "true",
		},
		{
			name:          "data attribute",
			attrName:      "data-state",
			attrValue:     "active",
			wantAttrName:  "data-state",
			wantAttrValue: "active",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attr := Attr(tt.attrName, tt.attrValue)
			if attr.Key != "_optimistic" {
				t.Errorf("Attr() key = %q, want %q", attr.Key, "_optimistic")
			}
			config, ok := attr.Value.(render.OptimisticConfig)
			if !ok {
				t.Fatalf("Attr() value is not render.OptimisticConfig, got %T", attr.Value)
			}
			if config.Attr != tt.wantAttrName {
				t.Errorf("Attr() config.Attr = %q, want %q", config.Attr, tt.wantAttrName)
			}
			if config.Value != tt.wantAttrValue {
				t.Errorf("Attr() config.Value = %q, want %q", config.Value, tt.wantAttrValue)
			}
		})
	}
}

func TestAttrRemove(t *testing.T) {
	attr := AttrRemove("disabled")
	if attr.Key != "_optimistic" {
		t.Errorf("AttrRemove() key = %q, want %q", attr.Key, "_optimistic")
	}
	config, ok := attr.Value.(render.OptimisticConfig)
	if !ok {
		t.Fatalf("AttrRemove() value is not render.OptimisticConfig, got %T", attr.Value)
	}
	if config.Attr != "disabled" {
		t.Errorf("AttrRemove() config.Attr = %q, want %q", config.Attr, "disabled")
	}
	if config.Value != "" {
		t.Errorf("AttrRemove() config.Value = %q, want empty string", config.Value)
	}
}

func TestParentClass(t *testing.T) {
	tests := []struct {
		name         string
		class        string
		addNotRemove bool
		wantValue    string
	}{
		{
			name:         "add parent class",
			class:        "open",
			addNotRemove: true,
			wantValue:    "open:add",
		},
		{
			name:         "remove parent class",
			class:        "open",
			addNotRemove: false,
			wantValue:    "open:remove",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attr := ParentClass(tt.class, tt.addNotRemove)
			if attr.Key != "data-optimistic-parent-class" {
				t.Errorf("ParentClass() key = %q, want %q", attr.Key, "data-optimistic-parent-class")
			}
			if attr.Value != tt.wantValue {
				t.Errorf("ParentClass() value = %q, want %q", attr.Value, tt.wantValue)
			}
		})
	}
}

func TestParentClassToggle(t *testing.T) {
	attr := ParentClassToggle("expanded")
	if attr.Key != "data-optimistic-parent-class" {
		t.Errorf("ParentClassToggle() key = %q, want %q", attr.Key, "data-optimistic-parent-class")
	}
	if attr.Value != "expanded:toggle" {
		t.Errorf("ParentClassToggle() value = %q, want %q", attr.Value, "expanded:toggle")
	}
}

func TestDisable(t *testing.T) {
	attr := Disable()
	if attr.Key != "data-optimistic-disable" {
		t.Errorf("Disable() key = %q, want %q", attr.Key, "data-optimistic-disable")
	}
	if attr.Value != "true" {
		t.Errorf("Disable() value = %q, want %q", attr.Value, "true")
	}
}

func TestHide(t *testing.T) {
	attr := Hide()
	if attr.Key != "data-optimistic-hide" {
		t.Errorf("Hide() key = %q, want %q", attr.Key, "data-optimistic-hide")
	}
	if attr.Value != "true" {
		t.Errorf("Hide() value = %q, want %q", attr.Value, "true")
	}
}

func TestShow(t *testing.T) {
	attr := Show()
	if attr.Key != "data-optimistic-show" {
		t.Errorf("Show() key = %q, want %q", attr.Key, "data-optimistic-show")
	}
	if attr.Value != "true" {
		t.Errorf("Show() value = %q, want %q", attr.Value, "true")
	}
}

func TestLoading(t *testing.T) {
	tests := []struct {
		name      string
		disable   bool
		wantValue string
	}{
		{
			name:      "loading without disable",
			disable:   false,
			wantValue: "class",
		},
		{
			name:      "loading with disable",
			disable:   true,
			wantValue: "class,disable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attr := Loading(tt.disable)
			if attr.Key != "data-optimistic-loading" {
				t.Errorf("Loading() key = %q, want %q", attr.Key, "data-optimistic-loading")
			}
			if attr.Value != tt.wantValue {
				t.Errorf("Loading() value = %q, want %q", attr.Value, tt.wantValue)
			}
		})
	}
}

func TestMultiple(t *testing.T) {
	attr := Multiple(
		Class("submitting", true),
		Text("Submitting..."),
		Disable(),
	)

	if attr.Key != "data-optimistic-multi" {
		t.Errorf("Multiple() key = %q, want %q", attr.Key, "data-optimistic-multi")
	}

	// Value should contain all three parts
	value := attr.Value.(string)
	if value == "" {
		t.Error("Multiple() value is empty")
	}

	// Should contain each attribute - verify it has multiple parts separated by ;
	if !contains(value, ";") {
		t.Error("Multiple() should contain multiple parts separated by ;")
	}
}

func TestTarget(t *testing.T) {
	tests := []struct {
		name     string
		selector string
	}{
		{
			name:     "id selector",
			selector: "#header",
		},
		{
			name:     "class selector",
			selector: ".content",
		},
		{
			name:     "hid selector",
			selector: "[data-hid='h42']",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attr := Target(tt.selector)
			if attr.Key != "data-optimistic-target" {
				t.Errorf("Target() key = %q, want %q", attr.Key, "data-optimistic-target")
			}
			if attr.Value != tt.selector {
				t.Errorf("Target() value = %q, want %q", attr.Value, tt.selector)
			}
		})
	}
}

func TestSwap(t *testing.T) {
	html := "<span class='loading'>Loading...</span>"
	attr := Swap(html)

	if attr.Key != "data-optimistic-swap" {
		t.Errorf("Swap() key = %q, want %q", attr.Key, "data-optimistic-swap")
	}
	if attr.Value != html {
		t.Errorf("Swap() value = %q, want %q", attr.Value, html)
	}
}

func TestParseClassAction(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		wantClass  string
		wantAction Action
		wantOk     bool
	}{
		{
			name:       "add action",
			value:      "liked:add",
			wantClass:  "liked",
			wantAction: ActionAdd,
			wantOk:     true,
		},
		{
			name:       "remove action",
			value:      "active:remove",
			wantClass:  "active",
			wantAction: ActionRemove,
			wantOk:     true,
		},
		{
			name:       "toggle action",
			value:      "open:toggle",
			wantClass:  "open",
			wantAction: ActionToggle,
			wantOk:     true,
		},
		{
			name:       "set action",
			value:      "state:set",
			wantClass:  "state",
			wantAction: ActionSet,
			wantOk:     true,
		},
		{
			name:       "invalid format - no colon",
			value:      "liked",
			wantClass:  "",
			wantAction: "",
			wantOk:     false,
		},
		{
			name:       "invalid action",
			value:      "liked:invalid",
			wantClass:  "",
			wantAction: "",
			wantOk:     false,
		},
		{
			name:       "empty string",
			value:      "",
			wantClass:  "",
			wantAction: "",
			wantOk:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			class, action, ok := ParseClassAction(tt.value)
			if ok != tt.wantOk {
				t.Errorf("ParseClassAction() ok = %v, want %v", ok, tt.wantOk)
			}
			if ok {
				if class != tt.wantClass {
					t.Errorf("ParseClassAction() class = %q, want %q", class, tt.wantClass)
				}
				if action != tt.wantAction {
					t.Errorf("ParseClassAction() action = %q, want %q", action, tt.wantAction)
				}
			}
		})
	}
}

func TestParseAttrAction(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantName  string
		wantValue string
		wantOk    bool
	}{
		{
			name:      "simple attribute",
			value:     "checked:true",
			wantName:  "checked",
			wantValue: "true",
			wantOk:    true,
		},
		{
			name:      "attribute with colons in value",
			value:     "data-url:https://example.com",
			wantName:  "data-url",
			wantValue: "https://example.com",
			wantOk:    true,
		},
		{
			name:      "invalid format",
			value:     "nocolon",
			wantName:  "",
			wantValue: "",
			wantOk:    false,
		},
		{
			name:      "empty string",
			value:     "",
			wantName:  "",
			wantValue: "",
			wantOk:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, value, ok := ParseAttrAction(tt.value)
			if ok != tt.wantOk {
				t.Errorf("ParseAttrAction() ok = %v, want %v", ok, tt.wantOk)
			}
			if ok {
				if name != tt.wantName {
					t.Errorf("ParseAttrAction() name = %q, want %q", name, tt.wantName)
				}
				if value != tt.wantValue {
					t.Errorf("ParseAttrAction() value = %q, want %q", value, tt.wantValue)
				}
			}
		})
	}
}

func TestParseMultiple(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantCount int
	}{
		{
			name:      "single item",
			value:     "key=value",
			wantCount: 1,
		},
		{
			name:      "multiple items",
			value:     "key1=value1;key2=value2;key3=value3",
			wantCount: 3,
		},
		{
			name:      "empty string",
			value:     "",
			wantCount: 0,
		},
		{
			name:      "invalid format - no equals",
			value:     "noeq",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseMultiple(tt.value)
			if len(result) != tt.wantCount {
				t.Errorf("ParseMultiple() count = %d, want %d", len(result), tt.wantCount)
			}
		})
	}
}

func TestParseMultipleValues(t *testing.T) {
	value := "data-optimistic-class=liked:add;data-optimistic-text=42"
	result := ParseMultiple(value)

	if len(result) != 2 {
		t.Fatalf("ParseMultiple() count = %d, want 2", len(result))
	}

	if result[0].Key != "data-optimistic-class" {
		t.Errorf("ParseMultiple()[0].Key = %q, want %q", result[0].Key, "data-optimistic-class")
	}
	if result[0].Value != "liked:add" {
		t.Errorf("ParseMultiple()[0].Value = %q, want %q", result[0].Value, "liked:add")
	}

	if result[1].Key != "data-optimistic-text" {
		t.Errorf("ParseMultiple()[1].Key = %q, want %q", result[1].Key, "data-optimistic-text")
	}
	if result[1].Value != "42" {
		t.Errorf("ParseMultiple()[1].Value = %q, want %q", result[1].Value, "42")
	}
}

// Test that Attr types are correct for use with vdom
func TestAttrIntegration(t *testing.T) {
	// All functions should return vdom.Attr with non-empty key
	attrs := []struct {
		name string
		attr interface{ IsEmpty() bool }
	}{
		{"Class", Class("test", true)},
		{"ClassToggle", ClassToggle("test")},
		{"Text", Text("test")},
		{"Attr", Attr("test", "value")},
		{"AttrRemove", AttrRemove("test")},
		{"ParentClass", ParentClass("test", true)},
		{"ParentClassToggle", ParentClassToggle("test")},
		{"Disable", Disable()},
		{"Hide", Hide()},
		{"Show", Show()},
		{"Loading", Loading(false)},
		{"Target", Target("#test")},
		{"Swap", Swap("<div></div>")},
	}

	for _, tt := range attrs {
		t.Run(tt.name, func(t *testing.T) {
			if tt.attr.IsEmpty() {
				t.Errorf("%s() returned empty attr", tt.name)
			}
		})
	}
}
