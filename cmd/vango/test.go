package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/vango-go/vango/internal/config"
	"github.com/vango-go/vango/internal/dev"
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
	// Ensure we're in a Vango project if possible.
	cfg, _ := config.LoadFromWorkingDir()

	// Default to ./...
	if len(packages) == 0 {
		packages = []string{"./..."}
	}

	// Build go test command
	args := buildTestArgs(packages, coverage, verbose, race)

	if !watch {
		return runGoTest(args)
	}

	return runTestWatch(cfg, args)
}

func buildTestArgs(packages []string, coverage, verbose, race bool) []string {
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
	return args
}

func runGoTest(args []string) error {
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func runTestWatch(cfg *config.Config, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	watchPaths := []string{"."}
	ignore := dev.DefaultIgnore
	if cfg != nil {
		watchPaths = dev.CollectWatchPaths(cfg)
		ignore = append(ignore, cfg.Dev.Ignore...)
	}

	watcher := dev.NewWatcher(dev.WatcherConfig{
		Paths:    watchPaths,
		Ignore:   ignore,
		Debounce: 150 * time.Millisecond,
	})

	trigger := make(chan struct{}, 1)
	trigger <- struct{}{}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-trigger:
				fmt.Println("  Running tests...")
				start := time.Now()
				if err := runGoTest(args); err != nil {
					warn("Tests failed (%s)", time.Since(start).Round(time.Millisecond))
				} else {
					success("Tests passed (%s)", time.Since(start).Round(time.Millisecond))
				}
				fmt.Println("  Watching for changes...")
			}
		}
	}()

	watcher.OnChange(func(change dev.Change) {
		fmt.Printf("[%s] Changed: %s\n", time.Now().Format("15:04:05"), filepath.ToSlash(change.Path))
		select {
		case trigger <- struct{}{}:
		default:
		}
	})

	fmt.Println("  Watching for changes...")
	go watcher.Start(ctx)
	<-ctx.Done()
	watcher.Stop()
	return nil
}
