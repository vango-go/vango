package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/vango-go/vango/internal/config"
	"github.com/vango-go/vango/internal/dev"
)

func devCmd() *cobra.Command {
	var (
		port        int
		host        string
		openBrowser bool
	)

	cmd := &cobra.Command{
		Use:   "dev",
		Short: "Start the development server",
		Long: `Start the development server with hot reload.

The dev server watches for file changes, recompiles, and
automatically refreshes connected browsers.

Features:
  • Hot reload on file change
  • Error overlay in browser
  • Tailwind CSS watch mode (if enabled)
  • Proxy support for external APIs

Examples:
  vango dev
  vango dev --port=8080
  vango dev --host=0.0.0.0`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDev(port, host, openBrowser)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 0, "Port to run on (default from vango.json)")
	cmd.Flags().StringVarP(&host, "host", "H", "", "Host to bind to (default from vango.json)")
	cmd.Flags().BoolVarP(&openBrowser, "open", "o", false, "Open browser on start")

	return cmd
}

func runDev(port int, host string, openBrowser bool) error {
	// Check for Go
	if _, err := exec.LookPath("go"); err != nil {
		errorMsg("Go is not installed or not in PATH")
		info("Install Go from https://go.dev/dl/")
		return err
	}

	// Load config
	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return err
	}

	// Apply command-line overrides
	if port > 0 {
		cfg.Dev.Port = port
	}
	if host != "" {
		cfg.Dev.Host = host
	}
	if openBrowser {
		cfg.Dev.OpenBrowser = true
	}

	// Print banner
	printBanner()
	fmt.Println("  dev")
	fmt.Println()

	// Create server
	server := dev.NewServer(dev.ServerOptions{
		Config:  cfg,
		Verbose: true,
		OnBuildStart: func() {
			// Build starting
		},
		OnBuildComplete: func(result dev.BuildResult) {
			if result.Success {
				success("Built in %s", result.Duration.Round(1000000))
			}
		},
		OnReload: func(clients int) {
			success("Reloaded %d browsers", clients)
		},
	})

	// Handle signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\n\n  Shutting down...")
		cancel()
		server.Stop()
	}()

	// Open browser if requested
	if cfg.Dev.OpenBrowser {
		go func() {
			// Wait a bit for server to start
			// Note: In a real implementation, we'd wait for the server to be ready
			openURL(cfg.DevURL())
		}()
	}

	// Start server
	return server.Start(ctx)
}

// openURL opens a URL in the default browser.
func openURL(url string) {
	var cmd *exec.Cmd

	switch {
	case commandExists("xdg-open"):
		cmd = exec.Command("xdg-open", url)
	case commandExists("open"):
		cmd = exec.Command("open", url)
	case commandExists("start"):
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return
	}

	cmd.Start()
}

// commandExists checks if a command exists in PATH.
func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
