package dev

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vango-go/vango/internal/config"
	"github.com/vango-go/vango/internal/tailwind"
	"github.com/vango-go/vango/pkg/router"
)

// ServerOptions configures the development server.
type ServerOptions struct {
	// Config is the project configuration.
	Config *config.Config

	// Verbose enables verbose logging.
	Verbose bool

	// OnBuildStart is called when a build starts.
	OnBuildStart func()

	// OnBuildComplete is called when a build completes.
	OnBuildComplete func(result BuildResult)

	// OnReload is called when browsers are reloaded.
	OnReload func(clients int)
}

// Server is the development server.
type Server struct {
	config        *config.Config
	options       ServerOptions
	compiler      *Compiler
	watcher       *Watcher
	reloadServer  *ReloadServer
	changeCh      chan Change
	tailwind      *tailwind.Runner
	errorRecovery *ErrorRecovery
	httpServer    *http.Server
	appProxy      *httputil.ReverseProxy
	mu            sync.Mutex
	running       bool
	appPort       int
	hotReload     bool
}

// NewServer creates a new development server.
func NewServer(options ServerOptions) *Server {
	cfg := options.Config
	projectDir := cfg.Dir()
	hotReload := cfg.Dev.HotReload

	// Create compiler
	compiler := NewCompiler(CompilerConfig{
		ProjectPath: projectDir,
		Tags:        cfg.Build.Tags,
		LDFlags:     cfg.Build.LDFlags,
	})

	// Create watcher
	watchPaths := CollectWatchPaths(cfg)

	watcher := NewWatcher(WatcherConfig{
		Paths:    watchPaths,
		Ignore:   append(DefaultIgnore, cfg.Dev.Ignore...),
		Debounce: 100 * time.Millisecond,
	})

	// Create reload server
	var reloadServer *ReloadServer
	if hotReload {
		reloadServer = NewReloadServer()
	}

	// Create tailwind runner if enabled
	var tw *tailwind.Runner
	if cfg.HasTailwind() {
		binary := tailwind.NewBinary()
		tw = tailwind.NewRunner(binary, projectDir)
	}

	// App will run on port + 1
	appPort := cfg.Dev.Port + 1

	// Create error recovery handler
	modulePath, _ := GetModulePath(projectDir)
	errorRecovery := NewErrorRecovery(projectDir, cfg.RoutesPath(), modulePath)

	return &Server{
		config:        cfg,
		options:       options,
		compiler:      compiler,
		watcher:       watcher,
		reloadServer:  reloadServer,
		tailwind:      tw,
		errorRecovery: errorRecovery,
		appPort:       appPort,
		hotReload:     hotReload,
	}
}

// Start starts the development server.
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	s.mu.Unlock()

	// Initial build
	s.log("Building...")
	result := s.compiler.Build(ctx)
	if !result.Success {
		s.logError("Build failed:\n%s", result.Output)
		s.notifyError(result.Output)
	} else {
		s.log("Built in %s", result.Duration.Round(time.Millisecond))
	}

	// Start tailwind if enabled
	if s.tailwind != nil {
		s.log("Starting Tailwind CSS...")
		cfg := tailwind.RunnerConfig{
			InputPath:  s.config.Tailwind.Input,
			OutputPath: s.config.Tailwind.Output,
			Watch:      true,
		}
		if cfg.InputPath == "" {
			cfg.InputPath = "app/styles/input.css"
		}
		if cfg.OutputPath == "" {
			cfg.OutputPath = "public/styles.css"
		}
		if err := s.tailwind.StartWatch(ctx, cfg); err != nil {
			s.logError("Tailwind error: %v", err)
		}
	}

	// Start the app
	if result.Success {
		if err := s.startApp(ctx); err != nil {
			s.logError("Failed to start app: %v", err)
		}
	}

	// Set up watcher callback
	s.changeCh = make(chan Change, 64)
	s.watcher.OnChange(func(change Change) {
		select {
		case s.changeCh <- change:
		default:
		}
	})

	// Start watcher in background
	go s.watcher.Start(ctx)
	go s.processChanges(ctx)

	// Set up HTTP server
	mux := http.NewServeMux()
	if s.reloadEnabled() {
		mux.HandleFunc("/_vango/reload", s.reloadServer.HandleWebSocket)
	}
	mux.HandleFunc("/", s.proxyHandler)

	s.httpServer = &http.Server{
		Addr:    s.config.DevAddress(),
		Handler: mux,
	}

	// Start HTTP server
	s.log("Server running at %s", s.config.DevURL())

	errCh := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		s.Stop()
		return nil
	case err := <-errCh:
		if err != nil {
			s.Stop()
			return err
		}
		s.Stop()
		return nil
	}
}

