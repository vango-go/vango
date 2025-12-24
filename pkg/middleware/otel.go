package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/vango-dev/vango/v2/pkg/router"
	"github.com/vango-dev/vango/v2/pkg/server"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Default tracer name for Vango applications.
const defaultTracerName = "vango"

// OTelConfig configures the OpenTelemetry middleware.
type OTelConfig struct {
	// TracerName is the name of the tracer (default: "vango").
	TracerName string

	// IncludeUserID includes the user ID in traces if available.
	// May contain sensitive information - disabled by default.
	IncludeUserID bool

	// IncludeRoute includes the current route in traces.
	// Enabled by default.
	IncludeRoute bool

	// Filter determines which events to trace.
	// Return true to trace the event, false to skip.
	// If nil, all events are traced.
	Filter func(ctx server.Ctx) bool

	// AttributeExtractor extracts custom attributes from the context.
	// Called for each traced event.
	AttributeExtractor func(ctx server.Ctx) []attribute.KeyValue

	// tracer is the resolved tracer instance.
	tracer trace.Tracer
}

// OTelOption configures the OpenTelemetry middleware.
type OTelOption func(*OTelConfig)

// WithTracerName sets the tracer name.
func WithTracerName(name string) OTelOption {
	return func(c *OTelConfig) {
		c.TracerName = name
	}
}

// WithIncludeUserID enables including user ID in traces.
func WithIncludeUserID(include bool) OTelOption {
	return func(c *OTelConfig) {
		c.IncludeUserID = include
	}
}

// WithIncludeRoute enables/disables including route in traces.
func WithIncludeRoute(include bool) OTelOption {
	return func(c *OTelConfig) {
		c.IncludeRoute = include
	}
}

// WithEventFilter sets a filter function for events.
func WithEventFilter(filter func(ctx server.Ctx) bool) OTelOption {
	return func(c *OTelConfig) {
		c.Filter = filter
	}
}

// WithAttributeExtractor sets a custom attribute extractor.
func WithAttributeExtractor(extractor func(ctx server.Ctx) []attribute.KeyValue) OTelOption {
	return func(c *OTelConfig) {
		c.AttributeExtractor = extractor
	}
}

// defaultOTelConfig returns the default OpenTelemetry configuration.
func defaultOTelConfig() OTelConfig {
	return OTelConfig{
		TracerName:    defaultTracerName,
		IncludeUserID: false,
		IncludeRoute:  true,
		Filter:        nil,
	}
}

// OpenTelemetry creates middleware that traces every Vango event.
//
// The middleware:
//   - Creates a span for each event with type, target, and session ID
//   - Injects trace context into ctx.StdContext() for downstream calls
//   - Records errors and sets span status
//   - Records patch count as a span attribute
//
// Example:
//
//	app := vango.NewApp(
//	    vango.WithMiddleware(
//	        middleware.OpenTelemetry(
//	            middleware.WithTracerName("my-app"),
//	            middleware.WithIncludeUserID(true),
//	        ),
//	    ),
//	)
//
// The tracer uses the global OpenTelemetry tracer provider. Configure it
// in your main() before starting the server:
//
//	tp := sdktrace.NewTracerProvider(
//	    sdktrace.WithBatcher(exporter),
//	    sdktrace.WithResource(resource.NewWithAttributes(
//	        semconv.SchemaURL,
//	        semconv.ServiceName("my-app"),
//	    )),
//	)
//	otel.SetTracerProvider(tp)
func OpenTelemetry(opts ...OTelOption) router.Middleware {
	config := defaultOTelConfig()
	for _, opt := range opts {
		opt(&config)
	}

	// Resolve tracer from global provider
	config.tracer = otel.Tracer(config.TracerName)

	return router.MiddlewareFunc(func(ctx server.Ctx, next func() error) error {
		// Apply filter if configured
		if config.Filter != nil && !config.Filter(ctx) {
			return next()
		}

		// Create span name from path (may be overridden by event type)
		spanName := formatSpanName(ctx)

		// Build span attributes
		attrs := []attribute.KeyValue{
			attribute.String("vango.path", ctx.Path()),
		}

		// Add session ID if available
		if session := ctx.Session(); session != nil {
			attrs = append(attrs, attribute.String("vango.session_id", session.ID))
		}

		// Add event type and target HID if available (Phase 13)
		if event := ctx.Event(); event != nil {
			attrs = append(attrs,
				attribute.String("vango.event_type", event.TypeString()),
				attribute.String("vango.event_target", event.HID),
			)
			// Override span name with event type for WebSocket events
			spanName = fmt.Sprintf("vango.%s", event.TypeString())
		}

		// Add route if enabled
		if config.IncludeRoute {
			attrs = append(attrs, attribute.String("vango.route", ctx.Path()))
		}

		// Add user ID if enabled and available
		if config.IncludeUserID {
			if user := ctx.User(); user != nil {
				attrs = append(attrs, attribute.String("vango.user_id", fmt.Sprintf("%v", user)))
			}
		}

		// Add custom attributes
		if config.AttributeExtractor != nil {
			attrs = append(attrs, config.AttributeExtractor(ctx)...)
		}

		// Start span
		spanCtx, span := config.tracer.Start(
			ctx.StdContext(),
			spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(attrs...),
			trace.WithTimestamp(time.Now()),
		)
		defer span.End()

		// Inject trace context into Ctx for downstream calls
		// Store the span context in Ctx values so handlers can access it
		// via ctx.Value(spanContextKey{}) or middleware.SpanFromContext(ctx)
		ctx.SetValue(spanContextKey{}, spanCtx)

		// Also store the wrapped context for StdContext() callers
		_ = ctx.WithStdContext(spanCtx) // Used for documentation - actual injection via SetValue

		// Execute the handler
		err := next()

		// Record result
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "")
		}

		// Record patch count (Phase 13)
		span.SetAttributes(attribute.Int("vango.patch_count", ctx.PatchCount()))

		return err
	})
}

// spanContextKey is the key for storing the span context in Ctx values.
type spanContextKey struct{}

// SpanFromContext retrieves the current trace span from the context.
// Returns nil if no span is available.
//
// Example:
//
//	func MyHandler(ctx server.Ctx) error {
//	    if span := middleware.SpanFromContext(ctx); span != nil {
//	        span.SetAttributes(attribute.Int("my.count", 42))
//	    }
//	    return nil
//	}
func SpanFromContext(ctx server.Ctx) trace.Span {
	if spanCtx, ok := ctx.Value(spanContextKey{}).(context.Context); ok {
		return trace.SpanFromContext(spanCtx)
	}
	return nil
}

// formatSpanName creates a span name from the context.
func formatSpanName(ctx server.Ctx) string {
	path := ctx.Path()
	if path == "" {
		path = "/"
	}
	return fmt.Sprintf("vango %s", path)
}

// TraceContext returns the trace context from the Ctx for propagation.
// Use this to propagate trace context to external services.
//
// Example:
//
//	func MyHandler(ctx server.Ctx) error {
//	    traceCtx := middleware.TraceContext(ctx)
//	    req, _ := http.NewRequestWithContext(traceCtx, "GET", url, nil)
//	    return nil
//	}
func TraceContext(ctx server.Ctx) context.Context {
	if spanCtx, ok := ctx.Value(spanContextKey{}).(context.Context); ok {
		return spanCtx
	}
	return ctx.StdContext()
}
