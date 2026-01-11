package build

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	vangoerrors "github.com/vango-go/vango/internal/errors"
	"github.com/vango-go/vango/internal/config"
	"github.com/vango-go/vango/internal/tailwind"
)

func mustWriteFile(t *testing.T, path string, content []byte, perm os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, content, perm); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	return b
}

func mustHashContentHex(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func systemGoPath(t *testing.T) string {
	t.Helper()
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("go not found in PATH: %v", err)
	}
	return goPath
}

func buildTestTool(t *testing.T, goPath, outPath, src string) {
	t.Helper()
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "main.go")
	mustWriteFile(t, srcPath, []byte(src), 0644)

	cmd := exec.Command(goPath, "build", "-o", outPath, srcPath)
	cmd.Dir = tmpDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build tool %q: %v\n%s", outPath, err, string(out))
	}
}

func toolPath(dir, name string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(dir, name+".exe")
	}
	return filepath.Join(dir, name)
}

func writeMinimalGoModule(t *testing.T, projectDir string) {
	t.Helper()
	mustWriteFile(t, filepath.Join(projectDir, "go.mod"), []byte("module example.com/testapp\n\ngo 1.22\n"), 0644)
	mustWriteFile(t, filepath.Join(projectDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644)
}

func writeConfigAt(t *testing.T, projectDir string, cfg *config.Config) {
	t.Helper()
	if err := cfg.SaveTo(filepath.Join(projectDir, config.ConfigFileName)); err != nil {
		t.Fatalf("SaveTo(vango.json): %v", err)
	}
}

func writePrebuiltClient(t *testing.T, projectDir string, content []byte) {
	t.Helper()
	mustWriteFile(t, filepath.Join(projectDir, "client", "dist", "vango.min.js"), content, 0644)
}

func tailwindBinaryNameForHost() string {
	name := "tailwindcss"
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return name + "-macos-arm64"
		}
		return name + "-macos-x64"
	case "linux":
		if runtime.GOARCH == "arm64" {
			return name + "-linux-arm64"
		}
		return name + "-linux-x64"
	case "windows":
		return name + "-windows-x64.exe"
	default:
		return name + "-linux-x64"
	}
}

func installFakeTailwindBinary(t *testing.T, goPath, homeDir string) string {
	t.Helper()
	binPath := filepath.Join(homeDir, ".vango", "bin", tailwind.Version, tailwindBinaryNameForHost())
	if err := os.MkdirAll(filepath.Dir(binPath), 0755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(binPath), err)
	}

	buildTestTool(t, goPath, binPath, `
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type capture struct {
	Args []string `+"`json:\"args\"`"+`
}

func main() {
	out := ""
	for i := 0; i < len(os.Args)-1; i++ {
		if os.Args[i] == "-o" {
			out = os.Args[i+1]
			break
		}
	}
	if out != "" {
		_ = os.MkdirAll(filepath.Dir(out), 0755)
		_ = os.WriteFile(out, []byte(os.Getenv("TAILWIND_CONTENT")), 0644)
	}
	if p := os.Getenv("TAILWIND_CAPTURE"); p != "" {
		b, _ := json.Marshal(capture{Args: os.Args})
		_ = os.WriteFile(p, b, 0644)
	}
}
`)

	return binPath
}

func readManifestMap(t *testing.T, outputDir string) map[string]string {
	t.Helper()
	data := mustReadFile(t, filepath.Join(outputDir, "manifest.json"))
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal(manifest.json): %v", err)
	}
	return m
}

