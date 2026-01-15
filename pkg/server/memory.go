package server

import (
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// MemoryMonitor monitors system memory and triggers actions when thresholds are exceeded.
type MemoryMonitor struct {
	// Configuration
	softLimit    int64 // Soft limit - start evicting sessions
	hardLimit    int64 // Hard limit - aggressive eviction
	checkInterval time.Duration

	// State
	lastCheck    time.Time
	lastGC       time.Time
	gcCooldown   time.Duration
	paused       atomic.Bool

	// Callbacks
	onSoftLimit func(current, limit int64)
	onHardLimit func(current, limit int64)

	// Control
	done   chan struct{}
	ticker *time.Ticker
	mu     sync.Mutex
}

// MemoryMonitorConfig configures the memory monitor.
type MemoryMonitorConfig struct {
	SoftLimit     int64         // Bytes at which to start evicting (default: 80% of system memory)
	HardLimit     int64         // Bytes at which to aggressively evict (default: 90% of system memory)
	CheckInterval time.Duration // How often to check memory (default: 10s)
	GCCooldown    time.Duration // Minimum time between forced GCs (default: 30s)
}

// DefaultMemoryMonitorConfig returns sensible defaults.
func DefaultMemoryMonitorConfig() *MemoryMonitorConfig {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Use system memory as baseline, or estimate 4GB if unavailable
	systemMem := int64(memStats.Sys)
	if systemMem == 0 {
		systemMem = 4 * 1024 * 1024 * 1024 // 4GB default
	}

	return &MemoryMonitorConfig{
		SoftLimit:     systemMem * 80 / 100, // 80%
		HardLimit:     systemMem * 90 / 100, // 90%
		CheckInterval: 10 * time.Second,
		GCCooldown:    30 * time.Second,
	}
}

// NewMemoryMonitor creates a new memory monitor.
func NewMemoryMonitor(config *MemoryMonitorConfig) *MemoryMonitor {
	if config == nil {
		config = DefaultMemoryMonitorConfig()
	}

	return &MemoryMonitor{
		softLimit:     config.SoftLimit,
		hardLimit:     config.HardLimit,
		checkInterval: config.CheckInterval,
		gcCooldown:    config.GCCooldown,
		done:          make(chan struct{}),
	}
}

// SetOnSoftLimit sets the callback for soft limit breach.
func (m *MemoryMonitor) SetOnSoftLimit(fn func(current, limit int64)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onSoftLimit = fn
}

// SetOnHardLimit sets the callback for hard limit breach.
func (m *MemoryMonitor) SetOnHardLimit(fn func(current, limit int64)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onHardLimit = fn
}

// Start begins monitoring memory.
func (m *MemoryMonitor) Start() {
	m.ticker = time.NewTicker(m.checkInterval)

	go func() {
		for {
			select {
			case <-m.ticker.C:
				if !m.paused.Load() {
					m.check()
				}
			case <-m.done:
				return
			}
		}
	}()
}

// Stop stops the memory monitor.
func (m *MemoryMonitor) Stop() {
	close(m.done)
	if m.ticker != nil {
		m.ticker.Stop()
	}
}

// Pause temporarily pauses memory checking.
func (m *MemoryMonitor) Pause() {
	m.paused.Store(true)
}

// Resume resumes memory checking.
func (m *MemoryMonitor) Resume() {
	m.paused.Store(false)
}

// check performs a memory check.
func (m *MemoryMonitor) check() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.lastCheck = time.Now()
	current := CurrentMemoryUsage()

	// Check hard limit first
	if current >= m.hardLimit {
		if m.onHardLimit != nil {
			m.onHardLimit(current, m.hardLimit)
		}
		m.maybeGC()
		return
	}

	// Check soft limit
	if current >= m.softLimit {
		if m.onSoftLimit != nil {
			m.onSoftLimit(current, m.softLimit)
		}
	}
}

// maybeGC triggers GC if cooldown has passed.
func (m *MemoryMonitor) maybeGC() {
	if time.Since(m.lastGC) >= m.gcCooldown {
		runtime.GC()
		m.lastGC = time.Now()
	}
}

// ForceCheck performs an immediate memory check.
func (m *MemoryMonitor) ForceCheck() {
	m.check()
}

// CurrentMemoryUsage returns the current heap memory usage in bytes.
func CurrentMemoryUsage() int64 {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return int64(memStats.HeapAlloc)
}

// TotalSystemMemory returns the total system memory seen by the Go runtime.
func TotalSystemMemory() int64 {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return int64(memStats.Sys)
}

// MemoryStats returns detailed memory statistics.
type MemoryStats struct {
	HeapAlloc    int64 // Bytes allocated on heap
	HeapSys      int64 // Bytes obtained from OS for heap
	HeapIdle     int64 // Bytes in idle spans
	HeapInuse    int64 // Bytes in non-idle spans
	HeapReleased int64 // Bytes released to OS
	StackInuse   int64 // Bytes used by stack
	NumGC        uint32 // Number of completed GC cycles
	LastGC       time.Time // Time of last GC
	GCPauseTotal time.Duration // Total GC pause time
}

// GetMemoryStats returns current memory statistics.
func GetMemoryStats() *MemoryStats {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	var lastGC time.Time
	if memStats.LastGC > 0 {
		lastGC = time.Unix(0, int64(memStats.LastGC))
	}

	return &MemoryStats{
		HeapAlloc:    int64(memStats.HeapAlloc),
		HeapSys:      int64(memStats.HeapSys),
		HeapIdle:     int64(memStats.HeapIdle),
		HeapInuse:    int64(memStats.HeapInuse),
		HeapReleased: int64(memStats.HeapReleased),
		StackInuse:   int64(memStats.StackSys),
		NumGC:        memStats.NumGC,
		LastGC:       lastGC,
		GCPauseTotal: time.Duration(memStats.PauseTotalNs),
	}
}

