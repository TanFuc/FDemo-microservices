# Cache Package

A comprehensive, reusable Go cache package with support for multiple backend storage systems (Redis, In-Memory) that can be imported and used across all microservices.

## Features

- **Multi-backend Support**: Redis, In-Memory, Multi-layer (L1+L2)
- **Unified Interface**: Consistent API across all backends
- **Type Safety**: Go generics support (Go 1.18+)
- **Flexible Key Generation**: Dynamic cache key builder
- **List Cache Management**: CRUD operations for cached arrays/lists
- **Automatic Cache Invalidation**: CRUD helper with list sync
- **Metrics & Statistics**: Hit/miss ratio, operation tracking
- **Pattern Operations**: Delete by pattern, key scanning

## Project Structure

```
pkg/cache/
├── # Documentation
├── README.md                 # User documentation
├── ARCHITECTURE.md           # Architecture guide
├── cache-package-prompt.md   # Implementation prompt
│
├── # Core Package Files (8 files - clean & organized)
├── cache.go                  # Main interfaces (Cache, BatchCache, etc.)
├── cache_test.go             # Core tests
├── factory.go                # Factory functions
├── types.go                  # Config, errors, options (consolidated)
├── helpers.go                # Serialization & generic helpers (consolidated)
├── keybuilder.go             # Flexible cache key generation
├── crud.go                   # CRUD operations with cache sync
├── metrics.go                # Metrics collection
│
├── # Backend Implementations
├── redis/                    # Redis implementation
│   ├── redis.go
│   ├── cluster.go
│   ├── pipeline.go
│   └── list.go
│
├── memory/                   # In-memory implementation
│   ├── memory.go
│   ├── lru.go
│   └── memory_test.go
│
├── multilayer/               # Multi-layer cache
│   └── multilayer.go
│
└── examples/                 # Usage examples
    ├── basic_usage.go
    ├── redis_example.go
    └── crud_example.go
```

## Installation

```bash
go get microservices/pkg/cache
```

## Quick Start

### Redis Cache

```go
import (
    "context"
    "time"

    "microservices/pkg/cache"
    "microservices/pkg/cache/redis"
)

func main() {
    // Create Redis cache
    c, err := redis.New(&cache.RedisConfig{
        Addr:     "localhost:6379",
        Password: "",
        DB:       0,
        PoolSize: 10,
    }, nil)
    if err != nil {
        log.Fatal(err)
    }
    defer c.Close()

    ctx := context.Background()

    // Set value
    user := map[string]string{"name": "John", "email": "john@example.com"}
    err = c.Set(ctx, "user:123", user, 5*time.Minute)

    // Get value
    var retrieved map[string]string
    err = c.Get(ctx, "user:123", &retrieved)
}
```

### Memory Cache

```go
import (
    "microservices/pkg/cache"
    "microservices/pkg/cache/memory"
)

func main() {
    c, err := memory.New(&cache.MemoryConfig{
        MaxSize:         100 * 1024 * 1024, // 100MB
        DefaultTTL:      5 * time.Minute,
        CleanupInterval: 1 * time.Minute,
        EvictionPolicy:  "lru",
    }, nil)
    if err != nil {
        log.Fatal(err)
    }
    defer c.Close()

    // Use same API as Redis
    c.Set(ctx, "key", value, 5*time.Minute)
}
```

### Multi-layer Cache (L1 Memory + L2 Redis)

```go
import (
    "microservices/pkg/cache/memory"
    "microservices/pkg/cache/multilayer"
    "microservices/pkg/cache/redis"
)

func main() {
    // L1: Fast memory cache
    l1, _ := memory.New(&cache.MemoryConfig{
        MaxSize:    50 * 1024 * 1024, // 50MB
        DefaultTTL: 1 * time.Minute,
    }, nil)

    // L2: Redis for persistence
    l2, _ := redis.New(&cache.RedisConfig{
        Addr: "localhost:6379",
    }, nil)

    // Multi-layer cache
    c, _ := multilayer.New(l1, l2, nil)

    // Get checks L1 -> L2, populates L1 on L2 hit
    c.Get(ctx, "key", &result)
}
```