func TestBuilder_buildGo_SetsEnvAndFlags(t *testing.T) {
	goPath := systemGoPath(t)

	projectDir := t.TempDir()
	cfg := config.New()
	writeConfigAt(t, projectDir, cfg)

	binDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	capturePath := filepath.Join(t.TempDir(), "go_capture.json")

	buildTestTool(t, goPath, toolPath(binDir, "go"), `
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type capture struct {
	Args []string          `+"`json:\"args\"`"+`
	Env  map[string]string `+"`json:\"env\"`"+`
}

func main() {
	outFile := ""
	for i := 0; i < len(os.Args)-1; i++ {
		if os.Args[i] == "-o" {
			outFile = os.Args[i+1]
			break
		}
	}

	if outFile != "" {
		_ = os.MkdirAll(filepath.Dir(outFile), 0755)
		_ = os.WriteFile(outFile, []byte("fake binary"), 0644)
	}

	cap := capture{
		Args: os.Args,
		Env: map[string]string{
			"GOOS":        os.Getenv("GOOS"),
			"GOARCH":      os.Getenv("GOARCH"),
			"CGO_ENABLED": os.Getenv("CGO_ENABLED"),
		},
	}
	b, _ := json.Marshal(cap)
	if p := os.Getenv("GO_CAPTURE"); p != "" {
		_ = os.WriteFile(p, b, 0644)
	}
	fmt.Fprint(os.Stdout, "")
}
`)

	// Only expose our fake "go" in PATH to avoid picking up real "go".
	t.Setenv("PATH", binDir)
	t.Setenv("GO_CAPTURE", capturePath)

	builder := New(cfg, Options{
		Target:  "linux/amd64",
		LDFlags: "-X main.version=1.2.3",
		Tags:    []string{"prod", "vango"},
	})

	outPath := filepath.Join(projectDir, "dist", "server")
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		t.Fatal(err)
	}

	if err := builder.buildGo(context.Background(), outPath); err != nil {
		t.Fatalf("buildGo: %v", err)
	}

	var got struct {
		Args []string          `json:"args"`
		Env  map[string]string `json:"env"`
	}
	if err := json.Unmarshal(mustReadFile(t, capturePath), &got); err != nil {
		t.Fatalf("Unmarshal(capture): %v", err)
	}

	if got.Env["GOOS"] != "linux" || got.Env["GOARCH"] != "amd64" {
		t.Fatalf("GOOS/GOARCH = %q/%q, want linux/amd64", got.Env["GOOS"], got.Env["GOARCH"])
	}
	if got.Env["CGO_ENABLED"] != "0" {
		t.Fatalf("CGO_ENABLED = %q, want 0", got.Env["CGO_ENABLED"])
	}

	argsJoined := strings.Join(got.Args, "\n")
	for _, want := range []string{"build", "-o", outPath, "-ldflags", "-X main.version=1.2.3 -s -w", "-tags", "prod,vango", "-trimpath", "."} {
		if !strings.Contains(argsJoined, want) {
			t.Fatalf("go args missing %q\nargs:\n%s", want, strings.Join(got.Args, " "))
		}
	}
}

func TestBuilder_buildGo_FailureWrapsStderr(t *testing.T) {
	goPath := systemGoPath(t)

	projectDir := t.TempDir()
	cfg := config.New()
	writeConfigAt(t, projectDir, cfg)

	binDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}

	buildTestTool(t, goPath, toolPath(binDir, "go"), `
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprint(os.Stderr, os.Getenv("GO_STDERR"))
	os.Exit(1)
}
`)

	t.Setenv("PATH", binDir)
	t.Setenv("GO_STDERR", "compiler blew up")

	builder := New(cfg, Options{})
	err := builder.buildGo(context.Background(), filepath.Join(projectDir, "dist", "server"))
	if err == nil {
		t.Fatal("expected error")
	}

	var ve *vangoerrors.VangoError
	if !errors.As(err, &ve) {
		t.Fatalf("expected VangoError, got %T", err)
	}
	if ve.Code != "E142" {
		t.Fatalf("Code = %q, want E142", ve.Code)
	}
	if !strings.Contains(ve.Detail, "compiler blew up") {
		t.Fatalf("Detail missing stderr; got %q", ve.Detail)
	}
}