// Stop stops the development server.
func (s *Server) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.running = false
	s.watcher.Stop()
	s.compiler.Stop()
	if s.reloadServer != nil {
		s.reloadServer.Close()
	}

	if s.tailwind != nil {
		s.tailwind.Stop()
	}

	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.httpServer.Shutdown(ctx)
	}
}

// processChanges serializes file change handling and coalesces bursts.
func (s *Server) processChanges(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case change := <-s.changeCh:
			changes := []Change{change}
			draining := true
			for draining {
				select {
				case next := <-s.changeCh:
					changes = append(changes, next)
				default:
					draining = false
				}
			}
			s.handleChanges(ctx, changes)
		}
	}
}

// handleChanges handles a batch of file changes.
func (s *Server) handleChanges(ctx context.Context, changes []Change) {
	if len(changes) == 0 {
		return
	}

	hasGo := false
	hasCSS := false
	hasAsset := false
	hasTemplate := false

	for _, change := range changes {
		s.log("Changed: %s", change.Path)
		switch change.Type {
		case ChangeGo:
			hasGo = true
		case ChangeCSS:
			hasCSS = true
		case ChangeTemplate:
			hasTemplate = true
		case ChangeAsset:
			hasAsset = true
		}
	}

	if hasGo {
		s.handleGoChange(ctx, changes)
		return
	}

	if hasCSS {
		s.handleCSSChange(changes)
		return
	}

	if hasAsset || hasTemplate {
		s.handleAssetChange()
	}
}

func (s *Server) handleGoChange(ctx context.Context, changes []Change) {
	for _, change := range changes {
		if change.Type == ChangeGo && s.isRouteFile(change.Path) {
			s.regenerateRoutesIfNeeded()
			break
		}
	}

	if s.options.OnBuildStart != nil {
		s.options.OnBuildStart()
	}

	s.log("Rebuilding...")
	result := s.compiler.Build(ctx)

	if s.options.OnBuildComplete != nil {
		s.options.OnBuildComplete(result)
	}

	if !result.Success {
		if IsRecoverableError(result.Output) && s.errorRecovery != nil {
			s.log("Attempting automatic recovery...")
			recovery := s.errorRecovery.AttemptRecovery(result.Output)
			if recovery.Recovered {
				s.log("Recovery: %s (%s)", recovery.Action, recovery.Details)

				s.log("Retrying build...")
				result = s.compiler.Build(ctx)
				if s.options.OnBuildComplete != nil {
					s.options.OnBuildComplete(result)
				}

				if !result.Success {
					s.logError("Build still failing after recovery:\n%s", result.Output)
					s.notifyError(result.Output)
					return
				}
			} else {
				s.logError("Build failed:\n%s", result.Output)
				s.notifyError(result.Output)
				return
			}
		} else {
			s.logError("Build failed:\n%s", result.Output)
			s.notifyError(result.Output)
			return
		}
	}

	s.log("Built in %s", result.Duration.Round(time.Millisecond))
	s.clearReloadError()

	if err := s.restartApp(ctx); err != nil {
		s.logError("Failed to restart app: %v", err)
		return
	}

	time.Sleep(100 * time.Millisecond)
	s.notifyReload()
}

