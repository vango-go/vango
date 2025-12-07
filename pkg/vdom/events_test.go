package vdom

import "testing"

func TestEventHandlers(t *testing.T) {
	handler := func() {}

	tests := []struct {
		name     string
		handler  EventHandler
		expected string
	}{
		// Mouse events
		{"OnClick", OnClick(handler), "onclick"},
		{"OnDblClick", OnDblClick(handler), "ondblclick"},
		{"OnMouseDown", OnMouseDown(handler), "onmousedown"},
		{"OnMouseUp", OnMouseUp(handler), "onmouseup"},
		{"OnMouseMove", OnMouseMove(handler), "onmousemove"},
		{"OnMouseEnter", OnMouseEnter(handler), "onmouseenter"},
		{"OnMouseLeave", OnMouseLeave(handler), "onmouseleave"},
		{"OnMouseOver", OnMouseOver(handler), "onmouseover"},
		{"OnMouseOut", OnMouseOut(handler), "onmouseout"},
		{"OnContextMenu", OnContextMenu(handler), "oncontextmenu"},
		{"OnWheel", OnWheel(handler), "onwheel"},

		// Keyboard events
		{"OnKeyDown", OnKeyDown(handler), "onkeydown"},
		{"OnKeyUp", OnKeyUp(handler), "onkeyup"},
		{"OnKeyPress", OnKeyPress(handler), "onkeypress"},

		// Form events
		{"OnInput", OnInput(handler), "oninput"},
		{"OnChange", OnChange(handler), "onchange"},
		{"OnSubmit", OnSubmit(handler), "onsubmit"},
		{"OnFocus", OnFocus(handler), "onfocus"},
		{"OnBlur", OnBlur(handler), "onblur"},
		{"OnFocusIn", OnFocusIn(handler), "onfocusin"},
		{"OnFocusOut", OnFocusOut(handler), "onfocusout"},
		{"OnSelect", OnSelect(handler), "onselect"},
		{"OnInvalid", OnInvalid(handler), "oninvalid"},
		{"OnReset", OnReset(handler), "onreset"},

		// Drag events
		{"OnDragStart", OnDragStart(handler), "ondragstart"},
		{"OnDrag", OnDrag(handler), "ondrag"},
		{"OnDragEnd", OnDragEnd(handler), "ondragend"},
		{"OnDragEnter", OnDragEnter(handler), "ondragenter"},
		{"OnDragOver", OnDragOver(handler), "ondragover"},
		{"OnDragLeave", OnDragLeave(handler), "ondragleave"},
		{"OnDrop", OnDrop(handler), "ondrop"},

		// Touch events
		{"OnTouchStart", OnTouchStart(handler), "ontouchstart"},
		{"OnTouchMove", OnTouchMove(handler), "ontouchmove"},
		{"OnTouchEnd", OnTouchEnd(handler), "ontouchend"},
		{"OnTouchCancel", OnTouchCancel(handler), "ontouchcancel"},

		// Pointer events
		{"OnPointerDown", OnPointerDown(handler), "onpointerdown"},
		{"OnPointerUp", OnPointerUp(handler), "onpointerup"},
		{"OnPointerMove", OnPointerMove(handler), "onpointermove"},
		{"OnPointerEnter", OnPointerEnter(handler), "onpointerenter"},
		{"OnPointerLeave", OnPointerLeave(handler), "onpointerleave"},
		{"OnPointerCancel", OnPointerCancel(handler), "onpointercancel"},

		// Scroll events
		{"OnScroll", OnScroll(handler), "onscroll"},
		{"OnScrollEnd", OnScrollEnd(handler), "onscrollend"},

		// Media events
		{"OnPlay", OnPlay(handler), "onplay"},
		{"OnPause", OnPause(handler), "onpause"},
		{"OnEnded", OnEnded(handler), "onended"},
		{"OnTimeUpdate", OnTimeUpdate(handler), "ontimeupdate"},
		{"OnLoadStart", OnLoadStart(handler), "onloadstart"},
		{"OnLoadedData", OnLoadedData(handler), "onloadeddata"},
		{"OnLoadedMetadata", OnLoadedMetadata(handler), "onloadedmetadata"},
		{"OnCanPlay", OnCanPlay(handler), "oncanplay"},
		{"OnCanPlayThrough", OnCanPlayThrough(handler), "oncanplaythrough"},
		{"OnProgress", OnProgress(handler), "onprogress"},
		{"OnSeeking", OnSeeking(handler), "onseeking"},
		{"OnSeeked", OnSeeked(handler), "onseeked"},
		{"OnVolumeChange", OnVolumeChange(handler), "onvolumechange"},
		{"OnRateChange", OnRateChange(handler), "onratechange"},
		{"OnDurationChange", OnDurationChange(handler), "ondurationchange"},
		{"OnWaiting", OnWaiting(handler), "onwaiting"},
		{"OnPlaying", OnPlaying(handler), "onplaying"},
		{"OnStalled", OnStalled(handler), "onstalled"},
		{"OnSuspend", OnSuspend(handler), "onsuspend"},
		{"OnEmptied", OnEmptied(handler), "onemptied"},

		// Error events
		{"OnError", OnError(handler), "onerror"},

		// Load events
		{"OnLoad", OnLoad(handler), "onload"},
		{"OnAbort", OnAbort(handler), "onabort"},

		// Animation events
		{"OnAnimationStart", OnAnimationStart(handler), "onanimationstart"},
		{"OnAnimationEnd", OnAnimationEnd(handler), "onanimationend"},
		{"OnAnimationIteration", OnAnimationIteration(handler), "onanimationiteration"},
		{"OnAnimationCancel", OnAnimationCancel(handler), "onanimationcancel"},

		// Transition events
		{"OnTransitionStart", OnTransitionStart(handler), "ontransitionstart"},
		{"OnTransitionEnd", OnTransitionEnd(handler), "ontransitionend"},
		{"OnTransitionRun", OnTransitionRun(handler), "ontransitionrun"},
		{"OnTransitionCancel", OnTransitionCancel(handler), "ontransitioncancel"},

		// Clipboard events
		{"OnCopy", OnCopy(handler), "oncopy"},
		{"OnCut", OnCut(handler), "oncut"},
		{"OnPaste", OnPaste(handler), "onpaste"},

		// Details events
		{"OnToggle", OnToggle(handler), "ontoggle"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.handler.Event != tt.expected {
				t.Errorf("Event = %v, want %v", tt.handler.Event, tt.expected)
			}
			if tt.handler.Handler == nil {
				t.Error("Handler is nil")
			}
		})
	}
}

