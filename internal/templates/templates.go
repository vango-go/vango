// Package templates provides project scaffolding templates for vango create.
package templates

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds configuration for generating a new project from a template.
type Config struct {
	ProjectName  string
	ModulePath   string
	Description  string
	HasTailwind  bool
	HasDatabase  bool
	DatabaseType string
	HasAuth      bool
}

// Template represents a project scaffold template.
type Template struct {
	Name        string
	Description string
	files       []templateFile
}

// templateFile represents a single file in a template.
type templateFile struct {
	Path    string
	Content string
	Binary  []byte // For binary files like favicon.ico
}

// registry holds all available templates.
var registry = map[string]*Template{}

// Register adds a template to the registry.
func Register(t *Template) {
	registry[t.Name] = t
}

// Get returns a template by name.
func Get(name string) (*Template, error) {
	t, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("template %q not found", name)
	}
	return t, nil
}

// GetWithOptions returns a template by name, with additional configuration applied.
// Currently this is the same as Get, but allows for future expansion.
func GetWithOptions(name string, cfg Config) (*Template, error) {
	t, err := Get(name)
	if err != nil {
		return nil, err
	}
	if !cfg.HasAuth {
		return t, nil
	}

	clone := *t
	clone.files = append([]templateFile{}, t.files...)
	applyAuthScaffold(&clone)
	return &clone, nil
}

// Create generates a new project from the template.
func (t *Template) Create(dir string, cfg Config) error {
	for _, f := range t.files {
		path := filepath.Join(dir, f.Path)

		// Replace template variables in path
		path = replaceTemplateVars(path, cfg)

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", f.Path, err)
		}

		// Write the file
		if f.Binary != nil {
			if err := os.WriteFile(path, f.Binary, 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", f.Path, err)
			}
		} else {
			content := replaceTemplateVars(f.Content, cfg)
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", f.Path, err)
			}
		}
	}
	return nil
}

// replaceTemplateVars replaces template variables in content.
func replaceTemplateVars(content string, cfg Config) string {
	replacements := map[string]string{
		"{{.ProjectName}}":  cfg.ProjectName,
		"{{.ModulePath}}":   cfg.ModulePath,
		"{{.Description}}":  cfg.Description,
		"{{.DatabaseType}}": cfg.DatabaseType,
	}
	for placeholder, value := range replacements {
		content = strings.ReplaceAll(content, placeholder, value)
	}
	return content
}

func replaceTemplateFile(t *Template, path, content string) {
	for i := range t.files {
		if t.files[i].Path == path {
			t.files[i].Content = content
			return
		}
	}
	t.files = append(t.files, templateFile{Path: path, Content: content})
}

// List returns all available template names.
func List() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}
