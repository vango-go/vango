package protocol

import (
	"errors"
	"io"
)

// EventType identifies the type of client event.
type EventType uint8

// Event type constants.
const (
	// Mouse events (0x01-0x08)
	EventClick      EventType = 0x01
	EventDblClick   EventType = 0x02
	EventMouseDown  EventType = 0x03
	EventMouseUp    EventType = 0x04
	EventMouseMove  EventType = 0x05
	EventMouseEnter EventType = 0x06
	EventMouseLeave EventType = 0x07
	EventWheel      EventType = 0x08 // Mouse wheel event

	// Form events (0x10-0x14)
	EventInput  EventType = 0x10
	EventChange EventType = 0x11
	EventSubmit EventType = 0x12
	EventFocus  EventType = 0x13
	EventBlur   EventType = 0x14

	// Keyboard events (0x20-0x22)
	EventKeyDown  EventType = 0x20
	EventKeyUp    EventType = 0x21
	EventKeyPress EventType = 0x22

	// Scroll/Resize events (0x30-0x31)
	EventScroll EventType = 0x30
	EventResize EventType = 0x31

	// Touch events (0x40-0x42)
	EventTouchStart EventType = 0x40
	EventTouchMove  EventType = 0x41
	EventTouchEnd   EventType = 0x42

	// Drag events (0x50-0x52)
	EventDragStart EventType = 0x50
	EventDragEnd   EventType = 0x51
	EventDrop      EventType = 0x52

	// Animation events (0x53-0x56)
	EventAnimationStart     EventType = 0x53
	EventAnimationEnd       EventType = 0x54
	EventAnimationIteration EventType = 0x55
	EventAnimationCancel    EventType = 0x56

	// Transition events (0x57-0x5A)
	EventTransitionStart  EventType = 0x57
	EventTransitionEnd    EventType = 0x58
	EventTransitionRun    EventType = 0x59
	EventTransitionCancel EventType = 0x5A

	// Special events (0x60+)
	EventHook     EventType = 0x60 // Client hook event
	EventNavigate EventType = 0x70 // Navigation request
	EventCustom   EventType = 0xFF // Custom event
)

// String returns the string representation of the event type.
func (et EventType) String() string {
	switch et {
	case EventClick:
		return "Click"
	case EventDblClick:
		return "DblClick"
	case EventMouseDown:
		return "MouseDown"
	case EventMouseUp:
		return "MouseUp"
	case EventMouseMove:
		return "MouseMove"
	case EventMouseEnter:
		return "MouseEnter"
	case EventMouseLeave:
		return "MouseLeave"
	case EventInput:
		return "Input"
	case EventChange:
		return "Change"
	case EventSubmit:
		return "Submit"
	case EventFocus:
		return "Focus"
	case EventBlur:
		return "Blur"
	case EventKeyDown:
		return "KeyDown"
	case EventKeyUp:
		return "KeyUp"
	case EventKeyPress:
		return "KeyPress"
	case EventScroll:
		return "Scroll"
	case EventResize:
		return "Resize"
	case EventTouchStart:
		return "TouchStart"
	case EventTouchMove:
		return "TouchMove"
	case EventTouchEnd:
		return "TouchEnd"
	case EventDragStart:
		return "DragStart"
	case EventDragEnd:
		return "DragEnd"
	case EventDrop:
		return "Drop"
	case EventWheel:
		return "Wheel"
	case EventAnimationStart:
		return "AnimationStart"
	case EventAnimationEnd:
		return "AnimationEnd"
	case EventAnimationIteration:
		return "AnimationIteration"
	case EventAnimationCancel:
		return "AnimationCancel"
	case EventTransitionStart:
		return "TransitionStart"
	case EventTransitionEnd:
		return "TransitionEnd"
	case EventTransitionRun:
		return "TransitionRun"
	case EventTransitionCancel:
		return "TransitionCancel"
	case EventHook:
		return "Hook"
	case EventNavigate:
		return "Navigate"
	case EventCustom:
		return "Custom"
	default:
		return "Unknown"
	}
}

// Modifiers represents keyboard/mouse modifier keys.
type Modifiers uint8

const (
	ModCtrl  Modifiers = 0x01
	ModShift Modifiers = 0x02
	ModAlt   Modifiers = 0x04
	ModMeta  Modifiers = 0x08
)

