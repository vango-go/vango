package urlparam

import "github.com/vango-dev/vango/v2/pkg/protocol"

// NavigatorKey is the context key for the URL navigator.
// The session sets this on the root owner so URLParam can queue patches.
var NavigatorKey = &struct{ name string }{"URLNavigator"}

// InitialParamsKey is the context key for initial URL params.
// The session sets this from the client handshake payload.
var InitialParamsKey = &struct{ name string }{"InitialURLParams"}

// InitialURLState holds initial URL params with consume-once semantics.
// Once consumed by the first URLParam, it won't be used again to avoid
// re-hydrating on every render.
type InitialURLState struct {
	Path     string
	Params   map[string]string
	consumed bool
}

// IsConsumed returns whether the initial state has been consumed.
func (s *InitialURLState) IsConsumed() bool {
	return s.consumed
}

// Consume marks the initial state as consumed and returns the params.
// Subsequent calls return nil.
func (s *InitialURLState) Consume() map[string]string {
	if s.consumed {
		return nil
	}
	s.consumed = true
	return s.Params
}

// Navigator handles URL updates for URLParam.
// It queues URL patches that are sent to the client along with DOM patches.
type Navigator struct {
	queuePatch func(protocol.Patch)
}

// NewNavigator creates a navigator that queues patches via the provided function.
// The session passes in a closure that appends to its pending patch buffer.
func NewNavigator(queuePatch func(protocol.Patch)) *Navigator {
	return &Navigator{queuePatch: queuePatch}
}

// Navigate queues a URL update patch.
// The patch will be sent to the client with other DOM patches in the same tick.
func (n *Navigator) Navigate(params map[string]string, mode URLMode) {
	if n.queuePatch == nil {
		return
	}
	var patch protocol.Patch
	if mode == ModeReplace {
		patch = protocol.NewURLReplacePatch(params)
	} else {
		patch = protocol.NewURLPushPatch(params)
	}
	n.queuePatch(patch)
}
