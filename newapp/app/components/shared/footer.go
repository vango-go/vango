package shared

import (
	"github.com/vango-go/vango"
	. "github.com/vango-go/vango/el"
)

func AppFooter() *vango.VNode {
	return Footer(Class("border-t border-gray-200 dark:border-gray-700 mt-auto"),
		Div(Class("max-w-5xl mx-auto px-5 py-4 text-sm text-gray-500 dark:text-gray-400"),
			Text("© 2026 newapp"),
		),
	)
}
