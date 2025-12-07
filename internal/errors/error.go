package errors

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Category represents the type of error.
type Category string

const (
	CategoryCompile    Category = "compile"
	CategoryRuntime    Category = "runtime"
	CategoryHydration  Category = "hydration"
	CategoryProtocol   Category = "protocol"
	CategoryValidation Category = "validation"
	CategoryConfig     Category = "config"
	CategoryCLI        Category = "cli"
)

// Location represents a source code location.
type Location struct {
	File   string
	Line   int
	Column int
}

// String returns the location as a formatted string.
func (l *Location) String() string {
	if l == nil {
		return ""
	}
	if l.Column > 0 {
		return fmt.Sprintf("%s:%d:%d", l.File, l.Line, l.Column)
	}
	return fmt.Sprintf("%s:%d", l.File, l.Line)
}

// VangoError is a structured error with source location, suggestions, and documentation.
type VangoError struct {
	// Code is a unique error identifier (e.g., "E001").
	Code string

	// Category is the error type (compile, runtime, etc.).
	Category Category

	// Message is a short description of the error.
	Message string

	// Detail is a longer explanation of the error.
	Detail string

	// Location is the source code location where the error occurred.
	Location *Location

	// Context contains surrounding source code lines.
	Context []string

	// Suggestion is a hint on how to fix the error.
	Suggestion string

	// Example is code showing the correct approach.
	Example string

	// DocURL is a link to documentation about this error.
	DocURL string

	// Wrapped is the underlying error, if any.
	Wrapped error
}

// Error implements the error interface.
func (e *VangoError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return e.Message
}

// Unwrap returns the wrapped error for errors.Is/As support.
func (e *VangoError) Unwrap() error {
	return e.Wrapped
}

// WithLocation adds source location to the error.
func (e *VangoError) WithLocation(file string, line, column int) *VangoError {
	e.Location = &Location{File: file, Line: line, Column: column}
	e.Context = readContextLines(file, line, 5)
	return e
}

// WithLocationFromError extracts location from a Go compiler error.
func (e *VangoError) WithLocationFromError(err error) *VangoError {
	// Parse Go compiler error format: "file.go:line:column: message"
	if err == nil {
		return e
	}
	msg := err.Error()
	parts := strings.SplitN(msg, ":", 4)
	if len(parts) >= 3 {
		var line, col int
		fmt.Sscanf(parts[1], "%d", &line)
		fmt.Sscanf(parts[2], "%d", &col)
		if line > 0 {
			e.Location = &Location{File: parts[0], Line: line, Column: col}
			e.Context = readContextLines(parts[0], line, 5)
		}
	}
	return e
}

// WithSuggestion adds a fix suggestion to the error.
func (e *VangoError) WithSuggestion(s string) *VangoError {
	e.Suggestion = s
	return e
}

// WithExample adds a code example to the error.
func (e *VangoError) WithExample(ex string) *VangoError {
	e.Example = ex
	return e
}

// WithDetail adds a detailed explanation to the error.
func (e *VangoError) WithDetail(d string) *VangoError {
	e.Detail = d
	return e
}

// WithContext adds custom context lines to the error.
func (e *VangoError) WithContext(lines []string) *VangoError {
	e.Context = lines
	return e
}

// Wrap wraps another error.
func (e *VangoError) Wrap(err error) *VangoError {
	e.Wrapped = err
	return e
}

// readContextLines reads lines around the specified line number from a file.
func readContextLines(filename string, targetLine, contextSize int) []string {
	file, err := os.Open(filename)
	if err != nil {
		return nil
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	lineNum := 0
	startLine := targetLine - contextSize/2
	endLine := targetLine + contextSize/2

	for scanner.Scan() {
		lineNum++
		if lineNum >= startLine && lineNum <= endLine {
			lines = append(lines, scanner.Text())
		}
		if lineNum > endLine {
			break
		}
	}

	return lines
}

// New creates a VangoError from a registered error code.
func New(code string) *VangoError {
	template, ok := registry[code]
	if !ok {
		return &VangoError{
			Code:    code,
			Message: "Unknown error",
		}
	}
	return &VangoError{
		Code:     code,
		Category: template.Category,
		Message:  template.Message,
		Detail:   template.Detail,
		DocURL:   template.DocURL,
	}
}

// Newf creates a new VangoError with a formatted message (no code).
func Newf(category Category, format string, args ...any) *VangoError {
	return &VangoError{
		Category: category,
		Message:  fmt.Sprintf(format, args...),
	}
}

// FromError wraps a standard error in a VangoError.
func FromError(err error, code string) *VangoError {
	if err == nil {
		return nil
	}
	if ve, ok := err.(*VangoError); ok {
		return ve
	}
	return New(code).Wrap(err)
}
