// Package templates provides project scaffolding templates.
//
// This package contains embedded templates for creating new Vango projects.
// Templates include all necessary files for a working application.
//
// # Available Templates
//
//   - minimal: Just the essentials for a Vango app
//   - full: Complete starter with example pages and components
//   - api: API-only project without UI
//
// # Usage
//
//	tmpl := templates.Get("full")
//	if err := tmpl.Create(projectDir, config); err != nil {
//	    log.Fatal(err)
//	}
//
// # Template Variables
//
// Templates support variable substitution:
//
//	{{.ProjectName}}     - Name of the project
//	{{.ModulePath}}      - Go module path
//	{{.Description}}     - Project description
//	{{.HasTailwind}}     - Whether Tailwind is enabled
//	{{.HasDatabase}}     - Whether database is enabled
package templates