func TestBuilder_bundleClient_Esbuild_HashesOutputAndRespectsFlags(t *testing.T) {
	goPath := systemGoPath(t)

	projectDir := t.TempDir()
	cfg := config.New()
	writeConfigAt(t, projectDir, cfg)

	mustWriteFile(t, filepath.Join(projectDir, "client", "src", "index.js"), []byte("console.log('src');\n"), 0644)

	binDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	capturePath := filepath.Join(t.TempDir(), "esbuild_capture.json")
	content := []byte("console.log('built');\n")

	buildTestTool(t, goPath, toolPath(binDir, "esbuild"), `
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type capture struct {
	Args []string `+"`json:\"args\"`"+`
}

func main() {
	out := ""
	for _, a := range os.Args {
		const p = "--outfile="
		if len(a) >= len(p) && a[:len(p)] == p {
			out = a[len(p):]
			break
		}
	}
	if out != "" {
		_ = os.MkdirAll(filepath.Dir(out), 0755)
		_ = os.WriteFile(out, []byte(os.Getenv("ESBUILD_CONTENT")), 0644)
	}
	if p := os.Getenv("ESBUILD_CAPTURE"); p != "" {
		b, _ := json.Marshal(capture{Args: os.Args})
		_ = os.WriteFile(p, b, 0644)
	}
}
`)

	goDir := filepath.Dir(goPath)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+goDir)
	t.Setenv("ESBUILD_CONTENT", string(content))
	t.Setenv("ESBUILD_CAPTURE", capturePath)

	builder := New(cfg, Options{Minify: true, SourceMaps: true})
	publicDir := filepath.Join(t.TempDir(), "public")
	if err := os.MkdirAll(publicDir, 0755); err != nil {
		t.Fatal(err)
	}

	key, rel, size, err := builder.bundleClient(context.Background(), publicDir)
	if err != nil {
		t.Fatalf("bundleClient: %v", err)
	}
	if key != "vango.min.js" {
		t.Fatalf("key = %q, want %q", key, "vango.min.js")
	}
	if size != int64(len(content)) {
		t.Fatalf("size = %d, want %d", size, len(content))
	}
	outPath := filepath.Join(publicDir, filepath.FromSlash(rel))
	if filepath.Base(outPath) == "vango.min.js" {
		t.Fatalf("expected hashed filename, got %q", outPath)
	}
	if !strings.HasPrefix(filepath.Base(outPath), "vango.min.") || !strings.HasSuffix(filepath.Base(outPath), ".js") {
		t.Fatalf("unexpected output filename %q", filepath.Base(outPath))
	}

	gotContent := mustReadFile(t, outPath)
	if string(gotContent) != string(content) {
		t.Fatalf("output content mismatch")
	}

	var got struct {
		Args []string `json:"args"`
	}
	if err := json.Unmarshal(mustReadFile(t, capturePath), &got); err != nil {
		t.Fatalf("Unmarshal(esbuild capture): %v", err)
	}
	argsJoined := strings.Join(got.Args, " ")
	for _, want := range []string{"--bundle", "--format=iife", "--global-name=Vango", "--minify", "--sourcemap"} {
		if !strings.Contains(argsJoined, want) {
			t.Fatalf("esbuild args missing %q\nargs: %s", want, argsJoined)
		}
	}
}

func TestBuilder_bundleClient_EsbuildFailure_FallsBackToPrebuilt(t *testing.T) {
	goPath := systemGoPath(t)

	projectDir := t.TempDir()
	cfg := config.New()
	writeConfigAt(t, projectDir, cfg)

	mustWriteFile(t, filepath.Join(projectDir, "client", "src", "index.js"), []byte("console.log('src');\n"), 0644)
	prebuilt := []byte("console.log('prebuilt');\n")
	writePrebuiltClient(t, projectDir, prebuilt)

	binDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}

	buildTestTool(t, goPath, toolPath(binDir, "esbuild"), `
package main

import "os"

func main() { os.Exit(1) }
`)

	goDir := filepath.Dir(goPath)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+goDir)

	builder := New(cfg, Options{Minify: true})
	publicDir := filepath.Join(t.TempDir(), "public")
	if err := os.MkdirAll(publicDir, 0755); err != nil {
		t.Fatal(err)
	}

	_, rel, _, err := builder.bundleClient(context.Background(), publicDir)
	if err != nil {
		t.Fatalf("bundleClient: %v", err)
	}
	outPath := filepath.Join(publicDir, filepath.FromSlash(rel))

	wantPrefix := "vango.min." + mustHashContentHex(prebuilt)[:8] + ".js"
	if filepath.Base(outPath) != wantPrefix {
		t.Fatalf("output = %q, want %q", filepath.Base(outPath), wantPrefix)
	}
}

