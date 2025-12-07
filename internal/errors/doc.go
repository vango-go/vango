// Package errors provides structured, actionable error messages for Vango.
//
// The errors package implements a comprehensive error system that:
//   - Shows exact source locations (file, line, column)
//   - Explains what went wrong in plain language
//   - Suggests how to fix issues with code examples
//   - Links to documentation for deeper understanding
//
// # Error Categories
//
// Errors are organized into categories:
//   - compile: Build-time errors (type mismatches, missing imports)
//   - runtime: Execution errors (signal read outside component, nil pointer)
//   - hydration: Server/client mismatch errors
//   - protocol: Wire protocol errors (invalid messages, connection issues)
//   - validation: User input errors (form validation, route parameters)
//
// # Error Codes
//
// Each error has a unique code (e.g., "E001") that maps to:
//   - A short message describing the error
//   - A detailed explanation
//   - A documentation URL
//
// # Usage
//
//	err := errors.New("E001").
//	    WithLocation("app/routes/index.go", 15, 12).
//	    WithSuggestion("Wrap your component logic in vango.Func()")
//
//	fmt.Println(err.Format())
//	// Output:
//	// ERROR E001: Signal read outside component context
//	//
//	//   app/routes/index.go:15:12
//	//
//	//     13 │ func HomePage() *vango.VNode {
//	//     14 │     count := vango.Signal(0)
//	//   → 15 │     value := count()
//	//        │             ^
//	//     16 │     return Div(Text(fmt.Sprintf("%d", value)))
//	//     17 │ }
//	//
//	//   Hint: Wrap your component logic in vango.Func(func() *vango.VNode { ... })
//	//
//	//   Learn more: https://vango.dev/docs/errors/E001
package errors
