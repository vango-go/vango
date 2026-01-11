package middleware

import (
	"context"
	"errors"
	"testing"

	"github.com/vango-go/vango/pkg/protocol"
	"github.com/vango-go/vango/pkg/server"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func TestOpenTelemetryMiddleware_StoresTraceContext(t *testing.T) {
	session := server.NewMockSession()
	session.ID = "sess-1"

	ctx := newMockCtx("/projects")
	ctx.session = session
	ctx.user = "user-123"
	ctx.event = &server.Event{Type: protocol.EventClick, HID: "btn-1"}
	ctx.AddPatchCount(3)

	mw := OpenTelemetry(
		WithIncludeUserID(true),
		WithIncludeRoute(true),
		WithAttributeExtractor(func(server.Ctx) []attribute.KeyValue {
			return []attribute.KeyValue{attribute.String("test.attr", "ok")}
		}),
	)

	err := mw.Handle(ctx, func() error {
		span := SpanFromContext(ctx)
		if span == nil {
			t.Fatal("expected SpanFromContext to return a span during execution")
		}
		_ = trace.SpanContextFromContext(TraceContext(ctx)) // Should not panic
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stored := ctx.Value(spanContextKey{})
	spanCtx, ok := stored.(context.Context)
	if !ok || spanCtx == nil {
		t.Fatalf("expected span context to be stored on ctx, got %T", stored)
	}
	if TraceContext(ctx) != spanCtx {
		t.Fatal("expected TraceContext(ctx) to return stored span context")
	}
	if SpanFromContext(ctx) == nil {
		t.Fatal("expected SpanFromContext(ctx) to return a span after middleware execution")
	}
}

func TestOpenTelemetryMiddleware_ErrorPropagatesAndStillStoresContext(t *testing.T) {
	ctx := newMockCtx("/projects")
	ctx.event = &server.Event{Type: protocol.EventInput, HID: "name"}
	ctx.AddPatchCount(1)

	wantErr := errors.New("boom")
	err := OpenTelemetry().Handle(ctx, func() error { return wantErr })
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}

	stored := ctx.Value(spanContextKey{})
	if _, ok := stored.(context.Context); !ok {
		t.Fatalf("expected span context to be stored on ctx, got %T", stored)
	}
	if SpanFromContext(ctx) == nil {
		t.Fatal("expected SpanFromContext(ctx) to return a span after middleware execution")
	}
}

func TestOpenTelemetryMiddleware_FilterSkipsTracing(t *testing.T) {
	ctx := newMockCtx("/healthz")

	nextCalled := false
	err := OpenTelemetry(
		WithEventFilter(func(c server.Ctx) bool { return c.Path() != "/healthz" }),
	).Handle(ctx, func() error {
		nextCalled = true
		if SpanFromContext(ctx) != nil {
			t.Fatal("expected no span when filter skips tracing")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !nextCalled {
		t.Fatal("expected next to be called")
	}
	if ctx.Value(spanContextKey{}) != nil {
		t.Fatalf("expected no span context to be stored when filter skips tracing, got %T", ctx.Value(spanContextKey{}))
	}
}

func TestSpanFromContext_NoSpan(t *testing.T) {
	ctx := newMockCtx("/test")
	if SpanFromContext(ctx) != nil {
		t.Fatal("expected nil span when no span context is stored")
	}
}

