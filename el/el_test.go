package el

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/vango-go/vango"
	"github.com/vango-go/vango/pkg/vdom"
)

var (
	_ vdom.VNode         = VNode{}
	_ vdom.VKind         = VKind(0)
	_ vdom.Props         = Props{}
	_ vdom.Attr          = Attr{}
	_ vdom.EventHandler  = EventHandler{}
	_ vdom.Component     = Component(nil)
	_ vdom.Case[int]     = Case[int]{}
	_ vdom.ScriptsOption = ScriptsOption(nil)
	_ vdom.PathProvider  = PathProvider(nil)
)

type testPathProvider struct {
	path string
}

func (t testPathProvider) Path() string {
	return t.path
}

func TestElementConstructorsMatchVDOM(t *testing.T) {
	args := []any{
		vdom.ID("root"),
		vdom.Class("one", "two"),
		vdom.Hidden(false),
		vdom.OnClick("noop"),
		"hello",
		vdom.Span("child"),
	}

	got := Div(args...)
	want := vdom.Div(args...)

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Div() mismatch:\n got: %#v\nwant: %#v", got, want)
	}
}

func TestElementNamesMatchVDOM(t *testing.T) {
	cases := []struct {
		name string
		got  *VNode
		want *vdom.VNode
	}{
		{"time", Time_("now"), vdom.Time_("now")},
		{"data", DataElement("value"), vdom.DataElement("value")},
		{"link", LinkEl(vdom.Rel("stylesheet")), vdom.LinkEl(vdom.Rel("stylesheet"))},
	}

	for _, tc := range cases {
		if !reflect.DeepEqual(tc.got, tc.want) {
			t.Fatalf("%s element mismatch:\n got: %#v\nwant: %#v", tc.name, tc.got, tc.want)
		}
	}
}

func TestIsVoidElement(t *testing.T) {
	if !IsVoidElement("br") {
		t.Fatalf("IsVoidElement(\"br\") expected true")
	}
	if IsVoidElement("div") {
		t.Fatalf("IsVoidElement(\"div\") expected false")
	}
}

func TestTextHelpersMatchVDOM(t *testing.T) {
	if !reflect.DeepEqual(Text("hi"), vdom.Text("hi")) {
		t.Fatalf("Text() mismatch")
	}
	if !reflect.DeepEqual(Textf("hi %d", 2), vdom.Textf("hi %d", 2)) {
		t.Fatalf("Textf() mismatch")
	}
	if !reflect.DeepEqual(Raw("<b>hi</b>"), vdom.Raw("<b>hi</b>")) {
		t.Fatalf("Raw() mismatch")
	}
}

func TestFragmentHelpersMatchVDOM(t *testing.T) {
	args := []any{
		nil,
		"hello",
		vdom.Div("child"),
		[]*vdom.VNode{vdom.Span("nested")},
	}

	got := Fragment(args...)
	want := vdom.Fragment(args...)

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Fragment() mismatch:\n got: %#v\nwant: %#v", got, want)
	}
}

func TestConditionalHelpers(t *testing.T) {
	node := Text("ok")

	if If(true, node) != node {
		t.Fatalf("If(true) should return node")
	}
	if If(false, node) != nil {
		t.Fatalf("If(false) should return nil")
	}
	if IfElse(true, node, nil) != node {
		t.Fatalf("IfElse(true) should return ifTrue")
	}
	if IfElse(false, node, nil) != nil {
		t.Fatalf("IfElse(false) should return ifFalse")
	}
	if Unless(false, node) != node {
		t.Fatalf("Unless(false) should return node")
	}
	if Unless(true, node) != nil {
		t.Fatalf("Unless(true) should return nil")
	}
	if Show(true, node) != node {
		t.Fatalf("Show(true) should return node")
	}
	if Hide(true, node) != nil {
		t.Fatalf("Hide(true) should return nil")
	}
	if Either(node, nil) != node {
		t.Fatalf("Either should return first non-nil")
	}
	if Maybe(node) != node {
		t.Fatalf("Maybe should return node")
	}

	calls := 0
	result := When(false, func() *VNode {
		calls++
		return node
	})
	if result != nil || calls != 0 {
		t.Fatalf("When(false) should not call fn")
	}
	result = When(true, func() *VNode {
		calls++
		return node
	})
	if result != node || calls != 1 {
		t.Fatalf("When(true) should call fn once")
	}
	result = IfLazy(true, func() *VNode {
		calls++
		return node
	})
	if result != node || calls != 2 {
		t.Fatalf("IfLazy(true) should call fn once")
	}
	result = ShowWhen(true, func() *VNode {
		calls++
		return node
	})
	if result != node || calls != 3 {
		t.Fatalf("ShowWhen(true) should call fn once")
	}
}

