package protocol

import (
	"testing"
)

func TestEventEncodeDecode(t *testing.T) {
	tests := []struct {
		name  string
		event *Event
	}{
		{
			name: "click",
			event: &Event{
				Seq:  1,
				Type: EventClick,
				HID:  "h1",
			},
		},
		{
			name: "dblclick",
			event: &Event{
				Seq:  2,
				Type: EventDblClick,
				HID:  "h42",
			},
		},
		{
			name: "input",
			event: &Event{
				Seq:     3,
				Type:    EventInput,
				HID:     "h5",
				Payload: "hello world",
			},
		},
		{
			name: "change",
			event: &Event{
				Seq:     4,
				Type:    EventChange,
				HID:     "h6",
				Payload: "new value",
			},
		},
		{
			name: "submit",
			event: &Event{
				Seq:  5,
				Type: EventSubmit,
				HID:  "h7",
				Payload: &SubmitEventData{
					Fields: map[string]string{
						"name":  "John",
						"email": "john@example.com",
					},
				},
			},
		},
		{
			name: "keydown",
			event: &Event{
				Seq:  6,
				Type: EventKeyDown,
				HID:  "h8",
				Payload: &KeyboardEventData{
					Key:       "Enter",
					Modifiers: ModCtrl | ModShift,
				},
			},
		},
		{
			name: "keyup",
			event: &Event{
				Seq:  7,
				Type: EventKeyUp,
				HID:  "h9",
				Payload: &KeyboardEventData{
					Key:       "Escape",
					Modifiers: 0,
				},
			},
		},
		{
			name: "mousedown",
			event: &Event{
				Seq:  8,
				Type: EventMouseDown,
				HID:  "h10",
				Payload: &MouseEventData{
					ClientX:   100,
					ClientY:   200,
					Button:    0,
					Modifiers: ModAlt,
				},
			},
		},
		{
			name: "mousemove_negative",
			event: &Event{
				Seq:  9,
				Type: EventMouseMove,
				HID:  "h11",
				Payload: &MouseEventData{
					ClientX:   -50,
					ClientY:   -100,
					Button:    2,
					Modifiers: ModMeta,
				},
			},
		},
		{
			name: "scroll",
			event: &Event{
				Seq:  10,
				Type: EventScroll,
				HID:  "h12",
				Payload: &ScrollEventData{
					ScrollTop:  500,
					ScrollLeft: 100,
				},
			},
		},
		{
			name: "resize",
			event: &Event{
				Seq:  11,
				Type: EventResize,
				HID:  "h13",
				Payload: &ResizeEventData{
					Width:  1920,
					Height: 1080,
				},
			},
		},
		{
			name: "touchstart",
			event: &Event{
				Seq:  12,
				Type: EventTouchStart,
				HID:  "h14",
				Payload: &TouchEventData{
					Touches: []TouchPoint{
						{ID: 0, ClientX: 100, ClientY: 200},
						{ID: 1, ClientX: 300, ClientY: 400},
					},
				},
			},
		},
		{
			name: "dragstart",
			event: &Event{
				Seq:  13,
				Type: EventDragStart,
				HID:  "h15",
				Payload: &MouseEventData{
					ClientX:   150,
					ClientY:   250,
					Modifiers: ModShift,
				},
			},
		},
		{
			name: "hook",
			event: &Event{
				Seq:  14,
				Type: EventHook,
				HID:  "h16",
				Payload: &HookEventData{
					Name: "sortable:reorder",
					Data: map[string]any{
						"from":   int64(0),
						"to":     int64(2),
						"itemId": "item-123",
					},
				},
			},
		},
		{
			name: "navigate",
			event: &Event{
				Seq:  15,
				Type: EventNavigate,
				HID:  "",
				Payload: &NavigateEventData{
					Path:    "/users/123",
					Replace: true,
				},
			},
		},
		{
			name: "custom",
			event: &Event{
				Seq:  16,
				Type: EventCustom,
				HID:  "h17",
				Payload: &CustomEventData{
					Name: "my-event",
					Data: []byte{0x01, 0x02, 0x03},
				},
			},
		},
		{
			name: "focus",
			event: &Event{
				Seq:  17,
				Type: EventFocus,
				HID:  "h18",
			},
		},
		{
			name: "blur",
			event: &Event{
				Seq:  18,
				Type: EventBlur,
				HID:  "h19",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Encode
			encoded := EncodeEvent(tc.event)
			if len(encoded) == 0 {
				t.Fatal("Encoded event is empty")
			}

			// Decode
			decoded, err := DecodeEvent(encoded)
			if err != nil {
				t.Fatalf("DecodeEvent() error = %v", err)
			}

			// Verify basic fields
			if decoded.Seq != tc.event.Seq {
				t.Errorf("Seq = %d, want %d", decoded.Seq, tc.event.Seq)
			}
			if decoded.Type != tc.event.Type {
				t.Errorf("Type = %v, want %v", decoded.Type, tc.event.Type)
			}
			if decoded.HID != tc.event.HID {
				t.Errorf("HID = %q, want %q", decoded.HID, tc.event.HID)
			}

			// Verify payloads
			verifyPayload(t, tc.name, decoded.Payload, tc.event.Payload)
		})
	}
}

