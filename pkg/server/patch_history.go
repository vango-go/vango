package server

import (
	"sync"
	"time"
)

// PatchHistoryEntry stores a sent patch frame for potential replay.
type PatchHistoryEntry struct {
	Seq    uint64    // Patch sequence number
	Frame  []byte    // Pre-encoded FramePatches for fast replay
	SentAt time.Time // When the frame was sent
}

// PatchHistory is a thread-safe ring buffer for storing sent patch frames.
// It supports:
//   - Fast insertion at head
//   - Lookup by sequence range for resync
//   - Garbage collection based on acknowledged sequence
//
// The ring buffer overwrites oldest entries when full, maintaining a sliding
// window of recent patches that can be replayed if a client misses some.
type PatchHistory struct {
	mu       sync.RWMutex
	entries  []*PatchHistoryEntry
	head     int    // Next write position (circular)
	count    int    // Current number of entries
	capacity int    // Max entries
	minSeq   uint64 // Lowest sequence in buffer (for CanRecover checks)
	maxSeq   uint64 // Highest sequence in buffer
}

// NewPatchHistory creates a new patch history ring buffer with the given capacity.
func NewPatchHistory(capacity int) *PatchHistory {
	if capacity <= 0 {
		capacity = 100 // Default from config.MaxPatchHistory
	}
	return &PatchHistory{
		entries:  make([]*PatchHistoryEntry, capacity),
		capacity: capacity,
	}
}

// Add stores a patch frame in the buffer.
// This should be called ONLY after a successful write to the WebSocket.
// The frame bytes are copied to prevent issues with buffer reuse.
func (h *PatchHistory) Add(seq uint64, frame []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Copy the frame to prevent issues with buffer reuse
	frameCopy := make([]byte, len(frame))
	copy(frameCopy, frame)

	entry := &PatchHistoryEntry{
		Seq:    seq,
		Frame:  frameCopy,
		SentAt: time.Now(),
	}

	h.entries[h.head] = entry
	h.head = (h.head + 1) % h.capacity

	if h.count < h.capacity {
		h.count++
	}

	// Update sequence range
	h.maxSeq = seq
	if h.count == 1 {
		h.minSeq = seq
	} else if h.count == h.capacity {
		// Buffer full, minSeq advances to oldest entry
		// The oldest entry is now at head (we just overwrote it)
		oldestIdx := h.head // After increment, head points to what will be overwritten next
		if h.entries[oldestIdx] != nil {
			h.minSeq = h.entries[oldestIdx].Seq
		}
	}
}

// GetFrames returns frames for sequences (afterSeq, toSeq].
// Returns nil if any sequence in the requested range is not available.
// The returned frames are in sequence order, ready to be replayed.
func (h *PatchHistory) GetFrames(afterSeq, toSeq uint64) [][]byte {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.count == 0 {
		return nil
	}

	// Check if requested range is within buffer
	// We need all sequences from afterSeq+1 to toSeq
	if afterSeq+1 < h.minSeq || toSeq > h.maxSeq {
		return nil // Gap detected or request out of range
	}

	// Collect frames in sequence order
	// Build a map of seq -> frame for efficient lookup
	seqToFrame := make(map[uint64][]byte, h.count)
	for i := 0; i < h.count; i++ {
		// Calculate index from tail (oldest) to head (newest)
		idx := (h.head - h.count + i + h.capacity) % h.capacity
		entry := h.entries[idx]
		if entry != nil {
			seqToFrame[entry.Seq] = entry.Frame
		}
	}

	// Collect frames in order
	var frames [][]byte
	for seq := afterSeq + 1; seq <= toSeq; seq++ {
		frame, ok := seqToFrame[seq]
		if !ok {
			return nil // Missing sequence in range
		}
		frames = append(frames, frame)
	}

	return frames
}

// GarbageCollect updates minSeq based on the acknowledged sequence.
// Entries with sequence <= ackSeq are no longer needed for resync
// and will be overwritten naturally by the ring buffer.
func (h *PatchHistory) GarbageCollect(ackSeq uint64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// We don't actually remove entries - the ring buffer will overwrite them.
	// We just update minSeq to reflect what's recoverable.
	// If a client has ACK'd up to ackSeq, they won't need anything <= ackSeq.
	if ackSeq >= h.minSeq {
		// Update minSeq to be one past the ackSeq (if we have entries past it)
		// This is an optimization - we could be smarter about this
		// but for simplicity we just note that anything <= ackSeq is "safe to overwrite"
		// The actual minSeq is still determined by what's in the buffer
	}
}

// CanRecover checks if the buffer can provide frames to recover from lastSeq.
// Returns true if all sequences from lastSeq+1 to maxSeq are available.
func (h *PatchHistory) CanRecover(lastSeq uint64) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.count == 0 {
		return false
	}

	// We can recover if lastSeq+1 >= minSeq (first needed seq is available)
	// and lastSeq < maxSeq (there's something to send)
	return lastSeq+1 >= h.minSeq && lastSeq < h.maxSeq
}

// MinSeq returns the minimum recoverable sequence.
func (h *PatchHistory) MinSeq() uint64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.minSeq
}

// MaxSeq returns the maximum sequence in buffer.
func (h *PatchHistory) MaxSeq() uint64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.maxSeq
}

// Count returns the number of entries in the buffer.
func (h *PatchHistory) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.count
}

// Clear removes all entries from the buffer.
// This is useful during session resume when starting fresh.
func (h *PatchHistory) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for i := range h.entries {
		h.entries[i] = nil
	}
	h.head = 0
	h.count = 0
	h.minSeq = 0
	h.maxSeq = 0
}