func TestSwitchHelpers(t *testing.T) {
	one := Text("one")
	two := Text("two")
	def := Text("default")

	got := Switch("two",
		Case_("one", one),
		Case_("two", two),
		Default[string](def),
	)
	if got != two {
		t.Fatalf("Switch() should return matching case")
	}

	got = Switch("none",
		Case_("one", one),
		Default[string](def),
	)
	if got != def {
		t.Fatalf("Switch() should return default when no match")
	}
}

func TestRangeHelpers(t *testing.T) {
	items := []string{"a", "b", "c"}
	got := Range(items, func(item string, index int) *VNode {
		return Textf("%s:%d", item, index)
	})
	if len(got) != len(items) {
		t.Fatalf("Range() length mismatch: got %d want %d", len(got), len(items))
	}
	for i, node := range got {
		want := fmt.Sprintf("%s:%d", items[i], i)
		if node == nil || node.Kind != vdom.KindText || node.Text != want {
			t.Fatalf("Range() node mismatch at %d: got %#v want text %q", i, node, want)
		}
	}
}

func TestRangeMapHelper(t *testing.T) {
	items := map[string]int{"a": 1, "b": 2}
	got := RangeMap(items, func(key string, value int) *VNode {
		return Textf("%s:%d", key, value)
	})
	if len(got) != len(items) {
		t.Fatalf("RangeMap() length mismatch: got %d want %d", len(got), len(items))
	}

	seen := make(map[string]bool, len(items))
	for _, node := range got {
		if node == nil || node.Kind != vdom.KindText {
			t.Fatalf("RangeMap() returned non-text node: %#v", node)
		}
		seen[node.Text] = true
	}
	for key, value := range items {
		text := fmt.Sprintf("%s:%d", key, value)
		if !seen[text] {
			t.Fatalf("RangeMap() missing node %q", text)
		}
	}
}

func TestRepeatHelper(t *testing.T) {
	got := Repeat(3, func(i int) *VNode {
		return Textf("item-%d", i)
	})
	if len(got) != 3 {
		t.Fatalf("Repeat() length mismatch: got %d want 3", len(got))
	}
	for i, node := range got {
		want := fmt.Sprintf("item-%d", i)
		if node == nil || node.Kind != vdom.KindText || node.Text != want {
			t.Fatalf("Repeat() node mismatch at %d: got %#v want text %q", i, node, want)
		}
	}
}

func TestAttributeHelpersMatchVDOM(t *testing.T) {
	cases := []struct {
		name string
		got  Attr
		want vdom.Attr
	}{
		{"ID", ID("main"), vdom.ID("main")},
		{"Class", Class("a", "b"), vdom.Class("a", "b")},
		{"Data", Data("key", "value"), vdom.Data("key", "value")},
		{"AriaHidden", AriaHidden(true), vdom.AriaHidden(true)},
		{"HiddenFalse", Hidden(false), vdom.Hidden(false)},
		{"Download", Download("file.txt"), vdom.Download("file.txt")},
		{"Disabled", Disabled(), vdom.Disabled()},
	}

	for _, tc := range cases {
		if !reflect.DeepEqual(tc.got, tc.want) {
			t.Fatalf("%s attribute mismatch:\n got: %#v\nwant: %#v", tc.name, tc.got, tc.want)
		}
	}
}

