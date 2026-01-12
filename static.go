package vango

import (
	"net/http"
	"path"
	"path/filepath"
	"strings"
)

// =============================================================================
// Static File Serving
// =============================================================================

// staticRelPath returns a sanitized relative path for a static file request.
// It rejects traversal and absolute-path tricks to ensure static serving cannot
// escape the configured static directory.
func (a *App) staticRelPath(urlPath string) (string, bool) {
	if a.staticFS == nil || a.staticDir == "" {
		return "", false
	}

	rel := a.stripStaticPrefix(urlPath)
	if rel == "" {
		return "", false
	}

	// Reject NUL early (can appear via %00).
	if strings.IndexByte(rel, 0) != -1 {
		return "", false
	}

	// Reject platform-dependent separators.
	if strings.Contains(rel, "\\") {
		return "", false
	}

	// After prefix stripping, a leading "/" indicates an absolute-path attempt
	// (e.g. "/static//etc/passwd" => "/etc/passwd").
	if strings.HasPrefix(rel, "/") {
		return "", false
	}

	// Reject dot-segments before cleaning to avoid "cleaning away" traversal
	// attempts and changing the meaning of the request path.
	for _, seg := range strings.Split(rel, "/") {
		if seg == "." || seg == ".." {
			return "", false
		}
	}

	clean := path.Clean(rel)
	if clean == "." || clean == "" || clean == ".." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") {
		return "", false
	}

	// Defense-in-depth: reject OS-absolute/volume paths after slash conversion.
	osPath := filepath.FromSlash(clean)
	if filepath.IsAbs(osPath) || filepath.VolumeName(osPath) != "" {
		return "", false
	}

	return clean, true
}

// shouldServeStatic checks if a request path should be served as a static file.
// Returns true if the file exists in the static directory.
func (a *App) shouldServeStatic(urlPath string) bool {
	rel, ok := a.staticRelPath(urlPath)
	if !ok {
		return false
	}

	f, err := a.staticFS.Open(rel)
	if err != nil {
		return false
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return false
	}

	return !info.IsDir()
}

// serveStatic handles static file requests.
func (a *App) serveStatic(w http.ResponseWriter, r *http.Request) {
	// Only serve GET and HEAD requests for static files
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	rel, ok := a.staticRelPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	f, err := a.staticFS.Open(rel)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}

	// Apply cache control headers
	a.applyCacheHeaders(w, rel)

	// Apply custom headers
	for key, value := range a.config.Static.Headers {
		w.Header().Set(key, value)
	}

	http.ServeContent(w, r, rel, info.ModTime(), f)
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
