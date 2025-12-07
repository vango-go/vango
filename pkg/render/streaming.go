package render

import (
	"fmt"
	"io"
	"net/http"
)

// StreamingRenderer wraps Renderer with chunked output support.
// It flushes content incrementally for faster time-to-first-byte.
type StreamingRenderer struct {
	*Renderer
	flusher http.Flusher
	w       io.Writer
}

// NewStreamingRenderer creates a streaming renderer that writes to
// an http.ResponseWriter. If the writer implements http.Flusher,
// content will be flushed after each section for faster TTFB.
func NewStreamingRenderer(w http.ResponseWriter, config RendererConfig) *StreamingRenderer {
	flusher, _ := w.(http.Flusher)
	return &StreamingRenderer{
		Renderer: NewRenderer(config),
		flusher:  flusher,
		w:        w,
	}
}

// RenderPage renders a complete HTML document with incremental flushing.
// The head section is flushed immediately for faster first paint.
func (s *StreamingRenderer) RenderPage(page PageData) error {
	// Set default language
	lang := page.Lang
	if lang == "" {
		lang = "en"
	}

	// DOCTYPE
	if _, err := s.w.Write([]byte("<!DOCTYPE html>\n")); err != nil {
		return err
	}

	// HTML tag with lang
	if _, err := fmt.Fprintf(s.w, `<html lang="%s">`+"\n", escapeAttr(lang)); err != nil {
		return err
	}

	// Head section
	if err := s.renderHead(s.w, page); err != nil {
		return err
	}

	// Flush head immediately for faster first paint
	s.flush()

	// Body
	if _, err := s.w.Write([]byte("<body>\n")); err != nil {
		return err
	}

	// Main content
	if err := s.RenderToWriter(s.w, page.Body); err != nil {
		return err
	}

	// Flush body content
	s.flush()

	// Inject Vango client script
	if err := s.renderClientScript(s.w, page); err != nil {
		return err
	}

	// Close body and html
	if _, err := s.w.Write([]byte("</body>\n</html>\n")); err != nil {
		return err
	}

	// Final flush
	s.flush()

	return nil
}

// flush flushes the writer if it supports flushing.
func (s *StreamingRenderer) flush() {
	if s.flusher != nil {
		s.flusher.Flush()
	}
}

// FlushableWriter wraps an io.Writer with optional flushing capability.
// This is useful for testing streaming behavior without using http.ResponseWriter.
type FlushableWriter struct {
	io.Writer
	FlushCount int
}

// Flush implements http.Flusher.
func (w *FlushableWriter) Flush() {
	w.FlushCount++
}
