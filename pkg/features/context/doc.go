// Package context provides a type-safe dependency injection mechanism for Vango applications.
//
// It allows passing data through the component tree without manually passing props
// at every level (prop drilling). Features include:
//
//   - Generic, type-safe Context[T]
//   - Provider components for value injection
//   - Default values
//   - Nestable providers
//
// Usage:
//
//	// Define context
//	var ThemeContext = context.Create("light")
//
//	// Provide value
//	func App() *vdom.VNode {
//	    return ThemeContext.Provider("dark",
//	        MainContent(),
//	    )
//	}
//
//	// Consume value
//	func MainContent() *vdom.VNode {
//	    theme := ThemeContext.Use()
//	    return vdom.Div(vdom.Text("Current theme: " + theme))
//	}
package context
