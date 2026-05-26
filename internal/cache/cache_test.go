package cache

import (
	"testing"
	"time"
)

func TestCache_SetAndGet(t *testing.T) {
	cache := New(time.Second)
	defer cache.Close()

	cache.Set("key1", "value1")

	val, ok := cache.Get("key1")
	if !ok {
		t.Fatal("expected to find key1")
	}
	if val != "value1" {
		t.Fatalf("expected value1, got %v", val)
	}
}

func TestCache_Expiration(t *testing.T) {
	cache := New(50 * time.Millisecond)
	defer cache.Close()

	cache.Set("key1", "value1")

	time.Sleep(100 * time.Millisecond)

	_, ok := cache.Get("key1")
	if ok {
		t.Fatal("expected key1 to be expired")
	}
}

func TestCache_Delete(t *testing.T) {
	cache := New(time.Second)
	defer cache.Close()

	cache.Set("key1", "value1")
	cache.Delete("key1")

	_, ok := cache.Get("key1")
	if ok {
		t.Fatal("expected key1 to be deleted")
	}
}

func TestCache_Clear(t *testing.T) {
	cache := New(time.Second)
	defer cache.Close()

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Clear()

	if cache.Size() != 0 {
		t.Fatalf("expected 0 items, got %d", cache.Size())
	}
}

func TestCache_SetWithTTL(t *testing.T) {
	cache := New(time.Second)
	defer cache.Close()

	cache.SetWithTTL("key1", "value1", 50*time.Millisecond)

	time.Sleep(100 * time.Millisecond)

	_, ok := cache.Get("key1")
	if ok {
		t.Fatal("expected key1 to be expired")
	}
}

func TestCache_Size(t *testing.T) {
	cache := New(time.Second)
	defer cache.Close()

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	if cache.Size() != 3 {
		t.Fatalf("expected 3 items, got %d", cache.Size())
	}
}
