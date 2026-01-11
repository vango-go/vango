package render

import (
	"errors"
	"io"
	"testing"

	"github.com/vango-go/vango/pkg/vdom"
)

var errTestWrite = errors.New("test write error")

type countingWriter struct {
	Writes int
}

func (w *countingWriter) Write(p []byte) (int, error) {
	w.Writes++
	return len(p), nil
}

type failingWriter struct {
	FailAt int
	Writes int
}

func (w *failingWriter) Write(p []byte) (int, error) {
	w.Writes++
	if w.Writes == w.FailAt {
		return 0, errTestWrite
	}
	return len(p), nil
}

type countingFlushWriter struct {
	io.Writer
	Flushes int
}

func (w *countingFlushWriter) Flush() { w.Flushes++ }

func TestRenderMetaTagWriteErrorPaths(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})
	meta := MetaTag{
		Charset:   "utf-8",
		Name:      "description",
		Property:  "og:title",
		HTTPEquiv: "X-UA-Compatible",
		Content:   "content",
	}

	cw := &countingWriter{}
	if err := renderer.renderMetaTag(cw, meta); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := 1; i <= cw.Writes; i++ {
		fw := &failingWriter{FailAt: i}
		if err := renderer.renderMetaTag(fw, meta); !errors.Is(err, errTestWrite) {
			t.Fatalf("failAt=%d: err=%v, want %v", i, err, errTestWrite)
		}
	}
}

func TestRenderLinkTagWriteErrorPaths(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})
	link := LinkTag{
		Rel:         "icon",
		Href:        "/favicon.ico",
		Type:        "image/x-icon",
		Sizes:       "32x32",
		CrossOrigin: "anonymous",
		Media:       "screen",
	}

	cw := &countingWriter{}
	if err := renderer.renderLinkTag(cw, link); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := 1; i <= cw.Writes; i++ {
		fw := &failingWriter{FailAt: i}
		if err := renderer.renderLinkTag(fw, link); !errors.Is(err, errTestWrite) {
			t.Fatalf("failAt=%d: err=%v, want %v", i, err, errTestWrite)
		}
	}
}

func TestRenderScriptTagWriteErrorPaths(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})
	script := ScriptTag{
		Src:    "/js/app.js",
		Type:   "text/javascript",
		Defer:  true,
		Async:  true,
		Module: false,
		Inline: "console.log('x')",
	}

	cw := &countingWriter{}
	if err := renderer.renderScriptTag(cw, script); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := 1; i <= cw.Writes; i++ {
		fw := &failingWriter{FailAt: i}
		if err := renderer.renderScriptTag(fw, script); !errors.Is(err, errTestWrite) {
			t.Fatalf("failAt=%d: err=%v, want %v", i, err, errTestWrite)
		}
	}
}

func TestRenderHeadWriteErrorPaths(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})
	page := PageData{
		Title: "Title",
		Meta: []MetaTag{
			{Charset: "utf-8", Name: "description", Content: "c"},
		},
		Links: []LinkTag{
			{Rel: "icon", Href: "/favicon.ico"},
		},
		StyleSheets: []string{"/css/app.css"},
		Styles:      []string{"body{margin:0}"},
		Scripts: []ScriptTag{
			{Src: "/js/defer.js", Defer: true},
			{Src: "/js/async.js", Async: true},
		},
	}

	cw := &countingWriter{}
	if err := renderer.renderHead(cw, page); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := 1; i <= cw.Writes; i++ {
		fw := &failingWriter{FailAt: i}
		if err := renderer.renderHead(fw, page); !errors.Is(err, errTestWrite) {
			t.Fatalf("failAt=%d: err=%v, want %v", i, err, errTestWrite)
		}
	}
}

func TestRenderPageWriteErrorPaths(t *testing.T) {
	renderer := NewRenderer(RendererConfig{})
	page := PageData{
		Body:      nil,
		SessionID: "sess",
		CSRFToken: "csrf",
	}

	cw := &countingWriter{}
	if err := renderer.RenderPage(cw, page); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i := 1; i <= cw.Writes; i++ {
		fw := &failingWriter{FailAt: i}
		if err := renderer.RenderPage(fw, page); !errors.Is(err, errTestWrite) {
			t.Fatalf("failAt=%d: err=%v, want %v", i, err, errTestWrite)
		}
	}
}

func TestStreamingRendererRenderPageWriteErrorPaths(t *testing.T) {
	base := &countingWriter{}
	w := &countingFlushWriter{Writer: base}

	sr := &StreamingRenderer{
		Renderer: NewRenderer(RendererConfig{}),
		flusher:  w,
		w:        w,
	}

	page := PageData{
		Body:  vdom.Div(vdom.Text("x")),
		Title: "Title",
	}

	base.Writes = 0
	if err := sr.RenderPage(page); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	writes := base.Writes
	if writes == 0 {
		t.Fatalf("expected writes")
	}

	for i := 1; i <= writes; i++ {
		fw := &failingWriter{FailAt: i}
		srFail := &StreamingRenderer{
			Renderer: NewRenderer(RendererConfig{}),
			flusher:  nil,
			w:        fw,
		}
		if err := srFail.RenderPage(page); !errors.Is(err, errTestWrite) {
			t.Fatalf("failAt=%d: err=%v, want %v", i, err, errTestWrite)
		}
	}
}

