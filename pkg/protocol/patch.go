package protocol

// PatchOp is the type of patch operation.
// This is a superset of vdom.PatchOp, with additional operations for the protocol.
type PatchOp uint8

// Patch operation constants.
// Values 0x01-0x0B match vdom.PatchOp for compatibility.
const (
	// Core operations (matching vdom.PatchOp)
	PatchSetText     PatchOp = 0x01 // Update text content
	PatchSetAttr     PatchOp = 0x02 // Set attribute
	PatchRemoveAttr  PatchOp = 0x03 // Remove attribute
	PatchInsertNode  PatchOp = 0x04 // Insert new node
	PatchRemoveNode  PatchOp = 0x05 // Remove node
	PatchMoveNode    PatchOp = 0x06 // Move node
	PatchReplaceNode PatchOp = 0x07 // Replace node
	PatchSetValue    PatchOp = 0x08 // Set input value
	PatchSetChecked  PatchOp = 0x09 // Set checkbox checked
	PatchSetSelected PatchOp = 0x0A // Set select option selected
	PatchFocus       PatchOp = 0x0B // Focus element

	// Extended operations (protocol-only)
	PatchBlur        PatchOp = 0x0C // Blur element
	PatchScrollTo    PatchOp = 0x0D // Scroll to position
	PatchAddClass    PatchOp = 0x10 // Add CSS class
	PatchRemoveClass PatchOp = 0x11 // Remove CSS class
	PatchToggleClass PatchOp = 0x12 // Toggle CSS class
	PatchSetStyle    PatchOp = 0x13 // Set style property
	PatchRemoveStyle PatchOp = 0x14 // Remove style property
	PatchSetData     PatchOp = 0x15 // Set data attribute
	PatchDispatch    PatchOp = 0x20 // Dispatch client event
	// NOTE: PatchEval (0x21) has been REMOVED for security.
	// Sending arbitrary JS from server to client is an XSS/RCE risk.

	// URL operations (Phase 12: URLParam 2.0)
	PatchURLPush    PatchOp = 0x30 // Update query params, push to history
	PatchURLReplace PatchOp = 0x31 // Update query params, replace current entry
)

// String returns the string representation of the patch operation.
func (op PatchOp) String() string {
	switch op {
	case PatchSetText:
		return "SetText"
	case PatchSetAttr:
		return "SetAttr"
	case PatchRemoveAttr:
		return "RemoveAttr"
	case PatchInsertNode:
		return "InsertNode"
	case PatchRemoveNode:
		return "RemoveNode"
	case PatchMoveNode:
		return "MoveNode"
	case PatchReplaceNode:
		return "ReplaceNode"
	case PatchSetValue:
		return "SetValue"
	case PatchSetChecked:
		return "SetChecked"
	case PatchSetSelected:
		return "SetSelected"
	case PatchFocus:
		return "Focus"
	case PatchBlur:
		return "Blur"
	case PatchScrollTo:
		return "ScrollTo"
	case PatchAddClass:
		return "AddClass"
	case PatchRemoveClass:
		return "RemoveClass"
	case PatchToggleClass:
		return "ToggleClass"
	case PatchSetStyle:
		return "SetStyle"
	case PatchRemoveStyle:
		return "RemoveStyle"
	case PatchSetData:
		return "SetData"
	case PatchDispatch:
		return "Dispatch"
	case PatchURLPush:
		return "URLPush"
	case PatchURLReplace:
		return "URLReplace"
	default:
		return "Unknown"
	}
}

// ScrollBehavior represents the scroll behavior for PatchScrollTo.
type ScrollBehavior uint8

const (
	ScrollInstant ScrollBehavior = 0
	ScrollSmooth  ScrollBehavior = 1
)

// Patch represents a single DOM operation.
type Patch struct {
	Op       PatchOp
	HID      string         // Target element's hydration ID
	Key      string         // Attribute/style/class key
	Value    string         // Value for text/attr/style/class
	ParentID string         // Parent HID for InsertNode/MoveNode
	Index    int            // Insert/Move position
	Node     *VNodeWire     // For InsertNode/ReplaceNode
	Bool     bool           // For SetChecked/SetSelected
	X        int            // For ScrollTo
	Y        int            // For ScrollTo
	Behavior ScrollBehavior // For ScrollTo
	Params   map[string]string // For URLPush/URLReplace
}

