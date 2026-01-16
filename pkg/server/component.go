package server

import (
	"fmt"
	"sync/atomic"

	"github.com/vango-go/vango/pkg/vango"
	"github.com/vango-go/vango/pkg/vdom"
)

// Component is the interface for renderable components.
// Components produce VNode trees that represent the UI.
type Component interface {
	// Render returns the VNode tree for this component.
	Render() *vdom.VNode
}

// FuncComponent wraps a render function as a Component.
type FuncComponent func() *vdom.VNode

// Render calls the wrapped function.
func (f FuncComponent) Render() *vdom.VNode {
	return f()
}

// ComponentInstance represents a mounted component with its state.
// It holds the component's reactive ownership, tracking, and rendering context.
type ComponentInstance struct {
	// InstanceID is the unique instance identifier.
	InstanceID string

	// Component is the component being rendered.
	Component Component

	// HID is the hydration ID of the root element.
	HID string

	// Owner manages signal ownership for this component.
	Owner *vango.Owner

	// Parent is the parent component instance (nil for root).
	Parent *ComponentInstance

	// Children are child component instances.
	Children []*ComponentInstance

	// Props are the current props passed to the component.
	Props map[string]any

	// dirty indicates the component needs re-rendering.
	dirty atomic.Bool

	// session is the owning session.
	session *Session

	// lastTree is the last rendered VNode tree (for diffing).
	lastTree *vdom.VNode
}

var _ vango.Listener = (*ComponentInstance)(nil)

// componentIDCounter is used to generate unique component IDs.
var componentIDCounter atomic.Uint64

// generateComponentID generates a unique component ID.
func generateComponentID() string {
	id := componentIDCounter.Add(1)
	return fmt.Sprintf("c%d", id)
}

// newComponentInstance creates a new ComponentInstance.
func newComponentInstance(component Component, parent *ComponentInstance, session *Session) *ComponentInstance {
	var parentOwner *vango.Owner
	if parent != nil {
		parentOwner = parent.Owner
	} else if session != nil {
		parentOwner = session.owner
	}

	return &ComponentInstance{
		InstanceID: generateComponentID(),
		Component:  component,
		Owner:      vango.NewOwner(parentOwner),
		Parent:     parent,
		Children:   nil,
		Props:      make(map[string]any),
		session:    session,
	}
}

// Render renders the component and returns the VNode tree.
// It sets up the tracking context so signals are properly tracked,
// and sets the runtime context so UseCtx() works during render.
func (c *ComponentInstance) Render() *vdom.VNode {
	if c.Component == nil {
		return nil
	}

	var tree *vdom.VNode

	// Create render context so UseCtx() works during component render
	var ctx Ctx
	if c.session != nil {
		ctx = c.session.createRenderContext()
	}

	// Set up tracking context for this component's owner
	// This ensures signals created during render are owned by this component
	// Also set the runtime context for UseCtx()
	vango.WithCtx(ctx, func() {
		vango.WithOwner(c.Owner, func() {
			// Start render for hook order tracking (dev mode)
			c.Owner.StartRender()
			defer c.Owner.EndRender()

			vango.WithListener(c, func() {
				tree = c.Component.Render()
			})
		})
	})

	// Store for diffing
	c.lastTree = tree

	return tree
}

// MarkDirty marks the component as needing re-render.
func (c *ComponentInstance) MarkDirty() {
	if DebugMode {
		fmt.Printf("[DEBUG] MarkDirty called for component %s\n", c.InstanceID)
	}
	if c.dirty.CompareAndSwap(false, true) {
		if DebugMode {
			fmt.Printf("[DEBUG] Component %s marked dirty\n", c.InstanceID)
		}
		if c.session != nil {
			c.session.scheduleRender(c)
		}
	}
}

// IsDirty returns whether the component needs re-rendering.
func (c *ComponentInstance) IsDirty() bool {
	return c.dirty.Load()
}

// ClearDirty clears the dirty flag.
func (c *ComponentInstance) ClearDirty() {
	c.dirty.Store(false)
}

// LastTree returns the last rendered VNode tree.
func (c *ComponentInstance) LastTree() *vdom.VNode {
	return c.lastTree
}

// SetLastTree sets the last rendered tree (used after diffing).
func (c *ComponentInstance) SetLastTree(tree *vdom.VNode) {
	c.lastTree = tree
}

// AddChild adds a child component instance.
func (c *ComponentInstance) AddChild(child *ComponentInstance) {
	c.Children = append(c.Children, child)
}

// RemoveChild removes a child component instance.
func (c *ComponentInstance) RemoveChild(child *ComponentInstance) {
	for i, ch := range c.Children {
		if ch == child {
			c.Children = append(c.Children[:i], c.Children[i+1:]...)
			return
		}
	}
}

// Dispose disposes the component instance and all its children.
func (c *ComponentInstance) Dispose() {
	// Dispose children first (reverse order)
	for i := len(c.Children) - 1; i >= 0; i-- {
		c.Children[i].Dispose()
	}
	c.Children = nil

	// Dispose our owner (cleans up signals and effects)
	if c.Owner != nil {
		c.Owner.Dispose()
	}

	// Remove from parent
	if c.Parent != nil {
		c.Parent.RemoveChild(c)
	}

	// Clear references
	c.Component = nil
	c.session = nil
	c.Owner = nil
	c.lastTree = nil
	c.Props = nil
}

// Session returns the owning session.
func (c *ComponentInstance) Session() *Session {
	return c.session
}

// SetProp sets a prop value.
func (c *ComponentInstance) SetProp(key string, value any) {
	c.Props[key] = value
}

// GetProp gets a prop value.
func (c *ComponentInstance) GetProp(key string) any {
	return c.Props[key]
}

// MemoryUsage returns an estimate of memory used by this instance.
func (c *ComponentInstance) MemoryUsage() int64 {
	if c == nil {
		return 0
	}
	var size int64 = 128 // Base struct size

	// Props
	for k, v := range c.Props {
		size += int64(len(k))
		if s, ok := v.(string); ok {
			size += int64(len(s))
		} else {
			size += 16 // Estimate for other types
		}
	}

	// LastTree
	if c.lastTree != nil {
		size += estimateVNodeSize(c.lastTree)
	}

	// Children (just count pointers, they track their own usage)
	size += int64(len(c.Children)) * 8

	return size
}

// ID implements vango.Listener and returns a globally unique identifier.
func (c *ComponentInstance) ID() uint64 {
	if c.Owner != nil {
		return c.Owner.ID()
	}
	return 0
}

// estimateVNodeSize estimates the memory size of a VNode tree.
func estimateVNodeSize(node *vdom.VNode) int64 {
	if node == nil {
		return 0
	}

	var size int64 = 64 // Base VNode size
	size += int64(len(node.Tag))
	size += int64(len(node.Text))
	size += int64(len(node.HID))
	size += int64(len(node.Key))

	// Props
	for k, v := range node.Props {
		size += int64(len(k))
		if s, ok := v.(string); ok {
			size += int64(len(s))
		} else {
			size += 16
		}
	}

	// Children
	for _, child := range node.Children {
		size += estimateVNodeSize(child)
	}

	return size
}
