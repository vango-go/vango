// Vango E2E Load Benchmark
//
// This benchmark is designed to answer the questions we actually care about in production:
// - What is the p50/p95/p99 roundtrip latency under concurrent load?
// - How much allocation + GC work does that load generate?
//
// It runs the real Vango WebSocket server and drives N concurrent WebSocket clients that
// send real protocol frames (handshake + event frames) and wait for the corresponding
// patch response.
//
// This is intentionally "browserless" (no DOM). It measures:
// client send → kernel → server decode → handler → render → diff → patch encode → WS write → client read/decode
//
// Run:
//   cd vango_v2/benchmark/e2e_load
//   go run . -clients=200 -duration=30s -rps=5 -list=50
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"runtime"
	"runtime/metrics"
	"runtime/debug"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vango-go/vango/pkg/protocol"
	"github.com/vango-go/vango/pkg/server"
	"github.com/vango-go/vango/pkg/vango"
	. "github.com/vango-go/vango/pkg/vdom"
)

func main() {
	var (
		clients      = flag.Int("clients", 100, "number of concurrent websocket clients")
		duration     = flag.Duration("duration", 15*time.Second, "how long to run the load test")
		rps          = flag.Float64("rps", 2, "target events/sec per client (best-effort, response-gated)")
		listSize     = flag.Int("list", 50, "list size rendered per session (affects render/diff cost)")
		payloadBytes = flag.Int("payload-bytes", 24, "bytes of token payload per event (affects protocol + patch size)")
	)
	flag.Parse()

	if *clients <= 0 {
		log.Fatal("-clients must be > 0")
	}
	if *duration <= 0 {
		log.Fatal("-duration must be > 0")
	}
	if *rps <= 0 {
		log.Fatal("-rps must be > 0")
	}
	if *listSize < 0 {
		log.Fatal("-list must be >= 0")
	}
	if *payloadBytes < 0 {
		log.Fatal("-payload-bytes must be >= 0")
	}

	// Reduce incidental variability a bit.
	debug.SetGCPercent(100)

	srv := server.New(&server.ServerConfig{
		// Address isn't used by httptest, but some internal logging/heuristics read it.
		Address: "127.0.0.1:0",
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	})
	srv.SetRootComponent(func() server.Component {
		return NewLoadApp(*listSize)
	})

	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	httpServer := &http.Server{Handler: srv.Handler()}
	go func() {
		_ = httpServer.Serve(ln)
	}()
	defer func() {
		_ = httpServer.Shutdown(context.Background())
	}()

	wsURL := "ws://" + ln.Addr().String() + "/_vango/ws"

	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	samplesCh := make(chan time.Duration, 1024)
	var samples []time.Duration
	var samplesMu sync.Mutex
	collectorDone := make(chan struct{})
	go func() {
		defer close(collectorDone)
		for rtt := range samplesCh {
			samplesMu.Lock()
			samples = append(samples, rtt)
			samplesMu.Unlock()
		}
	}()

	var (
		totalEvents atomic.Uint64
		totalErrors atomic.Uint64
	)

	var before runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	beforeMetrics := readRuntimeMetrics()

	var wg sync.WaitGroup
	wg.Add(*clients)
	for i := 0; i < *clients; i++ {
		clientID := i
		go func() {
			defer wg.Done()
			if err := runClient(ctx, wsURL, clientID, *rps, *payloadBytes, samplesCh, &totalEvents, &totalErrors); err != nil {
				totalErrors.Add(1)
			}
		}()
	}

	wg.Wait()
	close(samplesCh)
	<-collectorDone

	var after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&after)
	afterMetrics := readRuntimeMetrics()

	samplesMu.Lock()
	latencies := append([]time.Duration(nil), samples...)
	samplesMu.Unlock()
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	total := totalEvents.Load()
	errs := totalErrors.Load()
	runSeconds := math.Max(0.001, (*duration).Seconds())

	fmt.Println("=== Vango E2E Load Benchmark ===")
	fmt.Printf("Clients: %d\n", *clients)
	fmt.Printf("Duration: %s\n", (*duration).String())
	fmt.Printf("Target per-client rate: %.2f events/s\n", *rps)
	fmt.Printf("List size: %d\n", *listSize)
	fmt.Printf("Payload bytes: %d\n", *payloadBytes)
	fmt.Printf("Total events: %d\n", total)
	fmt.Printf("Errors: %d\n", errs)
	fmt.Printf("Throughput: %.1f events/s\n", float64(total)/runSeconds)
	fmt.Println()

	if len(latencies) == 0 {
		fmt.Println("No latency samples recorded.")
	} else {
		fmt.Println("RTT (client send → server → client receive+decode):")
		fmt.Printf("  min: %s\n", latencies[0])
		fmt.Printf("  p50: %s\n", percentile(latencies, 0.50))
		fmt.Printf("  p95: %s\n", percentile(latencies, 0.95))
		fmt.Printf("  p99: %s\n", percentile(latencies, 0.99))
		fmt.Printf("  max: %s\n", latencies[len(latencies)-1])
	}
	fmt.Println()

	fmt.Println("Go runtime / GC (process-wide):")
	fmt.Printf("  alloc:     %.2f MB\n", float64(after.TotalAlloc-before.TotalAlloc)/(1024*1024))
	fmt.Printf("  heap_live: %.2f MB\n", float64(after.HeapAlloc)/(1024*1024))
	fmt.Printf("  num_gc:    %d\n", after.NumGC-before.NumGC)
	fmt.Printf("  gc_pause:  %s (total)\n", time.Duration(after.PauseTotalNs-before.PauseTotalNs))
	fmt.Printf("  gc_pause:  %s (avg)\n", avgPause(after, before))
	fmt.Printf("  gc_cpu:    %.2f%%\n", 100*cpuFraction(afterMetrics, beforeMetrics))
	fmt.Printf("  allocs:    %.2f M objects\n", float64(afterMetrics.heapAllocsObjects-beforeMetrics.heapAllocsObjects)/1_000_000)
}