// PatchesFrame represents a batch of patches with sequence number.
type PatchesFrame struct {
	Seq     uint64
	Patches []Patch
}

// EncodePatches encodes a patches frame to bytes.
func EncodePatches(pf *PatchesFrame) []byte {
	e := NewEncoder()
	EncodePatchesTo(e, pf)
	return e.Bytes()
}

// EncodePatchesTo encodes a patches frame using the provided encoder.
func EncodePatchesTo(e *Encoder, pf *PatchesFrame) {
	e.WriteUvarint(pf.Seq)
	e.WriteUvarint(uint64(len(pf.Patches)))

	for i := range pf.Patches {
		encodePatch(e, &pf.Patches[i])
	}
}

// encodePatch encodes a single patch.
func encodePatch(e *Encoder, p *Patch) {
	e.WriteByte(byte(p.Op))
	e.WriteString(p.HID)

	switch p.Op {
	case PatchSetText:
		e.WriteString(p.Value)

	case PatchSetAttr:
		e.WriteString(p.Key)
		e.WriteString(p.Value)

	case PatchRemoveAttr:
		e.WriteString(p.Key)

	case PatchInsertNode:
		e.WriteString(p.ParentID)
		e.WriteUvarint(uint64(p.Index))
		EncodeVNodeWire(e, p.Node)

	case PatchRemoveNode:
		// No additional data (HID is sufficient)

	case PatchMoveNode:
		e.WriteString(p.ParentID)
		e.WriteUvarint(uint64(p.Index))

	case PatchReplaceNode:
		EncodeVNodeWire(e, p.Node)

	case PatchSetValue:
		e.WriteString(p.Value)

	case PatchSetChecked, PatchSetSelected:
		e.WriteBool(p.Bool)

	case PatchFocus, PatchBlur:
		// No additional data

	case PatchScrollTo:
		e.WriteSvarint(int64(p.X))
		e.WriteSvarint(int64(p.Y))
		e.WriteByte(byte(p.Behavior))

	case PatchAddClass, PatchRemoveClass, PatchToggleClass:
		e.WriteString(p.Value)

	case PatchSetStyle:
		e.WriteString(p.Key)
		e.WriteString(p.Value)

	case PatchRemoveStyle:
		e.WriteString(p.Key)

	case PatchSetData:
		e.WriteString(p.Key)
		e.WriteString(p.Value)

	case PatchDispatch:
		e.WriteString(p.Key)   // Event name
		e.WriteString(p.Value) // Event detail (JSON)
		// NOTE: PatchEval case removed for security

	case PatchURLPush, PatchURLReplace:
		// Encode params as varint count + key/value pairs
		e.WriteUvarint(uint64(len(p.Params)))
		for key, value := range p.Params {
			e.WriteString(key)
			e.WriteString(value)
		}
	}
}

// DecodePatches decodes a patches frame from bytes.
func DecodePatches(data []byte) (*PatchesFrame, error) {
	d := NewDecoder(data)
	return DecodePatchesFrom(d)
}

// DecodePatchesFrom decodes a patches frame from a decoder.
// SECURITY: Enforces MaxPatchDepth to prevent stack overflow attacks.
func DecodePatchesFrom(d *Decoder) (*PatchesFrame, error) {
	return decodePatchesFromWithDepth(d, 0)
}

// decodePatchesFromWithDepth decodes a patches frame with depth tracking.
func decodePatchesFromWithDepth(d *Decoder, depth int) (*PatchesFrame, error) {
	// SECURITY: Check depth limit before any work
	if err := checkDepth(depth, MaxPatchDepth); err != nil {
		return nil, err
	}

	seq, err := d.ReadUvarint()
	if err != nil {
		return nil, err
	}

	// SECURITY: Use ReadCollectionCount to prevent DoS via huge allocation
	count, err := d.ReadCollectionCount()
	if err != nil {
		return nil, err
	}

	patches := make([]Patch, count)
	for i := 0; i < count; i++ {
		// SECURITY: Pass depth for VNode decoding inside patches
		if err := decodePatchWithDepth(d, &patches[i], depth+1); err != nil {
			return nil, err
		}
	}

	return &PatchesFrame{
		Seq:     seq,
		Patches: patches,
	}, nil
}

