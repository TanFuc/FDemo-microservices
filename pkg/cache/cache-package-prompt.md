# Golang Cache Package - Implementation Prompt

## Objective
Build a highly reusable Golang cache package with support for multiple backend storage systems (Redis, In-Memory) that can be imported and used across all microservices in the system. The package must provide flexible cache key generation, comprehensive array/list cache management, and complete CRUD operations with automatic cache invalidation.

## Functional Requirements

### 1. Core Features
- **Multi-backend Support**: Support Redis, In-Memory cache (go-cache, bigcache), extensible architecture
- **Unified Interface**: Provide unified interface for all backends
- **TTL Management**: Support set/get with automatic Time-To-Live
- **Serialization**: Automatic serialize/deserialize for complex data types (struct, map, slice, array)
- **Error Handling**: Clear error handling, distinguish between cache miss and actual errors
- **Context Support**: Support context.Context for timeout and cancellation
- **Type Safety**: Use Generics (Go 1.18+) to ensure type safety
- **Flexible Cache Key Generation**: Dynamic cache key builder supporting multiple patterns and namespaces

### 2. Advanced Features
- **Multi-layer Cache**: Support L1 (memory) + L2 (Redis) cache hierarchy
- **Cache Warming**: Preload cache mechanism on startup
- **Batch Operations**: Get/Set/Delete multiple keys simultaneously
- **Cache Invalidation**: Pattern-based invalidation (delete by prefix/pattern)
- **Array/List Cache Management**: Comprehensive CRUD operations for cached arrays/lists
- **Statistics & Monitoring**: Hit/Miss ratio, operation metrics
- **Compression**: Optional compression for large values
- **Cache Aside Pattern**: Helper functions for cache-aside pattern with automatic array synchronization
- **Tag-based Invalidation**: Group-based cache invalidation using tags

### 3. Array/List Cache Management (Critical Feature)
- **Create Operations**: Push new data to cached array, update Redis atomically
- **Read Operations**: Get single item or entire list from cache
- **Update Operations**: Update item in cached array, re-serialize and push to Redis
- **Delete Operations**: Remove item from cached array, update Redis with new array
- **Atomic Operations**: Ensure data consistency during CRUD operations
- **Cache Synchronization**: Keep individual item cache and list cache in sync

### 4. Performance Requirements
- **Connection Pooling**: Efficient connection pool management for Redis
- **Pipelining**: Support Redis pipelining for batch operations
- **Non-blocking**: Async operations when needed
- **Memory Efficient**: Memory limits for in-memory cache
- **Optimized Serialization**: Fast JSON/MessagePack serialization

## Directory Structure

```
pkg/cache/
├── go.mod                    # Go module definition
├── go.sum                    # Dependencies checksum
├── README.md                 # User documentation
├── cache-package-prompt.md   # This implementation guide
│
├── # Core Files (Main Package)
├── cache.go                  # Main interfaces (Cache, BatchCache, PatternCache, etc.)
├── cache_test.go             # Core interface tests
├── factory.go                # Factory functions (New, NewRedis, NewMemory, etc.)
│
├── # Configuration & Types
├── config.go                 # Configuration structures (Config, RedisConfig, MemoryConfig)
├── errors.go                 # Custom errors (ErrCacheMiss, ErrInvalidValue, etc.)
├── options.go                # Functional options pattern
│
├── # Utilities & Helpers
├── keybuilder.go             # Cache key builder with fluent API
├── crud.go                   # CRUD operations with automatic cache sync
├── generics.go               # Generic helper functions (GetTyped, SetTyped, etc.)
├── serializer.go             # Serialization utilities (JSON, MessagePack)
├── metrics.go                # Metrics collection and statistics
│
├── # Backend Implementations
├── redis/
│   ├── redis.go              # Redis cache implementation
│   ├── cluster.go            # Redis cluster support
│   ├── pipeline.go           # Redis pipelining operations
│   └── list.go               # Redis list cache operations
│
├── memory/
│   ├── memory.go             # In-memory cache implementation
│   ├── lru.go                # LRU eviction policy
│   └── memory_test.go        # Memory cache tests
│
├── multilayer/
│   └── multilayer.go         # Multi-layer cache (L1 Memory + L2 Redis)
│
└── examples/
    ├── basic_usage.go        # Basic usage examples
    ├── redis_example.go      # Redis-specific examples
    └── crud_example.go       # CRUD operations examples
```

### File Organization Philosophy

- **Flat structure at root**: All core package files are at the root level for easy access and import
- **Grouped by function**: Files are named to indicate their purpose (config, errors, crud, etc.)
- **Sub-packages for implementations**: Each backend (redis, memory, multilayer) has its own package
- **Clear separation**: Interface definitions, configurations, implementations, and examples are clearly separated

## Interface Design

### Core Interface
```go
package cache

import (
    "context"
    "time"
)

// Cache defines the main interface for all cache implementations
type Cache interface {
    // Get retrieves a value from cache
    Get(ctx context.Context, key string, dest interface{}) error
    
    // Set stores a value in cache with TTL
    Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
    
    // Delete removes a key from cache
    Delete(ctx context.Context, key string) error
    
    // Exists checks if a key exists in cache
    Exists(ctx context.Context, key string) (bool, error)
    
    // Close closes the cache connection
    Close() error
}

// BatchCache interface for batch operations
type BatchCache interface {
    Cache
    
    // MGet retrieves multiple keys at once
    MGet(ctx context.Context, keys []string) (map[string]interface{}, error)
    
    // MSet sets multiple key-value pairs
    MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error
    
    // MDelete deletes multiple keys
    MDelete(ctx context.Context, keys []string) error
}

// PatternCache interface for pattern operations
type PatternCache interface {
    Cache
    
    // DeleteByPattern deletes keys matching pattern (e.g., "user:*")
    DeleteByPattern(ctx context.Context, pattern string) error
    
    // Keys returns list of keys matching pattern
    Keys(ctx context.Context, pattern string) ([]string, error)
}

// MetricsCache interface for metrics collection
type MetricsCache interface {
    Cache
    
    // Stats returns cache statistics
    Stats() CacheStats
    
    // ResetStats resets statistics
    ResetStats()
}

// ListCache interface for array/list cache management
type ListCache interface {
    Cache
    
    // GetList retrieves entire list from cache
    GetList(ctx context.Context, key string, dest interface{}) error
    
    // SetList stores entire list in cache
    SetList(ctx context.Context, key string, list interface{}, ttl time.Duration) error
    
    // AppendToList adds item to cached list
    AppendToList(ctx context.Context, key string, item interface{}, ttl time.Duration) error
    
    // RemoveFromList removes item from cached list by predicate
    RemoveFromList(ctx context.Context, key string, predicate func(item interface{}) bool, ttl time.Duration) error
    
    // UpdateInList updates item in cached list by predicate
    UpdateInList(ctx context.Context, key string, predicate func(item interface{}) bool, updater func(item interface{}) interface{}, ttl time.Duration) error
    
    // GetListSize returns the size of cached list
    GetListSize(ctx context.Context, key string) (int, error)
}

// CacheStats contains cache statistics
type CacheStats struct {
    Hits        int64
    Misses      int64
    Sets        int64
    Deletes     int64
    HitRate     float64
    Errors      int64
    LastError   error
    LastErrorAt time.Time
}
```

