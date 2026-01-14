// Package redis provides a Redis-based cache implementation.
package redis

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"microservices/pkg/cache"
)

// Ensure RedisCache implements the required interfaces.
var (
	_ cache.Cache        = (*RedisCache)(nil)
	_ cache.BatchCache   = (*RedisCache)(nil)
	_ cache.PatternCache = (*RedisCache)(nil)
	_ cache.ListCache    = (*RedisCache)(nil)
	_ cache.MetricsCache = (*RedisCache)(nil)
)

// RedisCache implements cache.Cache using Redis as the backend.
type RedisCache struct {
	client     redis.UniversalClient
	serializer cache.Serializer
	logger     *slog.Logger
	metrics    cache.MetricsRecorder
	prefix     string
	closed     bool
}

// Options contains configuration options for RedisCache.
type Options struct {
	Serializer cache.Serializer
	Logger     *slog.Logger
	Metrics    bool
	Prefix     string
}

// DefaultOptions returns the default options.
func DefaultOptions() *Options {
	return &Options{
		Serializer: cache.DefaultSerializer(),
		Logger:     slog.Default(),
		Metrics:    false,
		Prefix:     "",
	}
}

// New creates a new RedisCache with the given configuration.
func New(cfg *cache.RedisConfig, opts *Options) (*RedisCache, error) {
	if cfg == nil {
		return nil, fmt.Errorf("%w: redis config is nil", cache.ErrInvalidConfig)
	}

	cfg = cfg.WithDefaults()

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	if opts == nil {
		opts = DefaultOptions()
	}

	var client redis.UniversalClient

	if cfg.Cluster {
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        cfg.ClusterAddrs,
			Password:     cfg.Password,
			PoolSize:     cfg.PoolSize,
			MinIdleConns: cfg.MinIdleConns,
			DialTimeout:  cfg.DialTimeout,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			MaxRetries:   cfg.MaxRetries,
		})
	} else {
		client = redis.NewClient(&redis.Options{
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
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), cfg.DialTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("%w: %v", cache.ErrConnection, err)
	}

	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	serializer := opts.Serializer
	if serializer == nil {
		serializer = cache.DefaultSerializer()
	}

	prefix := opts.Prefix
	if prefix == "" && cfg.KeyPrefix != "" {
		prefix = cfg.KeyPrefix
	}

	return &RedisCache{
		client:     client,
		serializer: serializer,
		logger:     logger,
		metrics:    cache.NewMetricsRecorder(opts.Metrics),
		prefix:     prefix,
	}, nil
}

// prefixKey adds the prefix to a key if configured.
func (c *RedisCache) prefixKey(key string) string {
	if c.prefix == "" {
		return key
	}
	return c.prefix + ":" + key
}

// Get retrieves a value from cache and unmarshals it into dest.
func (c *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	if c.closed {
		return cache.ErrClosed
	}

	if dest == nil {
		return cache.ErrNilPointer
	}

	start := time.Now()
	prefixedKey := c.prefixKey(key)

	data, err := c.client.Get(ctx, prefixedKey).Bytes()
	latency := time.Since(start)

	if err != nil {
		if err == redis.Nil {
			c.metrics.RecordMiss(latency)
			return cache.ErrCacheMiss
		}
		c.metrics.RecordError(err)
		c.logger.Error("cache get error", "key", key, "error", err)
		return cache.WrapError("Get", key, err)
	}

	if err := c.serializer.Unmarshal(data, dest); err != nil {
		c.metrics.RecordError(err)
		return cache.WrapError("Get", key, err)
	}

	c.metrics.RecordHit(latency)
	c.logger.Debug("cache hit", "key", key)
	return nil
}

// Set stores a value in cache with the specified TTL.
func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if c.closed {
		return cache.ErrClosed
	}

	start := time.Now()
	prefixedKey := c.prefixKey(key)

	data, err := c.serializer.Marshal(value)
	if err != nil {
		c.metrics.RecordError(err)
		return cache.WrapError("Set", key, err)
	}

	if err := c.client.Set(ctx, prefixedKey, data, ttl).Err(); err != nil {
		c.metrics.RecordError(err)
		c.logger.Error("cache set error", "key", key, "error", err)
		return cache.WrapError("Set", key, err)
	}

	c.metrics.RecordSet(time.Since(start))
	c.logger.Debug("cache set", "key", key, "ttl", ttl)
	return nil
}

// Delete removes a key from cache.
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	if c.closed {
		return cache.ErrClosed
	}

	prefixedKey := c.prefixKey(key)

	if err := c.client.Del(ctx, prefixedKey).Err(); err != nil {
		c.metrics.RecordError(err)
		c.logger.Error("cache delete error", "key", key, "error", err)
		return cache.WrapError("Delete", key, err)
	}

	c.metrics.RecordDelete()
	c.logger.Debug("cache delete", "key", key)
	return nil
}

