package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// IdempotencyStore provides deduplication for processed events
type IdempotencyStore struct {
	redis *redis.Client
	ttl   time.Duration
}

// NewIdempotencyStore creates a new idempotency store backed by Redis
func NewIdempotencyStore(redisClient *redis.Client) *IdempotencyStore {
	return &IdempotencyStore{
		redis: redisClient,
		ttl:   48 * time.Hour,
	}
}

// IsProcessed returns true if eventID was already processed
func (s *IdempotencyStore) IsProcessed(ctx context.Context, eventID string) (bool, error) {
	key := fmt.Sprintf("noti:processed:%s", eventID)
	val, err := s.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return val == "1", nil
}

// MarkProcessed records an eventID as processed
func (s *IdempotencyStore) MarkProcessed(ctx context.Context, eventID string) error {
	key := fmt.Sprintf("noti:processed:%s", eventID)
	return s.redis.Set(ctx, key, "1", s.ttl).Err()
}

// SetTTL sets the TTL for processed event records
func (s *IdempotencyStore) SetTTL(ttl time.Duration) {
	s.ttl = ttl
}
