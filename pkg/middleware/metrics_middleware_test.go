package middleware

import (
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func resetGlobalMetricsForTest() {
	globalMetricsMu.Lock()
	globalMetrics = nil
	globalMetricsMu.Unlock()
}

func metricCounterValue(t *testing.T, c prometheus.Counter) float64 {
	t.Helper()
	var m dto.Metric
	if err := c.Write(&m); err != nil {
		t.Fatalf("counter Write() error: %v", err)
	}
	if m.Counter == nil {
		t.Fatal("expected counter metric to have Counter field")
	}
	return m.GetCounter().GetValue()
}

func metricGaugeValue(t *testing.T, g prometheus.Gauge) float64 {
	t.Helper()
	var m dto.Metric
	if err := g.Write(&m); err != nil {
		t.Fatalf("gauge Write() error: %v", err)
	}
	if m.Gauge == nil {
		t.Fatal("expected gauge metric to have Gauge field")
	}
	return m.GetGauge().GetValue()
}

func metricHistogramCount(t *testing.T, o prometheus.Observer) uint64 {
	t.Helper()
	metric, ok := o.(prometheus.Metric)
	if !ok {
		t.Fatalf("observer %T does not implement prometheus.Metric", o)
	}
	var m dto.Metric
	if err := metric.Write(&m); err != nil {
		t.Fatalf("histogram Write() error: %v", err)
	}
	if m.Histogram == nil {
		t.Fatal("expected histogram metric to have Histogram field")
	}
	return m.GetHistogram().GetSampleCount()
}

func TestPrometheusMiddleware_RecordsSuccessAndError(t *testing.T) {
	t.Run("success increments success counter and duration", func(t *testing.T) {
		resetGlobalMetricsForTest()
		reg := prometheus.NewRegistry()

		mw := Prometheus(WithRegistry(reg))
		ctx := newMockCtx("/test")

		err := mw.Handle(ctx, func() error { return nil })
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		c := GetMetrics()
		if c == nil {
			t.Fatal("expected GetMetrics to return collector after initialization")
		}

		if got := metricCounterValue(t, c.eventsTotal.WithLabelValues("/test", "success")); got != 1 {
			t.Fatalf("events_total(success)=%v, want 1", got)
		}
		if got := metricCounterValue(t, c.eventsTotal.WithLabelValues("/test", "error")); got != 0 {
			t.Fatalf("events_total(error)=%v, want 0", got)
		}
		if got := metricHistogramCount(t, c.eventDuration.WithLabelValues("/test")); got == 0 {
			t.Fatal("expected event_duration_seconds histogram to have sample count > 0")
		}
	})

	t.Run("error increments error counter and categorizes", func(t *testing.T) {
		resetGlobalMetricsForTest()
		reg := prometheus.NewRegistry()

		mw := Prometheus(WithRegistry(reg))
		ctx := newMockCtx("/test")

		err := mw.Handle(ctx, func() error { return errors.New("timeout exceeded") })
		if err == nil {
			t.Fatal("expected error to propagate")
		}

		c := GetMetrics()
		if c == nil {
			t.Fatal("expected GetMetrics to return collector after initialization")
		}

		if got := metricCounterValue(t, c.eventsTotal.WithLabelValues("/test", "error")); got != 1 {
			t.Fatalf("events_total(error)=%v, want 1", got)
		}
		if got := metricCounterValue(t, c.eventErrors.WithLabelValues("/test", "timeout")); got != 1 {
			t.Fatalf("event_errors_total(timeout)=%v, want 1", got)
		}
	})
}

func TestPrometheusMiddleware_EmptyPathNormalizesToSlash(t *testing.T) {
	resetGlobalMetricsForTest()
	reg := prometheus.NewRegistry()

	mw := Prometheus(WithRegistry(reg))
	ctx := newMockCtx("")

	err := mw.Handle(ctx, func() error { return nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	c := GetMetrics()
	if c == nil {
		t.Fatal("expected GetMetrics to return collector after initialization")
	}
	if got := metricCounterValue(t, c.eventsTotal.WithLabelValues("/", "success")); got != 1 {
		t.Fatalf("events_total(/,success)=%v, want 1", got)
	}
}

func TestMetricsRecordFunctions_WithInitializedMetrics(t *testing.T) {
	resetGlobalMetricsForTest()
	reg := prometheus.NewRegistry()

	_ = Prometheus(WithRegistry(reg)) // initialize global metrics
	c := GetMetrics()
	if c == nil {
		t.Fatal("expected GetMetrics to return collector after initialization")
	}

	RecordPatches(5)
	RecordSessionCreate()
	RecordSessionDetach()
	RecordSessionReattach()
	RecordSessionDestroy(2048)
	RecordWebSocketError("close")
	RecordReconnect()
	RecordResume()
	RecordResumeFailed("token_expired")
	RecordEviction()

	if got := metricCounterValue(t, c.patchesSent); got != 5 {
		t.Fatalf("patches_sent_total=%v, want 5", got)
	}
	if got := metricGaugeValue(t, c.activeSessions); got != 0 {
		t.Fatalf("active_sessions=%v, want 0 (create+detach+reattach+destroy)", got)
	}
	if got := metricGaugeValue(t, c.detachedSessions); got != 0 {
		t.Fatalf("detached_sessions=%v, want 0 (detach+reattach)", got)
	}
	if got := metricCounterValue(t, c.wsErrors.WithLabelValues("close")); got != 1 {
		t.Fatalf("websocket_errors_total(close)=%v, want 1", got)
	}
	if got := metricCounterValue(t, c.reconnectsTotal); got != 1 {
		t.Fatalf("reconnects_total=%v, want 1", got)
	}
	if got := metricCounterValue(t, c.resumesTotal); got != 1 {
		t.Fatalf("session_resumes_total=%v, want 1", got)
	}
	if got := metricCounterValue(t, c.resumeFailuresTotal.WithLabelValues("token_expired")); got != 1 {
		t.Fatalf("session_resume_failures_total(token_expired)=%v, want 1", got)
	}
	if got := metricCounterValue(t, c.evictionsTotal); got != 1 {
		t.Fatalf("session_evictions_total=%v, want 1", got)
	}
	if got := metricHistogramCount(t, c.sessionMemory); got == 0 {
		t.Fatal("expected session_memory_bytes histogram to have sample count > 0")
	}
}
