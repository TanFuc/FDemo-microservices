package cache

import (
	"context"
	"testing"
	"time"
)

func TestErrors(t *testing.T) {
	// Test IsNotFound
	if !IsNotFound(ErrCacheMiss) {
		t.Error("IsNotFound should return true for ErrCacheMiss")
	}

	if !IsNotFound(ErrListNotFound) {
		t.Error("IsNotFound should return true for ErrListNotFound")
	}

	if !IsNotFound(ErrItemNotFound) {
		t.Error("IsNotFound should return true for ErrItemNotFound")
	}

	if IsNotFound(ErrConnection) {
		t.Error("IsNotFound should return false for ErrConnection")
	}

	// Test IsConnectionError
	if !IsConnectionError(ErrConnection) {
		t.Error("IsConnectionError should return true for ErrConnection")
	}

	if !IsConnectionError(ErrClosed) {
		t.Error("IsConnectionError should return true for ErrClosed")
	}

	if !IsConnectionError(ErrTimeout) {
		t.Error("IsConnectionError should return true for ErrTimeout")
	}

	// Test IsSerializationError
	if !IsSerializationError(ErrSerialization) {
		t.Error("IsSerializationError should return true for ErrSerialization")
	}

	if !IsSerializationError(ErrDeserialization) {
		t.Error("IsSerializationError should return true for ErrDeserialization")
	}
}

func TestCacheError(t *testing.T) {
	err := WrapError("Get", "test-key", ErrCacheMiss)
	if err == nil {
		t.Fatal("WrapError should return an error")
	}

	cacheErr, ok := err.(*CacheError)
	if !ok {
		t.Fatal("WrapError should return a *CacheError")
	}

	if cacheErr.Op != "Get" {
		t.Errorf("Expected Op 'Get', got '%s'", cacheErr.Op)
	}

	if cacheErr.Key != "test-key" {
		t.Errorf("Expected Key 'test-key', got '%s'", cacheErr.Key)
	}

	expectedMsg := "cache Get [key=test-key]: cache: key not found"
	if cacheErr.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, cacheErr.Error())
	}

	// Test nil error
	if WrapError("Get", "key", nil) != nil {
		t.Error("WrapError should return nil for nil error")
	}
}

