package clientdist

import _ "embed"

// VangoMinJS is the production thin client JavaScript bundle.
//
// It is served by the framework at "/_vango/client.js".
//go:embed vango.min.js
var VangoMinJS []byte

