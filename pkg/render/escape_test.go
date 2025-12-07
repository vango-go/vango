package render

import "testing"

func TestEscapeHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "plain text",
			input:    "Hello, World!",
			expected: "Hello, World!",
		},
		{
			name:     "ampersand",
			input:    "Tom & Jerry",
			expected: "Tom &amp; Jerry",
		},
		{
			name:     "less than",
			input:    "a < b",
			expected: "a &lt; b",
		},
		{
			name:     "greater than",
			input:    "a > b",
			expected: "a &gt; b",
		},
		{
			name:     "double quote",
			input:    `say "hello"`,
			expected: "say &quot;hello&quot;",
		},
		{
			name:     "single quote",
			input:    "it's fine",
			expected: "it&#39;s fine",
		},
		{
			name:     "script tag",
			input:    "<script>alert('xss')</script>",
			expected: "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;",
		},
		{
			name:     "multiple special chars",
			input:    `<a href="test?a=1&b=2">link</a>`,
			expected: `&lt;a href=&quot;test?a=1&amp;b=2&quot;&gt;link&lt;/a&gt;`,
		},
		{
			name:     "unicode preserved",
			input:    "Hello 世界 🌍",
			expected: "Hello 世界 🌍",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeHTML(tt.input)
			if result != tt.expected {
				t.Errorf("escapeHTML(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEscapeAttr(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "plain text",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "ampersand",
			input:    "a&b",
			expected: "a&amp;b",
		},
		{
			name:     "double quote",
			input:    `value="test"`,
			expected: "value=&quot;test&quot;",
		},
		{
			name:     "newline",
			input:    "line1\nline2",
			expected: "line1&#10;line2",
		},
		{
			name:     "carriage return",
			input:    "line1\rline2",
			expected: "line1&#13;line2",
		},
		{
			name:     "tab",
			input:    "col1\tcol2",
			expected: "col1&#9;col2",
		},
		{
			name:     "mixed whitespace",
			input:    "a\n\r\tb",
			expected: "a&#10;&#13;&#9;b",
		},
		{
			name:     "all special chars",
			input:    `<>&"'` + "\n\r\t",
			expected: "&lt;&gt;&amp;&quot;&#39;&#10;&#13;&#9;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeAttr(tt.input)
			if result != tt.expected {
				t.Errorf("escapeAttr(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func BenchmarkEscapeHTML(b *testing.B) {
	b.Run("plain text", func(b *testing.B) {
		s := "Hello, World! This is a plain text string without special characters."
		for i := 0; i < b.N; i++ {
			escapeHTML(s)
		}
	})

	b.Run("with special chars", func(b *testing.B) {
		s := `<script>alert("xss")</script> & more content here`
		for i := 0; i < b.N; i++ {
			escapeHTML(s)
		}
	})
}

func BenchmarkEscapeAttr(b *testing.B) {
	b.Run("plain text", func(b *testing.B) {
		s := "simple-value"
		for i := 0; i < b.N; i++ {
			escapeAttr(s)
		}
	})

	b.Run("with special chars", func(b *testing.B) {
		s := `value="test" with 'quotes' & newlines
and tabs	here`
		for i := 0; i < b.N; i++ {
			escapeAttr(s)
		}
	})
}
