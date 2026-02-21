package redis

import (
	"context"
	_ "embed"
	"fmt"
	"strconv"
	"time"

	"microservices/inventory/internal/config"
	"microservices/inventory/internal/repository"

	"github.com/redis/go-redis/v9"
)

//go:embed scripts/reserve.lua
var reserveScript string

var _ repository.CacheRepository = (*CacheRepository)(nil)

type CacheRepository struct {
	client        *redis.Client
	reserveScript *redis.Script
}

func NewCacheRepository(client *redis.Client) *CacheRepository {
	return &CacheRepository{
		client:        client,
		reserveScript: redis.NewScript(reserveScript),
	}
}

func Connect(cfg *config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return client, nil
}

func (r *CacheRepository) inventoryKey(skuID string) string {
	return fmt.Sprintf("inventory:%s", skuID)
}

func (r *CacheRepository) Reserve(ctx context.Context, skuID string, quantity int) (bool, error) {
	key := r.inventoryKey(skuID)
	result, err := r.reserveScript.Run(ctx, r.client, []string{key}, quantity).Int()
	if err != nil {
		return false, fmt.Errorf("failed to execute reserve script: %w", err)
	}
	return result == 1, nil
}

func (r *CacheRepository) Release(ctx context.Context, skuID string, quantity int) error {
	key := r.inventoryKey(skuID)
	return r.client.HIncrBy(ctx, key, "reserved", int64(-quantity)).Err()
}

func (r *CacheRepository) SetInventory(ctx context.Context, skuID string, total, reserved int) error {
	key := r.inventoryKey(skuID)
	return r.client.HSet(ctx, key,
		"total", total,
		"reserved", reserved,
	).Err()
}

func (r *CacheRepository) GetInventory(ctx context.Context, skuID string) (total int, reserved int, err error) {
	key := r.inventoryKey(skuID)
	result, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return 0, 0, err
	}

	if len(result) == 0 {
		return 0, 0, nil
	}

	if t, ok := result["total"]; ok {
		total, _ = strconv.Atoi(t)
	}
	if res, ok := result["reserved"]; ok {
		reserved, _ = strconv.Atoi(res)
	}

	return total, reserved, nil
}

func (r *CacheRepository) DeleteInventory(ctx context.Context, skuID string) error {
	key := r.inventoryKey(skuID)
	return r.client.Del(ctx, key).Err()
}