func verifyPayload(t *testing.T, _ string, got, want any) {
	t.Helper()

	if want == nil {
		if got != nil {
			t.Errorf("Payload = %v, want nil", got)
		}
		return
	}

	switch w := want.(type) {
	case string:
		if g, ok := got.(string); !ok || g != w {
			t.Errorf("Payload = %v, want %q", got, w)
		}

	case *SubmitEventData:
		g, ok := got.(*SubmitEventData)
		if !ok {
			t.Errorf("Payload type = %T, want *SubmitEventData", got)
			return
		}
		if len(g.Fields) != len(w.Fields) {
			t.Errorf("Fields count = %d, want %d", len(g.Fields), len(w.Fields))
			return
		}
		for k, v := range w.Fields {
			if g.Fields[k] != v {
				t.Errorf("Field[%q] = %q, want %q", k, g.Fields[k], v)
			}
		}

	case *KeyboardEventData:
		g, ok := got.(*KeyboardEventData)
		if !ok {
			t.Errorf("Payload type = %T, want *KeyboardEventData", got)
			return
		}
		if g.Key != w.Key {
			t.Errorf("Key = %q, want %q", g.Key, w.Key)
		}
		if g.Modifiers != w.Modifiers {
			t.Errorf("Modifiers = %v, want %v", g.Modifiers, w.Modifiers)
		}

	case *MouseEventData:
		g, ok := got.(*MouseEventData)
		if !ok {
			t.Errorf("Payload type = %T, want *MouseEventData", got)
			return
		}
		if g.ClientX != w.ClientX || g.ClientY != w.ClientY {
			t.Errorf("Position = (%d,%d), want (%d,%d)", g.ClientX, g.ClientY, w.ClientX, w.ClientY)
		}
		if g.Button != w.Button {
			t.Errorf("Button = %d, want %d", g.Button, w.Button)
		}
		if g.Modifiers != w.Modifiers {
			t.Errorf("Modifiers = %v, want %v", g.Modifiers, w.Modifiers)
		}

	case *ScrollEventData:
		g, ok := got.(*ScrollEventData)
		if !ok {
			t.Errorf("Payload type = %T, want *ScrollEventData", got)
			return
		}
		if g.ScrollTop != w.ScrollTop || g.ScrollLeft != w.ScrollLeft {
			t.Errorf("Scroll = (%d,%d), want (%d,%d)", g.ScrollTop, g.ScrollLeft, w.ScrollTop, w.ScrollLeft)
		}

	case *ResizeEventData:
		g, ok := got.(*ResizeEventData)
		if !ok {
			t.Errorf("Payload type = %T, want *ResizeEventData", got)
			return
		}
		if g.Width != w.Width || g.Height != w.Height {
			t.Errorf("Size = (%d,%d), want (%d,%d)", g.Width, g.Height, w.Width, w.Height)
		}

	case *TouchEventData:
		g, ok := got.(*TouchEventData)
		if !ok {
			t.Errorf("Payload type = %T, want *TouchEventData", got)
			return
		}
		if len(g.Touches) != len(w.Touches) {
			t.Errorf("Touches count = %d, want %d", len(g.Touches), len(w.Touches))
			return
		}
		for i, wt := range w.Touches {
			gt := g.Touches[i]
			if gt.ID != wt.ID || gt.ClientX != wt.ClientX || gt.ClientY != wt.ClientY {
				t.Errorf("Touch[%d] = %+v, want %+v", i, gt, wt)
			}
		}

	case *HookEventData:
		g, ok := got.(*HookEventData)
		if !ok {
			t.Errorf("Payload type = %T, want *HookEventData", got)
			return
		}
		if g.Name != w.Name {
			t.Errorf("Name = %q, want %q", g.Name, w.Name)
		}
		// Compare data keys and values
		if len(g.Data) != len(w.Data) {
			t.Errorf("Data count = %d, want %d", len(g.Data), len(w.Data))
		}

	case *NavigateEventData:
		g, ok := got.(*NavigateEventData)
		if !ok {
			t.Errorf("Payload type = %T, want *NavigateEventData", got)
			return
		}
		if g.Path != w.Path {
			t.Errorf("Path = %q, want %q", g.Path, w.Path)
		}
		if g.Replace != w.Replace {
			t.Errorf("Replace = %v, want %v", g.Replace, w.Replace)
		}

	case *CustomEventData:
		g, ok := got.(*CustomEventData)
		if !ok {
			t.Errorf("Payload type = %T, want *CustomEventData", got)
			return
		}
		if g.Name != w.Name {
			t.Errorf("Name = %q, want %q", g.Name, w.Name)
		}
		if string(g.Data) != string(w.Data) {
			t.Errorf("Data = %v, want %v", g.Data, w.Data)
		}
	}
}

