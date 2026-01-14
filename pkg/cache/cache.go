package cache

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Cache defines the main interface for all cache implementations.
type Cache interface {
	// Get retrieves a value from cache and unmarshals it into dest.
	// Returns ErrCacheMiss if the key is not found.
	Get(ctx context.Context, key string, dest interface{}) error

	// Set stores a value in cache with the specified TTL.
	// TTL of 0 means no expiration.
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Delete removes a key from cache.
	// Returns nil if the key doesn't exist.
	Delete(ctx context.Context, key string) error

	// Exists checks if a key exists in cache.
	Exists(ctx context.Context, key string) (bool, error)

	// Close closes the cache connection and releases resources.
	Close() error

	// Ping checks if the cache connection is alive.
	Ping(ctx context.Context) error
}

// BatchCache extends Cache with batch operations support.
type BatchCache interface {
	Cache

	// MGet retrieves multiple keys at once.
	// Returns a map of key to raw bytes. Missing keys are not included.
	MGet(ctx context.Context, keys []string) (map[string][]byte, error)

	// MSet sets multiple key-value pairs with the same TTL.
	MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error

	// MDelete deletes multiple keys.
	// Returns the number of keys that were deleted.
	MDelete(ctx context.Context, keys []string) (int64, error)
}

// PatternCache extends Cache with pattern-based operations.
type PatternCache interface {
	Cache

	// DeleteByPattern deletes keys matching the pattern (e.g., "user:*").
	// Returns the number of keys deleted.
	DeleteByPattern(ctx context.Context, pattern string) (int64, error)

	// Keys returns all keys matching the pattern.
	// Use with caution in production as it can be slow for large datasets.
	Keys(ctx context.Context, pattern string) ([]string, error)

	// Scan iterates over keys matching the pattern with cursor-based pagination.
	// Returns keys and the next cursor. Cursor of 0 indicates end of iteration.
	Scan(ctx context.Context, pattern string, cursor uint64, count int64) ([]string, uint64, error)
}

// MetricsCache extends Cache with metrics collection.
type MetricsCache interface {
	Cache

	// Stats returns the current cache statistics.
	Stats() CacheStats

	// ResetStats resets all statistics to zero.
	ResetStats()
}

// ListCache extends Cache with list/array operations.
type ListCache interface {
	Cache

	// GetList retrieves an entire list from cache.
	// Returns ErrListNotFound if the list doesn't exist.
	GetList(ctx context.Context, key string, dest interface{}) error

	// SetList stores an entire list in cache.
	SetList(ctx context.Context, key string, list interface{}, ttl time.Duration) error

	// AppendToList adds an item to the end of a cached list.
	// Creates the list if it doesn't exist.
	AppendToList(ctx context.Context, key string, item interface{}, ttl time.Duration) error

	// PrependToList adds an item to the beginning of a cached list.
	// Creates the list if it doesn't exist.
	PrependToList(ctx context.Context, key string, item interface{}, ttl time.Duration) error

	// RemoveFromList removes items matching the predicate from the list.
	// The predicate receives a raw JSON representation of each item.
	RemoveFromList(ctx context.Context, key string, predicate func(item interface{}) bool, ttl time.Duration) error

	// UpdateInList updates items matching the predicate in the list.
	// The updater function receives the matched item and returns the updated item.
	UpdateInList(ctx context.Context, key string, predicate func(item interface{}) bool, updater func(item interface{}) interface{}, ttl time.Duration) error

	// GetListSize returns the number of items in the cached list.
	// Returns 0 if the list doesn't exist.
	GetListSize(ctx context.Context, key string) (int, error)
}

