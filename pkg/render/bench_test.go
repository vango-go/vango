package render

import (
	"fmt"
	"io"
	"testing"

	"github.com/vango-go/vango/pkg/vdom"
)

func BenchmarkRenderSimple(b *testing.B) {
	renderer := NewRenderer(RendererConfig{})
	node := vdom.Div(vdom.Class("card"),
		vdom.H1(vdom.Text("Title")),
		vdom.P(vdom.Text("Content")),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.Reset()
		renderer.RenderToString(node)
	}
}

func BenchmarkRenderLargeTree(b *testing.B) {
	renderer := NewRenderer(RendererConfig{})

	// Build a tree with 1000 elements
	var items []any
	for i := 0; i < 1000; i++ {
		items = append(items, vdom.Li(vdom.Text(fmt.Sprintf("Item %d", i))))
	}
	node := vdom.Ul(items...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.Reset()
		renderer.RenderToString(node)
	}
}

func BenchmarkRenderWithHandlers(b *testing.B) {
	renderer := NewRenderer(RendererConfig{})
	handler := func() {}

	var buttons []any
	for i := 0; i < 100; i++ {
		buttons = append(buttons, vdom.Button(vdom.OnClick(handler), vdom.Text(fmt.Sprintf("Button %d", i))))
	}
	node := vdom.Div(buttons...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.Reset()
		renderer.RenderToString(node)
	}
}

func BenchmarkRenderToWriter(b *testing.B) {
	renderer := NewRenderer(RendererConfig{})
	node := vdom.Div(vdom.Class("card"),
		vdom.H1(vdom.Text("Title")),
		vdom.P(vdom.Text("Content")),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.Reset()
		renderer.RenderToWriter(io.Discard, node)
	}
}

func BenchmarkRenderPage(b *testing.B) {
	renderer := NewRenderer(RendererConfig{})
	page := PageData{
		Body:        vdom.Div(vdom.H1(vdom.Text("Hello")), vdom.P(vdom.Text("World"))),
		Title:       "Test Page",
		SessionID:   "sess_123",
		CSRFToken:   "csrf_abc",
		StyleSheets: []string{"/css/main.css"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.Reset()
		renderer.RenderPage(io.Discard, page)
	}
}

func BenchmarkRenderDeepNesting(b *testing.B) {
	renderer := NewRenderer(RendererConfig{})

	// Build a deeply nested tree (20 levels)
	var node *vdom.VNode = vdom.Span(vdom.Text("Leaf"))
	for i := 0; i < 20; i++ {
		node = vdom.Div(node)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.Reset()
		renderer.RenderToString(node)
	}
}

func BenchmarkRenderManyAttributes(b *testing.B) {
	renderer := NewRenderer(RendererConfig{})

	node := vdom.Div(
		vdom.ID("main"),
		vdom.Class("container", "primary", "active"),
		vdom.DataAttr("id", "123"),
		vdom.DataAttr("type", "content"),
		vdom.DataAttr("status", "published"),
		vdom.AriaLabel("Main content"),
		vdom.Role("main"),
		vdom.TabIndex(0),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.Reset()
		renderer.RenderToString(node)
	}
}

func BenchmarkRenderPretty(b *testing.B) {
	renderer := NewRenderer(RendererConfig{Pretty: true, Indent: "  "})

	node := vdom.Div(vdom.Class("card"),
		vdom.H1(vdom.Text("Title")),
		vdom.P(vdom.Text("Content")),
		vdom.Ul(
			vdom.Li(vdom.Text("Item 1")),
			vdom.Li(vdom.Text("Item 2")),
			vdom.Li(vdom.Text("Item 3")),
		),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.Reset()
		renderer.RenderToString(node)
	}
}

func BenchmarkRenderComplexPage(b *testing.B) {
	renderer := NewRenderer(RendererConfig{})
	handler := func() {}

	// Build a realistic page structure
	var rows []any
	for i := 0; i < 50; i++ {
		rows = append(rows, vdom.Tr(
			vdom.Td(vdom.Text(fmt.Sprintf("%d", i+1))),
			vdom.Td(vdom.Text(fmt.Sprintf("User %d", i))),
			vdom.Td(vdom.Text(fmt.Sprintf("user%d@example.com", i))),
			vdom.Td(vdom.Button(vdom.OnClick(handler), vdom.Text("Edit"))),
		))
	}

	node := vdom.Div(vdom.Class("container"),
		vdom.Header(
			vdom.Nav(vdom.Class("navbar"),
				vdom.A(vdom.Href("/"), vdom.Text("Home")),
				vdom.A(vdom.Href("/about"), vdom.Text("About")),
				vdom.A(vdom.Href("/contact"), vdom.Text("Contact")),
			),
		),
		vdom.Main(
			vdom.H1(vdom.Text("Users")),
			vdom.Table(vdom.Class("table"),
				vdom.Thead(
					vdom.Tr(
						vdom.Th(vdom.Text("ID")),
						vdom.Th(vdom.Text("Name")),
						vdom.Th(vdom.Text("Email")),
						vdom.Th(vdom.Text("Actions")),
					),
				),
				vdom.Tbody(rows...),
			),
		),
		vdom.Footer(
			vdom.P(vdom.Text("© 2024 Vango")),
		),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.Reset()
		renderer.RenderToString(node)
	}
}

func BenchmarkRenderFragment(b *testing.B) {
	renderer := NewRenderer(RendererConfig{})

	var items []*vdom.VNode
	for i := 0; i < 100; i++ {
		items = append(items, vdom.Div(vdom.Text(fmt.Sprintf("Item %d", i))))
	}

	node := vdom.Fragment(func() []any {
		result := make([]any, len(items))
		for i, item := range items {
			result[i] = item
		}
		return result
	}()...)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderer.Reset()
		renderer.RenderToString(node)
	}
}
