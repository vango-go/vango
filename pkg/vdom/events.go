package vdom

// event creates an EventHandler with the given name and handler.
// The name is prefixed with "on" (e.g., "click" becomes "onclick").
func event(name string, handler any) EventHandler {
	return EventHandler{Event: "on" + name, Handler: handler}
}

// Mouse events

// OnClick handles click events.
func OnClick(handler any) EventHandler { return event("click", handler) }

// OnDblClick handles double-click events.
func OnDblClick(handler any) EventHandler { return event("dblclick", handler) }

// OnMouseDown handles mousedown events.
func OnMouseDown(handler any) EventHandler { return event("mousedown", handler) }

// OnMouseUp handles mouseup events.
func OnMouseUp(handler any) EventHandler { return event("mouseup", handler) }

// OnMouseMove handles mousemove events.
func OnMouseMove(handler any) EventHandler { return event("mousemove", handler) }

// OnMouseEnter handles mouseenter events.
func OnMouseEnter(handler any) EventHandler { return event("mouseenter", handler) }

// OnMouseLeave handles mouseleave events.
func OnMouseLeave(handler any) EventHandler { return event("mouseleave", handler) }

// OnMouseOver handles mouseover events.
func OnMouseOver(handler any) EventHandler { return event("mouseover", handler) }

// OnMouseOut handles mouseout events.
func OnMouseOut(handler any) EventHandler { return event("mouseout", handler) }

// OnContextMenu handles contextmenu (right-click) events.
func OnContextMenu(handler any) EventHandler { return event("contextmenu", handler) }

// OnWheel handles wheel (scroll wheel) events.
func OnWheel(handler any) EventHandler { return event("wheel", handler) }

// Keyboard events

// OnKeyDown handles keydown events.
func OnKeyDown(handler any) EventHandler { return event("keydown", handler) }

// OnKeyUp handles keyup events.
func OnKeyUp(handler any) EventHandler { return event("keyup", handler) }

// OnKeyPress handles keypress events (deprecated, but still supported).
func OnKeyPress(handler any) EventHandler { return event("keypress", handler) }

// Form events

// OnInput handles input events (fired when value changes).
func OnInput(handler any) EventHandler { return event("input", handler) }

// OnChange handles change events (fired when value is committed).
func OnChange(handler any) EventHandler { return event("change", handler) }

// OnSubmit handles form submit events.
func OnSubmit(handler any) EventHandler { return event("submit", handler) }

// OnFocus handles focus events.
func OnFocus(handler any) EventHandler { return event("focus", handler) }

// OnBlur handles blur events.
func OnBlur(handler any) EventHandler { return event("blur", handler) }

// OnFocusIn handles focusin events (bubbles, unlike focus).
func OnFocusIn(handler any) EventHandler { return event("focusin", handler) }

// OnFocusOut handles focusout events (bubbles, unlike blur).
func OnFocusOut(handler any) EventHandler { return event("focusout", handler) }

// OnSelect handles select events (text selection).
func OnSelect(handler any) EventHandler { return event("select", handler) }

// OnInvalid handles invalid events (form validation).
func OnInvalid(handler any) EventHandler { return event("invalid", handler) }

// OnReset handles form reset events.
func OnReset(handler any) EventHandler { return event("reset", handler) }

// Drag events

// OnDragStart handles dragstart events.
func OnDragStart(handler any) EventHandler { return event("dragstart", handler) }

// OnDrag handles drag events.
func OnDrag(handler any) EventHandler { return event("drag", handler) }

// OnDragEnd handles dragend events.
func OnDragEnd(handler any) EventHandler { return event("dragend", handler) }

// OnDragEnter handles dragenter events.
func OnDragEnter(handler any) EventHandler { return event("dragenter", handler) }

// OnDragOver handles dragover events.
func OnDragOver(handler any) EventHandler { return event("dragover", handler) }

// OnDragLeave handles dragleave events.
func OnDragLeave(handler any) EventHandler { return event("dragleave", handler) }

// OnDrop handles drop events.
func OnDrop(handler any) EventHandler { return event("drop", handler) }

// Touch events

// OnTouchStart handles touchstart events.
func OnTouchStart(handler any) EventHandler { return event("touchstart", handler) }

// OnTouchMove handles touchmove events.
func OnTouchMove(handler any) EventHandler { return event("touchmove", handler) }

// OnTouchEnd handles touchend events.
func OnTouchEnd(handler any) EventHandler { return event("touchend", handler) }

// OnTouchCancel handles touchcancel events.
func OnTouchCancel(handler any) EventHandler { return event("touchcancel", handler) }

// Pointer events

// OnPointerDown handles pointerdown events.
func OnPointerDown(handler any) EventHandler { return event("pointerdown", handler) }

// OnPointerUp handles pointerup events.
func OnPointerUp(handler any) EventHandler { return event("pointerup", handler) }

// OnPointerMove handles pointermove events.
func OnPointerMove(handler any) EventHandler { return event("pointermove", handler) }

// OnPointerEnter handles pointerenter events.
func OnPointerEnter(handler any) EventHandler { return event("pointerenter", handler) }

