package vango

import "sync/atomic"

// globalIDCounter is the source of unique IDs for all reactive primitives.
// Using atomic operations ensures thread-safe ID generation without locks.
var globalIDCounter uint64

// nextID returns the next unique ID for a reactive primitive.
// IDs are monotonically increasing and never reused.
func nextID() uint64 {
	return atomic.AddUint64(&globalIDCounter, 1)
}
