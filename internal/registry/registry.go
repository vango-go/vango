package registry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/vango-dev/vango/v2/internal/config"
	"github.com/vango-dev/vango/v2/internal/errors"
)

// Manifest represents the registry manifest.
type Manifest struct {
	ManifestVersion int                  `json:"manifestVersion"`
	Version         string               `json:"version"`
	Registry        string               `json:"registry"`
	Components      map[string]Component `json:"components"`
}

// Component represents a component in the registry.
type Component struct {
	Files     []string `json:"files"`
	DependsOn []string `json:"dependsOn"`
	Internal  bool     `json:"internal,omitempty"`
}

// InstalledComponent represents a locally installed component.
type InstalledComponent struct {
	Name     string
	Version  string
	Checksum string
	Modified bool // True if local changes detected
}

// Registry handles VangoUI component management.
type Registry struct {
	config    *config.Config
	manifest  *Manifest
	client    *http.Client
	cacheDir  string
}

// New creates a new Registry.
func New(cfg *config.Config) *Registry {
	return &Registry{
		config: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		cacheDir: filepath.Join(cfg.Dir(), ".vango", "registry"),
	}
}

// FetchManifest downloads and parses the registry manifest.
func (r *Registry) FetchManifest(ctx context.Context) (*Manifest, error) {
	if r.manifest != nil {
		return r.manifest, nil
	}

	registryURL := r.config.UI.Registry
	if registryURL == "" {
		registryURL = config.DefaultRegistry
	}

	req, err := http.NewRequestWithContext(ctx, "GET", registryURL, nil)
	if err != nil {
		return nil, errors.New("E144").Wrap(err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, errors.New("E144").
			WithDetail("Could not connect to registry: " + err.Error()).
			WithSuggestion("Check your internet connection")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("E144").
			WithDetail(fmt.Sprintf("Registry returned status %d", resp.StatusCode))
	}

	var manifest Manifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, errors.New("E144").
			WithDetail("Invalid registry manifest: " + err.Error())
	}

	r.manifest = &manifest
	return &manifest, nil
}

// Install installs components and their dependencies.
func (r *Registry) Install(ctx context.Context, components []string) error {
	manifest, err := r.FetchManifest(ctx)
	if err != nil {
		return err
	}

	// Resolve dependencies
	toInstall, err := r.resolveDependencies(manifest, components)
	if err != nil {
		return err
	}

	// Ensure output directory exists
	outputDir := r.config.UIComponentsPath()
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	// Install each component
	for _, name := range toInstall {
		comp, ok := manifest.Components[name]
		if !ok {
			continue
		}

		for _, file := range comp.Files {
			if err := r.installFile(ctx, manifest.Registry, manifest.Version, name, file, outputDir); err != nil {
				return err
			}
		}
	}

	// Update vango.json
	return r.updateConfig(toInstall)
}

// resolveDependencies returns components in dependency order.
func (r *Registry) resolveDependencies(manifest *Manifest, components []string) ([]string, error) {
	resolved := make(map[string]bool)
	var order []string

	var resolve func(name string) error
	resolve = func(name string) error {
		if resolved[name] {
			return nil
		}

		comp, ok := manifest.Components[name]
		if !ok {
			return errors.New("E143").
				WithDetail("Component '" + name + "' not found in registry")
		}

		// Resolve dependencies first
		for _, dep := range comp.DependsOn {
			if err := resolve(dep); err != nil {
				return err
			}
		}

		resolved[name] = true
		order = append(order, name)
		return nil
	}

	for _, name := range components {
		if err := resolve(name); err != nil {
			return nil, err
		}
	}

	return order, nil
}

// installFile downloads and writes a component file.
func (r *Registry) installFile(ctx context.Context, registryBase, version, component, filename, outputDir string) error {
	// Download file
	url := fmt.Sprintf("%s/components/%s/%s", strings.TrimSuffix(registryBase, "/registry.json"), component, filename)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return errors.New("E144").Wrap(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("E143").
			WithDetail(fmt.Sprintf("Could not download %s/%s: status %d", component, filename, resp.StatusCode))
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Calculate checksum
	hash := sha256.Sum256(content)
	checksum := hex.EncodeToString(hash[:])

	// Check if file exists and has local modifications
	outputPath := filepath.Join(outputDir, filename)
	if existingContent, err := os.ReadFile(outputPath); err == nil {
		existingChecksum, _ := parseHeader(string(existingContent))
		if existingChecksum != "" && existingChecksum != checksum {
			// File has been modified
			localHash := sha256.Sum256(existingContent)
			localChecksum := hex.EncodeToString(localHash[:])

			if localChecksum != existingChecksum {
				// Local modifications detected
				// For now, skip and warn
				fmt.Printf("Warning: %s has local modifications, skipping\n", filename)
				return nil
			}
		}
	}

	// Add header
	header := fmt.Sprintf(`// Source: vango.dev/ui/%s
// Version: %s
// Checksum: sha256:%s

`, component, version, checksum[:16])

	finalContent := header + string(content)

	// Write file
	return os.WriteFile(outputPath, []byte(finalContent), 0644)
}

// updateConfig updates vango.json with installed components.
func (r *Registry) updateConfig(installed []string) error {
	// Merge with existing installed components
	existing := make(map[string]bool)
	for _, name := range r.config.UI.Installed {
		existing[name] = true
	}
	for _, name := range installed {
		existing[name] = true
	}

	// Convert to sorted slice
	var all []string
	for name := range existing {
		all = append(all, name)
	}
	sort.Strings(all)

	r.config.UI.Installed = all

	// Update version if manifest is loaded
	if r.manifest != nil {
		r.config.UI.Version = r.manifest.Version
	}

	return r.config.Save()
}

// ListInstalled returns all locally installed components.
func (r *Registry) ListInstalled() ([]InstalledComponent, error) {
	outputDir := r.config.UIComponentsPath()
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		return nil, nil
	}

	var components []InstalledComponent

	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}

		content, err := os.ReadFile(filepath.Join(outputDir, entry.Name()))
		if err != nil {
			continue
		}

		checksum, version := parseHeader(string(content))
		if checksum == "" {
			continue // Not a VangoUI component
		}

		// Check for modifications
		actualHash := sha256.Sum256(content)
		actualChecksum := hex.EncodeToString(actualHash[:])[:16]
		modified := actualChecksum != checksum

		// Extract component name from filename
		name := strings.TrimSuffix(entry.Name(), ".go")

		components = append(components, InstalledComponent{
			Name:     name,
			Version:  version,
			Checksum: checksum,
			Modified: modified,
		})
	}

	return components, nil
}