### Cache Key Builder Interface
```go
// KeyBuilder provides flexible cache key generation
type KeyBuilder interface {
    // Build constructs cache key from components
    Build(parts ...string) string
    
    // WithPrefix adds prefix to key
    WithPrefix(prefix string) KeyBuilder
    
    // WithNamespace adds namespace to key
    WithNamespace(namespace string) KeyBuilder
    
    // WithID adds ID to key
    WithID(id interface{}) KeyBuilder
    
    // WithTimestamp adds timestamp to key
    WithTimestamp() KeyBuilder
    
    // ForService creates service-specific key
    ForService(serviceName string) KeyBuilder
    
    // ForEntity creates entity-specific key
    ForEntity(entityName string) KeyBuilder
    
    // ForList creates list/collection key
    ForList(entityName string) KeyBuilder
    
    // ForItem creates item key with ID
    ForItem(entityName string, id interface{}) KeyBuilder
    
    // String returns the final key as string
    String() string
}

// NewKeyBuilder creates a new key builder instance
func NewKeyBuilder(separator string) KeyBuilder

// Common key patterns
type KeyPattern string

const (
    KeyPatternItem       KeyPattern = "{service}:{entity}:{id}"           // user-service:user:123
    KeyPatternList       KeyPattern = "{service}:{entity}:all"            // user-service:user:all
    KeyPatternSearch     KeyPattern = "{service}:{entity}:search:{query}" // user-service:user:search:john
    KeyPatternCount      KeyPattern = "{service}:{entity}:count"          // user-service:user:count
    KeyPatternSession    KeyPattern = "{service}:session:{token}"         // auth-service:session:abc123
    KeyPatternTimeSeries KeyPattern = "{service}:{entity}:{timestamp}"    // analytics:event:20260114
)
```

### CRUD Helper Interface
```go
// CRUDCache provides high-level CRUD operations with automatic cache invalidation
type CRUDCache interface {
    // Create adds new item and invalidates list cache
    Create(ctx context.Context, entityName string, id interface{}, data interface{}, ttl time.Duration) error
    
    // Read retrieves single item from cache
    Read(ctx context.Context, entityName string, id interface{}, dest interface{}) error
    
    // Update updates item and invalidates related caches
    Update(ctx context.Context, entityName string, id interface{}, data interface{}, ttl time.Duration) error
    
    // Delete removes item and invalidates related caches
    Delete(ctx context.Context, entityName string, id interface{}) error
    
    // List retrieves all items of entity type
    List(ctx context.Context, entityName string, dest interface{}) error
    
    // CreateWithListSync creates item and adds to cached list atomically
    CreateWithListSync(ctx context.Context, entityName string, id interface{}, data interface{}, ttl time.Duration) error
    
    // DeleteWithListSync deletes item and removes from cached list atomically
    DeleteWithListSync(ctx context.Context, entityName string, id interface{}, matchFn func(interface{}) bool, ttl time.Duration) error
    
    // UpdateWithListSync updates item and updates in cached list atomically
    UpdateWithListSync(ctx context.Context, entityName string, id interface{}, data interface{}, matchFn func(interface{}) bool, ttl time.Duration) error
}

// NewCRUDCache creates a CRUD cache wrapper
func NewCRUDCache(cache Cache, keyBuilder KeyBuilder, serviceName string) CRUDCache
```

### Configuration
```go
// Config is the main configuration for cache
type Config struct {
    Type        CacheType         // redis, memory, multilayer
    ServiceName string            // Service name for key prefixing
    Redis       *RedisConfig      // Redis configuration
    Memory      *MemoryConfig     // Memory cache configuration
    Metrics     bool              // Enable metrics collection
    Compress    bool              // Enable compression
    KeySeparator string           // Key separator (default ":")
}

// CacheType represents cache backend type
type CacheType string

const (
    CacheTypeRedis      CacheType = "redis"
    CacheTypeMemory     CacheType = "memory"
    CacheTypeMultilayer CacheType = "multilayer"
)

// RedisConfig contains Redis configuration
type RedisConfig struct {
    Addr         string        // Redis address (localhost:6379)
    Password     string        // Redis password
    DB           int           // Database number
    PoolSize     int           // Connection pool size
    MinIdleConns int           // Minimum idle connections
    DialTimeout  time.Duration // Connection timeout
    ReadTimeout  time.Duration // Read timeout
    WriteTimeout time.Duration // Write timeout
    Cluster      bool          // Use Redis cluster
    ClusterAddrs []string      // Cluster addresses
}

// MemoryConfig contains in-memory cache configuration
type MemoryConfig struct {
    MaxSize         int64         // Maximum memory size (bytes)
    DefaultTTL      time.Duration // Default TTL
    CleanupInterval time.Duration // Cleanup interval
    EvictionPolicy  string        // lru, lfu, fifo
}
```

### Factory Pattern
```go
// New creates cache instance from config
func New(cfg Config) (Cache, error)

// NewRedis creates Redis cache
func NewRedis(cfg RedisConfig) (Cache, error)

// NewMemory creates in-memory cache
func NewMemory(cfg MemoryConfig) (Cache, error)

// NewMultilayer creates multi-layer cache (L1: Memory, L2: Redis)
func NewMultilayer(l1 Cache, l2 Cache) (Cache, error)

// NewWithCRUD creates cache with CRUD helper
func NewWithCRUD(cfg Config) (CRUDCache, error)
```

## Implementation Requirements

### 1. Error Handling
```go
// Define custom errors
var (
    ErrCacheMiss       = errors.New("cache: key not found")
    ErrInvalidValue    = errors.New("cache: invalid value type")
    ErrSerialization   = errors.New("cache: serialization failed")
    ErrConnection      = errors.New("cache: connection failed")
    ErrInvalidKey      = errors.New("cache: invalid key format")
    ErrListNotFound    = errors.New("cache: list not found")
    ErrItemNotFound    = errors.New("cache: item not found in list")
    ErrConcurrentWrite = errors.New("cache: concurrent write detected")
)

// IsNotFound checks if error is cache miss
func IsNotFound(err error) bool

// IsConnectionError checks if error is connection-related
func IsConnectionError(err error) bool
```

### 2. Serialization
```go
// Serializer interface for data serialization
type Serializer interface {
    Marshal(v interface{}) ([]byte, error)
    Unmarshal(data []byte, v interface{}) error
}

// Support JSON, MessagePack, Protobuf
// Default: JSON with automatic fallback

// JSONSerializer default JSON serializer
type JSONSerializer struct{}

// MessagePackSerializer for high-performance serialization
type MessagePackSerializer struct{}
```