func TestEventHelpersMatchVDOM(t *testing.T) {
	cases := []struct {
		name string
		got  EventHandler
		want vdom.EventHandler
	}{
		{"OnClick", OnClick("noop"), vdom.OnClick("noop")},
		{"OnInput", OnInput("noop"), vdom.OnInput("noop")},
		{"OnSubmit", OnSubmit("noop"), vdom.OnSubmit("noop")},
		{"OnScrollEnd", OnScrollEnd("noop"), vdom.OnScrollEnd("noop")},
		{"OnLoad", OnLoad("noop"), vdom.OnLoad("noop")},
	}

	for _, tc := range cases {
		if !reflect.DeepEqual(tc.got, tc.want) {
			t.Fatalf("%s event mismatch:\n got: %#v\nwant: %#v", tc.name, tc.got, tc.want)
		}
	}
}

func TestNavigationHelpersMatchVDOM(t *testing.T) {
	ctx := testPathProvider{path: "/about"}

	cases := []struct {
		name string
		got  *VNode
		want *vdom.VNode
	}{
		{"Link", Link("/about", Text("About")), vdom.Link("/about", vdom.Text("About"))},
		{"LinkPrefetch", LinkPrefetch("/about", Text("About")), vdom.LinkPrefetch("/about", vdom.Text("About"))},
		{"NavLinkActive", NavLink(ctx, "/about", Text("About")), vdom.NavLink(ctx, "/about", vdom.Text("About"))},
		{"NavLinkInactive", NavLink(ctx, "/blog", Text("Blog")), vdom.NavLink(ctx, "/blog", vdom.Text("Blog"))},
		{"NavLinkPrefix", NavLinkPrefix(ctx, "/about", Text("About")), vdom.NavLinkPrefix(ctx, "/about", vdom.Text("About"))},
	}

	for _, tc := range cases {
		if !reflect.DeepEqual(tc.got, tc.want) {
			t.Fatalf("%s mismatch:\n got: %#v\nwant: %#v", tc.name, tc.got, tc.want)
		}
	}
}

func TestVangoScriptsMatchVDOM(t *testing.T) {
	cases := []struct {
		name string
		got  *VNode
		want *vdom.VNode
	}{
		{"default", VangoScripts(), vdom.VangoScripts()},
		{
			"options",
			VangoScripts(
				WithDebug(),
				WithScriptPath("/custom.js"),
				WithCSRFToken("token"),
				WithoutDefer(),
			),
			vdom.VangoScripts(
				vdom.WithDebug(),
				vdom.WithScriptPath("/custom.js"),
				vdom.WithCSRFToken("token"),
				vdom.WithoutDefer(),
			),
		},
	}

	for _, tc := range cases {
		if !reflect.DeepEqual(tc.got, tc.want) {
			t.Fatalf("%s mismatch:\n got: %#v\nwant: %#v", tc.name, tc.got, tc.want)
		}
	}
}

func TestHookHelpers(t *testing.T) {
	config := map[string]any{"key": "value"}
	got := Hook("example", config)
	want := vango.Hook("example", config)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Hook() mismatch:\n got: %#v\nwant: %#v", got, want)
	}

	called := 0
	handlerAttr := OnEvent("ready", func(_ vango.HookEvent) {
		called++
	})
	handler, ok := handlerAttr.Value.(func(vango.HookEvent))
	if !ok {
		t.Fatalf("OnEvent() handler has unexpected type %T", handlerAttr.Value)
	}
	handler(vango.HookEvent{Name: "other"})
	if called != 0 {
		t.Fatalf("OnEvent() should ignore non-matching events")
	}
	handler(vango.HookEvent{Name: "ready"})
	if called != 1 {
		t.Fatalf("OnEvent() should call handler for matching event")
	}
}
