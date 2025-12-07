// Package registry provides VangoUI component management.
//
// This package implements the "copy-paste ownership" model for UI components.
// Components are distributed as source code that developers add to their
// projects and own completely.
//
// # Features
//
//   - Fetch components from the VangoUI registry
//   - Resolve and install dependencies
//   - Track installed components and versions
//   - Detect and handle local modifications
//   - Upgrade components with 3-way merge support
//
// # Registry Manifest
//
// The registry provides a manifest.json file:
//
//	{
//	  "manifestVersion": 1,
//	  "version": "1.0.0",
//	  "components": {
//	    "button": {
//	      "files": ["button.go"],
//	      "dependsOn": ["utils", "base"]
//	    }
//	  }
//	}
//
// # Component Headers
//
// Installed components include metadata headers:
//
//	// Source: vango.dev/ui/button
//	// Version: 1.0.0
//	// Checksum: sha256:a1b2c3d4...
//
// # Usage
//
//	reg := registry.New(cfg)
//
//	// Install components
//	err := reg.Install(ctx, []string{"button", "dialog"})
//
//	// Upgrade all components
//	err := reg.Upgrade(ctx)
package registry
