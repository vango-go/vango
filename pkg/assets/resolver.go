package assets

// Resolver provides asset path resolution.
// It combines manifest lookup with path prefixing.
type Resolver interface {
	// Asset resolves a source asset path to its full URL path.
	// This includes any configured prefix and fingerprinted filename.
	//
	// Example:
	//   resolver.Asset("vango.js") → "/public/vango.a1b2c3d4.min.js"
	Asset(source string) string
}

// manifestResolver wraps a Manifest to implement Resolver.
type manifestResolver struct {
	manifest *Manifest
	prefix   string
}

// NewResolver creates a Resolver from a Manifest with an optional path prefix.
//
// The prefix is prepended to all resolved paths. Common prefixes:
//   - "/public/" - standard static file path
//   - "/assets/" - alternative static path
//   - "" - no prefix (use fingerprinted name directly)
//
// Example:
//
//	manifest, _ := assets.Load("dist/manifest.json")
//	resolver := assets.NewResolver(manifest, "/public/")
//	resolver.Asset("vango.js") // "/public/vango.a1b2c3d4.min.js"
func NewResolver(m *Manifest, prefix string) Resolver {
	return &manifestResolver{
		manifest: m,
		prefix:   prefix,
	}
}

func (r *manifestResolver) Asset(source string) string {
	resolved := r.manifest.Resolve(source)
	return r.prefix + resolved
}

// passthrough returns assets unchanged (for development mode).
type passthrough struct {
	prefix string
}

// NewPassthroughResolver creates a resolver that returns paths unchanged.
// Use this in development mode where fingerprinting is disabled.
//
// The prefix is still applied, so dev and prod paths remain consistent:
//
//	// Development:
//	resolver := assets.NewPassthroughResolver("/public/")
//	resolver.Asset("vango.js") // "/public/vango.js"
//
//	// Production:
//	resolver := assets.NewResolver(manifest, "/public/")
//	resolver.Asset("vango.js") // "/public/vango.a1b2c3d4.min.js"
func NewPassthroughResolver(prefix string) Resolver {
	return &passthrough{prefix: prefix}
}

func (p *passthrough) Asset(source string) string {
	return p.prefix + source
}
