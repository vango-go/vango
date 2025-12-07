package render

import "strings"

// escapeHTML escapes text for safe inclusion in HTML content.
// It converts special characters to their HTML entity equivalents
// to prevent XSS attacks.
func escapeHTML(s string) string {
	var buf strings.Builder
	buf.Grow(len(s))

	for _, r := range s {
		switch r {
		case '&':
			buf.WriteString("&amp;")
		case '<':
			buf.WriteString("&lt;")
		case '>':
			buf.WriteString("&gt;")
		case '"':
			buf.WriteString("&quot;")
		case '\'':
			buf.WriteString("&#39;")
		default:
			buf.WriteRune(r)
		}
	}

	return buf.String()
}

// escapeAttr escapes text for safe inclusion in HTML attribute values.
// In addition to the standard HTML entities, it also escapes
// whitespace characters that could break attribute parsing.
func escapeAttr(s string) string {
	var buf strings.Builder
	buf.Grow(len(s))

	for _, r := range s {
		switch r {
		case '&':
			buf.WriteString("&amp;")
		case '<':
			buf.WriteString("&lt;")
		case '>':
			buf.WriteString("&gt;")
		case '"':
			buf.WriteString("&quot;")
		case '\'':
			buf.WriteString("&#39;")
		case '\n':
			buf.WriteString("&#10;")
		case '\r':
			buf.WriteString("&#13;")
		case '\t':
			buf.WriteString("&#9;")
		default:
			buf.WriteRune(r)
		}
	}

	return buf.String()
}
