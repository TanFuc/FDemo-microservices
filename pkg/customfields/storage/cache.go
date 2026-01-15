package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"

	"microservices/pkg/customfields"
)

// RedisCache implements the Cache interface using Redis
type RedisCache struct {
	client     *redis.Client
	keyPrefix  string
	defaultTTL time.Duration

	// Statistics
	hits   int64
	misses int64
}

// RedisCacheConfig represents Redis cache configuration
type RedisCacheConfig struct {
	Addr         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	MaxRetries   int
	KeyPrefix    string
	DefaultTTL   time.Duration
}

// DefaultRedisCacheConfig returns the default Redis cache configuration
func DefaultRedisCacheConfig() *RedisCacheConfig {
	return &RedisCacheConfig{
		Addr:         "localhost:6379",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		MaxRetries:   3,
		KeyPrefix:    "cf:",
		DefaultTTL:   5 * time.Minute,
	}
}

// NewRedisCache creates a new Redis cache
func NewRedisCache(cfg *RedisCacheConfig) (*RedisCache, error) {
	if cfg == nil {
		cfg = DefaultRedisCacheConfig()
	}

	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		MaxRetries:   cfg.MaxRetries,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), cfg.DialTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("%w: failed to connect to Redis: %v", customfields.ErrConnectionFailed, err)
	}

	keyPrefix := cfg.KeyPrefix
	if keyPrefix == "" {
		keyPrefix = "cf:"
	}

	defaultTTL := cfg.DefaultTTL
	if defaultTTL == 0 {
		defaultTTL = 5 * time.Minute
	}

	return &RedisCache{
		client:     client,
		keyPrefix:  keyPrefix,
		defaultTTL: defaultTTL,
	}, nil
}

// NewRedisCacheWithClient creates a new Redis cache with an existing client
func NewRedisCacheWithClient(client *redis.Client, keyPrefix string, defaultTTL time.Duration) *RedisCache {
	if keyPrefix == "" {
		keyPrefix = "cf:"
	}
	if defaultTTL == 0 {
		defaultTTL = 5 * time.Minute
	}
	return &RedisCache{
		client:     client,
		keyPrefix:  keyPrefix,
		defaultTTL: defaultTTL,
	}
}

// Key generation helpers

func (c *RedisCache) definitionKey(id string) string {
	return c.keyPrefix + "def:" + id
}

func (c *RedisCache) definitionsByEntityKey(entityType string) string {
	return c.keyPrefix + "defs:" + entityType
}

func (c *RedisCache) valuesKey(entityType, entityID string) string {
	return c.keyPrefix + "vals:" + entityType + ":" + entityID
}

// GetDefinition retrieves a cached field definition
func (c *RedisCache) GetDefinition(ctx context.Context, id string) (*customfields.FieldDefinition, error) {
	data, err := c.client.Get(ctx, c.definitionKey(id)).Bytes()
	if err != nil {
		if err == redis.Nil {
			atomic.AddInt64(&c.misses, 1)
			return nil, nil
		}
		return nil, fmt.Errorf("%w: failed to get definition from cache: %v", customfields.ErrCacheOperation, err)
	}

	var def customfields.FieldDefinition
	if err := json.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("%w: failed to unmarshal definition: %v", customfields.ErrCacheOperation, err)
	}

	atomic.AddInt64(&c.hits, 1)
	return &def, nil
}

// SetDefinition caches a field definition
func (c *RedisCache) SetDefinition(ctx context.Context, def *customfields.FieldDefinition) error {
	data, err := json.Marshal(def)
	if err != nil {
		return fmt.Errorf("%w: failed to marshal definition: %v", customfields.ErrCacheOperation, err)
	}

	if err := c.client.Set(ctx, c.definitionKey(def.ID), data, c.defaultTTL).Err(); err != nil {
		return fmt.Errorf("%w: failed to set definition in cache: %v", customfields.ErrCacheOperation, err)
	}

	return nil
}

// DeleteDefinition removes a cached field definition
func (c *RedisCache) DeleteDefinition(ctx context.Context, id string) error {
	if err := c.client.Del(ctx, c.definitionKey(id)).Err(); err != nil {
		return fmt.Errorf("%w: failed to delete definition from cache: %v", customfields.ErrCacheOperation, err)
	}
	return nil
}

// GetDefinitionsByEntity retrieves cached field definitions for an entity type
func (c *RedisCache) GetDefinitionsByEntity(ctx context.Context, entityType string) ([]*customfields.FieldDefinition, error) {
	data, err := c.client.Get(ctx, c.definitionsByEntityKey(entityType)).Bytes()
	if err != nil {
		if err == redis.Nil {
			atomic.AddInt64(&c.misses, 1)
			return nil, nil
		}
		return nil, fmt.Errorf("%w: failed to get definitions from cache: %v", customfields.ErrCacheOperation, err)
	}

	var defs []*customfields.FieldDefinition
	if err := json.Unmarshal(data, &defs); err != nil {
		return nil, fmt.Errorf("%w: failed to unmarshal definitions: %v", customfields.ErrCacheOperation, err)
	}

	atomic.AddInt64(&c.hits, 1)
	return defs, nil
}

