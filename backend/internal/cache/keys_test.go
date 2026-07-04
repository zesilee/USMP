package cache

import (
	"sort"
	"testing"
	"time"
)

// TestKeysReturnsLiveEntries verifies Keys() returns all non-expired keys.
func TestKeysReturnsLiveEntries(t *testing.T) {
	c := NewTTLLRUCache(100, 30*time.Second, 0)
	c.Set("a", 1)
	c.Set("b", 2)
	c.Set("c", 3)

	got := c.Keys()
	sort.Strings(got)
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("Keys() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Keys() = %v, want %v", got, want)
		}
	}
}

// TestKeysEmpty verifies Keys() on an empty cache returns an empty (non-nil-length) slice.
func TestKeysEmpty(t *testing.T) {
	c := NewTTLLRUCache(100, 30*time.Second, 0)
	if got := c.Keys(); len(got) != 0 {
		t.Fatalf("Keys() on empty cache = %v, want empty", got)
	}
}

// TestKeysExcludesExpired verifies expired entries are not returned by Keys().
func TestKeysExcludesExpired(t *testing.T) {
	c := NewTTLLRUCache(100, 20*time.Millisecond, 0)
	c.Set("fresh", 1)
	time.Sleep(30 * time.Millisecond) // let it expire
	c.Set("new", 2)                   // fresh, after expiry window

	got := c.Keys()
	if len(got) != 1 || got[0] != "new" {
		t.Fatalf("Keys() = %v, want only [new] (expired excluded)", got)
	}
}
