package vango

import (
	"testing"
	"time"
)

// =============================================================================
// Event Modifier Tests (via root package re-exports)
// These tests verify that the root package properly re-exports event modifiers
// from pkg/vango/event_modifiers.go
// =============================================================================

func TestRootPackage_PreventDefault(t *testing.T) {
	handler := func() {}
	mh := PreventDefault(handler)

	if !mh.PreventDefault {
		t.Error("PreventDefault should set PreventDefault flag")
	}
	if mh.Handler == nil {
		t.Error("PreventDefault should preserve handler")
	}
}

func TestRootPackage_StopPropagation(t *testing.T) {
	handler := func() {}
	mh := StopPropagation(handler)

	if !mh.StopPropagation {
		t.Error("StopPropagation should set StopPropagation flag")
	}
	if mh.Handler == nil {
		t.Error("StopPropagation should preserve handler")
	}
}

func TestRootPackage_Self(t *testing.T) {
	handler := func() {}
	mh := Self(handler)

	if !mh.Self {
		t.Error("Self should set Self flag")
	}
	if mh.Handler == nil {
		t.Error("Self should preserve handler")
	}
}

func TestRootPackage_Once(t *testing.T) {
	handler := func() {}
	mh := Once(handler)

	if !mh.Once {
		t.Error("Once should set Once flag")
	}
	if mh.Handler == nil {
		t.Error("Once should preserve handler")
	}
}

func TestRootPackage_Passive(t *testing.T) {
	handler := func() {}
	mh := Passive(handler)

	if !mh.Passive {
		t.Error("Passive should set Passive flag")
	}
	if mh.Handler == nil {
		t.Error("Passive should preserve handler")
	}
}

func TestRootPackage_Capture(t *testing.T) {
	handler := func() {}
	mh := Capture(handler)

	if !mh.Capture {
		t.Error("Capture should set Capture flag")
	}
	if mh.Handler == nil {
		t.Error("Capture should preserve handler")
	}
}

func TestRootPackage_Debounce(t *testing.T) {
	handler := func() {}
	delay := 300 * time.Millisecond
	mh := Debounce(delay, handler)

	if mh.Debounce != delay {
		t.Errorf("Debounce = %v, want %v", mh.Debounce, delay)
	}
	if mh.Handler == nil {
		t.Error("Debounce should preserve handler")
	}
}

func TestRootPackage_Throttle(t *testing.T) {
	handler := func() {}
	interval := 200 * time.Millisecond
	mh := Throttle(interval, handler)

	if mh.Throttle != interval {
		t.Errorf("Throttle = %v, want %v", mh.Throttle, interval)
	}
	if mh.Handler == nil {
		t.Error("Throttle should preserve handler")
	}
}

func TestRootPackage_Hotkey(t *testing.T) {
	handler := func() {}
	mh := Hotkey("Enter", handler)

	if mh.KeyFilter != "Enter" {
		t.Errorf("KeyFilter = %q, want %q", mh.KeyFilter, "Enter")
	}
	if mh.Handler == nil {
		t.Error("Hotkey should preserve handler")
	}
}

func TestRootPackage_Keys(t *testing.T) {
	handler := func() {}
	keys := []string{"Enter", "NumpadEnter", "Space"}
	mh := Keys(keys, handler)

	if len(mh.KeysFilter) != 3 {
		t.Errorf("len(KeysFilter) = %d, want %d", len(mh.KeysFilter), 3)
	}
	for i, key := range keys {
		if mh.KeysFilter[i] != key {
			t.Errorf("KeysFilter[%d] = %q, want %q", i, mh.KeysFilter[i], key)
		}
	}
	if mh.Handler == nil {
		t.Error("Keys should preserve handler")
	}
}

