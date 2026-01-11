package tailwind

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func buildTestTool(t *testing.T, outPath, src string) {
	t.Helper()

	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("go not found in PATH: %v", err)
	}

	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatalf("WriteFile(%q): %v", srcPath, err)
	}

	cmd := exec.Command(goPath, "build", "-o", outPath, srcPath)
	cmd.Dir = tmpDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build tool %q: %v\n%s", outPath, err, string(out))
	}
}

func TestBinary_binaryPath_IncludesVersionDir(t *testing.T) {
	b := &Binary{
		Version: "vTEST",
		BinDir:  filepath.Join(t.TempDir(), "bin"),
	}

	got := b.binaryPath()
	if !strings.Contains(got, string(filepath.Separator)+"vTEST"+string(filepath.Separator)) {
		t.Fatalf("binaryPath = %q, expected it to include version dir %q", got, "vTEST")
	}
}

func TestBinary_Path_CachesResolvedPath(t *testing.T) {
	binDir := t.TempDir()
	b := &Binary{Version: "vTEST", BinDir: binDir}

	path := b.binaryPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte("bin"), 0755); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}

	p1, err := b.Path()
	if err != nil {
		t.Fatalf("Path(): %v", err)
	}
	p2, err := b.Path()
	if err != nil {
		t.Fatalf("Path() second call: %v", err)
	}
	if p1 != p2 {
		t.Fatalf("Path() = %q then %q, expected cached same value", p1, p2)
	}
}

func TestBinary_Path_Missing_ReturnsError(t *testing.T) {
	b := &Binary{Version: "vTEST", BinDir: t.TempDir()}
	_, err := b.Path()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBinary_IsInstalled(t *testing.T) {
	binDir := t.TempDir()
	b := &Binary{Version: "vTEST", BinDir: binDir}
	if b.IsInstalled() {
		t.Fatal("expected IsInstalled() false before writing binary")
	}

	path := b.binaryPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte("bin"), 0755); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
	if !b.IsInstalled() {
		t.Fatal("expected IsInstalled() true after writing binary")
	}
}

func TestBinary_EnsureInstalled_UsesBaseURLAndProgress(t *testing.T) {
	var requests atomic.Int64
	wantBody := []byte("fake-binary-bytes")

	binDir := t.TempDir()
	b := &Binary{
		Version:          "vTEST",
		BinDir:           binDir,
		DownloadBaseURL:  "https://example.test/releases/download",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requests.Add(1)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(wantBody)),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		})},
	}

	var progress []string
	gotPath, err := b.EnsureInstalled(context.Background(), func(msg string) {
		progress = append(progress, msg)
	})
	if err != nil {
		t.Fatalf("EnsureInstalled: %v", err)
	}

	if requests.Load() != 1 {
		t.Fatalf("requests = %d, want 1", requests.Load())
	}

	// Should install to the versioned path.
	if gotPath != b.binaryPath() {
		t.Fatalf("path = %q, want %q", gotPath, b.binaryPath())
	}

	gotBytes, err := os.ReadFile(gotPath)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", gotPath, err)
	}
	if string(gotBytes) != string(wantBody) {
		t.Fatalf("installed bytes mismatch: got %q want %q", string(gotBytes), string(wantBody))
	}

	if len(progress) < 2 {
		t.Fatalf("expected progress messages, got %v", progress)
	}

	// Second call should be a no-op.
	progress = nil
	gotPath2, err := b.EnsureInstalled(context.Background(), func(msg string) {
		progress = append(progress, msg)
	})
	if err != nil {
		t.Fatalf("EnsureInstalled second: %v", err)
	}
	if gotPath2 != gotPath {
		t.Fatalf("path second = %q, want %q", gotPath2, gotPath)
	}
	if requests.Load() != 1 {
		t.Fatalf("requests after second call = %d, want 1", requests.Load())
	}
	if len(progress) != 0 {
		t.Fatalf("expected no progress messages when already installed, got %v", progress)
	}
}

func TestBinary_EnsureInstalled_NonOKStatus_ReturnsError(t *testing.T) {
	b := &Binary{
		Version:          "vTEST",
		BinDir:           t.TempDir(),
		DownloadBaseURL:  "https://example.test/releases/download",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader("nope")),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		})},
	}

	_, err := b.EnsureInstalled(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "download failed with status 404") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBinary_downloadURL_DefaultsToGitHub(t *testing.T) {
	b := &Binary{Version: "vTEST", BinDir: t.TempDir()}
	url := b.downloadURL()
	if !strings.HasPrefix(url, GitHubReleaseURL+"/") {
		t.Fatalf("downloadURL = %q, want prefix %q", url, GitHubReleaseURL+"/")
	}
}