func TestEventHandlerInElement(t *testing.T) {
	clicked := false
	handler := func() { clicked = true }

	node := Button(OnClick(handler), Text("Click me"))

	if node.Props["onclick"] == nil {
		t.Fatal("onclick handler not set in Props")
	}

	// Get and call the handler
	if fn, ok := node.Props["onclick"].(func()); ok {
		fn()
		if !clicked {
			t.Error("Handler was not executed")
		}
	} else {
		t.Error("onclick handler is not a func()")
	}
}

func TestMultipleEventHandlers(t *testing.T) {
	node := Button(
		OnClick(func() {}),
		OnMouseEnter(func() {}),
		OnMouseLeave(func() {}),
	)

	if node.Props["onclick"] == nil {
		t.Error("onclick not set")
	}
	if node.Props["onmouseenter"] == nil {
		t.Error("onmouseenter not set")
	}
	if node.Props["onmouseleave"] == nil {
		t.Error("onmouseleave not set")
	}
}

func TestEventHandlerWithValue(t *testing.T) {
	var received string
	handler := func(value string) { received = value }

	node := Input(OnInput(handler), Type("text"))

	if node.Props["oninput"] == nil {
		t.Fatal("oninput handler not set")
	}

	// Call with value
	if fn, ok := node.Props["oninput"].(func(string)); ok {
		fn("test value")
		if received != "test value" {
			t.Errorf("received = %v, want 'test value'", received)
		}
	}
}