func TestRootPackage_KeyWithModifiers(t *testing.T) {
	handler := func() {}
	mh := KeyWithModifiers("s", Ctrl|Shift, handler)

	if mh.KeyFilter != "s" {
		t.Errorf("KeyFilter = %q, want %q", mh.KeyFilter, "s")
	}
	if mh.KeyModifiers != (Ctrl | Shift) {
		t.Errorf("KeyModifiers = %v, want %v", mh.KeyModifiers, Ctrl|Shift)
	}
	if mh.Handler == nil {
		t.Error("KeyWithModifiers should preserve handler")
	}
}

func TestRootPackage_ModifierChaining(t *testing.T) {
	handler := func() {}

	// Chain multiple modifiers
	mh := PreventDefault(StopPropagation(Once(handler)))

	if !mh.PreventDefault {
		t.Error("Chained modifiers should preserve PreventDefault")
	}
	if !mh.StopPropagation {
		t.Error("Chained modifiers should preserve StopPropagation")
	}
	if !mh.Once {
		t.Error("Chained modifiers should preserve Once")
	}
}

func TestRootPackage_KeyModConstants(t *testing.T) {
	// Verify key modifier constants are exported correctly
	if Ctrl == 0 {
		t.Error("Ctrl should not be 0")
	}
	if Shift == 0 {
		t.Error("Shift should not be 0")
	}
	if Alt == 0 {
		t.Error("Alt should not be 0")
	}
	if Meta == 0 {
		t.Error("Meta should not be 0")
	}

	// They should all be different
	mods := []KeyMod{Ctrl, Shift, Alt, Meta}
	seen := make(map[KeyMod]bool)
	for _, m := range mods {
		if seen[m] {
			t.Errorf("Duplicate modifier value: %v", m)
		}
		seen[m] = true
	}
}

func TestRootPackage_KeyMod_Has(t *testing.T) {
	mods := Ctrl | Shift

	if !mods.Has(Ctrl) {
		t.Error("mods.Has(Ctrl) should be true")
	}
	if !mods.Has(Shift) {
		t.Error("mods.Has(Shift) should be true")
	}
	if mods.Has(Alt) {
		t.Error("mods.Has(Alt) should be false")
	}
	if mods.Has(Meta) {
		t.Error("mods.Has(Meta) should be false")
	}
}

// =============================================================================
// Hook System Tests
// =============================================================================

func TestRootPackage_Hook_CreatesAttr(t *testing.T) {
	attr := Hook("tooltip", map[string]any{
		"content":   "Hello",
		"placement": "top",
	})

	if attr.Key != "_hook" {
		t.Errorf("Hook attr.Key = %q, want %q", attr.Key, "_hook")
	}
	if attr.Value == nil {
		t.Error("Hook attr.Value should not be nil")
	}
}

func TestRootPackage_OnEvent_FiltersEventsByName(t *testing.T) {
	calls := 0
	attr := OnEvent("save", func(e HookEvent) {
		calls++
	})

	if attr.Key != "onhook" {
		t.Errorf("OnEvent attr.Key = %q, want %q", attr.Key, "onhook")
	}

	handler, ok := attr.Value.(func(HookEvent))
	if !ok {
		t.Fatalf("OnEvent attr.Value type = %T, want func(HookEvent)", attr.Value)
	}

	// Should only call for matching event name
	handler(HookEvent{Name: "other"})
	handler(HookEvent{Name: "save"})
	handler(HookEvent{Name: "cancel"})
	handler(HookEvent{Name: "save"})

	if calls != 2 {
		t.Errorf("calls = %d, want %d", calls, 2)
	}
}

// =============================================================================
// Key Constant Tests
// =============================================================================