### 3. Generic Support (Go 1.18+)
```go
// GetTyped is generic version of Get
func GetTyped[T any](ctx context.Context, c Cache, key string) (T, error)

// SetTyped is generic version of Set
func SetTyped[T any](ctx context.Context, c Cache, key string, value T, ttl time.Duration) error

// GetListTyped retrieves typed list from cache
func GetListTyped[T any](ctx context.Context, c ListCache, key string) ([]T, error)

// SetListTyped stores typed list in cache
func SetListTyped[T any](ctx context.Context, c ListCache, key string, list []T, ttl time.Duration) error

// AppendToListTyped adds typed item to list
func AppendToListTyped[T any](ctx context.Context, c ListCache, key string, item T, ttl time.Duration) error
```

### 4. Helper Functions
```go
// Remember pattern: Get or compute and set
func Remember[T any](ctx context.Context, c Cache, key string, ttl time.Duration, fn func() (T, error)) (T, error)

// RememberForever pattern without TTL
func RememberForever[T any](ctx context.Context, c Cache, key string, fn func() (T, error)) (T, error)

// RememberList gets or computes list
func RememberList[T any](ctx context.Context, c ListCache, key string, ttl time.Duration, fn func() ([]T, error)) ([]T, error)

// Tags support for group invalidation
type TaggedCache interface {
    Tags(tags ...string) Cache
    Flush() error
}
```

### 5. Cache Key Builder Implementation
```go
type keyBuilder struct {
    parts     []string
    separator string
    prefix    string
    namespace string
}

// Build constructs final cache key
func (kb *keyBuilder) Build(parts ...string) string {
    allParts := make([]string, 0, len(kb.parts)+len(parts))
    
    if kb.prefix != "" {
        allParts = append(allParts, kb.prefix)
    }
    if kb.namespace != "" {
        allParts = append(allParts, kb.namespace)
    }
    
    allParts = append(allParts, kb.parts...)
    allParts = append(allParts, parts...)
    
    return strings.Join(allParts, kb.separator)
}

// ForItem creates item-specific key: "service:entity:id"
func (kb *keyBuilder) ForItem(entityName string, id interface{}) KeyBuilder {
    newKb := kb.clone()
    newKb.parts = append(newKb.parts, entityName, fmt.Sprint(id))
    return newKb
}

// ForList creates list key: "service:entity:all"
func (kb *keyBuilder) ForList(entityName string) KeyBuilder {
    newKb := kb.clone()
    newKb.parts = append(newKb.parts, entityName, "all")
    return newKb
}

// Example usage:
// kb := NewKeyBuilder(":")
// itemKey := kb.ForService("user-service").ForItem("user", 123).String()
// // Result: "user-service:user:123"
// listKey := kb.ForService("user-service").ForList("user").String()
// // Result: "user-service:user:all"
```

### 6. List/Array Cache Management Implementation
```go
// AppendToList implementation logic:
// 1. Get existing list from cache
// 2. Deserialize to array
// 3. Append new item
// 4. Serialize updated array
// 5. Store back to cache with TTL

func (c *redisCache) AppendToList(ctx context.Context, key string, item interface{}, ttl time.Duration) error {
    // Use Redis transaction for atomicity
    pipe := c.client.TxPipeline()
    
    // Get current list
    var currentList []interface{}
    err := c.Get(ctx, key, &currentList)
    if err != nil && !IsNotFound(err) {
        return err
    }
    
    // Initialize empty list if not found
    if IsNotFound(err) {
        currentList = []interface{}{}
    }
    
    // Append new item
    currentList = append(currentList, item)
    
    // Serialize and store
    data, err := c.serializer.Marshal(currentList)
    if err != nil {
        return ErrSerialization
    }
    
    pipe.Set(ctx, key, data, ttl)
    _, err = pipe.Exec(ctx)
    
    return err
}

// RemoveFromList implementation logic:
// 1. Get list from cache
// 2. Deserialize to array
// 3. Filter array using predicate function
// 4. Serialize filtered array
// 5. Store back to cache

func (c *redisCache) RemoveFromList(ctx context.Context, key string, predicate func(item interface{}) bool, ttl time.Duration) error {
    var currentList []interface{}
    err := c.Get(ctx, key, &currentList)
    if err != nil {
        return err
    }
    
    // Filter list
    filtered := make([]interface{}, 0, len(currentList))
    for _, item := range currentList {
        if !predicate(item) {
            filtered = append(filtered, item)
        }
    }
    
    // Store filtered list
    return c.SetList(ctx, key, filtered, ttl)
}

// UpdateInList implementation logic:
// 1. Get list from cache
// 2. Deserialize to array
// 3. Find and update matching items
// 4. Serialize updated array
// 5. Store back to cache

func (c *redisCache) UpdateInList(ctx context.Context, key string, predicate func(item interface{}) bool, updater func(item interface{}) interface{}, ttl time.Duration) error {
    var currentList []interface{}
    err := c.Get(ctx, key, &currentList)
    if err != nil {
        return err
    }
    
    // Update matching items
    updated := make([]interface{}, len(currentList))
    for i, item := range currentList {
        if predicate(item) {
            updated[i] = updater(item)
        } else {
            updated[i] = item
        }
    }
    
    // Store updated list
    return c.SetList(ctx, key, updated, ttl)
}
```