// ByteSize represents a byte size with human-readable formatting.
type ByteSize int64

const (
	_           = iota
	KB ByteSize = 1 << (10 * iota)
	MB
	GB
	TB
)

// String returns a human-readable representation of the byte size.
func (b ByteSize) String() string {
	switch {
	case b >= TB:
		return formatByteSize(float64(b)/float64(TB), "TB")
	case b >= GB:
		return formatByteSize(float64(b)/float64(GB), "GB")
	case b >= MB:
		return formatByteSize(float64(b)/float64(MB), "MB")
	case b >= KB:
		return formatByteSize(float64(b)/float64(KB), "KB")
	default:
		return formatByteSize(float64(b), "B")
	}
}

func formatByteSize(value float64, unit string) string {
	intPart := int64(value)
	fracPart := int64((value - float64(intPart)) * 10)
	if fracPart == 0 {
		return intToString(intPart) + unit
	}
	return intToString(intPart) + "." + intToString(fracPart) + unit
}

func intToString(n int64) string {
	if n == 0 {
		return "0"
	}
	negative := n < 0
	if negative {
		n = -n
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if negative {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}

// MemoryPressureLevel indicates the current memory pressure.
type MemoryPressureLevel int

const (
	MemoryPressureNone     MemoryPressureLevel = iota // Below soft limit
	MemoryPressureLow                                  // Between soft and hard limit
	MemoryPressureHigh                                 // Above hard limit
	MemoryPressureCritical                             // Near OOM
)

// String returns the string representation of the pressure level.
func (l MemoryPressureLevel) String() string {
	switch l {
	case MemoryPressureNone:
		return "none"
	case MemoryPressureLow:
		return "low"
	case MemoryPressureHigh:
		return "high"
	case MemoryPressureCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// GetMemoryPressureLevel returns the current memory pressure level.
func GetMemoryPressureLevel(softLimit, hardLimit int64) MemoryPressureLevel {
	current := CurrentMemoryUsage()

	// Critical is 95% of hard limit
	criticalLimit := hardLimit * 95 / 100

	switch {
	case current >= criticalLimit:
		return MemoryPressureCritical
	case current >= hardLimit:
		return MemoryPressureHigh
	case current >= softLimit:
		return MemoryPressureLow
	default:
		return MemoryPressureNone
	}
}

// EstimateStringMemory estimates memory usage of a string.
func EstimateStringMemory(s string) int64 {
	// String header (16 bytes on 64-bit) + actual string data
	return 16 + int64(len(s))
}

// EstimateSliceMemory estimates memory usage of a slice.
func EstimateSliceMemory(length, elementSize int) int64 {
	// Slice header (24 bytes on 64-bit) + elements
	return 24 + int64(length*elementSize)
}

// EstimateMapMemory estimates memory usage of a map.
func EstimateMapMemory(length, keySize, valueSize int) int64 {
	// Map has significant overhead - roughly 8 bytes per bucket + entries
	// This is a rough estimate
	buckets := (length / 8) + 1
	bucketOverhead := int64(buckets * 8)
	entrySize := int64(keySize + valueSize + 8) // 8 bytes for pointer overhead
	return 48 + bucketOverhead + int64(length)*entrySize
}

const estimateAnyMemoryMaxDepth = 4

// EstimateAnyMemory approximates the memory usage of a value.
// It is depth-limited to avoid runaway recursion on complex graphs.
func EstimateAnyMemory(value any) int64 {
	return estimateAnyMemoryValue(reflect.ValueOf(value), 0)
}

func estimateAnyMemoryValue(v reflect.Value, depth int) int64 {
	if !v.IsValid() {
		return 0
	}
	if depth >= estimateAnyMemoryMaxDepth {
		return 16
	}

	switch v.Kind() {
	case reflect.String:
		return EstimateStringMemory(v.String())
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return 8
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return 8
	case reflect.Float32, reflect.Float64:
		return 8
	case reflect.Interface, reflect.Ptr:
		if v.IsNil() {
			return 0
		}
		return 8 + estimateAnyMemoryValue(v.Elem(), depth+1)
	case reflect.Slice:
		if v.IsNil() {
			return 0
		}
		size := EstimateSliceMemory(v.Len(), 16)
		for i := 0; i < v.Len(); i++ {
			size += estimateAnyMemoryValue(v.Index(i), depth+1)
		}
		return size
	case reflect.Array:
		size := int64(16)
		for i := 0; i < v.Len(); i++ {
			size += estimateAnyMemoryValue(v.Index(i), depth+1)
		}
		return size
	case reflect.Map:
		if v.IsNil() {
			return 0
		}
		size := EstimateMapMemory(v.Len(), 16, 16)
		iter := v.MapRange()
		for iter.Next() {
			size += estimateAnyMemoryValue(iter.Key(), depth+1)
			size += estimateAnyMemoryValue(iter.Value(), depth+1)
		}
		return size
	case reflect.Struct:
		size := int64(16)
		for i := 0; i < v.NumField(); i++ {
			size += estimateAnyMemoryValue(v.Field(i), depth+1)
		}
		return size
	default:
		return 16
	}
}
