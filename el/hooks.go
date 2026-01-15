package el

import "github.com/vango-go/vango"

// Hook attaches a client hook to an element.
func Hook(name string, config any) Attr {
	return vango.Hook(name, config)
}

// OnEvent attaches a hook event handler to an element.
func OnEvent(name string, handler func(vango.HookEvent)) Attr {
	return vango.OnEvent(name, handler)
}

