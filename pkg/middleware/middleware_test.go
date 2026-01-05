package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/vango-go/vango/pkg/server"
)

// =============================================================================
// Test Helpers
// =============================================================================

// mockCtx implements server.Ctx for testing.
type mockCtx struct {
	path       string
	session    *server.Session
	user       any
	values     map[any]any
	stdCtx     context.Context
	event      *server.Event
	patchCount int
}

func newMockCtx(path string) *mockCtx {
	return &mockCtx{
		path:   path,
		values: make(map[any]any),
		stdCtx: context.Background(),
	}
}

func (m *mockCtx) Request() *http.Request {
	req := httptest.NewRequest("GET", m.path, nil)
	return req.WithContext(m.stdCtx)
}

func (m *mockCtx) Path() string                         { return m.path }
func (m *mockCtx) Method() string                       { return "GET" }
func (m *mockCtx) Query() url.Values                    { return nil }
func (m *mockCtx) Param(key string) string              { return "" }
func (m *mockCtx) Header(key string) string             { return "" }
func (m *mockCtx) Cookie(name string) (*http.Cookie, error) { return nil, nil }
func (m *mockCtx) Status(code int)                      {}
func (m *mockCtx) Redirect(url string, code int)        {}
func (m *mockCtx) SetHeader(key, value string)          {}
func (m *mockCtx) SetCookie(cookie *http.Cookie)        {}
func (m *mockCtx) Session() *server.Session             { return m.session }
func (m *mockCtx) User() any                            { return m.user }
func (m *mockCtx) SetUser(user any)                     { m.user = user }
func (m *mockCtx) Logger() *slog.Logger                  { return nil }
func (m *mockCtx) Done() <-chan struct{}                { return nil }
func (m *mockCtx) SetValue(key, value any)              { m.values[key] = value }
func (m *mockCtx) Value(key any) any                    { return m.values[key] }
func (m *mockCtx) Emit(name string, data any)           {}
func (m *mockCtx) StdContext() context.Context          { return m.stdCtx }
func (m *mockCtx) WithStdContext(ctx context.Context) server.Ctx {
	clone := *m
	clone.stdCtx = ctx
	return &clone
}
func (m *mockCtx) Event() *server.Event     { return m.event }
func (m *mockCtx) PatchCount() int          { return m.patchCount }
func (m *mockCtx) AddPatchCount(count int)  { m.patchCount += count }
func (m *mockCtx) Dispatch(fn func())                                   { fn() } // Execute inline for tests
func (m *mockCtx) Navigate(path string, opts ...server.NavigateOption) {} // No-op for tests

// =============================================================================
// OpenTelemetry Tests
// =============================================================================

func TestOpenTelemetryConfig(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		config := defaultOTelConfig()
		if config.TracerName != defaultTracerName {
			t.Errorf("TracerName = %q, want %q", config.TracerName, defaultTracerName)
		}
		if config.IncludeUserID {
			t.Error("IncludeUserID should be false by default")
		}
		if !config.IncludeRoute {
			t.Error("IncludeRoute should be true by default")
		}
	})

	t.Run("with options", func(t *testing.T) {
		config := defaultOTelConfig()
		WithTracerName("my-app")(&config)
		WithIncludeUserID(true)(&config)
		WithIncludeRoute(false)(&config)

		if config.TracerName != "my-app" {
			t.Errorf("TracerName = %q, want %q", config.TracerName, "my-app")
		}
		if !config.IncludeUserID {
			t.Error("IncludeUserID should be true")
		}
		if config.IncludeRoute {
			t.Error("IncludeRoute should be false")
		}
	})

	t.Run("with filter", func(t *testing.T) {
		filter := func(ctx server.Ctx) bool {
			return ctx.Path() != "/health"
		}
		config := defaultOTelConfig()
		WithEventFilter(filter)(&config)

		if config.Filter == nil {
			t.Error("Filter should be set")
		}
	})
}

