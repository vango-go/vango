// Package vango provides the reactive core for the Vango framework.
//
// The reactive system provides fine-grained reactivity similar to SolidJS,
// where dependencies are tracked automatically at runtime. Reading a signal
// during component render automatically subscribes the component to that
// signal's changes.
//
// # Core Types
//
// Signal[T] is a reactive value container:
//
//	count := NewSignal(0)
//	value := count.Get()  // Read (subscribes current listener)
//	count.Set(5)          // Write (notifies subscribers)
//	count.Update(func(n int) int { return n + 1 })
//
// Memo[T] is a cached derived computation:
//
//	doubled := NewMemo(func() int { return count.Get() * 2 })
//	value := doubled.Get()  // Recomputes only if dependencies changed
//
// Effect runs side effects when dependencies change:
//
//	Effect(func() Cleanup {
//	    fmt.Println("Count is:", count.Get())
//	    return func() { /* cleanup */ }
//	})
//
// # Batching
//
// Multiple signal updates can be batched to trigger a single notification:
//
//	Batch(func() {
//	    a.Set(1)
//	    b.Set(2)
//	    c.Set(3)
//	})  // Single notification after all updates
//
// # Thread Safety
//
// All reactive primitives are thread-safe and can be accessed from multiple
// goroutines. The tracking context is per-goroutine, so spawning goroutines
// requires explicit context propagation via WithOwner.
package vango