func (s *Server) handleCSSChange(changes []Change) {
	var cssPath string
	if s.tailwind != nil {
		outputPath := s.tailwindOutputPath()
		var publicCSS string
		for _, change := range changes {
			if change.Type != ChangeCSS {
				continue
			}
			if outputPath != "" && isSamePath(change.Path, outputPath) {
				cssPath = change.Path
				break
			}
			if publicCSS == "" && isWithinDir(change.Path, s.config.PublicPath()) {
				publicCSS = change.Path
			}
		}
		if cssPath == "" {
			if publicCSS == "" {
				s.log("CSS change detected (waiting for Tailwind output)")
				return
			}
			cssPath = publicCSS
		}
	} else {
		for _, change := range changes {
			if change.Type == ChangeCSS {
				cssPath = change.Path
				break
			}
		}
	}

	if !s.reloadEnabled() {
		s.log("CSS changed (hot reload disabled)")
		return
	}

	s.reloadServer.NotifyCSS(cssPath)
	s.log("CSS reloaded")
}

func (s *Server) handleAssetChange() {
	if !s.reloadEnabled() {
		s.log("Asset changed (hot reload disabled)")
		return
	}

	s.reloadServer.NotifyReload()
	s.log("Reloaded %d browsers", s.reloadServer.ClientCount())
}

// startApp starts the application process.
func (s *Server) startApp(ctx context.Context) error {
	// Set the port for the app
	os.Setenv("PORT", fmt.Sprintf("%d", s.appPort))
	os.Setenv("VANGO_DEV", "1")
	return s.compiler.Start(ctx)
}

// isRouteFile checks if a file path is within the routes directory.
func (s *Server) isRouteFile(path string) bool {
	return isWithinDir(path, s.config.RoutesPath())
}

// regenerateRoutesIfNeeded scans routes and regenerates routes_gen.go if the manifest changed.
// This implements the smart regeneration from Phase 14 spec section 5.5.
func (s *Server) regenerateRoutesIfNeeded() {
	routesPath := s.config.RoutesPath()
	routesGenPath := filepath.Join(routesPath, "routes_gen.go")

	// Get current routes_gen.go content
	currentContent, err := os.ReadFile(routesGenPath)
	if err != nil && !os.IsNotExist(err) {
		s.logError("Failed to read routes_gen.go: %v", err)
		return
	}

	// Scan routes
	scanner := router.NewScanner(routesPath)
	routes, err := scanner.Scan()
	if err != nil {
		s.logError("Failed to scan routes: %v", err)
		return
	}

	// Generate new routes_gen.go
	modulePath, err := GetModulePath(s.config.Dir())
	if err != nil {
		s.logError("Failed to get module path: %v", err)
		return
	}

	gen := router.NewGenerator(routes, modulePath)
	newContent, err := gen.Generate()
	if err != nil {
		s.logError("Failed to generate routes: %v", err)
		return
	}

	// Only write if content changed (deterministic output means minimal churn)
	if string(currentContent) != string(newContent) {
		if err := os.WriteFile(routesGenPath, newContent, 0644); err != nil {
			s.logError("Failed to write routes_gen.go: %v", err)
			return
		}
		s.log("Regenerated routes_gen.go")
	}
}

// restartApp restarts the application process.
func (s *Server) restartApp(ctx context.Context) error {
	s.compiler.Stop()
	return s.startApp(ctx)
}

// proxyHandler proxies requests to the app.
func (s *Server) proxyHandler(w http.ResponseWriter, r *http.Request) {
	// Check for proxy rules
	for prefix, target := range s.config.Dev.Proxy {
		if strings.HasPrefix(r.URL.Path, prefix) {
			s.proxyTo(w, r, target)
			return
		}
	}

	// Proxy to app
	s.proxyToApp(w, r)
}

