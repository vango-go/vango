package vdom

// PatchOp is the type of patch operation.
type PatchOp uint8

const (
	PatchSetText      PatchOp = 0x01 // Update text content
	PatchSetAttr      PatchOp = 0x02 // Set/update attribute
	PatchRemoveAttr   PatchOp = 0x03 // Remove attribute
	PatchInsertNode   PatchOp = 0x04 // Insert new node
	PatchRemoveNode   PatchOp = 0x05 // Remove node
	PatchMoveNode     PatchOp = 0x06 // Move node to new position
	PatchReplaceNode  PatchOp = 0x07 // Replace node entirely
	PatchSetValue     PatchOp = 0x08 // Set input value
	PatchSetChecked   PatchOp = 0x09 // Set checkbox checked
	PatchSetSelected  PatchOp = 0x0A // Set select option selected
	PatchFocus        PatchOp = 0x0B // Focus element
)

// String returns the string representation of the PatchOp.
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
	default:
		return "Unknown"
	}
}

// Patch represents a single DOM operation to apply.
type Patch struct {
	Op       PatchOp // Operation type
	HID      string  // Target element's hydration ID
	Key      string  // Attribute key (for SetAttr/RemoveAttr)
	Value    string  // New value
	Node     *VNode  // For InsertNode/ReplaceNode
	Index    int     // Insert position
	ParentID string  // Parent for InsertNode
}
