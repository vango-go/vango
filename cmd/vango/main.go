package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version information set at build time.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

const banner = `
  ╦  ╦┌─┐┌┐┌┌─┐┌─┐
  ╚╗╔╝├─┤│││├─┤│ │
   ╚╝ ┴ ┴┘└┘┴ ┴└─┘
`

func main() {
	rootCmd := &cobra.Command{
		Use:   "vango",
		Short: "The Go framework for modern web applications",
		Long: `Vango is a server-driven web framework for Go.

Build fast, interactive web applications with Go on the server
and a thin JavaScript client. Features include:

  • Server-side components with reactive state
  • Real-time updates via WebSocket
  • SSR with hydration
  • File-based routing
  • Hot reload development server
  • < 15KB JavaScript client`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Add commands
	rootCmd.AddCommand(
		createCmd(),
		devCmd(),
		buildCmd(),
		testCmd(),
		genCmd(),
		addCmd(),
		versionCmd(),
	)

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "\033[31mError:\033[0m %s\n", err)
		os.Exit(1)
	}
}

// printBanner prints the Vango ASCII art banner.
func printBanner() {
	fmt.Print(banner)
}

// success prints a success message.
func success(format string, args ...any) {
	fmt.Printf("\033[32m✓\033[0m %s\n", fmt.Sprintf(format, args...))
}

// info prints an info message.
func info(format string, args ...any) {
	fmt.Printf("  %s\n", fmt.Sprintf(format, args...))
}

// warn prints a warning message.
func warn(format string, args ...any) {
	fmt.Printf("\033[33m⚠\033[0m %s\n", fmt.Sprintf(format, args...))
}

// errorMsg prints an error message.
func errorMsg(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "\033[31m✗\033[0m %s\n", fmt.Sprintf(format, args...))
}
