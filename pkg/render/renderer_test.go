package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/vango-dev/vango/v2/pkg/vdom"
)

func TestRenderText(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	node := vdom.Text("Hello, World!")
	html, err := renderer.RenderToString(node)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if html != "Hello, World!" {
		t.Errorf("got %q, want %q", html, "Hello, World!")
	}
}

func TestRenderTextEscaping(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	node := vdom.Text("<script>alert('xss')</script>")
	html, err := renderer.RenderToString(node)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(html, "<script>") {
		t.Errorf("HTML should be escaped, got %q", html)
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Errorf("should contain escaped script tag, got %q", html)
	}
}

func TestRenderElement(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	node := vdom.Div(vdom.Class("container"),
		vdom.H1(vdom.Text("Title")),
		vdom.P(vdom.Text("Content")),
	)
	html, err := renderer.RenderToString(node)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Note: All elements now get data-hid for full VDOM patching support
	if !strings.Contains(html, `class="container"`) {
		t.Errorf("should contain div with class, got %q", html)
	}
	if !strings.Contains(html, `<h1`) && !strings.Contains(html, `>Title</h1>`) {
		t.Errorf("should contain h1 with Title, got %q", html)
	}
	if !strings.Contains(html, `>Content</p>`) {
		t.Errorf("should contain p with Content, got %q", html)
	}
}

func TestRenderVoidElements(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	tests := []struct {
		name string
		node *vdom.VNode
		must []string // strings that must be present
	}{
		{
			name: "input",
			node: vdom.Input(vdom.Type("text"), vdom.Name("email")),
			must: []string{`<input`, `name="email"`, `type="text"`},
		},
		{
			name: "br",
			node: vdom.Br(),
			must: []string{`<br`},
		},
		{
			name: "img",
			node: vdom.Img(vdom.Src("/image.png"), vdom.Alt("test")),
			must: []string{`<img`, `src="/image.png"`, `alt="test"`},
		},
		{
			name: "hr",
			node: vdom.Hr(),
			must: []string{`<hr`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer.Reset()
			html, err := renderer.RenderToString(tt.node)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for _, s := range tt.must {
				if !strings.Contains(html, s) {
					t.Errorf("should contain %q, got %q", s, html)
				}
			}
			// Verify no closing tag
			if strings.Contains(html, "</"+tt.name+">") {
				t.Errorf("void element should not have closing tag, got %q", html)
			}
		})
	}
}

func TestRenderBooleanAttributes(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	node := vdom.Input(
		vdom.Type("checkbox"),
		vdom.Checked(),
		vdom.Disabled(),
	)
	html, err := renderer.RenderToString(node)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(html, " checked") {
		t.Errorf("should contain checked, got %q", html)
	}
	if !strings.Contains(html, " disabled") {
		t.Errorf("should contain disabled, got %q", html)
	}
	if strings.Contains(html, `checked="true"`) {
		t.Errorf("boolean attrs should not have values, got %q", html)
	}
	if strings.Contains(html, `disabled="true"`) {
		t.Errorf("boolean attrs should not have values, got %q", html)
	}
}

func TestRenderFragment(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	node := vdom.Fragment(
		vdom.Div(vdom.Text("One")),
		vdom.Div(vdom.Text("Two")),
	)
	html, err := renderer.RenderToString(node)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// All elements now get data-hid
	if !strings.Contains(html, ">One</div>") {
		t.Errorf("should contain One, got %q", html)
	}
	if !strings.Contains(html, ">Two</div>") {
		t.Errorf("should contain Two, got %q", html)
	}
}

func TestRenderHydrationID(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	handler := func() {}
	node := vdom.Button(vdom.OnClick(handler), vdom.Text("Click"))
	html, err := renderer.RenderToString(node)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(html, `data-hid="h1"`) {
		t.Errorf("should contain hydration ID, got %q", html)
	}
	if !strings.Contains(html, `data-ve="click"`) {
		t.Errorf("should contain event marker data-ve, got %q", html)
	}
}

func TestRenderMultipleHandlers(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	handler := func() {}
	node := vdom.Div(
		vdom.Button(vdom.OnClick(handler), vdom.Text("First")),
		vdom.Button(vdom.OnClick(handler), vdom.Text("Second")),
		vdom.Input(vdom.OnInput(handler)),
	)
	html, err := renderer.RenderToString(node)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// All elements get HIDs: div=h1, button=h2, button=h3, input=h4
	if !strings.Contains(html, `data-hid="h1"`) {
		t.Errorf("should contain h1 (div), got %q", html)
	}
	if !strings.Contains(html, `data-hid="h2"`) {
		t.Errorf("should contain h2 (button), got %q", html)
	}
	if !strings.Contains(html, `data-hid="h3"`) {
		t.Errorf("should contain h3 (button), got %q", html)
	}
	if !strings.Contains(html, `data-hid="h4"`) {
		t.Errorf("should contain h4 (input), got %q", html)
	}

	// Check handlers were registered (buttons with onclick)
	handlers := renderer.GetHandlers()
	if _, ok := handlers["h2_onclick"]; !ok {
		t.Error("h2_onclick handler should be registered")
	}
	if _, ok := handlers["h3_onclick"]; !ok {
		t.Error("h3_onclick handler should be registered")
	}
	if _, ok := handlers["h4_oninput"]; !ok {
		t.Error("h4_oninput handler should be registered")
	}
}

