package clientdist

import _ "embed"

// VangoMinJS is the production thin client JavaScript bundle.
//
// It is served by the framework at "/_vango/client.js".
//go:embed vango.min.js
var VangoMinJS []byte

// VangoJS is the non-minified thin client JavaScript bundle.
//go:embed vango.js
var VangoJS []byte

// VangoJSMap is the source map for VangoJS.
//go:embed vango.js.map
var VangoJSMap []byte
