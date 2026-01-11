// Package tailwind manages the Tailwind CSS standalone binary.
// It handles downloading, caching, and running the binary without requiring Node.js.
package tailwind

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	// Version is the Tailwind CSS version to use.
	// Update this when a new stable version is released.
	// Note: v4.0.0-v4.0.5 had a bug where --watch exited immediately.
	// See: https://github.com/rails/tailwindcss-rails/issues/475
	Version = "v4.1.18"

	// GitHubReleaseURL is the base URL for downloading Tailwind binaries.
	GitHubReleaseURL = "https://github.com/tailwindlabs/tailwindcss/releases/download"

	// DefaultBinDir is the default directory for storing the binary.
	DefaultBinDir = ".vango/bin"
)

// Binary represents the Tailwind CSS standalone binary.
type Binary struct {
	// Version is the Tailwind version.
	Version string

	// BinDir is the directory where the binary is stored.
	BinDir string

	// DownloadBaseURL is the base URL for downloading Tailwind binaries.
	// If empty, GitHubReleaseURL is used.
	DownloadBaseURL string

	// HTTPClient is used for downloads. If nil, a default client is used.
	HTTPClient *http.Client

	// path is the cached path to the binary.
	path string
	mu   sync.Mutex
}

// NewBinary creates a new Binary with default settings.
func NewBinary() *Binary {
	return &Binary{
		Version:         Version,
		BinDir:          defaultBinDir(),
		DownloadBaseURL: GitHubReleaseURL,
	}
}

// NewBinaryWithVersion creates a new Binary with a specific version.
func NewBinaryWithVersion(version string) *Binary {
	return &Binary{
		Version:         version,
		BinDir:          defaultBinDir(),
		DownloadBaseURL: GitHubReleaseURL,
	}
}

// defaultBinDir returns the default binary directory (~/.vango/bin).
func defaultBinDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", DefaultBinDir)
	}
	return filepath.Join(home, DefaultBinDir)
}

// Path returns the path to the Tailwind binary, downloading if necessary.
func (b *Binary) Path() (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.path != "" {
		return b.path, nil
	}

	path := b.binaryPath()

	// Check if binary exists
	if _, err := os.Stat(path); err == nil {
		b.path = path
		return path, nil
	}

	return "", fmt.Errorf("tailwind binary not found at %s (run 'vango create' with Tailwind or download manually)", path)
}

// EnsureInstalled downloads the binary if it doesn't exist.
// Returns the path to the binary.
func (b *Binary) EnsureInstalled(ctx context.Context, progress func(msg string)) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	path := b.binaryPath()

	// Check if already installed
	if _, err := os.Stat(path); err == nil {
		b.path = path
		return path, nil
	}

	// Download
	if err := b.download(ctx, progress); err != nil {
		return "", err
	}

	b.path = path
	return path, nil
}

// IsInstalled checks if the binary is installed.
func (b *Binary) IsInstalled() bool {
	path := b.binaryPath()
	_, err := os.Stat(path)
	return err == nil
}

// binaryPath returns the path where the binary should be stored.
func (b *Binary) binaryPath() string {
	// Store per-version so upgrades don't silently keep using an older binary.
	return filepath.Join(b.BinDir, b.Version, binaryName())
}

// downloadURL returns the URL to download the binary.
func (b *Binary) downloadURL() string {
	base := b.DownloadBaseURL
	if base == "" {
		base = GitHubReleaseURL
	}
	return fmt.Sprintf("%s/%s/%s", strings.TrimRight(base, "/"), b.Version, binaryName())
}

// download downloads the binary from GitHub releases.
func (b *Binary) download(ctx context.Context, progress func(msg string)) error {
	url := b.downloadURL()

	if progress != nil {
		progress(fmt.Sprintf("Downloading Tailwind CSS %s...", b.Version))
	}

	// Create bin directory
	if err := os.MkdirAll(filepath.Dir(b.binaryPath()), 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Download
	client := b.HTTPClient
	if client == nil {
		client = &http.Client{}
	}
	if client.Timeout == 0 {
		client.Timeout = 5 * time.Minute // Large binary, allow time
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d (URL: %s)", resp.StatusCode, url)
	}

	// Create temp file first, then rename (atomic)
	tmpPath := b.binaryPath() + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	// Copy with progress
	written, err := io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write file: %w", err)
	}

	if progress != nil {
		progress(fmt.Sprintf("Downloaded %.1f MB", float64(written)/1024/1024))
	}

	// Make executable (not needed on Windows, but doesn't hurt)
	if err := os.Chmod(tmpPath, 0755); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to make executable: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, b.binaryPath()); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to install binary: %w", err)
	}

	if progress != nil {
		progress(fmt.Sprintf("Installed to %s", b.binaryPath()))
	}

	return nil
}

