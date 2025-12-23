package cache

import (
	"testing"
	"time"
)

func TestTTLCacheSetGet(t *testing.T) {
	cache := NewTTLCache[string, int](2, time.Second)
	cache.Set("a", 1)

	value, ok := cache.Get("a")
	if !ok {
		t.Fatalf("expected value")
	}
	if value != 1 {
		t.Fatalf("expected 1, got %d", value)
	}
}

func TestTTLCacheEvictsOldest(t *testing.T) {
	cache := NewTTLCache[string, int](2, time.Second)
	cache.Set("a", 1)
	cache.Set("b", 2)
	cache.Set("c", 3)

	if _, ok := cache.Get("a"); ok {
		t.Fatalf("expected key 'a' to be evicted")
	}
	if value, ok := cache.Get("b"); !ok || value != 2 {
		t.Fatalf("expected key 'b' to remain")
	}
	if value, ok := cache.Get("c"); !ok || value != 3 {
		t.Fatalf("expected key 'c' to remain")
	}
}

func TestTTLCacheExpires(t *testing.T) {
	cache := NewTTLCache[string, int](2, 20*time.Millisecond)
	cache.Set("a", 1)
	time.Sleep(50 * time.Millisecond)

	if _, ok := cache.Get("a"); ok {
		t.Fatalf("expected key 'a' to expire")
	}
}
