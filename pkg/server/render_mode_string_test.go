package server

import "testing"

func TestRenderMode_String(t *testing.T) {
	if ModeNormal.String() != "normal" {
		t.Fatalf("ModeNormal.String()=%q, want %q", ModeNormal.String(), "normal")
	}
	if ModePrefetch.String() != "prefetch" {
		t.Fatalf("ModePrefetch.String()=%q, want %q", ModePrefetch.String(), "prefetch")
	}
	var unknown RenderMode = 99
	if unknown.String() != "unknown" {
		t.Fatalf("unknown.String()=%q, want %q", unknown.String(), "unknown")
	}
}