// Has returns true if the specified modifier is set.
func (m Modifiers) Has(mod Modifiers) bool {
	return m&mod != 0
}

// Event payload types.

// KeyboardEventData contains keyboard event data.
type KeyboardEventData struct {
	Key       string
	Code      string    // Physical key code (e.g., "KeyA", "Enter")
	Modifiers Modifiers
	Repeat    bool      // True if key is held down (auto-repeat)
	Location  uint8     // 0=standard, 1=left, 2=right, 3=numpad
}

// MouseEventData contains mouse event data.
type MouseEventData struct {
	ClientX   int
	ClientY   int
	PageX     int       // Position relative to document
	PageY     int
	OffsetX   int       // Position relative to target element
	OffsetY   int
	Button    uint8
	Buttons   uint8     // Bitmask of currently pressed buttons
	Modifiers Modifiers
}

// WheelEventData contains mouse wheel event data.
type WheelEventData struct {
	DeltaX    float64
	DeltaY    float64
	DeltaZ    float64
	DeltaMode uint8     // 0=pixels, 1=lines, 2=pages
	ClientX   int
	ClientY   int
	Modifiers Modifiers
}

// InputEventData contains input event data with full details.
type InputEventData struct {
	Value     string
	InputType string    // e.g., "insertText", "deleteContentBackward"
	Data      string    // Inserted text (if any)
}

// ScrollEventData contains scroll event data.
type ScrollEventData struct {
	ScrollTop  int
	ScrollLeft int
}

// SubmitEventData contains form submission data.
type SubmitEventData struct {
	Fields map[string]string
}

// ResizeEventData contains resize event data.
type ResizeEventData struct {
	Width  int
	Height int
}

// TouchPoint represents a single touch point.
type TouchPoint struct {
	ID      int
	ClientX int
	ClientY int
	PageX   int // Position relative to document
	PageY   int
}

// TouchEventData contains touch event data.
type TouchEventData struct {
	Touches        []TouchPoint // All current touches
	TargetTouches  []TouchPoint // Touches on this element
	ChangedTouches []TouchPoint // Touches that changed in this event
}

// AnimationEventData contains CSS animation event data.
type AnimationEventData struct {
	AnimationName string
	ElapsedTime   float64
	PseudoElement string
}

// TransitionEventData contains CSS transition event data.
type TransitionEventData struct {
	PropertyName  string
	ElapsedTime   float64
	PseudoElement string
}

// HookValueType identifies the type of a hook data value.
type HookValueType uint8

const (
	HookValueNull   HookValueType = 0x00
	HookValueBool   HookValueType = 0x01
	HookValueInt    HookValueType = 0x02
	HookValueFloat  HookValueType = 0x03
	HookValueString HookValueType = 0x04
	HookValueArray  HookValueType = 0x05
	HookValueObject HookValueType = 0x06
)

// HookEventData contains client hook event data.
type HookEventData struct {
	Name string
	Data map[string]any
}

// NavigateEventData contains navigation event data.
type NavigateEventData struct {
	Path    string
	Replace bool
}

// CustomEventData contains custom event data.
type CustomEventData struct {
	Name string
	Data []byte
}

// Event represents a decoded event from the client.
type Event struct {
	Seq     uint64
	Type    EventType
	HID     string
	Payload any // Type-specific payload (nil for simple events like Click)
}

// Event encoding errors.
var (
	ErrInvalidEventType = errors.New("protocol: invalid event type")
	ErrInvalidPayload   = errors.New("protocol: invalid event payload")
	ErrMaxDepthExceeded = errors.New("protocol: maximum nesting depth exceeded")
)

// EncodeEvent encodes an event to bytes.
func EncodeEvent(e *Event) []byte {
	enc := NewEncoder()
	EncodeEventTo(enc, e)
	return enc.Bytes()
}

