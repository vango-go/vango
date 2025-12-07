package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vango-dev/vango/v2/internal/errors"
	"github.com/vango-dev/vango/v2/internal/templates"
)

func createCmd() *cobra.Command {
	var (
		template    string
		description string
		tailwind    bool
		skipPrompts bool
	)

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new Vango project",
		Long: `Create a new Vango project with the specified name.

Templates:
  minimal   Just the essentials for a Vango app
  full      Complete starter with example pages and components (default)
  api       API-only project without UI

Examples:
  vango create my-app
  vango create my-app --template=minimal
  vango create my-api --template=api`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			return runCreate(name, template, description, tailwind, skipPrompts)
		},
	}

	cmd.Flags().StringVarP(&template, "template", "t", "full", "Project template (minimal, full, api)")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Project description")
	cmd.Flags().BoolVar(&tailwind, "tailwind", true, "Include Tailwind CSS")
	cmd.Flags().BoolVarP(&skipPrompts, "yes", "y", false, "Skip prompts and use defaults")

	return cmd
}

func runCreate(name, templateName, description string, tailwind, skipPrompts bool) error {
	printBanner()
	fmt.Println("  Creating a new Vango project...")
	fmt.Println()

	// Validate project name
	if !isValidProjectName(name) {
		return errors.New("E147").
			WithDetail("Project name must be a valid Go module name").
			WithSuggestion("Use lowercase letters, numbers, and hyphens")
	}

	// Check if directory exists
	projectDir, err := filepath.Abs(name)
	if err != nil {
		return err
	}

	if _, err := os.Stat(projectDir); !os.IsNotExist(err) {
		return errors.New("E140").
			WithDetail("Directory '" + name + "' already exists").
			WithSuggestion("Choose a different name or remove the existing directory")
	}

	// Interactive prompts if not skipped
	if !skipPrompts {
		var err error
		description, tailwind, err = promptForConfig(name, description, tailwind)
		if err != nil {
			return err
		}
	}

	// Set defaults
	if description == "" {
		description = "A Vango web application"
	}

	// Get template
	tmpl, err := templates.Get(templateName)
	if err != nil {
		return err
	}

	// Create project directory
	info("Creating project directory...")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return err
	}

	// Create project from template
	config := templates.Config{
		ProjectName: name,
		ModulePath:  name, // Simple module path for local projects
		Description: description,
		HasTailwind: tailwind && templateName != "api",
	}

	info("Creating project from '%s' template...", templateName)
	if err := tmpl.Create(projectDir, config); err != nil {
		// Clean up on error
		os.RemoveAll(projectDir)
		return err
	}

	// Initialize go.mod
	info("Initializing Go module...")
	if err := initGoMod(projectDir, name); err != nil {
		return err
	}

	// Run go mod tidy
	info("Installing dependencies...")
	if err := goModTidy(projectDir); err != nil {
		warn("Could not run 'go mod tidy': %v", err)
	}

	// Initialize Tailwind if enabled
	if config.HasTailwind {
		info("Setting up Tailwind CSS...")
		if err := setupTailwind(projectDir); err != nil {
			warn("Could not set up Tailwind: %v", err)
		}
	}

	// Print success message
	fmt.Println()
	success("Created %s/", name)
	fmt.Println()
	fmt.Println("  To get started:")
	fmt.Println()
	fmt.Printf("    cd %s\n", name)
	fmt.Println("    vango dev")
	fmt.Println()
	fmt.Printf("  Your app will be running at http://localhost:3000\n")
	fmt.Println()

	return nil
}

func promptForConfig(name, description string, tailwind bool) (string, bool, error) {
	reader := bufio.NewReader(os.Stdin)

	// Description
	if description == "" {
		fmt.Printf("? Description: ")
		desc, err := reader.ReadString('\n')
		if err != nil {
			return "", false, err
		}
		description = strings.TrimSpace(desc)
		if description == "" {
			description = "A Vango web application"
		}
	}

	// Tailwind
	fmt.Printf("? Include Tailwind CSS? [Y/n] ")
	answer, err := reader.ReadString('\n')
	if err != nil {
		return "", false, err
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer == "n" || answer == "no" {
		tailwind = false
	} else {
		tailwind = true
	}

	return description, tailwind, nil
}

func isValidProjectName(name string) bool {
	if name == "" {
		return false
	}
	// Basic validation: no spaces, no starting with number
	for i, r := range name {
		if r == ' ' || r == '/' || r == '\\' {
			return false
		}
		if i == 0 && r >= '0' && r <= '9' {
			return false
		}
	}
	return true
}

func initGoMod(dir, moduleName string) error {
	// Read go.mod to check if it already has the module line
	goModPath := filepath.Join(dir, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return err
	}

	// If the template already has a go.mod with proper module path, skip
	if strings.Contains(string(content), "module "+moduleName) {
		return nil
	}

	// Run go mod init
	cmd := exec.Command("go", "mod", "init", moduleName)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Remove existing go.mod first
	os.Remove(goModPath)

	return cmd.Run()
}

func goModTidy(dir string) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dir
	return cmd.Run()
}

func setupTailwind(dir string) error {
	// Check if npm/npx is available
	if _, err := exec.LookPath("npx"); err != nil {
		return fmt.Errorf("npx not found, please install Node.js")
	}

	// Initialize package.json if not exists
	packageJSON := filepath.Join(dir, "package.json")
	if _, err := os.Stat(packageJSON); os.IsNotExist(err) {
		content := `{
  "name": "` + filepath.Base(dir) + `",
  "private": true,
  "scripts": {
    "build:css": "npx tailwindcss -i ./app/styles/input.css -o ./public/styles.css --minify",
    "watch:css": "npx tailwindcss -i ./app/styles/input.css -o ./public/styles.css --watch"
  },
  "devDependencies": {
    "tailwindcss": "^3.4.0"
  }
}
`
		os.WriteFile(packageJSON, []byte(content), 0644)
	}

	return nil
}
