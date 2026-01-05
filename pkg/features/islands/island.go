package islands

import (
	"encoding/json"

	"github.com/vango-go/vango/pkg/vango"
	"github.com/vango-go/vango/pkg/vdom"
)

// JSModule is a path to a JavaScript module.
type JSModule string

// JSProps are properties passed to the island.
type JSProps map[string]any

// JSIsland creates a new JavaScript island VNode.
func JSIsland(id string, module JSModule, props JSProps) *vdom.VNode {
	propsJSON, _ := json.Marshal(props)

	// Render a container div that the JS module will mount into.
	// We use data-attributes for the thin client to identify and mount.
	return &vdom.VNode{
		Kind: vdom.KindElement,
		Tag:  "div",
		Key:  id,
		Props: vdom.Props{
			"id":          id,
			"data-island": id,
			"data-module": string(module),
			"data-props":  string(propsJSON),
			"class":       "vango-island",
		},
	}
}

// SendToIsland sends a message to the client-side island.
func SendToIsland(id string, message map[string]any) {
	// In a real implementation, this would queue a message to the active session.
	// For now, we mock it or require session context.
	// TODO: Integrate with Session.Send(id, message)
}

// OnIslandMessage registers a handler for messages from the island.
func OnIslandMessage(id string, handler func(map[string]any)) {
	// This registers an effect that listens for events.
	// In Vango, interacting with outside inputs typically happens via Signals or specialized listeners.
	// The runtime would route events to this handler.

	// We use CreateEffect to ensure this runs in the right scope?
	// Actually, handlers are usually static.
	// But if the handler closes over component state (Signal), it's fine.

	// We need to register this with the Session so incoming WebSocket messages
	// for this Island ID are routed here.

	vango.CreateEffect(func() vango.Cleanup {
		// Registration logic
		// session := vango.GetContext(SessionKey)?

		return func() {
			// Cleanup/Unregister logic
		}
	})
}
