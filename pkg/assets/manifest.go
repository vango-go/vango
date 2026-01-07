// Package assets provides runtime resolution of fingerprinted asset paths.
//
// During the build process, Vango generates a manifest.json mapping source
// asset names to their fingerprinted (hashed) versions:
//
//	{
//	  "vango.js": "vango.a1b2c3d4.min.js",
//	  "styles.css": "styles.e5f6g7h8.css"
//	}
//
// This package loads that manifest and provides resolution functions for use
// in templates and components:
//
//	manifest, _ := assets.Load("dist/manifest.json")
//	resolver := assets.NewResolver(manifest, "/public/")
//
//	// In component:
//	Script(Src(ctx.Asset("vango.js")))
//	// Outputs: <script src="/public/vango.a1b2c3d4.min.js">
package assets

import (
	"encoding/json"
	"os"
	"sync"
)

// Manifest holds the mapping from source asset paths to fingerprinted paths.
// It is safe for concurrent use.
type Manifest struct {
	entries map[string]string
	mu      sync.RWMutex
}

// NewManifest creates an empty manifest.
// Use Load() to create a manifest from a JSON file.
func NewManifest() *Manifest {
	return &Manifest{
		entries: make(map[string]string),
	}
}

// Load reads a manifest.json file and returns a Manifest.
// The manifest file is expected to be in JSON format: {"source.js": "source.abc123.js"}
//
// If the file does not exist or cannot be read, an error is returned.
// In development, you may want to ignore the error and use NewPassthroughResolver.
func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var entries map[string]string
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}

	return &Manifest{entries: entries}, nil
}

// Resolve returns the fingerprinted path for the given source path.
// If not found, returns the original path unchanged.
//
// This is the core resolution function. For most use cases, prefer using
// a Resolver with a configured prefix.
func (m *Manifest) Resolve(source string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if resolved, ok := m.entries[source]; ok {
		return resolved
	}
	return source
}

// Has returns true if the manifest contains the given source path.
func (m *Manifest) Has(source string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.entries[source]
	return ok
}

// Set adds or updates an entry in the manifest.
// This is primarily useful for testing or dynamic manifest building.
func (m *Manifest) Set(source, resolved string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.entries[source] = resolved
}

// Len returns the number of entries in the manifest.
func (m *Manifest) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.entries)
}

// All returns a copy of all manifest entries.
func (m *Manifest) All() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]string, len(m.entries))
	for k, v := range m.entries {
		result[k] = v
	}
	return result
}
