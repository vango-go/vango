package vango

import (
	"net/http"
	"os"
	"path"
	"strings"
)

// =============================================================================
// Static File Serving
// =============================================================================

// shouldServeStatic checks if a request path should be served as a static file.
// Returns true if the file exists in the static directory.
func (a *App) shouldServeStatic(urlPath string) bool {
	if a.staticFS == nil || a.staticDir == "" {
		return false
	}

	// Remove the prefix to get the file path
	filePath := a.stripStaticPrefix(urlPath)
	if filePath == "" {
		return false
	}

	// Check if file exists (not a directory)
	fullPath := path.Join(a.staticDir, filePath)
	info, err := os.Stat(fullPath)
	if err != nil {
		return false
	}

	// Don't serve directories as static files
	return !info.IsDir()
}

// serveStatic handles static file requests.
func (a *App) serveStatic(w http.ResponseWriter, r *http.Request) {
	// Only serve GET and HEAD requests for static files
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get the file path
	filePath := a.stripStaticPrefix(r.URL.Path)
	if filePath == "" {
		http.NotFound(w, r)
		return
	}

	// Apply cache control headers
	a.applyCacheHeaders(w, filePath)

	// Apply custom headers
	for key, value := range a.config.Static.Headers {
		w.Header().Set(key, value)
	}

	// Serve the file
	http.ServeFile(w, r, path.Join(a.staticDir, filePath))
}

// stripStaticPrefix removes the static prefix from a URL path.
// Returns the relative file path within the static directory.
func (a *App) stripStaticPrefix(urlPath string) string {
	prefix := a.staticPrefix
	if prefix == "" {
		prefix = "/"
	}

	// Ensure prefix ends with /
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	// Handle root prefix specially
	if prefix == "/" {
		// For root prefix, all paths are candidates
		return strings.TrimPrefix(urlPath, "/")
	}

	// Check if path starts with prefix
	if !strings.HasPrefix(urlPath, prefix) {
		return ""
	}

	return strings.TrimPrefix(urlPath, prefix)
}

// applyCacheHeaders applies cache control headers based on the configuration.
func (a *App) applyCacheHeaders(w http.ResponseWriter, filePath string) {
	switch a.config.Static.CacheControl {
	case CacheControlNone:
		// No caching - useful for development
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")

	case CacheControlProduction:
		// Production caching strategy
		if isFingerprinted(filePath) {
			// Fingerprinted files are immutable - cache for 1 year
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			// Other files - short cache with revalidation
			w.Header().Set("Cache-Control", "public, max-age=3600, must-revalidate")
		}
	}
}

// isFingerprinted checks if a file path appears to be fingerprinted.
// Fingerprinted files have a hash in their name, e.g., "app.a1b2c3d4.css"
func isFingerprinted(filePath string) bool {
	// Get the base name
	base := path.Base(filePath)

	// Split by dots: ["app", "a1b2c3d4", "css"]
	parts := strings.Split(base, ".")
	if len(parts) < 3 {
		return false
	}

	// Check if the second-to-last part (before extension) looks like a hash
	// Hashes are typically 8+ hex characters
	hash := parts[len(parts)-2]
	if len(hash) < 8 {
		return false
	}

	// Check if it's all hex characters
	for _, c := range hash {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}

	return true
}