// OnPointerLeave handles pointerleave events.
func OnPointerLeave(handler any) EventHandler { return event("pointerleave", handler) }

// OnPointerCancel handles pointercancel events.
func OnPointerCancel(handler any) EventHandler { return event("pointercancel", handler) }

// Scroll events

// OnScroll handles scroll events.
func OnScroll(handler any) EventHandler { return event("scroll", handler) }

// OnScrollEnd handles scrollend events.
func OnScrollEnd(handler any) EventHandler { return event("scrollend", handler) }

// Media events

// OnPlay handles play events (media starts playing).
func OnPlay(handler any) EventHandler { return event("play", handler) }

// OnPause handles pause events (media is paused).
func OnPause(handler any) EventHandler { return event("pause", handler) }

// OnEnded handles ended events (media playback finished).
func OnEnded(handler any) EventHandler { return event("ended", handler) }

// OnTimeUpdate handles timeupdate events (playback position changed).
func OnTimeUpdate(handler any) EventHandler { return event("timeupdate", handler) }

// OnLoadStart handles loadstart events (loading begins).
func OnLoadStart(handler any) EventHandler { return event("loadstart", handler) }

// OnLoadedData handles loadeddata events (first frame loaded).
func OnLoadedData(handler any) EventHandler { return event("loadeddata", handler) }

// OnLoadedMetadata handles loadedmetadata events (metadata loaded).
func OnLoadedMetadata(handler any) EventHandler { return event("loadedmetadata", handler) }

// OnCanPlay handles canplay events (can begin playback).
func OnCanPlay(handler any) EventHandler { return event("canplay", handler) }

// OnCanPlayThrough handles canplaythrough events (can play without buffering).
func OnCanPlayThrough(handler any) EventHandler { return event("canplaythrough", handler) }

// OnProgress handles progress events (loading progress).
func OnProgress(handler any) EventHandler { return event("progress", handler) }

// OnSeeking handles seeking events (seek operation started).
func OnSeeking(handler any) EventHandler { return event("seeking", handler) }

// OnSeeked handles seeked events (seek operation completed).
func OnSeeked(handler any) EventHandler { return event("seeked", handler) }

// OnVolumeChange handles volumechange events.
func OnVolumeChange(handler any) EventHandler { return event("volumechange", handler) }

// OnRateChange handles ratechange events (playback rate changed).
func OnRateChange(handler any) EventHandler { return event("ratechange", handler) }

// OnDurationChange handles durationchange events.
func OnDurationChange(handler any) EventHandler { return event("durationchange", handler) }

// OnWaiting handles waiting events (playback stopped, waiting for data).
func OnWaiting(handler any) EventHandler { return event("waiting", handler) }

// OnPlaying handles playing events (playback resumed after pausing/buffering).
func OnPlaying(handler any) EventHandler { return event("playing", handler) }

// OnStalled handles stalled events (fetching media data stalled).
func OnStalled(handler any) EventHandler { return event("stalled", handler) }

// OnSuspend handles suspend events (loading suspended).
func OnSuspend(handler any) EventHandler { return event("suspend", handler) }

// OnEmptied handles emptied events (media emptied).
func OnEmptied(handler any) EventHandler { return event("emptied", handler) }

// Error events

// OnError handles error events.
func OnError(handler any) EventHandler { return event("error", handler) }

// Load events

// OnLoad handles load events.
func OnLoad(handler any) EventHandler { return event("load", handler) }

// OnAbort handles abort events.
func OnAbort(handler any) EventHandler { return event("abort", handler) }

// Animation events

// OnAnimationStart handles animationstart events.
func OnAnimationStart(handler any) EventHandler { return event("animationstart", handler) }

// OnAnimationEnd handles animationend events.
func OnAnimationEnd(handler any) EventHandler { return event("animationend", handler) }

// OnAnimationIteration handles animationiteration events.
func OnAnimationIteration(handler any) EventHandler { return event("animationiteration", handler) }

// OnAnimationCancel handles animationcancel events.
func OnAnimationCancel(handler any) EventHandler { return event("animationcancel", handler) }

// Transition events

// OnTransitionStart handles transitionstart events.
func OnTransitionStart(handler any) EventHandler { return event("transitionstart", handler) }

// OnTransitionEnd handles transitionend events.
func OnTransitionEnd(handler any) EventHandler { return event("transitionend", handler) }

// OnTransitionRun handles transitionrun events.
func OnTransitionRun(handler any) EventHandler { return event("transitionrun", handler) }

// OnTransitionCancel handles transitioncancel events.
func OnTransitionCancel(handler any) EventHandler { return event("transitioncancel", handler) }

// Clipboard events

// OnCopy handles copy events.
func OnCopy(handler any) EventHandler { return event("copy", handler) }

// OnCut handles cut events.
func OnCut(handler any) EventHandler { return event("cut", handler) }

// OnPaste handles paste events.
func OnPaste(handler any) EventHandler { return event("paste", handler) }

// Details events

// OnToggle handles toggle events (for details element).
func OnToggle(handler any) EventHandler { return event("toggle", handler) }