func TestRunner_Build_And_Watch_Lifecycle(t *testing.T) {
	if runtime.GOOS == "windows" {
		// This test compiles and runs a helper executable; it should work on Windows too,
		// but we keep the runtime short and stable across platforms.
	}

	projectDir := t.TempDir()
	captureBuild := filepath.Join(projectDir, "capture_build.json")
	captureWatch := filepath.Join(projectDir, "capture_watch.json")

	b := &Binary{
		Version: "vTEST",
		BinDir:  t.TempDir(),
	}
	if err := os.MkdirAll(filepath.Dir(b.binaryPath()), 0755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(b.binaryPath()), err)
	}

	buildTestTool(t, b.binaryPath(), `
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type capture struct {
	Args []string `+"`json:\"args\"`"+`
	Cwd  string   `+"`json:\"cwd\"`"+`
	Out  string   `+"`json:\"out\"`"+`
}

func main() {
	capPath := os.Getenv("CAPTURE_PATH")
	wd, _ := os.Getwd()
	out := ""
	for i := 0; i < len(os.Args)-1; i++ {
		if os.Args[i] == "-o" {
			out = os.Args[i+1]
			break
		}
	}
	if out != "" {
		_ = os.MkdirAll(filepath.Dir(out), 0755)
		_ = os.WriteFile(out, []byte("/* compiled */\n"), 0644)
	}
	if capPath != "" {
		_ = os.WriteFile(capPath, []byte{}, 0644)
		b, _ := json.Marshal(capture{Args: os.Args[1:], Cwd: wd, Out: out})
		_ = os.WriteFile(capPath, b, 0644)
	}
	for _, a := range os.Args[1:] {
		if strings.HasPrefix(a, "--watch") {
			for {
				time.Sleep(200 * time.Millisecond)
			}
		}
	}
}
`)

	r := NewRunner(b, projectDir)

	// One-shot build.
	t.Setenv("CAPTURE_PATH", captureBuild)
	outCSS := filepath.Join(projectDir, "public", "styles.css")
	cfg := RunnerConfig{
		InputPath:  "app/styles/input.css",
		OutputPath: "public/styles.css",
		ConfigPath: "tailwind.config.js",
		Minify:     true,
	}
	if err := os.MkdirAll(filepath.Join(projectDir, "app", "styles"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "app", "styles", "input.css"), []byte(`@import "tailwindcss";`+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "tailwind.config.js"), []byte("export default {};\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := r.Build(context.Background(), cfg); err != nil {
		t.Fatalf("Build: %v", err)
	}
	if _, err := os.Stat(outCSS); err != nil {
		t.Fatalf("expected output css at %q: %v", outCSS, err)
	}

	type toolCapture struct {
		Args []string `json:"args"`
		Cwd  string   `json:"cwd"`
		Out  string   `json:"out"`
	}

	var got toolCapture
	if err := json.Unmarshal(mustReadFile(t, captureBuild), &got); err != nil {
		t.Fatalf("Unmarshal capture: %v", err)
	}
	argsJoined := strings.Join(got.Args, " ")
	for _, want := range []string{"-i app/styles/input.css", "-o public/styles.css", "-c tailwind.config.js", "--minify"} {
		if !strings.Contains(argsJoined, want) {
			t.Fatalf("missing %q in args: %s", want, argsJoined)
		}
	}
	if got.Cwd != projectDir {
		t.Fatalf("cwd = %q, want %q", got.Cwd, projectDir)
	}

	// Watch mode start/stop.
	t.Setenv("CAPTURE_PATH", captureWatch)
	if err := r.StartWatch(context.Background(), RunnerConfig{
		InputPath:  "app/styles/input.css",
		OutputPath: "public/styles.css",
		ConfigPath: "tailwind.config.js",
		Watch:      true,
	}); err != nil {
		t.Fatalf("StartWatch: %v", err)
	}
	if !r.IsRunning() {
		t.Fatalf("expected runner to be running")
	}

	// Give the helper process a moment to write its capture file before stopping it.
	deadline := time.Now().Add(1 * time.Second)
	for {
		if _, err := os.Stat(captureWatch); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected watch capture file at %q", captureWatch)
		}
		time.Sleep(10 * time.Millisecond)
	}

	r.Stop()
	deadline = time.Now().Add(2 * time.Second)
	for r.IsRunning() && time.Now().Before(deadline) {
		time.Sleep(25 * time.Millisecond)
	}
	if r.IsRunning() {
		t.Fatalf("expected runner to stop")
	}

	var gotW toolCapture
	if err := json.Unmarshal(mustReadFile(t, captureWatch), &gotW); err != nil {
		t.Fatalf("Unmarshal watch capture: %v", err)
	}
	argsJoined = strings.Join(gotW.Args, " ")
	for _, want := range []string{"--watch=always", "-c tailwind.config.js"} {
		if !strings.Contains(argsJoined, want) {
			t.Fatalf("missing %q in watch args: %s", want, argsJoined)
		}
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

func TestRunner_Stop_NoOpWhenNotRunning(t *testing.T) {
	r := NewRunner(&Binary{Version: "vTEST", BinDir: t.TempDir()}, t.TempDir())
	r.Stop()
}

func TestRunner_Build_DownloadError_ReturnsError(t *testing.T) {
	b := &Binary{
		Version:         "vTEST",
		BinDir:          t.TempDir(),
		DownloadBaseURL: "https://example.test/releases/download",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("boom")
		})},
	}
	r := NewRunner(b, t.TempDir())

	err := r.Build(context.Background(), RunnerConfig{
		InputPath:  "in.css",
		OutputPath: "out.css",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBinary_downloadURL_TrimsTrailingSlash(t *testing.T) {
	b := &Binary{
		Version:         "vTEST",
		BinDir:          t.TempDir(),
		DownloadBaseURL: "https://example.com/releases/download/",
	}
	url := b.downloadURL()
	if strings.Contains(url, "download//vTEST") {
		t.Fatalf("downloadURL has double slash: %q", url)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
