package routes

import (
	"github.com/vango-go/vango"
	. "github.com/vango-go/vango/el"
)

func IndexPage(ctx vango.Ctx) *vango.VNode {
	count := vango.NewSignal(0)

	return Div(Class("space-y-6"),
		H1(Class("text-3xl font-bold"), Text("Welcome to Vango")),
		P(Class("text-gray-600 dark:text-gray-400"), Text("Build modern web apps with Go.")),

		// Interactive counter demo
		Div(Class("flex items-center gap-4 py-6"),
			Button(
				Class("w-10 h-10 text-xl font-semibold border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"),
				OnClick(count.Dec),
				Text("-"),
			),
			Span(Class("text-2xl font-semibold min-w-12 text-center"), Textf("%d", count.Get())),
			Button(
				Class("w-10 h-10 text-xl font-semibold border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"),
				OnClick(count.Inc),
				Text("+"),
			),
		),

		P(Class("text-gray-600 dark:text-gray-400"),
			Text("Try the "),
			Link("/about", Class("text-blue-600 dark:text-blue-400 hover:underline"), Text("About page")),
			Text("."),
		),
	)
}
