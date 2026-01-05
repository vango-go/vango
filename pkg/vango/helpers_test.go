package vango

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestInterval(t *testing.T) {
	mockC := newMockCtx()
	setCurrentCtx(mockC)
	defer setCurrentCtx(nil)

	var count atomic.Int32
	cleanup := Interval(10*time.Millisecond, func() {
		count.Add(1)
	})

	// Wait for a few ticks
	time.Sleep(35 * time.Millisecond)

	// Should have at least 2 ticks (maybe 3)
	c := count.Load()
	if c < 2 {
		t.Errorf("Expected at least 2 ticks, got %d", c)
	}

	// Cleanup should stop ticks
	cleanup()
	countAfterCleanup := count.Load()
	time.Sleep(20 * time.Millisecond)

	if count.Load() != countAfterCleanup {
		t.Error("Interval continued after cleanup")
	}
}

func TestIntervalImmediate(t *testing.T) {
	mockC := newMockCtx()
	setCurrentCtx(mockC)
	defer setCurrentCtx(nil)

	var count atomic.Int32
	cleanup := Interval(50*time.Millisecond, func() {
		count.Add(1)
	}, IntervalImmediate())

	// Should have immediate tick
	time.Sleep(10 * time.Millisecond)
	if count.Load() < 1 {
		t.Error("IntervalImmediate should fire immediately")
	}

	cleanup()
}

func TestIntervalTxName(t *testing.T) {
	mockC := newMockCtx()
	setCurrentCtx(mockC)
	defer setCurrentCtx(nil)

	// Just verify it doesn't panic
	cleanup := Interval(10*time.Millisecond, func() {}, IntervalTxName("test"))
	time.Sleep(5 * time.Millisecond)
	cleanup()
}

func TestIntervalPanicWithoutContext(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when calling Interval without context")
		}
	}()

	// Clear context
	setCurrentCtx(nil)

	Interval(time.Second, func() {})
}

// mockStream implements Stream for testing Subscribe.
type mockStream[T any] struct {
	handlers []func(T)
	mu       sync.Mutex
}

func (m *mockStream[T]) Subscribe(handler func(T)) func() {
	m.mu.Lock()
	m.handlers = append(m.handlers, handler)
	idx := len(m.handlers) - 1
	m.mu.Unlock()

	return func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		// Remove handler
		if idx < len(m.handlers) {
			m.handlers = append(m.handlers[:idx], m.handlers[idx+1:]...)
		}
	}
}

func (m *mockStream[T]) emit(value T) {
	m.mu.Lock()
	handlers := make([]func(T), len(m.handlers))
	copy(handlers, m.handlers)
	m.mu.Unlock()

	for _, h := range handlers {
		h(value)
	}
}

func TestSubscribe(t *testing.T) {
	mockC := newMockCtx()
	setCurrentCtx(mockC)
	defer setCurrentCtx(nil)

	stream := &mockStream[string]{}
	var received []string
	var mu sync.Mutex

	cleanup := Subscribe(stream, func(msg string) {
		mu.Lock()
		received = append(received, msg)
		mu.Unlock()
	})

	// Emit messages
	stream.emit("hello")
	stream.emit("world")

	time.Sleep(5 * time.Millisecond)

	mu.Lock()
	if len(received) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(received))
	}
	if len(received) >= 2 && (received[0] != "hello" || received[1] != "world") {
		t.Errorf("Messages = %v, want [hello, world]", received)
	}
	mu.Unlock()

	cleanup()
}

func TestGoLatestBasic(t *testing.T) {
	mockC := newMockCtx()
	setCurrentCtx(mockC)
	defer setCurrentCtx(nil)

	// Create owner and effect context
	owner := NewOwner(nil)
	setCurrentOwner(owner)
	defer setCurrentOwner(nil)

	// Create a mock effect
	var result string
	var resultErr error
	var wg sync.WaitGroup

	CreateEffect(func() Cleanup {
		wg.Add(1)
		return GoLatest("key1",
			func(ctx context.Context, key string) (string, error) {
				return "result for " + key, nil
			},
			func(r string, err error) {
				result = r
				resultErr = err
				wg.Done()
			},
		)
	})

	wg.Wait()

	if result != "result for key1" {
		t.Errorf("Result = %q, want %q", result, "result for key1")
	}
	if resultErr != nil {
		t.Errorf("Error = %v, want nil", resultErr)
	}
}

func TestGoLatestKeyCoalescing(t *testing.T) {
	mockC := newMockCtx()
	setCurrentCtx(mockC)
	defer setCurrentCtx(nil)

	owner := NewOwner(nil)
	setCurrentOwner(owner)
	defer setCurrentOwner(nil)

	var workCount atomic.Int32
	var applyCount atomic.Int32

	// First effect run
	effect := CreateEffect(func() Cleanup {
		workCount.Add(1)
		return GoLatest("same-key",
			func(ctx context.Context, key string) (int, error) {
				return int(workCount.Load()), nil
			},
			func(r int, err error) {
				applyCount.Add(1)
			},
		)
	})

	time.Sleep(10 * time.Millisecond)

	// Force effect to re-run (simulates dep change)
	effect.pending.Store(true)
	effect.run()

	time.Sleep(10 * time.Millisecond)

	// With key coalescing, work should only be done once for the same key
	// The second run should reuse state
	if applyCount.Load() < 1 {
		t.Error("Apply should have been called at least once")
	}
}

func TestTimeoutBasic(t *testing.T) {
	mockC := newMockCtx()
	setCurrentCtx(mockC)
	defer setCurrentCtx(nil)

	var called atomic.Bool
	cleanup := Timeout(10*time.Millisecond, func() {
		called.Store(true)
	})

	time.Sleep(20 * time.Millisecond)

	if !called.Load() {
		t.Error("Timeout callback should have been called")
	}

	// Cleanup should be safe to call after firing
	cleanup()
}

func TestTimeoutCancellation(t *testing.T) {
	mockC := newMockCtx()
	setCurrentCtx(mockC)
	defer setCurrentCtx(nil)

	var called atomic.Bool
	cleanup := Timeout(50*time.Millisecond, func() {
		called.Store(true)
	})

	// Cancel before timeout fires
	cleanup()
	time.Sleep(60 * time.Millisecond)

	if called.Load() {
		t.Error("Timeout callback should not have been called after cancel")
	}
}
