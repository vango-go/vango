package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"runtime/debug"
	"runtime/metrics"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vango-go/vango/pkg/protocol"
	"github.com/vango-go/vango/pkg/server"
	"github.com/vango-go/vango/pkg/vango"
	. "github.com/vango-go/vango/pkg/vdom"
)

const (
	gib = int64(1024 * 1024 * 1024)
)

type profile struct {
	Name          string
	Clients       int
	Duration      time.Duration
	RPS           float64
	ListSize      int
	PayloadBytes  int
	MaxProcs      int
	MemLimitBytes int64
}

var profiles = map[string]profile{
	"fast": {
		Name:         "fast",
		Clients:      50,
		Duration:     10 * time.Second,
		RPS:          2,
		ListSize:     20,
		PayloadBytes: 24,
	},
	"standard": {
		Name:         "standard",
		Clients:      200,
		Duration:     30 * time.Second,
		RPS:          5,
		ListSize:     50,
		PayloadBytes: 24,
	},
	"stress": {
		Name:          "stress",
		Clients:       500,
		Duration:      60 * time.Second,
		RPS:           10,
		ListSize:      100,
		PayloadBytes:  24,
		MaxProcs:      4,
		MemLimitBytes: 2 * gib,
	},
}

type benchConfig struct {
	Profile       string
	Clients       int
	Duration      time.Duration
	RPS           float64
	ListSize      int
	PayloadBytes  int
	MaxProcs      int
	MemLimitBytes int64
	JSONOutput    string
	EventTimeout  time.Duration
}

type benchCounters struct {
	eventsSent     atomic.Uint64
	eventsComplete atomic.Uint64
	eventBytes     atomic.Uint64
	patchBytes     atomic.Uint64
	patchFrames    atomic.Uint64
	patchesTotal   atomic.Uint64
}

type benchErrors struct {
	handshakeFailures   atomic.Uint64
	eventWriteFailures  atomic.Uint64
	frameDecodeFailures atomic.Uint64
	patchDecodeFailures atomic.Uint64
	serverErrorFrames   atomic.Uint64
	tokenMissing        atomic.Uint64
	totalErrors         atomic.Uint64
}

type patchOpCounts struct {
	counts [256]atomic.Uint64
}

func (p *patchOpCounts) add(op protocol.PatchOp) {
	p.counts[uint8(op)].Add(1)
}

func (p *patchOpCounts) snapshot() map[string]uint64 {
	out := make(map[string]uint64)
	for i := range p.counts {
		count := p.counts[i].Load()
		if count == 0 {
			continue
		}
		name := protocol.PatchOp(uint8(i)).String()
		if name == "Unknown" {
			name = fmt.Sprintf("0x%02x", i)
		}
		out[name] = count
	}
	return out
}

func main() {
	log.SetFlags(0)

	cfg, err := parseConfig()
	if err != nil {
		log.Fatal(err)
	}

	if cfg.MaxProcs > 0 {
		runtime.GOMAXPROCS(cfg.MaxProcs)
	}
	if cfg.MemLimitBytes > 0 {
		debug.SetMemoryLimit(cfg.MemLimitBytes)
	}

	debug.SetGCPercent(100)

	srv := server.New(&server.ServerConfig{
		Address: "127.0.0.1:0",
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	})
	listSize := cfg.ListSize
	srv.SetRootComponent(func() server.Component {
		return NewLoadApp(listSize)
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

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Duration)
	defer cancel()

	samplesCh := make(chan time.Duration, sampleBuffer(cfg.Clients))
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

	var counters benchCounters
	var errCounts benchErrors
	var patchOps patchOpCounts

	var before runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	beforeMetrics := readRuntimeMetrics()

	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(cfg.Clients)
	for i := 0; i < cfg.Clients; i++ {
		clientID := i
		go func() {
			defer wg.Done()
			if err := runClient(ctx, wsURL, clientID, cfg, &counters, &errCounts, &patchOps, samplesCh); err != nil {
				errCounts.totalErrors.Add(1)
			}
		}()
	}

	wg.Wait()
	close(samplesCh)
	<-collectorDone

	elapsed := time.Since(start)

	var after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&after)
	afterMetrics := readRuntimeMetrics()

	samplesMu.Lock()
	latencies := append([]time.Duration(nil), samples...)
	samplesMu.Unlock()
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	report := buildReport(cfg, elapsed, latencies, &counters, &errCounts, &patchOps, before, after, beforeMetrics, afterMetrics)

	writeSummary(os.Stderr, report)
	if err := writeJSON(cfg.JSONOutput, report); err != nil {
		log.Fatalf("write json: %v", err)
	}
}

