package server

import (
	"testing"
	"time"
)

func TestNewMetricsCollector(t *testing.T) {
	mc := NewMetricsCollector()
	if mc == nil {
		t.Fatal("NewMetricsCollector should not return nil")
	}
	if mc.latencies == nil {
		t.Error("latencies should be initialized")
	}
}

func TestRecordEventReceived(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordEventReceived()
	mc.RecordEventReceived()
	mc.RecordEventReceived()

	snapshot := mc.Snapshot()
	if snapshot.EventsReceived != 3 {
		t.Errorf("EventsReceived = %d, want 3", snapshot.EventsReceived)
	}
}

func TestRecordEventProcessed(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordEventProcessed()
	mc.RecordEventProcessed()

	snapshot := mc.Snapshot()
	if snapshot.EventsProcessed != 2 {
		t.Errorf("EventsProcessed = %d, want 2", snapshot.EventsProcessed)
	}
}

func TestRecordEventDropped(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordEventDropped()

	snapshot := mc.Snapshot()
	if snapshot.EventsDropped != 1 {
		t.Errorf("EventsDropped = %d, want 1", snapshot.EventsDropped)
	}
}

func TestRecordPatchesSent(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordPatchesSent(5, 1024)
	mc.RecordPatchesSent(3, 512)

	snapshot := mc.Snapshot()
	if snapshot.PatchesSent != 8 {
		t.Errorf("PatchesSent = %d, want 8", snapshot.PatchesSent)
	}
	if snapshot.PatchBytes != 1536 {
		t.Errorf("PatchBytes = %d, want 1536", snapshot.PatchBytes)
	}
}

func TestRecordBytesSent(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordBytesSent(100)
	mc.RecordBytesSent(200)

	snapshot := mc.Snapshot()
	if snapshot.BytesSent != 300 {
		t.Errorf("BytesSent = %d, want 300", snapshot.BytesSent)
	}
}

func TestRecordBytesReceived(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordBytesReceived(50)
	mc.RecordBytesReceived(75)

	snapshot := mc.Snapshot()
	if snapshot.BytesReceived != 125 {
		t.Errorf("BytesReceived = %d, want 125", snapshot.BytesReceived)
	}
}

func TestRecordErrors(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordHandlerPanic()
	mc.RecordWriteError()
	mc.RecordWriteError()
	mc.RecordReadError()
	mc.RecordReadError()
	mc.RecordReadError()

	snapshot := mc.Snapshot()
	if snapshot.HandlerPanics != 1 {
		t.Errorf("HandlerPanics = %d, want 1", snapshot.HandlerPanics)
	}
	if snapshot.WriteErrors != 2 {
		t.Errorf("WriteErrors = %d, want 2", snapshot.WriteErrors)
	}
	if snapshot.ReadErrors != 3 {
		t.Errorf("ReadErrors = %d, want 3", snapshot.ReadErrors)
	}
}

func TestRecordEventLatency(t *testing.T) {
	mc := NewMetricsCollector()

	// Record some latencies
	for i := 0; i < 100; i++ {
		mc.RecordEventLatency(int64(i * 100)) // 0, 100, 200, ..., 9900 microseconds
	}

	snapshot := mc.Snapshot()

	// P50 should be around 50th percentile
	if snapshot.EventLatencyP50 < 4000 || snapshot.EventLatencyP50 > 5500 {
		t.Errorf("EventLatencyP50 = %d, expected around 4500-5000", snapshot.EventLatencyP50)
	}

	// P99 should be around 99th percentile
	if snapshot.EventLatencyP99 < 9000 {
		t.Errorf("EventLatencyP99 = %d, expected >= 9000", snapshot.EventLatencyP99)
	}
}

func TestRecordEventLatencyEmpty(t *testing.T) {
	mc := NewMetricsCollector()

	snapshot := mc.Snapshot()
	if snapshot.EventLatencyP50 != 0 {
		t.Errorf("EventLatencyP50 should be 0 when no latencies recorded")
	}
	if snapshot.EventLatencyP99 != 0 {
		t.Errorf("EventLatencyP99 should be 0 when no latencies recorded")
	}
}

func TestRecordEventLatencyOverflow(t *testing.T) {
	mc := NewMetricsCollector()

	// Record more than capacity
	for i := 0; i < 1500; i++ {
		mc.RecordEventLatency(int64(i))
	}

	// Should not panic and latencies should be trimmed
	snapshot := mc.Snapshot()
	if snapshot.EventLatencyP50 == 0 {
		t.Error("P50 should not be 0 after recording many latencies")
	}
}

func TestReset(t *testing.T) {
	mc := NewMetricsCollector()

	// Record various metrics
	mc.RecordEventReceived()
	mc.RecordEventProcessed()
	mc.RecordPatchesSent(10, 5000)
	mc.RecordEventLatency(100)

	// Reset
	mc.Reset()

	// Verify all are zero
	snapshot := mc.Snapshot()
	if snapshot.EventsReceived != 0 {
		t.Error("EventsReceived should be 0 after reset")
	}
	if snapshot.EventsProcessed != 0 {
		t.Error("EventsProcessed should be 0 after reset")
	}
	if snapshot.PatchesSent != 0 {
		t.Error("PatchesSent should be 0 after reset")
	}
	if snapshot.PatchBytes != 0 {
		t.Error("PatchBytes should be 0 after reset")
	}
	if snapshot.EventLatencyP50 != 0 {
		t.Error("EventLatencyP50 should be 0 after reset")
	}
}

func TestSnapshotTimestamp(t *testing.T) {
	mc := NewMetricsCollector()

	before := time.Now()
	snapshot := mc.Snapshot()
	after := time.Now()

	if snapshot.CollectedAt.Before(before) || snapshot.CollectedAt.After(after) {
		t.Error("CollectedAt should be between before and after")
	}
}

func TestServerMetricsStruct(t *testing.T) {
	// Test that ServerMetrics can hold all expected fields
	sm := ServerMetrics{
		ActiveSessions:  100,
		TotalSessions:   500,
		SessionCreates:  500,
		SessionCloses:   400,
		PeakSessions:    150,
		EventsReceived:  10000,
		EventsProcessed: 9990,
		EventsDropped:   10,
		PatchesSent:     5000,
		PatchBytes:      1024 * 1024,
		BytesSent:       2 * 1024 * 1024,
		BytesReceived:   512 * 1024,
		HandlerPanics:   2,
		WriteErrors:     5,
		ReadErrors:      3,
		EventLatencyP50: 500,
		EventLatencyP99: 2000,
		TotalMemory:     50 * 1024 * 1024,
		CollectedAt:     time.Now(),
	}

	if sm.ActiveSessions != 100 {
		t.Error("ActiveSessions not stored correctly")
	}
	if sm.TotalMemory != 50*1024*1024 {
		t.Error("TotalMemory not stored correctly")
	}
}

func TestConcurrentMetricsRecording(t *testing.T) {
	mc := NewMetricsCollector()
	done := make(chan bool)

	// Run multiple goroutines recording metrics
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				mc.RecordEventReceived()
				mc.RecordEventProcessed()
				mc.RecordEventLatency(int64(j))
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	snapshot := mc.Snapshot()
	if snapshot.EventsReceived != 1000 {
		t.Errorf("EventsReceived = %d, want 1000", snapshot.EventsReceived)
	}
	if snapshot.EventsProcessed != 1000 {
		t.Errorf("EventsProcessed = %d, want 1000", snapshot.EventsProcessed)
	}
}