func TestEventTypeString(t *testing.T) {
	tests := []struct {
		et   EventType
		want string
	}{
		{EventClick, "Click"},
		{EventDblClick, "DblClick"},
		{EventMouseDown, "MouseDown"},
		{EventMouseUp, "MouseUp"},
		{EventMouseMove, "MouseMove"},
		{EventMouseEnter, "MouseEnter"},
		{EventMouseLeave, "MouseLeave"},
		{EventInput, "Input"},
		{EventChange, "Change"},
		{EventSubmit, "Submit"},
		{EventFocus, "Focus"},
		{EventBlur, "Blur"},
		{EventKeyDown, "KeyDown"},
		{EventKeyUp, "KeyUp"},
		{EventKeyPress, "KeyPress"},
		{EventScroll, "Scroll"},
		{EventResize, "Resize"},
		{EventTouchStart, "TouchStart"},
		{EventTouchMove, "TouchMove"},
		{EventTouchEnd, "TouchEnd"},
		{EventDragStart, "DragStart"},
		{EventDragEnd, "DragEnd"},
		{EventDrop, "Drop"},
		{EventHook, "Hook"},
		{EventNavigate, "Navigate"},
		{EventCustom, "Custom"},
		{EventType(0x99), "Unknown"},
	}

	for _, tc := range tests {
		if got := tc.et.String(); got != tc.want {
			t.Errorf("EventType(%d).String() = %q, want %q", tc.et, got, tc.want)
		}
	}
}

func TestModifiersHas(t *testing.T) {
	mods := ModCtrl | ModShift

	if !mods.Has(ModCtrl) {
		t.Error("Has(ModCtrl) = false, want true")
	}
	if !mods.Has(ModShift) {
		t.Error("Has(ModShift) = false, want true")
	}
	if mods.Has(ModAlt) {
		t.Error("Has(ModAlt) = true, want false")
	}
	if mods.Has(ModMeta) {
		t.Error("Has(ModMeta) = true, want false")
	}
}

func TestEventEncodingSize(t *testing.T) {
	// Verify that click events are compact (target: <10 bytes)
	click := &Event{
		Seq:  1,
		Type: EventClick,
		HID:  "h1",
	}
	encoded := EncodeEvent(click)
	if len(encoded) > 10 {
		t.Errorf("Click event size = %d bytes, want <= 10", len(encoded))
	}

	// Input event with short value should be reasonable
	input := &Event{
		Seq:     1,
		Type:    EventInput,
		HID:     "h5",
		Payload: "hello",
	}
	encoded = EncodeEvent(input)
	if len(encoded) > 15 {
		t.Errorf("Short input event size = %d bytes, want <= 15", len(encoded))
	}
}