func sampleBuffer(clients int) int {
	if clients < 1 {
		return 1024
	}
	buf := clients * 4
	if buf < 1024 {
		buf = 1024
	}
	return buf
}

func parseConfig() (benchConfig, error) {
	profileFlag := flag.String("profile", "standard", "profile: fast|standard|stress")
	clientsFlag := flag.Int("clients", -1, "number of concurrent websocket clients")
	durationFlag := flag.String("duration", "", "benchmark duration, e.g. 30s")
	rpsFlag := flag.Float64("rps", -1, "target events/sec per client")
	listFlag := flag.Int("list", -1, "list size rendered per session")
	payloadFlag := flag.Int("payload-bytes", -1, "bytes of token payload per event")
	maxProcsFlag := flag.Int("max-procs", -1, "GOMAXPROCS cap (0 to leave unchanged)")
	memLimitFlag := flag.String("mem-limit", "", "GOMEMLIMIT (e.g. 2GiB)")
	jsonFlag := flag.String("json", "-", "JSON output path ('-' for stdout)")
	flag.Parse()

	name := strings.ToLower(strings.TrimSpace(*profileFlag))
	if name == "" {
		name = "standard"
	}

	base, ok := profiles[name]
	if !ok {
		return benchConfig{}, fmt.Errorf("unknown profile %q", name)
	}

	cfg := benchConfig{
		Profile:       base.Name,
		Clients:       base.Clients,
		Duration:      base.Duration,
		RPS:           base.RPS,
		ListSize:      base.ListSize,
		PayloadBytes:  base.PayloadBytes,
		MaxProcs:      base.MaxProcs,
		MemLimitBytes: base.MemLimitBytes,
		JSONOutput:    strings.TrimSpace(*jsonFlag),
	}

	if *clientsFlag != -1 {
		cfg.Clients = *clientsFlag
	}
	if *durationFlag != "" {
		d, err := time.ParseDuration(*durationFlag)
		if err != nil {
			return benchConfig{}, fmt.Errorf("invalid -duration: %w", err)
		}
		cfg.Duration = d
	}
	if *rpsFlag != -1 {
		cfg.RPS = *rpsFlag
	}
	if *listFlag != -1 {
		cfg.ListSize = *listFlag
	}
	if *payloadFlag != -1 {
		cfg.PayloadBytes = *payloadFlag
	}
	if *maxProcsFlag != -1 {
		cfg.MaxProcs = *maxProcsFlag
	}
	if *memLimitFlag != "" {
		limit, err := parseBytes(*memLimitFlag)
		if err != nil {
			return benchConfig{}, fmt.Errorf("invalid -mem-limit: %w", err)
		}
		cfg.MemLimitBytes = limit
	}
	if cfg.JSONOutput == "" {
		cfg.JSONOutput = "-"
	}

	if cfg.Clients <= 0 {
		return benchConfig{}, errors.New("-clients must be > 0")
	}
	if cfg.Duration <= 0 {
		return benchConfig{}, errors.New("-duration must be > 0")
	}
	if cfg.RPS <= 0 {
		return benchConfig{}, errors.New("-rps must be > 0")
	}
	if cfg.ListSize < 0 {
		return benchConfig{}, errors.New("-list must be >= 0")
	}
	if cfg.PayloadBytes <= 0 {
		return benchConfig{}, errors.New("-payload-bytes must be > 0")
	}
	if cfg.MaxProcs < 0 {
		return benchConfig{}, errors.New("-max-procs must be >= 0")
	}
	if cfg.MemLimitBytes < 0 {
		return benchConfig{}, errors.New("-mem-limit must be >= 0")
	}

	cfg.EventTimeout = eventTimeout(cfg.RPS)
	return cfg, nil
}