### 7. CRUD Operations with Cache Synchronization
```go
type crudCache struct {
    cache       Cache
    keyBuilder  KeyBuilder
    serviceName string
}

// CreateWithListSync creates item and updates list cache
func (cc *crudCache) CreateWithListSync(ctx context.Context, entityName string, id interface{}, data interface{}, ttl time.Duration) error {
    // 1. Store individual item
    itemKey := cc.keyBuilder.ForService(cc.serviceName).ForItem(entityName, id).String()
    if err := cc.cache.Set(ctx, itemKey, data, ttl); err != nil {
        return err
    }
    
    // 2. Get list cache
    listKey := cc.keyBuilder.ForService(cc.serviceName).ForList(entityName).String()
    
    // 3. Cast cache to ListCache if supported
    if listCache, ok := cc.cache.(ListCache); ok {
        // Append to list using ListCache interface
        return listCache.AppendToList(ctx, listKey, data, ttl)
    }
    
    // 4. Fallback: manual list management
    var currentList []interface{}
    err := cc.cache.Get(ctx, listKey, &currentList)
    if err != nil && !IsNotFound(err) {
        return err
    }
    
    if IsNotFound(err) {
        currentList = []interface{}{}
    }
    
    currentList = append(currentList, data)
    return cc.cache.Set(ctx, listKey, currentList, ttl)
}

// DeleteWithListSync deletes item and updates list cache
func (cc *crudCache) DeleteWithListSync(ctx context.Context, entityName string, id interface{}, matchFn func(interface{}) bool, ttl time.Duration) error {
    // 1. Delete individual item
    itemKey := cc.keyBuilder.ForService(cc.serviceName).ForItem(entityName, id).String()
    if err := cc.cache.Delete(ctx, itemKey); err != nil {
        return err
    }
    
    // 2. Update list cache
    listKey := cc.keyBuilder.ForService(cc.serviceName).ForList(entityName).String()
    
    if listCache, ok := cc.cache.(ListCache); ok {
        return listCache.RemoveFromList(ctx, listKey, matchFn, ttl)
    }
    
    // Fallback: manual list management
    var currentList []interface{}
    if err := cc.cache.Get(ctx, listKey, &currentList); err != nil {
        if IsNotFound(err) {
            return nil // List doesn't exist, nothing to update
        }
        return err
    }
    
    // Filter list
    filtered := make([]interface{}, 0, len(currentList))
    for _, item := range currentList {
        if !matchFn(item) {
            filtered = append(filtered, item)
        }
    }
    
    return cc.cache.Set(ctx, listKey, filtered, ttl)
}

// UpdateWithListSync updates item and list cache
func (cc *crudCache) UpdateWithListSync(ctx context.Context, entityName string, id interface{}, data interface{}, matchFn func(interface{}) bool, ttl time.Duration) error {
    // 1. Update individual item
    itemKey := cc.keyBuilder.ForService(cc.serviceName).ForItem(entityName, id).String()
    if err := cc.cache.Set(ctx, itemKey, data, ttl); err != nil {
        return err
    }
    
    // 2. Update in list cache
    listKey := cc.keyBuilder.ForService(cc.serviceName).ForList(entityName).String()
    
    if listCache, ok := cc.cache.(ListCache); ok {
        updater := func(item interface{}) interface{} {
            return data
        }
        return listCache.UpdateInList(ctx, listKey, matchFn, updater, ttl)
    }
    
    // Fallback: manual list management
    var currentList []interface{}
    if err := cc.cache.Get(ctx, listKey, &currentList); err != nil {
        if IsNotFound(err) {
            return nil
        }
        return err
    }
    
    // Update matching items
    for i, item := range currentList {
        if matchFn(item) {
            currentList[i] = data
        }
    }
    
    return cc.cache.Set(ctx, listKey, currentList, ttl)
}
```

## Usage Examples

### 1. Basic Usage
```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/yourorg/microservices/pkg/cache"
)

func main() {
    // Initialize cache
    cfg := cache.Config{
        Type:        cache.CacheTypeRedis,
        ServiceName: "user-service",
        Redis: &cache.RedisConfig{
            Addr:     "localhost:6379",
            Password: "",
            DB:       0,
            PoolSize: 10,
        },
    }

    c, err := cache.New(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer c.Close()

    ctx := context.Background()
    
    // Set value
    user := map[string]string{"name": "John", "email": "john@example.com"}
    err = c.Set(ctx, "user:123", user, 5*time.Minute)
    
    // Get value
    var retrievedUser map[string]string
    err = c.Get(ctx, "user:123", &retrievedUser)
}
```

### 2. Using Key Builder
```go
package main

import (
    "context"
    "github.com/yourorg/microservices/pkg/cache"
)

func main() {
    c, _ := cache.New(cache.Config{
        Type:        cache.CacheTypeRedis,
        ServiceName: "user-service",
    })
    
    kb := cache.NewKeyBuilder(":")
    ctx := context.Background()
    
    // Build item key: "user-service:user:123"
    itemKey := kb.ForService("user-service").ForItem("user", 123).String()
    
    // Build list key: "user-service:user:all"
    listKey := kb.ForService("user-service").ForList("user").String()
    
    // Build search key: "user-service:user:search:john"
    searchKey := kb.ForService("user-service").
        ForEntity("user").
        Build("search", "john")
    
    user := User{ID: 123, Name: "John"}
    c.Set(ctx, itemKey, user, 5*time.Minute)
}
```

### 3. Using Generics
```go
type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func main() {
    c, _ := cache.New(cache.Config{
        Type: cache.CacheTypeRedis,
    })
    
    ctx := context.Background()
    
    // Set with type safety
    user := User{ID: 123, Name: "John", Email: "john@example.com"}
    err := cache.SetTyped(ctx, c, "user:123", user, 5*time.Minute)
    
    // Get with type safety
    retrievedUser, err := cache.GetTyped[User](ctx, c, "user:123")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("User: %+v\n", retrievedUser)
}
```

### 4. Remember Pattern (Cache-Aside)
```go
func GetUserByID(ctx context.Context, c cache.Cache, db *sql.DB, userID int) (User, error) {
    key := fmt.Sprintf("user:%d", userID)
    
    // Try cache first, compute if miss
    user, err := cache.Remember(ctx, c, key, 5*time.Minute, func() (User, error) {
        // This function only runs on cache miss
        return fetchUserFromDB(db, userID)
    })
    
    return user, err
}

func fetchUserFromDB(db *sql.DB, userID int) (User, error) {
    var user User
    err := db.QueryRow("SELECT id, name, email FROM users WHERE id = ?", userID).
        Scan(&user.ID, &user.Name, &user.Email)
    return user, err
}
```

### 5. List/Array Cache Management
```go
func main() {
    c, _ := cache.New(cache.Config{
        Type: cache.CacheTypeRedis,
    })
    
    ctx := context.Background()
    listKey := "users:all"
    
    // Store initial list
    users := []User{
        {ID: 1, Name: "John"},
        {ID: 2, Name: "Jane"},
    }
    cache.SetListTyped(ctx, c.(cache.ListCache), listKey, users, 10*time.Minute)
    
    // Append new user to list
    newUser := User{ID: 3, Name: "Bob"}
    cache.AppendToListTyped(ctx, c.(cache.ListCache), listKey, newUser, 10*time.Minute)
    
    // Get entire list
    allUsers, _ := cache.GetListTyped[User](ctx, c.(cache.ListCache), listKey)
    fmt.Printf("All users: %+v\n", allUsers)
}
```

