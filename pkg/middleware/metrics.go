package middleware

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/vango-dev/vango/v2/pkg/router"
	"github.com/vango-dev/vango/v2/pkg/server"
)

// MetricsConfig configures the Prometheus metrics middleware.
type MetricsConfig struct {
	// Namespace is the metrics namespace (default: "vango").
	Namespace string

	// Subsystem is the metrics subsystem (default: "").
	Subsystem string

	// ConstLabels are constant labels added to all metrics.
	ConstLabels prometheus.Labels

	// Buckets are the histogram buckets for event duration.
	// Default: prometheus.DefBuckets
	Buckets []float64

	// Registry is the Prometheus registry to use.
	// Default: prometheus.DefaultRegisterer
	Registry prometheus.Registerer
}

// MetricsOption configures the Prometheus metrics middleware.
type MetricsOption func(*MetricsConfig)

// WithNamespace sets the metrics namespace.
func WithNamespace(namespace string) MetricsOption {
	return func(c *MetricsConfig) {
		c.Namespace = namespace
	}
}

// WithSubsystem sets the metrics subsystem.
func WithSubsystem(subsystem string) MetricsOption {
	return func(c *MetricsConfig) {
		c.Subsystem = subsystem
	}
}

// WithConstLabels sets constant labels for all metrics.
func WithConstLabels(labels prometheus.Labels) MetricsOption {
	return func(c *MetricsConfig) {
		c.ConstLabels = labels
	}
}

// WithBuckets sets the histogram buckets.
func WithBuckets(buckets []float64) MetricsOption {
	return func(c *MetricsConfig) {
		c.Buckets = buckets
	}
}

// WithRegistry sets the Prometheus registry.
func WithRegistry(registry prometheus.Registerer) MetricsOption {
	return func(c *MetricsConfig) {
		c.Registry = registry
	}
}

// defaultMetricsConfig returns the default metrics configuration.
func defaultMetricsConfig() MetricsConfig {
	return MetricsConfig{
		Namespace:   "vango",
		Subsystem:   "",
		ConstLabels: nil,
		Buckets:     prometheus.DefBuckets,
		Registry:    prometheus.DefaultRegisterer,
	}
}

// metrics holds the Prometheus metrics for Vango.
type metrics struct {
	eventsTotal      *prometheus.CounterVec
	eventDuration    *prometheus.HistogramVec
	eventErrors      *prometheus.CounterVec
	patchesSent      prometheus.Counter
	activeSessions   prometheus.Gauge
	detachedSessions prometheus.Gauge
	sessionMemory    prometheus.Histogram
	wsErrors         *prometheus.CounterVec
	reconnectsTotal  prometheus.Counter // Phase 13 audit: added per spec
}

// globalMetrics is the singleton metrics instance.
// Created on first call to Prometheus().
var (
	globalMetrics     *metrics
	globalMetricsOnce sync.Once
	globalMetricsMu   sync.Mutex
)

