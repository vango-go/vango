package build

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/vango-go/vango/internal/config"
	"github.com/vango-go/vango/internal/errors"
)

// Result contains the build output.
type Result struct {
	// Duration is how long the build took.
	Duration time.Duration

	// Binary is the path to the compiled Go binary.
	Binary string

	// Public is the path to the public directory.
	Public string

	// Manifest is the asset manifest.
	Manifest map[string]string

	// ClientSize is the size of the thin client in bytes.
	ClientSize int64

	// ClientGzipSize is the gzipped size of the thin client.
	ClientGzipSize int64

	// CSSSize is the size of the CSS in bytes.
	CSSSize int64
}

// Options configures the builder.
type Options struct {
	// Minify enables minification.
	Minify bool

	// SourceMaps enables source map generation.
	SourceMaps bool

	// Target is the Go build target (e.g., "linux/amd64").
	Target string

	// LDFlags are linker flags for go build.
	LDFlags string

	// Tags are build tags.
	Tags []string

	// Verbose enables verbose output.
	Verbose bool

	// OnProgress is called with progress updates.
	OnProgress func(step string)
}

// Builder handles production builds.
type Builder struct {
	config  *config.Config
	options Options
}

// New creates a new builder.
func New(cfg *config.Config, options Options) *Builder {
	// Apply config defaults to options
	if !options.Minify && cfg.Build.Minify {
		options.Minify = true
	}
	if !options.SourceMaps && cfg.Build.SourceMaps {
		options.SourceMaps = true
	}
	if options.Target == "" && cfg.Build.Target != "" {
		options.Target = cfg.Build.Target
	}
	if options.LDFlags == "" && cfg.Build.LDFlags != "" {
		options.LDFlags = cfg.Build.LDFlags
	}
	if len(options.Tags) == 0 && len(cfg.Build.Tags) > 0 {
		options.Tags = cfg.Build.Tags
	}

	return &Builder{
		config:  cfg,
		options: options,
	}
}

// Build performs a production build.
func (b *Builder) Build(ctx context.Context) (*Result, error) {
	start := time.Now()
	result := &Result{
		Manifest: make(map[string]string),
	}

	outputDir := b.config.OutputPath()
	publicDir := filepath.Join(outputDir, "public")

	// Clean output directory
	b.progress("Cleaning output directory...")
	if err := os.RemoveAll(outputDir); err != nil {
		return nil, errors.New("E142").Wrap(err)
	}
	if err := os.MkdirAll(publicDir, 0755); err != nil {
		return nil, errors.New("E142").Wrap(err)
	}

	// Build Go binary
	b.progress("Compiling Go...")
	binaryPath := filepath.Join(outputDir, "server")
	if err := b.buildGo(ctx, binaryPath); err != nil {
		return nil, err
	}
	result.Binary = binaryPath

	// Bundle thin client
	b.progress("Bundling thin client...")
	clientPath, size, err := b.bundleClient(ctx, publicDir)
	if err != nil {
		return nil, err
	}
	result.ClientSize = size
	result.Manifest["vango.min.js"] = filepath.Base(clientPath)

	// Compile Tailwind if enabled
	if b.config.HasTailwind() {
		b.progress("Compiling Tailwind CSS...")
		cssPath, size, err := b.compileTailwind(ctx, publicDir)
		if err != nil {
			return nil, err
		}
		result.CSSSize = size
		result.Manifest["styles.css"] = filepath.Base(cssPath)
	}

	// Copy static assets
	b.progress("Copying static assets...")
	if err := b.copyAssets(publicDir, result.Manifest); err != nil {
		return nil, err
	}

	// Write manifest
	b.progress("Writing manifest...")
	if err := b.writeManifest(outputDir, result.Manifest); err != nil {
		return nil, err
	}

	result.Duration = time.Since(start)
	result.Public = publicDir

	return result, nil
}