func avgPause(after, before runtime.MemStats) time.Duration {
	gcCount := after.NumGC - before.NumGC
	if gcCount == 0 {
		return 0
	}
	return time.Duration((after.PauseTotalNs - before.PauseTotalNs) / uint64(gcCount))
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[len(sorted)-1]
	}
	idx := int(math.Ceil(float64(len(sorted))*p)) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

type runtimeMetricsSnapshot struct {
	cpuTotalSeconds float64
	cpuGCSeconds    float64

	heapAllocsBytes   uint64
	heapAllocsObjects uint64
}

func readRuntimeMetrics() runtimeMetricsSnapshot {
	samples := []metrics.Sample{
		{Name: "/cpu/classes/total:cpu-seconds"},
		{Name: "/cpu/classes/gc/total:cpu-seconds"},
		{Name: "/gc/heap/allocs:bytes"},
		{Name: "/gc/heap/allocs:objects"},
	}
	metrics.Read(samples)

	var out runtimeMetricsSnapshot
	for _, s := range samples {
		switch s.Name {
		case "/cpu/classes/total:cpu-seconds":
			out.cpuTotalSeconds = s.Value.Float64()
		case "/cpu/classes/gc/total:cpu-seconds":
			out.cpuGCSeconds = s.Value.Float64()
		case "/gc/heap/allocs:bytes":
			out.heapAllocsBytes = s.Value.Uint64()
		case "/gc/heap/allocs:objects":
			out.heapAllocsObjects = s.Value.Uint64()
		}
	}
	return out
}

func cpuFraction(after, before runtimeMetricsSnapshot) float64 {
	total := after.cpuTotalSeconds - before.cpuTotalSeconds
	if total <= 0 {
		return 0
	}
	gc := after.cpuGCSeconds - before.cpuGCSeconds
	if gc < 0 {
		return 0
	}
	return gc / total
}

func runClient(
	ctx context.Context,
	wsURL string,
	clientID int,
	rps float64,
	payloadBytes int,
	samples chan<- time.Duration,
	totalEvents *atomic.Uint64,
	totalErrors *atomic.Uint64,
) error {
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	// Handshake: client hello is sent as raw bytes (not framed).
	ch := protocol.NewClientHello("")
	ch.ViewportW = 1280
	ch.ViewportH = 720
	ch.TZOffset = 0

	if err := conn.WriteMessage(websocket.BinaryMessage, protocol.EncodeClientHello(ch)); err != nil {
		return fmt.Errorf("handshake write: %w", err)
	}

	_, msg, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("handshake read: %w", err)
	}
	frame, err := protocol.DecodeFrame(msg)
	if err != nil {
		return fmt.Errorf("handshake frame decode: %w", err)
	}
	if frame.Type != protocol.FrameHandshake {
		return fmt.Errorf("handshake: expected FrameHandshake, got %v", frame.Type)
	}
	sh, err := protocol.DecodeServerHello(frame.Payload)
	if err != nil {
		return fmt.Errorf("handshake server hello decode: %w", err)
	}
	if sh.Status != protocol.HandshakeOK {
		return fmt.Errorf("handshake failed: %s", sh.Status.String())
	}

	// Benchmark contract: the root component renders a single input element as the first child.
	// With pre-order HID assignment, the input element is always HID "h2":
	//   root Div = h1
	//   first child Input = h2
	const inputHID = "h2"

	period := time.Duration(float64(time.Second) / rps)
	var seq uint64

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		seq++
		token := makeToken(clientID, seq, payloadBytes)

		start := time.Now()

		evt := &protocol.Event{
			Seq:     seq,
			Type:    protocol.EventInput,
			HID:     inputHID,
			Payload: token,
		}
		evFrame := protocol.NewFrame(protocol.FrameEvent, protocol.EncodeEvent(evt))
		if err := conn.WriteMessage(websocket.BinaryMessage, evFrame.Encode()); err != nil {
			totalErrors.Add(1)
			return fmt.Errorf("event write: %w", err)
		}

		// Wait for the patch that echoes our token (SetText or SetValue).
		found, err := waitForToken(ctx, conn, token)
		if err != nil {
			totalErrors.Add(1)
			return fmt.Errorf("wait for token: %w", err)
		}
		if !found {
			totalErrors.Add(1)
			return fmt.Errorf("token not observed in patches")
		}

		rtt := time.Since(start)
		totalEvents.Add(1)
		samples <- rtt

		// Best-effort pacing. We intentionally gate on response to measure real queueing/tail behavior.
		elapsed := time.Since(start)
		if sleep := period - elapsed; sleep > 0 {
			timer := time.NewTimer(sleep)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil
			case <-timer.C:
			}
		}
	}
}