// SetDefinitionsByEntity caches field definitions for an entity type
func (c *RedisCache) SetDefinitionsByEntity(ctx context.Context, entityType string, defs []*customfields.FieldDefinition) error {
	data, err := json.Marshal(defs)
	if err != nil {
		return fmt.Errorf("%w: failed to marshal definitions: %v", customfields.ErrCacheOperation, err)
	}

	if err := c.client.Set(ctx, c.definitionsByEntityKey(entityType), data, c.defaultTTL).Err(); err != nil {
		return fmt.Errorf("%w: failed to set definitions in cache: %v", customfields.ErrCacheOperation, err)
	}

	return nil
}

// DeleteDefinitionsByEntity removes cached field definitions for an entity type
func (c *RedisCache) DeleteDefinitionsByEntity(ctx context.Context, entityType string) error {
	if err := c.client.Del(ctx, c.definitionsByEntityKey(entityType)).Err(); err != nil {
		return fmt.Errorf("%w: failed to delete definitions from cache: %v", customfields.ErrCacheOperation, err)
	}
	return nil
}

// GetValues retrieves cached field values for an entity
func (c *RedisCache) GetValues(ctx context.Context, entityType, entityID string) (map[string]interface{}, error) {
	data, err := c.client.Get(ctx, c.valuesKey(entityType, entityID)).Bytes()
	if err != nil {
		if err == redis.Nil {
			atomic.AddInt64(&c.misses, 1)
			return nil, nil
		}
		return nil, fmt.Errorf("%w: failed to get values from cache: %v", customfields.ErrCacheOperation, err)
	}

	var values map[string]interface{}
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("%w: failed to unmarshal values: %v", customfields.ErrCacheOperation, err)
	}

	atomic.AddInt64(&c.hits, 1)
	return values, nil
}

// SetValues caches field values for an entity
func (c *RedisCache) SetValues(ctx context.Context, entityType, entityID string, values map[string]interface{}) error {
	data, err := json.Marshal(values)
	if err != nil {
		return fmt.Errorf("%w: failed to marshal values: %v", customfields.ErrCacheOperation, err)
	}

	if err := c.client.Set(ctx, c.valuesKey(entityType, entityID), data, c.defaultTTL).Err(); err != nil {
		return fmt.Errorf("%w: failed to set values in cache: %v", customfields.ErrCacheOperation, err)
	}

	return nil
}

// DeleteValues removes cached field values for an entity
func (c *RedisCache) DeleteValues(ctx context.Context, entityType, entityID string) error {
	if err := c.client.Del(ctx, c.valuesKey(entityType, entityID)).Err(); err != nil {
		return fmt.Errorf("%w: failed to delete values from cache: %v", customfields.ErrCacheOperation, err)
	}
	return nil
}

// GetStats returns cache statistics
func (c *RedisCache) GetStats(ctx context.Context) (*CacheStats, error) {
	hits := atomic.LoadInt64(&c.hits)
	misses := atomic.LoadInt64(&c.misses)

	var hitRate float64
	total := hits + misses
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}

	// Get Redis info for memory usage
	info, _ := c.client.Info(ctx, "memory").Result()
	_ = info // Could parse for memory stats

	// Count keys
	var size int64
	var cursor uint64
	for {
		var keys []string
		var err error
		keys, cursor, err = c.client.Scan(ctx, cursor, c.keyPrefix+"*", 100).Result()
		if err != nil {
			break
		}
		size += int64(len(keys))
		if cursor == 0 {
			break
		}
	}

	return &CacheStats{
		Hits:    hits,
		Misses:  misses,
		HitRate: hitRate,
		Size:    size,
	}, nil
}

// Ping checks the cache connection
func (c *RedisCache) Ping(ctx context.Context) error {
	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("%w: Redis ping failed: %v", customfields.ErrCacheOperation, err)
	}
	return nil
}

// Clear clears all cached data
func (c *RedisCache) Clear(ctx context.Context) error {
	var cursor uint64
	var keys []string

	for {
		var batch []string
		var err error
		batch, cursor, err = c.client.Scan(ctx, cursor, c.keyPrefix+"*", 100).Result()
		if err != nil {
			return fmt.Errorf("%w: failed to scan keys: %v", customfields.ErrCacheOperation, err)
		}
		keys = append(keys, batch...)
		if cursor == 0 {
			break
		}
	}

	if len(keys) > 0 {
		if err := c.client.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("%w: failed to delete keys: %v", customfields.ErrCacheOperation, err)
		}
	}

	// Reset stats
	atomic.StoreInt64(&c.hits, 0)
	atomic.StoreInt64(&c.misses, 0)

	return nil
}

// Close closes the cache connection
func (c *RedisCache) Close() error {
	return c.client.Close()
}

// Client returns the underlying Redis client
func (c *RedisCache) Client() *redis.Client {
	return c.client
}