## Key Builder

Generate consistent cache keys using the fluent KeyBuilder API:

```go
kb := cache.NewKeyBuilder(":")

// Item key: "user-service:user:123"
itemKey := kb.ForService("user-service").ForItem("user", 123).String()

// List key: "user-service:user:all"
listKey := kb.ForService("user-service").ForList("user").String()

// Search key: "user-service:user:search:john"
searchKey := kb.ForService("user-service").ForSearch("user", "john").String()

// Session key: "auth-service:session:abc123"
sessionKey := kb.ForService("auth-service").ForSession("abc123").String()
```

Helper functions for common patterns:

```go
cache.ItemKey("catalog", "product", 456)    // "catalog:product:456"
cache.ListKey("catalog", "product")         // "catalog:product:all"
cache.SearchKey("catalog", "product", "q")  // "catalog:product:search:q"
cache.SessionKey("auth", "token123")        // "auth:session:token123"
```

## Type-Safe Generics

```go
// Type-safe Get
user, err := cache.GetTyped[User](ctx, c, "user:123")

// Type-safe Set
err := cache.SetTyped(ctx, c, "user:123", user, 5*time.Minute)

// Remember pattern (cache-aside)
user, err := cache.Remember(ctx, c, "user:123", 5*time.Minute, func() (User, error) {
    return db.GetUser(123) // Only called on cache miss
})

// Remember for lists
users, err := cache.RememberList(ctx, c, "users:all", 10*time.Minute, func() ([]User, error) {
    return db.GetAllUsers()
})
```

## CRUD Helper with List Sync

Automatically synchronize individual item cache with list cache:

```go
import "microservices/pkg/cache"

// Create CRUD cache wrapper
crud := cache.NewCRUDCache(redisCache, cache.NewKeyBuilder(":"), "product-service")

// Create with list sync - stores item AND appends to list cache
err := crud.CreateWithListSync(ctx, "product", product.ID, product, 30*time.Minute)
// Keys updated: "product-service:product:123" AND "product-service:product:all"

// Update with list sync
matchFn := func(item interface{}) bool {
    if p, ok := item.(Product); ok {
        return p.ID == product.ID
    }
    return false
}
err := crud.UpdateWithListSync(ctx, "product", product.ID, product, matchFn, 30*time.Minute)

// Delete with list sync
err := crud.DeleteWithListSync(ctx, "product", product.ID, matchFn, 30*time.Minute)
```

## Typed CRUD Helper

```go
// Create typed CRUD cache for a specific entity
productCache := cache.NewCRUDCacheTyped[Product](
    redisCache,
    "product-service",
    "product",
    30*time.Minute,
)

// Type-safe operations
err := productCache.Create(ctx, product.ID, product)
product, err := productCache.Read(ctx, 123)
products, err := productCache.List(ctx)
err := productCache.Delete(ctx, 123)
```

## Batch Operations

```go
// Batch set
items := map[string]interface{}{
    "user:1": user1,
    "user:2": user2,
    "user:3": user3,
}
err := c.(cache.BatchCache).MSet(ctx, items, 5*time.Minute)

// Batch get
results, err := c.(cache.BatchCache).MGet(ctx, []string{"user:1", "user:2", "user:3"})

// Batch delete
deleted, err := c.(cache.BatchCache).MDelete(ctx, []string{"user:1", "user:2"})
```

## Pattern Operations (Redis)

```go
// Delete all user caches
deleted, err := c.(cache.PatternCache).DeleteByPattern(ctx, "user-service:user:*")

// Get all keys matching pattern
keys, err := c.(cache.PatternCache).Keys(ctx, "user-service:session:*")

// Scan with cursor (for large datasets)
keys, cursor, err := c.(cache.PatternCache).Scan(ctx, "user:*", 0, 100)
```

## List Cache Operations

