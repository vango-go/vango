package main

import (
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/vango-go/vango/internal/config"
)

func testCmd() *cobra.Command {
	var (
		coverage bool
		verbose  bool
		watch    bool
		race     bool
	)

	cmd := &cobra.Command{
		Use:   "test [packages...]",
		Short: "Run tests",
		Long: `Run tests for your Vango application.

This is a wrapper around 'go test' with convenient defaults
for Vango projects.

Examples:
  vango test
  vango test ./...
  vango test --coverage
  vango test --verbose
  vango test --race`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTest(args, coverage, verbose, watch, race)
		},
	}

	cmd.Flags().BoolVarP(&coverage, "coverage", "c", false, "Show coverage report")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	cmd.Flags().BoolVarP(&watch, "watch", "w", false, "Watch for changes and re-run")
	cmd.Flags().BoolVar(&race, "race", false, "Enable race detector")

	return cmd
}

func runTest(packages []string, coverage, verbose, watch, race bool) error {
	// Ensure we're in a Vango project
	_, err := config.LoadFromWorkingDir()
	if err != nil {
		// Allow running tests even without vango.json
		// Just run from current directory
	}

	// Default to ./...
	if len(packages) == 0 {
		packages = []string{"./..."}
	}

	// Build go test command
	args := []string{"test"}

	if verbose {
		args = append(args, "-v")
	}

	if coverage {
		args = append(args, "-cover")
	}

	if race {
		args = append(args, "-race")
	}

	args = append(args, packages...)

	// Run tests
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}
