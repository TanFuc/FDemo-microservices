package memory

import (
	"context"
	"testing"
	"time"

	"microservices/pkg/cache"
)

func TestMemoryCache_BasicOperations(t *testing.T) {
	c, err := New(nil, nil)
	if err != nil {
		t.Fatalf("Failed to create memory cache: %v", err)
	}
	defer c.Close()

	ctx := context.Background()

	// Test Set and Get
	t.Run("Set and Get", func(t *testing.T) {
		data := map[string]string{"name": "test", "value": "123"}
		err := c.Set(ctx, "test-key", data, time.Minute)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		var result map[string]string
		err = c.Get(ctx, "test-key", &result)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if result["name"] != "test" || result["value"] != "123" {
			t.Errorf("Got unexpected result: %v", result)
		}
	})

	// Test cache miss
	t.Run("Cache Miss", func(t *testing.T) {
		var result string
		err := c.Get(ctx, "nonexistent", &result)
		if err == nil {
			t.Error("Expected error for nonexistent key")
		}
		if !cache.IsNotFound(err) {
			t.Errorf("Expected cache miss error, got: %v", err)
		}
	})

	// Test Exists
	t.Run("Exists", func(t *testing.T) {
		c.Set(ctx, "exists-key", "value", time.Minute)

		exists, err := c.Exists(ctx, "exists-key")
		if err != nil {
			t.Fatalf("Exists failed: %v", err)
		}
		if !exists {
			t.Error("Expected key to exist")
		}

		exists, err = c.Exists(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("Exists failed: %v", err)
		}
		if exists {
			t.Error("Expected key to not exist")
		}
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		c.Set(ctx, "delete-key", "value", time.Minute)

		err := c.Delete(ctx, "delete-key")
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		exists, _ := c.Exists(ctx, "delete-key")
		if exists {
			t.Error("Key should not exist after delete")
		}
	})
}

