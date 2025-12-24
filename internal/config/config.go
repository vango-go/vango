package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/vango-dev/vango/v2/internal/errors"
)

const (
	// ConfigFileName is the name of the configuration file.
	ConfigFileName = "vango.json"

	// DefaultPort is the default development server port.
	DefaultPort = 3000

	// DefaultHost is the default development server host.
	DefaultHost = "localhost"

	// DefaultOutput is the default build output directory.
	DefaultOutput = "dist"

	// DefaultRegistry is the default component registry URL.
	DefaultRegistry = "https://vango.dev/registry.json"
)

// Config represents the complete vango.json configuration.
// This matches the Phase 14 specification schema.
type Config struct {
	// Name is the project name.
	Name string `json:"name,omitempty"`

	// Version is the project version.
	Version string `json:"version,omitempty"`

	// Port is the default server port (convenience field, also in Dev).
	Port int `json:"port,omitempty"`

	// Paths contains path configuration for various directories.
	Paths PathsConfig `json:"paths,omitempty"`

	// Static contains static file serving configuration.
	Static StaticConfig `json:"static,omitempty"`

	// Dev contains development server configuration.
	Dev DevConfig `json:"dev,omitempty"`

	// Build contains production build configuration.
	Build BuildConfig `json:"build,omitempty"`

	// Tailwind contains Tailwind CSS configuration.
	Tailwind TailwindConfig `json:"tailwind,omitempty"`

	// Session contains session configuration.
	Session SessionConfig `json:"session,omitempty"`

	// UI contains VangoUI component configuration (legacy, use Paths.UI).
	UI UIConfig `json:"ui,omitempty"`

	// Hooks is the path to custom client hooks JavaScript file.
	Hooks string `json:"hooks,omitempty"`

	// Routes is the path to the routes directory (legacy, use Paths.Routes).
	Routes string `json:"routes,omitempty"`

	// Components is the path to the components directory (legacy, use Paths.Components).
	Components string `json:"components,omitempty"`

	// Public is the path to the public static files directory (legacy, use Static.Dir).
	Public string `json:"public,omitempty"`

	// configPath stores the path where the config was loaded from.
	configPath string
}

// PathsConfig contains path configuration for project directories.
type PathsConfig struct {
	// Routes is the path to the routes directory.
	Routes string `json:"routes,omitempty"`

	// Components is the path to the components directory.
	Components string `json:"components,omitempty"`

	// UI is the path to the UI components directory.
	UI string `json:"ui,omitempty"`

	// Store is the path to the store directory.
	Store string `json:"store,omitempty"`

	// Middleware is the path to the middleware directory.
	Middleware string `json:"middleware,omitempty"`
}

// StaticConfig contains static file serving configuration.
type StaticConfig struct {
	// Dir is the directory containing static files.
	Dir string `json:"dir,omitempty"`

	// Prefix is the URL prefix for static files (default: "/").
	Prefix string `json:"prefix,omitempty"`
}

// SessionConfig contains session configuration.
type SessionConfig struct {
	// ResumeWindow is the duration for session resumption (e.g., "30s").
	ResumeWindow string `json:"resumeWindow,omitempty"`
}

// DevConfig contains development server settings.
type DevConfig struct {
	// Port is the port to run the dev server on.
	Port int `json:"port,omitempty"`

	// Host is the host to bind to.
	Host string `json:"host,omitempty"`

	// OpenBrowser opens the browser automatically on start.
	OpenBrowser bool `json:"openBrowser,omitempty"`

	// HTTPS enables HTTPS for the dev server.
	HTTPS bool `json:"https,omitempty"`

	// Proxy contains proxy rules for forwarding requests.
	Proxy map[string]string `json:"proxy,omitempty"`

	// Watch contains paths to watch for changes.
	Watch []string `json:"watch,omitempty"`

	// Ignore contains patterns to ignore during watch.
	Ignore []string `json:"ignore,omitempty"`

	// HotReload enables hot reload in development.
	HotReload bool `json:"hotReload,omitempty"`
}

// BuildConfig contains production build settings.
type BuildConfig struct {
	// Output is the output directory for builds.
	Output string `json:"output,omitempty"`

	// Minify enables minification.
	Minify bool `json:"minify,omitempty"`

	// MinifyAssets enables minification of CSS and JS assets.
	MinifyAssets bool `json:"minifyAssets,omitempty"`

	// StripSymbols strips debug symbols from the binary (-ldflags="-s -w").
	StripSymbols bool `json:"stripSymbols,omitempty"`

	// SourceMaps enables source map generation.
	SourceMaps bool `json:"sourceMaps,omitempty"`

	// Target is the Go build target (e.g., "linux/amd64").
	Target string `json:"target,omitempty"`

	// LDFlags are additional linker flags for go build.
	LDFlags string `json:"ldflags,omitempty"`

	// Tags are build tags to pass to go build.
	Tags []string `json:"tags,omitempty"`
}