// initMetrics initializes the Prometheus metrics.
func initMetrics(config MetricsConfig) *metrics {
	factory := promauto.With(config.Registry)

	return &metrics{
		eventsTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace:   config.Namespace,
			Subsystem:   config.Subsystem,
			Name:        "events_total",
			Help:        "Total number of Vango events processed",
			ConstLabels: config.ConstLabels,
		}, []string{"path", "status"}),

		eventDuration: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace:   config.Namespace,
			Subsystem:   config.Subsystem,
			Name:        "event_duration_seconds",
			Help:        "Event processing duration in seconds",
			ConstLabels: config.ConstLabels,
			Buckets:     config.Buckets,
		}, []string{"path"}),

		eventErrors: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace:   config.Namespace,
			Subsystem:   config.Subsystem,
			Name:        "event_errors_total",
			Help:        "Total number of event processing errors",
			ConstLabels: config.ConstLabels,
		}, []string{"path", "error_type"}),

		patchesSent: factory.NewCounter(prometheus.CounterOpts{
			Namespace:   config.Namespace,
			Subsystem:   config.Subsystem,
			Name:        "patches_sent_total",
			Help:        "Total number of patches sent to clients",
			ConstLabels: config.ConstLabels,
		}),

		activeSessions: factory.NewGauge(prometheus.GaugeOpts{
			Namespace:   config.Namespace,
			Subsystem:   config.Subsystem,
			Name:        "active_sessions",
			Help:        "Number of active WebSocket sessions",
			ConstLabels: config.ConstLabels,
		}),

		detachedSessions: factory.NewGauge(prometheus.GaugeOpts{
			Namespace:   config.Namespace,
			Subsystem:   config.Subsystem,
			Name:        "detached_sessions",
			Help:        "Number of detached (disconnected but resumable) sessions",
			ConstLabels: config.ConstLabels,
		}),

		sessionMemory: factory.NewHistogram(prometheus.HistogramOpts{
			Namespace:   config.Namespace,
			Subsystem:   config.Subsystem,
			Name:        "session_memory_bytes",
			Help:        "Estimated memory usage per session in bytes",
			ConstLabels: config.ConstLabels,
			Buckets:     []float64{1024, 10240, 102400, 1048576, 10485760}, // 1KB to 10MB
		}),

		wsErrors: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace:   config.Namespace,
			Subsystem:   config.Subsystem,
			Name:        "websocket_errors_total",
			Help:        "Total WebSocket errors by type",
			ConstLabels: config.ConstLabels,
		}, []string{"type"}),

		reconnectsTotal: factory.NewCounter(prometheus.CounterOpts{
			Namespace:   config.Namespace,
			Subsystem:   config.Subsystem,
			Name:        "reconnects_total",
			Help:        "Total number of session reconnections",
			ConstLabels: config.ConstLabels,
		}),
	}
}

// Prometheus creates middleware that collects Prometheus metrics for Vango events.
//
// Metrics collected:
//   - vango_events_total: Counter of events by path and status
//   - vango_event_duration_seconds: Histogram of event processing duration
//   - vango_event_errors_total: Counter of event errors by path and error type
//   - vango_patches_sent_total: Counter of patches sent (when RecordPatches is called)
//   - vango_active_sessions: Gauge of active sessions (when session hooks are used)
//   - vango_detached_sessions: Gauge of detached sessions
//   - vango_session_memory_bytes: Histogram of session memory usage
//   - vango_websocket_errors_total: Counter of WebSocket errors
//   - vango_reconnects_total: Counter of session reconnections
//
// Example:
//
//	app := vango.NewApp(
//	    vango.WithMiddleware(
//	        middleware.Prometheus(
//	            middleware.WithNamespace("myapp"),
//	        ),
//	    ),
//	)
//
//	// Expose metrics endpoint
//	http.Handle("/metrics", promhttp.Handler())
func Prometheus(opts ...MetricsOption) router.Middleware {
	config := defaultMetricsConfig()
	for _, opt := range opts {
		opt(&config)
	}

	// Initialize metrics once
	globalMetricsMu.Lock()
	if globalMetrics == nil {
		globalMetrics = initMetrics(config)
	}
	m := globalMetrics
	globalMetricsMu.Unlock()

	return router.MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
		path := ctx.Path()
		if path == "" {
			path = "/"
		}

		// Time the event
		start := time.Now()

		// Execute handler
		err := next()

		// Record duration
		duration := time.Since(start).Seconds()
		m.eventDuration.WithLabelValues(path).Observe(duration)

		// Record event count
		status := "success"
		if err != nil {
			status = "error"
			// Record error with type
			errorType := "unknown"
			if err.Error() != "" {
				// Use a simple categorization
				errorType = categorizeError(err)
			}
			m.eventErrors.WithLabelValues(path, errorType).Inc()
		}
		m.eventsTotal.WithLabelValues(path, status).Inc()

		return err
	})
}

