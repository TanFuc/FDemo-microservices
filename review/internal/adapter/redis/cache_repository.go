package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"tafu-review/internal/core/domain"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	ratingKeyPrefix = "rating:"
	cacheTTL        = 30 * time.Minute
)

var ErrCacheMiss = errors.New("cache miss")

type CacheRepository struct {
	client *redis.Client
}

func NewCacheRepository(client *redis.Client) *CacheRepository {
	return &CacheRepository{client: client}
}

func (r *CacheRepository) ratingKey(productID string) string {
	return fmt.Sprintf("%s%s", ratingKeyPrefix, productID)
}

func (r *CacheRepository) GetRatingSummary(ctx context.Context, productID string) (*domain.RatingSummary, error) {
	key := r.ratingKey(productID)
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrCacheMiss
		}
		return nil, err
	}

	var summary domain.RatingSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		return nil, err
	}

	return &summary, nil
}

func (r *CacheRepository) SetRatingSummary(ctx context.Context, productID string, summary *domain.RatingSummary) error {
	key := r.ratingKey(productID)
	data, err := json.Marshal(summary)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, key, data, cacheTTL).Err()
}

func (r *CacheRepository) InvalidateRatingSummary(ctx context.Context, productID string) error {
	key := r.ratingKey(productID)
	return r.client.Del(ctx, key).Err()
}
