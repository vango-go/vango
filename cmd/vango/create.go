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
		minimal     bool
		withTailwind bool
		withDB      string
		withAuth    bool
		full        bool
		description string
		skipPrompts bool
	)

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new Vango project",
		Long: `Create a new Vango project with the specified name.

Scaffold Variants:
  (default)      Standard starter with home, about, health API, navbar, footer
  --minimal      Just the essentials (home page, health API)
  --with-tailwind Include Tailwind CSS configuration
  --with-db=*    Include database setup (sqlite, postgres)
  --with-auth    Include admin routes with authentication middleware
  --full         All features: Tailwind, database, auth

Examples:
  vango create my-app
  vango create my-app --minimal
  vango create my-app --with-tailwind
  vango create my-app --with-db=postgres
  vango create my-app --with-auth
  vango create my-app --full`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			return runCreate(name, createOptions{
				minimal:      minimal,
				withTailwind: withTailwind,
				withDB:       withDB,
				withAuth:     withAuth,
				full:         full,
				description:  description,
				skipPrompts:  skipPrompts,
			})
		},
	}

	cmd.Flags().BoolVar(&minimal, "minimal", false, "Create minimal scaffold (home page only)")
	cmd.Flags().BoolVar(&withTailwind, "with-tailwind", false, "Include Tailwind CSS")
	cmd.Flags().StringVar(&withDB, "with-db", "", "Include database setup (sqlite, postgres)")
	cmd.Flags().BoolVar(&withAuth, "with-auth", false, "Include admin routes with auth middleware")
	cmd.Flags().BoolVar(&full, "full", false, "Include all features (Tailwind, database, auth)")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Project description")
	cmd.Flags().BoolVarP(&skipPrompts, "yes", "y", false, "Skip prompts and use defaults")

	return cmd
}

type createOptions struct {
	minimal      bool
	withTailwind bool
	withDB       string
	withAuth     bool
	full         bool
	description  string
	skipPrompts  bool
}

func runCreate(name string, opts createOptions) error {
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
	if !opts.skipPrompts {
		var err error
		opts.description, err = promptForDescription(opts.description)
		if err != nil {
			return err
		}
	}

	// Set defaults
	if opts.description == "" {
		opts.description = "A Vango web application"
	}

	// --full enables all options
	if opts.full {
		opts.withTailwind = true
		opts.withAuth = true
		if opts.withDB == "" {
			opts.withDB = "sqlite"
		}
	}

	// Determine template name
	templateName := "standard"
	if opts.minimal {
		templateName = "minimal"
	} else if opts.withTailwind || opts.full {
		templateName = "full"
	}

	// Create template configuration
	cfg := templates.Config{
		ProjectName:  name,
		ModulePath:   name, // Simple module path for local projects
		Description:  opts.description,
		HasTailwind:  opts.withTailwind || opts.full,
		HasDatabase:  opts.withDB != "",
		DatabaseType: opts.withDB,
		HasAuth:      opts.withAuth || opts.full,
	}

	// Get template with options
	var tmpl *templates.Template
	if cfg.HasDatabase || cfg.HasAuth {
		tmpl, err = templates.GetWithOptions(templateName, cfg)
	} else {
		tmpl, err = templates.Get(templateName)
	}
	if err != nil {
		return err
	}

	// Create project directory
	info("Creating project directory...")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return err
	}

	// Create project from template
	info("Creating project from '%s' template...", tmpl.Name)
	if cfg.HasTailwind {
		info("  Including Tailwind CSS")
	}
	if cfg.HasDatabase {
		info("  Including database setup (%s)", cfg.DatabaseType)
	}
	if cfg.HasAuth {
		info("  Including authentication scaffold")
	}

	if err := tmpl.Create(projectDir, cfg); err != nil {
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
	if cfg.HasTailwind {
		info("Setting up Tailwind CSS...")
		if err := setupTailwind(projectDir, name); err != nil {
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

	// Show additional setup for optional features
	if cfg.HasTailwind {
		fmt.Println("  Tailwind CSS setup:")
		fmt.Println()
		fmt.Println("    npm install")
		fmt.Println("    npm run watch:css")
		fmt.Println()
	}

	if cfg.HasDatabase {
		fmt.Println("  Database setup:")
		fmt.Println()
		fmt.Printf("    Configure your %s database in db/db.go\n", cfg.DatabaseType)
		fmt.Println()
	}

	return nil
}

func promptForDescription(description string) (string, error) {
	if description != "" {
		return description, nil
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("? Description: ")
	desc, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	description = strings.TrimSpace(desc)
	if description == "" {
		description = "A Vango web application"
	}
	return description, nil
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

func setupTailwind(dir, projectName string) error {
	// Check if npm/npx is available
	if _, err := exec.LookPath("npx"); err != nil {
		return fmt.Errorf("npx not found, please install Node.js")
	}

	// Initialize package.json if not exists
	packageJSON := filepath.Join(dir, "package.json")
	if _, err := os.Stat(packageJSON); os.IsNotExist(err) {
		content := `{
  "name": "` + projectName + `",
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
		if err := os.WriteFile(packageJSON, []byte(content), 0644); err != nil {
			return err
		}
	}

	return nil
}