// Upgrade upgrades all installed components.
func (r *Registry) Upgrade(ctx context.Context) error {
	manifest, err := r.FetchManifest(ctx)
	if err != nil {
		return err
	}

	// Check if upgrade is needed
	if r.config.UI.Version == manifest.Version {
		return nil // Already up to date
	}

	// Get installed components
	installed, err := r.ListInstalled()
	if err != nil {
		return err
	}

	// Collect non-modified components for upgrade
	var toUpgrade []string
	for _, comp := range installed {
		if !comp.Modified {
			toUpgrade = append(toUpgrade, comp.Name)
		}
	}

	if len(toUpgrade) == 0 {
		return nil
	}

	return r.Install(ctx, toUpgrade)
}

// Init initializes the UI components directory with base files.
func (r *Registry) Init(ctx context.Context) error {
	outputDir := r.config.UIComponentsPath()
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	// Create utils.go
	utilsContent := `package ui

import "strings"

// CN joins class names, filtering empty strings.
func CN(classes ...string) string {
	var result []string
	for _, c := range classes {
		if c = strings.TrimSpace(c); c != "" {
			result = append(result, c)
		}
	}
	return strings.Join(result, " ")
}
`
	if err := os.WriteFile(filepath.Join(outputDir, "utils.go"), []byte(utilsContent), 0644); err != nil {
		return err
	}

	// Create base.go
	baseContent := `package ui

import . "github.com/vango-dev/vango/v2/pkg/vdom"

// BaseConfig contains common configuration for all components.
type BaseConfig struct {
	Class    string
	Attrs    map[string]string
	Children []*VNode
}

// ConfigProvider is implemented by component configs.
type ConfigProvider interface {
	GetBase() *BaseConfig
}

// Option is a functional option for configuring components.
type Option[T ConfigProvider] func(T)

// Class adds CSS classes to a component.
func Class[T ConfigProvider](class string) Option[T] {
	return func(cfg T) {
		base := cfg.GetBase()
		base.Class = CN(base.Class, class)
	}
}

// Attr sets an attribute on a component.
func Attr[T ConfigProvider](name, value string) Option[T] {
	return func(cfg T) {
		base := cfg.GetBase()
		if base.Attrs == nil {
			base.Attrs = make(map[string]string)
		}
		base.Attrs[name] = value
	}
}

// Child adds children to a component.
func Child[T ConfigProvider](children ...*VNode) Option[T] {
	return func(cfg T) {
		base := cfg.GetBase()
		base.Children = append(base.Children, children...)
	}
}
`
	if err := os.WriteFile(filepath.Join(outputDir, "base.go"), []byte(baseContent), 0644); err != nil {
		return err
	}

	// Update vango.json
	r.config.UI.Installed = []string{}
	if r.manifest != nil {
		r.config.UI.Version = r.manifest.Version
	}

	return r.config.Save()
}

// parseHeader extracts checksum and version from a component header.
func parseHeader(content string) (checksum, version string) {
	checksumRe := regexp.MustCompile(`// Checksum: sha256:([a-f0-9]+)`)
	versionRe := regexp.MustCompile(`// Version: (.+)`)

	if match := checksumRe.FindStringSubmatch(content); len(match) > 1 {
		checksum = match[1]
	}
	if match := versionRe.FindStringSubmatch(content); len(match) > 1 {
		version = strings.TrimSpace(match[1])
	}

	return
}

// ComponentInfo returns information about a component.
type ComponentInfo struct {
	Name         string
	Files        []string
	Dependencies []string
	Installed    bool
	LocalVersion string
	LatestVersion string
	Modified     bool
}

// Info returns information about a component.
func (r *Registry) Info(ctx context.Context, name string) (*ComponentInfo, error) {
	manifest, err := r.FetchManifest(ctx)
	if err != nil {
		return nil, err
	}

	comp, ok := manifest.Components[name]
	if !ok {
		return nil, errors.New("E143").
			WithDetail("Component '" + name + "' not found")
	}

	info := &ComponentInfo{
		Name:          name,
		Files:         comp.Files,
		Dependencies:  comp.DependsOn,
		LatestVersion: manifest.Version,
	}

	// Check local installation
	installed, _ := r.ListInstalled()
	for _, ic := range installed {
		if ic.Name == name {
			info.Installed = true
			info.LocalVersion = ic.Version
			info.Modified = ic.Modified
			break
		}
	}

	return info, nil
}
