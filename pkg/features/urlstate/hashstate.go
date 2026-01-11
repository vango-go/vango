package urlstate

import (
	"encoding/json"

	"github.com/vango-go/vango/pkg/protocol"
	"github.com/vango-go/vango/pkg/urlparam"
	"github.com/vango-go/vango/pkg/vango"
)

// HashState represents reactive state synchronized with the URL hash.
type HashState struct {
	defaultValue string
	signal       *vango.Signal[string]
}

// UseHash creates a new HashState.
// Since the hash is a single string, we generally don't use generics for the whole hash,
// but one could parse it. This implementation treats hash as a single string.
func UseHash(defaultValue string) *HashState {
	vango.TrackHook(vango.HookURLParam)

	slot := vango.UseHookSlot()
	var h *HashState
	first := false
	if slot != nil {
		existing, ok := slot.(*HashState)
		if !ok {
			panic("vango: hook slot type mismatch for HashState")
		}
		h = existing
	} else {
		first = true
		h = &HashState{}
		vango.SetHookSlot(h)
	}

	if first {
		h.defaultValue = defaultValue
	}

	// Must be called on every render to preserve hook slot order.
	h.signal = vango.NewSignal(defaultValue)

	return h
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
	navCtx := vango.GetContext(urlparam.NavigatorKey)
	if navCtx == nil {
		return
	}
	nav, ok := navCtx.(*urlparam.Navigator)
	if !ok || nav == nil {
		return
	}

	detail, err := json.Marshal(struct {
		Value   string `json:"value"`
		Replace bool   `json:"replace"`
	}{
		Value:   value,
		Replace: false,
	})
	if err != nil {
		return
	}

	nav.QueuePatch(protocol.NewDispatchPatch("", "vango:hash", string(detail)))
}
