package server

import (
	"testing"
	"time"

	"github.com/vango-dev/vango/v2/pkg/protocol"
)

func TestMouseEvent(t *testing.T) {
	me := MouseEvent{
		ClientX:  100,
		ClientY:  200,
		Button:   0,
		CtrlKey:  true,
		ShiftKey: false,
		AltKey:   true,
		MetaKey:  false,
	}

	if me.ClientX != 100 {
		t.Errorf("ClientX = %d, want 100", me.ClientX)
	}
	if me.ClientY != 200 {
		t.Errorf("ClientY = %d, want 200", me.ClientY)
	}
	if !me.CtrlKey {
		t.Error("CtrlKey should be true")
	}
	if me.ShiftKey {
		t.Error("ShiftKey should be false")
	}
	if !me.AltKey {
		t.Error("AltKey should be true")
	}
}

func TestKeyboardEvent(t *testing.T) {
	ke := KeyboardEvent{
		Key:      "Enter",
		CtrlKey:  false,
		ShiftKey: true,
		AltKey:   false,
		MetaKey:  true,
	}

	if ke.Key != "Enter" {
		t.Errorf("Key = %s, want Enter", ke.Key)
	}
	if !ke.ShiftKey {
		t.Error("ShiftKey should be true")
	}
	if !ke.MetaKey {
		t.Error("MetaKey should be true")
	}
}

func TestFormDataGet(t *testing.T) {
	fd := FormData{
		values: map[string]string{
			"name":  "John",
			"email": "john@example.com",
		},
	}

	if fd.Get("name") != "John" {
		t.Errorf("Get(name) = %s, want John", fd.Get("name"))
	}
	if fd.Get("email") != "john@example.com" {
		t.Errorf("Get(email) = %s, want john@example.com", fd.Get("email"))
	}
	if fd.Get("missing") != "" {
		t.Error("Get(missing) should return empty string")
	}
}

func TestFormDataHas(t *testing.T) {
	fd := FormData{
		values: map[string]string{
			"present": "value",
		},
	}

	if !fd.Has("present") {
		t.Error("Has(present) should return true")
	}
	if fd.Has("missing") {
		t.Error("Has(missing) should return false")
	}
}

func TestFormDataAll(t *testing.T) {
	fd := FormData{
		values: map[string]string{
			"a": "1",
			"b": "2",
			"c": "3",
		},
	}

	all := fd.All()
	if len(all) != 3 {
		t.Errorf("len(All()) = %d, want 3", len(all))
	}
	if all["a"] != "1" || all["b"] != "2" || all["c"] != "3" {
		t.Error("All should return all values")
	}
}

func TestFormDataEmpty(t *testing.T) {
	fd := FormData{
		values: nil,
	}

	// Should not panic with nil values
	if fd.Get("any") != "" {
		t.Error("Get on nil values should return empty string")
	}
	if fd.Has("any") {
		t.Error("Has on nil values should return false")
	}
}

func TestHookEvent(t *testing.T) {
	he := HookEvent{
		Name: "custom-hook",
		Data: map[string]any{
			"key":   "value",
			"count": 42,
			"flag":  true,
		},
	}

	if he.Name != "custom-hook" {
		t.Errorf("Name = %s, want custom-hook", he.Name)
	}
	if he.Get("key") != "value" {
		t.Error("Get(key) should return value")
	}
	if he.GetString("key") != "value" {
		t.Error("GetString(key) should return value")
	}
	if he.GetInt("count") != 42 {
		t.Error("GetInt(count) should return 42")
	}
	if he.GetBool("flag") != true {
		t.Error("GetBool(flag) should return true")
	}
}

