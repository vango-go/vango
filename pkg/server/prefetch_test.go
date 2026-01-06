package server

import (
	"testing"
	"time"

	"github.com/vango-go/vango/pkg/vdom"
)

// =============================================================================
// PrefetchCache Tests
// =============================================================================

func TestPrefetchCacheBasic(t *testing.T) {
	config := DefaultPrefetchConfig()
	cache := NewPrefetchCache(config)

	// Test set and get
	tree := &vdom.VNode{Tag: "div"}
	cache.Set("/test", tree)

	entry := cache.Get("/test")
	if entry == nil {
		t.Fatal("Expected entry, got nil")
	}
	if entry.Tree != tree {
		t.Error("Expected same tree")
	}
}

func TestPrefetchCacheMiss(t *testing.T) {
	config := DefaultPrefetchConfig()
	cache := NewPrefetchCache(config)

	// Get non-existent key
	entry := cache.Get("/nonexistent")
	if entry != nil {
		t.Error("Expected nil for miss, got entry")
	}
}

func TestPrefetchCacheLRUEviction(t *testing.T) {
	config := DefaultPrefetchConfig()
	config.MaxEntries = 3
	cache := NewPrefetchCache(config)

	// Set entries up to max
	cache.Set("/first", &vdom.VNode{Tag: "first"})
	cache.Set("/second", &vdom.VNode{Tag: "second"})
	cache.Set("/third", &vdom.VNode{Tag: "third"})

	// All should be present
	if cache.Get("/first") == nil {
		t.Error("/first should be present")
	}
	if cache.Get("/second") == nil {
		t.Error("/second should be present")
	}
	if cache.Get("/third") == nil {
		t.Error("/third should be present")
	}

	// Set one more - should evict oldest (first)
	cache.Set("/fourth", &vdom.VNode{Tag: "fourth"})

	// First should be evicted
	if cache.Get("/first") != nil {
		t.Error("/first should be evicted")
	}

	// Others should still be present
	if cache.Get("/second") == nil {
		t.Error("/second should still be present")
	}
	if cache.Get("/third") == nil {
		t.Error("/third should still be present")
	}
	if cache.Get("/fourth") == nil {
		t.Error("/fourth should be present")
	}
}

func TestPrefetchCacheLRUReorder(t *testing.T) {
	config := DefaultPrefetchConfig()
	config.MaxEntries = 3
	cache := NewPrefetchCache(config)

	// Set three entries
	cache.Set("/a", &vdom.VNode{Tag: "a"})
	cache.Set("/b", &vdom.VNode{Tag: "b"})
	cache.Set("/c", &vdom.VNode{Tag: "c"})

	// Access /a to move it to front
	cache.Get("/a")

	// Add new entry - should evict /b (least recently used), not /a
	cache.Set("/d", &vdom.VNode{Tag: "d"})

	if cache.Get("/a") == nil {
		t.Error("/a should still be present (was accessed)")
	}
	if cache.Get("/b") != nil {
		t.Error("/b should be evicted (least recently used)")
	}
	if cache.Get("/c") == nil {
		t.Error("/c should still be present")
	}
	if cache.Get("/d") == nil {
		t.Error("/d should be present")
	}
}

func TestPrefetchCacheTTLExpiration(t *testing.T) {
	config := DefaultPrefetchConfig()
	config.TTL = 50 * time.Millisecond
	cache := NewPrefetchCache(config)

	cache.Set("/test", &vdom.VNode{Tag: "test"})

	// Should be present immediately
	if cache.Get("/test") == nil {
		t.Error("Entry should be present")
	}

	// Wait for expiration
	time.Sleep(60 * time.Millisecond)

	// Should be expired
	if cache.Get("/test") != nil {
		t.Error("Entry should be expired")
	}
}

