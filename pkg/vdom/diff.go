package vdom

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Diff compares two VNode trees and returns the patches needed to transform prev into next.
func Diff(prev, next *VNode) []Patch {
	var patches []Patch
	diff(prev, next, "", &patches)
	return patches
}

// diff recursively compares nodes and appends patches.
// parentHID is the HID of the parent element, used for text patches that don't have their own HID.
func diff(prev, next *VNode, parentHID string, patches *[]Patch) {
	// Both nil - nothing to do
	if prev == nil && next == nil {
		return
	}

	// Node added (handled by parent via InsertNode)
	if prev == nil {
		return
	}

	// Node removed
	if next == nil {
		*patches = append(*patches, Patch{
			Op:  PatchRemoveNode,
			HID: prev.HID,
		})
		return
	}

	// Different types - replace
	if prev.Kind != next.Kind {
		*patches = append(*patches, Patch{
			Op:   PatchReplaceNode,
			HID:  prev.HID,
			Node: next,
		})
		return
	}

	// Same type, diff by kind
	switch prev.Kind {
	case KindText:
		diffText(prev, next, parentHID, patches)
	case KindElement:
		diffElement(prev, next, patches)
	case KindFragment:
		diffFragment(prev, next, parentHID, patches)
	case KindComponent:
		diffComponent(prev, next, parentHID, patches)
	case KindRaw:
		diffRaw(prev, next, parentHID, patches)
	}
}

// diffText compares text nodes.
// parentHID is used when the text node doesn't have its own HID.
func diffText(prev, next *VNode, parentHID string, patches *[]Patch) {
	// Copy HID from prev to next
	next.HID = prev.HID

	if prev.Text != next.Text {
		// Text nodes don't have HIDs, so use the parent element's HID
		// The client will update the parent's textContent
		targetHID := prev.HID
		if targetHID == "" {
			targetHID = parentHID
		}
		if targetHID != "" {
			*patches = append(*patches, Patch{
				Op:    PatchSetText,
				HID:   targetHID,
				Value: next.Text,
			})
		}
	}
}

// diffElement compares element nodes.
func diffElement(prev, next *VNode, patches *[]Patch) {
	// Different tag - replace entire node
	if prev.Tag != next.Tag {
		*patches = append(*patches, Patch{
			Op:   PatchReplaceNode,
			HID:  prev.HID,
			Node: next,
		})
		return
	}

	// Copy HID from prev to next
	next.HID = prev.HID

	// Diff props
	diffProps(prev, next, patches)

	// Diff children - pass this element's HID as the parent for text nodes
	diffChildren(prev, next, prev.HID, patches)
}

// diffFragment compares fragment nodes.
func diffFragment(prev, next *VNode, parentHID string, patches *[]Patch) {
	// Copy HID (fragments typically don't have HIDs, but just in case)
	next.HID = prev.HID

	// Diff children - pass the parentHID since fragments don't have their own
	diffChildren(prev, next, parentHID, patches)
}

// diffComponent compares component nodes.
func diffComponent(prev, next *VNode, parentHID string, patches *[]Patch) {
	// Copy HID
	next.HID = prev.HID

	// For components, we need to render them and diff the output.
	// However, in the server-driven model, components are rendered by the runtime.
	// Here we just mark that the component needs to be re-rendered.
	// The actual comparison happens at the rendered VNode level.

	// If both have rendered output, diff those
	if prev.Comp != nil && next.Comp != nil {
		// Render both and diff
		prevRendered := prev.Comp.Render()
		nextRendered := next.Comp.Render()
		diff(prevRendered, nextRendered, parentHID, patches)
	}
}

// diffRaw compares raw HTML nodes.
func diffRaw(prev, next *VNode, parentHID string, patches *[]Patch) {
	// Copy HID
	next.HID = prev.HID

	if prev.Text != next.Text {
		// Raw HTML changed - need to replace
		// Use parentHID if the raw node doesn't have its own HID
		targetHID := prev.HID
		if targetHID == "" {
			targetHID = parentHID
		}
		if targetHID != "" {
			*patches = append(*patches, Patch{
				Op:   PatchReplaceNode,
				HID:  targetHID,
				Node: next,
			})
		}
	}
}

// diffProps compares and patches attributes.
func diffProps(prev, next *VNode, patches *[]Patch) {
	// Check for removed/changed props
	for key, prevVal := range prev.Props {
		if isEventHandler(key) {
			continue // Events handled separately by runtime
		}
		if key == "key" {
			continue // Key is not a real attribute
		}

		nextVal, exists := next.Props[key]
		if !exists {
			// Attribute removed
			*patches = append(*patches, Patch{
				Op:  PatchRemoveAttr,
				HID: prev.HID,
				Key: key,
			})
		} else if !propsEqual(prevVal, nextVal) {
			// Attribute changed
			*patches = append(*patches, Patch{
				Op:    PatchSetAttr,
				HID:   prev.HID,
				Key:   key,
				Value: propToString(nextVal),
			})
		}
	}

	// Check for added props
	for key, nextVal := range next.Props {
		if isEventHandler(key) {
			continue
		}
		if key == "key" {
			continue
		}

		if _, exists := prev.Props[key]; !exists {
			// Attribute added
			*patches = append(*patches, Patch{
				Op:    PatchSetAttr,
				HID:   prev.HID,
				Key:   key,
				Value: propToString(nextVal),
			})
		}
	}
}

