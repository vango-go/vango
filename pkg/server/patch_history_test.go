package server

import (
	"sync"
	"testing"
)

func TestPatchHistory_Add(t *testing.T) {
	h := NewPatchHistory(5)

	// Add first entry
	h.Add(1, []byte("frame1"))
	if h.Count() != 1 {
		t.Errorf("expected count 1, got %d", h.Count())
	}
	if h.MinSeq() != 1 {
		t.Errorf("expected minSeq 1, got %d", h.MinSeq())
	}
	if h.MaxSeq() != 1 {
		t.Errorf("expected maxSeq 1, got %d", h.MaxSeq())
	}

	// Add more entries
	h.Add(2, []byte("frame2"))
	h.Add(3, []byte("frame3"))

	if h.Count() != 3 {
		t.Errorf("expected count 3, got %d", h.Count())
	}
	if h.MinSeq() != 1 {
		t.Errorf("expected minSeq 1, got %d", h.MinSeq())
	}
	if h.MaxSeq() != 3 {
		t.Errorf("expected maxSeq 3, got %d", h.MaxSeq())
	}
}

func TestPatchHistory_GetFrames(t *testing.T) {
	h := NewPatchHistory(10)

	// Add frames 1-5
	for i := uint64(1); i <= 5; i++ {
		h.Add(i, []byte{byte(i)})
	}

	// Get frames (0, 5] - should return all 5 frames
	frames := h.GetFrames(0, 5)
	if len(frames) != 5 {
		t.Fatalf("expected 5 frames, got %d", len(frames))
	}
	for i, frame := range frames {
		if len(frame) != 1 || frame[0] != byte(i+1) {
			t.Errorf("frame %d: expected [%d], got %v", i, i+1, frame)
		}
	}

	// Get frames (2, 4] - should return frames 3 and 4
	frames = h.GetFrames(2, 4)
	if len(frames) != 2 {
		t.Fatalf("expected 2 frames, got %d", len(frames))
	}
	if frames[0][0] != 3 || frames[1][0] != 4 {
		t.Errorf("expected frames [3, 4], got %v", frames)
	}

	// Get frames that don't exist (should return nil)
	frames = h.GetFrames(10, 15)
	if frames != nil {
		t.Errorf("expected nil for out of range, got %v", frames)
	}
}

func TestPatchHistory_CircularOverwrite(t *testing.T) {
	h := NewPatchHistory(3) // Small buffer

	// Add 5 entries to a buffer of size 3
	for i := uint64(1); i <= 5; i++ {
		h.Add(i, []byte{byte(i)})
	}

	// Count should be capped at capacity
	if h.Count() != 3 {
		t.Errorf("expected count 3, got %d", h.Count())
	}

	// MaxSeq should be 5
	if h.MaxSeq() != 5 {
		t.Errorf("expected maxSeq 5, got %d", h.MaxSeq())
	}

	// MinSeq should be 3 (oldest in buffer)
	if h.MinSeq() != 3 {
		t.Errorf("expected minSeq 3, got %d", h.MinSeq())
	}

	// Should be able to get frames (2, 5]
	frames := h.GetFrames(2, 5)
	if len(frames) != 3 {
		t.Fatalf("expected 3 frames, got %d", len(frames))
	}

	// Should NOT be able to get frames (0, 5] (1 and 2 are gone)
	frames = h.GetFrames(0, 5)
	if frames != nil {
		t.Errorf("expected nil for request including overwritten frames, got %v", frames)
	}
}

func TestPatchHistory_CanRecover(t *testing.T) {
	h := NewPatchHistory(5)

	// Empty buffer
	if h.CanRecover(0) {
		t.Error("expected CanRecover(0) = false for empty buffer")
	}

	// Add frames 1-5
	for i := uint64(1); i <= 5; i++ {
		h.Add(i, []byte{byte(i)})
	}

	// Can recover from 0 (need 1-5)
	if !h.CanRecover(0) {
		t.Error("expected CanRecover(0) = true")
	}

	// Can recover from 3 (need 4-5)
	if !h.CanRecover(3) {
		t.Error("expected CanRecover(3) = true")
	}

	// Can recover from 4 (need 5)
	if !h.CanRecover(4) {
		t.Error("expected CanRecover(4) = true")
	}

	// Cannot recover from 5 (nothing needed)
	if h.CanRecover(5) {
		t.Error("expected CanRecover(5) = false (already up to date)")
	}

	// Cannot recover from 10 (past maxSeq)
	if h.CanRecover(10) {
		t.Error("expected CanRecover(10) = false (past maxSeq)")
	}
}