func eventTimeout(rps float64) time.Duration {
	if rps <= 0 {
		return 0
	}
	period := time.Duration(float64(time.Second) / rps)
	timeout := period * 10
	if timeout < 2*time.Second {
		timeout = 2 * time.Second
	}
	return timeout
}

func parseBytes(input string) (int64, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return 0, errors.New("empty size")
	}

	var i int
	for i < len(s) {
		c := s[i]
		if (c >= '0' && c <= '9') || c == '.' {
			i++
			continue
		}
		break
	}
	if i == 0 {
		return 0, fmt.Errorf("invalid size %q", input)
	}

	numPart := strings.TrimSpace(s[:i])
	suffix := strings.ToLower(strings.TrimSpace(s[i:]))

	value, err := strconv.ParseFloat(numPart, 64)
	if err != nil {
		return 0, err
	}

	multiplier := float64(1)
	switch suffix {
	case "", "b":
		multiplier = 1
	case "kb":
		multiplier = 1e3
	case "mb":
		multiplier = 1e6
	case "gb":
		multiplier = 1e9
	case "tb":
		multiplier = 1e12
	case "kib":
		multiplier = 1024
	case "mib":
		multiplier = 1024 * 1024
	case "gib":
		multiplier = 1024 * 1024 * 1024
	case "tib":
		multiplier = 1024 * 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("unknown size suffix %q", suffix)
	}

	bytes := value * multiplier
	if bytes < 0 {
		return 0, fmt.Errorf("invalid size %q", input)
	}

	return int64(bytes + 0.5), nil
}

func runClient(
	ctx context.Context,
	wsURL string,
	clientID int,
	cfg benchConfig,
	counters *benchCounters,
	errCounts *benchErrors,
	patchOps *patchOpCounts,
	samples chan<- time.Duration,
) error {
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		errCounts.handshakeFailures.Add(1)
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	ch := protocol.NewClientHello("")
	ch.ViewportW = 1280
	ch.ViewportH = 720
	ch.TZOffset = 0

	handshakeFrame := protocol.NewFrame(protocol.FrameHandshake, protocol.EncodeClientHello(ch))
	if err := conn.WriteMessage(websocket.BinaryMessage, handshakeFrame.Encode()); err != nil {
		errCounts.handshakeFailures.Add(1)
		return fmt.Errorf("handshake write: %w", err)
	}

	_, msg, err := conn.ReadMessage()
	if err != nil {
		errCounts.handshakeFailures.Add(1)
		return fmt.Errorf("handshake read: %w", err)
	}
	frame, err := protocol.DecodeFrame(msg)
	if err != nil {
		errCounts.handshakeFailures.Add(1)
		return fmt.Errorf("handshake frame decode: %w", err)
	}
	if frame.Type != protocol.FrameHandshake {
		errCounts.handshakeFailures.Add(1)
		return fmt.Errorf("handshake: expected FrameHandshake, got %v", frame.Type)
	}
	sh, err := protocol.DecodeServerHello(frame.Payload)
	if err != nil {
		errCounts.handshakeFailures.Add(1)
		return fmt.Errorf("handshake server hello decode: %w", err)
	}
	if sh.Status != protocol.HandshakeOK {
		errCounts.handshakeFailures.Add(1)
		return fmt.Errorf("handshake failed: %s", sh.Status.String())
	}

	const inputHID = "h2"
	period := time.Duration(float64(time.Second) / cfg.RPS)
	var seq uint64

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		seq++
		token := makeToken(clientID, seq, cfg.PayloadBytes)

		start := time.Now()

		evt := &protocol.Event{
			Seq:     seq,
			Type:    protocol.EventInput,
			HID:     inputHID,
			Payload: token,
		}
		evFrame := protocol.NewFrame(protocol.FrameEvent, protocol.EncodeEvent(evt))
		frameData := evFrame.Encode()
		if err := conn.WriteMessage(websocket.BinaryMessage, frameData); err != nil {
			errCounts.eventWriteFailures.Add(1)
			return fmt.Errorf("event write: %w", err)
		}

		counters.eventsSent.Add(1)
		counters.eventBytes.Add(uint64(len(frameData)))

		if cfg.EventTimeout > 0 {
			conn.SetReadDeadline(time.Now().Add(cfg.EventTimeout))
		}
		eventCtx, cancel := context.WithTimeout(ctx, cfg.EventTimeout)
		found, err := waitForToken(eventCtx, conn, token, counters, errCounts, patchOps)
		cancel()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || isTimeout(err) {
				errCounts.tokenMissing.Add(1)
				return fmt.Errorf("token not observed in patches")
			}
			return fmt.Errorf("wait for token: %w", err)
		}
		if !found {
			errCounts.tokenMissing.Add(1)
			return fmt.Errorf("token not observed in patches")
		}

		rtt := time.Since(start)
		counters.eventsComplete.Add(1)
		samples <- rtt

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

