package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/vango-dev/vango/v2/pkg/vdom"
)

func TestRenderPage(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	page := PageData{
		Body:  vdom.Div(vdom.Text("Hello, World!")),
		Title: "Test Page",
	}

	var buf bytes.Buffer
	err := renderer.RenderPage(&buf, page)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := buf.String()

	// Check DOCTYPE
	if !strings.HasPrefix(html, "<!DOCTYPE html>") {
		t.Errorf("should start with DOCTYPE, got %q", html[:50])
	}

	// Check html tag
	if !strings.Contains(html, `<html lang="en">`) {
		t.Errorf("should contain html tag with lang, got %q", html)
	}

	// Check head
	if !strings.Contains(html, "<head>") {
		t.Errorf("should contain head tag, got %q", html)
	}
	if !strings.Contains(html, `<meta charset="utf-8">`) {
		t.Errorf("should contain charset meta, got %q", html)
	}
	if !strings.Contains(html, `<meta name="viewport"`) {
		t.Errorf("should contain viewport meta, got %q", html)
	}
	if !strings.Contains(html, "<title>Test Page</title>") {
		t.Errorf("should contain title, got %q", html)
	}

	// Check body
	if !strings.Contains(html, "<body>") {
		t.Errorf("should contain body tag, got %q", html)
	}
	if !strings.Contains(html, "<div>Hello, World!</div>") {
		t.Errorf("should contain body content, got %q", html)
	}

	// Check client script
	if !strings.Contains(html, `<script src="/_vango/client.js" defer></script>`) {
		t.Errorf("should contain client script, got %q", html)
	}
}

func TestRenderPageWithSessionAndCSRF(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	page := PageData{
		Body:      vdom.Div(vdom.Text("Content")),
		Title:     "Secure Page",
		SessionID: "sess_123abc",
		CSRFToken: "csrf_token_xyz",
	}

	var buf bytes.Buffer
	err := renderer.RenderPage(&buf, page)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := buf.String()

	// Check CSRF token
	if !strings.Contains(html, `window.__VANGO_CSRF__="csrf_token_xyz"`) {
		t.Errorf("should contain CSRF token, got %q", html)
	}

	// Check session ID
	if !strings.Contains(html, `window.__VANGO_SESSION__="sess_123abc"`) {
		t.Errorf("should contain session ID, got %q", html)
	}
}

func TestRenderPageWithMeta(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	page := PageData{
		Body:  vdom.Div(),
		Title: "Meta Test",
		Meta: []MetaTag{
			{Name: "description", Content: "Test description"},
			{Property: "og:title", Content: "OG Title"},
			{HTTPEquiv: "X-UA-Compatible", Content: "IE=edge"},
		},
	}

	var buf bytes.Buffer
	err := renderer.RenderPage(&buf, page)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := buf.String()

	if !strings.Contains(html, `<meta name="description" content="Test description">`) {
		t.Errorf("should contain description meta, got %q", html)
	}
	if !strings.Contains(html, `<meta property="og:title" content="OG Title">`) {
		t.Errorf("should contain og:title meta, got %q", html)
	}
	if !strings.Contains(html, `<meta http-equiv="X-UA-Compatible" content="IE=edge">`) {
		t.Errorf("should contain http-equiv meta, got %q", html)
	}
}

func TestRenderPageWithLinks(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	page := PageData{
		Body:  vdom.Div(),
		Title: "Links Test",
		Links: []LinkTag{
			{Rel: "icon", Href: "/favicon.ico"},
			{Rel: "preconnect", Href: "https://fonts.googleapis.com", CrossOrigin: "anonymous"},
		},
	}

	var buf bytes.Buffer
	err := renderer.RenderPage(&buf, page)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := buf.String()

	if !strings.Contains(html, `<link rel="icon" href="/favicon.ico">`) {
		t.Errorf("should contain favicon link, got %q", html)
	}
	if !strings.Contains(html, `<link rel="preconnect" href="https://fonts.googleapis.com" crossorigin="anonymous">`) {
		t.Errorf("should contain preconnect link, got %q", html)
	}
}

func TestRenderPageWithStyleSheets(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	page := PageData{
		Body:  vdom.Div(),
		Title: "Styles Test",
		StyleSheets: []string{
			"/css/main.css",
			"/css/theme.css",
		},
	}

	var buf bytes.Buffer
	err := renderer.RenderPage(&buf, page)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := buf.String()

	if !strings.Contains(html, `<link rel="stylesheet" href="/css/main.css">`) {
		t.Errorf("should contain main.css stylesheet, got %q", html)
	}
	if !strings.Contains(html, `<link rel="stylesheet" href="/css/theme.css">`) {
		t.Errorf("should contain theme.css stylesheet, got %q", html)
	}
}