// decodePatch decodes a single patch.
// This is the legacy version that calls the depth-aware version with depth 0.
func decodePatch(d *Decoder, p *Patch) error {
	return decodePatchWithDepth(d, p, 0)
}

// decodePatchWithDepth decodes a single patch with depth tracking.
// The depth parameter is passed to VNode decoding for patches that contain nodes.
func decodePatchWithDepth(d *Decoder, p *Patch, depth int) error {
	opByte, err := d.ReadByte()
	if err != nil {
		return err
	}
	p.Op = PatchOp(opByte)

	p.HID, err = d.ReadString()
	if err != nil {
		return err
	}

	switch p.Op {
	case PatchSetText:
		p.Value, err = d.ReadString()

	case PatchSetAttr:
		p.Key, err = d.ReadString()
		if err != nil {
			return err
		}
		p.Value, err = d.ReadString()

	case PatchRemoveAttr:
		p.Key, err = d.ReadString()

	case PatchInsertNode:
		p.ParentID, err = d.ReadString()
		if err != nil {
			return err
		}
		var idx uint64
		idx, err = d.ReadUvarint()
		if err != nil {
			return err
		}
		p.Index = int(idx)
		// SECURITY: Use depth-aware VNode decoding
		p.Node, err = decodeVNodeWireWithDepth(d, depth)

	case PatchRemoveNode:
		// No additional data

	case PatchMoveNode:
		p.ParentID, err = d.ReadString()
		if err != nil {
			return err
		}
		var idx uint64
		idx, err = d.ReadUvarint()
		p.Index = int(idx)

	case PatchReplaceNode:
		// SECURITY: Use depth-aware VNode decoding
		p.Node, err = decodeVNodeWireWithDepth(d, depth)

	case PatchSetValue:
		p.Value, err = d.ReadString()

	case PatchSetChecked, PatchSetSelected:
		p.Bool, err = d.ReadBool()

	case PatchFocus, PatchBlur:
		// No additional data

	case PatchScrollTo:
		var x, y int64
		x, err = d.ReadSvarint()
		if err != nil {
			return err
		}
		y, err = d.ReadSvarint()
		if err != nil {
			return err
		}
		p.X = int(x)
		p.Y = int(y)
		var beh byte
		beh, err = d.ReadByte()
		p.Behavior = ScrollBehavior(beh)

	case PatchAddClass, PatchRemoveClass, PatchToggleClass:
		p.Value, err = d.ReadString()

	case PatchSetStyle:
		p.Key, err = d.ReadString()
		if err != nil {
			return err
		}
		p.Value, err = d.ReadString()

	case PatchRemoveStyle:
		p.Key, err = d.ReadString()

	case PatchSetData:
		p.Key, err = d.ReadString()
		if err != nil {
			return err
		}
		p.Value, err = d.ReadString()

	case PatchDispatch:
		p.Key, err = d.ReadString()
		if err != nil {
			return err
		}
		p.Value, err = d.ReadString()
		// NOTE: PatchEval case removed for security

	case PatchURLPush, PatchURLReplace:
		// Decode params
		count, err := d.ReadCollectionCount()
		if err != nil {
			return err
		}
		p.Params = make(map[string]string, count)
		for i := 0; i < count; i++ {
			key, err := d.ReadString()
			if err != nil {
				return err
			}
			value, err := d.ReadString()
			if err != nil {
				return err
			}
			p.Params[key] = value
		}

	default:
		// Unknown patch op - skip for forward compatibility
	}

	return err
}

// NewSetTextPatch creates a SetText patch.
func NewSetTextPatch(hid, text string) Patch {
	return Patch{Op: PatchSetText, HID: hid, Value: text}
}

// NewSetAttrPatch creates a SetAttr patch.
func NewSetAttrPatch(hid, key, value string) Patch {
	return Patch{Op: PatchSetAttr, HID: hid, Key: key, Value: value}
}

// NewRemoveAttrPatch creates a RemoveAttr patch.
func NewRemoveAttrPatch(hid, key string) Patch {
	return Patch{Op: PatchRemoveAttr, HID: hid, Key: key}
}

