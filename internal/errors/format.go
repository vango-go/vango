package errors

import (
	"fmt"
	"os"
	"strings"
)

// ANSI color codes for terminal output.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

// colorEnabled controls whether ANSI colors are used.
var colorEnabled = true

// DisableColors disables ANSI color output.
func DisableColors() {
	colorEnabled = false
}

// EnableColors enables ANSI color output.
func EnableColors() {
	colorEnabled = true
}

// color wraps text in ANSI color codes if colors are enabled.
func color(code, text string) string {
	if !colorEnabled {
		return text
	}
	return code + text + colorReset
}

func red(text string) string    { return color(colorRed, text) }
func green(text string) string  { return color(colorGreen, text) }
func yellow(text string) string { return color(colorYellow, text) }
func blue(text string) string   { return color(colorBlue, text) }
func cyan(text string) string   { return color(colorCyan, text) }
func white(text string) string  { return color(colorWhite, text) }
func gray(text string) string   { return color(colorGray, text) }
func bold(text string) string   { return color(colorBold, text) }

// Format returns a beautifully formatted error message for terminal display.
func (e *VangoError) Format() string {
	var b strings.Builder

	// Header line
	b.WriteString("\n")
	if e.Code != "" {
		b.WriteString(red(bold("ERROR ")))
		b.WriteString(white(bold(e.Code + ": ")))
		b.WriteString(white(e.Message))
	} else {
		b.WriteString(red(bold("ERROR: ")))
		b.WriteString(white(e.Message))
	}
	b.WriteString("\n\n")

	// Location
	if e.Location != nil {
		b.WriteString("  ")
		b.WriteString(cyan(e.Location.String()))
		b.WriteString("\n\n")

		// Context with line numbers and arrow
		if len(e.Context) > 0 {
			startLine := e.Location.Line - len(e.Context)/2
			for i, line := range e.Context {
				lineNum := startLine + i
				if lineNum == e.Location.Line {
					// Highlighted line with arrow
					b.WriteString("  ")
					b.WriteString(red("→ "))
					b.WriteString(fmt.Sprintf("%4d", lineNum))
					b.WriteString(gray(" │ "))
					b.WriteString(line)
					b.WriteString("\n")

					// Column indicator
					if e.Location.Column > 0 {
						b.WriteString("       ")
						b.WriteString(gray("│ "))
						b.WriteString(strings.Repeat(" ", e.Location.Column-1))
						b.WriteString(red("^"))
						b.WriteString("\n")
					}
				} else {
					// Normal line
					b.WriteString("    ")
					b.WriteString(fmt.Sprintf("%4d", lineNum))
					b.WriteString(gray(" │ "))
					b.WriteString(line)
					b.WriteString("\n")
				}
			}
			b.WriteString("\n")
		}
	}

	// Detail
	if e.Detail != "" {
		for _, line := range wrapText(e.Detail, 70) {
			b.WriteString("  ")
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Suggestion
	if e.Suggestion != "" {
		b.WriteString("  ")
		b.WriteString(cyan("Hint: "))
		b.WriteString(e.Suggestion)
		b.WriteString("\n\n")
	}

	// Example
	if e.Example != "" {
		b.WriteString("  ")
		b.WriteString(cyan("Example:"))
		b.WriteString("\n")
		for _, line := range strings.Split(e.Example, "\n") {
			b.WriteString("    ")
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Doc URL
	if e.DocURL != "" {
		b.WriteString("  ")
		b.WriteString(gray("Learn more: "))
		b.WriteString(blue(e.DocURL))
		b.WriteString("\n")
	}

	return b.String()
}

// FormatCompact returns a compact single-line error format.
func (e *VangoError) FormatCompact() string {
	var b strings.Builder

	if e.Location != nil {
		b.WriteString(e.Location.String())
		b.WriteString(": ")
	}

	if e.Code != "" {
		b.WriteString(e.Code)
		b.WriteString(": ")
	}

	b.WriteString(e.Message)

	return b.String()
}

// FormatJSON returns the error as a JSON object.
func (e *VangoError) FormatJSON() string {
	var b strings.Builder
	b.WriteString("{")

	if e.Code != "" {
		b.WriteString(fmt.Sprintf(`"code":%q,`, e.Code))
	}
	b.WriteString(fmt.Sprintf(`"category":%q,`, e.Category))
	b.WriteString(fmt.Sprintf(`"message":%q`, e.Message))

	if e.Detail != "" {
		b.WriteString(fmt.Sprintf(`,"detail":%q`, e.Detail))
	}
	if e.Location != nil {
		b.WriteString(fmt.Sprintf(`,"location":{"file":%q,"line":%d,"column":%d}`,
			e.Location.File, e.Location.Line, e.Location.Column))
	}
	if e.Suggestion != "" {
		b.WriteString(fmt.Sprintf(`,"suggestion":%q`, e.Suggestion))
	}
	if e.DocURL != "" {
		b.WriteString(fmt.Sprintf(`,"docUrl":%q`, e.DocURL))
	}

	b.WriteString("}")
	return b.String()
}

// wrapText wraps text to the specified width.
func wrapText(text string, width int) []string {
	if text == "" {
		return nil
	}
	if len(text) <= width {
		return []string{text}
	}

	var lines []string
	words := strings.Fields(text)
	var current strings.Builder

	for _, word := range words {
		if current.Len()+len(word)+1 > width {
			if current.Len() > 0 {
				lines = append(lines, current.String())
				current.Reset()
			}
		}
		if current.Len() > 0 {
			current.WriteString(" ")
		}
		current.WriteString(word)
	}

	if current.Len() > 0 {
		lines = append(lines, current.String())
	}

	return lines
}

// PrintError prints a formatted error to stderr.
func PrintError(err error) {
	if ve, ok := err.(*VangoError); ok {
		fmt.Fprint(os.Stderr, ve.Format())
	} else {
		fmt.Fprintf(os.Stderr, "\n%sERROR:%s %s\n\n", colorRed+colorBold, colorReset, err.Error())
	}
}

