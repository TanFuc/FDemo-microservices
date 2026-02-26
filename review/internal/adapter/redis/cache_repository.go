package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"microservices/review/internal/core/domain"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	ratingKeyPrefix     = "rating:"
	idempotencyPrefix   = "review:idempotent:"
	cacheTTL            = 30 * time.Minute
	idempotencyTTL      = 24 * time.Hour // Prevent duplicate reviews for 24 hours
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

// idempotencyKey generates the idempotency key for review creation
// Uses order_id + user_id to ensure one review per order per user
func (r *CacheRepository) idempotencyKey(orderID, userID string) string {
	return fmt.Sprintf("%s%s:%s", idempotencyPrefix, orderID, userID)
}

// CheckAndSetReviewIdempotency checks if a review has already been created for this order+user combination
// Returns true if this is a new request (not a duplicate), false if it's a duplicate
// Uses SETNX (SET if Not eXists) for atomic check-and-set
func (r *CacheRepository) CheckAndSetReviewIdempotency(ctx context.Context, orderID, userID, reviewID string) (bool, error) {
	key := r.idempotencyKey(orderID, userID)

	// Use SETNX to atomically check and set
	// Returns true if the key was set (new request), false if it already exists (duplicate)
	wasSet, err := r.client.SetNX(ctx, key, reviewID, idempotencyTTL).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check idempotency: %w", err)
	}

	return wasSet, nil
}

// GetExistingReviewID returns the review ID if a review already exists for this order+user combination
// Returns empty string if no existing review is found
func (r *CacheRepository) GetExistingReviewID(ctx context.Context, orderID, userID string) (string, error) {
	key := r.idempotencyKey(orderID, userID)

	reviewID, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", nil // No existing review
		}
		return "", fmt.Errorf("failed to get existing review ID: %w", err)
	}

	return reviewID, nil
}

// ClearReviewIdempotency clears the idempotency key (useful for testing or manual cleanup)
func (r *CacheRepository) ClearReviewIdempotency(ctx context.Context, orderID, userID string) error {
	key := r.idempotencyKey(orderID, userID)
	return r.client.Del(ctx, key).Err()
}
