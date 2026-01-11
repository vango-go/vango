package render

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/vango-go/vango/pkg/vango"
	"github.com/vango-go/vango/pkg/vdom"
)

func TestRenderElementDangerouslySetInnerHTMLProp(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	node := &vdom.VNode{
		Kind: vdom.KindElement,
		Tag:  "div",
		Props: vdom.Props{
			"dangerouslySetInnerHTML": `<span class="x">Raw</span>`,
		},
		Children: []*vdom.VNode{
			vdom.Text("SHOULD_NOT_RENDER"),
		},
	}

	html, err := renderer.RenderToString(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(html, "dangerouslySetInnerHTML") {
		t.Fatalf("should not render dangerouslySetInnerHTML attribute, got %q", html)
	}
	if strings.Contains(html, "SHOULD_NOT_RENDER") {
		t.Fatalf("should not render children when dangerouslySetInnerHTML is used, got %q", html)
	}
	if !strings.Contains(html, `<span class="x">Raw</span>`) {
		t.Fatalf("should include raw inner HTML, got %q", html)
	}
}

func TestRenderElementPreservesExistingHID(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	handler := func() {}
	node := &vdom.VNode{
		Kind: vdom.KindElement,
		Tag:  "button",
		HID:  "h10",
		Props: vdom.Props{
			"onclick": handler,
		},
		Children: []*vdom.VNode{
			vdom.Text("Click"),
		},
	}

	html, err := renderer.RenderToString(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(html, `data-hid="h10"`) {
		t.Fatalf("should preserve existing HID, got %q", html)
	}
	if !strings.Contains(html, `data-ve="click"`) {
		t.Fatalf("should emit data-ve when handlers are present, got %q", html)
	}
	if _, ok := renderer.GetHandlers()["h10_onclick"]; !ok {
		t.Fatalf("should register handler under preserved HID")
	}
}

func TestRenderAttributesSpecialMappingsAndSkipping(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	node := &vdom.VNode{
		Kind: vdom.KindElement,
		Tag:  "label",
		Props: vdom.Props{
			"className": "a b",
			"htmlFor":   "target",
			"_internal": "should-not-render",
			"disabled":  false,
			"key":       "should-not-render",
		},
		Children: []*vdom.VNode{vdom.Text("Label")},
	}

	html, err := renderer.RenderToString(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(html, `class="a b"`) {
		t.Fatalf("should map className -> class, got %q", html)
	}
	if !strings.Contains(html, `for="target"`) {
		t.Fatalf("should map htmlFor -> for, got %q", html)
	}
	if strings.Contains(html, "_internal") {
		t.Fatalf("should skip internal props, got %q", html)
	}
	if strings.Contains(html, `disabled="`) || strings.Contains(html, " disabled") {
		t.Fatalf("should not render false boolean attributes, got %q", html)
	}
	if strings.Contains(html, ` key=`) {
		t.Fatalf("should not render key attribute, got %q", html)
	}
}

func TestRenderAttributesRejectsInjectedOnAttributes(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	node := &vdom.VNode{
		Kind: vdom.KindElement,
		Tag:  "div",
		Props: vdom.Props{
			"onclick": `alert("xss")`,
			"OnLoad":  `alert("xss")`,
			"id":      "safe",
		},
	}

	html, err := renderer.RenderToString(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(strings.ToLower(html), "onclick=") || strings.Contains(strings.ToLower(html), "onload=") {
		t.Fatalf("should not render injected on* attributes, got %q", html)
	}
	if strings.Contains(html, "data-ve=") {
		t.Fatalf("should not emit data-ve for non-handlers, got %q", html)
	}
	if len(renderer.GetHandlers()) != 0 {
		t.Fatalf("should not register handlers for non-handler values")
	}
}

func TestRenderAttributesDataVeSortedAndModifiersRendered(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	node := &vdom.VNode{
		Kind: vdom.KindElement,
		Tag:  "button",
		Props: vdom.Props{
			"oninput": func(any) {},
			"onclick": vango.ModifiedHandler{
				Handler:         func() {},
				PreventDefault:  true,
				StopPropagation: true,
				Self:            true,
				Once:            true,
				Passive:         true,
				Capture:         true,
				Debounce:        150 * time.Millisecond,
				Throttle:        200 * time.Millisecond,
			},
		},
		Children: []*vdom.VNode{vdom.Text("Click")},
	}

	html, err := renderer.RenderToString(node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(html, `data-ve="click,input"`) {
		t.Fatalf("should include sorted data-ve values, got %q", html)
	}

	// Modifier attributes (spec §3.9.3)
	for _, want := range []string{
		`data-pd-click="true"`,
		`data-sp-click="true"`,
		`data-self-click="true"`,
		`data-once-click="true"`,
		`data-passive-click="true"`,
		`data-capture-click="true"`,
		`data-debounce-click="150"`,
		`data-throttle-click="200"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in %q", want, html)
		}
	}
}

func TestRenderNodeUnknownKindErrors(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})
	_, err := renderer.RenderToString(&vdom.VNode{Kind: 99})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestNeedsHIDNonElementReturnsFalse(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	if renderer.needsHID(vdom.Text("x")) {
		t.Fatalf("text nodes should not need HIDs")
	}
	if renderer.needsHID(vdom.Fragment(vdom.Text("x"))) {
		t.Fatalf("fragment nodes should not need HIDs")
	}
}

func TestRenderFragmentPropagatesChildError(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	node := &vdom.VNode{
		Kind: vdom.KindFragment,
		Children: []*vdom.VNode{
			{Kind: 99},
		},
	}

	_, err := renderer.RenderToString(node)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestRenderComponentNilIsNoop(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})
	html, err := renderer.RenderToString(&vdom.VNode{Kind: vdom.KindComponent})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if html != "" {
		t.Fatalf("expected empty output, got %q", html)
	}
}

func TestGetModifierAttrsAllFields(t *testing.T) {
	attrs := getModifierAttrs("click", vango.ModifiedHandler{
		PreventDefault:  true,
		StopPropagation: true,
		Self:            true,
		Once:            true,
		Passive:         true,
		Capture:         true,
		Debounce:        1 * time.Second,
		Throttle:        250 * time.Millisecond,
	})

	joined := strings.Join(attrs, "")
	for _, want := range []string{
		` data-pd-click="true"`,
		` data-sp-click="true"`,
		` data-self-click="true"`,
		` data-once-click="true"`,
		` data-passive-click="true"`,
		` data-capture-click="true"`,
		` data-debounce-click="1000"`,
		` data-throttle-click="250"`,
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in %q", want, joined)
		}
	}
}

func TestIsEventHandlerAndAttrToStringVariants(t *testing.T) {
	if isEventHandler(nil) {
		t.Fatalf("nil should not be an event handler")
	}
	if !isEventHandler(func() {}) {
		t.Fatalf("func() should be an event handler")
	}
	if !isEventHandler(func(any) {}) {
		t.Fatalf("func(any) should be an event handler")
	}
	if !isEventHandler(vango.ModifiedHandler{Handler: func() {}}) {
		t.Fatalf("ModifiedHandler should be an event handler")
	}
	if !isEventHandler(vdom.EventHandler{Event: "onclick", Handler: func() {}}) {
		t.Fatalf("vdom.EventHandler value should be an event handler")
	}
	if !isEventHandler(func(int) {}) {
		t.Fatalf("unknown func signatures should still be detected")
	}
	if isEventHandler(123) {
		t.Fatalf("non-functions should not be event handlers")
	}

	type sample struct{ X int }
	for _, tt := range []struct {
		in   any
		want string
	}{
		{in: "s", want: "s"},
		{in: true, want: "true"},
		{in: false, want: "false"},
		{in: 7, want: "7"},
		{in: int64(9), want: "9"},
		{in: float64(1.25), want: "1.25"},
		{in: sample{X: 3}, want: "{3}"},
		{in: nil, want: ""},
	} {
		if got := attrToString(tt.in); got != tt.want {
			t.Fatalf("attrToString(%T) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestRenderVHookSuccessAndErrors(t *testing.T) {
	var buf bytes.Buffer
	if err := renderVHook(&buf, `Sortable:{"group":"items"}`); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, `data-hook="Sortable"`) {
		t.Fatalf("should contain hook name, got %q", got)
	}
	if !strings.Contains(got, `data-hook-config='{"group":"items"}'`) {
		t.Fatalf("should contain hook config, got %q", got)
	}

	buf.Reset()
	if err := renderVHook(&buf, `Sortable:{}`); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got = buf.String()
	if !strings.Contains(got, `data-hook="Sortable"`) {
		t.Fatalf("should contain hook name, got %q", got)
	}
	if strings.Contains(got, "data-hook-config") {
		t.Fatalf("should omit empty config, got %q", got)
	}

	if err := renderVHook(&buf, "invalid"); err == nil {
		t.Fatalf("expected error for invalid v-hook format")
	}
}

func TestRenderHookConfigMarshalError(t *testing.T) {
	var buf bytes.Buffer
	err := renderHookConfig(&buf, HookConfig{
		Name:   "Bad",
		Config: func() {},
	})
	if err == nil {
		t.Fatalf("expected marshal error")
	}
}

func TestRenderAttributesVHookInvalidPropErrors(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})
	node := &vdom.VNode{
		Kind: vdom.KindElement,
		Tag:  "div",
		Props: vdom.Props{
			"v-hook": "InvalidFormat",
		},
	}
	_, err := renderer.RenderToString(node)
	if err == nil {
		t.Fatalf("expected error")
	}
}
