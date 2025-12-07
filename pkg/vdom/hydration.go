package vdom

import (
	"fmt"
	"sync"
)

// HIDGenerator generates unique hydration IDs for interactive elements.
type HIDGenerator struct {
	counter uint32
	mu      sync.Mutex
}

// NewHIDGenerator creates a new HIDGenerator.
func NewHIDGenerator() *HIDGenerator {
	return &HIDGenerator{}
}

// Next returns the next hydration ID (e.g., "h1", "h2", ...).
func (g *HIDGenerator) Next() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.counter++
	return fmt.Sprintf("h%d", g.counter)
}

// Reset resets the counter to 0.
func (g *HIDGenerator) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.counter = 0
}

// Current returns the current counter value without incrementing.
func (g *HIDGenerator) Current() uint32 {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.counter
}

// AssignHIDs walks the tree and assigns HIDs to interactive elements.
// An element is interactive if it has event handlers (props starting with "on").
func AssignHIDs(node *VNode, gen *HIDGenerator) {
	if node == nil {
		return
	}

	// Only elements can be interactive
	if node.Kind == KindElement && node.IsInteractive() {
		node.HID = gen.Next()
	}

	// Recurse into children
	for _, child := range node.Children {
		AssignHIDs(child, gen)
	}

	// For component nodes, we might need to assign HIDs to the rendered output
	// However, components are typically rendered at runtime, so this is handled there
}

// AssignAllHIDs assigns HIDs to ALL element nodes, not just interactive ones.
// This is useful for debugging or when all elements need to be addressable.
func AssignAllHIDs(node *VNode, gen *HIDGenerator) {
	if node == nil {
		return
	}

	// Assign HID to all elements
	if node.Kind == KindElement {
		node.HID = gen.Next()
	}

	// Recurse into children
	for _, child := range node.Children {
		AssignAllHIDs(child, gen)
	}
}

// CollectHIDs returns a map of HID to VNode for all nodes with HIDs.
func CollectHIDs(node *VNode) map[string]*VNode {
	result := make(map[string]*VNode)
	collectHIDs(node, result)
	return result
}

func collectHIDs(node *VNode, result map[string]*VNode) {
	if node == nil {
		return
	}

	if node.HID != "" {
		result[node.HID] = node
	}

	for _, child := range node.Children {
		collectHIDs(child, result)
	}
}

// FindByHID finds a node by its HID in the tree.
func FindByHID(node *VNode, hid string) *VNode {
	if node == nil {
		return nil
	}

	if node.HID == hid {
		return node
	}

	for _, child := range node.Children {
		if found := FindByHID(child, hid); found != nil {
			return found
		}
	}

	return nil
}

// CountInteractive returns the number of interactive elements in the tree.
func CountInteractive(node *VNode) int {
	if node == nil {
		return 0
	}

	count := 0
	if node.Kind == KindElement && node.IsInteractive() {
		count = 1
	}

	for _, child := range node.Children {
		count += CountInteractive(child)
	}

	return count
}

// ClearHIDs removes all HIDs from the tree.
func ClearHIDs(node *VNode) {
	if node == nil {
		return
	}

	node.HID = ""

	for _, child := range node.Children {
		ClearHIDs(child)
	}
}

// CopyHIDs copies HIDs from the source tree to the destination tree.
// This is useful when diffing to preserve HIDs between renders.
// Returns true if all HIDs were successfully copied.
func CopyHIDs(src, dst *VNode) bool {
	if src == nil || dst == nil {
		return src == nil && dst == nil
	}

	// Copy HID
	dst.HID = src.HID

	// For same-structure trees, copy children HIDs
	if len(src.Children) != len(dst.Children) {
		return false
	}

	for i := range src.Children {
		if !CopyHIDs(src.Children[i], dst.Children[i]) {
			return false
		}
	}

	return true
}
