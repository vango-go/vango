package server

import (
	"container/list"
	"sync"
	"time"

	"github.com/vango-go/vango/pkg/vdom"
)

// =============================================================================
// Prefetch Configuration (Section 8)
// =============================================================================

// PrefetchConfig holds configuration for the prefetch system.
// These defaults are per Section 8.2-8.3 of the Routing Spec.
type PrefetchConfig struct {
	// TTL is how long a prefetched result is valid.
	// Default: 30 seconds
	TTL time.Duration

	// MaxEntries is the maximum number of cached entries per session.
	// Uses LRU eviction when exceeded.
	// Default: 10
	MaxEntries int

	// Timeout is the maximum time to wait for a prefetch to complete.
	// If exceeded, the prefetch is aborted and result is not cached.
	// Default: 100ms
	Timeout time.Duration

	// RateLimit is the maximum prefetch requests per second per session.
	// Excess requests are silently dropped.
	// Default: 5
	RateLimit float64

	// SessionConcurrency is the max simultaneous prefetch evaluations per session.
	// Default: 2
	SessionConcurrency int

	// GlobalConcurrency is the max simultaneous prefetch evaluations globally.
	// Default: 50
	GlobalConcurrency int
}

// DefaultPrefetchConfig returns the default prefetch configuration.
// Per Section 8.2 and 8.3.3 of the Routing Spec.
func DefaultPrefetchConfig() *PrefetchConfig {
	return &PrefetchConfig{
		TTL:                30 * time.Second,
		MaxEntries:         10,
		Timeout:            100 * time.Millisecond,
		RateLimit:          5.0,
		SessionConcurrency: 2,
		GlobalConcurrency:  50,
	}
}

// =============================================================================
// Prefetch Cache (Section 8.2)
// =============================================================================

// PrefetchCacheEntry holds a cached prefetch result.
type PrefetchCacheEntry struct {
	// Tree is the rendered VNode tree
	Tree *vdom.VNode

	// CreatedAt is when this entry was created
	CreatedAt time.Time

	// ExpiresAt is when this entry expires
	ExpiresAt time.Time
}

// IsExpired returns true if the entry has expired.
func (e *PrefetchCacheEntry) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

// PrefetchCache is an LRU cache for prefetched route renders.
// Each session has its own cache instance.
//
// Per Section 8.2:
//   - Keyed by canonical path
//   - TTL: 30 seconds (configurable)
//   - Max entries: 10 (LRU eviction)
type PrefetchCache struct {
	mu         sync.Mutex
	config     *PrefetchConfig
	entries    map[string]*list.Element
	order      *list.List // LRU order (front = most recent)
	expireOnce sync.Once  // Lazy start expiry goroutine
	bytes      int64
}

// prefetchItem holds an entry in the LRU list.
type prefetchItem struct {
	key   string
	entry *PrefetchCacheEntry
	size  int64
}

// NewPrefetchCache creates a new prefetch cache.
func NewPrefetchCache(config *PrefetchConfig) *PrefetchCache {
	if config == nil {
		config = DefaultPrefetchConfig()
	}
	return &PrefetchCache{
		config:  config,
		entries: make(map[string]*list.Element),
		order:   list.New(),
	}
}

// Get retrieves a cached prefetch result.
// Returns nil if not found or expired.
func (c *PrefetchCache) Get(path string) *PrefetchCacheEntry {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.entries[path]
	if !ok {
		return nil
	}

	item := elem.Value.(*prefetchItem)
	if item.entry.IsExpired() {
		// Remove expired entry
		c.bytes -= item.size
		c.order.Remove(elem)
		delete(c.entries, path)
		return nil
	}

	// Move to front (most recently used)
	c.order.MoveToFront(elem)
	return item.entry
}

// Set stores a prefetch result in the cache.
// If the cache is full, the least recently used entry is evicted.
func (c *PrefetchCache) Set(path string, tree *vdom.VNode) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	entry := &PrefetchCacheEntry{
		Tree:      tree,
		CreatedAt: now,
		ExpiresAt: now.Add(c.config.TTL),
	}
	entrySize := estimatePrefetchEntrySize(path, entry)

	// If path exists, update in place
	if elem, ok := c.entries[path]; ok {
		item := elem.Value.(*prefetchItem)
		c.bytes -= item.size
		item.entry = entry
		item.size = entrySize
		c.bytes += entrySize
		c.order.MoveToFront(elem)
		return
	}

	// Evict LRU entries if at capacity
	for c.order.Len() >= c.config.MaxEntries {
		oldest := c.order.Back()
		if oldest == nil {
			break
		}
		item := oldest.Value.(*prefetchItem)
		c.bytes -= item.size
		c.order.Remove(oldest)
		delete(c.entries, item.key)
	}

	// Add new entry
	item := &prefetchItem{key: path, entry: entry, size: entrySize}
	elem := c.order.PushFront(item)
	c.entries[path] = elem
	c.bytes += entrySize
}