func TestSerializer(t *testing.T) {
	serializer := NewJSONSerializer()

	// Test Marshal
	data := map[string]string{"name": "test", "value": "123"}
	bytes, err := serializer.Marshal(data)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Test Unmarshal
	var result map[string]string
	err = serializer.Unmarshal(bytes, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if result["name"] != "test" || result["value"] != "123" {
		t.Errorf("Unmarshal returned incorrect data: %v", result)
	}

	// Test ContentType
	if serializer.ContentType() != "application/json" {
		t.Errorf("Expected content type 'application/json', got '%s'", serializer.ContentType())
	}
}

func TestNewSerializer(t *testing.T) {
	// Test JSON serializer
	jsonSerializer := NewSerializer(SerializerJSON)
	if jsonSerializer == nil {
		t.Error("NewSerializer(SerializerJSON) should not return nil")
	}

	// Test default serializer
	defaultSerializer := DefaultSerializer()
	if defaultSerializer == nil {
		t.Error("DefaultSerializer() should not return nil")
	}
}

func TestKeyBuilder(t *testing.T) {
	kb := NewKeyBuilder(":")

	// Test ForService + ForItem
	key := kb.ForService("user-service").ForItem("user", 123).String()
	expected := "user-service:user:123"
	if key != expected {
		t.Errorf("Expected '%s', got '%s'", expected, key)
	}

	// Test ForService + ForList
	kb2 := NewKeyBuilder(":")
	listKey := kb2.ForService("user-service").ForList("user").String()
	expectedList := "user-service:user:all"
	if listKey != expectedList {
		t.Errorf("Expected '%s', got '%s'", expectedList, listKey)
	}

	// Test ForSearch
	kb3 := NewKeyBuilder(":")
	searchKey := kb3.ForService("user-service").ForSearch("user", "john").String()
	expectedSearch := "user-service:user:search:john"
	if searchKey != expectedSearch {
		t.Errorf("Expected '%s', got '%s'", expectedSearch, searchKey)
	}

	// Test ForSession
	kb4 := NewKeyBuilder(":")
	sessionKey := kb4.ForService("auth-service").ForSession("abc123").String()
	expectedSession := "auth-service:session:abc123"
	if sessionKey != expectedSession {
		t.Errorf("Expected '%s', got '%s'", expectedSession, sessionKey)
	}

	// Test WithPrefix
	kb5 := NewKeyBuilder(":")
	prefixedKey := kb5.WithPrefix("app").ForService("user").ForItem("profile", 1).String()
	expectedPrefixed := "app:user:profile:1"
	if prefixedKey != expectedPrefixed {
		t.Errorf("Expected '%s', got '%s'", expectedPrefixed, prefixedKey)
	}
}

func TestKeyBuilderHelpers(t *testing.T) {
	// Test ItemKey
	itemKey := ItemKey("catalog", "product", 456)
	if itemKey != "catalog:product:456" {
		t.Errorf("Expected 'catalog:product:456', got '%s'", itemKey)
	}

	// Test ListKey
	listKey := ListKey("catalog", "product")
	if listKey != "catalog:product:all" {
		t.Errorf("Expected 'catalog:product:all', got '%s'", listKey)
	}

	// Test SearchKey
	searchKey := SearchKey("catalog", "product", "laptop")
	if searchKey != "catalog:product:search:laptop" {
		t.Errorf("Expected 'catalog:product:search:laptop', got '%s'", searchKey)
	}

	// Test SessionKey
	sessionKey := SessionKey("auth", "token123")
	if sessionKey != "auth:session:token123" {
		t.Errorf("Expected 'auth:session:token123', got '%s'", sessionKey)
	}

	// Test CountKey
	countKey := CountKey("catalog", "product")
	if countKey != "catalog:product:count" {
		t.Errorf("Expected 'catalog:product:count', got '%s'", countKey)
	}

	// Test PatternKey
	patternKey := PatternKey("catalog", "product")
	if patternKey != "catalog:product:*" {
		t.Errorf("Expected 'catalog:product:*', got '%s'", patternKey)
	}
}

func TestConfig(t *testing.T) {
	// Test DefaultConfig
	cfg := DefaultConfig()
	if cfg.Type != CacheTypeRedis {
		t.Errorf("Expected default type to be Redis, got %s", cfg.Type)
	}

	if cfg.KeySeparator != ":" {
		t.Errorf("Expected default separator to be ':', got '%s'", cfg.KeySeparator)
	}

	// Test Validate
	invalidCfg := Config{}
	if err := invalidCfg.Validate(); err == nil {
		t.Error("Validate should return error for empty config")
	}

	// Test valid Redis config
	validRedisCfg := Config{
		Type: CacheTypeRedis,
		Redis: &RedisConfig{
			Addr: "localhost:6379",
		},
	}
	if err := validRedisCfg.Validate(); err != nil {
		t.Errorf("Validate should pass for valid Redis config: %v", err)
	}

	// Test valid Memory config
	validMemoryCfg := Config{
		Type:   CacheTypeMemory,
		Memory: DefaultMemoryConfig(),
	}
	if err := validMemoryCfg.Validate(); err != nil {
		t.Errorf("Validate should pass for valid Memory config: %v", err)
	}
}

func TestRedisConfig(t *testing.T) {
	cfg := DefaultRedisConfig()

	if cfg.Addr != "localhost:6379" {
		t.Errorf("Expected default addr 'localhost:6379', got '%s'", cfg.Addr)
	}

	if cfg.PoolSize != 10 {
		t.Errorf("Expected default pool size 10, got %d", cfg.PoolSize)
	}

	// Test GetAddr
	if cfg.GetAddr() != "localhost:6379" {
		t.Errorf("GetAddr should return 'localhost:6379'")
	}

	emptyAddrCfg := &RedisConfig{}
	if emptyAddrCfg.GetAddr() != "localhost:6379" {
		t.Errorf("GetAddr should return default for empty addr")
	}
}

func TestMemoryConfig(t *testing.T) {
	cfg := DefaultMemoryConfig()

	if cfg.MaxSize != 100*1024*1024 {
		t.Errorf("Expected default max size 100MB, got %d", cfg.MaxSize)
	}

	if cfg.DefaultTTL != 5*time.Minute {
		t.Errorf("Expected default TTL 5 minutes, got %v", cfg.DefaultTTL)
	}

	if cfg.EvictionPolicy != "lru" {
		t.Errorf("Expected default eviction policy 'lru', got '%s'", cfg.EvictionPolicy)
	}
}

func TestMetrics(t *testing.T) {
	m := newMetricsCollector()

	// Test RecordHit
	m.RecordHit(10 * time.Millisecond)
	m.RecordHit(20 * time.Millisecond)

	// Test RecordMiss
	m.RecordMiss(5 * time.Millisecond)

	// Test RecordSet
	m.RecordSet(15 * time.Millisecond)

	// Test RecordDelete
	m.RecordDelete()

	// Test RecordError
	m.RecordError(ErrConnection)

	// Get stats
	stats := m.Stats()

	if stats.Hits != 2 {
		t.Errorf("Expected 2 hits, got %d", stats.Hits)
	}

	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}

	if stats.Sets != 1 {
		t.Errorf("Expected 1 set, got %d", stats.Sets)
	}

	if stats.Deletes != 1 {
		t.Errorf("Expected 1 delete, got %d", stats.Deletes)
	}

	if stats.Errors != 1 {
		t.Errorf("Expected 1 error, got %d", stats.Errors)
	}

	// Test hit rate (2 hits out of 3 total = 0.666...)
	expectedHitRate := float64(2) / float64(3)
	if stats.HitRate != expectedHitRate {
		t.Errorf("Expected hit rate %.4f, got %.4f", expectedHitRate, stats.HitRate)
	}

	// Test Reset
	m.Reset()
	stats = m.Stats()
	if stats.Hits != 0 || stats.Misses != 0 || stats.Sets != 0 {
		t.Error("Reset should clear all stats")
	}
}