func TestHookEventGetInt(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]any
		key      string
		expected int
	}{
		{"int", map[string]any{"v": 42}, "v", 42},
		{"int64", map[string]any{"v": int64(42)}, "v", 42},
		{"float64", map[string]any{"v": float64(42.9)}, "v", 42},
		{"missing", map[string]any{}, "v", 0},
		{"wrong type", map[string]any{"v": "str"}, "v", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			he := HookEvent{Data: tt.data}
			if got := he.GetInt(tt.key); got != tt.expected {
				t.Errorf("GetInt() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestHookEventGetBool(t *testing.T) {
	he := HookEvent{Data: map[string]any{
		"true":    true,
		"false":   false,
		"string":  "true",
		"missing": nil,
	}}

	if !he.GetBool("true") {
		t.Error("GetBool(true) should return true")
	}
	if he.GetBool("false") {
		t.Error("GetBool(false) should return false")
	}
	if he.GetBool("string") {
		t.Error("GetBool(string) should return false for non-bool")
	}
	if he.GetBool("nonexistent") {
		t.Error("GetBool(nonexistent) should return false")
	}
}

func TestScrollEvent(t *testing.T) {
	se := ScrollEvent{
		ScrollTop:  100,
		ScrollLeft: 200,
	}

	if se.ScrollTop != 100 {
		t.Errorf("ScrollTop = %d, want 100", se.ScrollTop)
	}
	if se.ScrollLeft != 200 {
		t.Errorf("ScrollLeft = %d, want 200", se.ScrollLeft)
	}
}

func TestResizeEvent(t *testing.T) {
	re := ResizeEvent{
		Width:  1920,
		Height: 1080,
	}

	if re.Width != 1920 {
		t.Errorf("Width = %d, want 1920", re.Width)
	}
	if re.Height != 1080 {
		t.Errorf("Height = %d, want 1080", re.Height)
	}
}

func TestTouchEvent(t *testing.T) {
	te := TouchEvent{
		Touches: []TouchPoint{
			{ID: 0, ClientX: 100, ClientY: 200},
			{ID: 1, ClientX: 150, ClientY: 250},
		},
	}

	if len(te.Touches) != 2 {
		t.Errorf("len(Touches) = %d, want 2", len(te.Touches))
	}
	if te.Touches[0].ClientX != 100 {
		t.Error("First touch ClientX should be 100")
	}
	if te.Touches[1].ID != 1 {
		t.Error("Second touch ID should be 1")
	}
}

func TestNavigateEvent(t *testing.T) {
	ne := NavigateEvent{
		Path:    "/users/123",
		Replace: true,
	}

	if ne.Path != "/users/123" {
		t.Errorf("Path = %s, want /users/123", ne.Path)
	}
	if !ne.Replace {
		t.Error("Replace should be true")
	}
}

func TestEventFromProtocol(t *testing.T) {
	pe := &protocol.Event{
		Seq:  123,
		Type: protocol.EventClick,
		HID:  "hid-test",
	}

	event := eventFromProtocol(pe, nil)

	if event.Seq != 123 {
		t.Errorf("Seq = %d, want 123", event.Seq)
	}
	if event.Type != protocol.EventClick {
		t.Errorf("Type = %v, want EventClick", event.Type)
	}
	if event.HID != "hid-test" {
		t.Errorf("HID = %s, want hid-test", event.HID)
	}
	if event.Time.IsZero() {
		t.Error("Time should be set")
	}
}

func TestEventTimestamp(t *testing.T) {
	before := time.Now()
	pe := &protocol.Event{
		Seq:  1,
		Type: protocol.EventInput,
		HID:  "input-1",
	}
	event := eventFromProtocol(pe, nil)
	after := time.Now()

	if event.Time.Before(before) || event.Time.After(after) {
		t.Error("Event time should be between before and after")
	}
}

func TestWrapHandlerSimple(t *testing.T) {
	called := false
	fn := func() {
		called = true
	}

	handler := wrapHandler(fn)
	if handler == nil {
		t.Fatal("wrapHandler should return a handler for func()")
	}

	handler(&Event{})
	if !called {
		t.Error("Handler should have called the function")
	}
}

func TestWrapHandlerWithEvent(t *testing.T) {
	var receivedEvent *Event
	fn := func(e *Event) {
		receivedEvent = e
	}

	handler := wrapHandler(fn)
	if handler == nil {
		t.Fatal("wrapHandler should return a handler for func(*Event)")
	}

	testEvent := &Event{Seq: 42}
	handler(testEvent)
	if receivedEvent != testEvent {
		t.Error("Handler should pass through the event")
	}
}

func TestWrapHandlerWithString(t *testing.T) {
	var receivedValue string
	fn := func(value string) {
		receivedValue = value
	}

	handler := wrapHandler(fn)
	if handler == nil {
		t.Fatal("wrapHandler should return a handler for func(string)")
	}

	// String payload
	testEvent := &Event{
		Payload: "test-value",
	}
	handler(testEvent)
	if receivedValue != "test-value" {
		t.Errorf("receivedValue = %s, want test-value", receivedValue)
	}
}

func TestWrapHandlerUnknownType(t *testing.T) {
	fn := func(x int, y int) int { return x + y }

	handler := wrapHandler(fn)
	// Should return a no-op handler, not nil
	if handler == nil {
		t.Error("wrapHandler should return no-op handler for unsupported types")
	}
	// Should not panic when called
	handler(&Event{})
}

func TestWrapHandlerNil(t *testing.T) {
	handler := wrapHandler(nil)
	// nil input returns no-op handler
	if handler == nil {
		t.Error("wrapHandler(nil) should return no-op handler")
	}
	// Should not panic
	handler(&Event{})
}

func TestIsClickLike(t *testing.T) {
	clickLike := []protocol.EventType{
		protocol.EventClick,
		protocol.EventDblClick,
		protocol.EventFocus,
		protocol.EventBlur,
		protocol.EventMouseEnter,
		protocol.EventMouseLeave,
	}

	for _, et := range clickLike {
		if !isClickLike(et) {
			t.Errorf("isClickLike(%v) should return true", et)
		}
	}

	notClickLike := []protocol.EventType{
		protocol.EventInput,
		protocol.EventKeyDown,
		protocol.EventSubmit,
	}

	for _, et := range notClickLike {
		if isClickLike(et) {
			t.Errorf("isClickLike(%v) should return false", et)
		}
	}
}

func TestIsInputLike(t *testing.T) {
	if !isInputLike(protocol.EventInput) {
		t.Error("isInputLike(EventInput) should return true")
	}
	if !isInputLike(protocol.EventChange) {
		t.Error("isInputLike(EventChange) should return true")
	}
	if isInputLike(protocol.EventClick) {
		t.Error("isInputLike(EventClick) should return false")
	}
}

func TestIsMouseEvent(t *testing.T) {
	mouseEvents := []protocol.EventType{
		protocol.EventMouseDown,
		protocol.EventMouseUp,
		protocol.EventMouseMove,
		protocol.EventDragStart,
		protocol.EventDragEnd,
		protocol.EventDrop,
	}

	for _, et := range mouseEvents {
		if !isMouseEvent(et) {
			t.Errorf("isMouseEvent(%v) should return true", et)
		}
	}

	if isMouseEvent(protocol.EventClick) {
		t.Error("isMouseEvent(EventClick) should return false")
	}
}

func TestIsKeyboardEvent(t *testing.T) {
	kbEvents := []protocol.EventType{
		protocol.EventKeyDown,
		protocol.EventKeyUp,
		protocol.EventKeyPress,
	}

	for _, et := range kbEvents {
		if !isKeyboardEvent(et) {
			t.Errorf("isKeyboardEvent(%v) should return true", et)
		}
	}

	if isKeyboardEvent(protocol.EventClick) {
		t.Error("isKeyboardEvent(EventClick) should return false")
	}
}
