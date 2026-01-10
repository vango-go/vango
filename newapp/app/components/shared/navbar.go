package shared

import (
	"github.com/vango-go/vango"
	. "github.com/vango-go/vango/el"
)

func Navbar(ctx vango.Ctx) *vango.VNode {
	return Header(Class("border-b border-gray-200 dark:border-gray-700"),
		Div(Class("max-w-5xl mx-auto px-5 py-4 flex items-center justify-between"),
			// Logo and brand
			Div(Class("flex items-center gap-3"),
				Img(Src("/logo.svg"), Alt("Vango"), Class("h-8 w-8")),
				Link("/", Class("font-bold text-lg hover:opacity-80 transition-opacity"), Text("newapp")),
			),
			// Navigation
			Div(Class("flex items-center gap-6"),
				Nav(Class("flex items-center gap-4"),
					NavLink(ctx, "/", Text("Home")),
					NavLink(ctx, "/about", Text("About")),
				),
				ThemeToggle(),
			),
		),
	)
}

func ThemeToggle() *vango.VNode {
	return Button(
		ID("theme-toggle"),
		Class("p-2 text-gray-600 dark:text-gray-300 hover:text-gray-900 dark:hover:text-white transition-colors"),
		AriaLabel("Toggle theme"),
		// Sun icon (shown in dark mode)
		Span(Class("hidden dark:block"),
			Raw(`<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>`),
		),
		// Moon icon (shown in light mode)
		Span(Class("block dark:hidden"),
			Raw(`<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>`),
		),
	)
}
