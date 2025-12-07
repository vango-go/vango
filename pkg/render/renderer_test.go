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
	if !strings.Contains(html, `<div class="container">`) {
		t.Errorf("should contain div with class, got %q", html)
	}
	if !strings.Contains(html, `<h1>Title</h1>`) {
		t.Errorf("should contain h1, got %q", html)
	}
	if !strings.Contains(html, `<p>Content</p>`) {
		t.Errorf("should contain p, got %q", html)
	}
}

func TestRenderVoidElements(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	tests := []struct {
		name string
		node *vdom.VNode
		want string
	}{
		{
			name: "input",
			node: vdom.Input(vdom.Type("text"), vdom.Name("email")),
			want: `<input name="email" type="text">`,
		},
		{
			name: "br",
			node: vdom.Br(),
			want: `<br>`,
		},
		{
			name: "img",
			node: vdom.Img(vdom.Src("/image.png"), vdom.Alt("test")),
			want: `<img alt="test" src="/image.png">`,
		},
		{
			name: "hr",
			node: vdom.Hr(),
			want: `<hr>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html, err := renderer.RenderToString(tt.node)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if html != tt.want {
				t.Errorf("got %q, want %q", html, tt.want)
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
	expected := "<div>One</div><div>Two</div>"
	if html != expected {
		t.Errorf("got %q, want %q", html, expected)
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
	if !strings.Contains(html, `data-on-click="true"`) {
		t.Errorf("should contain event marker, got %q", html)
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
	if !strings.Contains(html, `data-hid="h1"`) {
		t.Errorf("should contain h1, got %q", html)
	}
	if !strings.Contains(html, `data-hid="h2"`) {
		t.Errorf("should contain h2, got %q", html)
	}
	if !strings.Contains(html, `data-hid="h3"`) {
		t.Errorf("should contain h3, got %q", html)
	}

	// Check handlers were registered
	handlers := renderer.GetHandlers()
	if _, ok := handlers["h1_onclick"]; !ok {
		t.Error("h1_onclick handler should be registered")
	}
	if _, ok := handlers["h2_onclick"]; !ok {
		t.Error("h2_onclick handler should be registered")
	}
	if _, ok := handlers["h3_oninput"]; !ok {
		t.Error("h3_oninput handler should be registered")
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
	if !strings.Contains(html, "  <h1>") {
		t.Errorf("pretty output should have indentation, got %q", html)
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
	expected := "<span>A</span><span>B</span><span>C</span>"
	if html != expected {
		t.Errorf("got %q, want %q", html, expected)
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
	if html != "<div>From Component</div>" {
		t.Errorf("got %q, want %q", html, "<div>From Component</div>")
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

	if buf.String() != "<div>Hello</div>" {
		t.Errorf("got %q, want %q", buf.String(), "<div>Hello</div>")
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
	if html != "<div></div>" {
		t.Errorf("got %q, want %q", html, "<div></div>")
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
