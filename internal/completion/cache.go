package completion

import (
	"sync"
	"time"

	"github.com/sqve/grove/internal/logger"
)

// Cache value constants.
const (
	CacheValueTrue  = "true"
	CacheValueFalse = "false"
)

type CacheEntry struct {
	Value     []string
	Timestamp time.Time
	TTL       time.Duration
}

func (c *CacheEntry) IsExpired() bool {
	return time.Since(c.Timestamp) > c.TTL
}

type CompletionCache struct {
	cache map[string]*CacheEntry
	mutex sync.RWMutex
}

func NewCompletionCache() *CompletionCache {
	return &CompletionCache{
		cache: make(map[string]*CacheEntry),
	}
}

func (c *CompletionCache) Get(key string) ([]string, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.cache[key]
	if !exists {
		return nil, false
	}

	if entry.IsExpired() {
		go c.Delete(key)
		return nil, false
	}

	return entry.Value, true
}

func (c *CompletionCache) Set(key string, value []string, ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.cache[key] = &CacheEntry{
		Value:     value,
		Timestamp: time.Now(),
		TTL:       ttl,
	}
}

func (c *CompletionCache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.cache, key)
}

func (c *CompletionCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.cache = make(map[string]*CacheEntry)
}

func (c *CompletionCache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return len(c.cache)
}

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

var GlobalCache = NewCompletionCache()

type CacheKeyBuilder struct {
	parts []string
}

func NewCacheKeyBuilder() *CacheKeyBuilder {
	return &CacheKeyBuilder{
		parts: make([]string, 0),
	}
}

func (b *CacheKeyBuilder) Add(part string) *CacheKeyBuilder {
	b.parts = append(b.parts, part)
	return b
}

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

const (
	BranchCacheTTL     = 30 * time.Second
	WorktreeCacheTTL   = 10 * time.Second
	RepositoryCacheTTL = 5 * time.Second
	URLCacheTTL        = 5 * time.Minute
	NetworkCacheTTL    = 30 * time.Second
)

func GetCachedBranches(ctx *CompletionContext) ([]string, bool) {
	key := NewCacheKeyBuilder().Add("branches").Build()
	return GlobalCache.Get(key)
}

func SetCachedBranches(ctx *CompletionContext, branches []string) {
	key := NewCacheKeyBuilder().Add("branches").Build()
	GlobalCache.Set(key, branches, BranchCacheTTL)
}

func GetCachedWorktrees(ctx *CompletionContext) ([]string, bool) {
	key := NewCacheKeyBuilder().Add("worktrees").Build()
	return GlobalCache.Get(key)
}

func SetCachedWorktrees(ctx *CompletionContext, worktrees []string) {
	key := NewCacheKeyBuilder().Add("worktrees").Build()
	GlobalCache.Set(key, worktrees, WorktreeCacheTTL)
}

func GetCachedRepositoryState() (isGroveRepo, exists bool) {
	key := NewCacheKeyBuilder().Add("repo_state").Build()
	if value, exists := GlobalCache.Get(key); exists && len(value) > 0 {
		return value[0] == CacheValueTrue, true
	}
	return false, false
}

func SetCachedRepositoryState(isGroveRepo bool) {
	key := NewCacheKeyBuilder().Add("repo_state").Build()
	value := CacheValueFalse
	if isGroveRepo {
		value = CacheValueTrue
	}
	GlobalCache.Set(key, []string{value}, RepositoryCacheTTL)
}

func GetCachedNetworkState() (isOnline, exists bool) {
	key := NewCacheKeyBuilder().Add("network_state").Build()
	if value, exists := GlobalCache.Get(key); exists && len(value) > 0 {
		return value[0] == CacheValueTrue, true
	}
	return false, false
}

func SetCachedNetworkState(isOnline bool) {
	key := NewCacheKeyBuilder().Add("network_state").Build()
	value := CacheValueFalse
	if isOnline {
		value = CacheValueTrue
	}
	GlobalCache.Set(key, []string{value}, NetworkCacheTTL)
}

// Uses adaptive cleanup intervals based on cache size for better performance.
func StartCacheCleanup() {
	go func() {
		for {
			// Adaptive cleanup interval based on cache size.
			interval := calculateCleanupInterval()
			ticker := time.NewTicker(interval)

			<-ticker.C
			GlobalCache.CleanupExpired()
			ticker.Stop()
		}
	}()
}

func calculateCleanupInterval() time.Duration {
	size := GlobalCache.Size()

	switch {
	case size == 0:
		// No entries, clean up less frequently.
		return 5 * time.Minute
	case size <= 10:
		// Small cache, moderate cleanup.
		return 2 * time.Minute
	case size <= 50:
		// Medium cache, regular cleanup.
		return 60 * time.Second
	default:
		// Large cache, frequent cleanup.
		return 30 * time.Second
	}
}