// EncodeEventTo encodes an event using the provided encoder.
func EncodeEventTo(enc *Encoder, e *Event) {
	enc.WriteUvarint(e.Seq)
	enc.WriteByte(byte(e.Type))
	enc.WriteString(e.HID)

	switch e.Type {
	case EventClick, EventDblClick, EventFocus, EventBlur,
		EventMouseEnter, EventMouseLeave:
		// No payload

	case EventInput, EventChange:
		// String payload
		if s, ok := e.Payload.(string); ok {
			enc.WriteString(s)
		} else {
			enc.WriteString("")
		}

	case EventSubmit:
		data, ok := e.Payload.(*SubmitEventData)
		if !ok || data == nil {
			enc.WriteUvarint(0)
		} else {
			enc.WriteUvarint(uint64(len(data.Fields)))
			for k, v := range data.Fields {
				enc.WriteString(k)
				enc.WriteString(v)
			}
		}

	case EventKeyDown, EventKeyUp, EventKeyPress:
		data, ok := e.Payload.(*KeyboardEventData)
		if !ok || data == nil {
			enc.WriteString("")
			enc.WriteString("")
			enc.WriteByte(0)
			enc.WriteBool(false)
			enc.WriteByte(0)
		} else {
			enc.WriteString(data.Key)
			enc.WriteString(data.Code)
			enc.WriteByte(byte(data.Modifiers))
			enc.WriteBool(data.Repeat)
			enc.WriteByte(data.Location)
		}

	case EventMouseDown, EventMouseUp, EventMouseMove:
		data, ok := e.Payload.(*MouseEventData)
		if !ok || data == nil {
			enc.WriteSvarint(0)
			enc.WriteSvarint(0)
			enc.WriteSvarint(0)
			enc.WriteSvarint(0)
			enc.WriteSvarint(0)
			enc.WriteSvarint(0)
			enc.WriteByte(0)
			enc.WriteByte(0)
			enc.WriteByte(0)
		} else {
			enc.WriteSvarint(int64(data.ClientX))
			enc.WriteSvarint(int64(data.ClientY))
			enc.WriteSvarint(int64(data.PageX))
			enc.WriteSvarint(int64(data.PageY))
			enc.WriteSvarint(int64(data.OffsetX))
			enc.WriteSvarint(int64(data.OffsetY))
			enc.WriteByte(data.Button)
			enc.WriteByte(data.Buttons)
			enc.WriteByte(byte(data.Modifiers))
		}

	case EventWheel:
		data, ok := e.Payload.(*WheelEventData)
		if !ok || data == nil {
			enc.WriteFloat64(0)
			enc.WriteFloat64(0)
			enc.WriteFloat64(0)
			enc.WriteByte(0)
			enc.WriteSvarint(0)
			enc.WriteSvarint(0)
			enc.WriteByte(0)
		} else {
			enc.WriteFloat64(data.DeltaX)
			enc.WriteFloat64(data.DeltaY)
			enc.WriteFloat64(data.DeltaZ)
			enc.WriteByte(data.DeltaMode)
			enc.WriteSvarint(int64(data.ClientX))
			enc.WriteSvarint(int64(data.ClientY))
			enc.WriteByte(byte(data.Modifiers))
		}

	case EventScroll:
		data, ok := e.Payload.(*ScrollEventData)
		if !ok || data == nil {
			enc.WriteSvarint(0)
			enc.WriteSvarint(0)
		} else {
			enc.WriteSvarint(int64(data.ScrollTop))
			enc.WriteSvarint(int64(data.ScrollLeft))
		}

	case EventResize:
		data, ok := e.Payload.(*ResizeEventData)
		if !ok || data == nil {
			enc.WriteSvarint(0)
			enc.WriteSvarint(0)
		} else {
			enc.WriteSvarint(int64(data.Width))
			enc.WriteSvarint(int64(data.Height))
		}

	case EventTouchStart, EventTouchMove, EventTouchEnd:
		data, ok := e.Payload.(*TouchEventData)
		if !ok || data == nil {
			enc.WriteUvarint(0)
			enc.WriteUvarint(0)
			enc.WriteUvarint(0)
		} else {
			// Encode Touches
			enc.WriteUvarint(uint64(len(data.Touches)))
			for _, t := range data.Touches {
				enc.WriteSvarint(int64(t.ID))
				enc.WriteSvarint(int64(t.ClientX))
				enc.WriteSvarint(int64(t.ClientY))
				enc.WriteSvarint(int64(t.PageX))
				enc.WriteSvarint(int64(t.PageY))
			}
			// Encode TargetTouches
			enc.WriteUvarint(uint64(len(data.TargetTouches)))
			for _, t := range data.TargetTouches {
				enc.WriteSvarint(int64(t.ID))
				enc.WriteSvarint(int64(t.ClientX))
				enc.WriteSvarint(int64(t.ClientY))
				enc.WriteSvarint(int64(t.PageX))
				enc.WriteSvarint(int64(t.PageY))
			}
			// Encode ChangedTouches
			enc.WriteUvarint(uint64(len(data.ChangedTouches)))
			for _, t := range data.ChangedTouches {
				enc.WriteSvarint(int64(t.ID))
				enc.WriteSvarint(int64(t.ClientX))
				enc.WriteSvarint(int64(t.ClientY))
				enc.WriteSvarint(int64(t.PageX))
				enc.WriteSvarint(int64(t.PageY))
			}
		}

	case EventDragStart, EventDragEnd, EventDrop:
		// Similar to mouse events - include position and modifiers
		data, ok := e.Payload.(*MouseEventData)
		if !ok || data == nil {
			enc.WriteSvarint(0)
			enc.WriteSvarint(0)
			enc.WriteSvarint(0)
			enc.WriteSvarint(0)
			enc.WriteByte(0)
		} else {
			enc.WriteSvarint(int64(data.ClientX))
			enc.WriteSvarint(int64(data.ClientY))
			enc.WriteSvarint(int64(data.PageX))
			enc.WriteSvarint(int64(data.PageY))
			enc.WriteByte(byte(data.Modifiers))
		}

	case EventAnimationStart, EventAnimationEnd, EventAnimationIteration, EventAnimationCancel:
		data, ok := e.Payload.(*AnimationEventData)
		if !ok || data == nil {
			enc.WriteString("")
			enc.WriteFloat64(0)
			enc.WriteString("")
		} else {
			enc.WriteString(data.AnimationName)
			enc.WriteFloat64(data.ElapsedTime)
			enc.WriteString(data.PseudoElement)
		}

	case EventTransitionStart, EventTransitionEnd, EventTransitionRun, EventTransitionCancel:
		data, ok := e.Payload.(*TransitionEventData)
		if !ok || data == nil {
			enc.WriteString("")
			enc.WriteFloat64(0)
			enc.WriteString("")
		} else {
			enc.WriteString(data.PropertyName)
			enc.WriteFloat64(data.ElapsedTime)
			enc.WriteString(data.PseudoElement)
		}

	case EventHook:
		data, ok := e.Payload.(*HookEventData)
		if !ok || data == nil {
			enc.WriteString("")
			enc.WriteUvarint(0)
		} else {
			enc.WriteString(data.Name)
			encodeHookData(enc, data.Data)
		}

	case EventNavigate:
		data, ok := e.Payload.(*NavigateEventData)
		if !ok || data == nil {
			enc.WriteString("")
			enc.WriteBool(false)
		} else {
			enc.WriteString(data.Path)
			enc.WriteBool(data.Replace)
		}

	case EventCustom:
		data, ok := e.Payload.(*CustomEventData)
		if !ok || data == nil {
			enc.WriteString("")
			enc.WriteLenBytes(nil)
		} else {
			enc.WriteString(data.Name)
			enc.WriteLenBytes(data.Data)
		}
	}
}