func waitForToken(
	ctx context.Context,
	conn *websocket.Conn,
	token string,
	counters *benchCounters,
	errCounts *benchErrors,
	patchOps *patchOpCounts,
) (bool, error) {
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
			errCounts.frameDecodeFailures.Add(1)
			return false, err
		}

		switch frame.Type {
		case protocol.FramePatches:
			counters.patchFrames.Add(1)
			counters.patchBytes.Add(uint64(len(msg)))
			pf, err := protocol.DecodePatches(frame.Payload)
			if err != nil {
				errCounts.patchDecodeFailures.Add(1)
				return false, err
			}
			for _, p := range pf.Patches {
				patchOps.add(p.Op)
				counters.patchesTotal.Add(1)
				if (p.Op == protocol.PatchSetText || p.Op == protocol.PatchSetValue) && p.Value == token {
					return true, nil
				}
			}

		case protocol.FrameError:
			errCounts.serverErrorFrames.Add(1)
			return false, fmt.Errorf("server error frame")

		default:
			// Ignore control/ack/etc.
		}
	}
}

func makeToken(clientID int, seq uint64, payloadBytes int) string {
	if payloadBytes <= 0 {
		return ""
	}
	seed := (uint64(clientID) << 32) ^ seq
	base := strings.ToLower(strconv.FormatUint(seed, 36))
	if len(base) >= payloadBytes {
		return base[len(base)-payloadBytes:]
	}
	pad := strings.Repeat("x", payloadBytes-len(base))
	return base + pad
}

func isTimeout(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}
	return false
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

func avgPause(after, before runtime.MemStats) time.Duration {
	gcCount := after.NumGC - before.NumGC
	if gcCount == 0 {
		return 0
	}
	return time.Duration((after.PauseTotalNs - before.PauseTotalNs) / uint64(gcCount))
}

func ms(d time.Duration) float64 {
	return float64(d) / float64(time.Millisecond)
}

type benchReport struct {
	Version    string         `json:"version"`
	Run        runInfo        `json:"run"`
	Workload   workloadInfo   `json:"workload"`
	LatencyMS  latencyInfo    `json:"latency_ms"`
	Throughput throughputInfo `json:"throughput"`
	GC         gcInfo         `json:"gc"`
	Protocol   protocolInfo   `json:"protocol"`
	Errors     errorInfo      `json:"errors"`
}