func TestPatchHistory_GarbageCollect(t *testing.T) {
	h := NewPatchHistory(5)

	// Add frames 1-5
	for i := uint64(1); i <= 5; i++ {
		h.Add(i, []byte{byte(i)})
	}

	// GC up to seq 3
	h.GarbageCollect(3)

	// Buffer should still have all 5 entries (GC is lazy)
	if h.Count() != 5 {
		t.Errorf("expected count 5 after GC, got %d", h.Count())
	}

	// But CanRecover should still work since entries are still there
	if !h.CanRecover(3) {
		t.Error("expected CanRecover(3) = true after GC")
	}
}

func TestPatchHistory_Clear(t *testing.T) {
	h := NewPatchHistory(5)

	// Add some entries
	for i := uint64(1); i <= 3; i++ {
		h.Add(i, []byte{byte(i)})
	}

	// Clear
	h.Clear()

	if h.Count() != 0 {
		t.Errorf("expected count 0 after clear, got %d", h.Count())
	}
	if h.MinSeq() != 0 {
		t.Errorf("expected minSeq 0 after clear, got %d", h.MinSeq())
	}
	if h.MaxSeq() != 0 {
		t.Errorf("expected maxSeq 0 after clear, got %d", h.MaxSeq())
	}
}

func TestPatchHistory_MemoryUsageTracksFrames(t *testing.T) {
	h := NewPatchHistory(2)

	before := h.MemoryUsage()
	h.Add(1, []byte("frame-1"))
	h.Add(2, []byte("frame-2-longer"))
	afterAdd := h.MemoryUsage()
	if afterAdd <= before {
		t.Errorf("MemoryUsage should increase after adding frames (before=%d after=%d)", before, afterAdd)
	}

	h.Clear()
	afterClear := h.MemoryUsage()
	if afterClear >= afterAdd {
		t.Errorf("MemoryUsage should decrease after clear (before=%d after=%d)", afterAdd, afterClear)
	}
}

func TestPatchHistory_Concurrent(t *testing.T) {
	h := NewPatchHistory(100)
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(base int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				seq := uint64(base*10 + j + 1)
				h.Add(seq, []byte{byte(seq)})
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				_ = h.Count()
				_ = h.MinSeq()
				_ = h.MaxSeq()
				_ = h.CanRecover(uint64(j))
			}
		}()
	}

	wg.Wait()

	// Should have 100 entries (all writes completed, buffer at capacity)
	if h.Count() != 100 {
		t.Errorf("expected count 100 after concurrent writes, got %d", h.Count())
	}
}

func TestPatchHistory_FrameCopyIsolation(t *testing.T) {
	h := NewPatchHistory(5)

	// Create a frame and add it
	frame := []byte{1, 2, 3}
	h.Add(1, frame)

	// Modify the original frame
	frame[0] = 99

	// Get the frame back - should have original value
	frames := h.GetFrames(0, 1)
	if len(frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(frames))
	}
	if frames[0][0] != 1 {
		t.Errorf("expected frame[0] = 1, got %d (frame not copied properly)", frames[0][0])
	}
}

func TestPatchHistory_EmptyRange(t *testing.T) {
	h := NewPatchHistory(5)

	// Add frames 1-5
	for i := uint64(1); i <= 5; i++ {
		h.Add(i, []byte{byte(i)})
	}

	// Empty range (3, 3] should return empty slice
	frames := h.GetFrames(3, 3)
	if frames != nil && len(frames) != 0 {
		t.Errorf("expected nil or empty for empty range, got %v", frames)
	}
}

func TestPatchHistory_DefaultCapacity(t *testing.T) {
	// Test zero capacity defaults to 100
	h := NewPatchHistory(0)
	if h.capacity != 100 {
		t.Errorf("expected default capacity 100, got %d", h.capacity)
	}

	// Test negative capacity defaults to 100
	h = NewPatchHistory(-5)
	if h.capacity != 100 {
		t.Errorf("expected default capacity 100 for negative input, got %d", h.capacity)
	}
}