// encodeHookData encodes hook event data map.
func encodeHookData(enc *Encoder, data map[string]any) {
	enc.WriteUvarint(uint64(len(data)))
	for k, v := range data {
		enc.WriteString(k)
		encodeHookValue(enc, v)
	}
}

// encodeHookValue encodes a single hook data value.
func encodeHookValue(enc *Encoder, v any) {
	switch val := v.(type) {
	case nil:
		enc.WriteByte(byte(HookValueNull))
	case bool:
		enc.WriteByte(byte(HookValueBool))
		enc.WriteBool(val)
	case int:
		enc.WriteByte(byte(HookValueInt))
		enc.WriteSvarint(int64(val))
	case int64:
		enc.WriteByte(byte(HookValueInt))
		enc.WriteSvarint(val)
	case float64:
		enc.WriteByte(byte(HookValueFloat))
		enc.WriteFloat64(val)
	case string:
		enc.WriteByte(byte(HookValueString))
		enc.WriteString(val)
	case []any:
		enc.WriteByte(byte(HookValueArray))
		enc.WriteUvarint(uint64(len(val)))
		for _, item := range val {
			encodeHookValue(enc, item)
		}
	case map[string]any:
		enc.WriteByte(byte(HookValueObject))
		enc.WriteUvarint(uint64(len(val)))
		for k, item := range val {
			enc.WriteString(k)
			encodeHookValue(enc, item)
		}
	default:
		// Encode as null for unknown types
		enc.WriteByte(byte(HookValueNull))
	}
}

