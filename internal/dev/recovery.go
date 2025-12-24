package dev

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vango-dev/vango/v2/pkg/router"
)

// ErrorRecovery handles automatic recovery from common build errors.
type ErrorRecovery struct {
	projectDir string
	routesDir  string
	modulePath string
}

// NewErrorRecovery creates a new error recovery handler.
func NewErrorRecovery(projectDir, routesDir, modulePath string) *ErrorRecovery {
	return &ErrorRecovery{
		projectDir: projectDir,
		routesDir:  routesDir,
		modulePath: modulePath,
	}
}

// RecoveryResult contains the result of an attempted recovery.
type RecoveryResult struct {
	// Recovered indicates if recovery was successful.
	Recovered bool

	// Action describes what was done.
	Action string

	// Details provides additional information.
	Details string
}

// AttemptRecovery tries to automatically fix common build errors.
// Returns true if a fix was applied and the build should be retried.
func (r *ErrorRecovery) AttemptRecovery(buildOutput string) RecoveryResult {
	// Check for undefined symbol errors (renamed handlers)
	if result := r.recoverFromUndefinedSymbols(buildOutput); result.Recovered {
		return result
	}

	// Check for missing import errors
	if result := r.recoverFromMissingImports(buildOutput); result.Recovered {
		return result
	}

	return RecoveryResult{Recovered: false}
}

// recoverFromUndefinedSymbols handles "undefined: SymbolName" errors.
// This happens when a handler is renamed but routes_gen.go still references the old name.
func (r *ErrorRecovery) recoverFromUndefinedSymbols(buildOutput string) RecoveryResult {
	// Match patterns like:
	// ./app/routes/routes_gen.go:15:12: undefined: IndexPage
	// or: app/routes/routes_gen.go:15: undefined: IndexPage
	undefinedPattern := regexp.MustCompile(`routes_gen\.go:\d+(?::\d+)?:\s*undefined:\s*(\w+)`)
	matches := undefinedPattern.FindAllStringSubmatch(buildOutput, -1)

	if len(matches) == 0 {
		return RecoveryResult{Recovered: false}
	}

	// Collect undefined symbols
	undefinedSymbols := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			undefinedSymbols[match[1]] = true
		}
	}

	if len(undefinedSymbols) == 0 {
		return RecoveryResult{Recovered: false}
	}

	// Re-scan routes to find current symbols
	scanner := router.NewScanner(r.routesDir)
	routes, err := scanner.Scan()
	if err != nil {
		return RecoveryResult{
			Recovered: false,
			Details:   "failed to scan routes: " + err.Error(),
		}
	}

	// Regenerate routes_gen.go with current symbols
	gen := router.NewGenerator(routes, r.modulePath)
	code, err := gen.Generate()
	if err != nil {
		return RecoveryResult{
			Recovered: false,
			Details:   "failed to generate routes: " + err.Error(),
		}
	}

	// Write the new routes_gen.go
	routesGenPath := filepath.Join(r.routesDir, "routes_gen.go")
	if err := os.WriteFile(routesGenPath, code, 0644); err != nil {
		return RecoveryResult{
			Recovered: false,
			Details:   "failed to write routes_gen.go: " + err.Error(),
		}
	}

	// Format the symbols list for display
	var symbolList []string
	for sym := range undefinedSymbols {
		symbolList = append(symbolList, sym)
	}

	return RecoveryResult{
		Recovered: true,
		Action:    "regenerated routes_gen.go",
		Details:   "fixed undefined symbols: " + strings.Join(symbolList, ", "),
	}
}

// recoverFromMissingImports handles "could not import" errors for route packages.
// This happens when a route package is deleted but routes_gen.go still imports it.
func (r *ErrorRecovery) recoverFromMissingImports(buildOutput string) RecoveryResult {
	// Match patterns like:
	// could not import my-app/app/routes/admin (no required module provides package)
	importPattern := regexp.MustCompile(`could not import\s+([^\s]+/app/routes/[^\s]+)`)
	matches := importPattern.FindAllStringSubmatch(buildOutput, -1)

	if len(matches) == 0 {
		return RecoveryResult{Recovered: false}
	}

	// Collect missing imports
	missingImports := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			missingImports[match[1]] = true
		}
	}

	if len(missingImports) == 0 {
		return RecoveryResult{Recovered: false}
	}

	// Re-scan routes (deleted packages won't be found)
	scanner := router.NewScanner(r.routesDir)
	routes, err := scanner.Scan()
	if err != nil {
		return RecoveryResult{
			Recovered: false,
			Details:   "failed to scan routes: " + err.Error(),
		}
	}

	// Regenerate routes_gen.go with current packages
	gen := router.NewGenerator(routes, r.modulePath)
	code, err := gen.Generate()
	if err != nil {
		return RecoveryResult{
			Recovered: false,
			Details:   "failed to generate routes: " + err.Error(),
		}
	}

	// Write the new routes_gen.go
	routesGenPath := filepath.Join(r.routesDir, "routes_gen.go")
	if err := os.WriteFile(routesGenPath, code, 0644); err != nil {
		return RecoveryResult{
			Recovered: false,
			Details:   "failed to write routes_gen.go: " + err.Error(),
		}
	}

	return RecoveryResult{
		Recovered: true,
		Action:    "regenerated routes_gen.go",
		Details:   "removed deleted package imports",
	}
}

// IsRecoverableError checks if a build error might be recoverable.
func IsRecoverableError(buildOutput string) bool {
	// Check for undefined symbols in routes_gen.go
	if strings.Contains(buildOutput, "routes_gen.go") &&
		strings.Contains(buildOutput, "undefined:") {
		return true
	}

	// Check for missing route package imports
	if strings.Contains(buildOutput, "could not import") &&
		strings.Contains(buildOutput, "/app/routes/") {
		return true
	}

	return false
}

// GetModulePath reads the module path from go.mod.
func GetModulePath(projectDir string) (string, error) {
	goModPath := filepath.Join(projectDir, "go.mod")
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimPrefix(line, "module "), nil
		}
	}

	return "", nil
}
