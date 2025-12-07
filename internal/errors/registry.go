package errors

// ErrorTemplate defines a registered error type.
type ErrorTemplate struct {
	Category Category
	Message  string
	Detail   string
	DocURL   string
}

// registry maps error codes to their templates.
var registry = map[string]ErrorTemplate{
	// ============================================
	// Runtime Errors (E001-E099)
	// ============================================

	"E001": {
		Category: CategoryRuntime,
		Message:  "Signal read outside component context",
		Detail:   "Signals must be read inside a component's render function, wrapped with vango.Func().",
		DocURL:   "https://vango.dev/docs/errors/E001",
	},
	"E002": {
		Category: CategoryRuntime,
		Message:  "Effect created outside component context",
		Detail:   "Effects must be created inside a component's render function.",
		DocURL:   "https://vango.dev/docs/errors/E002",
	},
	"E003": {
		Category: CategoryRuntime,
		Message:  "Memo created outside component context",
		Detail:   "Memos must be created inside a component's render function.",
		DocURL:   "https://vango.dev/docs/errors/E003",
	},
	"E004": {
		Category: CategoryRuntime,
		Message:  "Signal set during render",
		Detail:   "Signal values should not be modified during component rendering. Use an Effect or event handler instead.",
		DocURL:   "https://vango.dev/docs/errors/E004",
	},
	"E005": {
		Category: CategoryRuntime,
		Message:  "Owner disposed",
		Detail:   "The component owner has been disposed. This usually means you're accessing signals from a component that has been unmounted.",
		DocURL:   "https://vango.dev/docs/errors/E005",
	},
	"E006": {
		Category: CategoryRuntime,
		Message:  "Circular dependency detected",
		Detail:   "A circular dependency was detected between reactive values. Check your signal and memo dependencies.",
		DocURL:   "https://vango.dev/docs/errors/E006",
	},
	"E007": {
		Category: CategoryRuntime,
		Message:  "Resource fetch failed",
		Detail:   "The resource fetcher returned an error.",
		DocURL:   "https://vango.dev/docs/errors/E007",
	},
	"E008": {
		Category: CategoryRuntime,
		Message:  "Invalid signal type",
		Detail:   "The signal type does not match the expected type.",
		DocURL:   "https://vango.dev/docs/errors/E008",
	},
	"E009": {
		Category: CategoryRuntime,
		Message:  "Handler not found",
		Detail:   "The event handler for this element was not found. The component may have re-rendered with different handlers.",
		DocURL:   "https://vango.dev/docs/errors/E009",
	},
	"E010": {
		Category: CategoryRuntime,
		Message:  "Session not found",
		Detail:   "The session ID is invalid or the session has expired.",
		DocURL:   "https://vango.dev/docs/errors/E010",
	},

	// ============================================
	// Hydration Errors (E040-E059)
	// ============================================

	"E040": {
		Category: CategoryHydration,
		Message:  "Hydration mismatch: element type differs",
		Detail:   "The server-rendered element type doesn't match what the client expected. This usually means the component renders differently on client vs server.",
		DocURL:   "https://vango.dev/docs/errors/E040",
	},
	"E041": {
		Category: CategoryHydration,
		Message:  "Hydration mismatch: text content differs",
		Detail:   "The server-rendered text doesn't match what the client expected. This can happen when using browser-only values like Date.now() during render.",
		DocURL:   "https://vango.dev/docs/errors/E041",
	},
	"E042": {
		Category: CategoryHydration,
		Message:  "Hydration mismatch: attribute differs",
		Detail:   "An attribute value differs between server and client rendering.",
		DocURL:   "https://vango.dev/docs/errors/E042",
	},
	"E043": {
		Category: CategoryHydration,
		Message:  "Hydration mismatch: missing element",
		Detail:   "An element exists on the server that wasn't expected by the client, or vice versa.",
		DocURL:   "https://vango.dev/docs/errors/E043",
	},
	"E044": {
		Category: CategoryHydration,
		Message:  "Hydration ID not found",
		Detail:   "The hydration ID referenced by an event doesn't exist in the DOM.",
		DocURL:   "https://vango.dev/docs/errors/E044",
	},

	// ============================================
	// Protocol Errors (E060-E079)
	// ============================================

	"E060": {
		Category: CategoryProtocol,
		Message:  "WebSocket connection failed",
		Detail:   "Unable to establish WebSocket connection to the server.",
		DocURL:   "https://vango.dev/docs/errors/E060",
	},
	"E061": {
		Category: CategoryProtocol,
		Message:  "Invalid message format",
		Detail:   "The received message could not be decoded. The protocol version may be mismatched.",
		DocURL:   "https://vango.dev/docs/errors/E061",
	},
	"E062": {
		Category: CategoryProtocol,
		Message:  "Unknown event type",
		Detail:   "The event type is not recognized by the server.",
		DocURL:   "https://vango.dev/docs/errors/E062",
	},
	"E063": {
		Category: CategoryProtocol,
		Message:  "Unknown patch type",
		Detail:   "The patch type is not recognized by the client.",
		DocURL:   "https://vango.dev/docs/errors/E063",
	},
	"E064": {
		Category: CategoryProtocol,
		Message:  "Protocol version mismatch",
		Detail:   "The client and server are using incompatible protocol versions.",
		DocURL:   "https://vango.dev/docs/errors/E064",
	},
	"E065": {
		Category: CategoryProtocol,
		Message:  "Message sequence error",
		Detail:   "Messages were received out of order. This may indicate network issues.",
		DocURL:   "https://vango.dev/docs/errors/E065",
	},
	"E066": {
		Category: CategoryProtocol,
		Message:  "Handshake failed",
		Detail:   "The WebSocket handshake with the server failed.",
		DocURL:   "https://vango.dev/docs/errors/E066",
	},

	// ============================================
	// Validation Errors (E080-E099)
	// ============================================

	"E080": {
		Category: CategoryValidation,
		Message:  "Required field missing",
		Detail:   "A required form field was not provided.",
		DocURL:   "https://vango.dev/docs/errors/E080",
	},
	"E081": {
		Category: CategoryValidation,
		Message:  "Invalid email format",
		Detail:   "The provided email address is not valid.",
		DocURL:   "https://vango.dev/docs/errors/E081",
	},
	"E082": {
		Category: CategoryValidation,
		Message:  "Value too short",
		Detail:   "The provided value is shorter than the minimum length.",
		DocURL:   "https://vango.dev/docs/errors/E082",
	},
	"E083": {
		Category: CategoryValidation,
		Message:  "Value too long",
		Detail:   "The provided value exceeds the maximum length.",
		DocURL:   "https://vango.dev/docs/errors/E083",
	},
	"E084": {
		Category: CategoryValidation,
		Message:  "Value out of range",
		Detail:   "The numeric value is outside the allowed range.",
		DocURL:   "https://vango.dev/docs/errors/E084",
	},
	"E085": {
		Category: CategoryValidation,
		Message:  "Pattern mismatch",
		Detail:   "The value doesn't match the required pattern.",
		DocURL:   "https://vango.dev/docs/errors/E085",
	},

	// ============================================
	// Routing Errors (E100-E119)
	// ============================================

	"E100": {
		Category: CategoryRuntime,
		Message:  "Route not found",
		Detail:   "No route matches the requested URL.",
		DocURL:   "https://vango.dev/docs/errors/E100",
	},
	"E101": {
		Category: CategoryRuntime,
		Message:  "Route parameter type mismatch",
		Detail:   "A route parameter couldn't be converted to the expected type.",
		DocURL:   "https://vango.dev/docs/errors/E101",
	},
	"E102": {
		Category: CategoryRuntime,
		Message:  "Missing route parameter",
		Detail:   "A required route parameter was not provided.",
		DocURL:   "https://vango.dev/docs/errors/E102",
	},
	"E103": {
		Category: CategoryCompile,
		Message:  "Invalid route file",
		Detail:   "The route file doesn't export a valid handler.",
		DocURL:   "https://vango.dev/docs/errors/E103",
	},
	"E104": {
		Category: CategoryCompile,
		Message:  "Duplicate route",
		Detail:   "Multiple route files resolve to the same URL pattern.",
		DocURL:   "https://vango.dev/docs/errors/E104",
	},

	// ============================================
	// Configuration Errors (E120-E139)
	// ============================================

	"E120": {
		Category: CategoryConfig,
		Message:  "Invalid vango.json",
		Detail:   "The vango.json configuration file is malformed.",
		DocURL:   "https://vango.dev/docs/errors/E120",
	},
	"E121": {
		Category: CategoryConfig,
		Message:  "Missing required configuration",
		Detail:   "A required configuration value is not set.",
		DocURL:   "https://vango.dev/docs/errors/E121",
	},
	"E122": {
		Category: CategoryConfig,
		Message:  "Invalid port number",
		Detail:   "The configured port number is invalid or already in use.",
		DocURL:   "https://vango.dev/docs/errors/E122",
	},
	"E123": {
		Category: CategoryConfig,
		Message:  "Tailwind not found",
		Detail:   "Tailwind CSS is enabled but not installed.",
		DocURL:   "https://vango.dev/docs/errors/E123",
	},

	// ============================================
	// CLI Errors (E140-E159)
	// ============================================

	"E140": {
		Category: CategoryCLI,
		Message:  "Project directory already exists",
		Detail:   "A directory with this name already exists.",
		DocURL:   "https://vango.dev/docs/errors/E140",
	},
	"E141": {
		Category: CategoryCLI,
		Message:  "Not a Vango project",
		Detail:   "The current directory is not a Vango project. Run this command from a directory with vango.json.",
		DocURL:   "https://vango.dev/docs/errors/E141",
	},
	"E142": {
		Category: CategoryCLI,
		Message:  "Build failed",
		Detail:   "The Go build command failed. Check the output for compiler errors.",
		DocURL:   "https://vango.dev/docs/errors/E142",
	},
	"E143": {
		Category: CategoryCLI,
		Message:  "Component not found",
		Detail:   "The requested component is not available in the registry.",
		DocURL:   "https://vango.dev/docs/errors/E143",
	},
	"E144": {
		Category: CategoryCLI,
		Message:  "Registry unavailable",
		Detail:   "Unable to connect to the component registry.",
		DocURL:   "https://vango.dev/docs/errors/E144",
	},
	"E145": {
		Category: CategoryCLI,
		Message:  "Invalid template",
		Detail:   "The specified project template doesn't exist.",
		DocURL:   "https://vango.dev/docs/errors/E145",
	},
	"E146": {
		Category: CategoryCLI,
		Message:  "Go not found",
		Detail:   "Go is not installed or not in PATH.",
		DocURL:   "https://vango.dev/docs/errors/E146",
	},
	"E147": {
		Category: CategoryCLI,
		Message:  "Invalid project name",
		Detail:   "Project names must be valid Go module names.",
		DocURL:   "https://vango.dev/docs/errors/E147",
	},

	// ============================================
	// Compile Errors (E160-E179)
	// ============================================

	"E160": {
		Category: CategoryCompile,
		Message:  "Code generation failed",
		Detail:   "Route or element code generation failed.",
		DocURL:   "https://vango.dev/docs/errors/E160",
	},
	"E161": {
		Category: CategoryCompile,
		Message:  "Invalid component signature",
		Detail:   "The component function has an invalid signature.",
		DocURL:   "https://vango.dev/docs/errors/E161",
	},
	"E162": {
		Category: CategoryCompile,
		Message:  "Missing import",
		Detail:   "A required package import is missing.",
		DocURL:   "https://vango.dev/docs/errors/E162",
	},
}

// GetAllCodes returns all registered error codes.
func GetAllCodes() []string {
	codes := make([]string, 0, len(registry))
	for code := range registry {
		codes = append(codes, code)
	}
	return codes
}

// GetTemplate returns the template for an error code.
func GetTemplate(code string) (ErrorTemplate, bool) {
	t, ok := registry[code]
	return t, ok
}

// Register adds a new error template to the registry.
func Register(code string, template ErrorTemplate) {
	registry[code] = template
}
