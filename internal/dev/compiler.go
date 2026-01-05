package dev

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/vango-go/vango/internal/errors"
)

// CompilerConfig configures the Go compiler.
type CompilerConfig struct {
	// ProjectPath is the root directory of the project.
	ProjectPath string

	// BinaryPath is where to write the compiled binary.
	BinaryPath string

	// CachePath is where to store the Go build cache.
	CachePath string

	// Tags are build tags to pass to go build.
	Tags []string

	// LDFlags are linker flags to pass to go build.
	LDFlags string

	// Env are additional environment variables.
	Env []string
}

// BuildResult contains the result of a build.
type BuildResult struct {
	// Success indicates if the build succeeded.
	Success bool

	// Duration is how long the build took.
	Duration time.Duration

	// Output is the compiler output.
	Output string

	// Error is the build error, if any.
	Error error
}

// Compiler handles Go compilation and process management.
type Compiler struct {
	config  CompilerConfig
	process *os.Process
	mu      sync.Mutex
}

// NewCompiler creates a new Go compiler.
func NewCompiler(config CompilerConfig) *Compiler {
	if config.BinaryPath == "" {
		config.BinaryPath = filepath.Join(config.ProjectPath, ".vango", "server")
	}
	if config.CachePath == "" {
		config.CachePath = filepath.Join(config.ProjectPath, ".vango", "cache")
	}

	return &Compiler{
		config: config,
	}
}

// Build compiles the Go project.
func (c *Compiler) Build(ctx context.Context) BuildResult {
	start := time.Now()

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(c.config.BinaryPath), 0755); err != nil {
		return BuildResult{
			Duration: time.Since(start),
			Error:    errors.New("E142").Wrap(err),
		}
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(c.config.CachePath, 0755); err != nil {
		return BuildResult{
			Duration: time.Since(start),
			Error:    errors.New("E142").Wrap(err),
		}
	}

	// Build command
	args := []string{"build", "-o", c.config.BinaryPath}

	// Add tags
	if len(c.config.Tags) > 0 {
		args = append(args, "-tags", joinTags(c.config.Tags))
	}

	// Add ldflags
	if c.config.LDFlags != "" {
		args = append(args, "-ldflags", c.config.LDFlags)
	}

	// Target package
	args = append(args, ".")

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = c.config.ProjectPath

	// Set up environment
	env := os.Environ()
	env = append(env, "GOCACHE="+c.config.CachePath)
	env = append(env, c.config.Env...)
	cmd.Env = env

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start)

	output := stderr.String()
	if output == "" {
		output = stdout.String()
	}

	if err != nil {
		return BuildResult{
			Success:  false,
			Duration: duration,
			Output:   output,
			Error:    errors.New("E142").WithDetail(output).Wrap(err),
		}
	}

	return BuildResult{
		Success:  true,
		Duration: duration,
		Output:   output,
	}
}

// Start runs the compiled binary.
func (c *Compiler) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Kill existing process
	if c.process != nil {
		c.killProcess()
	}

	// Start new process
	cmd := exec.CommandContext(ctx, c.config.BinaryPath)
	cmd.Dir = c.config.ProjectPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set up environment for dev mode
	cmd.Env = append(os.Environ(), "VANGO_DEV=1")

	// Start in new process group so we can kill children
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		return errors.New("E142").Wrap(err)
	}

	c.process = cmd.Process
	return nil
}

// Stop stops the running process.
func (c *Compiler) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.killProcess()
}

// Restart stops the current process and starts a new one.
func (c *Compiler) Restart(ctx context.Context) error {
	c.Stop()
	return c.Start(ctx)
}

// killProcess kills the current process and its children.
func (c *Compiler) killProcess() {
	if c.process == nil {
		return
	}

	// Kill process group
	pgid, err := syscall.Getpgid(c.process.Pid)
	if err == nil {
		syscall.Kill(-pgid, syscall.SIGTERM)
	} else {
		c.process.Kill()
	}

	// Wait for process to exit
	done := make(chan struct{})
	go func() {
		c.process.Wait()
		close(done)
	}()

	// Give it time to gracefully shutdown
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		// Force kill
		if pgid > 0 {
			syscall.Kill(-pgid, syscall.SIGKILL)
		} else {
			c.process.Kill()
		}
		<-done
	}

	c.process = nil
}

// IsRunning returns whether the process is running.
func (c *Compiler) IsRunning() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.process != nil
}

// BinaryPath returns the path to the compiled binary.
func (c *Compiler) BinaryPath() string {
	return c.config.BinaryPath
}

// Clean removes the build cache and binary.
func (c *Compiler) Clean() error {
	c.Stop()

	if err := os.RemoveAll(c.config.CachePath); err != nil && !os.IsNotExist(err) {
		return err
	}

	if err := os.Remove(c.config.BinaryPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// joinTags joins build tags with commas.
func joinTags(tags []string) string {
	result := ""
	for i, tag := range tags {
		if i > 0 {
			result += ","
		}
		result += tag
	}
	return result
}

// BuildError represents a compilation error with context.
type BuildError struct {
	File    string
	Line    int
	Column  int
	Message string
	Output  string
}

func (e *BuildError) Error() string {
	if e.File != "" {
		return fmt.Sprintf("%s:%d:%d: %s", e.File, e.Line, e.Column, e.Message)
	}
	return e.Message
}

// Format returns a formatted error suitable for display.
func (e *BuildError) Format() string {
	return e.Output
}
