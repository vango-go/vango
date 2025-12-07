package server

import (
	"testing"
	"time"
)

func TestDefaultMemoryMonitorConfig(t *testing.T) {
	config := DefaultMemoryMonitorConfig()

	if config.SoftLimit <= 0 {
		t.Error("SoftLimit should be positive")
	}
	if config.HardLimit <= 0 {
		t.Error("HardLimit should be positive")
	}
	if config.SoftLimit >= config.HardLimit {
		t.Error("SoftLimit should be less than HardLimit")
	}
	if config.CheckInterval <= 0 {
		t.Error("CheckInterval should be positive")
	}
	if config.GCCooldown <= 0 {
		t.Error("GCCooldown should be positive")
	}
}

func TestNewMemoryMonitor(t *testing.T) {
	mm := NewMemoryMonitor(nil)
	if mm == nil {
		t.Fatal("NewMemoryMonitor should not return nil")
	}
	if mm.done == nil {
		t.Error("done channel should be initialized")
	}
}

func TestNewMemoryMonitorWithConfig(t *testing.T) {
	config := &MemoryMonitorConfig{
		SoftLimit:     100 * 1024 * 1024,
		HardLimit:     200 * 1024 * 1024,
		CheckInterval: 5 * time.Second,
		GCCooldown:    15 * time.Second,
	}

	mm := NewMemoryMonitor(config)
	if mm.softLimit != config.SoftLimit {
		t.Errorf("softLimit = %d, want %d", mm.softLimit, config.SoftLimit)
	}
	if mm.hardLimit != config.HardLimit {
		t.Errorf("hardLimit = %d, want %d", mm.hardLimit, config.HardLimit)
	}
}

func TestMemoryMonitorPauseResume(t *testing.T) {
	mm := NewMemoryMonitor(nil)

	if mm.paused.Load() {
		t.Error("Monitor should not be paused initially")
	}

	mm.Pause()
	if !mm.paused.Load() {
		t.Error("Monitor should be paused after Pause()")
	}

	mm.Resume()
	if mm.paused.Load() {
		t.Error("Monitor should not be paused after Resume()")
	}
}

func TestMemoryMonitorCallbacks(t *testing.T) {
	mm := NewMemoryMonitor(nil)

	softCalled := false
	hardCalled := false

	mm.SetOnSoftLimit(func(current, limit int64) {
		softCalled = true
	})

	mm.SetOnHardLimit(func(current, limit int64) {
		hardCalled = true
	})

	// Set very low limits to trigger callbacks
	mm.softLimit = 1
	mm.hardLimit = 1

	mm.ForceCheck()

	// At least one should be called since we're using memory
	if !hardCalled && !softCalled {
		t.Error("At least one callback should have been called")
	}
}

func TestCurrentMemoryUsage(t *testing.T) {
	usage := CurrentMemoryUsage()
	if usage <= 0 {
		t.Error("CurrentMemoryUsage should return positive value")
	}
}

func TestTotalSystemMemory(t *testing.T) {
	total := TotalSystemMemory()
	if total <= 0 {
		t.Error("TotalSystemMemory should return positive value")
	}
}

func TestGetMemoryStats(t *testing.T) {
	stats := GetMemoryStats()
	if stats == nil {
		t.Fatal("GetMemoryStats should not return nil")
	}
	if stats.HeapAlloc <= 0 {
		t.Error("HeapAlloc should be positive")
	}
	if stats.HeapSys <= 0 {
		t.Error("HeapSys should be positive")
	}
	if stats.NumGC == 0 {
		// This might be 0 if GC hasn't run, which is fine
	}
}

func TestByteSizeString(t *testing.T) {
	tests := []struct {
		size     ByteSize
		expected string
	}{
		{0, "0B"},
		{500, "500B"},
		{KB, "1KB"},
		{2 * KB, "2KB"},
		{MB, "1MB"},
		{5 * MB, "5MB"},
		{GB, "1GB"},
		{TB, "1TB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.size.String()
			if got != tt.expected {
				t.Errorf("ByteSize(%d).String() = %s, want %s", tt.size, got, tt.expected)
			}
		})
	}
}