// DecodeEvent decodes an event from bytes.
func DecodeEvent(data []byte) (*Event, error) {
	d := NewDecoder(data)
	return DecodeEventFrom(d)
}

// DecodeEventFrom decodes an event from a decoder.
func DecodeEventFrom(d *Decoder) (*Event, error) {
	seq, err := d.ReadUvarint()
	if err != nil {
		return nil, err
	}

	typeByte, err := d.ReadByte()
	if err != nil {
		return nil, err
	}
	eventType := EventType(typeByte)

	hid, err := d.ReadString()
	if err != nil {
		return nil, err
	}

	e := &Event{
		Seq:  seq,
		Type: eventType,
		HID:  hid,
	}

	// Decode payload based on event type
	switch eventType {
	case EventClick, EventDblClick, EventFocus, EventBlur,
		EventMouseEnter, EventMouseLeave:
		// No payload

	case EventInput, EventChange:
		s, err := d.ReadString()
		if err != nil {
			return nil, err
		}
		e.Payload = s

	case EventSubmit:
		count, err := d.ReadCollectionCount()
		if err != nil {
			return nil, err
		}
		fields := make(map[string]string, count)
		for i := 0; i < count; i++ {
			k, err := d.ReadString()
			if err != nil {
				return nil, err
			}
			v, err := d.ReadString()
			if err != nil {
				return nil, err
			}
			fields[k] = v
		}
		e.Payload = &SubmitEventData{Fields: fields}

	case EventKeyDown, EventKeyUp, EventKeyPress:
		key, err := d.ReadString()
		if err != nil {
			return nil, err
		}
		code, err := d.ReadString()
		if err != nil {
			return nil, err
		}
		mods, err := d.ReadByte()
		if err != nil {
			return nil, err
		}
		repeat, err := d.ReadBool()
		if err != nil {
			return nil, err
		}
		location, err := d.ReadByte()
		if err != nil {
			return nil, err
		}
		e.Payload = &KeyboardEventData{
			Key:       key,
			Code:      code,
			Modifiers: Modifiers(mods),
			Repeat:    repeat,
			Location:  location,
		}

	case EventMouseDown, EventMouseUp, EventMouseMove:
		x, err := d.ReadSvarint()
		if err != nil {
			return nil, err
		}
		y, err := d.ReadSvarint()
		if err != nil {
			return nil, err
		}
		pageX, err := d.ReadSvarint()
		if err != nil {
			return nil, err
		}
		pageY, err := d.ReadSvarint()
		if err != nil {
			return nil, err
		}
		offsetX, err := d.ReadSvarint()
		if err != nil {
			return nil, err
		}
		offsetY, err := d.ReadSvarint()
		if err != nil {
			return nil, err
		}
		button, err := d.ReadByte()
		if err != nil {
			return nil, err
		}
		buttons, err := d.ReadByte()
		if err != nil {
			return nil, err
		}
		mods, err := d.ReadByte()
		if err != nil {
			return nil, err
		}
		e.Payload = &MouseEventData{
			ClientX:   int(x),
			ClientY:   int(y),
			PageX:     int(pageX),
			PageY:     int(pageY),
			OffsetX:   int(offsetX),
			OffsetY:   int(offsetY),
			Button:    button,
			Buttons:   buttons,
			Modifiers: Modifiers(mods),
		}

	case EventWheel:
		deltaX, err := d.ReadFloat64()
		if err != nil {
			return nil, err
		}
		deltaY, err := d.ReadFloat64()
		if err != nil {
			return nil, err
		}
		deltaZ, err := d.ReadFloat64()
		if err != nil {
			return nil, err
		}
		deltaMode, err := d.ReadByte()
		if err != nil {
			return nil, err
		}
		clientX, err := d.ReadSvarint()
		if err != nil {
			return nil, err
		}
		clientY, err := d.ReadSvarint()
		if err != nil {
			return nil, err
		}
		mods, err := d.ReadByte()
		if err != nil {
			return nil, err
		}
		e.Payload = &WheelEventData{
			DeltaX:    deltaX,
			DeltaY:    deltaY,
			DeltaZ:    deltaZ,
			DeltaMode: deltaMode,
			ClientX:   int(clientX),
			ClientY:   int(clientY),
			Modifiers: Modifiers(mods),
		}

	case EventScroll:
		top, err := d.ReadSvarint()
		if err != nil {
			return nil, err
		}
		left, err := d.ReadSvarint()
		if err != nil {
			return nil, err
		}
		e.Payload = &ScrollEventData{
			ScrollTop:  int(top),
			ScrollLeft: int(left),
		}

	case EventResize:
		w, err := d.ReadSvarint()
		if err != nil {
			return nil, err
		}
		h, err := d.ReadSvarint()
		if err != nil {
			return nil, err
		}
		e.Payload = &ResizeEventData{
			Width:  int(w),
			Height: int(h),
		}

	case EventTouchStart, EventTouchMove, EventTouchEnd:
		// Decode Touches
		touchCount, err := d.ReadCollectionCount()
		if err != nil {
			return nil, err
		}
		touches := make([]TouchPoint, touchCount)
		for i := 0; i < touchCount; i++ {
			id, err := d.ReadSvarint()
			if err != nil {
				return nil, err
			}
			x, err := d.ReadSvarint()
			if err != nil {
				return nil, err
			}
			y, err := d.ReadSvarint()
			if err != nil {
				return nil, err
			}
			pageX, err := d.ReadSvarint()
			if err != nil {
				return nil, err
			}
			pageY, err := d.ReadSvarint()
			if err != nil {
				return nil, err
			}
			touches[i] = TouchPoint{
				ID:      int(id),
				ClientX: int(x),
				ClientY: int(y),
				PageX:   int(pageX),
				PageY:   int(pageY),
			}
		}
		// Decode TargetTouches
		targetCount, err := d.ReadCollectionCount()
		if err != nil {
			return nil, err
		}
		targetTouches := make([]TouchPoint, targetCount)
		for i := 0; i < targetCount; i++ {
			id, err := d.ReadSvarint()
			if err != nil {
				return nil, err
			}
			x, err := d.ReadSvarint()
			if err != nil {
				return nil, err
			}
			y, err := d.ReadSvarint()
			if err != nil {
				return nil, err
			}
			pageX, err := d.ReadSvarint()
			if err != nil {
				return nil, err
			}
			pageY, err := d.ReadSvarint()
			if err != nil {
				return nil, err
			}
			targetTouches[i] = TouchPoint{
				ID:      int(id),
				ClientX: int(x),
				ClientY: int(y),
				PageX:   int(pageX),
				PageY:   int(pageY),
			}
		}
		// Decode ChangedTouches
		changedCount, err := d.ReadCollectionCount()
		if err != nil {
			return nil, err
		}
		changedTouches := make([]TouchPoint, changedCount)
		for i := 0; i < changedCount; i++ {
			id, err := d.ReadSvarint()
			if err != nil {
				return nil, err
			}
			x, err := d.ReadSvarint()
			if err != nil {
				return nil, err
			}
			y, err := d.ReadSvarint()
			if err != nil {
				return nil, err
			}
			pageX, err := d.ReadSvarint()
			if err != nil {
				return nil, err
			}
			pageY, err := d.ReadSvarint()
			if err != nil {
				return nil, err
			}
			changedTouches[i] = TouchPoint{
				ID:      int(id),
				ClientX: int(x),
				ClientY: int(y),
				PageX:   int(pageX),
				PageY:   int(pageY),
			}
		}
		e.Payload = &TouchEventData{
			Touches:        touches,
			TargetTouches:  targetTouches,
			ChangedTouches: changedTouches,
		}

	case EventDragStart, EventDragEnd, EventDrop:
		x, err := d.ReadSvarint()
		if err != nil {
			return nil, err
		}
		y, err := d.ReadSvarint()
		if err != nil {
			return nil, err
		}
		pageX, err := d.ReadSvarint()
		if err != nil {
			return nil, err
		}
		pageY, err := d.ReadSvarint()
		if err != nil {
			return nil, err
		}
		mods, err := d.ReadByte()
		if err != nil {
			return nil, err
		}
		e.Payload = &MouseEventData{
			ClientX:   int(x),
			ClientY:   int(y),
			PageX:     int(pageX),
			PageY:     int(pageY),
			Modifiers: Modifiers(mods),
		}

	case EventAnimationStart, EventAnimationEnd, EventAnimationIteration, EventAnimationCancel:
		animName, err := d.ReadString()
		if err != nil {
			return nil, err
		}
		elapsed, err := d.ReadFloat64()
		if err != nil {
			return nil, err
		}
		pseudo, err := d.ReadString()
		if err != nil {
			return nil, err
		}
		e.Payload = &AnimationEventData{
			AnimationName: animName,
			ElapsedTime:   elapsed,
			PseudoElement: pseudo,
		}

	case EventTransitionStart, EventTransitionEnd, EventTransitionRun, EventTransitionCancel:
		propName, err := d.ReadString()
		if err != nil {
			return nil, err
		}
		elapsed, err := d.ReadFloat64()
		if err != nil {
			return nil, err
		}
		pseudo, err := d.ReadString()
		if err != nil {
			return nil, err
		}
		e.Payload = &TransitionEventData{
			PropertyName:  propName,
			ElapsedTime:   elapsed,
			PseudoElement: pseudo,
		}

	case EventHook:
		name, err := d.ReadString()
		if err != nil {
			return nil, err
		}
		data, err := decodeHookData(d)
		if err != nil {
			return nil, err
		}
		e.Payload = &HookEventData{Name: name, Data: data}

	case EventNavigate:
		path, err := d.ReadString()
		if err != nil {
			return nil, err
		}
		replace, err := d.ReadBool()
		if err != nil {
			return nil, err
		}
		e.Payload = &NavigateEventData{Path: path, Replace: replace}

	case EventCustom:
		name, err := d.ReadString()
		if err != nil {
			return nil, err
		}
		data, err := d.ReadLenBytes()
		if err != nil {
			return nil, err
		}
		e.Payload = &CustomEventData{Name: name, Data: data}

	default:
		// Unknown event type - try to continue but mark as unknown
		// Skip any remaining bytes for forward compatibility
	}

	return e, nil
}

