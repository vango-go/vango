// Package middleware provides production-grade middleware for Vango applications.
//
// This package includes:
//   - OpenTelemetry distributed tracing middleware
//   - Prometheus metrics middleware
//   - Recovery and logging utilities
//
// # OpenTelemetry Middleware
//
// The OpenTelemetry middleware automatically traces every Vango event, providing
// distributed tracing across your application. Traces include session ID, event type,
// route, and patch counts.
//
//	app := vango.NewApp(
//	    vango.WithMiddleware(
//	        middleware.OpenTelemetry(),
//	    ),
//	)
//
// Configure with options:
//
//	middleware.OpenTelemetry(
//	    middleware.WithTracerName("my-app"),
//	    middleware.WithIncludeUserID(true),
//	    middleware.WithEventFilter(func(ctx server.Ctx) bool {
//	        return ctx.Path() != "/healthz"
//	    }),
//	)
//
// # Prometheus Metrics
//
// The Prometheus middleware collects metrics about your Vango application:
//   - vango_active_sessions: Current number of active sessions
//   - vango_events_total: Total events processed by type
//   - vango_event_duration_seconds: Event processing duration histogram
//   - vango_patches_sent_total: Total patches sent to clients
//
//	app := vango.NewApp(
//	    vango.WithMiddleware(
//	        middleware.Prometheus(),
//	    ),
//	)
//
// Then expose metrics on a separate port:
//
//	http.Handle("/metrics", promhttp.Handler())
//	go http.ListenAndServe(":9090", nil)
//
// # Context Propagation
//
// Both middlewares inject trace context into ctx.StdContext(), allowing
// database drivers and HTTP clients to inherit the trace:
//
//	func MyHandler(ctx server.Ctx) error {
//	    // Database call inherits trace context
//	    row := db.QueryRowContext(ctx.StdContext(), "SELECT ...")
//
//	    // HTTP call inherits trace context
//	    req, _ := http.NewRequestWithContext(ctx.StdContext(), "GET", url, nil)
//	    return nil
//	}
//
// # Phase 13: Production Hardening & Observability
//
// This package was introduced in Phase 13 to provide production-grade
// observability for Vango applications.
package middleware