// diffChildren compares and patches child nodes.
// parentHID is passed through so text node patches can target the parent element.
func diffChildren(prev, next *VNode, parentHID string, patches *[]Patch) {
	prevChildren := prev.Children
	nextChildren := next.Children

	// Check if children are keyed
	if hasKeys(prevChildren) || hasKeys(nextChildren) {
		diffKeyedChildren(prev, prevChildren, nextChildren, parentHID, patches)
	} else {
		diffUnkeyedChildren(prev, prevChildren, nextChildren, parentHID, patches)
	}
}

// diffUnkeyedChildren handles children without keys using positional matching.
func diffUnkeyedChildren(parent *VNode, prev, next []*VNode, parentHID string, patches *[]Patch) {
	maxLen := len(prev)
	if len(next) > maxLen {
		maxLen = len(next)
	}

	for i := 0; i < maxLen; i++ {
		var prevChild, nextChild *VNode

		if i < len(prev) {
			prevChild = prev[i]
		}
		if i < len(next) {
			nextChild = next[i]
		}

		if prevChild == nil && nextChild != nil {
			// Insert new child
			*patches = append(*patches, Patch{
				Op:       PatchInsertNode,
				ParentID: parent.HID,
				Index:    i,
				Node:     nextChild,
			})
		} else if prevChild != nil && nextChild == nil {
			// Remove child
			*patches = append(*patches, Patch{
				Op:  PatchRemoveNode,
				HID: prevChild.HID,
			})
		} else {
			// Diff existing - pass parent HID for text nodes
			diff(prevChild, nextChild, parentHID, patches)
		}
	}
}

// diffKeyedChildren handles children with keys for efficient reordering.
func diffKeyedChildren(parent *VNode, prev, next []*VNode, parentHID string, patches *[]Patch) {
	// Build key maps: key -> index
	prevKeyMap := make(map[string]int)
	nextKeyMap := make(map[string]int)

	for i, child := range prev {
		if key := getKey(child); key != "" {
			prevKeyMap[key] = i
		}
	}
	for i, child := range next {
		if key := getKey(child); key != "" {
			nextKeyMap[key] = i
		}
	}

	// Track which prev nodes have been matched
	matched := make(map[int]bool)

	// Process next children in order
	for nextIdx, nextChild := range next {
		key := getKey(nextChild)

		if key != "" {
			if prevIdx, exists := prevKeyMap[key]; exists {
				// Found matching key
				matched[prevIdx] = true
				prevChild := prev[prevIdx]

				// Check if position changed
				if prevIdx != nextIdx {
					*patches = append(*patches, Patch{
						Op:       PatchMoveNode,
						HID:      prevChild.HID,
						ParentID: parent.HID,
						Index:    nextIdx,
					})
				}

				// Diff the node itself - pass parent HID for text nodes
				diff(prevChild, nextChild, parentHID, patches)
			} else {
				// New node with key
				*patches = append(*patches, Patch{
					Op:       PatchInsertNode,
					ParentID: parent.HID,
					Index:    nextIdx,
					Node:     nextChild,
				})
			}
		} else {
			// Unkeyed node in keyed list - treat as insert
			*patches = append(*patches, Patch{
				Op:       PatchInsertNode,
				ParentID: parent.HID,
				Index:    nextIdx,
				Node:     nextChild,
			})
		}
	}

	// Remove unmatched prev nodes
	for i, prevChild := range prev {
		if !matched[i] {
			*patches = append(*patches, Patch{
				Op:  PatchRemoveNode,
				HID: prevChild.HID,
			})
		}
	}
}

// getKey extracts the key from a node's props.
func getKey(node *VNode) string {
	if node == nil {
		return ""
	}
	// Check Key field first (faster)
	if node.Key != "" {
		return node.Key
	}
	// Fall back to Props
	if node.Props == nil {
		return ""
	}
	if key, ok := node.Props["key"].(string); ok {
		return key
	}
	return ""
}

// hasKeys returns true if any child has a key.
func hasKeys(children []*VNode) bool {
	for _, child := range children {
		if getKey(child) != "" {
			return true
		}
	}
	return false
}

// isEventHandler returns true if the key is an event handler (starts with "on").
// SECURITY: Case-insensitive to catch onclick, ONCLICK, onClick, OnLoad, etc.
func isEventHandler(key string) bool {
	return len(key) > 2 && strings.EqualFold(key[:2], "on")
}

// propsEqual compares two prop values for equality.
func propsEqual(a, b any) bool {
	// Fast path for common types
	switch av := a.(type) {
	case string:
		if bv, ok := b.(string); ok {
			return av == bv
		}
		return false
	case int:
		if bv, ok := b.(int); ok {
			return av == bv
		}
		return false
	case int64:
		if bv, ok := b.(int64); ok {
			return av == bv
		}
		return false
	case float64:
		if bv, ok := b.(float64); ok {
			return av == bv
		}
		return false
	case bool:
		if bv, ok := b.(bool); ok {
			return av == bv
		}
		return false
	case nil:
		return b == nil
	}
	// Fallback to reflect for complex types
	return reflect.DeepEqual(a, b)
}

// propToString converts a prop value to a string for the patch.
func propToString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case bool:
		if val {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", v)
	}
}