func TestMemoryCache_TTL(t *testing.T) {
	c, _ := New(&cache.MemoryConfig{
		CleanupInterval: 50 * time.Millisecond,
	}, nil)
	defer c.Close()

	ctx := context.Background()

	// Set with short TTL
	err := c.Set(ctx, "ttl-key", "value", 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Should exist immediately
	var result string
	err = c.Get(ctx, "ttl-key", &result)
	if err != nil {
		t.Fatalf("Get failed immediately after set: %v", err)
	}

	// Wait for expiration
	time.Sleep(200 * time.Millisecond)

	// Should be expired now
	err = c.Get(ctx, "ttl-key", &result)
	if !cache.IsNotFound(err) {
		t.Errorf("Expected cache miss after TTL, got: %v", err)
	}
}

func TestMemoryCache_BatchOperations(t *testing.T) {
	c, _ := New(nil, nil)
	defer c.Close()

	ctx := context.Background()

	// Test MSet
	t.Run("MSet", func(t *testing.T) {
		items := map[string]interface{}{
			"batch:1": "value1",
			"batch:2": "value2",
			"batch:3": "value3",
		}

		err := c.MSet(ctx, items, time.Minute)
		if err != nil {
			t.Fatalf("MSet failed: %v", err)
		}
	})

	// Test MGet
	t.Run("MGet", func(t *testing.T) {
		results, err := c.MGet(ctx, []string{"batch:1", "batch:2", "batch:999"})
		if err != nil {
			t.Fatalf("MGet failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}

		if _, ok := results["batch:1"]; !ok {
			t.Error("Expected batch:1 in results")
		}
		if _, ok := results["batch:2"]; !ok {
			t.Error("Expected batch:2 in results")
		}
	})

	// Test MDelete
	t.Run("MDelete", func(t *testing.T) {
		deleted, err := c.MDelete(ctx, []string{"batch:1", "batch:2", "batch:999"})
		if err != nil {
			t.Fatalf("MDelete failed: %v", err)
		}

		if deleted != 2 {
			t.Errorf("Expected 2 deleted, got %d", deleted)
		}
	})
}

func TestMemoryCache_ListOperations(t *testing.T) {
	c, _ := New(nil, nil)
	defer c.Close()

	ctx := context.Background()

	type Item struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	// Test SetList and GetList
	t.Run("SetList and GetList", func(t *testing.T) {
		items := []Item{
			{ID: 1, Name: "Item 1"},
			{ID: 2, Name: "Item 2"},
		}

		err := c.SetList(ctx, "items:all", items, time.Minute)
		if err != nil {
			t.Fatalf("SetList failed: %v", err)
		}

		var result []Item
		err = c.GetList(ctx, "items:all", &result)
		if err != nil {
			t.Fatalf("GetList failed: %v", err)
		}

		if len(result) != 2 {
			t.Errorf("Expected 2 items, got %d", len(result))
		}
	})

	// Test AppendToList
	t.Run("AppendToList", func(t *testing.T) {
		newItem := Item{ID: 3, Name: "Item 3"}
		err := c.AppendToList(ctx, "items:all", newItem, time.Minute)
		if err != nil {
			t.Fatalf("AppendToList failed: %v", err)
		}

		size, _ := c.GetListSize(ctx, "items:all")
		if size != 3 {
			t.Errorf("Expected list size 3, got %d", size)
		}
	})

	// Test PrependToList
	t.Run("PrependToList", func(t *testing.T) {
		newItem := Item{ID: 0, Name: "Item 0"}
		err := c.PrependToList(ctx, "items:all", newItem, time.Minute)
		if err != nil {
			t.Fatalf("PrependToList failed: %v", err)
		}

		size, _ := c.GetListSize(ctx, "items:all")
		if size != 4 {
			t.Errorf("Expected list size 4, got %d", size)
		}
	})

	// Test RemoveFromList
	t.Run("RemoveFromList", func(t *testing.T) {
		err := c.RemoveFromList(ctx, "items:all", func(item interface{}) bool {
			if m, ok := item.(map[string]interface{}); ok {
				return m["id"] == float64(2)
			}
			return false
		}, time.Minute)
		if err != nil {
			t.Fatalf("RemoveFromList failed: %v", err)
		}

		size, _ := c.GetListSize(ctx, "items:all")
		if size != 3 {
			t.Errorf("Expected list size 3 after removal, got %d", size)
		}
	})

	// Test UpdateInList
	t.Run("UpdateInList", func(t *testing.T) {
		err := c.UpdateInList(ctx, "items:all",
			func(item interface{}) bool {
				if m, ok := item.(map[string]interface{}); ok {
					return m["id"] == float64(1)
				}
				return false
			},
			func(item interface{}) interface{} {
				if m, ok := item.(map[string]interface{}); ok {
					m["name"] = "Updated Item 1"
					return m
				}
				return item
			},
			time.Minute,
		)
		if err != nil {
			t.Fatalf("UpdateInList failed: %v", err)
		}
	})

	// Test GetListSize for non-existent list
	t.Run("GetListSize NonExistent", func(t *testing.T) {
		size, err := c.GetListSize(ctx, "nonexistent:list")
		if err != nil {
			t.Fatalf("GetListSize failed: %v", err)
		}
		if size != 0 {
			t.Errorf("Expected size 0 for non-existent list, got %d", size)
		}
	})
}

func TestMemoryCache_Stats(t *testing.T) {
	c, _ := New(nil, &Options{Metrics: true})
	defer c.Close()

	ctx := context.Background()

	// Perform operations
	c.Set(ctx, "key1", "value1", time.Minute)
	c.Set(ctx, "key2", "value2", time.Minute)

	var v string
	c.Get(ctx, "key1", &v) // Hit
	c.Get(ctx, "key2", &v) // Hit
	c.Get(ctx, "key3", &v) // Miss

	c.Delete(ctx, "key1")

	// Check stats
	stats := c.Stats()

	if stats.Hits != 2 {
		t.Errorf("Expected 2 hits, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}
	if stats.Sets != 2 {
		t.Errorf("Expected 2 sets, got %d", stats.Sets)
	}
	if stats.Deletes != 1 {
		t.Errorf("Expected 1 delete, got %d", stats.Deletes)
	}
	if stats.Size != 1 {
		t.Errorf("Expected size 1, got %d", stats.Size)
	}
}

func TestMemoryCache_Eviction(t *testing.T) {
	// Create cache with small capacity
	c, _ := New(&cache.MemoryConfig{
		MaxItems: 3,
	}, nil)
	defer c.Close()

	ctx := context.Background()

	// Add items beyond capacity
	for i := 0; i < 5; i++ {
		c.Set(ctx, string(rune('a'+i)), i, time.Minute)
	}

	// Should have at most 3 items
	size := c.Size()
	if size > 3 {
		t.Errorf("Expected at most 3 items after eviction, got %d", size)
	}
}

func TestMemoryCache_Concurrent(t *testing.T) {
	c, _ := New(nil, nil)
	defer c.Close()

	ctx := context.Background()

	// Concurrent writes
	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func(n int) {
			c.Set(ctx, "concurrent-key", n, time.Minute)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	// Should have a value (any of them)
	var result int
	err := c.Get(ctx, "concurrent-key", &result)
	if err != nil {
		t.Fatalf("Get failed after concurrent writes: %v", err)
	}
}

func TestMemoryCache_Clear(t *testing.T) {
	c, _ := New(nil, nil)
	defer c.Close()

	ctx := context.Background()

	// Add some items
	c.Set(ctx, "key1", "value1", time.Minute)
	c.Set(ctx, "key2", "value2", time.Minute)
	c.Set(ctx, "key3", "value3", time.Minute)

	if c.Size() != 3 {
		t.Errorf("Expected 3 items, got %d", c.Size())
	}

	// Clear
	c.Clear()

	if c.Size() != 0 {
		t.Errorf("Expected 0 items after clear, got %d", c.Size())
	}
}

func TestMemoryCache_ClosedState(t *testing.T) {
	c, _ := New(nil, nil)
	c.Close()

	ctx := context.Background()

	// Operations should fail on closed cache
	err := c.Set(ctx, "key", "value", time.Minute)
	if err != cache.ErrClosed {
		t.Errorf("Expected ErrClosed, got: %v", err)
	}

	var result string
	err = c.Get(ctx, "key", &result)
	if err != cache.ErrClosed {
		t.Errorf("Expected ErrClosed, got: %v", err)
	}

	err = c.Delete(ctx, "key")
	if err != cache.ErrClosed {
		t.Errorf("Expected ErrClosed, got: %v", err)
	}

	_, err = c.Exists(ctx, "key")
	if err != cache.ErrClosed {
		t.Errorf("Expected ErrClosed, got: %v", err)
	}

	err = c.Ping(ctx)
	if err != cache.ErrClosed {
		t.Errorf("Expected ErrClosed, got: %v", err)
	}
}

func BenchmarkMemoryCache_Set(b *testing.B) {
	c, _ := New(nil, nil)
	defer c.Close()

	ctx := context.Background()
	data := map[string]string{"name": "test", "value": "benchmark"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Set(ctx, "benchmark-key", data, time.Minute)
	}
}

func BenchmarkMemoryCache_Get(b *testing.B) {
	c, _ := New(nil, nil)
	defer c.Close()

	ctx := context.Background()
	data := map[string]string{"name": "test", "value": "benchmark"}
	c.Set(ctx, "benchmark-key", data, time.Minute)

	var result map[string]string

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get(ctx, "benchmark-key", &result)
	}
}

func BenchmarkMemoryCache_Concurrent(b *testing.B) {
	c, _ := New(nil, nil)
	defer c.Close()

	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Set(ctx, "concurrent-key", "value", time.Minute)
			var result string
			c.Get(ctx, "concurrent-key", &result)
		}
	})
}