func TestPrefetchCacheOverwrite(t *testing.T) {
	config := DefaultPrefetchConfig()
	cache := NewPrefetchCache(config)

	// Set initial
	cache.Set("/test", &vdom.VNode{Tag: "first"})

	// Overwrite
	cache.Set("/test", &vdom.VNode{Tag: "second"})

	entry := cache.Get("/test")
	if entry == nil {
		t.Fatal("Expected entry")
	}
	if entry.Tree.Tag != "second" {
		t.Errorf("Expected tag 'second', got '%s'", entry.Tree.Tag)
	}
}

// =============================================================================
// PrefetchRateLimiter Tests
// =============================================================================

func TestPrefetchRateLimiterAllow(t *testing.T) {
	limiter := NewPrefetchRateLimiter(5.0) // 5 per second

	// Should allow first request immediately
	if !limiter.Allow() {
		t.Error("First request should be allowed")
	}
}

func TestPrefetchRateLimiterBurst(t *testing.T) {
	limiter := NewPrefetchRateLimiter(5.0) // 5 per second, burst = 5

	// Should allow up to burst
	allowed := 0
	for i := 0; i < 10; i++ {
		if limiter.Allow() {
			allowed++
		}
	}

	if allowed < 5 {
		t.Errorf("Expected at least 5 requests allowed (burst), got %d", allowed)
	}
}

func TestPrefetchRateLimiterRefill(t *testing.T) {
	limiter := NewPrefetchRateLimiter(100.0) // 100 per second for faster test

	// Exhaust burst
	for i := 0; i < 20; i++ {
		limiter.Allow()
	}

	// Wait for refill
	time.Sleep(50 * time.Millisecond)

	// Should have some tokens again
	if !limiter.Allow() {
		t.Error("Should have tokens after waiting")
	}
}

// =============================================================================
// PrefetchSemaphore Tests
// =============================================================================

func TestPrefetchSemaphoreAcquire(t *testing.T) {
	sem := NewPrefetchSemaphore(2)

	// First two acquires should succeed
	if !sem.Acquire() {
		t.Error("First acquire should succeed")
	}
	if !sem.Acquire() {
		t.Error("Second acquire should succeed")
	}

	// Third should fail
	if sem.Acquire() {
		t.Error("Third acquire should fail (at capacity)")
	}
}

func TestPrefetchSemaphoreRelease(t *testing.T) {
	sem := NewPrefetchSemaphore(1)

	// Acquire
	if !sem.Acquire() {
		t.Error("First acquire should succeed")
	}

	// Second should fail
	if sem.Acquire() {
		t.Error("Second acquire should fail")
	}

	// Release
	sem.Release()

	// Now should succeed
	if !sem.Acquire() {
		t.Error("Acquire after release should succeed")
	}
}

// =============================================================================
// RenderMode Tests
// =============================================================================

func TestRenderModeNormal(t *testing.T) {
	c := &ctx{
		mode: ModeNormal,
	}

	if c.Mode() != 0 {
		t.Errorf("Expected ModeNormal (0), got %d", c.Mode())
	}

	if c.IsPrefetch() {
		t.Error("Normal mode should not be prefetch")
	}
}

func TestRenderModePrefetch(t *testing.T) {
	c := &ctx{
		mode: ModePrefetch,
	}

	if c.Mode() != 1 {
		t.Errorf("Expected ModePrefetch (1), got %d", c.Mode())
	}

	if !c.IsPrefetch() {
		t.Error("Prefetch mode should report as prefetch")
	}
}

// =============================================================================
// Session Prefetch Methods Tests
// =============================================================================

func TestSessionPrefetchCacheAccess(t *testing.T) {
	s := &Session{
		prefetchConfig:    DefaultPrefetchConfig(),
		prefetchCache:     NewPrefetchCache(DefaultPrefetchConfig()),
		prefetchLimiter:   NewPrefetchRateLimiter(5.0),
		prefetchSemaphore: NewPrefetchSemaphore(2),
	}

	cache := s.PrefetchCache()
	if cache == nil {
		t.Error("Expected prefetch cache, got nil")
	}

	// Set and get via session
	tree := &vdom.VNode{Tag: "test"}
	cache.Set("/test", tree)

	entry := cache.Get("/test")
	if entry == nil || entry.Tree != tree {
		t.Error("Cache should store and retrieve tree")
	}
}

