package routes

import (
	"github.com/vango-go/vango"
	. "github.com/vango-go/vango/el"

	"newapp/app/components/shared"
)

func Layout(ctx vango.Ctx, children vango.Slot) *vango.VNode {
	return Html(Lang("en"), Class(""),
		Head(
			Meta(Charset("utf-8")),
			Meta(Name("viewport"), Content("width=device-width, initial-scale=1")),
			Meta(Name("color-scheme"), Content("light dark")),
			Title(Text("newapp")),
			LinkEl(Rel("stylesheet"), Href(ctx.Asset("styles.css"))),
			// Theme initialization - runs before page renders to prevent flash
			Script(Raw(`(function(){var t=localStorage.getItem('theme');if(t==='dark'||(!t&&window.matchMedia('(prefers-color-scheme:dark)').matches)){document.documentElement.classList.add('dark')}})();`)),
		),
		Body(Class("bg-white dark:bg-[#091D39] text-gray-900 dark:text-white min-h-screen transition-colors"),
			shared.Navbar(ctx),
			Main(Class("max-w-5xl mx-auto px-5 py-8"), children),
			shared.AppFooter(),
			// Theme toggle script - attaches click handler
			Script(Raw(`document.getElementById('theme-toggle').onclick=function(){document.documentElement.classList.toggle('dark');localStorage.setItem('theme',document.documentElement.classList.contains('dark')?'dark':'light')};`)),
			VangoScripts(),
		),
	)
}