func TestBuilder_Build_EndToEnd_NoTailwind_CopiesAssetsAndWritesManifest(t *testing.T) {
	goPath := systemGoPath(t)
	goDir := filepath.Dir(goPath)

	projectDir := t.TempDir()
	writeMinimalGoModule(t, projectDir)

	cfg := config.New()
	cfg.Build.Output = "dist"
	cfg.Tailwind.Enabled = false
	writeConfigAt(t, projectDir, cfg)

	prebuilt := []byte("console.log('prebuilt');\n")
	writePrebuiltClient(t, projectDir, prebuilt)

	// Public assets: include a nested file and a couple of assets that should be skipped (.js/.css).
	mustWriteFile(t, filepath.Join(projectDir, "public", "images", "logo.png"), []byte("png-bytes"), 0644)
	mustWriteFile(t, filepath.Join(projectDir, "public", "app.js"), []byte("should-not-copy"), 0644)
	mustWriteFile(t, filepath.Join(projectDir, "public", "styles.css"), []byte("should-not-copy"), 0644)

	// Ensure we can run "go", but force the client bundler/Tailwind paths into fallback mode.
	t.Setenv("PATH", goDir)

	builder := New(cfg, Options{})
	res, err := builder.Build(context.Background())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	outputDir := cfg.OutputPath()
	binPath := filepath.Join(outputDir, "server")
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}
	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("missing binary at %q: %v", binPath, err)
	}
	if res.Public != filepath.Join(outputDir, "public") {
		t.Fatalf("Public = %q, want %q", res.Public, filepath.Join(outputDir, "public"))
	}

	manifest := readManifestMap(t, outputDir)
	if got := manifest["vango.min.js"]; got == "" || got == "vango.min.js" {
		t.Fatalf("manifest vango.min.js missing hashed output, got %q", got)
	}
	if _, ok := manifest["styles.css"]; ok {
		t.Fatalf("did not expect styles.css in manifest when Tailwind disabled")
	}

	// Verify assets were fingerprinted and .js/.css were skipped.
	if _, ok := manifest["app.js"]; ok {
		t.Fatalf("did not expect app.js in manifest")
	}
	if _, ok := manifest["styles.css"]; ok {
		t.Fatalf("did not expect public/styles.css in manifest")
	}

	assetKey := filepath.ToSlash(filepath.Join("images", "logo.png"))
	gotAsset := manifest[assetKey]
	if gotAsset == "" || !strings.HasPrefix(gotAsset, "assets/") || !strings.Contains(gotAsset, "logo.") || !strings.HasSuffix(gotAsset, ".png") {
		t.Fatalf("unexpected manifest entry for %q: %q", assetKey, gotAsset)
	}

	copiedAssetPath := filepath.Join(outputDir, "public", filepath.FromSlash(gotAsset))
	if _, err := os.Stat(copiedAssetPath); err != nil {
		t.Fatalf("missing copied asset at %q: %v", copiedAssetPath, err)
	}
}

