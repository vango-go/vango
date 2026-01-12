package render

import (
	"strings"
	"testing"
)

func extractAttrValue(t *testing.T, s string, attr string) string {
	t.Helper()

	needle := attr + "="
	idx := strings.Index(s, needle)
	if idx == -1 {
		t.Fatalf("expected %q in %q", needle, s)
	}

	start := idx + len(needle)
	if start >= len(s) {
		t.Fatalf("malformed attribute %q in %q", attr, s)
	}

	quote := s[start]
	if quote != '"' && quote != '\'' {
		t.Fatalf("expected quote for %q in %q", attr, s)
	}
	start++

	endRel := strings.IndexByte(s[start:], quote)
	if endRel == -1 {
		t.Fatalf("unterminated attribute %q in %q", attr, s)
	}

	return s[start : start+endRel]
}