// CRUDCache provides high-level CRUD operations with automatic cache invalidation.
type CRUDCache interface {
	// Create stores a new item and optionally invalidates related caches.
	Create(ctx context.Context, entityName string, id interface{}, data interface{}, ttl time.Duration) error

	// Read retrieves an item from cache.
	// Returns ErrCacheMiss if not found.
	Read(ctx context.Context, entityName string, id interface{}, dest interface{}) error

	// Update updates an item in cache.
	Update(ctx context.Context, entityName string, id interface{}, data interface{}, ttl time.Duration) error

	// Delete removes an item from cache.
	Delete(ctx context.Context, entityName string, id interface{}) error

	// List retrieves all items of an entity type from the list cache.
	List(ctx context.Context, entityName string, dest interface{}) error

	// CreateWithListSync creates an item and appends it to the list cache atomically.
	CreateWithListSync(ctx context.Context, entityName string, id interface{}, data interface{}, ttl time.Duration) error

	// UpdateWithListSync updates an item and updates it in the list cache.
	UpdateWithListSync(ctx context.Context, entityName string, id interface{}, data interface{}, matchFn func(interface{}) bool, ttl time.Duration) error

	// DeleteWithListSync deletes an item and removes it from the list cache.
	DeleteWithListSync(ctx context.Context, entityName string, id interface{}, matchFn func(interface{}) bool, ttl time.Duration) error
}

// TaggedCache provides tag-based cache invalidation.
type TaggedCache interface {
	Cache

	// Tags returns a cache instance that will tag all subsequent Set operations.
	Tags(tags ...string) Cache

	// FlushTags removes all entries associated with the given tags.
	FlushTags(ctx context.Context, tags ...string) error
}

// CacheStats contains cache statistics.
type CacheStats struct {
	// Hits is the number of cache hits.
	Hits int64 `json:"hits"`

	// Misses is the number of cache misses.
	Misses int64 `json:"misses"`

	// Sets is the number of set operations.
	Sets int64 `json:"sets"`

	// Deletes is the number of delete operations.
	Deletes int64 `json:"deletes"`

	// Errors is the number of errors encountered.
	Errors int64 `json:"errors"`

	// HitRate is the cache hit rate (hits / (hits + misses)).
	HitRate float64 `json:"hitRate"`

	// Size is the current number of items in cache (if available).
	Size int64 `json:"size"`

	// MemoryUsage is the current memory usage in bytes (if available).
	MemoryUsage int64 `json:"memoryUsage"`

	// AvgGetLatency is the average latency for Get operations.
	AvgGetLatency time.Duration `json:"avgGetLatency"`

	// AvgSetLatency is the average latency for Set operations.
	AvgSetLatency time.Duration `json:"avgSetLatency"`

	// LastError is the most recent error (not serialized to JSON).
	LastError error `json:"-"`

	// LastErrorAt is the timestamp of the most recent error.
	LastErrorAt time.Time `json:"lastErrorAt,omitempty"`
}

// FullCache combines all cache interfaces.
// Use type assertions to check which features are available.
type FullCache interface {
	Cache
	BatchCache
	PatternCache
	ListCache
	MetricsCache
}

// New creates a cache instance based on the configuration.
func New(cfg Config, opts ...Option) (Cache, error) {
	cfg.WithDefaults()

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	options := applyOptions(opts...)

	switch cfg.Type {
	case CacheTypeRedis:
		return newRedisCache(cfg, options)
	case CacheTypeMemory:
		return newMemoryCache(cfg, options)
	case CacheTypeMultilayer:
		return newMultilayerCache(cfg, options)
	default:
		return nil, fmt.Errorf("%w: unsupported cache type: %s", ErrInvalidConfig, cfg.Type)
	}
}

// newRedisCache creates a Redis cache (lazy loaded).
// This is a placeholder that returns an error asking to import redis subpackage.
func newRedisCache(cfg Config, opts *cacheOptions) (Cache, error) {
	return nil, fmt.Errorf("use microservices/pkg/cache/redis.New() to create a Redis cache")
}

// newMemoryCache creates a memory cache (lazy loaded).
// This is a placeholder that returns an error asking to import memory subpackage.
func newMemoryCache(cfg Config, opts *cacheOptions) (Cache, error) {
	return nil, fmt.Errorf("use microservices/pkg/cache/memory.New() to create a memory cache")
}

// newMultilayerCache creates a multilayer cache.
// This is a placeholder that returns an error asking to import multilayer subpackage.
func newMultilayerCache(cfg Config, opts *cacheOptions) (Cache, error) {
	return nil, fmt.Errorf("use microservices/pkg/cache/multilayer.New() to create a multilayer cache")
}

