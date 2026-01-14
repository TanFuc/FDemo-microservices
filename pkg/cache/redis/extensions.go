package redis

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"microservices/pkg/cache"
)

// =============================================================================
// Cluster Operations
// =============================================================================

// NewCluster creates a new RedisCache configured for Redis Cluster.
func NewCluster(cfg *cache.RedisConfig, opts *Options) (*RedisCache, error) {
	if cfg == nil {
		return nil, fmt.Errorf("%w: redis config is nil", cache.ErrInvalidConfig)
	}

	if len(cfg.ClusterAddrs) == 0 {
		return nil, fmt.Errorf("%w: cluster addresses are required", cache.ErrInvalidConfig)
	}

	// Ensure cluster mode is enabled
	cfg.Cluster = true
	cfg = cfg.WithDefaults()

	if opts == nil {
		opts = DefaultOptions()
	}

	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:        cfg.ClusterAddrs,
		Password:     cfg.Password,
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

// ClusterInfo returns information about the Redis Cluster.
func (c *RedisCache) ClusterInfo(ctx context.Context) (string, error) {
	if c.closed {
		return "", cache.ErrClosed
	}

	clusterClient, ok := c.client.(*redis.ClusterClient)
	if !ok {
		return "", fmt.Errorf("cache: not a cluster client")
	}

	info, err := clusterClient.ClusterInfo(ctx).Result()
	if err != nil {
		return "", cache.WrapError("ClusterInfo", "", err)
	}

	return info, nil
}

// ClusterNodes returns information about cluster nodes.
func (c *RedisCache) ClusterNodes(ctx context.Context) (string, error) {
	if c.closed {
		return "", cache.ErrClosed
	}

	clusterClient, ok := c.client.(*redis.ClusterClient)
	if !ok {
		return "", fmt.Errorf("cache: not a cluster client")
	}

	nodes, err := clusterClient.ClusterNodes(ctx).Result()
	if err != nil {
		return "", cache.WrapError("ClusterNodes", "", err)
	}

	return nodes, nil
}

// IsCluster returns true if the cache is using Redis Cluster.
func (c *RedisCache) IsCluster() bool {
	_, ok := c.client.(*redis.ClusterClient)
	return ok
}

// ForEachShard executes a function on each shard in the cluster.
// Only works when using Redis Cluster.
func (c *RedisCache) ForEachShard(ctx context.Context, fn func(ctx context.Context, client *redis.Client) error) error {
	if c.closed {
		return cache.ErrClosed
	}

	clusterClient, ok := c.client.(*redis.ClusterClient)
	if !ok {
		return fmt.Errorf("cache: not a cluster client")
	}

	return clusterClient.ForEachShard(ctx, fn)
}

// =============================================================================
// Batch & Pipeline Operations
// =============================================================================

// MGet retrieves multiple keys at once using a pipeline.
// Returns a map of key to raw bytes. Missing keys are not included in the result.
func (c *RedisCache) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	if c.closed {
		return nil, cache.ErrClosed
	}

	if len(keys) == 0 {
		return make(map[string][]byte), nil
	}

	// Prefix all keys
	prefixedKeys := make([]string, len(keys))
	keyMap := make(map[string]string, len(keys)) // prefixed -> original
	for i, key := range keys {
		prefixedKey := c.prefixKey(key)
		prefixedKeys[i] = prefixedKey
		keyMap[prefixedKey] = key
	}

	// Use MGET command
	results, err := c.client.MGet(ctx, prefixedKeys...).Result()
	if err != nil {
		c.metrics.RecordError(err)
		return nil, cache.WrapError("MGet", "", err)
	}

	// Build result map
	resultMap := make(map[string][]byte, len(keys))
	for i, result := range results {
		if result == nil {
			continue
		}
		originalKey := keyMap[prefixedKeys[i]]
		switch v := result.(type) {
		case string:
			resultMap[originalKey] = []byte(v)
		case []byte:
			resultMap[originalKey] = v
		}
	}

	return resultMap, nil
}

// MSet sets multiple key-value pairs with the same TTL using a pipeline.
func (c *RedisCache) MSet(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if c.closed {
		return cache.ErrClosed
	}

	if len(items) == 0 {
		return nil
	}

	pipe := c.client.Pipeline()

	for key, value := range items {
		prefixedKey := c.prefixKey(key)

		data, err := c.serializer.Marshal(value)
		if err != nil {
			return cache.WrapError("MSet", key, err)
		}

		pipe.Set(ctx, prefixedKey, data, ttl)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		c.metrics.RecordError(err)
		return cache.WrapError("MSet", "", err)
	}

	// Record metrics for each set
	for range items {
		c.metrics.RecordSet(0)
	}

	return nil
}

// MDelete deletes multiple keys using a pipeline.
// Returns the number of keys that were deleted.
func (c *RedisCache) MDelete(ctx context.Context, keys []string) (int64, error) {
	if c.closed {
		return 0, cache.ErrClosed
	}

	if len(keys) == 0 {
		return 0, nil
	}

	// Prefix all keys
	prefixedKeys := make([]string, len(keys))
	for i, key := range keys {
		prefixedKeys[i] = c.prefixKey(key)
	}

	// Use DEL command
	result, err := c.client.Del(ctx, prefixedKeys...).Result()
	if err != nil {
		c.metrics.RecordError(err)
		return 0, cache.WrapError("MDelete", "", err)
	}

	// Record metrics for each delete
	for i := int64(0); i < result; i++ {
		c.metrics.RecordDelete()
	}

	return result, nil
}

// MGetTyped retrieves multiple keys and unmarshals them into the provided type.
// Returns a map of key to unmarshaled value. Missing keys are not included.
func MGetTyped[T any](ctx context.Context, c *RedisCache, keys []string) (map[string]T, error) {
	rawResults, err := c.MGet(ctx, keys)
	if err != nil {
		return nil, err
	}

	results := make(map[string]T, len(rawResults))
	for key, data := range rawResults {
		var value T
		if err := c.serializer.Unmarshal(data, &value); err != nil {
			continue // Skip items that fail to unmarshal
		}
		results[key] = value
	}

	return results, nil
}

// Pipeline returns a new pipeline for executing multiple commands.
func (c *RedisCache) Pipeline() redis.Pipeliner {
	return c.client.Pipeline()
}

// TxPipeline returns a new transactional pipeline.
func (c *RedisCache) TxPipeline() redis.Pipeliner {
	return c.client.TxPipeline()
}

// RunPipeline executes a series of operations in a pipeline.
func (c *RedisCache) RunPipeline(ctx context.Context, fn func(pipe redis.Pipeliner) error) error {
	if c.closed {
		return cache.ErrClosed
	}

	pipe := c.client.Pipeline()
	if err := fn(pipe); err != nil {
		return err
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		c.metrics.RecordError(err)
		return cache.WrapError("RunPipeline", "", err)
	}

	return nil
}

// RunTxPipeline executes a series of operations in a transactional pipeline.
func (c *RedisCache) RunTxPipeline(ctx context.Context, fn func(pipe redis.Pipeliner) error) error {
	if c.closed {
		return cache.ErrClosed
	}

	pipe := c.client.TxPipeline()
	if err := fn(pipe); err != nil {
		return err
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		c.metrics.RecordError(err)
		return cache.WrapError("RunTxPipeline", "", err)
	}

	return nil
}
