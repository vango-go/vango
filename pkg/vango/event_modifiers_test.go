package vango

import (
	"reflect"
	"testing"
	"time"
)

func TestKeyMod_HasAndString(t *testing.T) {
	var none KeyMod
	if none.Has(Ctrl) {
		t.Fatalf("none.Has(Ctrl) should be false")
	}
	if got := none.String(); got != "none" {
		t.Fatalf("none.String() = %q, want %q", got, "none")
	}

	mods := Ctrl | Shift | Meta
	if !mods.Has(Ctrl) || !mods.Has(Shift) || mods.Has(Alt) || !mods.Has(Meta) {
		t.Fatalf("Has results unexpected for %v", mods)
	}
	if got := mods.String(); got != "Ctrl+Shift+Meta" {
		t.Fatalf("String() = %q, want %q", got, "Ctrl+Shift+Meta")
	}
}

func TestModifiedHandler_Unwrap_Merge_Modifiers(t *testing.T) {
	base := func() {}

	nested := ModifiedHandler{
		Handler: ModifiedHandler{
			Handler: base,
			Once:    true,
		},
		PreventDefault: true,
	}
	if nested.Unwrap() == nil {
		t.Fatalf("Unwrap() should return innermost handler")
	}
	if reflect.ValueOf(nested.Unwrap()).Pointer() != reflect.ValueOf(base).Pointer() {
		t.Fatalf("Unwrap() should return original handler")
	}

	merged := ModifiedHandler{Handler: base, Self: true}.merge(ModifiedHandler{
		Handler:          nested,
		PreventDefault:   true,
		StopPropagation: true,
		Debounce:        200 * time.Millisecond,
		KeyFilter:       "Enter",
		KeyModifiers:    Ctrl,
	})
	if !merged.Self || !merged.PreventDefault || !merged.StopPropagation {
		t.Fatalf("merge() should combine flags; got %+v", merged)
	}
	if merged.Debounce != 200*time.Millisecond {
		t.Fatalf("merge() Debounce = %v, want %v", merged.Debounce, 200*time.Millisecond)
	}
	if merged.KeyFilter != "Enter" || merged.KeyModifiers != Ctrl {
		t.Fatalf("merge() key filter/mods = (%q, %v), want (%q, %v)", merged.KeyFilter, merged.KeyModifiers, "Enter", Ctrl)
	}
	if reflect.ValueOf(merged.Handler).Pointer() != reflect.ValueOf(base).Pointer() {
		t.Fatalf("merge() should preserve innermost handler")
	}

	mh := StopPropagation(PreventDefault(base))
	if !mh.PreventDefault || !mh.StopPropagation {
		t.Fatalf("expected chained flags to be set; got %+v", mh)
	}
	if reflect.ValueOf(mh.Unwrap()).Pointer() != reflect.ValueOf(base).Pointer() {
		t.Fatalf("modifier chain should keep original handler")
	}

	other := Self(Once(Passive(Capture(Debounce(10*time.Millisecond, Throttle(5*time.Millisecond, base))))))
	if !other.Self || !other.Once || !other.Passive || !other.Capture {
		t.Fatalf("expected flags Self/Once/Passive/Capture to be set; got %+v", other)
	}
	if other.Debounce != 10*time.Millisecond || other.Throttle != 5*time.Millisecond {
		t.Fatalf("expected Debounce/Throttle to be set; got %+v", other)
	}

	k := KeyWithModifiers("s", Ctrl|Shift, base)
	if k.KeyFilter != "s" || k.KeyModifiers != (Ctrl|Shift) {
		t.Fatalf("KeyWithModifiers key/mods = (%q, %v), want (%q, %v)", k.KeyFilter, k.KeyModifiers, "s", Ctrl|Shift)
	}
	kh := Hotkey("Enter", base)
	if kh.KeyFilter != "Enter" {
		t.Fatalf("Hotkey key = %q, want %q", kh.KeyFilter, "Enter")
	}
	kk := Keys([]string{"Enter", "NumpadEnter"}, base)
	if len(kk.KeysFilter) != 2 || kk.KeysFilter[0] != "Enter" || kk.KeysFilter[1] != "NumpadEnter" {
		t.Fatalf("KeysFilter = %+v, want %+v", kk.KeysFilter, []string{"Enter", "NumpadEnter"})
	}
}