### 6. CRUD Operations with Auto Sync
```go
type UserService struct {
    cache cache.CRUDCache
    db    *sql.DB
}

func NewUserService(db *sql.DB) *UserService {
    crudCache, _ := cache.NewWithCRUD(cache.Config{
        Type:        cache.CacheTypeRedis,
        ServiceName: "user-service",
        Redis: &cache.RedisConfig{
            Addr: "localhost:6379",
        },
    })
    
    return &UserService{
        cache: crudCache,
        db:    db,
    }
}

// Create user - auto updates both item cache and list cache
func (s *UserService) CreateUser(ctx context.Context, user User) error {
    // Save to database
    result, err := s.db.ExecContext(ctx, 
        "INSERT INTO users (name, email) VALUES (?, ?)", 
        user.Name, user.Email)
    if err != nil {
        return err
    }
    
    id, _ := result.LastInsertId()
    user.ID = int(id)
    
    // Cache with automatic list sync
    // This will:
    // 1. Set "user-service:user:123" = user data
    // 2. Append user to "user-service:user:all" list
    return s.cache.CreateWithListSync(ctx, "user", user.ID, user, 10*time.Minute)
}

// Update user - auto updates both item and list cache
func (s *UserService) UpdateUser(ctx context.Context, user User) error {
    // Update database
    _, err := s.db.ExecContext(ctx,
        "UPDATE users SET name = ?, email = ? WHERE id = ?",
        user.Name, user.Email, user.ID)
    if err != nil {
        return err
    }
    
    // Update cache with list sync
    // Match function to find user in list by ID
    matchFn := func(item interface{}) bool {
        if u, ok := item.(User); ok {
            return u.ID == user.ID
        }
        return false
    }
    
    // This will:
    // 1. Update "user-service:user:123" = new user data
    // 2. Find and update user in "user-service:user:all" list
    return s.cache.UpdateWithListSync(ctx, "user", user.ID, user, matchFn, 10*time.Minute)
}

// Delete user - auto removes from both caches
func (s *UserService) DeleteUser(ctx context.Context, userID int) error {
    // Delete from database
    _, err := s.db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", userID)
    if err != nil {
        return err
    }
    
    // Match function for deletion
    matchFn := func(item interface{}) bool {
        if u, ok := item.(User); ok {
            return u.ID == userID
        }
        return false
    }
    
    // This will:
    // 1. Delete "user-service:user:123"
    // 2. Remove user from "user-service:user:all" list
    return s.cache.DeleteWithListSync(ctx, "user", userID, matchFn, 10*time.Minute)
}

// List all users with cache
func (s *UserService) ListUsers(ctx context.Context) ([]User, error) {
    return cache.RememberList(ctx, s.cache.(cache.ListCache), 
        "user-service:user:all", 10*time.Minute, func() ([]User, error) {
            // Fetch from DB on cache miss
            rows, err := s.db.QueryContext(ctx, "SELECT id, name, email FROM users")
            if err != nil {
                return nil, err
            }
            defer rows.Close()
            
            var users []User
            for rows.Next() {
                var u User
                rows.Scan(&u.ID, &u.Name, &u.Email)
                users = append(users, u)
            }
            return users, nil
        })
}
```

### 7. Multi-layer Cache
```go
func main() {
    // L1: In-memory cache (fast, small capacity)
    memCache, _ := cache.NewMemory(cache.MemoryConfig{
        MaxSize:     100 * 1024 * 1024, // 100MB
        DefaultTTL:  1 * time.Minute,
    })

    // L2: Redis cache (slower, large capacity)
    redisCache, _ := cache.NewRedis(cache.RedisConfig{
        Addr: "localhost:6379",
    })

    // Create multi-layer cache
    mlCache, _ := cache.NewMultilayer(memCache, redisCache)
    
    ctx := context.Background()
    
    // Get automatically checks L1 -> L2 -> miss
    var user User
    err := mlCache.Get(ctx, "user:123", &user)
    
    // Set stores in both layers
    mlCache.Set(ctx, "user:123", user, 5*time.Minute)
}
```

### 8. Batch Operations
```go
func main() {
    c, _ := cache.New(cache.Config{
        Type: cache.CacheTypeRedis,
    })
    
    ctx := context.Background()
    
    // Batch set
    items := map[string]interface{}{
        "user:1": User{ID: 1, Name: "John"},
        "user:2": User{ID: 2, Name: "Jane"},
        "user:3": User{ID: 3, Name: "Bob"},
    }
    
    if batchCache, ok := c.(cache.BatchCache); ok {
        batchCache.MSet(ctx, items, 5*time.Minute)
        
        // Batch get
        keys := []string{"user:1", "user:2", "user:3"}
        results, _ := batchCache.MGet(ctx, keys)
        
        for key, value := range results {
            fmt.Printf("%s: %+v\n", key, value)
        }
        
        // Batch delete
        batchCache.MDelete(ctx, keys)
    }
}
```

### 9. Pattern-based Operations
```go
func main() {
    c, _ := cache.New(cache.Config{
        Type: cache.CacheTypeRedis,
    })
    
    ctx := context.Background()
    
    if patternCache, ok := c.(cache.PatternCache); ok {
        // Delete all user caches
        patternCache.DeleteByPattern(ctx, "user-service:user:*")
        
        // Get all session keys
        sessionKeys, _ := patternCache.Keys(ctx, "user-service:session:*")
        fmt.Printf("Active sessions: %v\n", sessionKeys)
    }
}
```