func waitForToken(ctx context.Context, conn *websocket.Conn, token string) (bool, error) {
	for {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			return false, err
		}
		frame, err := protocol.DecodeFrame(msg)
		if err != nil {
			return false, err
		}

		switch frame.Type {
		case protocol.FramePatches:
			pf, err := protocol.DecodePatches(frame.Payload)
			if err != nil {
				return false, err
			}
			for _, p := range pf.Patches {
				if (p.Op == protocol.PatchSetText || p.Op == protocol.PatchSetValue) && p.Value == token {
					return true, nil
				}
			}

		case protocol.FrameError:
			return false, fmt.Errorf("server error frame")

		default:
			// Ignore control/ack/etc.
		}
	}
}

func makeToken(clientID int, seq uint64, payloadBytes int) string {
	// Always include client+seq for debugging, then pad with random bytes.
	prefix := fmt.Sprintf("c%d:%d:", clientID, seq)
	if payloadBytes <= len(prefix) {
		return prefix[:payloadBytes]
	}

	need := payloadBytes - len(prefix)
	if need < 0 {
		need = 0
	}

	raw := make([]byte, (need+1)/2)
	_, _ = rand.Read(raw)
	suffix := hex.EncodeToString(raw)
	if len(suffix) > need {
		suffix = suffix[:need]
	}
	return prefix + suffix
}

// LoadApp is a component that is intentionally small but non-trivial:
// - a single input handler (for stable HID targeting)
// - an echo element that updates with every event (for client correlation)
// - a rendered list that changes on every event (to exercise render+diff cost)
type LoadApp struct {
	echo  *vango.Signal[string]
	items *vango.Signal[[]string]
}

func NewLoadApp(listSize int) *LoadApp {
	items := make([]string, listSize)
	for i := range items {
		items[i] = fmt.Sprintf("Item %d", i)
	}
	return &LoadApp{
		echo:  vango.NewSignal(""),
		items: vango.NewSignal(items),
	}
}

func (a *LoadApp) Render() *VNode {
	// Note: the input is the first child to keep the HID stable (h2).
	return Div(
		Input(
			Type("text"),
			OnInput(func(value string) {
				a.echo.Set(value)

				// Update one list item based on a stable hash of the payload.
				// IMPORTANT: because Signal equality for slices uses reflect.DeepEqual,
				// we must copy to avoid mutating the old value in-place (which would be "invisible").
				a.items.Update(func(prev []string) []string {
					if len(prev) == 0 {
						return prev
					}
					next := make([]string, len(prev))
					copy(next, prev)
					idx := int(fnv1a32(value) % uint32(len(next)))
					next[idx] = value
					return next
				})
			}),
		),

		// Echo element: patches will contain the token in PatchSetText for correlation.
		Div(ID("echo"), Text(a.echo.Get())),

		// A small list to force a meaningful diff/render on each event.
		Ul(a.renderItems()...),
	)
}

func (a *LoadApp) renderItems() []any {
	items := a.items.Get()
	nodes := make([]any, 0, len(items))
	for i, it := range items {
		nodes = append(nodes, Li(Key(i), Text(it)))
	}
	return nodes
}

func fnv1a32(s string) uint32 {
	const (
		offset32 = 2166136261
		prime32  = 16777619
	)
	var h uint32 = offset32
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= prime32
	}
	return h
}