// decodeHookData decodes hook event data map.
func decodeHookData(d *Decoder) (map[string]any, error) {
	count, err := d.ReadCollectionCount()
	if err != nil {
		return nil, err
	}

	data := make(map[string]any, count)
	for i := 0; i < count; i++ {
		key, err := d.ReadString()
		if err != nil {
			return nil, err
		}
		val, err := decodeHookValue(d)
		if err != nil {
			return nil, err
		}
		data[key] = val
	}
	return data, nil
}

// MaxHookDepth is the maximum nesting depth for hook values.
// Prevents stack overflow from maliciously deeply nested payloads.
const MaxHookDepth = 64

// decodeHookValue decodes a single hook data value.
func decodeHookValue(d *Decoder) (any, error) {
	return decodeHookValueWithDepth(d, 0)
}

// decodeHookValueWithDepth decodes a hook value with depth tracking.
func decodeHookValueWithDepth(d *Decoder, depth int) (any, error) {
	if depth > MaxHookDepth {
		return nil, ErrMaxDepthExceeded
	}

	typeByte, err := d.ReadByte()
	if err != nil {
		return nil, err
	}

	switch HookValueType(typeByte) {
	case HookValueNull:
		return nil, nil

	case HookValueBool:
		return d.ReadBool()

	case HookValueInt:
		return d.ReadSvarint()

	case HookValueFloat:
		return d.ReadFloat64()

	case HookValueString:
		return d.ReadString()

	case HookValueArray:
		count, err := d.ReadCollectionCount()
		if err != nil {
			return nil, err
		}
		arr := make([]any, count)
		for i := 0; i < count; i++ {
			val, err := decodeHookValueWithDepth(d, depth+1)
			if err != nil {
				return nil, err
			}
			arr[i] = val
		}
		return arr, nil

	case HookValueObject:
		count, err := d.ReadCollectionCount()
		if err != nil {
			return nil, err
		}
		obj := make(map[string]any, count)
		for i := 0; i < count; i++ {
			key, err := d.ReadString()
			if err != nil {
				return nil, err
			}
			val, err := decodeHookValueWithDepth(d, depth+1)
			if err != nil {
				return nil, err
			}
			obj[key] = val
		}
		return obj, nil

	default:
		return nil, io.ErrUnexpectedEOF
	}
}