// Runner manages running the Tailwind CLI.
type Runner struct {
	binary     *Binary
	cmd        *exec.Cmd
	mu         sync.Mutex
	running    bool
	projectDir string
	done       chan struct{}
}

// RunnerConfig configures the Tailwind runner.
type RunnerConfig struct {
	// InputPath is the input CSS file path (relative to project).
	InputPath string

	// OutputPath is the output CSS file path (relative to project).
	OutputPath string

	// ConfigPath is the tailwind.config.js path (relative to project or absolute).
	// If empty, Tailwind uses its default config resolution.
	ConfigPath string

	// ProjectDir is the project directory.
	ProjectDir string

	// Minify enables CSS minification.
	Minify bool

	// Watch enables watch mode.
	Watch bool
}

// NewRunner creates a new Tailwind runner.
func NewRunner(binary *Binary, projectDir string) *Runner {
	return &Runner{
		binary:     binary,
		projectDir: projectDir,
	}
}

// Build runs Tailwind CSS build (one-shot).
func (r *Runner) Build(ctx context.Context, cfg RunnerConfig) error {
	path, err := r.binary.EnsureInstalled(ctx, nil)
	if err != nil {
		return err
	}

	args := []string{
		"-i", cfg.InputPath,
		"-o", cfg.OutputPath,
	}
	if cfg.ConfigPath != "" {
		args = append(args, "-c", cfg.ConfigPath)
	}
	if cfg.Minify {
		args = append(args, "--minify")
	}

	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Dir = r.projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// StartWatch starts Tailwind in watch mode.
func (r *Runner) StartWatch(ctx context.Context, cfg RunnerConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		return nil
	}

	path, err := r.binary.EnsureInstalled(ctx, nil)
	if err != nil {
		return err
	}

	args := []string{
		"-i", cfg.InputPath,
		"-o", cfg.OutputPath,
		"--watch=always",
	}
	if cfg.ConfigPath != "" {
		args = append(args, "-c", cfg.ConfigPath)
	}

	// Use exec.Command instead of CommandContext to prevent context
	// cancellation from killing Tailwind prematurely. The Stop() method
	// handles cleanup explicitly.
	r.cmd = exec.Command(path, args...)
	r.cmd.Dir = r.projectDir
	r.cmd.Stdout = os.Stdout
	r.cmd.Stderr = os.Stderr

	// Note: --watch=always flag prevents Tailwind from exiting when stdin
	// closes, so we don't need a stdin pipe. See:
	// https://github.com/tailwindlabs/tailwindcss/issues/9870

	if err := r.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start tailwind: %w", err)
	}

	r.running = true
	r.done = make(chan struct{})

	// Wait for process in background
	cmd := r.cmd
	done := r.done
	go func() {
		_ = cmd.Wait()
		close(done)
		r.mu.Lock()
		r.running = false
		if r.cmd == cmd {
			r.cmd = nil
		}
		r.done = nil
		r.mu.Unlock()
	}()

	return nil
}

// Stop stops the Tailwind watcher.
func (r *Runner) Stop() {
	r.mu.Lock()
	cmd := r.cmd
	done := r.done
	running := r.running
	r.mu.Unlock()

	if !running || cmd == nil || cmd.Process == nil {
		return
	}

	_ = cmd.Process.Kill()
	if done != nil {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
	}

	r.mu.Lock()
	r.running = false
	if r.cmd == cmd {
		r.cmd = nil
	}
	r.done = nil
	r.mu.Unlock()
}

// IsRunning returns whether Tailwind is running.
func (r *Runner) IsRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running
}
