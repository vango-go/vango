package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/vango-dev/vango/v2/internal/config"
	"github.com/vango-dev/vango/v2/internal/registry"
)

var addPath string

func addCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add [command]",
		Short: "Add VangoUI components",
		Long: `Add VangoUI components to your project.

Components are copied to your project as source code that you own.
You can modify them as needed.

Commands:
  init       Initialize VangoUI in your project
  upgrade    Upgrade installed components
  list       List all available components

Examples:
  vango add init
  vango add button dialog
  vango add button --path=app/components/custom
  vango add upgrade
  vango add list`,
	}

	cmd.AddCommand(
		addInitCmd(),
		addUpgradeCmd(),
		addListCmd(),
	)

	cmd.PersistentFlags().StringVar(&addPath, "path", "", "Override component output path")

	// Handle direct component installation
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return runAddComponents(args, addPath)
	}

	return cmd
}

func addInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize VangoUI",
		Long: `Initialize VangoUI in your project.

This creates the base files needed for VangoUI components:
  • utils.go - Utility functions (CN, etc.)
  • base.go  - Base component configuration`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAddInit()
		},
	}
}

func addUpgradeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade installed components",
		Long: `Upgrade all installed VangoUI components to the latest version.

Components with local modifications will be skipped.
Use --force to overwrite modified components.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAddUpgrade()
		},
	}
}

func runAddInit() error {
	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return err
	}

	fmt.Println("  Initializing VangoUI...")
	fmt.Println()

	reg := registry.New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	if err := reg.Init(ctx); err != nil {
		return err
	}

	success("Created %s/utils.go", cfg.UI.Path)
	success("Created %s/base.go", cfg.UI.Path)

	fmt.Println()
	fmt.Println("  Ready! Add components with:")
	fmt.Println()
	fmt.Println("    vango add button card dialog")
	fmt.Println()

	return nil
}

func addListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all available components",
		Long:  `List all components available in the VangoUI registry.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAddList()
		},
	}
}

func runAddList() error {
	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return err
	}

	reg := registry.New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	manifest, err := reg.FetchManifest(ctx)
	if err != nil {
		return err
	}

	fmt.Println("  Available VangoUI Components:")
	fmt.Println()

	// Get installed components for status
	installed, _ := reg.ListInstalled()
	installedMap := make(map[string]bool)
	for _, ic := range installed {
		installedMap[ic.Name] = true
	}

	// Sort component names for consistent output
	var names []string
	for name := range manifest.Components {
		names = append(names, name)
	}
	sortStrings(names)

	for _, name := range names {
		comp := manifest.Components[name]
		if comp.Internal {
			continue // Skip internal components
		}

		status := "    "
		if installedMap[name] {
			status = " ✓  "
		}

		deps := ""
		if len(comp.DependsOn) > 0 {
			deps = fmt.Sprintf(" (requires: %s)", joinStrings(comp.DependsOn, ", "))
		}

		fmt.Printf("%s%s%s\n", status, name, deps)
	}

	fmt.Println()
	fmt.Printf("  Registry version: %s\n", manifest.Version)
	fmt.Println()

	return nil
}

func runAddComponents(components []string, overridePath string) error {
	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return err
	}

	// Override path if specified
	if overridePath != "" {
		cfg.UI.Path = overridePath
	}

	fmt.Println("  Installing components...")
	fmt.Println()

	reg := registry.New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Fetch manifest to show dependency resolution
	manifest, err := reg.FetchManifest(ctx)
	if err != nil {
		return err
	}

	// Show what will be installed
	info("Resolving dependencies...")
	for _, comp := range components {
		if c, ok := manifest.Components[comp]; ok {
			if len(c.DependsOn) > 0 {
				fmt.Printf("    %s → [%s]\n", comp, joinStrings(c.DependsOn, ", "))
			} else {
				fmt.Printf("    %s\n", comp)
			}
		}
	}
	fmt.Println()

	// Install
	if err := reg.Install(ctx, components); err != nil {
		return err
	}

	fmt.Println()
	success("Installed components to %s/", cfg.UI.Path)
	fmt.Println()

	return nil
}

func runAddUpgrade() error {
	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return err
	}

	fmt.Println("  Checking for updates...")
	fmt.Println()

	reg := registry.New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Get current and latest versions
	manifest, err := reg.FetchManifest(ctx)
	if err != nil {
		return err
	}

	currentVersion := cfg.UI.Version
	latestVersion := manifest.Version

	fmt.Printf("    Current: %s\n", currentVersion)
	fmt.Printf("    Latest:  %s\n", latestVersion)
	fmt.Println()

	if currentVersion == latestVersion {
		info("Already up to date!")
		return nil
	}

	// List installed components
	installed, err := reg.ListInstalled()
	if err != nil {
		return err
	}

	if len(installed) == 0 {
		info("No components installed")
		return nil
	}

	// Show what will be upgraded
	var toUpgrade, skipped []string
	for _, comp := range installed {
		if comp.Modified {
			skipped = append(skipped, comp.Name)
		} else {
			toUpgrade = append(toUpgrade, comp.Name)
		}
	}

	if len(skipped) > 0 {
		warn("Skipping modified: %s", joinStrings(skipped, ", "))
	}

	if len(toUpgrade) == 0 {
		info("No components to upgrade")
		return nil
	}

	info("Upgrading: %s", joinStrings(toUpgrade, ", "))
	fmt.Println()

	// Perform upgrade
	if err := reg.Upgrade(ctx); err != nil {
		return err
	}

	success("Upgraded %d components", len(toUpgrade))
	return nil
}

func joinStrings(s []string, sep string) string {
	if len(s) == 0 {
		return ""
	}
	result := s[0]
	for i := 1; i < len(s); i++ {
		result += sep + s[i]
	}
	return result
}

func sortStrings(s []string) {
	for i := 0; i < len(s); i++ {
		for j := i + 1; j < len(s); j++ {
			if s[i] > s[j] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}
