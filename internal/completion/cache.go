package completion

import (
	"sync"
	"time"

	"github.com/sqve/grove/internal/logger"
)

// Cache value constants
const (
	CacheValueTrue  = "true"
	CacheValueFalse = "false"
)

// CacheEntry represents a cached completion result
type CacheEntry struct {
	Value     []string
	Timestamp time.Time
	TTL       time.Duration
}

// IsExpired checks if the cache entry has expired
func (c *CacheEntry) IsExpired() bool {
	return time.Since(c.Timestamp) > c.TTL
}

// CompletionCache provides caching for completion results
type CompletionCache struct {
	cache map[string]*CacheEntry
	mutex sync.RWMutex
}

// NewCompletionCache creates a new completion cache
func NewCompletionCache() *CompletionCache {
	return &CompletionCache{
		cache: make(map[string]*CacheEntry),
	}
}

// Get retrieves a cached completion result
func (c *CompletionCache) Get(key string) ([]string, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.cache[key]
	if !exists {
		return nil, false
	}

	if entry.IsExpired() {
		// Clean up expired entry
		go c.Delete(key)
		return nil, false
	}

	return entry.Value, true
}

// Set stores a completion result in the cache
func (c *CompletionCache) Set(key string, value []string, ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.cache[key] = &CacheEntry{
		Value:     value,
		Timestamp: time.Now(),
		TTL:       ttl,
	}
}

// Delete removes a cache entry
func (c *CompletionCache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.cache, key)
}

// Clear removes all cache entries
func (c *CompletionCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.cache = make(map[string]*CacheEntry)
}

// Size returns the number of cached entries
func (c *CompletionCache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return len(c.cache)
}

// CleanupExpired removes expired cache entries
func (c *CompletionCache) CleanupExpired() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	log := logger.WithComponent("completion_cache")
	cleaned := 0

	for key, entry := range c.cache {
		if entry.IsExpired() {
			delete(c.cache, key)
			cleaned++
		}
	}

	if cleaned > 0 {
		log.Debug("cleaned up expired cache entries", "count", cleaned, "remaining", len(c.cache))
	}
}

// GlobalCache is the global completion cache instance
var GlobalCache = NewCompletionCache()

// CacheKeyBuilder helps build consistent cache keys
type CacheKeyBuilder struct {
	parts []string
}

// NewCacheKeyBuilder creates a new cache key builder
func NewCacheKeyBuilder() *CacheKeyBuilder {
	return &CacheKeyBuilder{
		parts: make([]string, 0),
	}
}

// Add adds a part to the cache key
func (b *CacheKeyBuilder) Add(part string) *CacheKeyBuilder {
	b.parts = append(b.parts, part)
	return b
}

// Build builds the final cache key
func (b *CacheKeyBuilder) Build() string {
	if len(b.parts) == 0 {
		return ""
	}

	result := b.parts[0]
	for i := 1; i < len(b.parts); i++ {
		result += ":" + b.parts[i]
	}

	return result
}

// Common cache TTL values
const (
	BranchCacheTTL     = 30 * time.Second
	WorktreeCacheTTL   = 10 * time.Second
	RepositoryCacheTTL = 5 * time.Second
	URLCacheTTL        = 5 * time.Minute
	NetworkCacheTTL    = 30 * time.Second
)

// GetCachedBranches retrieves cached branch list
func GetCachedBranches(ctx *CompletionContext) ([]string, bool) {
	key := NewCacheKeyBuilder().Add("branches").Build()
	return GlobalCache.Get(key)
}

// SetCachedBranches stores branch list in cache
func SetCachedBranches(ctx *CompletionContext, branches []string) {
	key := NewCacheKeyBuilder().Add("branches").Build()
	GlobalCache.Set(key, branches, BranchCacheTTL)
}

// GetCachedWorktrees retrieves cached worktree list
func GetCachedWorktrees(ctx *CompletionContext) ([]string, bool) {
	key := NewCacheKeyBuilder().Add("worktrees").Build()
	return GlobalCache.Get(key)
}

// SetCachedWorktrees stores worktree list in cache
func SetCachedWorktrees(ctx *CompletionContext, worktrees []string) {
	key := NewCacheKeyBuilder().Add("worktrees").Build()
	GlobalCache.Set(key, worktrees, WorktreeCacheTTL)
}

// GetCachedRepositoryState retrieves cached repository state
func GetCachedRepositoryState() (isGroveRepo, exists bool) {
	key := NewCacheKeyBuilder().Add("repo_state").Build()
	if value, exists := GlobalCache.Get(key); exists && len(value) > 0 {
		return value[0] == CacheValueTrue, true
	}
	return false, false
}

// SetCachedRepositoryState stores repository state in cache
func SetCachedRepositoryState(isGroveRepo bool) {
	key := NewCacheKeyBuilder().Add("repo_state").Build()
	value := CacheValueFalse
	if isGroveRepo {
		value = CacheValueTrue
	}
	GlobalCache.Set(key, []string{value}, RepositoryCacheTTL)
}

// GetCachedNetworkState retrieves cached network connectivity state
func GetCachedNetworkState() (isOnline, exists bool) {
	key := NewCacheKeyBuilder().Add("network_state").Build()
	if value, exists := GlobalCache.Get(key); exists && len(value) > 0 {
		return value[0] == CacheValueTrue, true
	}
	return false, false
}

// SetCachedNetworkState stores network connectivity state in cache
func SetCachedNetworkState(isOnline bool) {
	key := NewCacheKeyBuilder().Add("network_state").Build()
	value := CacheValueFalse
	if isOnline {
		value = CacheValueTrue
	}
	GlobalCache.Set(key, []string{value}, NetworkCacheTTL)
}

// StartCacheCleanup starts a goroutine to periodically clean up expired cache entries
// Uses adaptive cleanup intervals based on cache size for better performance
func StartCacheCleanup() {
	go func() {
		for {
			// Adaptive cleanup interval based on cache size
			interval := calculateCleanupInterval()
			ticker := time.NewTicker(interval)

			<-ticker.C
			GlobalCache.CleanupExpired()
			ticker.Stop()
		}
	}()
}

// calculateCleanupInterval returns appropriate cleanup interval based on cache size
func calculateCleanupInterval() time.Duration {
	size := GlobalCache.Size()

	switch {
	case size == 0:
		// No entries, clean up less frequently
		return 5 * time.Minute
	case size <= 10:
		// Small cache, moderate cleanup
		return 2 * time.Minute
	case size <= 50:
		// Medium cache, regular cleanup
		return 60 * time.Second
	default:
		// Large cache, frequent cleanup
		return 30 * time.Second
	}
}