### 10. Integration Example - Complete Service
```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "time"
    
    "github.com/yourorg/microservices/pkg/cache"
)

type Product struct {
    ID          int     `json:"id"`
    Name        string  `json:"name"`
    Price       float64 `json:"price"`
    Description string  `json:"description"`
}

type ProductService struct {
    cache cache.CRUDCache
    db    *sql.DB
}

func NewProductService(db *sql.DB) (*ProductService, error) {
    crudCache, err := cache.NewWithCRUD(cache.Config{
        Type:         cache.CacheTypeRedis,
        ServiceName:  "product-service",
        KeySeparator: ":",
        Redis: &cache.RedisConfig{
            Addr:         "localhost:6379",
            PoolSize:     10,
            MinIdleConns: 5,
        },
        Metrics: true,
    })
    
    if err != nil {
        return nil, err
    }
    
    return &ProductService{
        cache: crudCache,
        db:    db,
    }, nil
}

func (s *ProductService) CreateProduct(ctx context.Context, product Product) (*Product, error) {
    // Insert to database
    result, err := s.db.ExecContext(ctx,
        "INSERT INTO products (name, price, description) VALUES (?, ?, ?)",
        product.Name, product.Price, product.Description)
    if err != nil {
        return nil, err
    }
    
    id, _ := result.LastInsertId()
    product.ID = int(id)
    
    // Cache with list sync
    // Key: "product-service:product:123"
    // List Key: "product-service:product:all"
    err = s.cache.CreateWithListSync(ctx, "product", product.ID, product, 30*time.Minute)
    if err != nil {
        // Log error but don't fail the operation
        fmt.Printf("Cache error: %v\n", err)
    }
    
    return &product, nil
}

func (s *ProductService) GetProduct(ctx context.Context, id int) (*Product, error) {
    var product Product
    
    // Try cache first
    err := s.cache.Read(ctx, "product", id, &product)
    if err == nil {
        return &product, nil
    }
    
    if !cache.IsNotFound(err) {
        fmt.Printf("Cache error: %v\n", err)
    }
    
    // Fetch from database
    err = s.db.QueryRowContext(ctx,
        "SELECT id, name, price, description FROM products WHERE id = ?", id).
        Scan(&product.ID, &product.Name, &product.Price, &product.Description)
    
    if err != nil {
        return nil, err
    }
    
    // Store in cache
    s.cache.Create(ctx, "product", product.ID, product, 30*time.Minute)
    
    return &product, nil
}

func (s *ProductService) UpdateProduct(ctx context.Context, product Product) error {
    // Update database
    _, err := s.db.ExecContext(ctx,
        "UPDATE products SET name = ?, price = ?, description = ? WHERE id = ?",
        product.Name, product.Price, product.Description, product.ID)
    if err != nil {
        return err
    }
    
    // Update cache with list sync
    matchFn := func(item interface{}) bool {
        if p, ok := item.(Product); ok {
            return p.ID == product.ID
        }
        return false
    }
    
    return s.cache.UpdateWithListSync(ctx, "product", product.ID, product, matchFn, 30*time.Minute)
}

func (s *ProductService) DeleteProduct(ctx context.Context, id int) error {
    // Delete from database
    _, err := s.db.ExecContext(ctx, "DELETE FROM products WHERE id = ?", id)
    if err != nil {
        return err
    }
    
    // Delete from cache with list sync
    matchFn := func(item interface{}) bool {
        if p, ok := item.(Product); ok {
            return p.ID == id
        }
        return false
    }
    
    return s.cache.DeleteWithListSync(ctx, "product", id, matchFn, 30*time.Minute)
}

func (s *ProductService) ListProducts(ctx context.Context) ([]Product, error) {
    // Use RememberList pattern
    kb := cache.NewKeyBuilder(":")
    listKey := kb.ForService("product-service").ForList("product").String()
    
    return cache.RememberList(ctx, s.cache.(cache.ListCache), listKey, 30*time.Minute,
        func() ([]Product, error) {
            rows, err := s.db.QueryContext(ctx,
                "SELECT id, name, price, description FROM products")
            if err != nil {
                return nil, err
            }
            defer rows.Close()
            
            var products []Product
            for rows.Next() {
                var p Product
                err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.Description)
                if err != nil {
                    return nil, err
                }
                products = append(products, p)
            }
            
            return products, nil
        })
}

func main() {
    db, _ := sql.Open("mysql", "user:password@tcp(localhost:3306)/shop")
    defer db.Close()
    
    productService, _ := NewProductService(db)
    ctx := context.Background()
    
    // Create product
    product, _ := productService.CreateProduct(ctx, Product{
        Name:        "Laptop",
        Price:       999.99,
        Description: "High-performance laptop",
    })
    
    // Get product
    retrieved, _ := productService.GetProduct(ctx, product.ID)
    fmt.Printf("Product: %+v\n", retrieved)
    
    // Update product
    product.Price = 899.99
    productService.UpdateProduct(ctx, *product)
    
    // List all products
    products, _ := productService.ListProducts(ctx)
    fmt.Printf("All products: %+v\n", products)
    
    // Delete product
    productService.DeleteProduct(ctx, product.ID)
}
```

## Testing Requirements

### Unit Tests
- Test all interface methods with mocked backends
- Test serialization/deserialization for various data types
- Test TTL expiration behavior
- Test concurrent access with race detector
- Test error cases and edge conditions
- Test cache key builder with various patterns
- Test list operations (append, remove, update)
- Test CRUD operations with cache synchronization

### Integration Tests
- Test with real Redis instance (using testcontainers or docker-compose)
- Test with Redis cluster configuration
- Test memory cache under high load
- Test multi-layer cache L1/L2 synchronization
- Test batch operations performance
- Test pattern-based operations

### Performance Benchmarks
```go
// Benchmark core operations
func BenchmarkRedisSet(b *testing.B)
func BenchmarkRedisGet(b *testing.B)
func BenchmarkMemorySet(b *testing.B)
func BenchmarkMemoryGet(b *testing.B)

// Benchmark serialization
func BenchmarkJSONSerialization(b *testing.B)
func BenchmarkMessagePackSerialization(b *testing.B)

// Benchmark list operations
func BenchmarkListAppend(b *testing.B)
func BenchmarkListUpdate(b *testing.B)
func BenchmarkListRemove(b *testing.B)

// Benchmark CRUD operations
func BenchmarkCRUDCreate(b *testing.B)
func BenchmarkCRUDUpdate(b *testing.B)
func BenchmarkCRUDDelete(b *testing.B)

// Benchmark concurrent access
func BenchmarkConcurrentReads(b *testing.B)
func BenchmarkConcurrentWrites(b *testing.B)
```

### Test Coverage Goals
- Overall coverage: > 80%
- Critical paths: > 95%
- Error handling: 100%

## Documentation Requirements

### README.md must include:
1. **Installation**: Clear `go get` command and version requirements
2. **Quick Start**: Basic setup and usage examples
3. **Configuration Guide**: All configuration options with examples
4. **Key Builder Guide**: How to use flexible key generation
5. **List Cache Guide**: Managing arrays/lists in cache
6. **CRUD Operations Guide**: Using CRUD helper with auto sync
7. **Examples**: Real-world use cases for each service type
8. **Best Practices**: When to use which cache type, TTL recommendations
9. **Performance Tuning**: Tips for optimization
10. **Migration Guide**: How to integrate into existing services
11. **API Documentation**: Links to GoDoc
12. **Troubleshooting**: Common issues and solutions

### Code Comments (GoDoc Format)
- All exported types, functions, and methods must have comments
- Include usage examples in comments for complex functions
- Document all parameters and return values
- Add "Deprecated:" notices where applicable
- Include performance notes for critical operations

### Additional Documentation
- Architecture diagrams for multi-layer cache
- Sequence diagrams for CRUD operations with sync
- Flowcharts for cache invalidation strategies
- API reference documentation

## Dependencies

### Required
- `github.com/redis/go-redis/v9` - Redis client (latest stable)
- Go 1.18+ for generics support
- No framework-specific dependencies (framework-agnostic)

### Optional
- `github.com/allegro/bigcache/v3` - High-performance in-memory cache
- `github.com/vmihailenco/msgpack/v5` - MessagePack serialization
- `github.com/prometheus/client_golang` - Prometheus metrics integration
- `github.com/stretchr/testify` - Testing utilities (dev dependency)

### Development Dependencies
- `github.com/testcontainers/testcontainers-go` - Integration testing with Docker
- `github.com/golang/mock` - Mock generation
- `github.com/golangci/golangci-lint` - Linting

## Best Practices

### 1. Connection Management
- Use singleton pattern for cache instance per service
- Implement health checks for Redis connection
- Graceful shutdown with proper Close() calls
- Connection pooling configuration based on load
- Retry logic with exponential backoff

### 2. Key Naming Convention
- Use consistent separators (default: ":")
- Follow pattern: `{service}:{entity}:{id}` (e.g., `user-service:user:123`)
- For lists: `{service}:{entity}:all` (e.g., `user-service:user:all`)
- For searches: `{service}:{entity}:search:{query}`
- For sessions: `{service}:session:{token}`
- Use KeyBuilder for consistency across services

