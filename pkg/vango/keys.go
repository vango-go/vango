package vango

// =============================================================================
// Key Constants
// =============================================================================
//
// These constants match the standard JavaScript KeyboardEvent.key values.
// Per spec section 3.9.3, lines 1105-1122.
//
// Usage:
//
//	OnKeyDown(func(e vango.KeyboardEvent) {
//	    if e.Key == vango.KeyEnter {
//	        submit()
//	    }
//	})
//
//	OnKeyDown(vango.Hotkey(vango.KeyEscape, func() {
//	    closeModal()
//	}))

// Common key constants matching JavaScript KeyboardEvent.key values.
const (
	// Control keys
	KeyEnter     = "Enter"
	KeyEscape    = "Escape"
	KeySpace     = " "
	KeyTab       = "Tab"
	KeyBackspace = "Backspace"
	KeyDelete    = "Delete"

	// Arrow keys
	KeyArrowUp    = "ArrowUp"
	KeyArrowDown  = "ArrowDown"
	KeyArrowLeft  = "ArrowLeft"
	KeyArrowRight = "ArrowRight"

	// Navigation keys
	KeyHome     = "Home"
	KeyEnd      = "End"
	KeyPageUp   = "PageUp"
	KeyPageDown = "PageDown"

	// Function keys
	KeyF1  = "F1"
	KeyF2  = "F2"
	KeyF3  = "F3"
	KeyF4  = "F4"
	KeyF5  = "F5"
	KeyF6  = "F6"
	KeyF7  = "F7"
	KeyF8  = "F8"
	KeyF9  = "F9"
	KeyF10 = "F10"
	KeyF11 = "F11"
	KeyF12 = "F12"

	// Modifier keys (for reference, usually checked via event flags)
	KeyControl = "Control"
	KeyShift   = "Shift"
	KeyAlt     = "Alt"
	KeyMeta    = "Meta"

	// Other common keys
	KeyInsert      = "Insert"
	KeyPrintScreen = "PrintScreen"
	KeyScrollLock  = "ScrollLock"
	KeyPause       = "Pause"
	KeyCapsLock    = "CapsLock"
	KeyNumLock     = "NumLock"
	KeyContextMenu = "ContextMenu"
)
