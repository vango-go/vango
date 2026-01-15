package server

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vango-go/vango/pkg/protocol"
	"github.com/vango-go/vango/pkg/vango"
	"github.com/vango-go/vango/pkg/vdom"
)

func TestGenerateSessionID(t *testing.T) {
	ids := make(map[string]bool)

	// Generate 100 IDs and ensure uniqueness
	for i := 0; i < 100; i++ {
		id := generateSessionID()
		if id == "" {
			t.Error("Session ID should not be empty")
		}
		if len(id) != 32 { // 16 bytes hex encoded = 32 chars
			t.Errorf("Session ID length = %d, want 32", len(id))
		}
		if ids[id] {
			t.Error("Session ID should be unique")
		}
		ids[id] = true
	}
}

func TestNewSession(t *testing.T) {
	config := DefaultSessionConfig()
	logger := slog.Default()

	session := newSession(nil, "user123", config, logger)

	if session == nil {
		t.Fatal("newSession should not return nil")
	}
	if session.ID == "" {
		t.Error("Session ID should not be empty")
	}
	if session.UserID != "user123" {
		t.Errorf("UserID = %s, want user123", session.UserID)
	}
	if session.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if session.LastActive.IsZero() {
		t.Error("LastActive should be set")
	}
	if session.handlers == nil {
		t.Error("handlers should be initialized")
	}
	if session.components == nil {
		t.Error("components should be initialized")
	}
	if session.owner == nil {
		t.Error("owner should be initialized")
	}
	if session.events == nil {
		t.Error("events channel should be initialized")
	}
	if session.dispatchCh == nil {
		t.Error("dispatchCh channel should be initialized")
	}
}

func TestSessionDispatch(t *testing.T) {
	config := DefaultSessionConfig()
	session := newSession(nil, "", config, slog.Default())

	// Dispatch should queue function
	called := false
	session.Dispatch(func() {
		called = true
	})

	// Verify function is in channel
	select {
	case fn := <-session.dispatchCh:
		fn()
		if !called {
			t.Error("Dispatched function should be callable")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Function should be dispatched")
	}
}

func TestSessionDispatchWhenClosed(t *testing.T) {
	config := DefaultSessionConfig()
	session := newSession(nil, "", config, slog.Default())

	session.closed.Store(true)

	// Should not panic or block
	session.Dispatch(func() {
		t.Error("Should not be called when session is closed")
	})

	// Verify nothing was queued
	select {
	case <-session.dispatchCh:
		t.Error("Nothing should be queued when closed")
	default:
		// Good
	}
}

func TestSessionDispatchConcurrent(t *testing.T) {
	config := DefaultSessionConfig()
	config.MaxEventQueue = 1000
	session := newSession(nil, "", config, slog.Default())

	var wg sync.WaitGroup
	dispatched := atomic.Int32{}

	// Dispatch from many goroutines
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			session.Dispatch(func() {
				dispatched.Add(1)
			})
		}()
	}

	wg.Wait()

	// Drain and count
	count := 0
	for {
		select {
		case fn := <-session.dispatchCh:
			fn()
			count++
		default:
			goto done
		}
	}
done:

	if count != 100 {
		t.Errorf("Dispatched %d functions, want 100", count)
	}
}

func TestSessionCreateEventContext(t *testing.T) {
	config := DefaultSessionConfig()
	session := newSession(nil, "", config, slog.Default())

	event := &Event{
		HID:  "h1",
		Type: protocol.EventClick,
		Seq:  42,
	}

	ctx := session.createEventContext(event)

	if ctx == nil {
		t.Fatal("createEventContext should not return nil")
	}
	if ctx.Session() != session {
		t.Error("Session should match")
	}
	if ctx.Event() != event {
		t.Error("Event should match")
	}
	if ctx.StdContext() == nil {
		t.Error("StdContext should not be nil")
	}
}