func TestNoopMetrics(t *testing.T) {
	m := &noopMetrics{}

	// All operations should be no-ops and not panic
	m.RecordHit(time.Millisecond)
	m.RecordMiss(time.Millisecond)
	m.RecordSet(time.Millisecond)
	m.RecordDelete()
	m.RecordError(ErrConnection)
	m.UpdateSize(100)
	m.UpdateMemoryUsage(1024)
	m.Reset()

	stats := m.Stats()
	if stats.Hits != 0 || stats.Misses != 0 {
		t.Error("Noop metrics should always return zero stats")
	}
}

func TestCacheStats(t *testing.T) {
	stats := CacheStats{
		Hits:        100,
		Misses:      20,
		Sets:        50,
		Deletes:     10,
		HitRate:     0.833,
		Size:        1000,
		MemoryUsage: 1024 * 1024,
	}

	if stats.Hits != 100 {
		t.Errorf("Expected Hits to be 100")
	}

	if stats.HitRate != 0.833 {
		t.Errorf("Expected HitRate to be 0.833")
	}
}

// Benchmark tests
func BenchmarkKeyBuilder(b *testing.B) {
	kb := NewKeyBuilder(":")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = kb.ForService("user-service").ForItem("user", 123).String()
	}
}

func BenchmarkItemKey(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = ItemKey("user-service", "user", 123)
	}
}

func BenchmarkJSONSerializer(b *testing.B) {
	serializer := NewJSONSerializer()
	data := map[string]interface{}{
		"id":    123,
		"name":  "test",
		"email": "test@example.com",
		"tags":  []string{"tag1", "tag2", "tag3"},
	}

	b.Run("Marshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = serializer.Marshal(data)
		}
	})

	bytes, _ := serializer.Marshal(data)
	b.Run("Unmarshal", func(b *testing.B) {
		var result map[string]interface{}
		for i := 0; i < b.N; i++ {
			_ = serializer.Unmarshal(bytes, &result)
		}
	})
}

// Example tests
func ExampleKeyBuilder() {
	kb := NewKeyBuilder(":")

	// Create item key
	itemKey := kb.ForService("user-service").ForItem("user", 123).String()
	_ = itemKey // "user-service:user:123"

	// Create list key
	listKey := kb.ForService("user-service").ForList("user").String()
	_ = listKey // "user-service:user:all"
}

func ExampleRemember() {
	// This is a pseudo-example showing the Remember pattern
	// In real usage, you would have an actual cache implementation

	ctx := context.Background()
	_ = ctx
	// cache, _ := memory.New(nil, nil)

	// user, err := Remember(ctx, cache, "user:123", 5*time.Minute, func() (User, error) {
	//     // This function runs on cache miss
	//     return fetchUserFromDB(123)
	// })
}