### 3. TTL Strategy
- Set appropriate TTL based on data volatility
- Short TTL (1-5 min) for frequently changing data
- Medium TTL (10-30 min) for relatively stable data
- Long TTL (1-24 hours) for rarely changing data
- No TTL for static reference data (with manual invalidation)
- Consider using different TTLs for item vs list caches

### 4. Error Handling
- Log errors but don't crash the service
- Implement fallback to database when cache fails
- Use circuit breaker pattern for Redis failures
- Distinguish between cache miss and actual errors
- Monitor error rates

### 5. Cache Invalidation
- Invalidate specific keys on updates
- Use pattern-based invalidation for related keys
- Invalidate list cache when items are added/removed
- Consider using tags for group invalidation
- Implement cache warming for critical data

### 6. Security
- Don't cache sensitive data without encryption
- Sanitize cache keys to prevent injection
- Use separate Redis databases per environment
- Implement rate limiting for cache operations
- Secure Redis with password and TLS

### 7. Monitoring and Metrics
- Track hit/miss ratios
- Monitor cache size and memory usage
- Alert on high error rates
- Track operation latencies
- Monitor TTL distribution

### 8. Performance Optimization
- Use batch operations when possible
- Enable compression for large values
- Use MessagePack for better performance
- Implement multi-layer cache for hot data
- Use pipelining for multiple operations

### 9. List Cache Management
- Keep list cache size manageable (< 1000 items)
- Consider pagination for large lists
- Use atomic operations for list updates
- Invalidate list cache on any item change
- Balance between item cache and list cache freshness

### 10. Service Integration
- Initialize cache in service constructor
- Use dependency injection for testability
- Implement graceful degradation
- Don't make cache a hard dependency
- Test service behavior when cache is unavailable

## Service Integration Examples

### Example 1: Auth Service
```go
package main

import (
    "context"
    "time"
    
    "github.com/yourorg/microservices/pkg/cache"
)

type Session struct {
    UserID    int       `json:"user_id"`
    Token     string    `json:"token"`
    ExpiresAt time.Time `json:"expires_at"`
}

type AuthService struct {
    cache cache.Cache
    kb    cache.KeyBuilder
}

func NewAuthService() *AuthService {
    c, _ := cache.New(cache.Config{
        Type:        cache.CacheTypeRedis,
        ServiceName: "auth-service",
        Redis: &cache.RedisConfig{
            Addr: os.Getenv("REDIS_ADDR"),
        },
    })
    
    return &AuthService{
        cache: c,
        kb:    cache.NewKeyBuilder(":"),
    }
}

func (s *AuthService) StoreSession(ctx context.Context, session Session) error {
    key := s.kb.ForService("auth-service").Build("session", session.Token)
    return s.cache.Set(ctx, key, session, 15*time.Minute)
}

func (s *AuthService) GetSession(ctx context.Context, token string) (*Session, error) {
    key := s.kb.ForService("auth-service").Build("session", token)
    
    return cache.Remember(ctx, s.cache, key, 15*time.Minute, func() (*Session, error) {
        // Validate token and create session from database
        return s.validateTokenFromDB(token)
    })
}
```

### Example 2: Product Service with Full CRUD
```go
package main

import (
    "context"
    "github.com/yourorg/microservices/pkg/cache"
)

type ProductService struct {
    cache cache.CRUDCache
}

func NewProductService() *ProductService {
    crudCache, _ := cache.NewWithCRUD(cache.Config{
        Type:        cache.CacheTypeRedis,
        ServiceName: "product-service",
    })
    
    return &ProductService{cache: crudCache}
}

func (s *ProductService) CreateProduct(ctx context.Context, product Product) error {
    // Save to DB first
    if err := s.db.SaveProduct(product); err != nil {
        return err
    }
    
    // Cache with auto list sync
    return s.cache.CreateWithListSync(ctx, "product", product.ID, product, 30*time.Minute)
}

func (s *ProductService) UpdateProduct(ctx context.Context, product Product) error {
    if err := s.db.UpdateProduct(product); err != nil {
        return err
    }
    
    matchFn := func(item interface{}) bool {
        if p, ok := item.(Product); ok {
            return p.ID == product.ID
        }
        return false
    }
    
    return s.cache.UpdateWithListSync(ctx, "product", product.ID, product, matchFn, 30*time.Minute)
}

func (s *ProductService) DeleteProduct(ctx context.Context, id int) error {
    if err := s.db.DeleteProduct(id); err != nil {
        return err
    }
    
    matchFn := func(item interface{}) bool {
        if p, ok := item.(Product); ok {
            return p.ID == id
        }
        return false
    }
    
    return s.cache.DeleteWithListSync(ctx, "product", id, matchFn, 30*time.Minute)
}
```

### Example 3: Cart Service
```go
package main

import (
    "context"
    "github.com/yourorg/microservices/pkg/cache"
)

type CartItem struct {
    ProductID int     `json:"product_id"`
    Quantity  int     `json:"quantity"`
    Price     float64 `json:"price"`
}

type Cart struct {
    UserID int        `json:"user_id"`
    Items  []CartItem `json:"items"`
}

type CartService struct {
    cache cache.ListCache
    kb    cache.KeyBuilder
}

func NewCartService() *CartService {
    c, _ := cache.New(cache.Config{
        Type:        cache.CacheTypeRedis,
        ServiceName: "cart-service",
    })
    
    return &CartService{
        cache: c.(cache.ListCache),
        kb:    cache.NewKeyBuilder(":"),
    }
}

func (s *CartService) AddToCart(ctx context.Context, userID int, item CartItem) error {
    key := s.kb.ForService("cart-service").ForItem("cart", userID).String()
    
    // Get current cart
    var cart Cart
    err := s.cache.Get(ctx, key, &cart)
    if err != nil && !cache.IsNotFound(err) {
        return err
    }
    
    if cache.IsNotFound(err) {
        cart = Cart{UserID: userID, Items: []CartItem{}}
    }
    
    // Add or update item
    found := false
    for i, existingItem := range cart.Items {
        if existingItem.ProductID == item.ProductID {
            cart.Items[i].Quantity += item.Quantity
            found = true
            break
        }
    }
    
    if !found {
        cart.Items = append(cart.Items, item)
    }
    
    return s.cache.Set(ctx, key, cart, 24*time.Hour)
}

func (s *CartService) RemoveFromCart(ctx context.Context, userID, productID int) error {
    key := s.kb.ForService("cart-service").ForItem("cart", userID).String()
    
    var cart Cart
    if err := s.cache.Get(ctx, key, &cart); err != nil {
        return err
    }
    
    // Remove item
    filtered := make([]CartItem, 0)
    for _, item := range cart.Items {
        if item.ProductID != productID {
            filtered = append(filtered, item)
        }
    }
    
    cart.Items = filtered
    return s.cache.Set(ctx, key, cart, 24*time.Hour)
}
```

