// Package build provides production build functionality for Vango applications.
//
// This package handles:
//   - Go binary compilation with optimizations
//   - Thin client JavaScript bundling and minification
//   - Tailwind CSS compilation
//   - Static asset copying with cache busting
//   - Build manifest generation
//
// # Usage
//
//	builder := build.New(cfg)
//	result, err := builder.Build(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Printf("Built in %s\n", result.Duration)
//	fmt.Printf("Binary: %s\n", result.Binary)
//	fmt.Printf("Public: %s\n", result.Public)
//
// # Output Structure
//
//	dist/
//	├── server              # Go binary
//	├── public/
//	│   ├── vango.min.js   # Thin client
//	│   ├── styles.css     # Compiled CSS
//	│   └── assets/        # Static files with hashes
//	└── manifest.json      # Asset manifest
//
// # Manifest
//
// The manifest maps original asset paths to their hashed versions:
//
//	{
//	  "vango.min.js": "vango.a1b2c3.min.js",
//	  "styles.css": "styles.d4e5f6.css",
//	  "logo.png": "assets/logo.g7h8i9.png"
//	}
package build