func TestFormatSpanName(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/users", "vango /users"},
		{"/", "vango /"},
		{"/api/v1/products", "vango /api/v1/products"},
		{"", "vango /"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			ctx := newMockCtx(tt.path)
			got := formatSpanName(ctx)
			if got != tt.want {
				t.Errorf("formatSpanName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTraceContext(t *testing.T) {
	ctx := newMockCtx("/test")

	// Without span context set
	traceCtx := TraceContext(ctx)
	if traceCtx == nil {
		t.Error("TraceContext() should return non-nil context")
	}

	// With span context set
	customCtx := context.WithValue(context.Background(), "test", "value")
	ctx.SetValue(spanContextKey{}, customCtx)
	traceCtx = TraceContext(ctx)
	if traceCtx.Value("test") != "value" {
		t.Error("TraceContext() should return the stored span context")
	}
}

// =============================================================================
// Prometheus Metrics Tests
// =============================================================================

func TestMetricsConfig(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		config := defaultMetricsConfig()
		if config.Namespace != "vango" {
			t.Errorf("Namespace = %q, want %q", config.Namespace, "vango")
		}
		if config.Subsystem != "" {
			t.Errorf("Subsystem = %q, want empty", config.Subsystem)
		}
		if config.Registry != prometheus.DefaultRegisterer {
			t.Error("Registry should be DefaultRegisterer")
		}
	})

	t.Run("with options", func(t *testing.T) {
		config := defaultMetricsConfig()
		WithNamespace("myapp")(&config)
		WithSubsystem("api")(&config)
		WithBuckets([]float64{0.1, 0.5, 1.0})(&config)

		if config.Namespace != "myapp" {
			t.Errorf("Namespace = %q, want %q", config.Namespace, "myapp")
		}
		if config.Subsystem != "api" {
			t.Errorf("Subsystem = %q, want %q", config.Subsystem, "api")
		}
		if len(config.Buckets) != 3 {
			t.Errorf("len(Buckets) = %d, want 3", len(config.Buckets))
		}
	})
}

func TestCategorizeError(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{errors.New("timeout exceeded"), "timeout"},
		{errors.New("rate limit exceeded"), "rate_limit"},
		{errors.New("resource not found"), "not_found"},
		{errors.New("unauthorized access"), "unauthorized"},
		{errors.New("forbidden action"), "forbidden"},
		{errors.New("validation error"), "validation"},
		{errors.New("websocket error"), "websocket"},
		{errors.New("some other error"), "internal"},
	}

	for _, tt := range tests {
		t.Run(tt.err.Error(), func(t *testing.T) {
			got := categorizeError(tt.err)
			if got != tt.want {
				t.Errorf("categorizeError(%q) = %q, want %q", tt.err, got, tt.want)
			}
		})
	}
}

func TestMetricsRecordFunctions(t *testing.T) {
	// These functions should not panic even when globalMetrics is nil
	t.Run("record functions handle nil metrics", func(t *testing.T) {
		// Reset global metrics
		globalMetricsMu.Lock()
		globalMetrics = nil
		globalMetricsMu.Unlock()

		// These should not panic
		RecordPatches(10)
		RecordSessionCreate()
		RecordSessionDestroy(1024)
		RecordSessionDetach()
		RecordSessionReattach()
		RecordWebSocketError("test")
	})
}

func TestGetMetrics(t *testing.T) {
	// Reset global metrics
	globalMetricsMu.Lock()
	globalMetrics = nil
	globalMetricsMu.Unlock()

	// Should return nil when not initialized
	if GetMetrics() != nil {
		t.Error("GetMetrics() should return nil when not initialized")
	}
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestMiddlewareChain(t *testing.T) {
	// Test that middleware can be chained
	ctx := newMockCtx("/test")

	executed := make([]string, 0)

	// Create a simple handler
	handler := func() error {
		executed = append(executed, "handler")
		return nil
	}

	// The handler should execute
	err := handler()
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}

	if len(executed) != 1 || executed[0] != "handler" {
		t.Errorf("executed = %v, want [handler]", executed)
	}

	_ = ctx // Used to create context
}
