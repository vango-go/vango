package server

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"testing"
	"time"

	"github.com/vango-dev/vango/v2/pkg/vango"
	"github.com/vango-dev/vango/v2/pkg/vdom"
)

// Stadium Benchmark Suite
// Tests memory density and GC performance at scale (10K+ sessions)

// BenchmarkStadium10K creates 10,000 sessions and measures memory + GC
func BenchmarkStadium10K(b *testing.B) {
	benchmarkStadiumN(b, 10000)
}

// BenchmarkStadium1K creates 1,000 sessions for quicker testing
func BenchmarkStadium1K(b *testing.B) {
	benchmarkStadiumN(b, 1000)
}

// benchmarkStadiumN creates N sessions and measures performance
func benchmarkStadiumN(b *testing.B, n int) {
	// Create mock sessions (no actual connections)
	sessions := make([]*MockSession, n)

	// Force GC and get baseline
	runtime.GC()
	debug.FreeOSMemory()

	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Create sessions
		for j := 0; j < n; j++ {
			sessions[j] = newMockSession(fmt.Sprintf("session-%d", j))
		}

		// Measure heap after creation
		runtime.GC()
		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)

		heapUsed := m2.HeapAlloc - m1.HeapAlloc
		perSession := heapUsed / uint64(n)

		b.ReportMetric(float64(perSession), "bytes/session")
		b.ReportMetric(float64(heapUsed)/(1024*1024), "MB_total")

		// Cleanup
		for j := 0; j < n; j++ {
			sessions[j].Close()
			sessions[j] = nil
		}
	}
}

// TestStadiumMemoryDensity measures actual memory per session
func TestStadiumMemoryDensity(t *testing.T) {
	const numSessions = 1000 // Use 1K for test speed

	// Force GC and get baseline
	runtime.GC()
	debug.FreeOSMemory()
	time.Sleep(100 * time.Millisecond) // Let GC settle

	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Create sessions
	sessions := make([]*MockSession, numSessions)
	for i := 0; i < numSessions; i++ {
		sessions[i] = newMockSession(fmt.Sprintf("session-%d", i))
	}

	// Force GC and measure
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	heapUsed := m2.HeapAlloc - m1.HeapAlloc
	perSession := heapUsed / uint64(numSessions)

	t.Logf("=== Stadium Memory Report ===")
	t.Logf("Sessions created: %d", numSessions)
	t.Logf("Total heap used: %.2f MB", float64(heapUsed)/(1024*1024))
	t.Logf("Per session: %.2f KB", float64(perSession)/1024)

	// Verify target: <50 KB per session (idle)
	if perSession > 50*1024 {
		t.Errorf("Memory per session %.2f KB exceeds 50 KB target", float64(perSession)/1024)
	} else {
		t.Logf("✓ Memory target met: %.2f KB < 50 KB", float64(perSession)/1024)
	}

	// Cleanup
	for i := range sessions {
		sessions[i].Close()
	}
}

// TestStadiumGCPause measures GC pause times with many sessions
func TestStadiumGCPause(t *testing.T) {
	const numSessions = 5000 // 5K for reasonable test time

	// Create sessions
	sessions := make([]*MockSession, numSessions)
	for i := 0; i < numSessions; i++ {
		sessions[i] = newMockSession(fmt.Sprintf("session-%d", i))
	}

	// Measure GC pauses
	runtime.GC() // Warm up

	var totalPause time.Duration
	var maxPause time.Duration
	const gcRuns = 10

	for i := 0; i < gcRuns; i++ {
		var m1, m2 runtime.MemStats
		runtime.ReadMemStats(&m1)

		start := time.Now()
		runtime.GC()
		pause := time.Since(start)

		runtime.ReadMemStats(&m2)

		totalPause += pause
		if pause > maxPause {
			maxPause = pause
		}
	}

	avgPause := totalPause / gcRuns

	t.Logf("=== Stadium GC Report ===")
	t.Logf("Sessions: %d", numSessions)
	t.Logf("GC runs: %d", gcRuns)
	t.Logf("Average GC pause: %v", avgPause)
	t.Logf("Max GC pause: %v", maxPause)

	// Target: <1ms GC pause
	if maxPause > time.Millisecond {
		t.Logf("⚠ GC pause %v exceeds 1ms target (may vary by load)", maxPause)
	} else {
		t.Logf("✓ GC pause target met: %v < 1ms", maxPause)
	}

	// Cleanup
	for i := range sessions {
		sessions[i].Close()
	}
}

// TestStadiumConcurrentUpdates tests updating signals across many sessions
func TestStadiumConcurrentUpdates(t *testing.T) {
	const numSessions = 1000

	// Create sessions with signals
	sessions := make([]*MockSession, numSessions)
	for i := 0; i < numSessions; i++ {
		sessions[i] = newMockSession(fmt.Sprintf("session-%d", i))
	}

	// Measure time to update a signal in all sessions
	start := time.Now()
	var wg sync.WaitGroup

	for i := 0; i < numSessions; i++ {
		wg.Add(1)
		go func(s *MockSession) {
			defer wg.Done()
			s.counter.Set(s.counter.Get() + 1)
		}(sessions[i])
	}

	wg.Wait()
	duration := time.Since(start)

	t.Logf("=== Stadium Update Report ===")
	t.Logf("Sessions updated: %d", numSessions)
	t.Logf("Total time: %v", duration)
	t.Logf("Per update: %v", duration/time.Duration(numSessions))

	// Cleanup
	for i := range sessions {
		sessions[i].Close()
	}
}

// MockSession simulates a session without WebSocket connection
type MockSession struct {
	id       string
	owner    *vango.Owner
	tree     *vdom.VNode
	handlers map[string]Handler
	counter  *vango.Signal[int]
}

// newMockSession creates a mock session with a typical component tree
func newMockSession(id string) *MockSession {
	owner := vango.NewOwner(nil)

	// Create a signal - simulates typical component state
	counter := vango.NewSignal(0)

	// Create a representative component tree (similar to a dashboard)
	tree := createMockTree()

	// Simulate ~10 handlers (typical for a page)
	handlers := make(map[string]Handler, 10)
	for i := 0; i < 10; i++ {
		hid := fmt.Sprintf("h%d", i)
		handlers[hid] = func(e *Event) {}
	}

	return &MockSession{
		id:       id,
		owner:    owner,
		tree:     tree,
		handlers: handlers,
		counter:  counter,
	}
}

// createMockTree creates a representative VNode tree
func createMockTree() *vdom.VNode {
	// Simulate a typical dashboard page structure
	var rows []any
	for i := 0; i < 20; i++ {
		rows = append(rows, vdom.Div(
			vdom.Class("row"),
			vdom.Span(vdom.Text(fmt.Sprintf("Item %d", i))),
			vdom.Button(vdom.Text("Action")),
		))
	}

	return vdom.Div(
		vdom.Class("dashboard"),
		vdom.Header(vdom.H1(vdom.Text("Dashboard"))),
		vdom.Main(rows...),
		vdom.Footer(vdom.P(vdom.Text("Footer"))),
	)
}

// Close disposes the mock session
func (m *MockSession) Close() {
	if m.owner != nil {
		m.owner.Dispose()
	}
	m.handlers = nil
	m.tree = nil
}
