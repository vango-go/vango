package server

import "testing"

func TestApplyNavigateOptions_DefaultsAndNilOptions(t *testing.T) {
	applied := ApplyNavigateOptions(nil, WithReplace(), nil)
	if !applied.Replace {
		t.Fatal("Replace=false, want true")
	}
	if !applied.Scroll {
		t.Fatal("Scroll=false, want true by default")
	}
}

func TestBuildNavigateURL_AppliesParamsAndRejectsAbsoluteURLs(t *testing.T) {
	full, applied := BuildNavigateURL("/projects/123", WithNavigateParams(map[string]any{
		"tab": "details",
		"n":   5,
	}))
	if full == "" {
		t.Fatal("fullPath empty, want non-empty")
	}
	if applied.Params["tab"] != "details" {
		t.Fatalf("Params[tab]=%v, want %q", applied.Params["tab"], "details")
	}
	if full == "/projects/123" {
		t.Fatalf("fullPath=%q, want query params applied", full)
	}

	if got, _ := BuildNavigateURL("https://example.com/x"); got != "" {
		t.Fatalf("fullPath=%q, want empty for absolute URL", got)
	}
	if got, _ := BuildNavigateURL("//example.com/x"); got != "" {
		t.Fatalf("fullPath=%q, want empty for scheme-relative URL", got)
	}
}