// categorizeError returns a category for the error type.
// This prevents high-cardinality labels from error messages.
func categorizeError(err error) string {
	errStr := err.Error()
	switch {
	case contains(errStr, "timeout"):
		return "timeout"
	case contains(errStr, "rate limit"):
		return "rate_limit"
	case contains(errStr, "not found"):
		return "not_found"
	case contains(errStr, "unauthorized"):
		return "unauthorized"
	case contains(errStr, "forbidden"):
		return "forbidden"
	case contains(errStr, "validation"):
		return "validation"
	case contains(errStr, "websocket"):
		return "websocket"
	default:
		return "internal"
	}
}

// contains is a simple case-insensitive contains check.
func contains(s, substr string) bool {
	// Simple check - could be improved with strings.Contains
	return len(s) >= len(substr) && (s == substr ||
		len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// =============================================================================
// Metrics Recording Functions
// =============================================================================

// RecordPatches records the number of patches sent.
// Call this from your server code when patches are sent to clients.
func RecordPatches(count int) {
	if globalMetrics != nil {
		globalMetrics.patchesSent.Add(float64(count))
	}
}

// RecordSessionCreate records a new session creation.
func RecordSessionCreate() {
	if globalMetrics != nil {
		globalMetrics.activeSessions.Inc()
	}
}

// RecordSessionDestroy records a session destruction.
func RecordSessionDestroy(memoryBytes int64) {
	if globalMetrics != nil {
		globalMetrics.activeSessions.Dec()
		globalMetrics.sessionMemory.Observe(float64(memoryBytes))
	}
}

// RecordSessionDetach records a session becoming detached.
func RecordSessionDetach() {
	if globalMetrics != nil {
		globalMetrics.activeSessions.Dec()
		globalMetrics.detachedSessions.Inc()
	}
}

// RecordSessionReattach records a detached session being reattached.
func RecordSessionReattach() {
	if globalMetrics != nil {
		globalMetrics.activeSessions.Inc()
		globalMetrics.detachedSessions.Dec()
	}
}

// RecordWebSocketError records a WebSocket error.
func RecordWebSocketError(errorType string) {
	if globalMetrics != nil {
		globalMetrics.wsErrors.WithLabelValues(errorType).Inc()
	}
}

// RecordReconnect records a session reconnection.
// Call this when a detached session is successfully resumed.
func RecordReconnect() {
	if globalMetrics != nil {
		globalMetrics.reconnectsTotal.Inc()
	}
}

// =============================================================================
// Metrics Collector
// =============================================================================

// Collector returns the metrics for use in custom registrations.
// This allows collecting Vango metrics alongside other application metrics.
type Collector struct {
	eventsTotal      *prometheus.CounterVec
	eventDuration    *prometheus.HistogramVec
	eventErrors      *prometheus.CounterVec
	patchesSent      prometheus.Counter
	activeSessions   prometheus.Gauge
	detachedSessions prometheus.Gauge
	sessionMemory    prometheus.Histogram
	wsErrors         *prometheus.CounterVec
	reconnectsTotal  prometheus.Counter
}

// GetMetrics returns the global metrics collector.
// Returns nil if Prometheus middleware has not been initialized.
func GetMetrics() *Collector {
	if globalMetrics == nil {
		return nil
	}
	return &Collector{
		eventsTotal:      globalMetrics.eventsTotal,
		eventDuration:    globalMetrics.eventDuration,
		eventErrors:      globalMetrics.eventErrors,
		patchesSent:      globalMetrics.patchesSent,
		activeSessions:   globalMetrics.activeSessions,
		detachedSessions: globalMetrics.detachedSessions,
		sessionMemory:    globalMetrics.sessionMemory,
		wsErrors:         globalMetrics.wsErrors,
		reconnectsTotal:  globalMetrics.reconnectsTotal,
	}
}
