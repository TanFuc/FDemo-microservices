package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"microservices/cart/internal/config"
)

// NewClient creates a new Redis client with the given configuration.
func NewClient(cfg *config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return client, nil
}

// Close closes the Redis client connection.
func Close(client *redis.Client) error {
	if client != nil {
		return client.Close()
	}
	return nil
}