// TailwindConfig contains Tailwind CSS settings.
type TailwindConfig struct {
	// Enabled controls whether Tailwind CSS is used.
	Enabled bool `json:"enabled,omitempty"`

	// Config is the path to tailwind.config.js.
	Config string `json:"config,omitempty"`

	// Input is the input CSS file.
	Input string `json:"input,omitempty"`

	// Output is the output CSS file.
	Output string `json:"output,omitempty"`
}

// UIConfig contains VangoUI component settings.
type UIConfig struct {
	// Version is the pinned VangoUI version.
	Version string `json:"version,omitempty"`

	// Registry is the URL to the component registry.
	Registry string `json:"registry,omitempty"`

	// Installed is the list of installed components.
	Installed []string `json:"installed,omitempty"`

	// Path is the path where UI components are stored.
	Path string `json:"path,omitempty"`
}

// New creates a new Config with default values.
func New() *Config {
	return &Config{
		Version: "0.1.0",
		Port:    DefaultPort,
		Paths: PathsConfig{
			Routes:     "app/routes",
			Components: "app/components",
			UI:         "app/components/ui",
			Store:      "app/store",
			Middleware: "app/middleware",
		},
		Static: StaticConfig{
			Dir:    "public",
			Prefix: "/",
		},
		Dev: DevConfig{
			Port:        DefaultPort,
			Host:        DefaultHost,
			OpenBrowser: false,
			HotReload:   true,
			Watch:       []string{"app", "db", "public"},
		},
		Build: BuildConfig{
			Output:       DefaultOutput,
			Minify:       true,
			MinifyAssets: true,
			StripSymbols: true,
		},
		Session: SessionConfig{
			ResumeWindow: "30s",
		},
		UI: UIConfig{
			Registry: DefaultRegistry,
			Path:     "app/components/ui",
		},
		// Legacy fields for backwards compatibility
		Routes:     "app/routes",
		Components: "app/components",
		Public:     "public",
	}
}

// Load reads configuration from the specified directory.
// It looks for vango.json in the directory.
func Load(dir string) (*Config, error) {
	configPath := filepath.Join(dir, ConfigFileName)
	return LoadFile(configPath)
}

// LoadFile reads configuration from the specified file path.
func LoadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("E141").
				WithDetail("No vango.json found in " + filepath.Dir(path)).
				WithSuggestion("Run 'vango create' to create a new project or create vango.json manually")
		}
		return nil, errors.New("E120").Wrap(err)
	}

	cfg := New()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, errors.New("E120").
			WithDetail("Failed to parse vango.json: " + err.Error()).
			WithSuggestion("Check that vango.json is valid JSON")
	}

	cfg.configPath = path
	cfg.applyDefaults()

	return cfg, nil
}

// Save writes the configuration to the file it was loaded from.
func (c *Config) Save() error {
	if c.configPath == "" {
		return errors.Newf(errors.CategoryConfig, "no config path set")
	}
	return c.SaveTo(c.configPath)
}

// SaveTo writes the configuration to the specified path.
func (c *Config) SaveTo(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return errors.New("E120").Wrap(err)
	}

	// Add newline at end of file
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0644); err != nil {
		return errors.New("E120").Wrap(err)
	}

	c.configPath = path
	return nil
}

// Path returns the path where the config was loaded from.
func (c *Config) Path() string {
	return c.configPath
}

// Dir returns the directory containing the config file.
func (c *Config) Dir() string {
	if c.configPath == "" {
		return ""
	}
	return filepath.Dir(c.configPath)
}

// applyDefaults fills in default values for empty fields.
func (c *Config) applyDefaults() {
	// Port
	if c.Port == 0 {
		c.Port = DefaultPort
	}
	if c.Dev.Port == 0 {
		c.Dev.Port = c.Port
	}
	if c.Dev.Host == "" {
		c.Dev.Host = DefaultHost
	}
	if c.Dev.Watch == nil {
		c.Dev.Watch = []string{"app", "db", "public"}
	}

	// Build
	if c.Build.Output == "" {
		c.Build.Output = DefaultOutput
	}

	// Session
	if c.Session.ResumeWindow == "" {
		c.Session.ResumeWindow = "30s"
	}

	// Paths - prefer new Paths config, fall back to legacy fields
	if c.Paths.Routes == "" {
		if c.Routes != "" {
			c.Paths.Routes = c.Routes
		} else {
			c.Paths.Routes = "app/routes"
		}
	}
	if c.Paths.Components == "" {
		if c.Components != "" {
			c.Paths.Components = c.Components
		} else {
			c.Paths.Components = "app/components"
		}
	}
	if c.Paths.UI == "" {
		if c.UI.Path != "" {
			c.Paths.UI = c.UI.Path
		} else {
			c.Paths.UI = "app/components/ui"
		}
	}
	if c.Paths.Store == "" {
		c.Paths.Store = "app/store"
	}
	if c.Paths.Middleware == "" {
		c.Paths.Middleware = "app/middleware"
	}

	// Static
	if c.Static.Dir == "" {
		if c.Public != "" {
			c.Static.Dir = c.Public
		} else {
			c.Static.Dir = "public"
		}
	}
	if c.Static.Prefix == "" {
		c.Static.Prefix = "/"
	}

	// UI registry
	if c.UI.Registry == "" {
		c.UI.Registry = DefaultRegistry
	}
	if c.UI.Path == "" {
		c.UI.Path = c.Paths.UI
	}

	// Legacy fields - keep in sync for backwards compatibility
	if c.Routes == "" {
		c.Routes = c.Paths.Routes
	}
	if c.Components == "" {
		c.Components = c.Paths.Components
	}
	if c.Public == "" {
		c.Public = c.Static.Dir
	}
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.Dev.Port < 0 || c.Dev.Port > 65535 {
		return errors.New("E122").
			WithDetail("Port must be between 0 and 65535")
	}
	return nil
}

