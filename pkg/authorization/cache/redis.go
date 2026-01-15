package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"

	"microservices/pkg/authorization"
)

// RedisCache implements Cache interface using Redis
type RedisCache struct {
	client     *redis.Client
	keyBuilder *KeyBuilder
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
		KeyPrefix:    "authz:",
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
		return nil, fmt.Errorf("%w: failed to connect to Redis: %v", authorization.ErrConnectionFailed, err)
	}

	return &RedisCache{
		client:     client,
		keyBuilder: NewKeyBuilder(cfg.KeyPrefix),
		defaultTTL: cfg.DefaultTTL,
	}, nil
}

// NewRedisCacheWithClient creates a new Redis cache with an existing client
func NewRedisCacheWithClient(client *redis.Client, keyPrefix string, defaultTTL time.Duration) *RedisCache {
	if keyPrefix == "" {
		keyPrefix = "authz:"
	}
	if defaultTTL == 0 {
		defaultTTL = 5 * time.Minute
	}
	return &RedisCache{
		client:     client,
		keyBuilder: NewKeyBuilder(keyPrefix),
		defaultTTL: defaultTTL,
	}
}

// Get retrieves a cached permission decision
func (c *RedisCache) Get(ctx context.Context, key string) (*CachedDecision, error) {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			atomic.AddInt64(&c.misses, 1)
			return nil, nil
		}
		return nil, fmt.Errorf("%w: failed to get from cache: %v", authorization.ErrCacheOperation, err)
	}

	var decision CachedDecision
	if err := json.Unmarshal(data, &decision); err != nil {
		return nil, fmt.Errorf("%w: failed to unmarshal cached decision: %v", authorization.ErrCacheOperation, err)
	}

	if decision.IsExpired() {
		atomic.AddInt64(&c.misses, 1)
		// Delete expired entry
		c.client.Del(ctx, key)
		return nil, nil
	}

	atomic.AddInt64(&c.hits, 1)
	return &decision, nil
}

// Set caches a permission decision
func (c *RedisCache) Set(ctx context.Context, key string, decision *CachedDecision, ttl time.Duration) error {
	if ttl == 0 {
		ttl = c.defaultTTL
	}

	decision.CachedAt = time.Now()
	decision.ExpiresAt = time.Now().Add(ttl)

	data, err := json.Marshal(decision)
	if err != nil {
		return fmt.Errorf("%w: failed to marshal decision: %v", authorization.ErrCacheOperation, err)
	}

	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("%w: failed to set cache: %v", authorization.ErrCacheOperation, err)
	}

	return nil
}

// Delete removes a cached entry
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("%w: failed to delete from cache: %v", authorization.ErrCacheOperation, err)
	}
	return nil
}

// DeleteByPattern removes entries matching the pattern
func (c *RedisCache) DeleteByPattern(ctx context.Context, pattern string) error {
	var cursor uint64
	var keys []string

	for {
		var err error
		var batch []string
		batch, cursor, err = c.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("%w: failed to scan keys: %v", authorization.ErrCacheOperation, err)
		}
		keys = append(keys, batch...)
		if cursor == 0 {
			break
		}
	}

	if len(keys) > 0 {
		if err := c.client.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("%w: failed to delete keys: %v", authorization.ErrCacheOperation, err)
		}
	}

	return nil
}

// Clear removes all cached entries with the cache prefix
func (c *RedisCache) Clear(ctx context.Context) error {
	return c.DeleteByPattern(ctx, c.keyBuilder.AllPattern())
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

	// Get Redis info
	info, err := c.client.Info(ctx, "memory").Result()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get Redis info: %v", authorization.ErrCacheOperation, err)
	}

	// Get key count
	var size int64
	var cursor uint64
	for {
		var batch []string
		batch, cursor, err = c.client.Scan(ctx, cursor, c.keyBuilder.AllPattern(), 100).Result()
		if err != nil {
			break
		}
		size += int64(len(batch))
		if cursor == 0 {
			break
		}
	}

	_ = info // Can parse memory usage from info if needed

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
		return fmt.Errorf("%w: Redis ping failed: %v", authorization.ErrCacheOperation, err)
	}
	return nil
}

// Close closes the cache connection
func (c *RedisCache) Close() error {
	return c.client.Close()
}

// KeyBuilder returns the key builder for constructing cache keys
func (c *RedisCache) KeyBuilder() *KeyBuilder {
	return c.keyBuilder
}

// Client returns the underlying Redis client
func (c *RedisCache) Client() *redis.Client {
	return c.client
}

// SetEnforceDecision is a convenience method to cache an enforce decision
func (c *RedisCache) SetEnforceDecision(ctx context.Context, subject, object, action, domain string, allowed bool, ttl time.Duration) error {
	key := c.keyBuilder.EnforceKey(subject, object, action, domain)
	return c.Set(ctx, key, &CachedDecision{
		Allowed: allowed,
	}, ttl)
}

// GetEnforceDecision is a convenience method to get a cached enforce decision
func (c *RedisCache) GetEnforceDecision(ctx context.Context, subject, object, action, domain string) (*CachedDecision, error) {
	key := c.keyBuilder.EnforceKey(subject, object, action, domain)
	return c.Get(ctx, key)
}

// InvalidateUser invalidates all cached entries for a user
func (c *RedisCache) InvalidateUser(ctx context.Context, userID string) error {
	return c.DeleteByPattern(ctx, c.keyBuilder.UserPattern(userID))
}

// InvalidateDomain invalidates all cached entries for a domain
func (c *RedisCache) InvalidateDomain(ctx context.Context, domain string) error {
	return c.DeleteByPattern(ctx, c.keyBuilder.DomainPattern(domain))
}

// ResetStats resets the cache statistics
func (c *RedisCache) ResetStats() {
	atomic.StoreInt64(&c.hits, 0)
	atomic.StoreInt64(&c.misses, 0)
}
