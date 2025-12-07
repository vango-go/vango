package server

import (
	"sync/atomic"
	"time"
)

// ServerMetrics aggregates metrics across the server.
type ServerMetrics struct {
	// Sessions
	ActiveSessions  int64
	TotalSessions   int64
	SessionCreates  int64
	SessionCloses   int64
	PeakSessions    int64

	// Events
	EventsReceived  int64
	EventsProcessed int64
	EventsDropped   int64

	// Patches
	PatchesSent int64
	PatchBytes  int64

	// Network
	BytesSent     int64
	BytesReceived int64

	// Errors
	HandlerPanics int64
	WriteErrors   int64
	ReadErrors    int64

	// Latency (microseconds)
	EventLatencyP50 int64
	EventLatencyP99 int64

	// Memory
	TotalMemory int64

	// Timestamp
	CollectedAt time.Time
}

// Metrics collects and returns server metrics.
func (s *Server) Metrics() *ServerMetrics {
	stats := s.sessions.Stats()

	return &ServerMetrics{
		ActiveSessions: int64(stats.Active),
		TotalSessions:  int64(stats.TotalCreated),
		SessionCreates: int64(stats.TotalCreated),
		SessionCloses:  int64(stats.TotalClosed),
		PeakSessions:   int64(stats.Peak),
		TotalMemory:    stats.TotalMemory,
		CollectedAt:    time.Now(),
	}
}

// MetricsCollector collects and aggregates metrics over time.
type MetricsCollector struct {
	// Counters (atomic)
	eventsReceived  atomic.Int64
	eventsProcessed atomic.Int64
	eventsDropped   atomic.Int64
	patchesSent     atomic.Int64
	patchBytes      atomic.Int64
	bytesSent       atomic.Int64
	bytesReceived   atomic.Int64
	handlerPanics   atomic.Int64
	writeErrors     atomic.Int64
	readErrors      atomic.Int64

	// Latency tracking
	latencies []int64
	latencyMu atomic.Int32 // Simple spinlock
}

// NewMetricsCollector creates a new MetricsCollector.
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		latencies: make([]int64, 0, 1000),
	}
}

// RecordEventReceived records an event received.
func (m *MetricsCollector) RecordEventReceived() {
	m.eventsReceived.Add(1)
}

// RecordEventProcessed records an event processed.
func (m *MetricsCollector) RecordEventProcessed() {
	m.eventsProcessed.Add(1)
}

// RecordEventDropped records an event dropped.
func (m *MetricsCollector) RecordEventDropped() {
	m.eventsDropped.Add(1)
}

// RecordPatchesSent records patches sent.
func (m *MetricsCollector) RecordPatchesSent(count int, bytes int) {
	m.patchesSent.Add(int64(count))
	m.patchBytes.Add(int64(bytes))
}

// RecordBytesSent records bytes sent.
func (m *MetricsCollector) RecordBytesSent(n int) {
	m.bytesSent.Add(int64(n))
}

// RecordBytesReceived records bytes received.
func (m *MetricsCollector) RecordBytesReceived(n int) {
	m.bytesReceived.Add(int64(n))
}

// RecordHandlerPanic records a handler panic.
func (m *MetricsCollector) RecordHandlerPanic() {
	m.handlerPanics.Add(1)
}

// RecordWriteError records a write error.
func (m *MetricsCollector) RecordWriteError() {
	m.writeErrors.Add(1)
}

// RecordReadError records a read error.
func (m *MetricsCollector) RecordReadError() {
	m.readErrors.Add(1)
}

// RecordEventLatency records event processing latency in microseconds.
func (m *MetricsCollector) RecordEventLatency(latencyUs int64) {
	// Simple spinlock for latency array
	for !m.latencyMu.CompareAndSwap(0, 1) {
		// Spin
	}
	defer m.latencyMu.Store(0)

	// Keep only recent samples
	if len(m.latencies) >= 1000 {
		m.latencies = m.latencies[500:] // Drop oldest half
	}
	m.latencies = append(m.latencies, latencyUs)
}

// Snapshot returns current metrics.
func (m *MetricsCollector) Snapshot() *ServerMetrics {
	metrics := &ServerMetrics{
		EventsReceived:  m.eventsReceived.Load(),
		EventsProcessed: m.eventsProcessed.Load(),
		EventsDropped:   m.eventsDropped.Load(),
		PatchesSent:     m.patchesSent.Load(),
		PatchBytes:      m.patchBytes.Load(),
		BytesSent:       m.bytesSent.Load(),
		BytesReceived:   m.bytesReceived.Load(),
		HandlerPanics:   m.handlerPanics.Load(),
		WriteErrors:     m.writeErrors.Load(),
		ReadErrors:      m.readErrors.Load(),
		CollectedAt:     time.Now(),
	}

	// Calculate latency percentiles
	metrics.EventLatencyP50, metrics.EventLatencyP99 = m.latencyPercentiles()

	return metrics
}

// latencyPercentiles calculates P50 and P99 latencies.
func (m *MetricsCollector) latencyPercentiles() (p50, p99 int64) {
	// Simple spinlock
	for !m.latencyMu.CompareAndSwap(0, 1) {
		// Spin
	}
	defer m.latencyMu.Store(0)

	n := len(m.latencies)
	if n == 0 {
		return 0, 0
	}

	// Copy and sort
	sorted := make([]int64, n)
	copy(sorted, m.latencies)

	// Simple sort (not efficient but fine for small arrays)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	p50 = sorted[n/2]
	p99 = sorted[(n*99)/100]

	return p50, p99
}

// Reset resets all counters.
func (m *MetricsCollector) Reset() {
	m.eventsReceived.Store(0)
	m.eventsProcessed.Store(0)
	m.eventsDropped.Store(0)
	m.patchesSent.Store(0)
	m.patchBytes.Store(0)
	m.bytesSent.Store(0)
	m.bytesReceived.Store(0)
	m.handlerPanics.Store(0)
	m.writeErrors.Store(0)
	m.readErrors.Store(0)

	// Clear latencies
	for !m.latencyMu.CompareAndSwap(0, 1) {
		// Spin
	}
	m.latencies = m.latencies[:0]
	m.latencyMu.Store(0)
}
