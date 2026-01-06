package dev

import (
	"path/filepath"

	"github.com/vango-go/vango/internal/config"
)

// CollectWatchPaths returns a normalized list of watch paths for the project.
func CollectWatchPaths(cfg *config.Config) []string {
	projectDir := cfg.Dir()
	paths := []string{
		filepath.Join(projectDir, "main.go"),
		filepath.Join(projectDir, "pkg"),
		filepath.Join(projectDir, "internal"),
		filepath.Join(projectDir, "db"),
		cfg.RoutesPath(),
		cfg.ComponentsPath(),
		cfg.StorePath(),
		cfg.MiddlewarePath(),
		cfg.PublicPath(),
	}

	if cfg.Tailwind.Output != "" {
		paths = append(paths, resolvePath(projectDir, cfg.Tailwind.Output))
	}

	for _, path := range cfg.Dev.Watch {
		paths = append(paths, resolvePath(projectDir, path))
	}

	unique := make([]string, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		if path == "" {
			continue
		}
		clean := filepath.Clean(path)
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		unique = append(unique, clean)
	}

	return unique
}

func resolvePath(projectDir, path string) string {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(projectDir, path)
}
