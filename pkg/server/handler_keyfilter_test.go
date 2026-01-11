package server

import (
	"testing"

	"github.com/vango-go/vango/pkg/protocol"
	"github.com/vango-go/vango/pkg/vango"
)

func TestEvent_TypeString(t *testing.T) {
	e := &Event{Type: protocol.EventClick}
	if e.TypeString() != "Click" {
		t.Fatalf("TypeString()=%q, want %q", e.TypeString(), "Click")
	}
}

func TestKeyMatchesFilter(t *testing.T) {
	data := &protocol.KeyboardEventData{
		Key:       "Enter",
		Modifiers: protocol.ModCtrl,
	}

	if keyMatchesFilter(data, vango.ModifiedHandler{KeyFilter: "Escape"}) {
		t.Fatal("keyMatchesFilter()=true, want false for wrong key")
	}

	if !keyMatchesFilter(data, vango.ModifiedHandler{KeysFilter: []string{"Tab", "Enter"}}) {
		t.Fatal("keyMatchesFilter()=false, want true for included key")
	}

	// Exact modifier match: data has Ctrl only.
	if !keyMatchesFilter(data, vango.ModifiedHandler{KeyModifiers: vango.Ctrl}) {
		t.Fatal("keyMatchesFilter()=false, want true for matching modifiers")
	}
	if keyMatchesFilter(data, vango.ModifiedHandler{KeyModifiers: vango.Ctrl | vango.Shift}) {
		t.Fatal("keyMatchesFilter()=true, want false for extra required modifiers")
	}
	if keyMatchesFilter(data, vango.ModifiedHandler{KeyModifiers: vango.Shift}) {
		t.Fatal("keyMatchesFilter()=true, want false for mismatched modifiers")
	}
}