// Exists checks if a key exists in cache.
func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	if c.closed {
		return false, cache.ErrClosed
	}

	prefixedKey := c.prefixKey(key)

	result, err := c.client.Exists(ctx, prefixedKey).Result()
	if err != nil {
		c.metrics.RecordError(err)
		c.logger.Error("cache exists error", "key", key, "error", err)
		return false, cache.WrapError("Exists", key, err)
	}

	return result > 0, nil
}

// Close closes the Redis connection.
func (c *RedisCache) Close() error {
	if c.closed {
		return nil
	}
	c.closed = true
	return c.client.Close()
}

// Ping checks if the Redis connection is alive.
func (c *RedisCache) Ping(ctx context.Context) error {
	if c.closed {
		return cache.ErrClosed
	}

	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("%w: %v", cache.ErrConnection, err)
	}
	return nil
}

// Stats returns the current cache statistics.
func (c *RedisCache) Stats() cache.CacheStats {
	return c.metrics.Stats()
}

// ResetStats resets all statistics.
func (c *RedisCache) ResetStats() {
	c.metrics.Reset()
}

// GetClient returns the underlying Redis client.
// Use with caution for advanced operations not covered by the Cache interface.
func (c *RedisCache) GetClient() redis.UniversalClient {
	return c.client
}

// SetNX sets a value only if the key does not exist.
// Returns true if the key was set.
func (c *RedisCache) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	if c.closed {
		return false, cache.ErrClosed
	}

	prefixedKey := c.prefixKey(key)

	data, err := c.serializer.Marshal(value)
	if err != nil {
		return false, cache.WrapError("SetNX", key, err)
	}

	result, err := c.client.SetNX(ctx, prefixedKey, data, ttl).Result()
	if err != nil {
		return false, cache.WrapError("SetNX", key, err)
	}

	return result, nil
}

// GetSet atomically sets a new value and returns the old value.
func (c *RedisCache) GetSet(ctx context.Context, key string, value interface{}, dest interface{}) error {
	if c.closed {
		return cache.ErrClosed
	}

	prefixedKey := c.prefixKey(key)

	data, err := c.serializer.Marshal(value)
	if err != nil {
		return cache.WrapError("GetSet", key, err)
	}

	oldData, err := c.client.GetSet(ctx, prefixedKey, data).Bytes()
	if err != nil {
		if err == redis.Nil {
			return cache.ErrCacheMiss
		}
		return cache.WrapError("GetSet", key, err)
	}

	if dest != nil {
		if err := c.serializer.Unmarshal(oldData, dest); err != nil {
			return cache.WrapError("GetSet", key, err)
		}
	}

	return nil
}

// Expire updates the TTL of a key.
func (c *RedisCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if c.closed {
		return cache.ErrClosed
	}

	prefixedKey := c.prefixKey(key)

	if err := c.client.Expire(ctx, prefixedKey, ttl).Err(); err != nil {
		return cache.WrapError("Expire", key, err)
	}

	return nil
}

// TTL returns the remaining TTL of a key.
// Returns -1 if the key exists but has no TTL.
// Returns -2 if the key does not exist.
func (c *RedisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	if c.closed {
		return 0, cache.ErrClosed
	}

	prefixedKey := c.prefixKey(key)

	result, err := c.client.TTL(ctx, prefixedKey).Result()
	if err != nil {
		return 0, cache.WrapError("TTL", key, err)
	}

	return result, nil
}

// Incr increments a counter by 1.
func (c *RedisCache) Incr(ctx context.Context, key string) (int64, error) {
	if c.closed {
		return 0, cache.ErrClosed
	}

	prefixedKey := c.prefixKey(key)

	result, err := c.client.Incr(ctx, prefixedKey).Result()
	if err != nil {
		return 0, cache.WrapError("Incr", key, err)
	}

	return result, nil
}

// IncrBy increments a counter by the given amount.
func (c *RedisCache) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	if c.closed {
		return 0, cache.ErrClosed
	}

	prefixedKey := c.prefixKey(key)

	result, err := c.client.IncrBy(ctx, prefixedKey, value).Result()
	if err != nil {
		return 0, cache.WrapError("IncrBy", key, err)
	}

	return result, nil
}

// Decr decrements a counter by 1.
func (c *RedisCache) Decr(ctx context.Context, key string) (int64, error) {
	if c.closed {
		return 0, cache.ErrClosed
	}

	prefixedKey := c.prefixKey(key)

	result, err := c.client.Decr(ctx, prefixedKey).Result()
	if err != nil {
		return 0, cache.WrapError("Decr", key, err)
	}

	return result, nil
}

// GetSerializer returns the serializer used by this cache.
func (c *RedisCache) GetSerializer() cache.Serializer {
	return c.serializer
}