type runInfo struct {
	Timestamp string `json:"timestamp"`
	Go        string `json:"go"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	CPUCount  int    `json:"cpu_count"`
	GitCommit string `json:"git_commit,omitempty"`
}

type workloadInfo struct {
	Profile        string  `json:"profile"`
	Clients        int     `json:"clients"`
	DurationMS     int64   `json:"duration_ms"`
	RPSPerClient   float64 `json:"rps_per_client"`
	ListSize       int     `json:"list_size"`
	PayloadBytes   int     `json:"payload_bytes"`
	MaxProcs       int     `json:"max_procs"`
	MemLimitBytes  int64   `json:"mem_limit_bytes"`
	EventTimeoutMS int64   `json:"event_timeout_ms"`
}

type latencyInfo struct {
	Min float64 `json:"min"`
	P50 float64 `json:"p50"`
	P95 float64 `json:"p95"`
	P99 float64 `json:"p99"`
	Max float64 `json:"max"`
}

type throughputInfo struct {
	EventsTotal        uint64  `json:"events_total"`
	EventsPerSec       float64 `json:"events_per_sec"`
	EventsPerSecClient float64 `json:"events_per_sec_per_client"`
}

type gcInfo struct {
	AllocMB       float64 `json:"alloc_mb"`
	HeapLiveMB    float64 `json:"heap_live_mb"`
	NumGC         uint32  `json:"num_gc"`
	PauseTotalMS  float64 `json:"pause_total_ms"`
	PauseAvgMS    float64 `json:"pause_avg_ms"`
	GCCPUFraction float64 `json:"gc_cpu_fraction"`
	AllocsObjects uint64  `json:"allocs_objects"`
}

type protocolInfo struct {
	EventBytesTotal uint64            `json:"event_bytes_total"`
	PatchBytesTotal uint64            `json:"patch_bytes_total"`
	PatchFrames     uint64            `json:"patch_frames_total"`
	PatchesTotal    uint64            `json:"patches_total"`
	AvgEventBytes   float64           `json:"avg_event_bytes"`
	AvgPatchBytes   float64           `json:"avg_patch_bytes"`
	PatchesPerEvent float64           `json:"patches_per_event"`
	PatchOps        map[string]uint64 `json:"patch_ops"`
}

type errorInfo struct {
	TotalErrors         uint64 `json:"total_errors"`
	HandshakeFailures   uint64 `json:"handshake_failures"`
	EventWriteFailures  uint64 `json:"event_write_failures"`
	FrameDecodeFailures uint64 `json:"frame_decode_failures"`
	PatchDecodeFailures uint64 `json:"patch_decode_failures"`
	ServerErrorFrames   uint64 `json:"server_error_frames"`
	TokenMissing        uint64 `json:"token_missing"`
}

func buildReport(
	cfg benchConfig,
	elapsed time.Duration,
	latencies []time.Duration,
	counters *benchCounters,
	errors *benchErrors,
	patchOps *patchOpCounts,
	before runtime.MemStats,
	after runtime.MemStats,
	beforeMetrics runtimeMetricsSnapshot,
	afterMetrics runtimeMetricsSnapshot,
) benchReport {
	eventsTotal := counters.eventsComplete.Load()
	eventsSent := counters.eventsSent.Load()
	patchesTotal := counters.patchesTotal.Load()
	patchFrames := counters.patchFrames.Load()
	eventBytes := counters.eventBytes.Load()
	patchBytes := counters.patchBytes.Load()

	elapsedSeconds := math.Max(0.001, elapsed.Seconds())
	eventsPerSec := float64(eventsTotal) / elapsedSeconds
	eventsPerSecClient := eventsPerSec / float64(cfg.Clients)

	latency := latencyInfo{}
	if len(latencies) > 0 {
		latency = latencyInfo{
			Min: ms(latencies[0]),
			P50: ms(percentile(latencies, 0.50)),
			P95: ms(percentile(latencies, 0.95)),
			P99: ms(percentile(latencies, 0.99)),
			Max: ms(latencies[len(latencies)-1]),
		}
	}

	avgEventBytes := 0.0
	if eventsSent > 0 {
		avgEventBytes = float64(eventBytes) / float64(eventsSent)
	}
	avgPatchBytes := 0.0
	if eventsTotal > 0 {
		avgPatchBytes = float64(patchBytes) / float64(eventsTotal)
	}
	patchesPerEvent := 0.0
	if eventsTotal > 0 {
		patchesPerEvent = float64(patchesTotal) / float64(eventsTotal)
	}

	pauseTotal := time.Duration(after.PauseTotalNs - before.PauseTotalNs)
	pauseAvg := avgPause(after, before)

	report := benchReport{
		Version: "1",
		Run: runInfo{
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			Go:        runtime.Version(),
			OS:        runtime.GOOS,
			Arch:      runtime.GOARCH,
			CPUCount:  runtime.NumCPU(),
			GitCommit: gitCommit(),
		},
		Workload: workloadInfo{
			Profile:        cfg.Profile,
			Clients:        cfg.Clients,
			DurationMS:     cfg.Duration.Milliseconds(),
			RPSPerClient:   cfg.RPS,
			ListSize:       cfg.ListSize,
			PayloadBytes:   cfg.PayloadBytes,
			MaxProcs:       cfg.MaxProcs,
			MemLimitBytes:  cfg.MemLimitBytes,
			EventTimeoutMS: cfg.EventTimeout.Milliseconds(),
		},
		LatencyMS: latency,
		Throughput: throughputInfo{
			EventsTotal:        eventsTotal,
			EventsPerSec:       eventsPerSec,
			EventsPerSecClient: eventsPerSecClient,
		},
		GC: gcInfo{
			AllocMB:       float64(after.TotalAlloc-before.TotalAlloc) / (1024 * 1024),
			HeapLiveMB:    float64(after.HeapAlloc) / (1024 * 1024),
			NumGC:         after.NumGC - before.NumGC,
			PauseTotalMS:  ms(pauseTotal),
			PauseAvgMS:    ms(pauseAvg),
			GCCPUFraction: cpuFraction(afterMetrics, beforeMetrics),
			AllocsObjects: afterMetrics.heapAllocsObjects - beforeMetrics.heapAllocsObjects,
		},
		Protocol: protocolInfo{
			EventBytesTotal: eventBytes,
			PatchBytesTotal: patchBytes,
			PatchFrames:     patchFrames,
			PatchesTotal:    patchesTotal,
			AvgEventBytes:   avgEventBytes,
			AvgPatchBytes:   avgPatchBytes,
			PatchesPerEvent: patchesPerEvent,
			PatchOps:        patchOps.snapshot(),
		},
		Errors: errorInfo{
			TotalErrors:         errors.totalErrors.Load(),
			HandshakeFailures:   errors.handshakeFailures.Load(),
			EventWriteFailures:  errors.eventWriteFailures.Load(),
			FrameDecodeFailures: errors.frameDecodeFailures.Load(),
			PatchDecodeFailures: errors.patchDecodeFailures.Load(),
			ServerErrorFrames:   errors.serverErrorFrames.Load(),
			TokenMissing:        errors.tokenMissing.Load(),
		},
	}

	return report
}

func writeSummary(w io.Writer, report benchReport) {
	fmt.Fprintln(w, "=== Vango Macro Benchmark ===")
	fmt.Fprintf(w, "Profile: %s\n", report.Workload.Profile)
	fmt.Fprintf(w, "Clients: %d\n", report.Workload.Clients)
	fmt.Fprintf(w, "Duration: %s\n", time.Duration(report.Workload.DurationMS)*time.Millisecond)
	fmt.Fprintf(w, "Target per-client rate: %.2f events/s\n", report.Workload.RPSPerClient)
	fmt.Fprintf(w, "List size: %d\n", report.Workload.ListSize)
	fmt.Fprintf(w, "Payload bytes: %d\n", report.Workload.PayloadBytes)
	if report.Workload.MaxProcs > 0 {
		fmt.Fprintf(w, "GOMAXPROCS cap: %d\n", report.Workload.MaxProcs)
	}
	if report.Workload.MemLimitBytes > 0 {
		fmt.Fprintf(w, "GOMEMLIMIT cap: %.2f GiB\n", float64(report.Workload.MemLimitBytes)/float64(gib))
	}
	fmt.Fprintln(w)

	fmt.Fprintf(w, "Total events: %d\n", report.Throughput.EventsTotal)
	fmt.Fprintf(w, "Throughput: %.1f events/s (%.2f per client)\n", report.Throughput.EventsPerSec, report.Throughput.EventsPerSecClient)
	fmt.Fprintf(w, "Errors: %d\n", report.Errors.TotalErrors)
	fmt.Fprintln(w)

	if report.LatencyMS.Max == 0 {
		fmt.Fprintln(w, "No latency samples recorded.")
	} else {
		fmt.Fprintln(w, "RTT (client send -> server -> client receive+decode):")
		fmt.Fprintf(w, "  min: %.2f ms\n", report.LatencyMS.Min)
		fmt.Fprintf(w, "  p50: %.2f ms\n", report.LatencyMS.P50)
		fmt.Fprintf(w, "  p95: %.2f ms\n", report.LatencyMS.P95)
		fmt.Fprintf(w, "  p99: %.2f ms\n", report.LatencyMS.P99)
		fmt.Fprintf(w, "  max: %.2f ms\n", report.LatencyMS.Max)
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "Protocol (avg per event):")
	fmt.Fprintf(w, "  event bytes: %.1f\n", report.Protocol.AvgEventBytes)
	fmt.Fprintf(w, "  patch bytes: %.1f\n", report.Protocol.AvgPatchBytes)
	fmt.Fprintf(w, "  patches/event: %.2f\n", report.Protocol.PatchesPerEvent)
	fmt.Fprintln(w)

	fmt.Fprintln(w, "Go runtime / GC (process-wide):")
	fmt.Fprintf(w, "  alloc:     %.2f MB\n", report.GC.AllocMB)
	fmt.Fprintf(w, "  heap_live: %.2f MB\n", report.GC.HeapLiveMB)
	fmt.Fprintf(w, "  num_gc:    %d\n", report.GC.NumGC)
	fmt.Fprintf(w, "  gc_pause:  %.2f ms (total)\n", report.GC.PauseTotalMS)
	fmt.Fprintf(w, "  gc_pause:  %.2f ms (avg)\n", report.GC.PauseAvgMS)
	fmt.Fprintf(w, "  gc_cpu:    %.2f%%\n", report.GC.GCCPUFraction*100)
}

func writeJSON(path string, report benchReport) error {
	var out io.Writer
	if path == "-" {
		out = os.Stdout
	} else {
		file, err := os.Create(path)
		if err != nil {
			return err
		}
		defer file.Close()
		out = file
	}

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

func gitCommit() string {
	if val := strings.TrimSpace(os.Getenv("VANGO_GIT_COMMIT")); val != "" {
		return val
	}
	if val := strings.TrimSpace(os.Getenv("GIT_COMMIT")); val != "" {
		return val
	}
	cmd := exec.Command("git", "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// LoadApp is a small server-driven page for exercising render+diff+patch costs.
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
	return Div(
		Input(
			Type("text"),
			OnInput(func(value string) {
				a.echo.Set(value)
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
		Div(ID("echo"), Text(a.echo.Get())),
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