// buildGo compiles the Go binary.
func (b *Builder) buildGo(ctx context.Context, output string) error {
	args := []string{"build", "-o", output}

	// Add ldflags for smaller binary
	ldflags := "-s -w"
	if b.options.LDFlags != "" {
		ldflags = b.options.LDFlags + " " + ldflags
	}
	args = append(args, "-ldflags", ldflags)

	// Add tags
	if len(b.options.Tags) > 0 {
		tags := strings.Join(b.options.Tags, ",")
		args = append(args, "-tags", tags)
	}

	// Trimpath for reproducible builds
	args = append(args, "-trimpath")

	// Target package
	args = append(args, ".")

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = b.config.Dir()

	// Set up environment
	env := os.Environ()
	if b.options.Target != "" {
		parts := strings.Split(b.options.Target, "/")
		if len(parts) == 2 {
			env = append(env, "GOOS="+parts[0])
			env = append(env, "GOARCH="+parts[1])
		}
	}
	env = append(env, "CGO_ENABLED=0")
	cmd.Env = env

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return errors.New("E142").
			WithDetail(stderr.String()).
			Wrap(err)
	}

	return nil
}

// bundleClient bundles the thin client JavaScript.
func (b *Builder) bundleClient(ctx context.Context, publicDir string) (string, int64, error) {
	clientSrc := filepath.Join(b.config.Dir(), "client", "src", "index.js")

	// Check if custom client exists, otherwise use default
	if _, err := os.Stat(clientSrc); os.IsNotExist(err) {
		// Use embedded default client (for now, just create a placeholder)
		clientSrc = ""
	}

	// Check if esbuild is available
	esbuildPath, err := exec.LookPath("esbuild")
	if err != nil {
		// Try npx esbuild
		if _, err := exec.LookPath("npx"); err != nil {
			// Create a minimal fallback client
			return b.createFallbackClient(publicDir)
		}
		esbuildPath = "npx"
	}

	// Determine output file
	var outputFile string
	if b.options.Minify {
		outputFile = filepath.Join(publicDir, "vango.min.js")
	} else {
		outputFile = filepath.Join(publicDir, "vango.js")
	}

	// If we have esbuild and source file
	if clientSrc != "" && esbuildPath != "" {
		args := []string{}
		if esbuildPath == "npx" {
			args = append(args, "esbuild")
		}
		args = append(args,
			clientSrc,
			"--bundle",
			"--format=iife",
			"--global-name=Vango",
			"--outfile="+outputFile,
		)

		if b.options.Minify {
			args = append(args, "--minify")
		}

		if b.options.SourceMaps {
			args = append(args, "--sourcemap")
		}

		cmd := exec.CommandContext(ctx, esbuildPath, args...)
		cmd.Dir = b.config.Dir()

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			// Fall back to basic copy
			return b.createFallbackClient(publicDir)
		}
	} else {
		return b.createFallbackClient(publicDir)
	}

	// Get file size
	info, err := os.Stat(outputFile)
	if err != nil {
		return outputFile, 0, nil
	}

	// Add hash to filename
	hash, _ := hashFile(outputFile)
	if hash != "" {
		ext := filepath.Ext(outputFile)
		base := strings.TrimSuffix(filepath.Base(outputFile), ext)
		hashedName := fmt.Sprintf("%s.%s%s", base, hash[:8], ext)
		hashedPath := filepath.Join(publicDir, hashedName)
		os.Rename(outputFile, hashedPath)
		outputFile = hashedPath
	}

	return outputFile, info.Size(), nil
}

// createFallbackClient creates a minimal client when bundling isn't available.
func (b *Builder) createFallbackClient(publicDir string) (string, int64, error) {
	// Check if there's a pre-built client
	prebuiltPath := filepath.Join(b.config.Dir(), "client", "dist", "vango.min.js")
	if _, err := os.Stat(prebuiltPath); err == nil {
		// Copy pre-built client
		outputFile := filepath.Join(publicDir, "vango.min.js")
		if err := copyFile(prebuiltPath, outputFile); err != nil {
			return "", 0, err
		}

		info, _ := os.Stat(outputFile)
		size := int64(0)
		if info != nil {
			size = info.Size()
		}

		// Add hash
		hash, _ := hashFile(outputFile)
		if hash != "" {
			hashedName := fmt.Sprintf("vango.%s.min.js", hash[:8])
			hashedPath := filepath.Join(publicDir, hashedName)
			os.Rename(outputFile, hashedPath)
			outputFile = hashedPath
		}

		return outputFile, size, nil
	}

	// Create minimal placeholder
	outputFile := filepath.Join(publicDir, "vango.min.js")
	content := []byte(`// Vango thin client - placeholder
console.warn("Vango client not bundled. Run 'npm run build' in client/ directory.");
`)
	if err := os.WriteFile(outputFile, content, 0644); err != nil {
		return "", 0, err
	}

	return outputFile, int64(len(content)), nil
}

