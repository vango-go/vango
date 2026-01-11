package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/vango-go/vango/pkg/vdom"
)

func TestRenderMetaTagAllFields(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	var buf bytes.Buffer
	err := renderer.renderMetaTag(&buf, MetaTag{
		Charset:   "utf-8",
		Name:      "description",
		Property:  "og:title",
		HTTPEquiv: "Content-Security-Policy",
		Content:   `text "with" <chars> & stuff`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := buf.String()
	for _, want := range []string{
		`<meta`,
		`charset="utf-8"`,
		`name="description"`,
		`property="og:title"`,
		`http-equiv="Content-Security-Policy"`,
		`content="text &quot;with&quot; &lt;chars&gt; &amp; stuff"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in %q", want, html)
		}
	}
}

func TestRenderLinkTagAllFields(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	var buf bytes.Buffer
	err := renderer.renderLinkTag(&buf, LinkTag{
		Rel:         "icon",
		Href:        "/favicon.ico",
		Type:        "image/x-icon",
		Sizes:       "32x32",
		CrossOrigin: "anonymous",
		Media:       "(prefers-color-scheme: dark)",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := buf.String()
	for _, want := range []string{
		`<link`,
		`rel="icon"`,
		`href="/favicon.ico"`,
		`type="image/x-icon"`,
		`sizes="32x32"`,
		`crossorigin="anonymous"`,
		`media="(prefers-color-scheme: dark)"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in %q", want, html)
		}
	}
}

func TestRenderScriptTagTypeAndInline(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	var buf bytes.Buffer
	err := renderer.renderScriptTag(&buf, ScriptTag{
		Type:   "text/javascript",
		Inline: "window.__TEST__=true;",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	html := buf.String()
	if !strings.Contains(html, `type="text/javascript"`) {
		t.Fatalf("should include type attribute, got %q", html)
	}
	if !strings.Contains(html, "window.__TEST__=true;") {
		t.Fatalf("should include inline script content, got %q", html)
	}

	buf.Reset()
	err = renderer.renderScriptTag(&buf, ScriptTag{
		Src:   "/js/app.js",
		Defer: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	html = buf.String()
	if !strings.Contains(html, `src="/js/app.js"`) || !strings.Contains(html, " defer") {
		t.Fatalf("should include src and defer, got %q", html)
	}
}

func TestRenderPageSkipsNonDeferOrAsyncScriptsInHead(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	page := PageData{
		Body:  vdom.Div(vdom.Text("Content")),
		Title: "Scripts",
		Scripts: []ScriptTag{
			{Src: "/js/should-not-be-in-head.js"},
		},
	}

	var buf bytes.Buffer
	err := renderer.RenderPage(&buf, page)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := buf.String()
	if strings.Contains(html, "/js/should-not-be-in-head.js") {
		t.Fatalf("script without defer/async should not be rendered, got %q", html)
	}
}