func TestByteSizeConstants(t *testing.T) {
	if KB != 1024 {
		t.Errorf("KB = %d, want 1024", KB)
	}
	if MB != 1024*1024 {
		t.Errorf("MB = %d, want %d", MB, 1024*1024)
	}
	if GB != 1024*1024*1024 {
		t.Errorf("GB = %d, want %d", GB, 1024*1024*1024)
	}
	if TB != 1024*1024*1024*1024 {
		t.Errorf("TB = %d, want %d", TB, 1024*1024*1024*1024)
	}
}

func TestMemoryPressureLevel(t *testing.T) {
	tests := []struct {
		level    MemoryPressureLevel
		expected string
	}{
		{MemoryPressureNone, "none"},
		{MemoryPressureLow, "low"},
		{MemoryPressureHigh, "high"},
		{MemoryPressureCritical, "critical"},
		{MemoryPressureLevel(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.level.String() != tt.expected {
				t.Errorf("String() = %s, want %s", tt.level.String(), tt.expected)
			}
		})
	}
}

func TestGetMemoryPressureLevel(t *testing.T) {
	// Use very high limits so we're below soft limit
	level := GetMemoryPressureLevel(1024*1024*1024*1024, 2*1024*1024*1024*1024)
	if level != MemoryPressureNone {
		t.Errorf("Expected MemoryPressureNone, got %v", level)
	}

	// Use very low limits so we're above hard limit
	level = GetMemoryPressureLevel(1, 2)
	if level != MemoryPressureCritical {
		t.Errorf("Expected MemoryPressureCritical, got %v", level)
	}
}

func TestEstimateStringMemory(t *testing.T) {
	s := "hello"
	estimate := EstimateStringMemory(s)

	// Should be at least the string length plus header
	if estimate < int64(len(s)) {
		t.Errorf("Estimate %d is less than string length %d", estimate, len(s))
	}
	// Should include header (16 bytes on 64-bit)
	if estimate < 16 {
		t.Error("Estimate should include string header")
	}
}

func TestEstimateSliceMemory(t *testing.T) {
	// Slice of 10 int64s (8 bytes each)
	estimate := EstimateSliceMemory(10, 8)

	// Should be at least 10 * 8 = 80 bytes for data
	if estimate < 80 {
		t.Errorf("Estimate %d is less than expected data size", estimate)
	}
	// Should include header (24 bytes on 64-bit)
	if estimate < 24 {
		t.Error("Estimate should include slice header")
	}
}

func TestEstimateMapMemory(t *testing.T) {
	// Map with 10 entries, 16-byte keys, 8-byte values
	estimate := EstimateMapMemory(10, 16, 8)

	// Should be positive and reasonable
	if estimate <= 0 {
		t.Error("Estimate should be positive")
	}
	// Should be at least entry size * count
	minSize := int64(10 * (16 + 8))
	if estimate < minSize {
		t.Errorf("Estimate %d is less than minimum expected %d", estimate, minSize)
	}
}

func TestMemoryMonitorStartStop(t *testing.T) {
	mm := NewMemoryMonitor(&MemoryMonitorConfig{
		SoftLimit:     1024 * 1024 * 1024,
		HardLimit:     2 * 1024 * 1024 * 1024,
		CheckInterval: 100 * time.Millisecond,
		GCCooldown:    time.Second,
	})

	mm.Start()
	time.Sleep(50 * time.Millisecond) // Let it run briefly
	mm.Stop()

	// Should not panic and should stop cleanly
}

func TestIntToString(t *testing.T) {
	tests := []struct {
		n        int64
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{123, "123"},
		{-5, "-5"},
		{-100, "-100"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := intToString(tt.n)
			if got != tt.expected {
				t.Errorf("intToString(%d) = %s, want %s", tt.n, got, tt.expected)
			}
		})
	}
}