// NewWithCRUD creates a cache with CRUD helper.
func NewWithCRUD(cfg Config, opts ...Option) (CRUDCache, error) {
	c, err := New(cfg, opts...)
	if err != nil {
		return nil, err
	}

	kb := NewKeyBuilder(cfg.KeySeparator)
	return NewCRUDCache(c, kb, cfg.ServiceName), nil
}

// MustNew creates a cache or panics if it fails.
func MustNew(cfg Config, opts ...Option) Cache {
	c, err := New(cfg, opts...)
	if err != nil {
		panic(err)
	}
	return c
}

// QuickRedisConfig creates a quick Redis configuration.
func QuickRedisConfig(addr string) Config {
	return Config{
		Type: CacheTypeRedis,
		Redis: &RedisConfig{
			Addr: addr,
		},
	}
}

// QuickMemoryConfig creates a quick in-memory configuration.
func QuickMemoryConfig() Config {
	return Config{
		Type:   CacheTypeMemory,
		Memory: DefaultMemoryConfig(),
	}
}

// Builder provides a fluent interface for building cache configurations.
type Builder struct {
	cfg    Config
	logger *slog.Logger
}

// NewBuilder creates a new configuration builder.
func NewBuilder() *Builder {
	return &Builder{
		cfg: DefaultConfig(),
	}
}

// Redis sets the cache type to Redis.
func (b *Builder) Redis(addr string) *Builder {
	b.cfg.Type = CacheTypeRedis
	if b.cfg.Redis == nil {
		b.cfg.Redis = DefaultRedisConfig()
	}
	b.cfg.Redis.Addr = addr
	return b
}

// RedisCluster sets up Redis Cluster mode.
func (b *Builder) RedisCluster(addrs []string) *Builder {
	b.cfg.Type = CacheTypeRedis
	if b.cfg.Redis == nil {
		b.cfg.Redis = DefaultRedisConfig()
	}
	b.cfg.Redis.Cluster = true
	b.cfg.Redis.ClusterAddrs = addrs
	return b
}

// Memory sets the cache type to in-memory.
func (b *Builder) Memory() *Builder {
	b.cfg.Type = CacheTypeMemory
	if b.cfg.Memory == nil {
		b.cfg.Memory = DefaultMemoryConfig()
	}
	return b
}

// Multilayer sets the cache type to multilayer.
func (b *Builder) Multilayer() *Builder {
	b.cfg.Type = CacheTypeMultilayer
	return b
}

// WithService sets the service name.
func (b *Builder) WithService(name string) *Builder {
	b.cfg.ServiceName = name
	return b
}

// WithMetrics enables metrics collection.
func (b *Builder) WithMetrics() *Builder {
	b.cfg.Metrics = true
	return b
}

// WithCompression enables compression.
func (b *Builder) WithCompression() *Builder {
	b.cfg.Compress = true
	return b
}

// WithPassword sets the Redis password.
func (b *Builder) WithPassword(password string) *Builder {
	if b.cfg.Redis == nil {
		b.cfg.Redis = DefaultRedisConfig()
	}
	b.cfg.Redis.Password = password
	return b
}

// WithDB sets the Redis database.
func (b *Builder) WithDB(db int) *Builder {
	if b.cfg.Redis == nil {
		b.cfg.Redis = DefaultRedisConfig()
	}
	b.cfg.Redis.DB = db
	return b
}

// WithPoolSize sets the Redis connection pool size.
func (b *Builder) WithPoolSize(size int) *Builder {
	if b.cfg.Redis == nil {
		b.cfg.Redis = DefaultRedisConfig()
	}
	b.cfg.Redis.PoolSize = size
	return b
}

// WithKeyPrefix sets the key prefix.
func (b *Builder) WithKeyPrefix(prefix string) *Builder {
	if b.cfg.Redis == nil {
		b.cfg.Redis = DefaultRedisConfig()
	}
	b.cfg.Redis.KeyPrefix = prefix
	return b
}

// WithLogger sets the logger.
func (b *Builder) WithLogger(logger *slog.Logger) *Builder {
	b.logger = logger
	return b
}

// Config returns the built configuration.
func (b *Builder) Config() Config {
	return b.cfg
}

// Build creates the cache instance.
func (b *Builder) Build() (Cache, error) {
	var opts []Option
	if b.logger != nil {
		opts = append(opts, WithLogger(b.logger))
	}
	return New(b.cfg, opts...)
}
