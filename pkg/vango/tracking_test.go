package vango

import (
	"sync"
	"testing"
)

// testListener is a simple Listener implementation for testing.
type testListener struct {
	id         uint64
	dirtyCount int
	mu         sync.Mutex
}

func newTestListener() *testListener {
	return &testListener{id: nextID()}
}

func (l *testListener) MarkDirty() {
	l.mu.Lock()
	l.dirtyCount++
	l.mu.Unlock()
}

func (l *testListener) ID() uint64 {
	return l.id
}

func (l *testListener) getDirtyCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.dirtyCount
}

func TestGetTrackingContext(t *testing.T) {
	// Getting context should return the same context for same goroutine
	ctx1 := getTrackingContext()
	ctx2 := getTrackingContext()

	if ctx1 != ctx2 {
		t.Error("getTrackingContext should return same context for same goroutine")
	}
}

func TestTrackingContextIsolation(t *testing.T) {
	// Each goroutine should have its own context
	var wg sync.WaitGroup
	contexts := make(chan *TrackingContext, 2)

	wg.Add(2)

	go func() {
		defer wg.Done()
		ctx := getTrackingContext()
		ctx.batchDepth = 42
		contexts <- ctx
	}()

	go func() {
		defer wg.Done()
		ctx := getTrackingContext()
		ctx.batchDepth = 99
		contexts <- ctx
	}()

	wg.Wait()
	close(contexts)

	var ctxList []*TrackingContext
	for ctx := range contexts {
		ctxList = append(ctxList, ctx)
	}

	if len(ctxList) != 2 {
		t.Fatalf("expected 2 contexts, got %d", len(ctxList))
	}

	if ctxList[0] == ctxList[1] {
		t.Error("different goroutines should have different contexts")
	}

	// Verify each context has its own state
	depths := map[int]bool{}
	for _, ctx := range ctxList {
		depths[ctx.batchDepth] = true
	}

	if !depths[42] || !depths[99] {
		t.Error("contexts should maintain independent state")
	}
}

func TestCurrentListener(t *testing.T) {
	// Initially no listener
	if getCurrentListener() != nil {
		t.Error("should have no listener initially")
	}

	// Set a listener
	listener := newTestListener()
	old := setCurrentListener(listener)

	if old != nil {
		t.Error("old listener should be nil")
	}

	if getCurrentListener() != listener {
		t.Error("getCurrentListener should return set listener")
	}

	// Restore
	setCurrentListener(old)
	if getCurrentListener() != nil {
		t.Error("listener should be nil after restore")
	}
}

func TestWithListener(t *testing.T) {
	listener := newTestListener()

	// Verify listener is set within the function
	var capturedListener Listener
	WithListener(listener, func() {
		capturedListener = getCurrentListener()
	})

	if capturedListener != listener {
		t.Error("listener should be set during WithListener callback")
	}

	// Verify listener is restored after
	if getCurrentListener() != nil {
		t.Error("listener should be restored after WithListener")
	}
}

func TestWithListenerNested(t *testing.T) {
	listener1 := newTestListener()
	listener2 := newTestListener()

	var innerListener, outerAfterInner Listener

	WithListener(listener1, func() {
		WithListener(listener2, func() {
			innerListener = getCurrentListener()
		})
		outerAfterInner = getCurrentListener()
	})

	if innerListener != listener2 {
		t.Error("inner listener should be listener2")
	}

	if outerAfterInner != listener1 {
		t.Error("outer listener should be restored to listener1")
	}
}

func TestBatchDepth(t *testing.T) {
	// Initially 0
	if getBatchDepth() != 0 {
		t.Error("batch depth should start at 0")
	}

	// Increment
	incrementBatchDepth()
	if getBatchDepth() != 1 {
		t.Error("batch depth should be 1 after increment")
	}

	// Nested increment
	incrementBatchDepth()
	if getBatchDepth() != 2 {
		t.Error("batch depth should be 2 after second increment")
	}

	// Decrement returns false when not at 0
	if decrementBatchDepth() {
		t.Error("decrementBatchDepth should return false when depth > 0")
	}
	if getBatchDepth() != 1 {
		t.Error("batch depth should be 1 after first decrement")
	}

	// Final decrement returns true
	if !decrementBatchDepth() {
		t.Error("decrementBatchDepth should return true when reaching 0")
	}
	if getBatchDepth() != 0 {
		t.Error("batch depth should be 0 after second decrement")
	}
}

func TestPendingUpdates(t *testing.T) {
	listener1 := newTestListener()
	listener2 := newTestListener()

	// Initially empty
	updates := drainPendingUpdates()
	if len(updates) != 0 {
		t.Error("pending updates should be empty initially")
	}

	// Queue some updates
	queuePendingUpdate(listener1)
	queuePendingUpdate(listener2)
	queuePendingUpdate(listener1) // duplicate

	updates = drainPendingUpdates()
	if len(updates) != 3 {
		t.Errorf("expected 3 pending updates (including dupe), got %d", len(updates))
	}

	// Should be empty after drain
	updates = drainPendingUpdates()
	if len(updates) != 0 {
		t.Error("pending updates should be empty after drain")
	}
}

func TestCleanupGoroutineContext(t *testing.T) {
	// Set some state
	ctx := getTrackingContext()
	ctx.batchDepth = 5

	// Cleanup
	cleanupGoroutineContext()

	// Getting context again should return fresh context
	newCtx := getTrackingContext()
	if newCtx.batchDepth != 0 {
		t.Error("new context should have fresh state")
	}
}

func TestConcurrentContextAccess(t *testing.T) {
	// Test that concurrent access to tracking contexts is safe
	var wg sync.WaitGroup
	const numGoroutines = 100
	const numIterations = 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				ctx := getTrackingContext()
				ctx.batchDepth = id
				_ = getBatchDepth()
				incrementBatchDepth()
				decrementBatchDepth()

				listener := newTestListener()
				setCurrentListener(listener)
				_ = getCurrentListener()
				setCurrentListener(nil)

				queuePendingUpdate(listener)
				drainPendingUpdates()
			}
		}(i)
	}

	wg.Wait()
}
