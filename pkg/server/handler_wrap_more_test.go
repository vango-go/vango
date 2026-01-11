package server

import (
	"log/slog"
	"testing"
	"time"

	"github.com/vango-go/vango/pkg/features/hooks"
	"github.com/vango-go/vango/pkg/protocol"
	"github.com/vango-go/vango/pkg/vango"
)

func TestWrapHandler_EventPayloadConversions(t *testing.T) {
	t.Run("MouseEvent", func(t *testing.T) {
		var got MouseEvent
		called := false

		h := wrapHandler(func(me MouseEvent) {
			called = true
			got = me
		})
		h(&Event{Payload: &protocol.MouseEventData{
			ClientX:   1,
			ClientY:   2,
			Button:    3,
			Modifiers: protocol.ModCtrl | protocol.ModAlt,
		}})

		if !called {
			t.Fatal("handler not called")
		}
		if got.ClientX != 1 || got.ClientY != 2 || got.Button != 3 {
			t.Fatalf("mouse=%+v, want x=1 y=2 button=3", got)
		}
		if !got.CtrlKey || got.ShiftKey || !got.AltKey || got.MetaKey {
			t.Fatalf("modifiers=%+v, want ctrl+alt only", got)
		}
	})

	t.Run("KeyboardEvent", func(t *testing.T) {
		var got KeyboardEvent
		called := false

		h := wrapHandler(func(ke KeyboardEvent) {
			called = true
			got = ke
		})
		h(&Event{Payload: &protocol.KeyboardEventData{
			Key:       "Enter",
			Modifiers: protocol.ModShift | protocol.ModMeta,
		}})

		if !called {
			t.Fatal("handler not called")
		}
		if got.Key != "Enter" || got.CtrlKey || !got.ShiftKey || got.AltKey || !got.MetaKey {
			t.Fatalf("keyboard=%+v, want key=Enter shift+meta only", got)
		}
	})

	t.Run("FormData", func(t *testing.T) {
		var got FormData
		called := false

		h := wrapHandler(func(fd FormData) {
			called = true
			got = fd
		})
		h(&Event{Payload: &protocol.SubmitEventData{Fields: map[string]string{"name": "n"}}})

		if !called {
			t.Fatal("handler not called")
		}
		if got.Get("name") != "n" || !got.Has("name") {
			t.Fatalf("FormData=%v, want name=n", got.All())
		}
	})

	t.Run("HookEvent_Internal", func(t *testing.T) {
		var got HookEvent
		called := false

		h := wrapHandler(func(he HookEvent) {
			called = true
			got = he
		})
		h(&Event{Payload: &protocol.HookEventData{Name: "evt", Data: map[string]any{"k": "v"}}})

		if !called {
			t.Fatal("handler not called")
		}
		if got.Name != "evt" || got.GetString("k") != "v" {
			t.Fatalf("HookEvent=%+v, want name=evt k=v", got)
		}
	})

	t.Run("HookEvent_PublicHooksRevertDispatches", func(t *testing.T) {
		clientConn, serverConn := newWebSocketPair(t)

		sess := newSession(serverConn, "", DefaultSessionConfig(), slog.Default())
		e := &Event{
			HID:     "h1",
			Session: sess,
			Payload: &protocol.HookEventData{Name: "x", Data: map[string]any{}},
		}

		h := wrapHandler(func(he hooks.HookEvent) { he.Revert() })
		h(e)

		_ = clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, msg, err := clientConn.ReadMessage()
		if err != nil {
			t.Fatalf("ReadMessage failed: %v", err)
		}

		frame, err := protocol.DecodeFrame(msg)
		if err != nil {
			t.Fatalf("DecodeFrame failed: %v", err)
		}
		pf, err := protocol.DecodePatches(frame.Payload)
		if err != nil {
			t.Fatalf("DecodePatches failed: %v", err)
		}
		if len(pf.Patches) != 1 {
			t.Fatalf("patches=%d, want 1", len(pf.Patches))
		}
		if pf.Patches[0].Op != protocol.PatchDispatch || pf.Patches[0].Key != "vango:hook-revert" {
			t.Fatalf("patch=%+v, want Dispatch(vango:hook-revert)", pf.Patches[0])
		}
	})

	t.Run("HookEvent_VangoRevertDispatches", func(t *testing.T) {
		clientConn, serverConn := newWebSocketPair(t)

		sess := newSession(serverConn, "", DefaultSessionConfig(), slog.Default())
		e := &Event{
			HID:     "h1",
			Session: sess,
			Payload: &protocol.HookEventData{Name: "x", Data: map[string]any{}},
		}

		h := wrapHandler(func(he vango.HookEvent) { he.Revert() })
		h(e)

		_ = clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, msg, err := clientConn.ReadMessage()
		if err != nil {
			t.Fatalf("ReadMessage failed: %v", err)
		}
		frame, err := protocol.DecodeFrame(msg)
		if err != nil {
			t.Fatalf("DecodeFrame failed: %v", err)
		}
		pf, err := protocol.DecodePatches(frame.Payload)
		if err != nil {
			t.Fatalf("DecodePatches failed: %v", err)
		}
		if len(pf.Patches) != 1 || pf.Patches[0].Op != protocol.PatchDispatch || pf.Patches[0].Key != "revert" {
			t.Fatalf("patch=%+v, want Dispatch(revert)", pf.Patches)
		}
	})

	t.Run("NavigateEvent", func(t *testing.T) {
		var got NavigateEvent
		called := false

		h := wrapHandler(func(ne NavigateEvent) {
			called = true
			got = ne
		})
		h(&Event{Payload: &protocol.NavigateEventData{Path: "/p", Replace: true}})
		if !called || got.Path != "/p" || !got.Replace {
			t.Fatalf("NavigateEvent called=%v got=%+v, want /p replace", called, got)
		}
	})

	t.Run("VangoEvents", func(t *testing.T) {
		var wheel vango.WheelEvent
		var input vango.InputEvent
		var anim vango.AnimationEvent
		var trans vango.TransitionEvent

		wrapHandler(func(ev vango.WheelEvent) { wheel = ev })(&Event{Payload: &protocol.WheelEventData{
			DeltaX:    1,
			DeltaY:    2,
			DeltaMode: 1,
			ClientX:   10,
			ClientY:   20,
			Modifiers: protocol.ModAlt,
		}})
		if wheel.DeltaX != 1 || wheel.DeltaY != 2 || wheel.DeltaMode != 1 || !wheel.AltKey {
			t.Fatalf("wheel=%+v, want mapped fields", wheel)
		}

		wrapHandler(func(ev vango.InputEvent) { input = ev })(&Event{Payload: &protocol.InputEventData{
			Value:     "v",
			InputType: "insertText",
			Data:      "d",
		}})
		if input.Value != "v" || input.InputType != "insertText" || input.Data != "d" {
			t.Fatalf("input=%+v, want mapped fields", input)
		}

		wrapHandler(func(ev vango.AnimationEvent) { anim = ev })(&Event{Payload: &protocol.AnimationEventData{
			AnimationName: "fade",
			ElapsedTime:   1.5,
			PseudoElement: "::before",
		}})
		if anim.AnimationName != "fade" || anim.ElapsedTime != 1.5 {
			t.Fatalf("anim=%+v, want mapped fields", anim)
		}

		wrapHandler(func(ev vango.TransitionEvent) { trans = ev })(&Event{Payload: &protocol.TransitionEventData{
			PropertyName:  "opacity",
			ElapsedTime:   2.5,
			PseudoElement: "::after",
		}})
		if trans.PropertyName != "opacity" || trans.ElapsedTime != 2.5 {
			t.Fatalf("trans=%+v, want mapped fields", trans)
		}
	})

	t.Run("MoreVangoEvents", func(t *testing.T) {
		var scroll vango.ScrollEvent
		var resize vango.ResizeEvent
		var nav vango.NavigateEvent
		var drag vango.DragEvent
		var touch vango.TouchEvent
		var form vango.FormData

		wrapHandler(func(ev vango.ScrollEvent) { scroll = ev })(&Event{Payload: &protocol.ScrollEventData{
			ScrollTop:  10,
			ScrollLeft: 20,
		}})
		if scroll.ScrollTop != 10 || scroll.ScrollLeft != 20 {
			t.Fatalf("scroll=%+v, want mapped fields", scroll)
		}

		wrapHandler(func(ev vango.ResizeEvent) { resize = ev })(&Event{Payload: &protocol.ResizeEventData{
			Width:  800,
			Height: 600,
		}})
		if resize.Width != 800 || resize.Height != 600 {
			t.Fatalf("resize=%+v, want mapped fields", resize)
		}

		wrapHandler(func(ev vango.NavigateEvent) { nav = ev })(&Event{Payload: &protocol.NavigateEventData{
			Path:    "/x",
			Replace: true,
		}})
		if nav.Path != "/x" || !nav.Replace {
			t.Fatalf("nav=%+v, want /x replace", nav)
		}

		wrapHandler(func(ev vango.DragEvent) { drag = ev })(&Event{Payload: &protocol.MouseEventData{
			ClientX:   1,
			ClientY:   2,
			Modifiers: protocol.ModShift,
		}})
		if drag.ClientX != 1 || drag.ClientY != 2 || !drag.ShiftKey {
			t.Fatalf("drag=%+v, want mapped fields", drag)
		}

		wrapHandler(func(ev vango.TouchEvent) { touch = ev })(&Event{Payload: &protocol.TouchEventData{
			Touches: []protocol.TouchPoint{
				{ID: 1, ClientX: 10, ClientY: 20, PageX: 30, PageY: 40},
			},
		}})
		if len(touch.Touches) != 1 || touch.Touches[0].Identifier != 1 || touch.Touches[0].PageX != 30 {
			t.Fatalf("touch=%+v, want mapped touch point", touch)
		}

		wrapHandler(func(ev vango.FormData) { form = ev })(&Event{Payload: &protocol.SubmitEventData{
			Fields: map[string]string{"name": "n"},
		}})
		if !form.Has("name") || form.Get("name") != "n" {
			t.Fatalf("form=%v, want name=n", form.All())
		}
	})
}

func TestWrapHandler_ModifiedHandler_KeyFiltering(t *testing.T) {
	calls := 0
	inner := func(v vango.KeyboardEvent) { calls++ }

	h := wrapHandler(vango.ModifiedHandler{
		Handler:      inner,
		KeyFilter:    "Enter",
		KeyModifiers: vango.Ctrl,
	})

	h(&Event{Payload: &protocol.KeyboardEventData{Key: "Escape", Modifiers: protocol.ModCtrl}})
	h(&Event{Payload: &protocol.KeyboardEventData{Key: "Enter", Modifiers: protocol.ModShift}})
	h(&Event{Payload: &protocol.KeyboardEventData{Key: "Enter", Modifiers: protocol.ModCtrl}})

	if calls != 1 {
		t.Fatalf("calls=%d, want 1", calls)
	}
}
