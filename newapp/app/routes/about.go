package routes

import (
	"github.com/vango-go/vango"
	. "github.com/vango-go/vango/el"
)

func AboutPage(ctx vango.Ctx) *vango.VNode {
	return Div(Class("space-y-4"),
		H1(Class("text-3xl font-bold"), Text("About")),
		P(Class("text-gray-600 dark:text-gray-400"), Text("This is a Vango app scaffolded by vango create.")),
	)
}