func TestHookEventWithComplexData(t *testing.T) {
	event := &Event{
		Seq:  1,
		Type: EventHook,
		HID:  "h1",
		Payload: &HookEventData{
			Name: "complex-hook",
			Data: map[string]any{
				"string": "hello",
				"int":    int64(42),
				"float":  3.14,
				"bool":   true,
				"null":   nil,
				"array":  []any{int64(1), int64(2), int64(3)},
				"nested": map[string]any{
					"key": "value",
				},
			},
		},
	}

	encoded := EncodeEvent(event)
	decoded, err := DecodeEvent(encoded)
	if err != nil {
		t.Fatalf("DecodeEvent() error = %v", err)
	}

	hookData := decoded.Payload.(*HookEventData)
	if hookData.Name != "complex-hook" {
		t.Errorf("Name = %q, want %q", hookData.Name, "complex-hook")
	}

	// Check various types were decoded correctly
	if hookData.Data["string"] != "hello" {
		t.Errorf("string = %v, want \"hello\"", hookData.Data["string"])
	}
	if hookData.Data["int"] != int64(42) {
		t.Errorf("int = %v, want 42", hookData.Data["int"])
	}
	if hookData.Data["bool"] != true {
		t.Errorf("bool = %v, want true", hookData.Data["bool"])
	}
	if hookData.Data["null"] != nil {
		t.Errorf("null = %v, want nil", hookData.Data["null"])
	}
}

func TestEmptyPayloads(t *testing.T) {
	// Empty submit event
	event := &Event{
		Seq:     1,
		Type:    EventSubmit,
		HID:     "h1",
		Payload: &SubmitEventData{Fields: map[string]string{}},
	}

	encoded := EncodeEvent(event)
	decoded, err := DecodeEvent(encoded)
	if err != nil {
		t.Fatalf("DecodeEvent() error = %v", err)
	}

	submitData := decoded.Payload.(*SubmitEventData)
	if len(submitData.Fields) != 0 {
		t.Errorf("Fields count = %d, want 0", len(submitData.Fields))
	}

	// Empty touch event
	event = &Event{
		Seq:     2,
		Type:    EventTouchStart,
		HID:     "h2",
		Payload: &TouchEventData{Touches: []TouchPoint{}},
	}

	encoded = EncodeEvent(event)
	decoded, err = DecodeEvent(encoded)
	if err != nil {
		t.Fatalf("DecodeEvent() error = %v", err)
	}

	touchData := decoded.Payload.(*TouchEventData)
	if len(touchData.Touches) != 0 {
		t.Errorf("Touches count = %d, want 0", len(touchData.Touches))
	}
}

func BenchmarkEncodeClickEvent(b *testing.B) {
	event := &Event{
		Seq:  1,
		Type: EventClick,
		HID:  "h42",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EncodeEvent(event)
	}
}

func BenchmarkDecodeClickEvent(b *testing.B) {
	event := &Event{
		Seq:  1,
		Type: EventClick,
		HID:  "h42",
	}
	encoded := EncodeEvent(event)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeEvent(encoded)
	}
}

func BenchmarkEncodeInputEvent(b *testing.B) {
	event := &Event{
		Seq:     1,
		Type:    EventInput,
		HID:     "h5",
		Payload: "hello world",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EncodeEvent(event)
	}
}

func BenchmarkDecodeInputEvent(b *testing.B) {
	event := &Event{
		Seq:     1,
		Type:    EventInput,
		HID:     "h5",
		Payload: "hello world",
	}
	encoded := EncodeEvent(event)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeEvent(encoded)
	}
}

func BenchmarkEncodeSubmitEvent(b *testing.B) {
	event := &Event{
		Seq:  1,
		Type: EventSubmit,
		HID:  "h7",
		Payload: &SubmitEventData{
			Fields: map[string]string{
				"name":     "John Doe",
				"email":    "john@example.com",
				"password": "secret123",
				"confirm":  "secret123",
				"terms":    "on",
			},
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = EncodeEvent(event)
	}
}

func BenchmarkDecodeSubmitEvent(b *testing.B) {
	event := &Event{
		Seq:  1,
		Type: EventSubmit,
		HID:  "h7",
		Payload: &SubmitEventData{
			Fields: map[string]string{
				"name":     "John Doe",
				"email":    "john@example.com",
				"password": "secret123",
				"confirm":  "secret123",
				"terms":    "on",
			},
		},
	}
	encoded := EncodeEvent(event)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DecodeEvent(encoded)
	}
}
