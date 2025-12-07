package render

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vango-dev/vango/v2/pkg/vdom"
)

func TestStreamingRendererRenderPage(t *testing.T) {
	// Create a test ResponseWriter
	w := httptest.NewRecorder()

	sr := NewStreamingRenderer(w, RendererConfig{})

	page := PageData{
		Body:      vdom.Div(vdom.Text("Streamed Content")),
		Title:     "Streaming Test",
		SessionID: "test_session",
	}

	err := sr.RenderPage(page)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := w.Body.String()

	// Verify content
	if !strings.HasPrefix(html, "<!DOCTYPE html>") {
		t.Errorf("should start with DOCTYPE")
	}
	if !strings.Contains(html, "<title>Streaming Test</title>") {
		t.Errorf("should contain title")
	}
	if !strings.Contains(html, "<div>Streamed Content</div>") {
		t.Errorf("should contain body content")
	}
}

func TestStreamingRendererFlushes(t *testing.T) {
	// Create a custom writer that tracks flushes
	var buf bytes.Buffer
	fw := &FlushableWriter{Writer: &buf}

	// Create a mock http.ResponseWriter that uses our FlushableWriter
	sr := &StreamingRenderer{
		Renderer: NewRenderer(RendererConfig{}),
		flusher:  fw,
		w:        fw,
	}

	page := PageData{
		Body:  vdom.Div(vdom.Text("Content")),
		Title: "Flush Test",
	}

	err := sr.RenderPage(page)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have flushed at least 3 times:
	// 1. After head
	// 2. After body content
	// 3. At the end
	if fw.FlushCount < 3 {
		t.Errorf("expected at least 3 flushes, got %d", fw.FlushCount)
	}
}

func TestStreamingRendererWithHttpTest(t *testing.T) {
	w := httptest.NewRecorder()

	sr := NewStreamingRenderer(w, RendererConfig{})

	page := PageData{
		Body:  vdom.Div(vdom.H1(vdom.Text("Hello")), vdom.P(vdom.Text("World"))),
		Title: "HTTP Test",
	}

	err := sr.RenderPage(page)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := w.Result()
	if result.StatusCode != 200 {
		t.Errorf("unexpected status code: %d", result.StatusCode)
	}

	html := w.Body.String()
	if !strings.Contains(html, "</html>") {
		t.Errorf("should contain closing html tag")
	}
}

func TestStreamingRendererNilFlusher(t *testing.T) {
	// Test with a writer that doesn't implement Flusher
	var buf bytes.Buffer

	sr := &StreamingRenderer{
		Renderer: NewRenderer(RendererConfig{}),
		flusher:  nil,
		w:        &buf,
	}

	page := PageData{
		Body:  vdom.Div(vdom.Text("No Flush")),
		Title: "No Flush Test",
	}

	// Should not panic even without Flusher
	err := sr.RenderPage(page)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, "<div>No Flush</div>") {
		t.Errorf("should render content, got %q", html)
	}
}

func TestStreamingRendererLargeContent(t *testing.T) {
	w := httptest.NewRecorder()

	sr := NewStreamingRenderer(w, RendererConfig{})

	// Create a large list
	var items []any
	for i := 0; i < 100; i++ {
		items = append(items, vdom.Li(vdom.Textf("Item %d", i)))
	}

	page := PageData{
		Body:  vdom.Ul(items...),
		Title: "Large Content",
	}

	err := sr.RenderPage(page)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	html := w.Body.String()

	// Verify first and last items
	if !strings.Contains(html, "<li>Item 0</li>") {
		t.Errorf("should contain first item")
	}
	if !strings.Contains(html, "<li>Item 99</li>") {
		t.Errorf("should contain last item")
	}
}