func TestSessionCreateRenderContext(t *testing.T) {
	config := DefaultSessionConfig()
	session := newSession(nil, "", config, slog.Default())

	ctx := session.createRenderContext()

	if ctx == nil {
		t.Fatal("createRenderContext should not return nil")
	}
	if ctx.Session() != session {
		t.Error("Session should match")
	}
	if ctx.StdContext() == nil {
		t.Error("StdContext should not be nil")
	}
}

func TestSessionQueueEvent(t *testing.T) {
	config := DefaultSessionConfig()
	session := newSession(nil, "", config, slog.Default())

	event := &Event{HID: "h1", Type: protocol.EventClick}
	err := session.QueueEvent(event)
	if err != nil {
		t.Errorf("QueueEvent error: %v", err)
	}

	// Verify event is queued
	select {
	case e := <-session.events:
		if e != event {
			t.Error("Queued event should match")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Event should be queued")
	}
}

func TestSessionQueueEventFull(t *testing.T) {
	config := DefaultSessionConfig()
	config.MaxEventQueue = 1 // Very small queue
	session := newSession(nil, "", config, slog.Default())

	// Fill the queue
	session.QueueEvent(&Event{HID: "h1"})

	// Next one should fail
	err := session.QueueEvent(&Event{HID: "h2"})
	if err == nil {
		t.Error("QueueEvent should return error when queue is full")
	}
}

func TestSessionIsClosed(t *testing.T) {
	config := DefaultSessionConfig()
	session := newSession(nil, "", config, slog.Default())

	if session.IsClosed() {
		t.Error("New session should not be closed")
	}

	session.closed.Store(true)

	if !session.IsClosed() {
		t.Error("Session should be closed after setting flag")
	}
}

func TestSessionDone(t *testing.T) {
	config := DefaultSessionConfig()
	session := newSession(nil, "", config, slog.Default())

	done := session.Done()
	select {
	case <-done:
		t.Error("Done should not be closed for new session")
	default:
		// Good
	}
}

func TestSessionUpdateLastActive(t *testing.T) {
	config := DefaultSessionConfig()
	session := newSession(nil, "", config, slog.Default())

	originalTime := session.LastActive
	time.Sleep(10 * time.Millisecond)

	session.UpdateLastActive()

	if !session.LastActive.After(originalTime) {
		t.Error("LastActive should be updated")
	}
}

func TestSessionStats(t *testing.T) {
	config := DefaultSessionConfig()
	session := newSession(nil, "user123", config, slog.Default())

	session.eventCount.Store(10)
	session.patchCount.Store(5)
	session.bytesSent.Store(1000)
	session.bytesRecv.Store(500)

	stats := session.Stats()

	if stats.ID != session.ID {
		t.Error("Stats ID should match")
	}
	if stats.UserID != "user123" {
		t.Error("Stats UserID should match")
	}
	if stats.EventCount != 10 {
		t.Errorf("EventCount = %d, want 10", stats.EventCount)
	}
	if stats.PatchCount != 5 {
		t.Errorf("PatchCount = %d, want 5", stats.PatchCount)
	}
	if stats.BytesSent != 1000 {
		t.Errorf("BytesSent = %d, want 1000", stats.BytesSent)
	}
	if stats.BytesRecv != 500 {
		t.Errorf("BytesRecv = %d, want 500", stats.BytesRecv)
	}
}

func TestSessionBytesReceived(t *testing.T) {
	config := DefaultSessionConfig()
	session := newSession(nil, "", config, slog.Default())

	session.BytesReceived(100)
	session.BytesReceived(50)

	if session.bytesRecv.Load() != 150 {
		t.Errorf("bytesRecv = %d, want 150", session.bytesRecv.Load())
	}
}

func TestSessionGetSet(t *testing.T) {
	config := DefaultSessionConfig()
	session := newSession(nil, "", config, slog.Default())

	// Initially nil
	if session.Get("key") != nil {
		t.Error("Get should return nil for unset key")
	}

	// Set and get
	session.Set("string", "value")
	session.Set("int", 42)

	if session.Get("string") != "value" {
		t.Error("Get(string) should return value")
	}
	if session.Get("int") != 42 {
		t.Error("Get(int) should return 42")
	}
}

func TestSessionGetSetConcurrent(t *testing.T) {
	config := DefaultSessionConfig()
	session := newSession(nil, "", config, slog.Default())

	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			session.Set("key", i)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			session.Get("key")
		}()
	}

	wg.Wait()
	// No race condition = success
}

