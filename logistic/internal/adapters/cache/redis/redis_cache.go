package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"tafu-logistic/logistics-service/internal/core/ports"
)

const (
	// FeeCacheTTL is the default TTL for cached fees (1 hour)
	FeeCacheTTL = 1 * time.Hour
)

// Cache implements ports.Cache and ports.FeeCache
type Cache struct {
	client *redis.Client
}

// NewCache creates a new Redis cache
func NewCache(client *redis.Client) *Cache {
	return &Cache{client: client}
}

// Ensure interface implementation
var _ ports.Cache = (*Cache)(nil)
var _ ports.FeeCache = (*Cache)(nil)

// Get retrieves a value from cache
func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil
		}
		return "", fmt.Errorf("failed to get from cache: %w", err)
	}
	return val, nil
}

// Set stores a value in cache with TTL
func (c *Cache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	err := c.client.Set(ctx, key, value, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}
	return nil
}

// Delete removes a value from cache
func (c *Cache) Delete(ctx context.Context, key string) error {
	err := c.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete from cache: %w", err)
	}
	return nil
}

// Exists checks if a key exists in cache
func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	result, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}
	return result > 0, nil
}

// buildFeeKey builds a cache key for shipping fee
func buildFeeKey(provider string, fromDistrict, toDistrict, weight int) string {
	return fmt.Sprintf("fee:%s:%d:%d:%d", provider, fromDistrict, toDistrict, weight)
}

// GetFee retrieves cached shipping fee
func (c *Cache) GetFee(ctx context.Context, provider string, fromDistrict, toDistrict, weight int) (float64, bool, error) {
	key := buildFeeKey(provider, fromDistrict, toDistrict, weight)

	val, err := c.Get(ctx, key)
	if err != nil {
		return 0, false, err
	}

	if val == "" {
		return 0, false, nil // Cache miss
	}

	fee, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0, false, fmt.Errorf("failed to parse cached fee: %w", err)
	}

	return fee, true, nil
}

// SetFee caches a shipping fee
func (c *Cache) SetFee(ctx context.Context, provider string, fromDistrict, toDistrict, weight int, fee float64) error {
	key := buildFeeKey(provider, fromDistrict, toDistrict, weight)
	value := strconv.FormatFloat(fee, 'f', 2, 64)

	return c.Set(ctx, key, value, FeeCacheTTL)
}

// NewRedisClient creates a new Redis client from URL
func NewRedisClient(redisURL string) (*redis.Client, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return client, nil
}