func TestBuilder_Build_WithTailwind_UsesEsbuildAndFingerprintsCSS(t *testing.T) {
	goPath := systemGoPath(t)
	goDir := filepath.Dir(goPath)

	projectDir := t.TempDir()
	writeMinimalGoModule(t, projectDir)

	cfg := config.New()
	cfg.Build.Output = "dist"
	cfg.Tailwind.Enabled = true
	cfg.Tailwind.Input = "app/styles/input.css"
	cfg.Tailwind.Output = "public/styles.css"
	writeConfigAt(t, projectDir, cfg)

	mustWriteFile(t, filepath.Join(projectDir, "client", "src", "index.js"), []byte("console.log('src');\n"), 0644)
	mustWriteFile(t, filepath.Join(projectDir, "tailwind.config.js"), []byte("export default {};\n"), 0644)
	mustWriteFile(t, filepath.Join(projectDir, "app", "styles", "input.css"), []byte(`@import "tailwindcss";`+"\n"), 0644)

	binDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}

	esbuildContent := []byte("console.log('built');\n")
	tailwindContent := []byte("/* compiled */\n")

	buildTestTool(t, goPath, toolPath(binDir, "esbuild"), `
package main

import (
	"os"
	"path/filepath"
)

func main() {
	out := ""
	for _, a := range os.Args {
		const p = "--outfile="
		if len(a) >= len(p) && a[:len(p)] == p {
			out = a[len(p):]
			break
		}
	}
	if out != "" {
		_ = os.MkdirAll(filepath.Dir(out), 0755)
		_ = os.WriteFile(out, []byte(os.Getenv("ESBUILD_CONTENT")), 0644)
	}
}
`)

	t.Setenv("PATH", binDir+string(os.PathListSeparator)+goDir)
	t.Setenv("ESBUILD_CONTENT", string(esbuildContent))
	t.Setenv("TAILWIND_CONTENT", string(tailwindContent))

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", homeDir)
	}
	capturePath := filepath.Join(t.TempDir(), "tailwind_capture.json")
	t.Setenv("TAILWIND_CAPTURE", capturePath)
	_ = installFakeTailwindBinary(t, goPath, homeDir)

	builder := New(cfg, Options{Minify: true})
	res, err := builder.Build(context.Background())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	outputDir := cfg.OutputPath()
	manifest := readManifestMap(t, outputDir)

	gotJS := manifest["vango.min.js"]
	if gotJS == "" || gotJS == "vango.min.js" {
		t.Fatalf("manifest vango.min.js missing fingerprint, got %q", gotJS)
	}
	gotCSS := manifest["styles.css"]
	if gotCSS == "" || gotCSS == "styles.css" {
		t.Fatalf("manifest styles.css missing fingerprint, got %q", gotCSS)
	}

	if res.ClientSize != int64(len(esbuildContent)) {
		t.Fatalf("ClientSize = %d, want %d", res.ClientSize, len(esbuildContent))
	}
	if res.CSSSize != int64(len(tailwindContent)) {
		t.Fatalf("CSSSize = %d, want %d", res.CSSSize, len(tailwindContent))
	}

	jsPath := filepath.Join(outputDir, "public", gotJS)
	cssPath := filepath.Join(outputDir, "public", gotCSS)
	if string(mustReadFile(t, jsPath)) != string(esbuildContent) {
		t.Fatalf("fingerprinted JS content mismatch")
	}
	if string(mustReadFile(t, cssPath)) != string(tailwindContent) {
		t.Fatalf("fingerprinted CSS content mismatch")
	}

	var gotTW struct {
		Args []string `json:"args"`
	}
	if err := json.Unmarshal(mustReadFile(t, capturePath), &gotTW); err != nil {
		t.Fatalf("Unmarshal(tailwind capture): %v", err)
	}
	argsJoined := strings.Join(gotTW.Args, " ")
	if !strings.Contains(argsJoined, "-c "+filepath.Join(projectDir, "tailwind.config.js")) {
		t.Fatalf("expected -c tailwind.config.js in args; got %s", argsJoined)
	}
}

func TestBuilder_compileTailwind_MissingBinary_ReturnsE123(t *testing.T) {
	goPath := systemGoPath(t)

	projectDir := t.TempDir()
	cfg := config.New()
	cfg.Tailwind.Enabled = true
	cfg.Tailwind.Input = "app/styles/input.css"
	cfg.Tailwind.Output = "public/styles.css"
	writeConfigAt(t, projectDir, cfg)

	mustWriteFile(t, filepath.Join(projectDir, "app", "styles", "input.css"), []byte(`@import "tailwindcss";`+"\n"), 0644)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", homeDir)
	}

	publicDir := filepath.Join(t.TempDir(), "public")
	if err := os.MkdirAll(publicDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Make sure we don't accidentally pick up a user-installed tailwind binary.
	t.Setenv("PATH", filepath.Dir(goPath))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	builder := New(cfg, Options{})
	_, _, _, err := builder.compileTailwind(ctx, publicDir)
	if err == nil {
		t.Fatal("expected error")
	}

	var ve *vangoerrors.VangoError
	if !errors.As(err, &ve) {
		t.Fatalf("expected VangoError, got %T", err)
	}
	if ve.Code != "E123" {
		t.Fatalf("Code = %q, want E123", ve.Code)
	}
}
