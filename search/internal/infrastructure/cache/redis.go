package cache

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient wraps Redis operations for caching
type RedisClient struct {
	client *redis.Client
	logger *slog.Logger
}

// NewRedisClient creates a new Redis client wrapper
func NewRedisClient(addr, password string, db int, logger *slog.Logger) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	logger.Info("connected to redis", "addr", addr)

	return &RedisClient{
		client: client,
		logger: logger,
	}, nil
}

// Get retrieves a cached value by key
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil // Cache miss, return empty string
	}
	if err != nil {
		return "", fmt.Errorf("failed to get from cache: %w", err)
	}
	r.logger.Debug("cache hit", "key", key)
	return val, nil
}

// Set stores a value with TTL in seconds
func (r *RedisClient) Set(ctx context.Context, key string, value string, ttlSeconds int) error {
	err := r.client.Set(ctx, key, value, time.Duration(ttlSeconds)*time.Second).Err()
	if err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}
	r.logger.Debug("cache set", "key", key, "ttl", ttlSeconds)
	return nil
}

// Delete removes a cached key
func (r *RedisClient) Delete(ctx context.Context, key string) error {
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete from cache: %w", err)
	}
	r.logger.Debug("cache deleted", "key", key)
	return nil
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	return r.client.Close()
}