// compileTailwind compiles Tailwind CSS.
func (b *Builder) compileTailwind(ctx context.Context, publicDir string) (string, int64, error) {
	// Check if npx is available
	if _, err := exec.LookPath("npx"); err != nil {
		return "", 0, errors.New("E123").
			WithDetail("npx is required for Tailwind CSS").
			WithSuggestion("Install Node.js from https://nodejs.org")
	}

	inputFile := b.config.Tailwind.Input
	if inputFile == "" {
		inputFile = filepath.Join(b.config.Dir(), "app", "styles", "input.css")
	} else if !filepath.IsAbs(inputFile) {
		inputFile = filepath.Join(b.config.Dir(), inputFile)
	}

	// Check if input exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		// Create minimal input file
		os.MkdirAll(filepath.Dir(inputFile), 0755)
		content := `@tailwind base;
@tailwind components;
@tailwind utilities;
`
		os.WriteFile(inputFile, []byte(content), 0644)
	}

	outputFile := filepath.Join(publicDir, "styles.css")

	args := []string{
		"tailwindcss",
		"-i", inputFile,
		"-o", outputFile,
		"--minify",
	}

	configPath := b.config.TailwindConfigPath()
	if _, err := os.Stat(configPath); err == nil {
		args = append(args, "-c", configPath)
	}

	cmd := exec.CommandContext(ctx, "npx", args...)
	cmd.Dir = b.config.Dir()

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", 0, errors.New("E123").
			WithDetail(stderr.String()).
			Wrap(err)
	}

	// Get file size
	info, err := os.Stat(outputFile)
	if err != nil {
		return outputFile, 0, nil
	}

	// Add hash to filename
	hash, _ := hashFile(outputFile)
	if hash != "" {
		hashedName := fmt.Sprintf("styles.%s.css", hash[:8])
		hashedPath := filepath.Join(publicDir, hashedName)
		os.Rename(outputFile, hashedPath)
		outputFile = hashedPath
	}

	return outputFile, info.Size(), nil
}

// copyAssets copies static assets with cache busting.
func (b *Builder) copyAssets(publicDir string, manifest map[string]string) error {
	srcDir := b.config.PublicPath()
	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		return nil // No public directory
	}

	assetsDir := filepath.Join(publicDir, "assets")
	os.MkdirAll(assetsDir, 0755)

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Get relative path
		relPath, _ := filepath.Rel(srcDir, path)

		// Skip already processed files
		ext := strings.ToLower(filepath.Ext(relPath))
		if ext == ".js" || ext == ".css" {
			// These are handled separately
			return nil
		}

		// Copy with hash
		hash, _ := hashFile(path)
		base := strings.TrimSuffix(filepath.Base(relPath), ext)
		hashedName := fmt.Sprintf("%s.%s%s", base, hash[:8], ext)
		destPath := filepath.Join(assetsDir, hashedName)

		// Ensure destination directory exists
		os.MkdirAll(filepath.Dir(destPath), 0755)

		if err := copyFile(path, destPath); err != nil {
			return err
		}

		// Add to manifest
		manifest[relPath] = "assets/" + hashedName

		return nil
	})
}

// writeManifest writes the asset manifest.
func (b *Builder) writeManifest(outputDir string, manifest map[string]string) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}

	manifestPath := filepath.Join(outputDir, "manifest.json")
	return os.WriteFile(manifestPath, data, 0644)
}

// progress reports build progress.
func (b *Builder) progress(step string) {
	if b.options.OnProgress != nil {
		b.options.OnProgress(step)
	}
}

// hashFile returns the SHA256 hash of a file.
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// copyFile copies a file.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// Clean removes the build output directory.
func (b *Builder) Clean() error {
	return os.RemoveAll(b.config.OutputPath())
}