// DevAddress returns the address string for the dev server.
func (c *Config) DevAddress() string {
	return c.Dev.Host + ":" + itoa(c.Dev.Port)
}

// DevURL returns the full URL for the dev server.
func (c *Config) DevURL() string {
	scheme := "http"
	if c.Dev.HTTPS {
		scheme = "https"
	}
	return scheme + "://" + c.DevAddress()
}

// OutputPath returns the absolute path to the build output directory.
func (c *Config) OutputPath() string {
	if filepath.IsAbs(c.Build.Output) {
		return c.Build.Output
	}
	return filepath.Join(c.Dir(), c.Build.Output)
}

// RoutesPath returns the absolute path to the routes directory.
func (c *Config) RoutesPath() string {
	path := c.Paths.Routes
	if path == "" {
		path = c.Routes
	}
	if path == "" {
		path = "app/routes"
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(c.Dir(), path)
}

// ComponentsPath returns the absolute path to the components directory.
func (c *Config) ComponentsPath() string {
	path := c.Paths.Components
	if path == "" {
		path = c.Components
	}
	if path == "" {
		path = "app/components"
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(c.Dir(), path)
}

// PublicPath returns the absolute path to the public directory.
func (c *Config) PublicPath() string {
	path := c.Static.Dir
	if path == "" {
		path = c.Public
	}
	if path == "" {
		path = "public"
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(c.Dir(), path)
}

// UIComponentsPath returns the absolute path to the UI components directory.
func (c *Config) UIComponentsPath() string {
	path := c.Paths.UI
	if path == "" {
		path = c.UI.Path
	}
	if path == "" {
		path = "app/components/ui"
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(c.Dir(), path)
}

// StorePath returns the absolute path to the store directory.
func (c *Config) StorePath() string {
	path := c.Paths.Store
	if path == "" {
		path = "app/store"
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(c.Dir(), path)
}

// MiddlewarePath returns the absolute path to the middleware directory.
func (c *Config) MiddlewarePath() string {
	path := c.Paths.Middleware
	if path == "" {
		path = "app/middleware"
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(c.Dir(), path)
}

// StaticPrefix returns the URL prefix for static files.
func (c *Config) StaticPrefix() string {
	if c.Static.Prefix == "" {
		return "/"
	}
	return c.Static.Prefix
}

// HasTailwind returns true if Tailwind CSS is enabled.
func (c *Config) HasTailwind() bool {
	return c.Tailwind.Enabled
}

// TailwindConfigPath returns the absolute path to the Tailwind config.
func (c *Config) TailwindConfigPath() string {
	if c.Tailwind.Config == "" {
		return filepath.Join(c.Dir(), "tailwind.config.js")
	}
	if filepath.IsAbs(c.Tailwind.Config) {
		return c.Tailwind.Config
	}
	return filepath.Join(c.Dir(), c.Tailwind.Config)
}

// Exists checks if a config file exists in the given directory.
func Exists(dir string) bool {
	path := filepath.Join(dir, ConfigFileName)
	_, err := os.Stat(path)
	return err == nil
}

// FindProjectRoot walks up directories to find the project root.
// Returns the directory containing vango.json, or an error if not found.
func FindProjectRoot(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	for {
		if Exists(dir) {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("E141").
				WithDetail("No vango.json found in " + startDir + " or any parent directory").
				WithSuggestion("Run 'vango create' to create a new project")
		}
		dir = parent
	}
}

// LoadFromWorkingDir loads configuration from the current working directory.
func LoadFromWorkingDir() (*Config, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	root, err := FindProjectRoot(wd)
	if err != nil {
		return nil, err
	}

	return Load(root)
}

// itoa converts int to string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + itoa(-n)
	}
	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}
	// Reverse
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	return string(digits)
}
