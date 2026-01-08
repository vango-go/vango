package server

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"strings"

	clientdist "github.com/vango-go/vango/client/dist"
)

var thinClientETag = func() string {
	sum := sha256.Sum256(clientdist.VangoMinJS)
	return fmt.Sprintf("%q", fmt.Sprintf("%x", sum[:]))
}()

func (s *Server) serveThinClient(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if len(clientdist.VangoMinJS) == 0 {
		http.Error(w, "Thin client not available", http.StatusInternalServerError)
		return
	}

	// ETag-based caching (safe even without a versioned URL).
	w.Header().Set("ETag", thinClientETag)
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// Caching policy:
	// - DevMode: no-store to avoid stale client behavior while iterating.
	// - Prod: revalidate via ETag (no versioned URL yet), so updates are picked up safely.
	if s.config != nil && s.config.DevMode {
		w.Header().Set("Cache-Control", "no-store")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=0, must-revalidate")
	}

	if etagMatches(r.Header.Get("If-None-Match"), thinClientETag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(clientdist.VangoMinJS)
}

func etagMatches(ifNoneMatchHeader, etag string) bool {
	if ifNoneMatchHeader == "" || etag == "" {
		return false
	}
	// Handle lists: If-None-Match: "abc", W/"def"
	for _, part := range strings.Split(ifNoneMatchHeader, ",") {
		candidate := strings.TrimSpace(part)
		if candidate == etag {
			return true
		}
		if strings.HasPrefix(candidate, "W/") && strings.TrimPrefix(candidate, "W/") == etag {
			return true
		}
	}
	return false
}