// NewInsertNodePatch creates an InsertNode patch.
func NewInsertNodePatch(hid, parentID string, index int, node *VNodeWire) Patch {
	return Patch{Op: PatchInsertNode, HID: hid, ParentID: parentID, Index: index, Node: node}
}

// NewRemoveNodePatch creates a RemoveNode patch.
func NewRemoveNodePatch(hid string) Patch {
	return Patch{Op: PatchRemoveNode, HID: hid}
}

// NewMoveNodePatch creates a MoveNode patch.
func NewMoveNodePatch(hid, parentID string, index int) Patch {
	return Patch{Op: PatchMoveNode, HID: hid, ParentID: parentID, Index: index}
}

// NewReplaceNodePatch creates a ReplaceNode patch.
func NewReplaceNodePatch(hid string, node *VNodeWire) Patch {
	return Patch{Op: PatchReplaceNode, HID: hid, Node: node}
}

// NewSetValuePatch creates a SetValue patch.
func NewSetValuePatch(hid, value string) Patch {
	return Patch{Op: PatchSetValue, HID: hid, Value: value}
}

// NewSetCheckedPatch creates a SetChecked patch.
func NewSetCheckedPatch(hid string, checked bool) Patch {
	return Patch{Op: PatchSetChecked, HID: hid, Bool: checked}
}

// NewSetSelectedPatch creates a SetSelected patch.
func NewSetSelectedPatch(hid string, selected bool) Patch {
	return Patch{Op: PatchSetSelected, HID: hid, Bool: selected}
}

// NewFocusPatch creates a Focus patch.
func NewFocusPatch(hid string) Patch {
	return Patch{Op: PatchFocus, HID: hid}
}

// NewBlurPatch creates a Blur patch.
func NewBlurPatch(hid string) Patch {
	return Patch{Op: PatchBlur, HID: hid}
}

// NewScrollToPatch creates a ScrollTo patch.
func NewScrollToPatch(hid string, x, y int, behavior ScrollBehavior) Patch {
	return Patch{Op: PatchScrollTo, HID: hid, X: x, Y: y, Behavior: behavior}
}

// NewAddClassPatch creates an AddClass patch.
func NewAddClassPatch(hid, class string) Patch {
	return Patch{Op: PatchAddClass, HID: hid, Value: class}
}

// NewRemoveClassPatch creates a RemoveClass patch.
func NewRemoveClassPatch(hid, class string) Patch {
	return Patch{Op: PatchRemoveClass, HID: hid, Value: class}
}

// NewToggleClassPatch creates a ToggleClass patch.
func NewToggleClassPatch(hid, class string) Patch {
	return Patch{Op: PatchToggleClass, HID: hid, Value: class}
}

// NewSetStylePatch creates a SetStyle patch.
func NewSetStylePatch(hid, property, value string) Patch {
	return Patch{Op: PatchSetStyle, HID: hid, Key: property, Value: value}
}

// NewRemoveStylePatch creates a RemoveStyle patch.
func NewRemoveStylePatch(hid, property string) Patch {
	return Patch{Op: PatchRemoveStyle, HID: hid, Key: property}
}

// NewSetDataPatch creates a SetData patch.
func NewSetDataPatch(hid, key, value string) Patch {
	return Patch{Op: PatchSetData, HID: hid, Key: key, Value: value}
}

// NewDispatchPatch creates a Dispatch patch.
func NewDispatchPatch(hid, eventName, detail string) Patch {
	return Patch{Op: PatchDispatch, HID: hid, Key: eventName, Value: detail}
}

// NOTE: NewEvalPatch has been REMOVED for security.
// Sending arbitrary JS from server to client is an XSS/RCE risk.
// Use client-side hooks or PatchDispatch for safe interop.

// NewURLPushPatch creates a URLPush patch (adds history entry).
func NewURLPushPatch(params map[string]string) Patch {
	return Patch{Op: PatchURLPush, Params: params}
}

// NewURLReplacePatch creates a URLReplace patch (replaces current entry).
func NewURLReplacePatch(params map[string]string) Patch {
	return Patch{Op: PatchURLReplace, Params: params}
}
