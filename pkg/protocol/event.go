package protocol

import (
	"errors"
	"io"
)

// EventType identifies the type of client event.
type EventType uint8

// Event type constants.
const (
	// Mouse events (0x01-0x07)
	EventClick      EventType = 0x01
	EventDblClick   EventType = 0x02
	EventMouseDown  EventType = 0x03
	EventMouseUp    EventType = 0x04
	EventMouseMove  EventType = 0x05
	EventMouseEnter EventType = 0x06
	EventMouseLeave EventType = 0x07

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
	Modifiers Modifiers
}

// MouseEventData contains mouse event data.
type MouseEventData struct {
	ClientX   int
	ClientY   int
	Button    uint8
	Modifiers Modifiers
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
}

// TouchEventData contains touch event data.
type TouchEventData struct {
	Touches []TouchPoint
}

// HookValueType identifies the type of a hook data value.
type HookValueType uint8

const (
	HookValueNull    HookValueType = 0x00
	HookValueBool    HookValueType = 0x01
	HookValueInt     HookValueType = 0x02
	HookValueFloat   HookValueType = 0x03
	HookValueString  HookValueType = 0x04
	HookValueArray   HookValueType = 0x05
	HookValueObject  HookValueType = 0x06
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
			enc.WriteByte(0)
		} else {
			enc.WriteString(data.Key)
			enc.WriteByte(byte(data.Modifiers))
		}

	case EventMouseDown, EventMouseUp, EventMouseMove:
		data, ok := e.Payload.(*MouseEventData)
		if !ok || data == nil {
			enc.WriteSvarint(0)
			enc.WriteSvarint(0)
			enc.WriteByte(0)
			enc.WriteByte(0)
		} else {
			enc.WriteSvarint(int64(data.ClientX))
			enc.WriteSvarint(int64(data.ClientY))
			enc.WriteByte(data.Button)
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
		} else {
			enc.WriteUvarint(uint64(len(data.Touches)))
			for _, t := range data.Touches {
				enc.WriteSvarint(int64(t.ID))
				enc.WriteSvarint(int64(t.ClientX))
				enc.WriteSvarint(int64(t.ClientY))
			}
		}

	case EventDragStart, EventDragEnd, EventDrop:
		// Similar to mouse events - include position
		data, ok := e.Payload.(*MouseEventData)
		if !ok || data == nil {
			enc.WriteSvarint(0)
			enc.WriteSvarint(0)
			enc.WriteByte(0)
		} else {
			enc.WriteSvarint(int64(data.ClientX))
			enc.WriteSvarint(int64(data.ClientY))
			enc.WriteByte(byte(data.Modifiers))
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
		count, err := d.ReadUvarint()
		if err != nil {
			return nil, err
		}
		fields := make(map[string]string, count)
		for i := uint64(0); i < count; i++ {
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
		mods, err := d.ReadByte()
		if err != nil {
			return nil, err
		}
		e.Payload = &KeyboardEventData{
			Key:       key,
			Modifiers: Modifiers(mods),
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
		button, err := d.ReadByte()
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
			Button:    button,
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
		count, err := d.ReadUvarint()
		if err != nil {
			return nil, err
		}
		touches := make([]TouchPoint, count)
		for i := uint64(0); i < count; i++ {
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
			touches[i] = TouchPoint{
				ID:      int(id),
				ClientX: int(x),
				ClientY: int(y),
			}
		}
		e.Payload = &TouchEventData{Touches: touches}

	case EventDragStart, EventDragEnd, EventDrop:
		x, err := d.ReadSvarint()
		if err != nil {
			return nil, err
		}
		y, err := d.ReadSvarint()
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
			Modifiers: Modifiers(mods),
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
	count, err := d.ReadUvarint()
	if err != nil {
		return nil, err
	}

	data := make(map[string]any, count)
	for i := uint64(0); i < count; i++ {
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

// decodeHookValue decodes a single hook data value.
func decodeHookValue(d *Decoder) (any, error) {
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
		count, err := d.ReadUvarint()
		if err != nil {
			return nil, err
		}
		arr := make([]any, count)
		for i := uint64(0); i < count; i++ {
			val, err := decodeHookValue(d)
			if err != nil {
				return nil, err
			}
			arr[i] = val
		}
		return arr, nil

	case HookValueObject:
		count, err := d.ReadUvarint()
		if err != nil {
			return nil, err
		}
		obj := make(map[string]any, count)
		for i := uint64(0); i < count; i++ {
			key, err := d.ReadString()
			if err != nil {
				return nil, err
			}
			val, err := decodeHookValue(d)
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