func TestRenderPageWithInlineStyles(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	page := PageData{
		Body:   vdom.Div(),
		Title:  "Inline Styles Test",
		Styles: []string{"body { margin: 0; }", ".header { color: red; }"},
	}

	var buf bytes.Buffer
	err := renderer.RenderPage(&buf, page)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := buf.String()

	if !strings.Contains(html, "<style>body { margin: 0; }</style>") {
		t.Errorf("should contain first inline style, got %q", html)
	}
	if !strings.Contains(html, "<style>.header { color: red; }</style>") {
		t.Errorf("should contain second inline style, got %q", html)
	}
}

func TestRenderPageWithScripts(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	page := PageData{
		Body:  vdom.Div(),
		Title: "Scripts Test",
		Scripts: []ScriptTag{
			{Src: "/js/analytics.js", Async: true},
			{Src: "/js/app.js", Defer: true, Module: true},
		},
	}

	var buf bytes.Buffer
	err := renderer.RenderPage(&buf, page)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := buf.String()

	if !strings.Contains(html, `<script src="/js/analytics.js" async></script>`) {
		t.Errorf("should contain async script, got %q", html)
	}
	if !strings.Contains(html, `<script src="/js/app.js" type="module" defer></script>`) {
		t.Errorf("should contain deferred module script, got %q", html)
	}
}

func TestRenderPageWithCustomClientScript(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	page := PageData{
		Body:         vdom.Div(),
		Title:        "Custom Client Test",
		ClientScript: "/assets/vango-client.min.js",
	}

	var buf bytes.Buffer
	err := renderer.RenderPage(&buf, page)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := buf.String()

	if !strings.Contains(html, `<script src="/assets/vango-client.min.js" defer></script>`) {
		t.Errorf("should contain custom client script path, got %q", html)
	}
}

func TestRenderPageWithCustomLang(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	page := PageData{
		Body:  vdom.Div(),
		Title: "French Page",
		Lang:  "fr",
	}

	var buf bytes.Buffer
	err := renderer.RenderPage(&buf, page)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := buf.String()

	if !strings.Contains(html, `<html lang="fr">`) {
		t.Errorf("should contain custom lang, got %q", html)
	}
}

func TestRenderPageEscaping(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})

	page := PageData{
		Body:  vdom.Div(),
		Title: `<script>alert("xss")</script>`,
		Meta: []MetaTag{
			{Name: "description", Content: `Test "with" <special> & chars`},
		},
		CSRFToken: `token"with'quotes`,
	}

	var buf bytes.Buffer
	err := renderer.RenderPage(&buf, page)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := buf.String()

	// Title should be escaped
	if strings.Contains(html, "<script>alert") {
		t.Errorf("title should be escaped, got %q", html)
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Errorf("title should contain escaped script, got %q", html)
	}

	// Meta content should be escaped
	if !strings.Contains(html, "&quot;") || !strings.Contains(html, "&amp;") {
		t.Errorf("meta content should be escaped, got %q", html)
	}

	// CSRF token should be escaped
	if strings.Contains(html, `token"with'quotes`) && !strings.Contains(html, `&quot;`) {
		t.Errorf("CSRF token should be escaped, got %q", html)
	}
}

func TestRenderHookConfig(t *testing.T) {
	var buf bytes.Buffer

	hookConfig := HookConfig{
		Name: "Sortable",
		Config: map[string]any{
			"group":  "items",
			"handle": ".drag-handle",
		},
	}

	err := renderHookConfig(&buf, hookConfig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := buf.String()

	if !strings.Contains(html, `data-hook="Sortable"`) {
		t.Errorf("should contain hook name, got %q", html)
	}
	if !strings.Contains(html, `data-hook-config=`) {
		t.Errorf("should contain hook config, got %q", html)
	}
	if !strings.Contains(html, `"group":"items"`) {
		t.Errorf("should contain group in config, got %q", html)
	}
}

func TestRenderOptimisticConfig(t *testing.T) {
	var buf bytes.Buffer

	config := OptimisticConfig{
		Class: "pending",
		Text:  "Saving...",
	}

	err := renderOptimisticConfig(&buf, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := buf.String()

	if !strings.Contains(html, `data-optimistic-class="pending"`) {
		t.Errorf("should contain optimistic class, got %q", html)
	}
	if !strings.Contains(html, `data-optimistic-text="Saving..."`) {
		t.Errorf("should contain optimistic text, got %q", html)
	}
}