func TestRenderRaw(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	node := vdom.Raw("<strong>Bold</strong>")
	html, err := renderer.RenderToString(node)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if html != "<strong>Bold</strong>" {
		t.Errorf("raw HTML should not be escaped, got %q", html)
	}
}

func TestRenderPretty(t *testing.T) {
	renderer := NewRenderer(RendererConfig{Pretty: true, Indent: "  "})

	node := vdom.Div(
		vdom.H1(vdom.Text("Title")),
		vdom.P(vdom.Text("Content")),
	)
	html, err := renderer.RenderToString(node)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(html, "\n") {
		t.Errorf("pretty output should contain newlines, got %q", html)
	}
	// Note: indented elements also have data-hid now
	if !strings.Contains(html, "<h1") {
		t.Errorf("pretty output should have h1, got %q", html)
	}
}

func TestRenderNilNode(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	html, err := renderer.RenderToString(nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if html != "" {
		t.Errorf("nil node should produce empty string, got %q", html)
	}
}

func TestRenderNestedFragments(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	node := vdom.Fragment(
		vdom.Fragment(
			vdom.Span(vdom.Text("A")),
			vdom.Span(vdom.Text("B")),
		),
		vdom.Span(vdom.Text("C")),
	)
	html, err := renderer.RenderToString(node)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// All elements now get data-hid
	if !strings.Contains(html, ">A</span>") {
		t.Errorf("should contain A, got %q", html)
	}
	if !strings.Contains(html, ">B</span>") {
		t.Errorf("should contain B, got %q", html)
	}
	if !strings.Contains(html, ">C</span>") {
		t.Errorf("should contain C, got %q", html)
	}
}

func TestRenderComponent(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	comp := vdom.Func(func() *vdom.VNode {
		return vdom.Div(vdom.Text("From Component"))
	})
	node := &vdom.VNode{
		Kind: vdom.KindComponent,
		Comp: comp,
	}
	html, err := renderer.RenderToString(node)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Component div also gets data-hid
	if !strings.Contains(html, ">From Component</div>") {
		t.Errorf("should contain component content, got %q", html)
	}
}

func TestRendererReset(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	handler := func() {}

	// First render
	node1 := vdom.Button(vdom.OnClick(handler), vdom.Text("First"))
	html1, _ := renderer.RenderToString(node1)
	if !strings.Contains(html1, `data-hid="h1"`) {
		t.Errorf("first render should have h1, got %q", html1)
	}

	// Second render without reset (counter continues)
	node2 := vdom.Button(vdom.OnClick(handler), vdom.Text("Second"))
	html2, _ := renderer.RenderToString(node2)
	if !strings.Contains(html2, `data-hid="h2"`) {
		t.Errorf("second render should have h2, got %q", html2)
	}

	// Reset and render again
	renderer.Reset()
	node3 := vdom.Button(vdom.OnClick(handler), vdom.Text("Third"))
	html3, _ := renderer.RenderToString(node3)
	if !strings.Contains(html3, `data-hid="h1"`) {
		t.Errorf("after reset should have h1 again, got %q", html3)
	}

	// Verify handlers were cleared
	if len(renderer.GetHandlers()) != 1 {
		t.Errorf("should only have 1 handler after reset, got %d", len(renderer.GetHandlers()))
	}
}

func TestRenderToWriter(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	var buf bytes.Buffer
	node := vdom.Div(vdom.Text("Hello"))

	err := renderer.RenderToWriter(&buf, node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All elements get data-hid now
	if !strings.Contains(buf.String(), ">Hello</div>") {
		t.Errorf("should contain Hello in div, got %q", buf.String())
	}
	if !strings.Contains(buf.String(), "data-hid=") {
		t.Errorf("should contain data-hid, got %q", buf.String())
	}
}

func TestRenderAttributeEscaping(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	node := vdom.Input(vdom.Value(`test" onclick="alert('xss')`))
	html, err := renderer.RenderToString(node)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The double quote should be escaped, preventing attribute injection
	// The output should NOT have an actual onclick attribute that would execute
	// It should have &quot; instead of literal "
	if !strings.Contains(html, `&quot;`) {
		t.Errorf("quotes should be escaped, got %q", html)
	}
	// Verify the value attribute contains the escaped version
	if !strings.Contains(html, `value="test&quot;`) {
		t.Errorf("should have properly escaped value attribute, got %q", html)
	}
}

func TestRenderEmptyElement(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	node := vdom.Div()
	html, err := renderer.RenderToString(node)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty div also gets data-hid
	if !strings.Contains(html, "<div") || !strings.Contains(html, "></div>") {
		t.Errorf("should be an empty div, got %q", html)
	}
}

func TestRenderDataAttributes(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	node := vdom.Div(vdom.DataAttr("id", "123"), vdom.DataAttr("name", "test"))
	html, err := renderer.RenderToString(node)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(html, `data-id="123"`) {
		t.Errorf("should contain data-id, got %q", html)
	}
	if !strings.Contains(html, `data-name="test"`) {
		t.Errorf("should contain data-name, got %q", html)
	}
}