### Example 4: Order Service with Multi-layer Cache
```go
package main

import (
    "context"
    "github.com/yourorg/microservices/pkg/cache"
)

type OrderService struct {
    cache cache.Cache
}

func NewOrderService() *OrderService {
    // L1: Memory cache for recent orders
    memCache, _ := cache.NewMemory(cache.MemoryConfig{
        MaxSize:    50 * 1024 * 1024, // 50MB
        DefaultTTL: 5 * time.Minute,
    })
    
    // L2: Redis for all orders
    redisCache, _ := cache.NewRedis(cache.RedisConfig{
        Addr: "localhost:6379",
    })
    
    mlCache, _ := cache.NewMultilayer(memCache, redisCache)
    
    return &OrderService{cache: mlCache}
}

func (s *OrderService) GetOrder(ctx context.Context, orderID int) (*Order, error) {
    key := fmt.Sprintf("order-service:order:%d", orderID)
    
    return cache.Remember(ctx, s.cache, key, 10*time.Minute, func() (*Order, error) {
        return s.db.GetOrder(orderID)
    })
}
```

## Deliverables

1. **Source Code**: Complete implementation in `pkg/cache/`
   - All core files (cache.go, config.go, errors.go, etc.)
   - Backend implementations (redis/, memory/, multilayer/)
   - Helper utilities (keybuilder.go, list_cache.go, crud_helper.go)
   - Serialization support (serializer.go)
   - Metrics collection (metrics.go)

2. **Tests**: Comprehensive test coverage
   - Unit tests for all interfaces and implementations
   - Integration tests with real Redis
   - Benchmark tests for performance validation
   - Test coverage > 80%

3. **Documentation**: Complete user and developer docs
   - README.md with all sections
   - GoDoc comments for all exported APIs
   - Architecture diagrams
   - API reference documentation

4. **Examples**: Working examples for common scenarios
   - Basic usage example
   - Key builder usage
   - List cache management
   - CRUD operations with sync
   - Multi-layer cache setup
   - Service integration examples

5. **Benchmarks**: Performance validation
   - Benchmarks for all major operations
   - Comparison between backends
   - Concurrency benchmarks
   - Memory usage profiling

6. **Migration Guide**: Integration instructions
   - Step-by-step integration guide
   - Configuration examples per service
   - Common migration patterns
   - Troubleshooting guide

## Success Criteria

- [ ] All tests pass with coverage > 80%
- [ ] Benchmarks demonstrate acceptable performance (< 5ms for Redis ops)
- [ ] Documentation is complete, clear, and includes all examples
- [ ] Can be imported and used in any microservice without modification
- [ ] No breaking dependencies on other services or frameworks
- [ ] Thread-safe for concurrent operations
- [ ] Production-ready error handling with proper logging
- [ ] Metrics and monitoring support integrated
- [ ] Cache key builder works for all common patterns
- [ ] List/array cache operations are atomic and reliable
- [ ] CRUD operations correctly sync individual and list caches
- [ ] Multi-layer cache properly manages L1/L2 synchronization
- [ ] Graceful degradation when Redis is unavailable
- [ ] Memory limits are respected for in-memory cache
- [ ] Serialization handles all common Go types
- [ ] Generic functions work correctly with type safety
- [ ] Pattern-based operations work efficiently
- [ ] Code follows Go best practices and idioms
- [ ] No race conditions detected by race detector
- [ ] Compatible with Go 1.18+ for generics support

## Additional Notes

### Design Principles
- **Standalone Package**: No dependencies on other microservices code
- **Framework Agnostic**: Works with any Go web framework or standalone
- **Go Modules**: Use proper versioning with semantic versioning
- **Backward Compatibility**: Consider compatibility for future updates
- **Idiomatic Go**: Follow Go community standards and best practices
- **Clear Abstractions**: Interfaces over concrete implementations

### Architecture Decisions

#### 1. Why Generics?
- Type safety at compile time
- Reduces boilerplate code
- Better developer experience
- Eliminates type assertion errors

#### 2. Why Multiple Interfaces?
- Interface segregation principle
- Services use only what they need
- Easy to test with mocks
- Clear contracts for implementations

#### 3. Why Key Builder?
- Consistent key naming across services
- Reduces typos and bugs
- Easy to change key format
- Supports multiple patterns

#### 4. Why List Cache Management?
- Common pattern in CRUD operations
- Atomic updates prevent race conditions
- Simplifies cache invalidation
- Reduces cache inconsistency

#### 5. Why CRUD Helper?
- High-level API for common operations
- Automatic cache synchronization
- Reduces boilerplate in services
- Enforces best practices

### Performance Targets
- Redis SET: < 2ms (p95)
- Redis GET: < 1ms (p95)
- Memory SET: < 100μs (p95)
- Memory GET: < 50μs (p95)
- Serialization: < 500μs for typical structs
- List operations: < 5ms for lists < 100 items

### Scalability Considerations
- Support Redis Cluster for horizontal scaling
- Connection pooling prevents connection exhaustion
- Batch operations reduce network round trips
- Multi-layer cache reduces Redis load
- Compression reduces bandwidth for large values
- TTL prevents unbounded growth

### Maintenance Guidelines
- Use semantic versioning (v1.0.0, v1.1.0, v2.0.0)
- Maintain CHANGELOG.md
- Deprecate features before removal
- Provide migration guides for breaking changes
- Keep dependencies up to date
- Regular security audits

### Error Handling Philosophy
- Fail gracefully, never crash the service
- Distinguish between recoverable and fatal errors
- Log errors with context
- Provide meaningful error messages
- Use custom error types for specific scenarios
- Return errors, don't panic

### Monitoring Recommendations
- Track cache hit/miss ratios per entity type
- Monitor operation latencies (p50, p95, p99)
- Alert on error rate spikes
- Track memory usage for in-memory cache
- Monitor Redis connection pool usage
- Dashboard for real-time cache metrics

### Future Enhancements (Optional)
- Support for Memcached backend
- Support for distributed caching (Hazelcast, etc.)
- Advanced eviction policies (LFU, adaptive)
- Cache warming strategies
- Distributed cache invalidation
- Support for cache partitioning/sharding
- Integration with OpenTelemetry for tracing
- Support for cache transactions
- Cache versioning for schema migration
- A/B testing support with cache variants

### Common Pitfalls to Avoid
- Don't cache everything - cache strategically
- Don't use very long TTLs for frequently changing data
- Don't ignore cache errors - implement fallbacks
- Don't share cache instances across services
- Don't cache without considering memory limits
- Don't forget to close cache connections
- Don't use cache as source of truth
- Don't skip error handling in production

### Integration Checklist
When integrating this package into a service:
1. [ ] Add package dependency in go.mod
2. [ ] Configure cache in service config file
3. [ ] Initialize cache in service constructor
4. [ ] Implement graceful shutdown
5. [ ] Add cache metrics to monitoring
6. [ ] Configure appropriate TTLs
7. [ ] Test cache behavior under load
8. [ ] Test service behavior when cache is down
9. [ ] Document cache usage in service README
10. [ ] Add cache-related environment variables
