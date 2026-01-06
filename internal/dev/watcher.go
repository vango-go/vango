package dev

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ChangeType represents the type of file change.
type ChangeType int

const (
	ChangeGo ChangeType = iota
	ChangeCSS
	ChangeAsset
	ChangeTemplate
)

// Change represents a detected file change.
type Change struct {
	Path string
	Type ChangeType
}

// WatcherConfig configures the file watcher.
type WatcherConfig struct {
	// Paths are the directories to watch.
	Paths []string

	// Ignore patterns to skip (globs).
	Ignore []string

	// Debounce is the delay before triggering on change.
	Debounce time.Duration
}

// DefaultIgnore contains default patterns to ignore.
var DefaultIgnore = []string{
	"*_test.go",
	".git",
	"node_modules",
	"dist",
	"tmp",
	".vango",
	"*.tmp",
	"*.swp",
	"*~",
}

// Watcher monitors files for changes.
type Watcher struct {
	config     WatcherConfig
	onChange   func(Change)
	mu         sync.Mutex
	running    bool
	initialized bool
	stopCh     chan struct{}
	timestamps map[string]time.Time
}

// NewWatcher creates a new file watcher.
func NewWatcher(config WatcherConfig) *Watcher {
	if config.Debounce == 0 {
		config.Debounce = 100 * time.Millisecond
	}
	if len(config.Ignore) == 0 {
		config.Ignore = DefaultIgnore
	}

	return &Watcher{
		config:     config,
		timestamps: make(map[string]time.Time),
	}
}

// OnChange sets the callback for file changes.
func (w *Watcher) OnChange(fn func(Change)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.onChange = fn
}

// Start begins watching for file changes.
func (w *Watcher) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return nil
	}
	w.running = true
	w.stopCh = make(chan struct{})
	w.mu.Unlock()

	// Initialize timestamps
	w.scanInitial()

	ticker := time.NewTicker(w.config.Debounce)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-w.stopCh:
			return nil
		case <-ticker.C:
			w.checkForChanges()
		}
	}
}

// Stop stops the watcher.
func (w *Watcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.running {
		close(w.stopCh)
		w.running = false
	}
}

// scanInitial builds the initial timestamp map.
func (w *Watcher) scanInitial() {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, path := range w.config.Paths {
		filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				if w.shouldIgnore(p) {
					return filepath.SkipDir
				}
				return nil
			}
			if !w.shouldIgnore(p) {
				w.timestamps[p] = info.ModTime()
			}
			return nil
		})
	}

	w.initialized = true
}

// checkForChanges scans for modified files.
func (w *Watcher) checkForChanges() {
	w.mu.Lock()
	callback := w.onChange
	initialized := w.initialized
	w.mu.Unlock()

	if callback == nil {
		return
	}

	var changes []Change

	for _, path := range w.config.Paths {
		filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				if w.shouldIgnore(p) {
					return filepath.SkipDir
				}
				return nil
			}
			if w.shouldIgnore(p) {
				return nil
			}

			w.mu.Lock()
			lastMod, exists := w.timestamps[p]
			modTime := info.ModTime()
			w.mu.Unlock()

			if !exists || modTime.After(lastMod) {
				w.mu.Lock()
				w.timestamps[p] = modTime
				w.mu.Unlock()

				if exists || initialized {
					changes = append(changes, Change{
						Path: p,
						Type: classifyChange(p),
					})
				}
			}

			return nil
		})
	}

	// Also check for deleted files
	w.mu.Lock()
	for p := range w.timestamps {
		if _, err := os.Stat(p); os.IsNotExist(err) {
			delete(w.timestamps, p)
			changes = append(changes, Change{
				Path: p,
				Type: classifyChange(p),
			})
		}
	}
	w.mu.Unlock()

	// Report changes (debounced: report first change of each type)
	reportedTypes := make(map[ChangeType]bool)
	for _, change := range changes {
		if !reportedTypes[change.Type] {
			reportedTypes[change.Type] = true
			callback(change)
		}
	}
}

// shouldIgnore checks if a path should be ignored.
func (w *Watcher) shouldIgnore(fullPath string) bool {
	name := filepath.Base(fullPath)
	normalized := filepath.ToSlash(fullPath)

	for _, pattern := range w.config.Ignore {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}

		// Direct match
		if name == pattern {
			return true
		}

		hasPathSep := strings.Contains(pattern, "/") || strings.Contains(pattern, "\\")
		hasGlob := strings.ContainsAny(pattern, "*?[")

		if hasGlob {
			if hasPathSep {
				if matched, _ := path.Match(filepath.ToSlash(pattern), normalized); matched {
					return true
				}
			} else {
				if matched, _ := filepath.Match(pattern, name); matched {
					return true
				}
			}
			continue
		}

		if hasPathSep {
			if pathMatchesSegments(normalized, filepath.ToSlash(pattern)) {
				return true
			}
			continue
		}

		if pathHasSegment(normalized, pattern) {
			return true
		}
	}

	return false
}

func pathHasSegment(path, segment string) bool {
	if segment == "" {
		return false
	}
	parts := splitPathSegments(path)
	for _, part := range parts {
		if part == segment {
			return true
		}
	}
	return false
}

func pathMatchesSegments(path, pattern string) bool {
	pathParts := splitPathSegments(path)
	patternParts := splitPathSegments(pattern)
	if len(patternParts) == 0 || len(patternParts) > len(pathParts) {
		return false
	}

	for i := 0; i <= len(pathParts)-len(patternParts); i++ {
		match := true
		for j := range patternParts {
			if pathParts[i+j] != patternParts[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}

	return false
}

func splitPathSegments(path string) []string {
	if path == "" {
		return nil
	}
	parts := strings.Split(path, "/")
	result := parts[:0]
	for _, part := range parts {
		if part != "" && part != "." {
			result = append(result, part)
		}
	}
	return result
}

// classifyChange determines the type of change based on file extension.
func classifyChange(path string) ChangeType {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return ChangeGo
	case ".css", ".scss", ".sass", ".less":
		return ChangeCSS
	case ".html", ".gohtml", ".tmpl":
		return ChangeTemplate
	default:
		return ChangeAsset
	}
}

// IsRunning returns whether the watcher is running.
func (w *Watcher) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.running
}