func TestSessionCreatePrefetchContext(t *testing.T) {
	s := &Session{
		prefetchConfig:    DefaultPrefetchConfig(),
		prefetchCache:     NewPrefetchCache(DefaultPrefetchConfig()),
		prefetchLimiter:   NewPrefetchRateLimiter(5.0),
		prefetchSemaphore: NewPrefetchSemaphore(2),
	}

	prefetchCtx := s.createPrefetchContext()
	if prefetchCtx == nil {
		t.Fatal("Expected prefetch context, got nil")
	}

	// Check that the context implements the mode interface
	// We can't directly check the type since ctx is unexported,
	// but we can check via the interface
	if modeChecker, ok := prefetchCtx.(interface{ Mode() int }); ok {
		if modeChecker.Mode() != 1 { // 1 = ModePrefetch
			t.Errorf("Expected ModePrefetch (1), got %d", modeChecker.Mode())
		}
	} else {
		t.Error("Context doesn't implement Mode() interface")
	}
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestPrefetchConfigDefaults(t *testing.T) {
	config := DefaultPrefetchConfig()

	if config.TTL != 30*time.Second {
		t.Errorf("Expected TTL 30s, got %v", config.TTL)
	}
	if config.MaxEntries != 10 {
		t.Errorf("Expected MaxEntries 10, got %d", config.MaxEntries)
	}
	if config.Timeout != 100*time.Millisecond {
		t.Errorf("Expected Timeout 100ms, got %v", config.Timeout)
	}
	if config.RateLimit != 5.0 {
		t.Errorf("Expected RateLimit 5.0, got %f", config.RateLimit)
	}
	if config.SessionConcurrency != 2 {
		t.Errorf("Expected SessionConcurrency 2, got %d", config.SessionConcurrency)
	}
	if config.GlobalConcurrency != 50 {
		t.Errorf("Expected GlobalConcurrency 50, got %d", config.GlobalConcurrency)
	}
}

// =============================================================================
// ctx.SetUser() Prefetch Tests (Section 8.3.2)
// =============================================================================

func TestCtxSetUserPanicsInPrefetch(t *testing.T) {
	// Per Section 8.3.2: ctx.SetUser() should panic in BOTH modes
	c := &ctx{
		mode: ModePrefetch,
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when calling SetUser in prefetch mode")
		}
	}()

	c.SetUser("test-user")
}

func TestCtxSetUserAllowedInNormalMode(t *testing.T) {
	c := &ctx{
		mode: ModeNormal,
	}

	// Should not panic
	c.SetUser("test-user")

	if c.User() != "test-user" {
		t.Errorf("Expected user to be set, got %v", c.User())
	}
}

// =============================================================================
// ctx.Navigate() Prefetch Tests (Section 8.3.2)
// =============================================================================

func TestCtxNavigateIgnoredInPrefetch(t *testing.T) {
	c := &ctx{
		mode: ModePrefetch,
	}

	// Should not panic, should be silently ignored
	c.Navigate("/test/path")

	// pendingNavigation should remain nil
	path, _, has := c.PendingNavigation()
	if has {
		t.Errorf("Expected no pending navigation in prefetch mode, got %s", path)
	}
}

func TestCtxNavigateWorksInNormalMode(t *testing.T) {
	s := &Session{
		prefetchConfig:    DefaultPrefetchConfig(),
		prefetchCache:     NewPrefetchCache(DefaultPrefetchConfig()),
		prefetchLimiter:   NewPrefetchRateLimiter(5.0),
		prefetchSemaphore: NewPrefetchSemaphore(2),
	}

	c := &ctx{
		mode:    ModeNormal,
		session: s,
	}

	c.Navigate("/test/path")

	path, _, has := c.PendingNavigation()
	if !has {
		t.Error("Expected pending navigation to be set")
	}
	if path != "/test/path" {
		t.Errorf("Expected path /test/path, got %s", path)
	}
}
