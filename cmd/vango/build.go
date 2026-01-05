package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/vango-go/vango/internal/build"
	"github.com/vango-go/vango/internal/config"
)

func buildCmd() *cobra.Command {
	var (
		output     string
		minify     bool
		sourceMaps bool
		target     string
		clean      bool
	)

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build for production",
		Long: `Build the application for production deployment.

This command:
  • Compiles Go binary with optimizations
  • Bundles and minifies the thin client
  • Compiles Tailwind CSS (if enabled)
  • Copies static assets with cache busting
  • Generates asset manifest

Examples:
  vango build
  vango build --output=dist
  vango build --target=linux/amd64`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBuild(output, minify, sourceMaps, target, clean)
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "Output directory (default from vango.json)")
	cmd.Flags().BoolVar(&minify, "minify", true, "Minify output")
	cmd.Flags().BoolVar(&sourceMaps, "sourcemaps", false, "Generate source maps")
	cmd.Flags().StringVar(&target, "target", "", "Build target (e.g., linux/amd64)")
	cmd.Flags().BoolVar(&clean, "clean", false, "Clean output directory before build")

	return cmd
}

func runBuild(output string, minify, sourceMaps bool, target string, clean bool) error {
	// Load config
	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return err
	}

	// Apply command-line overrides
	if output != "" {
		cfg.Build.Output = output
	}

	fmt.Println("  Building for production...")
	fmt.Println()

	// Create builder
	builder := build.New(cfg, build.Options{
		Minify:     minify,
		SourceMaps: sourceMaps,
		Target:     target,
		OnProgress: func(step string) {
			info(step)
		},
	})

	// Clean if requested
	if clean {
		info("Cleaning output directory...")
		builder.Clean()
	}

	// Handle signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		cancel()
	}()

	// Build
	result, err := builder.Build(ctx)
	if err != nil {
		return err
	}

	// Print results
	fmt.Println()
	success("Build complete in %s", result.Duration.Round(1000000))
	fmt.Println()
	fmt.Println("  Output:")
	fmt.Printf("    %s/\n", cfg.Build.Output)
	fmt.Printf("    ├── server          # Go binary\n")
	fmt.Printf("    ├── public/\n")
	fmt.Printf("    │   ├── vango.min.js  (%s)\n", formatBytes(result.ClientSize))
	if result.CSSSize > 0 {
		fmt.Printf("    │   ├── styles.css    (%s)\n", formatBytes(result.CSSSize))
	}
	fmt.Printf("    │   └── assets/\n")
	fmt.Printf("    └── manifest.json\n")
	fmt.Println()
	fmt.Println("  To run:")
	fmt.Printf("    ./%s/server\n", cfg.Build.Output)
	fmt.Println()

	return nil
}

// formatBytes formats bytes as a human-readable string.
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
