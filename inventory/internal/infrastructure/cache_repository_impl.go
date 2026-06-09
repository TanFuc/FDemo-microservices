//go:build legacy

package infrastructure

import (
	"context"
	_ "embed"
	"fmt"
	"strconv"

	"microservices/inventory/internal/repository"

	"github.com/redis/go-redis/v9"
)

//go:embed scripts/reserve.lua
var reserveScript string

type cacheRepositoryImpl struct {
	client        *redis.Client
	reserveScript *redis.Script
}

func NewCacheRepository(client *redis.Client) repository.CacheRepository {
	return &cacheRepositoryImpl{
		client:        client,
		reserveScript: redis.NewScript(reserveScript),
	}
}

func (r *cacheRepositoryImpl) inventoryKey(skuID string) string {
	return fmt.Sprintf("inventory:%s", skuID)
}

func (r *cacheRepositoryImpl) Reserve(ctx context.Context, skuID string, quantity int) (bool, error) {
	key := r.inventoryKey(skuID)
	result, err := r.reserveScript.Run(ctx, r.client, []string{key}, quantity).Int()
	if err != nil {
		return false, fmt.Errorf("failed to execute reserve script: %w", err)
	}
	return result == 1, nil
}

func (r *cacheRepositoryImpl) Release(ctx context.Context, skuID string, quantity int) error {
	key := r.inventoryKey(skuID)
	return r.client.HIncrBy(ctx, key, "reserved", int64(-quantity)).Err()
}

func (r *cacheRepositoryImpl) SetInventory(ctx context.Context, skuID string, total, reserved int) error {
	key := r.inventoryKey(skuID)
	return r.client.HSet(ctx, key,
		"total", total,
		"reserved", reserved,
	).Err()
}

func (r *cacheRepositoryImpl) GetInventory(ctx context.Context, skuID string) (total int, reserved int, err error) {
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
	if r, ok := result["reserved"]; ok {
		reserved, _ = strconv.Atoi(r)
	}

	return total, reserved, nil
}
