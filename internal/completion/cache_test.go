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

	// Test cache miss
	value, exists := cache.Get("test_key")
	if exists {
		t.Error("expected cache miss")
	}
	if value != nil {
		t.Errorf("expected nil value, got %v", value)
	}

	// Test cache set and hit
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

	// Set a value with very short TTL
	testValue := []string{"main"}
	cache.Set("test_key", testValue, time.Millisecond)

	// Should be available immediately
	value, exists := cache.Get("test_key")
	if !exists {
		t.Error("expected cache hit")
	}
	if !equalSlices(value, testValue) {
		t.Errorf("expected %v, got %v", testValue, value)
	}

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Should be expired now
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

	// Set a value
	cache.Set("test_key", []string{"main"}, time.Hour)

	// Verify it exists
	_, exists := cache.Get("test_key")
	if !exists {
		t.Error("expected cache hit")
	}

	// Delete the value
	cache.Delete("test_key")

	// Verify it's gone
	_, exists = cache.Get("test_key")
	if exists {
		t.Error("expected cache miss after deletion")
	}
}

func TestCompletionCache_Clear(t *testing.T) {
	cache := NewCompletionCache()

	// Set multiple values
	cache.Set("key1", []string{"value1"}, time.Hour)
	cache.Set("key2", []string{"value2"}, time.Hour)

	// Verify size
	if cache.Size() != 2 {
		t.Errorf("expected size 2, got %d", cache.Size())
	}

	// Clear cache
	cache.Clear()

	// Verify empty
	if cache.Size() != 0 {
		t.Errorf("expected size 0 after clear, got %d", cache.Size())
	}

	// Verify values are gone
	_, exists := cache.Get("key1")
	if exists {
		t.Error("expected cache miss after clear")
	}
}

func TestCompletionCache_CleanupExpired(t *testing.T) {
	cache := NewCompletionCache()

	// Set values with different TTLs
	cache.Set("key1", []string{"value1"}, time.Hour)        // Long TTL
	cache.Set("key2", []string{"value2"}, time.Millisecond) // Short TTL
	cache.Set("key3", []string{"value3"}, time.Hour)        // Long TTL

	// Wait for short TTL to expire
	time.Sleep(10 * time.Millisecond)

	// Cleanup expired entries
	cache.CleanupExpired()

	// Verify only expired entry was removed
	if cache.Size() != 2 {
		t.Errorf("expected size 2 after cleanup, got %d", cache.Size())
	}

	// Verify specific entries
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
	// Test that Add() returns the builder for chaining
	builder := NewCacheKeyBuilder()
	result := builder.Add("part1").Add("part2").Add("part3").Build()

	expected := "part1:part2:part3"
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestGetSetCachedBranches(t *testing.T) {
	// Clear cache before test
	GlobalCache.Clear()

	ctx := NewCompletionContext(testutils.NewMockGitExecutor())

	// Test cache miss
	branches, exists := GetCachedBranches(ctx)
	if exists {
		t.Error("expected cache miss")
	}
	if branches != nil {
		t.Errorf("expected nil branches, got %v", branches)
	}

	// Test cache set and hit
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
	// Clear cache before test
	GlobalCache.Clear()

	ctx := NewCompletionContext(testutils.NewMockGitExecutor())

	// Test cache miss
	worktrees, exists := GetCachedWorktrees(ctx)
	if exists {
		t.Error("expected cache miss")
	}
	if worktrees != nil {
		t.Errorf("expected nil worktrees, got %v", worktrees)
	}

	// Test cache set and hit
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
	// Clear cache before test
	GlobalCache.Clear()

	// Test cache miss
	isGrove, exists := GetCachedRepositoryState()
	if exists {
		t.Error("expected cache miss")
	}
	if isGrove {
		t.Error("expected false for cache miss")
	}

	// Test cache set and hit - true value
	SetCachedRepositoryState(true)

	isGrove, exists = GetCachedRepositoryState()
	if !exists {
		t.Error("expected cache hit")
	}
	if !isGrove {
		t.Error("expected true")
	}

	// Test cache set and hit - false value
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

	// Test concurrent access
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			cache.Set("key", []string{"value"}, time.Hour)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			cache.Get("key")
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Verify cache is in a consistent state
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

	// Start with empty cache
	if cache.Size() != 0 {
		t.Errorf("expected size 0, got %d", cache.Size())
	}

	// Add entries
	cache.Set("key1", []string{"value1"}, time.Hour)
	if cache.Size() != 1 {
		t.Errorf("expected size 1, got %d", cache.Size())
	}

	cache.Set("key2", []string{"value2"}, time.Hour)
	if cache.Size() != 2 {
		t.Errorf("expected size 2, got %d", cache.Size())
	}

	// Update existing entry (should not change size)
	cache.Set("key1", []string{"new_value"}, time.Hour)
	if cache.Size() != 2 {
		t.Errorf("expected size 2 after update, got %d", cache.Size())
	}

	// Delete entry
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
			// Clear and populate cache with test size
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
	// Clear cache before test
	GlobalCache.Clear()

	// Test cache miss
	isOnline, exists := GetCachedNetworkState()
	if exists {
		t.Error("expected cache miss")
	}
	if isOnline {
		t.Error("expected false for cache miss")
	}

	// Test cache set and hit - true value
	SetCachedNetworkState(true)

	isOnline, exists = GetCachedNetworkState()
	if !exists {
		t.Error("expected cache hit")
	}
	if !isOnline {
		t.Error("expected true")
	}

	// Test cache set and hit - false value
	SetCachedNetworkState(false)

	isOnline, exists = GetCachedNetworkState()
	if !exists {
		t.Error("expected cache hit")
	}
	if isOnline {
		t.Error("expected false")
	}
}