// proxyToApp proxies a request to the application.
func (s *Server) proxyToApp(w http.ResponseWriter, r *http.Request) {
	target := fmt.Sprintf("http://localhost:%d", s.appPort)
	targetURL, _ := url.Parse(target)

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Modify response to inject dev client script
	proxy.ModifyResponse = func(resp *http.Response) error {
		if !s.reloadEnabled() {
			return nil
		}
		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "text/html") {
			return nil
		}

		// Read body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		resp.Body.Close()

		// Inject dev client script before </body>
		bodyStr := string(body)
		if idx := strings.LastIndex(bodyStr, "</body>"); idx != -1 {
			bodyStr = bodyStr[:idx] + DevClientScript + bodyStr[idx:]
		} else if idx := strings.LastIndex(bodyStr, "</html>"); idx != -1 {
			bodyStr = bodyStr[:idx] + DevClientScript + bodyStr[idx:]
		} else {
			bodyStr += DevClientScript
		}

		// Update response
		resp.Body = io.NopCloser(strings.NewReader(bodyStr))
		resp.ContentLength = int64(len(bodyStr))
		resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(bodyStr)))

		return nil
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusBadGateway)
		reloadScript := ""
		if s.reloadEnabled() {
			reloadScript = DevClientScript
		}
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>Vango Dev Server</title></head>
<body style="font-family: system-ui; padding: 40px; background: #1a1a1a; color: #fff;">
<h1 style="color: #ff5555;">Application Not Running</h1>
<p>The application server is not responding. This could mean:</p>
<ul>
<li>The app is still starting up</li>
<li>There was a build error (check your terminal)</li>
<li>The app crashed on startup</li>
</ul>
<p style="color: #888;">The page will automatically reload when the app is ready.</p>
%s
</body>
</html>`, reloadScript)
	}

	proxy.ServeHTTP(w, r)
}

// proxyTo proxies a request to an external target.
func (s *Server) proxyTo(w http.ResponseWriter, r *http.Request, target string) {
	targetURL, err := url.Parse(target)
	if err != nil {
		http.Error(w, "Invalid proxy target", http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.ServeHTTP(w, r)
}

// log logs a message if verbose mode is enabled.
func (s *Server) log(format string, args ...any) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("[%s] %s\n", timestamp, fmt.Sprintf(format, args...))
}

// logError logs an error message.
func (s *Server) logError(format string, args ...any) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Fprintf(os.Stderr, "[%s] %s%s%s\n", timestamp, "\033[31m", fmt.Sprintf(format, args...), "\033[0m")
}

func (s *Server) tailwindOutputPath() string {
	output := s.config.Tailwind.Output
	if output == "" {
		output = "public/styles.css"
	}
	return resolvePath(s.config.Dir(), output)
}

func (s *Server) reloadEnabled() bool {
	return s.hotReload && s.reloadServer != nil
}

func (s *Server) notifyReload() {
	if !s.reloadEnabled() {
		s.log("Hot reload disabled; rebuild complete")
		return
	}

	s.reloadServer.NotifyReload()
	if s.options.OnReload != nil {
		s.options.OnReload(s.reloadServer.ClientCount())
	}
	s.log("Reloaded %d browsers", s.reloadServer.ClientCount())
}

func (s *Server) notifyError(errMsg string) {
	if !s.reloadEnabled() {
		return
	}
	s.reloadServer.NotifyError(errMsg)
}

func (s *Server) clearReloadError() {
	if !s.reloadEnabled() {
		return
	}
	s.reloadServer.ClearError()
}

func isWithinDir(path, dir string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	absPath = filepath.Clean(absPath)
	absDir = filepath.Clean(absDir)
	if absPath == absDir {
		return true
	}
	if !strings.HasSuffix(absDir, string(os.PathSeparator)) {
		absDir += string(os.PathSeparator)
	}
	return strings.HasPrefix(absPath, absDir)
}

func isSamePath(a, b string) bool {
	absA, err := filepath.Abs(a)
	if err != nil {
		return false
	}
	absB, err := filepath.Abs(b)
	if err != nil {
		return false
	}
	return filepath.Clean(absA) == filepath.Clean(absB)
}