```go
// Store list
users := []User{{ID: 1}, {ID: 2}}
c.(cache.ListCache).SetList(ctx, "users:all", users, 10*time.Minute)

// Append to list
c.(cache.ListCache).AppendToList(ctx, "users:all", newUser, 10*time.Minute)

// Remove from list
c.(cache.ListCache).RemoveFromList(ctx, "users:all", func(item interface{}) bool {
    if u, ok := item.(map[string]interface{}); ok {
        return u["id"] == 123
    }
    return false
}, 10*time.Minute)

// Update in list
c.(cache.ListCache).UpdateInList(ctx, "users:all",
    func(item interface{}) bool { /* match predicate */ },
    func(item interface{}) interface{} { /* return updated */ },
    10*time.Minute,
)
```

## Metrics

```go
// Enable metrics
c, _ := redis.New(cfg, &redis.Options{Metrics: true})

// Get statistics
stats := c.(cache.MetricsCache).Stats()
fmt.Printf("Hit Rate: %.2f%%\n", stats.HitRate*100)
fmt.Printf("Hits: %d, Misses: %d\n", stats.Hits, stats.Misses)
fmt.Printf("Avg Get Latency: %v\n", stats.AvgGetLatency)

// Reset statistics
c.(cache.MetricsCache).ResetStats()
```

## Error Handling

```go
err := c.Get(ctx, "key", &result)

// Check for cache miss
if cache.IsNotFound(err) {
    // Key not in cache - fetch from database
}

// Check for connection errors
if cache.IsConnectionError(err) {
    // Redis connection issue - use fallback
}

// Check for serialization errors
if cache.IsSerializationError(err) {
    // Data format issue
}
```

## Configuration

### Redis Configuration

```go
&cache.RedisConfig{
    Addr:         "localhost:6379",  // Redis address
    Password:     "",                // Password (optional)
    DB:           0,                 // Database number
    PoolSize:     10,                // Connection pool size
    MinIdleConns: 2,                 // Minimum idle connections
    DialTimeout:  5 * time.Second,   // Connection timeout
    ReadTimeout:  3 * time.Second,   // Read timeout
    WriteTimeout: 3 * time.Second,   // Write timeout
    MaxRetries:   3,                 // Retry count
    Cluster:      false,             // Enable cluster mode
    ClusterAddrs: []string{},        // Cluster addresses
    KeyPrefix:    "",                // Key prefix
}
```

### Memory Configuration

```go
&cache.MemoryConfig{
    MaxSize:         100 * 1024 * 1024, // Max memory (bytes)
    DefaultTTL:      5 * time.Minute,   // Default TTL
    CleanupInterval: 1 * time.Minute,   // Cleanup interval
    EvictionPolicy:  "lru",             // Eviction policy
    MaxItems:        10000,             // Max items
}
```

## Interfaces

| Interface | Description |
|-----------|-------------|
| `Cache` | Core interface: Get, Set, Delete, Exists, Close, Ping |
| `BatchCache` | Batch operations: MGet, MSet, MDelete |
| `PatternCache` | Pattern operations: DeleteByPattern, Keys, Scan |
| `ListCache` | List operations: GetList, SetList, AppendToList, etc. |
| `MetricsCache` | Statistics: Stats, ResetStats |
| `CRUDCache` | CRUD with list sync: Create, Read, Update, Delete, List |

## Best Practices

### Key Naming
- Use consistent separators (default: `:`)
- Pattern: `{service}:{entity}:{id}`
- Lists: `{service}:{entity}:all`
- Use KeyBuilder for consistency

### TTL Strategy
- Short (1-5 min): Frequently changing data
- Medium (10-30 min): Relatively stable data
- Long (1-24 hours): Rarely changing data

### Error Handling
- Always check for cache miss vs actual errors
- Implement fallback to database on cache failures
- Log cache errors but don't fail the operation

### Performance
- Use batch operations for multiple keys
- Enable multi-layer cache for hot data
- Monitor hit/miss ratios

## License

MIT