func TestSessionDelete(t *testing.T) {
	config := DefaultSessionConfig()
	session := newSession(nil, "", config, slog.Default())

	session.Set("key", "value")
	session.Delete("key")

	if session.Get("key") != nil {
		t.Error("Delete should remove key")
	}
}

func TestSessionOwner(t *testing.T) {
	config := DefaultSessionConfig()
	session := newSession(nil, "", config, slog.Default())

	owner := session.Owner()
	if owner == nil {
		t.Error("Owner should not be nil")
	}
	if owner != session.owner {
		t.Error("Owner should match internal owner")
	}
}

func TestSessionConfig(t *testing.T) {
	config := DefaultSessionConfig()
	config.ReadTimeout = 99 * time.Second
	session := newSession(nil, "", config, slog.Default())

	if session.Config() != config {
		t.Error("Config should match")
	}
	if session.Config().ReadTimeout != 99*time.Second {
		t.Error("Config values should match")
	}
}

func TestSessionLogger(t *testing.T) {
	logger := slog.Default()
	session := newSession(nil, "", DefaultSessionConfig(), logger)

	if session.Logger() == nil {
		t.Error("Logger should not be nil")
	}
}

// Test that handleEvent sets up context correctly
func TestSessionHandleEventSetsContext(t *testing.T) {
	config := DefaultSessionConfig()
	session := newSession(nil, "", config, slog.Default())

	var capturedCtx vango.Ctx

	// Register a handler that captures the context
	// Key format is HID_eventtype (e.g., "h1_onclick")
	session.handlers["h1_onclick"] = func(event *Event) {
		capturedCtx = vango.UseCtx()
	}

	event := &Event{
		HID:  "h1",
		Type: protocol.EventClick,
		Seq:  1,
	}

	session.handleEvent(event)

	if capturedCtx == nil {
		t.Error("UseCtx should return context during handler execution")
	}
}

// Test component render with context
func TestComponentRenderWithContext(t *testing.T) {
	config := DefaultSessionConfig()
	session := newSession(nil, "", config, slog.Default())

	var capturedCtx vango.Ctx

	comp := FuncComponent(func() *vdom.VNode {
		capturedCtx = vango.UseCtx()
		return &vdom.VNode{Tag: "div"}
	})

	instance := newComponentInstance(comp, nil, session)
	instance.Render()

	if capturedCtx == nil {
		t.Error("UseCtx should return context during render")
	}
}

// Test that Dispatch executes on event loop with effects
func TestSessionExecuteDispatch(t *testing.T) {
	config := DefaultSessionConfig()
	session := newSession(nil, "", config, slog.Default())

	executed := false

	// Simulate executeDispatch behavior
	fn := func() {
		executed = true
	}

	session.executeDispatch(fn)

	if !executed {
		t.Error("Dispatched function should be executed")
	}
}

// Test executeDispatch panic recovery
func TestSessionExecuteDispatchPanicRecovery(t *testing.T) {
	config := DefaultSessionConfig()
	session := newSession(nil, "", config, slog.Default())

	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Error("executeDispatch should recover from panic")
		}
	}()

	session.executeDispatch(func() {
		panic("test panic")
	})
}

// Test scheduleRender
func TestSessionScheduleRender(t *testing.T) {
	config := DefaultSessionConfig()
	session := newSession(nil, "", config, slog.Default())

	comp := &mockComponent{}
	instance := newComponentInstance(comp, nil, session)

	session.scheduleRender(instance)

	// Verify render signal was sent
	select {
	case <-session.renderCh:
		// Good
	case <-time.After(100 * time.Millisecond):
		t.Error("scheduleRender should signal render channel")
	}
}

