//go:build !integration
// +build !integration

package completion

import (
	"fmt"
	"testing"
	"time"

	"github.com/sqve/grove/internal/testutils"
)

func TestCacheEntry_IsExpired(t *testing.T) {
	tests := []struct {
		name     string
		entry    *CacheEntry
		expected bool
	}{
		{
			name: "not expired",
			entry: &CacheEntry{
				Value:     []string{"main"},
				Timestamp: time.Now(),
				TTL:       time.Hour,
			},
			expected: false,
		},
		{
			name: "expired",
			entry: &CacheEntry{
				Value:     []string{"main"},
				Timestamp: time.Now().Add(-2 * time.Hour),
				TTL:       time.Hour,
			},
			expected: true,
		},
		{
			name: "just expired",
			entry: &CacheEntry{
				Value:     []string{"main"},
				Timestamp: time.Now().Add(-time.Hour - time.Millisecond),
				TTL:       time.Hour,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.entry.IsExpired()
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestCompletionCache_GetSet(t *testing.T) {
	cache := NewCompletionCache()

	value, exists := cache.Get("test_key")
	if exists {
		t.Error("expected cache miss")
	}
	if value != nil {
		t.Errorf("expected nil value, got %v", value)
	}

	testValue := []string{"main", "develop"}
	cache.Set("test_key", testValue, time.Hour)

	value, exists = cache.Get("test_key")
	if !exists {
		t.Error("expected cache hit")
	}
	if !equalSlices(value, testValue) {
		t.Errorf("expected %v, got %v", testValue, value)
	}
}

func TestCompletionCache_Expiration(t *testing.T) {
	cache := NewCompletionCache()

	testValue := []string{"main"}
	cache.Set("test_key", testValue, time.Millisecond)

	value, exists := cache.Get("test_key")
	if !exists {
		t.Error("expected cache hit")
	}
	if !equalSlices(value, testValue) {
		t.Errorf("expected %v, got %v", testValue, value)
	}

	time.Sleep(10 * time.Millisecond)

	value, exists = cache.Get("test_key")
	if exists {
		t.Error("expected cache miss after expiration")
	}
	if value != nil {
		t.Errorf("expected nil value after expiration, got %v", value)
	}
}

func TestCompletionCache_Delete(t *testing.T) {
	cache := NewCompletionCache()

	cache.Set("test_key", []string{"main"}, time.Hour)

	_, exists := cache.Get("test_key")
	if !exists {
		t.Error("expected cache hit")
	}

	cache.Delete("test_key")

	_, exists = cache.Get("test_key")
	if exists {
		t.Error("expected cache miss after deletion")
	}
}

func TestCompletionCache_Clear(t *testing.T) {
	cache := NewCompletionCache()

	cache.Set("key1", []string{"value1"}, time.Hour)
	cache.Set("key2", []string{"value2"}, time.Hour)

	if cache.Size() != 2 {
		t.Errorf("expected size 2, got %d", cache.Size())
	}

	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("expected size 0 after clear, got %d", cache.Size())
	}

	_, exists := cache.Get("key1")
	if exists {
		t.Error("expected cache miss after clear")
	}
}

func TestCompletionCache_CleanupExpired(t *testing.T) {
	cache := NewCompletionCache()

	cache.Set("key1", []string{"value1"}, time.Hour)        // Long TTL.
	cache.Set("key2", []string{"value2"}, time.Millisecond) // Short TTL.
	cache.Set("key3", []string{"value3"}, time.Hour)        // Long TTL.

	time.Sleep(10 * time.Millisecond)

	cache.CleanupExpired()

	if cache.Size() != 2 {
		t.Errorf("expected size 2 after cleanup, got %d", cache.Size())
	}

	_, exists := cache.Get("key1")
	if !exists {
		t.Error("expected key1 to still exist")
	}

	_, exists = cache.Get("key2")
	if exists {
		t.Error("expected key2 to be removed")
	}

	_, exists = cache.Get("key3")
	if !exists {
		t.Error("expected key3 to still exist")
	}
}

func TestCacheKeyBuilder(t *testing.T) {
	tests := []struct {
		name     string
		parts    []string
		expected string
	}{
		{
			name:     "empty builder",
			parts:    []string{},
			expected: "",
		},
		{
			name:     "single part",
			parts:    []string{"branches"},
			expected: "branches",
		},
		{
			name:     "multiple parts",
			parts:    []string{"branches", "local", "main"},
			expected: "branches:local:main",
		},
		{
			name:     "with empty parts",
			parts:    []string{"branches", "", "main"},
			expected: "branches::main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewCacheKeyBuilder()
			for _, part := range tt.parts {
				builder.Add(part)
			}
			result := builder.Build()

			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestCacheKeyBuilderChaining(t *testing.T) {
	builder := NewCacheKeyBuilder()
	result := builder.Add("part1").Add("part2").Add("part3").Build()

	expected := "part1:part2:part3"
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestGetSetCachedBranches(t *testing.T) {
	GlobalCache.Clear()

	ctx := NewCompletionContext(testutils.NewMockGitExecutor())

	branches, exists := GetCachedBranches(ctx)
	if exists {
		t.Error("expected cache miss")
	}
	if branches != nil {
		t.Errorf("expected nil branches, got %v", branches)
	}

	testBranches := []string{"main", "develop"}
	SetCachedBranches(ctx, testBranches)

	branches, exists = GetCachedBranches(ctx)
	if !exists {
		t.Error("expected cache hit")
	}
	if !equalSlices(branches, testBranches) {
		t.Errorf("expected %v, got %v", testBranches, branches)
	}
}

func TestGetSetCachedWorktrees(t *testing.T) {
	GlobalCache.Clear()

	ctx := NewCompletionContext(testutils.NewMockGitExecutor())

	worktrees, exists := GetCachedWorktrees(ctx)
	if exists {
		t.Error("expected cache miss")
	}
	if worktrees != nil {
		t.Errorf("expected nil worktrees, got %v", worktrees)
	}

	testWorktrees := []string{"main", "develop"}
	SetCachedWorktrees(ctx, testWorktrees)

	worktrees, exists = GetCachedWorktrees(ctx)
	if !exists {
		t.Error("expected cache hit")
	}
	if !equalSlices(worktrees, testWorktrees) {
		t.Errorf("expected %v, got %v", testWorktrees, worktrees)
	}
}

func TestGetSetCachedRepositoryState(t *testing.T) {
	GlobalCache.Clear()

	isGrove, exists := GetCachedRepositoryState()
	if exists {
		t.Error("expected cache miss")
	}
	if isGrove {
		t.Error("expected false for cache miss")
	}

	SetCachedRepositoryState(true)

	isGrove, exists = GetCachedRepositoryState()
	if !exists {
		t.Error("expected cache hit")
	}
	if !isGrove {
		t.Error("expected true")
	}

	SetCachedRepositoryState(false)

	isGrove, exists = GetCachedRepositoryState()
	if !exists {
		t.Error("expected cache hit")
	}
	if isGrove {
		t.Error("expected false")
	}
}

func TestCompletionCache_Concurrent(t *testing.T) {
	cache := NewCompletionCache()

	done := make(chan bool)

	go func() {
		for i := 0; i < 100; i++ {
			cache.Set("key", []string{"value"}, time.Hour)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			cache.Get("key")
		}
		done <- true
	}()

	<-done
	<-done

	value, exists := cache.Get("key")
	if !exists {
		t.Error("expected cache hit after concurrent access")
	}
	if !equalSlices(value, []string{"value"}) {
		t.Errorf("expected [\"value\"], got %v", value)
	}
}

func TestCompletionCache_Size(t *testing.T) {
	cache := NewCompletionCache()

	if cache.Size() != 0 {
		t.Errorf("expected size 0, got %d", cache.Size())
	}

	cache.Set("key1", []string{"value1"}, time.Hour)
	if cache.Size() != 1 {
		t.Errorf("expected size 1, got %d", cache.Size())
	}

	cache.Set("key2", []string{"value2"}, time.Hour)
	if cache.Size() != 2 {
		t.Errorf("expected size 2, got %d", cache.Size())
	}

	// Update existing entry (should not change size).
	cache.Set("key1", []string{"new_value"}, time.Hour)
	if cache.Size() != 2 {
		t.Errorf("expected size 2 after update, got %d", cache.Size())
	}

	cache.Delete("key1")
	if cache.Size() != 1 {
		t.Errorf("expected size 1 after delete, got %d", cache.Size())
	}
}

func TestCalculateCleanupInterval(t *testing.T) {
	tests := []struct {
		name        string
		cacheSize   int
		expectedMin time.Duration
		expectedMax time.Duration
	}{
		{
			name:        "empty cache",
			cacheSize:   0,
			expectedMin: 5 * time.Minute,
			expectedMax: 5 * time.Minute,
		},
		{
			name:        "small cache",
			cacheSize:   5,
			expectedMin: 2 * time.Minute,
			expectedMax: 2 * time.Minute,
		},
		{
			name:        "medium cache",
			cacheSize:   25,
			expectedMin: 60 * time.Second,
			expectedMax: 60 * time.Second,
		},
		{
			name:        "large cache",
			cacheSize:   100,
			expectedMin: 30 * time.Second,
			expectedMax: 30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			GlobalCache.Clear()
			for i := 0; i < tt.cacheSize; i++ {
				GlobalCache.Set(fmt.Sprintf("key%d", i), []string{"value"}, time.Hour)
			}

			interval := calculateCleanupInterval()

			if interval < tt.expectedMin || interval > tt.expectedMax {
				t.Errorf("expected interval between %v and %v, got %v", tt.expectedMin, tt.expectedMax, interval)
			}
		})
	}
}

func TestGetSetCachedNetworkState(t *testing.T) {
	GlobalCache.Clear()

	isOnline, exists := GetCachedNetworkState()
	if exists {
		t.Error("expected cache miss")
	}
	if isOnline {
		t.Error("expected false for cache miss")
	}

	SetCachedNetworkState(true)

	isOnline, exists = GetCachedNetworkState()
	if !exists {
		t.Error("expected cache hit")
	}
	if !isOnline {
		t.Error("expected true")
	}

	SetCachedNetworkState(false)

	isOnline, exists = GetCachedNetworkState()
	if !exists {
		t.Error("expected cache hit")
	}
	if isOnline {
		t.Error("expected false")
	}
}
