package protocol

// Depth limits to prevent stack overflow attacks via deeply nested structures.
// These limits complement the allocation limits in decoder.go.
const (
	// MaxVNodeDepth limits the maximum nesting depth of VNode trees.
	// This prevents stack overflow from maliciously deep component trees.
	// 256 levels is sufficient for any reasonable component hierarchy.
	MaxVNodeDepth = 256

	// MaxPatchDepth limits the maximum nesting depth of patch structures.
	// Patches can contain VNodes (InsertNode, ReplaceNode), so this must
	// account for VNode nesting within patches.
	MaxPatchDepth = 128

	// MaxHookDepth is defined in event.go and limits JSON payload nesting.
	// Included here for documentation: MaxHookDepth = 64
)

// NOTE: ErrMaxDepthExceeded is defined in event.go to maintain backwards compatibility.
// Use ErrMaxDepthExceeded from this package for depth limit errors.

// DepthLimits allows configuring custom depth limits for decoding.
// Use DefaultDepthLimits() for sensible defaults.
type DepthLimits struct {
	// MaxVNodeDepth is the maximum VNode tree depth.
	VNodeDepth int

	// MaxPatchDepth is the maximum patch structure depth.
	PatchDepth int

	// MaxHookDepth is the maximum JSON payload depth.
	HookDepth int
}

// DefaultDepthLimits returns the default depth limits.
func DefaultDepthLimits() *DepthLimits {
	return &DepthLimits{
		VNodeDepth: MaxVNodeDepth,
		PatchDepth: MaxPatchDepth,
		HookDepth:  MaxHookDepth,
	}
}

// depthContext tracks the current decoding depth for recursive structures.
// This is used internally by decode functions to enforce depth limits.
type depthContext struct {
	current int
	max     int
}

// newDepthContext creates a new depth context with the given maximum.
func newDepthContext(max int) *depthContext {
	return &depthContext{current: 0, max: max}
}

// enter increments the depth and returns an error if the limit would be exceeded.
// The depth is only incremented on success.
func (dc *depthContext) enter() error {
	if dc.current >= dc.max {
		return ErrMaxDepthExceeded
	}
	dc.current++
	return nil
}

// leave decrements the depth.
func (dc *depthContext) leave() {
	dc.current--
}

// checkDepth is a convenience function for one-time depth checks.
func checkDepth(current, max int) error {
	if current > max {
		return ErrMaxDepthExceeded
	}
	return nil
}