// Test that context propagation works with StdContext
func TestContextPropagationStdContext(t *testing.T) {
	config := DefaultSessionConfig()
	session := newSession(nil, "", config, slog.Default())

	var capturedStdCtx context.Context

	// Key format is HID_eventtype (e.g., "h1_onclick")
	session.handlers["h1_onclick"] = func(event *Event) {
		ctx := vango.UseCtx()
		if ctx != nil {
			capturedStdCtx = ctx.StdContext()
		}
	}

	event := &Event{HID: "h1", Type: protocol.EventClick}
	session.handleEvent(event)

	if capturedStdCtx == nil {
		t.Error("StdContext should be available during handler")
	}
}

// Test memory usage estimation
func TestSessionMemoryUsage(t *testing.T) {
	config := DefaultSessionConfig()
	session := newSession(nil, "", config, slog.Default())

	// Add some data
	session.Set("key1", "value1")
	session.Set("key2", "value2")

	usage := session.MemoryUsage()
	if usage <= 0 {
		t.Error("MemoryUsage should be positive")
	}
}

func TestSessionMemoryUsageIncludesCaches(t *testing.T) {
	config := DefaultSessionConfig()
	session := newSession(nil, "", config, slog.Default())

	before := session.MemoryUsage()

	session.Set("payload", strings.Repeat("a", 128))
	session.patchHistory.Add(1, []byte("patch-frame"))
	session.PrefetchCache().Set("/prefetch", &vdom.VNode{Tag: "div", Text: "cached"})

	after := session.MemoryUsage()
	if after <= before {
		t.Errorf("MemoryUsage should increase after caches/data (before=%d after=%d)", before, after)
	}
}

// Test safeExecute panic recovery
func TestSessionSafeExecute(t *testing.T) {
	config := DefaultSessionConfig()
	session := newSession(nil, "", config, slog.Default())

	event := &Event{HID: "h1", Type: protocol.EventClick}

	// Handler that panics
	panicHandler := func(e *Event) {
		panic("test panic")
	}

	// safeExecute should recover from panic and not propagate it
	// If this test completes without panicking, safeExecute worked correctly
	session.safeExecute(panicHandler, event)

	// If we got here, the panic was recovered
	// The test passes by not panicking
}

// Test that handleEvent updates sequence and metrics
func TestSessionHandleEventMetrics(t *testing.T) {
	config := DefaultSessionConfig()
	session := newSession(nil, "", config, slog.Default())

	// Key format is HID_eventtype (e.g., "h1_onclick")
	session.handlers["h1_onclick"] = func(e *Event) {}

	event := &Event{HID: "h1", Type: protocol.EventClick, Seq: 42}

	initialCount := session.eventCount.Load()
	session.handleEvent(event)

	if session.recvSeq.Load() != 42 {
		t.Errorf("recvSeq = %d, want 42", session.recvSeq.Load())
	}
	if session.eventCount.Load() != initialCount+1 {
		t.Error("eventCount should be incremented")
	}
}

// Test checkSerializable in debug mode
func TestCheckSerializableDebugMode(t *testing.T) {
	oldDebug := DebugMode
	DebugMode = true
	defer func() { DebugMode = oldDebug }()

	// Should panic on func
	defer func() {
		if r := recover(); r == nil {
			t.Error("checkSerializable should panic on func")
		}
	}()

	checkSerializable("key", func() {})
}

func TestCheckSerializableChan(t *testing.T) {
	oldDebug := DebugMode
	DebugMode = true
	defer func() { DebugMode = oldDebug }()

	defer func() {
		if r := recover(); r == nil {
			t.Error("checkSerializable should panic on chan")
		}
	}()

	checkSerializable("key", make(chan int))
}

func TestCheckSerializableNormal(t *testing.T) {
	oldDebug := DebugMode
	DebugMode = true
	defer func() { DebugMode = oldDebug }()

	// Should not panic on normal values
	checkSerializable("string", "hello")
	checkSerializable("int", 42)
	checkSerializable("slice", []int{1, 2, 3})
	checkSerializable("map", map[string]int{"a": 1})
	checkSerializable("nil", nil)
}
