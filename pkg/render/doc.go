// Package render provides server-side rendering (SSR) for Vango components.
//
// The render package converts VNode trees into HTML strings or streams,
// handling all aspects of producing valid, secure HTML output including:
//
//   - HTML5 compliant element rendering
//   - Proper text and attribute escaping (XSS prevention)
//   - Void element handling (input, br, img, etc.)
//   - Boolean attribute handling (disabled, checked, etc.)
//   - Hydration ID generation for client-side interactivity
//   - Full page rendering with DOCTYPE, head, body
//   - Thin client script injection
//
// # Basic Usage
//
// To render a VNode tree to a string:
//
//	renderer := render.NewRenderer(render.RendererConfig{})
//	html, err := renderer.RenderToString(node)
//
// To stream HTML to a writer:
//
//	renderer := render.NewRenderer(render.RendererConfig{})
//	err := renderer.RenderToWriter(w, node)
//
// # Full Page Rendering
//
// To render a complete HTML document:
//
//	page := render.PageData{
//	    Body:       bodyNode,
//	    Title:      "My Page",
//	    SessionID:  session.ID,
//	    CSRFToken:  session.CSRFToken,
//	}
//	err := renderer.RenderPage(w, page)
//
// # Hydration IDs
//
// Elements with event handlers automatically receive a data-hid attribute
// for client-side hydration. The handlers are collected during rendering
// and can be retrieved via GetHandlers().
//
// # Streaming
//
// For large pages, use StreamingRenderer to flush content incrementally:
//
//	sr := render.NewStreamingRenderer(w, config)
//	err := sr.RenderPage(page)
//
// # Security
//
// All text content is escaped by default to prevent XSS attacks.
// Raw HTML can be inserted using KindRaw nodes, but should only be
// used with trusted content.
package render