func TestRootPackage_KeyConstants(t *testing.T) {
	// Verify key constants are exported
	keyConstants := []string{
		KeyEnter, KeyEscape, KeySpace, KeyTab, KeyBackspace, KeyDelete,
		KeyArrowUp, KeyArrowDown, KeyArrowLeft, KeyArrowRight,
		KeyHome, KeyEnd, KeyPageUp, KeyPageDown,
		KeyF1, KeyF2, KeyF3, KeyF4, KeyF5, KeyF6,
		KeyF7, KeyF8, KeyF9, KeyF10, KeyF11, KeyF12,
		KeyControl, KeyShift, KeyAlt, KeyMeta,
		KeyInsert, KeyPrintScreen, KeyScrollLock, KeyPause,
		KeyCapsLock, KeyNumLock, KeyContextMenu,
	}

	for _, key := range keyConstants {
		if key == "" {
			t.Error("Key constant should not be empty")
		}
	}
}

// =============================================================================
// Event Type Tests
// =============================================================================

func TestRootPackage_EventTypesExported(t *testing.T) {
	// Verify event types are properly aliased
	var _ MouseEvent
	var _ KeyboardEvent
	var _ InputEvent
	var _ WheelEvent
	var _ DragEvent
	var _ DropEvent
	var _ TouchEvent
	var _ AnimationEvent
	var _ TransitionEvent
	var _ ScrollEvent
	var _ ResizeEvent
	var _ FormData
	var _ HookEvent
	var _ NavigateEvent
}

func TestRootPackage_MouseEvent_Fields(t *testing.T) {
	// Verify MouseEvent has expected fields
	e := MouseEvent{
		ClientX:  100,
		ClientY:  200,
		Button:   0,
		ShiftKey: true,
		CtrlKey:  false,
	}

	if e.ClientX != 100 {
		t.Errorf("ClientX = %d, want %d", e.ClientX, 100)
	}
	if e.ClientY != 200 {
		t.Errorf("ClientY = %d, want %d", e.ClientY, 200)
	}
	if !e.ShiftKey {
		t.Error("ShiftKey should be true")
	}
}

func TestRootPackage_KeyboardEvent_Fields(t *testing.T) {
	e := KeyboardEvent{
		Key:      "Enter",
		Code:     "Enter",
		ShiftKey: true,
		CtrlKey:  true,
		AltKey:   false,
		MetaKey:  false,
		Repeat:   false,
	}

	if e.Key != "Enter" {
		t.Errorf("Key = %q, want %q", e.Key, "Enter")
	}
	if !e.ShiftKey {
		t.Error("ShiftKey should be true")
	}
	if !e.CtrlKey {
		t.Error("CtrlKey should be true")
	}
}

func TestRootPackage_InputEvent_Fields(t *testing.T) {
	e := InputEvent{
		Value: "hello world",
	}

	if e.Value != "hello world" {
		t.Errorf("Value = %q, want %q", e.Value, "hello world")
	}
}

func TestRootPackage_FormData_Fields(t *testing.T) {
	fd := NewFormData(map[string][]string{
		"name": {"Alice"},
		"tags": {"go", "web"},
	})

	if got := fd.Get("name"); got != "Alice" {
		t.Errorf("Get(\"name\") = %q, want %q", got, "Alice")
	}
	if got := fd.GetAll("tags"); len(got) != 2 || got[0] != "go" || got[1] != "web" {
		t.Errorf("GetAll(\"tags\") = %v, want [\"go\", \"web\"]", got)
	}
}

func TestRootPackage_NewFormDataFromSingle(t *testing.T) {
	fd := NewFormDataFromSingle(map[string]string{
		"name":  "Bob",
		"email": "bob@example.com",
	})

	if got := fd.Get("name"); got != "Bob" {
		t.Errorf("Get(\"name\") = %q, want %q", got, "Bob")
	}
	if got := fd.Get("email"); got != "bob@example.com" {
		t.Errorf("Get(\"email\") = %q, want %q", got, "bob@example.com")
	}
}

func TestRootPackage_HookEvent_Fields(t *testing.T) {
	e := HookEvent{
		Name: "mounted",
		Data: map[string]any{
			"elementId": "test-element",
		},
	}

	if e.Name != "mounted" {
		t.Errorf("Name = %q, want %q", e.Name, "mounted")
	}
	if e.Data == nil {
		t.Error("Data should not be nil")
	}
}
