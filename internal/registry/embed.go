package registry

import (
	"embed"
	"encoding/json"
	"io/fs"
)

//go:embed components
var embeddedComponents embed.FS

// EmbeddedManifest returns the embedded component manifest.
func EmbeddedManifest() (*Manifest, error) {
	data, err := embeddedComponents.ReadFile("components/manifest.json")
	if err != nil {
		return nil, err
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// EmbeddedComponentSource returns the source code for an embedded component.
func EmbeddedComponentSource(name string) ([]byte, error) {
	path := "components/" + name + ".go"
	return embeddedComponents.ReadFile(path)
}

// EmbeddedFS returns the embedded filesystem for component access.
func EmbeddedFS() fs.FS {
	return embeddedComponents
}

// HasEmbeddedComponent checks if a component is available in the embedded registry.
func HasEmbeddedComponent(name string) bool {
	path := "components/" + name + ".go"
	_, err := embeddedComponents.ReadFile(path)
	return err == nil
}

// ListEmbeddedComponents returns all embedded component names.
func ListEmbeddedComponents() ([]string, error) {
	manifest, err := EmbeddedManifest()
	if err != nil {
		return nil, err
	}

	var names []string
	for name, comp := range manifest.Components {
		if !comp.Internal {
			names = append(names, name)
		}
	}

	return names, nil
}
