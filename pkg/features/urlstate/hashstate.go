package urlstate

import (
	"github.com/vango-go/vango/pkg/vango"
)

// HashState represents reactive state synchronized with the URL hash.
type HashState struct {
	signal *vango.Signal[string]
}

// UseHash creates a new HashState.
// Since the hash is a single string, we generally don't use generics for the whole hash,
// but one could parse it. This implementation treats hash as a single string.
func UseHash(defaultValue string) *HashState {
	return &HashState{
		signal: vango.NewSignal(defaultValue),
	}
}

// Get returns the current hash value.
func (h *HashState) Get() string {
	return h.signal.Get()
}

// Set updates the hash value.
func (h *HashState) Set(value string) {
	h.signal.Set(value)
	h.updateHash(value)
}

func (h *HashState) updateHash(value string) {
	// TODO: Integration with Router/Window to update location.hash
}