// Delete removes a cached entry.
func (c *PrefetchCache) Delete(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.entries[path]; ok {
		item := elem.Value.(*prefetchItem)
		c.bytes -= item.size
		c.order.Remove(elem)
		delete(c.entries, path)
	}
}

// Clear removes all cached entries.
func (c *PrefetchCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*list.Element)
	c.order = list.New()
	c.bytes = 0
}

// Len returns the number of cached entries.
func (c *PrefetchCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}

// MemoryUsage estimates the memory used by the prefetch cache.
func (c *PrefetchCache) MemoryUsage() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	var size int64 = 128 // Base struct + mutex overhead
	size += EstimateMapMemory(len(c.entries), 16, 16)
	size += EstimateSliceMemory(c.order.Len(), 32)
	size += c.bytes
	return size
}

func estimatePrefetchEntrySize(path string, entry *PrefetchCacheEntry) int64 {
	if entry == nil {
		return 0
	}
	var size int64 = 96 // Entry + list element overhead
	size += EstimateStringMemory(path)
	if entry.Tree != nil {
		size += estimateVNodeSize(entry.Tree)
	}
	return size
}

// =============================================================================
// Prefetch Rate Limiter (Section 8.5)
// =============================================================================

// PrefetchRateLimiter implements token bucket rate limiting for prefetch requests.
// Per Section 8.5:
//   - Max 5 prefetch requests per second per session
//   - Excess requests are silently dropped
type PrefetchRateLimiter struct {
	mu            sync.Mutex
	ratePerSecond float64
	tokens        float64
	lastRefill    time.Time
}

// NewPrefetchRateLimiter creates a new rate limiter.
func NewPrefetchRateLimiter(ratePerSecond float64) *PrefetchRateLimiter {
	return &PrefetchRateLimiter{
		ratePerSecond: ratePerSecond,
		tokens:        ratePerSecond, // Start with full bucket
		lastRefill:    time.Now(),
	}
}

// Allow returns true if a prefetch request is allowed.
// Returns false if rate limit is exceeded (request should be dropped).
func (r *PrefetchRateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(r.lastRefill).Seconds()

	// Refill tokens based on elapsed time
	r.tokens += elapsed * r.ratePerSecond
	if r.tokens > r.ratePerSecond {
		r.tokens = r.ratePerSecond // Cap at max bucket size
	}
	r.lastRefill = now

	// Check if we have a token available
	if r.tokens >= 1.0 {
		r.tokens -= 1.0
		return true
	}

	return false
}

// =============================================================================
// Prefetch Semaphore (Section 8.3.3)
// =============================================================================

// PrefetchSemaphore limits concurrent prefetch operations.
// Used both per-session and globally.
type PrefetchSemaphore struct {
	ch chan struct{}
}

// NewPrefetchSemaphore creates a new semaphore with the given limit.
func NewPrefetchSemaphore(limit int) *PrefetchSemaphore {
	return &PrefetchSemaphore{
		ch: make(chan struct{}, limit),
	}
}

// Acquire tries to acquire a slot. Returns true if successful.
// If the semaphore is full, returns false immediately (non-blocking).
func (s *PrefetchSemaphore) Acquire() bool {
	select {
	case s.ch <- struct{}{}:
		return true
	default:
		return false
	}
}

// Release releases a slot.
func (s *PrefetchSemaphore) Release() {
	select {
	case <-s.ch:
	default:
		// Shouldn't happen, but don't panic
	}
}

// =============================================================================
// Global Prefetch Manager
// =============================================================================

// PrefetchManager coordinates prefetch operations across all sessions.
// It provides global concurrency limiting.
var globalPrefetchManager = &prefetchManager{
	semaphore: NewPrefetchSemaphore(50), // Default global limit
}

type prefetchManager struct {
	semaphore *PrefetchSemaphore
}

// GlobalPrefetchSemaphore returns the global prefetch semaphore.
// Used by sessions to check global concurrency limits.
func GlobalPrefetchSemaphore() *PrefetchSemaphore {
	return globalPrefetchManager.semaphore
}

// SetGlobalPrefetchLimit sets the global prefetch concurrency limit.
// Should be called at server startup before any prefetch operations.
func SetGlobalPrefetchLimit(limit int) {
	globalPrefetchManager.semaphore = NewPrefetchSemaphore(limit)
}
